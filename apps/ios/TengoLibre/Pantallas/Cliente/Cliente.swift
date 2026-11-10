import MapKit
import SwiftUI

// Modo clienta (handoff › Cliente): Turnos · Lugares · Cuenta, con la API pública, como la web
// (/[slug]/reservar, /turno/[codigo] y /explorar). Los turnos que reserva o abre se guardan en el
// teléfono por su código.

struct NegocioPublico: Codable, Equatable, Hashable {
    var id: String; var slug: String; var nombre: String; var descripcion: String; var direccion: String; var ciudad: String
    var telefono: String; var whatsapp: String; var instagram: String; var logo: String?
}

struct TurnoCli: Codable, Identifiable, Equatable, Hashable {
    struct Pago: Codable, Equatable, Hashable { var monto: Int; var modo: String; var url: String? }
    var id: String; var codigo: String; var servicioId: String?; var servicio: String; var profesionalId: String?; var profesional: String
    var inicio: Date; var fin: Date; var precio: Int; var estado: String; var venceEn: Date?
    var negocio: NegocioPublico
    var puedeCancelar: Bool?
    var cancelarHastaHoras: Int?
    var pago: Pago?
    var confirmadoEn: Date?
    var activo: Bool { ["pendiente_pago", "pendiente", "confirmado"].contains(estado) && fin > Date() }
}

struct PaginaPublica: Codable {
    struct Serv: Codable, Identifiable, Hashable { var id: String; var categoriaId: String?; var nombre: String; var descripcion: String; var duracion: Int; var precio: Int; var foto: String?; var orden: Int; var activo: Bool }
    struct Pro: Codable, Identifiable, Hashable { var id: String; var nombre: String; var especialidad: String; var foto: String?; var servicios: [String] }
    struct WebP: Codable { var portada: String?; var colorPrimario: String; var acerca: String; var rubro: String }
    struct Resumen: Codable { var promedio: Double; var cantidad: Int }
    var negocio: NegocioPublico
    var servicios: [Serv]
    var profesionales: [Pro]
    var diasVista: Int
    var politicaCancelacion: String
    var web: WebP
    var resenas: Resumen
}

struct FichaDirectorio: Codable, Identifiable, Hashable {
    var slug: String; var nombre: String; var descripcion: String; var logo: String?; var portada: String?; var ciudad: String; var provincia: String
    var rubro: String; var servicios: [String]; var desde: Int?
    var id: String { slug }
}

/// Los códigos de los turnos de la clienta, en el teléfono.
enum MisTurnos {
    static var codigos: [String] {
        get { UserDefaults.standard.stringArray(forKey: "misTurnos") ?? [] }
        set { UserDefaults.standard.set(Array(Set(newValue)).sorted(), forKey: "misTurnos") }
    }
    static func sumar(_ c: String) { codigos = codigos + [c.uppercased()] }
    static func sacar(_ c: String) { codigos = codigos.filter { $0 != c.uppercased() } }
}

enum PestanaCliente: String, CaseIterable { case turnos, lugares, cuenta }

struct PrincipalCliente: View {
    @Environment(Estado.self) private var estado
    @State private var pestana = PestanaCliente.turnos
    @State private var turnos: [TurnoCli] = []
    @State private var cargando = true
    @State private var abierto: TurnoCli?
    @State private var reservarEn: String?

    var body: some View {
        TabView(selection: $pestana) {
            NavigationStack { TurnosCliente(turnos: turnos, cargando: cargando, abrir: { abierto = $0 }, reservar: { reservarEn = $0 }, recargar: cargar) }
                .toolbar(.hidden, for: .tabBar).tag(PestanaCliente.turnos)
            NavigationStack { Lugares(reservar: { reservarEn = $0 }) }
                .toolbar(.hidden, for: .tabBar).tag(PestanaCliente.lugares)
            NavigationStack { CuentaCliente() }
                .toolbar(.hidden, for: .tabBar).tag(PestanaCliente.cuenta)
        }
        .safeAreaInset(edge: .bottom, spacing: 0) { barra }
        .task { await cargar() }
        .sheet(item: $abierto) { t in TurnoClienteHoja(turno: t) { await cargar() } }
        .sheet(item: Binding(get: { reservarEn.map { IdTexto(id: $0) } }, set: { reservarEn = $0?.id })) { s in
            Reservar(slug: s.id) { codigo in
                MisTurnos.sumar(codigo)
                Task { await cargar() }
                pestana = .turnos
            }
        }
        // Los avisos de sus turnos al iPhone: se pide el permiso una vez y se anota cada turno activo.
        .task(id: "\(estado.tokenPush ?? "")-\(turnos.filter(\.activo).map(\.codigo).joined())") {
            await estado.pedirAvisosClienta()
            await estado.seguirTurnos(turnos.filter(\.activo).map(\.codigo))
        }
        .onChange(of: estado.version) { _, _ in Task { await cargar() } }
    }

