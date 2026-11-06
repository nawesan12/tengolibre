package ar.com.tengolibre.app.pantallas.agenda

import androidx.compose.animation.core.animateDpAsState
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectHorizontalDragGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.nucleo.*
import ar.com.tengolibre.app.pantallas.hoy.*
import ar.com.tengolibre.app.pantallas.principal.*
import ar.com.tengolibre.app.tema.*
import kotlinx.coroutines.launch
import java.time.DayOfWeek
import java.time.Instant
import java.time.temporal.TemporalAdjusters
import kotlin.math.abs

// Clon de Pantallas/Agenda/Agenda.swift.

private val POR_HORA = 76.dp

/** 07 Agenda · día y 08 Agenda · semana. */
@Composable
fun Agenda() {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    val alcance = rememberCoroutineScope()
    val datos = remember { DatosDia() }
    var modo by remember { mutableStateOf("dia") }
    var dia by remember { mutableStateOf(Formato.enArgentina(Instant.now()).toLocalDate()) }
    var indice by remember { mutableIntStateOf(1) } // 0 = Todos
    val ahora = Instant.now()

    val lunes = dia.with(TemporalAdjusters.previousOrSame(DayOfWeek.MONDAY))
    val semana = (0L..6L).map { lunes.plusDays(it) }
    fun inst(d: java.time.LocalDate) = d.atStartOfDay(Formato.zona).toInstant()
    val pros = Dia.profesionalesDelDia(datos.profesionales, datos.turnos, estado.rol)
    val esHoy = dia == Formato.enArgentina(Instant.now()).toLocalDate()
    val dias = Dia.armar(pros, datos.turnos, inst(dia), if (esHoy) Dia.minutosAhora(ahora) else null)
    val elegida = if (indice == 0 || dias.isEmpty()) null else dias[minOf(indice - 1, dias.lastIndex)]
    val lectura = estado.soloLectura || estado.sinConexion

    suspend fun cargar() {
        datos.cargar(estado, inst(semana[0]), inst(semana[6]))
        if (estado.esProfesional) indice = 1
    }
    LaunchedEffect(estado.version, lunes, estado.negocio?.id) { cargar() }

    Box(Modifier.fillMaxSize().background(C.fondo)) {
        Column(
            Modifier.fillMaxSize().verticalScroll(rememberScrollState()).pointerInput(modo, dias.size) {
                var total = 0f
                detectHorizontalDragGestures(onDragStart = { total = 0f }, onDragEnd = {
                    if (modo == "dia" && abs(total) > 40.dp.toPx()) {
                        indice = (indice + (if (total < 0) 1 else -1) + dias.size + 1) % (dias.size + 1)
                        Haptico.seleccion()
                    }
                }) { _, d -> total += d }
            },
        ) {
            Cabecera(
                "Agenda",
                if (modo == "dia") Formato.fechaLarga(inst(dia)) else "semana del ${semana[0].dayOfMonth} al ${Formato.dia(inst(semana[6]))}",
                conNegocio = modo == "dia" && !estado.esProfesional,
                acciones = if (modo == "dia") listOf(
                    AccionCabecera("square.and.arrow.up", "Compartir tu link") { nav.compartir = true },
                    AccionCabecera("bell", "Avisos", estado.sinLeer) { nav.avisos = true },
                ) else emptyList(),
            ) {
                Segmentado(listOf("dia" to "Día", "semana" to "Semana"), modo, { modo = it })
                if (modo == "dia") TiraSemana(semana, dia, datos) { dia = it }
            }
            Cuerpo {
                val c = datos.carga
                if (c == DatosDia.Carga.cargando && datos.turnos.isEmpty()) repeat(5) { Esqueleto(70.dp) }
                else if (c is DatosDia.Carga.error) ErrorConSalida("No pudimos traer la agenda", c.texto) { alcance.launch { cargar() } }
                else if (modo == "dia") {
                    if (elegida != null) LineaDeTiempo(elegida, inst(dia), esHoy, ahora, lectura, nav)
                    else Todos(datos, dias, inst(dia), ahora, estado.soloLectura, nav)
                } else VistaSemana(semana, pros, datos) { d -> dia = d; modo = "dia" }
            }
        }
        if (modo == "dia" && dias.size > 1) {
            Column(Modifier.align(Alignment.BottomCenter).navigationBarsPadding().padding(bottom = 16.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Box(Modifier.padding(horizontal = 24.dp)) { VistaConsejo(Consejos.capsula) }
                Capsula(dias, elegida, indice) { indice = it }
            }
        }
    }
}

/** La semana arriba, con el anillo menta en el día elegido. */
@Composable
private fun TiraSemana(semana: List<java.time.LocalDate>, dia: java.time.LocalDate, datos: DatosDia, alElegir: (java.time.LocalDate) -> Unit) {
    var arrastre by remember { mutableStateOf(0f) }
    Row(
        Modifier.fillMaxWidth().pointerInput(dia) {
            detectHorizontalDragGestures(onDragStart = { arrastre = 0f }, onDragEnd = { if (abs(arrastre) > 30.dp.toPx()) alElegir(dia.plusDays(if (arrastre < 0) 7 else -7)) }) { _, d -> arrastre += d }
        },
        horizontalArrangement = Arrangement.SpaceBetween,
    ) {
        semana.forEach { d ->
            val on = d == dia
            val iso = d.toString()
            val hay = datos.turnos.any { Formato.diaISO(it.inicio) == iso && it.ocupa }
            val i = d.atStartOfDay(Formato.zona).toInstant()
            Column(
                Modifier.size(44.dp, 60.dp).clip(CircleShape).then(if (on) Modifier.border(2.dp, C.menta, CircleShape) else Modifier)
                    .clickable { alElegir(d); Haptico.seleccion() }
                    .semantics { contentDescription = Formato.fechaLarga(i) + if (hay) ", con turnos" else ""; selected = on },
                horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(2.dp, Alignment.CenterVertically),
            ) {
                Texto(Formato.fechaLarga(i).take(1).uppercase(), Tipo.lato(12f, Peso.Bold), Color.White.copy(alpha = 0.7f))
                Texto("${d.dayOfMonth}", Tipo.lato(18f, Peso.Bold, numeros = true), Color.White)
                Box(Modifier.size(5.dp).clip(CircleShape).background(C.menta).alpha(if (hay) 1f else 0f))
            }
        }
    }
}

/** Una profesional: la línea de tiempo (76 dp por hora). */
@Composable
private fun LineaDeTiempo(p: Dia.DeProfesional, dia: Instant, esHoy: Boolean, ahora: Instant, lectura: Boolean, nav: Navegacion) {
    val (desde, hasta) = Dia.rango(listOf(p))
    fun y(m: Int): Dp = POR_HORA * ((m - desde) / 60f)
    Box(Modifier.fillMaxWidth().height(y(hasta) + 8.dp).padding(top = 4.dp)) {
        var h = desde
        while (h <= hasta) {
            Row(Modifier.offset(y = y(h)), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Texto("${h / 60}:00", Tipo.lato(12f, Peso.Bold, numeros = true), C.sub, Modifier.width(40.dp).offset(y = (-8).dp))
                Box(Modifier.weight(1f).height(1.dp).background(C.linea))
            }
            h += 60
        }
        p.ocupado.forEach { o ->
            Box(Modifier.padding(start = 48.dp).offset(y = y(o.tramo.desde) + 2.dp).height(maxOf(30.dp, y(o.tramo.hasta) - y(o.tramo.desde)) - 4.dp)) {
                BloqueTalon(o.turno, ahora) { nav.turno = o.turno }
            }
        }
        p.huecos.forEach { hu ->
            Box(Modifier.padding(start = 48.dp).offset(y = y(hu.desde) + 2.dp).height(maxOf(30.dp, y(hu.hasta) - y(hu.desde)) - 4.dp)) {
                Hueco(Modifier.fillMaxSize().clickable { if (!lectura) nav.nuevo = NuevoTurnoInicio(profesionalId = hu.profesionalId, inicio = Dia.fecha(dia, hu.desde)) }, 14.dp, alinear = Alignment.CenterStart) {
                    Box(Modifier.padding(horizontal = 12.dp)) {
                        PastillaPapel {
                            Icono("plus.circle", 14f, C.huecoTinta); Spacer(Modifier.width(6.dp))
                            Texto("Libre ${Formato.duracion(hu.minutos)} · tocá para cargar", Tipo.lato(14f, Peso.Black), C.huecoTinta)
                        }
                    }
                }
            }
        }
        if (esHoy) {
            val m = Dia.minutosAhora(ahora)
            if (m in desde..hasta) {
                Row(Modifier.padding(start = 40.dp).offset(y = y(m) - 5.dp), verticalAlignment = Alignment.CenterVertically) {
                    Box(Modifier.size(10.dp).clip(CircleShape).background(C.fondo).padding(1.5.dp).clip(CircleShape).background(C.menta))
                    Box(Modifier.weight(1f).height(2.dp).background(C.menta))
                }
            }
        }
    }
}

@Composable
private fun BloqueTalon(t: Turno, ahora: Instant, alTocar: () -> Unit) {
    val est = t.visible(ahora)
    val forma = RoundedCornerShape(14.dp)
    Row(
        Modifier.fillMaxSize().sombra(forma, hex(0x0F2328, 0.18f), 12.dp, y = 8.dp).clip(forma).background(C.tarjeta)
            .then(if (est == EstadoTurno.enCurso) Modifier.border(2.dp, C.seleccion, forma) else Modifier)
            .alpha(if (est == EstadoTurno.atendido) 0.55f else 1f).clickable(onClick = alTocar).padding(horizontal = 12.dp),
        verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        Row(Modifier.fillMaxHeight(), verticalAlignment = Alignment.CenterVertically) {
            Texto(Formato.horaDe(t.inicio), Tipo.lato(15f, Peso.Black, numeros = true), C.tinta, Modifier.padding(end = 10.dp))
            Troquel()
        }
        Column(Modifier.weight(1f)) {
            Texto(t.cliente.nombre, Tipo.lato(15f, Peso.Bold), C.tinta, lineas = 1)
            Texto(t.servicio, Tipo.lato(12f), C.sub, lineas = 1)
        }
        Icono(est.simbolo, 18f, C.tinta, descripcion = est.texto)
    }
}

/** Todos: la pila del día de todas. */
@Composable
private fun Todos(datos: DatosDia, dias: List<Dia.DeProfesional>, dia: Instant, ahora: Instant, soloLectura: Boolean, nav: Navegacion) {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    val turnos = datos.turnos.filter { Formato.diaISO(it.inicio) == Formato.diaISO(dia) && it.ocupa }.sortedBy { it.inicio }
    val huecos = dias.flatMap { it.huecos }.sortedBy { it.desde }
    if (turnos.isEmpty() && huecos.isEmpty()) VacioQueEnsena("Sin horario ese día", "Nadie atiende este día", "Configurá los horarios de cada profesional desde Más.")
    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        turnos.forEach { t ->
            VistaTalon(t, ahora = ahora, alTocar = { nav.turno = t },
                alDeslizar = if (soloLectura) null else { e -> alcance.launch { datos.cambiar(estado, t, e); Momentos.cambioDeEstado(estado, nav, t, e) } })
        }
        if (huecos.isNotEmpty()) {
            Rotulo("Libres")
            huecos.forEach { h ->
                val abrir = { nav.nuevo = NuevoTurnoInicio(profesionalId = h.profesionalId, inicio = Dia.fecha(dia, h.desde)) }
                VistaHueco(h, "Cargar", "plus", alTocar = abrir, alAccion = abrir)
            }
        }
    }
}

