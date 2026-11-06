package ar.com.tengolibre.app.pantallas.mas

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.text.BasicTextField
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
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.nucleo.*
import ar.com.tengolibre.app.pantallas.hoy.AReprogramar
import ar.com.tengolibre.app.pantallas.hoy.PedidoDias
import ar.com.tengolibre.app.pantallas.principal.*
import ar.com.tengolibre.app.pantallas.turno.Producto
import ar.com.tengolibre.app.tema.*
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable
import java.time.Instant
import java.time.LocalDate

// Clon de Pantallas/Mas/MasEquipo.swift.

@Serializable data class Categoria(val id: String, val nombre: String = "", val orden: Int = 0)
@Serializable data class Miembro(val id: String, val clerkId: String = "", val email: String = "", val nombre: String = "", val rol: String = "", val activo: Boolean = true, val creadoEn: FechaApi = Instant.EPOCH)
@Serializable data class InvitacionEquipo(val id: String, val email: String = "", val rol: String = "", val invito: String = "", val venceEn: FechaApi = Instant.EPOCH, val aceptadaEn: FechaApi? = null, val creadoEn: FechaApi = Instant.EPOCH)
@Serializable data class Bloqueo(val id: String, val fecha: String = "", val desde: String? = null, val hasta: String? = null, val motivo: String? = null)

val rolesEquipo = listOf("encargado" to "Encargada", "recepcion" to "Recepción", "profesional" to "Profesional")
fun nombreRol(r: String) = mapOf("duenio" to "Dueña", "encargado" to "Encargada", "recepcion" to "Recepción", "profesional" to "Profesional")[r] ?: r
private val NOMBRES_DIAS = listOf("Domingo", "Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado")
private fun hhmm(m: Int) = "%02d:%02d".format(m / 60, m % 60)

/** Un campo de texto con botón al lado, dentro de un Bloque (TextField + Button en un HStack). */
@Composable
private fun CampoConBoton(texto: String, alCambiar: (String) -> Unit, placeholder: String, boton: String, alTocar: () -> Unit) {
    Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp), verticalAlignment = Alignment.CenterVertically) {
        Box(Modifier.weight(1f)) {
            if (texto.isEmpty()) Texto(placeholder, Tipo.lato(16f), C.sub.copy(alpha = 0.7f))
            val tinta = C.tinta
            BasicTextField(texto, alCambiar, Modifier.fillMaxWidth(), singleLine = true, textStyle = Tipo.lato(16f).copy(color = tinta), cursorBrush = SolidColor(C.mentaTexto))
        }
        Box(Modifier.alpha(if (texto.isBlank()) 0.45f else 1f)) { BotonMentaChico(boton, { if (texto.isNotBlank()) alTocar() }) }
    }
}

// MARK: - Servicios (con categorías y productos)

