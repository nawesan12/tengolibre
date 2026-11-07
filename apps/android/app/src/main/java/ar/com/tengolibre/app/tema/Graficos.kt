package ar.com.tengolibre.app.tema

import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.Spring
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.spring
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.PathEffect
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.drawscope.clipRect
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.TextMeasurer
import androidx.compose.ui.text.drawText
import androidx.compose.ui.text.rememberTextMeasurer
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import ar.com.tengolibre.app.nucleo.Formato
import kotlinx.coroutines.delay
import kotlinx.serialization.Serializable
import java.time.Instant
import java.time.format.DateTimeFormatter
import kotlin.math.abs
import kotlin.math.atan2
import kotlin.math.hypot
import kotlin.math.roundToInt

// Clon de Tema/Graficos.swift: los mismos gráficos (en iOS con Swift Charts) dibujados con Canvas, con la paleta
// del handoff. Todos crecen al aparecer y se pueden tocar para ver el valor.

/** Colores para series (medios de pago, servicios, estados): de la marca, distinguibles entre sí. */
object PaletaSeries {
    val series = listOf(C.menta, C.petroleo, hex(0xE8A317), hex(0x7A5CFA), hex(0x2F80ED), hex(0xEC4899), hex(0x8B5E34))
    /** En oscuro el petróleo se pierde contra la tarjeta: va el petróleo aclarado del tema (C.ocupado). */
    @Composable fun color(i: Int): Color = if (i % series.size == 1) C.ocupado else series[i % series.size]
}

/** Un número que cambia contando (plata, cantidades): el texto nuevo entra desde abajo. */
@Composable
fun NumeroAnimado(texto: String, estilo: androidx.compose.ui.text.TextStyle, color: Color, modifier: Modifier = Modifier) {
    AnimatedContent(texto, modifier, transitionSpec = { (slideInVertically { it / 2 } + fadeIn()) togetherWith (slideOutVertically { -it / 2 } + fadeOut()) }, label = "numero") {
        Texto(it, estilo, color, lineas = 1)
    }
}

/** Entrada escalonada: cada tarjeta sube y aparece un poquito después que la anterior (una sola vez). */
@Composable
fun Modifier.entrada(indice: Int = 0): Modifier {
    if (!animacionesActivas()) return this
    var visto by rememberSaveable { mutableStateOf(false) }
    val t = remember { Animatable(if (visto) 1f else 0f) }
    LaunchedEffect(Unit) {
        if (!visto) { delay(minOf(indice, 8) * 45L); t.animateTo(1f, Movimiento.entrar()); visto = true }
    }
    return this.graphicsLayer { alpha = t.value; translationY = (1 - t.value) * 14.dp.toPx() }
}

/** Un punto que late (lo que está pasando ahora). */
@Composable
fun Pulso(color: Color = C.menta, tam: Dp = 10.dp) {
    val latir = if (animacionesActivas()) rememberInfiniteTransition(label = "pulso").animateFloat(0f, 1f, infiniteRepeatable(tween(1400, easing = LinearEasing), RepeatMode.Restart), label = "l").value else 0f
    Box(Modifier.size(tam * 2.2f), contentAlignment = Alignment.Center) {
        Box(Modifier.size(tam * 2.2f).scale(0.4f + 0.6f * latir).alpha(1 - latir).clip(CircleShape).background(color.copy(alpha = 0.35f)))
        Box(Modifier.size(tam).clip(CircleShape).background(color))
    }
}

@Composable
private fun rememberCrecer(): Float {
    val a = remember { Animatable(0f) }
    LaunchedEffect(Unit) { a.animateTo(1f, spring(dampingRatio = 0.85f, stiffness = Spring.StiffnessLow)) }
    return a.value
}

/** «$ 15 mil», «$ 1,2 M» (para ejes y centros de gráficos). */
fun Formato.pesosCortos(n: Int): String {
    if (abs(n) >= 1_000_000) return "$ " + "%.1f".format(n / 1_000_000.0).replace(".", ",").replace(",0", "") + " M"
    if (abs(n) >= 1_000) return "$ ${(n / 1_000.0).roundToInt()} mil"
    return pesos(n)
}

private fun TextMeasurer.texto(s: String, tam: Float, peso: Peso = Peso.Regular, color: Color) = measure(s, Tipo.lato(tam, peso, numeros = true).copy(color = color))

