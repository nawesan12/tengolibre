package ar.com.tengolibre.app.pantallas.turno

import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
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
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import ar.com.tengolibre.app.R
import ar.com.tengolibre.app.app.AccionesTurno
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.nucleo.*
import ar.com.tengolibre.app.pantallas.clientes.DestinoClientes
import ar.com.tengolibre.app.pantallas.hoy.BarraHoja
import ar.com.tengolibre.app.pantallas.hoy.Momentos
import ar.com.tengolibre.app.pantallas.hoy.TalonShape
import ar.com.tengolibre.app.pantallas.hoy.Troquel
import ar.com.tengolibre.app.pantallas.principal.LocalNavegacion
import ar.com.tengolibre.app.pantallas.principal.Pestana
import ar.com.tengolibre.app.tema.*
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable
import java.time.Duration
import java.time.Instant

// Clon de Pantallas/Turno/DetalleTurno.swift.

/** La caja de un turno (caja.md): lo que se cobra, lo pagado y lo que falta. */
@Serializable
data class CuentaCaja(
    val turnoId: String = "",
    val cliente: String = "",
    val servicio: String = "",
    val estadoTurno: String = "",
    val estado: String = "",
    val medio: String? = null,
    val cobradoEn: FechaApi? = null,
    val lineas: List<Linea> = emptyList(),
    val total: Int = 0,
    val medios: List<String> = emptyList(),
) {
    @Serializable data class Linea(val tipo: String = "", val concepto: String = "", val monto: Int = 0, val cantidad: Int? = null, val productoId: String? = null)
}

@Serializable
data class Producto(val id: String, val nombre: String = "", val precio: Int = 0, val stock: Int? = null, val activo: Boolean = true, val orden: Int = 0)

/** Un aviso que se le mandó al cliente por este turno (confirmación, recordatorio…). */
@Serializable
data class RegistroAviso(
    val id: String,
    val tipo: String = "",
    val para: String = "",
    val turnoId: String? = null,
    val cliente: String? = null,
    val estado: String = "",
    val intentos: Int = 0,
    val ultimoError: String? = null,
    val proximoIntento: FechaApi = Instant.EPOCH,
    val enviadoEn: FechaApi? = null,
    val creadoEn: FechaApi = Instant.EPOCH,
)

private fun medioTexto(m: String) = mapOf("efectivo" to "Efectivo", "transferencia" to "Transferencia", "mercadopago" to "QR de Mercado Pago", "debito" to "Débito", "credito" to "Crédito", "despues" to "Después")[m] ?: m

private data class Hito(val simbolo: String, val texto: String, val hora: String, val hecho: Boolean, val alerta: Boolean = false)

private fun corto(f: Instant) = if (Formato.diaISO(f) == Formato.diaISO(Instant.now())) "hoy · ${Formato.horaDe(f)}" else "${Formato.dia(f)} · ${Formato.horaDe(f)}"

/** 09 Detalle del turno: hoja mediana y grande. Llamar y WhatsApp, pagos, membresía, cómo viene el turno, los avisos
 *  que salieron y la nota interna. Abajo, «Marcar atendido y cobrar», Ausente, Mover y Cancelar (y, si es fijo,
 *  cancelar los siguientes). */
