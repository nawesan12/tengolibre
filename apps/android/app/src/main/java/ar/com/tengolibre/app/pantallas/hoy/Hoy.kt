package ar.com.tengolibre.app.pantallas.hoy

import android.content.Intent
import android.net.Uri
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.MutableTransitionState
import androidx.compose.animation.expandVertically
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.shrinkVertically
import androidx.compose.animation.slideInVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
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
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import ar.com.tengolibre.app.app.AccionesTurno
import ar.com.tengolibre.app.app.Estado
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.nucleo.*
import ar.com.tengolibre.app.pantallas.buscar.CampoBuscar
import ar.com.tengolibre.app.pantallas.mas.BarraAvance
import ar.com.tengolibre.app.pantallas.mas.DestinoMas
import ar.com.tengolibre.app.pantallas.principal.*
import ar.com.tengolibre.app.tema.*
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable
import java.time.Instant
import java.time.temporal.ChronoUnit

// Clon de Pantallas/Hoy/Hoy.swift.

/** 05 Hoy (y «Tu día» de la profesional): la cabecera de petróleo con el negocio, compartir, avisos y el día; abajo,
 *  «Para atender» y la pila con lo próximo primero. Estados 20–25: vacío que enseña, cargando, error con salida,
 *  sin conexión y sólo lectura (dueña y equipo). */
@Composable
fun Hoy() {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    val alcance = rememberCoroutineScope()
    val datos = remember { DatosDia() }
    var verTemprano by remember { mutableStateOf(false) }
    var ofrecer by remember { mutableStateOf<Dia.Hueco?>(null) }
    var ahora by remember { mutableStateOf(Instant.now()) }
    var avisados by remember { mutableStateOf(setOf<String>()) }
    val contexto = LocalContext.current

    val hoy = Formato.enArgentina(ahora).toLocalDate().atStartOfDay(Formato.zona).toInstant()
    val delDia = datos.turnos.filter { Formato.diaISO(it.inicio) == Formato.diaISO(hoy) }
    val profesionales = Dia.profesionalesDelDia(datos.profesionales, delDia, estado.rol)
    val dias = Dia.armar(profesionales, datos.turnos, hoy, Dia.minutosAhora(ahora))
    val lectura = estado.soloLectura || estado.sinConexion

    suspend fun cargar() {
        datos.cargar(estado, hoy, hoy)
        ActividadTurno.sincronizar(datos.turnos.filter { Formato.diaISO(it.inicio) == Formato.diaISO(hoy) }, estado)
        estado.vivo.cargar(estado)
    }

    LaunchedEffect(Unit) { Consejos.abrioHoy() }
    LaunchedEffect(estado.version, estado.negocio?.id) { cargar() }
    LaunchedEffect(Unit) {
        // El minuto corre: «Ahora», lo en curso y los huecos se acomodan solos.
        while (true) {
            delay(30_000)
            ahora = Instant.now()
            if (ahora.epochSecond % 120 < 30) datos.cargar(estado, hoy, hoy)
        }
    }

    fun cargarEn(h: Dia.Hueco) {
        if (estado.soloLectura || estado.sinConexion) return
        nav.nuevo = NuevoTurnoInicio(profesionalId = h.profesionalId, inicio = Dia.fecha(hoy, h.desde))
    }
    val alDeslizar: ((Turno) -> ((String) -> Unit)?) = { t -> if (lectura) null else { e -> alcance.launch { datos.cambiar(estado, t, e); Momentos.cambioDeEstado(estado, nav, t, e) } } }

    Refrescable({ cargar() }, Modifier.fillMaxSize().background(C.fondo)) { Column(Modifier.fillMaxSize().background(C.fondo).verticalScroll(rememberScrollState())) {
        Cabecera(
            if (estado.esProfesional) "Tu día" else "Hoy",
            if (estado.esProfesional) "${Formato.fechaLarga(hoy).split(" ").take(2).joinToString(" ")} · sólo tus turnos" else Formato.fechaLarga(hoy),
            conNegocio = !estado.esProfesional,
            acciones = listOf(
                AccionCabecera("square.and.arrow.up", "Compartir tu link") { nav.compartir = true },
                AccionCabecera("bell", "Avisos", estado.sinLeer) { nav.avisos = true },
            ),
        ) {
            CampoBuscar()
            if (datos.carga == DatosDia.Carga.cargando && datos.turnos.isEmpty()) Esqueleto(118.dp, 20.dp, Modifier.alpha(0.4f))
            else if (dias.isNotEmpty()) ElDia(dias, esHoy = true)
        }
        Cuerpo {
            banners(estado, nav).forEach { Banner(it) }
            when (val c = datos.carga) {
                is DatosDia.Carga.cargando -> if (datos.turnos.isEmpty()) {
                    listOf(80, 72, 80, 80).forEach { Esqueleto(it.dp) }
                    Texto("Trayendo tu día…", Tipo.lato(13f), C.sub, Modifier.fillMaxWidth(), alinear = androidx.compose.ui.text.style.TextAlign.Center)
                }
                is DatosDia.Carga.error -> ErrorConSalida("No pudimos traer tu día", "Tus turnos están a salvo. ${c.texto}") { alcance.launch { cargar() } }
                else -> {
                    if (delDia.none { it.ocupa }) {
                        Vacio(dias, estado, nav)
                        BarraAvance()
                        PlataHoy()
                        AtajosHoy()
                        DescubrirHoy()
                    } else {
                        ParaAtender(atender(estado, nav, datos, delDia, dias, ahora, contexto, alcance) { cargar() })
                        PlataHoy()
                        AtajosHoy()
                        Pila(estado, nav, datos, delDia, dias, ahora, hoy, lectura, verTemprano, { verTemprano = !verTemprano }, avisados,
                            alTocarHueco = { cargarEn(it) }, alAccionHueco = { h, n -> if (!lectura) { if (n > 0) ofrecer = h else cargarEn(h) } }, alDeslizar = alDeslizar)
                        BarraAvance()
                        DescubrirHoy()
                        VistaConsejo(Consejos.widget)
                    }
                }
            }
        }
    } }
    Hoja(ofrecer != null, { ofrecer = null }, listOf(Altura.media, Altura.grande)) {
        ofrecer?.let { h -> OfrecerHueco(h, hoy, datos, listo = { avisados = avisados + h.id }, cerrar = { ofrecer = null }) }
    }
}

