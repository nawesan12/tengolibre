import SwiftUI
import UserNotifications

/// Lo que se apila dentro de «Más»: todas las secciones del panel web, nativas.
enum DestinoMas: Hashable {
    case primerosPasos, resenas, suscripcion, diasLibres, cobros
    case servicios, equipo, profesional(String), miHorario
    case membresias, cupones, referidos, penalidades, etiquetas, caja
    case reportes, miWeb, plantillas, dominio, integraciones
    case configuracion, avisosClientes, whatsapp, auditoria, sucursales, cuenta, recorrido
}

extension View {
    /// Los destinos de todas las pilas (cada pestaña tiene la suya, pero todas saben abrir lo mismo).
    func destinos() -> some View {
        navigationDestination(for: DestinoMas.self) { d in
            Group {
                switch d {
                case .primerosPasos: Potencia()
                case .resenas: Resenas()
                case .suscripcion: SuscripcionVista()
                case .diasLibres: DiasLibres()
                case .cobros: Cobros()
                case .servicios: Servicios()
                case .equipo: Equipo()
                case .profesional(let id): FichaProfesional(id: id)
                case .miHorario: MiHorario()
                case .membresias: Membresias()
                case .cupones: Cupones()
                case .referidos: Referidos()
                case .penalidades: Penalidades()
                case .etiquetas: Etiquetas()
                case .caja: CajaDelDia()
                case .reportes: Reportes()
                case .miWeb: MiWeb()
                case .plantillas: Plantillas()
                case .dominio: DominioVista()
                case .integraciones: Integraciones()
                case .configuracion: Configuracion()
                case .avisosClientes: AvisosClientes()
                case .whatsapp: WhatsAppVista()
                case .auditoria: Auditoria()
                case .sucursales: Sucursales()
                case .cuenta: Cuenta()
                case .recorrido: RecorridoOtraVez()
                }
            }
            .toolbar(.hidden, for: .navigationBar)
        }
        .navigationDestination(for: DestinoClientes.self) { d in
            switch d {
            case .ficha(let id): FichaCliente(id: id).toolbar(.hidden, for: .navigationBar)
            }
        }
    }
}

/// Más: el negocio arriba y todas las secciones agrupadas, recortadas según el rol (como la web).
struct Mas: View {
    @Environment(Estado.self) private var estado
    @Environment(Navegacion.self) private var nav

    private struct Item: Identifiable { var destino: DestinoMas; var simbolo: String; var titulo: String; var detalle = ""; var pendiente = false; var id: String { titulo } }

    /// El estado de cada sección, en vivo («Sin conectar», «3 socias activas», «IA apagada»…).
    private func conEstado(_ it: Item) -> Item {
        let v = estado.vivo
        let t: (String, Bool)?
        switch it.destino {
        case .cobros: t = v.textoCobros
        case .integraciones: t = estado.esProfesional ? nil : v.textoIA
        case .membresias: t = v.textoMembresias
        case .cupones: t = v.textoCupones
        case .avisosClientes: t = v.textoRecordatorios
        case .miWeb: t = v.textoWeb
        case .dominio: t = v.textoDominio
        case .whatsapp: t = v.textoWhatsApp
        default: t = nil
        }
        var x = it
        if let t, !t.0.isEmpty { x.detalle = t.0; x.pendiente = t.1 }
        return x
    }

