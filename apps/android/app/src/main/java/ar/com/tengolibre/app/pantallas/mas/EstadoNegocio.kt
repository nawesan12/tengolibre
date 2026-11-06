package ar.com.tengolibre.app.pantallas.mas

import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.spring
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import ar.com.tengolibre.app.app.Estado
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.nucleo.*
import ar.com.tengolibre.app.pantallas.hoy.Arranque
import ar.com.tengolibre.app.pantallas.principal.LocalNavegacion
import ar.com.tengolibre.app.tema.*
import kotlinx.serialization.Serializable
import kotlin.math.roundToInt

// Clon de Pantallas/Mas/EstadoNegocio.swift.

/** Cómo está el negocio por dentro: qué está conectado, prendido o cargado. Lo usan Más (el estado de cada
 *  sección), Hoy («Descubrí» y la barra de avance) y «Potenciá tu negocio». */
class EstadoNegocio {
    data class Paso(val id: String, val simbolo: String, val titulo: String, val detalle: String, val listo: Boolean, val destino: DestinoMas?, val basico: Boolean)

    var cargado by mutableStateOf(false)
    var mp by mutableStateOf<ConexionMP?>(null)
    var cobro by mutableStateOf<ConfigCobro?>(null)
    var iaActiva by mutableStateOf<Boolean?>(null)
    var iaReservasMes by mutableIntStateOf(0)
    var clavesIA by mutableIntStateOf(0)
    var planes by mutableIntStateOf(0)
    var socias by mutableIntStateOf(0)
    var cupones by mutableIntStateOf(0)
    var recordatorios by mutableIntStateOf(0)
    var fotos by mutableIntStateOf(0)
    var logo by mutableStateOf(false)
    var dominio by mutableStateOf<String?>(null)
    var dominioListo by mutableStateOf(false)
    var arranque by mutableStateOf<Arranque?>(null)
    var whatsapp by mutableStateOf<Boolean?>(null)
    var widget by mutableStateOf(false)
    var avisos by mutableStateOf(false)
    var equipo by mutableIntStateOf(0)
    var servicios by mutableIntStateOf(0)
    private var negocio = ""

    @Serializable private data class IA(val activo: Boolean = false, val reservasMes: Int = 0)
    @Serializable private data class Mcp(val tokens: List<T> = emptyList()) { @Serializable data class T(val id: String) }
    @Serializable private data class Planes(val planes: List<PlanMembresia> = emptyList())
    @Serializable private data class Socios(val socios: List<Socio> = emptyList())
    @Serializable private data class Cups(val cupones: List<Cupon> = emptyList())
    @Serializable private data class Recs(val recordatorios: List<R> = emptyList()) { @Serializable data class R(val activo: Boolean = false) }
    @Serializable private data class Fotos(val fotos: List<FotoGaleria> = emptyList())
    @Serializable private data class Dom(val dominio: D? = null) { @Serializable data class D(val nombre: String = "", val estado: String = "") }
    @Serializable private data class WA(val disponible: Boolean = false, val confirmacion: Boolean = false)
    @Serializable private data class Pros(val profesionales: List<Profesional> = emptyList())
    @Serializable private data class Servs(val servicios: List<Servicio> = emptyList())

