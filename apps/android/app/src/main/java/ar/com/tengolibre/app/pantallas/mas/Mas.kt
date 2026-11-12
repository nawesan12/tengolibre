package ar.com.tengolibre.app.pantallas.mas

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import ar.com.tengolibre.app.app.Estado
import android.content.Intent
import android.provider.Settings
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalContext
import ar.com.tengolibre.app.BuildConfig
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.nucleo.ErrorApi
import ar.com.tengolibre.app.nucleo.Web
import ar.com.tengolibre.app.pantallas.buscar.CampoBuscar
import ar.com.tengolibre.app.pantallas.hoy.Arranque
import ar.com.tengolibre.app.pantallas.hoy.BarraHoja
import ar.com.tengolibre.app.pantallas.principal.*
import kotlinx.coroutines.launch
import ar.com.tengolibre.app.tema.*

// Clon de Pantallas/Mas/Mas.swift: Más, las piezas comunes de las secciones, Primeros pasos y Cuenta.

/** Una tarjeta con rótulo y renglones separados por línea. */
@Composable
fun Bloque(titulo: String = "", pie: String = "", vararg filas: @Composable () -> Unit) {
    Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(8.dp)) {
        if (titulo.isNotEmpty()) TextoRotulo(titulo, modifier = Modifier.padding(horizontal = 4.dp))
        Column(Modifier.fillMaxWidth().tarjeta()) {
            filas.forEachIndexed { i, f ->
                if (i > 0) Box(Modifier.padding(start = 16.dp).fillMaxWidth().height(1.dp).background(C.linea))
                f()
            }
        }
        if (pie.isNotEmpty()) Texto(pie, Tipo.lato(13f), C.sub, Modifier.padding(horizontal = 4.dp))
    }
}

/** Un renglón «título … valor» dentro de un Bloque. */
@Composable
fun Renglon(titulo: String, detalle: String = "", simbolo: String? = null, derecha: @Composable RowScope.() -> Unit = {}) {
    Row(Modifier.fillMaxWidth().heightIn(min = 56.dp).padding(horizontal = 16.dp, vertical = 10.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
        if (simbolo != null) Box(Modifier.width(26.dp), contentAlignment = Alignment.Center) { Icono(simbolo, 18f, C.mentaTexto) }
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Texto(titulo, Tipo.lato(16f), C.tinta)
            if (detalle.isNotEmpty()) Texto(detalle, Tipo.lato(13f), C.sub)
        }
        derecha()
    }
}

/** Un número que se cambia con − y + (minutos, horas, días, porcentajes). */
@Composable
fun Numerito(titulo: String, valor: Int, alCambiar: (Int) -> Unit, rango: IntRange, detalle: String = "", paso: Int = 1, unidad: (Int) -> String = { "$it" }) {
    Renglon(titulo, detalle) {
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
            val puedeMenos = valor > rango.first; val puedeMas = valor < rango.last
            Box(Modifier.size(36.dp).clip(CircleShape).background(C.chip).clickable(enabled = puedeMenos) { alCambiar(maxOf(rango.first, valor - paso)); Haptico.seleccion() }.alpha(if (puedeMenos) 1f else 0.4f), contentAlignment = Alignment.Center) { Icono("minus", 17f) }
            Texto(unidad(valor), Tipo.lato(15f, Peso.Bold, numeros = true), C.tinta, Modifier.widthIn(min = 64.dp), alinear = TextAlign.Center)
            Box(Modifier.size(36.dp).clip(CircleShape).background(C.chip).clickable(enabled = puedeMas) { alCambiar(minOf(rango.last, valor + paso)); Haptico.seleccion() }.alpha(if (puedeMas) 1f else 0.4f), contentAlignment = Alignment.Center) { Icono("plus", 17f) }
        }
    }
}

/** Un interruptor en un Bloque. */
@Composable
fun RenglonInterruptor(titulo: String, activo: Boolean, alCambiar: (Boolean) -> Unit, detalle: String = "") {
    Box(Modifier.padding(horizontal = 16.dp, vertical = 0.dp)) { Interruptor(titulo, activo, alCambiar, detalle) }
}

