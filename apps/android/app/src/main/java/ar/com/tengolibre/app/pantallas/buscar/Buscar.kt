package ar.com.tengolibre.app.pantallas.buscar

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import ar.com.tengolibre.app.app.Estado
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.nucleo.*
import ar.com.tengolibre.app.pantallas.clientes.DestinoClientes
import ar.com.tengolibre.app.pantallas.hoy.BarraHoja
import ar.com.tengolibre.app.pantallas.mas.DestinoMas
import ar.com.tengolibre.app.pantallas.principal.*
import ar.com.tengolibre.app.tema.*
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable
import java.text.Normalizer
import java.time.Instant

// Clon de Pantallas/Buscar/Buscar.swift.

@Serializable
data class Encontrado(val clientes: List<Cli>? = null, val turnos: List<Tur>? = null, val servicios: List<Ser>? = null) {
    @Serializable data class Cli(val id: String, val nombre: String = "", val telefono: String = "")
    @Serializable data class Tur(val id: String, val inicio: FechaApi, val estado: String = "", val servicio: String = "", val profesional: String = "", val cliente: String = "")
    @Serializable data class Ser(val id: String, val nombre: String = "", val precio: Int = 0, val duracion: Int = 0)
}

/** Lo que se puede hacer, con las palabras con que la gente lo busca. */
data class Cosa(val simbolo: String, val titulo: String, val palabras: String, val ir: Ir) {
    sealed class Ir { data class seccion(val d: DestinoMas) : Ir(); data class pestana(val p: Pestana) : Ir(); data object nuevo : Ir(); data object compartir : Ir(); data object avisos : Ir() }
}

fun cosas(estado: Estado): List<Cosa> {
    val g = estado.gestiona; val mostrador = g || estado.rol == "recepcion"
    val l = mutableListOf(
        Cosa("plus.circle", "Cargar un turno", "nuevo turno reservar agendar anotar cargar sobreturno fijo", Cosa.Ir.nuevo),
        Cosa("calendar", "Agenda", "semana dia calendario agenda horarios", Cosa.Ir.pestana(Pestana.agenda)),
        Cosa("square.and.arrow.up", "Compartir mi link", "link qr instagram whatsapp compartir pagina reservas historia", Cosa.Ir.compartir),
        Cosa("bell", "Avisos", "notificaciones campanita avisos novedades", Cosa.Ir.avisos),
        Cosa("play.circle", "Qué podés hacer", "ayuda tutorial recorrido widget como se usa", Cosa.Ir.seccion(DestinoMas.recorrido)),
        Cosa("person.crop.circle", "Tu cuenta", "cuenta salir cerrar sesion negocio cambiar modo", Cosa.Ir.seccion(DestinoMas.cuenta)),
    )
    if (estado.esProfesional) {
        l += listOf(Cosa("clock", "Mi horario", "horario dias libres vacaciones bloquear franco licencia", Cosa.Ir.seccion(DestinoMas.miHorario)),
            Cosa("star.bubble", "Reseñas", "reseñas opiniones estrellas comentarios", Cosa.Ir.pestana(Pestana.resenas)))
        return l
    }
    l += listOf(Cosa("banknote", "Caja", "plata dinero caja cobrar cobrado efectivo transferencia debe ingresos hoy", Cosa.Ir.seccion(DestinoMas.caja)),
        Cosa("star.bubble", "Reseñas", "reseñas opiniones estrellas comentarios responder", Cosa.Ir.seccion(DestinoMas.resenas)),
        Cosa("person.2.badge.gearshape", "Equipo y horarios", "equipo profesionales horario empleados invitar bloquear feriado", Cosa.Ir.seccion(DestinoMas.equipo)))
    if (mostrador) l += listOf(
        Cosa("hourglass", "Lista de espera", "espera lista avisar lugar hueco liberado", Cosa.Ir.pestana(Pestana.espera)),
        Cosa("person.2", "Clientes", "clientas clientes contactos fichas", Cosa.Ir.pestana(Pestana.clientes)),
        Cosa("ticket", "Membresías", "membresias abono socias plan mensual suscripcion clientes", Cosa.Ir.seccion(DestinoMas.membresias)),
        Cosa("tag", "Cupones", "cupon descuento promo promocion oferta codigo", Cosa.Ir.seccion(DestinoMas.cupones)),
        Cosa("tag.circle", "Etiquetas", "etiquetas vip categorias clientas", Cosa.Ir.seccion(DestinoMas.etiquetas)),
        Cosa("exclamationmark.shield", "Penalidades y ausencias", "ausentes faltas penalidad no vino bloquear clienta", Cosa.Ir.seccion(DestinoMas.penalidades)),
        Cosa("envelope.badge", "Recordatorios a clientes", "cumpleaños extrañamos mails recordatorios volver", Cosa.Ir.seccion(DestinoMas.avisosClientes)),
    )
    if (g) {
        l += listOf(
            Cosa("sparkles", "Servicios", "servicios precios duracion productos categorias carta", Cosa.Ir.seccion(DestinoMas.servicios)),
            Cosa("creditcard", "Pagos y señas", "seña señas mercado pago mp cobrar adelantado devolver pagos", Cosa.Ir.seccion(DestinoMas.cobros)),
            Cosa("chart.bar", "Reportes", "reportes estadisticas ingresos ocupacion horas muertas numeros ventas", Cosa.Ir.seccion(DestinoMas.reportes)),
            Cosa("beach.umbrella", "Días libres del equipo", "vacaciones dias libres pedidos aprobar licencia", Cosa.Ir.seccion(DestinoMas.diasLibres)),
            Cosa("gift", "Referidos", "referidos recomendar amigas premio", Cosa.Ir.seccion(DestinoMas.referidos)),
            Cosa("globe", "Mi web", "web pagina logo fotos portada colores direccion redes instagram", Cosa.Ir.seccion(DestinoMas.miWeb)),
            Cosa("paintpalette", "Plantillas", "plantilla diseño estilo como se ve pagina", Cosa.Ir.seccion(DestinoMas.plantillas)),
            Cosa("link", "Dominio propio", "dominio .com direccion propia url", Cosa.Ir.seccion(DestinoMas.dominio)),
            Cosa("point.3.connected.trianglepath.dotted", "Integraciones e IA", "ia inteligencia artificial chatgpt claude gemini asistente calendario google mcp", Cosa.Ir.seccion(DestinoMas.integraciones)),
            Cosa("gearshape", "Configuración", "configuracion ajustes reservas anticipacion cancelar politica intervalo", Cosa.Ir.seccion(DestinoMas.configuracion)),
            Cosa("message", "WhatsApp", "whatsapp confirmar confirmacion mensajes recordatorio", Cosa.Ir.seccion(DestinoMas.whatsapp)),
            Cosa("list.bullet.rectangle", "Actividad", "auditoria actividad quien hizo historial cambios", Cosa.Ir.seccion(DestinoMas.auditoria)),
            Cosa("chart.line.uptrend.xyaxis", "Potenciá tu negocio", "crecer mas reservas primeros pasos avance", Cosa.Ir.seccion(DestinoMas.primerosPasos)),
        )
        if (estado.esDuena) l += listOf(Cosa("crown", "Suscripción", "suscripcion plan pagar tengo libre prueba", Cosa.Ir.seccion(DestinoMas.suscripcion)),
            Cosa("building.2", "Sucursales", "sucursales locales grupo otro local", Cosa.Ir.seccion(DestinoMas.sucursales)))
    }
    return l
}