    /** Trae todo, de a un pedido por vez (como en iOS). */
    suspend fun cargar(estado: Estado) {
        val n = estado.negocio?.id ?: return
        if (n != negocio) { cargado = false; negocio = n }
        val api = estado.api
        val g = estado.gestiona; val mostrador = g || estado.rol == "recepcion"
        arranque = api.opcional("/v1/panel/arranque")
        if (g) {
            mp = api.opcional("/v1/panel/mercadopago")
            cobro = api.opcional("/v1/panel/cobros")
            api.opcional<IA>("/v1/panel/ia/publico")?.let { iaActiva = it.activo; iaReservasMes = it.reservasMes }
            clavesIA = api.opcional<Mcp>("/v1/panel/mcp")?.tokens?.size ?: clavesIA
            fotos = api.opcional<Fotos>("/v1/panel/mi-web/galeria")?.fotos?.size ?: fotos
            api.opcional<NegocioDatos>("/v1/panel/negocio")?.let { logo = !it.logo.isNullOrEmpty() }
            api.opcional<Dom>("/v1/panel/dominio")?.let { dominio = it.dominio?.nombre; dominioListo = it.dominio?.estado == "verificado" }
            api.opcional<WA>("/v1/panel/ajustes/whatsapp")?.let { whatsapp = if (it.disponible) it.confirmacion else null }
            equipo = api.opcional<Pros>("/v1/panel/profesionales")?.profesionales?.count { it.activo } ?: equipo
            servicios = api.opcional<Servs>("/v1/panel/servicios")?.servicios?.count { it.activo } ?: servicios
        }
        if (mostrador) {
            planes = api.opcional<Planes>("/v1/panel/membresias/planes")?.planes?.count { it.activo } ?: planes
            socias = api.opcional<Socios>("/v1/panel/membresias/socios", listOf("estado" to "activa"))?.socios?.size ?: socias
            cupones = api.opcional<Cups>("/v1/panel/cupones")?.cupones?.count { it.activo && it.referidoId == null } ?: cupones
            recordatorios = api.opcional<Recs>("/v1/panel/recordatorios")?.recordatorios?.count { it.activo } ?: recordatorios
        }
        avisos = hayAvisos()
        widget = hayWidget()
        cargado = true
    }

    // MARK: Lo que se ve en Más

    val textoCobros: Pair<String, Boolean> get() {
        val m = mp ?: return "" to false
        if (!m.conectada) return "Sin conectar · cobrá señas al reservar" to true
        if (m.reautorizar) return "Hay que volver a autorizar" to true
        return when (cobro?.modo) {
            "porcentaje" -> "Conectado · seña del ${cobro?.senaPorcentaje ?: 0} %" to false
            "fija" -> "Conectado · seña de ${Formato.pesos(cobro?.senaFija ?: 0)}" to false
            "total" -> "Conectado · cobra el total al reservar" to false
            else -> "Conectado · todavía no pide seña" to true
        }
    }

    val textoIA: Pair<String, Boolean> get() {
        val ia = iaActiva ?: return "Calendarios, Claude, ChatGPT" to false
        if (!ia && clavesIA == 0) return "Apagada · que te reserven desde ChatGPT" to true
        val p = mutableListOf<String>()
        if (ia) p.add(if (iaReservasMes > 0) "$iaReservasMes reservas por IA este mes" else "Reservas por IA prendidas")
        if (clavesIA > 0) p.add(Formato.cantidad(clavesIA, "asistente conectado", "asistentes conectados"))
        return p.joinToString(" · ") to false
    }

    val textoMembresias: Pair<String, Boolean> get() = if (cargado) (if (planes == 0) "Sin planes · plata fija todos los meses" to true
        else "${Formato.cantidad(socias, "socia activa", "socias activas")} · ${Formato.cantidad(planes, "plan", "planes")}" to false) else "Planes y socias" to false

    val textoCupones: Pair<String, Boolean> get() = if (cargado) (if (cupones == 0) "Ninguno activo" to false else Formato.cantidad(cupones, "activo", "activos") to false) else "" to false

    val textoRecordatorios: Pair<String, Boolean> get() = if (cargado) (if (recordatorios == 0) "Apagados · que vuelvan solas" to true else Formato.cantidad(recordatorios, "prendido", "prendidos") to false)
        else "Cumpleaños y «te extrañamos»" to false

    val textoWeb: Pair<String, Boolean> get() {
        if (!cargado) return "Datos, fotos, logo y portada" to false
        if (!logo) return "Falta el logo" to true
        if (fotos == 0) return "Sin fotos · sumá algunas" to true
        return "${Formato.cantidad(fotos, "foto", "fotos")} · logo cargado" to false
    }

    val textoDominio: Pair<String, Boolean> get() {
        val d = dominio ?: return (if (cargado) "Usá tu propia dirección" else "") to false
        return (if (dominioListo) "$d · funcionando" else "$d · falta configurar") to !dominioListo
    }

    val textoWhatsApp: Pair<String, Boolean> get() {
        val w = whatsapp ?: return "Confirmaciones y recordatorios" to false
        return if (w) "Pide confirmación el día antes" to false else "Apagado · que confirmen solas" to true
    }

