package ar.com.tengolibre.app.tema

import android.view.HapticFeedbackConstants
import android.view.View
import androidx.compose.animation.core.CubicBezierEasing
import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsPressedAsState
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxScope
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.defaultMinSize
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
import androidx.compose.foundation.text.BasicText
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.Immutable
import androidx.compose.runtime.ReadOnlyComposable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.draw.scale
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.ImageBitmap
import androidx.compose.ui.graphics.Outline
import androidx.compose.ui.graphics.Paint
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.graphics.ShaderBrush
import androidx.compose.ui.graphics.TileMode
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.graphics.drawOutline
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.drawscope.clipRect
import androidx.compose.ui.graphics.drawscope.drawIntoCanvas
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.Font
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.LineBreak
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.TextUnit
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import ar.com.tengolibre.app.R
import java.lang.ref.WeakReference
import kotlin.random.Random

// Tengo Libre · clon de Tema/Tema.swift (iOS): los mismos tokens, nombres y piezas. Lato sola; horas y plata
// con números tabulares; rótulos en MAYÚSCULAS con tracking 0,08em. Sobre menta sólo va petróleo hondo.
// Un punto de iOS es un dp acá (y un sp en el texto, que escala con el tamaño de letra del sistema).

fun hex(v: Long, alfa: Float = 1f): Color = Color(0xFF000000 or v).copy(alpha = alfa)

/** Un color «#rrggbb» que viene de la API (etiquetas, marca del negocio). */
fun hexTexto(t: String): Color? {
    val h = t.trim().removePrefix("#")
    if (h.length != 6) return null
    return h.toLongOrNull(16)?.let { hex(it) }
}

/** Los colores del tema (Assets.xcassets en iOS): claro y oscuro. */
@Immutable
data class Paleta(
    val oscuro: Boolean,
    val fondo: Color, val tarjeta: Color, val tinta: Color, val sub: Color, val linea: Color, val chip: Color,
    val cabecera: Color, val mentaTexto: Color, val mentaSuave: Color, val huecoTinta: Color, val huecoLinea: Color,
    val seleccion: Color, val seleccionTexto: Color, val campo: Color, val punto: Color, val barra: Color,
    val ocupado: Color, val bloquePetroleo: Color,
)

val PaletaClara = Paleta(
    oscuro = false,
    fondo = hex(0xF4EEE2), tarjeta = hex(0xFFFDF9), tinta = hex(0x0F2328), sub = hex(0x4A5D61), linea = hex(0xE2D6BF),
    chip = hex(0xEFE7D6), cabecera = hex(0x183C44), mentaTexto = hex(0x17805A), mentaSuave = hex(0xE3F6EC),
    huecoTinta = hex(0x0F3D2B), huecoLinea = hex(0x17805A, 0.45f), seleccion = hex(0x183C44), seleccionTexto = hex(0xFFFFFF),
    campo = hex(0xF4EEE2), punto = hex(0x183C44, 0.1f), barra = hex(0xFFFDF9, 0.94f), ocupado = hex(0x183C44),
    bloquePetroleo = hex(0x183C44),
)

val PaletaOscura = Paleta(
    oscuro = true,
    fondo = hex(0x0B181B), tarjeta = hex(0x14272C), tinta = hex(0xF4EEE2), sub = hex(0xA7B6B5), linea = hex(0x2A434A),
    chip = hex(0x1D353B), cabecera = hex(0x102B31), mentaTexto = hex(0x5FDCAA), mentaSuave = hex(0x3CCC90, 0.16f),
    huecoTinta = hex(0xC9F3DF), huecoLinea = hex(0x3CCC90, 0.5f), seleccion = hex(0x3CCC90), seleccionTexto = hex(0x102B31),
    campo = hex(0x0F2025), punto = hex(0xF4EEE2, 0.05f), barra = hex(0x0E1E22, 0.94f), ocupado = hex(0x5B8790),
    bloquePetroleo = hex(0x1F4A53),
)

val LocalPaleta = staticCompositionLocalOf { PaletaClara }

/** El tema: «auto» sigue al sistema (como @AppStorage("tema") en iOS). */
@Composable
fun TemaTL(tema: String = "auto", contenido: @Composable () -> Unit) {
    val oscuro = when (tema) { "oscuro" -> true; "claro" -> false; else -> isSystemInDarkTheme() }
    CompositionLocalProvider(LocalPaleta provides if (oscuro) PaletaOscura else PaletaClara, content = contenido)
}

