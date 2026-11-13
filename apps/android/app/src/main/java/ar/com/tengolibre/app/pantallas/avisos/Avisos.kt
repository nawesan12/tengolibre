package ar.com.tengolibre.app.pantallas.avisos

import android.content.ClipData
import android.content.ClipboardManager
import android.content.ContentValues
import android.content.Context
import android.content.Intent
import android.graphics.Bitmap
import android.graphics.Canvas
import android.graphics.Paint
import android.graphics.RectF
import android.net.Uri
import android.os.Build
import android.provider.MediaStore
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.FilterQuality
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.core.content.FileProvider
import androidx.core.content.res.ResourcesCompat
import ar.com.tengolibre.app.R
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.nucleo.*
import ar.com.tengolibre.app.pantallas.hoy.BarraHoja
import ar.com.tengolibre.app.pantallas.mas.DestinoMas
import ar.com.tengolibre.app.pantallas.mas.PreferenciasPush
import ar.com.tengolibre.app.pantallas.principal.*
import ar.com.tengolibre.app.tema.*
import com.google.zxing.BarcodeFormat
import com.google.zxing.EncodeHintType
import com.google.zxing.qrcode.QRCodeWriter
import com.google.zxing.qrcode.decoder.ErrorCorrectionLevel
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable
import java.io.File

// Clon de Pantallas/Avisos/Avisos.swift.

/** 17 Avisos (la campanita): lo de los últimos siete días agrupado HOY · AYER · antes, con su ícono, «SIN LEER» en las
 *  nuevas y el atajo a lo que corresponde. Al abrir se marcan como vistas. */
@Composable
fun ColumnScope.Avisos(cerrar: () -> Unit) {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    val alcance = rememberCoroutineScope()
    var lista by remember { mutableStateOf<List<Notificacion>>(emptyList()) }
    var cargando by remember { mutableStateOf(true) }
    var preferencias by remember { mutableStateOf(false) }

    @Serializable data class Campanita(val notificaciones: List<Notificacion> = emptyList(), val sinLeer: Int = 0)
    suspend fun cargar() {
        try {
            val r: Campanita = estado.api.get("/v1/panel/campanita")
            lista = r.notificaciones
            if (r.sinLeer > 0) { runCatching { estado.api.pedir("POST", "/v1/panel/campanita/vista") }; estado.sinLeer = 0 }
        } catch (e: Exception) { estado.avisar(e) }
        cargando = false
    }
    LaunchedEffect(Unit) { cargar() }

    if (preferencias) {
        Box(Modifier.weight(1f)) { PreferenciasPush(volver = { preferencias = false }) }
        return
    }

    fun abrir(n: Notificacion) {
        val id = n.turnoId
        if (id != null) {
            alcance.launch { estado.api.opcional<Turno>("/v1/panel/turnos/$id")?.let { cerrar(); delay(350); nav.turno = it } }
            return
        }
        cerrar()
        when (n.tipo) {
            "resena" -> if (estado.esProfesional) nav.pestana = Pestana.resenas else { nav.pestana = Pestana.mas; nav.pilaMas.add(DestinoMas.resenas) }
            "espera" -> nav.pestana = if (estado.esProfesional) Pestana.hoy else Pestana.espera
            "dias_libres", "equipo" -> { nav.pestana = Pestana.mas; nav.pilaMas.add(DestinoMas.diasLibres) }
        }
    }

    Box(Modifier.fillMaxWidth().height(52.dp).padding(horizontal = 16.dp)) {
        Box(Modifier.align(Alignment.CenterStart).size(44.dp).clickable { preferencias = true }.semantics { contentDescription = "Qué avisos te llegan" }, contentAlignment = Alignment.CenterStart) { Icono("gearshape", 19f, C.mentaTexto) }
        Texto("Avisos", Tipo.lato(17f, Peso.Bold), C.tinta, Modifier.align(Alignment.Center))
        Box(Modifier.align(Alignment.CenterEnd)) { BotonTexto("Listo", cerrar) }
    }
    val grupos = lista.groupBy { Formato.seccion(it.creadoEn).uppercase() }
    Refrescable({ cargar() }, Modifier.weight(1f).background(C.fondo)) { Column(Modifier.fillMaxSize().background(C.fondo).verticalScroll(rememberScrollState()).padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        if (cargando && lista.isEmpty()) repeat(5) { Esqueleto(72.dp) }
        else if (lista.isEmpty()) VacioQueEnsena("Todo tranquilo", "No hay avisos", "Acá vas a ver las reservas, cancelaciones, reseñas y todo lo que pase en tu negocio.")
        grupos.forEach { (s, ns) ->
            TextoRotulo(s, modifier = Modifier.padding(top = 6.dp))
            Column(Modifier.fillMaxWidth().tarjeta()) {
                ns.forEachIndexed { i, n ->
                    if (i > 0) Box(Modifier.padding(start = 68.dp).fillMaxWidth().height(1.dp).background(C.linea))
                    Row(Modifier.fillMaxWidth().clickable { abrir(n) }.padding(14.dp), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                        Box(Modifier.size(40.dp).clip(RoundedCornerShape(12.dp)).background(if (n.nueva) C.menta else C.chip), contentAlignment = Alignment.Center) {
                            Icono(n.simbolo, 18f, if (n.nueva) C.petroleoHondo else C.tinta)
                        }
                        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                            Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                                Texto(n.titulo, Tipo.lato(15f, Peso.Bold), C.tinta, Modifier.weight(1f), lineas = 2)
                                Texto(Formato.horaDe(n.creadoEn), Tipo.lato(12f, numeros = true), C.sub)
                            }
                            if (n.texto.isNotEmpty()) Texto(n.texto, Tipo.lato(14f), C.sub, lineas = 3)
                            if (n.nueva) Row(Modifier.padding(top = 2.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(4.dp)) {
                                Box(Modifier.size(6.dp).clip(CircleShape).background(C.menta))
                                Texto("SIN LEER", Tipo.lato(10f, Peso.Black), C.mentaTexto, tracking = 0.8.sp)
                            }
                        }
                    }
                }
            }
        }
    } }
}

