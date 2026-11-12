package ar.com.tengolibre.app.pantallas.hoy

import androidx.compose.animation.core.Animatable
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.gestures.detectHorizontalDragGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.offset
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
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Rect
import androidx.compose.ui.geometry.RoundRect
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Outline
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.PathEffect
import androidx.compose.ui.graphics.PathOperation
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.semantics.CustomAccessibilityAction
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.customActions
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.Density
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.LayoutDirection
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.Popup
import ar.com.tengolibre.app.nucleo.Dia
import ar.com.tengolibre.app.nucleo.EstadoTurno
import ar.com.tengolibre.app.nucleo.Formato
import ar.com.tengolibre.app.nucleo.Turno
import ar.com.tengolibre.app.tema.*
import kotlinx.coroutines.launch
import java.time.Instant
import kotlin.math.abs
import kotlin.math.roundToInt

// Clon de Pantallas/Hoy/Talon.swift.

/** El talón: troquel a 76 dp del borde izquierdo, medialunas de 8 dp del color del fondo y una línea punteada de
 *  2 dp [4,4] (handoff › Firmas de la marca). */
class TalonShape(private val troquelX: Dp = 76.dp, private val radio: Dp = Radio.talon, private val muesca: Dp = 8.dp) : Shape {
    override fun createOutline(size: Size, layoutDirection: LayoutDirection, density: Density): Outline {
        val r = with(density) { radio.toPx() }; val m = with(density) { muesca.toPx() }; val x = with(density) { troquelX.toPx() }
        val base = Path().apply { addRoundRect(RoundRect(Rect(Offset.Zero, size), r, r)) }
        val arriba = Path().apply { addOval(Rect(Offset(x - m, -m), Size(m * 2, m * 2))) }
        val abajo = Path().apply { addOval(Rect(Offset(x - m, size.height - m), Size(m * 2, m * 2))) }
        val p = Path().apply { op(base, arriba, PathOperation.Difference) }
        val q = Path().apply { op(p, abajo, PathOperation.Difference) }
        return Outline.Generic(q)
    }
}

/** La línea punteada del troquel. */
@Composable
fun Troquel(color: Color = C.linea, vertical: Boolean = true, modifier: Modifier = Modifier) {
    Canvas(if (vertical) modifier.width(2.dp).fillMaxHeight() else modifier.fillMaxWidth().height(2.dp)) {
        val efecto = PathEffect.dashPathEffect(floatArrayOf(4.dp.toPx(), 4.dp.toPx()))
        if (vertical) drawLine(color, Offset(1.dp.toPx(), 0f), Offset(1.dp.toPx(), size.height), 2.dp.toPx(), pathEffect = efecto)
        else drawLine(color, Offset(0f, 1.dp.toPx()), Offset(size.width, 1.dp.toPx()), 2.dp.toPx(), pathEffect = efecto)
    }
}

/** Una opción del menú que sale al mantener apretado (contextMenu en iOS). */
data class OpcionMenu(val titulo: String, val simbolo: String, val destructiva: Boolean = false, val accion: () -> Unit)

/** El menú contextual: una tarjeta flotante con las opciones. */
@Composable
fun MenuContextual(visible: Boolean, opciones: List<OpcionMenu>, alCerrar: () -> Unit) {
    if (!visible) return
    Popup(alignment = Alignment.TopCenter, onDismissRequest = alCerrar) {
        Column(Modifier.padding(top = 8.dp).widthIn(min = 240.dp).tarjeta(14.dp)) {
            opciones.forEachIndexed { i, o ->
                if (i > 0) Box(Modifier.fillMaxWidth().height(1.dp).background(C.linea))
                Row(
                    Modifier.fillMaxWidth().heightIn(min = 48.dp).clickable { alCerrar(); o.accion() }.padding(horizontal = 16.dp),
                    verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.SpaceBetween,
                ) {
                    Texto(o.titulo, Tipo.lato(16f), if (o.destructiva) C.error else C.tinta)
                    Icono(o.simbolo, 17f, if (o.destructiva) C.error else C.tinta)
                }
            }
        }
    }
}

/** Un turno como talón. A la izquierda la hora (900, 19) y la duración; a la derecha el nombre, el servicio y la
 *  pastilla de estado. Con letra de accesibilidad, la hora va arriba. */
