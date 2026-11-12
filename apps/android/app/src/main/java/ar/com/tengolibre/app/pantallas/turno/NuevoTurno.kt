package ar.com.tengolibre.app.pantallas.turno

import android.app.Activity
import android.content.ActivityNotFoundException
import android.content.Intent
import android.provider.ContactsContract
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
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
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
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
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import ar.com.tengolibre.app.app.AccionesTurno
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.nucleo.*
import ar.com.tengolibre.app.pantallas.hoy.BarraHoja
import ar.com.tengolibre.app.pantallas.hoy.Momentos
import ar.com.tengolibre.app.pantallas.principal.LocalNavegacion
import ar.com.tengolibre.app.pantallas.principal.NuevoTurnoInicio
import ar.com.tengolibre.app.tema.*
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable
import java.time.Instant
import kotlin.math.PI
import kotlin.math.abs
import kotlin.math.sin

// Clon de Pantallas/Turno/NuevoTurno.swift.

/** La vista previa del turno fijo (turno-fijo.md): las fechas libres ya elegidas; las que chocan, con su
 *  alternativa o para sacarlas. */
@Serializable
data class PreviaSerie(
    val cadaSemanas: Int = 1,
    val servicio: String = "",
    val profesional: String = "",
    val libres: Int = 0,
    val chocan: Int = 0,
    val fechas: List<FechaSerie> = emptyList(),
) {
    @Serializable
    data class FechaSerie(val inicio: FechaApi, val fecha: String = "", val hora: String = "", val estado: String = "", val motivo: String? = null, val alternativa: Alt? = null) {
        @Serializable data class Alt(val inicio: FechaApi, val hora: String = "")
    }
}

/** 10 Nuevo turno: clienta (buscar, o desde Contactos), servicio, con quién, horario libre y sobreturno; además
 *  turno fijo (cada N semanas, con vista previa) y nota, como en la web. «Guardar · hoy 16:00 con Carla» fijo abajo.
 *  Al guardar, 11: el talón reemplaza al hueco y aparece «Deshacer» durante 5 s. */