    private var barra: some View {
        HStack(spacing: 0) {
            ForEach(PestanaCliente.allCases, id: \.self) { p in
                let on = p == pestana
                let (t, s) = [PestanaCliente.turnos: ("Turnos", "calendar"), .lugares: ("Lugares", "mappin.and.ellipse"), .cuenta: ("Cuenta", "person.crop.circle")][p]!
                Button { withAnimation(Movimiento.tocar) { pestana = p } } label: {
                    VStack(spacing: 2) {
                        ZStack {
                            if on { Capsule().fill(C.seleccion).frame(width: 56, height: 30) }
                            Image(systemName: s).font(.system(size: 20, weight: on ? .bold : .medium)).foregroundStyle(on ? C.seleccionTexto : C.sub)
                        }
                        .frame(height: 30)
                        Text(t).font(Tipo.lato(11, on ? .black : .bold)).foregroundStyle(on ? C.tinta : C.sub)
                    }
                    .frame(maxWidth: .infinity, minHeight: 49)
                    .overlay(alignment: .top) { if on { UnevenRoundedRectangle(bottomLeadingRadius: 6, bottomTrailingRadius: 6).fill(C.menta).frame(width: 40, height: 5) } }
                    .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .accessibilityLabel(t).accessibilityAddTraits(on ? .isSelected : [])
            }
        }
        .background(C.barra.background(.ultraThinMaterial).ignoresSafeArea(edges: .bottom))
        .overlay(alignment: .top) { Rectangle().fill(C.linea).frame(height: 1) }
    }

    private func cargar() async {
        var out: [TurnoCli] = []
        await withTaskGroup(of: TurnoCli?.self) { g in
            for c in MisTurnos.codigos { g.addTask { try? await estado.api.get("/v1/t/\(c)") } }
            for await t in g { if let t { out.append(t) } }
        }
        turnos = out.sorted { $0.inicio < $1.inicio }
        cargando = false
    }
}

// MARK: - Turnos

struct TurnosCliente: View {
    var turnos: [TurnoCli]
    var cargando: Bool
    var abrir: (TurnoCli) -> Void
    var reservar: (String) -> Void
    var recargar: () async -> Void
    @Environment(Estado.self) private var estado
    @State private var codigo = false
    @State private var textoCodigo = ""