@OptIn(ExperimentalFoundationApi::class)
@Composable
fun VistaTalon(
    turno: Turno,
    mostrarProfesional: Boolean = true,
    ahora: Instant = Instant.now(),
    alTocar: () -> Unit = {},
    alDeslizar: ((String) -> Unit)? = null,
) {
    val est = turno.visible(ahora)
    val nuevo = turno.esNuevo(ahora)
    val tinta = if (nuevo) C.petroleoHondo else C.tinta
    val sub = if (nuevo) hex(0x1D4A3A) else C.sub
    val grande = LocalDensity.current.fontScale >= 1.6f
    val forma = TalonShape(if (grande) (-20).dp else 76.dp)
    val alcance = rememberCoroutineScope()
    val densidad = LocalDensity.current
    val corrimiento = remember { Animatable(0f) }
    var paso by remember { mutableStateOf(false) }
    var menu by remember { mutableStateOf(false) }
    val estado = ar.com.tengolibre.app.app.LocalEstado.current
    val umbral = with(densidad) { 100.dp.toPx() }; val abierto = with(densidad) { 164.dp.toPx() }; val tope = with(densidad) { 190.dp.toPx() }

    fun cerrarYHacer(e: String) {
        alcance.launch { corrimiento.animateTo(0f, Movimiento.entrar()) }
        paso = false
        alDeslizar?.invoke(e)
    }

    val opciones = buildList {
        add(OpcionMenu("Ver turno", "doc.text", accion = alTocar))
        if (alDeslizar != null) {
            if (EstadoTurno.puedePasar(turno.estado, "confirmado") && turno.estado != "confirmado") add(OpcionMenu("Confirmar", "checkmark") { alDeslizar("confirmado") })
            if (EstadoTurno.puedePasar(turno.estado, "atendido")) add(OpcionMenu("Atendido", "checkmark.circle") { alDeslizar("atendido") })
            if (EstadoTurno.puedePasar(turno.estado, "ausente")) add(OpcionMenu("No vino", "person.slash") { alDeslizar("ausente") })
            if (EstadoTurno.puedePasar(turno.estado, "cancelado")) add(OpcionMenu("Cancelar", "calendar.badge.minus", destructiva = true) { alDeslizar("cancelado") })
        }
        val tel = turno.cliente.telefono.filter { it in "+0123456789" }
        // abrirLink no se cae en un equipo sin marcador (tablets): avisa «Este equipo no puede llamar».
        if (tel.isNotEmpty()) add(OpcionMenu("Llamar", "phone") { estado.abrirLink("tel:$tel") })
    }

    Box(
        Modifier.fillMaxWidth().semantics {
            contentDescription = "${Formato.horaDe(turno.inicio)}, ${turno.cliente.nombre}, ${turno.servicio}. ${est.texto}"
            customActions = listOf(CustomAccessibilityAction("Ver turno") { alTocar(); true }) + if (alDeslizar != null) listOf(
                CustomAccessibilityAction("Marcar atendido") { alDeslizar("atendido"); true },
                CustomAccessibilityAction("Marcar ausente") { alDeslizar("ausente"); true },
            ) else emptyList()
        },
        contentAlignment = Alignment.CenterEnd,
    ) {
        // Deslizar: sigue el dedo; pasado el umbral, háptico de selección y quedan las dos acciones.
        if (alDeslizar != null && corrimiento.value < 0) {
            Row(Modifier.matchParentSize(), horizontalArrangement = Arrangement.spacedBy(6.dp, Alignment.End)) {
                Column(
                    Modifier.width(76.dp).fillMaxHeight().clip(RoundedCornerShape(18.dp)).background(C.menta).clickable { cerrarYHacer("atendido") },
                    horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(4.dp, Alignment.CenterVertically),
                ) { Icono("checkmark.circle", 22f, C.petroleoHondo); Texto("Atendido", Tipo.lato(13f, Peso.Black), C.petroleoHondo) }
                Column(
                    Modifier.width(76.dp).fillMaxHeight().clip(RoundedCornerShape(18.dp)).background(C.petroleo).clickable { cerrarYHacer("ausente") },
                    horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(4.dp, Alignment.CenterVertically),
                ) { Icono("person.slash", 22f, Color.White); Texto("Ausente", Tipo.lato(13f, Peso.Bold), Color.White) }
            }
        }
        Box(
            Modifier.offset { IntOffset(corrimiento.value.roundToInt(), 0) }
                .then(if (alDeslizar == null) Modifier else Modifier.pointerInput(turno.id) {
                    detectHorizontalDragGestures(
                        onDragEnd = { alcance.launch { corrimiento.animateTo(if (paso) -abierto else 0f, Movimiento.entrar()) } },
                    ) { _, d ->
                        alcance.launch {
                            val v = (corrimiento.value + d).coerceIn(-tope, 0f)
                            corrimiento.snapTo(v)
                            if (v < -umbral && !paso) { paso = true; Haptico.seleccion() }
                            if (v > -umbral && paso) paso = false
                        }
                    }
                })
                .sombra(forma, hex(0x0F2328, 0.18f), 14.dp, y = 10.dp)
                .clip(forma).background(if (nuevo) C.menta else C.tarjeta)
                .then(if (est == EstadoTurno.enCurso) Modifier.border(2.dp, C.seleccion, forma) else Modifier)
                .alpha(if (est == EstadoTurno.atendido || est == EstadoTurno.cancelado) 0.55f else 1f)
                .combinedClickable(onClick = alTocar, onLongClick = { Haptico.suave(); menu = true }),
        ) {
            if (!grande) {
                Row(Modifier.fillMaxWidth().heightIn(min = 80.dp).height(IntrinsicSize.Min)) {
                    Column(Modifier.width(76.dp).fillMaxHeight(), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(2.dp, Alignment.CenterVertically)) {
                        Texto(Formato.horaDe(turno.inicio), Tipo.horaTalon, tinta, tracking = (-0.4).sp)
                        Texto(Formato.duracion(turno.minutos), Tipo.lato(12f, numeros = true), sub)
                    }
                    Troquel(if (nuevo) hex(0x102B31, 0.35f) else C.linea)
                    Datos(turno, mostrarProfesional, nuevo, est, tinta, sub, Modifier.padding(horizontal = 14.dp, vertical = 10.dp))
                }
            } else {
                Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
                        Texto(Formato.horaDe(turno.inicio), Tipo.horaTalon, tinta)
                        Texto(Formato.duracion(turno.minutos), Tipo.lato(12f), sub)
                    }
                    Troquel(C.linea, vertical = false)
                    Datos(turno, mostrarProfesional, nuevo, est, tinta, sub)
                }
            }
        }
        MenuContextual(menu, opciones) { menu = false }
    }
}

