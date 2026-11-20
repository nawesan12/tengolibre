package ar.com.tengolibre.app.pantallas.asistente

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
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.text.selection.SelectionContainer
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
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.foundation.layout.width
import androidx.compose.ui.semantics.clearAndSetSemantics
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.nucleo.*
import ar.com.tengolibre.app.pantallas.hoy.BarraHoja
import ar.com.tengolibre.app.tema.*
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable
import java.util.UUID

// Clon de Pantallas/Asistente/Asistente.swift.

// MARK: Modelos (docs/api/asistente.md)

/** GET /v1/panel/asistente: si hay IA, los ejemplos y cuánto queda del cupo del mes. */
@Serializable
data class EstadoAsistente(
    val ia: Boolean = false,
    /** La plataforma tiene IA, la dueña la prendió para el negocio y quien mira la puede prender. */
    val iaPlataforma: Boolean = false,
    val iaNegocio: Boolean = false,
    val puedePrender: Boolean = false,
    val sugerencias: List<String> = emptyList(),
    val cupo: CupoAsistente? = null,
)

@Serializable
data class CupoAsistente(val cupoMensual: Int = 0, val usadasMes: Int = 0, val usadasHoy: Int = 0, val topeDiario: Int = 0)

/** Un cambio en la agenda que el asistente no hace solo: vuelve para que la persona lo confirme. */
@Serializable
data class PropuestaAsistente(val id: String, val accion: String = "", val resumen: String = "", val venceEn: String? = null, val boton: String? = null) {
    /** Lo que dice el botón («Sí, cancelarlo»): un «Confirmar» / «Cancelar» al lado de «Cancelar el turno de Rosa» se
     *  leía como cancelar el turno. */
    val textoBoton: String get() = boton?.takeIf { it.isNotEmpty() } ?: when (accion) {
        "cancelar_turno" -> "Sí, cancelarlo"
        "reprogramar_turno" -> "Sí, moverlo"
        "confirmar_turno" -> "Sí, confirmarlo"
        "crear_turno" -> "Sí, cargarlo"
        "ofrecer_a_la_lista" -> "Sí, ofrecerlo"
        else -> "Sí, hacerlo"
    }
}

/** Un turno de una lista del asistente: se muestra como fila que abre el turno. */
@Serializable
data class ItemTurnoAsistente(val id: String, val dia: String = "", val hora: String = "", val cliente: String = "", val servicio: String = "",
    val profesional: String? = null, val estado: String? = null)

/** No hay más IA por ahora (mes, día o plataforma): el intérprete sigue contestando. */
@Serializable
data class TopeAsistente(val motivo: String = "", val mensaje: String = "")

/** POST /v1/panel/asistente (y lo que vuelve de /confirmar y /descartar, que sólo trae `respuesta`). */
@Serializable
data class RespuestaAsistente(val respuesta: String = "", val propuestas: List<PropuestaAsistente>? = null, val origen: String? = null, val tope: TopeAsistente? = null,
    val encabezado: String? = null, val items: List<ItemTurnoAsistente>? = null)

/** Un turno de la charla, como lo pide la API en `historial`. */
@Serializable
data class TurnoChat(val rol: String, val texto: String)

// MARK: La entrada en Hoy

/** El renglón de Hoy que abre el asistente: preguntar por la agenda o pedir un cambio, escrito o dictado. La hoja la
 *  presenta Principal (nav.asistente). Sin conexión queda apagado. */