@Composable
fun ColumnScope.NuevoTurno(inicio: NuevoTurnoInicio, cerrar: () -> Unit) {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    val alcance = rememberCoroutineScope()
    val contexto = LocalContext.current
    var servicios by remember { mutableStateOf<List<Servicio>>(emptyList()) }
    var profesionales by remember { mutableStateOf<List<Profesional>>(emptyList()) }
    var cliente by remember { mutableStateOf<ClienteDelTurno?>(null) }
    var buscar by remember { mutableStateOf("") }
    var encontrados by remember { mutableStateOf<List<Cliente>>(emptyList()) }
    var nueva by remember { mutableStateOf(false) }
    var nombreNuevo by remember { mutableStateOf("") }
    var telefonoNuevo by remember { mutableStateOf("") }
    var emailNuevo by remember { mutableStateOf("") }
    var servicio by remember { mutableStateOf<String?>(null) }
    var profesional by remember { mutableStateOf<String?>(null) }
    var fecha by remember { mutableStateOf(Formato.enArgentina(Instant.now()).toLocalDate()) }
    var libres by remember { mutableStateOf<List<HorarioLibre>>(emptyList()) }
    var cargandoLibres by remember { mutableStateOf(false) }
    var elegido by remember { mutableStateOf<HorarioLibre?>(null) }
    var otroDia by remember { mutableStateOf(false) }
    var sobreturno by remember { mutableStateOf(false) }
    var horaSobreturno by remember { mutableIntStateOf(Dia.minutosAhora()) }
    var fijo by remember { mutableStateOf(false) }
    var cadaSemanas by remember { mutableIntStateOf(1) }
    var cantidad by remember { mutableIntStateOf(4) }
    var previa by remember { mutableStateOf<PreviaSerie?>(null) }
    var fechasSerie by remember { mutableStateOf<Set<Instant>>(emptySet()) }
    var nota by remember { mutableStateOf("") }
    var guardando by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf("") }
    val temblor = remember { Animatable(0f) }

    val hoy = Formato.enArgentina(Instant.now()).toLocalDate()
    val esHoy = fecha == hoy
    val diaInst = fecha.atStartOfDay(Formato.zona).toInstant()
    val pros = profesionales.filter { it.activo && (servicio == null || it.servicios.isEmpty() || servicio in it.servicios) }
    val inicioElegido: Instant? = if (sobreturno) Dia.fecha(diaInst, horaSobreturno) else elegido?.inicio
    val conQuien = profesional?.let { p -> profesionales.firstOrNull { it.id == p }?.nombre }
        ?: elegido?.profesionales?.firstOrNull()?.let { id -> profesionales.firstOrNull { it.id == id }?.nombre } ?: ""
    val textoGuardar = inicioElegido?.let { i ->
        if (fijo) "Ver las fechas del turno fijo"
        else "Guardar · ${if (esHoy) "hoy ${Formato.horaDe(i)}" else "${Formato.dia(i)} ${Formato.horaDe(i)}"}${if (conQuien.isEmpty()) "" else " con $conQuien"}"
    } ?: "Guardar"

    @Serializable data class Libres(val horarios: List<HorarioLibre> = emptyList())
    @Serializable data class Clientes(val clientes: List<Cliente> = emptyList())
    @Serializable data class Pros(val profesionales: List<Profesional> = emptyList())
    @Serializable data class Servs(val servicios: List<Servicio> = emptyList())

    suspend fun traerLibres() {
        val s = servicio ?: run { libres = emptyList(); return }
        cargandoLibres = true
        val q = buildList { add("servicio" to s); add("fecha" to fecha.toString()); profesional?.let { add("profesional" to it) } }
        libres = estado.api.opcional<Libres>("/v1/panel/libres", q)?.horarios ?: emptyList()
        cargandoLibres = false
    }

    LaunchedEffect(Unit) {
        profesionales = (estado.api.opcional<Pros>("/v1/panel/profesionales")?.profesionales ?: emptyList()).sortedBy { it.orden }
        servicios = estado.api.opcional<Servs>("/v1/panel/servicios")?.servicios ?: emptyList()
        inicio.cliente?.let { cliente = it }
        inicio.inicio?.let { fecha = Formato.enArgentina(it).toLocalDate(); horaSobreturno = Formato.minutosDelDia(it) }
        profesional = inicio.profesionalId
        if (estado.esProfesional && profesional == null) profesional = profesionales.firstOrNull()?.id
        servicio = inicio.servicioId ?: servicios.firstOrNull { s ->
            s.activo && (profesional == null || (profesionales.firstOrNull { it.id == profesional }?.servicios ?: emptyList()).contains(s.id))
        }?.id ?: servicios.firstOrNull { it.activo }?.id
        traerLibres()
        inicio.inicio?.let { i -> elegido = libres.firstOrNull { it.inicio == i } ?: libres.firstOrNull { abs(it.inicio.epochSecond - i.epochSecond) < 60 } }
    }
    var primeraVez by remember { mutableStateOf(true) }
    LaunchedEffect(servicio, profesional, fecha) { if (primeraVez) { primeraVez = false; return@LaunchedEffect }; elegido = null; traerLibres() }
    LaunchedEffect(buscar) {
        val t = buscar.trim()
        if (t.length < 2) { encontrados = emptyList(); return@LaunchedEffect }
        delay(250)
        encontrados = estado.api.opcional<Clientes>("/v1/panel/clientes", listOf("q" to t, "porPagina" to "8"))?.clientes ?: emptyList()
    }

    // Elegir de los Contactos (el selector del sistema: no pide permiso, como CNContactPickerViewController).
    val contactos = rememberLauncherForActivityResult(ActivityResultContracts.StartActivityForResult()) { r ->
        val uri = r.data?.data
        if (r.resultCode != Activity.RESULT_OK || uri == null) return@rememberLauncherForActivityResult
        contexto.contentResolver.query(uri, arrayOf(ContactsContract.CommonDataKinds.Phone.DISPLAY_NAME, ContactsContract.CommonDataKinds.Phone.NUMBER), null, null, null)?.use { c ->
            if (c.moveToFirst()) { cliente = ClienteDelTurno(null, c.getString(0).orEmpty(), c.getString(1).orEmpty(), ""); nueva = false }
        }
    }

    fun fallar(m: String) {
        error = m
        Haptico.error()
        alcance.launch { temblor.snapTo(0f); temblor.animateTo(1f, tween(400)) }
    }

    @Serializable data class Crear(val servicioId: String, val profesionalId: String, val inicio: String, val cliente: ClienteDelTurno, val nota: String, val forzar: Boolean)
    @Serializable data class PedidoPrevia(val servicioId: String, val profesionalId: String, val inicio: String, val cadaSemanas: Int, val cantidad: Int)

    fun guardar() {
        val c = cliente ?: return; val s = servicio ?: return; val i = inicioElegido ?: return
        error = ""
        if (fijo) {
            val pro = profesional ?: elegido?.profesionales?.firstOrNull() ?: run { fallar("Para un turno fijo elegí quién lo atiende."); return }
            guardando = true
            alcance.launch {
                try {
                    val p: PreviaSerie = estado.api.enviar("POST", "/v1/panel/series/previa", PedidoPrevia(s, pro, Fecha.texto(i), cadaSemanas, cantidad))
                    fechasSerie = p.fechas.mapNotNull { f -> if (f.estado == "libre") f.inicio else f.alternativa?.inicio }.toSet()
                    previa = p
                } catch (e: ErrorApi) { fallar(e.detalle) } catch (e: Exception) { fallar("No pudimos armar las fechas.") } finally { guardando = false }
            }
            return
        }
        val pro = profesional ?: elegido?.profesionales?.firstOrNull() ?: ""
        if (sobreturno && pro.isEmpty()) { fallar("Para un sobreturno elegí con quién."); return }
        guardando = true
        alcance.launch {
            try {
                val t: Turno = estado.api.enviar("POST", "/v1/panel/turnos", Crear(s, pro, Fecha.texto(i), c, nota, sobreturno))
                Haptico.exito()
                estado.version += 1
                val nombre = c.nombre.split(" ").firstOrNull() ?: c.nombre
                estado.avisar(Toast("checkmark", "Turno guardado", "Le mandamos la confirmación a $nombre", "Deshacer")) {
                    runCatching { estado.api.hacer("PATCH", "/v1/panel/turnos/${t.id}", AccionesTurno.CambioEstado("cancelado")) }
                }
                Momentos.cargaManual(estado, nav)
                cerrar()
            } catch (e: ErrorApi) { fallar(e.detalle) } catch (e: Exception) { fallar("No pudimos guardarlo. Probá de nuevo.") } finally { guardando = false }
        }
    }

    @Serializable data class T(val inicio: String)
    @Serializable data class PedidoSerie(val servicioId: String, val profesionalId: String, val cadaSemanas: Int, val clienteId: String?, val cliente: ClienteDelTurno?, val nota: String, val turnos: List<T>)
    @Serializable data class Serie(val id: String, val turnos: List<Turno> = emptyList())
    fun guardarSerie() {
        val s = servicio ?: return; val c = cliente ?: return; val pro = profesional ?: elegido?.profesionales?.firstOrNull() ?: return
        alcance.launch {
            try {
                val r: Serie = estado.api.enviar("POST", "/v1/panel/series", PedidoSerie(s, pro, cadaSemanas, c.id, if (c.id == null) c else null, nota, fechasSerie.sorted().map { T(Fecha.texto(it)) }))
                Haptico.exito()
                estado.version += 1
                estado.avisar("Turno fijo guardado", "${r.turnos.size} turnos de ${c.nombre}")
                previa = null
                cerrar()
            } catch (e: Exception) { estado.avisar(e) }
        }
    }

    BarraHoja("Nuevo turno", cerrar)
    Column(Modifier.weight(1f).verticalScroll(rememberScrollState()).padding(start = 16.dp, end = 16.dp, top = 6.dp, bottom = 24.dp), verticalArrangement = Arrangement.spacedBy(18.dp)) {
        Paso("1 · Cliente") {
            val c = cliente
            when {
                c != null -> Row(
                    Modifier.fillMaxWidth().heightIn(min = 60.dp).clip(RoundedCornerShape(16.dp)).background(C.fondo).border(2.dp, C.seleccion, RoundedCornerShape(16.dp))
                        .padding(start = 12.dp, end = 8.dp, top = 8.dp, bottom = 8.dp),
                    verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp),
                ) {
                    Iniciales(c.nombre, 40.dp)
                    Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(1.dp)) {
                        Texto(c.nombre, Tipo.lato(16f, Peso.Bold), C.tinta)
                        Texto(listOf(c.telefono, if (c.id == null && c.telefono.isNotEmpty()) "nueva" else "").filter { it.isNotEmpty() }.joinToString(" · "), Tipo.lato(13f, numeros = true), C.sub)
                    }
                    Box(Modifier.size(44.dp).clip(CircleShape).clickable { cliente = null }, contentAlignment = Alignment.Center) { Icono("xmark.circle.fill", 22f, C.sub, descripcion = "Cambiar cliente") }
                }
                nueva -> Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Campo("Nombre", nombreNuevo, { nombreNuevo = it })
                    Campo("Teléfono", telefonoNuevo, { telefonoNuevo = it }, teclado = KeyboardType.Phone)
                    Campo("Email (opcional)", emailNuevo, { emailNuevo = it }, teclado = KeyboardType.Email)
                    BotonContorno("Usar estos datos", { cliente = ClienteDelTurno(null, nombreNuevo.trim(), telefonoNuevo, emailNuevo) }, alto = 48.dp, tam = 16f, habilitado = nombreNuevo.trim().length >= 2)
                }
                else -> Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Row(Modifier.fillMaxWidth().height(48.dp).clip(RoundedCornerShape(14.dp)).background(C.campo).padding(horizontal = 14.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Icono("magnifyingglass", 18f, C.sub)
                        Box(Modifier.weight(1f)) {
                            if (buscar.isEmpty()) Texto("Buscá por nombre o teléfono", Tipo.lato(16f), C.sub.copy(alpha = 0.7f))
                            val tinta = C.tinta
                            BasicTextField(buscar, { buscar = it }, Modifier.fillMaxWidth(), singleLine = true, textStyle = Tipo.lato(16f).copy(color = tinta), cursorBrush = SolidColor(C.mentaTexto))
                        }
                    }
                    encontrados.take(5).forEach { e ->
                        Row(Modifier.fillMaxWidth().clickable { cliente = ClienteDelTurno(e.id, e.nombre, e.telefono, e.email) }.padding(vertical = 6.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                            Iniciales(e.nombre, 36.dp)
                            Column(Modifier.weight(1f)) {
                                Texto(e.nombre, Tipo.lato(15f, Peso.Bold), C.tinta)
                                Texto("${e.telefono} · ${e.visitas} ${if (e.visitas == 1) "visita" else "visitas"}", Tipo.lato(12f), C.sub)
                            }
                            if (e.penalizado) PastillaEstado("Penalidad", "exclamationmark.shield", C.chip, C.tinta)
                        }
                    }
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Box(Modifier.weight(1f)) { BotonContorno("De tus Contactos", {
                            // Sin app de Contactos el selector no abre: se avisa en vez de cerrarse.
                            try { contactos.launch(Intent(Intent.ACTION_PICK, ContactsContract.CommonDataKinds.Phone.CONTENT_URI)) }
                            catch (e: ActivityNotFoundException) { estado.avisar("Este equipo no tiene Contactos", ok = false) }
                        }, alto = 44.dp, tam = 15f, simbolo = "person.crop.circle") }
                        Box(Modifier.weight(1f)) { BotonContorno("Clienta nueva", { nueva = true; nombreNuevo = buscar }, alto = 44.dp, tam = 15f, simbolo = "person.badge.plus") }
                    }
                }
            }
        }
        Paso("2 · Servicio") {
            servicios.filter { it.activo }.chunked(2).forEach { fila ->
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    fila.forEach { s ->
                        val on = servicio == s.id
                        val forma = RoundedCornerShape(16.dp)
                        Column(
                            Modifier.weight(1f).heightIn(min = 64.dp).clip(forma).background(if (on) C.seleccion else C.fondo).then(if (on) Modifier else Modifier.border(1.5.dp, C.linea, forma))
                                .clickable { servicio = s.id }.padding(horizontal = 12.dp, vertical = 10.dp),
                            verticalArrangement = Arrangement.spacedBy(2.dp, Alignment.CenterVertically),
                        ) {
                            val tinta = if (on) C.seleccionTexto else C.tinta
                            Texto(s.nombre, Tipo.lato(15f, Peso.Bold), tinta, lineas = 2)
                            Texto(if (estado.vePrecios) "${Formato.duracion(s.duracion)} · ${Formato.pesos(s.precio)}" else Formato.duracion(s.duracion), Tipo.lato(13f, numeros = true), tinta.copy(alpha = 0.8f))
                        }
                    }
                    if (fila.size == 1) Spacer(Modifier.weight(1f))
                }
            }
        }
        Paso("3 · Con quién") {
            Row(Modifier.horizontalScroll(rememberScrollState()), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                (listOf<Pair<String?, String>>(null to "Cualquiera") + pros.map { it.id to it.nombre }).forEach { (id, t) ->
                    val on = profesional == id
                    Box(
                        Modifier.then(if (id == null) Modifier.widthIn(min = 110.dp) else Modifier).heightIn(min = 44.dp).clip(CircleShape).background(if (on) C.seleccion else C.fondo)
                            .then(if (on) Modifier else Modifier.border(1.5.dp, C.linea, CircleShape)).clickable { profesional = id }.padding(horizontal = 16.dp),
                        contentAlignment = Alignment.Center,
                    ) { Texto(t, Tipo.lato(14f, Peso.Bold), if (on) C.seleccionTexto else C.tinta) }
                }
            }
        }
        Paso(if (esHoy) "4 · Cuándo · libres hoy" else "4 · Cuándo · libres el ${Formato.dia(diaInst).uppercase()}") {
            if (!sobreturno) {
                when {
                    cargandoLibres -> Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) { repeat(3) { Box(Modifier.weight(1f)) { Esqueleto(60.dp, 16.dp) } } }
                    libres.isEmpty() -> Texto(if (servicio == null) "Elegí un servicio para ver los horarios libres." else "No hay horarios libres ese día. Probá otro día o cargalo como sobreturno.", Tipo.lato(14f), C.sub)
                    else -> libres.chunked(3).forEach { fila ->
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            fila.forEach { h ->
                                val on = elegido?.inicio == h.inicio
                                val quien = h.profesionales.firstOrNull()?.let { id -> profesionales.firstOrNull { it.id == id }?.nombre } ?: ""
                                val forma = RoundedCornerShape(16.dp)
                                Box(Modifier.weight(1f).heightIn(min = 60.dp).clip(forma).then(if (on) Modifier else Modifier.border(1.5.dp, C.huecoLinea, forma)).clickable { elegido = h; Haptico.seleccion() }, contentAlignment = Alignment.Center) {
                                    if (on) Box(Modifier.matchParentSize().background(C.seleccion)) else Rayas(Modifier.matchParentSize(), quieto = true)
                                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                                        Texto(h.hora, Tipo.lato(18f, Peso.Black, numeros = true), if (on) C.seleccionTexto else C.huecoTinta)
                                        Texto(if (profesional == null && h.profesionales.size > 1) "${h.profesionales.size} libres" else quien, Tipo.lato(12f, Peso.Bold), if (on) C.seleccionTexto else C.huecoTinta, lineas = 1)
                                    }
                                }
                            }
                            repeat(3 - fila.size) { Spacer(Modifier.weight(1f)) }
                        }
                    }
                }
            }
            Column {
                Row(Modifier.fillMaxWidth().heightIn(min = 52.dp).clickable { otroDia = !otroDia }.padding(horizontal = 4.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    Icono("calendar", 20f, C.tinta)
                    Texto(if (esHoy) "Otro día" else Formato.fechaLarga(diaInst), Tipo.lato(16f), C.tinta, Modifier.weight(1f))
                    Icono(if (otroDia) "chevron.up" else "chevron.right", 15f, C.sub)
                }
                if (otroDia) CalendarioMes(fecha, { fecha = it }, minimo = hoy)
                Box(Modifier.fillMaxWidth().height(1.dp).background(C.linea))
                Box(Modifier.padding(horizontal = 4.dp)) { Interruptor("Sobreturno", sobreturno, { sobreturno = it }, "Encimarlo aunque el horario esté ocupado", "square.stack.3d.up") }
                if (sobreturno) {
                    Row(Modifier.fillMaxWidth().heightIn(min = 52.dp).padding(horizontal = 4.dp), verticalAlignment = Alignment.CenterVertically) {
                        Texto("Hora", Tipo.lato(16f), C.tinta, Modifier.weight(1f))
                        SelectorHora(horaSobreturno, { horaSobreturno = it })
                    }
                    if (profesional == null) Texto("Para un sobreturno elegí con quién.", Tipo.lato(13f), C.sub, Modifier.padding(horizontal = 4.dp))
                }
            }
        }
        Paso("5 · Se repite") {
            Interruptor("Turno fijo", fijo, { fijo = it }, "La misma hora cada semana o cada tantas", "repeat")
            if (fijo) {
                Paso2("Cada $cadaSemanas ${if (cadaSemanas == 1) "semana" else "semanas"}", cadaSemanas, 1..8) { cadaSemanas = it }
                Paso2("$cantidad turnos", cantidad, 2..52) { cantidad = it }
            }
        }
        Paso("Nota interna") { Campo("", nota, { nota = it }, ayuda = "Sólo la ve el equipo (ej.: piel sensible)", multilinea = true) }
        if (error.isNotEmpty()) {
            Row(Modifier.graphicsLayer { translationX = 3.dp.toPx() * sin(temblor.value * PI.toFloat() * 4) }, horizontalArrangement = Arrangement.spacedBy(6.dp), verticalAlignment = Alignment.CenterVertically) {
                Icono("exclamationmark.circle", 14f, C.error)
                Texto(error, Tipo.lato(14f, Peso.Bold), C.error)
            }
        }
    }
    Column(Modifier.fillMaxWidth().background(C.hoja)) {
        Box(Modifier.fillMaxWidth().height(1.dp).background(C.linea))
        Box(Modifier.padding(start = 16.dp, end = 16.dp, top = 12.dp, bottom = 8.dp)) {
            BotonMenta(if (guardando) "Guardando…" else textoGuardar, { guardar() }, habilitado = !guardando && cliente != null && servicio != null && inicioElegido != null && !estado.soloLectura)
        }
    }
    Hoja(previa != null, { previa = null }) {
        previa?.let { p -> PreviaFijo(p, fechasSerie, { f -> fechasSerie = if (f in fechasSerie) fechasSerie - f else fechasSerie + f }, reservar = { guardarSerie() }, cerrar = { previa = null }) }
    }
}

