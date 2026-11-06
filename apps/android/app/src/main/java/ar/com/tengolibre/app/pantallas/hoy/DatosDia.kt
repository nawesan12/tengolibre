package ar.com.tengolibre.app.pantallas.hoy

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import ar.com.tengolibre.app.app.AccionesTurno
import ar.com.tengolibre.app.app.Estado
import ar.com.tengolibre.app.nucleo.*
import ar.com.tengolibre.app.tema.Haptico
import ar.com.tengolibre.app.tema.Toast
import kotlinx.serialization.Serializable
import java.time.Instant

// Clon de Pantallas/Hoy/DatosDia.swift.

/** Una oferta del hueco a la lista de espera (docs/api/espera-ofertas.md). */
@Serializable
data class Oferta(
    val id: String,
    val estado: String = "",
    val origen: String = "",
    val servicioId: String = "",
    val servicio: String = "",
    val profesionalId: String = "",
    val profesional: String = "",
    val inicio: FechaApi,
    val fin: FechaApi,
    val precio: Int = 0,
    val turnoId: String? = null,
    val creadoEn: FechaApi = Instant.EPOCH,
    val cerradaEn: FechaApi? = null,
    val actual: Intento? = null,
    val sigue: Intento? = null,
    val cola: List<Intento> = emptyList(),
) {
    @Serializable
    data class Intento(
        val id: String,
        val esperaId: String? = null,
        val nombre: String = "",
        val orden: Int = 0,
        val estado: String = "",
        val ofrecidaEn: FechaApi? = null,
        val venceEn: FechaApi? = null,
        val respondidaEn: FechaApi? = null,
    )
}

/** Un turno que hay que mover porque la profesional pidió días libres (dias-libres.md). */
@Serializable
data class AReprogramar(val id: String, val pedidoId: String = "", val motivo: String = "", val resueltoEn: FechaApi? = null, val turno: TurnoR) {
    @Serializable
    data class TurnoR(
        val id: String,
        val inicio: FechaApi,
        val fin: FechaApi,
        val servicio: String = "",
        val profesional: String = "",
        val estado: String = "",
        val cliente: ClienteDelTurno = ClienteDelTurno(),
    )
}

/** Un pedido de días libres de una profesional (dias-libres.md). */
@Serializable
data class PedidoDias(
    val id: String,
    val profesionalId: String = "",
    val profesional: String = "",
    val desde: String = "",
    val hasta: String = "",
    val motivo: String = "",
    val nota: String = "",
    val estado: String = "",
    val turnos: Int = 0,
    val afectados: Int = 0,
    val creadoEn: FechaApi = Instant.EPOCH,
) {
    val motivoTexto: String get() = mapOf("vacaciones" to "Vacaciones", "tramite" to "Trámite", "salud" to "Salud", "otro" to "Otro")[motivo] ?: motivo
}

/** Los primeros pasos del negocio (arranque). */
@Serializable
data class Arranque(val completo: Boolean = false, val reservable: Boolean = false, val pasos: List<Paso> = emptyList()) {
    @Serializable data class Paso(val clave: String, val titulo: String = "", val listo: Boolean = false, val obligatorio: Boolean = false)
}

/** Lo que usan Hoy y Agenda: los turnos de un rango, el catálogo, la lista de espera y lo que hay que atender.
 *  Se guarda para verlo sin conexión. */
class DatosDia {
    sealed class Carga { data object cargando : Carga(); data object listo : Carga(); data class error(val texto: String) : Carga() }

    var carga by mutableStateOf<Carga>(Carga.cargando)
    var turnos by mutableStateOf<List<Turno>>(emptyList())
    var profesionales by mutableStateOf<List<Profesional>>(emptyList())
    var servicios by mutableStateOf<List<Servicio>>(emptyList())
    var espera by mutableStateOf<List<Anotado>>(emptyList())
    var ofertas by mutableStateOf<List<Oferta>>(emptyList())
    var paraLlamar by mutableStateOf<List<ParaLlamar>>(emptyList())
    var reprogramar by mutableStateOf<List<AReprogramar>>(emptyList())
    var arranque by mutableStateOf<Arranque?>(null)
    var resenaSinResponder by mutableStateOf<Resena?>(null)
    var pedidos by mutableStateOf<List<PedidoDias>>(emptyList())
    var mpConectado by mutableStateOf<Boolean?>(null)
    /** Ids de los turnos que llegaron mientras se miraba (para el talón menta que entra desde arriba). */
    var recien by mutableStateOf<Set<String>>(emptySet())

    @Serializable private data class Turnos(val turnos: List<Turno> = emptyList())
    @Serializable private data class Profesionales(val profesionales: List<Profesional> = emptyList())
    @Serializable private data class Servicios(val servicios: List<Servicio> = emptyList())
    @Serializable private data class Espera(val espera: List<Anotado> = emptyList())
    @Serializable private data class Ofertas(val ofertas: List<Oferta> = emptyList())
    @Serializable private data class Llamar(val turnos: List<ParaLlamar> = emptyList())
    @Serializable private data class Reprog(val turnos: List<AReprogramar> = emptyList())
    @Serializable private data class Resenas(val resenas: List<Resena> = emptyList())
    @Serializable private data class Pedidos(val pedidos: List<PedidoDias> = emptyList())
    @Serializable private data class MP(val disponible: Boolean = false, val conectada: Boolean = false)
    @Serializable private data class Todo(val turnos: List<Turno>, val profesionales: List<Profesional>, val servicios: List<Servicio>, val espera: List<Anotado>)

