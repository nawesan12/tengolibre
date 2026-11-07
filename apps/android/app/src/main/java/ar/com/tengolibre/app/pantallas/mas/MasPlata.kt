package ar.com.tengolibre.app.pantallas.mas

import android.app.Activity
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.foundation.Image
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
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
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import ar.com.tengolibre.app.R
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.nucleo.*
import ar.com.tengolibre.app.pantallas.clientes.DestinoClientes
import ar.com.tengolibre.app.pantallas.clientes.EtiquetaConUso
import ar.com.tengolibre.app.pantallas.clientes.rememberContactos
import ar.com.tengolibre.app.pantallas.hoy.BarraHoja
import ar.com.tengolibre.app.pantallas.hoy.Troquel
import ar.com.tengolibre.app.pantallas.principal.*
import ar.com.tengolibre.app.tema.*
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonPrimitive
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId

// Clon de Pantallas/Mas/MasPlata.swift. La suscripción se paga con Google Play en lugar de la App Store.

@Serializable data class ConexionMP(val disponible: Boolean = false, val conectada: Boolean = false, val desde: FechaApi? = null, val venceEn: FechaApi? = null, val reautorizar: Boolean = false, val ultimoError: String? = null, val cuenta: String? = null)
@Serializable data class ConfigCobro(val modo: String = "sin_cobro", val senaPorcentaje: Int = 30, val senaFija: Int = 0)
@Serializable data class PagoSena(
    val id: String, val turnoId: String = "", val codigo: String = "", val servicio: String = "", val cliente: String = "", val inicio: FechaApi = Instant.EPOCH, val turnoEstado: String = "",
    val monto: Int = 0, val modo: String = "", val estado: String = "", val devuelto: Int = 0, val venceEn: FechaApi? = null, val vencido: Boolean = false, val creadoEn: FechaApi = Instant.EPOCH,
)
@Serializable data class Cupon(
    val id: String, val codigo: String = "", val tipo: String = "porcentaje", val valor: Int = 0, val minimo: Int? = null, val maxUsos: Int? = null, val usos: Int = 0,
    val desde: FechaApi? = null, val hasta: FechaApi? = null, val activo: Boolean = true, val referidoId: String? = null, val creadoEn: FechaApi? = null,
) { val texto: String get() = if (tipo == "porcentaje") "$valor % off" else "${Formato.pesos(valor)} off" }
@Serializable data class ResumenCaja(
    val fecha: String = "", val porMedio: Map<String, Int> = emptyMap(), val caja: Int = 0, val senas: Int = 0, val total: Int = 0, val productos: Int = 0, val cobros: Int = 0, val debe: Int = 0,
    val movimientos: List<Mov> = emptyList(),
) { @Serializable data class Mov(val turnoId: String, val cliente: String = "", val servicio: String = "", val medio: String = "", val total: Int = 0, val productos: Int = 0, val cobradoEn: FechaApi = Instant.EPOCH) }
@Serializable data class PlanMembresia(
    val id: String, val nombre: String = "", val descripcion: String = "", val precio: Int = 0, val visitas: Int = 4, val duracionDias: Int = 30, val prioridadDias: Int = 0,
    val beneficios: List<String> = emptyList(), val topeSocios: Int? = null, val servicios: List<String> = emptyList(), val activo: Boolean = true, val orden: Int = 0,
)
@Serializable data class Socio(
    val id: String, val planId: String = "", val plan: String = "", val visitasDelPlan: Int = 0, val clienteId: String = "", val cliente: String = "", val telefono: String = "",
    val desde: String = "", val hasta: String = "", val estado: String = "", val quedan: Int = 0, val nota: String = "",
)

val mediosPago = mapOf("efectivo" to "Efectivo", "transferencia" to "Transferencia", "mercadopago" to "QR de Mercado Pago", "debito" to "Débito", "credito" to "Crédito", "despues" to "Debe")
private val MEDIOS = listOf("efectivo", "transferencia", "mercadopago", "debito", "credito")
private fun diaTexto(s: String) = Fecha.iso(s)?.let { Formato.dia(it) } ?: s

/** El botón «Guardar» adentro de un Bloque (BotonGuardar con .padding(12) en iOS). */
@Composable
private fun GuardarEnBloque(texto: String = "Guardar", habilitado: Boolean = true, accion: suspend () -> Unit) {
    val alcance = rememberCoroutineScope()
    var guardando by remember { mutableStateOf(false) }
    Box(Modifier.padding(12.dp)) { BotonGuardar({ alcance.launch { guardando = true; accion(); guardando = false } }, texto, guardando, habilitado) }
}

/** Un interruptor suelto, sin título (Toggle con .labelsHidden() en iOS). */
@Composable
private fun Llave(activo: Boolean, alCambiar: (Boolean) -> Unit) {
    val x by animateFloatAsState(if (activo) 1f else 0f, Movimiento.tocar(), label = "llave")
    Box(Modifier.size(51.dp, 31.dp).clip(CircleShape).background(if (activo) C.menta else C.linea).clickable { alCambiar(!activo) }.padding(2.dp)) {
        Box(Modifier.offset(x = (20 * x).dp).size(27.dp).sombra(CircleShape, Color.Black.copy(alpha = 0.15f), 4.dp, y = 2.dp).clip(CircleShape).background(Color.White))
    }
}

/** Un Picker .segmented chico dentro de un renglón. */
@Composable
private fun SegmentadoChico(opciones: List<Pair<String, String>>, elegido: String, alElegir: (String) -> Unit) {
    Box(Modifier.widthIn(max = 200.dp)) { Segmentado(opciones, elegido, alElegir, sobrePetroleo = false) }
}

// MARK: - Pagos y señas

