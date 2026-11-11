package ar.com.tengolibre.app.app

import android.content.Context
import android.net.Uri
import android.os.Build
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import ar.com.tengolibre.app.BuildConfig
import ar.com.tengolibre.app.nucleo.*
import ar.com.tengolibre.app.pantallas.mas.DestinoMas
import ar.com.tengolibre.app.pantallas.mas.EstadoNegocio
import ar.com.tengolibre.app.tema.Haptico
import ar.com.tengolibre.app.tema.Toast
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable
import java.time.Instant

/** Adónde lleva un push, un widget o un link `tengolibre://` (clon de Ruta en App/Estado.swift). */
sealed class Ruta {
    data class turno(val id: String) : Ruta()
    data object hoy : Ruta()
    data object espera : Ruta()
    data object avisos : Ruta()
    data object resenas : Ruta()
    data object nuevoTurno : Ruta()
    data object suscripcion : Ruta()
    data object caja : Ruta()
    data object buscar : Ruta()
    data object agenda : Ruta()
    /** El turno de una clienta, por su código (link del mail o aviso al teléfono de la clienta). */
    data class turnoCliente(val codigo: String) : Ruta()
    /** Una invitación al equipo de un negocio. */
    data class invitacion(val token: String) : Ruta()
    data object clientes : Ruta()
    /** Una pantalla de Más (servicios, equipo, cobros, mi web…). */
    data class mas(val destino: DestinoMas) : Ruta()
    /** Un link de tengolibre.com.ar que la app no tiene: se abre la web dentro de la app. */
    data class web(val url: String) : Ruta()

    companion object {
        /** `turnos/{id}` (el `ruta` del push), `tengolibre://turno/{id}` o un App Link de tengolibre.com.ar (Vinculos). */
        fun de(texto: String): Ruta? {
            var t = texto
            val u = runCatching { Uri.parse(texto) }.getOrNull()
            val esquema = u?.scheme
            if (esquema == "tengolibre") {
                t = (listOf(u.host.orEmpty()) + u.pathSegments).joinToString("/")
            } else if (esquema == "https" || esquema == "http") {
                return Vinculos.ruta(u)
            }
            return deTexto(t)
        }

        /** Sin Uri (para los tests): sólo la forma `a/b`. */
        fun deTexto(t: String): Ruta? {
            val partes = t.split("/").filter { it.isNotEmpty() }
            val primera = partes.firstOrNull() ?: return null
            return when (primera) {
                "turnos", "turno" -> if (partes.size > 1) turno(partes[1]) else hoy
                "hoy" -> hoy
                "espera", "lista-espera" -> espera
                "notificaciones", "avisos" -> avisos
                "resenas" -> resenas
                "nuevo" -> nuevoTurno
                "suscripcion" -> suscripcion
                "caja" -> caja
                "buscar" -> buscar
                "agenda" -> agenda
                "turno-cliente" -> if (partes.size > 1) turnoCliente(partes[1]) else null
                "invitacion" -> if (partes.size > 1) invitacion(partes[1]) else null
                else -> null
            }
        }
    }
}

/** El estado de la app (clon de App/Estado.swift): la sesión, la persona, sus negocios y el elegido con su rol,
 *  el acceso (prueba / sólo lectura), la campanita y los avisos de arriba (toasts). Observable por Compose. */
class Estado(api: Api? = null, private val alcance: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Main)) {
    enum class Fase {
        arrancando, afuera,
        /** Elegir cómo la usa: atiende un negocio o reserva turnos (39). */
        eligiendoModo, eligiendoNegocio, permisos,
        /** Sin negocios: a crear uno (alta). */
        alta, adentro, cliente,
    }