    /** Trae `desde`…`hasta` (días) y todo lo de alrededor. */
    suspend fun cargar(estado: Estado, desde: Instant, hasta: Instant) {
        val api = estado.api
        val clave = "dia-${estado.negocio?.id.orEmpty()}-${Formato.diaISO(desde)}-${Formato.diaISO(hasta)}"
        if (turnos.isEmpty()) Guardado.leer<Todo>(clave)?.let { (g, f) ->
            turnos = g.turnos; profesionales = g.profesionales; servicios = g.servicios; espera = g.espera
            estado.guardadoEn = f
        }
        val mostrador = !estado.esProfesional
        val q = listOf("desde" to Formato.diaISO(desde), "hasta" to Formato.diaISO(hasta))
        try {
            val t: Turnos = api.get("/v1/panel/turnos", q)
            val p: Profesionales = api.get("/v1/panel/profesionales")
            val s: Servicios? = api.opcional("/v1/panel/servicios")
            val e: Espera? = if (mostrador) api.opcional("/v1/panel/espera") else null
            val of: Ofertas? = if (mostrador) api.opcional("/v1/panel/espera/ofertas") else null
            val l: Llamar? = if (mostrador) api.opcional("/v1/panel/para-llamar") else null
            val r: Reprog? = if (mostrador) api.opcional("/v1/panel/reprogramar") else null
            val ar: Arranque? = api.opcional("/v1/panel/arranque")
            val rs: Resenas? = if (estado.rol == "recepcion") null else api.opcional("/v1/panel/resenas")
            val pd: Pedidos? = if (estado.gestiona) api.opcional("/v1/panel/dias-libres", listOf("estado" to "pendiente")) else null
            val mp: MP? = if (estado.gestiona) api.opcional("/v1/panel/mercadopago") else null
            val antes = turnos.map { it.id }.toSet()
            if (carga == Carga.listo) {
                val nuevos = t.turnos.filter { it.id !in antes && it.origen != "panel" }
                if (nuevos.isNotEmpty()) { recien = recien + nuevos.map { it.id }; Haptico.suave() }
            }
            turnos = t.turnos
            profesionales = p.profesionales.sortedBy { it.orden }
            servicios = s?.servicios ?: servicios
            espera = e?.espera ?: emptyList()
            ofertas = (of?.ofertas ?: emptyList()).filter { it.estado == "ofreciendo" }
            paraLlamar = l?.turnos ?: emptyList()
            reprogramar = (r?.turnos ?: emptyList()).filter { it.resueltoEn == null }
            arranque = ar
            resenaSinResponder = (rs?.resenas ?: emptyList()).firstOrNull { it.respondidaEn == null && it.respuesta.isEmpty() }
            pedidos = pd?.pedidos ?: emptyList()
            mpConectado = if (mp != null && mp.disponible) mp.conectada else null
            carga = Carga.listo
            estado.sinConexion = false
            Guardado.guardar(Todo(turnos, profesionales, servicios, espera), clave)
        } catch (x: ErrorApi) {
            if (x.sinConexion) estado.sinConexion = true
            carga = if (turnos.isEmpty()) Carga.error(x.detalle) else Carga.listo
        } catch (x: kotlinx.coroutines.CancellationException) {
            throw x
        } catch (x: Exception) {
            carga = if (turnos.isEmpty()) Carga.error("Probá de nuevo en un rato.") else Carga.listo
        }
    }

    /** Cambia el estado de un turno al toque (y vuelve atrás si la API no lo acepta). */
    suspend fun cambiar(estado: Estado, t: Turno, nuevo: String) {
        val antes = turnos.firstOrNull { it.id == t.id }?.estado ?: return
        turnos = turnos.map { if (it.id == t.id) it.copy(estado = nuevo) else it }
        try {
            estado.api.hacer("PATCH", "/v1/panel/turnos/${t.id}", AccionesTurno.CambioEstado(nuevo))
            Haptico.exito()
            val texto = mapOf("atendido" to "Marcado atendido", "ausente" to "Marcado ausente", "confirmado" to "Turno confirmado", "cancelado" to "Turno cancelado")[nuevo] ?: "Listo"
            estado.avisar(Toast("checkmark", texto, "${t.cliente.nombre} · ${Formato.horaDe(t.inicio)}", "Deshacer")) {
                runCatching { estado.api.hacer("PATCH", "/v1/panel/turnos/${t.id}", AccionesTurno.CambioEstado(antes)) }
            }
        } catch (e: Exception) {
            turnos = turnos.map { if (it.id == t.id) it.copy(estado = antes) else it }
            estado.avisar(e)
        }
    }

    /** Cuántos de la lista esperan algo que entre en ese hueco (con esa profesional o con cualquiera). */
    fun esperanPara(h: Dia.Hueco): List<Anotado> = espera.filter { a -> a.avisadoEn == null && (a.profesionalId == null || a.profesionalId == h.profesionalId) }

    fun ofertaPara(h: Dia.Hueco, dia: Instant): Oferta? = ofertas.firstOrNull {
        it.profesionalId == h.profesionalId && Formato.diaISO(it.inicio) == Formato.diaISO(dia) &&
            Formato.minutosDelDia(it.inicio) >= h.desde && Formato.minutosDelDia(it.inicio) < h.hasta
    }
}