/** El QR de un link: petróleo sobre crema (falseColor en iOS), 12 px por módulo. */
fun imagenQR(texto: String, modulo: Int = 12): Bitmap? = runCatching {
    val m = QRCodeWriter().encode(texto, BarcodeFormat.QR_CODE, 0, 0, mapOf(EncodeHintType.ERROR_CORRECTION to ErrorCorrectionLevel.M, EncodeHintType.MARGIN to 1))
    val b = Bitmap.createBitmap(m.width * modulo, m.height * modulo, Bitmap.Config.ARGB_8888)
    val c = Canvas(b)
    c.drawColor(0xFFF4EEE2.toInt())
    val p = Paint().apply { color = 0xFF183C44.toInt() }
    for (y in 0 until m.height) for (x in 0 until m.width) if (m[x, y]) c.drawRect((x * modulo).toFloat(), (y * modulo).toFloat(), ((x + 1) * modulo).toFloat(), ((y + 1) * modulo).toFloat(), p)
    b
}.getOrNull()

/** La tarjeta del QR dibujada como imagen (la historia de Instagram y lo que se guarda): crema con el logo, el QR,
 *  el nombre y «Escaneá y reservá tu turno», sobre petróleo. */
private fun tarjetaQR(contexto: Context, nombre: String, qr: Bitmap): Bitmap {
    val d = contexto.resources.displayMetrics.density
    val ancho = (360 * d).toInt(); val margen = (40 * d)
    val alto = (440 * d).toInt()
    val b = Bitmap.createBitmap(ancho + (margen * 2).toInt(), alto + (margen * 2).toInt(), Bitmap.Config.ARGB_8888)
    val c = Canvas(b)
    c.drawColor(0xFF183C44.toInt())
    val tarjeta = RectF(margen, margen, margen + ancho, margen + alto)
    c.drawRoundRect(tarjeta, 24 * d, 24 * d, Paint(Paint.ANTI_ALIAS_FLAG).apply { color = 0xFFF4EEE2.toInt() })
    ResourcesCompat.getDrawable(contexto.resources, R.drawable.logotipo, null)?.let { logo ->
        val h = (26 * d).toInt(); val w = (h * logo.intrinsicWidth.toFloat() / maxOf(1, logo.intrinsicHeight)).toInt()
        val x = (b.width - w) / 2; val y = (margen + 24 * d).toInt()
        logo.setBounds(x, y, x + w, y + h); logo.draw(c)
    }
    val lado = (200 * d).toInt()
    val qx = (b.width - lado) / 2f; val qy = margen + 64 * d
    c.drawBitmap(qr, null, RectF(qx, qy, qx + lado, qy + lado), Paint().apply { isFilterBitmap = false })
    val lato = ResourcesCompat.getFont(contexto, R.font.lato_900)
    val t1 = Paint(Paint.ANTI_ALIAS_FLAG).apply { color = 0xFF183C44.toInt(); textSize = 20 * d; typeface = lato; textAlign = Paint.Align.CENTER }
    c.drawText(nombre, b.width / 2f, qy + lado + 40 * d, t1)
    val t2 = Paint(Paint.ANTI_ALIAS_FLAG).apply { color = 0xFF4A5D61.toInt(); textSize = 14 * d; typeface = ResourcesCompat.getFont(contexto, R.font.lato_400); textAlign = Paint.Align.CENTER }
    c.drawText("Escaneá y reservá tu turno", b.width / 2f, qy + lado + 66 * d, t2)
    return b
}

