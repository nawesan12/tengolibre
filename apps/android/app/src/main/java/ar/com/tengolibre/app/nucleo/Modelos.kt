package ar.com.tengolibre.app.nucleo

import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import ar.com.tengolibre.app.tema.C
import kotlinx.serialization.Serializable
import java.time.Duration
import java.time.Instant

// Las formas de la API de Tengo Libre (clon de Nucleo/Modelos.swift): las mismas que usa la web.
// Las fechas llegan en ISO 8601; los precios, en pesos.

@Serializable
data class Persona(val clerkId: String = "", val email: String = "", val nombre: String = "", val foto: String = "")

/** Un negocio de la persona y su rol en él (duenio, encargado, recepcion, profesional). */
@Serializable
data class NegocioDeLaPersona(val id: String, val slug: String = "", val nombre: String = "", val rol: String = "") {
    val rolTexto: String get() = when (rol) {
        "duenio" -> "Dueña"
        "encargado" -> "Encargada"
        "recepcion" -> "Recepción"
        "profesional" -> "Profesional · sólo tus turnos"
        else -> Formato.mayuscula(rol)
    }
}

@Serializable
data class Yo(val negocios: List<NegocioDeLaPersona> = emptyList())

@Serializable
data class Acceso(val bloqueado: Boolean = false, val pruebaHasta: FechaApi? = null, val diasDePrueba: Int = 0, val conSuscripcion: Boolean = false)

@Serializable
data class ClienteDelTurno(val id: String? = null, val nombre: String = "", val telefono: String = "", val email: String = "")

@Serializable
data class Turno(
    val id: String,
    val codigo: String = "",
    val servicioId: String? = null,
    val servicio: String = "",
    val profesionalId: String? = null,
    val profesional: String = "",
    val cliente: ClienteDelTurno = ClienteDelTurno(),
    val inicio: FechaApi,
    val fin: FechaApi,
    val precio: Int = 0,
    val estado: String = "pendiente",
    val origen: String = "",
    val nota: String = "",
    val venceEn: FechaApi? = null,
    val creadoEn: FechaApi = Instant.EPOCH,
    val serieId: String? = null,
    val confirmadoPor: String? = null,
    val confirmadoEn: FechaApi? = null,
    val cambioPedidoEn: FechaApi? = null,
    val llamarDesde: FechaApi? = null,
    val llamarMotivo: String? = null,
) {
    val minutos: Int get() = maxOf(0, Duration.between(inicio, fin).toMinutes().toInt())
    val ocupa: Boolean get() = estado != "cancelado"

    /** El estado que se ve: un confirmado (o pendiente) que está pasando ahora es «En curso». */
    fun visible(ahora: Instant = Instant.now()): EstadoTurno {
        if (estado in listOf("confirmado", "pendiente") && !inicio.isAfter(ahora) && fin.isAfter(ahora)) return EstadoTurno.enCurso
        return EstadoTurno.de(estado) ?: EstadoTurno.pendiente
    }

    /** Recién llegado (reserva de la web de hace menos de 10 minutos): el talón menta. */
    fun esNuevo(ahora: Instant = Instant.now()): Boolean =
        origen != "panel" && Duration.between(creadoEn, ahora).seconds < 600 && estado in listOf("pendiente", "pendiente_pago", "confirmado")
}

/** Los estados de un turno con cómo se ven (ícono + palabra siempre). */
enum class EstadoTurno(val valor: String) {
    pendientePago("pendiente_pago"), pendiente("pendiente"), confirmado("confirmado"), enCurso("en_curso"),
    atendido("atendido"), ausente("ausente"), cancelado("cancelado");

    val texto: String get() = when (this) {
        pendientePago -> "Seña pendiente"
        pendiente -> "Pendiente"
        confirmado -> "Confirmado"
        enCurso -> "En curso"
        atendido -> "Atendido"
        ausente -> "No vino"
        cancelado -> "Cancelado"
    }

    val simbolo: String get() = when (this) {
        pendientePago -> "hourglass"
        pendiente -> "circle"
        confirmado -> "checkmark.circle"
        enCurso -> "timelapse"
        atendido -> "checkmark.circle.badge.checkmark"
        ausente -> "person.slash"
        cancelado -> "xmark.circle"
    }

    val fondo: Color @Composable get() = when (this) {
        confirmado -> C.mentaSuave
        enCurso -> C.seleccion
        pendientePago -> C.pagoFondo
        ausente -> C.ausenteFondo
        else -> C.chip
    }

    val tinta: Color @Composable get() = when (this) {
        confirmado -> C.mentaTexto
        enCurso -> C.seleccionTexto
        pendientePago -> C.pagoTinta
        ausente -> C.ausenteTinta
        atendido, cancelado -> C.sub
        else -> C.tinta
    }

    companion object {
        fun de(v: String): EstadoTurno? = entries.firstOrNull { it.valor == v }

        /** Los pasos que el panel puede dar desde cada estado (los de la API, turnos.PuedePasar). */
        val pasos: Map<String, List<String>> = mapOf(
            "pendiente_pago" to listOf("pendiente", "confirmado", "cancelado"),
            "pendiente" to listOf("confirmado", "cancelado", "atendido", "ausente"),
            "confirmado" to listOf("pendiente", "cancelado", "atendido", "ausente"),
            "atendido" to listOf("ausente", "confirmado"),
            "ausente" to listOf("atendido", "confirmado"),
        )
        fun puedePasar(de: String, a: String): Boolean = de == a || (pasos[de] ?: emptyList()).contains(a)
    }
}

@Serializable
data class Franja(val dia: Int, val desde: String, val hasta: String)

