import SwiftUI

struct Categoria: Codable, Identifiable, Equatable, Hashable { var id: String; var nombre: String; var orden: Int }

struct Miembro: Codable, Identifiable, Equatable, Hashable {
    var id: String
    var clerkId: String
    var email: String
    var nombre: String
    var rol: String
    var activo: Bool
    var creadoEn: Date
}

struct InvitacionEquipo: Codable, Identifiable, Equatable, Hashable {
    var id: String
    var email: String
    var rol: String
    var invito: String
    var venceEn: Date
    var aceptadaEn: Date?
    var creadoEn: Date
}

struct Bloqueo: Codable, Identifiable, Equatable, Hashable { var id: String; var fecha: String; var desde: String?; var hasta: String?; var motivo: String? }

let rolesEquipo: [(String, String)] = [("encargado", "Encargada"), ("recepcion", "Recepción"), ("profesional", "Profesional")]
func nombreRol(_ r: String) -> String { ["duenio": "Dueña", "encargado": "Encargada", "recepcion": "Recepción", "profesional": "Profesional"][r] ?? r }

// MARK: - Servicios (con categorías y productos)

struct Servicios: View {
    @Environment(Estado.self) private var estado
    @State private var servicios: [Servicio] = []
    @State private var categorias: [Categoria] = []
    @State private var productos: [Producto] = []
    @State private var tab = "servicios"
    @State private var cargando = true
    @State private var error = ""
    @State private var editando: Servicio?
    @State private var nuevoServicio = false
    @State private var producto: Producto?
    @State private var nuevoProducto = false
    @State private var categoriaNueva = ""

    var body: some View {
        PantallaMas(titulo: "Servicios", subtitulo: "\(servicios.filter(\.activo).count) activos",
                    acciones: [AccionCabecera(simbolo: "plus", etiqueta: "Agregar") { if tab == "productos" { nuevoProducto = true } else { nuevoServicio = true } }],
                    cargando: cargando, error: error, recargar: cargar) {
            Segmentado(opciones: [("servicios", "Servicios"), ("categorias", "Categorías"), ("productos", "Productos")], elegido: $tab)
        } contenido: {
            switch tab {
            case "categorias": vistaCategorias
            case "productos": vistaProductos
            default: vistaServicios
            }
        }
        .task(id: estado.version) { await cargar() }
        .sheet(item: $editando) { s in EditarServicio(servicio: s, categorias: categorias) { await cargar() } }
        .sheet(isPresented: $nuevoServicio) {
            EditarServicio(servicio: Servicio(id: "", nombre: "", duracion: 30, precio: 0, activo: true), categorias: categorias) { await cargar() }
        }
        .sheet(item: $producto) { p in EditarProducto(producto: p) { await cargar() } }
        .sheet(isPresented: $nuevoProducto) { EditarProducto(producto: Producto(id: "", nombre: "", precio: 0, stock: nil, activo: true, orden: productos.count + 1)) { await cargar() } }
    }

    @ViewBuilder private var vistaServicios: some View {
        if servicios.isEmpty {
            VacioQueEnsena(rayas: "Sin servicios", titulo: "Cargá tu primer servicio", texto: "Con su duración y su precio. Es lo que tus clientes eligen al reservar.") {
                Button("Agregar un servicio") { nuevoServicio = true }.buttonStyle(BotonMenta(alto: 52, conSombra: false))
            }
        }
        let grupos: [(String, [Servicio])] = categorias.sorted { $0.orden < $1.orden }.map { c in (c.nombre, servicios.filter { $0.categoriaId == c.id }) }
            + [("Sin categoría", servicios.filter { s in s.categoriaId == nil || !categorias.contains { $0.id == s.categoriaId } })]
        ForEach(grupos.filter { !$0.1.isEmpty }, id: \.0) { g in
            Bloque(titulo: categorias.isEmpty ? "" : g.0) {
                ForEach(g.1) { s in
                    Button { editando = s } label: {
                        Renglon(titulo: s.nombre, detalle: "\(Formato.duracion(s.duracion))\(s.activo ? "" : " · pausado")") {
                            Text(Formato.pesos(s.precio)).font(Tipo.lato(15, .bold)).monospacedDigit()
                            Image(systemName: "chevron.right").font(.system(size: 13, weight: .semibold)).foregroundStyle(C.sub)
                        }
                        .opacity(s.activo ? 1 : 0.55)
                    }
                    .buttonStyle(.plain)
                }
            }
        }
    }

    @ViewBuilder private var vistaCategorias: some View {
        Bloque(pie: "Agrupan los servicios en tu página (ej.: Cortes, Color, Uñas).") {
            ForEach(categorias.sorted { $0.orden < $1.orden }) { c in
                Renglon(titulo: c.nombre, detalle: Formato.cantidad(servicios.filter { $0.categoriaId == c.id }.count, "servicio", "servicios")) {
                    Menu {
                        Button("Subir", systemImage: "arrow.up") { Task { await mover(c, -1) } }
                        Button("Bajar", systemImage: "arrow.down") { Task { await mover(c, 1) } }
                        Button("Borrar", systemImage: "trash", role: .destructive) {
                            Task { await estado.intentar { try await estado.api.hacer("DELETE", "/v1/panel/categorias/\(c.id)") }; await cargar() }
                        }
                    } label: { Image(systemName: "ellipsis").frame(width: 44, height: 44) }
                }
            }
            HStack {
                TextField("Nueva categoría", text: $categoriaNueva).font(Tipo.lato(16))
                Button("Agregar") {
                    Task {
                        await estado.intentar { try await estado.api.hacer("POST", "/v1/panel/categorias", ["nombre": categoriaNueva]) }
                        categoriaNueva = ""
                        await cargar()
                    }
                }
                .buttonStyle(BotonMentaChico()).disabled(categoriaNueva.trimmingCharacters(in: .whitespaces).isEmpty)
            }
            .padding(.horizontal, 16).padding(.vertical, 8)
        }
    }