@Composable
fun DetalleTurno(turnoInicial: Turno, cerrar: () -> Unit) {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    val alcance = rememberCoroutineScope()
    var turno by remember { mutableStateOf(turnoInicial) }
    var caja by remember { mutableStateOf<CuentaCaja?>(null) }
    var avisos by remember { mutableStateOf<List<RegistroAviso>>(emptyList()) }
    var nota by remember { mutableStateOf("") }
    var editandoNota by remember { mutableStateOf(false) }
    var cobrar by remember { mutableStateOf(false) }
    var mover by remember { mutableStateOf(false) }
    var confirmarCancelar by remember { mutableStateOf(false) }
    var confirmarSiguientes by remember { mutableStateOf(false) }
    val est = turno.visible()
    val lectura = estado.soloLectura || estado.sinConexion

    @Serializable data class Avisos(val avisos: List<RegistroAviso> = emptyList())
    suspend fun recargarTurno() { estado.api.opcional<Turno>("/v1/panel/turnos/${turno.id}")?.let { turno = it } }
    suspend fun cargar() {
        caja = estado.api.opcional("/v1/panel/turnos/${turno.id}/caja")
        avisos = estado.api.opcional<Avisos>("/v1/panel/avisos", listOf("turno" to turno.id))?.avisos ?: emptyList()
        recargarTurno()
    }
    LaunchedEffect(Unit) { cargar() }

    fun cambiar(nuevo: String) {
        val antes = turno.estado
        alcance.launch {
            try {
                estado.api.hacer("PATCH", "/v1/panel/turnos/${turno.id}", AccionesTurno.CambioEstado(nuevo))
                turno = turno.copy(estado = nuevo)
                Haptico.exito()
                estado.version += 1
                Momentos.cambioDeEstado(estado, nav, turno, nuevo)
                if (nuevo == "cancelado" || nuevo == "ausente") {
                    val t = turno
                    estado.avisar(Toast("checkmark", if (nuevo == "cancelado") "Turno cancelado" else "Marcado ausente", "${t.cliente.nombre} · ${Formato.horaDe(t.inicio)}", "Deshacer")) {
                        runCatching { estado.api.hacer("PATCH", "/v1/panel/turnos/${t.id}", AccionesTurno.CambioEstado(antes)) }
                    }
                    cerrar()
                }
            } catch (e: Exception) { estado.avisar(e) }
        }
    }

    @Serializable data class Cancelados(val cancelados: List<String> = emptyList())
    fun cancelarSiguientes() = alcance.launch {
        try {
            val r: Cancelados = estado.api.enviar("POST", "/v1/panel/turnos/${turno.id}/cancelar-siguientes", emptyMap<String, String>())
            estado.version += 1
            estado.avisar("Cancelamos ${r.cancelados.size} ${if (r.cancelados.size == 1) "turno" else "turnos"}", "Les avisamos a todos.")
            cerrar()
        } catch (e: Exception) { estado.avisar(e) }
    }

    val tel = turno.cliente.telefono.filter { it in "+0123456789" }
    val esHoy = Formato.diaISO(turno.inicio) == Formato.diaISO(Instant.now())
    val f2 = Formato.fechaLarga(turno.inicio).split(" ").take(2).joinToString(" ")
    val opcionesMenu = buildList {
        if (!lectura) {
            if (turno.estado == "atendido" || turno.estado == "ausente") add(OpcionDeMenu("Volver a confirmado", "arrow.uturn.backward") { cambiar("confirmado") })
            if (turno.serieId != null) add(OpcionDeMenu("Cancelar este y los siguientes", "repeat", destructiva = true) { confirmarSiguientes = true })
            if (turno.llamarDesde != null) add(OpcionDeMenu("Ya la llamé", "phone.badge.checkmark") { alcance.launch { runCatching { estado.api.hacer("POST", "/v1/panel/turnos/${turno.id}/llamado") }; recargarTurno() } })
        }
        turno.cliente.id?.let { id -> add(OpcionDeMenu("Ver la ficha", "person.text.rectangle") { cerrar(); nav.pestana = Pestana.clientes; nav.pilaClientes.add(DestinoClientes.ficha(id)) }) }
        add(OpcionDeMenu("Compartir los datos", "square.and.arrow.up") {
            estado.compartirTexto("Tu turno en ${estado.negocio?.nombre.orEmpty()}: ${Formato.diaYHora(turno.inicio)}, ${turno.servicio}. Código ${turno.codigo}.")
        })
    }

    Box(Modifier.fillMaxSize().background(C.hoja)) {
        Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(start = 16.dp, end = 16.dp, top = 10.dp, bottom = 200.dp), verticalArrangement = Arrangement.spacedBy(14.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                TextoRotulo(if (esHoy) "HOY · $f2" else f2, modifier = Modifier.weight(1f))
                MenuBoton(opcionesMenu, "Más acciones") { Box(Modifier.size(44.dp).clip(CircleShape).background(C.chip), contentAlignment = Alignment.Center) { Icono("ellipsis", 18f) } }
                Box(Modifier.size(44.dp).clip(CircleShape).background(C.chip).clickable(onClick = cerrar), contentAlignment = Alignment.Center) { Icono("xmark", 16f, descripcion = "Cerrar") }
            }
            // Talón grande (88 dp de troquel)
            val forma = TalonShape(88.dp, 20.dp, 9.dp)
            Row(
                Modifier.fillMaxWidth().heightIn(min = 104.dp).height(IntrinsicSize.Min).clip(forma).background(C.fondo)
                    .then(if (est == EstadoTurno.enCurso) Modifier.border(2.dp, C.seleccion, forma) else Modifier),
            ) {
                Column(Modifier.width(88.dp).fillMaxHeight(), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(2.dp, Alignment.CenterVertically)) {
                    Texto(Formato.horaDe(turno.inicio), Tipo.lato(24f, Peso.Black, numeros = true), C.tinta, tracking = (-0.5).sp)
                    Texto("a ${Formato.horaDe(turno.fin)}", Tipo.lato(13f, numeros = true), C.sub)
                }
                Troquel()
                Column(Modifier.weight(1f).padding(horizontal = 14.dp, vertical = 12.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                    Texto(turno.cliente.nombre, Tipo.lato(22f, Peso.Bold), C.tinta, tracking = (-0.4).sp)
                    Texto(if (turno.profesional.isEmpty()) turno.servicio else "${turno.servicio} · ${turno.profesional}", Tipo.lato(14f), C.sub)
                    val textoEstado = when {
                        est == EstadoTurno.enCurso -> "En curso · faltan ${maxOf(1, Duration.between(Instant.now(), turno.fin).toMinutes())} min"
                        est == EstadoTurno.pendientePago && turno.venceEn != null -> "Seña pendiente · hasta las ${Formato.horaDe(turno.venceEn!!)}"
                        else -> est.texto
                    }
                    PastillaEstado(textoEstado, est.simbolo, est.fondo, est.tinta, 26.dp)
                }
            }
            // Contacto
            Row(Modifier.alpha(if (tel.isEmpty()) 0.45f else 1f), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Box(Modifier.weight(1f)) { BotonChip("Llamar", { if (tel.isNotEmpty()) estado.abrirLink("tel:$tel") }, simbolo = "phone") }
                Row(
                    Modifier.weight(1f).heightIn(min = 52.dp).clip(RoundedCornerShape(16.dp)).background(C.chip).clickable(enabled = tel.isNotEmpty()) {
                        var d = tel.filter { it.isDigit() }
                        if (d.startsWith("0")) d = d.drop(1)
                        if (!d.startsWith("54")) d = "549$d"
                        estado.abrirLink("https://wa.me/$d")
                    },
                    horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically,
                ) {
                    Image(painterResource(R.drawable.logo_whatsapp), null, Modifier.size(22.dp)); Spacer(Modifier.width(8.dp))
                    Texto("WhatsApp", Tipo.lato(16f, Peso.Bold), C.tinta)
                }
            }
            // Pagos
            val c = caja
            if (c != null && estado.vePrecios) {
                Column(Modifier.fillMaxWidth().clip(RoundedCornerShape(18.dp)).background(C.fondo).padding(horizontal = 16.dp, vertical = 14.dp)) {
                    TextoRotulo("Pagos", modifier = Modifier.padding(bottom = 8.dp))
                    c.lineas.forEach { l ->
                        Row(Modifier.padding(vertical = 6.dp), verticalAlignment = Alignment.CenterVertically) {
                            if (l.tipo == "sena") { Image(painterResource(R.drawable.logo_mercadopago), null, Modifier.size(20.dp)); Spacer(Modifier.width(6.dp)) }
                            Texto(if (l.tipo == "sena") "Seña pagada" else l.cantidad?.let { if (it > 1) "${l.concepto} × $it" else l.concepto } ?: l.concepto, Tipo.lato(15f, numeros = true), C.tinta, Modifier.weight(1f))
                            Texto(if (l.monto < 0) "− ${Formato.pesos(-l.monto)}" else Formato.pesos(l.monto), Tipo.lato(15f, numeros = true), C.tinta)
                        }
                    }
                    Spacer(Modifier.height(6.dp)); Troquel(vertical = false); Spacer(Modifier.height(10.dp))
                    Row {
                        Texto(if (c.estado == "cobrado") "Cobrado${c.medio?.let { " · ${medioTexto(it)}" } ?: ""}" else if (c.estado == "debe") "Debe" else "Falta cobrar", Tipo.lato(17f, Peso.Black, numeros = true), C.tinta, Modifier.weight(1f))
                        Texto(Formato.pesos(c.total), Tipo.lato(17f, Peso.Black, numeros = true), C.tinta)
                    }
                }
            }
            caja?.lineas?.firstOrNull { it.tipo == "membresia" }?.let { m ->
                Row(Modifier.fillMaxWidth().clip(RoundedCornerShape(18.dp)).background(C.fondo).padding(horizontal = 16.dp, vertical = 12.dp), horizontalArrangement = Arrangement.spacedBy(12.dp), verticalAlignment = Alignment.CenterVertically) {
                    Icono("ticket", 20f, C.mentaTexto)
                    Column(verticalArrangement = Arrangement.spacedBy(1.dp)) {
                        Texto("Usa ${m.concepto}", Tipo.lato(15f, Peso.Bold), C.tinta)
                        Texto("Lo cubre la membresía", Tipo.lato(13f), C.sub)
                    }
                }
            }
            // Cómo viene
            val hitos = buildList {
                val origen = mapOf("web" to "Reservó por la web", "panel" to "Lo cargaron en el panel", "app" to "Reservó por la app", "whatsapp" to "Reservó por WhatsApp", "espera" to "Lo tomó de la lista de espera")[turno.origen] ?: "Reservó"
                add(Hito("checkmark.circle.fill", origen, corto(turno.creadoEn), true))
                if (caja?.lineas?.any { it.tipo == "sena" } == true) add(Hito("checkmark.circle.fill", "Pagó la seña", "", true))
                else if (turno.estado == "pendiente_pago") add(Hito("hourglass", "Esperando la seña${turno.venceEn?.let { " hasta las ${Formato.horaDe(it)}" } ?: ""}", "", false, true))
                avisos.forEach { a ->
                    val que = mapOf("confirmacion" to "Le mandamos la confirmación", "recordatorio" to "Le mandamos el recordatorio", "whatsapp_confirmar" to "Le pedimos que confirme por WhatsApp", "cancelacion" to "Le avisamos que se canceló", "reprogramacion" to "Le avisamos el cambio de horario", "resena" to "Le pedimos la reseña")[a.tipo] ?: "Aviso: ${a.tipo}"
                    val falla = a.estado == "fallido" || a.estado == "error"
                    add(Hito(if (falla) "exclamationmark.circle.fill" else if (a.enviadoEn != null) "checkmark.circle.fill" else "clock", if (falla) "$que · no salió" else que, a.enviadoEn?.let { corto(it) } ?: "", a.enviadoEn != null, falla))
                }
                turno.confirmadoEn?.let { c2 ->
                    val por = mapOf("whatsapp" to "Confirmó por WhatsApp", "web" to "Confirmó por la web", "panel" to "Lo confirmaron en el panel")[turno.confirmadoPor ?: ""] ?: "Confirmó"
                    add(Hito("checkmark.circle.fill", por, corto(c2), true))
                }
                turno.cambioPedidoEn?.let { add(Hito("arrow.triangle.2.circlepath", "Pidió cambiarlo", corto(it), false, true)) }
                val m = turno.llamarMotivo
                if (m != null && turno.llamarDesde != null) add(Hito("phone.badge.waveform", mapOf("cambio" to "Hay que llamarla: pidió cambiarlo", "sin_respuesta" to "Hay que llamarla: no contestó el WhatsApp", "no_llego" to "Hay que llamarla: no le llegó el WhatsApp")[m] ?: "Hay que llamarla", "", false, true))
                if (est == EstadoTurno.enCurso) add(Hito("timelapse", "Está en el turno", Formato.horaDe(turno.inicio), false))
                if (turno.estado == "atendido") add(Hito("checkmark.circle.fill", "Atendido", "", true))
                if (turno.estado == "ausente") add(Hito("person.slash", "No vino", "", false, true))
            }
            Column(Modifier.fillMaxWidth().clip(RoundedCornerShape(18.dp)).background(C.fondo).padding(horizontal = 16.dp, vertical = 14.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                TextoRotulo("Cómo viene")
                hitos.forEach { h ->
                    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                        Icono(h.simbolo, 18f, if (h.alerta) C.pagoTinta else if (h.hecho) C.mentaTexto else C.tinta)
                        Texto(h.texto, Tipo.lato(14f), C.tinta, Modifier.weight(1f))
                        Texto(h.hora, Tipo.lato(12f, numeros = true), C.sub)
                    }
                }
                val fallido = avisos.firstOrNull { it.estado == "fallido" || it.estado == "error" }
                if (fallido != null && !lectura) BotonTexto("Volver a mandarlo", { alcance.launch { runCatching { estado.api.hacer("POST", "/v1/panel/avisos/${fallido.id}/reenviar") }; cargar() } })
            }
            // Nota
            @Serializable data class CambioNota(val nota: String)
            Row(Modifier.fillMaxWidth().clip(RoundedCornerShape(18.dp)).background(C.fondo).padding(horizontal = 16.dp, vertical = 12.dp), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                Icono("note.text", 18f, C.sub)
                Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                    Texto("Nota interna · sólo la ve el equipo", Tipo.lato(13f, Peso.Bold), C.tinta)
                    if (editandoNota) {
                        val tinta = C.tinta
                        BasicTextField(nota, { nota = it }, Modifier.fillMaxWidth(), textStyle = Tipo.lato(15f).copy(color = tinta), minLines = 2, maxLines = 6, cursorBrush = SolidColor(C.mentaTexto))
                    } else Texto(turno.nota.ifEmpty { "Sin nota" }, Tipo.lato(15f), if (turno.nota.isEmpty()) C.sub else C.tinta)
                }
                if (!lectura) BotonTexto(if (editandoNota) "Guardar" else "Editar", {
                    if (editandoNota) alcance.launch {
                        try { estado.api.hacer("PATCH", "/v1/panel/turnos/${turno.id}", CambioNota(nota)); turno = turno.copy(nota = nota); editandoNota = false }
                        catch (e: Exception) { estado.avisar(e) }
                    } else { nota = turno.nota; editandoNota = true }
                })
            }
        }
        // Acciones
        if (!lectura) {
            Column(Modifier.align(Alignment.BottomCenter).fillMaxWidth().background(C.hoja)) {
                Box(Modifier.fillMaxWidth().height(1.dp).background(C.linea))
                Column(Modifier.padding(start = 16.dp, end = 16.dp, top = 12.dp, bottom = 8.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    when {
                        turno.estado == "pendiente" || turno.estado == "pendiente_pago" -> BotonMenta("Confirmar el turno", { cambiar("confirmado") }, simbolo = "checkmark.circle")
                        turno.estado != "cancelado" && caja?.estado != "cobrado" && estado.vePrecios -> BotonMenta(if (turno.estado == "atendido") "Cobrar" else "Marcar atendido y cobrar", { cobrar = true }, simbolo = "checkmark.circle")
                        turno.estado != "atendido" && turno.estado != "cancelado" -> BotonMenta("Marcar atendido", { cambiar("atendido") }, simbolo = "checkmark.circle")
                    }
                    if (turno.estado != "cancelado") Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        if (EstadoTurno.puedePasar(turno.estado, "ausente") && turno.estado != "ausente") Box(Modifier.weight(1f)) { BotonContorno("Ausente", { cambiar("ausente") }, alto = 48.dp, tam = 14f, simbolo = "person.slash") }
                        Box(Modifier.weight(1f)) { BotonContorno("Mover", { mover = true }, alto = 48.dp, tam = 14f, simbolo = "calendar.badge.clock") }
                        Box(Modifier.weight(1f)) { BotonContorno("Cancelar", { confirmarCancelar = true }, alto = 48.dp, tam = 14f, simbolo = "calendar.badge.minus") }
                    }
                }
            }
        }
    }
    Hoja(cobrar, { cobrar = false }, listOf(Altura.media, Altura.grande)) { Cobrar(turno, caja, listo = { alcance.launch { cargar() } }, cerrar = { cobrar = false }) }
    Hoja(mover, { mover = false }) { Reprogramar(turno, listo = { t -> turno = t; estado.version += 1 }, cerrar = { mover = false }) }
    Confirmacion(confirmarCancelar, "¿Cancelar el turno de ${turno.cliente.nombre}?", "Le avisamos y el horario queda libre para otra persona.",
        listOf(OpcionConfirmacion("Cancelar el turno", destructiva = true) { cambiar("cancelado") }), { confirmarCancelar = false })
    Confirmacion(confirmarSiguientes, "¿Cancelar este y los siguientes del turno fijo?", "",
        listOf(OpcionConfirmacion("Cancelar los siguientes", destructiva = true) { cancelarSiguientes() }), { confirmarSiguientes = false })
}


