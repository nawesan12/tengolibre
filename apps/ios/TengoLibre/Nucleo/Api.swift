import Foundation

/// Un error de la API (application/problem+json) o de la red, con lo que hay que decirle a la persona.
struct ErrorApi: Error, Equatable {
    var estado: Int
    var codigo: String
    var titulo: String
    var detalle: String
    /// Ingreso con contraseña: intentos que quedan antes de pausarlo.
    var restantes: Int?
    /// Segundos para volver a probar (429).
    var reintentarEn: Int?

    var sinConexion: Bool { codigo == "sin_conexion" }
    var sinSesion: Bool { estado == 401 && codigo == "sin_sesion" }

    static let sinRed = ErrorApi(estado: 0, codigo: "sin_conexion", titulo: "Sin conexión",
                                 detalle: "No hay internet. Revisá la conexión y probá de nuevo.")
    static let desconocido = ErrorApi(estado: 0, codigo: "interno", titulo: "Algo salió mal",
                                      detalle: "Probá de nuevo en un rato.")
}

/// El cliente de la API de Tengo Libre (apps/api). Va directo contra api.tengolibre.com.ar con la
/// sesión propia en `Authorization: Bearer` y, en /v1/panel, el negocio elegido en `X-Negocio`
/// (igual que el servidor de la web, apps/web/src/lib/server/api.ts).
final class Api: @unchecked Sendable {
    let base: URL
    var token: String?
    /// El negocio con el que se trabaja (el de arriba de Hoy y Agenda).
    var negocio: String?
    /// Se llama cuando la API dice que la sesión ya no vale (401 sin_sesion): la app vuelve al ingreso.
    var alPerderSesion: (@MainActor () -> Void)?
    private let http: URLSession

    init(base: URL, token: String? = nil, http: URLSession = .shared) {
        self.base = base
        self.token = token
        self.http = http
    }

    /// La API de la configuración (Info.plist, TL_API). En Debug se puede pisar con el argumento
    /// `-TLApi http://127.0.0.1:4200`.
    static func urlConfigurada() -> URL {
        #if DEBUG
        if let s = UserDefaults.standard.string(forKey: "TLApi"), let u = URL(string: s) { return u }
        #endif
        let s = Bundle.main.object(forInfoDictionaryKey: "TLApi") as? String ?? ""
        return URL(string: s.isEmpty ? "https://api.tengolibre.com.ar" : s)!
    }

    static let decodificador: JSONDecoder = {
        let d = JSONDecoder()
        d.dateDecodingStrategy = .custom { dec in
            let c = try dec.singleValueContainer()
            let s = try c.decode(String.self)
            if let f = Fecha.iso(s) { return f }
            throw DecodingError.dataCorruptedError(in: c, debugDescription: "fecha \(s)")
        }
        return d
    }()

    static let codificador: JSONEncoder = {
        let e = JSONEncoder()
        e.dateEncodingStrategy = .iso8601
        return e
    }()

    private struct Problema: Decodable {
        var status: Int?
        var code: String?
        var title: String?
        var detail: String?
        var restantes: Int?
        var issues: [Campo]?
        struct Campo: Decodable { var path: String; var message: String }
    }