@Composable
fun Servicios() {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    var servicios by remember { mutableStateOf<List<Servicio>>(emptyList()) }
    var categorias by remember { mutableStateOf<List<Categoria>>(emptyList()) }
    var productos by remember { mutableStateOf<List<Producto>>(emptyList()) }
    var tab by remember { mutableStateOf("servicios") }
    var cargando by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf("") }
    var editando by remember { mutableStateOf<Servicio?>(null) }
    var producto by remember { mutableStateOf<Producto?>(null) }
    var categoriaNueva by remember { mutableStateOf("") }

    @Serializable data class S(val servicios: List<Servicio> = emptyList())
    @Serializable data class Cs(val categorias: List<Categoria> = emptyList())
    @Serializable data class Ps(val productos: List<Producto> = emptyList())
    suspend fun cargar() {
        try {
            servicios = estado.api.get<S>("/v1/panel/servicios").servicios
            categorias = estado.api.opcional<Cs>("/v1/panel/categorias")?.categorias ?: emptyList()
            productos = estado.api.opcional<Ps>("/v1/panel/productos")?.productos ?: emptyList()
            error = ""
        } catch (e: ErrorApi) { error = e.detalle } catch (e: Exception) { error = "Probá de nuevo." }
        cargando = false
    }
    LaunchedEffect(estado.version) { cargar() }
    @Serializable data class Orden(val orden: Int)
    fun mover(c: Categoria, d: Int) = alcance.launch {
        val l = categorias.sortedBy { it.orden }.toMutableList()
        val i = l.indexOf(c); if (i + d !in l.indices) return@launch
        java.util.Collections.swap(l, i, i + d)
        l.forEachIndexed { n, x -> runCatching { estado.api.hacer("PATCH", "/v1/panel/categorias/${x.id}", Orden(n + 1)) } }
        cargar()
    }
    @Serializable data class Nombre(val nombre: String)

    PantallaMas("Servicios", "${servicios.count { it.activo }} activos",
        acciones = listOf(AccionCabecera("plus", "Agregar") { if (tab == "productos") producto = Producto("", "", 0, null, true, productos.size + 1) else editando = Servicio("", "", 30, 0, true) }),
        cargando = cargando, error = error, recargar = { cargar() },
        abajo = { Segmentado(listOf("servicios" to "Servicios", "categorias" to "Categorías", "productos" to "Productos"), tab, { tab = it }) }) {
        when (tab) {
            "categorias" -> Bloque("", "Agrupan los servicios en tu página (ej.: Cortes, Color, Uñas).", *(categorias.sortedBy { it.orden }.map { c -> @Composable {
                Renglon(c.nombre, Formato.cantidad(servicios.count { it.categoriaId == c.id }, "servicio", "servicios")) {
                    MenuBoton(listOf(
                        OpcionDeMenu("Subir", "arrow.up") { mover(c, -1) },
                        OpcionDeMenu("Bajar", "arrow.down") { mover(c, 1) },
                        OpcionDeMenu("Borrar", "trash", destructiva = true) { alcance.launch { estado.intentar { estado.api.hacer("DELETE", "/v1/panel/categorias/${c.id}") }; cargar() } },
                    ), "Opciones de ${c.nombre}") { Box(Modifier.size(44.dp), contentAlignment = Alignment.Center) { Icono("ellipsis", 17f) } }
                }
            } } + listOf(@Composable {
                CampoConBoton(categoriaNueva, { categoriaNueva = it }, "Nueva categoría", "Agregar") {
                    alcance.launch { estado.intentar { estado.api.hacer("POST", "/v1/panel/categorias", Nombre(categoriaNueva)) }; categoriaNueva = ""; cargar() }
                }
            })).toTypedArray())
            "productos" -> if (productos.isEmpty()) VacioQueEnsena("Sin productos", "Vendé productos en la caja", "Shampoo, esmaltes, lo que vendas: se suman al cobrar un turno.") {
                BotonMenta("Agregar un producto", { producto = Producto("", "", 0, null, true, productos.size + 1) }, alto = 52.dp, conSombra = false)
            } else Bloque("", "", *productos.sortedBy { it.orden }.map { p -> @Composable {
                Box(Modifier.clickable { producto = p }.alpha(if (p.activo) 1f else 0.55f)) {
                    Renglon(p.nombre, p.stock?.let { "Stock: $it" } ?: "Sin control de stock") { Texto(Formato.pesos(p.precio), Tipo.lato(15f, Peso.Bold, numeros = true), C.tinta) }
                }
            } }.toTypedArray())
            else -> {
                if (servicios.isEmpty()) VacioQueEnsena("Sin servicios", "Cargá tu primer servicio", "Con su duración y su precio. Es lo que tus clientes eligen al reservar.") {
                    BotonMenta("Agregar un servicio", { editando = Servicio("", "", 30, 0, true) }, alto = 52.dp, conSombra = false)
                }
                val grupos = categorias.sortedBy { it.orden }.map { c -> c.nombre to servicios.filter { it.categoriaId == c.id } } +
                    listOf("Sin categoría" to servicios.filter { s -> s.categoriaId == null || categorias.none { it.id == s.categoriaId } })
                grupos.filter { it.second.isNotEmpty() }.forEach { (t, l) ->
                    Bloque(if (categorias.isEmpty()) "" else t, "", *l.map { s -> @Composable {
                        Box(Modifier.clickable { editando = s }.alpha(if (s.activo) 1f else 0.55f)) {
                            Renglon(s.nombre, "${Formato.duracion(s.duracion)}${if (s.activo) "" else " · pausado"}") {
                                Texto(Formato.pesos(s.precio), Tipo.lato(15f, Peso.Bold, numeros = true), C.tinta); Icono("chevron.right", 13f, C.sub)
                            }
                        }
                    } }.toTypedArray())
                }
            }
        }
    }
    Hoja(editando != null, { editando = null }) { editando?.let { EditarServicio(it, categorias, listo = { alcance.launch { cargar() } }, cerrar = { editando = null }) } }
    Hoja(producto != null, { producto = null }) { producto?.let { EditarProducto(it, listo = { alcance.launch { cargar() } }, cerrar = { producto = null }) } }
}

@Composable
fun ColumnScope.EditarServicio(inicial: Servicio, categorias: List<Categoria>, listo: () -> Unit, cerrar: () -> Unit) {
    val estado = LocalEstado.current
    var s by remember { mutableStateOf(inicial) }
    var descripcion by remember { mutableStateOf(inicial.descripcion.orEmpty()) }
    @Serializable data class Cuerpo(val nombre: String, val descripcion: String, val duracion: Int, val precio: Int, val categoriaId: String?, val activo: Boolean)
    HojaEdicion(if (s.id.isEmpty()) "Servicio nuevo" else "Editar servicio", cerrar, habilitado = s.nombre.trim().length >= 2,
        borrar = if (s.id.isEmpty()) null else ({ estado.intentar("Sacamos el servicio") { estado.api.hacer("DELETE", "/v1/panel/servicios/${s.id}") }; estado.version += 1 }),
        accion = {
            val c = Cuerpo(s.nombre, descripcion, s.duracion, s.precio, s.categoriaId, s.activo)
            val ok = estado.intentar { if (s.id.isEmpty()) estado.api.hacer("POST", "/v1/panel/servicios", c) else estado.api.hacer("PATCH", "/v1/panel/servicios/${s.id}", c) }
            if (ok) { estado.version += 1; listo() }
            ok
        }) {
        Bloque("", "",
            { RenglonCampo("Nombre", s.nombre, { s = s.copy(nombre = it) }, "Corte y brushing") },
            { RenglonCampo("Descripción", descripcion, { descripcion = it }, "Opcional", multilinea = true) },
            { Numerito("Duración", s.duracion, { s = s.copy(duracion = it) }, 5..480, paso = 5) { Formato.duracion(it) } },
            { RenglonPesos("Precio", s.precio) { s = s.copy(precio = it) } })
        if (categorias.isNotEmpty()) Bloque("", "", { Renglon("Categoría") { SelectorMenu(listOf<Pair<String?, String>>(null to "Sin categoría") + categorias.map { it.id to it.nombre }, s.categoriaId, { s = s.copy(categoriaId = it) }) } })
        Bloque("", "Si lo pausás, no aparece en tu página ni se puede reservar.", { RenglonInterruptor("Activo", s.activo, { s = s.copy(activo = it) }) })
    }
}