@Composable
fun EntradaAsistente() {
    val estado = LocalEstado.current
    val nav = ar.com.tengolibre.app.pantallas.principal.LocalNavegacion.current
    Row(
        Modifier.fillMaxWidth().heightIn(min = 64.dp).alpha(if (estado.sinConexion) 0.55f else 1f).tarjeta(22.dp).clip(RoundedCornerShape(22.dp))
            .clickable(enabled = !estado.sinConexion) { Haptico.suave(); nav.asistente = true }.padding(horizontal = 16.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Box(Modifier.size(40.dp).clip(RoundedCornerShape(12.dp)).background(C.menta), contentAlignment = Alignment.Center) {
            Icono("sparkles", 18f, C.petroleoHondo)
        }
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Texto("Preguntale al asistente", Tipo.lato(16f, Peso.Bold), C.tinta)
            Texto(if (estado.sinConexion) "Sin conexión" else "«¿Qué tengo mañana?» o «Pasá a Juana al jueves»", Tipo.lato(13f), C.sub, lineas = 2)
        }
        Icono("chevron.right", 15f, C.sub)
    }
}

// MARK: La hoja

/** Un mensaje de la charla. `error`: un error de la API, se muestra pero no viaja en el historial. */
data class Mensaje(
    val id: String = UUID.randomUUID().toString(),
    val rol: String,
    val texto: String,
    val propuestas: List<PropuestaAsistente> = emptyList(),
    val encabezado: String? = null,
    val items: List<ItemTurnoAsistente> = emptyList(),
    val error: Boolean = false,
)

/** La charla con el asistente: los ejemplos, las burbujas (las listas de turnos como filas), «Pensando…», las
 *  propuestas con el botón de la acción y «No, dejalo así», el aviso del tope, el cupo del mes y, para la dueña,
 *  prender la IA. La charla vive mientras la hoja está abierta. */
