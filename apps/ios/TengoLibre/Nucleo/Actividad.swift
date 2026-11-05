import ActivityKit
import Foundation
import WidgetKit

/// La Live Activity del turno en curso: se prende cuando empieza uno (y la app está abierta en Hoy),
/// se actualiza con quién sigue y se apaga cuando termina o se marca atendido.
@MainActor
enum ActividadTurno {
    static func sincronizar(_ turnos: [Turno], ahora: Date = Date()) {
        WidgetCenter.shared.reloadAllTimelines()
        guard ActivityAuthorizationInfo().areActivitiesEnabled else { return }
        let activos = turnos.filter { ["pendiente", "confirmado"].contains($0.estado) }.sorted { $0.inicio < $1.inicio }
        let enCurso = activos.first { $0.inicio <= ahora && $0.fin > ahora }
        for a in Activity<AtributosTurno>.activities where a.attributes.turnoId != enCurso?.id || (turnos.first { $0.id == a.attributes.turnoId }?.estado ?? "") == "atendido" {
            Task { await a.end(nil, dismissalPolicy: .immediate) }
        }
        guard let t = enCurso else { return }
        let sigue = activos.first { $0.inicio >= t.fin }.map { "\(Formato.horaDe($0.inicio)) · \($0.cliente.nombre.split(separator: " ").first ?? "")" } ?? ""
        let estado = AtributosTurno.ContentState(fin: t.fin, siguiente: sigue)
        if let a = Activity<AtributosTurno>.activities.first(where: { $0.attributes.turnoId == t.id }) {
            Task { await a.update(ActivityContent(state: estado, staleDate: t.fin)) }
        } else {
            let at = AtributosTurno(turnoId: t.id, cliente: t.cliente.nombre, servicio: t.servicio, profesional: t.profesional, inicio: t.inicio)
            _ = try? Activity.request(attributes: at, content: ActivityContent(state: estado, staleDate: t.fin), pushType: nil)
        }
    }
}
