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

    /// `turnos/{id}` (el `ruta` del push) o `tengolibre://turno/{id}`.
    init?(texto: String) {
        var t = texto
        if let u = URL(string: texto), u.scheme == "tengolibre" {
            t = ([u.host ?? ""] + u.pathComponents.filter { $0 != "/" }).joined(separator: "/")
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
        /// Elegir cómo la usa: atiende un negocio o reserva turnos (39).
        case eligiendoModo
        case eligiendoNegocio
        case permisos
        /// Sin negocios: a crear uno (alta).
        case alta
        case adentro
        case cliente
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
    /// Cómo la usa: «negocio» o «cliente» (se cambia desde Cuenta / Más).
    var modo: String {
        get { UserDefaults.standard.string(forKey: "modo") ?? "" }
        set { UserDefaults.standard.set(newValue, forKey: "modo") }
    }
    private var negocioGuardado: String? {
        get { UserDefaults.standard.string(forKey: "negocio") }
        set { UserDefaults.standard.set(newValue, forKey: "negocio") }
    }

    init(api: Api? = nil) {
        self.api = api ?? Api(base: Api.urlConfigurada(), token: Llavero.leer("token"))
        self.api.negocio = UserDefaults.standard.string(forKey: "negocio")
        self.api.alPerderSesion = { [weak self] in self?.sesionPerdida() }
    }

    // MARK: Arranque

    func arrancar() async {
        guard api.token != nil else { fase = .afuera; return }
        do {
            try await cargarSesion()
            await despuesDeEntrar()
        } catch let e as ErrorApi where e.sinConexion {
            sinConexion = true
            persona = Guardado.leer(Persona.self, clave: "persona")?.0
            negocios = Guardado.leer([NegocioDeLaPersona].self, clave: "negocios")?.0 ?? []
            negocio = negocios.first { $0.id == negocioGuardado } ?? negocios.first
            fase = persona == nil ? .afuera : (modo == "cliente" ? .cliente : .adentro)
        } catch {
            olvidarSesion()
            fase = .afuera
        }
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
        if modo.isEmpty { fase = .eligiendoModo; return }
        if modo == "cliente" { fase = .cliente; return }
        if negocios.isEmpty { fase = .alta; return }
        if negocios.count > 1 && negocioGuardado == nil { fase = .eligiendoNegocio; return }
        if !permisosVistos { fase = .permisos; return }
        fase = .adentro
        await refrescar()
        await pedirTokenPushSiCorresponde()
    }

    func elegirModo(_ m: String) async {
        modo = m
        await despuesDeEntrar()
    }

    /// Trabajar con otro negocio (o sucursal): todo lo guardado era del anterior.
    func usar(_ n: NegocioDeLaPersona) {
        if negocio?.id != n.id { Guardado.borrarTodo() }
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

    // MARK: Ingreso

    /// El token de Clerk (Apple o Google) → sesión propia de la API (POST /v1/auth/clerk).
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
        try? await api.hacer("POST", "/v1/auth/salir")
        olvidarSesion()
        fase = .afuera
    }

    private func olvidarSesion() {
        api.token = nil
        Llavero.borrar("token")
        Guardado.borrarTodo()
        persona = nil
        negocios = []
        negocio = nil
        acceso = nil
    }

    private func sesionPerdida() {
        olvidarSesion()
        fase = .afuera
        avisar(Toast(simbolo: "lock", titulo: "Tu sesión venció", texto: "Entrá de nuevo para seguir."))
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

    func pedirPermisoPush() async -> Bool {
        let ok = (try? await UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .badge, .sound])) ?? false
        if ok { UIApplication.shared.registerForRemoteNotifications() }
        return ok
    }

    func registrarTokenPush(_ datos: Data) async {
        let hex = datos.map { String(format: "%02x", $0) }.joined()
        guard hex != tokenPush, api.token != nil else { return }
        tokenPush = hex
        struct Cuerpo: Encodable { var plataforma = "ios"; var token: String; var entorno: String; var modelo: String; var version: String }
        #if DEBUG
        let entorno = "sandbox"
        #else
        let entorno = "produccion"
        #endif
        let v = Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String ?? ""
        try? await api.hacer("POST", "/v1/yo/dispositivos", Cuerpo(token: hex, entorno: entorno, modelo: UIDevice.current.model, version: v))
    }
}