/** Los colores. Los de la marca no cambian con el modo; los demás salen de la paleta (claro y oscuro). */
object C {
    // Marca
    val petroleo = hex(0x183C44)
    val petroleoHondo = hex(0x102B31)
    val menta = hex(0x3CCC90)
    val crema = hex(0xF4EEE2)
    val papel = hex(0xFFFDF9)
    // Tema
    val fondo: Color @Composable @ReadOnlyComposable get() = LocalPaleta.current.fondo
    val tarjeta: Color @Composable @ReadOnlyComposable get() = LocalPaleta.current.tarjeta
    val tinta: Color @Composable @ReadOnlyComposable get() = LocalPaleta.current.tinta
    val sub: Color @Composable @ReadOnlyComposable get() = LocalPaleta.current.sub
    val linea: Color @Composable @ReadOnlyComposable get() = LocalPaleta.current.linea
    val chip: Color @Composable @ReadOnlyComposable get() = LocalPaleta.current.chip
    val cabecera: Color @Composable @ReadOnlyComposable get() = LocalPaleta.current.cabecera
    val mentaTexto: Color @Composable @ReadOnlyComposable get() = LocalPaleta.current.mentaTexto
    val mentaSuave: Color @Composable @ReadOnlyComposable get() = LocalPaleta.current.mentaSuave
    val huecoTinta: Color @Composable @ReadOnlyComposable get() = LocalPaleta.current.huecoTinta
    val huecoLinea: Color @Composable @ReadOnlyComposable get() = LocalPaleta.current.huecoLinea
    /** Lo elegido: petróleo con texto blanco en claro, menta con petróleo hondo en oscuro. */
    val seleccion: Color @Composable @ReadOnlyComposable get() = LocalPaleta.current.seleccion
    val seleccionTexto: Color @Composable @ReadOnlyComposable get() = LocalPaleta.current.seleccionTexto
    /** La hoja (detalle, nuevo turno): la tarjeta en los dos modos. */
    val hoja: Color @Composable @ReadOnlyComposable get() = LocalPaleta.current.tarjeta
    /** Los campos de texto: el fondo crema en claro, más hondo en oscuro. */
    val campo: Color @Composable @ReadOnlyComposable get() = LocalPaleta.current.campo
    /** Los puntos del fondo crema (18 dp). */
    val punto: Color @Composable @ReadOnlyComposable get() = LocalPaleta.current.punto
    /** La barra de pestañas (94 %). */
    val barra: Color @Composable @ReadOnlyComposable get() = LocalPaleta.current.barra
    /** Lo ocupado en «El día» de las tarjetas de la semana. */
    val ocupado: Color @Composable @ReadOnlyComposable get() = LocalPaleta.current.ocupado
    /** Tarjetas petróleo (membresía, sólo lectura): más claras en oscuro para que se despeguen. */
    val bloquePetroleo: Color @Composable @ReadOnlyComposable get() = LocalPaleta.current.bloquePetroleo
    // Estados que no son de la marca (los de la web, estados.ts): siempre con ícono y palabra.
    val pagoFondo = hex(0xFBE9C2)
    val pagoTinta = hex(0x7A4A00)
    val ausenteFondo = hex(0xFBE0DC)
    val ausenteTinta = hex(0xA3241A)
    val error = hex(0xB42318)
}

enum class Peso { Regular, Bold, Black }

/** La letra: Lato (400/700/900), que escala con el tamaño de letra del sistema. */
object Tipo {
    val familia = FontFamily(
        Font(R.font.lato_400, FontWeight.Normal),
        Font(R.font.lato_700, FontWeight.Bold),
        Font(R.font.lato_900, FontWeight.Black),
    )

    /** Igual que Tipo.lato(tam, peso) en iOS. `numeros`: dígitos tabulares (.monospacedDigit()). */
    fun lato(tam: Float, peso: Peso = Peso.Regular, numeros: Boolean = false): TextStyle = TextStyle(
        fontFamily = familia,
        fontSize = tam.sp,
        fontWeight = when (peso) { Peso.Black -> FontWeight.Black; Peso.Bold -> FontWeight.Bold; Peso.Regular -> FontWeight.Normal },
        fontFeatureSettings = if (numeros) "tnum" else null,
        lineBreak = LineBreak.Paragraph,
    )

    val tituloGrande = lato(34f, Peso.Black)
    val titulo = lato(22f, Peso.Black)
    val cuerpo = lato(17f, Peso.Bold)
    val texto = lato(15f)
    val chico = lato(13f)
    val rotulo = lato(12f, Peso.Black)
    val horaTalon = lato(19f, Peso.Black, numeros = true)
}

object Espacio { val s1 = 4.dp; val s2 = 8.dp; val s3 = 12.dp; val s4 = 16.dp; val s6 = 24.dp; val s8 = 32.dp; val s12 = 48.dp }
object Radio { val talon = 18.dp; val tarjeta = 20.dp; val bloque = 24.dp; val hoja = 16.dp }