    private var proximos: [TurnoCli] { turnos.filter(\.activo) }
    private var pasados: [TurnoCli] { turnos.filter { !$0.activo }.sorted { $0.inicio > $1.inicio } }
    private var lugares: [NegocioPublico] {
        var vistos = Set<String>(), out: [NegocioPublico] = []
        for t in turnos.sorted(by: { $0.inicio > $1.inicio }) where !vistos.contains(t.negocio.slug) { vistos.insert(t.negocio.slug); out.append(t.negocio) }
        return out
    }

    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                Cabecera(titulo: "Tus turnos", subtitulo: proximos.isEmpty ? "Nada reservado por ahora" : Formato.cantidad(proximos.count, "turno reservado", "turnos reservados"),
                         acciones: [AccionCabecera(simbolo: "plus", etiqueta: "Tengo un código") { codigo = true }])
                Cuerpo {
                    if cargando && turnos.isEmpty {
                        Esqueleto(alto: 160, radio: 22); Esqueleto(alto: 72)
                    } else if let t = proximos.first {
                        Text("TU PRÓXIMO TURNO").rotulo()
                        proximo(t)
                    } else {
                        VacioQueEnsena(rayas: "Todo libre", titulo: "No tenés turnos reservados", texto: "Buscá un lugar en Lugares o abrí el link del turno que te mandaron.") { EmptyView() }
                    }
                    if !lugares.isEmpty {
                        Rotulo("TUS LUGARES")
                        ScrollView(.horizontal, showsIndicators: false) {
                            HStack(spacing: 10) {
                                ForEach(lugares, id: \.slug) { n in
                                    Button { reservar(n.slug) } label: {
                                        VStack(alignment: .leading, spacing: 8) {
                                            Iniciales(nombre: n.nombre, tam: 40, fondo: C.menta, tinta: C.petroleoHondo, radio: 12)
                                            Text(n.nombre).font(Tipo.lato(15, .bold)).lineLimit(2)
                                            Text("Reservar de nuevo").font(Tipo.lato(12, .bold)).foregroundStyle(C.mentaTexto)
                                        }
                                        .foregroundStyle(C.tinta).padding(14).frame(width: 150, height: 130, alignment: .topLeading).tarjeta(18)
                                    }
                                    .buttonStyle(.plain)
                                }
                            }
                        }
                    }
                    if proximos.count > 1 {
                        Rotulo("DESPUÉS")
                        ForEach(proximos.dropFirst()) { t in fila(t) }
                    }
                    if !pasados.isEmpty {
                        Rotulo("ANTES")
                        ForEach(pasados.prefix(20)) { t in fila(t).opacity(0.7) }
                    }
                }
            }
        }
        .scrollIndicators(.hidden)
        .background(C.fondo)
        .toolbar(.hidden, for: .navigationBar)
        .refreshable { await recargar() }
        .alert("Agregar un turno", isPresented: $codigo) {
            TextField("Código del turno", text: $textoCodigo).textInputAutocapitalization(.characters)
            Button("Agregar") {
                let c = textoCodigo.trimmingCharacters(in: .whitespaces)
                    .components(separatedBy: "/").last ?? ""
                if !c.isEmpty { MisTurnos.sumar(c); Task { await recargar() } }
                textoCodigo = ""
            }
            Button("Cancelar", role: .cancel) {}
        } message: { Text("Está en el mail o el WhatsApp de confirmación (o pegá el link).") }
    }

    private func proximo(_ t: TurnoCli) -> some View {
        Button { abrir(t) } label: {
            VStack(alignment: .leading, spacing: 0) {
                HStack(spacing: 0) {
                    VStack(spacing: 2) {
                        Text(Formato.dia(t.inicio).uppercased()).font(Tipo.lato(12, .black)).tracking(0.6)
                        Text(Formato.horaDe(t.inicio)).font(Tipo.lato(26, .black)).monospacedDigit().tracking(-0.5)
                    }
                    .frame(width: 92)
                    Troquel(color: C.petroleoHondo.opacity(0.35))
                    VStack(alignment: .leading, spacing: 3) {
                        Text(t.negocio.nombre).font(Tipo.lato(20, .black)).lineLimit(2)
                        Text("\(t.servicio)\(t.profesional.isEmpty ? "" : " · \(t.profesional)")").font(Tipo.lato(14)).opacity(0.85)
                        if t.estado == "pendiente_pago" { Text("Falta pagar la seña").font(Tipo.lato(13, .bold)) }
                        else if t.estado == "confirmado" || t.confirmadoEn != nil { Label("Confirmado", systemImage: "checkmark.circle").font(Tipo.lato(13, .bold)) }
                    }
                    .padding(16).frame(maxWidth: .infinity, alignment: .leading)
                }
                .frame(minHeight: 110)
                .foregroundStyle(C.petroleoHondo)
                .background(C.menta, in: TalonShape(troquelX: 92, radio: 22, muesca: 10))
                HStack(spacing: 8) {
                    Button { abrir(t) } label: { Label("Mover", systemImage: "calendar.badge.clock") }.buttonStyle(BotonContorno(alto: 48, tam: 15))
                    Button {
                        let q = "\(t.negocio.direccion), \(t.negocio.ciudad)".addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? ""
                        if let u = URL(string: "http://maps.apple.com/?q=\(q)") { UIApplication.shared.open(u) }
                    } label: { Label("Cómo llegar", systemImage: "map") }
                    .buttonStyle(BotonContorno(alto: 48, tam: 15))
                    .disabled(t.negocio.direccion.isEmpty)
                }
                .padding(.top, 10)
            }
        }
        .buttonStyle(.plain)
    }

    private func fila(_ t: TurnoCli) -> some View {
        Button { abrir(t) } label: {
            HStack(spacing: 12) {
                VStack(spacing: 0) {
                    Text(Formato.horaDe(t.inicio)).font(Tipo.lato(17, .black)).monospacedDigit()
                    Text(Formato.dia(t.inicio)).font(Tipo.lato(11)).foregroundStyle(C.sub)
                }
                .frame(width: 64)
                Troquel()
                VStack(alignment: .leading, spacing: 2) {
                    Text(t.negocio.nombre).font(Tipo.lato(16, .bold))
                    Text(t.servicio).font(Tipo.lato(13)).foregroundStyle(C.sub)
                }
                Spacer()
                let e = EstadoTurno(rawValue: t.estado) ?? .pendiente
                PastillaEstado(texto: e.texto, simbolo: e.simbolo, fondo: e.fondo, tinta: e.tinta)
            }
            .foregroundStyle(C.tinta).padding(.trailing, 12).frame(minHeight: 72)
            .background(C.tarjeta, in: TalonShape(troquelX: 64, radio: 18, muesca: 8))
        }
        .buttonStyle(.plain)
    }
}

