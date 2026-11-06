package ar.com.tengolibre.app.push

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.app.RemoteInput
import androidx.core.content.ContextCompat
import ar.com.tengolibre.app.BuildConfig
import ar.com.tengolibre.app.MainActivity
import ar.com.tengolibre.app.R
import ar.com.tengolibre.app.app.Estado
import ar.com.tengolibre.app.nucleo.ActividadTurno
import ar.com.tengolibre.app.nucleo.Api
import ar.com.tengolibre.app.nucleo.FechaApi
import ar.com.tengolibre.app.nucleo.Llavero
import ar.com.tengolibre.app.nucleo.Preferencias
import ar.com.tengolibre.app.nucleo.Resena
import ar.com.tengolibre.app.nucleo.Turno
import com.google.firebase.FirebaseApp
import com.google.firebase.FirebaseOptions
import com.google.firebase.messaging.FirebaseMessaging
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import kotlinx.coroutines.tasks.await
import kotlinx.serialization.Serializable

// El push de Android: el Delegado de App/TengoLibreApp.swift (APNs y las categorías con acciones) con FCM.

/** FCM sin google-services.json (como Caigo): FirebaseApp se arma a mano con tl.firebase.* de local.properties. Sin
 *  esos datos el push queda apagado y la app anda igual. */
object Push {
    /** Los canales: los avisos del negocio (alta), el turno en curso (sin sonido) y los turnos de la clienta. */
    const val CANAL_AVISOS = "avisos"
    const val CANAL_TURNO = "turno"
    const val CANAL_CLIENTA = "clienta"

    val disponible: Boolean
        get() = BuildConfig.FIREBASE_APP_ID.isNotBlank() && BuildConfig.FIREBASE_API_KEY.isNotBlank() &&
            BuildConfig.FIREBASE_PROYECTO.isNotBlank() && BuildConfig.FIREBASE_REMITENTE.isNotBlank()

    fun iniciar(ctx: Context) {
        val nm = ctx.getSystemService(NotificationManager::class.java)
        nm.createNotificationChannel(NotificationChannel(CANAL_AVISOS, "Avisos del negocio", NotificationManager.IMPORTANCE_HIGH).apply {
            description = "Reservas, cancelaciones, reseñas y la lista de espera."
        })
        nm.createNotificationChannel(NotificationChannel(CANAL_TURNO, "Turno en curso", NotificationManager.IMPORTANCE_LOW).apply {
            description = "La cuenta regresiva del turno que estás atendiendo."
            setShowBadge(false)
        })
        nm.createNotificationChannel(NotificationChannel(CANAL_CLIENTA, "Tus turnos", NotificationManager.IMPORTANCE_HIGH).apply {
            description = "Confirmaciones, recordatorios y cambios de los turnos que reservaste."
        })
        if (!disponible || FirebaseApp.getApps(ctx).isNotEmpty()) return
        runCatching {
            FirebaseApp.initializeApp(ctx, FirebaseOptions.Builder()
                .setApplicationId(BuildConfig.FIREBASE_APP_ID).setApiKey(BuildConfig.FIREBASE_API_KEY)
                .setProjectId(BuildConfig.FIREBASE_PROYECTO).setGcmSenderId(BuildConfig.FIREBASE_REMITENTE).build())
        }
    }

    /** El token de FCM de este teléfono (null sin push). */
    suspend fun token(ctx: Context): String? {
        if (!disponible || FirebaseApp.getApps(ctx).isEmpty()) return null
        return runCatching { FirebaseMessaging.getInstance().token.await() }.getOrNull()
    }

    /** Si ya hay permiso, registra el token en la API (`POST /v1/yo/dispositivos`). */
    suspend fun registrar(ctx: Context, estado: Estado) {
        if (necesitaPermiso(ctx)) return
        token(ctx)?.let { estado.registrarTokenPush(it) }
    }

    /** Android 13+ pide permiso para mostrar avisos; antes viene dado. */
    fun necesitaPermiso(ctx: Context): Boolean =
        Build.VERSION.SDK_INT >= 33 && ContextCompat.checkSelfPermission(ctx, android.Manifest.permission.POST_NOTIFICATIONS) != android.content.pm.PackageManager.PERMISSION_GRANTED