// MARK: - Ingresos por día (barras, con el promedio y el valor al tocar)

data class DiaIngreso(val fecha: Instant, val monto: Int)

@Composable
fun GraficoIngresos(dias: List<DiaIngreso>, alto: Dp = 180.dp) {
    val crecer = rememberCrecer()
    var elegido by remember { mutableStateOf<Int?>(null) }
    val medidor = rememberTextMeasurer()
    val menta = C.menta; val petroleo = C.petroleo; val sub = C.sub; val linea = C.linea; val tinta = C.tinta; val tarjeta = C.tarjeta
    val con = dias.filter { it.monto > 0 }
    val promedio = if (con.isEmpty()) 0.0 else con.sumOf { it.monto }.toDouble() / con.size
    val maximo = maxOf(1, dias.maxOfOrNull { it.monto } ?: 1).toFloat()
    val fmt = DateTimeFormatter.ofPattern("d MMM", Formato.argentina).withZone(Formato.zona)
    Canvas(
        Modifier.fillMaxWidth().height(alto).semantics { contentDescription = "Ingresos por día" }
            .pointerInput(dias) {
                detectTapGestures { p ->
                    val izq = 44.dp.toPx(); val ancho = (size.width - izq) / maxOf(1, dias.size)
                    val i = ((p.x - izq) / ancho).toInt().coerceIn(0, dias.lastIndex)
                    elegido = if (elegido == i) null else i; Haptico.seleccion()
                }
            },
    ) {
        val izq = 44.dp.toPx(); val abajo = 18.dp.toPx(); val arriba = 30.dp.toPx()
        val alto2 = size.height - abajo - arriba
        // eje y: 3 marcas
        for (k in 0..2) {
            val v = maximo * k / 2
            val y = arriba + alto2 - alto2 * (v / maximo)
            drawLine(linea, Offset(izq, y), Offset(size.width, y), 1f)
            val t = medidor.texto(Formato.pesosCortos(v.toInt()), 10f, color = sub)
            drawText(t, topLeft = Offset(0f, y - t.size.height / 2))
        }
        val ancho = (size.width - izq) / maxOf(1, dias.size)
        dias.forEachIndexed { i, d ->
            val h = alto2 * (d.monto / maximo) * crecer
            val x = izq + i * ancho + ancho * 0.15f
            val brocha = if (elegido == i) Brush.verticalGradient(listOf(petroleo, petroleo)) else Brush.verticalGradient(listOf(menta, menta.copy(alpha = 0.55f)), startY = arriba + alto2 - h, endY = arriba + alto2)
            drawRoundRect(brocha, Offset(x, arriba + alto2 - h), Size(ancho * 0.7f, h), CornerRadius(4.dp.toPx()))
            if (dias.size <= 8 || i % maxOf(1, dias.size / 5) == 0) {
                val t = medidor.texto(fmt.format(d.fecha), 10f, color = sub)
                drawText(t, topLeft = Offset(izq + i * ancho + ancho / 2 - t.size.width / 2, size.height - t.size.height))
            }
        }
        if (promedio > 0) {
            val y = arriba + alto2 - alto2 * (promedio.toFloat() / maximo) * crecer
            drawLine(sub, Offset(izq, y), Offset(size.width, y), 1.dp.toPx(), pathEffect = PathEffect.dashPathEffect(floatArrayOf(4.dp.toPx(), 4.dp.toPx())))
            val t = medidor.texto("prom. ${Formato.pesos(promedio.toInt())}", 10f, Peso.Bold, sub)
            drawText(t, topLeft = Offset(size.width - t.size.width, y - t.size.height - 2))
        }
        elegido?.let { i ->
            val d = dias.getOrNull(i) ?: return@let
            val cx = izq + i * ancho + ancho / 2
            drawLine(linea, Offset(cx, arriba), Offset(cx, arriba + alto2), 1f)
            val t1 = medidor.texto(Formato.pesos(d.monto), 14f, Peso.Black, tinta)
            val t2 = medidor.texto(Formato.dia(d.fecha), 11f, color = sub)
            val w = maxOf(t1.size.width, t2.size.width) + 16.dp.toPx(); val hh = t1.size.height + t2.size.height + 10.dp.toPx()
            val x = (cx - w / 2).coerceIn(0f, size.width - w)
            drawRoundRect(Color.Black.copy(alpha = 0.08f), Offset(x, 3.dp.toPx()), Size(w, hh), CornerRadius(8.dp.toPx()))
            drawRoundRect(tarjeta, Offset(x, 0f), Size(w, hh), CornerRadius(8.dp.toPx()))
            drawText(t1, topLeft = Offset(x + (w - t1.size.width) / 2, 5.dp.toPx()))
            drawText(t2, topLeft = Offset(x + (w - t2.size.width) / 2, 5.dp.toPx() + t1.size.height))
        }
    }
}

