package ar.com.tengolibre.app.pantallas.hoy

import ar.com.tengolibre.app.nucleo.Formato
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.SpanStyle
import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import ar.com.tengolibre.app.app.Estado
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.nucleo.*
import ar.com.tengolibre.app.pantallas.principal.LocalNavegacion
import ar.com.tengolibre.app.tema.*
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable
import java.text.Normalizer
import java.time.Instant
import java.time.temporal.ChronoUnit

// Clon de Pantallas/Hoy/TuLink.swift.

/** El link de la página (tengolibre.com.ar/<slug>): es lo que trae las reservas, así que va arriba de Hoy mientras el
 *  negocio tiene pocos turnos, con cuatro acciones grandes (Copiar, WhatsApp, Instagram, QR) y «Cambiar link». La misma
 *  regla y los mismos textos que la web (TuLink.svelte). */
object LinkRegla {
    /** Con menos turnos que esto en los últimos 30 días, el link va primero y en grande. */
    const val pocosTurnos = 20

    @Serializable private data class Turnos(val turnos: List<Turno>)

    /** Cuántos turnos hubo en los últimos 30 días (hasta `pocosTurnos`: no hace falta contar más). */
    suspend fun movimiento(estado: Estado, hoy: Instant): Int? {
        val q = listOf("desde" to Formato.diaISO(hoy.minus(29, ChronoUnit.DAYS)), "hasta" to Formato.diaISO(hoy),
            "estado" to "pendiente_pago,pendiente,confirmado,atendido,ausente", "limite" to pocosTurnos.toString())
        return estado.api.opcional<Turnos>("/v1/panel/turnos", q)?.turnos?.size
    }

    /** Mientras se escribe: minúsculas, sin acentos, espacios a guiones (el resto lo valida la API). */
    fun limpiar(s: String): String {
        val base = Normalizer.normalize(s, Normalizer.Form.NFD).replace(Regex("\\p{M}"), "").lowercase()
        val out = StringBuilder()
        for (c in base) {
            if ((c.code < 128 && c.isLetterOrDigit()) || c == '-') out.append(c)
            else if (c == ' ' || c == '_' || c == '.') out.append('-')
        }
        return out.toString().take(48)
    }

    /** A medio escribir: termina en guion o tiene dos seguidos («peluqueria-»). No es un error todavía:
     *  el campo dice «Seguí escribiendo…» y no se le pregunta a la API. */
    fun aMedias(s: String): Boolean = s.endsWith("-") || s.contains("--")

    /** El formato que acepta la API (negocios.ValidarSlug). */
    fun valido(s: String): Boolean = Regex("^[a-z0-9]([a-z0-9-]{0,46}[a-z0-9])?$").matches(s) && !s.contains("--")
}

/** El consejo de la primera vez: señala el link. */
val consejoLink = Consejo("link", "Este es tu link", "Compartilo por WhatsApp o Instagram y te reservan solos.", "link")

private val host: String get() = "${Web.base.removePrefix("https://").removePrefix("http://")}/"

@Composable
fun TarjetaLink(compacta: Boolean = false) {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    val contexto = LocalContext.current
    val alcance = rememberCoroutineScope()
    var copiado by remember { mutableStateOf(false) }
    var cambiar by remember { mutableStateOf(false) }
    val slug = estado.negocio?.slug.orEmpty()
    val url = Web.pagina(slug)

    fun copiar() {
        (contexto.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager).setPrimaryClip(ClipData.newPlainText("link", url))
        Haptico.exito()
        copiado = true
        alcance.launch { delay(2000); copiado = false }
    }
    fun abrir(u: String) = runCatching { contexto.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(u))) }
    fun whatsapp() {
        val t = "Reservá tu turno en ${estado.negocio?.nombre.orEmpty()}: $url"
        val i = Intent(Intent.ACTION_SEND).setType("text/plain").putExtra(Intent.EXTRA_TEXT, t).setPackage("com.whatsapp")
        if (i.resolveActivity(contexto.packageManager) != null) contexto.startActivity(i) else abrir("https://wa.me/?text=${Uri.encode(t)}")
    }
    /** Instagram: el link copiado para pegarlo en la bio, y se abre la edición del perfil. */
    fun instagram() {
        copiar()
        estado.avisar("Copiamos tu link", "Pegalo en tu bio de Instagram.")
        abrir("https://www.instagram.com/accounts/edit/")
    }

    Column(
        Modifier.fillMaxWidth().tarjeta(Radio.bloque)
            .border(if (compacta) 1.dp else 2.dp, if (compacta) C.linea else C.menta, RoundedCornerShape(Radio.bloque))
            .padding(if (compacta) 16.dp else 18.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
            Icono("link", 12f, C.mentaTexto); TextoRotulo("Tu link", C.mentaTexto)
        }
        ConConsejo(true, consejoLink) {
            Column {
                Texto(host, Tipo.lato(if (compacta) 13f else 15f, Peso.Bold), C.sub, lineas = 1)
                Texto(slug, Tipo.lato(if (compacta) 22f else 28f, Peso.Black), C.mentaTexto, lineas = 2, tracking = (-0.6).sp)
            }
        }
        if (!compacta) Texto("Compartilo y te reservan solos.", Tipo.lato(15f), C.sub)
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            AccionLink(if (copiado) "Copiado" else "Copiar", if (copiado) "checkmark" else "doc.on.doc", principal = true) { copiar() }
            AccionLink("WhatsApp", "message.fill") { whatsapp() }
            AccionLink("Instagram", "camera.fill") { instagram() }
            AccionLink("QR", "qrcode") { nav.compartir = true }
        }
        Row(Modifier.heightIn(min = 40.dp), horizontalArrangement = Arrangement.spacedBy(18.dp), verticalAlignment = Alignment.CenterVertically) {
            if (estado.gestiona) EnlaceLink("Cambiar link", "pencil") { cambiar = true }
            EnlaceLink("Ver mi página", "arrow.up.right") { abrir(url) }
        }
    }
    Hoja(cambiar, { cambiar = false }) { CambiarLink { cambiar = false } }
}

