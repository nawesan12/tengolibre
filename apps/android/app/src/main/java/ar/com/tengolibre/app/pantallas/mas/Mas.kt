package ar.com.tengolibre.app.pantallas.mas

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import ar.com.tengolibre.app.app.Estado
import ar.com.tengolibre.app.tema.*

// Clon de Pantallas/Mas/Mas.swift. Por ahora, las piezas comunes de las secciones (Bloque, Renglon, Numerito,
// RenglonInterruptor, RenglonCampo, RenglonPesos, BotonGuardar y Estado.intentar); Mas, PantallaMas, HojaEdicion,
// PrimerosPasos y Cuenta se suman en la fase 3 (ESTADO.md).

/** Una tarjeta con rótulo y renglones separados por línea. */
@Composable
fun Bloque(titulo: String = "", pie: String = "", vararg filas: @Composable () -> Unit) {
    Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(8.dp)) {
        if (titulo.isNotEmpty()) TextoRotulo(titulo, modifier = Modifier.padding(horizontal = 4.dp))
        Column(Modifier.fillMaxWidth().tarjeta()) {
            filas.forEachIndexed { i, f ->
                if (i > 0) Box(Modifier.padding(start = 16.dp).fillMaxWidth().height(1.dp).background(C.linea))
                f()
            }
        }
        if (pie.isNotEmpty()) Texto(pie, Tipo.lato(13f), C.sub, Modifier.padding(horizontal = 4.dp))
    }
}

/** Un renglón «título … valor» dentro de un Bloque. */
@Composable
fun Renglon(titulo: String, detalle: String = "", simbolo: String? = null, derecha: @Composable RowScope.() -> Unit = {}) {
    Row(Modifier.fillMaxWidth().heightIn(min = 56.dp).padding(horizontal = 16.dp, vertical = 10.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
        if (simbolo != null) Box(Modifier.width(26.dp), contentAlignment = Alignment.Center) { Icono(simbolo, 18f, C.mentaTexto) }
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Texto(titulo, Tipo.lato(16f), C.tinta)
            if (detalle.isNotEmpty()) Texto(detalle, Tipo.lato(13f), C.sub)
        }
        derecha()
    }
}

/** Un número que se cambia con − y + (minutos, horas, días, porcentajes). */
@Composable
fun Numerito(titulo: String, valor: Int, alCambiar: (Int) -> Unit, rango: IntRange, detalle: String = "", paso: Int = 1, unidad: (Int) -> String = { "$it" }) {
    Renglon(titulo, detalle) {
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
            val puedeMenos = valor > rango.first; val puedeMas = valor < rango.last
            Box(Modifier.size(36.dp).clip(CircleShape).background(C.chip).clickable(enabled = puedeMenos) { alCambiar(maxOf(rango.first, valor - paso)); Haptico.seleccion() }.alpha(if (puedeMenos) 1f else 0.4f), contentAlignment = Alignment.Center) { Icono("minus", 17f) }
            Texto(unidad(valor), Tipo.lato(15f, Peso.Bold, numeros = true), C.tinta, Modifier.widthIn(min = 64.dp), alinear = TextAlign.Center)
            Box(Modifier.size(36.dp).clip(CircleShape).background(C.chip).clickable(enabled = puedeMas) { alCambiar(minOf(rango.last, valor + paso)); Haptico.seleccion() }.alpha(if (puedeMas) 1f else 0.4f), contentAlignment = Alignment.Center) { Icono("plus", 17f) }
        }
    }
}

/** Un interruptor en un Bloque. */
@Composable
fun RenglonInterruptor(titulo: String, activo: Boolean, alCambiar: (Boolean) -> Unit, detalle: String = "") {
    Box(Modifier.padding(horizontal = 16.dp, vertical = 0.dp)) { Interruptor(titulo, activo, alCambiar, detalle) }
}

/** Un campo de texto dentro de un Bloque. */
@Composable
fun RenglonCampo(titulo: String, texto: String, alCambiar: (String) -> Unit, placeholder: String = "", teclado: KeyboardType = KeyboardType.Text, multilinea: Boolean = false) {
    val tinta = C.tinta
    Column(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 10.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
        Texto(titulo, Tipo.lato(13f, Peso.Bold), C.sub)
        Box {
            if (texto.isEmpty() && placeholder.isNotEmpty()) Texto(placeholder, Tipo.lato(16f), C.sub.copy(alpha = 0.6f))
            BasicTextField(texto, alCambiar, Modifier.fillMaxWidth(), textStyle = Tipo.lato(16f).copy(color = tinta),
                singleLine = !multilinea, minLines = if (multilinea) 2 else 1, maxLines = if (multilinea) 8 else 1,
                keyboardOptions = KeyboardOptions(keyboardType = teclado), cursorBrush = SolidColor(C.mentaTexto))
        }
    }
}

/** Un precio en pesos que se escribe como número. */
@Composable
fun RenglonPesos(titulo: String, valor: Int, alCambiar: (Int) -> Unit) {
    val tinta = C.tinta
    Renglon(titulo) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Texto("$", Tipo.lato(16f, Peso.Bold), C.sub)
            Spacer(Modifier.width(2.dp))
            BasicTextField(if (valor == 0) "" else "$valor", { t -> alCambiar(t.filter(Char::isDigit).take(9).toIntOrNull() ?: 0) },
                Modifier.widthIn(min = 40.dp, max = 120.dp), textStyle = Tipo.lato(16f, Peso.Bold, numeros = true).copy(color = tinta, textAlign = TextAlign.End),
                singleLine = true, keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number), cursorBrush = SolidColor(C.mentaTexto))
        }
    }
}

/** Botón de guardar fijo abajo de una hoja o sección. */
@Composable
fun BotonGuardar(accion: () -> Unit, texto: String = "Guardar", guardando: Boolean = false, habilitado: Boolean = true) {
    BotonMenta(if (guardando) "Guardando…" else texto, accion, alto = 52.dp, conSombra = false, habilitado = !guardando && habilitado)
}

/** Corre una escritura; si falla, el toast con el error. Devuelve si salió. */
suspend fun Estado.intentar(ok: String = "", f: suspend () -> Unit): Boolean = try {
    f()
    if (ok.isNotEmpty()) avisar(ok)
    true
} catch (e: kotlinx.coroutines.CancellationException) {
    throw e
} catch (e: Exception) {
    avisar(e)
    false
}

/** Lo que se apila dentro de «Más»: todas las secciones del panel web, nativas. */
sealed class DestinoMas {
    data object primerosPasos : DestinoMas(); data object resenas : DestinoMas(); data object suscripcion : DestinoMas()
    data object diasLibres : DestinoMas(); data object cobros : DestinoMas()
    data object servicios : DestinoMas(); data object equipo : DestinoMas(); data class profesional(val id: String) : DestinoMas(); data object miHorario : DestinoMas()
    data object membresias : DestinoMas(); data object cupones : DestinoMas(); data object referidos : DestinoMas(); data object penalidades : DestinoMas()
    data object etiquetas : DestinoMas(); data object caja : DestinoMas()
    data object reportes : DestinoMas(); data object miWeb : DestinoMas(); data object plantillas : DestinoMas(); data object dominio : DestinoMas()
    data object integraciones : DestinoMas()
    data object configuracion : DestinoMas(); data object avisosClientes : DestinoMas(); data object whatsapp : DestinoMas(); data object auditoria : DestinoMas()
    data object sucursales : DestinoMas(); data object cuenta : DestinoMas(); data object recorrido : DestinoMas()
}
