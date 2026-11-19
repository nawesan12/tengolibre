import Foundation
import Observation
import SwiftUI
import WidgetKit
import UIKit
import UserNotifications

/// Adónde lleva un push, un widget o un link `tengolibre://`.
enum Ruta: Hashable {
    case turno(String)
    case hoy
    case espera
    case avisos
    case resenas
    case nuevoTurno
    case suscripcion
    case caja
    case buscar
    case agenda
    /// Una invitación al equipo de un negocio.
    case invitacion(String)
    case clientes
    /// Una pantalla de Más (servicios, equipo, cobros, mi web…).
    case mas(DestinoMas)
    /// Un link de tengolibre.com.ar que la app no tiene: se abre la web dentro de la app.
    case web(URL)

    /// `turnos/{id}` (el `ruta` del push), `tengolibre://turno/{id}` o un link de tengolibre.com.ar (Vinculos).
    init?(texto: String) {
        var t = texto
        if let u = URL(string: texto), u.scheme == "tengolibre" {
            t = ([u.host ?? ""] + u.pathComponents.filter { $0 != "/" }).joined(separator: "/")
        } else if let u = URL(string: texto), u.scheme == "https" || u.scheme == "http" {
            guard let r = Vinculos.ruta(u) else { return nil }
            self = r
            return
        }
        let partes = t.split(separator: "/").map(String.init)
        guard let primera = partes.first else { return nil }
        switch primera {
        case "turnos", "turno": guard partes.count > 1 else { self = .hoy; return }; self = .turno(partes[1])
        case "hoy": self = .hoy
        case "espera", "lista-espera": self = .espera
        case "notificaciones", "avisos": self = .avisos
        case "resenas": self = .resenas
        case "nuevo": self = .nuevoTurno
        case "suscripcion": self = .suscripcion
        case "caja": self = .caja
        case "buscar": self = .buscar
        case "agenda": self = .agenda
        case "invitacion": guard partes.count > 1 else { return nil }; self = .invitacion(partes[1])
        default: return nil
        }
    }
}

/// El estado de la app: la sesión (token en el Keychain), la persona, sus negocios y el elegido con
/// su rol, el acceso (prueba / sólo lectura), la campanita y los avisos de arriba (toasts).
@Observable
@MainActor
final class Estado {
    enum Fase: Equatable {
        case arrancando
        case afuera
        case eligiendoNegocio
        case permisos
        /// Sin negocios: a crear uno (alta).
        case alta
        case adentro
    }

    var fase: Fase = .arrancando
    let api: Api
    var persona: Persona?
    var negocios: [NegocioDeLaPersona] = []
    var negocio: NegocioDeLaPersona?
    var acceso: Acceso?
    var sinLeer = 0
    var toast: Toast?
    var rutaPendiente: Ruta?
    /// Una invitación al equipo para aceptar: la abre Raiz, en cualquier fase (espera al ingreso si hace falta).
    var invitacionPendiente: String?
    /// Un link de tengolibre.com.ar que la app no tiene: lo muestra Raiz en el navegador de adentro.
    var webAbierta: URL?
    /// Para que las listas se vuelvan a pedir después de un cambio.
    var version = 0
    /// Sin conexión: se ve lo guardado (handoff 23).
    var sinConexion = false
    var guardadoEn: Date?
    /// El SSE del panel (en vivo).
    let enVivo = EnVivo()
    /// Qué tiene conectado y prendido el negocio (Más, Hoy, «Potenciá tu negocio»).
    let vivo = EstadoNegocio()
    private(set) var tokenPush: String?
    private var toastTarea: Task<Void, Never>?
    private var deshacer: (() async -> Void)?
    /// Cuándo se volvieron a pedir el rol y el acceso por un 402/403 (una vez cada 30 s, sin bucles).
    private var accesoRevisadoEn: Date?

    var rol: String { negocio?.rol ?? "" }
    var esDuena: Bool { rol == "duenio" }
    var gestiona: Bool { rol == "duenio" || rol == "encargado" }
    var esProfesional: Bool { rol == "profesional" }
    /// Recepción no ve precios ni servicios en Más.
    var vePrecios: Bool { rol != "recepcion" }
    var soloLectura: Bool { acceso?.bloqueado ?? false }

    var permisosVistos: Bool {
        get { UserDefaults.standard.bool(forKey: "permisosVistos") }
        set { UserDefaults.standard.set(newValue, forKey: "permisosVistos") }
    }
    private var negocioGuardado: String? {
        get { UserDefaults.standard.string(forKey: "negocio") }
        set { UserDefaults.standard.set(newValue, forKey: "negocio") }
    }

