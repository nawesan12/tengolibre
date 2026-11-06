package ar.com.tengolibre.app.pantallas.hoy

import androidx.compose.foundation.background
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
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.unit.dp
import ar.com.tengolibre.app.app.Estado
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.nucleo.*
import ar.com.tengolibre.app.pantallas.clientes.DestinoClientes
import ar.com.tengolibre.app.pantallas.mas.DestinoMas
import ar.com.tengolibre.app.pantallas.mas.NuevaSocia
import ar.com.tengolibre.app.pantallas.mas.PlanMembresia
import ar.com.tengolibre.app.pantallas.principal.LocalNavegacion
import ar.com.tengolibre.app.pantallas.principal.Navegacion
import ar.com.tengolibre.app.pantallas.principal.Pestana
import ar.com.tengolibre.app.tema.*
import kotlinx.coroutines.MainScope
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable
import java.time.Duration
import java.time.Instant

// Clon de Pantallas/Hoy/Momentos.swift.

/** Sugerencias en el momento justo: después de algo que pasó (un ausente, el primer cobro, cargar turnos a mano…)
 *  la herramienta que lo resuelve, en un aviso con su botón. Cada una sale una sola vez por negocio, nunca dos
 *  seguidas, y espera a que se vaya el aviso que esté a la vista (el «Deshacer»). */
object Momentos {
    private var ultimo: Instant = Instant.EPOCH
    private val alcance = MainScope()

    private fun clave(estado: Estado, k: String) = "momento-$k-${estado.negocio?.id.orEmpty()}"
    private fun visto(estado: Estado, k: String) = Preferencias.bool(clave(estado, k))
    private fun pasoUnMinuto() = Duration.between(ultimo, Instant.now()).seconds > 60

    /** Muestra el aviso cuando no hay otro a la vista (hasta 12 s), si no se mostró antes. */
    private fun mostrar(estado: Estado, k: String, simbolo: String, titulo: String, texto: String, accion: String, repetible: Boolean = false, hacer: () -> Unit) {
        if (!(repetible || !visto(estado, k)) || !pasoUnMinuto()) return
        alcance.launch {
            var n = 0
            while (estado.toast != null && n < 24) { delay(500); n++ }
            if (estado.toast != null || !pasoUnMinuto()) return@launch
            ultimo = Instant.now()
            if (!repetible) Preferencias.ponerBool(clave(estado, k), true)
            estado.avisar(Toast(simbolo, titulo, texto, accion)) { hacer() }
        }
    }

    /** Se marcó un turno (deslizar, detalle o aviso). */
    fun cambioDeEstado(estado: Estado, nav: Navegacion, t: Turno, nuevo: String) {
        if (nuevo != "ausente") return
        val mp = estado.vivo.mp
        if (estado.gestiona && mp != null && mp.disponible && (!mp.conectada || (estado.vivo.cobro?.modo ?: "sin_cobro") == "sin_cobro")) {
            mostrar(estado, "ausente-sena", "creditcard", "Que no falten: pedí seña", "Con una seña al reservar, casi nadie falta.", "Ver") {
                nav.pestana = Pestana.hoy; nav.pilaHoy.add(DestinoMas.cobros)
            }
            return
        }
        val id = t.cliente.id ?: return
        if (!(estado.gestiona || estado.rol == "recepcion")) return
        alcance.launch {
            val f: Ficha = estado.api.opcional("/v1/panel/clientes/$id") ?: return@launch
            if (f.ausencias < 2 || f.penalizado) return@launch
            val nombre = f.nombre.split(" ").firstOrNull() ?: f.nombre
            mostrar(estado, "penalizar-$id", "exclamationmark.shield", "$nombre ya faltó ${f.ausencias} veces", "Podés penalizarla para que no reserve por la web un tiempo.", "Ver") {
                nav.pestana = Pestana.clientes; nav.pilaClientes.add(DestinoClientes.ficha(id))
            }
        }
    }

    /** El primer cobro desde la app. */
    fun cobro(estado: Estado, nav: Navegacion) {
        mostrar(estado, "primer-cobro", "banknote", "Tu caja, al día", "Mirá cuánto entró hoy y por qué medio.", "Ver la caja") {
            nav.pestana = Pestana.hoy; nav.pilaHoy.add(DestinoMas.caja)
        }
    }

    /** Cada turno cargado a mano: al tercero, que reserven solos. */
    fun cargaManual(estado: Estado, nav: Navegacion) {
        val k = clave(estado, "manuales")
        val n = Preferencias.entero(k) + 1
        Preferencias.ponerEntero(k, n)
        if (n < 3) return
        mostrar(estado, "reserven-solos", "qrcode", "Que te reserven solos", "Pasales tu link y se cargan los turnos sin vos.", "Compartir") { nav.compartir = true }
    }

    /** Una reseña de 5 estrellas respondida: compartirla. */
    fun resenaBuena(estado: Estado, r: Resena) {
        if (r.puntaje < 5 || r.comentario.isEmpty()) return
        mostrar(estado, "resena-${r.id}", "star.fill", "¡Qué linda reseña!", "Compartila en tus historias.", "Compartir", repetible = true) {
            estado.compartirTexto("«${r.comentario}» — ${r.nombre} ⭐️⭐️⭐️⭐️⭐️\nReservá en ${Web.textoPagina(estado.negocio?.slug.orEmpty())}")
        }
    }
}

/** En la ficha: una clienta fiel sin membresía → ofrecérsela (o crear el primer plan). */
@Composable
fun SugerenciaMembresia(ficha: Ficha, conMembresia: Boolean) {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    var vender by remember { mutableStateOf(false) }
    var planes by remember { mutableStateOf<List<PlanMembresia>>(emptyList()) }
    if (ficha.visitas < 5 || conMembresia || !(estado.gestiona || estado.rol == "recepcion")) return
    @Serializable data class P(val planes: List<PlanMembresia> = emptyList())
    LaunchedEffect(Unit) { planes = (estado.api.opcional<P>("/v1/panel/membresias/planes")?.planes ?: emptyList()).filter { it.activo } }
    Row(
        Modifier.fillMaxWidth().tarjeta(18.dp).clickable { if (estado.vivo.planes == 0) nav.pilaClientes.add(DestinoMas.membresias) else vender = true }.padding(14.dp),
        verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Box(Modifier.size(40.dp).clip(RoundedCornerShape(12.dp)).background(C.menta), contentAlignment = Alignment.Center) { Icono("ticket", 20f, C.petroleoHondo) }
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Texto("Viene seguido: ofrecele una membresía", Tipo.lato(15f, Peso.Bold), C.tinta)
            Texto(if (estado.vivo.planes == 0) "Creá tu primer plan con visitas incluidas" else "${ficha.visitas} visitas · plata fija todos los meses", Tipo.lato(13f), C.sub)
        }
        Icono("chevron.right", 13f, C.sub)
    }
    Hoja(vender, { vender = false }) {
        NuevaSocia(planes, ficha.nombre, ficha.telefono, ficha.email, listo = { estado.version += 1 }, cerrar = { vender = false })
    }
}
