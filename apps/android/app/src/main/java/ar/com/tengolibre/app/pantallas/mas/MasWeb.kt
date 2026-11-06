package ar.com.tengolibre.app.pantallas.mas

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.net.Uri
import android.provider.Settings
import androidx.activity.compose.BackHandler
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.combinedClickable
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
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.verticalScroll
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.input.pointer.PointerEventPass
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.nucleo.*
import ar.com.tengolibre.app.pantallas.hoy.BarraHoja
import ar.com.tengolibre.app.pantallas.principal.*
import ar.com.tengolibre.app.tema.*
import coil3.compose.AsyncImage
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonPrimitive
import java.io.ByteArrayOutputStream
import java.time.Instant
import java.time.LocalDate
import java.time.temporal.ChronoUnit

// Clon de Pantallas/Mas/MasWeb.swift.

private fun copiar(ctx: Context, t: String) {
    (ctx.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager).setPrimaryClip(ClipData.newPlainText("tengolibre", t))
}

/** Un renglón que lleva a otra sección (NavigationLink + FilaLista + .tarjeta()). */
@Composable
private fun IrA(simbolo: String, titulo: String, destino: DestinoMas) {
    val nav = LocalNavegacion.current
    Box(Modifier.fillMaxWidth().tarjeta().clickable { nav.pila.add(destino) }) { FilaLista(simbolo, titulo) }
}

@Composable
private fun GuardarSuelto(texto: String = "Guardar", habilitado: Boolean = true, accion: suspend () -> Unit) {
    val alcance = rememberCoroutineScope()
    var guardando by remember { mutableStateOf(false) }
    BotonGuardar({ alcance.launch { guardando = true; accion(); guardando = false } }, texto, guardando, habilitado)
}

@Composable
private fun GuardarEnBloque(texto: String = "Guardar", accion: suspend () -> Unit) { Box(Modifier.padding(12.dp)) { GuardarSuelto(texto, accion = accion) } }

// MARK: - Reportes

@Composable
fun Reportes() {
    @Serializable data class Fila(val id: String? = null, val nombre: String = "", val turnos: Int = 0, val atendidos: Int = 0, val ausentes: Int = 0, val ingresos: Int = 0, val minutosReservados: Int = 0, val ocupacion: Double? = null)
    @Serializable data class DiaR(val fecha: String, val turnos: Int = 0, val atendidos: Int = 0, val ingresos: Int = 0)
    @Serializable data class Resumen(val turnos: Int = 0, val atendidos: Int = 0, val ingresos: Int = 0, val ticketPromedio: Int = 0, val clientes: Int = 0, val tasaAusencias: Double? = null, val ocupacion: Double? = null)
    @Serializable data class Reporte(val desde: String = "", val hasta: String = "", val resumen: Resumen = Resumen(), val porEstado: Map<String, Int> = emptyMap(), val porProfesional: List<Fila> = emptyList(),
        val porServicio: List<Fila> = emptyList(), val porDia: List<DiaR> = emptyList(), val clientesNuevos: Int = 0, val clientesRecurrentes: Int = 0)
    @Serializable data class Peor(val texto: String = "")
    @Serializable data class Sugerencia(val porcentaje: Int = 0, val texto: String = "", val destinatarios: Int = 0, val codigo: String = "")
    @Serializable data class HorasMuertas(val peor: Peor? = null, val sugerencia: Sugerencia? = null, val mapa: MapaDeCalor? = null)

    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    var periodo by remember { mutableStateOf("mes") }
    var r by remember { mutableStateOf<Reporte?>(null) }
    var antes by remember { mutableStateOf<Reporte?>(null) }
    var hm by remember { mutableStateOf<HorasMuertas?>(null) }
    var error by remember { mutableStateOf("") }
    val hoy = Formato.enArgentina(Instant.now()).toLocalDate()
    val rango: Pair<LocalDate, LocalDate> = when (periodo) {
        "semana" -> hoy.minusDays(6) to hoy
        "trimestre" -> hoy.minusDays(89) to hoy
        else -> hoy.withDayOfMonth(1) to hoy
    }
    fun ins(d: LocalDate) = d.atStartOfDay(Formato.zona).plusHours(12).toInstant()
    suspend fun cargar() {
        val (d, h) = rango
        val largo = ChronoUnit.DAYS.between(d, h)
        val ad = d.minusDays(largo + 1); val ah = d.minusDays(1)
        try {
            r = estado.api.get("/v1/panel/reportes", listOf("desde" to d.toString(), "hasta" to h.toString()))
            antes = estado.api.opcional("/v1/panel/reportes", listOf("desde" to ad.toString(), "hasta" to ah.toString()))
            hm = estado.api.opcional("/v1/panel/reportes/horas-muertas")
            error = ""
        } catch (e: ErrorApi) { error = e.detalle } catch (e: Exception) { error = "Probá de nuevo." }
    }
    LaunchedEffect(periodo) { cargar() }
    fun cambio(a: Int, b: Int): String {
        if (b <= 0) return ""
        val p = (a - b).toDouble() / b * 100
        return "${if (p >= 0) "▲" else "▼"} ${Math.round(Math.abs(p))} % vs. antes"
    }
    @Composable fun cifra(t: String, v: String, d: String?, m: Modifier) {
        Column(m.heightIn(min = 96.dp).tarjeta(18.dp).padding(14.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            TextoRotulo(t.uppercase())
            NumeroAnimado(v, Tipo.lato(24f, Peso.Black, numeros = true), C.tinta)
            if (!d.isNullOrEmpty()) Texto(d, Tipo.lato(12f, Peso.Bold), if (d.startsWith("▼")) C.ausenteTinta else C.mentaTexto)
        }
    }
    @Composable fun tabla(t: String, f: List<Fila>) {
        if (f.isEmpty()) return
        val orden = f.sortedByDescending { it.ingresos }.take(8)
        Column(Modifier.entrada(8).fillMaxWidth().tarjeta().padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            TextoRotulo(t.uppercase())
            GraficoBarras(orden.map { FilaBarra(it.nombre, it.ingresos) }, { Formato.pesosCortos(it) })
            orden.forEach { x ->
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Texto(x.nombre, Tipo.lato(13f, Peso.Bold), C.tinta, Modifier.weight(1f), lineas = 1)
                    Texto("${x.atendidos} atendidos${if (x.ausentes > 0) " · ${x.ausentes} no vinieron" else ""}${x.ocupacion?.let { " · ${Formato.porcentaje(it)}" } ?: ""}", Tipo.lato(12f), C.sub)
                }
            }
        }
    }

    PantallaMas("Reportes", "${Formato.dia(ins(rango.first))} al ${Formato.dia(ins(rango.second))}", cargando = r == null && error.isEmpty(), error = error, recargar = { cargar() },
        abajo = { Segmentado(listOf("semana" to "7 días", "mes" to "Este mes", "trimestre" to "90 días"), periodo, { periodo = it }) }) {
        val x = r ?: return@PantallaMas
        val a = antes
        val cifras = listOf(
            Triple("Ingresos", Formato.pesos(x.resumen.ingresos), a?.let { cambio(x.resumen.ingresos, it.resumen.ingresos) }),
            Triple("Turnos atendidos", "${x.resumen.atendidos}", a?.let { cambio(x.resumen.atendidos, it.resumen.atendidos) }),
            Triple("Ticket promedio", Formato.pesos(x.resumen.ticketPromedio), null),
            Triple("Ocupación", x.resumen.ocupacion?.let { Formato.porcentaje(it) } ?: "—", null),
            Triple("Clientas", "${x.resumen.clientes}", "${x.clientesNuevos} nuevas · ${x.clientesRecurrentes} vuelven"),
            Triple("Ausencias", x.resumen.tasaAusencias?.let { Formato.porcentaje(it) } ?: "—", null),
        )
        cifras.chunked(2).forEachIndexed { fi, fila ->
            Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) { fila.forEachIndexed { i, (t, v, d) -> cifra(t, v, d, Modifier.weight(1f).entrada(fi * 2 + i)) } }
        }
        if (x.porDia.isNotEmpty()) Column(Modifier.entrada(6).fillMaxWidth().tarjeta().padding(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                TextoRotulo("INGRESOS POR DÍA", modifier = Modifier.weight(1f))
                Texto("Tocá una barra", Tipo.lato(11f), C.sub)
            }
            GraficoIngresos(x.porDia.mapNotNull { d -> Fecha.iso(d.fecha)?.let { DiaIngreso(it, d.ingresos) } })
        }
        // Cómo terminaron los turnos del período (dona).
        val nombres = listOf(Triple("atendido", "Atendidos", C.menta), Triple("confirmado", "Por venir", C.petroleo), Triple("pendiente", "Sin confirmar", hex(0xE8A317)),
            Triple("ausente", "No vinieron", C.ausenteTinta), Triple("cancelado", "Cancelados", C.sub))
        val partes = nombres.map { ParteDona(it.second, x.porEstado[it.first] ?: 0, it.third) }.filter { it.valor > 0 }
        if (partes.isNotEmpty()) Column(Modifier.entrada(7).fillMaxWidth().tarjeta().padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            TextoRotulo("CÓMO TERMINARON LOS TURNOS")
            GraficoDona(partes, "${x.resumen.turnos}", "turnos", mostrarPesos = false)
        }
        tabla("Por profesional", x.porProfesional)
        tabla("Por servicio", x.porServicio)
        val h = hm; val p = h?.peor
        if (h != null && p != null) Bloque("Horas muertas", "", *buildList<@Composable () -> Unit> {
            add { Renglon(p.texto, simbolo = "moon.zzz") }
            h.mapa?.takeIf { it.celdas.isNotEmpty() }?.let { m -> add { Box(Modifier.padding(16.dp)) { MapaCalor(m) } } }
            h.sugerencia?.let { s -> add {
                Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Texto(s.texto, Tipo.lato(14f), C.sub)
                    BotonMenta("Crear el cupón ${s.codigo} (${s.porcentaje} %)", {
                        alcance.launch { estado.intentar("Creamos el cupón ${s.codigo}") { estado.api.pedir("POST", "/v1/panel/reportes/horas-muertas/cupon", cuerpo = "{}".toByteArray()) } }
                    }, alto = 44.dp, tam = 15f, conSombra = false)
                }
            } }
        }.toTypedArray())
    }
}

