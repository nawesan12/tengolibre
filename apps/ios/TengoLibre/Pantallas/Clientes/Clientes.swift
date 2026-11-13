import SwiftUI
import TipKit

struct EtiquetaConUso: Codable, Identifiable, Equatable, Hashable { var id: String; var nombre: String; var color: String; var clientes: Int }

struct PlanMembresia: Codable, Identifiable, Equatable, Hashable {
    var id: String
    var nombre: String
    var descripcion: String
    var precio: Int
    var visitas: Int
    var duracionDias: Int
    var prioridadDias: Int
    var beneficios: [String]
    var topeSocios: Int?
    var servicios: [String]
    var activo: Bool
    var orden: Int
}

struct Socio: Codable, Identifiable, Equatable, Hashable {
    var id: String
    var planId: String
    var plan: String
    var visitasDelPlan: Int
    var clienteId: String
    var cliente: String
    var telefono: String
    var desde: String
    var hasta: String
    var estado: String
    var quedan: Int
    var nota: String
}

/// Lo que se apila dentro de la pestaña Clientes.
enum DestinoClientes: Hashable { case ficha(String) }

/// 14 Clientes: buscar, filtros (Todas · Frecuentes · Con penalidad · Nuevas · por etiqueta), orden
/// y la lista con iniciales, última visita y la etiqueta. «+» para cargar una clienta nueva.
struct Clientes: View {
    @Environment(Estado.self) private var estado
    @Environment(Navegacion.self) private var nav
    @State private var clientes: [Cliente] = []
    @State private var total = 0
    @State private var pagina = 1
    @State private var etiquetas: [EtiquetaConUso] = []
    @State private var buscar = ""
    @State private var filtro = "todas"
    @State private var orden = "nombre"
    @State private var cargando = true
    @State private var nueva = false