@Composable
private fun Paso(titulo: String, contenido: @Composable ColumnScope.() -> Unit) {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) { TextoRotulo(titulo); contenido() }
}

/** Un Stepper de iOS: el texto y − / +. */
@Composable
private fun Paso2(texto: String, valor: Int, rango: IntRange, alCambiar: (Int) -> Unit) {
    Row(Modifier.fillMaxWidth().heightIn(min = 48.dp), verticalAlignment = Alignment.CenterVertically) {
        Texto(texto, Tipo.lato(16f), C.tinta, Modifier.weight(1f))
        Row(Modifier.clip(RoundedCornerShape(9.dp)).background(C.chip)) {
            Box(Modifier.size(48.dp, 32.dp).clickable(enabled = valor > rango.first) { alCambiar(valor - 1) }.alpha(if (valor > rango.first) 1f else 0.35f), contentAlignment = Alignment.Center) { Icono("minus", 15f) }
            Box(Modifier.width(1.dp).height(32.dp).padding(vertical = 6.dp).background(C.linea))
            Box(Modifier.size(48.dp, 32.dp).clickable(enabled = valor < rango.last) { alCambiar(valor + 1) }.alpha(if (valor < rango.last) 1f else 0.35f), contentAlignment = Alignment.Center) { Icono("plus", 15f) }
        }
    }
}

