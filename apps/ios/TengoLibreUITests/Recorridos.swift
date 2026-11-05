import XCTest

/// Recorridos de punta a punta (como Playwright): la app en modo prueba (API simulada, Debug) en el
/// simulador de GitHub Actions. Tocan, escriben y deslizan como una persona y dejan una captura por
/// paso en $CAPTURAS (el CI las sube en el artifact «recorridos») y como adjunto del test.
final class Recorridos: XCTestCase {
    private var app: XCUIApplication!
    private var paso = 0

    override func setUp() {
        continueAfterFailure = false
    }

    private func abrir(_ extra: [String] = [], oscuro: Bool = false) {
        app = XCUIApplication()
        app.launchArguments = ["-modoPrueba"] + extra + (oscuro ? ["-AppleInterfaceStyle", "Dark"] : [])
        app.launch()
        paso = 0
    }

    /// Una captura con nombre: «<test>-<n>-<qué>.png».
    private func foto(_ que: String) {
        paso += 1
        let s = XCUIScreen.main.screenshot()
        let nombre = "\(name.replacingOccurrences(of: "-[Recorridos ", with: "").replacingOccurrences(of: "]", with: ""))-\(String(format: "%02d", paso))-\(que)"
        let a = XCTAttachment(screenshot: s)
        a.name = nombre
        a.lifetime = .keepAlways
        add(a)
        if let d = ProcessInfo.processInfo.environment["CAPTURAS"], !d.isEmpty {
            try? FileManager.default.createDirectory(atPath: d, withIntermediateDirectories: true)
            try? s.pngRepresentation.write(to: URL(fileURLWithPath: d).appendingPathComponent("\(nombre).png"))
        }
    }

    private func esperar(_ e: XCUIElement, _ seg: TimeInterval = 10, _ file: StaticString = #filePath, _ line: UInt = #line) {
        XCTAssertTrue(e.waitForExistence(timeout: seg), "No apareció: \(e)", file: file, line: line)
    }

    private func texto(_ t: String) -> XCUIElement { app.staticTexts[t].firstMatch }
    private func boton(_ t: String) -> XCUIElement { app.buttons[t].firstMatch }
    private func contiene(_ t: String) -> XCUIElement {
        app.descendants(matching: .any).matching(NSPredicate(format: "label CONTAINS %@", t)).firstMatch
    }
    private func pestana(_ t: String) { app.buttons[t].firstMatch.tap() }

    /// Cierra un consejo de TipKit si quedó a la vista.
    private func cerrarConsejos() {
        let x = app.buttons["Close"]
        if x.exists { x.tap() }
    }

    // MARK: Recorridos

    func testPrimeraVez() {
        abrir(["-conRecorrido"])
        esperar(texto("Hoy, siempre al día"))
        foto("recorrido-1")
        for i in 2...5 {
            boton("Siguiente").tap()
            foto("recorrido-\(i)")
        }
        boton("Empezar").tap()
        esperar(contiene("Tu negocio"))
        foto("hoy")
    }

    func testHoy() {
        abrir()
        esperar(contiene("COBRADO HOY"))
        foto("hoy")
        esperar(contiene("Sofía Méndez"))
        app.swipeUp()
        foto("hoy-abajo")
        app.swipeUp()
        foto("hoy-descubri")
    }

    func testHoyOscuro() {
        abrir(oscuro: true)
        esperar(contiene("COBRADO HOY"))
        foto("hoy-oscuro")
        app.swipeUp()
        foto("hoy-oscuro-abajo")
    }

    func testDeslizarAtendido() {
        abrir()
        let talon = contiene("Lucía Pérez")
        esperar(talon)
        talon.swipeLeft()
        foto("deslizado")
        let atendido = boton("Atendido")
        if atendido.waitForExistence(timeout: 3) { atendido.tap() }
        foto("marcado")
    }

    func testDetalleYCobrar() {
        abrir()
        let t = contiene("Sofía Méndez")
        esperar(t)
        t.tap()
        esperar(contiene("CÓMO VIENE"))
        foto("detalle")
        let cobrar = contiene("cobrar")
        esperar(cobrar)
        cobrar.tap()
        esperar(contiene("CÓMO PAGA"))
        foto("cobrar")
        boton("Transferencia").tap()
        contiene("Cobrar $").tap()
        foto("cobrado")
    }