// MARK: - Mi web

@Serializable data class NegocioDatos(
    val id: String = "", val slug: String = "", val nombre: String = "", val descripcion: String = "", val direccion: String = "", val ciudad: String = "",
    val telefono: String = "", val whatsapp: String = "", val instagram: String = "", val logo: String? = null, val zonaHoraria: String = "",
)
@Serializable data class WebDatos(
    val rubro: String = "", val provincia: String = "", val acerca: String = "", val portada: String? = null, val colorPrimario: String = "#183c44", val colorAcento: String = "#3ccc90",
    val email: String = "", val sitioWeb: String = "", val facebook: String = "", val tiktok: String = "", val latitud: Double? = null, val longitud: Double? = null,
    val estilo: JsonElement? = null,
)
@Serializable data class FotoGaleria(val id: String, val url: String = "", val epigrafe: String = "", val orden: Int = 0)

/** Una imagen remota recortada (AsyncImage con .scaledToFill), o el lugar vacío con el ícono de foto. */
@Composable
private fun Imagen(url: String?, alto: Dp, ancho: Dp?, radio: Dp) {
    val m = (if (ancho != null) Modifier.size(ancho, alto) else Modifier.fillMaxWidth().height(alto)).clip(RoundedCornerShape(radio)).background(C.chip)
    if (!url.isNullOrEmpty()) AsyncImage(url, null, m, contentScale = ContentScale.Crop)
    else Box(m, contentAlignment = Alignment.Center) { Icono("photo", 17f, C.sub) }
}

/** Achica la foto (lado largo 2000 px) y la deja en JPEG. */
private suspend fun jpegDe(ctx: Context, uri: Uri): ByteArray? = withContext(Dispatchers.IO) {
    runCatching {
        val img = ctx.contentResolver.openInputStream(uri)?.use { BitmapFactory.decodeStream(it) } ?: return@runCatching null
        val f = minOf(1f, 2000f / maxOf(img.width, img.height))
        val chica = if (f < 1f) Bitmap.createScaledBitmap(img, (img.width * f).toInt(), (img.height * f).toInt(), true) else img
        ByteArrayOutputStream().also { chica.compress(Bitmap.CompressFormat.JPEG, 85, it) }.toByteArray()
    }.getOrNull()
}

/** El elegidor de color (ColorPicker de iOS): la muestra redonda y, al tocarla, una grilla de colores y el código. */
@Composable
fun SelectorDeColor(hex: String, alCambiar: (String) -> Unit) {
    var abierto by remember { mutableStateOf(false) }
    Box(Modifier.size(30.dp).clip(CircleShape).background(hexTexto(hex) ?: C.petroleo).border(2.dp, C.linea, CircleShape).clickable { abierto = true }.semantics { contentDescription = "Elegir color" })
    val grilla = listOf(
        "#183c44", "#0f2328", "#3ccc90", "#17805a", "#2f80ed", "#1d4ed8", "#7a5cfa", "#6d28d9",
        "#ec4899", "#be185d", "#d9534f", "#b91c1c", "#e8a317", "#f97316", "#8b5e34", "#57534e",
        "#111827", "#374151", "#9ca3af", "#f4eee2", "#ffffff", "#0ea5e9", "#14b8a6", "#84cc16",
    )
    Hoja(abierto, { abierto = false }, alturas = listOf(Altura.media)) {
        var texto by remember(hex) { mutableStateOf(hex) }
        BarraHoja("Color", { abierto = false }, accion = "Listo", habilitada = hexTexto(texto) != null) { alCambiar(texto.lowercase()); abierto = false }
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
            grilla.chunked(8).forEach { fila ->
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                    fila.forEach { c ->
                        Box(Modifier.size(34.dp).clip(CircleShape).background(hexTexto(c) ?: Color.Gray).border(if (texto.equals(c, true)) 3.dp else 1.dp, if (texto.equals(c, true)) C.tinta else C.linea, CircleShape).clickable { texto = c })
                    }
                }
            }
            Campo("Código", texto, { texto = it })
        }
    }
}

