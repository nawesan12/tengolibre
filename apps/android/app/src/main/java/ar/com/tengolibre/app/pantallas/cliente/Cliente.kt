package ar.com.tengolibre.app.pantallas.cliente

import android.content.Intent
import android.net.Uri
import android.provider.Settings
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.foundation.background
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
import androidx.compose.foundation.layout.navigationBarsPadding
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
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.nucleo.*
import ar.com.tengolibre.app.pantallas.hoy.BarraHoja
import ar.com.tengolibre.app.pantallas.hoy.TalonShape
import ar.com.tengolibre.app.pantallas.hoy.Troquel
import ar.com.tengolibre.app.pantallas.mas.BotonContornoRojo
import ar.com.tengolibre.app.pantallas.mas.Bloque
import ar.com.tengolibre.app.pantallas.mas.Renglon
import ar.com.tengolibre.app.pantallas.mas.RenglonCampo
import ar.com.tengolibre.app.pantallas.principal.*
import ar.com.tengolibre.app.pantallas.resenas.Estrellas
import ar.com.tengolibre.app.tema.*
import coil3.compose.AsyncImage
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable
import java.time.Instant

// Clon de Pantallas/Cliente/Cliente.swift. Modo clienta (handoff › Cliente): Turnos · Lugares · Cuenta, con la API
// pública, como la web (/[slug]/reservar, /turno/[codigo] y /explorar). Los turnos que reserva o abre se guardan en
// el teléfono por su código.

@Serializable data class NegocioPublico(
    val id: String = "", val slug: String = "", val nombre: String = "", val descripcion: String = "", val direccion: String = "", val ciudad: String = "",
    val telefono: String = "", val whatsapp: String = "", val instagram: String = "", val logo: String? = null,
)

@Serializable data class TurnoCli(
    val id: String, val codigo: String, val servicioId: String? = null, val servicio: String = "", val profesionalId: String? = null, val profesional: String = "",
    val inicio: FechaApi, val fin: FechaApi, val precio: Int = 0, val estado: String = "", val venceEn: FechaApi? = null,
    val negocio: NegocioPublico = NegocioPublico(),
    val puedeCancelar: Boolean? = null, val cancelarHastaHoras: Int? = null, val pago: Pago? = null, val confirmadoEn: FechaApi? = null,
) {
    @Serializable data class Pago(val monto: Int = 0, val modo: String = "", val url: String? = null)
    val activo: Boolean get() = estado in listOf("pendiente_pago", "pendiente", "confirmado") && fin.isAfter(Instant.now())
}

@Serializable data class PaginaPublica(
    val negocio: NegocioPublico = NegocioPublico(), val servicios: List<Serv> = emptyList(), val profesionales: List<Pro> = emptyList(), val diasVista: Int = 30,
    val politicaCancelacion: String = "", val web: WebP = WebP(), val resenas: Resumen = Resumen(),
) {
    @Serializable data class Serv(val id: String, val categoriaId: String? = null, val nombre: String = "", val descripcion: String = "", val duracion: Int = 0, val precio: Int = 0, val foto: String? = null, val orden: Int = 0, val activo: Boolean = true)
    @Serializable data class Pro(val id: String, val nombre: String = "", val especialidad: String = "", val foto: String? = null, val servicios: List<String> = emptyList())
    @Serializable data class WebP(val portada: String? = null, val colorPrimario: String = "", val acerca: String = "", val rubro: String = "")
    @Serializable data class Resumen(val promedio: Double = 0.0, val cantidad: Int = 0)
}

@Serializable data class FichaDirectorio(
    val slug: String, val nombre: String = "", val descripcion: String = "", val logo: String? = null, val portada: String? = null, val ciudad: String = "", val provincia: String = "",
    val rubro: String = "", val servicios: List<String> = emptyList(), val desde: Int? = null,
)

@Serializable data class DiaLibre(val fecha: String, val libres: Int = 0)

/** Los códigos de los turnos de la clienta, en el teléfono. */
object MisTurnos {
    var codigos: List<String>
        get() = Preferencias.lista("misTurnos")
        set(v) = Preferencias.ponerLista("misTurnos", v.toSet().sorted())
    fun sumar(c: String) { codigos = codigos + c.uppercase() }
    fun sacar(c: String) { codigos = codigos.filter { it != c.uppercase() } }
}

enum class PestanaCliente { turnos, lugares, cuenta }

/** Abre la dirección en el mapa del teléfono (maps.apple.com en iOS). */
fun abrirMapa(estado: ar.com.tengolibre.app.app.Estado, n: NegocioPublico) = estado.abrirLink("geo:0,0?q=" + Uri.encode("${n.direccion}, ${n.ciudad}"))