    var fase by mutableStateOf(Fase.arrancando)
    val api: Api = api ?: (if (BuildConfig.DEBUG && ModoPrueba.activo) ModoPrueba.api() else Api(Api.urlConfigurada(), Llavero.leer("token")))
    var persona by mutableStateOf<Persona?>(null)
    var negocios by mutableStateOf<List<NegocioDeLaPersona>>(emptyList())
    var negocio by mutableStateOf<NegocioDeLaPersona?>(null)
    var acceso by mutableStateOf<Acceso?>(null)
    var sinLeer by mutableIntStateOf(0)
    var toast by mutableStateOf<Toast?>(null)
    var rutaPendiente by mutableStateOf<Ruta?>(null)
    /** El turno de una clienta para abrir (link o aviso), y una invitación al equipo para aceptar: los abre Raiz,
     *  en cualquier modo y fase (la invitación espera al ingreso si hace falta). */
    var turnoClienteAbierto by mutableStateOf<String?>(null)
    var invitacionPendiente by mutableStateOf<String?>(null)
    /** Para que las listas se vuelvan a pedir después de un cambio. */
    var version by mutableIntStateOf(0)
    /** Sin conexión: se ve lo guardado (handoff 23). */
    var sinConexion by mutableStateOf(false)
    var guardadoEn by mutableStateOf<Instant?>(null)
    /** Ingresando con Clerk (la Custom Tab está abierta): el botón muestra la espera. */
    var ingresando by mutableStateOf(false)
    /** Lo que salió mal al volver de Clerk: Ingreso lo muestra abajo de los botones, como en iOS. */
    var errorIngreso by mutableStateOf("")
    /** El SSE del panel (en vivo). */
    val enVivo = EnVivo(alcance)
    /** Qué tiene conectado y prendido el negocio (Más, Hoy, «Potenciá tu negocio»). */
    val vivo = EstadoNegocio()
    var tokenPush: String? = null
        private set
    private var toastTarea: Job? = null
    private var deshacer: (suspend () -> Unit)? = null
    /** Se llama al cambiar algo que muestran los widgets. */
    var alRecargarWidgets: () -> Unit = {}
    /** Abre una URL en una Custom Tab (la pone MainActivity). */
    var abrirPestana: (String) -> Unit = {}
    /** La hoja de compartir del sistema con un texto (UIActivityViewController en iOS). */
    var compartirTexto: (String) -> Unit = {}
    /** Abre un link externo (tel:, whatsapp:, mailto:, https:) con la app que corresponda. */
    var abrirLink: (String) -> Unit = {}

    val rol: String get() = negocio?.rol.orEmpty()
    val esDuena: Boolean get() = rol == "duenio"
    val gestiona: Boolean get() = rol == "duenio" || rol == "encargado"
    val esProfesional: Boolean get() = rol == "profesional"
    /** Recepción no ve precios ni servicios en Más. */
    val vePrecios: Boolean get() = rol != "recepcion"
    val soloLectura: Boolean get() = acceso?.bloqueado ?: false

    var permisosVistos: Boolean
        get() = Preferencias.bool("permisosVistos")
        set(v) = Preferencias.ponerBool("permisosVistos", v)
    /** Cómo la usa: «negocio» o «cliente» (se cambia desde Cuenta / Más). */
    var modo: String
        get() = Preferencias.texto("modo").orEmpty()
        set(v) = Preferencias.ponerTexto("modo", v)
    private var negocioGuardado: String?
        get() = Preferencias.texto("negocio")
        set(v) = Preferencias.ponerTexto("negocio", v)

    init {
        this.api.negocio = Preferencias.texto("negocio")
        this.api.alPerderSesion = { sesionPerdida() }
    }

    fun recargarWidgets() = alRecargarWidgets()

    // MARK: Arranque

    suspend fun arrancar() {
        if (api.token == null) { fase = Fase.afuera; return }
        try {
            cargarSesion()
            despuesDeEntrar()
        } catch (e: ErrorApi) {
            if (e.sinConexion) {
                sinConexion = true
                persona = Guardado.leer<Persona>("persona")?.first
                negocios = Guardado.leer<List<NegocioDeLaPersona>>("negocios")?.first ?: emptyList()
                negocio = negocios.firstOrNull { it.id == negocioGuardado } ?: negocios.firstOrNull()
                fase = if (persona == null) Fase.afuera else if (modo == "cliente") Fase.cliente else Fase.adentro
            } else {
                olvidarSesion(); fase = Fase.afuera
            }
        } catch (e: kotlinx.coroutines.CancellationException) {
            throw e
        } catch (e: Exception) {
            olvidarSesion(); fase = Fase.afuera
        }
    }

    suspend fun cargarSesion() {
        val p: Persona = api.get("/v1/auth/persona")
        val y: Yo = api.get("/v1/yo")
        persona = p
        negocios = y.negocios
        Guardado.guardar(p, "persona")
        Guardado.guardar(y.negocios, "negocios")
        (negocios.firstOrNull { it.id == negocioGuardado } ?: negocios.firstOrNull())?.let { usar(it) }
        sinConexion = false
    }