// MARK: - Dona (medios de pago, servicios, estados), con el total en el centro

data class ParteDona(val nombre: String, val valor: Int, val color: Color)

@Composable
fun GraficoDona(partes: List<ParteDona>, centro: String, pie: String, mostrarPesos: Boolean = true) {
    val crecer = rememberCrecer()
    var elegida by remember { mutableStateOf<String?>(null) }
    val total = partes.sumOf { it.valor }
    val conValor = partes.filter { it.valor > 0 }
    Row(horizontalArrangement = Arrangement.spacedBy(18.dp), verticalAlignment = Alignment.CenterVertically) {
        Box(Modifier.size(140.dp), contentAlignment = Alignment.Center) {
            Canvas(Modifier.size(140.dp).pointerInput(partes) {
                detectTapGestures { p ->
                    val c = Offset(size.width / 2f, size.height / 2f)
                    if (hypot(p.x - c.x, p.y - c.y) > size.width / 2f) { elegida = null; return@detectTapGestures }
                    var ang = Math.toDegrees(atan2((p.y - c.y).toDouble(), (p.x - c.x).toDouble())).toFloat() + 90f
                    if (ang < 0) ang += 360f
                    var acum = 0f
                    val nueva = conValor.firstOrNull { acum += 360f * it.valor / maxOf(1, total); ang <= acum }?.nombre
                    elegida = if (elegida == nueva) null else nueva; Haptico.seleccion()
                }
            }) {
                val grosor = size.width / 2 * 0.38f
                var inicio = -90f
                conValor.forEach { p ->
                    val barrido = 360f * p.valor / maxOf(1, total) * crecer
                    drawArc(p.color.copy(alpha = if (elegida == null || elegida == p.nombre) 1f else 0.35f), inicio + 0.75f, maxOf(0f, barrido - 1.5f), false,
                        topLeft = Offset(grosor / 2, grosor / 2), size = Size(size.width - grosor, size.height - grosor), style = Stroke(grosor))
                    inicio += barrido
                }
            }
            val p = conValor.firstOrNull { it.nombre == elegida }
            Column(Modifier.padding(horizontal = 22.dp), horizontalAlignment = Alignment.CenterHorizontally) {
                Texto(p?.let { if (mostrarPesos) Formato.pesosCortos(it.valor) else "${it.valor}" } ?: centro, Tipo.lato(17f, Peso.Black, numeros = true), C.tinta, lineas = 1)
                Texto(p?.nombre ?: pie, Tipo.lato(10f), C.sub, lineas = 1)
            }
        }
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            partes.forEach { p ->
                Row(Modifier.alpha(if (elegida == null || elegida == p.nombre) 1f else 0.45f), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Box(Modifier.size(12.dp).clip(RoundedCornerShape(3.dp)).background(p.color))
                    Texto(p.nombre, Tipo.lato(13f), if (p.valor == 0) C.sub else C.tinta, Modifier.weight(1f), lineas = 1)
                    Texto(if (total > 0) "${(p.valor * 100.0 / total).roundToInt()} %" else "—", Tipo.lato(13f, Peso.Bold, numeros = true), C.sub)
                }
            }
        }
    }
}

// MARK: - Barras horizontales (por profesional, por servicio, reseñas)

data class FilaBarra(val nombre: String, val valor: Int, val detalle: String = "")