    // MARK: Potenciá tu negocio

    fun pasos(estado: Estado): List<Paso> {
        val out = mutableListOf<Paso>()
        for (p in arranque?.pasos ?: emptyList()) out.add(Paso("a-${p.clave}", if (p.listo) "checkmark" else "circle.dashed", p.titulo,
            if (p.obligatorio) "Hace falta para que te reserven" else "Recomendado", p.listo, destinoArranque(p.clave), true))
        if (estado.gestiona) {
            out.add(Paso("mp", "creditcard", "Cobrá señas con Mercado Pago", "Menos ausencias, plata directo a tu cuenta", mp?.conectada == true && (cobro?.modo ?: "sin_cobro") != "sin_cobro", DestinoMas.cobros, false))
            out.add(Paso("ia", "sparkles", "Que te reserven desde ChatGPT y Claude", "Tus horarios a mano de los asistentes", iaActiva == true, DestinoMas.integraciones, false))
            out.add(Paso("mcp", "bubble.left.and.text.bubble.right", "Manejá tu agenda hablando", "Conectá Claude o ChatGPT a tu panel", clavesIA > 0, DestinoMas.integraciones, false))
            out.add(Paso("fotos", "photo.on.rectangle", "Logo y fotos en tu página", "Las páginas con fotos reciben más reservas", logo && fotos > 0, DestinoMas.miWeb, false))
            if (whatsapp != null) out.add(Paso("wa", "message", "Que confirmen por WhatsApp", "El día antes, solas", whatsapp == true, DestinoMas.whatsapp, false))
        }
        if (estado.gestiona || estado.rol == "recepcion") {
            out.add(Paso("memb", "ticket", "Vendé membresías", "Plata fija todos los meses", planes > 0, DestinoMas.membresias, false))
            out.add(Paso("rec", "envelope.badge", "Recordatorios automáticos", "Cumpleaños y «te extrañamos»", recordatorios > 0, DestinoMas.avisosClientes, false))
        }
        out.add(Paso("avisos", "bell.badge", "Avisos en este teléfono", "Cada reserva y cancelación al toque", avisos, DestinoMas.cuenta, false))
        out.add(Paso("widget", "apps.iphone", "El widget en tu pantalla de inicio", "Tu próximo turno sin abrir la app", widget, DestinoMas.recorrido, false))
        return out
    }

    fun avance(estado: Estado): Double {
        val p = pasos(estado)
        if (p.isEmpty()) return 1.0
        return p.count { it.listo }.toDouble() / p.size
    }

    private fun destinoArranque(clave: String): DestinoMas? = when (clave) {
        "servicios", "servicio" -> DestinoMas.servicios
        "profesionales", "profesional", "horario", "horarios", "equipo" -> DestinoMas.equipo
        "web", "mi_web", "miweb", "logo", "fotos", "datos", "direccion" -> DestinoMas.miWeb
        "cobros", "mercadopago", "pagos", "sena" -> DestinoMas.cobros
        "suscripcion" -> DestinoMas.suscripcion
        "plantilla" -> DestinoMas.plantillas
        else -> null
    }

    companion object {
        /** Si el permiso de avisos está dado y si hay un widget puesto: los fija MainActivity (hace falta el contexto). */
        var hayAvisos: () -> Boolean = { false }
        var hayWidget: () -> Boolean = { false }
    }
}