    private var grupos: [(String, [Item])] {
        let g = estado.gestiona
        let mostrador = g || estado.rol == "recepcion"
        var out: [(String, [Item])] = []
        if estado.esProfesional {
            out.append(("TU TRABAJO", [
                Item(destino: .miHorario, simbolo: "clock", titulo: "Mi horario", detalle: "Horario, bloqueos y días libres"),
                Item(destino: .integraciones, simbolo: "calendar.badge.plus", titulo: "Mi agenda en el calendario", detalle: "Google, Apple u Outlook")
            ]))
        } else {
            var negocio: [Item] = []
            if g { negocio.append(Item(destino: .servicios, simbolo: "sparkles", titulo: "Servicios", detalle: "Precios, duración, categorías y productos")) }
            negocio.append(Item(destino: .equipo, simbolo: "person.2.badge.gearshape", titulo: "Equipo y horarios", detalle: "Profesionales, horarios e invitaciones"))
            if g { negocio.append(Item(destino: .diasLibres, simbolo: "beach.umbrella", titulo: "Días libres", detalle: "Pedidos del equipo")) }
            negocio.append(Item(destino: .resenas, simbolo: "star.bubble", titulo: "Reseñas"))
            if g { negocio.append(Item(destino: .reportes, simbolo: "chart.bar", titulo: "Reportes", detalle: "Ingresos, ocupación y horas muertas")) }
            out.append(("TU NEGOCIO", negocio))
            var plata: [Item] = [Item(destino: .caja, simbolo: "banknote", titulo: "Caja", detalle: "Lo cobrado y lo que deben")]
            if g { plata.append(Item(destino: .cobros, simbolo: "creditcard", titulo: "Pagos y señas", detalle: "Mercado Pago y cobro por adelantado")) }
            if mostrador {
                plata.append(Item(destino: .membresias, simbolo: "ticket", titulo: "Membresías", detalle: "Planes y socias"))
                plata.append(Item(destino: .cupones, simbolo: "tag", titulo: "Cupones"))
            }
            if g { plata.append(Item(destino: .referidos, simbolo: "gift", titulo: "Referidos")) }
            out.append(("PLATA", plata))
            if mostrador {
                out.append(("CLIENTES", [
                    Item(destino: .etiquetas, simbolo: "tag.circle", titulo: "Etiquetas"),
                    Item(destino: .penalidades, simbolo: "exclamationmark.shield", titulo: "Penalidades y ausencias"),
                    Item(destino: .avisosClientes, simbolo: "envelope.badge", titulo: "Recordatorios a clientes", detalle: "Cumpleaños y «te extrañamos»")
                ]))
            }
            if g {
                out.append(("EN LA WEB", [
                    Item(destino: .miWeb, simbolo: "globe", titulo: "Mi web", detalle: "Datos, fotos, logo y portada"),
                    Item(destino: .plantillas, simbolo: "paintpalette", titulo: "Plantillas", detalle: "Cómo se ve tu página"),
                    Item(destino: .dominio, simbolo: "link", titulo: "Dominio propio"),
                    Item(destino: .integraciones, simbolo: "point.3.connected.trianglepath.dotted", titulo: "Integraciones e IA", detalle: "Calendarios, Claude, ChatGPT")
                ]))
                var ajustes = [
                    Item(destino: .configuracion, simbolo: "gearshape", titulo: "Configuración", detalle: "Reservas, cancelaciones y avisos"),
                    Item(destino: .whatsapp, simbolo: "message", titulo: "WhatsApp", detalle: "Confirmaciones y recordatorios"),
                    Item(destino: .auditoria, simbolo: "list.bullet.rectangle", titulo: "Actividad", detalle: "Quién hizo qué")
                ]
                if estado.esDuena {
                    ajustes.append(Item(destino: .suscripcion, simbolo: "crown", titulo: "Suscripción"))
                    ajustes.append(Item(destino: .sucursales, simbolo: "building.2", titulo: "Sucursales"))
                }
                out.append(("AJUSTES", ajustes))
            }
        }
        out.append(("CUENTA", [Item(destino: .recorrido, simbolo: "play.circle", titulo: "Qué podés hacer", detalle: "El recorrido y cómo sumar el widget"),
                               Item(destino: .cuenta, simbolo: "person.crop.circle", titulo: "Tu cuenta", detalle: estado.persona?.email ?? "")]))
        return out
    }

    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                Cabecera(titulo: "Más", subtitulo: estado.negocio.map { "\($0.nombre) · \($0.rolTexto)" } ?? "", conNegocio: true,
                         acciones: [AccionCabecera(simbolo: "square.and.arrow.up", etiqueta: "Compartir tu link") { nav.compartir = true }]) {
                    CampoBuscar()
                }
                Cuerpo {
                    if estado.gestiona {
                        let avance = estado.vivo.avance(estado)
                        NavigationLink(value: DestinoMas.primerosPasos) {
                            FilaLista(simbolo: "chart.line.uptrend.xyaxis", titulo: "Potenciá tu negocio",
                                      detalle: estado.vivo.cargado ? (avance >= 1 ? "Estás usando todo" : "Al \(Int((avance * 100).rounded())) % · ver lo que falta") : "Lo que te trae más reservas y más plata",
                                      pendiente: estado.vivo.cargado && avance < 1)
                        }
                        .buttonStyle(.plain).tarjeta()
                    }
                    Link(destination: Web.pagina(estado.negocio?.slug ?? "")) {
                        FilaLista(simbolo: "safari", titulo: "Ver mi página", detalle: Web.pagina(estado.negocio?.slug ?? "").absoluteString.replacingOccurrences(of: "https://", with: ""), final: "arrow.up.right")
                    }
                    .buttonStyle(.plain).tarjeta()
                    ForEach(Array(grupos.enumerated()), id: \.element.0) { gi, g in
                        Rotulo(g.0).entrada(gi)
                        VStack(spacing: 0) {
                            ForEach(Array(g.1.map(conEstado).enumerated()), id: \.element.id) { i, it in
                                if i > 0 { Rectangle().fill(C.linea).frame(height: 1).padding(.leading, 56) }
                                NavigationLink(value: it.destino) { FilaLista(simbolo: it.simbolo, titulo: it.titulo, detalle: it.detalle, pendiente: it.pendiente) }
                                    .buttonStyle(.plain)
                            }
                        }
                        .tarjeta()
                        .entrada(gi)
                    }
                    Text("Tengo Libre \(Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String ?? "") (\(Bundle.main.object(forInfoDictionaryKey: "CFBundleVersion") as? String ?? ""))")
                        .font(Tipo.lato(12)).foregroundStyle(C.sub).frame(maxWidth: .infinity).padding(.top, 8)
                }
            }
        }
        .scrollIndicators(.hidden)
        .background(C.fondo)
        .toolbar(.hidden, for: .navigationBar)
        .task(id: estado.version) { await estado.vivo.cargar(estado) }
    }
}