/// El turno de la clienta: confirmar, pagar la seña, mover, cancelar, cómo llegar y contacto.
struct TurnoClienteHoja: View {
    @State var turno: TurnoCli
    var listo: () async -> Void
    @Environment(Estado.self) private var estado
    @Environment(\.dismiss) private var cerrar
    @Environment(\.openURL) private var abrirURL
    @State private var mover = false
    @State private var cancelar = false

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(Formato.fechaLarga(turno.inicio).uppercased()).rotulo()
                        Text("\(Formato.horaDe(turno.inicio)) a \(Formato.horaDe(turno.fin))").font(Tipo.lato(34, .black)).monospacedDigit()
                        Text(turno.negocio.nombre).font(Tipo.lato(20, .bold))
                        Text("\(turno.servicio)\(turno.profesional.isEmpty ? "" : " con \(turno.profesional)") · \(Formato.pesos(turno.precio))").font(Tipo.lato(15)).foregroundStyle(C.sub)
                        let e = EstadoTurno(rawValue: turno.estado) ?? .pendiente
                        PastillaEstado(texto: e.texto, simbolo: e.simbolo, fondo: e.fondo, tinta: e.tinta, alto: 26).padding(.top, 4)
                    }
                    Text("Código \(turno.codigo)").font(.system(size: 14, design: .monospaced)).foregroundStyle(C.sub)
                    if turno.activo {
                        if turno.estado == "pendiente_pago" {
                            Button { Task { await pagar() } } label: { Label("Pagar la seña\(turno.pago.map { " · \(Formato.pesos($0.monto))" } ?? "")", systemImage: "creditcard") }
                                .buttonStyle(BotonMenta())
                            if let v = turno.venceEn { Text("Si no se paga antes de las \(Formato.horaDe(v)), el horario se libera.").font(Tipo.lato(13)).foregroundStyle(C.sub) }
                        } else if turno.estado == "pendiente" {
                            Button { Task { await post("confirmar", "¡Listo! Confirmaste el turno") } } label: { Label("Confirmar que voy", systemImage: "checkmark.circle") }
                                .buttonStyle(BotonMenta())
                        }
                        HStack(spacing: 8) {
                            Button { mover = true } label: { Label("Mover", systemImage: "calendar.badge.clock") }.buttonStyle(BotonContorno(alto: 48, tam: 15))
                            Button(role: .destructive) { cancelar = true } label: { Label("Cancelar", systemImage: "xmark") }.buttonStyle(BotonContorno(alto: 48, tam: 15))
                                .disabled(turno.puedeCancelar == false)
                        }
                        if turno.puedeCancelar == false, let h = turno.cancelarHastaHoras {
                            Text("Se puede cancelar hasta \(h) h antes. Escribile al lugar.").font(Tipo.lato(13)).foregroundStyle(C.sub)
                        }
                    }
                    Bloque(titulo: "El lugar") {
                        if !turno.negocio.direccion.isEmpty {
                            Button {
                                let q = "\(turno.negocio.direccion), \(turno.negocio.ciudad)".addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? ""
                                if let u = URL(string: "http://maps.apple.com/?q=\(q)") { abrirURL(u) }
                            } label: { Renglon(titulo: turno.negocio.direccion, detalle: turno.negocio.ciudad, simbolo: "map") }.buttonStyle(.plain)
                        }
                        if !turno.negocio.whatsapp.isEmpty {
                            Button {
                                var d = turno.negocio.whatsapp.filter(\.isNumber)
                                if d.hasPrefix("0") { d.removeFirst() }
                                if !d.hasPrefix("54") { d = "549" + d }
                                if let u = URL(string: "https://wa.me/\(d)") { abrirURL(u) }
                            } label: { Renglon(titulo: "Escribir por WhatsApp", simbolo: "message") }.buttonStyle(.plain)
                        }
                        if !turno.negocio.telefono.isEmpty, let u = URL(string: "tel:\(turno.negocio.telefono.filter { "+0123456789".contains($0) })") {
                            Link(destination: u) { Renglon(titulo: "Llamar", detalle: turno.negocio.telefono, simbolo: "phone") }.buttonStyle(.plain)
                        }
                    }
                    ShareLink(item: Web.base.appendingPathComponent("turno/\(turno.codigo)")) { Label("Compartir el turno", systemImage: "square.and.arrow.up") }
                        .buttonStyle(BotonTexto())
                    if !turno.activo {
                        Button("Sacarlo de la lista") { MisTurnos.sacar(turno.codigo); Task { await estado.dejarDeSeguir(turno.codigo); await listo() }; cerrar() }.buttonStyle(BotonTexto(color: C.error))
                    }
                }
                .foregroundStyle(C.tinta).padding(16)
            }
            .background(C.hoja)
            .navigationTitle("Tu turno").navigationBarTitleDisplayMode(.inline)
            .toolbar { ToolbarItem(placement: .confirmationAction) { Button("Listo") { cerrar() } } }
            .confirmationDialog("¿Cancelar el turno?", isPresented: $cancelar, titleVisibility: .visible) {
                Button("Cancelar el turno", role: .destructive) { Task { await post("cancelar", "Cancelamos el turno") } }
            }
            .sheet(isPresented: $mover) {
                MoverCliente(turno: turno) { t in turno = t; Task { await listo() } }
            }
        }
    }

    private struct Url: Decodable { var url: String }

    private func pagar() async {
        if let u = turno.pago?.url, let url = URL(string: u) { abrirURL(url); return }
        do {
            let r: Url = try await estado.api.enviar("POST", "/v1/t/\(turno.codigo)/pagar", [String: String]())
            if let url = URL(string: r.url) { abrirURL(url) }
        } catch { estado.avisar(error) }
    }

    private func post(_ que: String, _ ok: String) async {
        do {
            let t: TurnoCli = try await estado.api.enviar("POST", "/v1/t/\(turno.codigo)/\(que)", [String: String]())
            turno = t
            Haptico.exito()
            estado.avisar(ok)
            await listo()
        } catch { estado.avisar(error) }
    }
}