/** El movimiento de la marca (con las animaciones del sistema apagadas, sin animar). */
object Movimiento {
    val curva = CubicBezierEasing(0.2f, 0.7f, 0.2f, 1f)
    fun <T> entrar() = tween<T>(450, easing = curva)
    fun <T> tocar() = tween<T>(150, easing = curva)
    const val rayaHueco = 3.0
}

/** Los hápticos de iOS con su equivalente de Android. La vista se fija una vez en MainActivity. */
object Haptico {
    internal var vista: WeakReference<View>? = null
    private fun h(c: Int) { vista?.get()?.performHapticFeedback(c) }
    fun suave() = h(HapticFeedbackConstants.CLOCK_TICK)
    fun exito() = h(if (android.os.Build.VERSION.SDK_INT >= 30) HapticFeedbackConstants.CONFIRM else HapticFeedbackConstants.VIRTUAL_KEY)
    fun error() = h(if (android.os.Build.VERSION.SDK_INT >= 30) HapticFeedbackConstants.REJECT else HapticFeedbackConstants.LONG_PRESS)
    fun seleccion() = h(if (android.os.Build.VERSION.SDK_INT >= 34) HapticFeedbackConstants.SEGMENT_TICK else HapticFeedbackConstants.CLOCK_TICK)
}

// MARK: - Texto e íconos

/** El texto de la app (Text de SwiftUI). */
@Composable
fun Texto(
    texto: String,
    estilo: TextStyle = Tipo.texto,
    color: Color = C.tinta,
    modifier: Modifier = Modifier,
    lineas: Int = Int.MAX_VALUE,
    alinear: TextAlign = TextAlign.Start,
    tracking: TextUnit = TextUnit.Unspecified,
) {
    BasicText(
        texto, modifier,
        style = estilo.copy(color = color, textAlign = alinear, letterSpacing = if (tracking != TextUnit.Unspecified) tracking else estilo.letterSpacing),
        maxLines = lineas, overflow = TextOverflow.Ellipsis,
    )
}

private val simbolosContorno = FontFamily(Font(R.font.simbolos_contorno))
private val simbolosRelleno = FontFamily(Font(R.font.simbolos_relleno))

/** Un ícono por su nombre de SF Symbol (Image(systemName:) en iOS). «.fill» elige la variante rellena. */
@Composable
fun Icono(simbolo: String, tam: Float = 17f, color: Color = C.tinta, modifier: Modifier = Modifier, descripcion: String? = null) {
    val relleno = simbolo.endsWith(".fill") || simbolo.contains(".fill.")
    val nombre = simbolo.replace(".fill", "")
    val c = Simbolos.mapa[nombre] ?: Simbolos.mapa[nombre.substringBeforeLast('.')] ?: ''
    // Los símbolos de Material son un poco más chicos que los de SF a igual tamaño de letra.
    BasicText(
        c.toString(),
        modifier.then(if (descripcion != null) Modifier.semantics { contentDescription = descripcion } else Modifier),
        style = TextStyle(fontFamily = if (relleno) simbolosRelleno else simbolosContorno, fontSize = (tam * 1.18f).sp, color = color, lineHeight = (tam * 1.18f).sp),
    )
}

/** Rótulo: MAYÚSCULAS, 12 Black, tracking 0,08em, en el gris del tema. */
@Composable
fun TextoRotulo(texto: String, color: Color = C.sub, modifier: Modifier = Modifier) =
    Texto(texto.uppercase(), Tipo.rotulo, color, modifier, tracking = (12 * 0.08).sp)

/** Un rótulo de sección («PARA ATENDER», «DESPUÉS»). */
@Composable
fun Rotulo(texto: String, color: Color = C.sub) {
    TextoRotulo(texto, color, Modifier.fillMaxWidth().padding(start = 4.dp, end = 4.dp, top = 8.dp).semantics { heading() })
}

// MARK: - Fondos y formas

/** El grano de las cabeceras de petróleo: ruido blanco (180 dp, repetido). */
object Grano {
    private val base: android.graphics.Bitmap by lazy {
        val lado = 180
        val bmp = android.graphics.Bitmap.createBitmap(lado, lado, android.graphics.Bitmap.Config.ARGB_8888)
        val r = Random(7)
        for (y in 0 until lado) for (x in 0 until lado) {
            if (r.nextInt(100) < 55) bmp.setPixel(x, y, android.graphics.Color.argb((r.nextDouble(0.0, 0.22) * 255).toInt(), 255, 255, 255))
        }
        bmp
    }
    private val porDensidad = HashMap<Int, android.graphics.BitmapShader>()

