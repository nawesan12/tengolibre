import Foundation
import SwiftUI

// Las formas de la API de Tengo Libre: las mismas que usa la web (apps/web/src/lib/operacion/tipos.ts,
// lib/server/api.ts). Las fechas llegan en ISO 8601; los precios, en pesos.

struct Persona: Codable, Equatable {
    var clerkId: String
    var email: String
    var nombre: String
    var foto: String
}

/// Un negocio de la persona y su rol en él (dueño, encargado, recepcion, profesional).
struct NegocioDeLaPersona: Codable, Identifiable, Equatable, Hashable {
    var id: String
    var slug: String
    var nombre: String
    var rol: String

    var rolTexto: String {
        switch rol {
        case "duenio": "Dueña"
        case "encargado": "Encargada"
        case "recepcion": "Recepción"
        case "profesional": "Profesional · sólo tus turnos"
        default: Formato.mayuscula(rol)
        }
    }
}

struct Yo: Codable { var negocios: [NegocioDeLaPersona] }

struct Acceso: Codable, Equatable {
    var bloqueado: Bool
    var pruebaHasta: Date?
    var diasDePrueba: Int
    var conSuscripcion: Bool
}

struct ClienteDelTurno: Codable, Equatable, Hashable {
    var id: String?
    var nombre: String
    var telefono: String
    var email: String
}

struct Turno: Codable, Identifiable, Equatable, Hashable {
    var id: String
    var codigo: String
    var servicioId: String?
    var servicio: String
    var profesionalId: String?
    var profesional: String
    var cliente: ClienteDelTurno
    var inicio: Date
    var fin: Date
    var precio: Int
    var estado: String
    var origen: String
    var nota: String
    var venceEn: Date?
    var creadoEn: Date
    var serieId: String?
    var confirmadoPor: String?
    var confirmadoEn: Date?
    var cambioPedidoEn: Date?
    var llamarDesde: Date?
    var llamarMotivo: String?

    var minutos: Int { max(0, Int(fin.timeIntervalSince(inicio) / 60)) }
    var ocupa: Bool { estado != "cancelado" }

    /// El estado que se ve: un confirmado (o pendiente) que está pasando ahora es «En curso».
    func visible(_ ahora: Date = Date()) -> EstadoTurno {
        if ["confirmado", "pendiente"].contains(estado), inicio <= ahora, fin > ahora { return .enCurso }
        return EstadoTurno(rawValue: estado) ?? .pendiente
    }

    /// Recién llegado (reserva de la web de hace menos de 10 minutos): el talón menta.
    func esNuevo(_ ahora: Date = Date()) -> Bool {
        origen != "panel" && ahora.timeIntervalSince(creadoEn) < 600 && ["pendiente", "pendiente_pago", "confirmado"].contains(estado)
    }
}

/// Los estados de un turno (web: operacion/estados.ts) con cómo se ven en el iPhone (handoff:
/// ícono + palabra siempre).
enum EstadoTurno: String, CaseIterable {
    case pendientePago = "pendiente_pago"
    case pendiente, confirmado, enCurso = "en_curso", atendido, ausente, cancelado

    var texto: String {
        switch self {
        case .pendientePago: "Seña pendiente"
        case .pendiente: "Pendiente"
        case .confirmado: "Confirmado"
        case .enCurso: "En curso"
        case .atendido: "Atendido"
        case .ausente: "No vino"
        case .cancelado: "Cancelado"
        }
    }

    var simbolo: String {
        switch self {
        case .pendientePago: "hourglass"
        case .pendiente: "circle"
        case .confirmado: "checkmark.circle"
        case .enCurso: "timelapse"
        case .atendido: "checkmark.circle.badge.checkmark"
        case .ausente: "person.slash"
        case .cancelado: "xmark.circle"
        }
    }

