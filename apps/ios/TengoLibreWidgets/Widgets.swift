import ActivityKit
import AppIntents
import SwiftUI
import WidgetKit

// Los widgets del handoff: el próximo turno (chico), el día (mediano), y en la pantalla bloqueada
// la cantidad que queda (círculo) y el próximo (rectángulo). Más la Live Activity del turno en curso.

private enum W {
    static let petroleo = Color(red: 0x18 / 255, green: 0x3C / 255, blue: 0x44 / 255)
    static let hondo = Color(red: 0x10 / 255, green: 0x2B / 255, blue: 0x31 / 255)
    static let menta = Color(red: 0x3C / 255, green: 0xCC / 255, blue: 0x90 / 255)
    static let crema = Color(red: 0xF4 / 255, green: 0xEE / 255, blue: 0xE2 / 255)
    static func lato(_ t: CGFloat, _ p: Font.Weight = .regular) -> Font {
        let n = p == .black || p == .heavy ? "Lato-Black" : (p == .bold || p == .semibold ? "Lato-Bold" : "Lato-Regular")
        return .custom(n, size: t)
    }
    static func hora(_ d: Date) -> String {
        let f = DateFormatter()
        f.timeZone = SesionCompartida.zona
        f.dateFormat = "HH:mm"
        return f.string(from: d)
    }
    static func nombre(_ s: String) -> String { s.split(separator: " ").first.map(String.init) ?? s }
}

struct Entrada: TimelineEntry {
    var date: Date
    var dia: DiaChico
}

struct Proveedor: TimelineProvider {
    func placeholder(in context: Context) -> Entrada { Entrada(date: Date(), dia: .ejemplo) }

    func getSnapshot(in context: Context, completion: @escaping (Entrada) -> Void) {
        if context.isPreview { completion(placeholder(in: context)); return }
        Task { completion(Entrada(date: Date(), dia: await DiaChico.cargar())) }
    }

    func getTimeline(in context: Context, completion: @escaping (Timeline<Entrada>) -> Void) {
        Task {
            let d = await DiaChico.cargar()
            // Una entrada por cada turno que termina (para que «el próximo» avance solo) y se vuelve
            // a pedir a los 15 minutos.
            var entradas = [Entrada(date: Date(), dia: d)]
            for t in d.turnos.prefix(6) {
                let resto = d.turnos.filter { $0.fin > t.fin }
                entradas.append(Entrada(date: t.fin, dia: DiaChico(negocio: d.negocio, turnos: resto, hechos: d.hechos + 1, conSesion: d.conSesion)))
            }
            completion(Timeline(entries: entradas, policy: .after(Date().addingTimeInterval(15 * 60))))
        }
    }
}

struct VistaWidget: View {
    var e: Entrada
    @Environment(\.widgetFamily) private var familia

    var body: some View {
        switch familia {
        case .accessoryCircular: circular
        case .accessoryRectangular: rectangular
        case .accessoryInline: inline
        case .systemMedium: mediano
        default: chico
        }
    }

    private var sinSesion: some View {
        VStack(alignment: .leading, spacing: 4) {
            Image(systemName: "calendar").font(.system(size: 22, weight: .bold)).foregroundStyle(W.menta)
            Spacer()
            Text("Entrá a Tengo Libre").font(W.lato(15, .black)).foregroundStyle(.white)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .leading)
    }

    private var chico: some View {
        Group {
            if !e.dia.conSesion { sinSesion }
            else if let t = e.dia.proximo {
                VStack(alignment: .leading, spacing: 2) {
                    Text(t.inicio <= e.date ? "AHORA" : "PRÓXIMO").font(W.lato(11, .black)).tracking(0.8).foregroundStyle(W.menta)
                    Text(W.hora(t.inicio)).font(W.lato(30, .black)).monospacedDigit().foregroundStyle(.white)
                    Spacer(minLength: 0)
                    Text(t.cliente.nombre).font(W.lato(15, .bold)).foregroundStyle(.white).lineLimit(1)
                    Text(t.servicio).font(W.lato(12)).foregroundStyle(.white.opacity(0.75)).lineLimit(1)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .leading)
            } else {
                VStack(alignment: .leading, spacing: 4) {
                    Text("HOY").font(W.lato(11, .black)).tracking(0.8).foregroundStyle(W.menta)
                    Spacer()
                    Text("Todo libre").font(W.lato(22, .black)).foregroundStyle(.white)
                    Text(e.dia.hechos > 0 ? "\(e.dia.hechos) atendidos" : "Sin turnos que vengan").font(W.lato(12)).foregroundStyle(.white.opacity(0.75))
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .leading)
            }
        }
        .widgetURL(URL(string: e.dia.proximo.map { "tengolibre://turno/\($0.id)" } ?? "tengolibre://hoy"))
        .containerBackground(W.petroleo, for: .widget)
    }

