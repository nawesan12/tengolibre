package ar.com.tengolibre.app.tema

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import androidx.compose.ui.window.Popup
import ar.com.tengolibre.app.nucleo.Formato
import java.time.DayOfWeek
import java.time.LocalDate
import java.time.YearMonth
import java.time.format.DateTimeFormatter

// Las piezas del sistema que iOS da hechas (confirmationDialog, DatePicker, Menu) con el estilo de Tengo Libre.

data class OpcionConfirmacion(val titulo: String, val destructiva: Boolean = false, val accion: () -> Unit)

/** El diálogo de confirmación (confirmationDialog en iOS): título, mensaje, las opciones y «Volver». */
@Composable
fun Confirmacion(visible: Boolean, titulo: String, mensaje: String = "", opciones: List<OpcionConfirmacion>, alCerrar: () -> Unit, volver: String = "Volver") {
    if (!visible) return
    Dialog(onDismissRequest = alCerrar, properties = DialogProperties(usePlatformDefaultWidth = false)) {
        Column(Modifier.padding(horizontal = 12.dp).fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Column(Modifier.fillMaxWidth().clip(RoundedCornerShape(16.dp)).background(C.tarjeta)) {
                Column(Modifier.fillMaxWidth().padding(16.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(4.dp)) {
                    Texto(titulo, Tipo.lato(14f, Peso.Bold), C.sub, alinear = TextAlign.Center)
                    if (mensaje.isNotEmpty()) Texto(mensaje, Tipo.lato(13f), C.sub, alinear = TextAlign.Center)
                }
                opciones.forEach { o ->
                    Box(Modifier.fillMaxWidth().height(1.dp).background(C.linea))
                    Box(Modifier.fillMaxWidth().heightIn(min = 56.dp).clickable { o.accion(); alCerrar() }, contentAlignment = Alignment.Center) {
                        Texto(o.titulo, Tipo.lato(18f), if (o.destructiva) C.error else C.mentaTexto)
                    }
                }
            }
            Box(Modifier.fillMaxWidth().heightIn(min = 56.dp).clip(RoundedCornerShape(16.dp)).background(C.tarjeta).clickable(onClick = alCerrar), contentAlignment = Alignment.Center) {
                Texto(volver, Tipo.lato(18f, Peso.Bold), C.mentaTexto)
            }
        }
    }
}

/** Una opción de un menú desplegable (Menu en iOS). */
data class OpcionDeMenu(val titulo: String, val simbolo: String, val destructiva: Boolean = false, val accion: () -> Unit)

/** Un botón que abre un menú flotante con opciones (Menu { … } label: { … } en iOS). */
@Composable
fun MenuBoton(opciones: List<OpcionDeMenu>, descripcion: String, boton: @Composable () -> Unit) {
    var abierto by remember { mutableStateOf(false) }
    Box {
        Box(Modifier.clickable { abierto = true }.semantics { contentDescription = descripcion }) { boton() }
        if (abierto) Popup(alignment = Alignment.TopEnd, onDismissRequest = { abierto = false }) {
            Column(Modifier.padding(top = 48.dp).widthIn(min = 250.dp).tarjeta(14.dp)) {
                opciones.forEachIndexed { i, o ->
                    if (i > 0) Box(Modifier.fillMaxWidth().height(1.dp).background(C.linea))
                    Row(
                        Modifier.fillMaxWidth().heightIn(min = 48.dp).clickable { abierto = false; o.accion() }.padding(horizontal = 16.dp),
                        verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.SpaceBetween,
                    ) {
                        Texto(o.titulo, Tipo.lato(16f), if (o.destructiva) C.error else C.tinta)
                        Icono(o.simbolo, 17f, if (o.destructiva) C.error else C.tinta)
                    }
                }
            }
        }
    }
}

/** El calendario de un mes (DatePicker .graphical en iOS): flechas para cambiar de mes, la semana arranca el lunes,
 *  el día elegido en menta y los días anteriores a `minimo` apagados. */
@Composable
fun CalendarioMes(fecha: LocalDate, alElegir: (LocalDate) -> Unit, minimo: LocalDate? = null, maximo: LocalDate? = null) {
    var mes by remember { mutableStateOf(YearMonth.from(fecha)) }
    val titulo = DateTimeFormatter.ofPattern("MMMM yyyy", Formato.argentina).format(mes)
    val hoy = Formato.enArgentina(java.time.Instant.now()).toLocalDate()
    Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Texto(Formato.mayuscula(titulo), Tipo.lato(17f, Peso.Bold), C.tinta, Modifier.weight(1f))
            Box(Modifier.size(40.dp).clip(CircleShape).clickable { mes = mes.minusMonths(1) }.semantics { contentDescription = "Mes anterior" }, contentAlignment = Alignment.Center) { Icono("chevron.left", 18f, C.mentaTexto) }
            Box(Modifier.size(40.dp).clip(CircleShape).clickable { mes = mes.plusMonths(1) }.semantics { contentDescription = "Mes siguiente" }, contentAlignment = Alignment.Center) { Icono("chevron.right", 18f, C.mentaTexto) }
        }
        Row(Modifier.fillMaxWidth()) {
            listOf("LUN", "MAR", "MIÉ", "JUE", "VIE", "SÁB", "DOM").forEach { Texto(it, Tipo.lato(11f, Peso.Bold), C.sub, Modifier.weight(1f), alinear = TextAlign.Center) }
        }
        val primero = mes.atDay(1)
        val vacios = (primero.dayOfWeek.value - DayOfWeek.MONDAY.value + 7) % 7
        val celdas = List(vacios) { null } + (1..mes.lengthOfMonth()).map { mes.atDay(it) }
        celdas.chunked(7).forEach { fila ->
            Row(Modifier.fillMaxWidth()) {
                fila.forEach { d ->
                    Box(Modifier.weight(1f).aspectRatio(1f), contentAlignment = Alignment.Center) {
                        if (d != null) {
                            val habil = (minimo == null || !d.isBefore(minimo)) && (maximo == null || !d.isAfter(maximo))
                            val on = d == fecha
                            Box(
                                Modifier.size(40.dp).clip(CircleShape).background(if (on) C.menta else Color.Transparent)
                                    .clickable(enabled = habil) { alElegir(d); Haptico.seleccion() }.alpha(if (habil) 1f else 0.3f),
                                contentAlignment = Alignment.Center,
                            ) {
                                Texto("${d.dayOfMonth}", Tipo.lato(18f, if (on || d == hoy) Peso.Bold else Peso.Regular, numeros = true),
                                    if (on) C.petroleoHondo else if (d == hoy) C.mentaTexto else C.tinta)
                            }
                        }
                    }
                }
                repeat(7 - fila.size) { Spacer(Modifier.weight(1f)) }
            }
        }
    }
}