@Composable
fun ColumnScope.EditarProducto(inicial: Producto, listo: () -> Unit, cerrar: () -> Unit) {
    val estado = LocalEstado.current
    var p by remember { mutableStateOf(inicial) }
    var conStock by remember { mutableStateOf(inicial.stock != null) }
    var stock by remember { mutableIntStateOf(inicial.stock ?: 0) }
    @Serializable data class Cuerpo(val nombre: String, val precio: Int, val stock: Int?, val activo: Boolean, val orden: Int)
    HojaEdicion(if (p.id.isEmpty()) "Producto nuevo" else "Editar producto", cerrar, habilitado = p.nombre.isNotBlank(),
        borrar = if (p.id.isEmpty()) null else ({ estado.intentar { estado.api.hacer("DELETE", "/v1/panel/productos/${p.id}") }; listo() }),
        accion = {
            val c = Cuerpo(p.nombre, p.precio, if (conStock) stock else null, p.activo, p.orden)
            val ok = estado.intentar { if (p.id.isEmpty()) estado.api.hacer("POST", "/v1/panel/productos", c) else estado.api.hacer("PATCH", "/v1/panel/productos/${p.id}", c) }
            if (ok) listo()
            ok
        }) {
        Bloque("", "", *buildList<@Composable () -> Unit> {
            add { RenglonCampo("Nombre", p.nombre, { p = p.copy(nombre = it) }) }
            add { RenglonPesos("Precio", p.precio) { p = p.copy(precio = it) } }
            add { RenglonInterruptor("Controlar stock", conStock, { conStock = it }) }
            if (conStock) add { Numerito("Stock", stock, { stock = it }, 0..9999) }
            add { RenglonInterruptor("A la venta", p.activo, { p = p.copy(activo = it) }) }
        }.toTypedArray())
    }
}

// MARK: - Equipo y horarios

@Composable
fun Equipo() {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    val alcance = rememberCoroutineScope()
    var profesionales by remember { mutableStateOf<List<Profesional>>(emptyList()) }
    var miembros by remember { mutableStateOf<List<Miembro>>(emptyList()) }
    var invitaciones by remember { mutableStateOf<List<InvitacionEquipo>>(emptyList()) }
    var servicios by remember { mutableStateOf<List<Servicio>>(emptyList()) }
    var cargando by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf("") }
    var nuevo by remember { mutableStateOf(false) }
    var invitar by remember { mutableStateOf(false) }

    @Serializable data class P(val profesionales: List<Profesional> = emptyList())
    @Serializable data class M(val miembros: List<Miembro> = emptyList())
    @Serializable data class I(val invitaciones: List<InvitacionEquipo> = emptyList())
    @Serializable data class S(val servicios: List<Servicio> = emptyList())
    suspend fun cargar() {
        try {
            profesionales = estado.api.get<P>("/v1/panel/profesionales").profesionales
            miembros = if (estado.gestiona) estado.api.opcional<M>("/v1/panel/equipo")?.miembros ?: emptyList() else emptyList()
            invitaciones = if (estado.gestiona) estado.api.opcional<I>("/v1/panel/invitaciones")?.invitaciones ?: emptyList() else emptyList()
            servicios = estado.api.opcional<S>("/v1/panel/servicios")?.servicios ?: emptyList()
            error = ""
        } catch (e: ErrorApi) { error = e.detalle } catch (e: Exception) { error = "Probá de nuevo." }
        cargando = false
    }
    LaunchedEffect(estado.version) { cargar() }
    @Serializable data class Rol(val rol: String)
    @Serializable data class Activo(val activo: Boolean)

    PantallaMas("Equipo y horarios", Formato.cantidad(profesionales.count { it.activo }, "profesional", "profesionales"),
        acciones = if (estado.gestiona) listOf(AccionCabecera("plus", "Sumar profesional") { nuevo = true }) else emptyList(),
        cargando = cargando, error = error, recargar = { cargar() }) {
        Bloque("Quién atiende", "Tocá a cada una para cargarle el horario, los servicios que hace y los días que no viene.", *profesionales.sortedBy { it.orden }.map { p -> @Composable {
            Row(Modifier.fillMaxWidth().heightIn(min = 64.dp).clickable { nav.pila.add(DestinoMas.profesional(p.id)) }.padding(horizontal = 16.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                Iniciales(p.nombre, 40.dp)
                Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                    Texto(p.nombre, Tipo.lato(16f, Peso.Bold), C.tinta)
                    Texto(resumenHorario(p), Tipo.lato(13f), C.sub, lineas = 1)
                }
                if (!p.activo) Texto("Pausada", Tipo.lato(12f, Peso.Bold), C.sub)
                Icono("chevron.right", 13f, C.sub)
            }
        } }.toTypedArray())
        if (estado.gestiona) {
            Bloque("Quién entra al panel", "", *miembros.map { m -> @Composable {
                Renglon(m.nombre.ifEmpty { m.email }, "${nombreRol(m.rol)}${if (m.activo) "" else " · sin acceso"}") {
                    if (m.rol != "duenio") MenuBoton(rolesEquipo.map { (k, t) -> OpcionDeMenu(t, if (m.rol == k) "checkmark" else "circle") {
                        alcance.launch { estado.intentar("Cambiamos el rol") { estado.api.hacer("PATCH", "/v1/panel/equipo/${m.id}", Rol(k)) }; cargar() }
                    } } + OpcionDeMenu(if (m.activo) "Quitarle el acceso" else "Devolverle el acceso", if (m.activo) "person.slash" else "person.badge.plus") {
                        alcance.launch { estado.intentar(if (m.activo) "Ya no entra al panel" else "Volvió a entrar al panel") { estado.api.hacer("PATCH", "/v1/panel/equipo/${m.id}", Activo(!m.activo)) }; cargar() }
                    }, "Opciones de ${m.nombre}") { Box(Modifier.size(44.dp), contentAlignment = Alignment.Center) { Icono("ellipsis", 17f) } }
                }
            } }.toTypedArray())
            Bloque("Invitaciones", "", *(invitaciones.filter { it.aceptadaEn == null }.map { i -> @Composable {
                Renglon(i.email, "${nombreRol(i.rol)} · ${if (i.venceEn.isBefore(Instant.now())) "venció" else "vence ${Formato.dia(i.venceEn)}"}") {
                    BotonTexto("Anular", { alcance.launch { estado.intentar { estado.api.hacer("DELETE", "/v1/panel/invitaciones/${i.id}") }; cargar() } }, color = C.error)
                }
            } } + listOf(@Composable { Box(Modifier.clickable { invitar = true }) { Renglon("Invitar a alguien", simbolo = "envelope.badge.person.crop") } })).toTypedArray())
        }
    }
    Hoja(nuevo, { nuevo = false }) { NuevaProfesional(servicios, listo = { alcance.launch { cargar() } }, cerrar = { nuevo = false }) }
    Hoja(invitar, { invitar = false }) { Invitar(listo = { alcance.launch { cargar() } }, cerrar = { invitar = false }) }
}