    private var mediano: some View {
        HStack(spacing: 14) {
            chicoContenido.frame(width: 120)
            Rectangle().fill(.white.opacity(0.2)).frame(width: 1)
            VStack(alignment: .leading, spacing: 8) {
                Text("DESPUÉS").font(W.lato(11, .black)).tracking(0.8).foregroundStyle(W.menta)
                if e.dia.turnos.count <= 1 {
                    Text("Nada más por hoy").font(W.lato(14, .bold)).foregroundStyle(.white.opacity(0.8))
                }
                ForEach(e.dia.turnos.dropFirst().prefix(3)) { t in
                    Link(destination: URL(string: "tengolibre://turno/\(t.id)")!) {
                        HStack(spacing: 8) {
                            Text(W.hora(t.inicio)).font(W.lato(14, .black)).monospacedDigit().foregroundStyle(.white)
                            Text(W.nombre(t.cliente.nombre)).font(W.lato(14, .bold)).foregroundStyle(.white).lineLimit(1)
                            Spacer(minLength: 0)
                        }
                    }
                }
                Spacer(minLength: 0)
                Text("\(e.dia.turnos.count) por atender · \(e.dia.hechos) atendidos").font(W.lato(11)).foregroundStyle(.white.opacity(0.7))
            }
        }
        .widgetURL(URL(string: "tengolibre://hoy"))
        .containerBackground(W.petroleo, for: .widget)
    }

    @ViewBuilder private var chicoContenido: some View {
        if let t = e.dia.proximo {
            VStack(alignment: .leading, spacing: 2) {
                Text(t.inicio <= e.date ? "AHORA" : "PRÓXIMO").font(W.lato(11, .black)).tracking(0.8).foregroundStyle(W.menta)
                Text(W.hora(t.inicio)).font(W.lato(30, .black)).monospacedDigit().foregroundStyle(.white)
                Spacer(minLength: 0)
                Text(t.cliente.nombre).font(W.lato(15, .bold)).foregroundStyle(.white).lineLimit(2)
                Text(t.servicio).font(W.lato(12)).foregroundStyle(.white.opacity(0.75)).lineLimit(1)
            }
            .frame(maxHeight: .infinity, alignment: .leading)
        } else if !e.dia.conSesion {
            sinSesion
        } else {
            VStack(alignment: .leading) {
                Text("HOY").font(W.lato(11, .black)).foregroundStyle(W.menta)
                Spacer()
                Text("Todo libre").font(W.lato(20, .black)).foregroundStyle(.white)
            }
            .frame(maxHeight: .infinity, alignment: .leading)
        }
    }

    private var circular: some View {
        ZStack {
            AccessoryWidgetBackground()
            VStack(spacing: -2) {
                Text("\(e.dia.turnos.count)").font(.system(size: 22, weight: .black, design: .rounded)).monospacedDigit()
                Text("turnos").font(.system(size: 9, weight: .bold))
            }
        }
        .widgetURL(URL(string: "tengolibre://hoy"))
        .containerBackground(.clear, for: .widget)
        .accessibilityLabel("\(e.dia.turnos.count) turnos por atender hoy")
    }