    @ViewBuilder private var vistaProductos: some View {
        if productos.isEmpty {
            VacioQueEnsena(rayas: "Sin productos", titulo: "Vendé productos en la caja", texto: "Shampoo, esmaltes, lo que vendas: se suman al cobrar un turno.") {
                Button("Agregar un producto") { nuevoProducto = true }.buttonStyle(BotonMenta(alto: 52, conSombra: false))
            }
        } else {
            Bloque {
                ForEach(productos.sorted { $0.orden < $1.orden }) { p in
                    Button { producto = p } label: {
                        Renglon(titulo: p.nombre, detalle: p.stock.map { "Stock: \($0)" } ?? "Sin control de stock") {
                            Text(Formato.pesos(p.precio)).font(Tipo.lato(15, .bold)).monospacedDigit()
                        }
                        .opacity(p.activo ? 1 : 0.55)
                    }
                    .buttonStyle(.plain)
                }
            }
        }
    }

    private func mover(_ c: Categoria, _ d: Int) async {
        var l = categorias.sorted { $0.orden < $1.orden }
        guard let i = l.firstIndex(of: c), l.indices.contains(i + d) else { return }
        l.swapAt(i, i + d)
        for (n, x) in l.enumerated() { try? await estado.api.hacer("PATCH", "/v1/panel/categorias/\(x.id)", ["orden": n + 1]) }
        await cargar()
    }

    private struct S: Decodable { var servicios: [Servicio] }
    private struct Cs: Decodable { var categorias: [Categoria] }
    private struct Ps: Decodable { var productos: [Producto] }

    private func cargar() async {
        do {
            let s: S = try await estado.api.get("/v1/panel/servicios")
            let c: Cs? = try? await estado.api.get("/v1/panel/categorias")
            let p: Ps? = try? await estado.api.get("/v1/panel/productos")
            servicios = s.servicios
            categorias = c?.categorias ?? []
            productos = p?.productos ?? []
            error = ""
        } catch let e as ErrorApi { error = e.detalle } catch { self.error = "Probá de nuevo." }
        cargando = false
    }
}

struct EditarServicio: View {
    @State var servicio: Servicio
    var categorias: [Categoria]
    var listo: () async -> Void
    @Environment(Estado.self) private var estado
    @State private var descripcion = ""

    private struct Cuerpo: Encodable { var nombre: String; var descripcion: String; var duracion: Int; var precio: Int; var categoriaId: String?; var activo: Bool }

    var body: some View {
        HojaEdicion(titulo: servicio.id.isEmpty ? "Servicio nuevo" : "Editar servicio", habilitado: servicio.nombre.trimmingCharacters(in: .whitespaces).count >= 2,
                    borrar: servicio.id.isEmpty ? nil : {
                        await estado.intentar("Sacamos el servicio") { try await estado.api.hacer("DELETE", "/v1/panel/servicios/\(servicio.id)") }
                        estado.version += 1
                    }, accion: guardar) {
            Bloque {
                RenglonCampo(titulo: "Nombre", texto: $servicio.nombre, placeholder: "Corte y brushing")
                RenglonCampo(titulo: "Descripción", texto: $descripcion, placeholder: "Opcional", multilinea: true)
                Numerito(titulo: "Duración", valor: $servicio.duracion, rango: 5...480, paso: 5) { Formato.duracion($0) }
                RenglonPesos(titulo: "Precio", valor: $servicio.precio)
            }
            if !categorias.isEmpty {
                Bloque {
                    Renglon(titulo: "Categoría") {
                        Picker("Categoría", selection: $servicio.categoriaId) {
                            Text("Sin categoría").tag(String?.none)
                            ForEach(categorias) { c in Text(c.nombre).tag(Optional(c.id)) }
                        }
                        .tint(C.mentaTexto)
                    }
                }
            }
            Bloque(pie: "Si lo pausás, no aparece en tu página ni se puede reservar.") {
                RenglonInterruptor(titulo: "Activo", activo: $servicio.activo)
            }
        }
        .onAppear { descripcion = servicio.descripcion ?? "" }
    }

    private func guardar() async -> Bool {
        let c = Cuerpo(nombre: servicio.nombre, descripcion: descripcion, duracion: servicio.duracion, precio: servicio.precio, categoriaId: servicio.categoriaId, activo: servicio.activo)
        let ok = await estado.intentar {
            if servicio.id.isEmpty { try await estado.api.hacer("POST", "/v1/panel/servicios", c) }
            else { try await estado.api.hacer("PATCH", "/v1/panel/servicios/\(servicio.id)", c) }
        }
        if ok { estado.version += 1; await listo() }
        return ok
    }
}

struct EditarProducto: View {
    @State var producto: Producto
    var listo: () async -> Void
    @Environment(Estado.self) private var estado
    @State private var conStock = false
    @State private var stock = 0

    private struct Cuerpo: Encodable { var nombre: String; var precio: Int; var stock: Int?; var activo: Bool; var orden: Int }