// MARK: - Piezas comunes de las secciones

/// Una sección de «Más»: cabecera con «‹ Más», cuerpo, cargando / error y tirar para refrescar.
struct PantallaMas<Contenido: View, Abajo: View>: View {
    var titulo: String
    var subtitulo: String = ""
    var volver: String = "Más"
    var acciones: [AccionCabecera] = []
    var cargando = false
    var error = ""
    var recargar: () async -> Void
    @ViewBuilder var abajo: Abajo
    @ViewBuilder var contenido: Contenido

    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                Cabecera(titulo: titulo, subtitulo: subtitulo, volver: volver, acciones: acciones) { abajo }
                Cuerpo {
                    if !error.isEmpty {
                        ErrorConSalida(titulo: "No pudimos traer esto", texto: error) { Task { await recargar() } }
                    } else if cargando {
                        ForEach(0..<4, id: \.self) { _ in Esqueleto(alto: 72) }
                    } else {
                        contenido
                    }
                }
            }
        }
        .scrollIndicators(.hidden)
        .scrollDismissesKeyboard(.interactively)
        .background(C.fondo)
        .toolbar(.hidden, for: .navigationBar)
        .refreshable { await recargar() }
    }
}

extension PantallaMas where Abajo == EmptyView {
    init(titulo: String, subtitulo: String = "", volver: String = "Más", acciones: [AccionCabecera] = [], cargando: Bool = false, error: String = "",
         recargar: @escaping () async -> Void, @ViewBuilder contenido: () -> Contenido) {
        self.init(titulo: titulo, subtitulo: subtitulo, volver: volver, acciones: acciones, cargando: cargando, error: error, recargar: recargar,
                  abajo: { EmptyView() }, contenido: contenido)
    }
}

