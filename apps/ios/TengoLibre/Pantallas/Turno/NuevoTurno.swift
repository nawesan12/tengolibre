import ContactsUI
import SwiftUI

/// 10 Nuevo turno: clienta (buscar, o desde Contactos), servicio, con quién, horario libre y
/// sobreturno; además turno fijo (cada N semanas, con vista previa) y nota, como en la web.
/// «Guardar · hoy 16:00 con Carla» fijo abajo. Al guardar, 11: el talón reemplaza al hueco y aparece
/// «Deshacer» durante 5 s.
struct NuevoTurno: View {
    var inicio: NuevoTurnoInicio
    @Environment(Estado.self) private var estado
    @Environment(\.dismiss) private var cerrar

    @State private var servicios: [Servicio] = []
    @State private var profesionales: [Profesional] = []
    @State private var cliente: ClienteDelTurno?
    @State private var buscar = ""
    @State private var encontrados: [Cliente] = []
    @State private var nueva = false
    @State private var nombreNuevo = ""
    @State private var telefonoNuevo = ""
    @State private var emailNuevo = ""
    @State private var servicio: String?
    @State private var profesional: String? = nil
    @State private var fecha = Date()
    @State private var libres: [HorarioLibre] = []
    @State private var cargandoLibres = false
    @State private var elegido: HorarioLibre?
    @State private var otroDia = false
    @State private var sobreturno = false
    @State private var horaSobreturno = Date()
    @State private var fijo = false
    @State private var cadaSemanas = 1
    @State private var cantidad = 4
    @State private var previa: PreviaSerie?
    @State private var fechasSerie: Set<Date> = []
    @State private var nota = ""
    @State private var contactos = false
    @State private var guardando = false
    @State private var error = ""
    @State private var tiembla = 0

    private var serv: Servicio? { servicios.first { $0.id == servicio } }
    private var pros: [Profesional] { profesionales.filter { $0.activo && (servicio == nil || $0.servicios.isEmpty || $0.servicios.contains(servicio!)) } }
    private var esHoy: Bool { Formato.diaISO(fecha) == Formato.diaISO(Date()) }

    /// El inicio elegido: un horario libre, o día y hora del sobreturno.
    private var inicioElegido: Date? {
        if sobreturno {
            let c = Formato.calendario.dateComponents([.hour, .minute], from: horaSobreturno)
            return Dia.fecha(fecha, minutos: (c.hour ?? 0) * 60 + (c.minute ?? 0))
        }
        return elegido?.inicio
    }

    private var conQuien: String {
        if let p = profesional, let x = profesionales.first(where: { $0.id == p }) { return x.nombre }
        if let e = elegido, let id = e.profesionales.first, let x = profesionales.first(where: { $0.id == id }) { return x.nombre }
        return ""
    }

