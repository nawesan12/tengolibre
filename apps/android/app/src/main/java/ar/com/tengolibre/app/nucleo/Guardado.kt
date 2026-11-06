package ar.com.tengolibre.app.nucleo

import android.content.Context
import java.io.File
import java.net.URLEncoder
import java.time.Instant

/** Lo último que se trajo de cada lista, para verlo sin conexión («Ves lo guardado hoy, 14:20»).
 *  Un archivo por clave en la caché, con la fecha en que se guardó (clon de Guardado en Api.swift). */
object Guardado {
    internal var contexto: Context? = null
    private val carpeta: File? get() = contexto?.let { File(it.cacheDir, "guardado").apply { mkdirs() } }
    private fun archivo(clave: String) = carpeta?.let { File(it, URLEncoder.encode(clave, "UTF-8")) }

    inline fun <reified T> guardar(valor: T, clave: String) {
        val texto = runCatching { Api.json.encodeToString(valor) }.getOrNull() ?: return
        escribir(clave, texto)
    }

    fun escribir(clave: String, texto: String) { runCatching { archivo(clave)?.writeText(texto) } }

    inline fun <reified T> leer(clave: String): Pair<T, Instant>? {
        val (texto, fecha) = crudo(clave) ?: return null
        return runCatching { Api.json.decodeFromString<T>(texto) to fecha }.getOrNull()
    }

    fun crudo(clave: String): Pair<String, Instant>? {
        val f = archivo(clave)?.takeIf { it.exists() } ?: return null
        return runCatching { f.readText() to Instant.ofEpochMilli(f.lastModified()) }.getOrNull()
    }

    /** Al cerrar sesión o cambiar de negocio no queda nada del anterior. */
    fun borrarTodo() { carpeta?.deleteRecursively() }
}
