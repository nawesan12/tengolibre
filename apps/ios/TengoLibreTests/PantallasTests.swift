import SwiftUI
import XCTest
@testable import TengoLibre

/// Responde a la app con datos fijos, por ruta (lo que no está, 404).
final class ApiFalsa: URLProtocol {
    nonisolated(unsafe) static var respuestas: [String: String] = [:]
    override class func canInit(with request: URLRequest) -> Bool { true }
    override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }
    override func startLoading() {
        let ruta = request.url?.path ?? ""
        let cuerpo = ApiFalsa.respuestas[ruta]
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

    override func setUp() {
        let cliente = """
        {"id": "c1", "nombre": "Sofía Méndez", "telefono": "1144712290", "email": "sofia@prueba.test", "creadoEn": "\(Self.iso(-90000))",
         "visitas": 6, "ultimaVisita": "\(Self.iso(-20000))", "proximoTurno": "\(Self.iso(120))", "gastado": 98000, "ausencias": 1, "penalizado": false,
         "etiquetas": [{"id": "e1", "nombre": "VIP", "color": "#3ccc90"}]}
        """
        ApiFalsa.respuestas = [
            "/v1/panel/turnos": "{\"turnos\": [\(Self.turno), \(Self.turno2)]}",
            "/v1/panel/turnos/t1": Self.turno,
            "/v1/panel/turnos/t1/caja": """
            {"turnoId": "t1", "cliente": "Sofía", "servicio": "Corte", "estadoTurno": "confirmado", "estado": "pendiente", "medio": null, "cobradoEn": null,
             "lineas": [{"tipo": "servicio", "concepto": "Corte y brushing", "monto": 18000}, {"tipo": "sena", "concepto": "Seña", "monto": -5000}],
             "total": 13000, "medios": ["efectivo", "transferencia"]}
            """,
            "/v1/panel/avisos": """
            {"avisos": [{"id": "a1", "tipo": "confirmacion", "para": "sofia@prueba.test", "turnoId": "t1", "cliente": "Sofía", "estado": "enviado", "intentos": 1,
              "ultimoError": null, "proximoIntento": "\(Self.iso(-60))", "enviadoEn": "\(Self.iso(-60))", "creadoEn": "\(Self.iso(-60))"}]}
            """,
            "/v1/panel/profesionales": "{\"profesionales\": [\(Self.profesional)]}",
            "/v1/panel/servicios": "{\"servicios\": [{\"id\": \"s1\", \"nombre\": \"Corte y brushing\", \"duracion\": 45, \"precio\": 18000, \"activo\": true, \"categoriaId\": null}]}",
            "/v1/panel/espera": """
            {"espera": [{"id": "w1", "servicioId": "s1", "servicio": "Corte y brushing", "profesionalId": null, "profesional": null, "nombre": "Ana Torres",
              "email": "", "telefono": "1166660000", "desde": "2026-01-01", "hasta": "2099-12-31", "avisadoEn": null, "creadoEn": "\(Self.iso(-3000))", "momento": "tarde"}]}
            """,
            "/v1/panel/espera/ofertas": "{\"ofertas\": []}",
            "/v1/panel/para-llamar": "{\"turnos\": []}",
            "/v1/panel/reprogramar": "{\"turnos\": []}",
            "/v1/panel/arranque": "{\"completo\": false, \"reservable\": true, \"pasos\": [{\"clave\": \"servicios\", \"titulo\": \"Cargá tus servicios\", \"listo\": true, \"obligatorio\": true}, {\"clave\": \"mercadopago\", \"titulo\": \"Conectá Mercado Pago\", \"listo\": false, \"obligatorio\": false}]}",
            "/v1/panel/resenas": """
            {"promedio": 4.8, "cantidad": 12, "distribucion": {"5": 10, "4": 2}, "resenas": [{"id": "r1", "turnoId": "t0", "nombre": "Lucía", "servicio": "Corte",
              "profesional": "Carla", "puntaje": 5, "comentario": "¡Divino todo!", "respuesta": "", "respondidaEn": null, "visible": true, "creadoEn": "\(Self.iso(-2000))"}]}
            """,
            "/v1/panel/dias-libres": "{\"pedidos\": []}",
            "/v1/panel/mercadopago": "{\"disponible\": true, \"conectada\": false, \"reautorizar\": false}",
            "/v1/panel/cobros": "{\"modo\": \"sin_cobro\", \"senaPorcentaje\": 30, \"senaFija\": 0}",
            "/v1/panel/pagos": "{\"pagos\": []}",
            "/v1/panel/clientes": "{\"clientes\": [\(cliente)], \"total\": 1}",
            "/v1/panel/clientes/c1": """
            {"id": "c1", "nombre": "Sofía Méndez", "telefono": "1144712290", "email": "sofia@prueba.test", "creadoEn": "\(Self.iso(-90000))", "visitas": 6,
             "ultimaVisita": null, "proximoTurno": null, "gastado": 98000, "ausencias": 0, "penalizado": false, "etiquetas": [],
             "turnos": [{"id": "t1", "codigo": "ABC123", "servicio": "Corte", "profesional": "Carla", "inicio": "\(Self.iso(-20000))", "fin": "\(Self.iso(-19950))",
               "precio": 18000, "estado": "atendido", "origen": "web", "nota": ""}], "penalidad": null}
            """,
            "/v1/panel/clientes/c1/notas": "{\"notas\": [{\"id\": \"n1\", \"texto\": \"Prefiere la tarde\", \"miembroId\": null, \"autor\": \"Laura\", \"creadoEn\": \"\(Self.iso(-500))\"}]}",
            "/v1/panel/clientes/c1/penalidades": "{\"penalidades\": []}",
            "/v1/panel/etiquetas": "{\"etiquetas\": [{\"id\": \"e1\", \"nombre\": \"VIP\", \"color\": \"#3ccc90\", \"clientes\": 3}]}",
            "/v1/panel/membresias/socios": "{\"socios\": []}",
            "/v1/panel/membresias/planes": "{\"planes\": []}",
            "/v1/panel/campanita": """
            {"sinLeer": 1, "notificaciones": [{"id": "x1", "tipo": "reserva", "titulo": "Nueva reserva", "texto": "Sofía · mañana 10:30", "turnoId": "t1",
              "creadoEn": "\(Self.iso(-3))", "nueva": true}]}
            """,
            "/v1/panel/caja": """
            {"fecha": "2026-11-04", "porMedio": {"efectivo": 13000, "transferencia": 7000, "debito": 0, "credito": 0, "mercadopago": 0}, "caja": 20000, "senas": 4000,
             "total": 24000, "productos": 0, "cobros": 1, "debe": 0, "movimientos": [{"turnoId": "t1", "cliente": "Sofía", "servicio": "Corte", "medio": "efectivo",
             "total": 13000, "productos": 0, "cobradoEn": "\(Self.iso(-30))"}]}
            """,
            "/v1/panel/negocio": """
            {"id": "n1", "slug": "lumi", "nombre": "Lumi", "descripcion": "", "direccion": "Thames 2240", "ciudad": "CABA", "telefono": "", "whatsapp": "",
             "instagram": "", "logo": null, "zonaHoraria": "America/Argentina/Buenos_Aires"}
            """,
            "/v1/panel/mi-web": """
            {"rubro": "Estética", "provincia": "CABA", "acerca": "", "portada": null, "colorPrimario": "#183c44", "colorAcento": "#3ccc90", "email": "", "sitioWeb": "",
             "facebook": "", "tiktok": "", "latitud": null, "longitud": null, "estilo": {"plantilla": "carta", "portada": "color", "fondo": "crema", "titulos": "grandes",
             "forma": "redondos", "secciones": ["servicios", "resenas"]}}
            """,
            "/v1/panel/ajustes": "{\"intervalo\": 15, \"anticipacionMin\": 120, \"diasVista\": 60, \"margenMin\": 0, \"cancelarHastaHoras\": 24, \"politicaCancelacion\": \"\"}",
            "/v1/panel/suscripcion": "{\"bloqueado\": false, \"pruebaHasta\": \"\(Self.iso(20000))\", \"diasDePrueba\": 14, \"conSuscripcion\": false, \"plan\": \"\", \"nombrePlan\": \"\", \"catalogo\": [{\"clave\": \"base\", \"nombre\": \"Base\", \"precio\": 19000}], \"suscripcion\": null}",
            "/v1/p/lumi": """
            {"negocio": {"id": "n1", "slug": "lumi", "nombre": "Lumi", "descripcion": "", "direccion": "Thames 2240", "ciudad": "CABA", "telefono": "", "whatsapp": "",
             "instagram": "", "logo": null}, "servicios": [{"id": "s1", "categoriaId": null, "nombre": "Corte", "descripcion": "", "duracion": 45, "precio": 18000,
             "foto": null, "orden": 1, "activo": true}], "profesionales": [{"id": "p1", "nombre": "Carla", "especialidad": "", "foto": null, "servicios": []}],
             "diasVista": 30, "politicaCancelacion": "", "web": {"portada": null, "colorPrimario": "#183c44", "acerca": "", "rubro": "Estética"}, "resenas": {"promedio": 4.8, "cantidad": 3}}
            """,
            "/v1/explorar": "{\"negocios\": [{\"slug\": \"lumi\", \"nombre\": \"Lumi\", \"descripcion\": \"\", \"logo\": null, \"portada\": null, \"ciudad\": \"CABA\", \"provincia\": \"\", \"rubro\": \"Estética\", \"servicios\": [], \"desde\": 18000}]}"
        ]
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
                                   .integraciones, .configuracion, .avisosClientes, .whatsapp, .auditoria, .sucursales, .cuenta]
        for d in todas {
            montar(NavigationLinkDirecto(destino: d), segundos: 1, nombre: "Seccion-\(d)")
        }
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
        case .primerosPasos: PrimerosPasos()
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
