import SwiftUI

/// La plata, a la vista en Hoy: lo cobrado hoy y la semana contra la anterior. Tocar abre la caja o los
/// reportes. Se vuelve a pedir con cada cambio (los cobros y pagos llegan por el SSE).
struct PlataHoy: View {
    @Environment(Estado.self) private var estado
    @Environment(Navegacion.self) private var nav
    @State private var caja: ResumenCaja?
    @State private var semana: Int?
    @State private var antes: Int?
    @State private var ocupacion: Double?
    @State private var dias: [Int] = []

    var body: some View {
        if estado.vePrecios && !estado.esProfesional {
            HStack(spacing: 0) {
                Button { nav.pilaHoy.append(DestinoMas.caja) } label: {
                    VStack(alignment: .leading, spacing: 2) {
                        Text("COBRADO HOY").font(Tipo.lato(11, .black)).tracking(0.8).foregroundStyle(C.menta)
                        Text(caja.map { Formato.pesos($0.total) } ?? "—").font(Tipo.lato(26, .black)).monospacedDigit().minimumScaleFactor(0.6).lineLimit(1)
                            .numeroAnimado(caja?.total)
                        Text(textoHoy).font(Tipo.lato(12)).foregroundStyle(.white.opacity(0.75)).lineLimit(1)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading).padding(16)
                }
                .buttonStyle(.plain)
                .accessibilityLabel("Cobrado hoy: \(caja.map { Formato.pesos($0.total) } ?? "nada todavía"). Ver la caja")
                Rectangle().fill(.white.opacity(0.18)).frame(width: 1).padding(.vertical, 14)
                Button { nav.pilaHoy.append(DestinoMas.reportes) } label: {
                    VStack(alignment: .leading, spacing: 2) {
                        Text("ESTA SEMANA").font(Tipo.lato(11, .black)).tracking(0.8).foregroundStyle(C.menta)
                        Text(semana.map { Formato.pesos($0) } ?? "—").font(Tipo.lato(26, .black)).monospacedDigit().minimumScaleFactor(0.6).lineLimit(1)
                            .numeroAnimado(semana)
                        if dias.count > 1 { Sparkline(valores: dias).frame(height: 22).padding(.vertical, 2) }
                        Text(textoSemana).font(Tipo.lato(12, .bold)).foregroundStyle(colorSemana).lineLimit(1)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading).padding(16)
                }
                .buttonStyle(.plain)
                .accessibilityLabel("Esta semana: \(semana.map { Formato.pesos($0) } ?? "sin datos"). \(textoSemana). Ver reportes")
            }
            .foregroundStyle(.white)
            .background { ZStack { C.bloquePetroleo; Image(uiImage: Grano.imagen).resizable(resizingMode: .tile).opacity(0.5) } }
            .clipShape(RoundedRectangle(cornerRadius: 22, style: .continuous))
            .task(id: estado.version) { await cargar() }
        }
    }

    private var textoHoy: String {
        guard let c = caja else { return "Trayendo la caja…" }
        if c.total == 0 { return "Cobrá desde cada turno" }
        var p = [Formato.cantidad(c.cobros, "cobro", "cobros")]
        if c.debe > 0 { p.append("deben \(Formato.pesos(c.debe))") }
        return p.joined(separator: " · ")
    }

    private var textoSemana: String {
        guard let s = semana else { return "" }
        guard let a = antes, a > 0 else { return ocupacion.map { "\(Formato.porcentaje($0)) ocupado" } ?? "Primera semana" }
        let d = Double(s - a) / Double(a) * 100
        return "\(d >= 0 ? "▲" : "▼") \(Int(abs(d).rounded())) % vs. la anterior"
    }

    private var colorSemana: Color {
        guard let s = semana, let a = antes, a > 0 else { return .white.opacity(0.75) }
        return s >= a ? C.menta : Color(hex: 0xFFB4A8)
    }

    private struct Resumen: Decodable { var ingresos: Int; var ocupacion: Double? }
    private struct DiaR: Decodable { var fecha: String; var ingresos: Int }
    private struct Rep: Decodable { var resumen: Resumen; var porDia: [DiaR]? }

    private func cargar() async {
        let cal = Formato.calendario, hoy = cal.startOfDay(for: Date())
        let lunes = cal.date(from: cal.dateComponents([.yearForWeekOfYear, .weekOfYear], from: hoy)) ?? hoy
        let transcurridos = cal.dateComponents([.day], from: lunes, to: hoy).day ?? 0
        let lunesAntes = cal.date(byAdding: .day, value: -7, to: lunes) ?? lunes
        let hastaAntes = cal.date(byAdding: .day, value: transcurridos, to: lunesAntes) ?? lunesAntes
        caja = try? await estado.api.get("/v1/panel/caja", query: [URLQueryItem(name: "fecha", value: Formato.diaISO(hoy))])
        let r: Rep? = try? await estado.api.get("/v1/panel/reportes", query: [URLQueryItem(name: "desde", value: Formato.diaISO(lunes)), URLQueryItem(name: "hasta", value: Formato.diaISO(hoy))])
        semana = r?.resumen.ingresos
        dias = (r?.porDia ?? []).sorted { $0.fecha < $1.fecha }.map(\.ingresos)
        ocupacion = r?.resumen.ocupacion
        // La semana anterior, hasta el mismo día (lunes a hoy contra lunes a «hoy» de la otra semana).
        let a: Rep? = try? await estado.api.get("/v1/panel/reportes", query: [URLQueryItem(name: "desde", value: Formato.diaISO(lunesAntes)), URLQueryItem(name: "hasta", value: Formato.diaISO(hastaAntes))])
        antes = a?.resumen.ingresos
    }
}