@Composable
fun MiWeb() {
    val estado = LocalEstado.current
    val contexto = LocalContext.current
    val alcance = rememberCoroutineScope()
    var n by remember { mutableStateOf<NegocioDatos?>(null) }
    var w by remember { mutableStateOf<WebDatos?>(null) }
    var fotos by remember { mutableStateOf<List<FotoGaleria>>(emptyList()) }
    var error by remember { mutableStateOf("") }
    var subiendo by remember { mutableStateOf(false) }
    var menuFoto by remember { mutableStateOf<FotoGaleria?>(null) }

    @Serializable data class Fotos(val fotos: List<FotoGaleria> = emptyList())
    suspend fun cargar() {
        try {
            n = estado.api.get("/v1/panel/negocio")
            w = estado.api.get("/v1/panel/mi-web")
            fotos = estado.api.opcional<Fotos>("/v1/panel/mi-web/galeria")?.fotos ?: emptyList()
            error = ""
        } catch (e: ErrorApi) { error = e.detalle } catch (e: Exception) { error = "Probá de nuevo." }
    }
    LaunchedEffect(Unit) { cargar() }
    suspend fun subir(uri: Uri?): String? {
        uri ?: return null
        subiendo = true
        try { val jpg = jpegDe(contexto, uri) ?: return null; return estado.api.subirImagen(jpg) } catch (e: Exception) { estado.avisar(e); return null } finally { subiendo = false }
    }
    suspend fun patchNegocio(c: Map<String, String>) { estado.intentar { estado.api.hacer("PATCH", "/v1/panel/negocio", c) }; cargar() }
    suspend fun patchWeb(c: Map<String, JsonElement>) { estado.intentar { estado.api.hacer("PATCH", "/v1/panel/mi-web", c) }; cargar() }
    val pedirLogo = rememberLauncherForActivityResult(ActivityResultContracts.PickVisualMedia()) { u -> alcance.launch { subir(u)?.let { patchNegocio(mapOf("logo" to it)) } } }
    val pedirPortada = rememberLauncherForActivityResult(ActivityResultContracts.PickVisualMedia()) { u -> alcance.launch { subir(u)?.let { patchWeb(mapOf("portada" to JsonPrimitive(it))) } } }
    @Serializable data class FotoNueva(val url: String, val epigrafe: String = "")
    val pedirFoto = rememberLauncherForActivityResult(ActivityResultContracts.PickVisualMedia()) { u ->
        alcance.launch { subir(u)?.let { url -> estado.intentar("Sumamos la foto") { estado.api.hacer("POST", "/v1/panel/mi-web/galeria", FotoNueva(url)) }; cargar() } }
    }
    val soloImagenes = PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly)
    @Serializable data class Ids(val ids: List<String>)
    fun ordenar(f: FotoGaleria, primero: Boolean) = alcance.launch {
        val ids = fotos.sortedBy { it.orden }.map { it.id }.filter { it != f.id }.toMutableList()
        if (primero) ids.add(0, f.id) else ids.add(f.id)
        estado.intentar { estado.api.hacer("PUT", "/v1/panel/mi-web/galeria/orden", Ids(ids)) }
        cargar()
    }
    @Serializable data class CuerpoNegocio(val nombre: String, val descripcion: String, val direccion: String, val ciudad: String, val telefono: String, val whatsapp: String, val instagram: String)
    @Serializable data class CuerpoWeb(val rubro: String, val provincia: String, val acerca: String, val colorPrimario: String, val colorAcento: String, val email: String, val sitioWeb: String, val facebook: String, val tiktok: String)

    val pagina = Web.pagina(estado.negocio?.slug.orEmpty())
    PantallaMas("Mi web", pagina.removePrefix("https://"), acciones = listOf(AccionCabecera("safari", "Ver mi página") { estado.abrirPestana(pagina) }),
        cargando = (n == null || w == null) && error.isEmpty(), error = error, recargar = { cargar() }) {
        val nn = n ?: return@PantallaMas
        val ww = w ?: return@PantallaMas
        Box {
            Bloque("Logo y portada", "",
                {
                    Row(Modifier.padding(16.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(14.dp)) {
                        Imagen(nn.logo, 72.dp, 72.dp, 18.dp)
                        Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                            Texto(if (nn.logo == null) "Subir logo" else "Cambiar logo", Tipo.lato(15f, Peso.Bold), C.mentaTexto, Modifier.clickable { pedirLogo.launch(soloImagenes) })
                            if (nn.logo != null) Texto("Sacar", Tipo.lato(14f), C.error, Modifier.clickable { alcance.launch { patchNegocio(mapOf("logo" to "")) } })
                        }
                    }
                },
                {
                    Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Imagen(ww.portada, 140.dp, null, 14.dp)
                        Row {
                            Texto(if (ww.portada == null) "Subir portada" else "Cambiar portada", Tipo.lato(15f, Peso.Bold), C.mentaTexto, Modifier.clickable { pedirPortada.launch(soloImagenes) })
                            Spacer(Modifier.weight(1f))
                            if (ww.portada != null) Texto("Sacar", Tipo.lato(14f), C.error, Modifier.clickable { alcance.launch { patchWeb(mapOf("portada" to JsonPrimitive(""))) } })
                        }
                    }
                })
            if (subiendo) Box(Modifier.matchParentSize(), contentAlignment = Alignment.Center) { Pulso(C.menta, 20.dp) }
        }
        Bloque("Tu negocio", "",
            { RenglonCampo("Nombre", nn.nombre, { n = nn.copy(nombre = it) }) },
            { RenglonCampo("Descripción corta", nn.descripcion, { n = nn.copy(descripcion = it) }, multilinea = true) },
            { RenglonCampo("Rubro", ww.rubro, { w = ww.copy(rubro = it) }, "Peluquería, estética…") },
            { RenglonCampo("Sobre ustedes", ww.acerca, { w = ww.copy(acerca = it) }, multilinea = true) })
        Bloque("Dónde están", "",
            { RenglonCampo("Dirección", nn.direccion, { n = nn.copy(direccion = it) }) },
            { RenglonCampo("Ciudad", nn.ciudad, { n = nn.copy(ciudad = it) }) },
            { RenglonCampo("Provincia", ww.provincia, { w = ww.copy(provincia = it) }) })
        Bloque("Contacto y redes", "",
            { RenglonCampo("Teléfono", nn.telefono, { n = nn.copy(telefono = it) }, teclado = KeyboardType.Phone) },
            { RenglonCampo("WhatsApp", nn.whatsapp, { n = nn.copy(whatsapp = it) }, teclado = KeyboardType.Phone) },
            { RenglonCampo("Email", ww.email, { w = ww.copy(email = it) }, teclado = KeyboardType.Email) },
            { RenglonCampo("Instagram", nn.instagram, { n = nn.copy(instagram = it) }, "@tunegocio", teclado = KeyboardType.Uri) },
            { RenglonCampo("Facebook", ww.facebook, { w = ww.copy(facebook = it) }, teclado = KeyboardType.Uri) },
            { RenglonCampo("TikTok", ww.tiktok, { w = ww.copy(tiktok = it) }, teclado = KeyboardType.Uri) },
            { RenglonCampo("Otra web", ww.sitioWeb, { w = ww.copy(sitioWeb = it) }, teclado = KeyboardType.Uri) })
        Bloque("Colores", "",
            { Renglon("Principal") { SelectorDeColor(ww.colorPrimario) { w = ww.copy(colorPrimario = it) } } },
            { Renglon("Acento") { SelectorDeColor(ww.colorAcento) { w = ww.copy(colorAcento = it) } } })
        GuardarSuelto("Guardar los cambios") {
            val ok = estado.intentar("Guardamos tu web") {
                estado.api.hacer("PATCH", "/v1/panel/negocio", CuerpoNegocio(nn.nombre, nn.descripcion, nn.direccion, nn.ciudad, nn.telefono, nn.whatsapp, nn.instagram))
                estado.api.hacer("PATCH", "/v1/panel/mi-web", CuerpoWeb(ww.rubro, ww.provincia, ww.acerca, ww.colorPrimario, ww.colorAcento, ww.email, ww.sitioWeb, ww.facebook, ww.tiktok))
            }
            if (ok) runCatching { estado.cargarSesion() }
        }
        Bloque("Fotos (${fotos.size})", "Mantené apretada una foto para moverla o sacarla.", {
            val celdas: List<FotoGaleria?> = fotos.sortedBy { it.orden } + listOf(null)
            Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                celdas.chunked(3).forEach { fila ->
                    Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                        fila.forEach { f ->
                            Box(Modifier.weight(1f)) {
                                if (f != null) Box(Modifier.combinedClickableCompat(onLongClick = { Haptico.seleccion(); menuFoto = f })) { Imagen(f.url, 104.dp, null, 10.dp) }
                                else Hueco(Modifier.fillMaxWidth().height(104.dp).clickable { pedirFoto.launch(soloImagenes) }.semantics { contentDescription = "Sumar una foto" }, radio = 10.dp, quieto = true) {
                                    Icono("plus", 26f, C.mentaTexto)
                                }
                            }
                        }
                        repeat(3 - fila.size) { Spacer(Modifier.weight(1f)) }
                    }
                }
            }
        })
        IrA("paintpalette", "Plantilla y estilo", DestinoMas.plantillas)
        IrA("link", "Dominio propio", DestinoMas.dominio)
    }
    val mf = menuFoto
    Confirmacion(mf != null, "Foto", "", listOf(
        OpcionConfirmacion("Primera") { mf?.let { ordenar(it, true) } },
        OpcionConfirmacion("Última") { mf?.let { ordenar(it, false) } },
        OpcionConfirmacion("Sacar", destructiva = true) { mf?.let { f -> alcance.launch { estado.intentar { estado.api.hacer("DELETE", "/v1/panel/mi-web/galeria/${f.id}") }; cargar() } } },
    ), { menuFoto = null })
}