    private var textoGuardar: String {
        guard let i = inicioElegido else { return "Guardar" }
        let cuando = esHoy ? "hoy \(Formato.horaDe(i))" : "\(Formato.dia(i)) \(Formato.horaDe(i))"
        if fijo { return "Ver las fechas del turno fijo" }
        return "Guardar · \(cuando)\(conQuien.isEmpty ? "" : " con \(conQuien)")"
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 18) {
                    paso("1 · CLIENTE") { seccionCliente }
                    paso("2 · SERVICIO") { seccionServicio }
                    paso("3 · CON QUIÉN") { seccionQuien }
                    paso(esHoy ? "4 · CUÁNDO · LIBRES HOY" : "4 · CUÁNDO · LIBRES EL \(Formato.dia(fecha).uppercased())") { seccionCuando }
                    paso("5 · SE REPITE") { seccionFijo }
                    paso("NOTA INTERNA") {
                        Campo(titulo: "", texto: $nota, ayuda: "Sólo la ve el equipo (ej.: piel sensible)", multilinea: true)
                    }
                    if !error.isEmpty {
                        Label(error, systemImage: "exclamationmark.circle").font(Tipo.lato(14, .bold)).foregroundStyle(C.error)
                            .modifier(Temblor(veces: CGFloat(tiembla)))
                    }
                }
                .padding(.horizontal, 16).padding(.top, 6).padding(.bottom, 120)
            }
            .scrollDismissesKeyboard(.interactively)
            .background(C.hoja)
            .safeAreaInset(edge: .bottom) {
                Button {
                    Task { await guardar() }
                } label: { Text(guardando ? "Guardando…" : textoGuardar) }
                .buttonStyle(BotonMenta())
                .disabled(guardando || cliente == nil || servicio == nil || inicioElegido == nil || estado.soloLectura)
                .padding(.horizontal, 16).padding(.top, 12).padding(.bottom, 8)
                .background(C.hoja.shadow(.drop(color: C.linea, radius: 0, y: -1)))
            }
            .navigationTitle("Nuevo turno").navigationBarTitleDisplayMode(.inline)
            .toolbar { ToolbarItem(placement: .cancellationAction) { Button("Cancelar") { cerrar() }.foregroundStyle(C.mentaTexto) } }
            .sheet(isPresented: $contactos) {
                SelectorContacto { n, t, e in
                    cliente = ClienteDelTurno(id: nil, nombre: n, telefono: t, email: e)
                    nueva = false
                }
                .ignoresSafeArea()
            }
            .sheet(item: $previa) { p in
                PreviaFijo(previa: p, elegidas: $fechasSerie) { Task { await guardarSerie() } }
                    .presentationDetents([.large])
            }
            .task { await cargar() }
            .onChange(of: servicio) { _, _ in elegido = nil; Task { await traerLibres() } }
            .onChange(of: profesional) { _, _ in elegido = nil; Task { await traerLibres() } }
            .onChange(of: fecha) { _, _ in elegido = nil; Task { await traerLibres() } }
            .onChange(of: buscar) { _, q in Task { await buscarClientes(q) } }
        }
    }

    private func paso<V: View>(_ t: String, @ViewBuilder _ v: () -> V) -> some View {
        VStack(alignment: .leading, spacing: 8) { Text(t).rotulo(); v() }
    }

    // MARK: 1 · Cliente

    @ViewBuilder private var seccionCliente: some View {
        if let c = cliente {
            HStack(spacing: 10) {
                Iniciales(nombre: c.nombre, tam: 40)
                VStack(alignment: .leading, spacing: 1) {
                    Text(c.nombre).font(Tipo.lato(16, .bold))
                    Text([c.telefono, c.id == nil && !c.telefono.isEmpty ? "nueva" : ""].filter { !$0.isEmpty }.joined(separator: " · "))
                        .font(Tipo.lato(13)).foregroundStyle(C.sub).monospacedDigit()
                }
                Spacer()
                Button { cliente = nil } label: { Image(systemName: "xmark.circle.fill").font(.system(size: 22)).foregroundStyle(C.sub).frame(width: 44, height: 44) }
                    .accessibilityLabel("Cambiar cliente")
            }
            .padding(.leading, 12).padding(.trailing, 8).padding(.vertical, 8).frame(minHeight: 60)
            .background(C.fondo, in: RoundedRectangle(cornerRadius: 16, style: .continuous))
            .overlay(RoundedRectangle(cornerRadius: 16, style: .continuous).strokeBorder(C.seleccion, lineWidth: 2))
        } else if nueva {
            VStack(spacing: 10) {
                Campo(titulo: "Nombre", texto: $nombreNuevo)
                Campo(titulo: "Teléfono", texto: $telefonoNuevo, teclado: .phonePad)
                Campo(titulo: "Email (opcional)", texto: $emailNuevo, teclado: .emailAddress)
                Button("Usar estos datos") {
                    cliente = ClienteDelTurno(id: nil, nombre: nombreNuevo.trimmingCharacters(in: .whitespaces), telefono: telefonoNuevo, email: emailNuevo)
                }
                .buttonStyle(BotonContorno(alto: 48, tam: 16))
                .disabled(nombreNuevo.trimmingCharacters(in: .whitespaces).count < 2)
            }
        } else {
            HStack(spacing: 8) {
                Image(systemName: "magnifyingglass").font(.system(size: 18)).foregroundStyle(C.sub)
                TextField("Buscá por nombre o teléfono", text: $buscar).font(Tipo.lato(16))
            }
            .padding(.horizontal, 14).frame(height: 48)
            .background(C.campo, in: RoundedRectangle(cornerRadius: 14, style: .continuous))
            ForEach(encontrados.prefix(5)) { c in
                Button { cliente = ClienteDelTurno(id: c.id, nombre: c.nombre, telefono: c.telefono, email: c.email) } label: {
                    HStack(spacing: 10) {
                        Iniciales(nombre: c.nombre, tam: 36)
                        VStack(alignment: .leading) {
                            Text(c.nombre).font(Tipo.lato(15, .bold))
                            Text("\(c.telefono) · \(c.visitas) \(c.visitas == 1 ? "visita" : "visitas")").font(Tipo.lato(12)).foregroundStyle(C.sub)
                        }
                        Spacer()
                        if c.penalizado { PastillaEstado(texto: "Penalidad", simbolo: "exclamationmark.shield", fondo: C.chip, tinta: C.tinta) }
                    }
                    .foregroundStyle(C.tinta).padding(.vertical, 6)
                }
                .buttonStyle(.plain)
            }
            HStack(spacing: 8) {
                Button { contactos = true } label: { Label("De tus Contactos", systemImage: "person.crop.circle") }
                    .buttonStyle(BotonContorno(alto: 44, tam: 15))
                Button { nueva = true; nombreNuevo = buscar } label: { Label("Clienta nueva", systemImage: "person.badge.plus") }
                    .buttonStyle(BotonContorno(alto: 44, tam: 15))
            }
        }
    }

    // MARK: 2 · Servicio

    private var seccionServicio: some View {
        LazyVGrid(columns: [GridItem(.flexible(), spacing: 8), GridItem(.flexible(), spacing: 8)], spacing: 8) {
            ForEach(servicios.filter(\.activo)) { s in
                let on = servicio == s.id
                Button { servicio = s.id } label: {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(s.nombre).font(Tipo.lato(15, .bold)).multilineTextAlignment(.leading).lineLimit(2)
                        Text(estado.vePrecios ? "\(Formato.duracion(s.duracion)) · \(Formato.pesos(s.precio))" : Formato.duracion(s.duracion))
                            .font(Tipo.lato(13)).monospacedDigit().opacity(0.8)
                    }
                    .frame(maxWidth: .infinity, minHeight: 64, alignment: .leading).padding(.horizontal, 12).padding(.vertical, 10)
                    .foregroundStyle(on ? C.seleccionTexto : C.tinta)
                    .background(on ? C.seleccion : C.fondo, in: RoundedRectangle(cornerRadius: 16, style: .continuous))
                    .contorno(16, color: on ? .clear : C.linea)
                }
                .buttonStyle(.plain)
                .accessibilityAddTraits(on ? .isSelected : [])
            }
        }
    }

    // MARK: 3 · Con quién

    private var seccionQuien: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                botonQuien(nil, "Cualquiera").frame(minWidth: 110)
                ForEach(pros) { p in botonQuien(p.id, p.nombre) }
            }
        }
    }

    private func botonQuien(_ id: String?, _ t: String) -> some View {
        let on = profesional == id
        return Button { profesional = id } label: {
            Text(t).font(Tipo.lato(14, .bold)).padding(.horizontal, 16).frame(minHeight: 44)
                .foregroundStyle(on ? C.seleccionTexto : C.tinta)
                .background(on ? C.seleccion : C.fondo, in: Capsule())
                .contorno(999, color: on ? .clear : C.linea)
        }
        .buttonStyle(.plain)
    }

    // MARK: 4 · Cuándo

    @ViewBuilder private var seccionCuando: some View {
        if !sobreturno {
            if cargandoLibres {
                HStack(spacing: 8) { ForEach(0..<3, id: \.self) { _ in Esqueleto(alto: 60, radio: 16) } }
            } else if libres.isEmpty {
                Text(servicio == nil ? "Elegí un servicio para ver los horarios libres." : "No hay horarios libres ese día. Probá otro día o cargalo como sobreturno.")
                    .font(Tipo.lato(14)).foregroundStyle(C.sub)
            } else {
                LazyVGrid(columns: Array(repeating: GridItem(.flexible(), spacing: 8), count: 3), spacing: 8) {
                    ForEach(libres, id: \.inicio) { h in
                        let on = elegido?.inicio == h.inicio
                        let quien = h.profesionales.first.flatMap { id in profesionales.first { $0.id == id }?.nombre } ?? ""
                        Button { elegido = h; Haptico.seleccion() } label: {
                            VStack(spacing: 0) {
                                Text(h.hora).font(Tipo.lato(18, .black)).monospacedDigit()
                                Text(profesional == nil && h.profesionales.count > 1 ? "\(h.profesionales.count) libres" : quien).font(Tipo.lato(12, .bold)).lineLimit(1)
                            }
                            .frame(maxWidth: .infinity, minHeight: 60)
                            .foregroundStyle(on ? C.seleccionTexto : C.huecoTinta)
                            .background { if on { RoundedRectangle(cornerRadius: 16, style: .continuous).fill(C.seleccion) } else { Rayas(quieto: true) } }
                            .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
                            .contorno(16, color: on ? .clear : C.huecoLinea)
                        }
                        .buttonStyle(.plain)
                    }
                }
            }
        }
        VStack(spacing: 0) {
            Button { withAnimation { otroDia.toggle() } } label: {
                HStack(spacing: 10) {
                    Image(systemName: "calendar").font(.system(size: 20))
                    Text(esHoy ? "Otro día" : Formato.fechaLarga(fecha)).font(Tipo.lato(16)).frame(maxWidth: .infinity, alignment: .leading)
                    Image(systemName: otroDia ? "chevron.up" : "chevron.right").foregroundStyle(C.sub)
                }
                .foregroundStyle(C.tinta).frame(minHeight: 52).padding(.horizontal, 4)
            }
            .buttonStyle(.plain)
            if otroDia {
                DatePicker("Día", selection: $fecha, in: Formato.calendario.startOfDay(for: Date())..., displayedComponents: .date)
                    .datePickerStyle(.graphical).tint(C.mentaTexto).environment(\.locale, Formato.argentina)
            }
            Rectangle().fill(C.linea).frame(height: 1)
            Toggle(isOn: $sobreturno.animation()) {
                HStack(spacing: 10) {
                    Image(systemName: "square.stack.3d.up").font(.system(size: 20))
                    VStack(alignment: .leading, spacing: 1) {
                        Text("Sobreturno").font(Tipo.lato(16))
                        Text("Encimarlo aunque el horario esté ocupado").font(Tipo.lato(12)).foregroundStyle(C.sub)
                    }
                }
            }
            .tint(C.menta).frame(minHeight: 56).padding(.horizontal, 4)
            if sobreturno {
                DatePicker("Hora", selection: $horaSobreturno, displayedComponents: .hourAndMinute)
                    .font(Tipo.lato(16)).padding(.horizontal, 4).frame(minHeight: 52)
                if profesional == nil {
                    Text("Para un sobreturno elegí con quién.").font(Tipo.lato(13)).foregroundStyle(C.sub).padding(.horizontal, 4)
                }
            }
        }
    }

    // MARK: 5 · Turno fijo

    @ViewBuilder private var seccionFijo: some View {
        Toggle(isOn: $fijo.animation()) {
            HStack(spacing: 10) {
                Image(systemName: "repeat").font(.system(size: 20))
                VStack(alignment: .leading, spacing: 1) {
                    Text("Turno fijo").font(Tipo.lato(16))
                    Text("La misma hora cada semana o cada tantas").font(Tipo.lato(12)).foregroundStyle(C.sub)
                }
            }
        }
        .tint(C.menta).frame(minHeight: 56)
        if fijo {
            Stepper("Cada \(cadaSemanas) \(cadaSemanas == 1 ? "semana" : "semanas")", value: $cadaSemanas, in: 1...8).font(Tipo.lato(16))
            Stepper("\(cantidad) turnos", value: $cantidad, in: 2...52).font(Tipo.lato(16))
        }
    }

    // MARK: Datos

    private struct Libres: Decodable { var horarios: [HorarioLibre] }
    private struct Clientes: Decodable { var clientes: [Cliente] }
    private struct Pros: Decodable { var profesionales: [Profesional] }
    private struct Servs: Decodable { var servicios: [Servicio] }

    private func cargar() async {
        let p: Pros? = try? await estado.api.get("/v1/panel/profesionales")
        let s: Servs? = try? await estado.api.get("/v1/panel/servicios")
        let (pp, ss) = (p, s)
        profesionales = (pp?.profesionales ?? []).sorted { $0.orden < $1.orden }
        servicios = ss?.servicios ?? []
        if let c = inicio.cliente { cliente = c }
        if let i = inicio.inicio {
            fecha = Formato.calendario.startOfDay(for: i)
            horaSobreturno = i
        }
        profesional = inicio.profesionalId
        if estado.esProfesional, profesional == nil { profesional = profesionales.first?.id }
        servicio = inicio.servicioId ?? servicios.first { s in
            s.activo && (profesional == nil || (profesionales.first { $0.id == profesional }?.servicios ?? []).contains(s.id))
        }?.id ?? servicios.first(where: \.activo)?.id
        await traerLibres()
        if let i = inicio.inicio { elegido = libres.first { $0.inicio == i } ?? libres.first { abs($0.inicio.timeIntervalSince(i)) < 60 } }
    }

    private func traerLibres() async {
        guard let servicio else { libres = []; return }
        cargandoLibres = true
        defer { cargandoLibres = false }
        var q = [URLQueryItem(name: "servicio", value: servicio), URLQueryItem(name: "fecha", value: Formato.diaISO(fecha))]
        if let profesional { q.append(URLQueryItem(name: "profesional", value: profesional)) }
        libres = (try? await estado.api.get("/v1/panel/libres", query: q) as Libres)?.horarios ?? []
    }

    private func buscarClientes(_ q: String) async {
        let t = q.trimmingCharacters(in: .whitespaces)
        guard t.count >= 2 else { encontrados = []; return }
        try? await Task.sleep(for: .milliseconds(250))
        guard t == buscar.trimmingCharacters(in: .whitespaces) else { return }
        let r: Clientes? = try? await estado.api.get("/v1/panel/clientes", query: [URLQueryItem(name: "q", value: t), URLQueryItem(name: "porPagina", value: "8")])
        encontrados = r?.clientes ?? []
    }

    private struct Crear: Encodable {
        var servicioId: String
        var profesionalId: String
        var inicio: Date
        var cliente: ClienteDelTurno
        var nota: String
        var forzar: Bool
    }

    private func guardar() async {
        guard let cliente, let servicio, let i = inicioElegido else { return }
        error = ""
        if fijo { await pedirPrevia(); return }
        let pro = profesional ?? elegido?.profesionales.first ?? ""
        if sobreturno && pro.isEmpty { fallar("Para un sobreturno elegí con quién."); return }
        guardando = true
        defer { guardando = false }
        do {
            let t: Turno = try await estado.api.enviar("POST", "/v1/panel/turnos",
                Crear(servicioId: servicio, profesionalId: pro, inicio: i, cliente: cliente, nota: nota, forzar: sobreturno))
            Haptico.exito()
            estado.version += 1
            let nombre = cliente.nombre.split(separator: " ").first.map(String.init) ?? cliente.nombre
            estado.avisar(Toast(simbolo: "checkmark", titulo: "Turno guardado", texto: "Le mandamos la confirmación a \(nombre)", accion: "Deshacer")) {
                try? await estado.api.hacer("PATCH", "/v1/panel/turnos/\(t.id)", AccionesTurno.CambioEstado(estado: "cancelado"))
            }
            cerrar()
        } catch let e as ErrorApi {
            fallar(e.detalle)
        } catch {
            fallar("No pudimos guardarlo. Probá de nuevo.")
        }
    }

    private func fallar(_ m: String) {
        error = m
        Haptico.error()
        withAnimation(.default) { tiembla += 1 }
    }

    // Turno fijo: primero la vista previa (qué fechas quedan libres y cuáles chocan), después la serie.

    private struct PedidoPrevia: Encodable { var servicioId: String; var profesionalId: String; var inicio: Date; var cadaSemanas: Int; var cantidad: Int }

    private func pedirPrevia() async {
        guard let servicio, let i = inicioElegido else { return }
        guard let pro = profesional ?? elegido?.profesionales.first else { fallar("Para un turno fijo elegí quién lo atiende."); return }
        guardando = true
        defer { guardando = false }
        do {
            let p: PreviaSerie = try await estado.api.enviar("POST", "/v1/panel/series/previa",
                PedidoPrevia(servicioId: servicio, profesionalId: pro, inicio: i, cadaSemanas: cadaSemanas, cantidad: cantidad))
            fechasSerie = Set(p.fechas.compactMap { f in f.estado == "libre" ? f.inicio : f.alternativa?.inicio })
            previa = p
        } catch let e as ErrorApi { fallar(e.detalle) } catch { fallar("No pudimos armar las fechas.") }
    }

    private struct PedidoSerie: Encodable {
        struct T: Encodable { var inicio: Date }
        var servicioId: String
        var profesionalId: String
        var cadaSemanas: Int
        var clienteId: String?
        var cliente: ClienteDelTurno?
        var nota: String
        var turnos: [T]
    }
    private struct Serie: Decodable { var id: String; var turnos: [Turno] }

    private func guardarSerie() async {
        guard let servicio, let cliente, let pro = profesional ?? elegido?.profesionales.first else { return }
        do {
            let r: Serie = try await estado.api.enviar("POST", "/v1/panel/series", PedidoSerie(
                servicioId: servicio, profesionalId: pro, cadaSemanas: cadaSemanas,
                clienteId: cliente.id, cliente: cliente.id == nil ? cliente : nil, nota: nota,
                turnos: fechasSerie.sorted().map { .init(inicio: $0) }))
            Haptico.exito()
            estado.version += 1
            estado.avisar("Turno fijo guardado", texto: "\(r.turnos.count) turnos de \(cliente.nombre)")
            previa = nil
            cerrar()
        } catch {
            estado.avisar(error)
        }
    }
}