@Composable
private fun Datos(turno: Turno, mostrarProfesional: Boolean, nuevo: Boolean, est: EstadoTurno, tinta: Color, sub: Color, modifier: Modifier = Modifier) {
    Column(modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(4.dp, Alignment.CenterVertically)) {
        Texto(turno.cliente.nombre, Tipo.lato(17f, Peso.Bold), tinta, lineas = 1)
        Texto(if (mostrarProfesional && turno.profesional.isNotEmpty()) "${turno.servicio} · ${turno.profesional}" else turno.servicio, Tipo.lato(13f), sub, lineas = 1)
        if (nuevo) PastillaEstado("Nueva · recién", "bolt.fill", C.petroleoHondo, C.menta)
        else PastillaEstado(est.texto, est.simbolo, est.fondo, est.tinta)
    }
}

/** Un hueco (tiempo libre): rayas que corren, «LIBRE · 45 MIN», la hora con quién, y la acción. */
@Composable
fun VistaHueco(hueco: Dia.Hueco, accion: String, simboloAccion: String = "megaphone", avisado: Boolean = false, alTocar: () -> Unit, alAccion: () -> Unit) {
    Hueco(
        Modifier.fillMaxWidth().heightIn(min = 72.dp).clickable(onClick = alTocar)
            .semantics { contentDescription = "Libre ${Formato.duracion(hueco.minutos)}, ${Dia.hhmm(hueco.desde)} con ${hueco.profesional}" },
        Radio.talon, quieto = avisado,
    ) {
        Row(Modifier.fillMaxWidth().padding(start = 16.dp, top = 10.dp, bottom = 10.dp, end = 10.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            Column(Modifier.weight(1f).offset(x = (-8).dp), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                PastillaPapel { Texto("LIBRE · ${Formato.duracion(hueco.minutos).uppercase()}", Tipo.lato(12f, Peso.Black), C.huecoTinta, tracking = 0.96.sp) }
                PastillaPapel(8.dp) { Texto("${Dia.hhmm(hueco.desde)} · ${hueco.profesional}", Tipo.lato(17f, Peso.Bold, numeros = true), C.huecoTinta) }
            }
            Row(
                Modifier.sombra(CircleShape, hex(0x17805A, 0.9f), 8.dp, y = 8.dp).height(44.dp).clip(CircleShape).background(C.menta)
                    .clickable(enabled = !avisado, onClick = alAccion).padding(start = 10.dp, end = 14.dp),
                verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp),
            ) {
                Icono(if (avisado) "checkmark" else simboloAccion, 16f, C.petroleoHondo)
                Texto(if (avisado) "Avisado" else accion, Tipo.lato(14f, Peso.Black), C.petroleoHondo)
            }
        }
    }
}