    var body: some View {
        HojaEdicion(titulo: producto.id.isEmpty ? "Producto nuevo" : "Editar producto", habilitado: !producto.nombre.trimmingCharacters(in: .whitespaces).isEmpty,
                    borrar: producto.id.isEmpty ? nil : {
                        await estado.intentar { try await estado.api.hacer("DELETE", "/v1/panel/productos/\(producto.id)") }
                        await listo()
                    }, accion: guardar) {
            Bloque {
                RenglonCampo(titulo: "Nombre", texto: $producto.nombre)
                RenglonPesos(titulo: "Precio", valor: $producto.precio)
                RenglonInterruptor(titulo: "Controlar stock", activo: $conStock)
                if conStock { Numerito(titulo: "Stock", valor: $stock, rango: 0...9999) }
                RenglonInterruptor(titulo: "A la venta", activo: $producto.activo)
            }
        }
        .onAppear { conStock = producto.stock != nil; stock = producto.stock ?? 0 }
    }

    private func guardar() async -> Bool {
        let c = Cuerpo(nombre: producto.nombre, precio: producto.precio, stock: conStock ? stock : nil, activo: producto.activo, orden: producto.orden)
        let ok = await estado.intentar {
            if producto.id.isEmpty { try await estado.api.hacer("POST", "/v1/panel/productos", c) }
            else { try await estado.api.hacer("PATCH", "/v1/panel/productos/\(producto.id)", c) }
        }
        if ok { await listo() }
        return ok
    }
}

// MARK: - Equipo y horarios

struct Equipo: View {
    @Environment(Estado.self) private var estado
    @State private var profesionales: [Profesional] = []
    @State private var miembros: [Miembro] = []
    @State private var invitaciones: [InvitacionEquipo] = []
    @State private var servicios: [Servicio] = []
    @State private var cargando = true
    @State private var error = ""
    @State private var nuevo = false
    @State private var invitar = false

    var body: some View {
        PantallaMas(titulo: "Equipo y horarios", subtitulo: Formato.cantidad(profesionales.filter(\.activo).count, "profesional", "profesionales"),
                    acciones: estado.gestiona ? [AccionCabecera(simbolo: "plus", etiqueta: "Sumar profesional") { nuevo = true }] : [],
                    cargando: cargando, error: error, recargar: cargar) {
            Bloque(titulo: "Quién atiende", pie: "Tocá a cada una para cargarle el horario, los servicios que hace y los días que no viene.") {
                ForEach(profesionales.sorted { $0.orden < $1.orden }) { p in
                    NavigationLink(value: DestinoMas.profesional(p.id)) {
                        HStack(spacing: 12) {
                            Iniciales(nombre: p.nombre, tam: 40)
                            VStack(alignment: .leading, spacing: 2) {
                                Text(p.nombre).font(Tipo.lato(16, .bold))
                                Text(resumenHorario(p)).font(Tipo.lato(13)).foregroundStyle(C.sub).lineLimit(1)
                            }
                            Spacer()
                            if !p.activo { Text("Pausada").font(Tipo.lato(12, .bold)).foregroundStyle(C.sub) }
                            Image(systemName: "chevron.right").font(.system(size: 13, weight: .semibold)).foregroundStyle(C.sub)
                        }
                        .foregroundStyle(C.tinta).padding(.horizontal, 16).frame(minHeight: 64)
                    }
                    .buttonStyle(.plain)
                }
            }
            if estado.gestiona {
                Bloque(titulo: "Quién entra al panel") {
                    ForEach(miembros) { m in
                        Renglon(titulo: m.nombre.isEmpty ? m.email : m.nombre, detalle: "\(nombreRol(m.rol))\(m.activo ? "" : " · sin acceso")") {
                            if m.rol != "duenio" {
                                Menu {
                                    Picker("Rol", selection: Binding(get: { m.rol }, set: { r in Task { await cambiar(m, ["rol": r]) } })) {
                                        ForEach(rolesEquipo, id: \.0) { Text($0.1).tag($0.0) }
                                    }
                                    Button(m.activo ? "Quitarle el acceso" : "Devolverle el acceso", systemImage: m.activo ? "person.slash" : "person.badge.plus") {
                                        Task { await cambiarActivo(m) }
                                    }
                                } label: { Image(systemName: "ellipsis").frame(width: 44, height: 44) }
                            }
                        }
                    }
                }
                Bloque(titulo: "Invitaciones") {
                    ForEach(invitaciones.filter { $0.aceptadaEn == nil }) { i in
                        Renglon(titulo: i.email, detalle: "\(nombreRol(i.rol)) · \(i.venceEn < Date() ? "venció" : "vence \(Formato.dia(i.venceEn))")") {
                            Button("Anular") {
                                Task { await estado.intentar { try await estado.api.hacer("DELETE", "/v1/panel/invitaciones/\(i.id)") }; await cargar() }
                            }
                            .buttonStyle(BotonTexto(color: C.error))
                        }
                    }
                    Button { invitar = true } label: { Renglon(titulo: "Invitar a alguien", simbolo: "envelope.badge.person.crop") }.buttonStyle(.plain)
                }
            }
        }
        .task(id: estado.version) { await cargar() }
        .sheet(isPresented: $nuevo) { NuevaProfesional(servicios: servicios) { await cargar() } }
        .sheet(isPresented: $invitar) { Invitar { await cargar() } }
    }

    private func resumenHorario(_ p: Profesional) -> String {
        if p.horario.isEmpty { return "Sin horario cargado" }
        let dias = Set(p.horario.map(\.dia))
        let letras = ["D", "L", "M", "M", "J", "V", "S"]
        let orden = [1, 2, 3, 4, 5, 6, 0].filter { dias.contains($0) }.map { letras[$0] }.joined(separator: " ")
        return "\(orden) · \(p.horario.map(\.desde).min() ?? "") a \(p.horario.map(\.hasta).max() ?? "")"
    }