/// Una tarjeta con rótulo y renglones separados por línea.
struct Bloque<Contenido: View>: View {
    var titulo: String = ""
    var pie: String = ""
    @ViewBuilder var contenido: Contenido
    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            if !titulo.isEmpty { Text(titulo.uppercased()).rotulo().padding(.horizontal, 4) }
            VStack(alignment: .leading, spacing: 0) {
                _VariadicView.Tree(Separados()) { contenido }
            }
            .tarjeta()
            if !pie.isEmpty { Text(pie).font(Tipo.lato(13)).foregroundStyle(C.sub).padding(.horizontal, 4) }
        }
    }
}

/// Pone una línea entre los renglones de un Bloque.
struct Separados: _VariadicView_MultiViewRoot {
    func body(children: _VariadicView.Children) -> some View {
        ForEach(children) { c in
            if c.id != children.first?.id { Rectangle().fill(C.linea).frame(height: 1).padding(.leading, 16) }
            c
        }
    }
}

/// Un renglón «título … valor» dentro de un Bloque.
struct Renglon<Derecha: View>: View {
    var titulo: String
    var detalle: String = ""
    var simbolo: String? = nil
    @ViewBuilder var derecha: Derecha
    var body: some View {
        HStack(spacing: 12) {
            if let simbolo { Image(systemName: simbolo).font(.system(size: 18, weight: .semibold)).foregroundStyle(C.mentaTexto).frame(width: 26) }
            VStack(alignment: .leading, spacing: 2) {
                Text(titulo).font(Tipo.lato(16))
                if !detalle.isEmpty { Text(detalle).font(Tipo.lato(13)).foregroundStyle(C.sub) }
            }
            Spacer(minLength: 8)
            derecha
        }
        .foregroundStyle(C.tinta).padding(.horizontal, 16).padding(.vertical, 10).frame(minHeight: 56)
    }
}

extension Renglon where Derecha == EmptyView {
    init(titulo: String, detalle: String = "", simbolo: String? = nil) {
        self.init(titulo: titulo, detalle: detalle, simbolo: simbolo) { EmptyView() }
    }
}

/// Un número que se cambia con − y + (minutos, horas, días, porcentajes).
struct Numerito: View {
    var titulo: String
    var detalle: String = ""
    @Binding var valor: Int
    var rango: ClosedRange<Int>
    var paso: Int = 1
    var unidad: (Int) -> String = { "\($0)" }
    var body: some View {
        Renglon(titulo: titulo, detalle: detalle) {
            HStack(spacing: 6) {
                Button { valor = max(rango.lowerBound, valor - paso); Haptico.seleccion() } label: { Image(systemName: "minus").frame(width: 36, height: 36).background(C.chip, in: Circle()) }
                    .disabled(valor <= rango.lowerBound).accessibilityLabel("Menos")
                Text(unidad(valor)).font(Tipo.lato(15, .bold)).monospacedDigit().frame(minWidth: 64)
                Button { valor = min(rango.upperBound, valor + paso); Haptico.seleccion() } label: { Image(systemName: "plus").frame(width: 36, height: 36).background(C.chip, in: Circle()) }
                    .disabled(valor >= rango.upperBound).accessibilityLabel("Más")
            }
            .buttonStyle(.plain).foregroundStyle(C.tinta)
        }
        .accessibilityElement(children: .combine)
        .accessibilityValue(unidad(valor))
        .accessibilityAdjustableAction { d in
            if d == .increment { valor = min(rango.upperBound, valor + paso) } else { valor = max(rango.lowerBound, valor - paso) }
        }
    }
}

/// Un interruptor en un Bloque.
struct RenglonInterruptor: View {
    var titulo: String
    var detalle: String = ""
    @Binding var activo: Bool
    var body: some View {
        Toggle(isOn: $activo) {
            VStack(alignment: .leading, spacing: 2) {
                Text(titulo).font(Tipo.lato(16))
                if !detalle.isEmpty { Text(detalle).font(Tipo.lato(13)).foregroundStyle(C.sub) }
            }
        }
        .tint(C.menta).foregroundStyle(C.tinta).padding(.horizontal, 16).padding(.vertical, 10).frame(minHeight: 56)
    }
}