/** Un campo de texto dentro de un Bloque. */
@Composable
fun RenglonCampo(titulo: String, texto: String, alCambiar: (String) -> Unit, placeholder: String = "", teclado: KeyboardType = KeyboardType.Text, multilinea: Boolean = false) {
    val tinta = C.tinta
    Column(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 10.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
        Texto(titulo, Tipo.lato(13f, Peso.Bold), C.sub)
        Box {
            if (texto.isEmpty() && placeholder.isNotEmpty()) Texto(placeholder, Tipo.lato(16f), C.sub.copy(alpha = 0.6f))
            BasicTextField(texto, alCambiar, Modifier.fillMaxWidth(), textStyle = Tipo.lato(16f).copy(color = tinta),
                singleLine = !multilinea, minLines = if (multilinea) 2 else 1, maxLines = if (multilinea) 8 else 1,
                keyboardOptions = KeyboardOptions(keyboardType = teclado), cursorBrush = SolidColor(C.mentaTexto))
        }
    }
}

/** Un precio en pesos que se escribe como número. */
@Composable
fun RenglonPesos(titulo: String, valor: Int, alCambiar: (Int) -> Unit) {
    val tinta = C.tinta
    Renglon(titulo) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Texto("$", Tipo.lato(16f, Peso.Bold), C.sub)
            Spacer(Modifier.width(2.dp))
            BasicTextField(if (valor == 0) "" else "$valor", { t -> alCambiar(t.filter(Char::isDigit).take(9).toIntOrNull() ?: 0) },
                Modifier.widthIn(min = 40.dp, max = 120.dp), textStyle = Tipo.lato(16f, Peso.Bold, numeros = true).copy(color = tinta, textAlign = TextAlign.End),
                singleLine = true, keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number), cursorBrush = SolidColor(C.mentaTexto))
        }
    }
}

/** Botón de guardar fijo abajo de una hoja o sección. */
@Composable
fun BotonGuardar(accion: () -> Unit, texto: String = "Guardar", guardando: Boolean = false, habilitado: Boolean = true) {
    BotonMenta(if (guardando) "Guardando…" else texto, accion, alto = 52.dp, conSombra = false, habilitado = !guardando && habilitado)
}

/** Corre una escritura; si falla, el toast con el error. Devuelve si salió. */
suspend fun Estado.intentar(ok: String = "", f: suspend () -> Unit): Boolean = try {
    f()
    if (ok.isNotEmpty()) avisar(ok)
    true
} catch (e: kotlinx.coroutines.CancellationException) {
    throw e
} catch (e: Exception) {
    avisar(e)
    false
}

/** Lo que se apila dentro de «Más»: todas las secciones del panel web, nativas. */
sealed class DestinoMas {
    data object primerosPasos : DestinoMas(); data object resenas : DestinoMas(); data object suscripcion : DestinoMas()
    data object diasLibres : DestinoMas(); data object cobros : DestinoMas()
    data object servicios : DestinoMas(); data object equipo : DestinoMas(); data class profesional(val id: String) : DestinoMas(); data object miHorario : DestinoMas()
    data object membresias : DestinoMas(); data object cupones : DestinoMas(); data object referidos : DestinoMas(); data object penalidades : DestinoMas()
    data object etiquetas : DestinoMas(); data object caja : DestinoMas()
    data object reportes : DestinoMas(); data object miWeb : DestinoMas(); data object plantillas : DestinoMas(); data object dominio : DestinoMas()
    data object integraciones : DestinoMas()
    data object configuracion : DestinoMas(); data object avisosClientes : DestinoMas(); data object whatsapp : DestinoMas(); data object auditoria : DestinoMas()
    data object sucursales : DestinoMas(); data object cuenta : DestinoMas(); data object recorrido : DestinoMas()
}

private data class ItemMas(val destino: DestinoMas, val simbolo: String, val titulo: String, val detalle: String = "", val pendiente: Boolean = false)

