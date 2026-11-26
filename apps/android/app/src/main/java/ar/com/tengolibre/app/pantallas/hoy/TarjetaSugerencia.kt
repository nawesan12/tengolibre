package ar.com.tengolibre.app.pantallas.hoy

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeOut
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
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
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.semantics.clearAndSetSemantics
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.pantallas.principal.LocalRecorridoVisto
import ar.com.tengolibre.app.tema.*
import coil3.compose.AsyncImage
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable

// Clon de Pantallas/Hoy/TarjetaSugerencia.swift.

/** Una sugerencia de la casa (GET /v1/panel/sugerencia): otra app de Grupo Vimo, con los textos del catálogo de la
 *  API. Todo opcional menos lo que se muestra: si falta algo o llega null, no se dibuja. */
@Serializable
data class Sugerencia(
    val id: String,
    val marca: String,
    val sello: String,
    val titulo: String,
    val cta: String,
    val url: String,
    val imagen: String,
    val banner: String? = null,
    val color: String? = null,
)

@Serializable private data class RespuestaSugerencia(val sugerencia: Sugerencia? = null)
@Serializable private data class CierreSugerencia(val hasta: String)

/** Al final de Hoy, sólo para la dueña o la encargada: la foto de la otra app, el sello «De Grupo Vimo», el título y
 *  el link (se abre en una Custom Tab). La cruz abre «Ocultar por 30 días» o «Ya la uso / No me interesa» (con «Deshacer»). Mientras
 *  carga, con error o si la imagen no baja, no se dibuja nada. La misma pieza que TarjetaSugerencia.svelte (web) y
 *  TarjetaSugerencia.swift; lámina en docs/laminas/sugerencia.html. */
@Composable
fun TarjetaSugerencia() {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    val vista = LocalView.current
    var sugerencia by remember { mutableStateOf<Sugerencia?>(null) }
    var cerrada by remember { mutableStateOf(false) }
    var fallo by remember { mutableStateOf(false) }
    var negocioCargado by remember { mutableStateOf<String?>(null) }
    // La vista se cuenta una vez por tarjeta, cuando la foto quedó dibujada (la API además cuenta una por negocio y por
    // día: pedirla con cada cambio de estado.version no infla nada).
    var vistaDe by remember { mutableStateOf<String?>(null) }

    suspend fun cargar() {
        val negocio = estado.negocio?.id
        if (negocio != negocioCargado) { cerrada = false; fallo = false; negocioCargado = negocio }
        if (!estado.gestiona || negocio == null) { sugerencia = null; return }
        try {
            val r: RespuestaSugerencia = estado.api.get("/v1/panel/sugerencia")
            sugerencia = r.sugerencia
        } catch (e: CancellationException) {
            throw e
        } catch (e: Exception) {
            sugerencia = null
        }
    }
    LaunchedEffect(estado.version, estado.negocio?.id) { cargar() }

    val s = sugerencia
    val visible = s != null && !cerrada && !fallo && LocalRecorridoVisto.current && !estado.soloLectura &&
        s.url.startsWith("https://") && s.imagen.isNotEmpty()
    AnimatedVisibility(visible, enter = androidx.compose.animation.fadeIn(), exit = fadeOut()) {
        val x = s ?: return@AnimatedVisibility
        val forma = RoundedCornerShape(18.dp)
        Box(Modifier.fillMaxWidth().clip(forma).background(C.tarjeta).border(1.dp, C.linea, forma)) {
            Row(
                Modifier.fillMaxWidth()
                    .clickable(onClickLabel = "abrir en el navegador") {
                        Haptico.suave()
                        estado.abrirPestana(x.url)
                        alcance.launch { try { estado.api.hacer("POST", "/v1/panel/sugerencia/${x.id}/clic") } catch (e: Exception) {} }
                    }
                    // Sin rol: TalkBack no tiene «enlace»; lee el texto y «doble toque para abrir en el navegador», como
                    // VoiceOver con «enlace» y la pista «Se abre en el navegador».
                    .clearAndSetSemantics { contentDescription = "${x.sello}. ${x.titulo}. ${x.cta}" }
                    .padding(12.dp),
                verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Box(Modifier.size(72.dp).clip(RoundedCornerShape(14.dp)).background(x.color?.let { hexTexto(it) } ?: C.chip)) {
                    AsyncImage(x.imagen, null, Modifier.size(72.dp), contentScale = ContentScale.Crop, alignment = Alignment.BottomCenter,
                        onError = { fallo = true },
                        onSuccess = {
                            if (vistaDe != x.id) {
                                vistaDe = x.id
                                alcance.launch { try { estado.api.hacer("POST", "/v1/panel/sugerencia/${x.id}/vista") } catch (e: Exception) {} }
                            }
                        })
                }
                Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(3.dp)) {
                    Texto(x.sello.uppercase(), Tipo.lato(11f, Peso.Black), C.sub, Modifier.padding(end = 36.dp), tracking = (11 * 0.08).sp)
                    // Sin tope de renglones (como iOS): con letra grande crecen en vez de cortarse.
                    Texto(x.titulo, Tipo.lato(15f, Peso.Bold), C.tinta)
                    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(2.dp)) {
                        Texto(x.cta, Tipo.lato(14f, Peso.Bold), C.mentaTexto)
                        Icono("arrow.up.right", 12f, C.mentaTexto)
                    }
                }
            }
            Box(Modifier.align(Alignment.TopEnd)) {
                MenuBoton(
                    listOf(
                        OpcionDeMenu("Ocultar por 30 días", "clock") { cerrar(estado, alcance, vista, x, "30d") { cerrada = !it } },
                        OpcionDeMenu("Ya la uso / No me interesa", "minus.circle") { cerrar(estado, alcance, vista, x, "siempre") { cerrada = !it } },
                    ),
                    "Ocultar sugerencia",
                ) {
                    Box(Modifier.size(48.dp), contentAlignment = Alignment.Center) { Icono("xmark", 14f, C.sub) }
                }
            }
        }
    }
}

