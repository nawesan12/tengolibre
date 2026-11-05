import Foundation
import Observation

/// Una oferta del hueco a la lista de espera (docs/api/espera-ofertas.md).
struct Oferta: Codable, Identifiable, Equatable, Hashable {
    struct Intento: Codable, Equatable, Hashable {
        var id: String
        var esperaId: String?
        var nombre: String
        var orden: Int
        var estado: String
        var ofrecidaEn: Date?
        var venceEn: Date?
        var respondidaEn: Date?
    }
    var id: String
    var estado: String
    var origen: String
    var servicioId: String
    var servicio: String
    var profesionalId: String
    var profesional: String
    var inicio: Date
    var fin: Date
    var precio: Int
    var turnoId: String?
    var creadoEn: Date
    var cerradaEn: Date?
    var actual: Intento?
    var sigue: Intento?
    var cola: [Intento]
}

/// Un turno que hay que mover porque la profesional pidió días libres (dias-libres.md).
struct AReprogramar: Codable, Identifiable, Equatable, Hashable {
    struct TurnoR: Codable, Equatable, Hashable {
        var id: String
        var inicio: Date
        var fin: Date
        var servicio: String
        var profesional: String
        var estado: String
        var cliente: ClienteDelTurno
    }
    var id: String
    var pedidoId: String
    var motivo: String
    var resueltoEn: Date?
    var turno: TurnoR
}

/// Un pedido de días libres de una profesional (dias-libres.md).
struct PedidoDias: Codable, Identifiable, Equatable, Hashable {
    var id: String
    var profesionalId: String
    var profesional: String
    var desde: String
    var hasta: String
    var motivo: String
    var nota: String
    var estado: String
    var turnos: Int
    var afectados: Int
    var creadoEn: Date
    var motivoTexto: String { ["vacaciones": "Vacaciones", "tramite": "Trámite", "salud": "Salud", "otro": "Otro"][motivo] ?? motivo }
}

/// Los primeros pasos del negocio (arranque).
struct Arranque: Codable, Equatable {
    struct Paso: Codable, Equatable, Hashable { var clave: String; var titulo: String; var listo: Bool; var obligatorio: Bool }
    var completo: Bool
    var reservable: Bool
    var pasos: [Paso]
}

/// Lo que usan Hoy y Agenda: los turnos de un rango, el catálogo (profesionales y servicios), la
/// lista de espera y lo que hay que atender. Se guarda para verlo sin conexión.
@Observable
@MainActor
final class DatosDia {
    enum Carga: Equatable { case cargando, listo, error(String) }

    var carga: Carga = .cargando
    var turnos: [Turno] = []
    var profesionales: [Profesional] = []
    var servicios: [Servicio] = []
    var espera: [Anotado] = []
    var ofertas: [Oferta] = []
    var paraLlamar: [ParaLlamar] = []
    var reprogramar: [AReprogramar] = []
    var arranque: Arranque?
    var resenaSinResponder: Resena?
    var pedidos: [PedidoDias] = []
    var mpConectado: Bool?
    /// Ids de los turnos que llegaron mientras se miraba (para el talón menta que entra desde arriba).
    var recien: Set<String> = []

    private struct Turnos: Codable { var turnos: [Turno] }
    private struct Profesionales: Codable { var profesionales: [Profesional] }
    private struct Servicios: Codable { var servicios: [Servicio] }
    private struct Espera: Codable { var espera: [Anotado] }
    private struct Ofertas: Codable { var ofertas: [Oferta] }
    private struct Llamar: Codable { var turnos: [ParaLlamar] }
    private struct Reprog: Codable { var turnos: [AReprogramar] }
    private struct Resenas: Codable { var resenas: [Resena] }
    private struct Pedidos: Codable { var pedidos: [PedidoDias] }
    private struct MP: Codable { var disponible: Bool; var conectada: Bool }
    private struct Todo: Codable { var turnos: [Turno]; var profesionales: [Profesional]; var servicios: [Servicio]; var espera: [Anotado] }