    /** El mosaico a 180 dp para esta densidad. */
    fun sombreador(densidad: Float): android.graphics.BitmapShader = porDensidad.getOrPut((densidad * 100).toInt()) {
        val lado = (180 * densidad).toInt().coerceAtLeast(1)
        android.graphics.BitmapShader(android.graphics.Bitmap.createScaledBitmap(base, lado, lado, false),
            android.graphics.Shader.TileMode.REPEAT, android.graphics.Shader.TileMode.REPEAT)
    }
}

private fun Modifier.grano(opacidad: Float = 0.5f) = drawBehind {
    drawIntoCanvas { canvas ->
        val paint = Paint()
        paint.asFrameworkPaint().shader = Grano.sombreador(density)
        paint.alpha = opacidad
        canvas.drawRect(0f, 0f, size.width, size.height, paint)
    }
}

/** Fondo de petróleo con grano (cabeceras, ingreso, elegir negocio). `brillo`: centro del brillo menta. */
fun Modifier.fondoPetroleo(color: Color, brillo: Offset? = null): Modifier = this
    .background(color)
    .then(if (brillo != null) Modifier.drawBehind {
        drawRect(Brush.radialGradient(listOf(C.menta.copy(alpha = 0.22f), Color.Transparent),
            center = Offset(size.width * brillo.x, size.height * brillo.y), radius = 320.dp.toPx()))
    } else Modifier)
    .grano()

/** El fondo crema con puntos de 1,2 dp cada 18 dp. */
@Composable
fun Modifier.fondoPuntos(): Modifier {
    val fondo = C.fondo
    val punto = C.punto
    return this.background(fondo).drawBehind {
        val paso = 18.dp.toPx(); val r = 1.2.dp.toPx()
        var y = paso / 2
        while (y < size.height) {
            var x = paso / 2
            while (x < size.width) { drawCircle(punto, r, Offset(x, y)); x += paso }
            y += paso
        }
    }
}

/** Sombra como la de SwiftUI (.shadow(color:radius:x:y:)), dibujada detrás de la forma. */
fun Modifier.sombra(forma: Shape, color: Color, radio: Dp, x: Dp = 0.dp, y: Dp = 0.dp): Modifier = drawBehind {
    if (color.alpha == 0f) return@drawBehind
    val contorno = forma.createOutline(size, layoutDirection, this)
    drawIntoCanvas { canvas ->
        val paint = Paint()
        val fp = paint.asFrameworkPaint()
        fp.isAntiAlias = true
        fp.color = android.graphics.Color.TRANSPARENT
        fp.setShadowLayer(radio.toPx(), x.toPx(), y.toPx(), color.toArgb())
        canvas.drawOutline(contorno, paint)
    }
}

/** Tarjeta: el fondo de tarjeta, radio 20 y la sombra suave del handoff. */
@Composable
fun Modifier.tarjeta(radio: Dp = Radio.tarjeta, fondo: Color = C.tarjeta): Modifier {
    val forma = RoundedCornerShape(radio)
    return this.sombra(forma, hex(0x0F2328, 0.18f), 14.dp, y = 10.dp).clip(forma).background(fondo)
}

/** Contorno de 1,5 dp en la línea del tema (botones secundarios, campos). */
@Composable
fun Modifier.contorno(radio: Dp, color: Color = C.linea, ancho: Dp = 1.5.dp): Modifier =
    this.border(ancho, color, if (radio >= 999.dp) CircleShape else RoundedCornerShape(radio))

// MARK: - El hueco (tiempo libre)

/** Rayas a −45°, de 2 dp de menta al 42 % cada 10 dp, que corren con un ciclo de 3 s. Quietas si `quieto`. */
@Composable
fun Rayas(modifier: Modifier = Modifier, quieto: Boolean = false, paso: Dp = 10.dp, color: Color = C.menta.copy(alpha = 0.42f), fondo: Color = C.tarjeta.copy(alpha = 0.92f)) {
    val t = if (quieto || !animacionesActivas()) 0f else {
        val tr = rememberInfiniteTransition(label = "rayas")
        tr.animateFloat(0f, 1f, infiniteRepeatable(tween((Movimiento.rayaHueco * 1000).toInt(), easing = LinearEasing), RepeatMode.Restart), label = "t").value
    }
    Canvas(modifier) {
        drawRect(fondo)
        val p = paso.toPx()
        val corrimiento = t * p * 2
        var x = -size.height - p * 2 + corrimiento
        clipRect {
            while (x < size.width + p) {
                drawLine(color, Offset(x, size.height), Offset(x + size.height, 0f), strokeWidth = 2.dp.toPx())
                x += p
            }
        }
    }
}

/** Si el sistema tiene las animaciones prendidas (Reduce Motion en iOS). */
@Composable
fun animacionesActivas(): Boolean {
    val vista = LocalView.current
    return remember(vista) {
        android.provider.Settings.Global.getFloat(vista.context.contentResolver, android.provider.Settings.Global.ANIMATOR_DURATION_SCALE, 1f) != 0f
    }
}