/** Se va en el acto y sale el aviso con «Deshacer» (que la trae de vuelta), anunciado a TalkBack. */
private fun cerrar(
    estado: ar.com.tengolibre.app.app.Estado, alcance: kotlinx.coroutines.CoroutineScope, vista: android.view.View, s: Sugerencia, hasta: String,
    mostrar: (Boolean) -> Unit,
) {
    Haptico.suave()
    mostrar(false)
    val titulo = if (hasta == "30d") "La ocultamos por 30 días" else "Listo, no la vas a ver más"
    alcance.launch {
        // Un reintento al segundo; si tampoco se guardó, vuelve y se avisa: el retiro no se pierde en silencio.
        val ruta = "/v1/panel/sugerencia/${s.id}/cerrar"
        val guardado = try {
            try { estado.api.hacer("POST", ruta, CierreSugerencia(hasta)) } catch (e: CancellationException) { throw e } catch (e: Exception) {
                delay(1000)
                estado.api.hacer("POST", ruta, CierreSugerencia(hasta))
            }
            true
        } catch (e: CancellationException) { throw e } catch (e: Exception) { false }
        if (!guardado) {
            mostrar(true)
            estado.avisar(Toast("exclamationmark", "No pudimos ocultarla", "Probá de nuevo en un rato.", error = true))
            return@launch
        }
        estado.avisar(Toast("checkmark", titulo, accion = "Deshacer")) {
            try {
                estado.api.hacer("DELETE", "/v1/panel/sugerencia/${s.id}/cerrar")
                mostrar(true)
            } catch (e: CancellationException) { throw e } catch (e: Exception) { estado.avisar(e) }
        }
        @Suppress("DEPRECATION") vista.announceForAccessibility(titulo)
    }
}
