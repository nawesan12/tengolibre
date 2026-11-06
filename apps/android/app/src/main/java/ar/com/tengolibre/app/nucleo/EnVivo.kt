package ar.com.tengolibre.app.nucleo

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import kotlinx.serialization.Serializable
import ar.com.tengolibre.app.app.Estado

/** En vivo (clon de Nucleo/EnVivo.swift): escucha el SSE de la API (`GET /v1/panel/eventos`, el mismo que usa la
 *  web) mientras la app está abierta. Cada cambio sube `estado.version` y las pantallas se vuelven a pedir solas;
 *  si se corta, reconecta a los 1, 2, 4… hasta 30 s. */
class EnVivo(private val alcance: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Main)) {
    @Serializable data class Evento(val negocio: String = "", val tipo: String = "", val id: String = "", val op: String = "")

    private var tarea: Job? = null
    private var pendiente: Job? = null
    private var negocio: String? = null
    @Volatile private var llamada: okhttp3.Call? = null
    private var estado: Estado? = null

    fun arrancar(estado: Estado) {
        val n = estado.negocio?.id
        if (n == null || estado.api.token == null) { parar(); return }
        if (tarea != null && negocio == n) return
        parar()
        this.estado = estado
        negocio = n
        tarea = alcance.launch {
            var espera = 1L
            while (isActive) {
                val (pedido, http) = estado.api.flujo("/v1/panel/eventos")
                runCatching {
                    withContext(Dispatchers.IO) {
                        // Al parar (o cambiar de negocio) se corta la conexión en vez de esperar la próxima línea.
                        val llamada = http.newCall(pedido).also { this@EnVivo.llamada = it }
                        llamada.execute().use { r ->
                            if (r.code != 200) error("estado ${r.code}")
                            withContext(Dispatchers.Main) { espera = 1; estado.sinConexion = false }
                            val fuente = r.body!!.source()
                            var tipo = ""
                            while (isActive) {
                                val linea = fuente.readUtf8Line() ?: break
                                if (linea.startsWith("event:")) tipo = linea.drop(6).trim()
                                else if (linea.startsWith("data:")) {
                                    val e = runCatching { Api.json.decodeFromString<Evento>(linea.drop(5).trim()) }.getOrNull()
                                    val t = e?.tipo?.ifEmpty { null } ?: tipo
                                    withContext(Dispatchers.Main) { llego(t) }
                                }
                            }
                        }
                    }
                }
                if (!isActive) return@launch
                // Se cortó: vuelve a los 1, 2, 4… hasta 30 s.
                delay(espera * 1000)
                espera = minOf(30, espera * 2)
            }
        }
    }

    fun parar() {
        tarea?.cancel()
        tarea = null
        llamada?.cancel()
        llamada = null
        negocio = null
    }

    /** Junta los avisos que llegan seguidos (un cobro toca turno, caja y pago a la vez). */
    private fun llego(tipo: String) {
        val estado = estado ?: return
        pendiente?.cancel()
        val campanita = tipo == "notificaciones"
        pendiente = alcance.launch {
            delay(350)
            estado.version += 1
            if (campanita) estado.refrescar()
            estado.recargarWidgets()
        }
    }

    companion object {
        /** Lee las líneas de un SSE: (tipo, data). Para los tests. */
        fun parsear(texto: String): List<Pair<String, Evento?>> {
            var tipo = ""
            return texto.lines().mapNotNull { l ->
                when {
                    l.startsWith("event:") -> { tipo = l.drop(6).trim(); null }
                    l.startsWith("data:") -> tipo to runCatching { Api.json.decodeFromString<Evento>(l.drop(5).trim()) }.getOrNull()
                    else -> null
                }
            }
        }
    }
}