/** Más: el negocio arriba y todas las secciones agrupadas, recortadas según el rol (como la web). */
@Composable
fun Mas() {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    val v = estado.vivo
    LaunchedEffect(estado.version) { v.cargar(estado) }

    /** El estado de cada sección, en vivo («Sin conectar», «3 socias activas», «IA apagada»…). */
    fun conEstado(it: ItemMas): ItemMas {
        val t = when (it.destino) {
            DestinoMas.cobros -> v.textoCobros
            DestinoMas.integraciones -> if (estado.esProfesional) null else v.textoIA
            DestinoMas.membresias -> v.textoMembresias
            DestinoMas.cupones -> v.textoCupones
            DestinoMas.avisosClientes -> v.textoRecordatorios
            DestinoMas.miWeb -> v.textoWeb
            DestinoMas.dominio -> v.textoDominio
            DestinoMas.whatsapp -> v.textoWhatsApp
            else -> null
        }
        return if (t != null && t.first.isNotEmpty()) it.copy(detalle = t.first, pendiente = t.second) else it
    }

    val g = estado.gestiona; val mostrador = g || estado.rol == "recepcion"
    val grupos = buildList {
        if (estado.esProfesional) {
            add("Tu trabajo" to listOf(ItemMas(DestinoMas.miHorario, "clock", "Mi horario", "Horario, bloqueos y días libres"),
                ItemMas(DestinoMas.integraciones, "calendar.badge.plus", "Mi agenda en el calendario", "Google, Apple u Outlook")))
        } else {
            add("Tu negocio" to buildList {
                if (g) add(ItemMas(DestinoMas.servicios, "sparkles", "Servicios", "Precios, duración, categorías y productos"))
                add(ItemMas(DestinoMas.equipo, "person.2.badge.gearshape", "Equipo y horarios", "Profesionales, horarios e invitaciones"))
                if (g) add(ItemMas(DestinoMas.diasLibres, "beach.umbrella", "Días libres", "Pedidos del equipo"))
                add(ItemMas(DestinoMas.resenas, "star.bubble", "Reseñas"))
                if (g) add(ItemMas(DestinoMas.reportes, "chart.bar", "Reportes", "Ingresos, ocupación y horas muertas"))
            })
            add("Plata" to buildList {
                add(ItemMas(DestinoMas.caja, "banknote", "Caja", "Lo cobrado y lo que deben"))
                if (g) add(ItemMas(DestinoMas.cobros, "creditcard", "Pagos y señas", "Mercado Pago y cobro por adelantado"))
                if (mostrador) { add(ItemMas(DestinoMas.membresias, "ticket", "Membresías", "Planes y socias")); add(ItemMas(DestinoMas.cupones, "tag", "Cupones")) }
                if (g) add(ItemMas(DestinoMas.referidos, "gift", "Referidos"))
            })
            if (mostrador) add("Clientes" to listOf(ItemMas(DestinoMas.etiquetas, "tag.circle", "Etiquetas"), ItemMas(DestinoMas.penalidades, "exclamationmark.shield", "Penalidades y ausencias"),
                ItemMas(DestinoMas.avisosClientes, "envelope.badge", "Recordatorios a clientes", "Cumpleaños y «te extrañamos»")))
            if (g) {
                add("En la web" to listOf(ItemMas(DestinoMas.miWeb, "globe", "Mi web", "Datos, fotos, logo y portada"), ItemMas(DestinoMas.plantillas, "paintpalette", "Plantillas", "Cómo se ve tu página"),
                    ItemMas(DestinoMas.dominio, "link", "Dominio propio"), ItemMas(DestinoMas.integraciones, "point.3.connected.trianglepath.dotted", "Integraciones e IA", "Calendarios, Claude, ChatGPT")))
                add("Ajustes" to buildList {
                    add(ItemMas(DestinoMas.configuracion, "gearshape", "Configuración", "Reservas, cancelaciones y avisos"))
                    add(ItemMas(DestinoMas.whatsapp, "message", "WhatsApp", "Confirmaciones y recordatorios"))
                    add(ItemMas(DestinoMas.auditoria, "list.bullet.rectangle", "Actividad", "Quién hizo qué"))
                    if (estado.esDuena) { add(ItemMas(DestinoMas.suscripcion, "crown", "Suscripción")); add(ItemMas(DestinoMas.sucursales, "building.2", "Sucursales")) }
                })
            }
        }
        add("Cuenta" to listOf(ItemMas(DestinoMas.recorrido, "play.circle", "Qué podés hacer", "El recorrido y cómo sumar el widget"),
            ItemMas(DestinoMas.cuenta, "person.crop.circle", "Tu cuenta", estado.persona?.email.orEmpty())))
    }

    Refrescable({ v.cargar(estado) }, Modifier.fillMaxSize().background(C.fondo)) { Column(Modifier.fillMaxSize().background(C.fondo).verticalScroll(rememberScrollState())) {
        Cabecera("Más", estado.negocio?.let { "${it.nombre} · ${it.rolTexto}" } ?: "", conNegocio = true,
            acciones = listOf(AccionCabecera("square.and.arrow.up", "Compartir tu link") { nav.compartir = true })) { CampoBuscar() }
        Cuerpo {
            if (estado.gestiona) {
                val avance = v.avance(estado)
                Box(Modifier.fillMaxWidth().tarjeta().clickable { nav.pila.add(DestinoMas.primerosPasos) }) {
                    FilaLista("chart.line.uptrend.xyaxis", "Potenciá tu negocio",
                        if (v.cargado) (if (avance >= 1) "Estás usando todo" else "Al ${Math.round(avance * 100)} % · ver lo que falta") else "Lo que te trae más reservas y más plata",
                        pendiente = v.cargado && avance < 1)
                }
            }
            val pagina = Web.pagina(estado.negocio?.slug.orEmpty())
            Box(Modifier.fillMaxWidth().tarjeta().clickable { estado.abrirPestana(pagina) }) { FilaLista("safari", "Ver mi página", pagina.removePrefix("https://"), final = "arrow.up.right") }
            grupos.forEachIndexed { gi, (t, items) ->
                Box(Modifier.entrada(gi)) { Rotulo(t) }
                Column(Modifier.entrada(gi).fillMaxWidth().tarjeta()) {
                    items.map(::conEstado).forEachIndexed { i, it ->
                        if (i > 0) Box(Modifier.padding(start = 56.dp).fillMaxWidth().height(1.dp).background(C.linea))
                        Box(Modifier.clickable { nav.pila.add(it.destino) }) { FilaLista(it.simbolo, it.titulo, it.detalle, pendiente = it.pendiente) }
                    }
                }
            }
            Texto("Tengo Libre ${BuildConfig.VERSION_NAME} (${BuildConfig.VERSION_CODE})", Tipo.lato(12f), C.sub, Modifier.fillMaxWidth().padding(top = 8.dp), alinear = TextAlign.Center)
        }
    } }
}