/// Mover el turno (la clienta): otro día y horario libre del mismo servicio.
struct MoverCliente: View {
    var turno: TurnoCli
    var listo: (TurnoCli) -> Void
    @Environment(Estado.self) private var estado
    @Environment(\.dismiss) private var cerrar
    @State private var dias: [DiaLibre] = []
    @State private var fecha = ""
    @State private var horarios: [HorarioLibre] = []

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    SelectorDias(dias: dias, fecha: $fecha)
                    GrillaHorarios(horarios: horarios) { h in Task { await mover(h) } }
                }
                .padding(16)
            }
            .background(C.hoja)
            .navigationTitle("Mover el turno").navigationBarTitleDisplayMode(.inline)
            .toolbar { ToolbarItem(placement: .cancellationAction) { Button("Cancelar") { cerrar() } } }
            .task { await cargarDias() }
            .onChange(of: fecha) { _, _ in Task { await cargarHorarios() } }
        }
    }

    private var filtro: [URLQueryItem] {
        var q = [URLQueryItem(name: "servicio", value: turno.servicioId ?? "")]
        if let p = turno.profesionalId { q.append(URLQueryItem(name: "profesional", value: p)) }
        return q
    }

    private struct Ds: Decodable { var dias: [DiaLibre] }
    private struct Hs: Decodable { var horarios: [HorarioLibre] }

    private func cargarDias() async {
        dias = (try? await estado.api.get("/v1/p/\(turno.negocio.slug)/dias", query: filtro) as Ds)?.dias ?? []
        fecha = dias.first { $0.libres > 0 }?.fecha ?? ""
    }

    private func cargarHorarios() async {
        guard !fecha.isEmpty else { horarios = []; return }
        horarios = (try? await estado.api.get("/v1/p/\(turno.negocio.slug)/libres", query: filtro + [URLQueryItem(name: "fecha", value: fecha)]) as Hs)?.horarios ?? []
    }

    private func mover(_ h: HorarioLibre) async {
        do {
            let t: TurnoCli = try await estado.api.enviar("POST", "/v1/t/\(turno.codigo)/reprogramar", ["inicio": h.inicio])
            Haptico.exito()
            estado.avisar("Movimos tu turno", texto: Formato.diaYHora(t.inicio))
            listo(t)
            cerrar()
        } catch { estado.avisar(error) }
    }
}

struct DiaLibre: Codable, Hashable { var fecha: String; var libres: Int }

struct SelectorDias: View {
    var dias: [DiaLibre]
    @Binding var fecha: String
    var body: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                ForEach(dias, id: \.fecha) { d in
                    let on = d.fecha == fecha, f = Fecha.iso(d.fecha) ?? Date()
                    Button { fecha = d.fecha; Haptico.seleccion() } label: {
                        VStack(spacing: 2) {
                            Text(String(Formato.fechaLarga(f).prefix(3)).uppercased()).font(Tipo.lato(11, .black))
                            Text("\(Formato.calendario.component(.day, from: f))").font(Tipo.lato(20, .black)).monospacedDigit()
                            Text(d.libres > 0 ? "\(d.libres)" : "—").font(Tipo.lato(11, .bold)).opacity(0.8)
                        }
                        .frame(width: 56, height: 72)
                        .foregroundStyle(on ? C.seleccionTexto : (d.libres > 0 ? C.tinta : C.sub))
                        .background(on ? C.seleccion : C.fondo, in: RoundedRectangle(cornerRadius: 16, style: .continuous))
                        .contorno(16, color: on ? .clear : C.linea)
                    }
                    .buttonStyle(.plain).disabled(d.libres == 0)
                }
            }
        }
    }
}

struct GrillaHorarios: View {
    var horarios: [HorarioLibre]
    var elegir: (HorarioLibre) -> Void
    var body: some View {
        if horarios.isEmpty {
            Text("No hay horarios libres ese día.").font(Tipo.lato(15)).foregroundStyle(C.sub)
        } else {
            LazyVGrid(columns: Array(repeating: GridItem(.flexible(), spacing: 8), count: 4), spacing: 8) {
                ForEach(horarios, id: \.inicio) { h in
                    Button { elegir(h) } label: {
                        Text(h.hora).font(Tipo.lato(17, .black)).monospacedDigit().frame(maxWidth: .infinity, minHeight: 52)
                            .foregroundStyle(C.huecoTinta)
                            .background { Rayas(quieto: true) }
                            .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
                            .contorno(14, color: C.huecoLinea)
                    }
                    .buttonStyle(.plain)
                }
            }
        }
    }
}

// MARK: - Reservar

