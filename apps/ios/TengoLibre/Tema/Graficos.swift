import Charts
import SwiftUI

// Los gráficos de la app (Swift Charts), con la paleta del handoff: menta para lo que entra, petróleo
// para lo ocupado, crema para el fondo. Todos crecen al aparecer y se pueden tocar para ver el valor.

/// Colores para series (medios de pago, servicios, estados): de la marca, distinguibles entre sí.
enum Paleta {
    static let series: [Color] = [C.menta, C.petroleo, Color(hex: 0xE8A317), Color(hex: 0x7A5CFA), Color(hex: 0x2F80ED), Color(hex: 0xEC4899), Color(hex: 0x8B5E34)]
    static func color(_ i: Int) -> Color { series[i % series.count] }
}

/// Un número que cambia contando (plata, cantidades).
extension View {
    func numeroAnimado<V: Equatable>(_ valor: V) -> some View {
        contentTransition(.numericText()).animation(Movimiento.entrar, value: valor)
    }
}

/// Entrada escalonada: cada tarjeta sube y aparece un poquito después que la anterior (una sola vez).
struct Entrada: ViewModifier {
    var indice: Int
    @State private var visto = false
    @Environment(\.accessibilityReduceMotion) private var reducir
    func body(content: Content) -> some View {
        content
            .opacity(visto || reducir ? 1 : 0)
            .offset(y: visto || reducir ? 0 : 14)
            .onAppear {
                guard !visto else { return }
                withAnimation(Movimiento.entrar.delay(Double(min(indice, 8)) * 0.045)) { visto = true }
            }
    }
}

extension View {
    func entrada(_ indice: Int = 0) -> some View { modifier(Entrada(indice: indice)) }
}

/// Un punto que late (lo que está pasando ahora).
struct Pulso: View {
    var color: Color = C.menta
    var tam: CGFloat = 10
    @State private var latir = false
    @Environment(\.accessibilityReduceMotion) private var reducir
    var body: some View {
        ZStack {
            Circle().fill(color.opacity(0.35)).frame(width: tam * 2.2, height: tam * 2.2)
                .scaleEffect(latir ? 1 : 0.4).opacity(latir ? 0 : 1)
            Circle().fill(color).frame(width: tam, height: tam)
        }
        .frame(width: tam * 2.2, height: tam * 2.2)
        .onAppear {
            guard !reducir else { return }
            withAnimation(.easeOut(duration: 1.4).repeatForever(autoreverses: false)) { latir = true }
        }
        .accessibilityHidden(true)
    }
}

// MARK: - Ingresos por día (barras, con el promedio y el valor al tocar)

struct GraficoIngresos: View {
    struct Dia: Identifiable { var fecha: Date; var monto: Int; var id: Date { fecha } }
    var dias: [Dia]
    var alto: CGFloat = 180
    @State private var crecer: CGFloat = 0
    @State private var elegido: Date?

    private var promedio: Double {
        let con = dias.filter { $0.monto > 0 }
        return con.isEmpty ? 0 : Double(con.reduce(0) { $0 + $1.monto }) / Double(con.count)
    }
    private var dElegido: Dia? {
        guard let elegido else { return nil }
        return dias.min { abs($0.fecha.timeIntervalSince(elegido)) < abs($1.fecha.timeIntervalSince(elegido)) }
    }