// MARK: Banners (sólo lectura y sin conexión)

private fun banners(estado: Estado, nav: Navegacion): List<DatosBanner> {
    val out = mutableListOf<DatosBanner>()
    if (estado.soloLectura) {
        if (estado.esDuena) out.add(DatosBanner("lock.circle", "Tu prueba terminó", "Podés ver todo, pero para cargar o cambiar turnos tenés que suscribirte. Tus clientes siguen viendo tu página.", "Elegir un plan", true) {
            nav.pestana = Pestana.mas; nav.pilaMas.add(DestinoMas.suscripcion)
        })
        else out.add(DatosBanner("lock.circle", "Por ahora, sólo para mirar", "La prueba de ${estado.negocio?.nombre ?: "tu negocio"} terminó. La dueña tiene que activar la suscripción para que vuelvas a cargar turnos.", "", true) {})
    }
    if (estado.sinConexion) {
        val h = estado.guardadoEn?.let { " de las ${Formato.horaDe(it)}" } ?: ""
        out.add(DatosBanner("icloud.slash", "Sin conexión · tu día$h", "Podés ver los turnos y llamar. Para cargar o cambiar algo necesitás conexión.", "", false) {})
    }
    return out
}

// MARK: Para atender

private data class Atender(val id: String, val simbolo: String, val titulo: String, val texto: String, val accion: String, val hacer: () -> Unit)

