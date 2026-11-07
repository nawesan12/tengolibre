package ar.com.tengolibre.app.pantallas.clientes

import android.app.Activity
import android.content.Intent
import android.provider.ContactsContract
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.nucleo.*
import ar.com.tengolibre.app.pantallas.hoy.BarraHoja
import ar.com.tengolibre.app.pantallas.hoy.MenuContextual
import ar.com.tengolibre.app.pantallas.hoy.OpcionMenu
import ar.com.tengolibre.app.pantallas.hoy.SugerenciaMembresia
import ar.com.tengolibre.app.pantallas.hoy.TalonShape
import ar.com.tengolibre.app.pantallas.hoy.Troquel
import ar.com.tengolibre.app.pantallas.mas.PlanMembresia
import ar.com.tengolibre.app.pantallas.mas.Socio
import ar.com.tengolibre.app.pantallas.principal.*
import ar.com.tengolibre.app.tema.*
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable
import java.time.Instant
import java.time.YearMonth
import java.time.temporal.ChronoUnit
import kotlin.math.roundToInt

// Clon de Pantallas/Clientes/Clientes.swift. (PlanMembresia y Socio viven en mas/MasPlata.kt.)

@Serializable data class EtiquetaConUso(val id: String, val nombre: String = "", val color: String = "", val clientes: Int = 0)

/** Lo que se apila dentro de la pestaña Clientes. */
sealed class DestinoClientes { data class ficha(val id: String) : DestinoClientes() }

/** Elegir de los Contactos con el selector del sistema (no pide permiso): devuelve nombre y teléfono. */
@Composable
fun rememberContactos(alElegir: (String, String, String) -> Unit): () -> Unit {
    val contexto = LocalContext.current
    val l = rememberLauncherForActivityResult(ActivityResultContracts.StartActivityForResult()) { r ->
        val uri = r.data?.data
        if (r.resultCode != Activity.RESULT_OK || uri == null) return@rememberLauncherForActivityResult
        contexto.contentResolver.query(uri, arrayOf(ContactsContract.CommonDataKinds.Phone.DISPLAY_NAME, ContactsContract.CommonDataKinds.Phone.NUMBER), null, null, null)?.use { c ->
            if (c.moveToFirst()) alElegir(c.getString(0).orEmpty(), c.getString(1).orEmpty(), "")
        }
    }
    return { l.launch(Intent(Intent.ACTION_PICK, ContactsContract.CommonDataKinds.Phone.CONTENT_URI)) }
}

/** 14 Clientes: buscar, filtros (Todas · Frecuentes · Con penalidad · Nuevas · por etiqueta), orden y la lista con
 *  iniciales, última visita y la etiqueta. «+» para cargar una clienta nueva. */