/// Un campo de texto dentro de un Bloque.
struct RenglonCampo: View {
    var titulo: String
    @Binding var texto: String
    var placeholder: String = ""
    var teclado: UIKeyboardType = .default
    var multilinea = false
    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(titulo).font(Tipo.lato(13, .bold)).foregroundStyle(C.sub)
            if multilinea {
                TextField(placeholder, text: $texto, axis: .vertical).lineLimit(2...8)
            } else {
                TextField(placeholder, text: $texto).keyboardType(teclado)
                    .textInputAutocapitalization(teclado == .emailAddress || teclado == .URL ? .never : .sentences)
            }
        }
        .font(Tipo.lato(16)).foregroundStyle(C.tinta).padding(.horizontal, 16).padding(.vertical, 10)
    }
}

/// Un precio en pesos que se escribe como número.
struct RenglonPesos: View {
    var titulo: String
    @Binding var valor: Int
    var body: some View {
        Renglon(titulo: titulo) {
            HStack(spacing: 2) {
                Text("$").foregroundStyle(C.sub)
                TextField("0", value: $valor, format: .number).keyboardType(.numberPad).multilineTextAlignment(.trailing).frame(maxWidth: 120)
            }
            .font(Tipo.lato(16, .bold)).monospacedDigit()
        }
    }
}

/// Botón de guardar fijo abajo de una hoja o sección.
struct BotonGuardar: View {
    var texto = "Guardar"
    var guardando = false
    var habilitado = true
    var accion: () async -> Void
    var body: some View {
        Button { Task { await accion() } } label: { Text(guardando ? "Guardando…" : texto) }
            .buttonStyle(BotonMenta(alto: 52, conSombra: false))
            .disabled(guardando || !habilitado)
    }
}

/// Una hoja de edición con Cancelar / Guardar.
struct HojaEdicion<Contenido: View>: View {
    var titulo: String
    var guardar: String = "Guardar"
    var habilitado = true
    var borrar: (() async -> Void)? = nil
    var accion: () async -> Bool
    @ViewBuilder var contenido: Contenido
    @Environment(\.dismiss) private var cerrar
    @State private var guardando = false
    @State private var confirmarBorrar = false

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    contenido
                    if let borrar {
                        Button(role: .destructive) { confirmarBorrar = true } label: { Label("Borrar", systemImage: "trash") }
                            .buttonStyle(BotonContorno(alto: 48, tam: 15)).foregroundStyle(C.error)
                            .confirmationDialog("¿Seguro?", isPresented: $confirmarBorrar) {
                                Button("Borrar", role: .destructive) { Task { await borrar(); cerrar() } }
                            }
                    }
                }
                .padding(16)
            }
            .scrollDismissesKeyboard(.interactively)
            .background(C.fondo)
            .navigationTitle(titulo).navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancelar") { cerrar() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button(guardando ? "…" : guardar) {
                        Task {
                            guardando = true
                            let ok = await accion()
                            guardando = false
                            if ok { Haptico.exito(); cerrar() }
                        }
                    }
                    .bold().disabled(!habilitado || guardando)
                }
            }
        }
        .presentationCornerRadius(Radio.hoja)
    }
}

extension Estado {
    /// Corre una escritura; si falla, el toast con el error. Devuelve si salió.
    @discardableResult
    func intentar(_ ok: String = "", _ f: () async throws -> Void) async -> Bool {
        do {
            try await f()
            if !ok.isEmpty { avisar(ok) }
            return true
        } catch {
            avisar(error)
            return false
        }
    }
}

// MARK: - Primeros pasos

struct PrimerosPasos: View {
    @Environment(Estado.self) private var estado
    @State private var a: Arranque?
    @State private var error = ""

    private func destino(_ clave: String) -> DestinoMas? {
        switch clave {
        case "servicios", "servicio": .servicios
        case "profesionales", "profesional", "horario", "horarios", "equipo": .equipo
        case "web", "mi_web", "miweb", "logo", "fotos", "datos": .miWeb
        case "cobros", "mercadopago", "pagos": .cobros
        case "suscripcion": .suscripcion
        case "plantilla": .plantillas
        default: nil
        }
    }