private fun resumenHorario(p: Profesional): String {
    if (p.horario.isEmpty()) return "Sin horario cargado"
    val dias = p.horario.map { it.dia }.toSet()
    val letras = listOf("D", "L", "M", "M", "J", "V", "S")
    val orden = listOf(1, 2, 3, 4, 5, 6, 0).filter { it in dias }.joinToString(" ") { letras[it] }
    return "$orden · ${p.horario.minOf { it.desde }} a ${p.horario.maxOf { it.hasta }}"
}

@Composable
fun ColumnScope.NuevaProfesional(servicios: List<Servicio>, listo: () -> Unit, cerrar: () -> Unit) {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    var nombre by remember { mutableStateOf("") }
    var especialidad by remember { mutableStateOf("") }
    var email by remember { mutableStateOf("") }
    var telefono by remember { mutableStateOf("") }
    var hace by remember { mutableStateOf(servicios.filter { it.activo }.map { it.id }.toSet()) }
    @Serializable data class Cuerpo(val nombre: String, val especialidad: String, val email: String, val telefono: String, val servicios: List<String>, val activo: Boolean = true)
    HojaEdicion("Sumar profesional", cerrar, guardar = "Sumar", habilitado = nombre.trim().length >= 2, accion = {
        try {
            val p: Profesional = estado.api.enviar("POST", "/v1/panel/profesionales", Cuerpo(nombre, especialidad, email, telefono, hace.toList()))
            listo()
            nav.pilaMas.add(DestinoMas.profesional(p.id))
            estado.avisar("Sumamos a ${p.nombre}", "Ahora cargale el horario.")
            true
        } catch (e: Exception) { estado.avisar(e); false }
    }) {
        Bloque("", "",
            { RenglonCampo("Nombre", nombre, { nombre = it }) },
            { RenglonCampo("Especialidad", especialidad, { especialidad = it }, "Colorista, manicura…") },
            { RenglonCampo("Email (para avisarle sus turnos)", email, { email = it }, teclado = KeyboardType.Email) },
            { RenglonCampo("Teléfono", telefono, { telefono = it }, teclado = KeyboardType.Phone) })
        Bloque("Qué servicios hace", "", *servicios.filter { it.activo }.map { s -> @Composable {
            RenglonInterruptor(s.nombre, s.id in hace, { hace = if (it) hace + s.id else hace - s.id })
        } }.toTypedArray())
    }
}