/** Más › «Potenciá tu negocio»: el anillo de avance, lo básico y lo que hace crecer, cada uno con su botón. */
@Composable
fun Potencia() {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    val vivo = estado.vivo
    val pasos = vivo.pasos(estado)
    val avance = vivo.avance(estado).toFloat()
    val llenado = remember { Animatable(0f) }
    LaunchedEffect(estado.version) { vivo.cargar(estado) }
    LaunchedEffect(avance) { llenado.animateTo(avance, spring(dampingRatio = 0.8f, stiffness = 60f)) }
    PantallaMas("Potenciá tu negocio", "${pasos.count { it.listo }} de ${pasos.size} listos", cargando = !vivo.cargado, recargar = { vivo.cargar(estado) }) {
        Row(Modifier.fillMaxWidth().tarjeta().padding(16.dp), horizontalArrangement = Arrangement.spacedBy(18.dp), verticalAlignment = Alignment.CenterVertically) {
            Box(Modifier.size(88.dp), contentAlignment = Alignment.Center) {
                val chip = C.chip; val menta = C.menta
                Canvas(Modifier.size(88.dp)) {
                    val g = 10.dp.toPx()
                    drawArc(chip, 0f, 360f, false, Offset(g / 2, g / 2), Size(size.width - g, size.height - g), style = Stroke(g))
                    drawArc(menta, -90f, 360f * llenado.value, false, Offset(g / 2, g / 2), Size(size.width - g, size.height - g), style = Stroke(g, cap = StrokeCap.Round))
                }
                Texto("${(llenado.value * 100).roundToInt()} %", Tipo.lato(20f, Peso.Black, numeros = true), C.tinta)
            }
            Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Texto(if (avance >= 1) "Estás usando todo" else "Sacale todo el jugo", Tipo.lato(19f, Peso.Black), C.tinta)
                Texto(if (avance >= 1) "Tu negocio tiene todo prendido. ¡Bien ahí!" else "Cada paso te trae más reservas o más plata.", Tipo.lato(14f), C.sub)
            }
        }
        listOf("Lo básico" to pasos.filter { it.basico }, "Para crecer" to pasos.filter { !it.basico }).forEach { (t, l) ->
            if (l.isNotEmpty()) Bloque(t, "", *l.map { p -> @Composable {
                Row(
                    Modifier.fillMaxWidth().heightIn(min = 60.dp).then(if (p.destino != null && !p.listo) Modifier.clickable { nav.pila.add(p.destino) } else Modifier)
                        .alpha(if (p.listo) 0.7f else 1f).padding(horizontal = 16.dp, vertical = 10.dp),
                    verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    Box(Modifier.size(34.dp).clip(CircleShape).background(if (p.listo) C.menta else C.mentaSuave), contentAlignment = Alignment.Center) {
                        Icono(if (p.listo) "checkmark" else p.simbolo, 16f, if (p.listo) C.petroleoHondo else C.mentaTexto)
                    }
                    Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                        Texto(p.titulo, Tipo.lato(15f, Peso.Bold).copy(textDecoration = if (p.listo) TextDecoration.LineThrough else null), C.tinta)
                        Texto(if (p.listo) "Listo" else p.detalle, Tipo.lato(13f), C.sub)
                    }
                    if (!p.listo && p.destino != null) Texto("Hacerlo", Tipo.lato(13f, Peso.Bold), C.mentaTexto)
                }
            } }.toTypedArray())
        }
    }
}

/** En Hoy, mientras falte algo: «Tu negocio al 60 % · Seguir». */
@Composable
fun BarraAvance() {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    val vivo = estado.vivo
    val avance = vivo.avance(estado)
    if (!vivo.cargado || avance >= 1 || !estado.gestiona) return
    val siguiente = vivo.pasos(estado).firstOrNull { !it.listo }
    val pct = (avance * 100).roundToInt()
    Column(
        Modifier.fillMaxWidth().tarjeta(18.dp).clickable { nav.pilaHoy.add(DestinoMas.primerosPasos) }.padding(14.dp)
            .semantics(mergeDescendants = true) { contentDescription = "Tu negocio al $pct por ciento. Seguir" },
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Texto("Tu negocio al $pct %", Tipo.lato(15f, Peso.Black), C.tinta, Modifier.weight(1f))
            Texto("Seguir", Tipo.lato(14f, Peso.Bold), C.mentaTexto)
            Icono("chevron.right", 12f, C.mentaTexto)
        }
        Box(Modifier.fillMaxWidth().height(8.dp).clip(CircleShape).background(C.chip)) {
            Box(Modifier.fillMaxWidth(avance.toFloat()).height(8.dp).clip(CircleShape).background(C.menta))
        }
        siguiente?.let { Texto("Lo próximo: ${it.titulo}", Tipo.lato(13f), C.sub, lineas = 1) }
    }
}