@OptIn(ExperimentalFoundationApi::class)
private fun Modifier.combinedClickableCompat(onLongClick: () -> Unit): Modifier = this.combinedClickable(onClick = {}, onLongClick = onLongClick)

// MARK: - Plantillas

val PLANTILLAS = listOf(
    Triple("carta", "Carta", "La portada con tu color y los servicios como una carta. Ideal si tenés muchos servicios."),
    Triple("vidriera", "Vidriera", "Una foto grande de punta a punta y servicios con foto. Para lugares que se lucen."),
    Triple("sobria", "Sobria", "Sin fotos: tu color, tu logo y tipografía grande. Para consultorios y quien no tiene fotos."),
    Triple("galeria", "Galería", "Mosaico de fotos arriba y filtros por categoría. Para uñas, peinados y tatuajes."),
    Triple("editorial", "Editorial", "El nombre en grande como tapa de revista y una foto chica. Para estudios con personalidad."),
    Triple("bloques", "Bloques", "Todo en una grilla de bloques: nombre, fotos, reseñas y precios a la vista de un golpe."),
)
private val AJUSTES_PLANTILLA = listOf(
    Triple("portada", "Portada", listOf("foto" to "Con foto", "color" to "Sólo color")),
    Triple("fondo", "Fondo", listOf("crema" to "Crema", "blanco" to "Blanco", "noche" to "Noche")),
    Triple("titulos", "Títulos", listOf("grandes" to "Grandes", "medianos" to "Medianos", "livianos" to "Livianos")),
    Triple("forma", "Bordes", listOf("redondos" to "Redondos", "suaves" to "Suaves", "rectos" to "Rectos")),
)
private val SECCIONES_WEB = listOf("servicios" to "Servicios", "resenas" to "Reseñas", "equipo" to "Equipo", "fotos" to "Fotos", "membresias" to "Membresías")

@Composable
fun Plantillas() {
    val estado = LocalEstado.current
    val estilo = remember { mutableStateMapOf<String, JsonElement>() }
    var cargando by remember { mutableStateOf(true) }
    fun valor(k: String) = (estilo[k] as? JsonPrimitive)?.takeIf { it.isString }?.content.orEmpty()
    val secciones = (estilo["secciones"] as? JsonArray)?.mapNotNull { it.jsonPrimitive.contentOrNull }
    LaunchedEffect(Unit) {
        (estado.api.opcional<WebDatos>("/v1/panel/mi-web")?.estilo as? JsonObject)?.let { estilo.clear(); estilo.putAll(it) }
        cargando = false
    }
    @Serializable data class Cuerpo(val estilo: JsonObject)
    PantallaMas("Plantillas", PLANTILLAS.firstOrNull { it.first == valor("plantilla") }?.second.orEmpty(), volver = "Más", cargando = cargando, recargar = {}) {
        PLANTILLAS.forEach { (k, t, d) ->
            val on = valor("plantilla") == k
            Row(Modifier.fillMaxWidth().tarjeta().then(if (on) Modifier.border(2.dp, C.menta, RoundedCornerShape(Radio.tarjeta)) else Modifier)
                .clickable { estilo["plantilla"] = JsonPrimitive(k) }.padding(16.dp), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                Icono(if (on) "checkmark.circle.fill" else "circle", 22f, if (on) C.mentaTexto else C.linea)
                Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                    Texto(t, Tipo.lato(17f, Peso.Bold), C.tinta)
                    Texto(d, Tipo.lato(14f), C.sub)
                }
            }
        }
        Bloque("Ajustes", "", *AJUSTES_PLANTILLA.map { (k, t, ops) -> @Composable {
            Renglon(t) { SelectorMenu(ops, valor(k), { estilo[k] = JsonPrimitive(it) }, t) }
        } }.toTypedArray())
        if (secciones != null) Bloque("Secciones", "Las prendidas se muestran en tu página, en este orden.", *SECCIONES_WEB.map { (k, t) -> @Composable {
            RenglonInterruptor(t, k in secciones, { on -> estilo["secciones"] = JsonArray((if (on) secciones + k else secciones - k).map { JsonPrimitive(it) }) })
        } }.toTypedArray())
        GuardarSuelto("Publicar") { estado.intentar("Publicada. Tus clientes ya la ven.") { estado.api.hacer("PATCH", "/v1/panel/mi-web", Cuerpo(JsonObject(estilo.toMap()))) } }
        BotonContorno("Ver cómo quedó", { estado.abrirPestana(Web.pagina(estado.negocio?.slug.orEmpty())) }, alto = 48.dp, tam = 15f, simbolo = "safari")
    }
}

// MARK: - Dominio

