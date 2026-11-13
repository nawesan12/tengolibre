package ar.com.tengolibre.app.nucleo

import ar.com.tengolibre.app.BuildConfig
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.withContext
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import okhttp3.Call
import okhttp3.Callback
import okhttp3.HttpUrl.Companion.toHttpUrl
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.Response
import java.io.IOException
import java.util.UUID
import java.util.concurrent.TimeUnit
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException

/** Un error de la API (application/problem+json) o de la red, con lo que hay que decirle a la persona. */
data class ErrorApi(
    val estado: Int,
    val codigo: String,
    val titulo: String,
    val detalle: String,
    /** Ingreso con contraseña: intentos que quedan antes de pausarlo. */
    val restantes: Int? = null,
    /** Segundos para volver a probar (429). */
    val reintentarEn: Int? = null,
) : Exception(detalle) {
    val sinConexion: Boolean get() = codigo == "sin_conexion"
    val sinSesion: Boolean get() = estado == 401 && codigo == "sin_sesion"

    companion object {
        val sinRed = ErrorApi(0, "sin_conexion", "Sin conexión", "No hay internet. Revisá la conexión y probá de nuevo.")
        val desconocido = ErrorApi(0, "interno", "Algo salió mal", "Probá de nuevo en un rato.")
    }
}

/** Lo que hay que decirle a la persona de cualquier error. */
fun mensajeDe(e: Throwable): String = (e as? ErrorApi)?.detalle ?: ErrorApi.desconocido.detalle

private suspend fun Call.esperar(): Response = suspendCancellableCoroutine { cont ->
    enqueue(object : Callback {
        override fun onFailure(call: Call, e: IOException) = cont.resumeWithException(e)
        override fun onResponse(call: Call, response: Response) = cont.resume(response)
    })
    cont.invokeOnCancellation { runCatching { cancel() } }
}

/** El cliente de la API de Tengo Libre (clon de Nucleo/Api.swift). Va directo contra api.tengolibre.com.ar con la
 *  sesión propia en `Authorization: Bearer` y, en /v1/panel, el negocio elegido en `X-Negocio`. */
class Api(val base: String, var token: String? = null, val http: OkHttpClient = cliente()) {
    /** El negocio con el que se trabaja (el de arriba de Hoy y Agenda). */
    @Volatile var negocio: String? = null
    /** Se llama cuando la API dice que la sesión ya no vale (401 sin_sesion): la app vuelve al ingreso. */
    var alPerderSesion: (suspend () -> Unit)? = null

    @Serializable
    private data class Problema(
        val status: Int? = null, val code: String? = null, val title: String? = null, val detail: String? = null,
        val restantes: Int? = null, val issues: List<Campo>? = null,
    ) {
        @Serializable data class Campo(val path: String = "", val message: String = "")
    }

    suspend fun pedir(metodo: String, ruta: String, query: List<Pair<String, String>> = emptyList(), cuerpo: ByteArray? = null, tipo: String = "application/json"): String {
        val url = (base.trimEnd('/') + ruta).toHttpUrl().newBuilder().apply { query.forEach { (k, v) -> addQueryParameter(k, v) } }.build()
        val r = Request.Builder().url(url).header("Accept", "application/json")
        token?.let { r.header("Authorization", "Bearer $it") }
        negocio?.let { if (ruta.startsWith("/v1/panel/")) r.header("X-Negocio", it) }
        val cuerpoHttp = cuerpo?.toRequestBody(tipo.toMediaType()) ?: if (metodo in listOf("POST", "PUT", "PATCH")) ByteArray(0).toRequestBody(null) else null
        r.method(metodo, cuerpoHttp)
        val (codigo, texto, reintentar) = try {
            withContext(Dispatchers.IO) {
                http.newCall(r.build()).esperar().use { Triple(it.code, it.body?.string().orEmpty(), it.header("Retry-After")) }
            }
        } catch (e: CancellationException) {
            throw e
        } catch (e: IOException) {
            throw ErrorApi.sinRed
        }
        if (codigo in 200..299) return texto
        val p = runCatching { json.decodeFromString<Problema>(texto) }.getOrElse { Problema() }
        val e = ErrorApi(codigo, p.code ?: "interno", p.title ?: "Algo salió mal",
            p.issues?.firstOrNull()?.message ?: p.detail ?: p.title ?: "Probá de nuevo en un rato.", p.restantes, reintentar?.toIntOrNull())
        if (e.sinSesion && token != null) alPerderSesion?.invoke()
        throw e
    }

