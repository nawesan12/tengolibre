package ar.com.tengolibre.app.nucleo

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.unit.dp
import ar.com.tengolibre.app.tema.*
import java.time.Duration
import java.time.Instant

// Clon de Nucleo/Consejos.swift. En Android no hay TipKit: las mismas reglas a mano. Cada consejo aparece la
// primera vez que se ve el gesto, una vez y listo; nunca más de uno por hora; apagados en las pruebas.

data class Consejo(val id: String, val titulo: String, val mensaje: String, val simbolo: String, val regla: () -> Boolean = { true })

object Consejos {
    /** Apagados en los tests y en el modo prueba (Tips.hideAllTipsForTesting en iOS). */
    var apagados = false

    private const val FRECUENCIA_HORAS = 1L

    /** Cuántas veces se abrió Hoy (para el consejo del widget, a partir del tercer día de la semana). */
    fun abrioHoy() {
        val ahora = Instant.now()
        val previas = Preferencias.lista("consejo-abrioHoy").mapNotNull { runCatching { Instant.parse(it) }.getOrNull() }
            .filter { Duration.between(it, ahora).toDays() < 7 }
        Preferencias.ponerLista("consejo-abrioHoy", (previas + ahora).map { it.toString() }.takeLast(20))
    }

    private fun abiertoEnLaSemana(): Int = Preferencias.lista("consejo-abrioHoy").count {
        runCatching { Duration.between(Instant.parse(it), Instant.now()).toDays() < 7 }.getOrDefault(false)
    }

    fun puedeMostrar(c: Consejo): Boolean {
        if (apagados || ModoPrueba.activo) return false
        if (Preferencias.bool("consejo-${c.id}")) return false
        val ultimo = Preferencias.texto("consejo-ultimo")?.let { runCatching { Instant.parse(it) }.getOrNull() }
        val mostrandoEste = Preferencias.texto("consejo-actual") == c.id
        if (!mostrandoEste && ultimo != null && Duration.between(ultimo, Instant.now()).toHours() < FRECUENCIA_HORAS) return false
        return c.regla()
    }

    fun marcarVisto(c: Consejo) {
        Preferencias.ponerTexto("consejo-actual", c.id)
        if (Preferencias.texto("consejo-ultimo") == null || Preferencias.texto("consejo-actual") != c.id) Preferencias.ponerTexto("consejo-ultimo", Instant.now().toString())
    }

    fun cerrar(c: Consejo) {
        Preferencias.ponerBool("consejo-${c.id}", true)
        Preferencias.ponerTexto("consejo-actual", null)
        Preferencias.ponerTexto("consejo-ultimo", Instant.now().toString())
    }

    val deslizar = Consejo("deslizar", "Deslizá el turno", "Hacia la izquierda: atendido o no vino, sin abrirlo.", "hand.draw")
    val hueco = Consejo("hueco", "Tocá un hueco libre", "Cargás un turno ahí mismo, con la hora y la profesional ya puestas.", "hand.tap")
    val avisar = Consejo("avisar", "Hay gente esperando este lugar", "«Avisar» se lo ofrece de a una persona por vez. Si nadie lo toma, sigue la próxima.", "megaphone")
    val capsula = Consejo("capsula", "Pasá de profesional", "Deslizá la agenda o usá las flechas de abajo para ver a cada una o a todas juntas.", "person.2")
    val etiquetas = Consejo("etiquetas", "Etiquetá a tus clientas", "VIP, nueva, alérgica… después las filtrás en la lista.", "tag")
    val widget = Consejo("widget", "Tu día sin abrir la app", "Mantené apretada la pantalla de inicio y sumá el widget de Tengo Libre.", "apps.iphone") { abiertoEnLaSemana() >= 3 }
    val buscar = Consejo("buscar", "¿No encontrás algo?", "Escribí lo que querés hacer: «caja», «seña», «IA», o el nombre de una clienta.", "magnifyingglass")
}

/** Un consejo en tarjeta (TipView en iOS), con la cruz para cerrarlo. Si no corresponde, no ocupa lugar. */
@Composable
fun VistaConsejo(c: Consejo, modifier: Modifier = Modifier) {
    var visible by remember(c.id) { mutableStateOf(Consejos.puedeMostrar(c)) }
    if (!visible) return
    remember(c.id) { Consejos.marcarVisto(c); true }
    Row(modifier.fillMaxWidth().tarjeta(18.dp).padding(14.dp), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
        Icono(c.simbolo, 24f, C.mentaTexto)
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Texto(c.titulo, Tipo.lato(16f, Peso.Bold), C.tinta)
            Texto(c.mensaje, Tipo.lato(14f), C.sub)
        }
        Box(Modifier.size(28.dp).clip(CircleShape).clickable { Consejos.cerrar(c); visible = false }, contentAlignment = Alignment.Center) { Icono("xmark", 14f, C.sub) }
    }
}

/** Un consejo en globo arriba de una pieza, sólo si se cumple la condición (popoverTip en iOS). */
@Composable
fun ConConsejo(si: Boolean, c: Consejo, contenido: @Composable () -> Unit) {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        if (si) VistaConsejo(c)
        contenido()
    }
}