private fun atender(estado: Estado, nav: Navegacion, datos: DatosDia, delDia: List<Turno>, dias: List<Dia.DeProfesional>, ahora: Instant, contexto: android.content.Context, alcance: CoroutineScope, recargar: suspend () -> Unit): List<Atender> {
    val out = mutableListOf<Atender>()
    for (t in delDia) if (t.estado == "pendiente_pago" && t.inicio.isAfter(ahora.minusSeconds(3600))) {
        out.add(Atender("sena-${t.id}", "creditcard", "Seña pendiente", "${t.cliente.nombre} · hoy ${Formato.horaDe(t.inicio)}", "Ver") { nav.turno = t })
    }
    val esperando = datos.espera.filter { it.avisadoEn == null }
    if (esperando.isNotEmpty() && !estado.esProfesional) {
        val match = dias.flatMap { it.huecos }.firstOrNull { datos.esperanPara(it).isNotEmpty() }
        val quien = match?.let { datos.esperanPara(it).firstOrNull()?.nombre?.split(" ")?.firstOrNull() }
        out.add(Atender("espera", "hourglass", "${esperando.size} esperando un lugar",
            if (match != null && quien != null) "A $quien le sirve el hueco de las ${Dia.hhmm(match.desde)}" else "En la lista de espera", "Ver") { nav.pestana = Pestana.espera })
    }
    for (l in datos.paraLlamar) out.add(Atender("llamar-${l.turnoId}", "phone", "Llamar a ${l.cliente}", "${l.motivoTexto} · ${Formato.horaDe(l.inicio)}", "Llamar") {
        contexto.startActivity(Intent(Intent.ACTION_DIAL, Uri.parse("tel:${l.telefono.filter { it in "+0123456789" }}")).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK))
        alcance.launch { runCatching { estado.api.hacer("POST", "/v1/panel/turnos/${l.turnoId}/llamado") }; recargar() }
    })
    for (r in datos.reprogramar) out.add(Atender("repro-${r.id}", "calendar.badge.clock", "Mover a ${r.turno.cliente.nombre}", "${r.turno.profesional} pidió días libres · ${Formato.dia(r.turno.inicio)}", "Mover") {
        alcance.launch { runCatching { estado.api.get<Turno>("/v1/panel/turnos/${r.turno.id}") }.getOrNull()?.let { nav.turno = it } }
    })
    for (p in datos.pedidos) out.add(Atender("pedido-${p.id}", "beach.umbrella", "${p.profesional} pide días libres", "${p.motivoTexto} · ${p.turnos} ${if (p.turnos == 1) "turno" else "turnos"} para mover", "Ver") {
        nav.pestana = Pestana.mas; nav.pilaMas.add(DestinoMas.diasLibres)
    })
    datos.resenaSinResponder?.let { r ->
        out.add(Atender("resena", "star.bubble", "Reseña sin responder", "${r.nombre} · ${r.puntaje} de 5", "Responder") {
            if (estado.esProfesional) nav.pestana = Pestana.resenas else { nav.pestana = Pestana.mas; nav.pilaMas.add(DestinoMas.resenas) }
        })
    }
    if (datos.mpConectado == false && estado.esDuena) out.add(Atender("mp", "creditcard.and.123", "Cobrá señas con Mercado Pago", "Así nadie reserva y no viene", "Conectar") {
        nav.pestana = Pestana.mas; nav.pilaMas.add(DestinoMas.cobros)
    })
    return out
}

@Composable
private fun ParaAtender(lista: List<Atender>) {
    if (lista.isEmpty()) return
    Column(Modifier.fillMaxWidth().tarjeta(22.dp)) {
        TextoRotulo("Para atender", modifier = Modifier.padding(start = 16.dp, end = 16.dp, top = 14.dp, bottom = 4.dp))
        lista.forEachIndexed { i, a ->
            if (i > 0) Box(Modifier.fillMaxWidth().height(1.dp).background(C.linea))
            Row(Modifier.fillMaxWidth().heightIn(min = 64.dp).padding(start = 16.dp, end = 12.dp, top = 8.dp, bottom = 8.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                Box(Modifier.width(24.dp), contentAlignment = Alignment.Center) { Icono(a.simbolo, 20f, C.mentaTexto) }
                Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                    Texto(a.titulo, Tipo.lato(15f, Peso.Bold), C.tinta)
                    Texto(a.texto, Tipo.lato(13f), C.sub, lineas = 2)
                }
                BotonContorno(a.accion, a.hacer, alto = 44.dp, tam = 15f, lleno = false)
            }
        }
    }
}

// MARK: La pila

private sealed class Item(val id: String) {
    class sep(val s: String) : Item("sep-$s")
    class ahora(val h: String) : Item("ahora")
    class talon(val t: Turno) : Item(t.id)
    class hueco(val h: Dia.Hueco) : Item(h.id)
    class plegado(val texto: String, val pasados: List<Turno>) : Item("plegado")
}