    func pedir(_ metodo: String, _ ruta: String, query: [URLQueryItem] = [], cuerpo: Data? = nil, tipo: String = "application/json") async throws -> Data {
        var c = URLComponents(url: base.appendingPathComponent(ruta), resolvingAgainstBaseURL: false)!
        if !query.isEmpty { c.queryItems = query }
        var r = URLRequest(url: c.url!, timeoutInterval: 20)
        r.httpMethod = metodo
        r.setValue("application/json", forHTTPHeaderField: "Accept")
        if let token { r.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization") }
        if let negocio, ruta.hasPrefix("/v1/panel/") { r.setValue(negocio, forHTTPHeaderField: "X-Negocio") }
        if let cuerpo {
            r.httpBody = cuerpo
            r.setValue(tipo, forHTTPHeaderField: "Content-Type")
        }
        let datos: Data
        let resp: URLResponse
        do {
            (datos, resp) = try await http.data(for: r)
        } catch let e as URLError {
            if e.code == .cancelled { throw CancellationError() }
            throw ErrorApi.sinRed
        }
        guard let h = resp as? HTTPURLResponse else { throw ErrorApi.desconocido }
        if (200..<300).contains(h.statusCode) { return datos }
        let p = (try? JSONDecoder().decode(Problema.self, from: datos)) ?? Problema()
        let e = ErrorApi(estado: h.statusCode, codigo: p.code ?? "interno", titulo: p.title ?? "Algo salió mal",
                         detalle: p.issues?.first?.message ?? p.detail ?? p.title ?? "Probá de nuevo en un rato.", restantes: p.restantes,
                         reintentarEn: h.value(forHTTPHeaderField: "Retry-After").flatMap(Int.init))
        if e.sinSesion, token != nil, let alPerderSesion {
            await alPerderSesion()
        }
        throw e
    }

    /// Sube una imagen (POST /v1/panel/archivos, multipart con el campo «archivo») y devuelve su URL.
    func subirImagen(_ datos: Data, nombre: String = "imagen.jpg") async throws -> String {
        let limite = "tl-\(UUID().uuidString)"
        var b = Data()
        b.append(Data("--\(limite)\r\nContent-Disposition: form-data; name=\"archivo\"; filename=\"\(nombre)\"\r\nContent-Type: image/jpeg\r\n\r\n".utf8))
        b.append(datos)
        b.append(Data("\r\n--\(limite)--\r\n".utf8))
        struct Archivo: Decodable { var id: String; var url: String }
        let a: Archivo = try decodificar(await pedir("POST", "/v1/panel/archivos", cuerpo: b, tipo: "multipart/form-data; boundary=\(limite)"))
        return a.url
    }

    /// Un pedido largo (SSE de /v1/panel/eventos): sin límite de tiempo y con la sesión y el negocio.
    func flujo(_ ruta: String) -> (URLRequest, URLSession) {
        var r = URLRequest(url: base.appendingPathComponent(ruta), timeoutInterval: .infinity)
        r.setValue("text/event-stream", forHTTPHeaderField: "Accept")
        if let token { r.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization") }
        if let negocio { r.setValue(negocio, forHTTPHeaderField: "X-Negocio") }
        return (r, http)
    }

    func get<T: Decodable>(_ ruta: String, query: [URLQueryItem] = []) async throws -> T {
        try decodificar(await pedir("GET", ruta, query: query))
    }

    func enviar<T: Decodable, C: Encodable>(_ metodo: String, _ ruta: String, _ cuerpo: C) async throws -> T {
        try decodificar(await pedir(metodo, ruta, cuerpo: Api.codificador.encode(cuerpo)))
    }

    /// Un DELETE que responde con datos (p. ej. borrar los datos de una clienta: borrado o anonimizado).
    func borrar<T: Decodable>(_ ruta: String) async throws -> T {
        try decodificar(await pedir("DELETE", ruta))
    }

    /// Un pedido sin respuesta (204).
    func hacer<C: Encodable>(_ metodo: String, _ ruta: String, _ cuerpo: C) async throws {
        _ = try await pedir(metodo, ruta, cuerpo: Api.codificador.encode(cuerpo))
    }

    func hacer(_ metodo: String, _ ruta: String) async throws {
        _ = try await pedir(metodo, ruta)
    }

    private func decodificar<T: Decodable>(_ d: Data) throws -> T {
        do {
            return try Api.decodificador.decode(T.self, from: d)
        } catch {
            #if DEBUG
            print("Api: no pude leer \(T.self): \(error)")
            #endif
            throw ErrorApi.desconocido
        }
    }
}

/// Lo último que se trajo de cada lista, para verlo sin conexión (handoff 5a: «Ves lo guardado hoy,
/// 14:20»). Un archivo por clave en Caches, con la fecha en que se guardó.
enum Guardado {
    private static var carpeta: URL {
        let u = FileManager.default.urls(for: .cachesDirectory, in: .userDomainMask)[0].appendingPathComponent("guardado")
        try? FileManager.default.createDirectory(at: u, withIntermediateDirectories: true)
        return u
    }

    private static func archivo(_ clave: String) -> URL {
        carpeta.appendingPathComponent(clave.addingPercentEncoding(withAllowedCharacters: .alphanumerics) ?? clave)
    }

    static func guardar<T: Encodable>(_ valor: T, clave: String) {
        guard let d = try? Api.codificador.encode(valor) else { return }
        try? d.write(to: archivo(clave), options: .atomic)
    }

    static func leer<T: Decodable>(_ tipo: T.Type, clave: String) -> (T, Date)? {
        let u = archivo(clave)
        guard let d = try? Data(contentsOf: u), let v = try? Api.decodificador.decode(T.self, from: d) else { return nil }
        let f = (try? FileManager.default.attributesOfItem(atPath: u.path)[.modificationDate] as? Date) ?? Date()
        return (v, f)
    }

    /// Al cerrar sesión o cambiar de negocio no queda nada del anterior.
    static func borrarTodo() {
        try? FileManager.default.removeItem(at: carpeta)
    }
}