private val MEDIOS = listOf(Triple("efectivo", "Efectivo", "banknote"), Triple("transferencia", "Transferencia", "building.columns"),
    Triple("mercadopago", "QR de Mercado Pago", "qrcode"), Triple("debito", "Débito", "creditcard"), Triple("credito", "Crédito", "creditcard.fill"))

/** Marcar atendido y cobrar (caja.md): medio de pago, productos vendidos y «Cobrar después». */
@Composable
fun ColumnScope.Cobrar(turno: Turno, caja: CuentaCaja?, listo: () -> Unit, cerrar: () -> Unit) {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    val alcance = rememberCoroutineScope()
    var medio by remember { mutableStateOf("efectivo") }
    var productos by remember { mutableStateOf<List<Producto>>(emptyList()) }
    var elegidos by remember { mutableStateOf(setOf<String>()) }
    var enviando by remember { mutableStateOf(false) }
    val total = (caja?.total ?: turno.precio) + productos.filter { it.id in elegidos }.sumOf { it.precio }

    @Serializable data class P(val productos: List<Producto> = emptyList())
    LaunchedEffect(Unit) { productos = estado.api.opcional<P>("/v1/panel/productos", listOf("activos" to "1"))?.productos ?: emptyList() }

    @Serializable data class L(val productoId: String, val cantidad: Int)
    @Serializable data class Pedido(val medio: String, val productos: List<L>)
    fun cobrarCon(m: String) {
        enviando = true
        alcance.launch {
            try {
                if (turno.estado != "atendido") estado.api.hacer("PATCH", "/v1/panel/turnos/${turno.id}", AccionesTurno.CambioEstado("atendido"))
                estado.api.hacer("POST", "/v1/panel/turnos/${turno.id}/caja", Pedido(m, elegidos.map { L(it, 1) }))
                Haptico.exito()
                estado.version += 1
                estado.avisar(if (m == "despues") "Atendido · queda debiendo" else "Atendido y cobrado", if (m == "despues") "Lo cobrás desde la caja cuando pague." else Formato.pesos(total))
                listo()
                cerrar()
                if (m != "despues") Momentos.cobro(estado, nav)
            } catch (e: Exception) { estado.avisar(e) } finally { enviando = false }
        }
    }

    BarraHoja("Cobrar", cerrar)
    Column(Modifier.weight(1f).verticalScroll(rememberScrollState()).padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
            TextoRotulo("Falta cobrar")
            Texto(Formato.pesos(total), Tipo.lato(40f, Peso.Black, numeros = true), C.tinta)
            Texto("${turno.cliente.nombre} · ${turno.servicio}", Tipo.lato(14f), C.sub)
        }
        TextoRotulo("Cómo paga")
        MEDIOS.chunked(2).forEach { fila ->
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                fila.forEach { (k, t, s) ->
                    val on = medio == k
                    val forma = RoundedCornerShape(16.dp)
                    Row(
                        Modifier.weight(1f).heightIn(min = 56.dp).clip(forma).background(if (on) C.seleccion else C.fondo).then(if (on) Modifier else Modifier.border(1.5.dp, C.linea, forma))
                            .clickable { medio = k; Haptico.seleccion() },
                        horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically,
                    ) { Icono(s, 15f, if (on) C.seleccionTexto else C.tinta); Spacer(Modifier.width(6.dp)); Texto(t, Tipo.lato(15f, Peso.Bold), if (on) C.seleccionTexto else C.tinta, lineas = 1) }
                }
                if (fila.size == 1) Spacer(Modifier.weight(1f))
            }
        }
        if (productos.isNotEmpty()) {
            TextoRotulo("Sumar productos")
            productos.forEach { p ->
                Row(Modifier.fillMaxWidth().heightIn(min = 48.dp).clickable { elegidos = if (p.id in elegidos) elegidos - p.id else elegidos + p.id }, verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Icono(if (p.id in elegidos) "checkmark.circle.fill" else "circle", 22f, C.mentaTexto)
                    Texto(p.nombre, Tipo.lato(16f), C.tinta, Modifier.weight(1f))
                    Texto(Formato.pesos(p.precio), Tipo.lato(15f, numeros = true), C.sub)
                }
            }
        }
    }
    Column(Modifier.fillMaxWidth().background(C.hoja).padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp), horizontalAlignment = Alignment.CenterHorizontally) {
        BotonMenta(if (enviando) "Cobrando…" else "Cobrar ${Formato.pesos(total)}", { cobrarCon(medio) }, habilitado = !enviando)
        BotonTexto("Cobrar después", { cobrarCon("despues") }, color = C.tinta)
    }
}