struct Reservar: View {
    var slug: String
    var listo: (String) -> Void
    @Environment(Estado.self) private var estado
    @Environment(\.dismiss) private var cerrar
    @Environment(\.openURL) private var abrirURL
    @State private var p: PaginaPublica?
    @State private var servicio: PaginaPublica.Serv?
    @State private var profesional: String? = nil
    @State private var elegirPro = false
    @State private var dias: [DiaLibre] = []
    @State private var fecha = ""
    @State private var horarios: [HorarioLibre] = []
    @State private var horario: HorarioLibre?
    @State private var nombre = ""
    @State private var telefono = ""
    @State private var email = ""
    @State private var nota = ""
    @State private var cupon = ""
    @State private var enviando = false
    @State private var error = ""

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    if let p { contenido(p) } else if !error.isEmpty { ErrorConSalida(titulo: "No pudimos abrir el lugar", texto: error) { Task { await cargar() } } }
                    else { ForEach(0..<4, id: \.self) { _ in Esqueleto(alto: 72) } }
                }
                .padding(16)
            }
            .scrollDismissesKeyboard(.interactively)
            .background(C.fondo)
            .navigationTitle(p?.negocio.nombre ?? "Reservar").navigationBarTitleDisplayMode(.inline)
            .toolbar { ToolbarItem(placement: .cancellationAction) { Button("Cerrar") { cerrar() } } }
            .task { await cargar() }
            .onChange(of: fecha) { _, _ in horario = nil; Task { await cargarHorarios() } }
        }
    }

    @ViewBuilder private func contenido(_ p: PaginaPublica) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(p.negocio.nombre).font(Tipo.lato(26, .black))
            Text([p.web.rubro, p.negocio.ciudad].filter { !$0.isEmpty }.joined(separator: " · ")).font(Tipo.lato(14)).foregroundStyle(C.sub)
            if p.resenas.cantidad > 0 {
                HStack(spacing: 6) { Estrellas(puntaje: Int(p.resenas.promedio.rounded()), tam: 12); Text("\(String(format: "%.1f", p.resenas.promedio).replacingOccurrences(of: ".", with: ",")) · \(Formato.cantidad(p.resenas.cantidad, "reseña", "reseñas"))").font(Tipo.lato(13)) }
            }
        }
        Text("1 · SERVICIO").rotulo()
        ForEach(p.servicios.filter(\.activo).sorted { $0.orden < $1.orden }) { s in
            let on = servicio?.id == s.id
            Button { servicio = s; profesional = nil; Task { await cargarDias() } } label: {
                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(s.nombre).font(Tipo.lato(16, .bold))
                        Text(Formato.duracion(s.duracion)).font(Tipo.lato(13)).opacity(0.8)
                    }
                    Spacer()
                    Text(Formato.pesos(s.precio)).font(Tipo.lato(15, .bold)).monospacedDigit()
                }
                .foregroundStyle(on ? C.seleccionTexto : C.tinta).padding(14)
                .background(on ? C.seleccion : C.tarjeta, in: RoundedRectangle(cornerRadius: 16, style: .continuous))
            }
            .buttonStyle(.plain)
        }
        if let s = servicio {
            let pros = p.profesionales.filter { $0.servicios.isEmpty || $0.servicios.contains(s.id) }
            if pros.count > 1 {
                Text("2 · CON QUIÉN").rotulo()
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 8) {
                        chip(nil, "Cualquiera")
                        ForEach(pros) { x in chip(x.id, x.nombre) }
                    }
                }
            }
            Text("3 · CUÁNDO").rotulo()
            SelectorDias(dias: dias, fecha: $fecha)
            if horario == nil {
                GrillaHorarios(horarios: horarios) { h in horario = h; Haptico.seleccion() }
            } else if let h = horario {
                HStack {
                    Label("\(Fecha.iso(fecha).map { Formato.fechaLarga($0) } ?? fecha) a las \(h.hora)", systemImage: "checkmark.circle.fill").font(Tipo.lato(15, .bold)).foregroundStyle(C.mentaTexto)
                    Spacer()
                    Button("Cambiar") { horario = nil }.buttonStyle(BotonTexto())
                }
                Text("4 · TUS DATOS").rotulo()
                Bloque {
                    RenglonCampo(titulo: "Nombre y apellido", texto: $nombre)
                    RenglonCampo(titulo: "Teléfono", texto: $telefono, teclado: .phonePad)
                    RenglonCampo(titulo: "Email", texto: $email, teclado: .emailAddress)
                    RenglonCampo(titulo: "Algo que quieras avisar (opcional)", texto: $nota)
                    RenglonCampo(titulo: "Cupón (opcional)", texto: $cupon)
                }
                if !p.politicaCancelacion.isEmpty { Text(p.politicaCancelacion).font(Tipo.lato(13)).foregroundStyle(C.sub) }
                if !error.isEmpty { Text(error).font(Tipo.lato(14, .bold)).foregroundStyle(C.error) }
                Button(enviando ? "Reservando…" : "Reservar · \(Formato.pesos(s.precio))") { Task { await reservar(s, h) } }
                    .buttonStyle(BotonMenta()).disabled(enviando || nombre.count < 2 || telefono.filter(\.isNumber).count < 8)
            }
        }
    }

    private func chip(_ id: String?, _ t: String) -> some View {
        let on = profesional == id
        return Button { profesional = id; Task { await cargarDias() } } label: {
            Text(t).font(Tipo.lato(14, .bold)).padding(.horizontal, 16).frame(minHeight: 44)
                .foregroundStyle(on ? C.seleccionTexto : C.tinta)
                .background(on ? C.seleccion : C.tarjeta, in: Capsule())
        }
        .buttonStyle(.plain)
    }

    private var filtro: [URLQueryItem] {
        var q = [URLQueryItem(name: "servicio", value: servicio?.id ?? "")]
        if let profesional { q.append(URLQueryItem(name: "profesional", value: profesional)) }
        return q
    }

    private struct Ds: Decodable { var dias: [DiaLibre] }
    private struct Hs: Decodable { var horarios: [HorarioLibre] }

    private func cargar() async {
        do {
            p = try await estado.api.get("/v1/p/\(slug)")
            nombre = estado.persona?.nombre ?? ""
            email = estado.persona?.email ?? ""
            telefono = UserDefaults.standard.string(forKey: "miTelefono") ?? ""
            if p?.servicios.filter(\.activo).count == 1 { servicio = p?.servicios.first(where: \.activo); await cargarDias() }
        } catch let e as ErrorApi { error = e.detalle } catch { self.error = "Probá de nuevo." }
    }

    private func cargarDias() async {
        horario = nil
        dias = (try? await estado.api.get("/v1/p/\(slug)/dias", query: filtro) as Ds)?.dias ?? []
        let f = dias.first { $0.libres > 0 }?.fecha ?? ""
        if f == fecha { await cargarHorarios() } else { fecha = f }
    }

    private func cargarHorarios() async {
        guard !fecha.isEmpty, servicio != nil else { horarios = []; return }
        horarios = (try? await estado.api.get("/v1/p/\(slug)/libres", query: filtro + [URLQueryItem(name: "fecha", value: fecha)]) as Hs)?.horarios ?? []
    }

    private struct Pedido: Encodable {
        struct Cli: Encodable { var nombre: String; var telefono: String; var email: String }
        var servicioId: String; var profesionalId: String; var inicio: Date; var cliente: Cli; var nota: String; var cupon: String?
    }

    private struct Reservado: Decodable { var codigo: String; var inicio: Date; var pago: TurnoCli.Pago? }

    private func reservar(_ s: PaginaPublica.Serv, _ h: HorarioLibre) async {
        enviando = true
        error = ""
        defer { enviando = false }
        do {
            let t: Reservado = try await estado.api.enviar("POST", "/v1/p/\(slug)/turnos", Pedido(
                servicioId: s.id, profesionalId: profesional ?? "", inicio: h.inicio,
                cliente: .init(nombre: nombre, telefono: telefono, email: email), nota: nota, cupon: cupon.isEmpty ? nil : cupon))
            UserDefaults.standard.set(telefono, forKey: "miTelefono")
            Haptico.exito()
            listo(t.codigo)
            if let u = t.pago?.url, let url = URL(string: u), u.hasPrefix("https://") { abrirURL(url) }
            estado.avisar("¡Reservaste!", texto: "\(Formato.diaYHora(t.inicio)) en \(p?.negocio.nombre ?? "")")
            cerrar()
        } catch let e as ErrorApi {
            error = e.detalle
            Haptico.error()
            if e.codigo == "ocupado" { horario = nil; await cargarHorarios() }
        } catch { self.error = "No pudimos reservar. Probá de nuevo." }
    }
}