/** Una sección de «Más»: cabecera con «‹ Más», cuerpo, cargando / error. */
@Composable
fun PantallaMas(
    titulo: String,
    subtitulo: String = "",
    volver: String = "Más",
    acciones: List<AccionCabecera> = emptyList(),
    cargando: Boolean = false,
    error: String = "",
    recargar: suspend () -> Unit,
    abajo: @Composable ColumnScope.() -> Unit = {},
    contenido: @Composable ColumnScope.() -> Unit,
) {
    val alcance = rememberCoroutineScope()
    Column(Modifier.fillMaxSize().background(C.fondo).verticalScroll(rememberScrollState())) {
        Cabecera(titulo, subtitulo, volver = volver, acciones = acciones, abajo = abajo)
        Cuerpo {
            if (error.isNotEmpty()) ErrorConSalida("No pudimos traer esto", error) { alcance.launch { recargar() } }
            else if (cargando) repeat(4) { Esqueleto(72.dp) }
            else contenido()
        }
    }
}

/** Una hoja de edición con Cancelar / Guardar (y Borrar, si corresponde). Va adentro de una Hoja. */
@Composable
fun ColumnScope.HojaEdicion(
    titulo: String,
    cerrar: () -> Unit,
    guardar: String = "Guardar",
    habilitado: Boolean = true,
    borrar: (suspend () -> Unit)? = null,
    accion: suspend () -> Boolean,
    contenido: @Composable ColumnScope.() -> Unit,
) {
    val alcance = rememberCoroutineScope()
    var guardando by remember { mutableStateOf(false) }
    var confirmarBorrar by remember { mutableStateOf(false) }
    BarraHoja(titulo, cerrar, accion = if (guardando) "…" else guardar, habilitada = habilitado && !guardando) {
        alcance.launch { guardando = true; val ok = accion(); guardando = false; if (ok) { Haptico.exito(); cerrar() } }
    }
    Column(Modifier.weight(1f).background(C.fondo).verticalScroll(rememberScrollState()).padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        contenido()
        if (borrar != null) {
            Row(
                Modifier.fillMaxWidth().heightIn(min = 48.dp).clip(CircleShape).contorno(999.dp).clickable { confirmarBorrar = true },
                horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically,
            ) { Icono("trash", 15f, C.error); Spacer(Modifier.width(8.dp)); Texto("Borrar", Tipo.lato(15f, Peso.Bold), C.error) }
        }
    }
    Confirmacion(confirmarBorrar, "¿Seguro?", "", listOf(OpcionConfirmacion("Borrar", destructiva = true) { alcance.launch { borrar?.invoke(); cerrar() } }), { confirmarBorrar = false })
}

