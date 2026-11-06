package ar.com.tengolibre.app.pantallas.principal

import androidx.activity.compose.BackHandler
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.core.animateDpAsState
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshots.SnapshotStateList
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.nucleo.ClienteDelTurno
import ar.com.tengolibre.app.nucleo.NegocioDeLaPersona
import ar.com.tengolibre.app.nucleo.Preferencias
import ar.com.tengolibre.app.nucleo.Turno
import ar.com.tengolibre.app.app.Ruta
import ar.com.tengolibre.app.tema.*
import kotlinx.coroutines.launch
import java.time.Instant
import java.util.UUID

// Clon de Pantallas/Principal/Principal.swift.

/** Las pestañas según el rol (handoff › Navegación). */
enum class Pestana {
    hoy, agenda, espera, clientes, resenas, mas;

    fun titulo(profesional: Boolean): String = when (this) {
        hoy -> if (profesional) "Tu día" else "Hoy"
        agenda -> "Agenda"
        espera -> "Espera"
        clientes -> "Clientes"
        resenas -> "Reseñas"
        mas -> "Más"
    }

    val simbolo: String get() = when (this) {
        hoy -> "sun.max"
        agenda -> "calendar"
        espera -> "hourglass"
        clientes -> "person.2"
        resenas -> "star.bubble"
        mas -> "line.3.horizontal"
    }
}

/** Con qué arranca «Nuevo turno»: un hueco tocado, una clienta, una oferta que se carga a mano. */
data class NuevoTurnoInicio(
    val id: String = UUID.randomUUID().toString(),
    val profesionalId: String? = null,
    val servicioId: String? = null,
    val inicio: Instant? = null,
    val cliente: ClienteDelTurno? = null,
    val esperaId: String? = null,
)

/** Lo que se abre encima (hojas) y desde dónde. Una pila por pestaña (NavigationPath en iOS). */
class Navegacion {
    var pestana by mutableStateOf(Pestana.hoy)
    var turno by mutableStateOf<Turno?>(null)
    var nuevo by mutableStateOf<NuevoTurnoInicio?>(null)
    var compartir by mutableStateOf(false)
    var avisos by mutableStateOf(false)
    var buscar by mutableStateOf(false)
    val pilas: Map<Pestana, SnapshotStateList<Any>> = Pestana.entries.associateWith { mutableStateListOf() }
    val pilaHoy get() = pilas.getValue(Pestana.hoy)
    val pilaAgenda get() = pilas.getValue(Pestana.agenda)
    val pilaEspera get() = pilas.getValue(Pestana.espera)
    val pilaClientes get() = pilas.getValue(Pestana.clientes)
    val pilaResenas get() = pilas.getValue(Pestana.resenas)
    val pilaMas get() = pilas.getValue(Pestana.mas)
    /** La pila de la pestaña que se ve. */
    val pila get() = pilas.getValue(pestana)
}

val LocalNavegacion = staticCompositionLocalOf<Navegacion> { error("Falta la Navegacion") }

/** Las pantallas de cada pestaña y los destinos de las pilas. Las registra el resto de las pantallas a medida que
 *  se portan (en iOS: .destinos() en Mas.swift). */
object Destinos {
    /** La raíz de cada pestaña. */
    var raiz: @Composable (Pestana) -> Unit = {}
    /** Un destino apilado (DestinoMas, DestinoClientes.ficha…). */
    var destino: @Composable (Any) -> Unit = {}
    /** Las hojas globales. */
    var detalleTurno: @Composable (Turno) -> Unit = {}
    var nuevoTurno: @Composable (NuevoTurnoInicio) -> Unit = {}
    var compartir: @Composable () -> Unit = {}
    var buscar: @Composable () -> Unit = {}
    var avisos: @Composable () -> Unit = {}
    var recorrido: @Composable (alTerminar: () -> Unit) -> Unit = { }
    /** Abrir la suscripción, la caja o las reseñas desde una ruta (DestinoMas.suscripcion / .caja / .resenas). */
    var destinoSuscripcion: Any = Unit
    var destinoCaja: Any = Unit
    var destinoResenas: Any = Unit
}