/** Un bloque libre: rayas + contorno menta de 1,5 dp (la letra del hueco va con C.huecoTinta). */
@Composable
fun Hueco(modifier: Modifier = Modifier, radio: Dp = Radio.talon, quieto: Boolean = false, alinear: Alignment = Alignment.Center, contenido: @Composable BoxScope.() -> Unit) {
    val forma = RoundedCornerShape(radio)
    Box(modifier.clip(forma).contorno(radio, C.huecoLinea), contentAlignment = alinear) {
        Rayas(Modifier.matchParentSize(), quieto = quieto)
        contenido()
    }
}

/** La pastilla de papel que va sobre las rayas («LIBRE · 45 MIN»). */
@Composable
fun PastillaPapel(radio: Dp = 999.dp, contenido: @Composable RowScope.() -> Unit) {
    Row(Modifier.background(C.tarjeta, if (radio >= 999.dp) CircleShape else RoundedCornerShape(radio)).padding(horizontal = 8.dp, vertical = 2.dp),
        verticalAlignment = Alignment.CenterVertically, content = contenido)
}

// MARK: - Botones

@Composable
private fun presionado(fuente: MutableInteractionSource): Boolean = fuente.collectIsPressedAsState().value

/** La acción principal: pastilla menta, petróleo hondo, 900. */
@Composable
fun BotonMenta(
    texto: String,
    alTocar: () -> Unit,
    modifier: Modifier = Modifier,
    alto: Dp = 56.dp,
    tam: Float = 17f,
    conSombra: Boolean = true,
    habilitado: Boolean = true,
    simbolo: String? = null,
) {
    val fuente = remember { MutableInteractionSource() }
    val p = presionado(fuente)
    val escala by animateFloatAsState(if (p) 0.98f else 1f, Movimiento.tocar(), label = "escala")
    Row(
        // La opacidad va antes del fondo y la sombra: deshabilitado se apaga la pastilla entera, como en iOS.
        modifier.scale(escala)
            .alpha(if (habilitado) (if (p) 0.85f else 1f) else 0.45f)
            .sombra(CircleShape, if (conSombra) hex(0x17805A, 0.6f) else Color.Transparent, 13.dp, y = 12.dp)
            .fillMaxWidth().heightIn(min = alto)
            .clip(CircleShape).background(C.menta)
            .clickable(fuente, null, enabled = habilitado, onClick = alTocar)
            .padding(horizontal = 16.dp),
        horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically,
    ) {
        if (simbolo != null) { Icono(simbolo, tam, C.petroleoHondo); Spacer(Modifier.width(8.dp)) }
        Texto(texto, Tipo.lato(tam, Peso.Black, numeros = true), C.petroleoHondo, lineas = 1)
    }
}

/** Pastilla menta chica, del ancho del texto (Avisar, Responder, Reservar). */
@Composable
fun BotonMentaChico(texto: String, alTocar: () -> Unit, modifier: Modifier = Modifier, alto: Dp = 44.dp, simbolo: String? = null) {
    val fuente = remember { MutableInteractionSource() }
    val p = presionado(fuente)
    Row(
        modifier.heightIn(min = alto).clip(CircleShape).background(C.menta)
            .clickable(fuente, null, onClick = alTocar).alpha(if (p) 0.85f else 1f).padding(horizontal = 16.dp),
        verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.Center,
    ) {
        if (simbolo != null) { Icono(simbolo, 14f, C.petroleoHondo); Spacer(Modifier.width(6.dp)) }
        Texto(texto, Tipo.lato(14f, Peso.Black), C.petroleoHondo, lineas = 1)
    }
}

/** Secundario: contorno de 1,5 dp, 700. */
@Composable
fun BotonContorno(
    texto: String,
    alTocar: () -> Unit,
    modifier: Modifier = Modifier,
    alto: Dp = 52.dp,
    tam: Float = 17f,
    lleno: Boolean = true,
    habilitado: Boolean = true,
    simbolo: String? = null,
) {
    val fuente = remember { MutableInteractionSource() }
    val p = presionado(fuente)
    Row(
        modifier.alpha(if (habilitado) 1f else 0.45f).then(if (lleno) Modifier.fillMaxWidth() else Modifier).heightIn(min = alto)
            .clip(CircleShape).background(if (p) C.chip else Color.Transparent).contorno(999.dp)
            .clickable(fuente, null, enabled = habilitado, onClick = alTocar)
            .padding(horizontal = 16.dp),
        horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically,
    ) {
        if (simbolo != null) { Icono(simbolo, tam, C.tinta); Spacer(Modifier.width(8.dp)) }
        Texto(texto, Tipo.lato(tam, Peso.Bold), C.tinta, lineas = 1)
    }
}

