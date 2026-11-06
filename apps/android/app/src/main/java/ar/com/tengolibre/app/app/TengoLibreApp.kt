package ar.com.tengolibre.app.app

import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.statusBars
import androidx.compose.foundation.layout.windowInsetsTopHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import ar.com.tengolibre.app.nucleo.Turno
import ar.com.tengolibre.app.nucleo.Resena
import ar.com.tengolibre.app.pantallas.cliente.PrincipalCliente
import ar.com.tengolibre.app.pantallas.ingreso.InvitacionVista
import ar.com.tengolibre.app.pantallas.ingreso.TurnoClienteCargar
import ar.com.tengolibre.app.tema.Hoja
import ar.com.tengolibre.app.pantallas.ingreso.*
import ar.com.tengolibre.app.pantallas.principal.Principal
import ar.com.tengolibre.app.tema.C
import ar.com.tengolibre.app.tema.Haptico
import ar.com.tengolibre.app.tema.Movimiento
import ar.com.tengolibre.app.tema.Toast
import ar.com.tengolibre.app.tema.VistaToast
import kotlinx.serialization.Serializable
import java.time.Instant

// Clon de App/TengoLibreApp.swift: Raiz (qué se ve según la sesión) y AccionesTurno.

val LocalEstado = staticCompositionLocalOf<Estado> { error("Falta el Estado") }

/** Qué se ve según la sesión: arranque, ingreso, cómo la usás, elegir negocio, permiso de avisos, alta, el panel
 *  o el modo cliente. */
@Composable
fun Raiz() {
    val estado = LocalEstado.current
    Box(Modifier.fillMaxSize()) {
        AnimatedContent(estado.fase, transitionSpec = { fadeIn(tween(250)) togetherWith fadeOut(tween(250)) }, label = "fase") { fase ->
            when (fase) {
                Estado.Fase.arrancando -> PantallaArranque()
                Estado.Fase.afuera -> Ingreso()
                Estado.Fase.eligiendoModo -> ElegirModo()
                Estado.Fase.eligiendoNegocio -> ElegirNegocio()
                Estado.Fase.permisos -> PermisoAvisos()
                Estado.Fase.alta -> Alta()
                Estado.Fase.adentro -> Principal()
                Estado.Fase.cliente -> PrincipalCliente()
            }
        }
        // La barra de estado, petróleo como la cabecera (la hora en blanco): el contenido que sube con el scroll pasa
        // por debajo, nunca por detrás de la hora.
        Box(Modifier.align(Alignment.TopCenter).fillMaxWidth().windowInsetsTopHeight(WindowInsets.statusBars).background(C.cabecera))
        AnimatedVisibility(
            estado.toast != null, Modifier.align(Alignment.TopCenter).statusBarsPadding().padding(top = 6.dp),
            enter = slideInVertically(Movimiento.entrar()) { -it } + fadeIn(), exit = slideOutVertically(Movimiento.entrar()) { -it } + fadeOut(),
        ) {
            estado.toast?.let { t -> VistaToast(t) { estado.tocarToast() } }
        }
    }
    // El turno de una clienta y las invitaciones valen en cualquier modo: se abren arriba de todo.
    Hoja(estado.turnoClienteAbierto != null, { estado.turnoClienteAbierto = null }) {
        estado.turnoClienteAbierto?.let { c -> TurnoClienteCargar(c) { estado.turnoClienteAbierto = null } }
    }
    val invitacion = if (estado.persona == null || estado.fase == Estado.Fase.arrancando) null else estado.invitacionPendiente
    Hoja(invitacion != null, { estado.invitacionPendiente = null }) {
        invitacion?.let { t -> InvitacionVista(t) { estado.invitacionPendiente = null } }
    }
    // En vivo mientras se ve el panel, con el negocio elegido.
    LaunchedEffect(estado.fase, estado.negocio?.id) {
        if (estado.fase == Estado.Fase.adentro) estado.enVivo.arrancar(estado) else estado.enVivo.parar()
    }
    LaunchedEffect(Unit) { estado.arrancar() }
}

/** Lo que se puede hacer con un turno sin abrir su pantalla (push, deslizar, menú contextual). */
object AccionesTurno {
    @Serializable data class CambioEstado(val estado: String)

    suspend fun cambiarEstado(estado: Estado, id: String, nuevo: String) {
        try {
            estado.api.hacer("PATCH", "/v1/panel/turnos/$id", CambioEstado(nuevo))
            estado.version += 1
        } catch (e: Exception) {
            estado.avisar(e)
        }
    }

    /** «Avisar a la lista»: ofrece el hueco de un turno cancelado a quienes esperan. */
    suspend fun avisarALista(estado: Estado, turnoId: String) {
        try {
            val t: Turno = estado.api.get("/v1/panel/turnos/$turnoId")
            val s = t.servicioId ?: return
            val p = t.profesionalId ?: return
            ofrecer(estado, s, p, t.inicio)
        } catch (e: Exception) {
            estado.avisar(e)
        }
    }

    @Serializable data class Oferta(val servicioId: String, val profesionalId: String, val inicio: String)

    suspend fun ofrecer(estado: Estado, servicioId: String, profesionalId: String, inicio: Instant) {
        estado.api.hacer("POST", "/v1/panel/espera/ofertas", Oferta(servicioId, profesionalId, ar.com.tengolibre.app.nucleo.Fecha.texto(inicio)))
        Haptico.exito()
        estado.version += 1
        estado.avisar(Toast("megaphone", "Le avisamos a la lista", "Se lo ofrecemos de a una persona por vez."))
    }

    @Serializable data class Respuesta(val respuesta: String)
    @Serializable data class Resenas(val resenas: List<Resena> = emptyList())

    suspend fun responderResena(estado: Estado, turnoId: String, texto: String) {
        val t = texto.trim()
        if (t.isEmpty()) return
        try {
            val r: Resenas = estado.api.get("/v1/panel/resenas")
            val resena = r.resenas.firstOrNull { it.turnoId == turnoId } ?: return
            estado.api.hacer("PATCH", "/v1/panel/resenas/${resena.id}", Respuesta(t))
            estado.version += 1
        } catch (e: Exception) {
            estado.avisar(e)
        }
    }
}
