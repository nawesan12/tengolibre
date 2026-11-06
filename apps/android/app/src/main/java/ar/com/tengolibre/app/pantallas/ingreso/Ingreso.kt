package ar.com.tengolibre.app.pantallas.ingreso

import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
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
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import ar.com.tengolibre.app.R
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.nucleo.Clerk
import ar.com.tengolibre.app.nucleo.ErrorApi
import ar.com.tengolibre.app.nucleo.NegocioDeLaPersona
import ar.com.tengolibre.app.pantallas.mas.Bloque
import ar.com.tengolibre.app.pantallas.mas.RenglonCampo
import ar.com.tengolibre.app.tema.*
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable
import java.text.Normalizer

// Clon de Pantallas/Ingreso/Ingreso.swift.

/** 00 Arranque: el símbolo sobre petróleo mientras se lee la sesión. */
@Composable
fun PantallaArranque() {
    Box(Modifier.fillMaxSize().background(C.petroleo).semantics { contentDescription = "Tengo Libre" }, contentAlignment = Alignment.Center) {
        Image(painterResource(R.drawable.simbolo_blanco), null, Modifier.size(72.dp))
    }
}

/** 01 Ingreso: petróleo con el logotipo, «Tu agenda, en el bolsillo.» y la hoja con Apple y Google. */
@Composable
fun Ingreso() {
    val estado = LocalEstado.current
    val contexto = LocalContext.current
    val alcance = rememberCoroutineScope()
    var entrando by remember { mutableStateOf<String?>(null) }
    var error by remember { mutableStateOf("") }
    // La Custom Tab vuelve sola: cuando termina el ingreso (o se cancela), se suelta el botón.
    LaunchedEffect(estado.ingresando) { if (!estado.ingresando) entrando = null }

    fun entrar(con: String) {
        entrando = con
        error = ""
        alcance.launch {
            try {
                if (con == "apple") estado.ingresarConApple(contexto) else estado.ingresarConGoogle(contexto)
            } catch (e: Clerk.ErrorClerk) {
                entrando = null
                if (e.codigo != "cancelado") { error = "No pudimos entrar. Probá de nuevo."; Haptico.error() }
            } catch (e: ErrorApi) {
                entrando = null
                error = e.detalle; Haptico.error()
            } catch (e: Exception) {
                entrando = null
                error = "No pudimos entrar. Probá de nuevo."; Haptico.error()
            }
        }
    }

    Box(Modifier.fillMaxSize().fondoPetroleo(C.petroleo)) {
        Column(Modifier.fillMaxSize().statusBarsPadding().padding(horizontal = 24.dp), verticalArrangement = Arrangement.spacedBy(18.dp)) {
            Image(painterResource(R.drawable.logotipo_negativo), null, Modifier.padding(top = 24.dp).height(30.dp), contentScale = ContentScale.Fit, alignment = Alignment.CenterStart)
            Spacer(Modifier.weight(1f))
            Texto("Tu agenda,\nen el bolsillo.", Tipo.lato(44f, Peso.Black).copy(lineHeight = 46.sp), Color.White, tracking = (-1.5).sp)
            Texto("Mirá el día, cargá turnos y enterate de cada reserva al toque.", Tipo.lato(17f), Color.White.copy(alpha = 0.8f))
            Spacer(Modifier.height(300.dp))
        }
        Column(
            Modifier.align(Alignment.BottomCenter).fillMaxWidth().clip(RoundedCornerShape(topStart = 28.dp, topEnd = 28.dp)).background(C.tarjeta)
                .padding(start = 20.dp, end = 20.dp, top = 24.dp, bottom = 12.dp).navigationBarsPadding(),
            verticalArrangement = Arrangement.spacedBy(12.dp), horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            val forma = RoundedCornerShape(16.dp)
            Row(
                Modifier.fillMaxWidth().heightIn(min = 56.dp).clip(forma).background(Color.Black).clickable(enabled = entrando == null) { entrar("apple") },
                horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically,
            ) {
                if (entrando == "apple") Texto("…", Tipo.lato(17f, Peso.Bold), Color.White) else Icono("apple.logo", 20f, Color.White)
                Spacer(Modifier.width(10.dp))
                Texto("Continuar con Apple", Tipo.lato(17f, Peso.Bold), Color.White)
            }
            Row(
                Modifier.fillMaxWidth().heightIn(min = 56.dp).clip(forma).background(Color.White).border(1.5.dp, hex(0xE2D6BF), forma)
                    .clickable(enabled = entrando == null) { entrar("google") },
                horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically,
            ) {
                if (entrando == "google") Texto("…", Tipo.lato(17f, Peso.Bold), hex(0x0F2328)) else Image(painterResource(R.drawable.logo_google), null, Modifier.size(20.dp))
                Spacer(Modifier.width(10.dp))
                Texto("Continuar con Google", Tipo.lato(17f, Peso.Bold), hex(0x0F2328))
            }
            if (error.isNotEmpty()) Texto(error, Tipo.lato(14f, Peso.Bold), C.error, alinear = TextAlign.Center)
            Texto("Sin contraseñas. Usá la misma cuenta que en la web.", Tipo.lato(13f), C.sub, alinear = TextAlign.Center)
        }
    }
}