    /// Trae `desde`…`hasta` (días) y todo lo de alrededor.
    func cargar(_ estado: Estado, desde: Date, hasta: Date) async {
        let api = estado.api
        let clave = "dia-\(estado.negocio?.id ?? "")-\(Formato.diaISO(desde))-\(Formato.diaISO(hasta))"
        if turnos.isEmpty, let (g, f) = Guardado.leer(Todo.self, clave: clave) {
            turnos = g.turnos; profesionales = g.profesionales; servicios = g.servicios; espera = g.espera
            estado.guardadoEn = f
        }
        let mostrador = !estado.esProfesional
        let q = [URLQueryItem(name: "desde", value: Formato.diaISO(desde)), URLQueryItem(name: "hasta", value: Formato.diaISO(hasta))]
        do {
            let t: Turnos = try await api.get("/v1/panel/turnos", query: q)
            let p: Profesionales = try await api.get("/v1/panel/profesionales")
            let s: Servicios? = try? await api.get("/v1/panel/servicios")
            let e: Espera? = mostrador ? (try? await api.get("/v1/panel/espera")) : nil
            let o: Ofertas? = mostrador ? (try? await api.get("/v1/panel/espera/ofertas")) : nil
            let l: Llamar? = mostrador ? (try? await api.get("/v1/panel/para-llamar")) : nil
            let r: Reprog? = mostrador ? (try? await api.get("/v1/panel/reprogramar")) : nil
            let ar: Arranque? = try? await api.get("/v1/panel/arranque")
            let rs: Resenas? = estado.rol == "recepcion" ? nil : (try? await api.get("/v1/panel/resenas"))
            let pd: Pedidos? = estado.gestiona ? (try? await api.get("/v1/panel/dias-libres", query: [URLQueryItem(name: "estado", value: "pendiente")])) : nil
            let mp: MP? = estado.gestiona ? (try? await api.get("/v1/panel/mercadopago")) : nil
            let (tt, pp) = (t, p)
            let antes = Set(turnos.map(\.id))
            if carga == .listo {
                let nuevos = tt.turnos.filter { !antes.contains($0.id) && $0.origen != "panel" }
                if !nuevos.isEmpty { recien.formUnion(nuevos.map(\.id)); Haptico.suave() }
            }
            turnos = tt.turnos
            profesionales = pp.profesionales.sorted { $0.orden < $1.orden }
            servicios = s?.servicios ?? servicios
            espera = e?.espera ?? []
            ofertas = (o?.ofertas ?? []).filter { $0.estado == "ofreciendo" }
            paraLlamar = l?.turnos ?? []
            reprogramar = (r?.turnos ?? []).filter { $0.resueltoEn == nil }
            arranque = ar
            resenaSinResponder = (rs?.resenas ?? []).first { $0.respondidaEn == nil && $0.respuesta.isEmpty }
            pedidos = pd?.pedidos ?? []
            if let m = mp, m.disponible { mpConectado = m.conectada } else { mpConectado = nil }
            carga = .listo
            estado.sinConexion = false
            Guardado.guardar(Todo(turnos: turnos, profesionales: profesionales, servicios: servicios, espera: espera), clave: clave)
        } catch let x as ErrorApi {
            if x.sinConexion { estado.sinConexion = true }
            carga = turnos.isEmpty ? .error(x.detalle) : .listo
        } catch {
            carga = turnos.isEmpty ? .error("Probá de nuevo en un rato.") : .listo
        }
    }

    /// Cambia el estado de un turno al toque (y vuelve atrás si la API no lo acepta).
    func cambiar(_ estado: Estado, _ t: Turno, a nuevo: String) async {
        guard let i = turnos.firstIndex(where: { $0.id == t.id }) else { return }
        let antes = turnos[i].estado
        turnos[i].estado = nuevo
        do {
            try await estado.api.hacer("PATCH", "/v1/panel/turnos/\(t.id)", AccionesTurno.CambioEstado(estado: nuevo))
            Haptico.exito()
            let texto = ["atendido": "Marcado atendido", "ausente": "Marcado ausente", "confirmado": "Turno confirmado", "cancelado": "Turno cancelado"][nuevo] ?? "Listo"
            estado.avisar(Toast(simbolo: "checkmark", titulo: texto, texto: "\(t.cliente.nombre) · \(Formato.horaDe(t.inicio))", accion: "Deshacer")) {
                try? await estado.api.hacer("PATCH", "/v1/panel/turnos/\(t.id)", AccionesTurno.CambioEstado(estado: antes))
            }
        } catch {
            if let j = turnos.firstIndex(where: { $0.id == t.id }) { turnos[j].estado = antes }
            estado.avisar(error)
        }
    }

    /// Cuántos de la lista esperan algo que entre en ese hueco (con esa profesional o con cualquiera).
    func esperanPara(_ h: Dia.Hueco) -> [Anotado] {
        espera.filter { a in a.avisadoEn == nil && (a.profesionalId == nil || a.profesionalId == h.profesionalId) }
    }

    func ofertaPara(_ h: Dia.Hueco, dia: Date) -> Oferta? {
        ofertas.first { $0.profesionalId == h.profesionalId && Formato.diaISO($0.inicio) == Formato.diaISO(dia) && Formato.minutosDelDia($0.inicio) >= h.desde && Formato.minutosDelDia($0.inicio) < h.hasta }
    }
}