@Composable
fun Cobros() {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    var mp by remember { mutableStateOf<ConexionMP?>(null) }
    var config by remember { mutableStateOf(ConfigCobro()) }
    var pagos by remember { mutableStateOf<List<PagoSena>>(emptyList()) }
    var cargando by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf("") }
    var devolver by remember { mutableStateOf<PagoSena?>(null) }

    @Serializable data class Pagos(val pagos: List<PagoSena> = emptyList())
    @Serializable data class Url(val url: String)
    suspend fun cargar() {
        try {
            mp = estado.api.get("/v1/panel/mercadopago")
            estado.api.opcional<ConfigCobro>("/v1/panel/cobros")?.let { config = it }
            pagos = estado.api.opcional<Pagos>("/v1/panel/pagos")?.pagos ?: emptyList()
            error = ""
        } catch (e: ErrorApi) { error = e.detalle } catch (e: Exception) { error = "Probá de nuevo." }
        cargando = false
    }
    LaunchedEffect(estado.version) { cargar() }
    fun conectar() = alcance.launch {
        try { val r: Url = estado.api.enviar("POST", "/v1/panel/mercadopago/conectar", emptyMap<String, String>()); estado.abrirPestana(r.url) } catch (e: Exception) { estado.avisar(e) }
    }
    fun estadoPago(p: PagoSena): String {
        if (p.devuelto > 0) return if (p.devuelto >= p.monto) "devuelto" else "devuelto en parte"
        return mapOf("aprobado" to "pagado", "pendiente" to (if (p.vencido) "venció" else "esperando"), "rechazado" to "rechazado")[p.estado] ?: p.estado
    }

    val m = mp
    PantallaMas("Pagos y señas", if (m?.conectada == true) "Mercado Pago conectado" else "Sin Mercado Pago", cargando = cargando, error = error, recargar = { cargar() }) {
        Bloque("Mercado Pago", "Las señas van directo a tu cuenta. Tengo Libre no cobra comisión.", *buildList<@Composable () -> Unit> {
            if (m != null && m.conectada) {
                add { Renglon(m.cuenta ?: "Cuenta conectada", m.desde?.let { "Desde el ${Formato.dia(it)}" } ?: "", "checkmark.seal.fill") }
                if (m.reautorizar) add { Box(Modifier.clickable { conectar() }) { Renglon("Volver a autorizar", m.ultimoError ?: "Mercado Pago pide que la conectes de nuevo", "exclamationmark.triangle") } }
                add { Box(Modifier.clickable { alcance.launch { estado.intentar("Desconectamos Mercado Pago") { estado.api.hacer("DELETE", "/v1/panel/mercadopago") }; cargar() } }) {
                    Row(Modifier.fillMaxWidth().heightIn(min = 52.dp).padding(horizontal = 16.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                        Icono("xmark.circle", 17f, C.error); Texto("Desconectar", Tipo.lato(16f), C.error)
                    }
                } }
            } else add {
                Row(Modifier.fillMaxWidth().heightIn(min = 56.dp).clickable(enabled = m?.disponible != false) { conectar() }.atenuado(m?.disponible == false),
                    horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically) {
                    Image(painterResource(R.drawable.logo_mercadopago), null, Modifier.size(28.dp)); Spacer(Modifier.width(10.dp))
                    Texto("Conectar Mercado Pago", Tipo.lato(16f, Peso.Bold), C.tinta)
                }
            }
        }.toTypedArray())
        val habilitado = m?.conectada == true
        Box(Modifier.atenuado(!(habilitado))) {
            Bloque("Cobro al reservar", "", *buildList<@Composable () -> Unit> {
                add { Renglon("Qué se cobra") { SelectorMenu(listOf("sin_cobro" to "Nada", "porcentaje" to "Seña en %", "fija" to "Seña fija", "total" to "El total"), config.modo, { if (habilitado) config = config.copy(modo = it) }) } }
                if (config.modo == "porcentaje") add { Numerito("Porcentaje", config.senaPorcentaje, { config = config.copy(senaPorcentaje = it) }, 5..100, paso = 5) { "$it %" } }
                if (config.modo == "fija") add { RenglonPesos("Monto", config.senaFija) { config = config.copy(senaFija = it) } }
                add { GuardarEnBloque(habilitado = habilitado) { estado.intentar("Guardamos cómo cobrás") { estado.api.hacer("PUT", "/v1/panel/cobros", config) } } }
            }.toTypedArray())
            if (!habilitado) Box(Modifier.matchParentSize().clickable(remember { MutableInteractionSource() }, null) {})
        }
        if (pagos.isNotEmpty()) Bloque("Últimos pagos", "", *pagos.take(40).map { p -> @Composable {
            Renglon("${p.cliente} · ${Formato.pesos(p.monto)}", "${p.servicio} · ${Formato.diaYHora(p.inicio)} · ${estadoPago(p)}") {
                if (p.estado == "aprobado" && p.devuelto < p.monto) BotonTexto("Devolver", { devolver = p }, color = C.error)
            }
        } }.toTypedArray())
    }
    val d = devolver
    Confirmacion(d != null, "¿Devolver ${d?.let { Formato.pesos(it.monto - it.devuelto) }.orEmpty()} a ${d?.cliente.orEmpty()}?", "",
        listOf(OpcionConfirmacion("Devolver y cancelar el turno", destructiva = true) {
            d?.let { p -> alcance.launch { estado.intentar("Devolvimos el pago y cancelamos el turno") { estado.api.hacer("POST", "/v1/panel/pagos/${p.id}/devolver", emptyMap<String, Int>()) }; cargar() } }
        }), { devolver = null })
}

// MARK: - Caja

@Composable
fun CajaDelDia() {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    val alcance = rememberCoroutineScope()
    val hoy = Formato.enArgentina(Instant.now()).toLocalDate()
    var dia by remember { mutableStateOf(hoy) }
    var r by remember { mutableStateOf<ResumenCaja?>(null) }
    var error by remember { mutableStateOf("") }
    var calendario by remember { mutableStateOf(false) }
    suspend fun cargar() {
        try { r = estado.api.get("/v1/panel/caja", listOf("fecha" to dia.toString())); error = "" }
        catch (e: ErrorApi) { error = e.detalle } catch (e: Exception) { error = "Probá de nuevo." }
    }
    LaunchedEffect(estado.version, dia) { cargar() }
    fun cambiar(d: Long) { val x = dia.plusDays(d); if (x <= hoy) dia = x }
    val instante = dia.atStartOfDay(ZoneId.of("America/Argentina/Buenos_Aires")).plusHours(12).toInstant()

    val x = r
    PantallaMas("Caja", Formato.fechaLarga(instante), cargando = x == null && error.isEmpty(), error = error, recargar = { cargar() }, abajo = {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Box(Modifier.size(44.dp).clickable { cambiar(-1) }.semantics { contentDescription = "Día anterior" }, contentAlignment = Alignment.Center) { Icono("chevron.left", 17f, Color.White) }
            Spacer(Modifier.weight(1f))
            Box(Modifier.clip(RoundedCornerShape(8.dp)).background(Color.White.copy(alpha = 0.12f)).clickable { calendario = !calendario }.padding(horizontal = 10.dp, vertical = 6.dp)) {
                Texto(java.time.format.DateTimeFormatter.ofPattern("d MMM yyyy", Formato.argentina).format(dia), Tipo.lato(16f), Color.White)
            }
            Spacer(Modifier.weight(1f))
            Box(Modifier.size(44.dp).clickable(enabled = dia < hoy) { cambiar(1) }.alpha(if (dia < hoy) 1f else 0.35f).semantics { contentDescription = "Día siguiente" }, contentAlignment = Alignment.Center) { Icono("chevron.right", 17f, Color.White) }
        }
        if (calendario) Box(Modifier.clip(RoundedCornerShape(16.dp)).background(C.tarjeta).padding(8.dp)) { CalendarioMes(dia, { dia = it; calendario = false }, maximo = hoy) }
    }) {
        if (x == null) return@PantallaMas
        Column(Modifier.fillMaxWidth().tarjeta().padding(16.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            TextoRotulo("TOTAL DEL DÍA")
            NumeroAnimado(Formato.pesos(x.total), Tipo.lato(40f, Peso.Black, numeros = true), C.tinta)
            Texto("${Formato.pesos(x.caja)} en el local · ${Formato.pesos(x.senas)} de señas", Tipo.lato(14f), C.sub)
        }
        if (x.total > 0) Column(Modifier.entrada(1).fillMaxWidth().tarjeta().padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            TextoRotulo("CÓMO TE PAGARON")
            GraficoDona(MEDIOS.mapIndexed { i, m -> ParteDona(mediosPago[m] ?: m, x.porMedio[m] ?: 0, PaletaSeries.color(i)) } +
                (if (x.senas > 0) listOf(ParteDona("Señas online", x.senas, PaletaSeries.color(5))) else emptyList()), Formato.pesosCortos(x.total), "en total")
        }
        Bloque("Por medio de pago", "", *(MEDIOS.map { m -> @Composable { Renglon(mediosPago[m] ?: m) { Texto(Formato.pesos(x.porMedio[m] ?: 0), Tipo.lato(15f, Peso.Bold, numeros = true), C.tinta) } } } +
            (if (x.productos > 0) listOf(@Composable { Renglon("De eso, productos") { Texto(Formato.pesos(x.productos), Tipo.lato(16f, numeros = true), C.tinta) } }) else emptyList()) +
            (if (x.debe > 0) listOf(@Composable { Renglon("Quedaron debiendo", simbolo = "exclamationmark.circle") { Texto(Formato.pesos(x.debe), Tipo.lato(15f, Peso.Bold, numeros = true), C.ausenteTinta) } }) else emptyList())).toTypedArray())
        if (x.movimientos.isNotEmpty()) Bloque("Cobros (${x.cobros})", "", *x.movimientos.map { mv -> @Composable {
            Box(Modifier.clickable { alcance.launch { estado.api.opcional<Turno>("/v1/panel/turnos/${mv.turnoId}")?.let { nav.turno = it } } }) {
                Renglon(mv.cliente, "${mv.servicio} · ${mediosPago[mv.medio] ?: mv.medio} · ${Formato.horaDe(mv.cobradoEn)}") { Texto(Formato.pesos(mv.total), Tipo.lato(15f, Peso.Bold, numeros = true), C.tinta) }
            }
        } }.toTypedArray())
        else Texto("Ese día no hubo cobros en el local.", Tipo.lato(14f), C.sub)
    }
}