@Composable
fun PrincipalCliente() {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    var pestana by remember { mutableStateOf(PestanaCliente.turnos) }
    var turnos by remember { mutableStateOf<List<TurnoCli>>(emptyList()) }
    var cargando by remember { mutableStateOf(true) }
    var abierto by remember { mutableStateOf<TurnoCli?>(null) }
    var reservarEn by remember { mutableStateOf<String?>(null) }

    suspend fun cargar() {
        turnos = coroutineScope { MisTurnos.codigos.map { c -> async { estado.api.opcional<TurnoCli>("/v1/t/$c") } }.awaitAll() }.filterNotNull().sortedBy { it.inicio }
        cargando = false
    }
    LaunchedEffect(estado.version) { cargar() }
    // Los avisos de sus turnos al teléfono: se pide el permiso una vez y se anota cada turno activo.
    val activos = turnos.filter { it.activo }.map { it.codigo }
    LaunchedEffect(estado.tokenPush, activos.joinToString()) {
        estado.pedirAvisosClienta()
        estado.seguirTurnos(activos)
    }

    Column(Modifier.fillMaxSize().background(C.fondo)) {
        Box(Modifier.weight(1f)) {
            when (pestana) {
                PestanaCliente.turnos -> TurnosCliente(turnos, cargando, abrir = { abierto = it }, reservar = { reservarEn = it }, recargar = { cargar() })
                PestanaCliente.lugares -> Lugares(reservar = { reservarEn = it })
                PestanaCliente.cuenta -> CuentaCliente()
            }
        }
        // La barra de abajo, como BarraPestanas.
        Column(Modifier.fillMaxWidth().background(C.barra).navigationBarsPadding()) {
            Box(Modifier.fillMaxWidth().height(1.dp).background(C.linea))
            Row(Modifier.fillMaxWidth()) {
                listOf(Triple(PestanaCliente.turnos, "Turnos", "calendar"), Triple(PestanaCliente.lugares, "Lugares", "mappin.and.ellipse"), Triple(PestanaCliente.cuenta, "Cuenta", "person.crop.circle")).forEach { (p, t, s) ->
                    val on = p == pestana
                    Box(Modifier.weight(1f).heightIn(min = 49.dp).clickable { pestana = p }.semantics { contentDescription = t; selected = on }) {
                        if (on) Box(Modifier.align(Alignment.TopCenter).size(40.dp, 5.dp).clip(RoundedCornerShape(bottomStart = 6.dp, bottomEnd = 6.dp)).background(C.menta))
                        Column(Modifier.align(Alignment.Center), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(2.dp)) {
                            Box(Modifier.height(30.dp), contentAlignment = Alignment.Center) {
                                if (on) Box(Modifier.size(56.dp, 30.dp).clip(CircleShape).background(C.seleccion))
                                Icono(s, 20f, if (on) C.seleccionTexto else C.sub)
                            }
                            Texto(t, Tipo.lato(11f, if (on) Peso.Black else Peso.Bold), if (on) C.tinta else C.sub)
                        }
                    }
                }
            }
        }
    }
    Hoja(abierto != null, { abierto = null }) { abierto?.let { TurnoClienteHoja(it, listo = { alcance.launch { cargar() } }, cerrar = { abierto = null }) } }
    Hoja(reservarEn != null, { reservarEn = null }) {
        reservarEn?.let { s -> Reservar(s, listo = { codigo -> MisTurnos.sumar(codigo); alcance.launch { cargar() }; pestana = PestanaCliente.turnos }, cerrar = { reservarEn = null }) }
    }
}

// MARK: - Turnos