    fun hayPermiso(ctx: Context): Boolean = !necesitaPermiso(ctx) && NotificationManagerCompat.from(ctx).areNotificationsEnabled()

    /** Una Api fuera de la app abierta (acciones de los avisos, widgets): la sesión del Llavero y el negocio elegido. */
    fun api(ctx: Context, negocio: String? = null): Api? {
        Llavero.contexto = Llavero.contexto ?: ctx.applicationContext
        Preferencias.contexto = Preferencias.contexto ?: ctx.applicationContext
        val token = Llavero.leer("token") ?: return null
        return Api(Api.urlConfigurada(), token).also { it.negocio = negocio ?: Llavero.leer("negocio") }
    }

    /** Lo que abre un aviso al tocarlo: la app con la ruta (tengolibre://…) y el negocio del aviso. */
    fun abrir(ctx: Context, ruta: String, negocio: String?, tipo: String?, codigo: Int): PendingIntent {
        val i = Intent(ctx, MainActivity::class.java).apply {
            action = Intent.ACTION_VIEW
            data = Uri.parse("tengolibre://" + ruta.ifEmpty { "hoy" })
            flags = Intent.FLAG_ACTIVITY_SINGLE_TOP or Intent.FLAG_ACTIVITY_CLEAR_TOP
            putExtra("negocio", negocio)
            putExtra("tipo", tipo)
        }
        return PendingIntent.getActivity(ctx, codigo, i, PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT)
    }

    /** Una acción que corre sin abrir la app (Confirmar, Avisar a la lista, Responder, Atendido). */
    fun accion(ctx: Context, que: String, turnoId: String, negocio: String?, aviso: Int): PendingIntent {
        val i = Intent(ctx, AccionesPush::class.java).apply {
            action = "ar.com.tengolibre.app.$que"
            putExtra("turno", turnoId); putExtra("negocio", negocio); putExtra("aviso", aviso)
        }
        val mutable = if (que == "RESPONDER") PendingIntent.FLAG_MUTABLE else PendingIntent.FLAG_IMMUTABLE
        return PendingIntent.getBroadcast(ctx, (que + turnoId).hashCode(), i, mutable or PendingIntent.FLAG_UPDATE_CURRENT)
    }

    /** Arma y muestra un aviso con las acciones de su categoría (las mismas que las categorías de iOS). */
    fun mostrar(ctx: Context, d: Map<String, String>) {
        val titulo = d["titulo"] ?: return
        val cuerpo = d["cuerpo"].orEmpty()
        val ruta = d["ruta"].orEmpty()
        val negocio = d["negocio"]
        val tipo = d["tipo"]
        val turnoId = if (ruta.startsWith("turnos/")) ruta.removePrefix("turnos/") else null
        val id = (d["colapso"] ?: ruta.ifEmpty { titulo + cuerpo }).hashCode()
        val clienta = tipo == "cliente"
        val b = NotificationCompat.Builder(ctx, if (clienta) CANAL_CLIENTA else CANAL_AVISOS)
            .setSmallIcon(R.drawable.ic_aviso)
            .setColor(0xFF3CCC90.toInt())
            .setContentTitle(titulo)
            .setContentText(cuerpo)
            .setStyle(NotificationCompat.BigTextStyle().bigText(cuerpo))
            .setAutoCancel(true)
            .setGroup(if (clienta) "clienta" else "negocio-$negocio")
            .setContentIntent(abrir(ctx, ruta, negocio, tipo, id))
        if (turnoId != null) when (d["categoria"]) {
            "TL_RESERVA" -> {
                b.addAction(0, "Confirmar", accion(ctx, "CONFIRMAR", turnoId, negocio, id))
                b.addAction(0, "Ver turno", abrir(ctx, ruta, negocio, tipo, id + 1))
            }
            "TL_CANCELO" -> {
                b.addAction(0, "Avisar a la lista", accion(ctx, "AVISAR", turnoId, negocio, id))
                b.addAction(0, "Cargar un turno", abrir(ctx, "nuevo", negocio, tipo, id + 1))
            }
            "TL_HUECO" -> b.addAction(0, "Avisar a la lista", accion(ctx, "AVISAR", turnoId, negocio, id))
            "TL_RESENA" -> b.addAction(NotificationCompat.Action.Builder(0, "Responder", accion(ctx, "RESPONDER", turnoId, negocio, id))
                .addRemoteInput(RemoteInput.Builder("respuesta").setLabel("Gracias por…").build()).setAllowGeneratedReplies(false).build())
        }
        if (!hayPermiso(ctx)) return
        runCatching { NotificationManagerCompat.from(ctx).notify(id, b.build()) }
    }
}