// MARK: - Membresías

@Composable
fun Membresias() {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    var planes by remember { mutableStateOf<List<PlanMembresia>>(emptyList()) }
    var socios by remember { mutableStateOf<List<Socio>>(emptyList()) }
    var servicios by remember { mutableStateOf<List<Servicio>>(emptyList()) }
    var tab by remember { mutableStateOf("activa") }
    var cargando by remember { mutableStateOf(true) }
    var plan by remember { mutableStateOf<PlanMembresia?>(null) }
    var nuevoSocio by remember { mutableStateOf(false) }
    var socio by remember { mutableStateOf<Socio?>(null) }
    @Serializable data class Ps(val planes: List<PlanMembresia> = emptyList())
    @Serializable data class Ss(val socios: List<Socio> = emptyList())
    @Serializable data class Sv(val servicios: List<Servicio> = emptyList())
    suspend fun cargar() {
        estado.api.opcional<Ps>("/v1/panel/membresias/planes")?.let { planes = it.planes }
        estado.api.opcional<Ss>("/v1/panel/membresias/socios", listOf("estado" to "todas"))?.let { socios = it.socios }
        estado.api.opcional<Sv>("/v1/panel/servicios")?.let { servicios = it.servicios }
        cargando = false
    }
    LaunchedEffect(estado.version) { cargar() }
    val planNuevo = { PlanMembresia("", orden = planes.size + 1) }

    PantallaMas("Membresías", "${socios.count { it.estado == "activa" }} socias activas",
        acciones = listOf(AccionCabecera("person.badge.plus", "Sumar socia") { nuevoSocio = true }), cargando = cargando, recargar = { cargar() },
        abajo = { Segmentado(listOf("activa" to "Activas", "vencida" to "Vencidas", "planes" to "Planes"), tab, { tab = it }) }) {
        if (tab == "planes") {
            planes.sortedBy { it.orden }.forEach { p ->
                Column(Modifier.fillMaxWidth().tarjeta().clickable { plan = p }.padding(16.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Texto(p.nombre, Tipo.lato(17f, Peso.Bold), C.tinta, Modifier.weight(1f))
                        Texto("${Formato.pesos(p.precio)}/mes", Tipo.lato(15f, Peso.Bold, numeros = true), C.tinta)
                    }
                    Texto("${Formato.cantidad(p.visitas, "visita", "visitas")} cada ${p.duracionDias} días${if (p.prioridadDias > 0) " · reserva ${p.prioridadDias} días antes" else ""}", Tipo.lato(13f), C.sub)
                    Texto("${socios.count { it.planId == p.id && it.estado == "activa" }}${p.topeSocios?.let { " de $it" } ?: ""} socias${if (p.activo) "" else " · pausado"}", Tipo.lato(12f, Peso.Bold), C.mentaTexto)
                }
            }
            BotonContorno("Crear un plan", { plan = planNuevo() }, simbolo = "plus")
        } else {
            val l = socios.filter { it.estado == tab }
            if (l.isEmpty()) VacioQueEnsena("Sin socias", if (tab == "activa") "Todavía no hay socias" else "No hay vencidas", "Las membresías son abonos mensuales con visitas incluidas. Se venden desde tu página o las cargás acá.")
            else Bloque("", "", *l.map { s -> @Composable {
                Box(Modifier.clickable { socio = s }) {
                    Renglon(s.cliente, "${s.plan} · quedan ${s.quedan} de ${s.visitasDelPlan} · vence ${diaTexto(s.hasta)}") { Icono("chevron.right", 13f, C.sub) }
                }
            } }.toTypedArray())
        }
    }
    Hoja(plan != null, { plan = null }) { plan?.let { EditarPlan(it, servicios, listo = { alcance.launch { cargar() } }, cerrar = { plan = null }) } }
    Hoja(nuevoSocio, { nuevoSocio = false }) { NuevaSocia(planes.filter { it.activo }, listo = { alcance.launch { cargar() } }, cerrar = { nuevoSocio = false }) }
    Hoja(socio != null, { socio = null }, alturas = listOf(Altura.media, Altura.grande)) { socio?.let { AccionesSocia(it, listo = { alcance.launch { cargar() } }, cerrar = { socio = null }) } }
}

@Composable
fun ColumnScope.EditarPlan(inicial: PlanMembresia, servicios: List<Servicio>, listo: () -> Unit, cerrar: () -> Unit) {
    val estado = LocalEstado.current
    var plan by remember { mutableStateOf(inicial) }
    var beneficios by remember { mutableStateOf(inicial.beneficios.joinToString("\n")) }
    var conTope by remember { mutableStateOf(inicial.topeSocios != null) }
    var tope by remember { mutableIntStateOf(inicial.topeSocios ?: 20) }
    @Serializable data class Cuerpo(val nombre: String, val descripcion: String, val precio: Int, val visitas: Int, val duracionDias: Int, val prioridadDias: Int,
        val beneficios: List<String>, val topeSocios: Int?, val servicios: List<String>, val activo: Boolean)
    HojaEdicion(if (plan.id.isEmpty()) "Plan nuevo" else "Editar plan", cerrar, habilitado = plan.nombre.length >= 2,
        borrar = if (plan.id.isEmpty()) null else ({ estado.intentar { estado.api.hacer("DELETE", "/v1/panel/membresias/planes/${plan.id}") }; listo() }),
        accion = {
            val c = Cuerpo(plan.nombre, plan.descripcion, plan.precio, plan.visitas, plan.duracionDias, plan.prioridadDias,
                beneficios.split("\n").map { it.trim() }.filter { it.isNotEmpty() }, if (conTope) tope else null, plan.servicios, plan.activo)
            val ok = estado.intentar { if (plan.id.isEmpty()) estado.api.hacer("POST", "/v1/panel/membresias/planes", c) else estado.api.hacer("PATCH", "/v1/panel/membresias/planes/${plan.id}", c) }
            if (ok) listo()
            ok
        }) {
        Bloque("", "", *buildList<@Composable () -> Unit> {
            add { RenglonCampo("Nombre", plan.nombre, { plan = plan.copy(nombre = it) }, "Club del brushing") }
            add { RenglonCampo("Descripción", plan.descripcion, { plan = plan.copy(descripcion = it) }, multilinea = true) }
            add { RenglonPesos("Precio por mes", plan.precio) { plan = plan.copy(precio = it) } }
            add { Numerito("Visitas incluidas", plan.visitas, { plan = plan.copy(visitas = it) }, 1..60) }
            add { Numerito("Dura", plan.duracionDias, { plan = plan.copy(duracionDias = it) }, 7..365) { "$it días" } }
            add { Numerito("Reserva antes que el resto", plan.prioridadDias, { plan = plan.copy(prioridadDias = it) }, 0..30) { if (it == 0) "No" else "$it días" } }
            add { RenglonInterruptor("Cupo de socias", conTope, { conTope = it }) }
            if (conTope) add { Numerito("Máximo", tope, { tope = it }, 1..999) }
            add { RenglonCampo("Beneficios (uno por renglón)", beneficios, { beneficios = it }, multilinea = true) }
            add { RenglonInterruptor("A la venta", plan.activo, { plan = plan.copy(activo = it) }) }
        }.toTypedArray())
        Bloque("Servicios que cubre", "", *servicios.filter { it.activo }.map { s -> @Composable {
            RenglonInterruptor(s.nombre, s.id in plan.servicios, { v -> plan = plan.copy(servicios = if (v) plan.servicios + s.id else plan.servicios - s.id) })
        } }.toTypedArray())
    }
}