    var body: some View {
        Chart {
            ForEach(dias) { d in
                BarMark(x: .value("Día", d.fecha, unit: .day), y: .value("Ingresos", Double(d.monto) * crecer))
                    .foregroundStyle(dElegido?.id == d.id ? AnyShapeStyle(C.petroleo) : AnyShapeStyle(LinearGradient(colors: [C.menta, C.menta.opacity(0.55)], startPoint: .top, endPoint: .bottom)))
                    .cornerRadius(4)
            }
            if promedio > 0 {
                RuleMark(y: .value("Promedio", promedio * Double(crecer)))
                    .lineStyle(StrokeStyle(lineWidth: 1, dash: [4, 4])).foregroundStyle(C.sub)
                    .annotation(position: .top, alignment: .trailing) {
                        Text("prom. \(Formato.pesos(Int(promedio)))").font(Tipo.lato(10, .bold)).foregroundStyle(C.sub)
                    }
            }
            if let d = dElegido {
                RuleMark(x: .value("Día", d.fecha, unit: .day)).foregroundStyle(C.linea)
                    .annotation(position: .top, overflowResolution: .init(x: .fit(to: .chart), y: .disabled)) {
                        VStack(spacing: 1) {
                            Text(Formato.pesos(d.monto)).font(Tipo.lato(14, .black)).monospacedDigit()
                            Text(Formato.dia(d.fecha)).font(Tipo.lato(11)).foregroundStyle(C.sub)
                        }
                        .padding(.horizontal, 8).padding(.vertical, 5)
                        .background(C.tarjeta, in: RoundedRectangle(cornerRadius: 8)).shadow(color: .black.opacity(0.1), radius: 6, y: 3)
                    }
            }
        }
        .chartXSelection(value: $elegido)
        .chartYAxis {
            AxisMarks(position: .leading, values: .automatic(desiredCount: 3)) { v in
                AxisGridLine().foregroundStyle(C.linea)
                AxisValueLabel { if let x = v.as(Double.self) { Text(Formato.pesosCortos(Int(x))).font(Tipo.lato(10)) } }
            }
        }
        .chartXAxis {
            AxisMarks(values: .automatic(desiredCount: 5)) { _ in
                AxisValueLabel(format: .dateTime.day().month(.abbreviated), centered: true).font(Tipo.lato(10))
            }
        }
        .frame(height: alto)
        .onAppear { withAnimation(.spring(response: 0.7, dampingFraction: 0.85)) { crecer = 1 } }
        .sensoryFeedback(.selection, trigger: dElegido?.id)
        .accessibilityLabel("Ingresos por día")
    }
}

// MARK: - Dona (medios de pago, servicios, estados), con el total en el centro

struct GraficoDona: View {
    struct Parte: Identifiable { var nombre: String; var valor: Int; var color: Color; var id: String { nombre } }
    var partes: [Parte]
    var centro: String
    var pie: String
    var mostrarPesos = true
    @State private var crecer: Double = 0
    @State private var elegido: Int?

    private var total: Int { partes.reduce(0) { $0 + $1.valor } }
    private var parteElegida: Parte? {
        guard let elegido else { return nil }
        var acum = 0
        for p in partes where p.valor > 0 {
            acum += p.valor
            if elegido <= acum { return p }
        }
        return nil
    }

    var body: some View {
        HStack(spacing: 18) {
            Chart(partes.filter { $0.valor > 0 }) { p in
                SectorMark(angle: .value("Valor", Double(p.valor) * crecer + 0.0001), innerRadius: .ratio(0.62), angularInset: 1.5)
                    .foregroundStyle(p.color)
                    .cornerRadius(4)
                    .opacity(parteElegida == nil || parteElegida?.id == p.id ? 1 : 0.35)
            }
            .chartAngleSelection(value: $elegido)
            .chartBackground { _ in
                VStack(spacing: 0) {
                    Text(parteElegida.map { mostrarPesos ? Formato.pesosCortos($0.valor) : "\($0.valor)" } ?? centro)
                        .font(Tipo.lato(17, .black)).monospacedDigit().minimumScaleFactor(0.6).lineLimit(1)
                    Text(parteElegida?.nombre ?? pie).font(Tipo.lato(10)).foregroundStyle(C.sub).lineLimit(1)
                }
                .padding(.horizontal, 22)
            }
            .frame(width: 140, height: 140)
            VStack(alignment: .leading, spacing: 8) {
                ForEach(partes) { p in
                    HStack(spacing: 8) {
                        RoundedRectangle(cornerRadius: 3).fill(p.color).frame(width: 12, height: 12)
                        Text(p.nombre).font(Tipo.lato(13)).foregroundStyle(p.valor == 0 ? C.sub : C.tinta).lineLimit(1)
                        Spacer(minLength: 4)
                        Text(total > 0 ? "\(Int((Double(p.valor) / Double(total) * 100).rounded())) %" : "—")
                            .font(Tipo.lato(13, .bold)).monospacedDigit().foregroundStyle(C.sub)
                    }
                    .opacity(parteElegida == nil || parteElegida?.id == p.id ? 1 : 0.45)
                }
            }
        }
        .onAppear { withAnimation(.spring(response: 0.9, dampingFraction: 0.9)) { crecer = 1 } }
        .sensoryFeedback(.selection, trigger: parteElegida?.id)
        .accessibilityElement(children: .combine)
    }
}