/** Recibe los avisos de FCM (vienen sólo con datos: la app arma la notificación con sus acciones) y la notificación
 *  en curso del turno que mueve el servidor. */
class ServicioPush : FirebaseMessagingService() {
    private val alcance = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    override fun onNewToken(token: String) {
        val api = Push.api(this) ?: return
        alcance.launch { runCatching { api.hacer("POST", "/v1/yo/dispositivos", Dispositivo(token = token, modelo = Build.MODEL ?: "Android", version = BuildConfig.VERSION_NAME)) } }
    }

    override fun onMessageReceived(m: RemoteMessage) {
        val d = m.data.toMutableMap()
        m.notification?.let { n -> d.putIfAbsent("titulo", n.title.orEmpty()); d.putIfAbsent("cuerpo", n.body.orEmpty()) }
        if (d["tipo"] == "actividad") { ActividadTurno.desdePush(applicationContext, d); return }
        Push.mostrar(applicationContext, d)
        // Con la app abierta: la campanita y la pantalla al día.
        EnPrimerPlano.alAviso?.invoke()
    }

    @Serializable private data class Dispositivo(val plataforma: String = "android", val token: String, val entorno: String = "produccion", val modelo: String, val version: String)
}

/** Lo que hace la app abierta cuando llega un aviso (refrescar, como willPresent en iOS). */
object EnPrimerPlano { @Volatile var alAviso: (() -> Unit)? = null }

/** Las acciones de los avisos que no abren la app: Confirmar, Avisar a la lista, Responder la reseña y Atendido. */
class AccionesPush : BroadcastReceiver() {
    @Serializable private data class CambioEstado(val estado: String)
    @Serializable private data class Oferta(val servicioId: String, val profesionalId: String, val inicio: FechaApi)
    @Serializable private data class Respuesta(val respuesta: String)
    @Serializable private data class Resenas(val resenas: List<Resena> = emptyList())

    override fun onReceive(ctx: Context, i: Intent) {
        val que = i.action?.substringAfterLast('.') ?: return
        val turno = i.getStringExtra("turno") ?: return
        val aviso = i.getIntExtra("aviso", 0)
        val api = Push.api(ctx, i.getStringExtra("negocio")) ?: return
        val texto = RemoteInput.getResultsFromIntent(i)?.getCharSequence("respuesta")?.toString()?.trim()
        val pendiente = goAsync()
        CoroutineScope(Dispatchers.IO).launch {
            val ok = runCatching {
                when (que) {
                    "CONFIRMAR" -> api.hacer("PATCH", "/v1/panel/turnos/$turno", CambioEstado("confirmado"))
                    "ATENDIDO" -> { api.hacer("PATCH", "/v1/panel/turnos/$turno", CambioEstado("atendido")); ActividadTurno.terminar(ctx, turno) }
                    "AVISAR" -> {
                        val t: Turno = api.get("/v1/panel/turnos/$turno")
                        val s = t.servicioId; val p = t.profesionalId
                        if (s != null && p != null) api.hacer("POST", "/v1/panel/espera/ofertas", Oferta(s, p, t.inicio))
                    }
                    "RESPONDER" -> if (!texto.isNullOrEmpty()) {
                        val r: Resenas = api.get("/v1/panel/resenas")
                        r.resenas.firstOrNull { it.turnoId == turno }?.let { api.hacer("PATCH", "/v1/panel/resenas/${it.id}", Respuesta(texto)) }
                    }
                }
            }.isSuccess
            if (ok && aviso != 0) NotificationManagerCompat.from(ctx).cancel(aviso)
            ar.com.tengolibre.app.widgets.Widgets.actualizar(ctx)
            EnPrimerPlano.alAviso?.invoke()
            pendiente.finish()
        }
    }
}