    init(api: Api? = nil) {
        #if DEBUG
        ModoPrueba.preparar()
        let api = api ?? (ModoPrueba.activo ? ModoPrueba.api() : nil)
        #endif
        self.api = api ?? Api(base: Api.urlConfigurada(), token: Llavero.leer("token"))
        Self.olvidarModoCliente()
        self.api.negocio = UserDefaults.standard.string(forKey: "negocio")
        self.api.alPerderSesion = { [weak self] in self?.sesionPerdida() }
        self.api.alNegarAcceso = { [weak self] in
            guard let self else { return }
            Task { await self.accesoNegado() }
        }
    }

    // MARK: Arranque

    func arrancar() async {
        // Turnos que este iPhone seguía como clienta (versiones viejas): se dejan de seguir aunque la
        // persona no llegue a Principal (el token se pide en cualquier fase, si ya hay permiso).
        if UserDefaults.standard.dictionaryRepresentation().keys.contains(where: { $0.hasPrefix("seguidos-") }) {
            await pedirTokenPushSiCorresponde()
        }
        guard api.token != nil else { fase = .afuera; return }
        do {
            try await cargarSesion()
            await despuesDeEntrar()
        } catch let e as ErrorApi where e.estado == 401 || e.codigo == "cuenta_borrada" {
            // La sesión se cierra sólo si la API dice que ya no vale (401) o que la cuenta está borrada.
            olvidarSesion()
            fase = .afuera
            if e.codigo == "cuenta_borrada" { avisar(e) }
        } catch is CancellationError {
            return
        } catch {
            // Cualquier otro error (sin red, un 500, un 502 durante un deploy, una respuesta que no se
            // pudo leer): se entra con lo guardado y se avisa, sin perder la sesión.
            entrarConLoGuardado()
            if !((error as? ErrorApi)?.sinConexion ?? false) {
                avisar(Toast(simbolo: "exclamationmark", titulo: "No pudimos actualizar", texto: "Ves lo último guardado. Probá de nuevo en un rato.", error: true))
            }
        }
    }

    /// Sin poder hablar con la API al abrir: lo último que se guardó, con el aviso de «sin conexión».
    /// Si no hay nada guardado queda en el ingreso, pero la sesión sigue (se reintenta al volver a abrir).
    private func entrarConLoGuardado() {
        sinConexion = true
        persona = Guardado.leer(Persona.self, clave: "persona")?.0
        negocios = Guardado.leer([NegocioDeLaPersona].self, clave: "negocios")?.0 ?? []
        negocio = negocios.first { $0.id == negocioGuardado } ?? negocios.first
        fase = persona == nil ? .afuera : (negocios.isEmpty ? .alta : .adentro)
    }

    func cargarSesion() async throws {
        let p: Persona = try await api.get("/v1/auth/persona")
        let y: Yo = try await api.get("/v1/yo")
        let (pp, yy) = (p, y)
        persona = pp
        negocios = yy.negocios
        Guardado.guardar(pp, clave: "persona")
        Guardado.guardar(yy.negocios, clave: "negocios")
        if let n = negocios.first(where: { $0.id == negocioGuardado }) ?? negocios.first { usar(n) }
        sinConexion = false
    }

    /// Después de entrar o de arrancar con sesión: adónde va.
    private func despuesDeEntrar() async {
        if negocios.isEmpty { fase = .alta; return }
        if negocios.count > 1 && negocioGuardado == nil { fase = .eligiendoNegocio; return }
        if !permisosVistos { fase = .permisos; return }
        fase = .adentro
        await refrescar()
        await pedirTokenPushSiCorresponde()
    }

    /// La app es sólo para quien atiende un negocio: las versiones viejas tenían un modo clienta («modo»,
    /// «misTurnos»). Se borra al abrir, y quien lo tenía va a su negocio o al alta.
    private static func olvidarModoCliente() {
        for clave in ["modo", "misTurnos", "avisosClientaPedidos"] { UserDefaults.standard.removeObject(forKey: clave) }
    }

    /// Trabajar con otro negocio (o sucursal): lo guardado del anterior se borra (la persona y sus
    /// negocios, recién guardados al entrar, siguen). Al abrir, el anterior es el que quedó elegido.
    func usar(_ n: NegocioDeLaPersona) {
        if let anterior = negocio?.id ?? negocioGuardado, anterior != n.id { Guardado.borrarDelNegocio() }
        negocio = n
        negocioGuardado = n.id
        api.negocio = n.id
        // Para los widgets y la Live Activity (leen el llavero compartido).
        Llavero.guardar(n.id, cuenta: "negocio")
        Llavero.guardar(n.nombre, cuenta: "negocioNombre")
        WidgetCenter.shared.reloadAllTimelines()
        version += 1
    }

