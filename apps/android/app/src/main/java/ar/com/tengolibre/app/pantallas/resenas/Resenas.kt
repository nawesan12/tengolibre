package ar.com.tengolibre.app.pantallas.resenas

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.verticalScroll
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableDoubleStateOf
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import ar.com.tengolibre.app.app.AccionesTurno
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.nucleo.*
import ar.com.tengolibre.app.pantallas.hoy.Momentos
import ar.com.tengolibre.app.pantallas.principal.*
import ar.com.tengolibre.app.tema.*
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable
import java.time.Instant

// Clon de Pantallas/Resenas/Resenas.swift.

/** 16 Reseñas: el promedio con la distribución, filtros (Todas · Sin responder · Ocultas) y cada reseña con sus
 *  estrellas, «Respondiste» o el campo para responder, y mostrar u ocultar en la web. */
@Composable
fun Resenas() {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    var resenas by remember { mutableStateOf<List<Resena>>(emptyList()) }
    var promedio by remember { mutableDoubleStateOf(0.0) }
    var cantidad by remember { mutableIntStateOf(0) }
    var distribucion by remember { mutableStateOf<Map<String, Int>>(emptyMap()) }
    var filtro by remember { mutableStateOf("todas") }
    var cargando by remember { mutableStateOf(true) }
    var respuestas by remember { mutableStateOf<Map<String, String>>(emptyMap()) }
    var pagina by remember { mutableIntStateOf(0) }
    val prom = "%.1f".format(promedio).replace(".", ",")

    @Serializable data class Lista(val promedio: Double = 0.0, val cantidad: Int = 0, val resenas: List<Resena> = emptyList(), val distribucion: Map<String, Int>? = null)
    suspend fun cargar(mas: Boolean = false) {
        val p = if (mas) pagina + 1 else 0
        val q = buildList { add("pagina" to "$p"); if (filtro == "sin") add("sinResponder" to "1"); if (filtro == "ocultas") add("visible" to "no") }
        try {
            val r: Lista = estado.api.get("/v1/panel/resenas", q)
            resenas = if (mas) resenas + r.resenas else r.resenas
            if (filtro == "todas") { promedio = r.promedio; cantidad = r.cantidad; distribucion = r.distribucion ?: distribucion }
            pagina = p
        } catch (e: Exception) { if (!mas) estado.avisar(e) }
        cargando = false
    }
    LaunchedEffect(estado.version, filtro) { cargar() }

    fun responder(r: Resena) = alcance.launch {
        val texto = respuestas[r.id].orEmpty()
        try {
            estado.api.hacer("PATCH", "/v1/panel/resenas/${r.id}", AccionesTurno.Respuesta(texto))
            respuestas = respuestas - r.id
            resenas = resenas.map { if (it.id == r.id) it.copy(respuesta = texto, respondidaEn = Instant.now()) else it }
            Haptico.exito()
            Momentos.resenaBuena(estado, r)
        } catch (e: Exception) { estado.avisar(e) }
    }
    @Serializable data class Visible(val visible: Boolean)
    fun visible(r: Resena, v: Boolean) = alcance.launch {
        try { estado.api.hacer("PATCH", "/v1/panel/resenas/${r.id}", Visible(v)); resenas = resenas.map { if (it.id == r.id) it.copy(visible = v) else it } }
        catch (e: Exception) { estado.avisar(e) }
    }

    Column(Modifier.fillMaxSize().background(C.fondo).verticalScroll(rememberScrollState())) {
        Cabecera("Reseñas", if (cantidad == 0) "Todavía sin reseñas" else "$prom de 5 · ${Formato.cantidad(cantidad, "reseña", "reseñas")}", volver = if (estado.esProfesional) "" else "Más") {
            Segmentado(listOf("todas" to "Todas", "sin" to "Sin responder", "ocultas" to "Ocultas"), filtro, { filtro = it })
        }
        Cuerpo {
            if (cantidad > 0 && filtro == "todas") {
                Row(Modifier.entrada(0).fillMaxWidth().tarjeta().padding(16.dp), horizontalArrangement = Arrangement.spacedBy(16.dp), verticalAlignment = Alignment.CenterVertically) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(2.dp)) {
                        Texto(prom, Tipo.lato(40f, Peso.Black, numeros = true), C.tinta)
                        Estrellas(Math.round(promedio).toInt(), 14f)
                    }
                    Box(Modifier.weight(1f)) {
                        GraficoBarras((5 downTo 1).map { n -> FilaBarra("$n ★", distribucion["$n"] ?: 0) }, formato = { "$it" }, color = hex(0xE8A317))
                    }
                }
            }
            if (cargando && resenas.isEmpty()) repeat(3) { Esqueleto(140.dp, 20.dp) }
            else if (resenas.isEmpty()) VacioQueEnsena("Sin reseñas", if (filtro == "sin") "Respondiste todas" else "Todavía no hay reseñas",
                "Después de cada turno atendido le pedimos la reseña a la clienta. Las buenas se muestran en tu página.")
            else resenas.forEachIndexed { i, r ->
                if (r.id == resenas.last().id && resenas.size < cantidad && filtro == "todas") LaunchedEffect(r.id) { cargar(mas = true) }
                Column(Modifier.entrada(i + 1).fillMaxWidth().tarjeta().alpha(if (r.visible) 1f else 0.75f).padding(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Row(verticalAlignment = Alignment.Top, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                        Iniciales(r.nombre, 40.dp)
                        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                            Texto(r.nombre, Tipo.lato(16f, Peso.Bold), C.tinta)
                            Texto("${r.servicio} · ${r.profesional} · ${Formato.hace(r.creadoEn)}", Tipo.lato(12f), C.sub)
                        }
                        Estrellas(r.puntaje, 13f)
                    }
                    if (r.comentario.isNotEmpty()) Texto(r.comentario, Tipo.lato(15f).copy(lineHeight = 20.sp), C.tinta)
                    if (r.respuesta.isNotEmpty()) {
                        Column(Modifier.fillMaxWidth().clip(RoundedCornerShape(14.dp)).background(C.mentaSuave.copy(alpha = C.mentaSuave.alpha * 0.6f)).padding(12.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            TextoRotulo("Respondiste", C.mentaTexto)
                            Texto(r.respuesta, Tipo.lato(14f), C.tinta)
                        }
                    } else if (!estado.soloLectura) {
                        Row(verticalAlignment = Alignment.Bottom, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            Box(Modifier.weight(1f).clip(RoundedCornerShape(12.dp)).background(C.campo).padding(horizontal = 12.dp, vertical = 10.dp)) {
                                val t = respuestas[r.id].orEmpty()
                                if (t.isEmpty()) Texto("Respondé a ${r.nombre.split(" ").firstOrNull().orEmpty()}", Tipo.lato(15f), C.sub.copy(alpha = 0.7f))
                                val tinta = C.tinta
                                BasicTextField(t, { respuestas = respuestas + (r.id to it) }, Modifier.fillMaxWidth(), textStyle = Tipo.lato(15f).copy(color = tinta), maxLines = 5, cursorBrush = SolidColor(C.mentaTexto))
                            }
                            Box(Modifier.alpha(if (respuestas[r.id].orEmpty().isBlank()) 0.45f else 1f)) { BotonMentaChico("Responder", { if (respuestas[r.id].orEmpty().isNotBlank()) responder(r) }) }
                        }
                    }
                    if (estado.gestiona) Interruptor(if (r.visible) "Se ve en tu página" else "Oculta en tu página", r.visible, { visible(r, it) })
                }
            }
        }
    }
}

@Composable
fun Estrellas(puntaje: Int, tam: Float = 14f) {
    Row(Modifier.semantics(mergeDescendants = true) { contentDescription = "$puntaje de 5 estrellas" }, horizontalArrangement = Arrangement.spacedBy(2.dp)) {
        (1..5).forEach { i -> Icono(if (i <= puntaje) "star.fill" else "star", tam, if (i <= puntaje) hex(0xE8A317) else C.linea) }
    }
}