    /** Después de entrar o de arrancar con sesión: adónde va. */
    private suspend fun despuesDeEntrar() {
        if (modo.isEmpty()) { fase = Fase.eligiendoModo; return }
        if (modo == "cliente") { fase = Fase.cliente; return }
        if (negocios.isEmpty()) { fase = Fase.alta; return }
        if (negocios.size > 1 && negocioGuardado == null) { fase = Fase.eligiendoNegocio; return }
        if (!permisosVistos) { fase = Fase.permisos; return }
        fase = Fase.adentro
        refrescar()
        pedirTokenPushSiCorresponde()
    }

    suspend fun elegirModo(m: String) {
        modo = m
        despuesDeEntrar()
    }

    /** Trabajar con otro negocio (o sucursal): todo lo guardado era del anterior. */
    fun usar(n: NegocioDeLaPersona) {
        if (negocio?.id != n.id) Guardado.borrarTodo()
        negocio = n
        negocioGuardado = n.id
        api.negocio = n.id
        // Para los widgets (leen el llavero).
        Llavero.guardar(n.id, "negocio")
        Llavero.guardar(n.nombre, "negocioNombre")
        recargarWidgets()
        version += 1
    }

    suspend fun entrarANegocio(n: NegocioDeLaPersona) {
        usar(n)
        despuesDeEntrar()
    }

    suspend fun terminarPermisos() {
        permisosVistos = true
        despuesDeEntrar()
    }

    @Serializable data class SinLeer(val sinLeer: Int = 0)

    /** El acceso (prueba, sólo lectura) y la campanita. */
    suspend fun refrescar() {
        if (negocio == null) return
        val a: Acceso? = runCatching { api.get<Acceso>("/v1/panel/acceso") }.getOrNull()
        val c: SinLeer? = runCatching { api.get<SinLeer>("/v1/panel/campanita/sin-leer") }.getOrNull()
        if (a != null) acceso = a
        if (c != null) sinLeer = c.sinLeer
    }

    // MARK: Ingreso

    @Serializable private data class CuerpoClerk(val token: String)
    @Serializable private data class RespuestaClerk(val token: String, val persona: Persona)

    /** El token de Clerk (Apple o Google) → sesión propia de la API (POST /v1/auth/clerk). */
    suspend fun ingresarCon(tokenClerk: String) {
        val r: RespuestaClerk = api.enviar("POST", "/v1/auth/clerk", CuerpoClerk(tokenClerk))
        api.token = r.token
        Llavero.guardar(r.token, "token")
        cargarSesion()
        despuesDeEntrar()
    }

    private var clerk: Clerk? = null

    /** Apple o Google: abre la Custom Tab; la vuelta (ar.com.tengolibre.app://callback) llega a [vueltaClerk].
     *  En Android no hay hoja nativa de Apple: los dos van por redirect, como en Caigo. */
    suspend fun ingresarCon(proveedor: String, contexto: Context) {
        val c = Clerk(contexto, api.http, BuildConfig.CLERK).takeIf { it.disponible } ?: throw Clerk.ErrorClerk("sin_clave")
        clerk = c
        errorIngreso = ""
        ingresando = true
        try {
            abrirPestana(c.iniciar(proveedor))
        } catch (e: Exception) {
            ingresando = false
            throw e
        }
    }

    suspend fun ingresarConApple(contexto: Context) = ingresarCon("apple", contexto)
    suspend fun ingresarConGoogle(contexto: Context) = ingresarCon("google", contexto)

    /** La vuelta de Clerk con el nonce. */
    fun vueltaClerk(nonce: String?, contexto: Context) {
        val c = clerk ?: Clerk(contexto, api.http, BuildConfig.CLERK)
        alcance.launch {
            try {
                ingresarCon(c.completar(nonce))
                Haptico.exito()
            } catch (e: Clerk.ErrorClerk) {
                if (e.codigo != "cancelado") { errorIngreso = "No pudimos entrar. Probá de nuevo."; Haptico.error() }
            } catch (e: ErrorApi) {
                errorIngreso = e.detalle; Haptico.error()
            } catch (e: kotlinx.coroutines.CancellationException) {
                throw e
            } catch (e: Exception) {
                errorIngreso = "No pudimos entrar. Probá de nuevo."; Haptico.error()
            } finally {
                ingresando = false
            }
        }
    }