@Composable
fun TurnosCliente(turnos: List<TurnoCli>, cargando: Boolean, abrir: (TurnoCli) -> Unit, reservar: (String) -> Unit, recargar: suspend () -> Unit) {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    var codigo by remember { mutableStateOf(false) }
    var textoCodigo by remember { mutableStateOf("") }
    val proximos = turnos.filter { it.activo }
    val pasados = turnos.filter { !it.activo }.sortedByDescending { it.inicio }
    val lugares = turnos.sortedByDescending { it.inicio }.map { it.negocio }.distinctBy { it.slug }

    @Composable fun proximo(t: TurnoCli) {
        Column(Modifier.clickable { abrir(t) }) {
            Row(Modifier.fillMaxWidth().heightIn(min = 110.dp).background(C.menta, TalonShape(92.dp, 22.dp, 10.dp)), verticalAlignment = Alignment.CenterVertically) {
                Column(Modifier.width(92.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(2.dp)) {
                    Texto(Formato.dia(t.inicio).uppercase(), Tipo.lato(12f, Peso.Black), C.petroleoHondo, tracking = 0.6.sp)
                    Texto(Formato.horaDe(t.inicio), Tipo.lato(26f, Peso.Black, numeros = true), C.petroleoHondo, tracking = (-0.5).sp)
                }
                Troquel(C.petroleoHondo.copy(alpha = 0.35f), modifier = Modifier.height(110.dp))
                Column(Modifier.weight(1f).padding(16.dp), verticalArrangement = Arrangement.spacedBy(3.dp)) {
                    Texto(t.negocio.nombre, Tipo.lato(20f, Peso.Black), C.petroleoHondo, lineas = 2)
                    Texto("${t.servicio}${if (t.profesional.isEmpty()) "" else " · ${t.profesional}"}", Tipo.lato(14f), C.petroleoHondo.copy(alpha = 0.85f))
                    if (t.estado == "pendiente_pago") Texto("Falta pagar la seña", Tipo.lato(13f, Peso.Bold), C.petroleoHondo)
                    else if (t.estado == "confirmado" || t.confirmadoEn != null) Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(4.dp)) {
                        Icono("checkmark.circle", 13f, C.petroleoHondo); Texto("Confirmado", Tipo.lato(13f, Peso.Bold), C.petroleoHondo)
                    }
                }
            }
            Row(Modifier.padding(top = 10.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Box(Modifier.weight(1f)) { BotonContorno("Mover", { abrir(t) }, alto = 48.dp, tam = 15f, simbolo = "calendar.badge.clock") }
                Box(Modifier.weight(1f)) { BotonContorno("Cómo llegar", { abrirMapa(estado, t.negocio) }, alto = 48.dp, tam = 15f, habilitado = t.negocio.direccion.isNotEmpty(), simbolo = "map") }
            }
        }
    }
    @Composable fun fila(t: TurnoCli, m: Modifier = Modifier) {
        Row(m.fillMaxWidth().heightIn(min = 72.dp).background(C.tarjeta, TalonShape(64.dp, 18.dp, 8.dp)).clickable { abrir(t) }.padding(end = 12.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            Column(Modifier.width(64.dp), horizontalAlignment = Alignment.CenterHorizontally) {
                Texto(Formato.horaDe(t.inicio), Tipo.lato(17f, Peso.Black, numeros = true), C.tinta)
                Texto(Formato.dia(t.inicio), Tipo.lato(11f), C.sub)
            }
            Troquel(modifier = Modifier.height(72.dp))
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                Texto(t.negocio.nombre, Tipo.lato(16f, Peso.Bold), C.tinta)
                Texto(t.servicio, Tipo.lato(13f), C.sub)
            }
            val e = EstadoTurno.de(t.estado) ?: EstadoTurno.pendiente
            PastillaEstado(e.texto, e.simbolo, e.fondo, e.tinta)
        }
    }

    Column(Modifier.fillMaxSize().background(C.fondo).verticalScroll(rememberScrollState())) {
        Cabecera("Tus turnos", if (proximos.isEmpty()) "Nada reservado por ahora" else Formato.cantidad(proximos.size, "turno reservado", "turnos reservados"),
            acciones = listOf(AccionCabecera("plus", "Tengo un código") { codigo = true }))
        Cuerpo {
            val t0 = proximos.firstOrNull()
            if (cargando && turnos.isEmpty()) { Esqueleto(160.dp, 22.dp); Esqueleto(72.dp) }
            else if (t0 != null) { TextoRotulo("TU PRÓXIMO TURNO"); proximo(t0) }
            else VacioQueEnsena("Todo libre", "No tenés turnos reservados", "Buscá un lugar en Lugares o abrí el link del turno que te mandaron.")
            if (lugares.isNotEmpty()) {
                Rotulo("TUS LUGARES")
                Row(Modifier.horizontalScroll(rememberScrollState()), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    lugares.forEach { n ->
                        Column(Modifier.size(150.dp, 130.dp).tarjeta(18.dp).clickable { reservar(n.slug) }.padding(14.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                            Iniciales(n.nombre, 40.dp, C.menta, C.petroleoHondo, 12.dp)
                            Texto(n.nombre, Tipo.lato(15f, Peso.Bold), C.tinta, lineas = 2)
                            Texto("Reservar de nuevo", Tipo.lato(12f, Peso.Bold), C.mentaTexto)
                        }
                    }
                }
            }
            if (proximos.size > 1) { Rotulo("DESPUÉS"); proximos.drop(1).forEach { fila(it) } }
            if (pasados.isNotEmpty()) { Rotulo("ANTES"); pasados.take(20).forEach { fila(it, Modifier.alpha(0.7f)) } }
        }
    }
    // El alert de iOS con un campo: una hoja media con el campo y «Agregar».
    Hoja(codigo, { codigo = false }, alturas = listOf(Altura.media)) {
        BarraHoja("Agregar un turno", { codigo = false; textoCodigo = "" }, accion = "Agregar", habilitada = textoCodigo.isNotBlank()) {
            val c = textoCodigo.trim().split("/").lastOrNull().orEmpty()
            if (c.isNotEmpty()) { MisTurnos.sumar(c); alcance.launch { recargar() } }
            textoCodigo = ""; codigo = false
        }
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Texto("Está en el mail o el WhatsApp de confirmación (o pegá el link).", Tipo.lato(15f), C.sub)
            Campo("Código del turno", textoCodigo, { textoCodigo = it.uppercase() })
        }
    }
}

/** El turno de la clienta: confirmar, pagar la seña, mover, cancelar, cómo llegar y contacto. */
@Composable
fun ColumnScope.TurnoClienteHoja(inicial: TurnoCli, listo: () -> Unit, cerrar: () -> Unit) {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    var turno by remember { mutableStateOf(inicial) }
    var mover by remember { mutableStateOf(false) }
    var cancelar by remember { mutableStateOf(false) }
    @Serializable data class Url(val url: String)
    fun pagar() = alcance.launch {
        turno.pago?.url?.let { estado.abrirPestana(it); return@launch }
        try { val r: Url = estado.api.enviar("POST", "/v1/t/${turno.codigo}/pagar", emptyMap<String, String>()); estado.abrirPestana(r.url) } catch (e: Exception) { estado.avisar(e) }
    }
    fun post(que: String, ok: String) = alcance.launch {
        try {
            turno = estado.api.enviar("POST", "/v1/t/${turno.codigo}/$que", emptyMap<String, String>())
            Haptico.exito(); estado.avisar(ok); listo()
        } catch (e: Exception) { estado.avisar(e) }
    }

    BarraHoja("Tu turno", cerrar, textoCancelar = "", accion = "Listo") { cerrar() }
    Column(Modifier.weight(1f).background(C.hoja).verticalScroll(rememberScrollState()).padding(16.dp), verticalArrangement = Arrangement.spacedBy(14.dp)) {
        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
            TextoRotulo(Formato.fechaLarga(turno.inicio).uppercase())
            Texto("${Formato.horaDe(turno.inicio)} a ${Formato.horaDe(turno.fin)}", Tipo.lato(34f, Peso.Black, numeros = true), C.tinta)
            Texto(turno.negocio.nombre, Tipo.lato(20f, Peso.Bold), C.tinta)
            Texto("${turno.servicio}${if (turno.profesional.isEmpty()) "" else " con ${turno.profesional}"} · ${Formato.pesos(turno.precio)}", Tipo.lato(15f), C.sub)
            val e = EstadoTurno.de(turno.estado) ?: EstadoTurno.pendiente
            Box(Modifier.padding(top = 4.dp)) { PastillaEstado(e.texto, e.simbolo, e.fondo, e.tinta, alto = 26.dp) }
        }
        Texto("Código ${turno.codigo}", Tipo.lato(14f).copy(fontFamily = FontFamily.Monospace), C.sub)
        if (turno.activo) {
            if (turno.estado == "pendiente_pago") {
                BotonMenta("Pagar la seña${turno.pago?.let { " · ${Formato.pesos(it.monto)}" } ?: ""}", { pagar() }, simbolo = "creditcard")
                turno.venceEn?.let { Texto("Si no se paga antes de las ${Formato.horaDe(it)}, el horario se libera.", Tipo.lato(13f), C.sub) }
            } else if (turno.estado == "pendiente") BotonMenta("Confirmar que voy", { post("confirmar", "¡Listo! Confirmaste el turno") }, simbolo = "checkmark.circle")
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Box(Modifier.weight(1f)) { BotonContorno("Mover", { mover = true }, alto = 48.dp, tam = 15f, simbolo = "calendar.badge.clock") }
                Box(Modifier.weight(1f).alpha(if (turno.puedeCancelar == false) 0.45f else 1f)) {
                    BotonContornoRojo("Cancelar", "xmark") { if (turno.puedeCancelar != false) cancelar = true }
                }
            }
            val h = turno.cancelarHastaHoras
            if (turno.puedeCancelar == false && h != null) Texto("Se puede cancelar hasta $h h antes. Escribile al lugar.", Tipo.lato(13f), C.sub)
        }
        val n = turno.negocio
        Bloque("El lugar", "", *buildList<@Composable () -> Unit> {
            if (n.direccion.isNotEmpty()) add { Box(Modifier.clickable { abrirMapa(estado, n) }) { Renglon(n.direccion, n.ciudad, "map") } }
            if (n.whatsapp.isNotEmpty()) add {
                Box(Modifier.clickable {
                    var d = n.whatsapp.filter { it.isDigit() }
                    if (d.startsWith("0")) d = d.drop(1)
                    if (!d.startsWith("54")) d = "549$d"
                    estado.abrirLink("https://wa.me/$d")
                }) { Renglon("Escribir por WhatsApp", simbolo = "message") }
            }
            if (n.telefono.isNotEmpty()) add { Box(Modifier.clickable { estado.abrirLink("tel:" + n.telefono.filter { it in "+0123456789" }) }) { Renglon("Llamar", n.telefono, "phone") } }
        }.toTypedArray())
        Box(Modifier.fillMaxWidth(), contentAlignment = Alignment.CenterStart) {
            Row(Modifier.clickable { estado.compartirTexto("${Web.base}/turno/${turno.codigo}") }.padding(vertical = 8.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                Icono("square.and.arrow.up", 16f, C.mentaTexto); Texto("Compartir el turno", Tipo.lato(16f, Peso.Bold), C.mentaTexto)
            }
        }
        if (!turno.activo) BotonTexto("Sacarlo de la lista", {
            MisTurnos.sacar(turno.codigo); alcance.launch { estado.dejarDeSeguir(turno.codigo); listo() }; cerrar()
        }, color = C.error)
    }
    Confirmacion(cancelar, "¿Cancelar el turno?", "", listOf(OpcionConfirmacion("Cancelar el turno", destructiva = true) { post("cancelar", "Cancelamos el turno") }), { cancelar = false })
    Hoja(mover, { mover = false }) { MoverCliente(turno, listo = { t -> turno = t; listo() }, cerrar = { mover = false }) }
}