/** La cápsula de abajo: profesional anterior / siguiente, con los puntos (incluye «Todos»). */
@Composable
private fun Capsula(dias: List<Dia.DeProfesional>, elegida: Dia.DeProfesional?, indice: Int, alCambiar: (Int) -> Unit) {
    val total = dias.size + 1
    val nombre = elegida?.let { "${it.nombre.split(" ").firstOrNull().orEmpty()} · ${it.ocupado.size} ${if (it.ocupado.size == 1) "turno" else "turnos"}" } ?: "Todos"
    Row(
        Modifier.sombra(CircleShape, Color.Black.copy(alpha = 0.4f), 15.dp, y = 14.dp).height(56.dp).clip(CircleShape).background(C.petroleoHondo).padding(horizontal = 6.dp),
        verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(2.dp),
    ) {
        Box(Modifier.size(44.dp).clip(CircleShape).clickable { alCambiar((indice - 1 + total) % total) }.semantics { contentDescription = "Profesional anterior" }, contentAlignment = Alignment.Center) { Icono("chevron.left", 20f, Color.White) }
        Column(Modifier.widthIn(min = 132.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(3.dp)) {
            Texto(nombre, Tipo.lato(15f, Peso.Bold), Color.White)
            Row(horizontalArrangement = Arrangement.spacedBy(5.dp)) {
                (0 until total).forEach { i ->
                    val ancho by animateDpAsState(if (i == indice) 16.dp else 5.dp, Movimiento.entrar(), label = "punto")
                    Box(Modifier.size(ancho, 5.dp).clip(CircleShape).background(if (i == indice) C.menta else Color.White.copy(alpha = 0.35f)))
                }
            }
        }
        Box(Modifier.size(44.dp).clip(CircleShape).clickable { alCambiar((indice + 1) % total) }.semantics { contentDescription = "Profesional siguiente" }, contentAlignment = Alignment.Center) { Icono("chevron.right", 20f, Color.White) }
    }
}

/** 08 Semana: una tarjeta por día con la franja de cada profesional y las horas libres. */
@Composable
private fun VistaSemana(semana: List<java.time.LocalDate>, pros: List<Profesional>, datos: DatosDia, alElegir: (java.time.LocalDate) -> Unit) {
    val hoy = Formato.enArgentina(Instant.now()).toLocalDate()
    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        semana.forEach { d ->
            val i = d.atStartOfDay(Formato.zona).toInstant()
            val delDia = Dia.armar(pros, datos.turnos, i, if (d == hoy) Dia.minutosAhora() else null)
            if (delDia.none { it.franjas.isNotEmpty() }) return@forEach
            val (desde, hasta) = Dia.rango(delDia)
            val total = maxOf(60, hasta - desde).toFloat()
            val libres = delDia.sumOf { it.libres }
            Column(
                Modifier.fillMaxWidth().tarjeta().then(if (d == hoy) Modifier.border(2.dp, C.menta, RoundedCornerShape(Radio.tarjeta)) else Modifier)
                    .clickable { alElegir(d) }.padding(14.dp).semantics(mergeDescendants = true) { contentDescription = "${Formato.fechaLarga(i)}, ${Formato.libres(libres)}" },
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Texto(Formato.mayuscula(Formato.fechaLarga(i).split(" de ").firstOrNull().orEmpty()), Tipo.lato(16f, Peso.Bold), C.tinta, Modifier.weight(1f))
                    Rayas(Modifier.size(14.dp, 10.dp).clip(RoundedCornerShape(3.dp)), quieto = true, paso = 5.dp, color = C.menta, fondo = Color.Transparent)
                    Spacer(Modifier.width(4.dp))
                    Texto(Formato.libres(libres), Tipo.lato(13f, Peso.Bold, numeros = true), C.mentaTexto)
                }
                delDia.forEach { p ->
                    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Texto(p.nombre.split(" ").firstOrNull() ?: p.nombre, Tipo.lato(12f, Peso.Bold), C.sub, Modifier.width(44.dp), lineas = 1)
                        BoxWithConstraints(Modifier.weight(1f).height(10.dp)) {
                            val ancho = maxWidth
                            Box(Modifier.fillMaxSize().clip(RoundedCornerShape(3.dp)).background(C.chip))
                            p.franjas.forEach { f ->
                                Rayas(Modifier.offset(x = ancho * ((f.desde - desde) / total)).width(ancho * (f.largo / total)).fillMaxHeight().clip(RoundedCornerShape(3.dp)),
                                    quieto = true, paso = 6.dp, color = C.menta, fondo = Color.Transparent)
                            }
                            p.ocupado.forEach { o ->
                                Box(Modifier.offset(x = ancho * ((o.tramo.desde - desde) / total)).width(maxOf(2.dp, ancho * (o.tramo.largo / total) - 2.dp)).fillMaxHeight()
                                    .clip(RoundedCornerShape(3.dp)).background(C.ocupado))
                            }
                        }
                    }
                }
            }
        }
    }
}