    private func cambiar(_ m: Miembro, _ c: [String: String]) async {
        await estado.intentar("Cambiamos el rol") { try await estado.api.hacer("PATCH", "/v1/panel/equipo/\(m.id)", c) }
        await cargar()
    }

    private func cambiarActivo(_ m: Miembro) async {
        await estado.intentar(m.activo ? "Ya no entra al panel" : "Volvió a entrar al panel") { try await estado.api.hacer("PATCH", "/v1/panel/equipo/\(m.id)", ["activo": !m.activo]) }
        await cargar()
    }

    private struct P: Decodable { var profesionales: [Profesional] }
    private struct M: Decodable { var miembros: [Miembro] }
    private struct I: Decodable { var invitaciones: [InvitacionEquipo] }
    private struct S: Decodable { var servicios: [Servicio] }

    private func cargar() async {
        do {
            let p: P = try await estado.api.get("/v1/panel/profesionales")
            let m: M? = estado.gestiona ? (try? await estado.api.get("/v1/panel/equipo")) : nil
            let i: I? = estado.gestiona ? (try? await estado.api.get("/v1/panel/invitaciones")) : nil
            let s: S? = try? await estado.api.get("/v1/panel/servicios")
            profesionales = p.profesionales
            miembros = m?.miembros ?? []
            invitaciones = i?.invitaciones ?? []
            servicios = s?.servicios ?? []
            error = ""
        } catch let e as ErrorApi { error = e.detalle } catch { self.error = "Probá de nuevo." }
        cargando = false
    }
}

struct NuevaProfesional: View {
    var servicios: [Servicio]
    var listo: () async -> Void
    @Environment(Estado.self) private var estado
    @Environment(Navegacion.self) private var nav
    @State private var nombre = ""
    @State private var especialidad = ""
    @State private var email = ""
    @State private var telefono = ""
    @State private var hace: Set<String> = []

    private struct Cuerpo: Encodable { var nombre: String; var especialidad: String; var email: String; var telefono: String; var servicios: [String]; var activo = true }

    var body: some View {
        HojaEdicion(titulo: "Sumar profesional", guardar: "Sumar", habilitado: nombre.trimmingCharacters(in: .whitespaces).count >= 2, accion: guardar) {
            Bloque {
                RenglonCampo(titulo: "Nombre", texto: $nombre)
                RenglonCampo(titulo: "Especialidad", texto: $especialidad, placeholder: "Colorista, manicura…")
                RenglonCampo(titulo: "Email (para avisarle sus turnos)", texto: $email, teclado: .emailAddress)
                RenglonCampo(titulo: "Teléfono", texto: $telefono, teclado: .phonePad)
            }
            Bloque(titulo: "Qué servicios hace") {
                ForEach(servicios.filter(\.activo)) { s in
                    RenglonInterruptor(titulo: s.nombre, activo: Binding(get: { hace.contains(s.id) }, set: { if $0 { hace.insert(s.id) } else { hace.remove(s.id) } }))
                }
            }
        }
        .onAppear { hace = Set(servicios.filter(\.activo).map(\.id)) }
    }

    private func guardar() async -> Bool {
        do {
            let p: Profesional = try await estado.api.enviar("POST", "/v1/panel/profesionales",
                Cuerpo(nombre: nombre, especialidad: especialidad, email: email, telefono: telefono, servicios: Array(hace)))
            await listo()
            nav.pilaMas.append(DestinoMas.profesional(p.id))
            estado.avisar("Sumamos a \(p.nombre)", texto: "Ahora cargale el horario.")
            return true
        } catch { estado.avisar(error); return false }
    }
}

struct Invitar: View {
    var listo: () async -> Void
    @Environment(Estado.self) private var estado
    @State private var email = ""
    @State private var rol = "recepcion"

    var body: some View {
        HojaEdicion(titulo: "Invitar", guardar: "Mandar", habilitado: email.contains("@"), accion: {
            let ok = await estado.intentar("Le mandamos la invitación a \(email)") {
                try await estado.api.hacer("POST", "/v1/panel/invitaciones", ["email": email.trimmingCharacters(in: .whitespaces), "rol": rol])
            }
            if ok { await listo() }
            return ok
        }) {
            Bloque(pie: "Le llega un mail para entrar con su cuenta de Google o Apple.") {
                RenglonCampo(titulo: "Email", texto: $email, teclado: .emailAddress)
                Renglon(titulo: "Rol") {
                    Picker("Rol", selection: $rol) { ForEach(rolesEquipo, id: \.0) { Text($0.1).tag($0.0) } }.tint(C.mentaTexto)
                }
            }
            VStack(alignment: .leading, spacing: 6) {
                Text("Encargada: todo menos la suscripción.").font(Tipo.lato(13))
                Text("Recepción: turnos, clientes y caja, sin ajustes.").font(Tipo.lato(13))
                Text("Profesional: sólo sus turnos y su horario.").font(Tipo.lato(13))
            }
            .foregroundStyle(C.sub).padding(.horizontal, 4)
        }
    }
}