/** Mover el turno (la clienta): otro día y horario libre del mismo servicio. */
@Composable
fun ColumnScope.MoverCliente(turno: TurnoCli, listo: (TurnoCli) -> Unit, cerrar: () -> Unit) {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    var dias by remember { mutableStateOf<List<DiaLibre>>(emptyList()) }
    var fecha by remember { mutableStateOf("") }
    var horarios by remember { mutableStateOf<List<HorarioLibre>>(emptyList()) }
    val filtro = listOf("servicio" to turno.servicioId.orEmpty()) + (turno.profesionalId?.let { listOf("profesional" to it) } ?: emptyList())
    @Serializable data class Ds(val dias: List<DiaLibre> = emptyList())
    @Serializable data class Hs(val horarios: List<HorarioLibre> = emptyList())
    @Serializable data class Inicio(val inicio: FechaApi)
    LaunchedEffect(Unit) {
        dias = estado.api.opcional<Ds>("/v1/p/${turno.negocio.slug}/dias", filtro)?.dias ?: emptyList()
        fecha = dias.firstOrNull { it.libres > 0 }?.fecha.orEmpty()
    }
    LaunchedEffect(fecha) {
        horarios = if (fecha.isEmpty()) emptyList() else estado.api.opcional<Hs>("/v1/p/${turno.negocio.slug}/libres", filtro + ("fecha" to fecha))?.horarios ?: emptyList()
    }
    BarraHoja("Mover el turno", cerrar)
    Column(Modifier.weight(1f).background(C.hoja).verticalScroll(rememberScrollState()).padding(16.dp), verticalArrangement = Arrangement.spacedBy(14.dp)) {
        SelectorDias(dias, fecha) { fecha = it }
        GrillaHorarios(horarios) { h ->
            alcance.launch {
                try {
                    val t: TurnoCli = estado.api.enviar("POST", "/v1/t/${turno.codigo}/reprogramar", Inicio(h.inicio))
                    Haptico.exito(); estado.avisar("Movimos tu turno", Formato.diaYHora(t.inicio)); listo(t); cerrar()
                } catch (e: Exception) { estado.avisar(e) }
            }
        }
    }
}