// MARK: - Lugares

struct Lugares: View {
    var reservar: (String) -> Void
    @Environment(Estado.self) private var estado
    @State private var lista: [FichaDirectorio] = []
    @State private var rubros: [String] = []
    @State private var rubro = ""
    @State private var buscar = ""
    @State private var hoy = false
    @State private var cargando = true

    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                Cabecera(titulo: "Lugares", subtitulo: "Reservá donde haya lugar") {
                    HStack(spacing: 8) {
                        Image(systemName: "magnifyingglass").foregroundStyle(.white.opacity(0.7))
                        TextField("", text: $buscar, prompt: Text("Peluquería, uñas, masajes…").foregroundStyle(.white.opacity(0.6))).foregroundStyle(.white).font(Tipo.lato(16))
                    }
                    .padding(.horizontal, 14).frame(height: 48)
                    .background(.white.opacity(0.12), in: RoundedRectangle(cornerRadius: 14, style: .continuous))
                    ScrollView(.horizontal, showsIndicators: false) {
                        HStack(spacing: 8) {
                            filtro("Hoy con lugar", hoy) { hoy.toggle() }
                            ForEach(rubros, id: \.self) { r in filtro(r, rubro == r) { rubro = rubro == r ? "" : r } }
                        }
                    }
                }
                Cuerpo {
                    if cargando && lista.isEmpty {
                        ForEach(0..<3, id: \.self) { _ in Esqueleto(alto: 180, radio: 20) }
                    } else if lista.isEmpty {
                        VacioQueEnsena(rayas: "Nada por acá", titulo: "No encontramos lugares", texto: "Probá con otra búsqueda o sacá los filtros.") { EmptyView() }
                    } else {
                        Text("CERCA TUYO").rotulo()
                        LazyVGrid(columns: [GridItem(.flexible(), spacing: 10), GridItem(.flexible(), spacing: 10)], spacing: 10) {
                            ForEach(lista) { f in tarjeta(f) }
                        }
                    }
                }
            }
        }
        .scrollIndicators(.hidden)
        .scrollDismissesKeyboard(.immediately)
        .background(C.fondo)
        .toolbar(.hidden, for: .navigationBar)
        .task(id: "\(rubro)-\(hoy)") { await cargar() }
        .task(id: buscar) { try? await Task.sleep(for: .milliseconds(350)); if !Task.isCancelled { await cargar() } }
    }

    private func filtro(_ t: String, _ on: Bool, _ a: @escaping () -> Void) -> some View {
        Button(action: a) {
            Text(t).font(Tipo.lato(14, .bold)).padding(.horizontal, 14).frame(height: 36)
                .foregroundStyle(on ? C.petroleoHondo : .white)
                .background(on ? C.menta : .white.opacity(0.1), in: Capsule())
        }
        .buttonStyle(.plain)
    }

    private func tarjeta(_ f: FichaDirectorio) -> some View {
        Button { reservar(f.slug) } label: {
            VStack(alignment: .leading, spacing: 0) {
                Color.clear.frame(height: 104).frame(maxWidth: .infinity)
                    .overlay {
                        ZStack {
                            C.chip
                            if let u = f.portada.flatMap(URL.init(string:)) {
                                AsyncImage(url: u) { i in i.resizable().scaledToFill() } placeholder: { C.chip }
                            } else {
                                Image(systemName: "sparkles").font(.system(size: 26)).foregroundStyle(C.sub)
                            }
                        }
                    }
                    .clipped()
                VStack(alignment: .leading, spacing: 2) {
                    Text(f.nombre).font(Tipo.lato(15, .bold)).lineLimit(2)
                    Text([f.rubro, f.ciudad].filter { !$0.isEmpty }.joined(separator: " · ")).font(Tipo.lato(12)).foregroundStyle(C.sub).lineLimit(1)
                    if let d = f.desde { Text("desde \(Formato.pesos(d))").font(Tipo.lato(12, .bold)).foregroundStyle(C.mentaTexto) }
                }
                .padding(10)
            }
            .foregroundStyle(C.tinta).frame(maxWidth: .infinity, alignment: .leading)
            .background(C.tarjeta).clipShape(RoundedRectangle(cornerRadius: 18, style: .continuous))
        }
        .buttonStyle(.plain)
    }

    private struct Dir: Decodable { var negocios: [FichaDirectorio] }
    private struct Rs: Decodable { var rubros: [String] }

    private func cargar() async {
        var q = [URLQueryItem(name: "limite", value: "24")]
        if !buscar.isEmpty { q.append(URLQueryItem(name: "q", value: buscar)) }
        if !rubro.isEmpty { q.append(URLQueryItem(name: "rubro", value: rubro)) }
        if hoy { q.append(URLQueryItem(name: "conLugarHoy", value: "1")) }
        lista = (try? await estado.api.get("/v1/explorar", query: q) as Dir)?.negocios ?? lista
        if rubros.isEmpty { rubros = (try? await estado.api.get("/v1/explorar/rubros") as Rs)?.rubros ?? [] }
        cargando = false
    }
}