@Composable
fun DominioVista() {
    @Serializable data class Registro(val tipo: String = "", val nombre: String = "", val valor: String = "", val alternativa: Boolean? = null)
    @Serializable data class Dominio(val nombre: String = "", val estado: String = "", val detalle: String = "", val registros: List<Registro> = emptyList())
    @Serializable data class Datos(val dominio: Dominio? = null, val destino: String = "", val ip: String = "")
    @Serializable data class Cuerpo(val dominio: String)
    val estado = LocalEstado.current
    val contexto = LocalContext.current
    val alcance = rememberCoroutineScope()
    var d by remember { mutableStateOf<Datos?>(null) }
    var nuevo by remember { mutableStateOf("") }
    var error by remember { mutableStateOf("") }
    suspend fun cargar() { try { d = estado.api.get("/v1/panel/dominio"); error = "" } catch (e: ErrorApi) { error = e.detalle } catch (e: Exception) { error = "Probá de nuevo." } }
    LaunchedEffect(Unit) { cargar() }
    PantallaMas("Dominio propio", d?.dominio?.nombre ?: "Usá tu propia dirección", cargando = d == null && error.isEmpty(), error = error, recargar = { cargar() }) {
        val dom = d?.dominio
        if (dom != null) {
            val ok = dom.estado == "verificado"
            Bloque("", "", { Renglon(dom.nombre, if (ok) "Funcionando" else dom.detalle.ifEmpty { "Esperando que configures el DNS" }, if (ok) "checkmark.seal.fill" else "hourglass") })
            if (!ok) {
                Bloque("Cargá esto en tu proveedor de dominio", "", *dom.registros.map { r -> @Composable {
                    Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                        Texto("${r.tipo} · ${r.nombre}${if (r.alternativa == true) " (o este)" else ""}", Tipo.lato(13f, Peso.Bold), C.sub)
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Texto(r.valor, Tipo.lato(15f).copy(fontFamily = FontFamily.Monospace), C.tinta, Modifier.weight(1f))
                            Box(Modifier.size(40.dp).clickable { copiar(contexto, r.valor); Haptico.exito() }, contentAlignment = Alignment.Center) { Icono("doc.on.doc", 17f, C.mentaTexto, descripcion = "Copiar") }
                        }
                    }
                } }.toTypedArray())
                GuardarSuelto("Ya lo cargué, verificar") { estado.intentar { estado.api.pedir("POST", "/v1/panel/dominio/verificar", cuerpo = "{}".toByteArray()) }; cargar() }
            }
            BotonContornoRojo("Sacar el dominio", "xmark.circle") { alcance.launch { estado.intentar("Sacamos el dominio") { estado.api.hacer("DELETE", "/v1/panel/dominio") }; cargar() } }
        } else {
            Bloque("", "Comprá el dominio donde quieras (NIC Argentina, GoDaddy…) y después lo apuntás a Tengo Libre.",
                { RenglonCampo("Tu dominio", nuevo, { nuevo = it }, "turnos.tunegocio.com.ar", teclado = KeyboardType.Uri) })
            GuardarSuelto("Usar este dominio", "." in nuevo) { estado.intentar { estado.api.hacer("PUT", "/v1/panel/dominio", Cuerpo(nuevo.lowercase().trim())) }; cargar() }
        }
    }
}

// MARK: - Integraciones e IA

@Composable
fun Integraciones() {
    @Serializable data class Calendario(val id: String? = null, val profesionalId: String? = null, val nombre: String = "", val url: String? = null)
    @Serializable data class Token(val id: String, val nombre: String = "", val pista: String = "", val permisos: List<String> = emptyList(), val cliente: String = "", val creadoEn: FechaApi = Instant.EPOCH, val usadoEn: FechaApi? = null)
    @Serializable data class Permiso(val clave: String, val nombre: String = "", val descripcion: String = "")
    @Serializable data class Mcp(val url: String = "", val tokens: List<Token> = emptyList(), val permisos: List<Permiso> = emptyList())
    @Serializable data class IaPublica(val activo: Boolean = false, val reservasMes: Int = 0, val url: String = "")
    @Serializable data class Creado(val token: Token, val clave: String, val url: String = "")
    @Serializable data class Cals(val calendarios: List<Calendario> = emptyList())
    @Serializable data class ParaProfesional(val profesionalId: String?)
    @Serializable data class Activo(val activo: Boolean)
    @Serializable data class Nombre(val nombre: String)
    val estado = LocalEstado.current
    val contexto = LocalContext.current
    val alcance = rememberCoroutineScope()
    var calendarios by remember { mutableStateOf<List<Calendario>>(emptyList()) }
    var mcp by remember { mutableStateOf<Mcp?>(null) }
    var ia by remember { mutableStateOf<IaPublica?>(null) }
    var cargando by remember { mutableStateOf(true) }
    var nombreClave by remember { mutableStateOf("Claude") }
    var creada by remember { mutableStateOf<Creado?>(null) }
    suspend fun cargar() {
        estado.api.opcional<Cals>("/v1/panel/calendarios")?.let { calendarios = it.calendarios }
        if (estado.gestiona) { mcp = estado.api.opcional("/v1/panel/mcp"); ia = estado.api.opcional("/v1/panel/ia/publico") }
        cargando = false
    }
    LaunchedEffect(Unit) { cargar() }
    PantallaMas(if (estado.esProfesional) "Tu calendario" else "Integraciones e IA", cargando = cargando, recargar = { cargar() }) {
        Bloque("Calendario", "Suscribite desde Google Calendar, Apple o Outlook y los turnos aparecen solos (se actualiza cada tanto).", *calendarios.map { c -> @Composable {
            val u = c.url
            if (u != null) Renglon(c.nombre, simbolo = "calendar") {
                Row(horizontalArrangement = Arrangement.spacedBy(4.dp)) {
                    Box(Modifier.size(40.dp).clickable { copiar(contexto, u); estado.avisar("Copiamos el link") }, contentAlignment = Alignment.Center) { Icono("doc.on.doc", 17f, C.mentaTexto, descripcion = "Copiar link") }
                    Box(Modifier.size(40.dp).clickable { estado.abrirLink(u.replace("https://", "webcal://")) }, contentAlignment = Alignment.Center) { Icono("calendar.badge.plus", 17f, C.mentaTexto, descripcion = "Agregar al calendario del teléfono") }
                }
            } else Box(Modifier.clickable { alcance.launch { estado.intentar { estado.api.hacer("POST", "/v1/panel/calendarios", ParaProfesional(c.profesionalId)) }; cargar() } }) {
                Renglon(c.nombre, "Crear el link", "plus.circle")
            }
        } }.toTypedArray())
        if (estado.gestiona) {
            ia?.let { x ->
                Bloque("Que te reserven desde ChatGPT y Claude", "Tus servicios y horarios libres quedan disponibles para los asistentes de IA. Este mes: ${Formato.cantidad(x.reservasMes, "reserva", "reservas")}.", {
                    RenglonInterruptor("Reservas por IA", x.activo, { v -> alcance.launch { estado.intentar { estado.api.hacer("PUT", "/v1/panel/ia/publico", Activo(v)) }; cargar() } })
                })
            }
            mcp?.let { m ->
                Bloque("Manejá tu agenda con Claude o ChatGPT", "Creá una clave y pegala en el conector del asistente. Ves y movés turnos hablando.", *(listOf<@Composable () -> Unit>({
                    Renglon("Dirección del conector", m.url) { Box(Modifier.size(40.dp).clickable { copiar(contexto, m.url); Haptico.exito() }, contentAlignment = Alignment.Center) { Icono("doc.on.doc", 17f, C.mentaTexto) } }
                }) + m.tokens.map { t -> @Composable {
                    Renglon(t.nombre, "…${t.pista} · ${t.cliente.ifEmpty { "sin usar" }}${t.usadoEn?.let { " · ${Formato.hace(it)}" } ?: ""}") {
                        Box(Modifier.size(44.dp).clickable { alcance.launch { estado.intentar("Borramos la clave") { estado.api.hacer("DELETE", "/v1/panel/mcp/tokens/${t.id}") }; cargar() } }, contentAlignment = Alignment.Center) {
                            Icono("trash", 17f, C.error, descripcion = "Borrar la clave")
                        }
                    }
                } } + listOf(@Composable {
                    Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp), verticalAlignment = Alignment.CenterVertically) {
                        SelectorMenu(listOf("Claude", "ChatGPT", "Gemini", "Otro (MCP)").map { it to it }, nombreClave, { nombreClave = it }, "Para")
                        Spacer(Modifier.weight(1f))
                        BotonMentaChico("Crear clave", { alcance.launch { try { creada = estado.api.enviar("POST", "/v1/panel/mcp/tokens", Nombre(nombreClave)); cargar() } catch (e: Exception) { estado.avisar(e) } } })
                    }
                })).toTypedArray())
            }
        }
    }
    Hoja(creada != null, { creada = null }, alturas = listOf(Altura.media)) {
        val c = creada
        BarraHoja("Tu clave", { creada = null }, textoCancelar = "", accion = "Listo") { creada = null }
        if (c != null) Column(Modifier.background(C.hoja).padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
            Texto("Copiala ahora: no la vamos a mostrar de nuevo.", Tipo.lato(15f), C.sub)
            Box(Modifier.fillMaxWidth().clip(RoundedCornerShape(12.dp)).background(C.chip).padding(14.dp)) { Texto(c.clave, Tipo.lato(15f).copy(fontFamily = FontFamily.Monospace), C.tinta) }
            BotonMenta("Copiar la clave", { copiar(contexto, c.clave); Haptico.exito() }, alto = 52.dp, conSombra = false, simbolo = "doc.on.doc")
        }
    }
}