@Composable
fun SelectorDias(dias: List<DiaLibre>, fecha: String, alElegir: (String) -> Unit) {
    Row(Modifier.horizontalScroll(rememberScrollState()), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        dias.forEach { d ->
            val on = d.fecha == fecha
            val f = Fecha.iso(d.fecha) ?: Instant.now()
            Column(
                Modifier.size(56.dp, 72.dp).clip(RoundedCornerShape(16.dp)).background(if (on) C.seleccion else C.fondo)
                    .then(if (on) Modifier else Modifier.contorno(16.dp))
                    .clickable(enabled = d.libres > 0) { alElegir(d.fecha); Haptico.seleccion() },
                horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(2.dp, Alignment.CenterVertically),
            ) {
                val tinta = if (on) C.seleccionTexto else if (d.libres > 0) C.tinta else C.sub
                Texto(Formato.fechaLarga(f).take(3).uppercase(), Tipo.lato(11f, Peso.Black), tinta)
                Texto("${Formato.enArgentina(f).dayOfMonth}", Tipo.lato(20f, Peso.Black, numeros = true), tinta)
                Texto(if (d.libres > 0) "${d.libres}" else "—", Tipo.lato(11f, Peso.Bold), tinta.copy(alpha = 0.8f))
            }
        }
    }
}

@Composable
fun GrillaHorarios(horarios: List<HorarioLibre>, elegir: (HorarioLibre) -> Unit) {
    if (horarios.isEmpty()) { Texto("No hay horarios libres ese día.", Tipo.lato(15f), C.sub); return }
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        horarios.chunked(4).forEach { fila ->
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                fila.forEach { h ->
                    Box(Modifier.weight(1f).heightIn(min = 52.dp).clip(RoundedCornerShape(14.dp)).contorno(14.dp, C.huecoLinea).clickable { elegir(h) }, contentAlignment = Alignment.Center) {
                        Rayas(Modifier.matchParentSize(), quieto = true)
                        Texto(h.hora, Tipo.lato(17f, Peso.Black, numeros = true), C.huecoTinta)
                    }
                }
                repeat(4 - fila.size) { Spacer(Modifier.weight(1f)) }
            }
        }
    }
}

