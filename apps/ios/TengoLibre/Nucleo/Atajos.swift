import AppIntents
import Foundation

/// Siri, Spotlight y la app Atajos: «Qué tengo hoy en Tengo Libre», «Cargar un turno en Tengo
/// Libre»… Aparecen solos (AppShortcutsProvider). Los que abren la app dejan la ruta pendiente.
@MainActor
enum AtajosSiri {
    static weak var estado: Estado?
    static func ir(_ r: Ruta) { estado?.rutaPendiente = r }
}

struct AbrirHoy: AppIntent {
    static var title: LocalizedStringResource = "Ver mi día"
    static var description = IntentDescription("Abre Hoy en Tengo Libre.")
    static var openAppWhenRun = true
    @MainActor func perform() async throws -> some IntentResult { AtajosSiri.ir(.hoy); return .result() }
}

struct CargarTurno: AppIntent {
    static var title: LocalizedStringResource = "Cargar un turno"
    static var description = IntentDescription("Abre «Nuevo turno» en Tengo Libre.")
    static var openAppWhenRun = true
    @MainActor func perform() async throws -> some IntentResult { AtajosSiri.ir(.nuevoTurno); return .result() }
}

struct VerCaja: AppIntent {
    static var title: LocalizedStringResource = "Ver la caja de hoy"
    static var description = IntentDescription("Cuánto se cobró hoy, por medio de pago.")
    static var openAppWhenRun = true
    @MainActor func perform() async throws -> some IntentResult { AtajosSiri.ir(.caja); return .result() }
}

struct VerAgenda: AppIntent {
    static var title: LocalizedStringResource = "Ver la agenda"
    static var openAppWhenRun = true
    @MainActor func perform() async throws -> some IntentResult { AtajosSiri.ir(.agenda); return .result() }
}

struct VerListaEspera: AppIntent {
    static var title: LocalizedStringResource = "Ver la lista de espera"
    static var openAppWhenRun = true
    @MainActor func perform() async throws -> some IntentResult { AtajosSiri.ir(.espera); return .result() }
}

/// Responde sin abrir la app: el próximo turno y cuántos quedan hoy.
struct ProximoTurno: AppIntent {
    static var title: LocalizedStringResource = "Mi próximo turno"
    static var description = IntentDescription("Te dice el próximo turno de hoy y cuántos quedan.")
    static var openAppWhenRun = false

    func perform() async throws -> some IntentResult & ProvidesDialog {
        let d = await DiaChico.cargar()
        guard d.conSesion else { return .result(dialog: "Entrá a Tengo Libre en el iPhone para ver tus turnos.") }
        guard let t = d.proximo else { return .result(dialog: "No te quedan turnos por hoy.") }
        let f = DateFormatter()
        f.timeZone = SesionCompartida.zona
        f.dateFormat = "HH:mm"
        let resto = d.turnos.count - 1
        let cola = resto == 0 ? "Es el último del día." : (resto == 1 ? "Después queda uno más." : "Después quedan \(resto).")
        return .result(dialog: "A las \(f.string(from: t.inicio)), \(t.cliente.nombre), \(t.servicio) con \(t.profesional). \(cola)")
    }
}

struct AtajosTengoLibre: AppShortcutsProvider {
    static var appShortcuts: [AppShortcut] {
        AppShortcut(intent: ProximoTurno(), phrases: ["Mi próximo turno en \(.applicationName)", "Qué tengo hoy en \(.applicationName)", "Próximo turno de \(.applicationName)"],
                    shortTitle: "Próximo turno", systemImageName: "clock")
        AppShortcut(intent: CargarTurno(), phrases: ["Cargar un turno en \(.applicationName)", "Nuevo turno en \(.applicationName)"],
                    shortTitle: "Cargar un turno", systemImageName: "plus.circle")
        AppShortcut(intent: AbrirHoy(), phrases: ["Ver mi día en \(.applicationName)", "Abrir hoy en \(.applicationName)"],
                    shortTitle: "Mi día", systemImageName: "sun.max")
        AppShortcut(intent: VerCaja(), phrases: ["Ver la caja en \(.applicationName)", "Cuánto cobré hoy en \(.applicationName)"],
                    shortTitle: "Caja de hoy", systemImageName: "banknote")
        AppShortcut(intent: VerAgenda(), phrases: ["Ver la agenda de \(.applicationName)"],
                    shortTitle: "Agenda", systemImageName: "calendar")
        AppShortcut(intent: VerListaEspera(), phrases: ["Ver la lista de espera de \(.applicationName)"],
                    shortTitle: "Lista de espera", systemImageName: "hourglass")
    }
}