@Composable
fun ColumnScope.PreviaFijo(previa: PreviaSerie, elegidas: Set<Instant>, alTocar: (Instant) -> Unit, reservar: () -> Unit, cerrar: () -> Unit) {
    BarraHoja("Turno fijo", cerrar, textoCancelar = "Volver")
    Column(Modifier.weight(1f).background(C.fondo).verticalScroll(rememberScrollState()).padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Texto("${previa.servicio} con ${previa.profesional}, cada ${previa.cadaSemanas} ${if (previa.cadaSemanas == 1) "semana" else "semanas"}. ${previa.libres} libres, ${previa.chocan} chocan.", Tipo.lato(15f), C.sub)
        Column(Modifier.fillMaxWidth().tarjeta()) {
            previa.fechas.forEachIndexed { i, f ->
                if (i > 0) Box(Modifier.padding(start = 16.dp).fillMaxWidth().height(1.dp).background(C.linea))
                val fecha = if (f.estado == "libre") f.inicio else f.alternativa?.inicio
                Row(Modifier.fillMaxWidth().clickable(enabled = fecha != null) { fecha?.let(alTocar) }.padding(horizontal = 16.dp, vertical = 10.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    Icono(if (fecha != null && fecha in elegidas) "checkmark.circle.fill" else "circle", 22f, if (fecha == null) C.linea else C.mentaTexto)
                    Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                        Texto(Formato.diaYHora(f.inicio), Tipo.lato(15f, Peso.Bold), C.tinta)
                        if (f.estado == "choca") {
                            val m = mapOf("ocupado" to "Ocupado a esa hora", "bloqueado" to "Está bloqueado ese día", "no_atiende" to "No atiende ese día", "pasado" to "Ya pasó")[f.motivo ?: ""] ?: "Choca"
                            Texto(f.alternativa?.let { "$m · podés a las ${it.hora}" } ?: m, Tipo.lato(13f), C.ausenteTinta)
                        }
                    }
                }
            }
        }
    }
    Box(Modifier.fillMaxWidth().background(C.hoja).padding(16.dp)) {
        BotonMenta("Reservar ${elegidas.size} ${if (elegidas.size == 1) "turno" else "turnos"}", reservar, habilitado = elegidas.isNotEmpty())
    }
}