    var body: some View {
        PantallaMas(titulo: "Primeros pasos", subtitulo: a.map { $0.completo ? "Listo: ya te pueden reservar" : "\($0.pasos.filter(\.listo).count) de \($0.pasos.count) listos" } ?? "",
                    cargando: a == nil && error.isEmpty, error: error, recargar: cargar) {
            if let a {
                Bloque {
                    ForEach(a.pasos, id: \.clave) { p in
                        if let d = destino(p.clave), !p.listo {
                            NavigationLink(value: d) { paso(p) }.buttonStyle(.plain)
                        } else {
                            paso(p)
                        }
                    }
                }
                if a.reservable {
                    Label("Tu página ya recibe reservas.", systemImage: "checkmark.seal").font(Tipo.lato(15, .bold)).foregroundStyle(C.mentaTexto)
                }
            }
        }
        .task(id: estado.version) { await cargar() }
    }

    private func paso(_ p: Arranque.Paso) -> some View {
        Renglon(titulo: p.titulo, detalle: p.obligatorio ? "Hace falta para que te reserven" : "Recomendado", simbolo: p.listo ? "checkmark.circle.fill" : "circle") {
            if !p.listo && destino(p.clave) != nil { Image(systemName: "chevron.right").foregroundStyle(C.sub) }
        }
        .opacity(p.listo ? 0.6 : 1)
    }

    private func cargar() async {
        do { a = try await estado.api.get("/v1/panel/arranque"); error = "" } catch let e as ErrorApi { error = e.detalle } catch { self.error = "Probá de nuevo." }
    }
}

// MARK: - Cuenta

struct Cuenta: View {
    @Environment(Estado.self) private var estado
    @State private var salir = false
    @State private var avisosOn = false

    var body: some View {
        PantallaMas(titulo: "Tu cuenta", subtitulo: estado.persona?.email ?? "", recargar: {}) {
            Bloque {
                Renglon(titulo: estado.persona?.nombre ?? "", detalle: estado.persona?.email ?? "", simbolo: "person.crop.circle")
            }
            Bloque(titulo: "Tus negocios") {
                ForEach(estado.negocios) { n in
                    Button { estado.usar(n); Task { await estado.refrescar() } } label: {
                        Renglon(titulo: n.nombre, detalle: n.rolTexto, simbolo: n.id == estado.negocio?.id ? "checkmark.circle.fill" : "circle")
                    }
                    .buttonStyle(.plain)
                }
            }
            Bloque(titulo: "Avisos en este iPhone") {
                Button {
                    if let u = URL(string: UIApplication.openNotificationSettingsURLString) { UIApplication.shared.open(u) }
                } label: { Renglon(titulo: avisosOn ? "Activados" : "Apagados", detalle: "Cambialo en Ajustes", simbolo: avisosOn ? "bell.badge" : "bell.slash") }
                .buttonStyle(.plain)
                NavigationLink { PreferenciasPush() } label: { Renglon(titulo: "Qué te avisamos", simbolo: "slider.horizontal.3") { Image(systemName: "chevron.right").foregroundStyle(C.sub) } }
                    .buttonStyle(.plain)
            }
            Bloque(titulo: "Modo") {
                Button { Task { await estado.elegirModo("cliente") } } label: {
                    Renglon(titulo: "Usar como clienta", detalle: "Ver tus turnos y los lugares donde reservás", simbolo: "person.2.wave.2")
                }
                .buttonStyle(.plain)
            }
            Bloque(titulo: "Ayuda") {
                Link(destination: URL(string: "mailto:hola@tengolibre.com.ar")!) { Renglon(titulo: "Escribinos", detalle: "hola@tengolibre.com.ar", simbolo: "questionmark.bubble") }.buttonStyle(.plain)
            }
            Button(role: .destructive) { salir = true } label: { Label("Cerrar sesión", systemImage: "rectangle.portrait.and.arrow.right") }
                .buttonStyle(BotonContorno(alto: 52, tam: 16)).foregroundStyle(C.error)
                .confirmationDialog("¿Cerrar sesión en este iPhone?", isPresented: $salir, titleVisibility: .visible) {
                    Button("Cerrar sesión", role: .destructive) { Task { await estado.salir() } }
                }
            BorrarCuenta()
        }
        .task {
            let a = await UNUserNotificationCenter.current().notificationSettings()
            avisosOn = a.authorizationStatus == .authorized || a.authorizationStatus == .provisional
        }
    }
}