@Composable
fun Principal() {
    val estado = LocalEstado.current
    remember { ar.com.tengolibre.app.app.Registro.instalar() }
    val nav = remember { Navegacion() }
    var recorridoVisto by remember { mutableStateOf(Preferencias.bool("recorridoVisto")) }
    val pestanas = if (estado.esProfesional) listOf(Pestana.hoy, Pestana.agenda, Pestana.resenas, Pestana.mas)
        else listOf(Pestana.hoy, Pestana.agenda, Pestana.espera, Pestana.clientes, Pestana.mas)

    androidx.compose.runtime.CompositionLocalProvider(LocalNavegacion provides nav) {
        Box(Modifier.fillMaxSize().background(C.fondo)) {
            Column(Modifier.fillMaxSize()) {
                Box(Modifier.weight(1f)) {
                    val pila = nav.pila
                    AnimatedContent(nav.pestana to pila.lastOrNull(), transitionSpec = {
                        if (initialState.first != targetState.first) fadeIn() togetherWith fadeOut()
                        else if (pila.size > 0 && targetState.second != null) slideInHorizontally { it } togetherWith slideOutHorizontally { -it / 3 }
                        else slideInHorizontally { -it / 3 } togetherWith slideOutHorizontally { it }
                    }, label = "pantalla") { (p, arriba) ->
                        if (arriba == null) Destinos.raiz(p) else Destinos.destino(arriba)
                    }
                    BackHandler(nav.pila.isNotEmpty()) { nav.pila.removeAt(nav.pila.lastIndex) }
                }
                BarraPestanas(pestanas, nav.pestana, { nav.pestana = it }, estado.esProfesional)
            }
            if (nav.pestana in listOf(Pestana.hoy, Pestana.clientes) && nav.pila.isEmpty()) {
                Box(Modifier.align(Alignment.BottomEnd).navigationBarsPadding().padding(end = 16.dp, bottom = 49.dp + 16.dp)) { BotonTurnoFlotante() }
            }
            Hoja(nav.turno != null, { nav.turno = null }, listOf(Altura.media, Altura.grande)) { nav.turno?.let { Destinos.detalleTurno(it) } }
            Hoja(nav.nuevo != null, { nav.nuevo = null }) { nav.nuevo?.let { Destinos.nuevoTurno(it) } }
            Hoja(nav.compartir, { nav.compartir = false }) { Destinos.compartir() }
            Hoja(nav.buscar, { nav.buscar = false }) { Destinos.buscar() }
            Hoja(nav.avisos, { nav.avisos = false }) { Destinos.avisos() }
            // La primera vez: el recorrido por todo lo que trae la app.
            if (!recorridoVisto) Destinos.recorrido { recorridoVisto = true; Preferencias.ponerBool("recorridoVisto", true) }
        }
    }
    LaunchedEffect(estado.rutaPendiente) { abrirRuta(estado, nav) }
}

/** Un push, un widget o un link abren su pantalla. */
private suspend fun abrirRuta(estado: ar.com.tengolibre.app.app.Estado, nav: Navegacion) {
    val r = estado.rutaPendiente ?: return
    estado.rutaPendiente = null
    when (r) {
        is Ruta.turno -> {
            nav.pestana = Pestana.hoy
            runCatching { estado.api.get<Turno>("/v1/panel/turnos/${r.id}") }.getOrNull()?.let { nav.turno = it }
        }
        Ruta.hoy -> nav.pestana = Pestana.hoy
        Ruta.espera -> nav.pestana = if (estado.esProfesional) Pestana.hoy else Pestana.espera
        Ruta.avisos -> nav.avisos = true
        Ruta.resenas -> if (estado.esProfesional) nav.pestana = Pestana.resenas else { nav.pestana = Pestana.mas; nav.pilaMas.add(Destinos.destinoResenas) }
        Ruta.nuevoTurno -> nav.nuevo = NuevoTurnoInicio()
        Ruta.suscripcion -> { nav.pestana = Pestana.mas; nav.pilaMas.add(Destinos.destinoSuscripcion) }
        Ruta.caja -> { nav.pestana = Pestana.hoy; if (estado.vePrecios && !estado.esProfesional) nav.pilaHoy.add(Destinos.destinoCaja) }
        Ruta.buscar -> nav.buscar = true
        Ruta.agenda -> nav.pestana = Pestana.agenda
        is Ruta.turnoCliente, is Ruta.invitacion -> {} // los abre Raiz (valen en cualquier modo)
    }
}

