import SwiftUI
import UIKit
import UserNotifications

@main
struct TengoLibreApp: App {
    @UIApplicationDelegateAdaptor(Delegado.self) private var delegado

    init() { Consejos.configurar() }
    @State private var estado = Estado()
    @Environment(\.scenePhase) private var fase
    @AppStorage("tema") private var tema = "sistema"

    var body: some Scene {
        WindowGroup {
            Raiz()
                .environment(estado)
                .preferredColorScheme(tema == "claro" ? .light : (tema == "oscuro" ? .dark : nil))
                .tint(C.mentaTexto)
                .onAppear { delegado.estado = estado; AtajosSiri.estado = estado }
                // Renovaciones y compras aprobadas de la App Store, mientras la app está abierta.
                .task { Compras.escuchar(estado: estado) }
                .onOpenURL { url in
                    if let r = Ruta(texto: url.absoluteString) { estado.abrir(r) }
                }
                .onChange(of: fase) { _, nueva in
                    if nueva == .active, estado.fase == .adentro {
                        estado.enVivo.arrancar(estado)
                        Task { await estado.refrescar(); estado.version += 1 }
                    }
                    if nueva == .background { estado.enVivo.parar() }
                }
        }
    }
}

/// Qué se ve según la sesión: arranque, ingreso, cómo la usás, elegir negocio, permiso de avisos,
/// alta, el panel o el modo cliente.
struct Raiz: View {
    @Environment(Estado.self) private var estado

    var body: some View {
        ZStack(alignment: .top) {
            switch estado.fase {
            case .arrancando: PantallaArranque()
            case .afuera: Ingreso().transition(.opacity)
            case .eligiendoModo: ElegirModo().transition(.opacity)
            case .eligiendoNegocio: ElegirNegocio().transition(.opacity)
            case .permisos: PermisoAvisos().transition(.opacity)
            case .alta: Alta().transition(.opacity)
            case .adentro: Principal().transition(.opacity)
            case .cliente: PrincipalCliente().transition(.opacity)
            }
            if let t = estado.toast {
                VistaToast(t: t) { estado.tocarToast() }
                    .padding(.top, 6)
                    .transition(.move(edge: .top).combined(with: .opacity))
                    .zIndex(10)
                    .onTapGesture { estado.tocarToast() }
            }
        }
        .animation(.easeInOut(duration: 0.25), value: estado.fase)
        // La barra de estado, petróleo como la cabecera (la hora en blanco: UIStatusBarStyle): el
        // contenido que sube con el scroll pasa por debajo, nunca por detrás de la hora.
        .overlay(alignment: .top) {
            C.cabecera.frame(height: 0).ignoresSafeArea(edges: .top).allowsHitTesting(false)
        }
        // Un turno de clienta (link o aviso) y una invitación al equipo: en cualquier modo.
        .sheet(item: Binding(get: { estado.turnoClienteAbierto.map { IdTexto(id: $0) } }, set: { estado.turnoClienteAbierto = $0?.id })) { c in
            TurnoClienteCargar(codigo: c.id).environment(estado)
        }
        .sheet(item: Binding(get: { estado.persona == nil || estado.fase == .arrancando ? nil : estado.invitacionPendiente.map { IdTexto(id: $0) } },
                             set: { estado.invitacionPendiente = $0?.id })) { i in
            InvitacionVista(token: i.id).environment(estado)
        }
        // En vivo mientras se ve el panel, con el negocio elegido.
        .onChange(of: "\(estado.fase)-\(estado.negocio?.id ?? "")") { _, _ in
            if estado.fase == .adentro { estado.enVivo.arrancar(estado) } else { estado.enVivo.parar() }
        }
        .task { await estado.arrancar() }
    }
}

/// APNs y los avisos con acciones (handoff › Avisos push).
final class Delegado: NSObject, UIApplicationDelegate, UNUserNotificationCenterDelegate {
    weak var estado: Estado?