private fun normal(s: String) = Normalizer.normalize(s, Normalizer.Form.NFD).replace(Regex("\\p{M}"), "").lowercase()

/** «¿Qué querés hacer?»: encuentra secciones y acciones de la app por lo que la gente dice («plata», «seña», «IA»,
 *  «vacaciones»…) y, con la API (`GET /v1/panel/buscar`), clientas, turnos y servicios. */
@Composable
fun ColumnScope.Buscar(cerrar: () -> Unit) {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    val alcance = rememberCoroutineScope()
    var q by remember { mutableStateOf("") }
    var r by remember { mutableStateOf<Encontrado?>(null) }
    val foco = remember { FocusRequester() }
    val todas = cosas(estado)
    val partes = normal(q).split(" ").filter { it.length >= 2 }
    val encontradas = if (partes.isEmpty()) emptyList() else todas.filter { c -> val base = normal(c.titulo + " " + c.palabras).split(" "); partes.all { p -> base.any { it.startsWith(p) } } }

    LaunchedEffect(Unit) { runCatching { foco.requestFocus() } }
    LaunchedEffect(q) {
        val t = q.trim()
        if (t.length < 2) { r = null; return@LaunchedEffect }
        delay(250)
        r = estado.api.opcional("/v1/panel/buscar", listOf("q" to t))
    }

    /** Cierra el buscador y después navega (dos hojas no se abren a la vez). */
    fun ir(hacer: () -> Unit) { cerrar(); alcance.launch { delay(350); hacer() } }
    fun abrir(c: Cosa) = when (val d = c.ir) {
        is Cosa.Ir.seccion -> ir { nav.pestana = Pestana.hoy; nav.pilaHoy.add(d.d) }
        is Cosa.Ir.pestana -> ir { nav.pestana = d.p }
        Cosa.Ir.nuevo -> ir { nav.nuevo = NuevoTurnoInicio() }
        Cosa.Ir.compartir -> ir { nav.compartir = true }
        Cosa.Ir.avisos -> ir { nav.avisos = true }
    }

    BarraHoja("Buscar", cerrar, textoCancelar = "Cerrar")
    Row(Modifier.padding(horizontal = 16.dp).fillMaxWidth().height(40.dp).clip(RoundedCornerShape(11.dp)).background(C.campo).padding(horizontal = 10.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        Icono("magnifyingglass", 16f, C.sub)
        Box(Modifier.weight(1f)) {
            if (q.isEmpty()) Texto("¿Qué querés hacer?", Tipo.lato(16f), C.sub)
            val tinta = C.tinta
            BasicTextField(q, { q = it }, Modifier.fillMaxWidth().focusRequester(foco), singleLine = true, textStyle = Tipo.lato(16f).copy(color = tinta), cursorBrush = SolidColor(C.mentaTexto))
        }
    }
    Column(Modifier.weight(1f).background(C.fondo).verticalScroll(rememberScrollState()).padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        @Composable fun seccion(t: String, filas: List<Pair<String, @Composable () -> Unit>>) {
            if (filas.isEmpty()) return
            Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                TextoRotulo(t, modifier = Modifier.padding(horizontal = 4.dp))
                Column(Modifier.fillMaxWidth().tarjeta()) {
                    filas.forEachIndexed { i, (_, f) -> if (i > 0) Box(Modifier.padding(start = 52.dp).fillMaxWidth().height(1.dp).background(C.linea)); f() }
                }
            }
        }
        @Composable fun fila(simbolo: String?, titulo: String, sub: String = "", alTocar: () -> Unit, icono: (@Composable () -> Unit)? = null) {
            Row(Modifier.fillMaxWidth().heightIn(min = 48.dp).clickable(onClick = alTocar).padding(horizontal = 14.dp, vertical = 8.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(14.dp)) {
                Box(Modifier.width(24.dp), contentAlignment = Alignment.Center) { if (icono != null) icono() else if (simbolo != null) Icono(simbolo, 18f, C.tinta) }
                Column(Modifier.weight(1f)) {
                    Texto(titulo, Tipo.lato(16f), C.tinta)
                    if (sub.isNotEmpty()) Texto(sub, Tipo.lato(13f), C.sub)
                }
            }
        }
        if (q.trim().length < 2) {
            seccion("Lo más buscado", todas.take(if (estado.esProfesional) 4 else 8).map { c -> c.titulo to @Composable { fila(c.simbolo, c.titulo, alTocar = { abrir(c) }) } })
        } else {
            seccion("En la app", encontradas.map { c -> c.titulo to @Composable { fila(c.simbolo, c.titulo, alTocar = { abrir(c) }) } })
            if (!estado.esProfesional) seccion("Clientas", (r?.clientes ?: emptyList()).map { c -> c.id to @Composable {
                fila(null, c.nombre, c.telefono, { ir { nav.pestana = Pestana.clientes; nav.pilaClientes.add(DestinoClientes.ficha(c.id)) } }) { Iniciales(c.nombre, 30.dp) }
            } })
            seccion("Turnos", (r?.turnos ?: emptyList()).map { t -> t.id to @Composable {
                fila("calendar", "${t.cliente} · ${t.servicio}", Formato.diaYHora(t.inicio), { alcance.launch { estado.api.opcional<Turno>("/v1/panel/turnos/${t.id}")?.let { x -> ir { nav.turno = x } } } })
            } })
            if (estado.gestiona) seccion("Servicios", (r?.servicios ?: emptyList()).map { s -> s.id to @Composable {
                fila("sparkles", "${s.nombre} · ${Formato.pesos(s.precio)}", alTocar = { ir { nav.pestana = Pestana.mas; nav.pilaMas.add(DestinoMas.servicios) } })
            } })
            if (encontradas.isEmpty() && r != null && r?.clientes.isNullOrEmpty() && r?.turnos.isNullOrEmpty() && r?.servicios.isNullOrEmpty())
                Texto("No encontramos «$q». Probá con otra palabra.", Tipo.lato(16f), C.sub)
        }
    }
}

/** El acceso al buscador: un campo de mentira que abre «¿Qué querés hacer?». */
@Composable
fun CampoBuscar() {
    val nav = LocalNavegacion.current
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Row(
            Modifier.fillMaxWidth().height(44.dp).clip(RoundedCornerShape(14.dp)).background(Color.White.copy(alpha = 0.12f)).clickable { nav.buscar = true }
                .semantics { contentDescription = "Buscar: ¿qué querés hacer?" }.padding(horizontal = 14.dp),
            verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Icono("magnifyingglass", 17f, Color.White.copy(alpha = 0.75f))
            Texto("¿Qué querés hacer?", Tipo.lato(16f), Color.White.copy(alpha = 0.75f))
        }
        VistaConsejo(Consejos.buscar)
    }
}