/// «Mi horario» de la profesional: su propia ficha.
struct MiHorario: View {
    @Environment(Estado.self) private var estado
    @State private var id: String?
    private struct P: Decodable { var profesionales: [Profesional] }
    var body: some View {
        Group {
            if let id { FichaProfesional(id: id, propia: true) }
            else { PantallaMas(titulo: "Mi horario", cargando: true, recargar: {}) { EmptyView() } }
        }
        .task {
            let r: P? = try? await estado.api.get("/v1/panel/profesionales")
            let mail = estado.persona?.email.lowercased() ?? ""
            id = (r?.profesionales.first { ($0.email ?? "").lowercased() == mail } ?? r?.profesionales.first)?.id
        }
    }
}

/// Una profesional: datos, servicios que hace, horario semanal, bloqueos y días libres.
struct FichaProfesional: View {
    var id: String
    var propia = false
    @Environment(Estado.self) private var estado
    @State private var p: Profesional?
    @State private var servicios: [Servicio] = []
    @State private var bloqueos: [Bloqueo] = []
    @State private var pedidos: [PedidoDias] = []
    @State private var error = ""
    @State private var horario = false
    @State private var bloquear = false
    @State private var datos = false
    @State private var pedirLibres = false

    var body: some View {
        PantallaMas(titulo: p?.nombre ?? (propia ? "Mi horario" : "Profesional"), subtitulo: p?.especialidad ?? "", volver: propia ? "Más" : "Equipo",
                    acciones: estado.gestiona && p != nil ? [AccionCabecera(simbolo: "pencil", etiqueta: "Editar datos") { datos = true }] : [],
                    cargando: p == nil && error.isEmpty, error: error, recargar: cargar) {
            if let p {
                Bloque(titulo: "Horario de la semana") {
                    ForEach([1, 2, 3, 4, 5, 6, 0], id: \.self) { d in
                        let fr = p.horario.filter { $0.dia == d }.sorted { $0.desde < $1.desde }
                        Renglon(titulo: EditorHorario.nombres[d]) {
                            Text(fr.isEmpty ? "No atiende" : fr.map { "\($0.desde) a \($0.hasta)" }.joined(separator: " · "))
                                .font(Tipo.lato(14, fr.isEmpty ? .regular : .bold)).foregroundStyle(fr.isEmpty ? C.sub : C.tinta).monospacedDigit()
                        }
                    }
                    if estado.gestiona || propia {
                        Button { horario = true } label: { Renglon(titulo: "Cambiar el horario", simbolo: "clock.badge") }.buttonStyle(.plain)
                    }
                }
                Bloque(titulo: "Días u horas que no atiende", pie: "Feriados, turnos médicos, vacaciones cortas. Esos horarios no se pueden reservar.") {
                    ForEach(bloqueos.sorted { $0.fecha < $1.fecha }) { b in
                        Renglon(titulo: Fecha.iso(b.fecha).map { Formato.fechaLarga($0) } ?? b.fecha,
                                detalle: [b.desde.map { "\($0) a \(b.hasta ?? "")" } ?? "Todo el día", b.motivo ?? ""].filter { !$0.isEmpty }.joined(separator: " · ")) {
                            Button { Task { await borrarBloqueo(b) } } label: { Image(systemName: "trash").foregroundStyle(C.error).frame(width: 44, height: 44) }
                                .accessibilityLabel("Sacar el bloqueo")
                        }
                    }
                    Button { bloquear = true } label: { Renglon(titulo: "Bloquear un día u horario", simbolo: "calendar.badge.minus") }.buttonStyle(.plain)
                }
                if estado.gestiona {
                    Bloque(titulo: "Servicios que hace") {
                        ForEach(servicios.filter(\.activo)) { s in
                            RenglonInterruptor(titulo: s.nombre, activo: Binding(get: { p.servicios.isEmpty || p.servicios.contains(s.id) }, set: { v in Task { await servicio(s.id, v) } }))
                        }
                    }
                }
                Bloque(titulo: "Días libres", pie: propia ? "Pedilos acá: le llega a la dueña y, si los aprueba, te avisamos." : "") {
                    ForEach(pedidos.filter { $0.profesionalId == p.id }) { d in
                        Renglon(titulo: "\(d.motivoTexto) · \(rangoDias(d))", detalle: "\(estadoPedido(d.estado)) · \(Formato.cantidad(d.turnos, "turno", "turnos")) para mover") {
                            if d.estado == "pendiente" && propia {
                                Button("Cancelar") { Task { await estado.intentar { _ = try await estado.api.pedir("POST", "/v1/panel/dias-libres/\(d.id)/cancelar", cuerpo: Data("{}".utf8)) }; await cargar() } }
                                    .buttonStyle(BotonTexto(color: C.error))
                            }
                        }
                    }
                    Button { pedirLibres = true } label: { Renglon(titulo: propia ? "Pedir días libres" : "Cargar días libres", simbolo: "beach.umbrella") }.buttonStyle(.plain)
                }
                if estado.gestiona {
                    Bloque {
                        RenglonInterruptor(titulo: "Atiende", detalle: "Si la pausás, no aparece para reservar", activo: Binding(get: { p.activo }, set: { v in Task { await patch(["activo": v]) } }))
                    }
                }
            }
        }
        .task(id: estado.version) { await cargar() }
        .sheet(isPresented: $horario) { if let p { EditorHorario(profesional: p) { await cargar() } } }
        .sheet(isPresented: $bloquear) { NuevoBloqueo(profesionalId: id) { await cargar() } }
        .sheet(isPresented: $datos) { if let p { DatosProfesional(p: p) { await cargar() } } }
        .sheet(isPresented: $pedirLibres) { PedirDiasLibres(profesionalId: id) { await cargar() } }
    }