    func application(_ application: UIApplication, didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]? = nil) -> Bool {
        let c = UNUserNotificationCenter.current()
        c.delegate = self
        c.setNotificationCategories(Delegado.categorias)
        return true
    }

    /// Las acciones de cada aviso (las categorías las pone la API: push/envio.go, CategoriasAPNs).
    static let categorias: Set<UNNotificationCategory> = [
        UNNotificationCategory(identifier: "TL_RESERVA", actions: [
            UNNotificationAction(identifier: "CONFIRMAR", title: "Confirmar", options: [], icon: UNNotificationActionIcon(systemImageName: "checkmark.circle")),
            UNNotificationAction(identifier: "VER", title: "Ver turno", options: [.foreground], icon: UNNotificationActionIcon(systemImageName: "calendar"))
        ], intentIdentifiers: []),
        UNNotificationCategory(identifier: "TL_CANCELO", actions: [
            UNNotificationAction(identifier: "AVISAR", title: "Avisar a la lista", options: [], icon: UNNotificationActionIcon(systemImageName: "megaphone")),
            UNNotificationAction(identifier: "CARGAR", title: "Cargar un turno", options: [.foreground], icon: UNNotificationActionIcon(systemImageName: "plus"))
        ], intentIdentifiers: []),
        UNNotificationCategory(identifier: "TL_HUECO", actions: [
            UNNotificationAction(identifier: "AVISAR", title: "Avisar a la lista", options: [], icon: UNNotificationActionIcon(systemImageName: "megaphone"))
        ], intentIdentifiers: []),
        UNNotificationCategory(identifier: "TL_RESENA", actions: [
            UNTextInputNotificationAction(identifier: "RESPONDER", title: "Responder", options: [], icon: UNNotificationActionIcon(systemImageName: "arrowshape.turn.up.left"),
                                          textInputButtonTitle: "Mandar", textInputPlaceholder: "Gracias por…")
        ], intentIdentifiers: [])
    ]

    func application(_ application: UIApplication, didRegisterForRemoteNotificationsWithDeviceToken deviceToken: Data) {
        Task { @MainActor in await estado?.registrarTokenPush(deviceToken) }
    }

    // Con la app abierta: el aviso arriba y la campanita al día.
    func userNotificationCenter(_ center: UNUserNotificationCenter, willPresent notification: UNNotification) async -> UNNotificationPresentationOptions {
        await MainActor.run {
            estado?.version += 1
            Task { await estado?.refrescar() }
        }
        return [.banner, .sound, .list]
    }

    func userNotificationCenter(_ center: UNUserNotificationCenter, didReceive response: UNNotificationResponse) async {
        let info = response.notification.request.content.userInfo
        let ruta = info["ruta"] as? String ?? ""
        let negocio = info["negocio"] as? String
        let turnoId = ruta.hasPrefix("turnos/") ? String(ruta.dropFirst(7)) : nil
        guard let estado = await MainActor.run(body: { self.estado }) else { return }
        await MainActor.run {
            // El aviso es de otro negocio de la persona: se trabaja con ese.
            if let negocio, estado.negocio?.id != negocio, let n = estado.negocios.first(where: { $0.id == negocio }) { estado.usar(n) }
        }
        switch response.actionIdentifier {
        case "CONFIRMAR":
            if let turnoId { await AccionesTurno.cambiarEstado(estado, turnoId, "confirmado") }
        case "AVISAR":
            if let turnoId { await AccionesTurno.avisarALista(estado, turnoId: turnoId) }
        case "CARGAR":
            await MainActor.run { estado.rutaPendiente = .nuevoTurno }
        case "RESPONDER":
            if let r = response as? UNTextInputNotificationResponse, let turnoId {
                await AccionesTurno.responderResena(estado, turnoId: turnoId, texto: r.userText)
            }
        default:
            await MainActor.run {
                if info["tipo"] as? String == "resena" { estado.rutaPendiente = .resenas }
                else if let r = Ruta(texto: ruta) { estado.abrir(r) }
            }
        }
    }
}

/// Lo que se puede hacer con un turno sin abrir su pantalla (push, deslizar, menú contextual).
enum AccionesTurno {
    struct CambioEstado: Encodable { var estado: String }

    @MainActor
    static func cambiarEstado(_ estado: Estado, _ id: String, _ nuevo: String) async {
        do {
            try await estado.api.hacer("PATCH", "/v1/panel/turnos/\(id)", CambioEstado(estado: nuevo))
            estado.version += 1
        } catch {
            estado.avisar(error)
        }
    }

    /// «Avisar a la lista»: ofrece el hueco de un turno cancelado a quienes esperan.
    @MainActor
    static func avisarALista(_ estado: Estado, turnoId: String) async {
        do {
            let t: Turno = try await estado.api.get("/v1/panel/turnos/\(turnoId)")
            guard let s = t.servicioId, let p = t.profesionalId else { return }
            try await ofrecer(estado, servicioId: s, profesionalId: p, inicio: t.inicio)
        } catch {
            estado.avisar(error)
        }
    }

    struct Oferta: Encodable { var servicioId: String; var profesionalId: String; var inicio: Date }

    @MainActor
    static func ofrecer(_ estado: Estado, servicioId: String, profesionalId: String, inicio: Date) async throws {
        _ = try await estado.api.pedir("POST", "/v1/panel/espera/ofertas",
                                       cuerpo: Api.codificador.encode(Oferta(servicioId: servicioId, profesionalId: profesionalId, inicio: inicio)))
        Haptico.exito()
        estado.version += 1
        estado.avisar(Toast(simbolo: "megaphone", titulo: "Le avisamos a la lista", texto: "Se lo ofrecemos de a una persona por vez."))
    }

    struct Respuesta: Encodable { var respuesta: String }
    struct Resenas: Decodable { var resenas: [Resena] }

    @MainActor
    static func responderResena(_ estado: Estado, turnoId: String, texto: String) async {
        let t = texto.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !t.isEmpty else { return }
        do {
            let r: Resenas = try await estado.api.get("/v1/panel/resenas")
            guard let resena = r.resenas.first(where: { $0.turnoId == turnoId }) else { return }
            try await estado.api.hacer("PATCH", "/v1/panel/resenas/\(resena.id)", Respuesta(respuesta: t))
            estado.version += 1
        } catch {
            estado.avisar(error)
        }
    }
}