// MARK: - Primeros pasos

@Composable
fun PrimerosPasos() {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    var a by remember { mutableStateOf<Arranque?>(null) }
    var error by remember { mutableStateOf("") }
    fun destino(clave: String): DestinoMas? = when (clave) {
        "servicios", "servicio" -> DestinoMas.servicios
        "profesionales", "profesional", "horario", "horarios", "equipo" -> DestinoMas.equipo
        "web", "mi_web", "miweb", "logo", "fotos", "datos" -> DestinoMas.miWeb
        "cobros", "mercadopago", "pagos" -> DestinoMas.cobros
        "suscripcion" -> DestinoMas.suscripcion
        "plantilla" -> DestinoMas.plantillas
        else -> null
    }
    suspend fun cargar() { try { a = estado.api.get("/v1/panel/arranque"); error = "" } catch (e: ErrorApi) { error = e.detalle } catch (e: Exception) { error = "Probá de nuevo." } }
    LaunchedEffect(estado.version) { cargar() }
    val ar = a
    PantallaMas("Primeros pasos", ar?.let { if (it.completo) "Listo: ya te pueden reservar" else "${it.pasos.count { p -> p.listo }} de ${it.pasos.size} listos" } ?: "",
        cargando = ar == null && error.isEmpty(), error = error, recargar = { cargar() }) {
        if (ar != null) {
            Bloque("", "", *ar.pasos.map { p -> @Composable {
                val d = destino(p.clave)
                Box(Modifier.then(if (d != null && !p.listo) Modifier.clickable { nav.pila.add(d) } else Modifier).alpha(if (p.listo) 0.6f else 1f)) {
                    Renglon(p.titulo, if (p.obligatorio) "Hace falta para que te reserven" else "Recomendado", if (p.listo) "checkmark.circle.fill" else "circle") {
                        if (!p.listo && d != null) Icono("chevron.right", 15f, C.sub)
                    }
                }
            } }.toTypedArray())
            if (ar.reservable) Row(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalAlignment = Alignment.CenterVertically) {
                Icono("checkmark.seal", 15f, C.mentaTexto); Texto("Tu página ya recibe reservas.", Tipo.lato(15f, Peso.Bold), C.mentaTexto)
            }
        }
    }
}

// MARK: - Cuenta

@Composable
fun Cuenta() {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    val contexto = LocalContext.current
    var salir by remember { mutableStateOf(false) }
    var preferencias by remember { mutableStateOf(false) }
    val avisosOn = remember { EstadoNegocio.hayAvisos() }
    if (preferencias) { PreferenciasPush(volver = { preferencias = false }); return }
    PantallaMas("Tu cuenta", estado.persona?.email.orEmpty(), recargar = {}) {
        Bloque("", "", { Renglon(estado.persona?.nombre.orEmpty(), estado.persona?.email.orEmpty(), "person.crop.circle") })
        Bloque("Tus negocios", "", *estado.negocios.map { n -> @Composable {
            Box(Modifier.clickable { estado.usar(n); alcance.launch { estado.refrescar() } }) {
                Renglon(n.nombre, n.rolTexto, if (n.id == estado.negocio?.id) "checkmark.circle.fill" else "circle")
            }
        } }.toTypedArray())
        Bloque("Avisos en este teléfono", "",
            { Box(Modifier.clickable { contexto.startActivity(Intent(Settings.ACTION_APP_NOTIFICATION_SETTINGS).putExtra(Settings.EXTRA_APP_PACKAGE, contexto.packageName)) }) {
                Renglon(if (avisosOn) "Activados" else "Apagados", "Cambialo en Ajustes", if (avisosOn) "bell.badge" else "bell.slash")
            } },
            { Box(Modifier.clickable { preferencias = true }) { Renglon("Qué te avisamos", simbolo = "slider.horizontal.3") { Icono("chevron.right", 15f, C.sub) } } })
        Bloque("Modo", "", { Box(Modifier.clickable { alcance.launch { estado.elegirModo("cliente") } }) {
            Renglon("Usar como clienta", "Ver tus turnos y los lugares donde reservás", "person.2.wave.2")
        } })
        Bloque("Ayuda", "", { Box(Modifier.clickable { estado.abrirLink("mailto:hola@tengolibre.com.ar") }) { Renglon("Escribinos", "hola@tengolibre.com.ar", "questionmark.bubble") } })
        Row(
            Modifier.fillMaxWidth().heightIn(min = 52.dp).clip(CircleShape).contorno(999.dp).clickable { salir = true },
            horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically,
        ) { Icono("rectangle.portrait.and.arrow.right", 16f, C.error); Spacer(Modifier.width(8.dp)); Texto("Cerrar sesión", Tipo.lato(16f, Peso.Bold), C.error) }
        BorrarCuenta()
    }
    Confirmacion(salir, "¿Cerrar sesión en este teléfono?", "", listOf(OpcionConfirmacion("Cerrar sesión", destructiva = true) { alcance.launch { estado.salir() } }), { salir = false })
}