    var body: some View {
        ScrollView {
            LazyVStack(spacing: 0) {
                Cabecera(titulo: "Clientes", subtitulo: "\(Formato.numero(total)) en total",
                         acciones: [AccionCabecera(simbolo: "plus", etiqueta: "Clienta nueva") { nueva = true }]) {
                    HStack(spacing: 8) {
                        Image(systemName: "magnifyingglass").font(.system(size: 18)).foregroundStyle(.white.opacity(0.7))
                        TextField("", text: $buscar, prompt: Text("Buscá por nombre o teléfono").foregroundStyle(.white.opacity(0.6)))
                            .font(Tipo.lato(16)).foregroundStyle(.white).submitLabel(.search)
                    }
                    .padding(.horizontal, 14).frame(height: 48)
                    .background(.white.opacity(0.12), in: RoundedRectangle(cornerRadius: 14, style: .continuous))
                    ScrollView(.horizontal, showsIndicators: false) {
                        HStack(spacing: 8) {
                            ForEach(filtros, id: \.0) { f in
                                Button { filtro = f.0 } label: {
                                    Text(f.1).font(Tipo.lato(14, .bold)).padding(.horizontal, 14).frame(height: 36)
                                        .foregroundStyle(filtro == f.0 ? C.petroleoHondo : .white)
                                        .background(filtro == f.0 ? C.menta : .white.opacity(0.1), in: Capsule())
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }
                }
                Cuerpo {
                    HStack {
                        Text("\(Formato.numero(total)) \(total == 1 ? "clienta" : "clientas")").rotulo()
                        Spacer()
                        Menu {
                            Picker("Ordenar", selection: $orden) {
                                Text("Por nombre").tag("nombre")
                                Text("Más recientes").tag("reciente")
                                Text("Última visita").tag("ultima")
                                Text("Más gastaron").tag("gastado")
                                Text("Más visitas").tag("visitas")
                            }
                        } label: {
                            Label("Ordenar", systemImage: "arrow.up.arrow.down").font(Tipo.lato(14, .bold)).foregroundStyle(C.mentaTexto).frame(minHeight: 44)
                        }
                    }
                    if cargando && clientes.isEmpty {
                        ForEach(0..<6, id: \.self) { _ in Esqueleto(alto: 68) }
                    } else if clientes.isEmpty {
                        VacioQueEnsena(rayas: "Sin resultados", titulo: buscar.isEmpty ? "Todavía no tenés clientas" : "No encontramos a nadie",
                                       texto: buscar.isEmpty ? "Se suman solas cuando reservan por tu página o les cargás un turno." : "Probá con otra parte del nombre o del teléfono.") {
                            Button { nueva = true } label: { Label("Cargar una clienta", systemImage: "plus") }.buttonStyle(BotonContorno())
                        }
                    } else {
                        VStack(spacing: 0) {
                            ForEach(Array(clientes.enumerated()), id: \.element.id) { i, c in
                                if i > 0 { Rectangle().fill(C.linea).frame(height: 1).padding(.leading, 72) }
                                NavigationLink(value: DestinoClientes.ficha(c.id)) { fila(c) }.buttonStyle(.plain).entrada(i)
                                    .onAppear { if c.id == clientes.last?.id && clientes.count < total { Task { await cargar(mas: true) } } }
                            }
                        }
                        .tarjeta()
                    }
                }
            }
        }
        .scrollIndicators(.hidden)
        .scrollDismissesKeyboard(.immediately)
        .background(C.fondo)
        .toolbar(.hidden, for: .navigationBar)
        .refreshable { await cargar() }
        .task(id: "\(estado.version)-\(filtro)-\(orden)") { await cargar() }
        .task(id: buscar) {
            try? await Task.sleep(for: .milliseconds(300))
            if !Task.isCancelled { await cargar() }
        }
        .sheet(isPresented: $nueva) { ClientaNueva().presentationDetents([.medium]) }
    }

    private var filtros: [(String, String)] {
        [("todas", "Todas"), ("frecuentes", "Frecuentes"), ("penalizados", "Con penalidad"), ("nuevas", "Nuevas")] + etiquetas.map { ("e:\($0.id)", $0.nombre) }
    }

    private func fila(_ c: Cliente) -> some View {
        HStack(spacing: 12) {
            Iniciales(nombre: c.nombre, tam: 44)
            VStack(alignment: .leading, spacing: 2) {
                Text(c.nombre).font(Tipo.lato(16, .bold)).lineLimit(1)
                Text(detalle(c)).font(Tipo.lato(13)).foregroundStyle(C.sub).lineLimit(1)
            }
            Spacer(minLength: 0)
            if c.penalizado {
                PastillaEstado(texto: "Penalidad", simbolo: "exclamationmark.shield", fondo: C.ausenteFondo, tinta: C.ausenteTinta)
            } else if let e = c.etiquetas.first {
                Text(e.nombre).font(Tipo.lato(12, .bold)).padding(.horizontal, 10).frame(height: 24)
                    .foregroundStyle(C.tinta).background(Color(hexTexto: e.color)?.opacity(0.22) ?? C.chip, in: Capsule())
            }
            Image(systemName: "chevron.right").font(.system(size: 13, weight: .semibold)).foregroundStyle(C.sub)
        }
        .foregroundStyle(C.tinta).padding(.horizontal, 14).frame(minHeight: 68).contentShape(Rectangle())
    }

    private func detalle(_ c: Cliente) -> String {
        var p: [String] = []
        if let pr = c.proximoTurno, pr > Date() { p.append("Viene \(Formato.cuando(pr))") }
        else if let u = c.ultimaVisita { p.append("Vino \(Formato.hace(u))") }
        else { p.append("Nueva") }
        p.append(Formato.cantidad(c.visitas, "visita", "visitas"))
        return p.joined(separator: " · ")
    }

    private struct Lista: Decodable { var clientes: [Cliente]; var total: Int }
    private struct Etiquetas: Decodable { var etiquetas: [EtiquetaConUso] }

    private func cargar(mas: Bool = false) async {
        let p = mas ? pagina + 1 : 1
        var q = [URLQueryItem(name: "pagina", value: "\(p)"), URLQueryItem(name: "porPagina", value: "30")]
        switch filtro {
        case "frecuentes": q.append(URLQueryItem(name: "orden", value: "visitas"))
        case "nuevas": q.append(URLQueryItem(name: "orden", value: "reciente"))
        case "penalizados": q.append(URLQueryItem(name: "penalizados", value: "1"))
        default:
            if filtro.hasPrefix("e:") { q.append(URLQueryItem(name: "etiqueta", value: String(filtro.dropFirst(2)))) }
        }
        if filtro != "frecuentes" && filtro != "nuevas" { q.append(URLQueryItem(name: "orden", value: orden)) }
        let t = buscar.trimmingCharacters(in: .whitespaces)
        if !t.isEmpty { q.append(URLQueryItem(name: "q", value: t)) }
        do {
            let r: Lista = try await estado.api.get("/v1/panel/clientes", query: q)
            clientes = mas ? clientes + r.clientes : r.clientes
            total = r.total
            pagina = p
        } catch { if !mas { estado.avisar(error) } }
        cargando = false
        if etiquetas.isEmpty { etiquetas = (try? await estado.api.get("/v1/panel/etiquetas") as Etiquetas)?.etiquetas ?? [] }
    }
}

/// Cargar una clienta a mano (nombre, teléfono, email).
struct ClientaNueva: View {
    @Environment(Estado.self) private var estado
    @Environment(Navegacion.self) private var nav
    @Environment(\.dismiss) private var cerrar
    @State private var nombre = ""
    @State private var telefono = ""
    @State private var email = ""
    @State private var contactos = false

    var body: some View {
        NavigationStack {
            VStack(spacing: 12) {
                Campo(titulo: "Nombre", texto: $nombre)
                Campo(titulo: "Teléfono", texto: $telefono, teclado: .phonePad)
                Campo(titulo: "Email (opcional)", texto: $email, teclado: .emailAddress)
                Button { contactos = true } label: { Label("Traer de Contactos", systemImage: "person.crop.circle") }.buttonStyle(BotonTexto())
                Spacer()
            }
            .padding(16).background(C.hoja)
            .navigationTitle("Clienta nueva").navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancelar") { cerrar() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Guardar") { Task { await guardar() } }.disabled(nombre.trimmingCharacters(in: .whitespaces).count < 2)
                }
            }
            .sheet(isPresented: $contactos) { SelectorContacto { n, t, e in nombre = n; telefono = t; email = e }.ignoresSafeArea() }
        }
    }

