import SwiftUI
import TipKit

/// Los consejos nativos (TipKit): aparecen la primera vez que se ve cada gesto, una vez y listo.
enum Consejos {
    static func configurar() {
        if ProcessInfo.processInfo.environment["XCTestConfigurationFilePath"] != nil {
            Tips.hideAllTipsForTesting()
        }
        try? Tips.configure([.displayFrequency(.hourly), .datastoreLocation(.applicationDefault)])
    }

    /// Cuántas veces se abrió Hoy (para el consejo del widget, a partir del tercer día).
    static let abrioHoy = Tips.Event(id: "abrioHoy")
}

struct ConsejoDeslizar: Tip {
    var title: Text { Text("Deslizá el turno") }
    var message: Text? { Text("Hacia la izquierda: atendido o no vino, sin abrirlo.") }
    var image: Image? { Image(systemName: "hand.draw") }
}

struct ConsejoHueco: Tip {
    var title: Text { Text("Tocá un hueco libre") }
    var message: Text? { Text("Cargás un turno ahí mismo, con la hora y la profesional ya puestas.") }
    var image: Image? { Image(systemName: "hand.tap") }
}

struct ConsejoAvisar: Tip {
    var title: Text { Text("Hay gente esperando este lugar") }
    var message: Text? { Text("«Avisar» se lo ofrece de a una persona por vez. Si nadie lo toma, sigue la próxima.") }
    var image: Image? { Image(systemName: "megaphone") }
}

struct ConsejoCapsula: Tip {
    var title: Text { Text("Pasá de profesional") }
    var message: Text? { Text("Deslizá la agenda o usá las flechas de abajo para ver a cada una o a todas juntas.") }
    var image: Image? { Image(systemName: "person.2") }
}

struct ConsejoEtiquetas: Tip {
    var title: Text { Text("Etiquetá a tus clientas") }
    var message: Text? { Text("VIP, nueva, alérgica… después las filtrás en la lista.") }
    var image: Image? { Image(systemName: "tag") }
}

struct ConsejoWidget: Tip {
    var title: Text { Text("Tu día sin abrir la app") }
    var message: Text? { Text("Mantené apretada la pantalla de inicio y sumá el widget de Tengo Libre.") }
    var image: Image? { Image(systemName: "apps.iphone") }
    var rules: [Rule] { [#Rule(Consejos.abrioHoy) { $0.donations.donatedWithin(.week).count >= 3 }] }
}

struct ConsejoBuscar: Tip {
    var title: Text { Text("¿No encontrás algo?") }
    var message: Text? { Text("Escribí lo que querés hacer: «caja», «seña», «IA», o el nombre de una clienta.") }
    var image: Image? { Image(systemName: "magnifyingglass") }
}

extension View {
    /// Un consejo en globo sólo si se cumple la condición (el primer hueco, la primera vez…).
    @ViewBuilder func consejo<T: Tip>(_ si: Bool, _ tip: T) -> some View {
        if si { popoverTip(tip) } else { self }
    }
}