// MARK: - Barras horizontales (por profesional, por servicio, reseñas)

struct GraficoBarras: View {
    struct Fila: Identifiable { var nombre: String; var valor: Int; var detalle: String = ""; var id: String { nombre } }
    var filas: [Fila]
    var formato: (Int) -> String = { Formato.pesos($0) }
    var color: Color = C.menta
    @State private var crecer: CGFloat = 0

    var body: some View {
        Chart(filas) { f in
            BarMark(x: .value("Valor", Double(f.valor) * crecer), y: .value("Nombre", f.nombre))
                .foregroundStyle(color.gradient)
                .cornerRadius(5)
                .annotation(position: .trailing, alignment: .leading) {
                    Text(formato(f.valor)).font(Tipo.lato(12, .bold)).monospacedDigit().foregroundStyle(C.tinta).opacity(crecer)
                }
        }
        .chartXAxis(.hidden)
        .chartYAxis {
            AxisMarks { _ in AxisValueLabel().font(Tipo.lato(12)).foregroundStyle(C.tinta) }
        }
        .chartXScale(domain: 0...(Double(filas.map(\.valor).max() ?? 1) * 1.35 + 1))
        .frame(height: CGFloat(max(1, filas.count)) * 38 + 8)
        .onAppear { withAnimation(.spring(response: 0.7, dampingFraction: 0.85).delay(0.1)) { crecer = 1 } }
    }
}

// MARK: - Línea chiquita (tendencia de la semana)

struct Sparkline: View {
    var valores: [Int]
    var color: Color = C.menta
    @State private var trazo: CGFloat = 0
    var body: some View {
        Chart(Array(valores.enumerated()), id: \.offset) { i, v in
            AreaMark(x: .value("i", i), y: .value("v", v))
                .foregroundStyle(LinearGradient(colors: [color.opacity(0.35), color.opacity(0)], startPoint: .top, endPoint: .bottom))
                .interpolationMethod(.catmullRom)
            LineMark(x: .value("i", i), y: .value("v", v))
                .foregroundStyle(color).lineStyle(StrokeStyle(lineWidth: 2.5, lineCap: .round))
                .interpolationMethod(.catmullRom)
        }
        .chartXAxis(.hidden).chartYAxis(.hidden).chartLegend(.hidden)
        .mask(alignment: .leading) { GeometryReader { g in Rectangle().frame(width: g.size.width * trazo) } }
        .onAppear { withAnimation(.easeOut(duration: 0.9)) { trazo = 1 } }
        .accessibilityHidden(true)
    }
}

// MARK: - Visitas por mes (la ficha de una clienta)

struct GraficoVisitas: View {
    struct Mes: Identifiable { var mes: Date; var visitas: Int; var id: Date { mes } }
    var meses: [Mes]
    @State private var crecer: CGFloat = 0
    var body: some View {
        Chart(meses) { m in
            BarMark(x: .value("Mes", m.mes, unit: .month), y: .value("Visitas", Double(m.visitas) * crecer))
                .foregroundStyle(m.visitas > 0 ? C.menta.gradient : C.chip.gradient)
                .cornerRadius(4)
        }
        .chartYAxis {
            AxisMarks(position: .leading, values: .automatic(desiredCount: 3)) { _ in
                AxisGridLine().foregroundStyle(C.linea)
                AxisValueLabel().font(Tipo.lato(10))
            }
        }
        .chartXAxis {
            AxisMarks(values: .stride(by: .month, count: 2)) { _ in
                AxisValueLabel(format: .dateTime.month(.narrow), centered: true).font(Tipo.lato(10))
            }
        }
        .frame(height: 120)
        .onAppear { withAnimation(.spring(response: 0.7, dampingFraction: 0.85)) { crecer = 1 } }
        .accessibilityLabel("Visitas por mes")
    }
}

