import Foundation
import WidgetKit

/// En vivo: escucha el SSE de la API (`GET /v1/panel/eventos`, el mismo que usa la web) mientras la
/// app está abierta. Cada cambio (turnos, espera, reseñas, pagos, la campanita…) sube
/// `estado.version` y las pantallas se vuelven a pedir solas; si se corta, reconecta.
@MainActor
final class EnVivo {
    struct Evento: Decodable { var negocio: String; var tipo: String; var id: String; var op: String }

    private var tarea: Task<Void, Never>?
    private var pendiente: Task<Void, Never>?
    private var negocio: String?
    private weak var estado: Estado?

    func arrancar(_ estado: Estado) {
        guard let n = estado.negocio?.id, estado.api.token != nil else { parar(); return }
        if tarea != nil && negocio == n { return }
        parar()
        self.estado = estado
        negocio = n
        tarea = Task { [weak self] in
            var espera: UInt64 = 1
            while !Task.isCancelled {
                guard let self, let estado = self.estado else { return }
                let (pedido, http) = estado.api.flujo("/v1/panel/eventos")
                do {
                    let (bytes, resp) = try await http.bytes(for: pedido)
                    guard (resp as? HTTPURLResponse)?.statusCode == 200 else { throw URLError(.badServerResponse) }
                    espera = 1
                    estado.sinConexion = false
                    var tipo = ""
                    for try await linea in bytes.lines {
                        if linea.hasPrefix("event:") { tipo = linea.dropFirst(6).trimmingCharacters(in: .whitespaces) }
                        else if linea.hasPrefix("data:") {
                            let d = Data(linea.dropFirst(5).trimmingCharacters(in: .whitespaces).utf8)
                            let e = try? JSONDecoder().decode(Evento.self, from: d)
                            self.llego(tipo: e?.tipo ?? tipo, op: e?.op ?? "")
                        }
                    }
                } catch {
                    if Task.isCancelled { return }
                }
                // Se cortó: vuelve a los 1, 2, 4… hasta 30 s.
                try? await Task.sleep(nanoseconds: espera * 1_000_000_000)
                espera = min(30, espera * 2)
            }
        }
    }

    func parar() {
        tarea?.cancel()
        tarea = nil
        negocio = nil
    }

    /// Junta los avisos que llegan seguidos (un cobro toca turno, caja y pago a la vez).
    private func llego(tipo: String, op: String) {
        guard let estado else { return }
        pendiente?.cancel()
        let campanita = tipo == "notificaciones"
        pendiente = Task {
            try? await Task.sleep(nanoseconds: 350_000_000)
            guard !Task.isCancelled else { return }
            estado.version += 1
            if campanita { await estado.refrescar() }
            WidgetCenter.shared.reloadAllTimelines()
        }
    }
}