@Composable
fun Clientes() {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    var clientes by remember { mutableStateOf<List<Cliente>>(emptyList()) }
    var total by remember { mutableIntStateOf(0) }
    var pagina by remember { mutableIntStateOf(1) }
    var etiquetas by remember { mutableStateOf<List<EtiquetaConUso>>(emptyList()) }
    var buscar by remember { mutableStateOf("") }
    var filtro by remember { mutableStateOf("todas") }
    var orden by remember { mutableStateOf("nombre") }
    var cargando by remember { mutableStateOf(true) }
    var nueva by remember { mutableStateOf(false) }

    @Serializable data class Lista(val clientes: List<Cliente> = emptyList(), val total: Int = 0)
    @Serializable data class Etiquetas(val etiquetas: List<EtiquetaConUso> = emptyList())
    suspend fun cargar(mas: Boolean = false) {
        val p = if (mas) pagina + 1 else 1
        val q = mutableListOf("pagina" to "$p", "porPagina" to "30")
        when (filtro) {
            "frecuentes" -> q.add("orden" to "visitas")
            "nuevas" -> q.add("orden" to "reciente")
            "penalizados" -> q.add("penalizados" to "1")
            else -> if (filtro.startsWith("e:")) q.add("etiqueta" to filtro.drop(2))
        }
        if (filtro != "frecuentes" && filtro != "nuevas") q.add("orden" to orden)
        buscar.trim().takeIf { it.isNotEmpty() }?.let { q.add("q" to it) }
        try {
            val r: Lista = estado.api.get("/v1/panel/clientes", q)
            clientes = if (mas) clientes + r.clientes else r.clientes
            total = r.total
            pagina = p
        } catch (e: Exception) { if (!mas) estado.avisar(e) }
        cargando = false
        if (etiquetas.isEmpty()) etiquetas = estado.api.opcional<Etiquetas>("/v1/panel/etiquetas")?.etiquetas ?: emptyList()
    }
    LaunchedEffect(estado.version, filtro, orden) { cargar() }
    LaunchedEffect(buscar) { delay(300); cargar() }

    val filtros = listOf("todas" to "Todas", "frecuentes" to "Frecuentes", "penalizados" to "Con penalidad", "nuevas" to "Nuevas") + etiquetas.map { "e:${it.id}" to it.nombre }

    Refrescable({ cargar() }, Modifier.fillMaxSize().background(C.fondo)) { Column(Modifier.fillMaxSize().background(C.fondo).verticalScroll(rememberScrollState())) {
        Cabecera("Clientes", "${Formato.numero(total)} en total", acciones = listOf(AccionCabecera("plus", "Clienta nueva") { nueva = true })) {
            Row(Modifier.fillMaxWidth().height(48.dp).clip(RoundedCornerShape(14.dp)).background(Color.White.copy(alpha = 0.12f)).padding(horizontal = 14.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Icono("magnifyingglass", 18f, Color.White.copy(alpha = 0.7f))
                Box(Modifier.weight(1f)) {
                    if (buscar.isEmpty()) Texto("Buscá por nombre o teléfono", Tipo.lato(16f), Color.White.copy(alpha = 0.6f))
                    BasicTextField(buscar, { buscar = it }, Modifier.fillMaxWidth(), singleLine = true, textStyle = Tipo.lato(16f).copy(color = Color.White), cursorBrush = SolidColor(C.menta))
                }
            }
            Row(Modifier.sangrado().horizontalScroll(rememberScrollState()).padding(horizontal = 16.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                filtros.forEach { (k, t) ->
                    val on = filtro == k
                    Box(Modifier.height(36.dp).clip(CircleShape).background(if (on) C.menta else Color.White.copy(alpha = 0.1f)).clickable { filtro = k }.padding(horizontal = 14.dp), contentAlignment = Alignment.Center) {
                        Texto(t, Tipo.lato(14f, Peso.Bold), if (on) C.petroleoHondo else Color.White)
                    }
                }
            }
        }
        Cuerpo {
            Row(verticalAlignment = Alignment.CenterVertically) {
                TextoRotulo("${Formato.numero(total)} ${if (total == 1) "clienta" else "clientas"}", modifier = Modifier.weight(1f))
                val ordenes = listOf("nombre" to "Por nombre", "reciente" to "Más recientes", "ultima" to "Última visita", "gastado" to "Más gastaron", "visitas" to "Más visitas")
                MenuBoton(ordenes.map { (k, t) -> OpcionDeMenu(t, if (orden == k) "checkmark" else "circle") { orden = k } }, "Ordenar") {
                    Row(Modifier.heightIn(min = 44.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                        Icono("arrow.up.arrow.down", 14f, C.mentaTexto); Texto("Ordenar", Tipo.lato(14f, Peso.Bold), C.mentaTexto)
                    }
                }
            }
            if (cargando && clientes.isEmpty()) repeat(6) { Esqueleto(68.dp) }
            else if (clientes.isEmpty()) VacioQueEnsena("Sin resultados", if (buscar.isEmpty()) "Todavía no tenés clientas" else "No encontramos a nadie",
                if (buscar.isEmpty()) "Se suman solas cuando reservan por tu página o les cargás un turno." else "Probá con otra parte del nombre o del teléfono.") {
                BotonContorno("Cargar una clienta", { nueva = true }, simbolo = "plus")
            }
            else Column(Modifier.fillMaxWidth().tarjeta()) {
                clientes.forEachIndexed { i, c ->
                    if (i > 0) Box(Modifier.padding(start = 72.dp).fillMaxWidth().height(1.dp).background(C.linea))
                    if (c.id == clientes.last().id && clientes.size < total) LaunchedEffect(c.id) { cargar(mas = true) }
                    Row(
                        Modifier.entrada(i).fillMaxWidth().heightIn(min = 68.dp).clickable { nav.pila.add(DestinoClientes.ficha(c.id)) }.padding(horizontal = 14.dp),
                        verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp),
                    ) {
                        Iniciales(c.nombre, 44.dp)
                        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                            Texto(c.nombre, Tipo.lato(16f, Peso.Bold), C.tinta, lineas = 1)
                            Texto(detalle(c), Tipo.lato(13f), C.sub, lineas = 1)
                        }
                        if (c.penalizado) PastillaEstado("Penalidad", "exclamationmark.shield", C.ausenteFondo, C.ausenteTinta)
                        else c.etiquetas.firstOrNull()?.let { e ->
                            Box(Modifier.height(24.dp).clip(CircleShape).background(hexTexto(e.color)?.copy(alpha = 0.22f) ?: C.chip).padding(horizontal = 10.dp), contentAlignment = Alignment.Center) {
                                Texto(e.nombre, Tipo.lato(12f, Peso.Bold), C.tinta)
                            }
                        }
                        Icono("chevron.right", 13f, C.sub)
                    }
                }
            }
        }
    } }
    Hoja(nueva, { nueva = false }, listOf(Altura.media)) { ClientaNueva { nueva = false } }
}

private fun detalle(c: Cliente): String {
    val p = mutableListOf<String>()
    val pr = c.proximoTurno; val u = c.ultimaVisita
    if (pr != null && pr.isAfter(Instant.now())) p.add("Viene ${Formato.cuando(pr)}") else if (u != null) p.add("Vino ${Formato.hace(u)}") else p.add("Nueva")
    p.add(Formato.cantidad(c.visitas, "visita", "visitas"))
    return p.joinToString(" · ")
}

/** Cargar una clienta a mano (nombre, teléfono, email). */
@Composable
fun ColumnScope.ClientaNueva(cerrar: () -> Unit) {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    val alcance = rememberCoroutineScope()
    var nombre by remember { mutableStateOf("") }
    var telefono by remember { mutableStateOf("") }
    var email by remember { mutableStateOf("") }
    val contactos = rememberContactos { n, t, e -> nombre = n; telefono = t; email = e }
    @Serializable data class Nueva(val nombre: String, val telefono: String, val email: String)
    BarraHoja("Clienta nueva", cerrar, accion = "Guardar", habilitada = nombre.trim().length >= 2) {
        alcance.launch {
            try {
                val c: Cliente = estado.api.enviar("POST", "/v1/panel/clientes", Nueva(nombre, telefono, email))
                estado.version += 1
                cerrar()
                nav.pilaClientes.add(DestinoClientes.ficha(c.id))
            } catch (e: Exception) { estado.avisar(e) }
        }
    }
    Column(Modifier.weight(1f).padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Campo("Nombre", nombre, { nombre = it })
        Campo("Teléfono", telefono, { telefono = it }, teclado = KeyboardType.Phone)
        Campo("Email (opcional)", email, { email = it }, teclado = KeyboardType.Email)
        BotonTexto("Traer de Contactos", contactos)
    }
}

/** 15 Ficha de la clienta: Llamar · WhatsApp · Turno, próximo turno, etiquetas, penalidad, membresía, notas del
 *  equipo e historial. */
@OptIn(ExperimentalFoundationApi::class, ExperimentalLayoutApi::class)
@Composable
fun FichaCliente(id: String) {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    val alcance = rememberCoroutineScope()
    var ficha by remember { mutableStateOf<Ficha?>(null) }
    var notas by remember { mutableStateOf<List<Nota>>(emptyList()) }
    var etiquetas by remember { mutableStateOf<List<EtiquetaConUso>>(emptyList()) }
    var socio by remember { mutableStateOf<Socio?>(null) }
    var notaNueva by remember { mutableStateOf("") }
    var editar by remember { mutableStateOf(false) }
    var penalizar by remember { mutableStateOf(false) }
    var diasPenalidad by remember { mutableIntStateOf(30) }
    var notaPenalidad by remember { mutableStateOf("") }
    var error by remember { mutableStateOf("") }
    var menuNota by remember { mutableStateOf<String?>(null) }

    @Serializable data class Notas(val notas: List<Nota> = emptyList())
    @Serializable data class Etiquetas(val etiquetas: List<EtiquetaConUso> = emptyList())
    @Serializable data class Socios(val socios: List<Socio> = emptyList())
    suspend fun cargar() {
        try {
            val f: Ficha = estado.api.get("/v1/panel/clientes/$id")
            notas = estado.api.opcional<Notas>("/v1/panel/clientes/$id/notas")?.notas ?: emptyList()
            etiquetas = estado.api.opcional<Etiquetas>("/v1/panel/etiquetas")?.etiquetas ?: emptyList()
            socio = estado.api.opcional<Socios>("/v1/panel/membresias/socios", listOf("estado" to "activa"))?.socios?.firstOrNull { it.clienteId == id }
            ficha = f
            error = ""
        } catch (e: ErrorApi) { error = e.detalle } catch (e: Exception) { error = "Probá de nuevo." }
    }
    LaunchedEffect(estado.version) { cargar() }

    fun etiqueta(eid: String, quitar: Boolean) = alcance.launch {
        try {
            if (quitar) estado.api.hacer("DELETE", "/v1/panel/clientes/$id/etiquetas/$eid") else estado.api.hacer("PUT", "/v1/panel/clientes/$id/etiquetas/$eid", emptyMap<String, String>())
            Haptico.seleccion(); cargar()
        } catch (e: Exception) { estado.avisar(e) }
    }
    @Serializable data class TextoNota(val texto: String)
    fun sumarNota() = alcance.launch { try { estado.api.hacer("POST", "/v1/panel/clientes/$id/notas", TextoNota(notaNueva)); notaNueva = ""; cargar() } catch (e: Exception) { estado.avisar(e) } }
    fun borrarNota(n: Nota) = alcance.launch { try { estado.api.hacer("DELETE", "/v1/panel/clientes/$id/notas/${n.id}"); notas = notas.filter { it.id != n.id } } catch (e: Exception) { estado.avisar(e) } }
    @Serializable data class Pena(val dias: Int, val nota: String)
    fun penalizarYa() = alcance.launch { try { estado.api.hacer("POST", "/v1/panel/clientes/$id/penalidades", Pena(diasPenalidad, notaPenalidad)); penalizar = false; Haptico.exito(); cargar() } catch (e: Exception) { estado.avisar(e) } }
    fun levantar(p: Penalidad) = alcance.launch { try { estado.api.hacer("POST", "/v1/panel/penalidades/${p.id}/levantar"); cargar() } catch (e: Exception) { estado.avisar(e) } }

    val f = ficha
    val subtitulo = f?.let { buildList {
        add(Formato.cantidad(it.visitas, "visita", "visitas"))
        if (estado.vePrecios && it.gastado > 0) add("${Formato.pesos(it.gastado)} gastado")
        if (it.ausencias > 0) add(Formato.cantidad(it.ausencias, "ausencia", "ausencias"))
    }.joinToString(" · ") } ?: ""
    val tel = f?.telefono.orEmpty().filter { it in "+0123456789" }

    Refrescable({ cargar() }, Modifier.fillMaxSize().background(C.fondo)) { Column(Modifier.fillMaxSize().background(C.fondo).verticalScroll(rememberScrollState())) {
        Cabecera(f?.nombre ?: "Clienta", subtitulo, volver = "Clientes", acciones = if (f == null) emptyList() else listOf(AccionCabecera("pencil", "Editar") { editar = true }))
        Cuerpo {
            if (f == null) {
                if (error.isNotEmpty()) ErrorConSalida("No pudimos traer la ficha", error) { alcance.launch { cargar() } } else repeat(4) { Esqueleto(90.dp) }
                return@Cuerpo
            }
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                BotonContacto("Llamar", "phone", false, tel.isNotEmpty(), Modifier.weight(1f)) { estado.abrirLink("tel:$tel") }
                BotonContacto("WhatsApp", "message", false, tel.isNotEmpty(), Modifier.weight(1f)) {
                    var d = tel.filter { it.isDigit() }; if (d.startsWith("0")) d = d.drop(1); if (!d.startsWith("54")) d = "549$d"
                    estado.abrirLink("https://wa.me/$d")
                }
                BotonContacto("Turno", "plus", true, !estado.soloLectura && !f.penalizado, Modifier.weight(1f)) {
                    nav.nuevo = NuevoTurnoInicio(cliente = ClienteDelTurno(f.id, f.nombre, f.telefono, f.email))
                }
            }
            f.penalidad?.takeIf { it.vigente }?.let { p ->
                Row(Modifier.fillMaxWidth().clip(RoundedCornerShape(18.dp)).background(C.ausenteFondo).padding(16.dp), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    Icono("exclamationmark.shield", 22f, C.ausenteTinta)
                    Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                        Texto("Penalizada hasta el ${Formato.dia(p.hasta)}", Tipo.lato(16f, Peso.Bold), hex(0x0F2328))
                        Texto(p.nota.ifEmpty { "No puede reservar por la web hasta esa fecha." }, Tipo.lato(14f), hex(0x4A5D61))
                        BotonTexto("Levantar la penalidad", { levantar(p) })
                    }
                }
            }
            socio?.let { Membresia(it) }
            SugerenciaMembresia(f, socio != null)
            Box(Modifier.entrada(1)) { Visitas(f, estado.vePrecios) }
            f.turnos.firstOrNull { it.inicio.isAfter(Instant.now()) && it.estado != "cancelado" }?.let { pr ->
                Rotulo("Próximo turno")
                Box(Modifier.fillMaxWidth().tarjeta()) { FilaTurno(pr, estado.vePrecios) { alcance.launch { estado.api.opcional<Turno>("/v1/panel/turnos/${pr.id}")?.let { nav.turno = it } } } }
            }
            // Etiquetas
            Column(Modifier.fillMaxWidth().tarjeta().padding(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                TextoRotulo("Etiquetas")
                VistaConsejo(Consejos.etiquetas)
                FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    etiquetas.forEach { e ->
                        val puesta = f.etiquetas.any { it.id == e.id }
                        Box(Modifier.clickable { etiqueta(e.id, puesta) }) { Chip(e.nombre, if (puesta) "checkmark" else "plus", puesta) }
                    }
                }
                if (etiquetas.isEmpty()) Texto("Creá etiquetas desde Más › Etiquetas.", Tipo.lato(13f), C.sub)
            }
            if (f.penalidad?.vigente != true && !estado.esProfesional) Box(Modifier.fillMaxWidth().tarjeta().clickable { penalizar = true }) { FilaLista("exclamationmark.shield", "Penalizar", "No va a poder reservar por la web") }
            // Notas
            Column(Modifier.fillMaxWidth().tarjeta().padding(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                TextoRotulo("Notas del equipo")
                notas.forEach { n ->
                    Box {
                        Column(Modifier.fillMaxWidth().combinedClickable(onClick = {}, onLongClick = { menuNota = n.id }), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                            Texto(n.texto, Tipo.lato(15f), C.tinta)
                            Texto("${n.autor} · ${Formato.hace(n.creadoEn)}", Tipo.lato(12f), C.sub)
                        }
                        MenuContextual(menuNota == n.id, listOf(OpcionMenu("Borrar", "trash", destructiva = true) { borrarNota(n) })) { menuNota = null }
                    }
                }
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Box(Modifier.weight(1f).clip(RoundedCornerShape(12.dp)).background(C.campo).padding(horizontal = 12.dp, vertical = 10.dp)) {
                        if (notaNueva.isEmpty()) Texto("Sumá una nota (ej.: prefiere a la tarde)", Tipo.lato(15f), C.sub.copy(alpha = 0.7f))
                        val tinta = C.tinta
                        BasicTextField(notaNueva, { notaNueva = it }, Modifier.fillMaxWidth(), textStyle = Tipo.lato(15f).copy(color = tinta), maxLines = 4, cursorBrush = SolidColor(C.mentaTexto))
                    }
                    Box(Modifier.atenuado(notaNueva.isBlank())) { BotonMentaChico("Sumar", { if (notaNueva.isNotBlank()) sumarNota() }) }
                }
            }
            // Historial
            val pasados = f.turnos.filter { !it.inicio.isAfter(Instant.now()) || it.estado == "cancelado" }
            if (pasados.isNotEmpty()) {
                Rotulo("Historial")
                Column(Modifier.fillMaxWidth().tarjeta()) {
                    pasados.forEachIndexed { i, t ->
                        if (i > 0) Box(Modifier.fillMaxWidth().height(1.dp).background(C.linea))
                        FilaTurno(t, estado.vePrecios) { alcance.launch { estado.api.opcional<Turno>("/v1/panel/turnos/${t.id}")?.let { nav.turno = it } } }
                    }
                }
            }
            Column(Modifier.padding(horizontal = 4.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                if (f.email.isNotEmpty()) Row(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalAlignment = Alignment.CenterVertically) { Icono("envelope", 14f, C.sub); Texto(f.email, Tipo.lato(14f), C.sub) }
                Row(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalAlignment = Alignment.CenterVertically) { Icono("calendar", 14f, C.sub); Texto("Clienta desde el ${Formato.dia(f.creadoEn)}", Tipo.lato(14f), C.sub) }
            }
        }
    } }
    Hoja(editar, { editar = false }, listOf(Altura.media)) { ficha?.let { EditarClienta(it, listo = { alcance.launch { cargar() } }, cerrar = { editar = false }) } }
    Hoja(penalizar, { penalizar = false }, listOf(Altura.media)) {
        BarraHoja("Penalizar", { penalizar = false }, accion = "Penalizar") { penalizarYa() }
        Column(Modifier.background(C.fondo).fillMaxSize().padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Column(Modifier.fillMaxWidth().tarjeta()) {
                Row(Modifier.fillMaxWidth().heightIn(min = 52.dp).padding(horizontal = 16.dp), verticalAlignment = Alignment.CenterVertically) {
                    Texto("$diasPenalidad días", Tipo.lato(16f), C.tinta, Modifier.weight(1f))
                    val paso = if (diasPenalidad >= 30) 15 else 5
                    Row(Modifier.clip(RoundedCornerShape(9.dp)).background(C.chip)) {
                        Box(Modifier.size(48.dp, 32.dp).clickable { diasPenalidad = maxOf(1, diasPenalidad - paso) }, contentAlignment = Alignment.Center) { Icono("minus", 15f) }
                        Box(Modifier.size(48.dp, 32.dp).clickable { diasPenalidad = minOf(365, diasPenalidad + paso) }, contentAlignment = Alignment.Center) { Icono("plus", 15f) }
                    }
                }
                Box(Modifier.padding(start = 16.dp).fillMaxWidth().height(1.dp).background(C.linea))
                Box(Modifier.padding(16.dp)) {
                    if (notaPenalidad.isEmpty()) Texto("Motivo (opcional)", Tipo.lato(16f), C.sub.copy(alpha = 0.7f))
                    val tinta = C.tinta
                    BasicTextField(notaPenalidad, { notaPenalidad = it }, Modifier.fillMaxWidth(), textStyle = Tipo.lato(16f).copy(color = tinta), cursorBrush = SolidColor(C.mentaTexto))
                }
            }
            Texto("Mientras dure, no puede reservar por tu página. Vos le podés seguir cargando turnos.", Tipo.lato(13f), C.sub)
        }
    }
}

