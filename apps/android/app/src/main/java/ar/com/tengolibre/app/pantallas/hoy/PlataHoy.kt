package ar.com.tengolibre.app.pantallas.hoy

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.nucleo.Formato
import ar.com.tengolibre.app.nucleo.opcional
import ar.com.tengolibre.app.pantallas.mas.DestinoMas
import ar.com.tengolibre.app.pantallas.mas.ResumenCaja
import ar.com.tengolibre.app.pantallas.principal.LocalNavegacion
import ar.com.tengolibre.app.tema.*
import kotlinx.serialization.Serializable
import java.time.DayOfWeek
import java.time.temporal.TemporalAdjusters
import kotlin.math.abs
import kotlin.math.roundToInt

// Clon de Pantallas/Hoy/PlataHoy.swift.

@Serializable private data class Resumen(val ingresos: Int = 0, val ocupacion: Double? = null)
@Serializable private data class DiaR(val fecha: String, val ingresos: Int = 0)
@Serializable private data class Rep(val resumen: Resumen = Resumen(), val porDia: List<DiaR>? = null)

/** La plata, a la vista en Hoy: lo cobrado hoy y la semana contra la anterior. Tocar abre la caja o los reportes.
 *  Se vuelve a pedir con cada cambio (los cobros y pagos llegan por el SSE). */
@Composable
fun PlataHoy() {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    if (!estado.vePrecios || estado.esProfesional) return
    var caja by remember { mutableStateOf<ResumenCaja?>(null) }
    var semana by remember { mutableStateOf<Int?>(null) }
    var antes by remember { mutableStateOf<Int?>(null) }
    var ocupacion by remember { mutableStateOf<Double?>(null) }
    var dias by remember { mutableStateOf<List<Int>>(emptyList()) }

    LaunchedEffect(estado.version) {
        val hoy = Formato.enArgentina(java.time.Instant.now()).toLocalDate()
        val lunes = hoy.with(TemporalAdjusters.previousOrSame(DayOfWeek.MONDAY))
        val transcurridos = java.time.temporal.ChronoUnit.DAYS.between(lunes, hoy)
        val lunesAntes = lunes.minusDays(7)
        val hastaAntes = lunesAntes.plusDays(transcurridos)
        caja = estado.api.opcional("/v1/panel/caja", listOf("fecha" to hoy.toString()))
        val r: Rep? = estado.api.opcional("/v1/panel/reportes", listOf("desde" to lunes.toString(), "hasta" to hoy.toString()))
        semana = r?.resumen?.ingresos
        dias = (r?.porDia ?: emptyList()).sortedBy { it.fecha }.map { it.ingresos }
        ocupacion = r?.resumen?.ocupacion
        // La semana anterior, hasta el mismo día (lunes a hoy contra lunes a «hoy» de la otra semana).
        val a: Rep? = estado.api.opcional("/v1/panel/reportes", listOf("desde" to lunesAntes.toString(), "hasta" to hastaAntes.toString()))
        antes = a?.resumen?.ingresos
    }

    val textoHoy = caja.let { c ->
        when {
            c == null -> "Trayendo la caja…"
            c.total == 0 -> "Cobrá desde cada turno"
            else -> buildList { add(Formato.cantidad(c.cobros, "cobro", "cobros")); if (c.debe > 0) add("deben ${Formato.pesos(c.debe)}") }.joinToString(" · ")
        }
    }
    val s = semana; val a = antes
    val textoSemana = when {
        s == null -> ""
        a == null || a <= 0 -> ocupacion?.let { "${Formato.porcentaje(it)} ocupado" } ?: "Primera semana"
        else -> { val d = (s - a).toDouble() / a * 100; "${if (d >= 0) "▲" else "▼"} ${abs(d).roundToInt()} % vs. la anterior" }
    }
    val colorSemana = if (s != null && a != null && a > 0) (if (s >= a) C.menta else hex(0xFFB4A8)) else Color.White.copy(alpha = 0.75f)

    Row(Modifier.fillMaxWidth().height(IntrinsicSize.Min).clip(RoundedCornerShape(22.dp)).fondoPetroleo(C.bloquePetroleo)) {
        Column(
            Modifier.weight(1f).clickable { nav.pilaHoy.add(DestinoMas.caja) }.padding(16.dp)
                .semantics(mergeDescendants = true) { contentDescription = "Cobrado hoy: ${caja?.let { Formato.pesos(it.total) } ?: "nada todavía"}. Ver la caja" },
            verticalArrangement = Arrangement.spacedBy(2.dp),
        ) {
            Texto("COBRADO HOY", Tipo.lato(11f, Peso.Black), C.menta, tracking = 0.8.sp)
            NumeroAnimado(caja?.let { Formato.pesos(it.total) } ?: "—", Tipo.lato(26f, Peso.Black, numeros = true), Color.White)
            Texto(textoHoy, Tipo.lato(12f), Color.White.copy(alpha = 0.75f), lineas = 1)
        }
        Box(Modifier.padding(vertical = 14.dp).width(1.dp).fillMaxHeight().background(Color.White.copy(alpha = 0.18f)))
        Column(
            Modifier.weight(1f).clickable { nav.pilaHoy.add(DestinoMas.reportes) }.padding(16.dp)
                .semantics(mergeDescendants = true) { contentDescription = "Esta semana: ${semana?.let { Formato.pesos(it) } ?: "sin datos"}. $textoSemana. Ver reportes" },
            verticalArrangement = Arrangement.spacedBy(2.dp),
        ) {
            Texto("ESTA SEMANA", Tipo.lato(11f, Peso.Black), C.menta, tracking = 0.8.sp)
            NumeroAnimado(semana?.let { Formato.pesos(it) } ?: "—", Tipo.lato(26f, Peso.Black, numeros = true), Color.White)
            if (dias.size > 1) Sparkline(dias, modifier = Modifier.fillMaxWidth().height(22.dp).padding(vertical = 2.dp))
            Texto(textoSemana, Tipo.lato(12f, Peso.Bold), colorSemana, lineas = 1)
        }
    }
}