/** La barra de pestañas del handoff: el ítem activo con una cápsula de 56×30 detrás del ícono (petróleo en claro,
 *  menta en oscuro) y una solapa menta de 40×5 pegada al borde de arriba. */
@Composable
fun BarraPestanas(pestanas: List<Pestana>, elegida: Pestana, alElegir: (Pestana) -> Unit, profesional: Boolean) {
    Column(Modifier.fillMaxWidth().background(C.barra).navigationBarsPadding()) {
        Box(Modifier.fillMaxWidth().height(1.dp).background(C.linea))
        Row(Modifier.fillMaxWidth()) {
            pestanas.forEach { p ->
                val on = p == elegida
                Box(
                    Modifier.weight(1f).heightIn(min = 49.dp).clickable { if (!on) alElegir(p) }
                        .semantics { contentDescription = p.titulo(profesional); selected = on },
                    contentAlignment = Alignment.TopCenter,
                ) {
                    if (on) Box(Modifier.size(40.dp, 5.dp).clip(RoundedCornerShape(bottomStart = 6.dp, bottomEnd = 6.dp)).background(C.menta))
                    Column(Modifier.padding(top = 4.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(2.dp)) {
                        Box(Modifier.size(56.dp, 30.dp).clip(CircleShape).background(if (on) C.seleccion else Color.Transparent), contentAlignment = Alignment.Center) {
                            Icono(if (on && p != Pestana.mas) p.simbolo + ".fill" else p.simbolo, 20f, if (on) C.seleccionTexto else C.sub)
                        }
                        Texto(p.titulo(profesional), Tipo.lato(11f, if (on) Peso.Black else Peso.Bold), if (on) C.tinta else C.sub, lineas = 1)
                    }
                }
            }
        }
    }
}

/** «Turno» flotando arriba de las pestañas. En sólo lectura o sin conexión, gris y con candado. */
@Composable
fun BotonTurnoFlotante() {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    val lectura = estado.soloLectura; val off = estado.sinConexion
    val forma = CircleShape
    Row(
        Modifier.sombra(forma, if (lectura || off) Color.Transparent else hex(0x17805A, 0.75f), 14.dp, y = 16.dp)
            .height(56.dp).clip(forma).background(if (lectura || off) C.chip else C.menta)
            .clickable {
                if (lectura) { nav.pestana = Pestana.mas; nav.pilaMas.add(Destinos.destinoSuscripcion); return@clickable }
                if (off) return@clickable
                Haptico.suave()
                nav.nuevo = NuevoTurnoInicio()
            }
            .semantics { contentDescription = if (lectura) "Turno, bloqueado: tu prueba terminó" else "Nuevo turno" }
            .padding(start = 16.dp, end = 22.dp),
        verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        val tinta = if (lectura || off) C.sub else C.petroleoHondo
        Icono(if (lectura) "lock.fill" else if (off) "icloud.slash" else "plus", 20f, tinta)
        Texto(if (off) "Sin conexión" else "Turno", Tipo.lato(17f, Peso.Black), tinta)
    }
}

data class AccionCabecera(val simbolo: String, val etiqueta: String, val numero: Int = 0, val accion: () -> Unit)

/** La cabecera de petróleo: selector de negocio, título, subtítulo, acciones y lo de abajo (segmentado, semana, «El día»). */
@Composable
fun Cabecera(
    titulo: String,
    subtitulo: String = "",
    volver: String = "",
    conNegocio: Boolean = false,
    acciones: List<AccionCabecera> = emptyList(),
    abajo: @Composable ColumnScope.() -> Unit = {},
) {
    val estado = LocalEstado.current
    val nav = runCatching { LocalNavegacion.current }.getOrNull()
    Column(
        Modifier.fillMaxWidth().fondoPetroleo(C.cabecera).statusBarsPadding().padding(start = 16.dp, end = 16.dp, top = 10.dp, bottom = 36.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        if (volver.isNotEmpty()) {
            Row(Modifier.offset(x = (-6).dp).height(44.dp).clickable { nav?.pila?.let { if (it.isNotEmpty()) it.removeAt(it.lastIndex) } }, verticalAlignment = Alignment.CenterVertically) {
                Icono("chevron.left", 22f, C.menta)
                Texto(volver, Tipo.lato(17f), C.menta)
            }
        }
        if (conNegocio) estado.negocio?.let { SelectorNegocio(it) }
        Row(verticalAlignment = Alignment.Bottom, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Texto(titulo, Tipo.lato(34f, Peso.Black), Color.White, Modifier.semantics { heading() }, lineas = 2, tracking = (-1).sp)
                if (subtitulo.isNotEmpty()) Texto(subtitulo, Tipo.lato(15f), Color.White.copy(alpha = 0.78f))
            }
            acciones.forEach { a -> Box { BotonRedondo(a.simbolo, a.etiqueta, a.accion, sobrePetroleo = true, numero = a.numero) } }
        }
        abajo()
    }
}

/** El negocio y la sucursal se eligen arriba de Hoy y de Agenda (Menu en iOS). */
@Composable
fun SelectorNegocio(actual: NegocioDeLaPersona) {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    var abierto by remember { mutableStateOf(false) }
    Box {
        Row(
            Modifier.offset(y = 4.dp).height(36.dp).clip(CircleShape).background(Color.White.copy(alpha = 0.1f))
                .clickable(enabled = estado.negocios.size > 1) { abierto = true }
                .semantics { contentDescription = "Negocio: ${actual.nombre}. Cambiar de negocio" }
                .padding(start = 6.dp, end = 12.dp),
            verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Iniciales(actual.nombre, 26.dp, C.menta, C.petroleoHondo)
            Texto(actual.nombre, Tipo.lato(14f, Peso.Bold), Color.White, lineas = 1)
            Icono("chevron.up.chevron.down", 12f, Color.White)
        }
        if (abierto) {
            androidx.compose.ui.window.Popup(onDismissRequest = { abierto = false }) {
                Column(Modifier.padding(top = 44.dp).widthIn(min = 260.dp).tarjeta(14.dp)) {
                    estado.negocios.forEachIndexed { i, n ->
                        if (i > 0) Box(Modifier.fillMaxWidth().height(1.dp).background(C.linea))
                        Row(
                            Modifier.fillMaxWidth().heightIn(min = 48.dp).clickable { abierto = false; estado.usar(n); alcance.launch { estado.refrescar() } }.padding(horizontal = 14.dp),
                            verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp),
                        ) {
                            Icono(if (n.id == actual.id) "checkmark.circle.fill" else "circle", 18f, if (n.id == actual.id) C.mentaTexto else C.sub)
                            Texto("${n.nombre} · ${n.rolTexto}", Tipo.lato(15f), C.tinta, lineas = 1)
                        }
                    }
                }
            }
        }
    }
}

/** El cuerpo de las pantallas con cabecera: sube 20 dp sobre la cabecera, radio 24 arriba, fondo con puntos. */
@Composable
fun Cuerpo(espacio: Dp = 12.dp, contenido: @Composable ColumnScope.() -> Unit) {
    Column(
        Modifier.offset(y = (-20).dp).fillMaxWidth().clip(RoundedCornerShape(topStart = 24.dp, topEnd = 24.dp)).fondoPuntos()
            .padding(start = 16.dp, end = 16.dp, top = 16.dp, bottom = 140.dp),
        verticalArrangement = Arrangement.spacedBy(espacio), content = contenido,
    )
}
