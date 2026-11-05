import ActivityKit
import AppIntents
import Foundation
import Security
import WidgetKit

/// Lo que comparten la app, los widgets y la Live Activity: la sesión en el llavero compartido
/// (TLGrupoLlavero), el negocio elegido y una lectura mínima del día.
enum SesionCompartida {
    static let servicio = "ar.com.tengolibre.app.sesion"

    static var grupo: String? {
        guard let g = Bundle.main.object(forInfoDictionaryKey: "TLGrupoLlavero") as? String, !g.isEmpty, !g.hasPrefix("$(") else { return nil }
        return g
    }

    static func leer(_ cuenta: String) -> String? {
        var q: [String: Any] = [kSecClass as String: kSecClassGenericPassword,
                                kSecAttrService as String: servicio,
                                kSecAttrAccount as String: cuenta,
                                kSecReturnData as String: true,
                                kSecMatchLimit as String: kSecMatchLimitOne]
        if let g = grupo { q[kSecAttrAccessGroup as String] = g }
        var r: CFTypeRef?
        guard SecItemCopyMatching(q as CFDictionary, &r) == errSecSuccess, let d = r as? Data else { return nil }
        return String(data: d, encoding: .utf8)
    }

    static var api: URL {
        let s = Bundle.main.object(forInfoDictionaryKey: "TLApi") as? String ?? ""
        return URL(string: s.isEmpty || s.hasPrefix("$(") ? "https://api.tengolibre.com.ar" : s)!
    }

    static let zona = TimeZone(identifier: "America/Argentina/Buenos_Aires")!

    static func pedido(_ metodo: String, _ ruta: String, query: [URLQueryItem] = [], cuerpo: Data? = nil) -> URLRequest? {
        guard let token = leer("token") else { return nil }
        var c = URLComponents(url: api.appendingPathComponent(ruta), resolvingAgainstBaseURL: false)!
        if !query.isEmpty { c.queryItems = query }
        var r = URLRequest(url: c.url!, timeoutInterval: 15)
        r.httpMethod = metodo
        r.setValue("application/json", forHTTPHeaderField: "Accept")
        r.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        if let n = leer("negocio") { r.setValue(n, forHTTPHeaderField: "X-Negocio") }
        if let cuerpo { r.httpBody = cuerpo; r.setValue("application/json", forHTTPHeaderField: "Content-Type") }
        return r
    }
}

/// Un turno como lo ven los widgets.
struct TurnoChico: Codable, Hashable, Identifiable {
    struct Cli: Codable, Hashable { var nombre: String }
    var id: String
    var servicio: String
    var profesional: String
    var cliente: Cli
    var inicio: Date
    var fin: Date
    var estado: String
}

/// El día para los widgets: los turnos de hoy que siguen y cuántos quedan.
struct DiaChico: Codable, Hashable {
    var negocio: String
    var turnos: [TurnoChico]
    var hechos: Int
    var conSesion: Bool

    static let vacio = DiaChico(negocio: "", turnos: [], hechos: 0, conSesion: true)
    static let ejemplo = DiaChico(negocio: "Lumi", turnos: [
        TurnoChico(id: "1", servicio: "Corte y brushing", profesional: "Carla", cliente: .init(nombre: "Sofía Méndez"), inicio: Date().addingTimeInterval(1800), fin: Date().addingTimeInterval(5400), estado: "confirmado"),
        TurnoChico(id: "2", servicio: "Color", profesional: "Abril", cliente: .init(nombre: "Martina López"), inicio: Date().addingTimeInterval(7200), fin: Date().addingTimeInterval(10800), estado: "pendiente")
    ], hechos: 4, conSesion: true)

    var proximo: TurnoChico? { turnos.first }

    static func cargar() async -> DiaChico {
        var cal = Calendar(identifier: .gregorian)
        cal.timeZone = SesionCompartida.zona
        let f = DateFormatter()
        f.calendar = cal; f.timeZone = SesionCompartida.zona; f.dateFormat = "yyyy-MM-dd"
        let hoy = f.string(from: Date())
        guard let r = SesionCompartida.pedido("GET", "/v1/panel/turnos", query: [URLQueryItem(name: "desde", value: hoy), URLQueryItem(name: "hasta", value: hoy)]) else {
            return DiaChico(negocio: "", turnos: [], hechos: 0, conSesion: false)
        }
        struct Lista: Decodable { var turnos: [TurnoChico] }
        let d = JSONDecoder()
        d.dateDecodingStrategy = .custom { dec in
            let s = try dec.singleValueContainer().decode(String.self)
            let iso = ISO8601DateFormatter()
            iso.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
            if let x = iso.date(from: s) { return x }
            iso.formatOptions = [.withInternetDateTime]
            if let x = iso.date(from: s) { return x }
            throw DecodingError.dataCorrupted(.init(codingPath: dec.codingPath, debugDescription: s))
        }
        guard let (datos, resp) = try? await URLSession.shared.data(for: r), (resp as? HTTPURLResponse)?.statusCode == 200,
              let l = try? d.decode(Lista.self, from: datos) else { return .vacio }
        let ahora = Date()
        let activos = l.turnos.filter { ["pendiente_pago", "pendiente", "confirmado"].contains($0.estado) }
        return DiaChico(negocio: SesionCompartida.leer("negocioNombre") ?? "",
                        turnos: activos.filter { $0.fin > ahora }.sorted { $0.inicio < $1.inicio },
                        hechos: l.turnos.filter { $0.estado == "atendido" }.count,
                        conSesion: true)
    }
}

// MARK: - Live Activity (el turno en curso)

struct AtributosTurno: ActivityAttributes {
    struct ContentState: Codable, Hashable {
        var fin: Date
        var siguiente: String
    }
    var turnoId: String
    var cliente: String
    var servicio: String
    var profesional: String
    var inicio: Date
}

/// «Atendido» desde la Dynamic Island o la pantalla bloqueada: marca el turno sin abrir la app.
struct MarcarAtendido: LiveActivityIntent {
    static var title: LocalizedStringResource = "Marcar atendido"
    static var openAppWhenRun = false

    @Parameter(title: "Turno") var turnoId: String

    init() {}
    init(turnoId: String) { self.turnoId = turnoId }

    func perform() async throws -> some IntentResult {
        if let r = SesionCompartida.pedido("PATCH", "/v1/panel/turnos/\(turnoId)", cuerpo: Data(#"{"estado":"atendido"}"#.utf8)) {
            _ = try? await URLSession.shared.data(for: r)
        }
        for a in Activity<AtributosTurno>.activities where a.attributes.turnoId == turnoId {
            await a.end(nil, dismissalPolicy: .immediate)
        }
        WidgetCenter.shared.reloadAllTimelines()
        return .result()
    }
}