private fun items(estado: Estado, delDia: List<Turno>, dias: List<Dia.DeProfesional>, ahora: Instant): List<Item> {
    val m = Dia.minutosAhora(ahora)
    val ocupan = delDia.filter { it.ocupa }.sortedBy { it.inicio }
    val pasados = ocupan.filter { !it.fin.isAfter(ahora) }
    val enCurso = ocupan.filter { !it.inicio.isAfter(ahora) && it.fin.isAfter(ahora) }
    val despues = ocupan.filter { it.inicio.isAfter(ahora) }
    val huecos = dias.flatMap { it.huecos }
    val proximoHueco = huecos.filter { it.hasta > m }.minByOrNull { it.desde }
    val out = mutableListOf<Item>(Item.sep("AHORA"), Item.ahora(Formato.horaDe(ahora)))
    out += enCurso.map { Item.talon(it) }
    if (proximoHueco != null && (proximoHueco.desde < m + 60 || enCurso.isEmpty())) out.add(Item.hueco(proximoHueco))
    if (despues.isNotEmpty() || huecos.size > 1) {
        out.add(Item.sep("DESPUÉS"))
        val resto = mutableListOf<Pair<Int, Item>>()
        despues.forEach { resto.add(Formato.minutosDelDia(it.inicio) to Item.talon(it)) }
        huecos.filter { it != proximoHueco && it.desde > m }.forEach { resto.add(it.desde to Item.hueco(it)) }
        out += resto.sortedBy { it.first }.map { it.second }
    }
    if (pasados.isNotEmpty()) {
        val at = pasados.count { it.estado == "atendido" }; val au = pasados.count { it.estado == "ausente" }
        var t = "Más temprano: $at ${if (at == 1) "atendido" else "atendidos"}"
        if (au > 0) t += " y $au ${if (au == 1) "ausente" else "ausentes"}"
        if (at == 0 && au == 0) t = "Más temprano: ${pasados.size} ${if (pasados.size == 1) "turno" else "turnos"}"
        out.add(Item.plegado(t, pasados))
    }
    return out
}

@Composable
private fun Pila(
    estado: Estado, nav: Navegacion, datos: DatosDia, delDia: List<Turno>, dias: List<Dia.DeProfesional>, ahora: Instant, hoy: Instant,
    lectura: Boolean, verTemprano: Boolean, alPlegar: () -> Unit, avisados: Set<String>,
    alTocarHueco: (Dia.Hueco) -> Unit, alAccionHueco: (Dia.Hueco, Int) -> Unit, alDeslizar: (Turno) -> ((String) -> Unit)?,
) {
    val lista = items(estado, delDia, dias, ahora)
    val primerHueco = lista.firstOrNull { it is Item.hueco }?.id
    if (!lectura && lista.any { it is Item.talon }) VistaConsejo(Consejos.deslizar)
    lista.forEachIndexed { i, it ->
        Box(Modifier.entrada(i)) {
            when (it) {
                is Item.sep -> Rotulo(it.s)
                is Item.ahora -> Row(Modifier.padding(horizontal = 0.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Pulso(tam = 8.dp)
                    Box(Modifier.height(26.dp).clip(CircleShape).background(C.menta).padding(horizontal = 10.dp), contentAlignment = Alignment.Center) {
                        Texto("AHORA · ${it.h}", Tipo.lato(12f, Peso.Black, numeros = true), C.petroleoHondo, tracking = 0.5.sp)
                    }
                    Box(Modifier.weight(1f).height(2.dp).clip(CircleShape).background(C.menta))
                }
                is Item.talon -> if (it.t.id in datos.recien) EntraDeArriba(it.t.id) { VistaTalon(it.t, !estado.esProfesional, ahora, { nav.turno = it.t }, alDeslizar(it.t)) }
                    else VistaTalon(it.t, !estado.esProfesional, ahora, { nav.turno = it.t }, alDeslizar(it.t))
                is Item.hueco -> {
                    val h = it.h
                    val n = datos.esperanPara(h).size
                    val oferta = datos.ofertaPara(h, hoy)
                    ConConsejo(h.id == primerHueco && !lectura, if (n > 0) Consejos.avisar else Consejos.hueco) {
                        VistaHueco(h, if (lectura) "Ver" else if (oferta != null) "Ofreciendo" else if (n > 0) "Avisar a $n" else "Cargar",
                            if (n > 0 && !lectura) "megaphone" else "plus", h.id in avisados || oferta != null, { alTocarHueco(h) }, { alAccionHueco(h, n) })
                    }
                }
                is Item.plegado -> Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                    Row(
                        Modifier.fillMaxWidth().heightIn(min = 52.dp).contorno(18.dp).clip(RoundedCornerShape(18.dp)).clickable(onClick = alPlegar).padding(horizontal = 16.dp),
                        verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp),
                    ) {
                        Icono("clock.arrow.circlepath", 18f, C.sub)
                        Texto(it.texto, Tipo.lato(15f), C.sub, Modifier.weight(1f))
                        Icono(if (verTemprano) "chevron.up" else "chevron.down", 15f, C.sub)
                    }
                    AnimatedVisibility(verTemprano, enter = expandVertically(Movimiento.entrar()) + fadeIn(Movimiento.entrar()), exit = shrinkVertically(Movimiento.entrar()) + fadeOut(Movimiento.entrar())) {
                        Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                            it.pasados.forEach { t -> VistaTalon(t, !estado.esProfesional, ahora, { nav.turno = t }, alDeslizar(t)) }
                        }
                    }
                }
            }
        }
    }
}