@Composable
fun ColumnScope.Invitar(listo: () -> Unit, cerrar: () -> Unit) {
    val estado = LocalEstado.current
    var email by remember { mutableStateOf("") }
    var rol by remember { mutableStateOf("recepcion") }
    @Serializable data class Cuerpo(val email: String, val rol: String)
    HojaEdicion("Invitar", cerrar, guardar = "Mandar", habilitado = "@" in email, accion = {
        val ok = estado.intentar("Le mandamos la invitación a $email") { estado.api.hacer("POST", "/v1/panel/invitaciones", Cuerpo(email.trim(), rol)) }
        if (ok) listo()
        ok
    }) {
        Bloque("", "Le llega un mail para entrar con su cuenta de Google o Apple.",
            { RenglonCampo("Email", email, { email = it }, teclado = KeyboardType.Email) },
            { Renglon("Rol") { SelectorMenu(rolesEquipo, rol, { rol = it }) } })
        Column(Modifier.padding(horizontal = 4.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Texto("Encargada: todo menos la suscripción.", Tipo.lato(13f), C.sub)
            Texto("Recepción: turnos, clientes y caja, sin ajustes.", Tipo.lato(13f), C.sub)
            Texto("Profesional: sólo sus turnos y su horario.", Tipo.lato(13f), C.sub)
        }
    }
}

/** «Mi horario» de la profesional: su propia ficha. */
@Composable
fun MiHorario() {
    val estado = LocalEstado.current
    var id by remember { mutableStateOf<String?>(null) }
    @Serializable data class P(val profesionales: List<Profesional> = emptyList())
    LaunchedEffect(Unit) {
        val r = estado.api.opcional<P>("/v1/panel/profesionales")
        val mail = estado.persona?.email?.lowercase().orEmpty()
        id = (r?.profesionales?.firstOrNull { it.email.orEmpty().lowercase() == mail } ?: r?.profesionales?.firstOrNull())?.id
    }
    val i = id
    if (i != null) FichaProfesional(i, propia = true) else PantallaMas("Mi horario", cargando = true, recargar = {}) {}
}

/** Una profesional: datos, servicios que hace, horario semanal, bloqueos y días libres. */
@Composable
fun FichaProfesional(id: String, propia: Boolean = false) {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    var p by remember { mutableStateOf<Profesional?>(null) }
    var servicios by remember { mutableStateOf<List<Servicio>>(emptyList()) }
    var bloqueos by remember { mutableStateOf<List<Bloqueo>>(emptyList()) }
    var pedidos by remember { mutableStateOf<List<PedidoDias>>(emptyList()) }
    var error by remember { mutableStateOf("") }
    var horario by remember { mutableStateOf(false) }
    var bloquear by remember { mutableStateOf(false) }
    var datos by remember { mutableStateOf(false) }
    var pedirLibres by remember { mutableStateOf(false) }

    @Serializable data class Ps(val profesionales: List<Profesional> = emptyList())
    @Serializable data class Ss(val servicios: List<Servicio> = emptyList())
    @Serializable data class Bs(val bloqueos: List<Bloqueo> = emptyList())
    @Serializable data class Ds(val pedidos: List<PedidoDias> = emptyList())
    suspend fun cargar() {
        try {
            val ps: Ps = estado.api.get("/v1/panel/profesionales")
            servicios = estado.api.opcional<Ss>("/v1/panel/servicios")?.servicios ?: emptyList()
            bloqueos = estado.api.opcional<Bs>("/v1/panel/profesionales/$id/bloqueos")?.bloqueos ?: emptyList()
            pedidos = estado.api.opcional<Ds>("/v1/panel/dias-libres")?.pedidos ?: emptyList()
            val x = ps.profesionales.firstOrNull { it.id == id }
            if (x == null) { error = "No encontramos a esa profesional."; return }
            p = x; error = ""
        } catch (e: ErrorApi) { error = e.detalle } catch (e: Exception) { error = "Probá de nuevo." }
    }
    LaunchedEffect(estado.version) { cargar() }
    @Serializable data class Lista(val servicios: List<String>)
    fun servicio(sid: String, on: Boolean) = alcance.launch {
        val x = p ?: return@launch
        val l = (if (x.servicios.isEmpty()) servicios.filter { it.activo }.map { it.id } else x.servicios).toMutableList()
        if (on) { if (sid !in l) l.add(sid) } else l.remove(sid)
        estado.intentar { estado.api.hacer("PUT", "/v1/panel/profesionales/$id/servicios", Lista(l)) }
        cargar()
    }
    @Serializable data class Activo(val activo: Boolean)

    val x = p
    PantallaMas(x?.nombre ?: (if (propia) "Mi horario" else "Profesional"), x?.especialidad.orEmpty(), volver = if (propia) "Más" else "Equipo",
        acciones = if (estado.gestiona && x != null) listOf(AccionCabecera("pencil", "Editar datos") { datos = true }) else emptyList(),
        cargando = x == null && error.isEmpty(), error = error, recargar = { cargar() }) {
        if (x == null) return@PantallaMas
        Bloque("Horario de la semana", "", *(listOf(1, 2, 3, 4, 5, 6, 0).map { d -> @Composable {
            val fr = x.horario.filter { it.dia == d }.sortedBy { it.desde }
            Renglon(NOMBRES_DIAS[d]) {
                Texto(if (fr.isEmpty()) "No atiende" else fr.joinToString(" · ") { "${it.desde} a ${it.hasta}" }, Tipo.lato(14f, if (fr.isEmpty()) Peso.Regular else Peso.Bold, numeros = true), if (fr.isEmpty()) C.sub else C.tinta)
            }
        } } + (if (estado.gestiona || propia) listOf(@Composable { Box(Modifier.clickable { horario = true }) { Renglon("Cambiar el horario", simbolo = "clock.badge") } }) else emptyList())).toTypedArray())
        Bloque("Días u horas que no atiende", "Feriados, turnos médicos, vacaciones cortas. Esos horarios no se pueden reservar.", *(bloqueos.sortedBy { it.fecha }.map { b -> @Composable {
            Renglon(Fecha.iso(b.fecha)?.let { Formato.fechaLarga(it) } ?: b.fecha, listOf(b.desde?.let { "$it a ${b.hasta.orEmpty()}" } ?: "Todo el día", b.motivo.orEmpty()).filter { it.isNotEmpty() }.joinToString(" · ")) {
                Box(Modifier.size(44.dp).clickable { alcance.launch { estado.intentar { estado.api.hacer("DELETE", "/v1/panel/profesionales/$id/bloqueos/${b.id}") }; estado.version += 1 } }, contentAlignment = Alignment.Center) {
                    Icono("trash", 17f, C.error, descripcion = "Sacar el bloqueo")
                }
            }
        } } + listOf(@Composable { Box(Modifier.clickable { bloquear = true }) { Renglon("Bloquear un día u horario", simbolo = "calendar.badge.minus") } })).toTypedArray())
        if (estado.gestiona) Bloque("Servicios que hace", "", *servicios.filter { it.activo }.map { s -> @Composable {
            RenglonInterruptor(s.nombre, x.servicios.isEmpty() || s.id in x.servicios, { servicio(s.id, it) })
        } }.toTypedArray())
        Bloque("Días libres", if (propia) "Pedilos acá: le llega a la dueña y, si los aprueba, te avisamos." else "", *(pedidos.filter { it.profesionalId == x.id }.map { d -> @Composable {
            Renglon("${d.motivoTexto} · ${rangoDias(d)}", "${estadoPedido(d.estado)} · ${Formato.cantidad(d.turnos, "turno", "turnos")} para mover") {
                if (d.estado == "pendiente" && propia) BotonTexto("Cancelar", { alcance.launch { estado.intentar { estado.api.pedir("POST", "/v1/panel/dias-libres/${d.id}/cancelar", cuerpo = "{}".toByteArray()) }; cargar() } }, color = C.error)
            }
        } } + listOf(@Composable { Box(Modifier.clickable { pedirLibres = true }) { Renglon(if (propia) "Pedir días libres" else "Cargar días libres", simbolo = "beach.umbrella") } })).toTypedArray())
        if (estado.gestiona) Bloque("", "", { RenglonInterruptor("Atiende", x.activo, { v -> alcance.launch { estado.intentar { estado.api.hacer("PATCH", "/v1/panel/profesionales/$id", Activo(v)) }; estado.version += 1 } }, "Si la pausás, no aparece para reservar") })
    }
    Hoja(horario, { horario = false }) { p?.let { EditorHorario(it, listo = { alcance.launch { cargar() } }, cerrar = { horario = false }) } }
    Hoja(bloquear, { bloquear = false }) { NuevoBloqueo(id, listo = { alcance.launch { cargar() } }, cerrar = { bloquear = false }) }
    Hoja(datos, { datos = false }) { p?.let { DatosProfesional(it, listo = { alcance.launch { cargar() } }, cerrar = { datos = false }) } }
    Hoja(pedirLibres, { pedirLibres = false }) { PedirDiasLibres(id, listo = { alcance.launch { cargar() } }, cerrar = { pedirLibres = false }) }
}

private fun rangoDias(d: PedidoDias): String {
    val a = Fecha.iso(d.desde) ?: return "${d.desde} a ${d.hasta}"
    val b = Fecha.iso(d.hasta) ?: return "${d.desde} a ${d.hasta}"
    return if (d.desde == d.hasta) Formato.dia(a) else "${Formato.dia(a)} al ${Formato.dia(b)}"
}
private fun estadoPedido(e: String) = mapOf("pendiente" to "Esperando respuesta", "aprobado" to "Aprobado", "rechazado" to "Rechazado", "cancelado" to "Cancelado")[e] ?: e

@Composable
fun ColumnScope.DatosProfesional(inicial: Profesional, listo: () -> Unit, cerrar: () -> Unit) {
    val estado = LocalEstado.current
    var p by remember { mutableStateOf(inicial) }
    var email by remember { mutableStateOf(inicial.email.orEmpty()) }
    var telefono by remember { mutableStateOf(inicial.telefono.orEmpty()) }
    var bio by remember { mutableStateOf(inicial.bio.orEmpty()) }
    @Serializable data class Cuerpo(val nombre: String, val especialidad: String, val email: String, val telefono: String, val bio: String)
    HojaEdicion("Datos", cerrar, accion = {
        val ok = estado.intentar { estado.api.hacer("PATCH", "/v1/panel/profesionales/${p.id}", Cuerpo(p.nombre, p.especialidad, email, telefono, bio)) }
        if (ok) listo()
        ok
    }) {
        Bloque("", "",
            { RenglonCampo("Nombre", p.nombre, { p = p.copy(nombre = it) }) },
            { RenglonCampo("Especialidad", p.especialidad, { p = p.copy(especialidad = it) }) },
            { RenglonCampo("Email", email, { email = it }, teclado = KeyboardType.Email) },
            { RenglonCampo("Teléfono", telefono, { telefono = it }, teclado = KeyboardType.Phone) },
            { RenglonCampo("Sobre ella (se ve en tu página)", bio, { bio = it }, multilinea = true) })
    }
}

/** El horario semanal: por día, una o dos franjas (mañana y tarde), con «copiar a todos». */
@Composable
fun ColumnScope.EditorHorario(profesional: Profesional, listo: () -> Unit, cerrar: () -> Unit) {
    val estado = LocalEstado.current
    var franjas by remember { mutableStateOf(profesional.horario) }
    @Serializable data class Cuerpo(val horario: List<Franja>)
    HojaEdicion("Horario", cerrar, accion = {
        val mal = franjas.firstOrNull { Dia.aMin(it.desde) >= Dia.aMin(it.hasta) }
        if (mal != null) {
            estado.avisar("Revisá el ${NOMBRES_DIAS[mal.dia].lowercase()}", "La hora de cierre tiene que ser después de la de apertura.", ok = false)
            false
        } else {
            val ok = estado.intentar("Guardamos el horario") { estado.api.hacer("PUT", "/v1/panel/profesionales/${profesional.id}/horario", Cuerpo(franjas)) }
            if (ok) { estado.version += 1; listo() }
            ok
        }
    }) {
        listOf(1, 2, 3, 4, 5, 6, 0).forEach { d ->
            val del = franjas.withIndex().filter { it.value.dia == d }
            Bloque(NOMBRES_DIAS[d], "", *buildList<@Composable () -> Unit> {
                add { RenglonInterruptor(if (del.isEmpty()) "No atiende" else "Atiende", del.isNotEmpty(), { on -> franjas = if (on) franjas + Franja(d, "09:00", "18:00") else franjas.filter { it.dia != d } }) }
                del.forEach { (i, f) -> add {
                    Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            SelectorHora(Dia.aMin(f.desde), { m -> franjas = franjas.toMutableList().also { it[i] = f.copy(desde = hhmm(m)) } })
                            Texto("a", Tipo.lato(14f), C.sub)
                            SelectorHora(Dia.aMin(f.hasta), { m -> franjas = franjas.toMutableList().also { it[i] = f.copy(hasta = hhmm(m)) } })
                        }
                        Spacer(Modifier.weight(1f))
                        if (del.size > 1) Box(Modifier.size(44.dp).clickable { franjas = franjas.toMutableList().also { it.removeAt(i) } }, contentAlignment = Alignment.Center) { Icono("minus.circle", 20f, C.error) }
                    }
                } }
                if (del.isNotEmpty()) add {
                    Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 6.dp), verticalAlignment = Alignment.CenterVertically) {
                        if (del.size < 3) BotonTexto("Sumar un corte", { franjas = franjas + Franja(d, "15:00", "20:00") })
                        Spacer(Modifier.weight(1f))
                        BotonTexto("Copiar a todos los días que atiende", {
                            val base = franjas.filter { it.dia == d }
                            val dias = franjas.map { it.dia }.toSet()
                            franjas = base + (dias - d).flatMap { x -> base.map { Franja(x, it.desde, it.hasta) } }
                        })
                    }
                }
            }.toTypedArray())
        }
    }
}