/** 02 Modo: «¿Cómo vas a usar Tengo Libre?» — atiendo un negocio / reservo turnos. */
@Composable
fun ElegirModo() {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    Column(Modifier.fillMaxSize().background(C.fondo).statusBarsPadding().padding(horizontal = 20.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Texto("¿Cómo vas a usar Tengo Libre?", Tipo.lato(30f, Peso.Black), C.tinta, Modifier.padding(top = 40.dp), tracking = (-0.8).sp)
        Texto("Lo podés cambiar cuando quieras desde tu cuenta.", Tipo.lato(16f), C.sub)
        Column(
            Modifier.fillMaxWidth().clip(RoundedCornerShape(24.dp)).background(C.bloquePetroleo).clickable { alcance.launch { estado.elegirModo("negocio") } }.padding(20.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Icono("storefront", 28f, C.menta)
            Texto("Atiendo un negocio", Tipo.lato(22f, Peso.Black), Color.White)
            Texto("Tu agenda, tu equipo y tus clientas. Cargás turnos y te enterás de cada reserva.", Tipo.lato(15f), Color.White.copy(alpha = 0.8f))
        }
        Column(
            Modifier.fillMaxWidth().tarjeta(24.dp).clickable { alcance.launch { estado.elegirModo("cliente") } }.padding(20.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Icono("calendar.badge.checkmark", 28f, C.mentaTexto)
            Texto("Reservo turnos", Tipo.lato(22f, Peso.Black), C.tinta)
            Texto("Tus turnos en un solo lugar, recordatorios y lugares cerca tuyo con horario libre.", Tipo.lato(15f), C.sub)
        }
    }
}

/** 03 Elegir negocio: «¿Dónde trabajás hoy?». */
@Composable
fun ElegirNegocio() {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    var elegido by remember { mutableStateOf<NegocioDeLaPersona?>(null) }
    Column(Modifier.fillMaxSize().background(C.fondo).statusBarsPadding().navigationBarsPadding().padding(horizontal = 20.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Texto("¿Dónde trabajás hoy?", Tipo.lato(30f, Peso.Black), C.tinta, Modifier.padding(top = 40.dp), tracking = (-0.8).sp)
        Texto("Tenés más de un lugar. Después cambiás desde arriba de Hoy.", Tipo.lato(16f), C.sub, Modifier.padding(bottom = 8.dp))
        Column(Modifier.weight(1f).verticalScroll(rememberScrollState()), verticalArrangement = Arrangement.spacedBy(10.dp)) {
            estado.negocios.forEach { n ->
                val on = elegido?.id == n.id
                val forma = RoundedCornerShape(20.dp)
                Row(
                    Modifier.fillMaxWidth().heightIn(min = 72.dp).tarjeta(20.dp).then(if (on) Modifier.border(2.dp, C.menta, forma) else Modifier)
                        .clickable { elegido = n; Haptico.seleccion() }.padding(horizontal = 16.dp),
                    verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    Iniciales(n.nombre, 44.dp, if (on) C.menta else C.chip, if (on) C.petroleoHondo else C.tinta)
                    Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                        Texto(n.nombre, Tipo.lato(17f, Peso.Bold), C.tinta)
                        Texto(n.rolTexto, Tipo.lato(13f), C.sub)
                    }
                    Icono(if (on) "checkmark.circle.fill" else "circle", 22f, if (on) C.mentaTexto else C.linea)
                }
            }
        }
        BotonMenta(elegido?.let { "Entrar a ${it.nombre}" } ?: "Elegí un lugar", { elegido?.let { n -> alcance.launch { estado.entrarANegocio(n) } } },
            Modifier.padding(bottom = 8.dp), habilitado = elegido != null)
    }
}

/** 04 Permiso de avisos: dos avisos de ejemplo, «Enterate al toque de cada turno.» y los dos botones. */
@Composable
fun PermisoAvisos() {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    Column(Modifier.fillMaxSize().background(C.fondo)) {
        Box(Modifier.weight(1f).fillMaxWidth().fondoPetroleo(C.petroleo).padding(horizontal = 20.dp), contentAlignment = Alignment.Center) {
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                Aviso("Nueva reserva", "Sofía · Corte y brushing · mañana 10:30", "ahora")
                Box(Modifier.alpha(0.85f).scale(0.96f)) { Aviso("Se liberó un horario", "Hoy 16:00 con Carla · 3 en la lista de espera", "hace 2 min") }
            }
        }
        Column(Modifier.background(C.fondo).padding(20.dp).navigationBarsPadding(), verticalArrangement = Arrangement.spacedBy(14.dp)) {
            Texto("Enterate al toque de cada turno.", Tipo.lato(30f, Peso.Black), C.tinta, tracking = (-0.8).sp)
            Texto("Reservas, cancelaciones y huecos que se liberan. Desde el aviso confirmás o le avisás a la lista sin abrir la app.", Tipo.lato(16f), C.sub)
            BotonMenta("Activar avisos", { alcance.launch { estado.pedirPermisoPush(); estado.terminarPermisos() } })
            Box(Modifier.fillMaxWidth().height(44.dp), contentAlignment = Alignment.Center) {
                BotonTexto("Ahora no", { alcance.launch { estado.terminarPermisos() } }, color = C.tinta)
            }
        }
    }
}

@Composable
private fun Aviso(t: String, x: String, h: String) {
    val tinta = hex(0x0F2328)
    Row(Modifier.fillMaxWidth().clip(RoundedCornerShape(20.dp)).background(Color.White.copy(alpha = 0.92f)).padding(12.dp), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
        Image(painterResource(R.drawable.simbolo), null, Modifier.size(36.dp).clip(RoundedCornerShape(9.dp)))
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Row {
                Texto("TENGO LIBRE", Tipo.lato(11f, Peso.Bold), tinta.copy(alpha = 0.55f), Modifier.weight(1f))
                Texto(h, Tipo.lato(11f), tinta.copy(alpha = 0.55f))
            }
            Texto(t, Tipo.lato(15f, Peso.Bold), tinta)
            Texto(x, Tipo.lato(14f), tinta)
        }
    }
}

/** Alta: quien entra sin negocio abre el suyo (nombre y link), o pasa a reservar. */
@Composable
fun Alta() {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    var nombre by remember { mutableStateOf("") }
    var slug by remember { mutableStateOf("") }
    var slugTocado by remember { mutableStateOf(false) }
    var creando by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf("") }

    @Serializable data class Creado(val id: String)
    @Serializable data class Cuerpo(val nombre: String, val slug: String)

    fun crear() {
        creando = true
        error = ""
        alcance.launch {
            try {
                val r: Creado = estado.api.enviar("POST", "/v1/negocios", Cuerpo(nombre.trim(), slug))
                estado.cargarSesion()
                estado.negocios.firstOrNull { it.id == r.id }?.let { estado.entrarANegocio(it) }
                Haptico.exito()
            } catch (e: ErrorApi) {
                error = e.detalle
            } catch (e: Exception) {
                error = "No pudimos crearlo. Probá de nuevo."
            } finally {
                creando = false
            }
        }
    }

    Column(
        Modifier.fillMaxSize().background(C.fondo).verticalScroll(rememberScrollState()).statusBarsPadding().navigationBarsPadding().padding(horizontal = 20.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Texto("Abrí tu negocio en Tengo Libre", Tipo.lato(30f, Peso.Black), C.tinta, Modifier.padding(top = 40.dp), tracking = (-0.8).sp)
        Texto("Gratis los primeros 30 días. Después lo terminás de armar desde Más: servicios, equipo y horarios.", Tipo.lato(16f), C.sub)
        Bloque(
            "", "",
            { RenglonCampo("Nombre del negocio", nombre, { nombre = it; if (!slugTocado) slug = aSlug(it) }, "Lumi Estética") },
            { RenglonCampo("Tu link", slug, { slug = aSlug(it); slugTocado = true }, "lumi", KeyboardType.Uri) },
        )
        Texto("tengolibre.com.ar/${slug.ifEmpty { "tu-negocio" }}", Tipo.lato(15f, Peso.Bold), C.mentaTexto)
        if (error.isNotEmpty()) Texto(error, Tipo.lato(14f, Peso.Bold), C.error)
        BotonMenta(if (creando) "Creando…" else "Crear mi negocio", { crear() }, habilitado = !creando && nombre.length >= 2 && slug.length >= 3)
        Box(Modifier.fillMaxWidth(), contentAlignment = Alignment.Center) { BotonTexto("En realidad, quiero reservar turnos", { alcance.launch { estado.elegirModo("cliente") } }, color = C.tinta) }
        Box(Modifier.fillMaxWidth(), contentAlignment = Alignment.Center) { BotonTexto("Salir", { alcance.launch { estado.salir() } }, color = C.sub) }
    }
}

/** «Lumi Estética» → «lumi-estetica» (como aSlug en iOS). */
fun aSlug(s: String): String {
    val base = Normalizer.normalize(s, Normalizer.Form.NFD).replace(Regex("\\p{M}"), "").lowercase()
    val out = StringBuilder()
    var guion = false
    for (c in base) {
        if ((c.isLetter() && c.code < 128) || c.isDigit()) { out.append(c); guion = false }
        else if (!guion && out.isNotEmpty()) { out.append('-'); guion = true }
    }
    return out.toString().trim('-').take(40)
}