    func testNuevoTurno() {
        abrir()
        let fab = boton("Nuevo turno")
        esperar(fab)
        fab.tap()
        esperar(contiene("1 · CLIENTE"))
        foto("nuevo")
        let campo = app.textFields["Buscá por nombre o teléfono"]
        esperar(campo)
        campo.tap()
        campo.typeText("Sof")
        let sofia = contiene("Sofía Méndez")
        esperar(sofia)
        sofia.tap()
        foto("clienta")
        app.swipeUp()
        let hora = boton("16:00")
        if hora.waitForExistence(timeout: 5) { hora.tap() }
        foto("horario")
        let guardar = contiene("Guardar ·")
        esperar(guardar)
        guardar.tap()
        foto("guardado")
    }

    func testBuscar() {
        abrir()
        let b = contiene("¿qué querés hacer?")
        esperar(b)
        b.tap()
        let campo = app.searchFields.firstMatch
        esperar(campo)
        campo.typeText("caja")
        foto("buscar-caja")
        esperar(boton("Caja"))
        boton("Caja").tap()
        esperar(contiene("TOTAL DEL DÍA"))
        foto("caja")
    }

    func testAgendaYEspera() {
        abrir()
        esperar(boton("Agenda"))
        pestana("Agenda")
        esperar(contiene("Agenda"))
        foto("agenda-dia")
        boton("Semana").tap()
        foto("agenda-semana")
        pestana("Espera")
        esperar(contiene("Ana Torres"))
        foto("espera")
    }

    func testClientes() {
        abrir()
        esperar(boton("Clientes"))
        pestana("Clientes")
        let s = contiene("Sofía Méndez")
        esperar(s)
        foto("clientes")
        s.tap()
        esperar(contiene("NOTAS DEL EQUIPO"))
        foto("ficha")
        app.swipeUp()
        foto("ficha-abajo")
    }

    func testMasYPotencia() {
        abrir()
        esperar(boton("Más"))
        pestana("Más")
        esperar(contiene("Potenciá tu negocio"))
        foto("mas")
        app.swipeUp()
        foto("mas-abajo")
        app.swipeDown()
        contiene("Potenciá tu negocio").tap()
        esperar(contiene("PARA CRECER"))
        foto("potencia")
    }

    func testSecciones() {
        abrir()
        esperar(boton("Más"))
        pestana("Más")
        for s in ["Pagos y señas", "Membresías", "Reportes", "Integraciones e IA", "Mi web"] {
            let fila = contiene(s)
            if !fila.waitForExistence(timeout: 3) { app.swipeUp() }
            esperar(fila)
            fila.tap()
            sleep(2)
            foto(s.replacingOccurrences(of: " ", with: "-"))
            let volver = app.buttons.matching(NSPredicate(format: "label CONTAINS 'Más'")).firstMatch
            if volver.exists { volver.tap() } else { app.swipeRight() }
            sleep(1)
        }
    }

    func testAvisosYCompartir() {
        abrir()
        let avisos = boton("Avisos")
        esperar(avisos)
        avisos.tap()
        esperar(contiene("Nueva reserva"))
        foto("avisos")
        boton("Listo").tap()
        boton("Compartir tu link").tap()
        esperar(contiene("Copiar"))
        foto("compartir")
    }

    func testModoClienta() {
        abrir(["-modoCliente"])
        esperar(contiene("TU PRÓXIMO TURNO"))
        foto("turnos")
        pestana("Lugares")
        let lumi = contiene("Lumi Estética")
        esperar(lumi)
        foto("lugares")
        lumi.tap()
        esperar(contiene("1 · SERVICIO"))
        contiene("Manicura").tap()
        foto("reservar")
        let hora = boton("16:00")
        esperar(hora)
        hora.tap()
        foto("datos")
        let tel = app.textFields.matching(NSPredicate(format: "placeholderValue == '' OR value == ''")).element(boundBy: 1)
        if tel.exists { tel.tap(); tel.typeText("1144712290") }
        app.swipeUp()
        foto("listo-para-reservar")
    }
}