/** El talón menta que llegó mientras se miraba: entra desde arriba (.transition(.move(edge: .top)) en iOS). */
@Composable
private fun EntraDeArriba(id: String, contenido: @Composable () -> Unit) {
    val visible = remember(id) { MutableTransitionState(false).apply { targetState = true } }
    AnimatedVisibility(visible, enter = slideInVertically(Movimiento.entrar()) { -it } + fadeIn(Movimiento.entrar())) { contenido() }
}

// MARK: Vacío que enseña

@Composable
private fun Vacio(dias: List<Dia.DeProfesional>, estado: Estado, nav: Navegacion) {
    val (d, h) = Dia.rango(dias)
    VacioQueEnsena("${Dia.hhmm(d)} a ${Dia.hhmm(h)} · todo libre", "Hoy no tenés turnos todavía",
        "Compartí tu link para que te reserven solos, o cargá el que te pidieron por teléfono.") {
        Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
            BotonMenta("Compartir mi link", { nav.compartir = true }, alto = 52.dp, conSombra = false, simbolo = "square.and.arrow.up")
            if (!estado.soloLectura) BotonContorno("Cargar un turno", { nav.nuevo = NuevoTurnoInicio() }, simbolo = "plus")
        }
    }
}

/** Un banner arriba del día (sólo lectura, sin conexión, primeros pasos). */
data class DatosBanner(val simbolo: String, val titulo: String, val texto: String, val accion: String, val petroleo: Boolean, val hacer: () -> Unit)

@Composable
fun Banner(b: DatosBanner) {
    val forma = RoundedCornerShape(22.dp)
    Column(
        Modifier.fillMaxWidth().sombra(forma, hex(0x0F2328, 0.18f), 14.dp, y = 10.dp).clip(forma)
            .then(if (b.petroleo) Modifier.fondoPetroleo(C.bloquePetroleo) else Modifier.background(C.tarjeta)).padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            Icono(b.simbolo, 22f, if (b.petroleo) C.menta else C.tinta)
            Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Texto(b.titulo, Tipo.lato(16f, Peso.Bold), if (b.petroleo) Color.White else C.tinta)
                Texto(b.texto, Tipo.lato(14f).copy(lineHeight = 19.sp), if (b.petroleo) Color.White.copy(alpha = 0.8f) else C.sub)
            }
        }
        if (b.accion.isNotEmpty()) BotonMenta(b.accion, b.hacer, alto = 48.dp, tam = 16f, conSombra = false)
    }
}