// MARK: - Reservar

@Composable
fun ColumnScope.Reservar(slug: String, listo: (String) -> Unit, cerrar: () -> Unit) {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    var p by remember { mutableStateOf<PaginaPublica?>(null) }
    var servicio by remember { mutableStateOf<PaginaPublica.Serv?>(null) }
    var profesional by remember { mutableStateOf<String?>(null) }
    var dias by remember { mutableStateOf<List<DiaLibre>>(emptyList()) }
    var fecha by remember { mutableStateOf("") }
    var horarios by remember { mutableStateOf<List<HorarioLibre>>(emptyList()) }
    var horario by remember { mutableStateOf<HorarioLibre?>(null) }
    var nombre by remember { mutableStateOf("") }
    var telefono by remember { mutableStateOf("") }
    var email by remember { mutableStateOf("") }
    var nota by remember { mutableStateOf("") }
    var cupon by remember { mutableStateOf("") }
    var enviando by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf("") }
    @Serializable data class Ds(val dias: List<DiaLibre> = emptyList())
    @Serializable data class Hs(val horarios: List<HorarioLibre> = emptyList())
    @Serializable data class Cli(val nombre: String, val telefono: String, val email: String)
    @Serializable data class Pedido(val servicioId: String, val profesionalId: String, val inicio: FechaApi, val cliente: Cli, val nota: String, val cupon: String?)
    @Serializable data class Reservado(val codigo: String, val inicio: FechaApi, val pago: TurnoCli.Pago? = null)
    fun filtro() = listOf("servicio" to servicio?.id.orEmpty()) + (profesional?.let { listOf("profesional" to it) } ?: emptyList())
    suspend fun cargarHorarios() {
        horarios = if (fecha.isEmpty() || servicio == null) emptyList() else estado.api.opcional<Hs>("/v1/p/$slug/libres", filtro() + ("fecha" to fecha))?.horarios ?: emptyList()
    }
    suspend fun cargarDias() {
        horario = null
        dias = estado.api.opcional<Ds>("/v1/p/$slug/dias", filtro())?.dias ?: emptyList()
        val f = dias.firstOrNull { it.libres > 0 }?.fecha.orEmpty()
        if (f == fecha) cargarHorarios() else fecha = f
    }
    suspend fun cargar() {
        try {
            val x: PaginaPublica = estado.api.get("/v1/p/$slug"); p = x
            nombre = estado.persona?.nombre.orEmpty(); email = estado.persona?.email.orEmpty()
            telefono = Preferencias.texto("miTelefono").orEmpty()
            if (x.servicios.count { it.activo } == 1) { servicio = x.servicios.first { it.activo }; cargarDias() }
        } catch (e: ErrorApi) { error = e.detalle } catch (e: Exception) { error = "Probá de nuevo." }
    }
    LaunchedEffect(Unit) { cargar() }
    LaunchedEffect(fecha) { horario = null; cargarHorarios() }
    fun reservar(s: PaginaPublica.Serv, h: HorarioLibre) = alcance.launch {
        enviando = true; error = ""
        try {
            val t: Reservado = estado.api.enviar("POST", "/v1/p/$slug/turnos", Pedido(s.id, profesional.orEmpty(), h.inicio, Cli(nombre, telefono, email), nota, cupon.ifEmpty { null }))
            Preferencias.ponerTexto("miTelefono", telefono)
            Haptico.exito()
            listo(t.codigo)
            t.pago?.url?.takeIf { it.startsWith("https://") }?.let { estado.abrirPestana(it) }
            estado.avisar("¡Reservaste!", "${Formato.diaYHora(t.inicio)} en ${p?.negocio?.nombre.orEmpty()}")
            cerrar()
        } catch (e: ErrorApi) {
            error = e.detalle; Haptico.error()
            if (e.codigo == "ocupado") { horario = null; cargarHorarios() }
        } catch (e: Exception) { error = "No pudimos reservar. Probá de nuevo." }
        enviando = false
    }

    BarraHoja(p?.negocio?.nombre ?: "Reservar", cerrar, textoCancelar = "Cerrar")
    Column(Modifier.weight(1f).background(C.fondo).verticalScroll(rememberScrollState()).padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        val x = p
        if (x == null) { if (error.isNotEmpty()) ErrorConSalida("No pudimos abrir el lugar", error) { alcance.launch { cargar() } } else repeat(4) { Esqueleto(72.dp) }; return@Column }
        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Texto(x.negocio.nombre, Tipo.lato(26f, Peso.Black), C.tinta)
            Texto(listOf(x.web.rubro, x.negocio.ciudad).filter { it.isNotEmpty() }.joinToString(" · "), Tipo.lato(14f), C.sub)
            if (x.resenas.cantidad > 0) Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                Estrellas(Math.round(x.resenas.promedio).toInt(), 12f)
                Texto("${"%.1f".format(Formato.argentina, x.resenas.promedio)} · ${x.resenas.cantidad} reseñas", Tipo.lato(13f), C.tinta)
            }
        }
        TextoRotulo("1 · SERVICIO")
        x.servicios.filter { it.activo }.sortedBy { it.orden }.forEach { s ->
            val on = servicio?.id == s.id
            Row(Modifier.fillMaxWidth().clip(RoundedCornerShape(16.dp)).background(if (on) C.seleccion else C.tarjeta).clickable { servicio = s; profesional = null; alcance.launch { cargarDias() } }.padding(14.dp), verticalAlignment = Alignment.CenterVertically) {
                val tinta = if (on) C.seleccionTexto else C.tinta
                Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                    Texto(s.nombre, Tipo.lato(16f, Peso.Bold), tinta)
                    Texto(Formato.duracion(s.duracion), Tipo.lato(13f), tinta.copy(alpha = 0.8f))
                }
                Texto(Formato.pesos(s.precio), Tipo.lato(15f, Peso.Bold, numeros = true), tinta)
            }
        }
        val s = servicio ?: return@Column
        val pros = x.profesionales.filter { it.servicios.isEmpty() || s.id in it.servicios }
        if (pros.size > 1) {
            TextoRotulo("2 · CON QUIÉN")
            Row(Modifier.horizontalScroll(rememberScrollState()), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                (listOf<Pair<String?, String>>(null to "Cualquiera") + pros.map { it.id to it.nombre }).forEach { (id, t) ->
                    val on = profesional == id
                    Box(Modifier.heightIn(min = 44.dp).clip(CircleShape).background(if (on) C.seleccion else C.tarjeta).clickable { profesional = id; alcance.launch { cargarDias() } }.padding(horizontal = 16.dp), contentAlignment = Alignment.Center) {
                        Texto(t, Tipo.lato(14f, Peso.Bold), if (on) C.seleccionTexto else C.tinta)
                    }
                }
            }
        }
        TextoRotulo("3 · CUÁNDO")
        SelectorDias(dias, fecha) { fecha = it }
        val h = horario
        if (h == null) GrillaHorarios(horarios) { horario = it; Haptico.seleccion() }
        else {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Row(Modifier.weight(1f), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    Icono("checkmark.circle.fill", 15f, C.mentaTexto)
                    Texto("${Fecha.iso(fecha)?.let { Formato.fechaLarga(it) } ?: fecha} a las ${h.hora}", Tipo.lato(15f, Peso.Bold), C.mentaTexto)
                }
                BotonTexto("Cambiar", { horario = null })
            }
            TextoRotulo("4 · TUS DATOS")
            Bloque("", "",
                { RenglonCampo("Nombre y apellido", nombre, { nombre = it }) },
                { RenglonCampo("Teléfono", telefono, { telefono = it }, teclado = KeyboardType.Phone) },
                { RenglonCampo("Email", email, { email = it }, teclado = KeyboardType.Email) },
                { RenglonCampo("Algo que quieras avisar (opcional)", nota, { nota = it }) },
                { RenglonCampo("Cupón (opcional)", cupon, { cupon = it }) })
            if (x.politicaCancelacion.isNotEmpty()) Texto(x.politicaCancelacion, Tipo.lato(13f), C.sub)
            if (error.isNotEmpty()) Texto(error, Tipo.lato(14f, Peso.Bold), C.error)
            BotonMenta(if (enviando) "Reservando…" else "Reservar · ${Formato.pesos(s.precio)}", { reservar(s, h) },
                habilitado = !enviando && nombre.length >= 2 && telefono.count { it.isDigit() } >= 8)
        }
    }
}