/// La vista previa del turno fijo (turno-fijo.md): las fechas libres ya elegidas; las que chocan,
/// con su alternativa o para sacarlas.
struct PreviaSerie: Codable, Identifiable, Equatable {
    struct FechaSerie: Codable, Equatable, Hashable {
        struct Alt: Codable, Equatable, Hashable { var inicio: Date; var hora: String }
        var inicio: Date
        var fecha: String
        var hora: String
        var estado: String
        var motivo: String?
        var alternativa: Alt?
    }
    var cadaSemanas: Int
    var servicio: String
    var profesional: String
    var libres: Int
    var chocan: Int
    var fechas: [FechaSerie]
    var id: String { "\(servicio)-\(fechas.first?.fecha ?? "")" }
}

struct PreviaFijo: View {
    var previa: PreviaSerie
    @Binding var elegidas: Set<Date>
    var reservar: () -> Void
    @Environment(\.dismiss) private var cerrar

    var body: some View {
        NavigationStack {
            List {
                Section {
                    Text("\(previa.servicio) con \(previa.profesional), cada \(previa.cadaSemanas) \(previa.cadaSemanas == 1 ? "semana" : "semanas"). \(previa.libres) libres, \(previa.chocan) chocan.")
                        .font(Tipo.lato(15)).foregroundStyle(C.sub)
                }
                ForEach(previa.fechas, id: \.inicio) { f in
                    let fecha = f.estado == "libre" ? f.inicio : f.alternativa?.inicio
                    Button {
                        guard let fecha else { return }
                        if elegidas.contains(fecha) { elegidas.remove(fecha) } else { elegidas.insert(fecha) }
                    } label: {
                        HStack {
                            Image(systemName: fecha.map { elegidas.contains($0) } == true ? "checkmark.circle.fill" : "circle")
                                .foregroundStyle(fecha == nil ? C.linea : C.mentaTexto).font(.system(size: 22))
                            VStack(alignment: .leading, spacing: 2) {
                                Text(Formato.diaYHora(f.inicio)).font(Tipo.lato(15, .bold)).foregroundStyle(C.tinta)
                                if f.estado == "choca" {
                                    let m = ["ocupado": "Ocupado a esa hora", "bloqueado": "Está bloqueado ese día", "no_atiende": "No atiende ese día", "pasado": "Ya pasó"][f.motivo ?? ""] ?? "Choca"
                                    Text(f.alternativa.map { "\(m) · podés a las \($0.hora)" } ?? m).font(Tipo.lato(13)).foregroundStyle(C.ausenteTinta)
                                }
                            }
                        }
                    }
                    .disabled(fecha == nil)
                }
            }
            .safeAreaInset(edge: .bottom) {
                Button("Reservar \(elegidas.count) \(elegidas.count == 1 ? "turno" : "turnos")", action: reservar)
                    .buttonStyle(BotonMenta()).disabled(elegidas.isEmpty).padding(16).background(C.hoja)
            }
            .navigationTitle("Turno fijo").navigationBarTitleDisplayMode(.inline)
            .toolbar { ToolbarItem(placement: .cancellationAction) { Button("Volver") { cerrar() } } }
        }
    }
}

