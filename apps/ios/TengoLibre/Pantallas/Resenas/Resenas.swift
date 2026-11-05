import SwiftUI

/// 16 Reseñas: el promedio con la distribución, filtros (Todas · Sin responder · Ocultas) y cada
/// reseña con sus estrellas, «Respondiste» o el campo para responder, y mostrar u ocultar en la web.
struct Resenas: View {
    @Environment(Estado.self) private var estado
    @State private var resenas: [Resena] = []
    @State private var promedio: Double = 0
    @State private var cantidad = 0
    @State private var distribucion: [String: Int] = [:]
    @State private var filtro = "todas"
    @State private var cargando = true
    @State private var respuestas: [String: String] = [:]
    @State private var pagina = 0

    var body: some View {
        ScrollView {
            LazyVStack(spacing: 0) {
                Cabecera(titulo: "Reseñas", subtitulo: cantidad == 0 ? "Todavía sin reseñas" : "\(String(format: "%.1f", promedio).replacingOccurrences(of: ".", with: ",")) de 5 · \(Formato.cantidad(cantidad, "reseña", "reseñas"))",
                         volver: estado.esProfesional ? "" : "Más") {
                    Segmentado(opciones: [("todas", "Todas"), ("sin", "Sin responder"), ("ocultas", "Ocultas")], elegido: $filtro)
                }
                Cuerpo {
                    if cantidad > 0 && filtro == "todas" { resumen }
                    if cargando && resenas.isEmpty {
                        ForEach(0..<3, id: \.self) { _ in Esqueleto(alto: 140, radio: 20) }
                    } else if resenas.isEmpty {
                        VacioQueEnsena(rayas: "Sin reseñas", titulo: filtro == "sin" ? "Respondiste todas" : "Todavía no hay reseñas",
                                       texto: "Después de cada turno atendido le pedimos la reseña a la clienta. Las buenas se muestran en tu página.") { EmptyView() }
                    } else {
                        ForEach(resenas) { r in
                            tarjeta(r).onAppear { if r.id == resenas.last?.id && resenas.count < cantidad && filtro == "todas" { Task { await cargar(mas: true) } } }
                        }
                    }
                }
            }
        }
        .scrollIndicators(.hidden)
        .scrollDismissesKeyboard(.interactively)
        .background(C.fondo)
        .toolbar(.hidden, for: .navigationBar)
        .refreshable { await cargar() }
        .task(id: "\(estado.version)-\(filtro)") { await cargar() }
    }

    private var resumen: some View {
        HStack(spacing: 16) {
            VStack(spacing: 2) {
                Text(String(format: "%.1f", promedio).replacingOccurrences(of: ".", with: ",")).font(Tipo.lato(40, .black)).monospacedDigit()
                Estrellas(puntaje: Int(promedio.rounded()), tam: 14)
            }
            VStack(spacing: 4) {
                ForEach((1...5).reversed(), id: \.self) { n in
                    let c = distribucion["\(n)"] ?? 0
                    HStack(spacing: 6) {
                        Text("\(n)").font(Tipo.lato(12, .bold)).monospacedDigit().frame(width: 10)
                        GeometryReader { g in
                            ZStack(alignment: .leading) {
                                Capsule().fill(C.chip)
                                Capsule().fill(C.menta).frame(width: g.size.width * CGFloat(c) / CGFloat(max(1, cantidad)))
                            }
                        }
                        .frame(height: 6)
                        Text("\(c)").font(Tipo.lato(12)).monospacedDigit().foregroundStyle(C.sub).frame(width: 28, alignment: .trailing)
                    }
                }
            }
        }
        .foregroundStyle(C.tinta).padding(16).tarjeta()
    }

