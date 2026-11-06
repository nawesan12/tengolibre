package ar.com.tengolibre.app.nucleo

import ar.com.tengolibre.app.BuildConfig
import kotlinx.serialization.json.JsonElement

/** Un JSON cualquiera: para editar un producto sin perder los campos que la app no muestra (ValorJSON en iOS). */
typealias ValorJSON = JsonElement

/** La web de Tengo Libre (tengolibre.com.ar): la página pública de cada negocio y la reserva. */
object Web {
    val base: String = BuildConfig.WEB.ifEmpty { "https://tengolibre.com.ar" }.trimEnd('/')

    /** La página de reservas de un negocio: tengolibre.com.ar/{slug}. */
    fun pagina(slug: String): String = "$base/$slug"

    /** Lo que se muestra del link, sin https://. */
    fun textoPagina(slug: String): String = "${base.removePrefix("https://").removePrefix("http://")}/$slug"
}