/** Cuatro acciones del mismo tamaño: ícono arriba, palabra abajo, 64 dp para el pulgar. */
@Composable
private fun androidx.compose.foundation.layout.RowScope.AccionLink(t: String, s: String, principal: Boolean = false, alTocar: () -> Unit) {
    Column(
        Modifier.weight(1f).heightIn(min = 64.dp).clip(RoundedCornerShape(18.dp)).background(if (principal) C.menta else C.chip)
            .clickable(onClick = alTocar).padding(vertical = 8.dp, horizontal = 2.dp),
        horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(4.dp, Alignment.CenterVertically),
    ) {
        Icono(s, 20f, if (principal) C.petroleoHondo else C.mentaTexto)
        Texto(t, Tipo.lato(13f, Peso.Bold), if (principal) C.petroleoHondo else C.tinta, lineas = 1)
    }
}

@Composable
private fun EnlaceLink(t: String, s: String, alTocar: () -> Unit) {
    Row(Modifier.heightIn(min = 40.dp).clickable(onClick = alTocar), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
        Icono(s, 14f, C.mentaTexto); Texto(t, Tipo.lato(14f, Peso.Bold), C.mentaTexto, lineas = 1)
    }
}

/** Cambiar el link: vista previa, si está libre mientras se escribe y el aviso de que el viejo sigue andando (la API lo
 *  guarda y la web redirige al nuevo). El mismo flujo que la web (EditorLink.svelte). */