    private func tarjeta(_ r: Resena) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(alignment: .top) {
                Iniciales(nombre: r.nombre, tam: 40)
                VStack(alignment: .leading, spacing: 2) {
                    Text(r.nombre).font(Tipo.lato(16, .bold))
                    Text("\(r.servicio) · \(r.profesional) · \(Formato.hace(r.creadoEn))").font(Tipo.lato(12)).foregroundStyle(C.sub)
                }
                Spacer()
                Estrellas(puntaje: r.puntaje, tam: 13)
            }
            if !r.comentario.isEmpty { Text(r.comentario).font(Tipo.lato(15)).lineSpacing(2) }
            if !r.respuesta.isEmpty {
                VStack(alignment: .leading, spacing: 4) {
                    Text("RESPONDISTE").rotulo(C.mentaTexto)
                    Text(r.respuesta).font(Tipo.lato(14))
                }
                .padding(12).frame(maxWidth: .infinity, alignment: .leading)
                .background(C.mentaSuave.opacity(0.6), in: RoundedRectangle(cornerRadius: 14, style: .continuous))
            } else if !estado.soloLectura {
                HStack(alignment: .bottom, spacing: 8) {
                    TextField("Respondé a \(r.nombre.split(separator: " ").first ?? "")", text: Binding(get: { respuestas[r.id] ?? "" }, set: { respuestas[r.id] = $0 }), axis: .vertical)
                        .font(Tipo.lato(15)).lineLimit(1...5)
                        .padding(.horizontal, 12).padding(.vertical, 10).background(C.campo, in: RoundedRectangle(cornerRadius: 12))
                    Button("Responder") { Task { await responder(r) } }.buttonStyle(BotonMentaChico())
                        .disabled((respuestas[r.id] ?? "").trimmingCharacters(in: .whitespaces).isEmpty)
                }
            }
            if estado.gestiona {
                Toggle(isOn: Binding(get: { r.visible }, set: { v in Task { await visible(r, v) } })) {
                    Text(r.visible ? "Se ve en tu página" : "Oculta en tu página").font(Tipo.lato(14)).foregroundStyle(C.sub)
                }
                .tint(C.menta)
            }
        }
        .foregroundStyle(C.tinta).padding(16).tarjeta()
        .opacity(r.visible ? 1 : 0.75)
    }

    private struct Lista: Decodable { var promedio: Double; var cantidad: Int; var resenas: [Resena]; var distribucion: [String: Int]? }

    private func cargar(mas: Bool = false) async {
        let p = mas ? pagina + 1 : 0
        var q = [URLQueryItem(name: "pagina", value: "\(p)")]
        if filtro == "sin" { q.append(URLQueryItem(name: "sinResponder", value: "1")) }
        if filtro == "ocultas" { q.append(URLQueryItem(name: "visible", value: "no")) }
        do {
            let r: Lista = try await estado.api.get("/v1/panel/resenas", query: q)
            resenas = mas ? resenas + r.resenas : r.resenas
            if filtro == "todas" { promedio = r.promedio; cantidad = r.cantidad; distribucion = r.distribucion ?? distribucion }
            pagina = p
        } catch { if !mas { estado.avisar(error) } }
        cargando = false
    }

    private func responder(_ r: Resena) async {
        let texto = respuestas[r.id] ?? ""
        do {
            try await estado.api.hacer("PATCH", "/v1/panel/resenas/\(r.id)", AccionesTurno.Respuesta(respuesta: texto))
            respuestas[r.id] = nil
            if let i = resenas.firstIndex(where: { $0.id == r.id }) { resenas[i].respuesta = texto; resenas[i].respondidaEn = Date() }
            Haptico.exito()
            Momentos.resenaBuena(estado, r)
        } catch { estado.avisar(error) }
    }

    private func visible(_ r: Resena, _ v: Bool) async {
        do {
            try await estado.api.hacer("PATCH", "/v1/panel/resenas/\(r.id)", ["visible": v])
            if let i = resenas.firstIndex(where: { $0.id == r.id }) { resenas[i].visible = v }
        } catch { estado.avisar(error) }
    }
}

struct Estrellas: View {
    var puntaje: Int
    var tam: CGFloat = 14
    var body: some View {
        HStack(spacing: 2) {
            ForEach(1...5, id: \.self) { i in
                Image(systemName: i <= puntaje ? "star.fill" : "star").font(.system(size: tam, weight: .semibold))
                    .foregroundStyle(i <= puntaje ? Color(hex: 0xE8A317) : C.linea)
            }
        }
        .accessibilityElement().accessibilityLabel("\(puntaje) de 5 estrellas")
    }
}