    private func rangoDias(_ d: PedidoDias) -> String {
        guard let a = Fecha.iso(d.desde), let b = Fecha.iso(d.hasta) else { return "\(d.desde) a \(d.hasta)" }
        return d.desde == d.hasta ? Formato.dia(a) : "\(Formato.dia(a)) al \(Formato.dia(b))"
    }

    private func estadoPedido(_ e: String) -> String { ["pendiente": "Esperando respuesta", "aprobado": "Aprobado", "rechazado": "Rechazado", "cancelado": "Cancelado"][e] ?? e }

    private struct Ps: Decodable { var profesionales: [Profesional] }
    private struct Ss: Decodable { var servicios: [Servicio] }
    private struct Bs: Decodable { var bloqueos: [Bloqueo] }
    private struct Ds: Decodable { var pedidos: [PedidoDias] }

    private func cargar() async {
        do {
            let ps: Ps = try await estado.api.get("/v1/panel/profesionales")
            let ss: Ss? = try? await estado.api.get("/v1/panel/servicios")
            let bs: Bs? = try? await estado.api.get("/v1/panel/profesionales/\(id)/bloqueos")
            let ds: Ds? = try? await estado.api.get("/v1/panel/dias-libres")
            guard let x = ps.profesionales.first(where: { $0.id == id }) else { error = "No encontramos a esa profesional."; return }
            p = x
            servicios = ss?.servicios ?? []
            bloqueos = bs?.bloqueos ?? []
            pedidos = ds?.pedidos ?? []
            error = ""
        } catch let e as ErrorApi { error = e.detalle } catch { self.error = "Probá de nuevo." }
    }

    private func borrarBloqueo(_ b: Bloqueo) async {
        await estado.intentar { try await estado.api.hacer("DELETE", "/v1/panel/profesionales/\(id)/bloqueos/\(b.id)") }
        estado.version += 1
    }

    private func servicio(_ sid: String, _ on: Bool) async {
        guard let p else { return }
        var l = p.servicios.isEmpty ? servicios.filter(\.activo).map(\.id) : p.servicios
        if on { if !l.contains(sid) { l.append(sid) } } else { l.removeAll { $0 == sid } }
        await estado.intentar { try await estado.api.hacer("PUT", "/v1/panel/profesionales/\(id)/servicios", ["servicios": l]) }
        await cargar()
    }

    private func patch(_ c: [String: Bool]) async {
        await estado.intentar { try await estado.api.hacer("PATCH", "/v1/panel/profesionales/\(id)", c) }
        estado.version += 1
    }
}

struct DatosProfesional: View {
    @State var p: Profesional
    var listo: () async -> Void
    @Environment(Estado.self) private var estado
    @State private var email = ""
    @State private var telefono = ""
    @State private var bio = ""

    private struct Cuerpo: Encodable { var nombre: String; var especialidad: String; var email: String; var telefono: String; var bio: String }

    var body: some View {
        HojaEdicion(titulo: "Datos", accion: {
            let ok = await estado.intentar {
                try await estado.api.hacer("PATCH", "/v1/panel/profesionales/\(p.id)", Cuerpo(nombre: p.nombre, especialidad: p.especialidad, email: email, telefono: telefono, bio: bio))
            }
            if ok { await listo() }
            return ok
        }) {
            Bloque {
                RenglonCampo(titulo: "Nombre", texto: $p.nombre)
                RenglonCampo(titulo: "Especialidad", texto: $p.especialidad)
                RenglonCampo(titulo: "Email", texto: $email, teclado: .emailAddress)
                RenglonCampo(titulo: "Teléfono", texto: $telefono, teclado: .phonePad)
                RenglonCampo(titulo: "Sobre ella (se ve en tu página)", texto: $bio, multilinea: true)
            }
        }
        .onAppear { email = p.email ?? ""; telefono = p.telefono ?? ""; bio = p.bio ?? "" }
    }
}

/// El horario semanal: por día, una o dos franjas (mañana y tarde), con «copiar a todos».
struct EditorHorario: View {
    static let nombres = ["Domingo", "Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado"]
    var profesional: Profesional
    var listo: () async -> Void
    @Environment(Estado.self) private var estado
    @State private var franjas: [Franja] = []

    var body: some View {
        HojaEdicion(titulo: "Horario", accion: guardar) {
            ForEach([1, 2, 3, 4, 5, 6, 0], id: \.self) { d in
                let del = franjas.indices.filter { franjas[$0].dia == d }
                Bloque(titulo: Self.nombres[d]) {
                    RenglonInterruptor(titulo: del.isEmpty ? "No atiende" : "Atiende", activo: Binding(get: { !del.isEmpty }, set: { on in
                        if on { franjas.append(Franja(dia: d, desde: "09:00", hasta: "18:00")) } else { franjas.removeAll { $0.dia == d } }
                    }))
                    ForEach(del, id: \.self) { i in
                        HStack(spacing: 8) {
                            hora($franjas[i].desde)
                            Text("a").foregroundStyle(C.sub)
                            hora($franjas[i].hasta)
                            Spacer()
                            if del.count > 1 {
                                Button { franjas.remove(at: i) } label: { Image(systemName: "minus.circle").foregroundStyle(C.error).frame(width: 44, height: 44) }
                            }
                        }
                        .padding(.horizontal, 16).padding(.vertical, 4)
                    }
                    if !del.isEmpty {
                        HStack {
                            if del.count < 3 {
                                Button("Sumar un corte") { franjas.append(Franja(dia: d, desde: "15:00", hasta: "20:00")) }.buttonStyle(BotonTexto())
                            }
                            Spacer()
                            Button("Copiar a todos los días que atiende") {
                                let base = franjas.filter { $0.dia == d }
                                let dias = Set(franjas.map(\.dia))
                                franjas = franjas.filter { $0.dia == d } + dias.subtracting([d]).flatMap { x in base.map { Franja(dia: x, desde: $0.desde, hasta: $0.hasta) } }
                            }
                            .buttonStyle(BotonTexto())
                        }
                        .padding(.horizontal, 16).padding(.vertical, 6).font(Tipo.lato(13))
                    }
                }
            }
        }
        .onAppear { franjas = profesional.horario }
    }