@Serializable
data class Profesional(
    val id: String,
    val nombre: String = "",
    val especialidad: String = "",
    val activo: Boolean = true,
    val orden: Int = 0,
    val servicios: List<String> = emptyList(),
    val horario: List<Franja> = emptyList(),
    val email: String? = null,
    val telefono: String? = null,
    val bio: String? = null,
    val miembroId: String? = null,
    val foto: String? = null,
)

/** Un servicio («prestación» en la web). */
@Serializable
data class Servicio(
    val id: String,
    val nombre: String = "",
    val duracion: Int = 0,
    val precio: Int = 0,
    val activo: Boolean = true,
    val descripcion: String? = null,
    val categoriaId: String? = null,
    val sena: Int? = null,
    val foto: String? = null,
)

@Serializable
data class HorarioLibre(val hora: String, val inicio: FechaApi, val profesionales: List<String> = emptyList())

@Serializable
data class Etiqueta(val id: String, val nombre: String = "", val color: String = "")

@Serializable
data class Cliente(
    val id: String,
    val nombre: String = "",
    val telefono: String = "",
    val email: String = "",
    val creadoEn: FechaApi = Instant.EPOCH,
    val visitas: Int = 0,
    val ultimaVisita: FechaApi? = null,
    val proximoTurno: FechaApi? = null,
    val gastado: Int = 0,
    val ausencias: Int = 0,
    val penalizado: Boolean = false,
    val etiquetas: List<Etiqueta> = emptyList(),
)

@Serializable
data class Penalidad(
    val id: String,
    val clienteId: String = "",
    val cliente: String = "",
    val telefono: String = "",
    val motivo: String = "",
    val nota: String = "",
    val ausencias: Int = 0,
    val hasta: FechaApi = Instant.EPOCH,
    val creadoEn: FechaApi = Instant.EPOCH,
    val levantadaEn: FechaApi? = null,
    val vigente: Boolean = false,
)

@Serializable
data class TurnoDeCliente(
    val id: String,
    val codigo: String = "",
    val servicio: String = "",
    val profesional: String = "",
    val inicio: FechaApi,
    val fin: FechaApi,
    val precio: Int = 0,
    val estado: String = "",
    val origen: String = "",
    val nota: String = "",
)

@Serializable
data class Ficha(
    val id: String,
    val nombre: String = "",
    val telefono: String = "",
    val email: String = "",
    val creadoEn: FechaApi = Instant.EPOCH,
    val visitas: Int = 0,
    val ultimaVisita: FechaApi? = null,
    val proximoTurno: FechaApi? = null,
    val gastado: Int = 0,
    val ausencias: Int = 0,
    val penalizado: Boolean = false,
    val etiquetas: List<Etiqueta> = emptyList(),
    val turnos: List<TurnoDeCliente> = emptyList(),
    val penalidad: Penalidad? = null,
)

@Serializable
data class Nota(val id: String, val texto: String = "", val miembroId: String? = null, val autor: String = "", val creadoEn: FechaApi = Instant.EPOCH)

/** Alguien en la lista de espera. */
@Serializable
data class Anotado(
    val id: String,
    val servicioId: String = "",
    val servicio: String = "",
    val profesionalId: String? = null,
    val profesional: String? = null,
    val nombre: String = "",
    val email: String = "",
    val telefono: String = "",
    val desde: String = "",
    val hasta: String = "",
    val avisadoEn: FechaApi? = null,
    val creadoEn: FechaApi = Instant.EPOCH,
    /** cualquiera | manana | tarde */
    val momento: String? = null,
    /** Lo que pasó la última vez que se le ofreció: ofrecida, tomada, paso, vencida, cancelada. */
    val resultado: String? = null,
) {
    val momentoTexto: String get() = mapOf("manana" to "Mañana", "tarde" to "Tardes")[momento ?: ""] ?: "Todo el día"
}

@Serializable
data class Resena(
    val id: String,
    val turnoId: String = "",
    val nombre: String = "",
    val servicio: String = "",
    val profesional: String = "",
    val puntaje: Int = 0,
    val comentario: String = "",
    val respuesta: String = "",
    val respondidaEn: FechaApi? = null,
    val visible: Boolean = true,
    val creadoEn: FechaApi = Instant.EPOCH,
)

/** Un aviso de la campanita. */
@Serializable
data class Notificacion(
    val id: String,
    val tipo: String = "",
    val titulo: String = "",
    val texto: String = "",
    val turnoId: String? = null,
    val creadoEn: FechaApi = Instant.EPOCH,
    val nueva: Boolean = false,
    val pedidoId: String? = null,
) {
    val simbolo: String get() = when (tipo) {
        "reserva" -> "calendar.badge.plus"
        "cancelacion" -> "calendar.badge.minus"
        "reprogramacion" -> "calendar.badge.clock"
        "confirmacion" -> "checkmark.message"
        "cambio_pedido" -> "arrow.triangle.2.circlepath"
        "sin_respuesta" -> "phone.badge.waveform"
        "resena" -> "star.bubble"
        "espera" -> "hourglass"
        "ausente" -> "person.slash"
        "equipo" -> "person.2"
        "pago", "sena" -> "creditcard"
        "aviso_fallido" -> "exclamationmark.bubble"
        else -> "bell"
    }
}

@Serializable
data class ParaLlamar(
    val turnoId: String,
    val cliente: String = "",
    val telefono: String = "",
    val servicio: String = "",
    val profesional: String = "",
    val inicio: FechaApi,
    val estado: String = "",
    val motivo: String = "",
    val desde: FechaApi = Instant.EPOCH,
) {
    val id: String get() = turnoId
    val motivoTexto: String get() = mapOf("cambio" to "Pidió cambiarlo", "sin_respuesta" to "No contestó el WhatsApp", "no_llego" to "No le llegó el WhatsApp")[motivo] ?: motivo
}