    private func guardar() async {
        do {
            let c: Cliente = try await estado.api.enviar("POST", "/v1/panel/clientes", ["nombre": nombre, "telefono": telefono, "email": email])
            estado.version += 1
            cerrar()
            nav.pilaClientes.append(DestinoClientes.ficha(c.id))
        } catch { estado.avisar(error) }
    }
}

/// 15 Ficha de la clienta: Llamar · WhatsApp · Turno, próximo turno, etiquetas, penalidad,
/// membresía, notas del equipo e historial.
struct FichaCliente: View {
    var id: String
    @Environment(Estado.self) private var estado
    @Environment(Navegacion.self) private var nav
    @Environment(\.dismiss) private var volver
    @State private var ficha: Ficha?
    @State private var notas: [Nota] = []
    @State private var etiquetas: [EtiquetaConUso] = []
    @State private var penalidades: [Penalidad] = []
    @State private var socio: Socio?
    @State private var notaNueva = ""
    @State private var editar = false
    @State private var penalizar = false
    @State private var diasPenalidad = 30
    @State private var notaPenalidad = ""
    @State private var error = ""

    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                Cabecera(titulo: ficha?.nombre ?? "Clienta", subtitulo: subtitulo, volver: "Clientes",
                         acciones: ficha == nil ? [] : [AccionCabecera(simbolo: "pencil", etiqueta: "Editar") { editar = true }]) { EmptyView() }
                Cuerpo {
                    if let f = ficha { contenido(f) }
                    else if !error.isEmpty { ErrorConSalida(titulo: "No pudimos traer la ficha", texto: error) { Task { await cargar() } } }
                    else { ForEach(0..<4, id: \.self) { _ in Esqueleto(alto: 90) } }
                }
            }
        }
        .scrollIndicators(.hidden)
        .background(C.fondo)
        .toolbar(.hidden, for: .navigationBar)
        .refreshable { await cargar() }
        .task(id: estado.version) { await cargar() }
        .sheet(isPresented: $editar) {
            if let f = ficha {
                EditarClienta(ficha: f, puedeBorrar: !estado.esProfesional, listo: { await cargar() }) { anonima in
                    estado.avisar("Borramos sus datos", texto: anonima ? "Sus turnos pasados quedan como «Cliente borrado»." : "")
                    estado.version += 1
                    volver()
                }
            }
        }
        .sheet(isPresented: $penalizar) { hojaPenalidad.presentationDetents([.medium]) }
    }

    private var subtitulo: String {
        guard let f = ficha else { return "" }
        var p = [Formato.cantidad(f.visitas, "visita", "visitas")]
        if estado.vePrecios && f.gastado > 0 { p.append("\(Formato.pesos(f.gastado)) gastado") }
        if f.ausencias > 0 { p.append(Formato.cantidad(f.ausencias, "ausencia", "ausencias")) }
        return p.joined(separator: " · ")
    }

    private var tel: String { (ficha?.telefono ?? "").filter { "+0123456789".contains($0) } }

    @ViewBuilder private func contenido(_ f: Ficha) -> some View {
        HStack(spacing: 8) {
            botonContacto("Llamar", "phone", menta: false) { if let u = URL(string: "tel:\(tel)") { UIApplication.shared.open(u) } }.disabled(tel.isEmpty)
            botonContacto("WhatsApp", "message", menta: false) {
                var d = tel.filter(\.isNumber)
                if d.hasPrefix("0") { d.removeFirst() }
                if !d.hasPrefix("54") { d = "549" + d }
                if let u = URL(string: "https://wa.me/\(d)") { UIApplication.shared.open(u) }
            }.disabled(tel.isEmpty)
            botonContacto("Turno", "plus", menta: true) {
                nav.nuevo = NuevoTurnoInicio(cliente: ClienteDelTurno(id: f.id, nombre: f.nombre, telefono: f.telefono, email: f.email))
            }.disabled(estado.soloLectura || f.penalizado)
        }
        if let p = f.penalidad, p.vigente {
            HStack(alignment: .top, spacing: 12) {
                Image(systemName: "exclamationmark.shield").font(.system(size: 22)).foregroundStyle(C.ausenteTinta)
                VStack(alignment: .leading, spacing: 4) {
                    Text("Penalizada hasta el \(Formato.dia(p.hasta))").font(Tipo.lato(16, .bold))
                    Text(p.nota.isEmpty ? "No puede reservar por la web hasta esa fecha." : p.nota).font(Tipo.lato(14)).foregroundStyle(C.sub)
                    Button("Levantar la penalidad") { Task { await levantar(p) } }.buttonStyle(BotonTexto())
                }
            }
            .padding(16).frame(maxWidth: .infinity, alignment: .leading)
            .background(C.ausenteFondo, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
        }
        if let s = socio { membresia(s) }
        SugerenciaMembresia(ficha: f, conMembresia: socio != nil)
        visitas(f).entrada(1)
        if let pr = f.turnos.first(where: { $0.inicio > Date() && $0.estado != "cancelado" }) {
            Rotulo("PRÓXIMO TURNO")
            filaTurno(pr).tarjeta()
        }
        // Etiquetas
        VStack(alignment: .leading, spacing: 10) {
            Text("ETIQUETAS").rotulo()
            TipView(ConsejoEtiquetas()).tipBackground(C.fondo).tint(C.mentaTexto)
            FlujoEtiquetas(items: etiquetas.map { e in (e.id, e.nombre, f.etiquetas.contains { $0.id == e.id }) }) { id, puesta in
                Task { await etiqueta(id, quitar: puesta) }
            }
            if etiquetas.isEmpty { Text("Creá etiquetas desde Más › Etiquetas.").font(Tipo.lato(13)).foregroundStyle(C.sub) }
        }
        .padding(16).frame(maxWidth: .infinity, alignment: .leading).tarjeta()
        // Penalidad
        if f.penalidad?.vigente != true && !estado.esProfesional {
            Button { penalizar = true } label: {
                FilaLista(simbolo: "exclamationmark.shield", titulo: "Penalizar", detalle: "No va a poder reservar por la web")
            }
            .buttonStyle(.plain).tarjeta()
        }
        // Notas
        VStack(alignment: .leading, spacing: 10) {
            Text("NOTAS DEL EQUIPO").rotulo()
            ForEach(notas) { n in
                VStack(alignment: .leading, spacing: 2) {
                    Text(n.texto).font(Tipo.lato(15))
                    Text("\(n.autor) · \(Formato.hace(n.creadoEn))").font(Tipo.lato(12)).foregroundStyle(C.sub)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .contextMenu { Button("Borrar", systemImage: "trash", role: .destructive) { Task { await borrarNota(n) } } }
            }
            HStack(spacing: 8) {
                TextField("Sumá una nota (ej.: prefiere a la tarde)", text: $notaNueva, axis: .vertical).font(Tipo.lato(15)).lineLimit(1...4)
                    .padding(.horizontal, 12).padding(.vertical, 10).background(C.campo, in: RoundedRectangle(cornerRadius: 12))
                Button("Sumar") { Task { await sumarNota() } }.buttonStyle(BotonMentaChico())
                    .disabled(notaNueva.trimmingCharacters(in: .whitespaces).isEmpty)
            }
        }
        .padding(16).frame(maxWidth: .infinity, alignment: .leading).tarjeta()
        // Historial
        let pasados = f.turnos.filter { $0.inicio <= Date() || $0.estado == "cancelado" }
        if !pasados.isEmpty {
            Rotulo("HISTORIAL")
            VStack(spacing: 0) {
                ForEach(Array(pasados.enumerated()), id: \.element.id) { i, t in
                    if i > 0 { Rectangle().fill(C.linea).frame(height: 1) }
                    filaTurno(t)
                }
            }
            .tarjeta()
        }
        VStack(alignment: .leading, spacing: 4) {
            if !f.email.isEmpty { Label(f.email, systemImage: "envelope").font(Tipo.lato(14)) }
            Label("Clienta desde el \(Formato.dia(f.creadoEn))", systemImage: "calendar").font(Tipo.lato(14))
        }
        .foregroundStyle(C.sub).frame(maxWidth: .infinity, alignment: .leading).padding(.horizontal, 4)
    }

    /// Las visitas de los últimos 12 meses (atendidas), con lo que gastó y cada cuánto viene.
    @ViewBuilder private func visitas(_ f: Ficha) -> some View {
        let cal = Formato.calendario
        let atendidos = f.turnos.filter { $0.estado == "atendido" }
        if !atendidos.isEmpty {
            let hoy = cal.date(from: cal.dateComponents([.year, .month], from: Date())) ?? Date()
            let meses = (0..<12).reversed().compactMap { cal.date(byAdding: .month, value: -$0, to: hoy) }.map { m in
                GraficoVisitas.Mes(mes: m, visitas: atendidos.filter { cal.isDate($0.inicio, equalTo: m, toGranularity: .month) }.count)
            }
            let fechas = atendidos.map(\.inicio).sorted()
            let cada: Int? = fechas.count > 1 ? Int((fechas.last!.timeIntervalSince(fechas.first!) / 86400 / Double(fechas.count - 1)).rounded()) : nil
            VStack(alignment: .leading, spacing: 12) {
                HStack(alignment: .firstTextBaseline) {
                    Text("VISITAS · ÚLTIMO AÑO").rotulo()
                    Spacer()
                    if let cada { Text("viene cada \(cada) días").font(Tipo.lato(12, .bold)).foregroundStyle(C.mentaTexto) }
                }
                GraficoVisitas(meses: meses)
                HStack(spacing: 18) {
                    dato("\(f.visitas)", "visitas")
                    if estado.vePrecios { dato(Formato.pesosCortos(f.gastado), "gastó en total") }
                    if f.visitas > 0 && estado.vePrecios { dato(Formato.pesosCortos(f.gastado / max(1, f.visitas)), "por visita") }
                }
            }
            .foregroundStyle(C.tinta).padding(16).frame(maxWidth: .infinity, alignment: .leading).tarjeta()
        }
    }

    private func dato(_ v: String, _ t: String) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            Text(v).font(Tipo.lato(18, .black)).monospacedDigit()
            Text(t).font(Tipo.lato(11)).foregroundStyle(C.sub)
        }
    }

    private func botonContacto(_ t: String, _ s: String, menta: Bool, _ a: @escaping () -> Void) -> some View {
        Button(action: a) {
            VStack(spacing: 4) {
                Image(systemName: s).font(.system(size: 20, weight: .semibold))
                Text(t).font(Tipo.lato(13, .bold))
            }
            .frame(maxWidth: .infinity, minHeight: 64)
            .foregroundStyle(menta ? C.petroleoHondo : C.tinta)
            .background(menta ? C.menta : C.tarjeta, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
            .contorno(18, color: menta ? .clear : C.linea)
        }
        .buttonStyle(.plain)
    }

    private func membresia(_ s: Socio) -> some View {
        let usadas = max(0, s.visitasDelPlan - s.quedan)
        return HStack(spacing: 0) {
            VStack(spacing: 0) {
                Text("\(usadas)/\(s.visitasDelPlan)").font(Tipo.lato(22, .black)).monospacedDigit()
                Text("USADAS").font(Tipo.lato(10, .black)).tracking(0.8).opacity(0.7)
            }
            .frame(width: 88)
            Troquel(color: .white.opacity(0.3))
            VStack(alignment: .leading, spacing: 4) {
                Text(s.plan).font(Tipo.lato(17, .bold))
                Text("Vence el \(Fecha.iso(s.hasta).map { Formato.dia($0) } ?? s.hasta) · quedan \(s.quedan)").font(Tipo.lato(13)).opacity(0.8)
                GeometryReader { g in
                    ZStack(alignment: .leading) {
                        Capsule().fill(.white.opacity(0.18))
                        Capsule().fill(C.menta).frame(width: g.size.width * CGFloat(usadas) / CGFloat(max(1, s.visitasDelPlan)))
                    }
                }
                .frame(height: 6)
            }
            .padding(14)
        }
        .foregroundStyle(.white).frame(minHeight: 88)
        .background(C.bloquePetroleo, in: TalonShape(troquelX: 88, radio: 20, muesca: 9))
    }

    private func filaTurno(_ t: TurnoDeCliente) -> some View {
        let est = EstadoTurno(rawValue: t.estado) ?? .pendiente
        return HStack(spacing: 12) {
            VStack(alignment: .leading, spacing: 2) {
                Text(Formato.diaYHora(t.inicio)).font(Tipo.lato(15, .bold))
                Text("\(t.servicio) · \(t.profesional)").font(Tipo.lato(13)).foregroundStyle(C.sub)
            }
            Spacer(minLength: 0)
            if estado.vePrecios { Text(Formato.pesos(t.precio)).font(Tipo.lato(14)).monospacedDigit().foregroundStyle(C.sub) }
            PastillaEstado(texto: est.texto, simbolo: est.simbolo, fondo: est.fondo, tinta: est.tinta)
        }
        .foregroundStyle(C.tinta).padding(.horizontal, 14).frame(minHeight: 60)
        .contentShape(Rectangle())
        .onTapGesture { Task { if let x: Turno = try? await estado.api.get("/v1/panel/turnos/\(t.id)") { nav.turno = x } } }
    }

    private var hojaPenalidad: some View {
        NavigationStack {
            Form {
                Stepper("\(diasPenalidad) días", value: $diasPenalidad, in: 1...365, step: diasPenalidad >= 30 ? 15 : 5)
                TextField("Motivo (opcional)", text: $notaPenalidad, axis: .vertical)
                Text("Mientras dure, no puede reservar por tu página. Vos le podés seguir cargando turnos.").font(Tipo.lato(13)).foregroundStyle(C.sub)
            }
            .navigationTitle("Penalizar").navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancelar") { penalizar = false } }
                ToolbarItem(placement: .confirmationAction) { Button("Penalizar") { Task { await penalizarYa() } } }
            }
        }
    }

    // MARK: Datos

    private struct Notas: Decodable { var notas: [Nota] }
    private struct Etiquetas: Decodable { var etiquetas: [EtiquetaConUso] }
    private struct Penalidades: Decodable { var penalidades: [Penalidad] }
    private struct Socios: Decodable { var socios: [Socio] }

    private func cargar() async {
        do {
            let f: Ficha = try await estado.api.get("/v1/panel/clientes/\(id)")
            let n: Notas? = try? await estado.api.get("/v1/panel/clientes/\(id)/notas")
            let e: Etiquetas? = try? await estado.api.get("/v1/panel/etiquetas")
            let p: Penalidades? = try? await estado.api.get("/v1/panel/clientes/\(id)/penalidades")
            let s: Socios? = try? await estado.api.get("/v1/panel/membresias/socios", query: [URLQueryItem(name: "estado", value: "activa")])
            ficha = f
            notas = n?.notas ?? []
            etiquetas = e?.etiquetas ?? []
            penalidades = p?.penalidades ?? []
            socio = s?.socios.first { $0.clienteId == id }
            error = ""
        } catch let e as ErrorApi { error = e.detalle } catch { self.error = "Probá de nuevo." }
    }

    private func etiqueta(_ eid: String, quitar: Bool) async {
        do {
            if quitar { try await estado.api.hacer("DELETE", "/v1/panel/clientes/\(id)/etiquetas/\(eid)") }
            else { try await estado.api.hacer("PUT", "/v1/panel/clientes/\(id)/etiquetas/\(eid)", [String: String]()) }
            Haptico.seleccion()
            await cargar()
        } catch { estado.avisar(error) }
    }

    private func sumarNota() async {
        do {
            try await estado.api.hacer("POST", "/v1/panel/clientes/\(id)/notas", ["texto": notaNueva])
            notaNueva = ""
            await cargar()
        } catch { estado.avisar(error) }
    }

    private func borrarNota(_ n: Nota) async {
        do { try await estado.api.hacer("DELETE", "/v1/panel/clientes/\(id)/notas/\(n.id)"); notas.removeAll { $0.id == n.id } } catch { estado.avisar(error) }
    }

    private struct Pena: Encodable { var dias: Int; var nota: String }

    private func penalizarYa() async {
        do {
            try await estado.api.hacer("POST", "/v1/panel/clientes/\(id)/penalidades", Pena(dias: diasPenalidad, nota: notaPenalidad))
            penalizar = false
            Haptico.exito()
            await cargar()
        } catch { estado.avisar(error) }
    }

    private func levantar(_ p: Penalidad) async {
        do { try await estado.api.hacer("POST", "/v1/panel/penalidades/\(p.id)/levantar"); await cargar() } catch { estado.avisar(error) }
    }
}

