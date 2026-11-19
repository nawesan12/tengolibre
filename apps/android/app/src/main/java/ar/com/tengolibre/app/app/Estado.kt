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
        eligiendoNegocio, permisos,
        /** Sin negocios: a crear uno (alta). */
        alta, adentro,
    }

    var fase by mutableStateOf(Fase.arrancando)
    val api: Api = api ?: (if (BuildConfig.DEBUG && ModoPrueba.activo) ModoPrueba.api() else Api(Api.urlConfigurada(), Llavero.leer("token")))
    var persona by mutableStateOf<Persona?>(null)
    var negocios by mutableStateOf<List<NegocioDeLaPersona>>(emptyList())
    var negocio by mutableStateOf<NegocioDeLaPersona?>(null)
    var acceso by mutableStateOf<Acceso?>(null)
    var sinLeer by mutableIntStateOf(0)
    var toast by mutableStateOf<Toast?>(null)
    /** Sube para cerrar las hojas abiertas (Hoja lo escucha): un aviso de tope lleva a Suscripción, que está debajo. */
    var cerrarHojas by mutableStateOf(0)
    var rutaPendiente by mutableStateOf<Ruta?>(null)
    /** Una invitación al equipo para aceptar: la abre Raiz, en cualquier fase (espera al ingreso si hace falta). */
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
    /** Cuándo se volvieron a pedir el rol y el acceso por un 402/403 (una vez cada 30 s, sin bucles). */
    private var accesoRevisadoEn: Instant? = null
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
    private var negocioGuardado: String?
        get() = Preferencias.texto("negocio")
        set(v) = Preferencias.ponerTexto("negocio", v)

    init {
        olvidarModoCliente()
        this.api.negocio = Preferencias.texto("negocio")
        this.api.alPerderSesion = { sesionPerdida() }
        this.api.alNegarAcceso = { alcance.launch { accesoNegado() } }
    }

    fun recargarWidgets() = alRecargarWidgets()

    // MARK: Arranque

    suspend fun arrancar() {
        // Turnos que este teléfono seguía como clienta (versiones viejas): se dejan de seguir aunque la persona no
        // llegue a Principal (el token se pide en cualquier fase, si ya hay permiso). Como iOS.
        if (Preferencias.hayListaConPrefijo("seguidos-")) runCatching { pedirTokenPushSiCorresponde() }
        if (api.token == null) { fase = Fase.afuera; return }
        try {
            cargarSesion()
            despuesDeEntrar()
        } catch (e: kotlinx.coroutines.CancellationException) {
            throw e
        } catch (e: Exception) {
            val x = e as? ErrorApi
            if (x != null && (x.estado == 401 || x.codigo == "cuenta_borrada")) {
                // La sesión se cierra sólo si la API dice que ya no vale (401) o que la cuenta está borrada.
                olvidarSesion(); fase = Fase.afuera
                if (x.codigo == "cuenta_borrada") avisar(x)
            } else {
                // Cualquier otro error (sin red, un 500, un 502 durante un deploy, una respuesta que no se pudo leer): se
                // entra con lo guardado y se avisa, sin perder la sesión.
                entrarConLoGuardado()
                if (x?.sinConexion != true) avisar(Toast("exclamationmark", "No pudimos actualizar", "Ves lo último guardado. Probá de nuevo en un rato.", error = true))
            }
        }
    }

    /** Sin poder hablar con la API al abrir: lo último que se guardó, con el aviso de «sin conexión». Si no hay nada
     *  guardado queda en el ingreso, pero la sesión sigue (se reintenta al volver a abrir). */
    private fun entrarConLoGuardado() {
        sinConexion = true
        persona = Guardado.leer<Persona>("persona")?.first
        negocios = Guardado.leer<List<NegocioDeLaPersona>>("negocios")?.first ?: emptyList()
        negocio = negocios.firstOrNull { it.id == negocioGuardado } ?: negocios.firstOrNull()
        fase = if (persona == null) Fase.afuera else if (negocios.isEmpty()) Fase.alta else Fase.adentro
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
        if (negocios.isEmpty()) { fase = Fase.alta; return }
        if (negocios.size > 1 && negocioGuardado == null) { fase = Fase.eligiendoNegocio; return }
        if (!permisosVistos) { fase = Fase.permisos; return }
        fase = Fase.adentro
        refrescar()
        pedirTokenPushSiCorresponde()
    }

    /** La app es sólo para quien atiende un negocio: las versiones viejas tenían un modo clienta («modo», «misTurnos»).
     *  Se borra al abrir, y quien lo tenía va a su negocio o al alta. */
    private fun olvidarModoCliente() {
        Preferencias.ponerTexto("modo", null)
        Preferencias.ponerLista("misTurnos", emptyList())
        Preferencias.ponerBool("avisosClientaPedidos", false)
    }

    /** Trabajar con otro negocio (o sucursal): lo guardado del anterior se borra (la persona y sus negocios, recién
     *  guardados al entrar, siguen). Al abrir, el anterior es el que quedó elegido. */
    fun usar(n: NegocioDeLaPersona) {
        val anterior = negocio?.id ?: negocioGuardado
        if (anterior != null && anterior != n.id) Guardado.borrarDelNegocio()
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

    /** Un pedido del panel volvió con 402 o 403: el rol, la membresía o el sólo lectura cambiaron. Se vuelven a pedir
     *  /v1/yo y el acceso, como mucho una vez cada 30 s (los propios pedidos de acá también pueden dar 403 y no tienen
     *  que volver a llamar). */
    suspend fun accesoNegado() {
        val r = accesoRevisadoEn
        if (r != null && Instant.now().isBefore(r.plusSeconds(30))) return
        accesoRevisadoEn = Instant.now()
        if (api.token == null) return
        val negocioAntes = negocio; val bloqueadoAntes = acceso?.bloqueado
        val y = api.opcional<Yo>("/v1/yo")
        if (y != null) {
            negocios = y.negocios
            Guardado.guardar(y.negocios, "negocios")
            val actual = negocio?.id
            if (actual != null) {
                val n = negocios.firstOrNull { it.id == actual }
                if (n != null) {
                    negocio = n
                } else if (negocios.isNotEmpty()) {
                    // Ya no es parte de ese negocio: se trabaja con otro de los suyos.
                    usar(negocios.first())
                } else {
                    negocio = null; fase = Fase.alta; return
                }
            }
        }
        refrescar()
        // Las pantallas se vuelven a pedir sólo si algo cambió (un 403 de una pantalla que ese rol no ve no las recarga).
        if (negocioAntes != negocio || bloqueadoAntes != acceso?.bloqueado) version += 1
    }

    // MARK: Ingreso

    @Serializable private data class CuerpoClerk(val token: String)
    @Serializable private data class RespuestaClerk(val token: String, val persona: Persona)

    /** El token de Clerk (Apple, Google o mail) → sesión propia de la API (POST /v1/auth/clerk). Si la cuenta está borrada,
     *  la API responde 409 `cuenta_borrada` con el detalle que se muestra. */
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
        dejarDeSeguirTodos()
        runCatching { api.hacer("POST", "/v1/auth/salir") }
        olvidarSesion()
        fase = Fase.afuera
    }

    /** Borra la cuenta (DELETE /v1/auth/cuenta, 204): se desactiva ahora y los datos personales se borran a los 30 días.
     *  Después se olvida la sesión y se vuelve al ingreso. */
    suspend fun borrarCuenta() {
        api.hacer("DELETE", "/v1/auth/cuenta")
        dejarDeSeguirTodos()
        olvidarSesion()
        fase = Fase.afuera
        avisar(Toast("checkmark", "Tu cuenta quedó desactivada", "Tus datos personales se borran a los 30 días."))
    }

    /** Todo lo de la sesión: también el token de avisos (si no, al volver a entrar no se registra de nuevo), el negocio
     *  elegido y los permisos vistos. */
    private fun olvidarSesion() {
        api.token = null
        api.negocio = null
        Llavero.borrar("token")
        Guardado.borrarTodo()
        persona = null
        negocios = emptyList()
        negocio = null
        acceso = null
        tokenPush = null
        negocioGuardado = null
        permisosVistos = false
        accesoRevisadoEn = null
    }

    private fun sesionPerdida() {
        olvidarSesion()
        fase = Fase.afuera
        avisar(Toast("lock", "Tu sesión venció", "Entrá de nuevo para seguir."))
    }

    /** Abre una ruta: las del panel quedan pendientes para Principal; las invitaciones las abre Raiz y la web, el navegador. */
    fun abrir(r: Ruta) {
        when (r) {
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
        // Salir de una pantalla mientras guarda cancela la tarea: no es un error para mostrar (igual que iOS).
        if (e is kotlinx.coroutines.CancellationException) return
        val x = e as? ErrorApi ?: ErrorApi.desconocido
        // Algo que el plan no incluye o un tope (402 tope_plan): el detalle dice a qué plan pasar, y el botón lleva a
        // Suscripción para cambiarlo.
        if (x.topePlan) {
            avisar(Toast("arrow.up", x.titulo, x.detalle, accion = "Ver planes", error = true)) {
                cerrarHojas += 1
                delay(400)
                abrir(Ruta.suscripcion)
            }
            return
        }
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

    /** Pide el permiso de notificaciones (Android 13+); lo resuelve MainActivity. */
    var alPedirPermisoPush: suspend () -> Boolean = { true }
    suspend fun pedirPermisoPush(): Boolean = alPedirPermisoPush()

    /** DELETE /v1/t/{codigo}/dispositivo sólo acepta el token (turnos/rutas.go, dejarDeSeguir). */
    @Serializable private data class SoloToken(val token: String)

    /** Los turnos que este teléfono seguía como clienta (versiones viejas, con modo clienta): deja de recibir sus avisos.
     *  Se hace al registrar el token, al salir y al borrar la cuenta; si falla la red, igual se olvida la lista. */
    private suspend fun dejarDeSeguirTodos() {
        val token = tokenPush ?: return
        val clave = "seguidos-${token.take(16)}"
        for (c in Preferencias.lista(clave)) runCatching { api.hacer("DELETE", "/v1/t/$c/dispositivo", SoloToken(token)) }
        Preferencias.ponerLista(clave, emptyList())
    }

    @Serializable private data class Dispositivo(val plataforma: String = "android", val token: String, val entorno: String = "produccion", val modelo: String, val version: String)

    suspend fun registrarTokenPush(token: String) {
        if (token == tokenPush) return
        tokenPush = token
        // Sin sesión (afuera) igual se dejan de seguir los turnos viejos: no pide sesión. El token no se guarda:
        // al entrar llega el mismo y tiene que registrarse.
        if (api.token == null) { dejarDeSeguirTodos(); tokenPush = null; return }
        runCatching { api.hacer("POST", "/v1/yo/dispositivos", Dispositivo(token = token, modelo = Build.MODEL ?: "Android", version = BuildConfig.VERSION_NAME)) }
        dejarDeSeguirTodos()
    }

    /** FCM cambió el token (onNewToken): el viejo se da de baja en la API (DELETE /v1/yo/dispositivos borra los de esta
     *  sesión) y se registra el nuevo. Los turnos que seguía el viejo como clienta (versiones viejas) se dejan de seguir. */
    suspend fun cambiarTokenPush(nuevo: String) {
        val viejo = tokenPush
        if (nuevo == viejo || api.token == null) return
        if (viejo != null) runCatching { api.hacer("DELETE", "/v1/yo/dispositivos") }
        tokenPush = null
        registrarTokenPush(nuevo)
        if (viejo != null) {
            val clave = "seguidos-${viejo.take(16)}"
            val seguidos = Preferencias.lista(clave)
            for (c in seguidos) runCatching { api.hacer("DELETE", "/v1/t/$c/dispositivo", SoloToken(viejo)) }
            Preferencias.ponerLista(clave, emptyList())
        }
    }
}