/** Las visitas de los últimos 12 meses (atendidas), con lo que gastó y cada cuánto viene. */
@Composable
private fun Visitas(f: Ficha, vePrecios: Boolean) {
    val atendidos = f.turnos.filter { it.estado == "atendido" }
    if (atendidos.isEmpty()) return
    val hoy = YearMonth.from(Formato.enArgentina(Instant.now()))
    val meses = (11 downTo 0).map { hoy.minusMonths(it.toLong()) }.map { m ->
        MesVisitas(m.atDay(1).atStartOfDay(Formato.zona).toInstant(), atendidos.count { YearMonth.from(Formato.enArgentina(it.inicio)) == m })
    }
    val fechas = atendidos.map { it.inicio }.sorted()
    val cada = if (fechas.size > 1) (ChronoUnit.DAYS.between(fechas.first(), fechas.last()).toDouble() / (fechas.size - 1)).roundToInt() else null
    Column(Modifier.fillMaxWidth().tarjeta().padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            TextoRotulo("Visitas · último año", modifier = Modifier.weight(1f))
            cada?.let { Texto("viene cada $it días", Tipo.lato(12f, Peso.Bold), C.mentaTexto) }
        }
        GraficoVisitas(meses)
        Row(horizontalArrangement = Arrangement.spacedBy(18.dp)) {
            Dato("${f.visitas}", "visitas")
            if (vePrecios) Dato(Formato.pesosCortos(f.gastado), "gastó en total")
            if (f.visitas > 0 && vePrecios) Dato(Formato.pesosCortos(f.gastado / maxOf(1, f.visitas)), "por visita")
        }
    }
}