    private var rectangular: some View {
        VStack(alignment: .leading, spacing: 1) {
            if let t = e.dia.proximo {
                Text("\(W.hora(t.inicio)) · \(W.nombre(t.cliente.nombre))").font(.system(size: 15, weight: .bold)).widgetAccentable()
                Text(t.servicio).font(.system(size: 13)).lineLimit(1)
                Text("Quedan \(e.dia.turnos.count)").font(.system(size: 12)).foregroundStyle(.secondary)
            } else {
                Text("Tengo Libre").font(.system(size: 15, weight: .bold)).widgetAccentable()
                Text(e.dia.conSesion ? "No quedan turnos hoy" : "Entrá a la app").font(.system(size: 13))
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .widgetURL(URL(string: e.dia.proximo.map { "tengolibre://turno/\($0.id)" } ?? "tengolibre://hoy"))
        .containerBackground(.clear, for: .widget)
    }

    private var inline: some View {
        Group {
            if let t = e.dia.proximo { Text("\(W.hora(t.inicio)) \(W.nombre(t.cliente.nombre))") } else { Text("Hoy, todo libre") }
        }
        .containerBackground(.clear, for: .widget)
    }
}

struct WidgetDia: Widget {
    var body: some WidgetConfiguration {
        StaticConfiguration(kind: "dia", provider: Proveedor()) { e in VistaWidget(e: e) }
            .configurationDisplayName("Tu día")
            .description("El próximo turno y lo que sigue.")
            .supportedFamilies([.systemSmall, .systemMedium, .accessoryCircular, .accessoryRectangular, .accessoryInline])
    }
}

// MARK: - Live Activity

struct WidgetTurnoEnCurso: Widget {
    var body: some WidgetConfiguration {
        ActivityConfiguration(for: AtributosTurno.self) { ctx in
            // Pantalla bloqueada.
            HStack(spacing: 14) {
                VStack(alignment: .leading, spacing: 2) {
                    Text("EN CURSO").font(W.lato(11, .black)).tracking(0.8).foregroundStyle(W.menta)
                    Text(ctx.attributes.cliente).font(W.lato(18, .black)).foregroundStyle(.white).lineLimit(1)
                    Text("\(ctx.attributes.servicio) · \(ctx.attributes.profesional)").font(W.lato(13)).foregroundStyle(.white.opacity(0.75)).lineLimit(1)
                    if !ctx.state.siguiente.isEmpty { Text("Sigue: \(ctx.state.siguiente)").font(W.lato(12, .bold)).foregroundStyle(.white.opacity(0.85)) }
                }
                Spacer()
                VStack(alignment: .trailing, spacing: 6) {
                    Text(timerInterval: Date()...max(Date(), ctx.state.fin), countsDown: true).font(W.lato(22, .black)).monospacedDigit().foregroundStyle(.white)
                        .multilineTextAlignment(.trailing).frame(width: 84)
                    Button(intent: MarcarAtendido(turnoId: ctx.attributes.turnoId)) {
                        Label("Atendido", systemImage: "checkmark").font(W.lato(13, .black))
                    }
                    .tint(W.menta).foregroundStyle(W.hondo)
                }
            }
            .padding(16)
            .activityBackgroundTint(W.petroleo)
            .activitySystemActionForegroundColor(W.menta)
            .widgetURL(URL(string: "tengolibre://turno/\(ctx.attributes.turnoId)"))
        } dynamicIsland: { ctx in
            DynamicIsland {
                DynamicIslandExpandedRegion(.leading) {
                    VStack(alignment: .leading, spacing: 2) {
                        Text("EN CURSO").font(W.lato(10, .black)).foregroundStyle(W.menta)
                        Text(W.nombre(ctx.attributes.cliente)).font(W.lato(16, .black)).lineLimit(1)
                    }
                }
                DynamicIslandExpandedRegion(.trailing) {
                    Text(timerInterval: Date()...max(Date(), ctx.state.fin), countsDown: true).font(W.lato(18, .black)).monospacedDigit()
                        .multilineTextAlignment(.trailing).frame(width: 70)
                }
                DynamicIslandExpandedRegion(.bottom) {
                    HStack {
                        Text(ctx.attributes.servicio).font(W.lato(13)).foregroundStyle(.secondary).lineLimit(1)
                        Spacer()
                        Button(intent: MarcarAtendido(turnoId: ctx.attributes.turnoId)) { Label("Atendido", systemImage: "checkmark").font(W.lato(13, .black)) }
                            .tint(W.menta).foregroundStyle(W.hondo)
                    }
                }
            } compactLeading: {
                Image(systemName: "timelapse").foregroundStyle(W.menta)
            } compactTrailing: {
                Text(timerInterval: Date()...max(Date(), ctx.state.fin), countsDown: true).monospacedDigit().frame(width: 44).font(.system(size: 13, weight: .bold))
            } minimal: {
                Image(systemName: "timelapse").foregroundStyle(W.menta)
            }
            .widgetURL(URL(string: "tengolibre://turno/\(ctx.attributes.turnoId)"))
            .keylineTint(W.menta)
        }
    }
}

@main
struct TengoLibreWidgets: WidgetBundle {
    var body: some Widget {
        WidgetDia()
        WidgetTurnoEnCurso()
    }
}