    func entrarANegocio(_ n: NegocioDeLaPersona) async {
        usar(n)
        await despuesDeEntrar()
    }

    func terminarPermisos() async {
        permisosVistos = true
        await despuesDeEntrar()
    }

    /// El acceso (prueba, sólo lectura) y la campanita.
    func refrescar() async {
        guard negocio != nil else { return }
        let a: Acceso? = try? await api.get("/v1/panel/acceso")
        let c: SinLeer? = try? await api.get("/v1/panel/campanita/sin-leer")
        let (aa, cc) = (a, c)
        if let aa { acceso = aa }
        if let cc {
            sinLeer = cc.sinLeer
            try? await UNUserNotificationCenter.current().setBadgeCount(cc.sinLeer)
        }
    }
    struct SinLeer: Decodable { var sinLeer: Int }

    /// Un pedido del panel volvió con 402 o 403: el rol, la membresía o el sólo lectura cambiaron. Se
    /// vuelven a pedir /v1/yo y el acceso, como mucho una vez cada 30 s (los propios pedidos de acá
    /// también pueden dar 403 y no tienen que volver a llamar).
    func accesoNegado() async {
        if let r = accesoRevisadoEn, Date().timeIntervalSince(r) < 30 { return }
        accesoRevisadoEn = Date()
        guard api.token != nil else { return }
        let antes = (negocio, acceso?.bloqueado)
        if let y: Yo = try? await api.get("/v1/yo") {
            negocios = y.negocios
            Guardado.guardar(y.negocios, clave: "negocios")
            if let actual = negocio?.id {
                if let n = negocios.first(where: { $0.id == actual }) {
                    negocio = n
                } else if let n = negocios.first {
                    // Ya no es parte de ese negocio: se trabaja con otro de los suyos.
                    usar(n)
                } else {
                    negocio = nil
                    fase = .alta
                    return
                }
            }
        }
        await refrescar()
        // Las pantallas se vuelven a pedir sólo si algo cambió (un 403 de una pantalla que ese rol no ve no las recarga).
        if antes.0 != negocio || antes.1 != acceso?.bloqueado { version += 1 }
    }

    // MARK: Ingreso

    /// El token de Clerk (Apple, Google o mail) → sesión propia de la API (POST /v1/auth/clerk). Si la
    /// cuenta está borrada, la API responde 409 `cuenta_borrada` con el detalle que se muestra.
    func ingresarCon(tokenClerk: String) async throws {
        struct Cuerpo: Encodable { var token: String }
        struct Respuesta: Decodable { var token: String; var persona: Persona }
        let r: Respuesta = try await api.enviar("POST", "/v1/auth/clerk", Cuerpo(token: tokenClerk))
        api.token = r.token
        Llavero.guardar(r.token, cuenta: "token")
        try await cargarSesion()
        await despuesDeEntrar()
    }

    func ingresarConApple() async throws {
        guard let clerk = Clerk(publicable: Bundle.main.object(forInfoDictionaryKey: "TLClerk") as? String) else {
            throw Clerk.ErrorClerk(codigo: "sin_clave")
        }
        let tokenApple = try await HojaApple().pedirToken()
        try await ingresarCon(tokenClerk: try await clerk.conApple(tokenApple: tokenApple))
    }

    func ingresarConGoogle() async throws {
        guard let clerk = Clerk(publicable: Bundle.main.object(forInfoDictionaryKey: "TLClerk") as? String) else {
            throw Clerk.ErrorClerk(codigo: "sin_clave")
        }
        try await ingresarCon(tokenClerk: try await clerk.conGoogle())
    }

    func salir() async {
        await dejarDeSeguirTodos()
        try? await api.hacer("POST", "/v1/auth/salir")
        olvidarSesion()
        fase = .afuera
    }

    /// Borra la cuenta (DELETE /v1/auth/cuenta, 204): se desactiva ahora y los datos personales se
    /// borran a los 30 días. Después se olvida la sesión y se vuelve al ingreso.
    func borrarCuenta() async throws {
        try await api.hacer("DELETE", "/v1/auth/cuenta")
        await dejarDeSeguirTodos()
        olvidarSesion()
        fase = .afuera
        avisar(Toast(simbolo: "checkmark", titulo: "Tu cuenta quedó desactivada", texto: "Tus datos personales se borran a los 30 días."))
    }

