import SwiftUI

/// 12 Lista de espera: Esperando · Avisados · Vencidos. Cada persona con su rango de días y momento,
/// el servicio y con quién, desde cuándo espera y, si hay un hueco que le sirve, «Avisar». Arriba,
/// los huecos que se están ofreciendo de a una persona por vez (Frenar · Cargarlo yo).
struct ListaEspera: View {
    @Environment(Estado.self) private var estado
    @Environment(Navegacion.self) private var nav
    @State private var tab = "esperando"
    @State private var espera: [Anotado] = []
    @State private var ofertas: [Oferta] = []
    @State private var sugeridos: [String: [HorarioLibre]] = [:]
    @State private var profesionales: [Profesional] = []
    @State private var cargando = true
    @State private var borrar: Anotado?

    private var hoy: String { Formato.diaISO(Date()) }
    private func tabDe(_ a: Anotado) -> String { a.avisadoEn != nil ? "avisados" : (a.hasta < hoy ? "vencidos" : "esperando") }
    private var lista: [Anotado] { espera.filter { tabDe($0) == tab } }
    private var esperando: Int { espera.filter { tabDe($0) == "esperando" }.count }
    private var conHueco: Int { espera.filter { tabDe($0) == "esperando" && !(sugeridos[$0.id] ?? []).isEmpty }.count }

    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                Cabecera(titulo: "Lista de espera",
                         subtitulo: "\(esperando) esperando\(conHueco > 0 ? " · a \(conHueco) le\(conHueco == 1 ? "" : "s") sirve un hueco" : "")") {
                    Segmentado(opciones: [("esperando", "Esperando"), ("avisados", "Avisados"), ("vencidos", "Vencidos")], elegido: $tab)
                }
                Cuerpo {
                    if tab == "esperando" { ForEach(ofertas) { o in tarjetaOferta(o) } }
                    if cargando && espera.isEmpty {
                        ForEach(0..<3, id: \.self) { _ in Esqueleto(alto: 120, radio: 20) }
                    } else if lista.isEmpty {
                        VacioQueEnsena(rayas: tab == "esperando" ? "Nadie esperando" : "Nada por acá",
                                       titulo: tab == "esperando" ? "Nadie está esperando un lugar" : (tab == "avisados" ? "Todavía no avisaste a nadie" : "No hay pedidos vencidos"),
                                       texto: "Cuando un día está lleno, en tu página aparece «Avisame si se libera». Si alguien cancela, se lo ofrecemos solos.") { EmptyView() }
                    } else {
                        ForEach(Array(lista.enumerated()), id: \.element.id) { i, a in tarjeta(a).entrada(i) }
                    }
                }
            }
        }
        .scrollIndicators(.hidden)
        .background(C.fondo)
        .toolbar(.hidden, for: .navigationBar)
        .refreshable { await cargar() }
        .task(id: estado.version) { await cargar() }
        .confirmationDialog("¿Sacar a \(borrar?.nombre ?? "") de la lista?", isPresented: Binding(get: { borrar != nil }, set: { if !$0 { borrar = nil } }), titleVisibility: .visible) {
            Button("Sacar de la lista", role: .destructive) { if let b = borrar { Task { await sacar(b) } } }
        }
    }

    // MARK: Una persona

    private func rango(_ a: Anotado) -> String {
        guard let d = Fecha.iso(a.desde), let h = Fecha.iso(a.hasta) else { return "" }
        if a.desde == a.hasta { return Formato.dia(d) }
        return "\(Formato.calendario.component(.day, from: d)) al \(Formato.dia(h))"
    }

    private func tarjeta(_ a: Anotado) -> some View {
        let match = sugeridos[a.id]?.first
        let quien = match?.profesionales.first.flatMap { id in profesionales.first { $0.id == id }?.nombre } ?? ""
        return VStack(alignment: .leading, spacing: 12) {
            HStack(spacing: 12) {
                VStack(alignment: .leading, spacing: 2) {
                    Text(rango(a)).font(Tipo.lato(15, .black)).monospacedDigit().lineLimit(2).minimumScaleFactor(0.8)
                    Text(a.momentoTexto).font(Tipo.lato(12)).foregroundStyle(C.sub)
                }
                .frame(width: 64, alignment: .leading).padding(.trailing, 12)
                .overlay(alignment: .trailing) { Troquel() }
                VStack(alignment: .leading, spacing: 2) {
                    Text(a.nombre).font(Tipo.lato(17, .bold))
                    Text("\(a.servicio) · con \(a.profesional ?? "cualquiera")").font(Tipo.lato(13)).foregroundStyle(C.sub)
                    Text(a.avisadoEn != nil ? textoAvisado(a) : "Espera desde el \(Formato.dia(a.creadoEn))").font(Tipo.lato(12)).foregroundStyle(C.sub)
                }
                Spacer(minLength: 0)
                Menu {
                    if !a.telefono.isEmpty, let u = URL(string: "tel:\(a.telefono.filter { "+0123456789".contains($0) })") { Link(destination: u) { Label("Llamar", systemImage: "phone") } }
                    Button("Cargarle un turno", systemImage: "plus") {
                        nav.nuevo = NuevoTurnoInicio(profesionalId: a.profesionalId, servicioId: a.servicioId, cliente: ClienteDelTurno(id: nil, nombre: a.nombre, telefono: a.telefono, email: a.email), esperaId: a.id)
                    }
                    Button("Sacar de la lista", systemImage: "trash", role: .destructive) { borrar = a }
                } label: {
                    Image(systemName: "ellipsis").font(.system(size: 17, weight: .bold)).foregroundStyle(C.sub).frame(width: 44, height: 44)
                }
                .accessibilityLabel("Más acciones para \(a.nombre)")
            }
            if let m = match, tab == "esperando" {
                HStack(spacing: 8) {
                    PastillaPapel(radio: 8) {
                        Text("\(Formato.diaISO(m.inicio) == hoy ? "HOY" : Formato.dia(m.inicio).uppercased()) \(m.hora)\(quien.isEmpty ? "" : " · \(quien.uppercased())")")
                            .font(Tipo.lato(13, .bold)).tracking(0.8).monospacedDigit()
                    }
                    .padding(.leading, -4)
                    Spacer(minLength: 0)
                    Button("Avisar") { Task { await avisar(a, m) } }.buttonStyle(BotonMentaChico())
                        .disabled(estado.soloLectura)
                }
                .padding(.leading, 12).padding([.vertical, .trailing], 8)
                .hueco(14)
            }
        }
        .padding(14).tarjeta()
    }

    private func textoAvisado(_ a: Anotado) -> String {
        let r = ["tomada": "lo tomó", "paso": "dijo que no le sirve", "vencida": "no respondió", "cancelada": "se cerró la oferta", "ofrecida": "tiene el link"][a.resultado ?? ""]
        return "Avisada el \(Formato.dia(a.avisadoEn!))\(r.map { " · \($0)" } ?? "")"
    }

    // MARK: Ofertas en curso

    private func tarjetaOferta(_ o: Oferta) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Image(systemName: "megaphone").foregroundStyle(C.mentaTexto)
                Text("Ofreciendo \(Formato.horaDe(o.inicio)) · \(o.profesional)").font(Tipo.lato(16, .bold))
            }
            if let a = o.actual {
                HStack {
                    Text("A \(a.nombre)").font(Tipo.lato(15))
                    if let v = a.venceEn { Text("· vence a las \(Formato.horaDe(v))").font(Tipo.lato(13)).foregroundStyle(C.sub) }
                }
            }
            Text(o.sigue.map { "SIGUE: \($0.nombre.uppercased())" } ?? "ES LA ÚLTIMA").rotulo()
            HStack(spacing: 8) {
                Button("Frenar") { Task { await ofertaAccion(o, "frenar") } }.buttonStyle(BotonContorno(alto: 44, tam: 15))
                Button("Cargarlo yo") { Task { await ofertaAccion(o, "cargar") } }.buttonStyle(BotonContorno(alto: 44, tam: 15))
            }
        }
        .padding(14)
        .hueco(20, quieto: false)
    }

    // MARK: Datos

    private struct Espera: Decodable { var espera: [Anotado] }
    private struct Ofertas: Decodable { var ofertas: [Oferta] }
    private struct Libres: Decodable { var horarios: [HorarioLibre] }
    private struct Pros: Decodable { var profesionales: [Profesional] }

    private func cargar() async {
        let e: Espera? = try? await estado.api.get("/v1/panel/espera", query: [URLQueryItem(name: "todos", value: "1")])
        let o: Ofertas? = try? await estado.api.get("/v1/panel/espera/ofertas")
        let p: Pros? = try? await estado.api.get("/v1/panel/profesionales")
        let (ee, oo, pp) = (e, o, p)
        espera = ee?.espera ?? espera
        ofertas = (oo?.ofertas ?? []).filter { $0.estado == "ofreciendo" }
        profesionales = pp?.profesionales ?? profesionales
        cargando = false
        // Los primeros horarios libres dentro de lo que pidió cada una (como la web: 7 días, hasta 6).
        for a in espera.filter({ tabDe($0) == "esperando" }).prefix(30) where sugeridos[a.id] == nil {
            sugeridos[a.id] = await libresPara(a)
        }
    }

    private func libresPara(_ a: Anotado) async -> [HorarioLibre] {
        guard var d = Fecha.iso(max(a.desde, hoy)), let fin = Fecha.iso(a.hasta) else { return [] }
        for _ in 0..<7 where d <= fin {
            var q = [URLQueryItem(name: "servicio", value: a.servicioId), URLQueryItem(name: "fecha", value: Formato.diaISO(d))]
            if let p = a.profesionalId { q.append(URLQueryItem(name: "profesional", value: p)) }
            if let r: Libres = try? await estado.api.get("/v1/panel/libres", query: q), !r.horarios.isEmpty {
                let m = r.horarios.filter { h in
                    let min = Formato.minutosDelDia(h.inicio)
                    return a.momento == "manana" ? min < 13 * 60 : a.momento == "tarde" ? min >= 13 * 60 : true
                }
                if !m.isEmpty { return Array(m.prefix(6)) }
            }
            d = Formato.calendario.date(byAdding: .day, value: 1, to: d) ?? fin.addingTimeInterval(86400)
        }
        return []
    }

    private struct Aviso: Encodable { var inicio: Date?; var nota: String }

    private func avisar(_ a: Anotado, _ h: HorarioLibre) async {
        do {
            try await estado.api.hacer("POST", "/v1/panel/espera/\(a.id)/avisar", Aviso(inicio: h.inicio, nota: ""))
            Haptico.exito()
            estado.avisar("Le avisamos a \(a.nombre.split(separator: " ").first ?? "")", texto: "\(Formato.dia(h.inicio)) a las \(h.hora)")
            await cargar()
        } catch { estado.avisar(error) }
    }

    private func sacar(_ a: Anotado) async {
        do {
            try await estado.api.hacer("DELETE", "/v1/panel/espera/\(a.id)")
            espera.removeAll { $0.id == a.id }
        } catch { estado.avisar(error) }
    }

    private struct Cargada: Decodable { var oferta: Oferta }

    private func ofertaAccion(_ o: Oferta, _ que: String) async {
        do {
            if que == "cargar" {
                let r: Cargada = try await estado.api.enviar("POST", "/v1/panel/espera/ofertas/\(o.id)/cargar", [String: String]())
                nav.nuevo = NuevoTurnoInicio(profesionalId: r.oferta.profesionalId, servicioId: r.oferta.servicioId, inicio: r.oferta.inicio)
            } else {
                _ = try await estado.api.pedir("POST", "/v1/panel/espera/ofertas/\(o.id)/frenar")
                estado.avisar("Dejamos de ofrecerlo", texto: "Quien tenía el link vuelve a esperar.")
            }
            await cargar()
        } catch { estado.avisar(error) }
    }
}