/// Las etiquetas como chips que se prenden y apagan, en renglones que se acomodan solos.
struct FlujoEtiquetas: View {
    var items: [(String, String, Bool)]
    var tocar: (String, Bool) -> Void
    var body: some View {
        Flujo(espacio: 8) {
            ForEach(items, id: \.0) { it in
                Button { tocar(it.0, it.2) } label: { Chip(texto: it.1, simbolo: it.2 ? "checkmark" : "plus", elegido: it.2) }
                    .buttonStyle(.plain)
            }
        }
    }
}

/// Un layout que pone las vistas en fila y salta de renglón cuando no entran.
struct Flujo: Layout {
    var espacio: CGFloat = 8
    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let ancho = proposal.width ?? .infinity
        var x: CGFloat = 0, y: CGFloat = 0, alto: CGFloat = 0, maxX: CGFloat = 0
        for s in subviews {
            let t = s.sizeThatFits(.unspecified)
            if x > 0 && x + t.width > ancho { x = 0; y += alto + espacio; alto = 0 }
            x += t.width + espacio
            maxX = max(maxX, x - espacio)
            alto = max(alto, t.height)
        }
        return CGSize(width: proposal.width ?? maxX, height: y + alto)
    }
    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        var x = bounds.minX, y = bounds.minY, alto: CGFloat = 0
        for s in subviews {
            let t = s.sizeThatFits(.unspecified)
            if x > bounds.minX && x + t.width > bounds.maxX { x = bounds.minX; y += alto + espacio; alto = 0 }
            s.place(at: CGPoint(x: x, y: y), proposal: ProposedViewSize(t))
            x += t.width + espacio
            alto = max(alto, t.height)
        }
    }
}