// MARK: - Lugares

@Composable
fun Lugares(reservar: (String) -> Unit) {
    val estado = LocalEstado.current
    var lista by remember { mutableStateOf<List<FichaDirectorio>>(emptyList()) }
    var rubros by remember { mutableStateOf<List<String>>(emptyList()) }
    var rubro by remember { mutableStateOf("") }
    var buscar by remember { mutableStateOf("") }
    var hoy by remember { mutableStateOf(false) }
    var cargando by remember { mutableStateOf(true) }
    @Serializable data class Dir(val negocios: List<FichaDirectorio> = emptyList())
    @Serializable data class Rs(val rubros: List<String> = emptyList())
    suspend fun cargar() {
        val q = buildList { add("limite" to "24"); if (buscar.isNotEmpty()) add("q" to buscar); if (rubro.isNotEmpty()) add("rubro" to rubro); if (hoy) add("conLugarHoy" to "1") }
        estado.api.opcional<Dir>("/v1/explorar", q)?.let { lista = it.negocios }
        if (rubros.isEmpty()) rubros = estado.api.opcional<Rs>("/v1/explorar/rubros")?.rubros ?: emptyList()
        cargando = false
    }
    LaunchedEffect(rubro, hoy) { cargar() }
    LaunchedEffect(buscar) { delay(350); cargar() }
    @Composable fun filtro(t: String, on: Boolean, a: () -> Unit) {
        Box(Modifier.height(36.dp).clip(CircleShape).background(if (on) C.menta else Color.White.copy(alpha = 0.1f)).clickable(onClick = a).padding(horizontal = 14.dp), contentAlignment = Alignment.Center) {
            Texto(t, Tipo.lato(14f, Peso.Bold), if (on) C.petroleoHondo else Color.White)
        }
    }
    Column(Modifier.fillMaxSize().background(C.fondo).verticalScroll(rememberScrollState())) {
        Cabecera("Lugares", "Reservá donde haya lugar") {
            Row(Modifier.fillMaxWidth().height(48.dp).clip(RoundedCornerShape(14.dp)).background(Color.White.copy(alpha = 0.12f)).padding(horizontal = 14.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Icono("magnifyingglass", 17f, Color.White.copy(alpha = 0.7f))
                Box(Modifier.weight(1f)) {
                    if (buscar.isEmpty()) Texto("Peluquería, uñas, masajes…", Tipo.lato(16f), Color.White.copy(alpha = 0.6f))
                    BasicTextField(buscar, { buscar = it }, Modifier.fillMaxWidth(), singleLine = true, textStyle = Tipo.lato(16f).copy(color = Color.White), cursorBrush = SolidColor(Color.White))
                }
            }
            Row(Modifier.horizontalScroll(rememberScrollState()), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                filtro("Hoy con lugar", hoy) { hoy = !hoy }
                rubros.forEach { r -> filtro(r, rubro == r) { rubro = if (rubro == r) "" else r } }
            }
        }
        Cuerpo {
            if (cargando && lista.isEmpty()) repeat(3) { Esqueleto(180.dp, 20.dp) }
            else if (lista.isEmpty()) VacioQueEnsena("Nada por acá", "No encontramos lugares", "Probá con otra búsqueda o sacá los filtros.")
            else {
                TextoRotulo("CERCA TUYO")
                lista.chunked(2).forEach { fila ->
                    Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                        fila.forEach { f ->
                            Column(Modifier.weight(1f).clip(RoundedCornerShape(18.dp)).background(C.tarjeta).clickable { reservar(f.slug) }) {
                                Box(Modifier.fillMaxWidth().height(104.dp).background(C.chip), contentAlignment = Alignment.Center) {
                                    if (f.portada != null) AsyncImage(f.portada, null, Modifier.matchParentSize(), contentScale = ContentScale.Crop)
                                    else Icono("sparkles", 26f, C.sub)
                                }
                                Column(Modifier.padding(10.dp), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                                    Texto(f.nombre, Tipo.lato(15f, Peso.Bold), C.tinta, lineas = 2)
                                    Texto(listOf(f.rubro, f.ciudad).filter { it.isNotEmpty() }.joinToString(" · "), Tipo.lato(12f), C.sub, lineas = 1)
                                    f.desde?.let { Texto("desde ${Formato.pesos(it)}", Tipo.lato(12f, Peso.Bold), C.mentaTexto) }
                                }
                            }
                        }
                        if (fila.size == 1) Spacer(Modifier.weight(1f))
                    }
                }
            }
        }
    }
}