/** Botón sobre fondo chip, radio 16 (Llamar, WhatsApp). */
@Composable
fun BotonChip(texto: String, alTocar: () -> Unit, modifier: Modifier = Modifier, alto: Dp = 52.dp, radio: Dp = 16.dp, simbolo: String? = null) {
    val fuente = remember { MutableInteractionSource() }
    val p = presionado(fuente)
    Row(
        modifier.fillMaxWidth().heightIn(min = alto).clip(RoundedCornerShape(radio))
            .background(C.chip.copy(alpha = if (p) 0.7f else 1f)).clickable(fuente, null, onClick = alTocar),
        horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically,
    ) {
        if (simbolo != null) { Icono(simbolo, 16f, C.tinta); Spacer(Modifier.width(8.dp)) }
        Texto(texto, Tipo.lato(16f, Peso.Bold), C.tinta, lineas = 1)
    }
}

/** Botón redondo de 44 dp (acciones de la cabecera y de las hojas). */
@Composable
fun BotonRedondo(simbolo: String, etiqueta: String, alTocar: () -> Unit, sobrePetroleo: Boolean = false, numero: Int = 0) {
    Box(
        Modifier.size(44.dp).semantics { contentDescription = if (numero > 0) "$etiqueta, $numero sin leer" else etiqueta }
            .clip(CircleShape).background(if (sobrePetroleo) Color.White.copy(alpha = 0.1f) else C.chip).clickable(onClick = alTocar),
        contentAlignment = Alignment.Center,
    ) {
        Icono(simbolo, 19f, if (sobrePetroleo) Color.White else C.tinta)
    }
    if (numero > 0) {
        Box(Modifier.offset(x = (-18).dp, y = 2.dp)) {
            Texto("$numero", Tipo.lato(11f, Peso.Black, numeros = true), C.petroleoHondo,
                Modifier.defaultMinSize(18.dp, 18.dp).clip(CircleShape).background(C.menta).padding(horizontal = 5.dp), alinear = TextAlign.Center)
        }
    }
}

/** Botón de texto menta (Cancelar en las hojas, «Ver lo último guardado»). */
@Composable
fun BotonTexto(texto: String, alTocar: () -> Unit, modifier: Modifier = Modifier, color: Color = C.mentaTexto) {
    val fuente = remember { MutableInteractionSource() }
    val p = presionado(fuente)
    Box(modifier.heightIn(min = 44.dp).clickable(fuente, null, onClick = alTocar).alpha(if (p) 0.6f else 1f), contentAlignment = Alignment.CenterStart) {
        Texto(texto, Tipo.lato(15f, Peso.Bold), color, lineas = 1)
    }
}

// MARK: - Piezas

/** Iniciales en un círculo chip. */
@Composable
fun Iniciales(nombre: String, tam: Dp = 44.dp, fondo: Color = C.chip, tinta: Color = C.tinta, radio: Dp? = null) {
    val ini = nombre.split(" ").filter { it.isNotEmpty() }.take(2).map { it.first() }.joinToString("").uppercase()
    Box(Modifier.size(tam).clip(RoundedCornerShape(radio ?: (tam / 2))).background(fondo), contentAlignment = Alignment.Center) {
        Texto(ini.ifEmpty { "·" }, Tipo.lato(tam.value * 0.34f, Peso.Black), tinta)
    }
}

/** Chip con ícono y palabra (etiquetas de clienta, filtros). */
@Composable
fun Chip(texto: String, simbolo: String? = null, elegido: Boolean = false, alto: Dp = 36.dp, modifier: Modifier = Modifier) {
    val tinta = if (elegido) C.seleccionTexto else C.tinta
    Row(
        modifier.height(alto).clip(CircleShape).background(if (elegido) C.seleccion else Color.Transparent)
            .contorno(999.dp, if (elegido) Color.Transparent else C.linea)
            .padding(start = if (simbolo == null) 14.dp else 8.dp, end = 14.dp),
        verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        if (simbolo != null) Icono(simbolo, 15f, tinta)
        Texto(texto, Tipo.lato(14f, Peso.Bold), tinta, lineas = 1)
    }
}

/** Segmentado del handoff: sobre petróleo (cabeceras) o sobre la tarjeta. */
@Composable
fun <V> Segmentado(opciones: List<Pair<V, String>>, elegido: V, alElegir: (V) -> Unit, sobrePetroleo: Boolean = true, modifier: Modifier = Modifier) {
    Row(
        modifier.fillMaxWidth().clip(RoundedCornerShape(14.dp)).background(if (sobrePetroleo) Color.White.copy(alpha = 0.1f) else C.chip).padding(4.dp),
        horizontalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        opciones.forEach { (v, t) ->
            val on = v == elegido
            Box(
                Modifier.weight(1f).heightIn(min = 36.dp).clip(RoundedCornerShape(10.dp)).background(if (on) C.menta else Color.Transparent)
                    .clickable { Haptico.seleccion(); alElegir(v) },
                contentAlignment = Alignment.Center,
            ) {
                Texto(t, Tipo.lato(14f, Peso.Bold), if (on) C.petroleoHondo else (if (sobrePetroleo) Color.White else C.tinta), lineas = 1)
            }
        }
    }
}