    private func hora(_ b: Binding<String>) -> some View {
        DatePicker("", selection: Binding(get: {
            Dia.fecha(Formato.calendario.startOfDay(for: Date()), minutos: Dia.aMin(b.wrappedValue))
        }, set: { f in
            let m = Formato.minutosDelDia(f)
            b.wrappedValue = String(format: "%02d:%02d", m / 60, m % 60)
        }), displayedComponents: .hourAndMinute)
        .labelsHidden()
        .environment(\.timeZone, Formato.zona)
    }

    private func guardar() async -> Bool {
        if let mal = franjas.first(where: { Dia.aMin($0.desde) >= Dia.aMin($0.hasta) }) {
            estado.avisar("Revisá el \(Self.nombres[mal.dia].lowercased())", texto: "La hora de cierre tiene que ser después de la de apertura.", ok: false)
            return false
        }
        let ok = await estado.intentar("Guardamos el horario") {
            try await estado.api.hacer("PUT", "/v1/panel/profesionales/\(profesional.id)/horario", ["horario": franjas])
        }
        if ok { estado.version += 1; await listo() }
        return ok
    }
}

struct NuevoBloqueo: View {
    var profesionalId: String
    var listo: () async -> Void
    @Environment(Estado.self) private var estado
    @State private var desde = Date()
    @State private var hastaDia = Date()
    @State private var varios = false
    @State private var entero = true
    @State private var horaDesde = Dia.fecha(Date(), minutos: 9 * 60)
    @State private var horaHasta = Dia.fecha(Date(), minutos: 13 * 60)
    @State private var motivo = ""

    private struct Cuerpo: Encodable { var fecha: String; var desde: String?; var hasta: String?; var motivo: String }

    var body: some View {
        HojaEdicion(titulo: "Bloquear", guardar: "Bloquear", accion: guardar) {
            Bloque {
                DatePicker("Día", selection: $desde, in: Formato.calendario.startOfDay(for: Date())..., displayedComponents: .date).padding(.horizontal, 16).frame(minHeight: 52)
                RenglonInterruptor(titulo: "Varios días seguidos", activo: $varios)
                if varios { DatePicker("Hasta", selection: $hastaDia, in: desde..., displayedComponents: .date).padding(.horizontal, 16).frame(minHeight: 52) }
                RenglonInterruptor(titulo: "Todo el día", activo: $entero)
                if !entero {
                    DatePicker("Desde", selection: $horaDesde, displayedComponents: .hourAndMinute).padding(.horizontal, 16).frame(minHeight: 52)
                    DatePicker("Hasta", selection: $horaHasta, displayedComponents: .hourAndMinute).padding(.horizontal, 16).frame(minHeight: 52)
                }
                RenglonCampo(titulo: "Motivo (opcional)", texto: $motivo, placeholder: "Feriado, médico…")
            }
            .font(Tipo.lato(16)).tint(C.mentaTexto)
        }
    }

    private func guardar() async -> Bool {
        var dias: [Date] = [desde]
        if varios {
            var d = desde
            while let s = Formato.calendario.date(byAdding: .day, value: 1, to: d), Formato.diaISO(s) <= Formato.diaISO(hastaDia), dias.count < 120 { dias.append(s); d = s }
        }
        let ok = await estado.intentar(dias.count > 1 ? "Bloqueamos esos \(dias.count) días" : "Bloqueamos esa fecha") {
            for d in dias {
                try await estado.api.hacer("POST", "/v1/panel/profesionales/\(profesionalId)/bloqueos", Cuerpo(
                    fecha: Formato.diaISO(d),
                    desde: entero ? nil : String(format: "%02d:%02d", Formato.minutosDelDia(horaDesde) / 60, Formato.minutosDelDia(horaDesde) % 60),
                    hasta: entero ? nil : String(format: "%02d:%02d", Formato.minutosDelDia(horaHasta) / 60, Formato.minutosDelDia(horaHasta) % 60),
                    motivo: motivo))
            }
        }
        if ok { estado.version += 1; await listo() }
        return ok
    }
}

struct PedirDiasLibres: View {
    var profesionalId: String
    var listo: () async -> Void
    @Environment(Estado.self) private var estado
    @State private var desde = Date()
    @State private var hasta = Date()
    @State private var motivo = "vacaciones"
    @State private var nota = ""

    private struct Cuerpo: Encodable { var profesionalId: String; var desde: String; var hasta: String; var motivo: String; var nota: String }