@Composable
fun ColumnScope.Asistente(cerrar: () -> Unit) {
    val nav = ar.com.tengolibre.app.pantallas.principal.LocalNavegacion.current
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    var info by remember { mutableStateOf<EstadoAsistente?>(null) }
    var mensajes by remember { mutableStateOf<List<Mensaje>>(emptyList()) }
    var texto by remember { mutableStateOf("") }
    var pensando by remember { mutableStateOf(false) }
    var tope by remember { mutableStateOf<TopeAsistente?>(null) }
    /** Propuestas ya resueltas: id → «Hecho», «Cancelada»… */
    var resueltas by remember { mutableStateOf(mapOf<String, String>()) }
    var resolviendo by remember { mutableStateOf<String?>(null) }
    var prendiendo by remember { mutableStateOf(false) }
    val scroll = rememberScrollState()
    val puedeEnviar = texto.isNotBlank() && !pensando

    suspend fun cargar() { info = estado.api.opcional<EstadoAsistente>("/v1/panel/asistente") }
    LaunchedEffect(Unit) { cargar() }
    // Después de que se dibuje lo nuevo (si no, maxValue todavía es el de antes).
    LaunchedEffect(mensajes.size, pensando) { delay(60); scroll.animateScrollTo(scroll.maxValue) }

    @Serializable data class Pregunta(val pregunta: String, val historial: List<TurnoChat>)
    @Serializable data class Cual(val propuestaId: String)
    @Serializable data class Activa(val activa: Boolean)
    @Serializable data class IANegocio(val iaNegocio: Boolean = false)

    /** La dueña prende la IA del negocio (Anthropic es un proveedor nuevo: lo acepta ella). */
    fun prenderIA() = alcance.launch {
        prendiendo = true
        try {
            estado.api.enviar<IANegocio, Activa>("PATCH", "/v1/panel/asistente/ia", Activa(true))
            Haptico.exito()
            cargar()
        } catch (e: CancellationException) {
            throw e
        } catch (e: Exception) {
            mensajes = mensajes + Mensaje(rol = "asistente", texto = mensajeDe(e), error = true)
        } finally {
            prendiendo = false
        }
    }

    /** Abre el turno de una fila: se cierra la hoja y Principal lo abre (como un push). */
    fun abrirTurno(id: String) {
        // El detalle se pide cuando esta hoja terminó de bajar (como en iOS).
        nav.despuesDeCerrar = { estado.rutaPendiente = ar.com.tengolibre.app.app.Ruta.turno(id) }
        cerrar()
    }

    suspend fun preguntar(q: String) {
        if (pensando) return
        val pregunta = q.take(500)
        // Los últimos 8 turnos de esta hoja, sin los errores (la API recorta cada uno a 600).
        val historial = mensajes.filter { !it.error }.takeLast(8).map { TurnoChat(it.rol, it.texto.take(600)) }
        mensajes = mensajes + Mensaje(rol = "usuario", texto = pregunta)
        pensando = true
        try {
            val r = estado.api.enviar<RespuestaAsistente, Pregunta>("POST", "/v1/panel/asistente", Pregunta(pregunta, historial))
            r.tope?.let { tope = it }
            // Con tope, la respuesta es el mismo aviso: va arriba del campo y no se repite en la charla.
            if (r.tope?.mensaje != r.respuesta || !r.propuestas.isNullOrEmpty()) {
                mensajes = mensajes + Mensaje(rol = "asistente", texto = r.respuesta, propuestas = r.propuestas ?: emptyList(),
                    encabezado = r.encabezado, items = r.items.orEmpty())
            }
            if (r.origen == "ia") cargar()
        } catch (e: CancellationException) {
            throw e
        } catch (e: Exception) {
            mensajes = mensajes + Mensaje(rol = "asistente", texto = mensajeDe(e), error = true)
        } finally {
            pensando = false
        }
    }

    fun enviar() {
        val q = texto.trim()
        if (q.isEmpty() || pensando) return
        texto = ""
        alcance.launch { preguntar(q) }
    }

    /** El detalle del error va a la charla; si la propuesta ya no se puede usar, queda apagada. */
    fun fallo(p: PropuestaAsistente, e: Exception) {
        val x = e as? ErrorApi ?: ErrorApi.desconocido
        Haptico.error()
        if (x.codigo == "propuesta_vencida") resueltas = resueltas + (p.id to "Vencida")
        if (x.codigo == "propuesta_resuelta") resueltas = resueltas + (p.id to "Ya resuelta")
        mensajes = mensajes + Mensaje(rol = "asistente", texto = x.detalle, error = true)
    }

    fun confirmar(p: PropuestaAsistente) = alcance.launch {
        resolviendo = p.id
        try {
            estado.api.enviar<RespuestaAsistente, Cual>("POST", "/v1/panel/asistente/confirmar", Cual(p.id))
            // La tarjeta con «Hecho» alcanza: antes se repetía la misma oración en una burbuja.
            resueltas = resueltas + (p.id to "Hecho")
            Haptico.exito()
            // La agenda cambió: Hoy y las demás listas se vuelven a pedir.
            estado.version += 1
        } catch (e: CancellationException) {
            throw e
        } catch (e: Exception) {
            fallo(p, e)
        } finally {
            resolviendo = null
        }
    }

    fun descartar(p: PropuestaAsistente) = alcance.launch {
        resolviendo = p.id
        try {
            estado.api.enviar<RespuestaAsistente, Cual>("POST", "/v1/panel/asistente/descartar", Cual(p.id))
            resueltas = resueltas + (p.id to "Sin cambios")
        } catch (e: CancellationException) {
            throw e
        } catch (e: Exception) {
            fallo(p, e)
        } finally {
            resolviendo = null
        }
    }

    BarraHoja("Asistente", cerrar, textoCancelar = "Cerrar")
    Column(Modifier.weight(1f).fillMaxWidth().verticalScroll(scroll).padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        if (mensajes.isEmpty()) {
            Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Texto("¿En qué te ayudo?", Tipo.lato(22f, Peso.Black), C.tinta)
                Texto("Preguntá por tu agenda o pedí un cambio. Antes de tocar un turno, te lo muestro para que lo confirmes.", Tipo.lato(14f), C.sub)
            }
            val s = info?.sugerencias.orEmpty()
            if (s.isNotEmpty()) {
                TextoRotulo("PROBÁ CON", modifier = Modifier.padding(top = 4.dp))
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    s.forEach { q ->
                        Chip(q, "sparkles", modifier = Modifier.clip(CircleShape).clickable(enabled = !pensando) { alcance.launch { preguntar(q) } })
                    }
                }
            }
        }
        mensajes.forEachIndexed { i, m ->
            // TalkBack lee lo último que contestó (la web lo hace con aria-live, iOS con un anuncio).
            val ultima = i == mensajes.lastIndex && m.rol == "asistente"
            Box(if (ultima) Modifier.semantics { liveRegion = LiveRegionMode.Polite } else Modifier) {
                Burbuja(m, resueltas, resolviendo, { confirmar(it) }, { descartar(it) }, { abrirTurno(it) })
            }
        }
        if (pensando) {
            Texto("Pensando…", Tipo.lato(15f), C.sub, Modifier.semantics { liveRegion = LiveRegionMode.Polite }
                .clip(RoundedCornerShape(18.dp)).background(C.chip).padding(horizontal = 14.dp, vertical = 10.dp))
        }
    }

    // Abajo: tope, campo y cupo.
    Column(Modifier.fillMaxWidth().background(C.hoja).padding(start = 16.dp, end = 16.dp, top = 8.dp, bottom = 8.dp), verticalArrangement = Arrangement.spacedBy(8.dp), horizontalAlignment = Alignment.CenterHorizontally) {
        tope?.let { t ->
            Row(Modifier.fillMaxWidth().clip(RoundedCornerShape(12.dp)).background(C.pagoFondo).padding(10.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Icono("exclamationmark.triangle", 15f, C.pagoTinta)
                Texto(t.mensaje, Tipo.lato(13f), C.pagoTinta)
            }
        }
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.Bottom, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            // El dictado es el del teclado: no hace falta pedir el micrófono.
            Box(Modifier.weight(1f).clip(RoundedCornerShape(22.dp)).background(C.campo).padding(horizontal = 14.dp, vertical = 12.dp)) {
                if (texto.isEmpty()) Texto("Escribí o dictá tu pregunta", Tipo.lato(16f), C.sub.copy(alpha = 0.7f), Modifier.clearAndSetSemantics {}, lineas = 1)
                BasicTextField(
                    texto, { texto = it }, Modifier.fillMaxWidth().semantics { contentDescription = "Escribí o dictá tu pregunta" },
                    textStyle = Tipo.lato(16f).copy(color = C.tinta),
                    minLines = 1, maxLines = 4,
                    keyboardOptions = KeyboardOptions(capitalization = KeyboardCapitalization.Sentences, imeAction = ImeAction.Send),
                    keyboardActions = KeyboardActions(onSend = { enviar() }),
                    cursorBrush = SolidColor(C.mentaTexto),
                )
            }
            Box(
                Modifier.size(44.dp).alpha(if (puedeEnviar) 1f else 0.45f).clip(CircleShape).background(C.menta)
                    .clickable(enabled = puedeEnviar) { enviar() }.semantics { contentDescription = "Enviar" },
                contentAlignment = Alignment.Center,
            ) { Icono("arrow.up", 18f, C.petroleoHondo) }
        }
        val i = info
        if (i != null && i.iaPlataforma && i.puedePrender && !i.iaNegocio) {
            Row(Modifier.fillMaxWidth().clip(RoundedCornerShape(12.dp)).background(C.chip).padding(10.dp),
                verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                Texto("Con IA (Claude) entiende preguntas más libres. Nunca le mandamos notas ni datos de salud.", Tipo.lato(13f), C.tinta, Modifier.weight(1f))
                BotonContorno(if (prendiendo) "Prendiendo…" else "Prender la IA", { prenderIA() }, alto = 40.dp, tam = 14f, lleno = false, habilitado = !prendiendo)
            }
        }
        val c = info?.cupo
        if (c != null && info?.ia == true && c.cupoMensual > 0) {
            Texto("IA: ${minOf(c.usadasMes, c.cupoMensual)} de ${c.cupoMensual} respuestas este mes", Tipo.lato(12f, numeros = true), C.sub, lineas = 1)
        }
    }
}