// MARK: - Borrar la cuenta

/// «Borrar mi cuenta» (para cualquier rol y también en modo clienta): pide escribir el mail de la
/// cuenta, la desactiva con DELETE /v1/auth/cuenta y vuelve al ingreso. Los datos personales se borran
/// a los 30 días; hasta entonces se recupera escribiendo a hola@.
struct BorrarCuenta: View {
    @Environment(Estado.self) private var estado
    @State private var abierta = false

    var body: some View {
        Button { abierta = true } label: { Text("Borrar mi cuenta") }
            .buttonStyle(BotonTexto(color: C.error)).frame(maxWidth: .infinity)
            .sheet(isPresented: $abierta) { HojaBorrarCuenta().presentationDetents([.large]) }
    }
}

struct HojaBorrarCuenta: View {
    @Environment(Estado.self) private var estado
    @Environment(\.dismiss) private var cerrar
    @State private var escrito = ""
    @State private var borrando = false
    @State private var error = ""

    private var email: String { estado.persona?.email ?? "" }
    private var coincide: Bool {
        !email.isEmpty && escrito.trimmingCharacters(in: .whitespaces).lowercased() == email.lowercased()
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Image(systemName: "exclamationmark.triangle").font(.system(size: 26, weight: .semibold)).foregroundStyle(C.error)
                        .frame(width: 56, height: 56).background(C.ausenteFondo, in: Circle())
                    Text("¿Borrar tu cuenta?").font(Tipo.lato(30, .black)).tracking(-0.8)
                    Text(Formato.sinViudas("Tu cuenta se desactiva ahora y tus datos personales se borran a los 30 días."))
                        .font(Tipo.lato(16, .bold))
                    Text(Formato.sinViudas("En esos 30 días la recuperás escribiendo a hola@tengolibre.com.ar. Después no se puede deshacer."))
                        .font(Tipo.lato(16)).foregroundStyle(C.sub)
                    Campo(titulo: "Para confirmar, escribí tu mail", texto: $escrito, ayuda: email, teclado: .emailAddress)
                        .textContentType(.emailAddress)
                    if !error.isEmpty { Text(error).font(Tipo.lato(14, .bold)).foregroundStyle(C.error) }
                    Button(role: .destructive) { Task { await borrar() } } label: {
                        Text(borrando ? "Borrando…" : "Borrar mi cuenta").font(Tipo.lato(17, .black))
                            .foregroundStyle(.white).frame(maxWidth: .infinity, minHeight: 52)
                            .background(C.error, in: Capsule())
                    }
                    .buttonStyle(.plain)
                    .disabled(!coincide || borrando).opacity(!coincide || borrando ? 0.45 : 1)
                }
                .foregroundStyle(C.tinta).padding(20)
            }
            .scrollDismissesKeyboard(.interactively)
            .background(C.fondo)
            .navigationTitle("Borrar mi cuenta").navigationBarTitleDisplayMode(.inline)
            .toolbar { ToolbarItem(placement: .cancellationAction) { Button("Cancelar") { cerrar() } } }
        }
        .presentationCornerRadius(Radio.hoja)
    }

    private func borrar() async {
        guard coincide, !borrando else { return }
        borrando = true
        defer { borrando = false }
        error = ""
        do {
            try await estado.borrarCuenta()
            Haptico.exito()
            cerrar()
        } catch {
            self.error = (error as? ErrorApi)?.detalle ?? ErrorApi.desconocido.detalle
            Haptico.error()
        }
    }
}