/// El bloque tiembla una vez, 3 pt (error al guardar).
struct Temblor: GeometryEffect {
    var veces: CGFloat
    var animatableData: CGFloat { get { veces } set { veces = newValue } }
    func effectValue(size: CGSize) -> ProjectionTransform {
        ProjectionTransform(CGAffineTransform(translationX: 3 * sin(veces * .pi * 4), y: 0))
    }
}

/// Elegir de los Contactos (CNContactPickerViewController: no pide permiso).
struct SelectorContacto: UIViewControllerRepresentable {
    var elegido: (String, String, String) -> Void
    func makeCoordinator() -> Coordinador { Coordinador(elegido: elegido) }
    func makeUIViewController(context: Context) -> CNContactPickerViewController {
        let c = CNContactPickerViewController()
        c.delegate = context.coordinator
        c.displayedPropertyKeys = [CNContactPhoneNumbersKey, CNContactEmailAddressesKey]
        return c
    }
    func updateUIViewController(_ uiViewController: CNContactPickerViewController, context: Context) {}

    final class Coordinador: NSObject, CNContactPickerDelegate {
        var elegido: (String, String, String) -> Void
        init(elegido: @escaping (String, String, String) -> Void) { self.elegido = elegido }
        func contactPicker(_ picker: CNContactPickerViewController, didSelect contact: CNContact) {
            let nombre = [contact.givenName, contact.familyName].filter { !$0.isEmpty }.joined(separator: " ")
            let tel = contact.phoneNumbers.first?.value.stringValue ?? ""
            let email = contact.emailAddresses.first.map { String($0.value) } ?? ""
            elegido(nombre, tel, email)
        }
    }
}
