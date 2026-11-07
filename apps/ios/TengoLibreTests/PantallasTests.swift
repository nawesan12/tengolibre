import SwiftUI
import XCTest
@testable import TengoLibre

/// Responde a la app con datos fijos, por «MÉTODO ruta» (o sólo ruta para GET). Lo que no está: GET → 404,
/// lo demás → 200 con `{}`.
final class ApiFalsa: URLProtocol {
    nonisolated(unsafe) static var respuestas: [String: String] = [:]
    override class func canInit(with request: URLRequest) -> Bool { true }
    override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }
    override func startLoading() {
        let ruta = request.url?.path ?? ""
        let metodo = request.httpMethod ?? "GET"
        let cuerpo = ApiFalsa.respuestas["\(metodo) \(ruta)"] ?? (metodo == "GET" ? ApiFalsa.respuestas[ruta] : "{}")
        let r = HTTPURLResponse(url: request.url!, statusCode: cuerpo == nil ? 404 : 200, httpVersion: nil,
                                headerFields: ["content-type": "application/json"])!
        client?.urlProtocol(self, didReceive: r, cacheStoragePolicy: .notAllowed)
        client?.urlProtocol(self, didLoad: Data((cuerpo ?? #"{"status":404,"code":"no_existe","title":"No existe","detail":"No existe"}"#).utf8))
        client?.urlProtocolDidFinishLoading(self)
    }
    override func stopLoading() {}
}

/// Monta cada pantalla en una ventana, en Release y con la API simulada: si se cae al abrirla (como el
/// `async let` que cerraba Tu Local Web en TestFlight), salta acá y el CI imprime el reporte.
@MainActor
final class PantallasTests: XCTestCase {
    static let capturas: URL? = {
        guard let d = ProcessInfo.processInfo.environment["CAPTURAS"], !d.isEmpty else { return nil }
        let u = URL(fileURLWithPath: d)
        try? FileManager.default.createDirectory(at: u, withIntermediateDirectories: true)
        return u
    }()

    static func iso(_ minutos: Int) -> String {
        ISO8601DateFormatter().string(from: Date().addingTimeInterval(TimeInterval(minutos * 60)))
    }

    static let turno = """
    {"id": "t1", "codigo": "ABC123", "servicioId": "s1", "servicio": "Corte y brushing", "profesionalId": "p1", "profesional": "Carla",
     "cliente": {"id": "c1", "nombre": "Sofía Méndez", "telefono": "1144712290", "email": "sofia@prueba.test"},
     "inicio": "\(iso(-10))", "fin": "\(iso(35))", "precio": 18000, "estado": "confirmado", "origen": "web", "nota": "",
     "venceEn": null, "creadoEn": "\(iso(-600))", "serieId": null}
    """
    static let turno2 = """
    {"id": "t2", "codigo": "DEF456", "servicioId": "s1", "servicio": "Color", "profesionalId": "p1", "profesional": "Carla",
     "cliente": {"id": "c2", "nombre": "Martina López", "telefono": "1155550000", "email": ""},
     "inicio": "\(iso(120))", "fin": "\(iso(180))", "precio": 30000, "estado": "pendiente_pago", "origen": "web", "nota": "Piel sensible",
     "venceEn": "\(iso(15))", "creadoEn": "\(iso(-5))"}
    """
    static let profesional = """
    {"id": "p1", "nombre": "Carla Ruiz", "especialidad": "Colorista", "activo": true, "orden": 1, "servicios": ["s1"],
     "horario": [{"dia": 0, "desde": "00:00", "hasta": "23:59"}, {"dia": 1, "desde": "00:00", "hasta": "23:59"}, {"dia": 2, "desde": "00:00", "hasta": "23:59"},
                 {"dia": 3, "desde": "00:00", "hasta": "23:59"}, {"dia": 4, "desde": "00:00", "hasta": "23:59"}, {"dia": 5, "desde": "00:00", "hasta": "23:59"},
                 {"dia": 6, "desde": "00:00", "hasta": "23:59"}], "email": "carla@prueba.test", "telefono": "", "bio": "", "miembroId": null, "foto": null}
    """

    /// Las respuestas compartidas con Android (apps/pruebas/respuestas.json): «MÉTODO ruta» o sólo la ruta para GET.
    /// Los marcadores "$iso(N)" y "$dia(N)" pasan a ahora + N minutos y hoy + N días, como en ModoPrueba.kt.
    static func respuestas() -> [String: String] {
        guard let url = Bundle(for: PantallasTests.self).url(forResource: "respuestas", withExtension: "json"),
              let datos = try? Data(contentsOf: url),
              let raiz = try? JSONSerialization.jsonObject(with: datos) as? [String: Any] else { return [:] }
        func fechas(_ v: Any) -> Any {
            if let d = v as? [String: Any] { return d.mapValues(fechas) }
            if let a = v as? [Any] { return a.map(fechas) }
            guard let t = v as? String, t.hasPrefix("$"), t.hasSuffix(")"), let abre = t.firstIndex(of: "("),
                  let n = Int(t[t.index(after: abre)..<t.index(before: t.endIndex)]) else { return v }
            switch t[..<abre] {
            case "$iso": return iso(n)
            case "$dia": return Formato.diaISO(Date().addingTimeInterval(TimeInterval(n * 86400)))
            default: return v
            }
        }
        var r: [String: String] = [:]
        for (k, v) in raiz where !k.hasPrefix("_") {
            if let d = try? JSONSerialization.data(withJSONObject: fechas(v), options: [.fragmentsAllowed]) { r[k] = String(decoding: d, as: UTF8.self) }
        }
        return r
    }

    override func setUp() {
        ApiFalsa.respuestas = Self.respuestas()
        XCTAssertFalse(ApiFalsa.respuestas.isEmpty, "Falta respuestas.json en el paquete de pruebas")
    }

    private func montar<V: View>(_ vista: V, rol: String = "duenio", segundos: Double = 2, nombre: String = #function) {
        montarEn(vista, rol: rol, segundos: segundos, nombre: nombre, oscuro: false)
        if Self.capturas != nil { montarEn(vista, rol: rol, segundos: segundos, nombre: nombre, oscuro: true) }
    }

    private func montarEn<V: View>(_ vista: V, rol: String, segundos: Double, nombre: String, oscuro: Bool) {
        let conf = URLSessionConfiguration.ephemeral
        conf.protocolClasses = [ApiFalsa.self]
        let estado = Estado(api: Api(base: URL(string: "https://api.prueba")!, token: "x", http: URLSession(configuration: conf)))
        estado.persona = Persona(clerkId: "u1", email: "laura@lumi.test", nombre: "Laura", foto: "")
        estado.negocios = [NegocioDeLaPersona(id: "n1", slug: "lumi", nombre: "Lumi", rol: rol)]
        estado.usar(estado.negocios[0])
        let nav = Navegacion()
        let raiz = NavigationStack { vista.destinos() }
            .environment(estado)
            .environment(nav)
        let ventana = UIWindow(frame: CGRect(x: 0, y: 0, width: 393, height: 852))
        ventana.overrideUserInterfaceStyle = oscuro ? .dark : .light
        ventana.rootViewController = UIHostingController(rootView: raiz)
        ventana.makeKeyAndVisible()
        RunLoop.main.run(until: Date().addingTimeInterval(segundos))
        if let dir = Self.capturas {
            let img = UIGraphicsImageRenderer(bounds: ventana.bounds).image { _ in ventana.drawHierarchy(in: ventana.bounds, afterScreenUpdates: true) }
            let n = nombre.replacingOccurrences(of: "()", with: "").replacingOccurrences(of: "test", with: "")
            try? img.pngData()?.write(to: dir.appendingPathComponent("\(n)-\(oscuro ? "oscuro" : "claro").png"))
        }
        ventana.isHidden = true
    }

    private func turno() throws -> Turno {
        let d = try Api.decodificador.decode(Turno.self, from: Data(Self.turno.utf8))
        return d
    }

    func testDecodificaTurno() throws { XCTAssertEqual(try turno().cliente.nombre, "Sofía Méndez") }
    func testHoy() { montar(Hoy()) }
    func testHoyProfesional() { montar(Hoy(), rol: "profesional") }
    func testHoyVacio() {
        ApiFalsa.respuestas["/v1/panel/turnos"] = "{\"turnos\": []}"
        montar(Hoy())
    }
    func testAgenda() { montar(Agenda()) }
    func testEspera() { montar(ListaEspera()) }
    func testClientes() { montar(Clientes()) }
    func testFicha() { montar(FichaCliente(id: "c1")) }
    func testResenas() { montar(Resenas()) }
    func testMas() { montar(Mas()) }
    func testMasProfesional() { montar(Mas(), rol: "profesional") }
    func testMasRecepcion() { montar(Mas(), rol: "recepcion") }
    func testDetalle() throws { montar(DetalleTurno(turno: try turno())) }
    func testNuevoTurno() { montar(NuevoTurno(inicio: NuevoTurnoInicio())) }
    func testCobrar() throws { montar(Cobrar(turno: try turno(), caja: nil) {}) }
    func testReprogramar() throws { montar(Reprogramar(turno: try turno()) { _ in }) }
    func testAvisos() { montar(Avisos()) }
    func testCompartir() { montar(Compartir()) }
    func testSecciones() {
        let todas: [DestinoMas] = [.primerosPasos, .resenas, .suscripcion, .diasLibres, .cobros, .servicios, .equipo, .profesional("p1"), .miHorario,
                                   .membresias, .cupones, .referidos, .penalidades, .etiquetas, .caja, .reportes, .miWeb, .plantillas, .dominio,
                                   .integraciones, .configuracion, .avisosClientes, .whatsapp, .auditoria, .sucursales, .cuenta, .recorrido]
        for d in todas {
            montar(NavigationLinkDirecto(destino: d), segundos: 1, nombre: "Seccion-\(String(describing: d).filter { $0.isLetter || $0.isNumber })")
        }
    }
    func testRecorrido() { montar(Recorrido {}, segundos: 1) }
    func testRecorridoProfesional() { montar(Recorrido {}, rol: "profesional", segundos: 1) }
    func testBuscar() { montar(Buscar(), segundos: 1) }
    func testPotencia() { montar(Potencia()) }
    func testPlataHoy() { montar(ScrollView { PlataHoy() }) }
    func testFichaFiel() {
        ApiFalsa.respuestas["/v1/panel/membresias/planes"] = "{\"planes\": []}"
        montar(FichaCliente(id: "c1"))
    }
    func testIngreso() { montar(Ingreso(), segundos: 1) }
    func testElegirModo() { montar(ElegirModo(), segundos: 1) }
    func testElegirNegocio() { montar(ElegirNegocio(), segundos: 1) }
    func testPermisos() { montar(PermisoAvisos(), segundos: 1) }
    func testAlta() { montar(Alta(), segundos: 1) }
    func testPrincipal() { montar(Principal()) }
    func testPrincipalProfesional() { montar(Principal(), rol: "profesional") }
    func testCliente() {
        MisTurnos.codigos = []
        montar(PrincipalCliente())
    }
    func testReservar() { montar(Reservar(slug: "lumi") { _ in }) }
    func testLugares() { montar(Lugares { _ in }) }
}

/// Abre una sección de «Más» como si se la hubiese tocado.
private struct NavigationLinkDirecto: View {
    var destino: DestinoMas
    var body: some View {
        Color.clear.navigationDestination(isPresented: .constant(true)) { vista }
    }
    @ViewBuilder private var vista: some View {
        switch destino {
        case .primerosPasos: Potencia()
        case .resenas: Resenas()
        case .suscripcion: SuscripcionVista()
        case .diasLibres: DiasLibres()
        case .cobros: Cobros()
        case .servicios: Servicios()
        case .equipo: Equipo()
        case .profesional(let id): FichaProfesional(id: id)
        case .miHorario: MiHorario()
        case .membresias: Membresias()
        case .cupones: Cupones()
        case .referidos: Referidos()
        case .penalidades: Penalidades()
        case .etiquetas: Etiquetas()
        case .caja: CajaDelDia()
        case .reportes: Reportes()
        case .miWeb: MiWeb()
        case .plantillas: Plantillas()
        case .dominio: DominioVista()
        case .integraciones: Integraciones()
        case .configuracion: Configuracion()
        case .avisosClientes: AvisosClientes()
        case .whatsapp: WhatsAppVista()
        case .auditoria: Auditoria()
        case .sucursales: Sucursales()
        case .cuenta: Cuenta()
        case .recorrido: RecorridoOtraVez()
        }
    }
}

final class RutasTests: XCTestCase {
    func testRutasDePush() {
        XCTAssertEqual(Ruta(texto: "turnos/abc"), .turno("abc"))
        XCTAssertEqual(Ruta(texto: "notificaciones"), .avisos)
        XCTAssertEqual(Ruta(texto: "tengolibre://espera"), .espera)
    }
    func testDia() {
        XCTAssertEqual(Dia.aMin("09:30"), 570)
        XCTAssertEqual(Dia.hhmm(570), "9:30")
    }
}