// MARK: - Configuración

@Composable
fun Configuracion() {
    @Serializable data class Ajustes(val intervalo: Int = 30, val anticipacionMin: Int = 0, val diasVista: Int = 30, val margenMin: Int = 0, val cancelarHastaHoras: Int = 0, val politicaCancelacion: String = "")
    @Serializable data class Avisos(val resenaDemoraHoras: Int = 2, val resumenDiario: Boolean = true)
    @Serializable data class Espera(val minutos: Int = 30, val automatica: Boolean = true)
    @Serializable data class Encargado(val id: String, val nombre: String = "", val email: String = "", val recibe: Boolean = false)
    @Serializable data class Mensual(val activo: Boolean = false, val encargados: List<Encargado> = emptyList())
    val estado = LocalEstado.current
    var a by remember { mutableStateOf<Ajustes?>(null) }
    var av by remember { mutableStateOf(Avisos()) }
    var es by remember { mutableStateOf(Espera()) }
    var me by remember { mutableStateOf<Mensual?>(null) }
    var error by remember { mutableStateOf("") }
    suspend fun cargar() {
        try {
            a = estado.api.get("/v1/panel/ajustes")
            estado.api.opcional<Avisos>("/v1/panel/ajustes/avisos")?.let { av = it }
            estado.api.opcional<Espera>("/v1/panel/ajustes/espera")?.let { es = it }
            me = estado.api.opcional("/v1/panel/reportes/mensual/ajustes")
            error = ""
        } catch (e: ErrorApi) { error = e.detalle } catch (e: Exception) { error = "Probá de nuevo." }
    }
    LaunchedEffect(Unit) { cargar() }
    PantallaMas("Configuración", "Reservas, cancelaciones y avisos", cargando = a == null && error.isEmpty(), error = error, recargar = { cargar() }) {
        val b = a ?: return@PantallaMas
        Bloque("Reservas", "",
            { Numerito("Horarios cada", b.intervalo, { a = b.copy(intervalo = it) }, 5..120, paso = 5) { "$it min" } },
            { Numerito("Reservar con anticipación", b.anticipacionMin / 60, { a = b.copy(anticipacionMin = it * 60) }, 0..72, "Lo mínimo antes del turno") { if (it == 0) "Sin mínimo" else "$it h" } },
            { Numerito("Se puede reservar hasta", b.diasVista, { a = b.copy(diasVista = it) }, 7..365, paso = 7) { "$it días" } },
            { Numerito("Margen entre turnos", b.margenMin, { a = b.copy(margenMin = it) }, 0..60, paso = 5) { if (it == 0) "Sin margen" else "$it min" } })
        Bloque("Cancelaciones", "",
            { Numerito("Cancelar hasta", b.cancelarHastaHoras, { a = b.copy(cancelarHastaHoras = it) }, 0..72, "Antes del turno, desde el link") { if (it == 0) "Siempre" else "$it h antes" } },
            { RenglonCampo("Política (se ve al reservar)", b.politicaCancelacion, { a = b.copy(politicaCancelacion = it) }, "Si no podés venir, avisanos 24 h antes.", multilinea = true) })
        GuardarSuelto { estado.intentar("Guardamos las reglas de reserva") { estado.api.hacer("PUT", "/v1/panel/ajustes", a) } }
        Bloque("Avisos", "",
            { Numerito("Pedir la reseña", av.resenaDemoraHoras, { av = av.copy(resenaDemoraHoras = it) }, 0..72, "Después del turno atendido") { if (it == 0) "No pedir" else "$it h después" } },
            { RenglonInterruptor("Resumen del día por mail", av.resumenDiario, { av = av.copy(resumenDiario = it) }, "A la mañana, con los turnos del día") },
            { GuardarEnBloque { estado.intentar("Guardamos los avisos") { estado.api.hacer("PUT", "/v1/panel/ajustes/avisos", av) } } })
        Bloque("Lista de espera", "Cuando se libera un horario, se lo ofrecemos a quien espera de a una persona por vez.",
            { RenglonInterruptor("Ofrecer solos los huecos", es.automatica, { es = es.copy(automatica = it) }) },
            { Numerito("Tiempo para tomarlo", es.minutos, { es = es.copy(minutos = it) }, 5..240, paso = 5) { "$it min" } },
            { GuardarEnBloque { estado.intentar("Guardamos la lista de espera") { estado.api.hacer("PUT", "/v1/panel/ajustes/espera", es) } } })
        me?.let { m ->
            Bloque("Reporte mensual por mail", "", *(listOf<@Composable () -> Unit>({ RenglonInterruptor("Mandarlo el 1° de cada mes", m.activo, { me = m.copy(activo = it) }) }) +
                m.encargados.mapIndexed { i, e -> @Composable { RenglonInterruptor(e.nombre, e.recibe, { v -> me = m.copy(encargados = m.encargados.toMutableList().also { it[i] = e.copy(recibe = v) }) }, e.email) } } +
                listOf(@Composable { GuardarEnBloque { estado.intentar("Guardamos el reporte mensual") { estado.api.hacer("PUT", "/v1/panel/reportes/mensual/ajustes", me) } } })).toTypedArray())
        }
        IrA("message", "WhatsApp", DestinoMas.whatsapp)
        IrA("exclamationmark.shield", "Ausencias y penalidades", DestinoMas.penalidades)
        IrA("list.bullet.rectangle", "Actividad", DestinoMas.auditoria)
    }
}

// MARK: - Recordatorios a clientes

@Composable
fun AvisosClientes() {
    @Serializable data class Semana(val clienteId: String = "", val nombre: String = "", val fecha: String = "", val detalle: String = "", val enviado: Boolean = false)
    @Serializable data class Recordatorio(val tipo: String, val titulo: String = "", val activo: Boolean = false, val texto: String = "", val cuponId: String? = null, val dias: Int = 60,
        val regla: String = "", val semana: List<Semana> = emptyList(), val semanaTexto: String = "")
    @Serializable data class Rs(val recordatorios: List<Recordatorio> = emptyList())
    @Serializable data class Cs(val cupones: List<Cupon> = emptyList())
    @Serializable data class Cuerpo(val activo: Boolean, val texto: String, val cuponId: String, val dias: Int?)
    val estado = LocalEstado.current
    var lista by remember { mutableStateOf<List<Recordatorio>>(emptyList()) }
    var cupones by remember { mutableStateOf<List<Cupon>>(emptyList()) }
    var cargando by remember { mutableStateOf(true) }
    suspend fun cargar() {
        estado.api.opcional<Rs>("/v1/panel/recordatorios")?.let { lista = it.recordatorios }
        estado.api.opcional<Cs>("/v1/panel/cupones")?.let { cupones = it.cupones }
        cargando = false
    }
    LaunchedEffect(Unit) { cargar() }
    PantallaMas("Recordatorios", "Mails automáticos a tus clientas", cargando = cargando, recargar = { cargar() }) {
        lista.forEachIndexed { i, r ->
            fun cambiar(x: Recordatorio) { lista = lista.toMutableList().also { it[i] = x } }
            Bloque(r.titulo, "${r.regla} ${r.semanaTexto}", *buildList<@Composable () -> Unit> {
                add { RenglonInterruptor(if (r.activo) "Prendido" else "Apagado", r.activo, { cambiar(r.copy(activo = it)) }) }
                if (r.tipo == "extraniamos") add { Numerito("Si no viene hace", r.dias, { cambiar(r.copy(dias = it)) }, 15..365, paso = 15) { "$it días" } }
                add { RenglonCampo("Mensaje", r.texto, { cambiar(r.copy(texto = it)) }, multilinea = true) }
                add { Renglon("Con cupón") { SelectorMenu(listOf("" to "Sin cupón") + cupones.filter { it.activo && it.referidoId == null }.map { it.id to it.codigo }, r.cuponId.orEmpty(), { cambiar(r.copy(cuponId = it.ifEmpty { null })) }) } }
                add { GuardarEnBloque { val x = lista[i]; estado.intentar("Guardamos el recordatorio") { estado.api.hacer("PUT", "/v1/panel/recordatorios/${x.tipo}", Cuerpo(x.activo, x.texto, x.cuponId.orEmpty(), if (x.tipo == "extraniamos") x.dias else null)) } } }
            }.toTypedArray())
        }
    }
}