    /// Todo lo de la sesión: también el token de avisos (si no, al volver a entrar no se registra de
    /// nuevo), el negocio elegido y los permisos vistos.
    private func olvidarSesion() {
        api.token = nil
        api.negocio = nil
        Llavero.borrar("token")
        Guardado.borrarTodo()
        persona = nil
        negocios = []
        negocio = nil
        acceso = nil
        tokenPush = nil
        negocioGuardado = nil
        permisosVistos = false
        accesoRevisadoEn = nil
    }

    private func sesionPerdida() {
        olvidarSesion()
        fase = .afuera
        avisar(Toast(simbolo: "lock", titulo: "Tu sesión venció", texto: "Entrá de nuevo para seguir."))
    }

    /// Abre una ruta: las del panel quedan pendientes para Principal; las invitaciones y la web las abre Raiz.
    func abrir(_ r: Ruta) {
        switch r {
        case .invitacion(let t): invitacionPendiente = t
        case .web(let u): webAbierta = u
        default: rutaPendiente = r
        }
    }

    // MARK: Avisos de arriba

    func avisar(_ t: Toast, deshacer: (() async -> Void)? = nil) {
        toastTarea?.cancel()
        self.deshacer = deshacer
        withAnimation(Movimiento.entrar) { toast = t }
        if t.error { Haptico.error() }
        toastTarea = Task {
            try? await Task.sleep(for: .seconds(deshacer == nil ? 3.5 : 5))
            guard !Task.isCancelled else { return }
            withAnimation(Movimiento.entrar) { toast = nil }
        }
    }

    func avisar(_ titulo: String, texto: String = "", ok: Bool = true) {
        avisar(Toast(simbolo: ok ? "checkmark" : "exclamationmark", titulo: titulo, texto: texto, error: !ok))
    }

    func avisar(_ e: Error) {
        // Salir de una pantalla mientras guarda cancela la tarea: no es un error para mostrar (igual que Android).
        if e is CancellationError { return }
        let x = e as? ErrorApi ?? .desconocido
        avisar(Toast(simbolo: x.sinConexion ? "icloud.slash" : "exclamationmark", titulo: x.sinConexion ? "Sin conexión" : x.titulo,
                     texto: x.detalle, error: true))
    }

    func tocarToast() {
        let d = deshacer
        deshacer = nil
        withAnimation(Movimiento.entrar) { toast = nil }
        if let d { Task { await d(); version += 1 } }
    }

    // MARK: Push

    func pedirTokenPushSiCorresponde() async {
        let ajustes = await UNUserNotificationCenter.current().notificationSettings()
        if ajustes.authorizationStatus == .authorized || ajustes.authorizationStatus == .provisional {
            UIApplication.shared.registerForRemoteNotifications()
        }
    }

    static var entornoPush: String {
        #if DEBUG
        "sandbox"
        #else
        "produccion"
        #endif
    }

    /// Los turnos que este iPhone seguía como clienta (versiones viejas, con modo clienta): deja de recibir
    /// sus avisos. Se hace al registrar el token, al salir y al borrar la cuenta; si falla la red, igual se
    /// olvida la lista.
    private func dejarDeSeguirTodos() async {
        guard let token = tokenPush else { return }
        let clave = "seguidos-\(token.prefix(16))"
        for c in UserDefaults.standard.stringArray(forKey: clave) ?? [] {
            try? await api.hacer("DELETE", "/v1/t/\(c)/dispositivo", ["token": token])
        }
        UserDefaults.standard.removeObject(forKey: clave)
    }

    func pedirPermisoPush() async -> Bool {
        let ok = (try? await UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .badge, .sound])) ?? false
        if ok { UIApplication.shared.registerForRemoteNotifications() }
        return ok
    }

    func registrarTokenPush(_ datos: Data) async {
        let hex = datos.map { String(format: "%02x", $0) }.joined()
        guard hex != tokenPush else { return }
        tokenPush = hex
        // Sin sesión (en el alta o afuera) igual se dejan de seguir los turnos viejos: no pide sesión.
        guard api.token != nil else { await dejarDeSeguirTodos(); return }
        struct Cuerpo: Encodable { var plataforma = "ios"; var token: String; var entorno: String; var modelo: String; var version: String }
        let entorno = Self.entornoPush
        let v = Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String ?? ""
        try? await api.hacer("POST", "/v1/yo/dispositivos", Cuerpo(token: hex, entorno: entorno, modelo: UIDevice.current.model, version: v))
        await dejarDeSeguirTodos()
    }
}