@Composable
fun GraficoBarras(filas: List<FilaBarra>, formato: (Int) -> String = { Formato.pesos(it) }, color: Color = C.menta) {
    val crecer = rememberCrecer()
    val maximo = (filas.maxOfOrNull { it.valor } ?: 1) * 1.35f + 1
    // Como el eje de Swift Charts: la columna de nombres toma el ancho del nombre más largo (hasta 120 dp) y las barras,
    // el resto; así «5 ★» no deja un hueco de 96 dp antes de la barra.
    Row(Modifier.padding(vertical = 4.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        Column(Modifier.widthIn(max = 120.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            filas.forEach { f ->
                Box(Modifier.height(32.dp), contentAlignment = Alignment.CenterStart) { Texto(f.nombre, Tipo.lato(12f), C.tinta, lineas = 1) }
            }
        }
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            filas.forEach { f ->
                Row(Modifier.height(32.dp), verticalAlignment = Alignment.CenterVertically) {
                    // En cero no hay barra (como Swift Charts): ni una rayita.
                    if (f.valor > 0) {
                        Box(Modifier.fillMaxWidth(maxOf(0.001f, f.valor / maximo * crecer)).height(22.dp).clip(RoundedCornerShape(5.dp))
                            .background(Brush.verticalGradient(listOf(color, color.copy(alpha = 0.75f)))))
                        Spacer(Modifier.width(6.dp))
                    }
                    Texto(formato(f.valor), Tipo.lato(12f, Peso.Bold, numeros = true), C.tinta, Modifier.alpha(crecer), lineas = 1)
                }
            }
        }
    }
}

// MARK: - Línea chiquita (tendencia de la semana)

@Composable
fun Sparkline(valores: List<Int>, color: Color = C.menta, modifier: Modifier = Modifier) {
    val trazo = remember { Animatable(0f) }
    LaunchedEffect(Unit) { trazo.animateTo(1f, tween(900)) }
    Canvas(modifier) {
        if (valores.size < 2) return@Canvas
        val max = maxOf(1, valores.max()).toFloat(); val min = valores.min().toFloat()
        val rango = maxOf(1f, max - min)
        val pts = valores.mapIndexed { i, v -> Offset(size.width * i / (valores.size - 1), size.height - 3.dp.toPx() - (size.height - 6.dp.toPx()) * (v - min) / rango) }
        val linea = Path().apply {
            moveTo(pts[0].x, pts[0].y)
            for (i in 1 until pts.size) { val a = pts[i - 1]; val b = pts[i]; val mx = (a.x + b.x) / 2; cubicTo(mx, a.y, mx, b.y, b.x, b.y) }
        }
        val area = Path().apply { addPath(linea); lineTo(size.width, size.height); lineTo(0f, size.height); close() }
        clipRect(right = size.width * trazo.value) {
            drawPath(area, Brush.verticalGradient(listOf(color.copy(alpha = 0.35f), color.copy(alpha = 0f))))
            drawPath(linea, color, style = Stroke(2.5.dp.toPx(), cap = StrokeCap.Round))
        }
    }
}

// MARK: - Visitas por mes (la ficha de una clienta)

data class MesVisitas(val mes: Instant, val visitas: Int)

@Composable
fun GraficoVisitas(meses: List<MesVisitas>) {
    val crecer = rememberCrecer()
    val medidor = rememberTextMeasurer()
    val menta = C.menta; val chip = C.chip; val linea = C.linea; val sub = C.sub
    val fmt = DateTimeFormatter.ofPattern("MMMMM", Formato.argentina).withZone(Formato.zona)
    Canvas(Modifier.fillMaxWidth().height(120.dp).semantics { contentDescription = "Visitas por mes" }) {
        val izq = 22.dp.toPx(); val abajo = 16.dp.toPx(); val alto = size.height - abajo
        // El tope del eje es par: las tres marcas (0, mitad, tope) son enteras y distintas («0, 1, 2», nunca «0, 1, 1»).
        val mayor = maxOf(1, meses.maxOfOrNull { it.visitas } ?: 1)
        val max = (((mayor + 1) / 2) * 2).toFloat()
        for (k in 0..2) {
            val y = alto - alto * k / 2f
            drawLine(linea, Offset(izq, y), Offset(size.width, y), 1f)
            val t = medidor.texto("${(max * k / 2).roundToInt()}", 10f, color = sub)
            drawText(t, topLeft = Offset(0f, y - t.size.height / 2))
        }
        val ancho = (size.width - izq) / maxOf(1, meses.size)
        meses.forEachIndexed { i, m ->
            val h = maxOf(2.dp.toPx(), alto * (m.visitas / max) * crecer)
            drawRoundRect(if (m.visitas > 0) menta else chip, Offset(izq + i * ancho + ancho * 0.15f, alto - h), Size(ancho * 0.7f, h), CornerRadius(4.dp.toPx()))
            if (i % 2 == 0) {
                val t = medidor.texto(fmt.format(m.mes).uppercase(), 10f, color = sub)
                drawText(t, topLeft = Offset(izq + i * ancho + ancho / 2 - t.size.width / 2, size.height - t.size.height))
            }
        }
    }
}

// MARK: - Mapa de calor de la semana (horas muertas)

@Serializable data class CeldaCalor(val dia: Int, val hora: Int, val ocupacion: Double? = null, val vacio: Double? = null, val estado: String = "")
@Serializable data class MapaDeCalor(val dias: List<Int> = emptyList(), val horas: List<Int> = emptyList(), val celdas: List<CeldaCalor> = emptyList())

private val DIAS = listOf("Dom", "Lun", "Mar", "Mié", "Jue", "Vie", "Sáb")

@Composable
fun MapaCalor(mapa: MapaDeCalor) {
    val aparecer = remember { Animatable(0f) }
    LaunchedEffect(Unit) { aparecer.animateTo(1f, tween(600)) }
    var elegida by remember { mutableStateOf<CeldaCalor?>(null) }
    val petroleo = C.petroleo; val chip = C.chip; val sub = C.sub; val tinta = C.tinta
    val medidor = rememberTextMeasurer()
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Canvas(Modifier.fillMaxWidth().height((mapa.dias.size * 30 + 24).dp).pointerInput(mapa) {
            detectTapGestures { p ->
                val izq = 36.dp.toPx(); val w = (size.width - izq) / maxOf(1, mapa.horas.size); val h = (size.height - 20.dp.toPx()) / maxOf(1, mapa.dias.size)
                val hi = ((p.x - izq) / w).toInt(); val di = (p.y / h).toInt()
                val hora = mapa.horas.getOrNull(hi); val dia = mapa.dias.getOrNull(di)
                elegida = mapa.celdas.firstOrNull { it.hora == hora && it.dia == dia }
            }
        }) {
            val izq = 36.dp.toPx(); val w = (size.width - izq) / maxOf(1, mapa.horas.size); val h = (size.height - 20.dp.toPx()) / maxOf(1, mapa.dias.size)
            mapa.dias.forEachIndexed { di, d ->
                val t = medidor.texto(DIAS[d % 7], 11f, Peso.Bold, tinta)
                drawText(t, topLeft = Offset(0f, di * h + (h - t.size.height) / 2))
                mapa.horas.forEachIndexed { hi, hr ->
                    val c = mapa.celdas.firstOrNull { it.dia == d && it.hora == hr } ?: return@forEachIndexed
                    val col = if (c.estado == "cerrado") chip else petroleo.copy(alpha = 0.12f + 0.88f * ((c.ocupacion ?: 0.0) / 100).toFloat())
                    drawRoundRect(col.copy(alpha = col.alpha * aparecer.value), Offset(izq + hi * w + w * 0.05f, di * h + h * 0.075f), Size(w * 0.9f, h * 0.85f), CornerRadius(4.dp.toPx()))
                }
            }
            mapa.horas.forEachIndexed { hi, hr ->
                val t = medidor.texto("$hr", 10f, color = sub)
                drawText(t, topLeft = Offset(izq + hi * w + (w - t.size.width) / 2, size.height - t.size.height))
            }
        }
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            val c = elegida
            Texto(
                if (c != null) "${DIAS[c.dia % 7]} ${c.hora} h: ${if (c.estado == "cerrado") "cerrado" else "ocupado el ${(c.ocupacion ?: 0.0).roundToInt()} %"}"
                else "Tocá una hora para ver qué tan llena suele estar.",
                Tipo.lato(12f, if (c != null) Peso.Bold else Peso.Regular), if (c != null) C.tinta else C.sub, Modifier.weight(1f),
            )
            Row(horizontalArrangement = Arrangement.spacedBy(3.dp), verticalAlignment = Alignment.CenterVertically) {
                Texto("vacío", Tipo.lato(10f), C.sub)
                listOf(0.12f, 0.4f, 0.7f, 1f).forEach { o -> Box(Modifier.size(12.dp).clip(RoundedCornerShape(2.dp)).background(petroleo.copy(alpha = o))) }
                Texto("lleno", Tipo.lato(10f), C.sub)
            }
        }
    }
}