/** 18 Compartir: el QR sobre crema con el logo, el link con «Copiar» y las cuatro formas de mandarlo. */
@Composable
fun ColumnScope.Compartir(cerrar: () -> Unit) {
    val estado = LocalEstado.current
    val contexto = LocalContext.current
    var copiado by remember { mutableStateOf(false) }
    val slug = estado.negocio?.slug.orEmpty()
    val url = Web.pagina(slug)
    val texto = Web.textoPagina(slug)
    val qr = remember(url) { imagenQR(url) }

    fun historia() {
        val q = qr ?: return
        val img = tarjetaQR(contexto, estado.negocio?.nombre.orEmpty(), q)
        val dir = File(contexto.cacheDir, "compartir").apply { mkdirs() }
        val f = File(dir, "historia.png"); f.outputStream().use { img.compress(Bitmap.CompressFormat.PNG, 100, it) }
        val uri = FileProvider.getUriForFile(contexto, "ar.com.tengolibre.app.archivos", f)
        val ig = Intent("com.instagram.share.ADD_TO_STORY").setDataAndType(uri, "image/png").addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION).setPackage("com.instagram.android")
            .putExtra("source_application", "ar.com.tengolibre.app")
        if (ig.resolveActivity(contexto.packageManager) != null) contexto.startActivity(ig)
        else contexto.startActivity(Intent.createChooser(Intent(Intent.ACTION_SEND).setType("image/png").putExtra(Intent.EXTRA_STREAM, uri).putExtra(Intent.EXTRA_TEXT, url).addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION), null))
    }

    // Como iOS: se guarda el QR solo (petróleo sobre crema), no la tarjeta.
    fun guardarQR() {
        val img = qr ?: return
        val valores = ContentValues().apply {
            put(MediaStore.Images.Media.DISPLAY_NAME, "tengolibre-qr-$slug.png")
            put(MediaStore.Images.Media.MIME_TYPE, "image/png")
            if (Build.VERSION.SDK_INT >= 29) put(MediaStore.Images.Media.RELATIVE_PATH, "Pictures/Tengo Libre")
        }
        val ok = runCatching {
            contexto.contentResolver.insert(MediaStore.Images.Media.EXTERNAL_CONTENT_URI, valores)?.let { u ->
                contexto.contentResolver.openOutputStream(u)?.use { img.compress(Bitmap.CompressFormat.PNG, 100, it) }
            } != null
        }.getOrDefault(false)
        if (ok) estado.avisar("Guardamos el QR en tus fotos") else estado.avisar("No pudimos guardarlo", ok = false)
    }

    BarraHoja("Tu link", cerrar, textoCancelar = "", accion = "Listo") { cerrar() }
    Column(Modifier.weight(1f).verticalScroll(rememberScrollState()).padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        // La tarjeta del QR (siempre clara, como en iOS).
        Column(Modifier.fillMaxWidth().clip(RoundedCornerShape(24.dp)).background(C.crema).padding(24.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(14.dp)) {
            Image(painterResource(R.drawable.logotipo), null, Modifier.height(26.dp))
            qr?.let { Image(it.asImageBitmap(), "Código QR de tu página", Modifier.size(200.dp), filterQuality = FilterQuality.None) }
            Texto(estado.negocio?.nombre.orEmpty(), Tipo.lato(20f, Peso.Black), C.petroleo)
            Texto("Escaneá y reservá tu turno", Tipo.lato(14f), hex(0x4A5D61))
        }
        Row(Modifier.fillMaxWidth().height(56.dp).clip(CircleShape).background(C.chip).padding(start = 16.dp, end = 6.dp), verticalAlignment = Alignment.CenterVertically) {
            Texto(url.removePrefix("https://"), Tipo.lato(16f, Peso.Bold), C.tinta, Modifier.weight(1f), lineas = 1)
            BotonMentaChico(if (copiado) "Copiado" else "Copiar", {
                (contexto.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager).setPrimaryClip(ClipData.newPlainText("link", url))
                Haptico.exito(); copiado = true
            })
        }
        val tiles = listOf(
            Triple("Historia de Instagram", "camera") { historia() },
            Triple("Estado de WhatsApp", "message") {
                val i = Intent(Intent.ACTION_SEND).setType("text/plain").putExtra(Intent.EXTRA_TEXT, texto).setPackage("com.whatsapp")
                if (i.resolveActivity(contexto.packageManager) != null) contexto.startActivity(i) else estado.compartirTexto(texto)
            },
            Triple("Guardar el QR", "square.and.arrow.down") { guardarQR() },
            Triple("Más opciones", "square.and.arrow.up") { estado.compartirTexto("$texto\n$url") },
        )
        tiles.chunked(2).forEach { fila ->
            Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                fila.forEach { (t, s, a) ->
                    Column(
                        Modifier.weight(1f).heightIn(min = 96.dp).clip(RoundedCornerShape(18.dp)).background(C.fondo).border(1.5.dp, C.linea, RoundedCornerShape(18.dp)).clickable(onClick = a).padding(14.dp),
                        verticalArrangement = Arrangement.spacedBy(10.dp),
                    ) { Icono(s, 22f, C.mentaTexto); Texto(t, Tipo.lato(15f, Peso.Bold), C.tinta) }
                }
            }
        }
    }
}