// MARK: - WhatsApp

@Composable
fun WhatsAppVista() {
    @Serializable data class Ajustes(val disponible: Boolean = false, val confirmacion: Boolean = false, val hora: String = "10:00", val sinRespuestaHoras: Int = 3, val recordatorio: Boolean = false)
    @Serializable data class Mensaje(val id: String, val tipo: String = "", val cliente: String = "", val estado: String = "", val respuesta: String? = null, val creadoEn: FechaApi = Instant.EPOCH, val ultimoError: String? = null)
    @Serializable data class Ms(val mensajes: List<Mensaje> = emptyList())
    @Serializable data class Cuerpo(val confirmacion: Boolean, val hora: String, val sinRespuestaHoras: Int, val recordatorio: Boolean)
    val estado = LocalEstado.current
    var a by remember { mutableStateOf<Ajustes?>(null) }
    var mensajes by remember { mutableStateOf<List<Mensaje>>(emptyList()) }
    var error by remember { mutableStateOf("") }
    suspend fun cargar() {
        try { a = estado.api.get("/v1/panel/ajustes/whatsapp"); mensajes = estado.api.opcional<Ms>("/v1/panel/whatsapp/mensajes")?.mensajes ?: emptyList(); error = "" }
        catch (e: ErrorApi) { error = e.detalle } catch (e: Exception) { error = "Probá de nuevo." }
    }
    LaunchedEffect(Unit) { cargar() }
    PantallaMas("WhatsApp", if (a?.disponible == false) "Todavía no disponible" else "Confirmaciones automáticas", volver = "Más", cargando = a == null && error.isEmpty(), error = error, recargar = { cargar() }) {
        val b = a ?: return@PantallaMas
        Box(Modifier.then(if (!b.disponible) Modifier.alphaDeshabilitado() else Modifier)) {
            Bloque("", "El día antes le pedimos a cada clienta que confirme. Si no contesta, aparece en «Para atender» para que la llames.",
                { RenglonInterruptor("Pedir confirmación", b.confirmacion, { a = b.copy(confirmacion = it) }) },
                { Numerito("A las", b.hora.take(2).toIntOrNull() ?: 10, { a = b.copy(hora = "%02d:00".format(it)) }, 7..21) { "%02d:00".format(it) } },
                { Numerito("Si no contesta, avisarte a las", b.sinRespuestaHoras, { a = b.copy(sinRespuestaHoras = it) }, 1..24) { "$it h" } },
                { RenglonInterruptor("Recordatorio el mismo día", b.recordatorio, { a = b.copy(recordatorio = it) }) },
                { GuardarEnBloque { val x = a ?: return@GuardarEnBloque; estado.intentar("Guardamos WhatsApp") { estado.api.hacer("PUT", "/v1/panel/ajustes/whatsapp", Cuerpo(x.confirmacion, x.hora, x.sinRespuestaHoras, x.recordatorio)) } } })
        }
        if (mensajes.isNotEmpty()) Bloque("Últimos mensajes", "", *mensajes.take(30).map { m -> @Composable {
            Renglon(m.cliente, "${mapOf("confirmacion" to "Confirmación", "recordatorio" to "Recordatorio", "respuesta" to "Respuesta")[m.tipo] ?: m.tipo} · ${Formato.hace(m.creadoEn)}") {
                val t = when (m.respuesta) { "confirmo" -> "Confirmó"; "cambiar" -> "Quiere cambiar"; else -> mapOf("enviado" to "Enviado", "pendiente" to "En cola", "fallido" to "No salió", "descartado" to "Descartado")[m.estado] ?: m.estado }
                Texto(t, Tipo.lato(13f, Peso.Bold), if (m.estado == "fallido") C.error else C.mentaTexto)
            }
        } }.toTypedArray())
    }
}

/** .disabled(true) de SwiftUI: atenuado y sin toques. */
private fun Modifier.alphaDeshabilitado(): Modifier = this.alpha(0.45f).pointerInput(Unit) {
    awaitPointerEventScope { while (true) awaitPointerEvent(PointerEventPass.Initial).changes.forEach { it.consume() } }
}

// MARK: - Actividad

@Composable
fun Auditoria() {
    @Serializable data class Entrada(val id: String, val nombre: String = "", val accion: String = "", val entidad: String = "", val creadoEn: FechaApi = Instant.EPOCH)
    @Serializable data class Es(val entradas: List<Entrada> = emptyList())
    val estado = LocalEstado.current
    var entradas by remember { mutableStateOf<List<Entrada>>(emptyList()) }
    var cargando by remember { mutableStateOf(true) }
    suspend fun cargar() { estado.api.opcional<Es>("/v1/panel/auditoria", listOf("limite" to "100"))?.let { entradas = it.entradas }; cargando = false }
    LaunchedEffect(Unit) { cargar() }
    fun texto(e: Entrada): String {
        val verbo = mapOf("crear" to "creó", "editar" to "editó", "borrar" to "borró", "cancelar" to "canceló", "cobrar" to "cobró", "mover" to "movió", "estado" to "cambió el estado de",
            "invitar" to "invitó a", "aprobar" to "aprobó", "rechazar" to "rechazó")
        val v = e.accion.split(".").lastOrNull()?.let { verbo[it] } ?: e.accion
        val cosa = mapOf("turno" to "un turno", "servicio" to "un servicio", "profesional" to "una profesional", "cliente" to "una clienta", "cupon" to "un cupón", "ajustes" to "los ajustes",
            "miembro" to "a alguien del equipo", "membresia" to "una membresía")[e.entidad] ?: e.entidad
        return "$v $cosa"
    }
    PantallaMas("Actividad", "Quién hizo qué en el panel", cargando = cargando, recargar = { cargar() }) {
        entradas.groupBy { Formato.seccion(it.creadoEn) }.forEach { (s, l) ->
            Bloque(s, "", *l.map { e -> @Composable { Renglon("${e.nombre.ifEmpty { "Alguien" }} ${texto(e)}", Formato.horaDe(e.creadoEn)) } }.toTypedArray())
        }
        if (entradas.isEmpty() && !cargando) Texto("Todavía no hay actividad.", Tipo.lato(14f), C.sub)
    }
}

// MARK: - Sucursales