@Composable
fun ColumnScope.NuevaSocia(planes: List<PlanMembresia>, nombreInicial: String = "", telefonoInicial: String = "", emailInicial: String = "", listo: () -> Unit, cerrar: () -> Unit) {
    val estado = LocalEstado.current
    var planId by remember { mutableStateOf(planes.firstOrNull()?.id.orEmpty()) }
    var nombre by remember { mutableStateOf(nombreInicial) }
    var telefono by remember { mutableStateOf(telefonoInicial) }
    var email by remember { mutableStateOf(emailInicial) }
    var nota by remember { mutableStateOf("") }
    val contactos = rememberContactos { n, t, e -> nombre = n; telefono = t; email = e }
    @Serializable data class Cuerpo(val planId: String, val nombre: String, val telefono: String, val email: String, val nota: String)
    HojaEdicion("Sumar socia", cerrar, guardar = "Sumar", habilitado = planId.isNotEmpty() && nombre.length >= 2, accion = {
        val ok = estado.intentar("Sumamos a $nombre") { estado.api.hacer("POST", "/v1/panel/membresias/socios", Cuerpo(planId, nombre, telefono, email, nota)) }
        if (ok) listo()
        ok
    }) {
        Bloque("", "",
            { Renglon("Plan") { SelectorMenu(planes.map { it.id to it.nombre }, planId, { planId = it }) } },
            { RenglonCampo("Nombre", nombre, { nombre = it }) },
            { RenglonCampo("Teléfono", telefono, { telefono = it }, teclado = KeyboardType.Phone) },
            { RenglonCampo("Email", email, { email = it }, teclado = KeyboardType.Email) },
            { RenglonCampo("Nota", nota, { nota = it }) })
        Row(Modifier.clickable { contactos() }.padding(vertical = 8.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
            Icono("person.crop.circle", 16f, C.mentaTexto); Texto("Traer de Contactos", Tipo.lato(16f, Peso.Bold), C.mentaTexto)
        }
    }
}

@Composable
fun ColumnScope.AccionesSocia(socio: Socio, listo: () -> Unit, cerrar: () -> Unit) {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    var ajustar by remember { mutableIntStateOf(1) }
    var motivo by remember { mutableStateOf("") }
    fun patch(c: Map<String, JsonPrimitive>, ok: String) = alcance.launch {
        if (estado.intentar(ok) { estado.api.hacer("PATCH", "/v1/panel/membresias/socios/${socio.id}", c) }) { listo(); cerrar() }
    }
    BarraHoja("Socia", cerrar, textoCancelar = "", accion = "Listo") { cerrar() }
    Column(Modifier.weight(1f).background(C.fondo).verticalScrollSimple().padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Texto(socio.cliente, Tipo.lato(22f, Peso.Black), C.tinta)
            Texto("${socio.plan} · quedan ${socio.quedan} de ${socio.visitasDelPlan} · vence ${diaTexto(socio.hasta)}", Tipo.lato(14f), C.sub)
        }
        Bloque("Ajustar visitas", "",
            { Numerito(if (ajustar >= 0) "Sumar" else "Restar", ajustar, { ajustar = it }, -20..20) { if (it > 0) "+$it" else "$it" } },
            { RenglonCampo("Motivo", motivo, { motivo = it }, "Le regalamos una, se descontó mal…") },
            { GuardarEnBloque("Ajustar", ajustar != 0) { patch(mapOf("ajustar" to JsonPrimitive(ajustar), "motivo" to JsonPrimitive(motivo)), if (ajustar > 0) "Sumamos las visitas" else "Restamos las visitas") } })
        BotonContorno("Renovar desde hoy", { patch(mapOf("renovar" to JsonPrimitive(true)), "Renovamos la membresía desde hoy") }, alto = 48.dp, tam = 16f, simbolo = "arrow.clockwise")
        if (socio.estado == "activa") BotonContornoRojo("Cancelar la membresía", "xmark.circle") { patch(mapOf("estado" to JsonPrimitive("cancelada")), "Cancelamos la membresía") }
        else BotonContorno("Reactivar", { patch(mapOf("estado" to JsonPrimitive("activa")), "Reactivamos la membresía") }, alto = 48.dp, tam = 16f, simbolo = "checkmark.circle")
    }
}

/** BotonContorno con el texto en rojo (role: .destructive). */
@Composable
fun BotonContornoRojo(texto: String, simbolo: String, alto: androidx.compose.ui.unit.Dp = 48.dp, alTocar: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().heightIn(min = alto).clip(CircleShape).contorno(999.dp).clickable(onClick = alTocar),
        horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically,
    ) { Icono(simbolo, 16f, C.error); Spacer(Modifier.width(8.dp)); Texto(texto, Tipo.lato(16f, Peso.Bold), C.error) }
}

@Composable
private fun Modifier.verticalScrollSimple(): Modifier = this.then(Modifier.verticalScroll(androidx.compose.foundation.rememberScrollState()))

// MARK: - Cupones

@Composable
fun Cupones() {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    var cupones by remember { mutableStateOf<List<Cupon>>(emptyList()) }
    var cargando by remember { mutableStateOf(true) }
    var editando by remember { mutableStateOf<Cupon?>(null) }
    @Serializable data class Cs(val cupones: List<Cupon> = emptyList())
    suspend fun cargar() { estado.api.opcional<Cs>("/v1/panel/cupones")?.let { cupones = it.cupones }; cargando = false }
    LaunchedEffect(estado.version) { cargar() }
    @Serializable data class Activo(val activo: Boolean)
    val nuevo = { Cupon("", "", "porcentaje", 10, usos = 0, activo = true) }
    PantallaMas("Cupones", "${cupones.count { it.activo }} activos", acciones = listOf(AccionCabecera("plus", "Crear cupón") { editando = nuevo() }), cargando = cargando, recargar = { cargar() }) {
        val l = cupones.filter { it.referidoId == null }
        if (l.isEmpty()) VacioQueEnsena("Sin cupones", "Creá tu primer cupón", "Un código que tus clientas escriben al reservar para tener un descuento.") {
            BotonMenta("Crear un cupón", { editando = nuevo() }, alto = 52.dp, conSombra = false)
        } else Bloque("", "", *l.map { c -> @Composable {
            Box(Modifier.clickable { editando = c }) {
                Renglon(c.codigo, "${c.texto} · ${c.usos}${c.maxUsos?.let { " de $it" } ?: ""} usos${c.hasta?.let { " · hasta el ${Formato.dia(it)}" } ?: ""}") {
                    Llave(c.activo) { v -> alcance.launch { estado.intentar(if (v) "Activamos el cupón" else "Pausamos el cupón") { estado.api.hacer("PATCH", "/v1/panel/cupones/${c.id}", Activo(v)) }; cargar() } }
                }
            }
        } }.toTypedArray())
    }
    Hoja(editando != null, { editando = null }) { editando?.let { EditarCupon(it, listo = { alcance.launch { cargar() } }, cerrar = { editando = null }) } }
}