/** Un campo de texto de 16 sp (mínimo del handoff) sobre el fondo de campo. */
@Composable
fun Campo(titulo: String, texto: String, alCambiar: (String) -> Unit, ayuda: String = "", teclado: KeyboardType = KeyboardType.Text, multilinea: Boolean = false, modifier: Modifier = Modifier) {
    Column(modifier, verticalArrangement = Arrangement.spacedBy(6.dp)) {
        if (titulo.isNotEmpty()) TextoRotulo(titulo)
        val tinta = C.tinta
        Box(Modifier.fillMaxWidth().clip(RoundedCornerShape(14.dp)).background(C.campo).padding(horizontal = 14.dp, vertical = 13.dp)) {
            if (texto.isEmpty() && ayuda.isNotEmpty()) Texto(ayuda, Tipo.lato(16f), C.sub.copy(alpha = 0.7f))
            BasicTextField(
                texto, alCambiar, Modifier.fillMaxWidth(),
                textStyle = Tipo.lato(16f).copy(color = tinta),
                singleLine = !multilinea, minLines = if (multilinea) 3 else 1, maxLines = if (multilinea) 8 else 1,
                keyboardOptions = KeyboardOptions(keyboardType = teclado),
                cursorBrush = androidx.compose.ui.graphics.SolidColor(C.mentaTexto),
            )
        }
    }
}

/** Una fila de lista (Más, ajustes): ícono en cuadradito chip, texto, detalle y chevron. */
@Composable
fun FilaLista(simbolo: String, titulo: String, detalle: String = "", final: String = "chevron.right", tinta: Color = C.tinta, pendiente: Boolean = false, modifier: Modifier = Modifier) {
    Row(modifier.fillMaxWidth().heightIn(min = 56.dp).padding(horizontal = 14.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
        Box {
            Box(Modifier.size(32.dp).clip(RoundedCornerShape(9.dp)).background(C.chip), contentAlignment = Alignment.Center) { Icono(simbolo, 17f, tinta) }
            if (pendiente) Box(Modifier.align(Alignment.TopEnd).offset(3.dp, (-3).dp).size(10.dp).clip(CircleShape).background(C.tarjeta).padding(2.dp).clip(CircleShape).background(C.menta))
        }
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(1.dp)) {
            Texto(titulo, Tipo.lato(16f), tinta)
            if (detalle.isNotEmpty()) Texto(detalle, Tipo.lato(12f, if (pendiente) Peso.Bold else Peso.Regular), if (pendiente) C.mentaTexto else C.sub)
        }
        if (final.isNotEmpty()) Icono(final, 15f, C.sub)
    }
}

/** Una tarjeta-lista: filas separadas por la línea del tema. */
@Composable
fun <T> ListaTarjeta(items: List<T>, modifier: Modifier = Modifier, fila: @Composable (T) -> Unit) {
    Column(modifier.fillMaxWidth().tarjeta()) {
        items.forEachIndexed { i, it ->
            if (i > 0) Box(Modifier.fillMaxWidth().height(1.dp).background(C.linea))
            fila(it)
        }
    }
}

/** El interruptor del handoff (51×31, menta encendido). */
@Composable
fun Interruptor(
    titulo: String, activo: Boolean, alCambiar: (Boolean) -> Unit, detalle: String = "", simbolo: String? = null, modifier: Modifier = Modifier,
    estiloTitulo: TextStyle = Tipo.lato(16f), colorTitulo: Color = C.tinta, altoMinimo: Dp = 56.dp,
) {
    Row(modifier.fillMaxWidth().heightIn(min = altoMinimo).clickable { alCambiar(!activo) }, verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
        if (simbolo != null) Icono(simbolo, 20f, C.tinta)
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Texto(titulo, estiloTitulo, colorTitulo)
            if (detalle.isNotEmpty()) Texto(detalle, Tipo.lato(12f), C.sub)
        }
        val x by animateFloatAsState(if (activo) 1f else 0f, Movimiento.tocar(), label = "interruptor")
        Box(Modifier.size(51.dp, 31.dp).clip(CircleShape).background(if (activo) C.menta else C.linea).padding(2.dp)) {
            Box(Modifier.offset(x = (20 * x).dp).size(27.dp).sombra(CircleShape, Color.Black.copy(alpha = 0.15f), 4.dp, y = 2.dp).clip(CircleShape).background(Color.White))
        }
    }
}