@Composable
fun ColumnScope.CambiarLink(cerrar: () -> Unit) {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    val actual = estado.negocio?.slug.orEmpty()
    var nuevo by remember { mutableStateOf(actual) }
    var disponible by remember { mutableStateOf<String?>(null) }
    var link by remember { mutableStateOf<LinkDelNegocio?>(null) }
    var guardando by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf("") }

    LaunchedEffect(Unit) { link = estado.api.opcional<LinkDelNegocio>("/v1/panel/negocio/link") }
    LaunchedEffect(nuevo) {
        // Con 300 ms de pausa: no se pregunta letra por letra.
        disponible = null
        if (nuevo == actual || nuevo.length < 3 || LinkRegla.aMedias(nuevo) || !LinkRegla.valido(nuevo)) return@LaunchedEffect
        delay(300)
        disponible = estado.api.opcional<Disponibilidad>("/v1/panel/negocio/link/$nuevo")?.estado
    }

    // Ícono, color y texto del estado. Nunca sólo color.
    data class EstadoCampo(val simbolo: String, val color: Color, val texto: String, val ok: Boolean)
    val campo = when {
        nuevo == actual -> EstadoCampo("checkmark.circle.fill", C.mentaTexto, "Es tu link de ahora.", false)
        nuevo.length < 3 -> EstadoCampo("info.circle.fill", C.pagoTinta, "Tiene que tener al menos 3 letras.", false)
        LinkRegla.aMedias(nuevo) -> EstadoCampo("ellipsis", C.sub, "Seguí escribiendo…", false)
        !LinkRegla.valido(nuevo) || disponible == "invalida" -> EstadoCampo("xmark.circle.fill", C.error, "Esa dirección no se puede usar.", false)
        disponible == "libre" -> EstadoCampo("checkmark.circle.fill", C.mentaTexto, "Está libre.", true)
        disponible == "tomada" -> EstadoCampo("xmark.circle.fill", C.error, "Ya la usa otro negocio.", false)
        disponible == "propia" -> EstadoCampo("checkmark.circle.fill", C.mentaTexto, "Es tu link de ahora.", false)
        else -> EstadoCampo("ellipsis", C.sub, "Fijándonos si está libre…", false)
    }
    val sinCambios = link?.restantes == 0

    fun guardar() {
        guardando = true
        error = ""
        alcance.launch {
            try {
                link = estado.api.enviar<LinkDelNegocio, NuevoLink>("PATCH", "/v1/panel/negocio/link", NuevoLink(nuevo))
                // El negocio trae el slug: se vuelve a pedir para que todo use el nuevo.
                runCatching { estado.cargarSesion() }
                Haptico.exito()
                estado.avisar("Listo. Tu link nuevo ya anda.")
                cerrar()
            } catch (e: ErrorApi) {
                error = e.detalle
            } catch (e: Exception) {
                error = "No se pudo cambiar. Probá de nuevo."
            } finally {
                guardando = false
            }
        }
    }

    BarraHoja("Cambiar tu link", cerrar, textoCancelar = "Cerrar")
    Column(Modifier.weight(1f).verticalScroll(rememberScrollState()).padding(16.dp), verticalArrangement = Arrangement.spacedBy(14.dp)) {
        Column {
            Texto(host, Tipo.lato(15f, Peso.Bold), C.sub, lineas = 1)
            Texto(nuevo.ifEmpty { "…" }, Tipo.lato(28f, Peso.Black), C.mentaTexto, lineas = 2, tracking = (-0.6).sp)
        }
        Row(
            Modifier.fillMaxWidth().heightIn(min = 56.dp).clip(RoundedCornerShape(16.dp)).background(C.campo)
                .border(2.dp, campo.color, RoundedCornerShape(16.dp)).padding(horizontal = 14.dp),
            verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            val tinta = C.tinta
            Box(Modifier.weight(1f)) {
                if (nuevo.isEmpty()) Texto("tu-negocio", Tipo.lato(17f, Peso.Bold), C.sub.copy(alpha = 0.7f))
                BasicTextField(
                    nuevo, { nuevo = LinkRegla.limpiar(it) }, Modifier.fillMaxWidth().semantics { contentDescription = "Tu link nuevo" },
                    textStyle = Tipo.lato(17f, Peso.Bold).copy(color = tinta), singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri), cursorBrush = SolidColor(C.mentaTexto),
                )
            }
            Icono(campo.simbolo, 22f, campo.color)
        }
        Texto(campo.texto, Tipo.lato(13f, Peso.Bold), campo.color)

        Row(
            Modifier.fillMaxWidth().clip(RoundedCornerShape(18.dp)).background(C.mentaSuave).padding(14.dp),
            horizontalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Icono("checkmark.seal.fill", 20f, C.mentaTexto)
            // Un solo párrafo, la primera frase en negrita (como iOS y la web).
            TextoRico(
                buildAnnotatedString {
                    withStyle(SpanStyle(fontWeight = FontWeight.Bold)) { append("Tu link de antes sigue andando. ") }
                    append(Formato.sinViudas("Los QR y carteles impresos llevan al nuevo."))
                },
                Tipo.lato(14f), C.tinta,
            )
        }

        if (error.isNotEmpty()) Texto(error, Tipo.lato(14f, Peso.Bold), C.error)
        BotonMenta(if (guardando) "Guardando…" else "Guardar link", { guardar() }, habilitado = campo.ok && !guardando && !sinCambios)
        link?.let { l ->
            val quedan = when {
                sinCambios -> "Ya lo cambiaste 5 veces este mes."
                l.restantes == 1 -> "Te queda 1 cambio este mes."
                else -> "Te quedan ${l.restantes} cambios este mes."
            }
            Texto(quedan, Tipo.lato(13f), C.sub, Modifier.fillMaxWidth(), alinear = androidx.compose.ui.text.style.TextAlign.Center)
            if (l.anteriores.isNotEmpty()) {
                Rotulo("SIGUEN LLEVANDO A TU PÁGINA")
                Texto("Se guardan los últimos 5.", Tipo.lato(12f), C.sub)
                l.anteriores.forEach { a ->
                    Row(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalAlignment = Alignment.CenterVertically) {
                        Icono("link", 14f, C.sub); Texto("$host${a.slug}", Tipo.lato(14f), C.sub)
                    }
                }
            }
        }
    }
}

@Serializable data class LinkDelNegocio(val slug: String = "", val anteriores: List<AnteriorLink> = emptyList(), val restantes: Int = 0)
@Serializable data class AnteriorLink(val slug: String = "")
@Serializable data class NuevoLink(val slug: String)
@Serializable private data class Disponibilidad(val estado: String = "")