    var fondo: Color {
        switch self {
        case .confirmado: C.mentaSuave
        case .enCurso: C.seleccion
        case .pendientePago: C.pagoFondo
        case .ausente: C.ausenteFondo
        default: C.chip
        }
    }

    var tinta: Color {
        switch self {
        case .confirmado: C.mentaTexto
        case .enCurso: C.seleccionTexto
        case .pendientePago: C.pagoTinta
        case .ausente: C.ausenteTinta
        case .atendido, .cancelado: C.sub
        default: C.tinta
        }
    }

    /// Los pasos que el panel puede dar desde cada estado (los de la API, turnos.PuedePasar).
    static let pasos: [String: [String]] = [
        "pendiente_pago": ["pendiente", "confirmado", "cancelado"],
        "pendiente": ["confirmado", "cancelado", "atendido", "ausente"],
        "confirmado": ["pendiente", "cancelado", "atendido", "ausente"],
        "atendido": ["ausente", "confirmado"],
        "ausente": ["atendido", "confirmado"]
    ]
    static func puedePasar(_ de: String, _ a: String) -> Bool { de == a || (pasos[de] ?? []).contains(a) }
}

struct Franja: Codable, Equatable, Hashable { var dia: Int; var desde: String; var hasta: String }

struct Profesional: Codable, Identifiable, Equatable, Hashable {
    var id: String
    var nombre: String
    var especialidad: String
    var activo: Bool
    var orden: Int
    var servicios: [String]
    var horario: [Franja]
    var email: String?
    var telefono: String?
    var bio: String?
    var miembroId: String?
    var foto: String?
}

/// Un servicio («prestación» en la web).
struct Servicio: Codable, Identifiable, Equatable, Hashable {
    var id: String
    var nombre: String
    var duracion: Int
    var precio: Int
    var activo: Bool
    var descripcion: String?
    var categoriaId: String?
    var sena: Int?
    var foto: String?
}

struct HorarioLibre: Codable, Equatable, Hashable { var hora: String; var inicio: Date; var profesionales: [String] }

struct Etiqueta: Codable, Identifiable, Equatable, Hashable { var id: String; var nombre: String; var color: String }

struct Cliente: Codable, Identifiable, Equatable, Hashable {
    var id: String
    var nombre: String
    var telefono: String
    var email: String
    var creadoEn: Date
    var visitas: Int
    var ultimaVisita: Date?
    var proximoTurno: Date?
    var gastado: Int
    var ausencias: Int
    var penalizado: Bool
    var etiquetas: [Etiqueta]
}

struct Penalidad: Codable, Identifiable, Equatable, Hashable {
    var id: String
    var clienteId: String
    var cliente: String
    var telefono: String
    var motivo: String
    var nota: String
    var ausencias: Int
    var hasta: Date
    var creadoEn: Date
    var levantadaEn: Date?
    var vigente: Bool
}

struct TurnoDeCliente: Codable, Identifiable, Equatable, Hashable {
    var id: String
    var codigo: String
    var servicio: String
    var profesional: String
    var inicio: Date
    var fin: Date
    var precio: Int
    var estado: String
    var origen: String
    var nota: String
}

struct Ficha: Codable, Equatable {
    var id: String
    var nombre: String
    var telefono: String
    var email: String
    var creadoEn: Date
    var visitas: Int
    var ultimaVisita: Date?
    var proximoTurno: Date?
    var gastado: Int
    var ausencias: Int
    var penalizado: Bool
    var etiquetas: [Etiqueta]
    var turnos: [TurnoDeCliente]
    var penalidad: Penalidad?
}

struct Nota: Codable, Identifiable, Equatable, Hashable { var id: String; var texto: String; var miembroId: String?; var autor: String; var creadoEn: Date }

