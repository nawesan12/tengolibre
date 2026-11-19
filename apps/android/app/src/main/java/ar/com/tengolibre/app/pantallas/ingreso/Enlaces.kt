package ar.com.tengolibre.app.pantallas.ingreso

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
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
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.nucleo.*
import ar.com.tengolibre.app.pantallas.hoy.BarraHoja
import ar.com.tengolibre.app.pantallas.mas.nombreRol
import ar.com.tengolibre.app.tema.*
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable

// Clon de Pantallas/Ingreso/Enlaces.swift.

/** Una invitación al equipo (tengolibre.com.ar/invitacion/<token>): quién invita, a qué negocio y con qué rol, y
 *  «Sumarme». Al aceptar, entra a ese negocio. Va adentro de una Hoja. */
@Composable
fun ColumnScope.InvitacionVista(token: String, cerrar: () -> Unit) {
    @Serializable data class Invitacion(val negocio: String = "", val slug: String = "", val email: String = "", val rol: String = "", val vencida: Boolean = false, val aceptada: Boolean = false, val invitadoPor: String? = null)
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    var inv by remember { mutableStateOf<Invitacion?>(null) }
    var error by remember { mutableStateOf("") }
    var aceptando by remember { mutableStateOf(false) }
    suspend fun cargar() { try { inv = estado.api.get("/v1/invitaciones/$token"); error = "" } catch (e: ErrorApi) { error = e.detalle } catch (e: Exception) { error = "Probá de nuevo." } }
    LaunchedEffect(token) { cargar() }
    fun aceptar(i: Invitacion) = alcance.launch {
        aceptando = true
        try {
            estado.api.pedir("POST", "/v1/invitaciones/$token/aceptar", cuerpo = "{}".toByteArray())
            estado.cargarSesion()
            Haptico.exito()
            cerrar()
            estado.negocios.firstOrNull { it.slug == i.slug }?.let { n ->
                estado.entrarANegocio(n)
            }
            estado.avisar("Ya sos parte de ${i.negocio}")
        } catch (e: Exception) { estado.avisar(e) }
        aceptando = false
    }

    BarraHoja("", cerrar, textoCancelar = "Ahora no")
    Column(Modifier.weight(1f).fillMaxSize().background(C.fondo).padding(20.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        val i = inv
        if (i != null) {
            Box(Modifier.size(64.dp).clip(RoundedCornerShape(18.dp)).background(C.menta), contentAlignment = Alignment.Center) { Icono("person.badge.plus", 30f, C.petroleoHondo) }
            Texto("Te invitaron a ${i.negocio}", Tipo.lato(28f, Peso.Black), C.tinta, tracking = (-0.6).sp)
            Texto("${i.invitadoPor?.let { "$it te sumó" } ?: "Te sumaron"} como ${nombreRol(i.rol).lowercase()}. Vas a ver la agenda y los avisos de ese negocio en esta app.", Tipo.lato(16f), C.sub)
            val mio = estado.persona?.email
            if (i.vencida || i.aceptada) Row(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalAlignment = Alignment.CenterVertically) {
                Icono("exclamationmark.circle", 15f, C.ausenteTinta)
                Texto(if (i.aceptada) "Esta invitación ya se usó." else "Esta invitación venció. Pedile otra a quien te invitó.", Tipo.lato(15f, Peso.Bold), C.ausenteTinta)
            } else if (i.email.isNotEmpty() && mio != null && !i.email.equals(mio, ignoreCase = true))
                Texto("La invitación es para ${i.email} y entraste como $mio. Igual podés aceptarla.", Tipo.lato(14f), C.sub)
            Spacer(Modifier.weight(1f))
            if (!i.vencida && !i.aceptada) BotonMenta(if (aceptando) "Sumándote…" else "Sumarme a ${i.negocio}", { aceptar(i) }, habilitado = !aceptando)
        } else if (error.isNotEmpty()) {
            ErrorConSalida("No encontramos la invitación", error) { alcance.launch { cargar() } }
            Spacer(Modifier.weight(1f))
        } else Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { Pulso() }
    }
}