/// Corregir los datos de la clienta y, si lo pide, borrarlos (sus derechos, Ley 25.326). Con turnos
/// pasados la API la deja anónima; con turnos por venir no deja: primero se cancelan.
struct EditarClienta: View {
    var ficha: Ficha
    var puedeBorrar: Bool
    var listo: () async -> Void
    /// Después de borrar: true si quedó anónima (tenía turnos pasados).
    var borrada: (Bool) -> Void
    @Environment(Estado.self) private var estado
    @Environment(\.dismiss) private var cerrar
    @State private var nombre = ""
    @State private var telefono = ""
    @State private var email = ""
    @State private var confirmarBorrar = false

    private struct Resultado: Decodable { var resultado: String }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 12) {
                    Campo(titulo: "Nombre", texto: $nombre)
                    Campo(titulo: "Teléfono", texto: $telefono, ayuda: "Con el que reserva: si lo cambiás, reserva con el nuevo.", teclado: .phonePad)
                    Campo(titulo: "Email", texto: $email, teclado: .emailAddress)
                    if puedeBorrar {
                        VStack(alignment: .leading, spacing: 8) {
                            Text("BORRAR SUS DATOS").rotulo()
                            Text("Si te lo pide, borramos su nombre, teléfono, email, notas y etiquetas. Sus turnos pasados quedan, sin datos.")
                                .font(Tipo.lato(13)).foregroundStyle(C.sub).fixedSize(horizontal: false, vertical: true)
                            Button(role: .destructive) { confirmarBorrar = true } label: { Label("Borrar sus datos", systemImage: "trash") }
                                .buttonStyle(BotonContorno(alto: 48, tam: 15)).foregroundStyle(C.error)
                                .confirmationDialog("¿Borrar los datos de \(ficha.nombre)?", isPresented: $confirmarBorrar, titleVisibility: .visible) {
                                    Button("Borrar", role: .destructive) { Task { await borrar() } }
                                } message: { Text("No se puede deshacer.") }
                        }
                        .padding(.top, 12).frame(maxWidth: .infinity, alignment: .leading)
                    }
                }
                .padding(16)
            }
            .scrollDismissesKeyboard(.interactively)
            .background(C.hoja)
            .navigationTitle("Editar").navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancelar") { cerrar() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Guardar") {
                        Task {
                            do {
                                try await estado.api.hacer("PATCH", "/v1/panel/clientes/\(ficha.id)", ["nombre": nombre, "telefono": telefono, "email": email])
                                await listo()
                                cerrar()
                            } catch { estado.avisar(error) }
                        }
                    }
                }
            }
            .onAppear { nombre = ficha.nombre; telefono = ficha.telefono; email = ficha.email }
        }
        .presentationDetents([.large])
    }

    private func borrar() async {
        do {
            let r: Resultado = try await estado.api.borrar("/v1/panel/clientes/\(ficha.id)")
            cerrar()
            borrada(r.resultado == "anonimizado")
        } catch { estado.avisar(error) }
    }
}