@Composable
fun ColumnScope.EditarCupon(inicial: Cupon, listo: () -> Unit, cerrar: () -> Unit) {
    val estado = LocalEstado.current
    val hoy = Formato.enArgentina(Instant.now()).toLocalDate()
    var c by remember { mutableStateOf(inicial) }
    var conMinimo by remember { mutableStateOf(inicial.minimo != null) }
    var minimo by remember { mutableIntStateOf(inicial.minimo ?: 0) }
    var conMax by remember { mutableStateOf(inicial.maxUsos != null) }
    var maxUsos by remember { mutableIntStateOf(inicial.maxUsos ?: 50) }
    var conHasta by remember { mutableStateOf(inicial.hasta != null) }
    var hasta by remember { mutableStateOf(inicial.hasta?.let { Formato.enArgentina(it).toLocalDate() } ?: hoy.plusDays(30)) }
    @Serializable data class Cuerpo(val codigo: String?, val tipo: String, val valor: Int, val minimo: Int?, val maxUsos: Int?, val hasta: FechaApi?, val activo: Boolean)
    HojaEdicion(if (c.id.isEmpty()) "Cupón nuevo" else c.codigo, cerrar, habilitado = c.codigo.length >= 3,
        borrar = if (c.id.isEmpty()) null else ({ estado.intentar("Sacamos el cupón") { estado.api.hacer("DELETE", "/v1/panel/cupones/${c.id}") }; listo() }),
        accion = {
            val fin = hasta.atTime(23, 59).atZone(ZoneId.of("America/Argentina/Buenos_Aires")).toInstant()
            val cuerpo = Cuerpo(if (c.id.isEmpty()) c.codigo else null, c.tipo, c.valor, if (conMinimo) minimo else null, if (conMax) maxUsos else null, if (conHasta) fin else null, c.activo)
            val ok = estado.intentar(if (c.id.isEmpty()) "Creamos el cupón" else "Guardamos el cupón") {
                if (c.id.isEmpty()) estado.api.hacer("POST", "/v1/panel/cupones", cuerpo) else estado.api.hacer("PATCH", "/v1/panel/cupones/${c.id}", cuerpo)
            }
            if (ok) listo()
            ok
        }) {
        Bloque("", "", *buildList<@Composable () -> Unit> {
            if (c.id.isEmpty()) add { RenglonCampo("Código", c.codigo, { t -> c = c.copy(codigo = t.uppercase().filter { it.isLetterOrDigit() }) }, "VERANO10") }
            add { Renglon("Tipo") { SegmentadoChico(listOf("porcentaje" to "Porcentaje", "fijo" to "Monto fijo"), c.tipo) { c = c.copy(tipo = it) } } }
            if (c.tipo == "porcentaje") add { Numerito("Descuento", c.valor, { c = c.copy(valor = it) }, 1..100, paso = 5) { "$it %" } }
            else add { RenglonPesos("Descuento", c.valor) { c = c.copy(valor = it) } }
            add { RenglonInterruptor("Compra mínima", conMinimo, { conMinimo = it }) }
            if (conMinimo) add { RenglonPesos("Mínimo", minimo) { minimo = it } }
            add { RenglonInterruptor("Límite de usos", conMax, { conMax = it }) }
            if (conMax) add { Numerito("Usos", maxUsos, { maxUsos = it }, 1..9999) }
            add { RenglonInterruptor("Vence", conHasta, { conHasta = it }) }
            if (conHasta) add { RenglonFecha("Hasta", hasta, { hasta = it }, minimo = hoy) }
            add { RenglonInterruptor("Activo", c.activo, { c = c.copy(activo = it) }) }
        }.toTypedArray())
    }
}

// MARK: - Referidos

@Composable
fun Referidos() {
    val estado = LocalEstado.current
    @Serializable data class Ajustes(val activo: Boolean = false, val tipo: String? = "porcentaje", val valor: Int? = 10, val tope: Int? = null)
    @Serializable data class Referido(val id: String, val codigo: String = "", val cliente: String = "", val usos: Int = 0, val creadoEn: FechaApi = Instant.EPOCH)
    @Serializable data class Rs(val referidos: List<Referido> = emptyList())
    var a by remember { mutableStateOf(Ajustes()) }
    var lista by remember { mutableStateOf<List<Referido>>(emptyList()) }
    var cargando by remember { mutableStateOf(true) }
    suspend fun cargar() {
        estado.api.opcional<Ajustes>("/v1/panel/referidos/config")?.let { a = it }
        lista = estado.api.opcional<Rs>("/v1/panel/referidos")?.referidos ?: emptyList()
        cargando = false
    }
    LaunchedEffect(Unit) { cargar() }
    PantallaMas("Referidos", if (a.activo) "Activo" else "Apagado", cargando = cargando, recargar = { cargar() }) {
        Bloque("", "Cada clienta recibe un código para compartir. Quien reserva con él tiene el descuento, y ella también en su próximo turno.", *buildList<@Composable () -> Unit> {
            add { RenglonInterruptor("Programa de referidos", a.activo, { a = a.copy(activo = it) }) }
            if (a.activo) {
                add { Renglon("Premio") { SegmentadoChico(listOf("porcentaje" to "Porcentaje", "fijo" to "Monto fijo"), a.tipo ?: "porcentaje") { a = a.copy(tipo = it) } } }
                if (a.tipo == "fijo") add { RenglonPesos("Descuento", a.valor ?: 0) { a = a.copy(valor = it) } }
                else add { Numerito("Descuento", a.valor ?: 10, { a = a.copy(valor = it) }, 1..100, paso = 5) { "$it %" } }
                add { Numerito("Usos por código", a.tope ?: 0, { a = a.copy(tope = if (it == 0) null else it) }, 0..100) { if (it == 0) "Sin tope" else "$it" } }
            }
            add { GuardarEnBloque { estado.intentar("Guardamos los referidos") { estado.api.hacer("PUT", "/v1/panel/referidos/config", a) } } }
        }.toTypedArray())
        if (lista.isNotEmpty()) Bloque("Códigos", "", *lista.map { r -> @Composable {
            Renglon(r.cliente, r.codigo) { Texto(Formato.cantidad(r.usos, "uso", "usos"), Tipo.lato(14f, Peso.Bold), C.tinta) }
        } }.toTypedArray())
    }
}

// MARK: - Penalidades y ausencias