// MARK: - Cuenta

struct CuentaCliente: View {
    @Environment(Estado.self) private var estado
    @State private var salir = false

    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                Cabecera(titulo: "Cuenta", subtitulo: estado.persona?.email ?? "")
                Cuerpo {
                    Bloque { Renglon(titulo: estado.persona?.nombre ?? "", detalle: estado.persona?.email ?? "", simbolo: "person.crop.circle") }
                    Bloque(titulo: "Modo") {
                        Button { Task { await estado.elegirModo("negocio") } } label: {
                            Renglon(titulo: "Atiendo un negocio", detalle: estado.negocios.isEmpty ? "Abrí el tuyo gratis" : estado.negocios.map(\.nombre).joined(separator: ", "), simbolo: "storefront")
                        }
                        .buttonStyle(.plain)
                    }
                    Bloque(titulo: "Avisos") {
                        Button { if let u = URL(string: UIApplication.openNotificationSettingsURLString) { UIApplication.shared.open(u) } } label: {
                            Renglon(titulo: "Avisos de tus turnos", detalle: "Se cambian en Ajustes", simbolo: "bell")
                        }
                        .buttonStyle(.plain)
                    }
                    Button(role: .destructive) { salir = true } label: { Label("Cerrar sesión", systemImage: "rectangle.portrait.and.arrow.right") }
                        .buttonStyle(BotonContorno(alto: 52, tam: 16)).foregroundStyle(C.error)
                        .confirmationDialog("¿Cerrar sesión?", isPresented: $salir, titleVisibility: .visible) {
                            Button("Cerrar sesión", role: .destructive) { Task { await estado.salir() } }
                        }
                }
            }
        }
        .background(C.fondo)
        .toolbar(.hidden, for: .navigationBar)
    }
}
