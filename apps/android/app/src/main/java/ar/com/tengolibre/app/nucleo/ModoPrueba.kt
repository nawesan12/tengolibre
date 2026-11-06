package ar.com.tengolibre.app.nucleo

import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonObject
import okhttp3.Interceptor
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Protocol
import okhttp3.Response
import okhttp3.ResponseBody.Companion.toResponseBody
import java.time.Instant
import java.time.temporal.ChronoUnit

/** Modo prueba (sólo debug, clon de Nucleo/ModoPrueba.swift): la app arranca con sesión y una API simulada.
 *  Las respuestas salen de pruebas/respuestas.json, el mismo archivo que usa iOS. Nunca llega a Play (release). */
object ModoPrueba {
    /** Lo prende MainActivity con el extra «modoPrueba» (o los tests). */
    var activo = false
    var cliente = false
    /** El texto de respuestas.json (assets/pruebas en debug, recursos en los tests). */
    var respuestas: () -> String = { "{}" }

    /** Deja la app como recién entrada: modo, permisos y recorrido. */
    fun preparar(conRecorrido: Boolean = false) {
        if (!activo) return
        Preferencias.borrarTodo()
        Preferencias.ponerTexto("modo", if (cliente) "cliente" else "negocio")
        Preferencias.ponerBool("permisosVistos", true)
        Preferencias.ponerBool("recorridoVisto", !conRecorrido)
        Preferencias.ponerTexto("negocio", "n1")
        if (cliente) Preferencias.ponerLista("misTurnos", listOf("ABC123"))
    }

    fun api(): Api = Api("https://api.prueba", "prueba", OkHttpClient.Builder().addInterceptor(ApiDePrueba { respuestas() }).build())
}

/** La API simulada: respuestas fijas por «MÉTODO ruta» (o sólo ruta para GET). Lo que no está: GET → 404,
 *  lo demás → 200 con `{}`. `extra` pisa o suma respuestas (los tests de cada pantalla). */
class ApiDePrueba(private val extra: Map<String, String> = emptyMap(), private val texto: () -> String) : Interceptor {
    private val tabla: Map<String, String> by lazy { Datos.respuestas(texto()) + extra }

    override fun intercept(chain: Interceptor.Chain): Response {
        val req = chain.request()
        val ruta = req.url.encodedPath
        val metodo = req.method
        val cuerpo = tabla["$metodo $ruta"] ?: if (metodo == "GET") tabla[ruta] else "{}"
        val codigo = if (cuerpo == null) 404 else 200
        return Response.Builder().request(req).protocol(Protocol.HTTP_1_1).code(codigo).message(if (codigo == 200) "OK" else "No existe")
            .body((cuerpo ?: """{"status":404,"code":"no_existe","title":"No existe","detail":"No existe"}""").toResponseBody("application/json".toMediaType()))
            .build()
    }

    object Datos {
        private val marcador = Regex("""^\$(iso|dia)\((-?\d+)\)$""")

        /** Lee respuestas.json y reemplaza los marcadores de fecha relativos a ahora. */
        fun respuestas(texto: String, ahora: Instant = Instant.now()): Map<String, String> {
            val raiz = Api.json.parseToJsonElement(texto).jsonObject
            return raiz.filterKeys { !it.startsWith("_") }.mapValues { (_, v) -> Api.json.encodeToString(JsonElement.serializer(), fechas(v, ahora)) }
        }

        private fun fechas(e: JsonElement, ahora: Instant): JsonElement = when (e) {
            is JsonObject -> JsonObject(e.mapValues { fechas(it.value, ahora) })
            is JsonArray -> JsonArray(e.map { fechas(it, ahora) })
            is JsonPrimitive -> {
                val m = if (e.isString) marcador.find(e.contentOrNull.orEmpty()) else null
                if (m == null) e else {
                    val n = m.groupValues[2].toLong()
                    if (m.groupValues[1] == "iso") JsonPrimitive(Fecha.texto(ahora.plus(n, ChronoUnit.MINUTES).truncatedTo(ChronoUnit.SECONDS)))
                    else JsonPrimitive(Formato.diaISO(ahora.plus(n, ChronoUnit.DAYS)))
                }
            }
            JsonNull -> e
        }
    }
}