@Composable
fun ColumnScope.NuevoBloqueo(profesionalId: String, listo: () -> Unit, cerrar: () -> Unit) {
    val estado = LocalEstado.current
    val hoy = Formato.enArgentina(Instant.now()).toLocalDate()
    var desde by remember { mutableStateOf(hoy) }
    var hastaDia by remember { mutableStateOf(hoy) }
    var varios by remember { mutableStateOf(false) }
    var entero by remember { mutableStateOf(true) }
    var horaDesde by remember { mutableIntStateOf(9 * 60) }
    var horaHasta by remember { mutableIntStateOf(13 * 60) }
    var motivo by remember { mutableStateOf("") }
    @Serializable data class Cuerpo(val fecha: String, val desde: String?, val hasta: String?, val motivo: String)
    HojaEdicion("Bloquear", cerrar, guardar = "Bloquear", accion = {
        val dias = mutableListOf(desde)
        if (varios) { var d = desde; while (d.plusDays(1) <= hastaDia && dias.size < 120) { d = d.plusDays(1); dias.add(d) } }
        val ok = estado.intentar(if (dias.size > 1) "Bloqueamos esos ${dias.size} días" else "Bloqueamos esa fecha") {
            for (d in dias) estado.api.hacer("POST", "/v1/panel/profesionales/$profesionalId/bloqueos", Cuerpo(d.toString(), if (entero) null else hhmm(horaDesde), if (entero) null else hhmm(horaHasta), motivo))
        }
        if (ok) { estado.version += 1; listo() }
        ok
    }) {
        Bloque("", "", *buildList<@Composable () -> Unit> {
            add { RenglonFecha("Día", desde, { desde = it; if (hastaDia < it) hastaDia = it }, minimo = hoy) }
            add { RenglonInterruptor("Varios días seguidos", varios, { varios = it }) }
            if (varios) add { RenglonFecha("Hasta", hastaDia, { hastaDia = it }, minimo = desde) }
            add { RenglonInterruptor("Todo el día", entero, { entero = it }) }
            if (!entero) { add { RenglonHora("Desde", horaDesde, { horaDesde = it }) }; add { RenglonHora("Hasta", horaHasta, { horaHasta = it }) } }
            add { RenglonCampo("Motivo (opcional)", motivo, { motivo = it }, "Feriado, médico…") }
        }.toTypedArray())
    }
}