@Composable
fun Penalidades() {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    val alcance = rememberCoroutineScope()
    @Serializable data class AjustesAusencias(val umbralAusencias: Int = 2, val diasPenalidad: Int = 30, val marcarAusentes: Boolean = false)
    @Serializable data class Ausencias(val total: Int = 0, val ultimos30: Int = 0, val penalidadesVigentes: Int = 0, val reincidentes: Int = 0)
    @Serializable data class Ps(val penalidades: List<Penalidad> = emptyList())
    @Serializable data class Marcados(val marcados: Int = 0)
    var lista by remember { mutableStateOf<List<Penalidad>>(emptyList()) }
    var aj by remember { mutableStateOf(AjustesAusencias()) }
    var aus by remember { mutableStateOf<Ausencias?>(null) }
    var todas by remember { mutableStateOf(false) }
    var cargando by remember { mutableStateOf(true) }
    suspend fun cargar() {
        estado.api.opcional<Ps>("/v1/panel/penalidades", if (todas) listOf("todas" to "1") else emptyList())?.let { lista = it.penalidades }
        estado.api.opcional<AjustesAusencias>("/v1/panel/ausencias/ajustes")?.let { aj = it }
        aus = estado.api.opcional("/v1/panel/ausencias")
        cargando = false
    }
    LaunchedEffect(estado.version, todas) { cargar() }
    fun marcar() = alcance.launch {
        try {
            val r: Marcados = estado.api.enviar("POST", "/v1/panel/ausencias/marcar", emptyMap<String, String>())
            estado.avisar(if (r.marcados == 0) "No había turnos para marcar" else "Marcamos ${Formato.cantidad(r.marcados, "ausente", "ausentes")}")
            estado.version += 1
        } catch (e: Exception) { estado.avisar(e) }
    }
    PantallaMas("Penalidades", aus?.let { "${it.ultimos30} ausencias en 30 días · ${it.penalidadesVigentes} vigentes" } ?: "", cargando = cargando, recargar = { cargar() }) {
        Bloque("Automático", "Si alguien falta sin avisar esa cantidad de veces, no puede reservar por la web durante esos días.",
            { Numerito("Penalizar desde", aj.umbralAusencias, { aj = aj.copy(umbralAusencias = it) }, 0..10) { if (it == 0) "Nunca" else "$it faltas" } },
            { Numerito("Durante", aj.diasPenalidad, { aj = aj.copy(diasPenalidad = it) }, 1..365, paso = 5) { "$it días" } },
            { RenglonInterruptor("Marcar ausentes solos", aj.marcarAusentes, { aj = aj.copy(marcarAusentes = it) }, "Los turnos sin marcar pasan a «No vino» al terminar el día") },
            { GuardarEnBloque { estado.intentar("Guardamos los ajustes") { estado.api.hacer("PUT", "/v1/panel/ausencias/ajustes", aj) } } })
        BotonContorno("Marcar ausentes los turnos pasados sin marcar", { marcar() }, alto = 48.dp, tam = 15f, simbolo = "person.slash")
        Box(Modifier.padding(horizontal = 4.dp)) { Interruptor("Ver también las levantadas", todas, { todas = it }) }
        if (lista.isEmpty()) Texto("No hay penalidades ${if (todas) "" else "vigentes"}.", Tipo.lato(14f), C.sub)
        else Bloque("", "", *lista.map { p -> @Composable {
            Box(Modifier.clickable { nav.pestana = Pestana.clientes; nav.pilaClientes.add(DestinoClientes.ficha(p.clienteId)) }) {
                Renglon(p.cliente, "${if (p.vigente) "Hasta el ${Formato.dia(p.hasta)}" else "Levantada"} · ${Formato.cantidad(p.ausencias, "ausencia", "ausencias")}") {
                    if (p.vigente) BotonTexto("Levantar", { alcance.launch { estado.intentar("Levantamos la penalidad") { estado.api.hacer("POST", "/v1/panel/penalidades/${p.id}/levantar") }; cargar() } })
                }
            }
        } }.toTypedArray())
    }
}

// MARK: - Etiquetas

val coloresEtiqueta = listOf("#3ccc90", "#183c44", "#e8a317", "#d9534f", "#7a5cfa", "#2f80ed", "#ec4899", "#8b5e34")

@Composable
fun Etiquetas() {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    var lista by remember { mutableStateOf<List<EtiquetaConUso>>(emptyList()) }
    var nombre by remember { mutableStateOf("") }
    var color by remember { mutableStateOf("#3ccc90") }
    var cargando by remember { mutableStateOf(true) }
    var editando by remember { mutableStateOf<EtiquetaConUso?>(null) }
    @Serializable data class Es(val etiquetas: List<EtiquetaConUso> = emptyList())
    @Serializable data class Cuerpo(val nombre: String, val color: String)
    suspend fun cargar() { estado.api.opcional<Es>("/v1/panel/etiquetas")?.let { lista = it.etiquetas }; cargando = false }
    LaunchedEffect(Unit) { cargar() }
    PantallaMas("Etiquetas", Formato.cantidad(lista.size, "etiqueta", "etiquetas"), cargando = cargando, recargar = { cargar() }) {
        Bloque("", "Sirven para separar clientas (VIP, nueva, alérgica) y filtrarlas en Clientes.", *(lista.map { e -> @Composable {
            Box(Modifier.clickable { editando = e }) {
                Renglon(e.nombre, Formato.cantidad(e.clientes, "clienta", "clientas")) { Box(Modifier.size(18.dp).clip(CircleShape).background(hexTexto(e.color) ?: C.chip)) }
            }
        } } + listOf(@Composable {
            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Box(Modifier.weight(1f)) {
                        if (nombre.isEmpty()) Texto("Nueva etiqueta", Tipo.lato(16f), C.sub.copy(alpha = 0.7f))
                        val tinta = C.tinta
                        BasicTextField(nombre, { nombre = it }, Modifier.fillMaxWidth(), singleLine = true, textStyle = Tipo.lato(16f).copy(color = tinta), cursorBrush = SolidColor(C.mentaTexto))
                    }
                    Box(Modifier.atenuado(nombre.isBlank())) {
                        BotonMentaChico("Crear", { if (nombre.isNotBlank()) alcance.launch { estado.intentar { estado.api.hacer("POST", "/v1/panel/etiquetas", Cuerpo(nombre, color)) }; nombre = ""; cargar() } })
                    }
                }
                SelectorColor(color) { color = it }
            }
        })).toTypedArray())
    }
    Hoja(editando != null, { editando = null }, alturas = listOf(Altura.media)) { editando?.let { EditarEtiqueta(it, listo = { alcance.launch { cargar() } }, cerrar = { editando = null }) } }
}

@Composable
fun SelectorColor(color: String, alElegir: (String) -> Unit) {
    Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
        coloresEtiqueta.forEach { c ->
            Box(Modifier.size(28.dp).clip(CircleShape).background(hexTexto(c) ?: Color.Gray)
                .then(if (color == c) Modifier.border(3.dp, C.tinta, CircleShape) else Modifier)
                .clickable { alElegir(c) }.semantics { contentDescription = "Color $c" })
        }
    }
}

@Composable
fun ColumnScope.EditarEtiqueta(inicial: EtiquetaConUso, listo: () -> Unit, cerrar: () -> Unit) {
    val estado = LocalEstado.current
    var e by remember { mutableStateOf(inicial) }
    @Serializable data class Cuerpo(val nombre: String, val color: String)
    HojaEdicion("Etiqueta", cerrar, habilitado = e.nombre.isNotEmpty(),
        borrar = { estado.intentar { estado.api.hacer("DELETE", "/v1/panel/etiquetas/${e.id}") }; listo() },
        accion = {
            val ok = estado.intentar { estado.api.hacer("PATCH", "/v1/panel/etiquetas/${e.id}", Cuerpo(e.nombre, e.color)) }
            if (ok) listo()
            ok
        }) {
        Bloque("", "", { RenglonCampo("Nombre", e.nombre, { e = e.copy(nombre = it) }) }, { Box(Modifier.padding(16.dp)) { SelectorColor(e.color) { e = e.copy(color = it) } } })
    }
}

// MARK: - Suscripción

/** Suscripción: se adapta a dónde se contrató. Desde el teléfono se paga con Google Play (Play Billing, se
 *  administra en Play Store › Suscripciones); desde el iPhone, con la App Store; desde la web, con Mercado Pago.
 *  Sin suscripción, los planes de Google Play con su precio, la renovación automática y los términos. */