    var body: some View {
        HojaEdicion(titulo: "Días libres", guardar: estado.esProfesional ? "Pedir" : "Cargar", accion: {
            let ok = await estado.intentar(estado.esProfesional ? "Mandamos el pedido" : "Cargamos los días libres") {
                try await estado.api.hacer("POST", "/v1/panel/dias-libres", Cuerpo(profesionalId: profesionalId, desde: Formato.diaISO(desde), hasta: Formato.diaISO(max(desde, hasta)), motivo: motivo, nota: nota))
            }
            if ok { estado.version += 1; await listo() }
            return ok
        }) {
            Bloque(pie: "Si hay turnos esos días, quedan para mover: le avisamos a cada clienta cuando los muevas.") {
                DatePicker("Desde", selection: $desde, in: Formato.calendario.startOfDay(for: Date())..., displayedComponents: .date).padding(.horizontal, 16).frame(minHeight: 52)
                DatePicker("Hasta", selection: $hasta, in: desde..., displayedComponents: .date).padding(.horizontal, 16).frame(minHeight: 52)
                Renglon(titulo: "Motivo") {
                    Picker("Motivo", selection: $motivo) {
                        Text("Vacaciones").tag("vacaciones"); Text("Trámite").tag("tramite"); Text("Personal").tag("personal"); Text("Otro").tag("otro")
                    }
                    .tint(C.mentaTexto)
                }
                RenglonCampo(titulo: "Nota", texto: $nota, multilinea: true)
            }
            .tint(C.mentaTexto)
        }
    }
}

/// Los pedidos de días libres del equipo, para aprobar o rechazar, y los turnos que quedaron para mover.
struct DiasLibres: View {
    @Environment(Estado.self) private var estado
    @Environment(Navegacion.self) private var nav
    @State private var pedidos: [PedidoDias] = []
    @State private var mover: [AReprogramar] = []
    @State private var cargando = true
    @State private var tab = "pendiente"

    var body: some View {
        PantallaMas(titulo: "Días libres", subtitulo: Formato.cantidad(pedidos.filter { $0.estado == "pendiente" }.count, "pedido sin responder", "pedidos sin responder"),
                    cargando: cargando, recargar: cargar) {
            Segmentado(opciones: [("pendiente", "Pendientes"), ("aprobado", "Aprobados"), ("todos", "Todos")], elegido: $tab)
        } contenido: {
            let lista = pedidos.filter { tab == "todos" || $0.estado == tab }
            if lista.isEmpty {
                VacioQueEnsena(rayas: "Nada por acá", titulo: "No hay pedidos", texto: "Cuando alguien del equipo pida días libres desde su app, aparece acá para aprobarlo.") { EmptyView() }
            }
            ForEach(lista) { d in
                VStack(alignment: .leading, spacing: 10) {
                    HStack {
                        Iniciales(nombre: d.profesional, tam: 40)
                        VStack(alignment: .leading, spacing: 2) {
                            Text(d.profesional).font(Tipo.lato(16, .bold))
                            Text("\(d.motivoTexto) · \(d.desde == d.hasta ? (Fecha.iso(d.desde).map { Formato.dia($0) } ?? d.desde) : "\(Fecha.iso(d.desde).map { Formato.dia($0) } ?? d.desde) al \(Fecha.iso(d.hasta).map { Formato.dia($0) } ?? d.hasta)")")
                                .font(Tipo.lato(13)).foregroundStyle(C.sub)
                        }
                    }
                    if !d.nota.isEmpty { Text("«\(d.nota)»").font(Tipo.lato(14)) }
                    Text(d.afectados > 0 ? "\(Formato.cantidad(d.afectados, "turno", "turnos")) para mover" : "No hay turnos esos días").font(Tipo.lato(13, .bold))
                        .foregroundStyle(d.afectados > 0 ? C.pagoTinta : C.mentaTexto)
                    if d.estado == "pendiente" {
                        HStack(spacing: 8) {
                            Button("Rechazar") { Task { await resolver(d, "rechazar") } }.buttonStyle(BotonContorno(alto: 44, tam: 15))
                            Button("Aprobar") { Task { await resolver(d, "aprobar") } }.buttonStyle(BotonMenta(alto: 44, tam: 15, conSombra: false))
                        }
                    } else {
                        Text(["aprobado": "Aprobado", "rechazado": "Rechazado", "cancelado": "Cancelado"][d.estado] ?? d.estado).rotulo()
                    }
                }
                .foregroundStyle(C.tinta).padding(16).tarjeta()
            }
            if !mover.isEmpty {
                Bloque(titulo: "Turnos para mover") {
                    ForEach(mover) { r in
                        Button {
                            Task { if let t: Turno = try? await estado.api.get("/v1/panel/turnos/\(r.turno.id)") { nav.turno = t } }
                        } label: {
                            Renglon(titulo: r.turno.cliente.nombre, detalle: "\(Formato.diaYHora(r.turno.inicio)) · \(r.turno.profesional)") {
                                Text("Mover").font(Tipo.lato(14, .bold)).foregroundStyle(C.mentaTexto)
                            }
                        }
                        .buttonStyle(.plain)
                    }
                }
            }
        }
        .task(id: estado.version) { await cargar() }
    }

    private struct Ds: Decodable { var pedidos: [PedidoDias] }
    private struct Rs: Decodable { var turnos: [AReprogramar] }

    private func cargar() async {
        pedidos = (try? await estado.api.get("/v1/panel/dias-libres") as Ds)?.pedidos ?? pedidos
        mover = (try? await estado.api.get("/v1/panel/reprogramar") as Rs)?.turnos ?? mover
        cargando = false
    }

    private func resolver(_ d: PedidoDias, _ que: String) async {
        await estado.intentar(que == "aprobar" ? "Aprobado. Le avisamos a \(d.profesional)" : "Rechazado. Le avisamos a \(d.profesional)") {
            _ = try await estado.api.pedir("POST", "/v1/panel/dias-libres/\(d.id)/\(que)", cuerpo: Data("{}".utf8))
        }
        estado.version += 1
    }
}