/** «El día» en la cabecera: una fila por profesional, de la primera a la última hora, con lo ocupado en crema al
 *  88 %, lo libre rayado y la línea menta de ahora. Para TalkBack, un solo elemento. */
@Composable
fun ElDia(dias: List<Dia.DeProfesional>, esHoy: Boolean) {
    val (desde, hasta) = Dia.rango(dias)
    val total = maxOf(60, hasta - desde).toFloat()
    val libres = dias.sumOf { it.libres }
    val resumen = "El día: " + dias.joinToString("; ") { d -> val n = d.ocupado.size; "${d.nombre}, $n ${if (n == 1) "turno" else "turnos"}, ${Formato.libres(d.libres)}" }
    Column(
        Modifier.fillMaxWidth().clip(RoundedCornerShape(20.dp)).background(Color.White.copy(alpha = 0.07f))
            .padding(start = 14.dp, end = 14.dp, top = 14.dp, bottom = 12.dp).semantics(mergeDescendants = true) { contentDescription = resumen },
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Texto("EL DÍA", Tipo.lato(12f, Peso.Black), Color.White.copy(alpha = 0.75f), Modifier.weight(1f), tracking = 0.96.sp)
            Rayas(Modifier.size(14.dp, 10.dp).clip(RoundedCornerShape(3.dp)), quieto = true, paso = 5.dp, color = C.menta, fondo = Color.Transparent)
            Spacer(Modifier.width(4.dp))
            Texto(Formato.libres(libres), Tipo.lato(12f, Peso.Black), Color.White.copy(alpha = 0.75f))
        }
        Box {
            Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                dias.forEach { d ->
                    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Texto(d.nombre.split(" ").firstOrNull() ?: d.nombre, Tipo.lato(13f, Peso.Bold), Color.White.copy(alpha = 0.85f), Modifier.width(44.dp), lineas = 1)
                        BoxWithConstraints(Modifier.weight(1f).height(16.dp)) {
                            val ancho = maxWidth
                            Box(Modifier.fillMaxSize().clip(RoundedCornerShape(4.dp)).background(Color.White.copy(alpha = 0.06f)))
                            d.franjas.forEach { f ->
                                Rayas(Modifier.offset(x = ancho * ((f.desde - desde) / total)).width(ancho * (f.largo / total)).fillMaxHeight().clip(RoundedCornerShape(4.dp)),
                                    quieto = true, paso = 6.dp, color = C.menta, fondo = C.menta.copy(alpha = 0.12f))
                            }
                            d.ocupado.forEach { o ->
                                Box(Modifier.offset(x = ancho * ((o.tramo.desde - desde) / total)).width(maxOf(2.dp, ancho * (o.tramo.largo / total) - 2.dp)).fillMaxHeight()
                                    .clip(RoundedCornerShape(4.dp)).background(C.crema.copy(alpha = 0.88f)))
                            }
                        }
                    }
                }
            }
            if (esHoy) {
                val m = Dia.minutosAhora()
                if (m in desde..hasta) {
                    BoxWithConstraints(Modifier.matchParentSize().padding(vertical = 0.dp)) {
                        Box(Modifier.offset(x = 52.dp + (maxWidth - 52.dp) * ((m - desde) / total) - 1.dp, y = (-5).dp).width(2.dp).height(maxHeight + 10.dp).background(C.menta))
                    }
                }
            }
        }
        val marcas = marcas(desde, hasta)
        Row(Modifier.padding(start = 52.dp)) {
            marcas.forEachIndexed { i, h ->
                Texto("${h / 60}", Tipo.lato(11f, Peso.Bold, numeros = true), Color.White.copy(alpha = 0.6f))
                if (i < marcas.lastIndex) Spacer(Modifier.weight(1f))
            }
        }
    }
}

private fun marcas(desde: Int, hasta: Int): List<Int> {
    val paso = maxOf(60, ((hasta - desde) / 4 / 60) * 60)
    val out = mutableListOf<Int>()
    var h = desde
    while (h < hasta) { out.add(h); h += paso }
    out.add(hasta)
    return out
}
