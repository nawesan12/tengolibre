import Foundation

/// Un JSON cualquiera: para editar un producto sin perder los campos que la app no muestra (opciones,
/// variantes, canales) al mandarlo de vuelta.
enum ValorJSON: Codable, Equatable {
    case texto(String), numero(Double), booleano(Bool), lista([ValorJSON]), objeto([String: ValorJSON]), nulo

    init(from decoder: Decoder) throws {
        let c = try decoder.singleValueContainer()
        if c.decodeNil() { self = .nulo }
        else if let b = try? c.decode(Bool.self) { self = .booleano(b) }
        else if let n = try? c.decode(Double.self) { self = .numero(n) }
        else if let s = try? c.decode(String.self) { self = .texto(s) }
        else if let l = try? c.decode([ValorJSON].self) { self = .lista(l) }
        else { self = .objeto(try c.decode([String: ValorJSON].self)) }
    }

    func encode(to encoder: Encoder) throws {
        var c = encoder.singleValueContainer()
        switch self {
        case let .texto(s): try c.encode(s)
        case let .numero(n):
            if n == n.rounded(), abs(n) < 1e15 { try c.encode(Int(n)) } else { try c.encode(n) }
        case let .booleano(b): try c.encode(b)
        case let .lista(l): try c.encode(l)
        case let .objeto(o): try c.encode(o)
        case .nulo: try c.encodeNil()
        }
    }

    var texto: String? { if case let .texto(s) = self { return s }; return nil }
    var entero: Int? { if case let .numero(n) = self { return Int(n) }; return nil }
    var bool: Bool? { if case let .booleano(b) = self { return b }; return nil }
    var lista: [ValorJSON]? { if case let .lista(l) = self { return l }; return nil }
    subscript(_ k: String) -> ValorJSON? { if case let .objeto(o) = self { return o[k] }; return nil }
}

/// La web de Tengo Libre (tengolibre.com.ar): la página pública de cada negocio y la reserva.
enum Web {
    static let base: URL = {
        let s = Bundle.main.object(forInfoDictionaryKey: "TLWeb") as? String ?? ""
        return URL(string: s.isEmpty ? "https://tengolibre.com.ar" : s)!
    }()

    /// La página de reservas de un negocio: tengolibre.com.ar/{slug}.
    static func pagina(_ slug: String) -> URL { base.appending(path: slug) }

    /// Lo que se muestra del link, sin https://.
    static func textoPagina(_ slug: String) -> String { "\(base.host() ?? "tengolibre.com.ar")/\(slug)" }
}