// MARK: - Borrar la cuenta

/** «Borrar mi cuenta» (para cualquier rol y también en modo clienta): pide escribir el mail de la cuenta, la desactiva
 *  con DELETE /v1/auth/cuenta y vuelve al ingreso. Los datos personales se borran a los 30 días; hasta entonces se
 *  recupera escribiendo a hola@. */
@Composable
fun BorrarCuenta() {
    var abierta by remember { mutableStateOf(false) }
    Box(Modifier.fillMaxWidth(), contentAlignment = Alignment.Center) { BotonTexto("Borrar mi cuenta", { abierta = true }, color = C.error) }
    Hoja(abierta, { abierta = false }) { HojaBorrarCuenta { abierta = false } }
}

@Composable
fun ColumnScope.HojaBorrarCuenta(cerrar: () -> Unit) {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    var escrito by remember { mutableStateOf("") }
    var borrando by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf("") }
    val email = estado.persona?.email.orEmpty()
    val coincide = email.isNotEmpty() && escrito.trim().lowercase() == email.lowercase()

    fun borrar() {
        if (!coincide || borrando) return
        borrando = true
        error = ""
        alcance.launch {
            try {
                estado.borrarCuenta()
                Haptico.exito()
                cerrar()
            } catch (e: kotlinx.coroutines.CancellationException) {
                throw e
            } catch (e: Exception) {
                error = (e as? ErrorApi)?.detalle ?: ErrorApi.desconocido.detalle
                Haptico.error()
            } finally {
                borrando = false
            }
        }
    }

    BarraHoja("Borrar mi cuenta", cerrar)
    Column(
        Modifier.weight(1f).background(C.fondo).verticalScroll(rememberScrollState()).padding(20.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Box(Modifier.size(56.dp).clip(CircleShape).background(C.ausenteFondo), contentAlignment = Alignment.Center) { Icono("exclamationmark.triangle", 26f, C.error) }
        Texto("¿Borrar tu cuenta?", Tipo.lato(30f, Peso.Black), C.tinta, tracking = (-0.8).sp)
        Texto("Tu cuenta se desactiva ahora y tus datos personales se borran a los 30 días.", Tipo.lato(16f, Peso.Bold), C.tinta)
        Texto("En esos 30 días la recuperás escribiendo a hola@tengolibre.com.ar. Después no se puede deshacer.", Tipo.lato(16f), C.sub)
        Campo("Para confirmar, escribí tu mail", escrito, { escrito = it }, ayuda = email, teclado = KeyboardType.Email)
        if (error.isNotEmpty()) Texto(error, Tipo.lato(14f, Peso.Bold), C.error)
        Row(
            Modifier.atenuado(!coincide || borrando).fillMaxWidth().heightIn(min = 52.dp).clip(CircleShape).background(C.error)
                .clickable(enabled = coincide && !borrando) { borrar() },
            horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically,
        ) { Texto(if (borrando) "Borrando…" else "Borrar mi cuenta", Tipo.lato(17f, Peso.Black), androidx.compose.ui.graphics.Color.White, lineas = 1) }
    }
}