/** Esqueleto de carga (shimmer de la tarjeta al chip). */
@Composable
fun Esqueleto(alto: Dp = 80.dp, radio: Dp = 18.dp, modifier: Modifier = Modifier) {
    val tarjeta = C.tarjeta; val chip = C.chip
    val anima = animacionesActivas()
    val fase = if (anima) rememberInfiniteTransition(label = "esqueleto").animateFloat(-1f, 1f, infiniteRepeatable(tween(1400, easing = LinearEasing)), label = "f").value else -1f
    Box(modifier.fillMaxWidth().height(alto).clip(RoundedCornerShape(radio)).drawBehind {
        drawRect(Brush.linearGradient(listOf(tarjeta, chip, tarjeta), start = Offset(size.width * fase, 0f), end = Offset(size.width * (fase + 1), 0f)))
    })
}

/** Vacío que enseña: rayas, título, texto y acciones (handoff 20). */
@Composable
fun VacioQueEnsena(rayas: String, titulo: String, texto: String, acciones: @Composable ColumnScope.() -> Unit = {}) {
    Column(Modifier.fillMaxWidth().tarjeta(Radio.bloque).padding(20.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Hueco(Modifier.fillMaxWidth().heightIn(min = 150.dp)) {
            PastillaPapel { Texto(rayas, Tipo.lato(15f, Peso.Black), C.huecoTinta, Modifier.padding(6.dp)) }
        }
        Texto(titulo, Tipo.lato(22f, Peso.Black), C.tinta, tracking = (-0.44).sp)
        Texto(texto, Tipo.lato(15f).copy(lineHeight = 21.sp), C.sub)
        acciones()
    }
}

/** Error con salida (handoff 22). */
@Composable
fun ErrorConSalida(titulo: String, texto: String, reintentar: () -> Unit) {
    Column(Modifier.fillMaxWidth().tarjeta(Radio.bloque).padding(vertical = 32.dp, horizontal = 20.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Box(Modifier.size(64.dp).clip(CircleShape).background(C.chip), contentAlignment = Alignment.Center) { Icono("arrow.triangle.2.circlepath", 28f) }
        Texto(titulo, Tipo.lato(20f, Peso.Bold), C.tinta, alinear = TextAlign.Center)
        Texto(texto, Tipo.lato(15f), C.sub, alinear = TextAlign.Center)
        BotonMenta("Reintentar", reintentar, alto = 52.dp, conSombra = false)
    }
}

/** Un aviso arriba de todo (toast del handoff): petróleo hondo, ícono en círculo menta y una acción. */
data class Toast(
    val simbolo: String,
    val titulo: String,
    val texto: String = "",
    val accion: String = "",
    val error: Boolean = false,
    val id: String = java.util.UUID.randomUUID().toString(),
)

@Composable
fun VistaToast(t: Toast, alTocar: () -> Unit) {
    val forma = RoundedCornerShape(24.dp)
    Row(
        Modifier.padding(horizontal = 12.dp).sombra(forma, Color.Black.copy(alpha = 0.35f), 20.dp, y = 14.dp)
            .clip(forma).background(C.petroleoHondo).padding(start = 14.dp, top = 12.dp, bottom = 12.dp, end = 12.dp),
        verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Box(Modifier.size(40.dp).clip(CircleShape).background(if (t.error) C.error else C.menta), contentAlignment = Alignment.Center) {
            Icono(t.simbolo, 19f, if (t.error) Color.White else C.petroleoHondo)
        }
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Texto(t.titulo, Tipo.lato(15f, Peso.Bold), Color.White)
            if (t.texto.isNotEmpty()) Texto(t.texto, Tipo.lato(13f), Color.White.copy(alpha = 0.78f))
        }
        if (t.accion.isNotEmpty()) {
            Box(Modifier.heightIn(min = 44.dp).clip(CircleShape).background(Color.White.copy(alpha = 0.12f)).clickable(onClick = alTocar).padding(horizontal = 16.dp), contentAlignment = Alignment.Center) {
                Texto(t.accion, Tipo.lato(15f, Peso.Bold), Color.White)
            }
        }
    }
}

/** La pastilla de un estado: siempre ícono + palabra. */
@Composable
fun PastillaEstado(texto: String, simbolo: String, fondo: Color, tinta: Color, alto: Dp = 24.dp) {
    Row(
        Modifier.height(alto).clip(CircleShape).background(fondo).padding(start = 6.dp, end = 9.dp),
        verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        Icono(simbolo, alto.value * 0.55f, tinta)
        Texto(texto, Tipo.lato(alto.value * 0.5f, Peso.Bold), tinta, lineas = 1)
    }
}