@Composable
fun ColumnScope.PedirDiasLibres(profesionalId: String, listo: () -> Unit, cerrar: () -> Unit) {
    val estado = LocalEstado.current
    val hoy = Formato.enArgentina(Instant.now()).toLocalDate()
    var desde by remember { mutableStateOf(hoy) }
    var hasta by remember { mutableStateOf(hoy) }
    var motivo by remember { mutableStateOf("vacaciones") }
    var nota by remember { mutableStateOf("") }
    @Serializable data class Cuerpo(val profesionalId: String, val desde: String, val hasta: String, val motivo: String, val nota: String)
    HojaEdicion("Días libres", cerrar, guardar = if (estado.esProfesional) "Pedir" else "Cargar", accion = {
        val ok = estado.intentar(if (estado.esProfesional) "Mandamos el pedido" else "Cargamos los días libres") {
            estado.api.hacer("POST", "/v1/panel/dias-libres", Cuerpo(profesionalId, desde.toString(), maxOf(desde, hasta).toString(), motivo, nota))
        }
        if (ok) { estado.version += 1; listo() }
        ok
    }) {
        Bloque("", "Si hay turnos esos días, quedan para mover: le avisamos a cada clienta cuando los muevas.",
            { RenglonFecha("Desde", desde, { desde = it; if (hasta < it) hasta = it }, minimo = hoy) },
            { RenglonFecha("Hasta", hasta, { hasta = it }, minimo = desde) },
            { Renglon("Motivo") { SelectorMenu(listOf("vacaciones" to "Vacaciones", "tramite" to "Trámite", "salud" to "Salud", "otro" to "Otro"), motivo, { motivo = it }) } },
            { RenglonCampo("Nota", nota, { nota = it }, multilinea = true) })
    }
}