@Composable
private fun Burbuja(m: Mensaje, resueltas: Map<String, String>, resolviendo: String?, alConfirmar: (PropuestaAsistente) -> Unit,
    alCancelar: (PropuestaAsistente) -> Unit, alAbrir: (String) -> Unit) {
    val mio = m.rol == "usuario"
    Column(
        Modifier.fillMaxWidth().padding(start = if (mio) 48.dp else 0.dp),
        horizontalAlignment = if (mio) Alignment.End else Alignment.Start, verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        if (m.items.isNotEmpty()) {
            // Una lista de turnos: una fila por turno (antes, renglones de texto que se partían en dos).
            Column(Modifier.fillMaxWidth().clip(RoundedCornerShape(18.dp)).background(C.chip).padding(horizontal = 14.dp, vertical = 10.dp)) {
                Texto(m.encabezado ?: m.texto, Tipo.lato(15f, Peso.Bold), C.tinta, Modifier.padding(bottom = 6.dp))
                m.items.forEachIndexed { i, t ->
                    Row(
                        Modifier.fillMaxWidth().heightIn(min = 48.dp).clickable(role = androidx.compose.ui.semantics.Role.Button) { alAbrir(t.id) }
                            .semantics(mergeDescendants = true) {},
                        verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp),
                    ) {
                        Texto(t.hora, Tipo.lato(15f, Peso.Bold, numeros = true), C.tinta, Modifier.width(48.dp), lineas = 1)
                        Column(Modifier.weight(1f)) {
                            Texto(t.cliente, Tipo.lato(15f, Peso.Bold), C.tinta, lineas = 1)
                            Texto(listOfNotNull(t.servicio, t.profesional, t.estado).filter { it.isNotEmpty() }.joinToString(" · "), Tipo.lato(13f), C.sub, lineas = 1)
                        }
                        Icono("chevron.right", 13f, C.sub)
                    }
                    if (i < m.items.lastIndex) Box(Modifier.fillMaxWidth().height(1.dp).background(C.linea))
                }
            }
        } else SelectionContainer {
            Texto(
                m.texto, Tipo.lato(15f).copy(lineHeight = 20.sp), if (mio) C.petroleoHondo else if (m.error) C.ausenteTinta else C.tinta,
                Modifier.clip(RoundedCornerShape(18.dp)).background(if (mio) C.menta else if (m.error) C.ausenteFondo else C.chip).padding(horizontal = 14.dp, vertical = 10.dp),
            )
        }
        m.propuestas.forEach { p -> TarjetaPropuesta(p, resueltas[p.id], resolviendo, alConfirmar, alCancelar) }
    }
}