    suspend fun salir() {
        runCatching { api.hacer("POST", "/v1/auth/salir") }
        olvidarSesion()
        fase = Fase.afuera
    }

    private fun olvidarSesion() {
        api.token = null
        Llavero.borrar("token")
        Guardado.borrarTodo()
        persona = null
        negocios = emptyList()
        negocio = null
        acceso = null
    }

    private fun sesionPerdida() {
        olvidarSesion()
        fase = Fase.afuera
        avisar(Toast("lock", "Tu sesión venció", "Entrá de nuevo para seguir."))
    }

    /** Abre una ruta: las del panel quedan pendientes para Principal; el turno de una clienta y las invitaciones
     *  las abre Raiz. */
    fun abrir(r: Ruta) {
        when (r) {
            is Ruta.turnoCliente -> turnoClienteAbierto = r.codigo
            is Ruta.invitacion -> invitacionPendiente = r.token
            is Ruta.web -> abrirPestana(r.url)
            else -> rutaPendiente = r
        }
    }

    // MARK: Avisos de arriba

    fun avisar(t: Toast, deshacer: (suspend () -> Unit)? = null) {
        toastTarea?.cancel()
        this.deshacer = deshacer
        toast = t
        if (t.error) Haptico.error()
        toastTarea = alcance.launch {
            delay(if (deshacer == null) 3500 else 5000)
            toast = null
        }
    }

    fun avisar(titulo: String, texto: String = "", ok: Boolean = true) =
        avisar(Toast(if (ok) "checkmark" else "exclamationmark", titulo, texto, error = !ok))

    fun avisar(e: Throwable) {
        val x = e as? ErrorApi ?: ErrorApi.desconocido
        avisar(Toast(if (x.sinConexion) "icloud.slash" else "exclamationmark", if (x.sinConexion) "Sin conexión" else x.titulo, x.detalle, error = true))
    }

    fun tocarToast() {
        val d = deshacer
        deshacer = null
        toast = null
        if (d != null) alcance.launch { d(); version += 1 }
    }

    // MARK: Push (FCM; el registro lo hace push/Push.kt)

    /** Se completa con Push: si el permiso ya está dado, registra el token. */
    var alPedirTokenPush: suspend () -> Unit = {}
    suspend fun pedirTokenPushSiCorresponde() = alPedirTokenPush()

    /** Modo clienta: el permiso de avisos se pide una vez (para los recordatorios de sus turnos). */
    var alPedirAvisosClienta: suspend () -> Unit = {}
    suspend fun pedirAvisosClienta() = alPedirAvisosClienta()

    /** Pide el permiso de notificaciones (Android 13+); lo resuelve MainActivity. */
    var alPedirPermisoPush: suspend () -> Boolean = { true }
    suspend fun pedirPermisoPush(): Boolean = alPedirPermisoPush()

    @Serializable private data class SeguirTurno(val token: String, val plataforma: String = "android", val entorno: String = "produccion")
    @Serializable private data class SoloToken(val token: String, val plataforma: String = "android")

    /** Anota este teléfono en cada turno (por su código) para que le lleguen sus avisos. Una vez por turno y token. */
    suspend fun seguirTurnos(codigos: List<String>) {
        val token = tokenPush ?: return
        val clave = "seguidos-${token.take(16)}"
        val hechos = Preferencias.lista(clave).toMutableSet()
        for (c in codigos) if (c !in hechos) {
            if (runCatching { api.hacer("POST", "/v1/t/$c/dispositivo", SeguirTurno(token)) }.isSuccess) hechos.add(c)
        }
        Preferencias.ponerLista(clave, hechos.toList())
    }

    /** Deja de mandarle a este teléfono los avisos de un turno (lo sacó de su lista). */
    suspend fun dejarDeSeguir(codigo: String) {
        val token = tokenPush ?: return
        runCatching { api.hacer("DELETE", "/v1/t/$codigo/dispositivo", SoloToken(token)) }
    }

    @Serializable private data class Dispositivo(val plataforma: String = "android", val token: String, val entorno: String = "produccion", val modelo: String, val version: String)

    suspend fun registrarTokenPush(token: String) {
        if (token == tokenPush || api.token == null) return
        tokenPush = token
        runCatching { api.hacer("POST", "/v1/yo/dispositivos", Dispositivo(token = token, modelo = Build.MODEL ?: "Android", version = BuildConfig.VERSION_NAME)) }
    }
}