/** «Avisar a la lista»: qué servicio se ofrece en ese hueco y a quiénes les llega, en orden. */
@Composable
fun ColumnScope.OfrecerHueco(hueco: Dia.Hueco, dia: Instant, datos: DatosDia, listo: () -> Unit, cerrar: () -> Unit) {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    var servicio by remember { mutableStateOf<String?>(null) }
    var candidatos by remember { mutableStateOf<List<Anotado>>(emptyList()) }
    var enviando by remember { mutableStateOf(false) }
    val pide = datos.esperanPara(hueco).map { it.servicioId }.toSet()
    val delPro = datos.profesionales.firstOrNull { it.id == hueco.profesionalId }?.servicios ?: emptyList()
    val opciones = datos.servicios.filter { it.activo && it.id in pide && (delPro.isEmpty() || it.id in delPro) && it.duracion <= hueco.minutos }

    @Serializable data class Esperan(val espera: List<Anotado> = emptyList())
    fun traer() {
        val s = servicio ?: return
        alcance.launch {
            candidatos = estado.api.opcional<Esperan>("/v1/panel/espera/candidatos", listOf("servicio" to s, "profesional" to hueco.profesionalId, "inicio" to Fecha.texto(Dia.fecha(dia, hueco.desde))))?.espera ?: emptyList()
        }
    }
    LaunchedEffect(Unit) { if (opciones.size == 1) { servicio = opciones[0].id; traer() } }

    BarraHoja("Avisar a la lista", cerrar)
    Column(Modifier.weight(1f).verticalScroll(rememberScrollState()).padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Texto("Libre ${Formato.duracion(hueco.minutos)} · ${Dia.hhmm(hueco.desde)} con ${hueco.profesional}", Tipo.lato(20f, Peso.Black), C.tinta)
            Texto("Se lo ofrecemos de a una persona por vez, en el orden en que se anotaron. Cada una tiene un rato para tomarlo.", Tipo.lato(14f), C.sub)
        }
        TextoRotulo("Qué se ofrece")
        if (opciones.isEmpty()) Texto("Nadie de la lista espera un servicio que entre en este hueco.", Tipo.lato(15f), C.sub)
        opciones.chunked(2).forEach { fila ->
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                fila.forEach { s ->
                    val on = servicio == s.id
                    val forma = RoundedCornerShape(16.dp)
                    Column(
                        Modifier.weight(1f).heightIn(min = 64.dp).clip(forma).background(if (on) C.seleccion else C.fondo)
                            .then(if (on) Modifier else Modifier.border(1.5.dp, C.linea, forma)).clickable { servicio = s.id; traer() }.padding(horizontal = 12.dp, vertical = 10.dp),
                        verticalArrangement = Arrangement.Center,
                    ) {
                        Texto(s.nombre, Tipo.lato(15f, Peso.Bold), if (on) C.seleccionTexto else C.tinta)
                        Texto(Formato.duracion(s.duracion), Tipo.lato(13f), (if (on) C.seleccionTexto else C.tinta).copy(alpha = 0.8f))
                    }
                }
                if (fila.size == 1) Box(Modifier.weight(1f))
            }
        }
        if (candidatos.isNotEmpty()) {
            TextoRotulo("A quiénes, en orden")
            candidatos.forEachIndexed { i, a ->
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    Texto("${i + 1}", Tipo.lato(15f, Peso.Black, numeros = true), C.tinta, Modifier.width(24.dp))
                    Iniciales(a.nombre, 36.dp)
                    Column {
                        Texto(a.nombre, Tipo.lato(15f, Peso.Bold), C.tinta)
                        Texto("Espera desde el ${Formato.dia(a.creadoEn)}", Tipo.lato(12f), C.sub)
                    }
                }
            }
        }
    }
    Box(Modifier.fillMaxWidth().background(C.hoja).padding(16.dp)) {
        BotonMenta(if (enviando) "Avisando…" else "Avisar a la lista", {
            val s = servicio ?: return@BotonMenta
            enviando = true
            alcance.launch {
                try { AccionesTurno.ofrecer(estado, s, hueco.profesionalId, Dia.fecha(dia, hueco.desde)); listo(); cerrar() }
                catch (e: Exception) { estado.avisar(e) }
                finally { enviando = false }
            }
        }, habilitado = servicio != null && !enviando, simbolo = "megaphone")
    }
}

/** La barra de arriba de una hoja: «Cancelar» a la izquierda, el título al centro y una acción a la derecha
 *  (la toolbar de las hojas en iOS, .navigationBarTitleDisplayMode(.inline)). */
@Composable
fun BarraHoja(titulo: String, cancelar: () -> Unit, textoCancelar: String = "Cancelar", accion: String = "", habilitada: Boolean = true, alAccion: () -> Unit = {}) {
    Box(Modifier.fillMaxWidth().height(52.dp).padding(horizontal = 16.dp)) {
        Box(Modifier.align(Alignment.CenterStart)) { BotonTexto(textoCancelar, cancelar) }
        Texto(titulo, Tipo.lato(17f, Peso.Bold), C.tinta, Modifier.align(Alignment.Center), lineas = 1)
        if (accion.isNotEmpty()) Box(Modifier.align(Alignment.CenterEnd).alpha(if (habilitada) 1f else 0.4f)) { BotonTexto(accion, { if (habilitada) alAccion() }) }
    }
}