/** Una propuesta: el resumen y los dos botones; resuelta, queda apagada con lo que pasó. */
@Composable
private fun TarjetaPropuesta(p: PropuestaAsistente, fin: String?, resolviendo: String?, alConfirmar: (PropuestaAsistente) -> Unit, alCancelar: (PropuestaAsistente) -> Unit) {
    val forma = RoundedCornerShape(18.dp)
    Column(
        Modifier.fillMaxWidth().alpha(if (fin == null) 1f else 0.7f).clip(forma).background(C.fondo).contorno(18.dp).padding(14.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        TextoRotulo("PARA CONFIRMAR")
        Texto(p.resumen, Tipo.lato(15f).copy(lineHeight = 20.sp), C.tinta)
        if (fin != null) {
            PastillaEstado(fin, if (fin == "Hecho") "checkmark" else "minus", if (fin == "Hecho") C.mentaSuave else C.chip, if (fin == "Hecho") C.mentaTexto else C.sub, alto = 28.dp)
        } else {
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
                BotonMenta(if (resolviendo == p.id) "Un momento…" else p.textoBoton, { alConfirmar(p) }, Modifier.weight(1f), alto = 44.dp, tam = 15f, conSombra = false, habilitado = resolviendo == null)
                BotonTexto("No, dejalo así", { if (resolviendo == null) alCancelar(p) }, color = C.sub)
            }
        }
    }
}