// MARK: - Cuenta

@Composable
fun CuentaCliente() {
    val estado = LocalEstado.current
    val contexto = LocalContext.current
    val alcance = rememberCoroutineScope()
    var salir by remember { mutableStateOf(false) }
    Column(Modifier.fillMaxSize().background(C.fondo).verticalScroll(rememberScrollState())) {
        Cabecera("Cuenta", estado.persona?.email.orEmpty())
        Cuerpo {
            Bloque("", "", { Renglon(estado.persona?.nombre.orEmpty(), estado.persona?.email.orEmpty(), "person.crop.circle") })
            Bloque("Modo", "", { Box(Modifier.clickable { alcance.launch { estado.elegirModo("negocio") } }) {
                Renglon("Atiendo un negocio", if (estado.negocios.isEmpty()) "Abrí el tuyo gratis" else estado.negocios.joinToString(", ") { it.nombre }, "storefront")
            } })
            Bloque("Avisos", "", { Box(Modifier.clickable { contexto.startActivity(Intent(Settings.ACTION_APP_NOTIFICATION_SETTINGS).putExtra(Settings.EXTRA_APP_PACKAGE, contexto.packageName)) }) {
                Renglon("Avisos de tus turnos", "Se cambian en Ajustes", "bell")
            } })
            BotonContornoRojo("Cerrar sesión", "rectangle.portrait.and.arrow.right", alto = 52.dp) { salir = true }
        }
    }
    Confirmacion(salir, "¿Cerrar sesión?", "", listOf(OpcionConfirmacion("Cerrar sesión", destructiva = true) { alcance.launch { estado.salir() } }), { salir = false })
}
