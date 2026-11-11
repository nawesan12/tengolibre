import SafariServices
import SwiftUI

/// Los links de tengolibre.com.ar que abren la app (Universal Links). Qué paths los declara
/// `apps/pruebas/enlaces.json` (de ahí sale el AASA de la web); acá, a qué pantalla lleva cada uno. Lo que la
/// app no tiene se abre en la web dentro de la app, nunca queda en nada.
enum Vinculos {
    static func ruta(_ u: URL) -> Ruta? {
        guard let host = u.host(), host == "tengolibre.com.ar" || host == "www.tengolibre.com.ar" else { return nil }
        let p = u.pathComponents.filter { $0 != "/" }
        if URLComponents(url: u, resolvingAgainstBaseURL: false)?.queryItems?.contains(where: { $0.name == "web" }) == true {
            return .web(u)
        }
        switch (p.first, p.count) {
        case ("turno", 2...): return .turnoCliente(p[1])
        case ("invitacion", 2...): return .invitacion(p[1])
        case ("panel", _): return panel(Array(p.dropFirst()), u)
        default: return .web(u)
        }
    }

    /// `/panel/<sección>/<id>` → la pantalla de la app que hace lo mismo.
    private static func panel(_ p: [String], _ u: URL) -> Ruta {
        let id = p.count > 1 ? p[1] : nil
        switch p.first {
        case nil: return .hoy
        case "turnos":
            guard let id else { return .hoy }
            return id == "nuevo" ? .nuevoTurno : .turno(id)
        case "calendario": return id == nil ? .agenda : .web(u)
        case "lista-espera": return .espera
        case "clientes": return .clientes
        case "notificaciones": return id == nil ? .avisos : .web(u)
        case "buscar": return id == nil ? .buscar : .web(u)
        case "resenas": return .resenas
        case "suscripcion": return .suscripcion
        case "servicios": return .mas(.servicios)
        case "equipo": return .mas(.equipo)
        case "profesionales": return id.map { .mas(.profesional($0)) } ?? .mas(.equipo)
        case "pagos": return .mas(.cobros)
        case "membresias": return .mas(.membresias)
        case "cupones": return .mas(.cupones)
        case "referidos": return .mas(.referidos)
        case "reportes": return id == nil || id == "mensual" ? .mas(.reportes) : .web(u)
        case "mi-web": return .mas(.miWeb)
        case "plantillas": return .mas(.plantillas)
        case "dominio": return .mas(.dominio)
        case "integraciones": return .mas(.integraciones)
        case "configuracion": return .mas(.configuracion)
        case "etiquetas": return .mas(.etiquetas)
        case "penalidades": return .mas(.penalidades)
        case "auditoria": return id == nil ? .mas(.auditoria) : .web(u)
        case "sucursales": return .mas(.sucursales)
        default: return .web(u)
        }
    }
}

extension Ruta {
    /// Cómo se escribe la ruta en `apps/pruebas/enlaces.json` («turno:42», «mas:servicios», «web-en-app»).
    var clave: String {
        switch self {
        case .turno(let id): "turno:\(id)"
        case .hoy: "hoy"
        case .espera: "espera"
        case .avisos: "avisos"
        case .resenas: "resenas"
        case .nuevoTurno: "nuevoTurno"
        case .suscripcion: "suscripcion"
        case .caja: "caja"
        case .buscar: "buscar"
        case .agenda: "agenda"
        case .turnoCliente(let c): "turnoCliente:\(c)"
        case .invitacion(let t): "invitacion:\(t)"
        case .clientes: "clientes"
        case .web: "web-en-app"
        case .mas(let d):
            switch d {
            case .profesional(let id): "mas:profesional:\(id)"
            default: "mas:\(String(describing: d))"
            }
        }
    }
}

/// La web de tengolibre.com.ar dentro de la app (un link que la app no tiene, o «Abrir en la web»).
struct NavegadorWeb: UIViewControllerRepresentable {
    var url: URL
    func makeUIViewController(context: Context) -> SFSafariViewController { SFSafariViewController(url: url) }
    func updateUIViewController(_ c: SFSafariViewController, context: Context) {}
}
