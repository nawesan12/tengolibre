package ar.com.tengolibre.app.widgets

import android.appwidget.AppWidgetManager
import android.content.ComponentName
import android.content.Context
import android.content.Intent
import android.net.Uri
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.DpSize
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.glance.ColorFilter
import androidx.glance.GlanceId
import androidx.glance.Image
import androidx.glance.ImageProvider
import androidx.glance.GlanceModifier
import androidx.glance.LocalSize
import androidx.glance.action.clickable
import androidx.glance.appwidget.GlanceAppWidget
import androidx.glance.appwidget.GlanceAppWidgetReceiver
import androidx.glance.appwidget.SizeMode
import androidx.glance.appwidget.action.actionStartActivity
import androidx.glance.appwidget.cornerRadius
import androidx.glance.appwidget.provideContent
import androidx.glance.appwidget.updateAll
import androidx.glance.background
import androidx.glance.layout.Alignment
import androidx.glance.layout.Box
import androidx.glance.layout.Column
import androidx.glance.layout.Row
import androidx.glance.layout.Spacer
import androidx.glance.layout.fillMaxHeight
import androidx.glance.layout.fillMaxSize
import androidx.glance.layout.fillMaxWidth
import androidx.glance.layout.height
import androidx.glance.layout.padding
import androidx.glance.layout.size
import androidx.glance.layout.width
import androidx.glance.text.FontWeight
import androidx.glance.text.Text
import androidx.glance.text.TextStyle
import androidx.glance.unit.ColorProvider
import ar.com.tengolibre.app.MainActivity
import ar.com.tengolibre.app.R
import ar.com.tengolibre.app.nucleo.FechaApi
import ar.com.tengolibre.app.nucleo.Formato
import ar.com.tengolibre.app.nucleo.Llavero
import ar.com.tengolibre.app.push.Push
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable
import java.time.Instant

// Clon de TengoLibreWidgets/Widgets.swift y de DiaChico (Compartido/Compartido.swift) con Glance: el próximo turno
// (chico) y el día (mediano). En los teléfonos Android no hay widgets de pantalla bloqueada: lo que en iOS va ahí lo
// muestra la notificación en curso (nucleo/Actividad.kt).

/** Un turno como lo ven los widgets. */
@Serializable data class TurnoChico(val id: String, val servicio: String = "", val profesional: String = "", val cliente: Cli = Cli(), val inicio: FechaApi, val fin: FechaApi, val estado: String = "") {
    @Serializable data class Cli(val nombre: String = "")
}

/** El día para los widgets: los turnos de hoy que siguen y cuántos quedan. */
data class DiaChico(val negocio: String, val turnos: List<TurnoChico>, val hechos: Int, val conSesion: Boolean) {
    val proximo: TurnoChico? get() = turnos.firstOrNull()

    companion object {
        val vacio = DiaChico("", emptyList(), 0, true)

        @Serializable private data class Lista(val turnos: List<TurnoChico> = emptyList())

        suspend fun cargar(ctx: Context): DiaChico {
            val api = Push.api(ctx) ?: return DiaChico("", emptyList(), 0, false)
            val hoy = Formato.diaISO(Instant.now())
            val l = runCatching { api.get<Lista>("/v1/panel/turnos", listOf("desde" to hoy, "hasta" to hoy)) }.getOrNull() ?: return vacio
            val ahora = Instant.now()
            val activos = l.turnos.filter { it.estado in listOf("pendiente_pago", "pendiente", "confirmado") }
            return DiaChico(Llavero.leer("negocioNombre").orEmpty(), activos.filter { it.fin.isAfter(ahora) }.sortedBy { it.inicio },
                l.turnos.count { it.estado == "atendido" }, true)
        }
    }
}

private object W {
    val petroleo = Color(0xFF183C44)
    val menta = Color(0xFF3CCC90)
    val blanco = Color.White
    fun c(x: Color) = ColorProvider(x)
    fun nombre(s: String) = s.substringBefore(' ')
}

object Widgets {
    private val alcance = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    /** Vuelve a dibujar los widgets (WidgetCenter.reloadAllTimelines en iOS). */
    fun actualizar(ctx: Context) { alcance.launch { runCatching { WidgetDia().updateAll(ctx.applicationContext) } } }

    /** Si hay un widget del día en la pantalla de inicio (para «Potenciá tu negocio»). */
    fun hay(ctx: Context): Boolean =
        runCatching { AppWidgetManager.getInstance(ctx).getAppWidgetIds(ComponentName(ctx, WidgetDiaReceiver::class.java)).isNotEmpty() }.getOrDefault(false)
}

/** «Tu día»: el próximo turno y lo que sigue. */
class WidgetDia : GlanceAppWidget() {
    companion object {
        private val CHICO = DpSize(110.dp, 110.dp)
        private val MEDIANO = DpSize(250.dp, 110.dp)
    }