@Composable
fun Sucursales() {
    @Serializable data class Sucursal(val id: String, val slug: String = "", val nombre: String = "", val ciudad: String = "")
    @Serializable data class Grupo(val id: String, val nombre: String = "", val sucursales: List<Sucursal> = emptyList())
    @Serializable data class Grupos(val grupos: List<Grupo> = emptyList(), val sinGrupo: List<Sucursal> = emptyList())
    @Serializable data class Numeros(val turnos: Int = 0, val atendidos: Int = 0, val ausentes: Int = 0, val cancelados: Int = 0, val clientes: Int = 0, val ingresos: Int = 0)
    @Serializable data class Reporte(val total: Numeros = Numeros())
    @Serializable data class Nombre(val nombre: String)
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    var g by remember { mutableStateOf<Grupos?>(null) }
    val totales = remember { mutableStateMapOf<String, Numeros>() }
    var nombre by remember { mutableStateOf("") }
    var error by remember { mutableStateOf("") }
    suspend fun cargar() {
        try {
            val x: Grupos = estado.api.get("/v1/grupos"); g = x
            val hoy = Formato.enArgentina(Instant.now()).toLocalDate()
            x.grupos.forEach { gr -> estado.api.opcional<Reporte>("/v1/grupos/${gr.id}/reporte", listOf("desde" to hoy.withDayOfMonth(1).toString(), "hasta" to hoy.toString()))?.let { totales[gr.id] = it.total } }
            error = ""
        } catch (e: ErrorApi) { error = e.detalle } catch (e: Exception) { error = "Probá de nuevo." }
    }
    LaunchedEffect(Unit) { cargar() }
    PantallaMas("Sucursales", "Varios locales, una sola mirada", cargando = g == null && error.isEmpty(), error = error, recargar = { cargar() }) {
        val x = g ?: return@PantallaMas
        x.grupos.forEach { gr ->
            Bloque(gr.nombre, totales[gr.id]?.let { "Este mes: ${it.atendidos} atendidos · ${Formato.pesos(it.ingresos)}" } ?: "", *(gr.sucursales.map { s -> @Composable {
                Renglon(s.nombre, s.ciudad) { BotonTexto("Sacar", { alcance.launch { estado.intentar { estado.api.hacer("DELETE", "/v1/grupos/${gr.id}/sucursales/${s.id}") }; cargar() } }, color = C.error) }
            } } + (if (x.sinGrupo.isNotEmpty()) listOf(@Composable {
                MenuBoton(x.sinGrupo.map { s -> OpcionDeMenu(s.nombre, "building.2") { alcance.launch { estado.intentar { estado.api.hacer("PUT", "/v1/grupos/${gr.id}/sucursales/${s.id}", emptyMap<String, String>()) }; cargar() } } }, "Sumar un local") {
                    Renglon("Sumar un local", simbolo = "plus.circle")
                }
            }) else emptyList())).toTypedArray())
        }
        Bloque("Nuevo grupo", "Para ver juntos los números de varios locales tuyos.", {
            Row(Modifier.padding(16.dp), verticalAlignment = Alignment.CenterVertically) {
                Box(Modifier.weight(1f)) {
                    if (nombre.isEmpty()) Texto("Nombre del grupo", Tipo.lato(16f), C.sub.copy(alpha = 0.7f))
                    val tinta = C.tinta
                    BasicTextField(nombre, { nombre = it }, Modifier.fillMaxWidth(), singleLine = true, textStyle = Tipo.lato(16f).copy(color = tinta), cursorBrush = SolidColor(C.mentaTexto))
                }
                Box(Modifier.then(if (nombre.length < 2) Modifier.alphaDeshabilitado() else Modifier)) {
                    BotonMentaChico("Crear", { if (nombre.length >= 2) alcance.launch { estado.intentar { estado.api.hacer("POST", "/v1/grupos", Nombre(nombre)) }; nombre = ""; cargar() } })
                }
            }
        })
        Texto("Para abrir otro local, entrá desde la compu a tengolibre.com.ar/empezar con esta misma cuenta.", Tipo.lato(13f), C.sub)
    }
}

// MARK: - Qué avisos llegan al teléfono

/** Qué avisos llegan al teléfono, por tipo y por negocio. En iOS es un Form con barra de navegación: acá, la misma
 *  lista agrupada con «‹ Atrás» y el título al medio. */
@Composable
fun PreferenciasPush(volver: () -> Unit) {
    @Serializable data class TipoPush(val tipo: String, val nombre: String = "", val activo: Boolean = true)
    @Serializable data class Prefs(val rol: String = "", val porDefecto: Boolean = true, val apagados: List<String> = emptyList(), val tipos: List<TipoPush> = emptyList())
    @Serializable data class Apagados(val apagados: List<String>)
    @Serializable data class PorDefecto(val porDefecto: Boolean)
    val estado = LocalEstado.current
    val contexto = LocalContext.current
    val alcance = rememberCoroutineScope()
    var p by remember { mutableStateOf<Prefs?>(null) }
    val permitido = remember { EstadoNegocio.hayAvisos() }
    suspend fun cargar() { p = estado.api.opcional("/v1/panel/push/preferencias") }
    LaunchedEffect(Unit) { cargar() }
    BackHandler { volver() }
    fun cambiar(tipo: String, on: Boolean) = alcance.launch {
        val x = p ?: return@launch
        val ap = if (on) x.apagados - tipo else if (tipo in x.apagados) x.apagados else x.apagados + tipo
        p = x.copy(apagados = ap)
        estado.intentar { estado.api.hacer("PUT", "/v1/panel/push/preferencias", Apagados(ap)) }
    }
    Column(Modifier.fillMaxSize().background(C.fondo)) {
        Box(Modifier.fillMaxWidth().statusBarsPadding().height(44.dp).padding(horizontal = 8.dp)) {
            Row(Modifier.align(Alignment.CenterStart).clickable(onClick = volver).padding(end = 8.dp), verticalAlignment = Alignment.CenterVertically) {
                Icono("chevron.left", 20f, C.mentaTexto); Texto("Atrás", Tipo.lato(17f), C.mentaTexto)
            }
            Texto("Qué te avisamos", Tipo.lato(17f, Peso.Bold), C.tinta, Modifier.align(Alignment.Center))
        }
        Column(Modifier.weight(1f).verticalScroll(rememberScrollState()).padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
            if (!permitido) Bloque("", "Los avisos están apagados para Tengo Libre en este teléfono.", {
                Box(Modifier.fillMaxWidth().heightIn(min = 48.dp).clickable {
                    contexto.startActivity(Intent(Settings.ACTION_APP_NOTIFICATION_SETTINGS).putExtra(Settings.EXTRA_APP_PACKAGE, contexto.packageName))
                }.padding(horizontal = 16.dp), contentAlignment = Alignment.CenterStart) { Texto("Activar los avisos en Ajustes", Tipo.lato(16f), C.mentaTexto) }
            })
            val x = p
            if (x == null) Box(Modifier.fillMaxWidth().padding(24.dp), contentAlignment = Alignment.Center) { Pulso() }
            else {
                Bloque("", "Vale para este negocio. Lo que apagues lo seguís viendo en la campanita.", *x.tipos.map { t -> @Composable {
                    RenglonInterruptor(t.nombre, t.tipo !in x.apagados, { cambiar(t.tipo, it) })
                } }.toTypedArray())
                if (!x.porDefecto) Bloque("", "", {
                    Box(Modifier.fillMaxWidth().heightIn(min = 48.dp).clickable {
                        alcance.launch { estado.intentar { estado.api.hacer("PUT", "/v1/panel/push/preferencias", PorDefecto(true)) }; cargar() }
                    }.padding(horizontal = 16.dp), contentAlignment = Alignment.CenterStart) { Texto("Volver a lo recomendado", Tipo.lato(16f), C.mentaTexto) }
                })
            }
        }
    }
}