@Composable
private fun Dato(v: String, t: String) = Column { Texto(v, Tipo.lato(18f, Peso.Black, numeros = true), C.tinta); Texto(t, Tipo.lato(11f), C.sub) }

@Composable
private fun BotonContacto(t: String, s: String, menta: Boolean, habilitado: Boolean, modifier: Modifier, a: () -> Unit) {
    val forma = RoundedCornerShape(18.dp)
    Column(
        modifier.heightIn(min = 64.dp).clip(forma).background(if (menta) C.menta else C.tarjeta).then(if (menta) Modifier else Modifier.border(1.5.dp, C.linea, forma))
            .clickable(enabled = habilitado, onClick = a).atenuado(!(habilitado)),
        horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(4.dp, Alignment.CenterVertically),
    ) {
        val tinta = if (menta) C.petroleoHondo else C.tinta
        Icono(s, 20f, tinta); Texto(t, Tipo.lato(13f, Peso.Bold), tinta)
    }
}

@Composable
private fun Membresia(s: Socio) {
    val usadas = maxOf(0, s.visitasDelPlan - s.quedan)
    val forma = TalonShape(88.dp, 20.dp, 9.dp)
    Row(Modifier.fillMaxWidth().heightIn(min = 88.dp).height(IntrinsicSize.Min).clip(forma).background(C.bloquePetroleo)) {
        Column(Modifier.width(88.dp).fillMaxHeight(), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.Center) {
            Texto("$usadas/${s.visitasDelPlan}", Tipo.lato(22f, Peso.Black, numeros = true), Color.White)
            Texto("USADAS", Tipo.lato(10f, Peso.Black), Color.White.copy(alpha = 0.7f), tracking = 0.8.sp)
        }
        Troquel(Color.White.copy(alpha = 0.3f))
        Column(Modifier.weight(1f).padding(14.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Texto(s.plan, Tipo.lato(17f, Peso.Bold), Color.White)
            Texto("Vence el ${Fecha.iso(s.hasta)?.let { Formato.dia(it) } ?: s.hasta} · quedan ${s.quedan}", Tipo.lato(13f), Color.White.copy(alpha = 0.8f))
            Box(Modifier.fillMaxWidth().height(6.dp).clip(CircleShape).background(Color.White.copy(alpha = 0.18f))) {
                Box(Modifier.fillMaxWidth(usadas.toFloat() / maxOf(1, s.visitasDelPlan)).height(6.dp).clip(CircleShape).background(C.menta))
            }
        }
    }
}

@Composable
private fun FilaTurno(t: TurnoDeCliente, vePrecios: Boolean, alTocar: () -> Unit) {
    val est = EstadoTurno.de(t.estado) ?: EstadoTurno.pendiente
    Row(Modifier.fillMaxWidth().heightIn(min = 60.dp).clickable(onClick = alTocar).padding(horizontal = 14.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Texto(Formato.diaYHora(t.inicio), Tipo.lato(15f, Peso.Bold), C.tinta)
            Texto("${t.servicio} · ${t.profesional}", Tipo.lato(13f), C.sub)
        }
        if (vePrecios) Texto(Formato.pesos(t.precio), Tipo.lato(14f, numeros = true), C.sub)
        PastillaEstado(est.texto, est.simbolo, est.fondo, est.tinta)
    }
}

@Composable
fun ColumnScope.EditarClienta(ficha: Ficha, listo: () -> Unit, cerrar: () -> Unit) {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    var nombre by remember { mutableStateOf(ficha.nombre) }
    var email by remember { mutableStateOf(ficha.email) }
    @Serializable data class Cambio(val nombre: String, val email: String)
    BarraHoja("Editar", cerrar, accion = "Guardar") {
        alcance.launch { try { estado.api.hacer("PATCH", "/v1/panel/clientes/${ficha.id}", Cambio(nombre, email)); listo(); cerrar() } catch (e: Exception) { estado.avisar(e) } }
    }
    Column(Modifier.weight(1f).padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Campo("Nombre", nombre, { nombre = it })
        Campo("Email", email, { email = it }, teclado = KeyboardType.Email)
        Texto("El teléfono identifica a la clienta y no se cambia.", Tipo.lato(13f), C.sub)
    }
}