/** La hora (DatePicker .hourAndMinute en iOS): horas y minutos con − y + (de a 5 minutos). */
@Composable
fun SelectorHora(minutos: Int, alCambiar: (Int) -> Unit, paso: Int = 5) {
    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        @Composable
        fun boton(s: String, d: String, f: () -> Unit) = Box(Modifier.size(36.dp).clip(CircleShape).background(C.chip).clickable { f(); Haptico.seleccion() }.semantics { contentDescription = d }, contentAlignment = Alignment.Center) { Icono(s, 15f) }
        boton("minus", "Una hora menos") { alCambiar(((minutos - 60) + 1440) % 1440) }
        Texto("%d".format(minutos / 60), Tipo.lato(22f, Peso.Black, numeros = true), C.tinta, Modifier.width(32.dp), alinear = TextAlign.Center)
        boton("plus", "Una hora más") { alCambiar((minutos + 60) % 1440) }
        Texto(":", Tipo.lato(22f, Peso.Black), C.tinta)
        boton("minus", "Menos minutos") { alCambiar(((minutos - paso) + 1440) % 1440) }
        Texto("%02d".format(minutos % 60), Tipo.lato(22f, Peso.Black, numeros = true), C.tinta, Modifier.width(32.dp), alinear = TextAlign.Center)
        boton("plus", "Más minutos") { alCambiar((minutos + paso) % 1440) }
    }
}

/** Un Picker de iOS en un renglón: la opción elegida en menta con ⌃⌄ y un menú para cambiarla. */
@Composable
fun <T> SelectorMenu(opciones: List<Pair<T, String>>, elegido: T, alElegir: (T) -> Unit, descripcion: String = "") {
    val actual = opciones.firstOrNull { it.first == elegido }?.second ?: ""
    MenuBoton(opciones.map { (v, t) -> OpcionDeMenu(t, if (v == elegido) "checkmark" else "circle") { alElegir(v) } }, descripcion.ifEmpty { actual }) {
        Row(Modifier.heightIn(min = 44.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(4.dp)) {
            Texto(actual, Tipo.lato(16f), C.mentaTexto, lineas = 1)
            Icono("chevron.up.chevron.down", 12f, C.mentaTexto)
        }
    }
}

/** Un DatePicker compacto de iOS en un renglón: el título, la fecha en una pastilla y, al tocarla, el calendario. */
@Composable
fun RenglonFecha(titulo: String, fecha: LocalDate, alCambiar: (LocalDate) -> Unit, minimo: LocalDate? = null) {
    var abierto by remember { mutableStateOf(false) }
    Column(Modifier.fillMaxWidth().padding(horizontal = 16.dp)) {
        Row(Modifier.fillMaxWidth().heightIn(min = 52.dp), verticalAlignment = Alignment.CenterVertically) {
            Texto(titulo, Tipo.lato(16f), C.tinta, Modifier.weight(1f))
            Box(Modifier.clip(RoundedCornerShape(8.dp)).background(C.chip).clickable { abierto = !abierto }.padding(horizontal = 10.dp, vertical = 6.dp)) {
                Texto(DateTimeFormatter.ofPattern("d MMM yyyy", Formato.argentina).format(fecha), Tipo.lato(16f), if (abierto) C.mentaTexto else C.tinta)
            }
        }
        if (abierto) Box(Modifier.padding(bottom = 8.dp)) { CalendarioMes(fecha, { alCambiar(it); abierto = false }, minimo = minimo) }
    }
}

/** Un DatePicker de hora en un renglón (minutos desde la medianoche). */
@Composable
fun RenglonHora(titulo: String, minutos: Int, alCambiar: (Int) -> Unit) {
    Row(Modifier.fillMaxWidth().heightIn(min = 52.dp).padding(horizontal = 16.dp), verticalAlignment = Alignment.CenterVertically) {
        Texto(titulo, Tipo.lato(16f), C.tinta, Modifier.weight(1f))
        SelectorHora(minutos, alCambiar)
    }
}