    /** Sube una imagen (POST /v1/panel/archivos, multipart con el campo «archivo») y devuelve su URL. */
    suspend fun subirImagen(datos: ByteArray, nombre: String = "imagen.jpg"): String {
        val limite = "tl-${UUID.randomUUID()}"
        val b = ("--$limite\r\nContent-Disposition: form-data; name=\"archivo\"; filename=\"$nombre\"\r\nContent-Type: image/jpeg\r\n\r\n").toByteArray() +
            datos + "\r\n--$limite--\r\n".toByteArray()
        @Serializable data class Archivo(val id: String, val url: String)
        return decodificar<Archivo>(pedir("POST", "/v1/panel/archivos", cuerpo = b, tipo = "multipart/form-data; boundary=$limite")).url
    }

    /** Un pedido largo (SSE de /v1/panel/eventos): sin límite de tiempo y con la sesión y el negocio. */
    fun flujo(ruta: String): Pair<Request, OkHttpClient> {
        val r = Request.Builder().url(base.trimEnd('/') + ruta).header("Accept", "text/event-stream")
        token?.let { r.header("Authorization", "Bearer $it") }
        negocio?.let { r.header("X-Negocio", it) }
        return r.build() to http.newBuilder().readTimeout(0, TimeUnit.MILLISECONDS).build()
    }

    suspend inline fun <reified T> get(ruta: String, query: List<Pair<String, String>> = emptyList()): T = decodificar(pedir("GET", ruta, query))

    suspend inline fun <reified T, reified B> enviar(metodo: String, ruta: String, cuerpo: B): T =
        decodificar(pedir(metodo, ruta, cuerpo = json.encodeToString(cuerpo).toByteArray()))

    /** Un DELETE que responde con datos (p. ej. borrar los datos de una clienta: borrado o anonimizado). */
    suspend inline fun <reified T> borrar(ruta: String): T = decodificar(pedir("DELETE", ruta))

    /** Un pedido sin respuesta (204). */
    suspend inline fun <reified B> hacer(metodo: String, ruta: String, cuerpo: B) {
        pedir(metodo, ruta, cuerpo = json.encodeToString(cuerpo).toByteArray())
    }

    suspend fun hacer(metodo: String, ruta: String) { pedir(metodo, ruta) }

    inline fun <reified T> decodificar(d: String): T = try {
        json.decodeFromString<T>(d)
    } catch (e: Exception) {
        if (BuildConfig.DEBUG) android.util.Log.w("Api", "no pude leer ${T::class.simpleName}: $e")
        throw ErrorApi.desconocido
    }

    companion object {
        /** La API de la configuración (BuildConfig.API, de tl.api en local.properties). */
        fun urlConfigurada(): String = BuildConfig.API.ifEmpty { "https://api.tengolibre.com.ar" }

        val json = Json { ignoreUnknownKeys = true; explicitNulls = false; coerceInputValues = true; encodeDefaults = true }

        fun cliente(): OkHttpClient = OkHttpClient.Builder()
            .connectTimeout(20, TimeUnit.SECONDS).readTimeout(20, TimeUnit.SECONDS).writeTimeout(60, TimeUnit.SECONDS)
            .build()
    }
}

/** Un GET que puede fallar sin cortar todo (try? en iOS). */
suspend inline fun <reified T> Api.opcional(ruta: String, query: List<Pair<String, String>> = emptyList()): T? =
    try { get<T>(ruta, query) } catch (e: kotlinx.coroutines.CancellationException) { throw e } catch (e: Exception) { null }