// MARK: - Mapa de calor de la semana (horas muertas)

struct MapaCalor: View {
    struct Celda: Codable, Hashable { var dia: Int; var hora: Int; var ocupacion: Double?; var vacio: Double?; var estado: String }
    struct Mapa: Codable { var dias: [Int]; var horas: [Int]; var celdas: [Celda] }
    var mapa: Mapa
    @State private var aparecer = 0.0
    @State private var elegida: Celda?
    private static let dias = ["Dom", "Lun", "Mar", "Mié", "Jue", "Vie", "Sáb"]

    private func color(_ c: Celda) -> Color {
        if c.estado == "cerrado" { return C.chip }
        let o = (c.ocupacion ?? 0) / 100
        return C.petroleo.opacity(0.12 + 0.88 * o)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Chart(mapa.celdas, id: \.self) { c in
                RectangleMark(x: .value("Hora", "\(c.hora)"), y: .value("Día", Self.dias[c.dia % 7]), width: .ratio(0.9), height: .ratio(0.85))
                    .foregroundStyle(color(c)).cornerRadius(4).opacity(aparecer)
            }
            .chartXScale(domain: mapa.horas.map { "\($0)" })
            .chartYScale(domain: mapa.dias.map { Self.dias[$0 % 7] })
            .chartXAxis { AxisMarks { _ in AxisValueLabel().font(Tipo.lato(10)) } }
            .chartYAxis { AxisMarks(position: .leading) { _ in AxisValueLabel().font(Tipo.lato(11, .bold)) } }
            .frame(height: CGFloat(mapa.dias.count) * 30 + 24)
            .chartOverlay { p in
                GeometryReader { g in
                    Rectangle().fill(.clear).contentShape(Rectangle())
                        .onTapGesture { punto in
                            guard let marco = p.plotFrame else { return }
                            let x = punto.x - g[marco].origin.x, y = punto.y - g[marco].origin.y
                            if let h: String = p.value(atX: x), let d: String = p.value(atY: y),
                               let hi = Int(h), let di = Self.dias.firstIndex(of: d) {
                                elegida = mapa.celdas.first { $0.hora == hi && $0.dia == di }
                            }
                        }
                }
            }
            HStack(spacing: 10) {
                if let c = elegida {
                    Text("\(Self.dias[c.dia % 7]) \(c.hora) h: ").font(Tipo.lato(12, .bold))
                    + Text(c.estado == "cerrado" ? "cerrado" : "ocupado el \(Int((c.ocupacion ?? 0).rounded())) %").font(Tipo.lato(12))
                } else {
                    Text("Tocá una hora para ver qué tan llena suele estar.").font(Tipo.lato(12)).foregroundStyle(C.sub)
                }
                Spacer()
                HStack(spacing: 3) {
                    Text("vacío").font(Tipo.lato(10)).foregroundStyle(C.sub)
                    ForEach([0.12, 0.4, 0.7, 1.0], id: \.self) { o in RoundedRectangle(cornerRadius: 2).fill(C.petroleo.opacity(o)).frame(width: 12, height: 12) }
                    Text("lleno").font(Tipo.lato(10)).foregroundStyle(C.sub)
                }
            }
        }
        .onAppear { withAnimation(.easeOut(duration: 0.6)) { aparecer = 1 } }
    }
}

extension Formato {
    /// «$ 15 mil», «$ 1,2 M» (para ejes y centros de gráficos).
    static func pesosCortos(_ n: Int) -> String {
        if abs(n) >= 1_000_000 { return "$ " + String(format: "%.1f", Double(n) / 1_000_000).replacingOccurrences(of: ".", with: ",").replacingOccurrences(of: ",0", with: "") + " M" }
        if abs(n) >= 1_000 { return "$ \(Int((Double(n) / 1_000).rounded())) mil" }
        return pesos(n)
    }
}
