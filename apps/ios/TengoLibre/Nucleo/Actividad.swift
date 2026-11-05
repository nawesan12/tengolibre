import ActivityKit
import Foundation
import WidgetKit

/// La Live Activity del turno en curso: se prende cuando empieza uno (con la app abierta en Hoy) y pide
/// un token de push que se le pasa a la API (`POST /v1/panel/turnos/{id}/actividad`): desde ahí el
/// servidor la actualiza (quién sigue, si lo movieron) y la termina (atendido, cancelado, ausente)
/// aunque la app esté cerrada. Con la app abierta también se actualiza desde acá.
@MainActor
enum ActividadTurno {
    private static var escuchando: Set<String> = []

    static func sincronizar(_ turnos: [Turno], estado: Estado, ahora: Date = Date()) {
        WidgetCenter.shared.reloadAllTimelines()
        guard ActivityAuthorizationInfo().areActivitiesEnabled else { return }
        let activos = turnos.filter { ["pendiente", "confirmado"].contains($0.estado) }.sorted { $0.inicio < $1.inicio }
        let enCurso = activos.first { $0.inicio <= ahora && $0.fin > ahora }
        for a in Activity<AtributosTurno>.activities where a.attributes.turnoId != enCurso?.id {
            Task { await a.end(nil, dismissalPolicy: .immediate) }
        }
        guard let t = enCurso else { return }
        let sigue = activos.first { $0.inicio >= t.fin }.map { "\(Formato.horaDe($0.inicio)) · \($0.cliente.nombre.split(separator: " ").first ?? "")" } ?? ""
        let contenido = ActivityContent(state: AtributosTurno.ContentState(fin: t.fin, siguiente: sigue), staleDate: t.fin)
        if let a = Activity<AtributosTurno>.activities.first(where: { $0.attributes.turnoId == t.id }) {
            Task { await a.update(contenido) }
            escuchar(a, estado: estado)
            return
        }
        let at = AtributosTurno(turnoId: t.id, cliente: t.cliente.nombre, servicio: t.servicio, profesional: t.profesional, inicio: t.inicio)
        // Con token de push (para que la API la mueva); si el sistema no lo da, sin push.
        if let a = try? Activity.request(attributes: at, content: contenido, pushType: .token) {
            escuchar(a, estado: estado)
        } else {
            _ = try? Activity.request(attributes: at, content: contenido, pushType: nil)
        }
    }

    /// Cada token nuevo de la actividad va a la API (cambia cuando el sistema lo renueva).
    private static func escuchar(_ a: Activity<AtributosTurno>, estado: Estado) {
        guard !escuchando.contains(a.id) else { return }
        escuchando.insert(a.id)
        let turno = a.attributes.turnoId
        Task {
            for await token in a.pushTokenUpdates {
                let hex = token.map { String(format: "%02x", $0) }.joined()
                try? await estado.api.hacer("POST", "/v1/panel/turnos/\(turno)/actividad", ["token": hex, "entorno": Estado.entornoPush])
            }
            escuchando.remove(a.id)
        }
    }
}