/** Los pedidos de días libres del equipo, para aprobar o rechazar, y los turnos que quedaron para mover. */
@Composable
fun DiasLibres() {
    val estado = LocalEstado.current
    val nav = LocalNavegacion.current
    val alcance = rememberCoroutineScope()
    var pedidos by remember { mutableStateOf<List<PedidoDias>>(emptyList()) }
    var mover by remember { mutableStateOf<List<AReprogramar>>(emptyList()) }
    var cargando by remember { mutableStateOf(true) }
    var tab by remember { mutableStateOf("pendiente") }
    @Serializable data class Ds(val pedidos: List<PedidoDias> = emptyList())
    @Serializable data class Rs(val turnos: List<AReprogramar> = emptyList())
    suspend fun cargar() {
        estado.api.opcional<Ds>("/v1/panel/dias-libres")?.let { pedidos = it.pedidos }
        estado.api.opcional<Rs>("/v1/panel/reprogramar")?.let { mover = it.turnos }
        cargando = false
    }
    LaunchedEffect(estado.version) { cargar() }
    fun resolver(d: PedidoDias, que: String) = alcance.launch {
        estado.intentar(if (que == "aprobar") "Aprobado. Le avisamos a ${d.profesional}" else "Rechazado. Le avisamos a ${d.profesional}") {
            estado.api.pedir("POST", "/v1/panel/dias-libres/${d.id}/$que", cuerpo = "{}".toByteArray())
        }
        estado.version += 1
    }
    fun f(s: String) = Fecha.iso(s)?.let { Formato.dia(it) } ?: s
    PantallaMas("Días libres", Formato.cantidad(pedidos.count { it.estado == "pendiente" }, "pedido sin responder", "pedidos sin responder"), cargando = cargando, recargar = { cargar() },
        abajo = { Segmentado(listOf("pendiente" to "Pendientes", "aprobado" to "Aprobados", "todos" to "Todos"), tab, { tab = it }) }) {
        val lista = pedidos.filter { tab == "todos" || it.estado == tab }
        if (lista.isEmpty()) VacioQueEnsena("Nada por acá", "No hay pedidos", "Cuando alguien del equipo pida días libres desde su app, aparece acá para aprobarlo.")
        lista.forEach { d ->
            Column(Modifier.fillMaxWidth().tarjeta().padding(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                Row(horizontalArrangement = Arrangement.spacedBy(10.dp), verticalAlignment = Alignment.CenterVertically) {
                    Iniciales(d.profesional, 40.dp)
                    Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                        Texto(d.profesional, Tipo.lato(16f, Peso.Bold), C.tinta)
                        Texto("${d.motivoTexto} · ${if (d.desde == d.hasta) f(d.desde) else "${f(d.desde)} al ${f(d.hasta)}"}", Tipo.lato(13f), C.sub)
                    }
                }
                if (d.nota.isNotEmpty()) Texto("«${d.nota}»", Tipo.lato(14f), C.tinta)
                Texto(if (d.afectados > 0) "${Formato.cantidad(d.afectados, "turno", "turnos")} para mover" else "No hay turnos esos días", Tipo.lato(13f, Peso.Bold), if (d.afectados > 0) C.pagoTinta else C.mentaTexto)
                if (d.estado == "pendiente") Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Box(Modifier.weight(1f)) { BotonContorno("Rechazar", { resolver(d, "rechazar") }, alto = 44.dp, tam = 15f) }
                    Box(Modifier.weight(1f)) { BotonMenta("Aprobar", { resolver(d, "aprobar") }, alto = 44.dp, tam = 15f, conSombra = false) }
                } else TextoRotulo(mapOf("aprobado" to "Aprobado", "rechazado" to "Rechazado", "cancelado" to "Cancelado")[d.estado] ?: d.estado)
            }
        }
        if (mover.isNotEmpty()) Bloque("Turnos para mover", "", *mover.map { r -> @Composable {
            Box(Modifier.clickable { alcance.launch { estado.api.opcional<Turno>("/v1/panel/turnos/${r.turno.id}")?.let { nav.turno = it } } }) {
                Renglon(r.turno.cliente.nombre, "${Formato.diaYHora(r.turno.inicio)} · ${r.turno.profesional}") { Texto("Mover", Tipo.lato(14f, Peso.Bold), C.mentaTexto) }
            }
        } }.toTypedArray())
    }
}