    override val sizeMode = SizeMode.Responsive(setOf(CHICO, MEDIANO))

    override suspend fun provideGlance(context: Context, id: GlanceId) {
        val d = DiaChico.cargar(context)
        provideContent { Vista(context, d) }
    }

    private fun abrir(ctx: Context, ruta: String) =
        actionStartActivity(Intent(ctx, MainActivity::class.java).setAction(Intent.ACTION_VIEW).setData(Uri.parse("tengolibre://$ruta")))

    @Composable
    private fun Vista(ctx: Context, d: DiaChico) {
        val mediano = LocalSize.current.width >= MEDIANO.width
        val ruta = if (mediano) "hoy" else d.proximo?.let { "turno/${it.id}" } ?: "hoy"
        Box(GlanceModifier.fillMaxSize().background(W.petroleo).cornerRadius(22.dp).padding(14.dp).clickable(abrir(ctx, ruta))) {
            if (!mediano) Chico(d)
            else Row(GlanceModifier.fillMaxSize()) {
                Box(GlanceModifier.width(120.dp).fillMaxHeight()) { Chico(d) }
                Spacer(GlanceModifier.width(14.dp))
                Box(GlanceModifier.width(1.dp).fillMaxHeight().background(Color.White.copy(alpha = 0.2f))) {}
                Spacer(GlanceModifier.width(14.dp))
                Column(GlanceModifier.fillMaxSize()) {
                    Text("DESPUÉS", style = TextStyle(W.c(W.menta), 11.sp, FontWeight.Bold))
                    Spacer(GlanceModifier.height(8.dp))
                    if (d.turnos.size <= 1) Text("Nada más por hoy", style = TextStyle(W.c(W.blanco.copy(alpha = 0.8f)), 14.sp, FontWeight.Bold))
                    d.turnos.drop(1).take(3).forEach { t ->
                        Row(GlanceModifier.fillMaxWidth().padding(bottom = 6.dp).clickable(abrir(ctx, "turno/${t.id}"))) {
                            Text(Formato.horaDe(t.inicio), style = TextStyle(W.c(W.blanco), 14.sp, FontWeight.Bold))
                            Spacer(GlanceModifier.width(8.dp))
                            Text(W.nombre(t.cliente.nombre), style = TextStyle(W.c(W.blanco), 14.sp, FontWeight.Medium), maxLines = 1)
                        }
                    }
                    Spacer(GlanceModifier.defaultWeight())
                    Text("${d.turnos.size} por atender · ${d.hechos} atendidos", style = TextStyle(W.c(W.blanco.copy(alpha = 0.7f)), 11.sp))
                }
            }
        }
    }

    @Composable
    private fun Chico(d: DiaChico) {
        val t = d.proximo
        Column(GlanceModifier.fillMaxSize(), verticalAlignment = Alignment.Top) {
            when {
                !d.conSesion -> {
                    Image(ImageProvider(R.drawable.ic_calendario), "Tengo Libre", GlanceModifier.size(22.dp), colorFilter = ColorFilter.tint(W.c(W.menta)))
                    Spacer(GlanceModifier.defaultWeight())
                    Text("Entrá a Tengo Libre", style = TextStyle(W.c(W.blanco), 15.sp, FontWeight.Bold))
                }
                t != null -> {
                    Text(if (!t.inicio.isAfter(Instant.now())) "AHORA" else "PRÓXIMO", style = TextStyle(W.c(W.menta), 11.sp, FontWeight.Bold))
                    Text(Formato.horaDe(t.inicio), style = TextStyle(W.c(W.blanco), 30.sp, FontWeight.Bold))
                    Spacer(GlanceModifier.defaultWeight())
                    Text(t.cliente.nombre, style = TextStyle(W.c(W.blanco), 15.sp, FontWeight.Bold), maxLines = 1)
                    Text(t.servicio, style = TextStyle(W.c(W.blanco.copy(alpha = 0.75f)), 12.sp), maxLines = 1)
                }
                else -> {
                    Text("HOY", style = TextStyle(W.c(W.menta), 11.sp, FontWeight.Bold))
                    Spacer(GlanceModifier.defaultWeight())
                    Text("Todo libre", style = TextStyle(W.c(W.blanco), 22.sp, FontWeight.Bold))
                    Text(if (d.hechos > 0) "${d.hechos} atendidos" else "Sin turnos que vengan", style = TextStyle(W.c(W.blanco.copy(alpha = 0.75f)), 12.sp))
                }
            }
        }
    }
}

class WidgetDiaReceiver : GlanceAppWidgetReceiver() {
    override val glanceAppWidget: GlanceAppWidget = WidgetDia()
}