@Composable
fun SuscripcionVista() {
    @Serializable data class PlanVenta(val clave: String, val nombre: String = "", val precio: Int = 0)
    @Serializable data class Sus(val id: String, val plan: String = "", val estado: String = "", val monto: Int = 0, val proximoCobro: FechaApi? = null, val ultimoCobro: FechaApi? = null,
        val graciaHasta: FechaApi? = null, val canceladaEn: FechaApi? = null, val origen: String? = null)
    @Serializable data class EstadoS(val bloqueado: Boolean = false, val pruebaHasta: FechaApi? = null, val diasDePrueba: Int = 0, val conSuscripcion: Boolean = false, val plan: String = "",
        val nombrePlan: String = "", val catalogo: List<PlanVenta> = emptyList(), val suscripcion: Sus? = null)
    val estado = LocalEstado.current
    val contexto = LocalContext.current
    val alcance = rememberCoroutineScope()
    var s by remember { mutableStateOf<EstadoS?>(null) }
    var error by remember { mutableStateOf("") }
    var productos by remember { mutableStateOf<List<Compras.Plan>>(emptyList()) }
    var cargoTienda by remember { mutableStateOf(false) }
    var comprando by remember { mutableStateOf<String?>(null) }
    var restaurando by remember { mutableStateOf(false) }
    var elegido by remember { mutableStateOf("") }

    /** Qué trae cada plan (las mismas filas que la web). */
    val filas = listOf(
        Triple("Turnos online y agenda", "Sí", "Sí"), Triple("Recordatorios por mail", "Sí", "Sí"), Triple("Profesionales", "Hasta 2", "Sin límite"),
        Triple("Señas y pagos con Mercado Pago", "No", "Sí"), Triple("Membresías, cupones y referidos", "No", "Sí"), Triple("Reportes", "Básicos", "Completos"),
        Triple("Dominio propio y plantillas", "No", "Sí"), Triple("Integraciones con IA", "No", "Sí"), Triple("Sucursales", "No", "Sí"),
    )
    val descripcion = mapOf("inicial" to "Para empezar: turnos online, agenda y recordatorios, hasta 2 profesionales.", "completo" to "Todo Tengo Libre: señas, membresías, reportes, IA y sin límite de equipo.")
    val vigente = s?.suscripcion?.takeIf { it.estado in listOf("autorizada", "pausada") }

    suspend fun cargarTienda() { cargoTienda = false; productos = Compras.planes(contexto); cargoTienda = true }
    suspend fun cargar() {
        try { s = estado.api.get("/v1/panel/suscripcion"); error = "" } catch (e: ErrorApi) { error = e.detalle } catch (e: Exception) { error = "Probá de nuevo." }
        val v = s?.suscripcion?.takeIf { it.estado in listOf("autorizada", "pausada") }
        if (v == null && estado.esDuena && !cargoTienda) cargarTienda()
    }
    LaunchedEffect(estado.version) { cargar() }
    fun administrar() = estado.abrirLink("https://play.google.com/store/account/subscriptions?package=${contexto.packageName}")
    fun comprar(p: Compras.Plan) = alcance.launch {
        val act = contexto as? Activity ?: return@launch
        comprando = p.id
        try {
            when (Compras.comprar(act, p, estado)) {
                Compras.Resultado.listo -> { Haptico.exito(); estado.avisar("¡Listo! Ya tenés el plan ${p.nombre}", "Gracias por sumarte a Tengo Libre."); cargar() }
                Compras.Resultado.pendiente -> estado.avisar("La compra quedó pendiente", "Cuando Google Play la apruebe, se activa sola.")
                Compras.Resultado.cancelado -> {}
            }
        } catch (e: Exception) { estado.avisar(e) }
        comprando = null
    }

    @Composable fun botonRestaurar() {
        Box(Modifier.fillMaxWidth().alpha(if (restaurando) 0.5f else 1f), contentAlignment = Alignment.Center) {
            BotonTexto(if (restaurando) "Buscando tus compras…" else "Restaurar compras", {
                if (!restaurando) alcance.launch {
                    restaurando = true
                    try {
                        val n = Compras.restaurar(contexto, estado)
                        estado.avisar(if (n == 0) "No encontramos compras para restaurar" else "Listo, recuperamos tu suscripción", ok = n > 0)
                        cargar()
                    } catch (e: Exception) { estado.avisar(e) }
                    restaurando = false
                }
            })
        }
    }
    @Composable fun fila(a: String, b: String) {
        Row(Modifier.fillMaxWidth()) {
            Texto(a, Tipo.lato(14f), C.sub); Spacer(Modifier.weight(1f).widthIn(min = 12.dp))
            Texto(b, Tipo.lato(14f, Peso.Bold), C.tinta, alinear = androidx.compose.ui.text.style.TextAlign.End)
        }
    }

    val subtitulo = s?.let { x -> if (vigente != null) "Plan ${x.nombrePlan} · ${origenTexto(vigente.origen)}" else if (x.bloqueado) "La prueba terminó" else "Prueba gratis · quedan ${x.diasDePrueba} días" } ?: ""
    PantallaMas("Suscripción", subtitulo, cargando = s == null && error.isEmpty(), error = error, recargar = { cargar() }) {
        val x = s ?: return@PantallaMas
        val v = vigente
        if (v != null) when (v.origen) {
            "google" -> {
                Bloque("Tu plan", "", *buildList<@Composable () -> Unit> {
                    add { Renglon("Plan ${x.nombrePlan}", if (v.estado == "pausada") "Google Play no pudo cobrar" else "Se paga con Google Play", "creditcard") }
                    v.proximoCobro?.let { p -> add { Renglon(if (v.canceladaEn == null) "Se renueva" else "Termina", Formato.fechaLarga(p), "calendar") } }
                    if (v.estado == "pausada") v.graciaHasta?.let { g -> add { Renglon("Revisá tu medio de pago en Google Play", "Todo sigue andando hasta el ${Formato.dia(g)}", "exclamationmark.triangle") } }
                }.toTypedArray())
                if (estado.esDuena) { BotonContorno("Cambiar o cancelar en Google Play", { administrar() }, alto = 52.dp, tam = 16f, simbolo = "gearshape"); botonRestaurar() }
                Texto("Se renueva sola cada mes. Para cambiar de plan o cancelar: Play Store › tu perfil › Pagos y suscripciones › Suscripciones › Tengo Libre, o el botón de arriba.", Tipo.lato(13f), C.sub)
            }
            "apple" -> {
                Bloque("Tu plan", "", *buildList<@Composable () -> Unit> {
                    add { Renglon("Plan ${x.nombrePlan}", "Se paga con la App Store, desde un iPhone", "creditcard") }
                    v.proximoCobro?.let { p -> add { Renglon(if (v.canceladaEn == null) "Se renueva" else "Termina", Formato.fechaLarga(p), "calendar") } }
                }.toTypedArray())
                Texto("La contrataste desde un iPhone: el plan y la cancelación se manejan en Ajustes › tu nombre › Suscripciones, en ese iPhone.", Tipo.lato(14f), C.sub)
            }
            else -> {
                Bloque("Tu plan", "", *buildList<@Composable () -> Unit> {
                    add { Renglon("Plan ${x.nombrePlan}", "Se paga con Mercado Pago, desde la web", "creditcard") }
                    if (v.estado == "autorizada") v.proximoCobro?.let { p -> add { Renglon("Próximo cobro", "${Formato.fechaLarga(p)} · ${Formato.pesos(v.monto)}", "calendar") } }
                    if (v.estado == "pausada") add { Renglon(if (v.canceladaEn == null) "Mercado Pago no pudo cobrar" else "Cancelada", v.graciaHasta?.let { "Todo sigue andando hasta el ${Formato.dia(it)}" } ?: "", "exclamationmark.triangle") }
                }.toTypedArray())
                Texto("La contrataste desde la web: el plan, la tarjeta y la cancelación se manejan desde el panel en la compu, en Suscripción.", Tipo.lato(14f), C.sub)
            }
        } else {
            Column(Modifier.fillMaxWidth().tarjeta().padding(16.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                Texto(if (x.bloqueado) "Tu prueba terminó" else "Estás en la prueba gratis", Tipo.lato(22f, Peso.Black), C.tinta)
                Texto(if (x.bloqueado) "Podés ver todo, pero para cargar o cambiar turnos hace falta un plan. Tu página sigue tomando reservas."
                    else "Hasta el ${x.pruebaHasta?.let { Formato.fechaLarga(it) }.orEmpty()} tenés todo el plan Completo.", Tipo.lato(15f), C.sub)
                if (x.suscripcion?.estado == "cancelada" && x.suscripcion.origen == "google") Texto("Tu suscripción de Google Play venció o se canceló.", Tipo.lato(14f, Peso.Bold), C.ausenteTinta)
            }
            if (!estado.esDuena) Texto("La suscripción la elige la dueña o el dueño del negocio.", Tipo.lato(14f), C.sub)
            else if (!cargoTienda) { repeat(2) { Esqueleto(96.dp, 18.dp) }; Esqueleto(260.dp, 20.dp) }
            else if (productos.isEmpty()) VacioQueEnsena("Pronto", "Los planes no cargaron", "Google Play no respondió. Probá de nuevo en un rato.") {
                BotonContorno("Reintentar", { alcance.launch { cargarTienda() } }, alto = 48.dp, tam = 15f)
            } else {
                val p = productos.firstOrNull { it.id == elegido } ?: productos.firstOrNull { Compras.productos[it.id] == "completo" } ?: productos.first()
                Box(Modifier.padding(horizontal = 4.dp)) { TextoRotulo("1 · ELEGÍ TU PLAN") }
                productos.forEach { q ->
                    val clave = Compras.productos[q.id].orEmpty()
                    val on = p.id == q.id
                    Row(
                        Modifier.fillMaxWidth()
                            .sombra(RoundedCornerShape(18.dp), if (on) hex(0x17805A, 0.25f) else Color.Transparent, 14.dp, y = 10.dp)
                            .clip(RoundedCornerShape(18.dp)).background(C.tarjeta).border(2.dp, if (on) C.menta else C.linea, RoundedCornerShape(18.dp))
                            .clickable { elegido = q.id; Haptico.seleccion() }.padding(16.dp),
                        verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp),
                    ) {
                        Icono(if (on) "largecircle.fill.circle" else "circle", 24f, if (on) C.mentaTexto else C.sub)
                        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                Texto(q.nombre.ifEmpty { Formato.mayuscula(clave) }, Tipo.lato(19f, Peso.Black), C.tinta, tracking = (-0.4).sp)
                                if (clave == "completo") Box(Modifier.height(20.dp).clip(CircleShape).background(C.menta).padding(horizontal = 7.dp), contentAlignment = Alignment.Center) {
                                    Texto("RECOMENDADO", Tipo.lato(10f, Peso.Black), C.petroleoHondo, tracking = 0.8.sp)
                                }
                            }
                            Texto(descripcion[clave] ?: q.descripcion, Tipo.lato(13f), C.sub)
                        }
                        Column(horizontalAlignment = Alignment.End) {
                            Texto(q.precio, Tipo.lato(20f, Peso.Black, numeros = true), C.tinta)
                            Texto("por mes", Tipo.lato(12f), C.sub)
                        }
                    }
                }
                val col = if (Compras.productos[p.id] == "completo") 2 else 1
                Column(Modifier.fillMaxWidth().clip(RoundedCornerShape(18.dp)).background(C.fondo).padding(16.dp), verticalArrangement = Arrangement.spacedBy(7.dp)) {
                    TextoRotulo("QUÉ INCLUYE")
                    filas.forEach { f ->
                        val valor = if (col == 1) f.second else f.third
                        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            Icono(if (valor == "No") "minus.circle" else "checkmark.circle.fill", 17f, if (valor == "No") C.linea else C.mentaTexto)
                            Texto(f.first, Tipo.lato(14f).copy(textDecoration = if (valor == "No") TextDecoration.LineThrough else null), if (valor == "No") C.sub else C.tinta, Modifier.weight(1f))
                            if (valor != "Sí" && valor != "No") Texto(valor, Tipo.lato(13f, Peso.Bold), C.sub)
                        }
                    }
                }
                Box(Modifier.padding(start = 4.dp, end = 4.dp, top = 6.dp)) { TextoRotulo("2 · TU SUSCRIPCIÓN") }
                // El «carrito»: el plan, cuándo y cuánto se cobra, el medio de pago y el total de hoy.
                val nombre = p.nombre.ifEmpty { Formato.mayuscula(Compras.productos[p.id].orEmpty()) }
                val dia = Formato.enArgentina(Instant.now()).dayOfMonth
                val quedanPrueba = x.pruebaHasta?.isAfter(Instant.now()) ?: false
                Column(Modifier.fillMaxWidth().sombra(RoundedCornerShape(22.dp), hex(0x0F2328, 0.12f), 20.dp, y = 14.dp).clip(RoundedCornerShape(22.dp)).background(C.tarjeta)
                    .border(1.dp, C.linea, RoundedCornerShape(22.dp)).padding(18.dp)) {
                    Row(Modifier.padding(bottom = 14.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                        Box(Modifier.size(40.dp).clip(RoundedCornerShape(12.dp)).background(C.menta), contentAlignment = Alignment.Center) { Icono("crown.fill", 18f, C.petroleoHondo) }
                        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                            Texto("Plan $nombre", Tipo.lato(16f, Peso.Bold), C.tinta)
                            Texto("Mensual · se renueva sola · cancelás cuando quieras", Tipo.lato(12f), C.sub)
                        }
                        Texto(p.precio, Tipo.lato(17f, Peso.Black, numeros = true), C.tinta)
                    }
                    Troquel(vertical = false)
                    Column(Modifier.padding(vertical = 14.dp), verticalArrangement = Arrangement.spacedBy(9.dp)) {
                        fila("Primer cobro", "hoy, al confirmar")
                        fila("Después", "${p.precio} cada mes, el día $dia")
                        fila("Medio de pago", "tu cuenta de Google Play")
                        if (quedanPrueba) Row(Modifier.fillMaxWidth().clip(RoundedCornerShape(12.dp)).background(C.pagoFondo).padding(10.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            Icono("info.circle", 15f, C.pagoTinta)
                            Texto("Te quedan ${x.diasDePrueba} días de prueba. Si te suscribís desde el teléfono, Google Play cobra hoy.", Tipo.lato(12f), C.pagoTinta)
                        }
                    }
                    Troquel(vertical = false)
                    Row(Modifier.padding(vertical = 14.dp), verticalAlignment = Alignment.Bottom) {
                        Texto("Hoy pagás", Tipo.lato(16f, Peso.Bold), C.tinta, Modifier.weight(1f))
                        NumeroAnimado(p.precio, Tipo.lato(32f, Peso.Black, numeros = true).copy(letterSpacing = (-1).sp), C.tinta)
                    }
                    BotonMenta(if (comprando == p.id) "Abriendo Google Play…" else "Pagar ${p.precio} con Google Play", { comprar(p) }, alto = 56.dp, habilitado = comprando == null)
                }
                botonRestaurar()
                Column(Modifier.padding(horizontal = 4.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Texto("La suscripción se cobra a tu cuenta de Google Play al confirmar y se renueva sola cada mes, al mismo precio, salvo que la canceles antes del fin del período. Se administra y cancela en Play Store › Pagos y suscripciones › Suscripciones.", Tipo.lato(12f), C.sub)
                    Row(horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                        Texto("Términos de uso", Tipo.lato(12f, Peso.Bold), C.mentaTexto, Modifier.clickable { estado.abrirPestana("https://tengolibre.com.ar/terminos") })
                        Texto("Privacidad", Tipo.lato(12f, Peso.Bold), C.mentaTexto, Modifier.clickable { estado.abrirPestana("https://tengolibre.com.ar/privacidad") })
                    }
                }
            }
        }
    }
}

private fun origenTexto(o: String?) = when (o) { "google" -> "Google Play"; "apple" -> "App Store"; else -> "Mercado Pago" }