/// Alguien en la lista de espera.
struct Anotado: Codable, Identifiable, Equatable, Hashable {
    var id: String
    var servicioId: String
    var servicio: String
    var profesionalId: String?
    var profesional: String?
    var nombre: String
    var email: String
    var telefono: String
    var desde: String
    var hasta: String
    var avisadoEn: Date?
    var creadoEn: Date
    /// cualquiera | manana | tarde
    var momento: String?
    /// Lo que pasó la última vez que se le ofreció: ofrecida, tomada, paso, vencida, cancelada.
    var resultado: String?

    var momentoTexto: String { ["manana": "Mañana", "tarde": "Tardes"][momento ?? ""] ?? "Todo el día" }
}

struct Resena: Codable, Identifiable, Equatable, Hashable {
    var id: String
    var turnoId: String
    var nombre: String
    var servicio: String
    var profesional: String
    var puntaje: Int
    var comentario: String
    var respuesta: String
    var respondidaEn: Date?
    var visible: Bool
    var creadoEn: Date
}

/// Un aviso de la campanita.
struct Notificacion: Codable, Identifiable, Equatable, Hashable {
    var id: String
    var tipo: String
    var titulo: String
    var texto: String
    var turnoId: String?
    var creadoEn: Date
    var nueva: Bool
    var pedidoId: String?

    var simbolo: String {
        switch tipo {
        case "reserva": "calendar.badge.plus"
        case "cancelacion": "calendar.badge.minus"
        case "reprogramacion": "calendar.badge.clock"
        case "confirmacion": "checkmark.message"
        case "cambio_pedido": "arrow.triangle.2.circlepath"
        case "sin_respuesta": "phone.badge.waveform"
        case "resena": "star.bubble"
        case "espera": "hourglass"
        case "ausente": "person.slash"
        case "equipo": "person.2"
        case "pago", "sena": "creditcard"
        case "aviso_fallido": "exclamationmark.bubble"
        default: "bell"
        }
    }
}

struct ParaLlamar: Codable, Identifiable, Equatable, Hashable {
    var turnoId: String
    var cliente: String
    var telefono: String
    var servicio: String
    var profesional: String
    var inicio: Date
    var estado: String
    var motivo: String
    var desde: Date
    var id: String { turnoId }
    var motivoTexto: String {
        ["cambio": "Pidió cambiarlo", "sin_respuesta": "No contestó el WhatsApp", "no_llego": "No le llegó el WhatsApp"][motivo] ?? motivo
    }
}

// MARK: - Ayudas de fecha y texto propias de los turnos

extension Formato {
    /// «45 min», «1 h», «1 h 30».
    static func duracion(_ minutos: Int) -> String {
        if minutos < 60 { return "\(minutos) min" }
        let h = minutos / 60, m = minutos % 60
        return m == 0 ? "\(h) h" : "\(h) h \(m)"
    }

    /// «45 min libres», «1 h 30 libre».
    static func libres(_ minutos: Int) -> String {
        "\(duracion(minutos)) \(minutos >= 60 && minutos < 120 ? "libre" : "libres")"
    }

    /// El día en formato de la API (2026-11-04), en hora de Argentina.
    static func diaISO(_ f: Date) -> String {
        var cal = Calendar(identifier: .gregorian)
        cal.timeZone = zona
        let c = cal.dateComponents([.year, .month, .day], from: f)
        return String(format: "%04d-%02d-%02d", c.year ?? 2026, c.month ?? 1, c.day ?? 1)
    }

    static var calendario: Calendar {
        var cal = Calendar(identifier: .gregorian)
        cal.timeZone = zona
        cal.locale = argentina
        cal.firstWeekday = 2
        return cal
    }

    /// «lunes 2 de noviembre, 10:30».
    static func diaYHora(_ f: Date) -> String { "\(fechaLarga(f)), \(horaDe(f))" }

    /// Minutos desde la medianoche de Argentina.
    static func minutosDelDia(_ f: Date) -> Int {
        let c = calendario.dateComponents([.hour, .minute], from: f)
        return (c.hour ?? 0) * 60 + (c.minute ?? 0)
    }
}
