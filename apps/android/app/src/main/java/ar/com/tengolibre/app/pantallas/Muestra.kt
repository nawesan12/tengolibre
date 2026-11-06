package ar.com.tengolibre.app.pantallas

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.unit.dp
import ar.com.tengolibre.app.tema.*

// Provisoria: las piezas del tema juntas, para comparar contra iOS mientras se porta el resto.
@Composable
fun Muestra() {
    var seg by remember { mutableStateOf("dia") }
    var on by remember { mutableStateOf(true) }
    var campo by remember { mutableStateOf("") }
    Column(Modifier.fillMaxSize().fondoPuntos().verticalScroll(rememberScrollState())) {
        Column(Modifier.fillMaxWidth().fondoPetroleo(C.cabecera, Offset(0.9f, 0f)).statusBarsPadding().padding(20.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                BotonRedondo("square.and.arrow.up", "Compartir", {}, sobrePetroleo = true)
                BotonRedondo("bell", "Avisos", {}, sobrePetroleo = true, numero = 3)
            }
            Texto("Hoy", Tipo.tituloGrande, androidx.compose.ui.graphics.Color.White)
            Segmentado(listOf("dia" to "Día", "semana" to "Semana"), seg, { seg = it })
        }
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(14.dp)) {
            Rotulo("Para atender")
            Hueco(Modifier.fillMaxWidth().height(56.dp)) { PastillaPapel { Texto("LIBRE · 45 MIN", Tipo.lato(13f, Peso.Black), C.huecoTinta) } }
            ListaTarjeta(listOf("calendar" to "Agenda", "person.2" to "Clientes", "star.bubble" to "Reseñas")) { (s, t) -> FilaLista(s, t, "Detalle", pendiente = t == "Clientes") }
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) { Chip("Todas", elegido = true); Chip("Frecuente", "star.fill"); PastillaEstado("Confirmado", "checkmark.circle", C.mentaSuave, C.mentaTexto) }
            Campo("Nombre", campo, { campo = it }, ayuda = "Rocío Vega")
            Interruptor("Recordatorios", on, { on = it }, "Un día antes", "bell")
            BotonMenta("Guardar · hoy 16:00", {})
            BotonContorno("Cancelar", {})
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) { Iniciales("Ana Gómez"); BotonMentaChico("Avisar", {}) }
            Esqueleto()
            VacioQueEnsena("SIN TURNOS HOY", "Todavía no hay turnos", "Compartí tu página para que reserven solas.") { BotonMenta("Compartir mi página", {}) }
            ErrorConSalida("No pudimos cargar el día", "Revisá la conexión y probá de nuevo.") {}
            VistaToast(Toast("checkmark", "Turno guardado", "Le avisamos a Rocío", "Deshacer")) {}
        }
    }
}