/** Mover un turno (reprogramar): otro día, otra profesional, un horario libre o encimado. */
@Composable
fun ColumnScope.Reprogramar(turno: Turno, listo: (Turno) -> Unit, cerrar: () -> Unit) {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    val hoy = Formato.enArgentina(Instant.now()).toLocalDate()
    var fecha by remember { mutableStateOf(maxOf(Formato.enArgentina(turno.inicio).toLocalDate(), hoy)) }
    var profesional by remember { mutableStateOf(turno.profesionalId) }
    var profesionales by remember { mutableStateOf<List<Profesional>>(emptyList()) }
    var libres by remember { mutableStateOf<List<HorarioLibre>>(emptyList()) }
    var elegido by remember { mutableStateOf<HorarioLibre?>(null) }
    var forzar by remember { mutableStateOf(false) }
    var hora by remember { mutableIntStateOf(Formato.minutosDelDia(turno.inicio)) }

    @Serializable data class Pros(val profesionales: List<Profesional> = emptyList())
    @Serializable data class Libres(val horarios: List<HorarioLibre> = emptyList())
    LaunchedEffect(Unit) { profesionales = estado.api.opcional<Pros>("/v1/panel/profesionales")?.profesionales ?: emptyList() }
    LaunchedEffect(fecha, profesional) {
        elegido = null
        val s = turno.servicioId ?: return@LaunchedEffect
        val q = buildList { add("servicio" to s); add("fecha" to fecha.toString()); profesional?.let { add("profesional" to it) } }
        libres = estado.api.opcional<Libres>("/v1/panel/libres", q)?.horarios ?: emptyList()
    }

    @Serializable data class Pedido(val inicio: String, val profesionalId: String, val forzar: Boolean)
    fun moverlo() = alcance.launch {
        val dia = fecha.atStartOfDay(Formato.zona).toInstant()
        val inicio = if (forzar) Dia.fecha(dia, hora) else (elegido?.inicio ?: turno.inicio)
        try {
            val t: Turno = estado.api.enviar("POST", "/v1/panel/turnos/${turno.id}/reprogramar", Pedido(Fecha.texto(inicio), profesional ?: turno.profesionalId.orEmpty(), forzar))
            Haptico.exito()
            estado.avisar("Turno movido", "${Formato.diaYHora(t.inicio)}. Le avisamos a ${t.cliente.nombre}.")
            listo(t)
            cerrar()
        } catch (e: Exception) { estado.avisar(e) }
    }

    BarraHoja("Mover el turno", cerrar, accion = "Mover", habilitada = forzar || elegido != null) { moverlo() }
    Column(Modifier.weight(1f).background(C.fondo).verticalScroll(rememberScrollState()).padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        TextoRotulo("Día")
        Box(Modifier.fillMaxWidth().tarjeta().padding(12.dp)) { CalendarioMes(fecha, { fecha = it }, minimo = hoy) }
        TextoRotulo("Con quién")
        val aptas = profesionales.filter { it.activo && (turno.servicioId == null || it.servicios.isEmpty() || turno.servicioId in it.servicios) }
        Column(Modifier.fillMaxWidth().tarjeta()) {
            aptas.forEachIndexed { i, p ->
                if (i > 0) Box(Modifier.padding(start = 16.dp).fillMaxWidth().height(1.dp).background(C.linea))
                Row(Modifier.fillMaxWidth().heightIn(min = 48.dp).clickable { profesional = p.id }.padding(horizontal = 16.dp), verticalAlignment = Alignment.CenterVertically) {
                    Texto(p.nombre, Tipo.lato(16f), C.tinta, Modifier.weight(1f))
                    if (profesional == p.id) Icono("checkmark", 16f, C.mentaTexto)
                }
            }
        }
        TextoRotulo(if (forzar) "Hora" else "Horarios libres")
        Column(Modifier.fillMaxWidth().tarjeta().padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            if (forzar) SelectorHora(hora, { hora = it })
            else if (libres.isEmpty()) Texto("No hay horarios libres ese día.", Tipo.lato(15f), C.sub)
            else libres.chunked(4).forEach { fila ->
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    fila.forEach { h ->
                        val on = elegido?.inicio == h.inicio
                        Box(Modifier.weight(1f).heightIn(min = 44.dp).clip(RoundedCornerShape(12.dp)).background(if (on) C.seleccion else C.mentaSuave).clickable { elegido = h }, contentAlignment = Alignment.Center) {
                            Texto(h.hora, Tipo.lato(16f, Peso.Black, numeros = true), if (on) C.seleccionTexto else C.huecoTinta)
                        }
                    }
                    repeat(4 - fila.size) { Spacer(Modifier.weight(1f)) }
                }
            }
            Interruptor("Encimarlo aunque esté ocupado", forzar, { forzar = it })
        }
    }
}
