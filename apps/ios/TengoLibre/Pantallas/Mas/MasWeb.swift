import Charts
import PhotosUI
import SwiftUI
import UserNotifications

// MARK: - Reportes

struct Reportes: View {
    struct Fila: Codable, Hashable { var id: String?; var nombre: String; var turnos: Int; var atendidos: Int; var ausentes: Int; var ingresos: Int; var minutosReservados: Int; var ocupacion: Double? }
    struct DiaR: Codable, Hashable { var fecha: String; var turnos: Int; var atendidos: Int; var ingresos: Int }
    struct Resumen: Codable { var turnos: Int; var atendidos: Int; var ingresos: Int; var ticketPromedio: Int; var clientes: Int; var tasaAusencias: Double?; var ocupacion: Double? }
    struct Reporte: Codable { var desde: String; var hasta: String; var resumen: Resumen; var porEstado: [String: Int]; var porProfesional: [Fila]; var porServicio: [Fila]; var porDia: [DiaR]; var clientesNuevos: Int; var clientesRecurrentes: Int }
    struct Peor: Codable { var texto: String }
    struct Sugerencia: Codable { var porcentaje: Int; var texto: String; var destinatarios: Int; var codigo: String }
    struct HorasMuertas: Codable { var peor: Peor?; var sugerencia: Sugerencia?; var mapa: MapaCalor.Mapa? }

    @Environment(Estado.self) private var estado
    @State private var periodo = "mes"
    @State private var r: Reporte?
    @State private var antes: Reporte?
    @State private var hm: HorasMuertas?
    @State private var error = ""

    private var rango: (Date, Date) {
        let cal = Formato.calendario, hoy = cal.startOfDay(for: Date())
        switch periodo {
        case "semana": return (cal.date(byAdding: .day, value: -6, to: hoy)!, hoy)
        case "trimestre": return (cal.date(byAdding: .day, value: -89, to: hoy)!, hoy)
        default: return (cal.date(from: cal.dateComponents([.year, .month], from: hoy))!, hoy)
        }
    }

    var body: some View {
        PantallaMas(titulo: "Reportes", subtitulo: "\(Formato.dia(rango.0)) al \(Formato.dia(rango.1))", cargando: r == nil && error.isEmpty, error: error, recargar: cargar) {
            Segmentado(opciones: [("semana", "7 días"), ("mes", "Este mes"), ("trimestre", "90 días")], elegido: $periodo)
        } contenido: {
            if let r {
                LazyVGrid(columns: [GridItem(.flexible(), spacing: 10), GridItem(.flexible(), spacing: 10)], spacing: 10) {
                    cifra("Ingresos", Formato.pesos(r.resumen.ingresos), antes.map { cambio(r.resumen.ingresos, $0.resumen.ingresos) }).entrada(0)
                    cifra("Turnos atendidos", "\(r.resumen.atendidos)", antes.map { cambio(r.resumen.atendidos, $0.resumen.atendidos) }).entrada(1)
                    cifra("Ticket promedio", Formato.pesos(r.resumen.ticketPromedio), nil).entrada(2)
                    cifra("Ocupación", r.resumen.ocupacion.map { Formato.porcentaje($0) } ?? "—", nil).entrada(3)
                    cifra("Clientas", "\(r.resumen.clientes)", "\(r.clientesNuevos) nuevas · \(r.clientesRecurrentes) vuelven").entrada(4)
                    cifra("Ausencias", r.resumen.tasaAusencias.map { Formato.porcentaje($0) } ?? "—", nil).entrada(5)
                }
                if !r.porDia.isEmpty { grafico(r.porDia).entrada(6) }
                estados(r).entrada(7)
                tabla("Por profesional", r.porProfesional).entrada(8)
                tabla("Por servicio", r.porServicio).entrada(8)
                if let hm, let p = hm.peor {
                    Bloque(titulo: "Horas muertas") {
                        Renglon(titulo: p.texto, simbolo: "moon.zzz")
                        if let m = hm.mapa, !m.celdas.isEmpty { MapaCalor(mapa: m).padding(16) }
                        if let s = hm.sugerencia {
                            VStack(alignment: .leading, spacing: 8) {
                                Text(s.texto).font(Tipo.lato(14)).foregroundStyle(C.sub)
                                Button("Crear el cupón \(s.codigo) (\(s.porcentaje) %)") {
                                    Task { await estado.intentar("Creamos el cupón \(s.codigo)") { _ = try await estado.api.pedir("POST", "/v1/panel/reportes/horas-muertas/cupon", cuerpo: Data("{}".utf8)) } }
                                }
                                .buttonStyle(BotonMenta(alto: 44, tam: 15, conSombra: false))
                            }
                            .padding(16)
                        }
                    }
                }
            }
        }
        .task(id: periodo) { await cargar() }
    }

    private func cambio(_ a: Int, _ b: Int) -> String {
        guard b > 0 else { return "" }
        let p = Double(a - b) / Double(b) * 100
        return "\(p >= 0 ? "▲" : "▼") \(Int(abs(p).rounded())) % vs. antes"
    }

    private func cifra(_ t: String, _ v: String, _ d: String?) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(t.uppercased()).rotulo()
            Text(v).font(Tipo.lato(24, .black)).monospacedDigit().minimumScaleFactor(0.6).lineLimit(1).numeroAnimado(v)
            if let d, !d.isEmpty { Text(d).font(Tipo.lato(12, .bold)).foregroundStyle(d.hasPrefix("▼") ? C.ausenteTinta : C.mentaTexto) }
        }
        .foregroundStyle(C.tinta).padding(14).frame(maxWidth: .infinity, minHeight: 96, alignment: .topLeading).tarjeta(18)
    }

    private func grafico(_ dias: [DiaR]) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Text("INGRESOS POR DÍA").rotulo()
                Spacer()
                Text("Tocá una barra").font(Tipo.lato(11)).foregroundStyle(C.sub)
            }
            GraficoIngresos(dias: dias.compactMap { d in Fecha.iso(d.fecha).map { GraficoIngresos.Dia(fecha: $0, monto: d.ingresos) } })
        }
        .padding(16).tarjeta()
        .id(periodo)
    }

    /// Cómo terminaron los turnos del período (dona).
    @ViewBuilder private func estados(_ r: Reporte) -> some View {
        let nombres: [(String, String, Color)] = [("atendido", "Atendidos", C.menta), ("confirmado", "Por venir", C.petroleo), ("pendiente", "Sin confirmar", Color(hex: 0xE8A317)),
                                                   ("ausente", "No vinieron", C.ausenteTinta), ("cancelado", "Cancelados", C.sub)]
        let partes = nombres.map { GraficoDona.Parte(nombre: $0.1, valor: r.porEstado[$0.0] ?? 0, color: $0.2) }.filter { $0.valor > 0 }
        if !partes.isEmpty {
            VStack(alignment: .leading, spacing: 12) {
                Text("CÓMO TERMINARON LOS TURNOS").rotulo()
                GraficoDona(partes: partes, centro: "\(r.resumen.turnos)", pie: "turnos", mostrarPesos: false)
            }
            .padding(16).tarjeta()
            .id("estados-\(periodo)")
        }
    }

    @ViewBuilder private func tabla(_ t: String, _ f: [Fila]) -> some View {
        if !f.isEmpty {
            let orden = f.sorted { $0.ingresos > $1.ingresos }
            VStack(alignment: .leading, spacing: 12) {
                Text(t.uppercased()).rotulo()
                GraficoBarras(filas: orden.prefix(8).map { GraficoBarras.Fila(nombre: $0.nombre, valor: $0.ingresos) }, formato: { Formato.pesosCortos($0) })
                    .id("\(t)-\(periodo)")
                ForEach(orden.prefix(8), id: \.self) { x in
                    HStack {
                        Text(x.nombre).font(Tipo.lato(13, .bold)).lineLimit(1)
                        Spacer()
                        Text("\(x.atendidos) atendidos\(x.ausentes > 0 ? " · \(x.ausentes) no vinieron" : "")\(x.ocupacion.map { " · \(Formato.porcentaje($0))" } ?? "")")
                            .font(Tipo.lato(12)).foregroundStyle(C.sub)
                    }
                }
            }
            .foregroundStyle(C.tinta).padding(16).tarjeta()
        }
    }

    private func cargar() async {
        let (d, h) = rango
        let largo = Formato.calendario.dateComponents([.day], from: d, to: h).day ?? 0
        let ad = Formato.calendario.date(byAdding: .day, value: -(largo + 1), to: d)!, ah = Formato.calendario.date(byAdding: .day, value: -1, to: d)!
        do {
            r = try await estado.api.get("/v1/panel/reportes", query: [URLQueryItem(name: "desde", value: Formato.diaISO(d)), URLQueryItem(name: "hasta", value: Formato.diaISO(h))])
            antes = try? await estado.api.get("/v1/panel/reportes", query: [URLQueryItem(name: "desde", value: Formato.diaISO(ad)), URLQueryItem(name: "hasta", value: Formato.diaISO(ah))])
            hm = try? await estado.api.get("/v1/panel/reportes/horas-muertas")
            error = ""
        } catch let e as ErrorApi { error = e.detalle } catch { self.error = "Probá de nuevo." }
    }
}

// MARK: - Mi web

struct NegocioDatos: Codable, Equatable {
    var id: String; var slug: String; var nombre: String; var descripcion: String; var direccion: String; var ciudad: String
    var telefono: String; var whatsapp: String; var instagram: String; var logo: String?; var zonaHoraria: String
}

struct WebDatos: Codable, Equatable {
    var rubro: String; var provincia: String; var acerca: String; var portada: String?; var colorPrimario: String; var colorAcento: String
    var email: String; var sitioWeb: String; var facebook: String; var tiktok: String; var latitud: Double?; var longitud: Double?
    var estilo: ValorJSON?
}

struct FotoGaleria: Codable, Identifiable, Equatable, Hashable { var id: String; var url: String; var epigrafe: String; var orden: Int }

struct MiWeb: View {
    @Environment(Estado.self) private var estado
    @State private var n: NegocioDatos?
    @State private var w: WebDatos?
    @State private var fotos: [FotoGaleria] = []
    @State private var error = ""
    @State private var logo: PhotosPickerItem?
    @State private var portada: PhotosPickerItem?
    @State private var foto: PhotosPickerItem?
    @State private var subiendo = false

    var body: some View {
        PantallaMas(titulo: "Mi web", subtitulo: Web.pagina(estado.negocio?.slug ?? "").absoluteString.replacingOccurrences(of: "https://", with: ""),
                    acciones: [AccionCabecera(simbolo: "safari", etiqueta: "Ver mi página") { UIApplication.shared.open(Web.pagina(estado.negocio?.slug ?? "")) }],
                    cargando: (n == nil || w == nil) && error.isEmpty, error: error, recargar: cargar) {
            if n != nil, w != nil {
                let nb = Binding(get: { n! }, set: { n = $0 })
                let wb = Binding(get: { w! }, set: { w = $0 })
                Bloque(titulo: "Logo y portada") {
                    HStack(spacing: 14) {
                        imagen(n?.logo, alto: 72, ancho: 72, radio: 18)
                        VStack(alignment: .leading, spacing: 6) {
                            PhotosPicker(selection: $logo, matching: .images) { Text(n?.logo == nil ? "Subir logo" : "Cambiar logo").font(Tipo.lato(15, .bold)) }
                            if n?.logo != nil { Button("Sacar") { Task { await patchNegocio(["logo": ""]) } }.font(Tipo.lato(14)).foregroundStyle(C.error) }
                        }
                        Spacer()
                    }
                    .padding(16)
                    VStack(alignment: .leading, spacing: 8) {
                        imagen(w?.portada, alto: 140, ancho: nil, radio: 14)
                        HStack {
                            PhotosPicker(selection: $portada, matching: .images) { Text(w?.portada == nil ? "Subir portada" : "Cambiar portada").font(Tipo.lato(15, .bold)) }
                            Spacer()
                            if w?.portada != nil { Button("Sacar") { Task { await patchWeb(["portada": .texto("")]) } }.font(Tipo.lato(14)).foregroundStyle(C.error) }
                        }
                    }
                    .padding(16)
                }
                .overlay { if subiendo { ProgressView().controlSize(.large) } }
                Bloque(titulo: "Tu negocio") {
                    RenglonCampo(titulo: "Nombre", texto: nb.nombre)
                    RenglonCampo(titulo: "Descripción corta", texto: nb.descripcion, multilinea: true)
                    RenglonCampo(titulo: "Rubro", texto: wb.rubro, placeholder: "Peluquería, estética…")
                    RenglonCampo(titulo: "Sobre ustedes", texto: wb.acerca, multilinea: true)
                }
                Bloque(titulo: "Dónde están") {
                    RenglonCampo(titulo: "Dirección", texto: nb.direccion)
                    RenglonCampo(titulo: "Ciudad", texto: nb.ciudad)
                    RenglonCampo(titulo: "Provincia", texto: wb.provincia)
                }
                Bloque(titulo: "Contacto y redes") {
                    RenglonCampo(titulo: "Teléfono", texto: nb.telefono, teclado: .phonePad)
                    RenglonCampo(titulo: "WhatsApp", texto: nb.whatsapp, teclado: .phonePad)
                    RenglonCampo(titulo: "Email", texto: wb.email, teclado: .emailAddress)
                    RenglonCampo(titulo: "Instagram", texto: nb.instagram, placeholder: "@tunegocio", teclado: .URL)
                    RenglonCampo(titulo: "Facebook", texto: wb.facebook, teclado: .URL)
                    RenglonCampo(titulo: "TikTok", texto: wb.tiktok, teclado: .URL)
                    RenglonCampo(titulo: "Otra web", texto: wb.sitioWeb, teclado: .URL)
                }
                Bloque(titulo: "Colores") {
                    Renglon(titulo: "Principal") { ColorPicker("", selection: color(wb.colorPrimario), supportsOpacity: false).labelsHidden() }
                    Renglon(titulo: "Acento") { ColorPicker("", selection: color(wb.colorAcento), supportsOpacity: false).labelsHidden() }
                }
                BotonGuardar(texto: "Guardar los cambios") { await guardar() }
                Bloque(titulo: "Fotos (\(fotos.count))", pie: "Mantené apretada una foto para moverla o sacarla.") {
                    LazyVGrid(columns: [GridItem(.flexible(), spacing: 6), GridItem(.flexible(), spacing: 6), GridItem(.flexible(), spacing: 6)], spacing: 6) {
                        ForEach(fotos.sorted { $0.orden < $1.orden }) { f in
                            imagen(f.url, alto: 104, ancho: nil, radio: 10)
                                .contextMenu {
                                    Button("Primera", systemImage: "arrow.up.to.line") { Task { await ordenar(f, primero: true) } }
                                    Button("Última", systemImage: "arrow.down.to.line") { Task { await ordenar(f, primero: false) } }
                                    Button("Sacar", systemImage: "trash", role: .destructive) {
                                        Task { await estado.intentar { try await estado.api.hacer("DELETE", "/v1/panel/mi-web/galeria/\(f.id)") }; await cargar() }
                                    }
                                }
                        }
                        PhotosPicker(selection: $foto, matching: .images) {
                            Image(systemName: "plus").font(.system(size: 26, weight: .semibold)).foregroundStyle(C.mentaTexto)
                                .frame(maxWidth: .infinity, minHeight: 104).hueco(10, quieto: true)
                        }
                        .accessibilityLabel("Sumar una foto")
                    }
                    .padding(12)
                }
                NavigationLink(value: DestinoMas.plantillas) { FilaLista(simbolo: "paintpalette", titulo: "Plantilla y estilo") }.buttonStyle(.plain).tarjeta()
                NavigationLink(value: DestinoMas.dominio) { FilaLista(simbolo: "link", titulo: "Dominio propio") }.buttonStyle(.plain).tarjeta()
            }
        }
        .task { await cargar() }
        .onChange(of: logo) { _, i in Task { if let u = await subir(i) { await patchNegocio(["logo": u]) } ; logo = nil } }
        .onChange(of: portada) { _, i in Task { if let u = await subir(i) { await patchWeb(["portada": .texto(u)]) } ; portada = nil } }
        .onChange(of: foto) { _, i in
            Task {
                if let u = await subir(i) {
                    await estado.intentar("Sumamos la foto") { try await estado.api.hacer("POST", "/v1/panel/mi-web/galeria", ["url": u, "epigrafe": ""]) }
                    await cargar()
                }
                foto = nil
            }
        }
    }

    private func color(_ b: Binding<String>) -> Binding<Color> {
        Binding(get: { Color(hexTexto: b.wrappedValue) ?? C.petroleo }, set: { c in
            let u = UIColor(c); var r: CGFloat = 0, g: CGFloat = 0, bl: CGFloat = 0, a: CGFloat = 0
            u.getRed(&r, green: &g, blue: &bl, alpha: &a)
            b.wrappedValue = String(format: "#%02x%02x%02x", Int(max(0, min(1, r)) * 255), Int(max(0, min(1, g)) * 255), Int(max(0, min(1, bl)) * 255))
        })
    }

    @ViewBuilder private func imagen(_ url: String?, alto: CGFloat, ancho: CGFloat?, radio: CGFloat) -> some View {
        Group {
            if let url, !url.isEmpty, let u = URL(string: url) {
                AsyncImage(url: u) { i in i.resizable().scaledToFill() } placeholder: { C.chip }
            } else {
                ZStack { C.chip; Image(systemName: "photo").foregroundStyle(C.sub) }
            }
        }
        .frame(width: ancho, height: alto).frame(maxWidth: ancho == nil ? .infinity : nil)
        .clipShape(RoundedRectangle(cornerRadius: radio, style: .continuous))
    }

    /// Achica la foto (lado largo 2000 px) y la sube como JPEG.
    private func subir(_ item: PhotosPickerItem?) async -> String? {
        guard let item, let d = try? await item.loadTransferable(type: Data.self), let img = UIImage(data: d) else { return nil }
        subiendo = true
        defer { subiendo = false }
        let lado: CGFloat = 2000
        let f = min(1, lado / max(img.size.width, img.size.height))
        let tam = CGSize(width: img.size.width * f, height: img.size.height * f)
        let chica = UIGraphicsImageRenderer(size: tam).image { _ in img.draw(in: CGRect(origin: .zero, size: tam)) }
        guard let jpg = chica.jpegData(compressionQuality: 0.85) else { return nil }
        do { return try await estado.api.subirImagen(jpg) } catch { estado.avisar(error); return nil }
    }

    private func patchNegocio(_ c: [String: String]) async {
        await estado.intentar { try await estado.api.hacer("PATCH", "/v1/panel/negocio", c) }
        await cargar()
    }

    private func patchWeb(_ c: [String: ValorJSON]) async {
        await estado.intentar { try await estado.api.hacer("PATCH", "/v1/panel/mi-web", c) }
        await cargar()
    }

    private struct CuerpoNegocio: Encodable { var nombre: String; var descripcion: String; var direccion: String; var ciudad: String; var telefono: String; var whatsapp: String; var instagram: String }
    private struct CuerpoWeb: Encodable { var rubro: String; var provincia: String; var acerca: String; var colorPrimario: String; var colorAcento: String; var email: String; var sitioWeb: String; var facebook: String; var tiktok: String }

    private func guardar() async {
        guard let n, let w else { return }
        let ok = await estado.intentar("Guardamos tu web") {
            try await estado.api.hacer("PATCH", "/v1/panel/negocio", CuerpoNegocio(nombre: n.nombre, descripcion: n.descripcion, direccion: n.direccion, ciudad: n.ciudad, telefono: n.telefono, whatsapp: n.whatsapp, instagram: n.instagram))
            try await estado.api.hacer("PATCH", "/v1/panel/mi-web", CuerpoWeb(rubro: w.rubro, provincia: w.provincia, acerca: w.acerca, colorPrimario: w.colorPrimario, colorAcento: w.colorAcento, email: w.email, sitioWeb: w.sitioWeb, facebook: w.facebook, tiktok: w.tiktok))
        }
        if ok { try? await estado.cargarSesion() }
    }

    private func ordenar(_ f: FotoGaleria, primero: Bool) async {
        var ids = fotos.sorted { $0.orden < $1.orden }.map(\.id).filter { $0 != f.id }
        if primero { ids.insert(f.id, at: 0) } else { ids.append(f.id) }
        await estado.intentar { try await estado.api.hacer("PUT", "/v1/panel/mi-web/galeria/orden", ["ids": ids]) }
        await cargar()
    }

    private struct Fotos: Decodable { var fotos: [FotoGaleria] }

    private func cargar() async {
        do {
            n = try await estado.api.get("/v1/panel/negocio")
            w = try await estado.api.get("/v1/panel/mi-web")
            fotos = (try? await estado.api.get("/v1/panel/mi-web/galeria") as Fotos)?.fotos ?? []
            error = ""
        } catch let e as ErrorApi { error = e.detalle } catch { self.error = "Probá de nuevo." }
    }
}

// MARK: - Plantillas

struct Plantillas: View {
    @Environment(Estado.self) private var estado
    @State private var estilo: [String: ValorJSON] = [:]
    @State private var cargando = true

    static let plantillas: [(String, String, String)] = [
        ("carta", "Carta", "La portada con tu color y los servicios como una carta. Ideal si tenés muchos servicios."),
        ("vidriera", "Vidriera", "Una foto grande de punta a punta y servicios con foto. Para lugares que se lucen."),
        ("sobria", "Sobria", "Sin fotos: tu color, tu logo y tipografía grande. Para consultorios y quien no tiene fotos."),
        ("galeria", "Galería", "Mosaico de fotos arriba y filtros por categoría. Para uñas, peinados y tatuajes."),
        ("editorial", "Editorial", "El nombre en grande como tapa de revista y una foto chica. Para estudios con personalidad."),
        ("bloques", "Bloques", "Todo en una grilla de bloques: nombre, fotos, reseñas y precios a la vista de un golpe.")
    ]
    static let ajustes: [(String, String, [(String, String)])] = [
        ("portada", "Portada", [("foto", "Con foto"), ("color", "Sólo color")]),
        ("fondo", "Fondo", [("crema", "Crema"), ("blanco", "Blanco"), ("noche", "Noche")]),
        ("titulos", "Títulos", [("grandes", "Grandes"), ("medianos", "Medianos"), ("livianos", "Livianos")]),
        ("forma", "Bordes", [("redondos", "Redondos"), ("suaves", "Suaves"), ("rectos", "Rectos")])
    ]
    static let secciones: [(String, String)] = [("servicios", "Servicios"), ("resenas", "Reseñas"), ("equipo", "Equipo"), ("fotos", "Fotos"), ("membresias", "Membresías")]

    private func valor(_ k: String) -> String { if case .texto(let s) = estilo[k] { return s }; return "" }
    private var secciones: [String]? {
        if case .lista(let l) = estilo["secciones"] { return l.compactMap { if case .texto(let s) = $0 { return s }; return nil } }
        return nil
    }

    var body: some View {
        PantallaMas(titulo: "Plantillas", subtitulo: Self.plantillas.first { $0.0 == valor("plantilla") }?.1 ?? "", volver: "Más", cargando: cargando, recargar: cargar) {
            ForEach(Self.plantillas, id: \.0) { p in
                let on = valor("plantilla") == p.0
                Button { estilo["plantilla"] = .texto(p.0) } label: {
                    HStack(alignment: .top, spacing: 12) {
                        Image(systemName: on ? "checkmark.circle.fill" : "circle").font(.system(size: 22)).foregroundStyle(on ? C.mentaTexto : C.linea)
                        VStack(alignment: .leading, spacing: 4) {
                            Text(p.1).font(Tipo.lato(17, .bold))
                            Text(p.2).font(Tipo.lato(14)).foregroundStyle(C.sub).multilineTextAlignment(.leading)
                        }
                    }
                    .foregroundStyle(C.tinta).padding(16).frame(maxWidth: .infinity, alignment: .leading).tarjeta()
                    .overlay { if on { RoundedRectangle(cornerRadius: Radio.tarjeta, style: .continuous).strokeBorder(C.menta, lineWidth: 2) } }
                }
                .buttonStyle(.plain)
            }
            Bloque(titulo: "Ajustes") {
                ForEach(Self.ajustes, id: \.0) { a in
                    Renglon(titulo: a.1) {
                        Picker(a.1, selection: Binding(get: { valor(a.0) }, set: { estilo[a.0] = .texto($0) })) {
                            ForEach(a.2, id: \.0) { Text($0.1).tag($0.0) }
                        }
                        .tint(C.mentaTexto)
                    }
                }
            }
            if let s = secciones {
                Bloque(titulo: "Secciones", pie: "Las prendidas se muestran en tu página, en este orden.") {
                    ForEach(Self.secciones, id: \.0) { x in
                        RenglonInterruptor(titulo: x.1, activo: Binding(get: { s.contains(x.0) }, set: { on in
                            var l = s
                            if on { l.append(x.0) } else { l.removeAll { $0 == x.0 } }
                            estilo["secciones"] = .lista(l.map { .texto($0) })
                        }))
                    }
                }
            }
            BotonGuardar(texto: "Publicar") {
                await estado.intentar("Publicada. Tus clientes ya la ven.") { try await estado.api.hacer("PATCH", "/v1/panel/mi-web", ["estilo": ValorJSON.objeto(estilo)]) }
            }
            Link(destination: Web.pagina(estado.negocio?.slug ?? "")) { Label("Ver cómo quedó", systemImage: "safari") }.buttonStyle(BotonContorno(alto: 48, tam: 15))
        }
        .task { await cargar() }
    }

    private func cargar() async {
        if let w: WebDatos = try? await estado.api.get("/v1/panel/mi-web"), case .objeto(let o)? = w.estilo { estilo = o }
        cargando = false
    }
}

// MARK: - Dominio

struct DominioVista: View {
    struct Registro: Codable, Hashable { var tipo: String; var nombre: String; var valor: String; var alternativa: Bool? }
    struct Dominio: Codable { var nombre: String; var estado: String; var detalle: String; var registros: [Registro] }
    struct Datos: Codable { var dominio: Dominio?; var destino: String; var ip: String }
    @Environment(Estado.self) private var estado
    @State private var d: Datos?
    @State private var nuevo = ""
    @State private var error = ""

    var body: some View {
        PantallaMas(titulo: "Dominio propio", subtitulo: d?.dominio?.nombre ?? "Usá tu propia dirección", cargando: d == nil && error.isEmpty, error: error, recargar: cargar) {
            if let dom = d?.dominio {
                Bloque {
                    Renglon(titulo: dom.nombre, detalle: dom.estado == "verificado" ? "Funcionando" : (dom.detalle.isEmpty ? "Esperando que configures el DNS" : dom.detalle),
                            simbolo: dom.estado == "verificado" ? "checkmark.seal.fill" : "hourglass")
                }
                if dom.estado != "verificado" {
                    Bloque(titulo: "Cargá esto en tu proveedor de dominio") {
                        ForEach(dom.registros, id: \.self) { r in
                            VStack(alignment: .leading, spacing: 4) {
                                Text("\(r.tipo) · \(r.nombre)\(r.alternativa == true ? " (o este)" : "")").font(Tipo.lato(13, .bold)).foregroundStyle(C.sub)
                                HStack {
                                    Text(r.valor).font(.system(size: 15, design: .monospaced)).textSelection(.enabled)
                                    Spacer()
                                    Button { UIPasteboard.general.string = r.valor; Haptico.exito() } label: { Image(systemName: "doc.on.doc") }.accessibilityLabel("Copiar")
                                }
                            }
                            .padding(16)
                        }
                    }
                    BotonGuardar(texto: "Ya lo cargué, verificar") {
                        await estado.intentar { _ = try await estado.api.pedir("POST", "/v1/panel/dominio/verificar", cuerpo: Data("{}".utf8)) }
                        await cargar()
                    }
                }
                Button(role: .destructive) { Task { await estado.intentar("Sacamos el dominio") { try await estado.api.hacer("DELETE", "/v1/panel/dominio") }; await cargar() } } label: { Text("Sacar el dominio") }
                    .buttonStyle(BotonContorno(alto: 48, tam: 15)).foregroundStyle(C.error)
            } else {
                Bloque(pie: "Comprá el dominio donde quieras (NIC Argentina, GoDaddy…) y después lo apuntás a Tengo Libre.") {
                    RenglonCampo(titulo: "Tu dominio", texto: $nuevo, placeholder: "turnos.tunegocio.com.ar", teclado: .URL)
                }
                BotonGuardar(texto: "Usar este dominio", habilitado: nuevo.contains(".")) {
                    await estado.intentar { try await estado.api.hacer("PUT", "/v1/panel/dominio", ["dominio": nuevo.lowercased().trimmingCharacters(in: .whitespaces)]) }
                    await cargar()
                }
            }
        }
        .task { await cargar() }
    }

    private func cargar() async {
        do { d = try await estado.api.get("/v1/panel/dominio"); error = "" } catch let e as ErrorApi { error = e.detalle } catch { self.error = "Probá de nuevo." }
    }
}

// MARK: - Integraciones e IA

struct Integraciones: View {
    struct Calendario: Codable, Hashable { var id: String?; var profesionalId: String?; var nombre: String; var url: String? }
    struct Token: Codable, Identifiable, Hashable { var id: String; var nombre: String; var pista: String; var permisos: [String]; var cliente: String; var creadoEn: Date; var usadoEn: Date? }
    struct Permiso: Codable, Hashable { var clave: String; var nombre: String; var descripcion: String }
    struct Mcp: Codable { var url: String; var tokens: [Token]; var permisos: [Permiso] }
    struct IaPublica: Codable { var activo: Bool; var reservasMes: Int; var url: String }
    struct Creado: Codable { var token: Token; var clave: String; var url: String }

    @Environment(Estado.self) private var estado
    @State private var calendarios: [Calendario] = []
    @State private var mcp: Mcp?
    @State private var ia: IaPublica?
    @State private var cargando = true
    @State private var nombreClave = "Claude"
    @State private var creada: Creado?

    var body: some View {
        PantallaMas(titulo: estado.esProfesional ? "Tu calendario" : "Integraciones e IA", cargando: cargando, recargar: cargar) {
            Bloque(titulo: "Calendario", pie: "Suscribite desde Google Calendar, Apple o Outlook y los turnos aparecen solos (se actualiza cada tanto).") {
                ForEach(calendarios, id: \.self) { c in
                    if let u = c.url {
                        Renglon(titulo: c.nombre, simbolo: "calendar") {
                            HStack(spacing: 4) {
                                Button { UIPasteboard.general.string = u; estado.avisar("Copiamos el link") } label: { Image(systemName: "doc.on.doc").frame(width: 40, height: 40) }
                                    .accessibilityLabel("Copiar link")
                                if let w = URL(string: u.replacingOccurrences(of: "https://", with: "webcal://")) {
                                    Link(destination: w) { Image(systemName: "calendar.badge.plus").frame(width: 40, height: 40) }.accessibilityLabel("Agregar al calendario del iPhone")
                                }
                            }
                            .foregroundStyle(C.mentaTexto)
                        }
                    } else {
                        Button { Task { await crearCalendario(c.profesionalId) } } label: { Renglon(titulo: c.nombre, detalle: "Crear el link", simbolo: "plus.circle") }.buttonStyle(.plain)
                    }
                }
            }
            if estado.gestiona {
                if let ia {
                    Bloque(titulo: "Que te reserven desde ChatGPT y Claude", pie: "Tus servicios y horarios libres quedan disponibles para los asistentes de IA. Este mes: \(Formato.cantidad(ia.reservasMes, "reserva", "reservas")).") {
                        RenglonInterruptor(titulo: "Reservas por IA", activo: Binding(get: { ia.activo }, set: { v in Task { await iaPublica(v) } }))
                    }
                }
                if let mcp {
                    Bloque(titulo: "Manejá tu agenda con Claude o ChatGPT", pie: "Creá una clave y pegala en el conector del asistente. Ves y movés turnos hablando.") {
                        Renglon(titulo: "Dirección del conector", detalle: mcp.url) {
                            Button { UIPasteboard.general.string = mcp.url; Haptico.exito() } label: { Image(systemName: "doc.on.doc") }.foregroundStyle(C.mentaTexto)
                        }
                        ForEach(mcp.tokens) { t in
                            Renglon(titulo: t.nombre, detalle: "…\(t.pista) · \(t.cliente.isEmpty ? "sin usar" : t.cliente)\(t.usadoEn.map { " · \(Formato.hace($0))" } ?? "")") {
                                Button { Task { await estado.intentar("Borramos la clave") { try await estado.api.hacer("DELETE", "/v1/panel/mcp/tokens/\(t.id)") }; await cargar() } } label: {
                                    Image(systemName: "trash").foregroundStyle(C.error).frame(width: 44, height: 44)
                                }
                                .accessibilityLabel("Borrar la clave")
                            }
                        }
                        HStack {
                            Picker("Para", selection: $nombreClave) { ForEach(["Claude", "ChatGPT", "Gemini", "Otro (MCP)"], id: \.self) { Text($0) } }.tint(C.mentaTexto)
                            Spacer()
                            Button("Crear clave") { Task { await crearClave() } }.buttonStyle(BotonMentaChico())
                        }
                        .padding(.horizontal, 16).padding(.vertical, 8)
                    }
                }
            }
        }
        .task { await cargar() }
        .sheet(item: Binding(get: { creada.map { IdTexto(id: $0.clave) } }, set: { if $0 == nil { creada = nil } })) { c in
            NavigationStack {
                VStack(alignment: .leading, spacing: 16) {
                    Text("Copiala ahora: no la vamos a mostrar de nuevo.").font(Tipo.lato(15)).foregroundStyle(C.sub)
                    Text(c.id).font(.system(size: 15, design: .monospaced)).textSelection(.enabled).padding(14).frame(maxWidth: .infinity, alignment: .leading)
                        .background(C.chip, in: RoundedRectangle(cornerRadius: 12))
                    Button { UIPasteboard.general.string = c.id; Haptico.exito() } label: { Label("Copiar la clave", systemImage: "doc.on.doc") }.buttonStyle(BotonMenta(alto: 52, conSombra: false))
                    Spacer()
                }
                .padding(16).background(C.hoja)
                .navigationTitle("Tu clave").navigationBarTitleDisplayMode(.inline)
                .toolbar { ToolbarItem(placement: .confirmationAction) { Button("Listo") { creada = nil } } }
            }
            .presentationDetents([.medium])
        }
    }

    private struct Cals: Decodable { var calendarios: [Calendario] }

    private func cargar() async {
        calendarios = (try? await estado.api.get("/v1/panel/calendarios") as Cals)?.calendarios ?? calendarios
        if estado.gestiona {
            mcp = try? await estado.api.get("/v1/panel/mcp")
            ia = try? await estado.api.get("/v1/panel/ia/publico")
        }
        cargando = false
    }

    private func crearCalendario(_ p: String?) async {
        await estado.intentar { try await estado.api.hacer("POST", "/v1/panel/calendarios", ["profesionalId": p]) }
        await cargar()
    }

    private func iaPublica(_ v: Bool) async {
        await estado.intentar { try await estado.api.hacer("PUT", "/v1/panel/ia/publico", ["activo": v]) }
        await cargar()
    }

    private func crearClave() async {
        do {
            creada = try await estado.api.enviar("POST", "/v1/panel/mcp/tokens", ["nombre": nombreClave])
            await cargar()
        } catch { estado.avisar(error) }
    }
}

struct IdTexto: Identifiable { var id: String }

// MARK: - Configuración

struct Configuracion: View {
    struct Ajustes: Codable { var intervalo: Int; var anticipacionMin: Int; var diasVista: Int; var margenMin: Int; var cancelarHastaHoras: Int; var politicaCancelacion: String }
    struct Avisos: Codable { var resenaDemoraHoras: Int; var resumenDiario: Bool }
    struct Espera: Codable { var minutos: Int; var automatica: Bool }
    struct Encargado: Codable, Hashable { var id: String; var nombre: String; var email: String; var recibe: Bool }
    struct Mensual: Codable { var activo: Bool; var encargados: [Encargado] }

    @Environment(Estado.self) private var estado
    @State private var a: Ajustes?
    @State private var av = Avisos(resenaDemoraHoras: 2, resumenDiario: true)
    @State private var es = Espera(minutos: 30, automatica: true)
    @State private var me: Mensual?
    @State private var error = ""

    var body: some View {
        PantallaMas(titulo: "Configuración", subtitulo: "Reservas, cancelaciones y avisos", cargando: a == nil && error.isEmpty, error: error, recargar: cargar) {
            if a != nil {
                let b = Binding(get: { a! }, set: { a = $0 })
                Bloque(titulo: "Reservas") {
                    Numerito(titulo: "Horarios cada", valor: b.intervalo, rango: 5...120, paso: 5) { "\($0) min" }
                    Numerito(titulo: "Reservar con anticipación", detalle: "Lo mínimo antes del turno", valor: Binding(get: { b.wrappedValue.anticipacionMin / 60 }, set: { b.wrappedValue.anticipacionMin = $0 * 60 }), rango: 0...72) { $0 == 0 ? "Sin mínimo" : "\($0) h" }
                    Numerito(titulo: "Se puede reservar hasta", valor: b.diasVista, rango: 7...365, paso: 7) { "\($0) días" }
                    Numerito(titulo: "Margen entre turnos", valor: b.margenMin, rango: 0...60, paso: 5) { $0 == 0 ? "Sin margen" : "\($0) min" }
                }
                Bloque(titulo: "Cancelaciones") {
                    Numerito(titulo: "Cancelar hasta", detalle: "Antes del turno, desde el link", valor: b.cancelarHastaHoras, rango: 0...72) { $0 == 0 ? "Siempre" : "\($0) h antes" }
                    RenglonCampo(titulo: "Política (se ve al reservar)", texto: b.politicaCancelacion, placeholder: "Si no podés venir, avisanos 24 h antes.", multilinea: true)
                }
                BotonGuardar { await estado.intentar("Guardamos las reglas de reserva") { try await estado.api.hacer("PUT", "/v1/panel/ajustes", a) } }
                Bloque(titulo: "Avisos") {
                    Numerito(titulo: "Pedir la reseña", detalle: "Después del turno atendido", valor: $av.resenaDemoraHoras, rango: 0...72) { $0 == 0 ? "No pedir" : "\($0) h después" }
                    RenglonInterruptor(titulo: "Resumen del día por mail", detalle: "A la mañana, con los turnos del día", activo: $av.resumenDiario)
                    BotonGuardar { await estado.intentar("Guardamos los avisos") { try await estado.api.hacer("PUT", "/v1/panel/ajustes/avisos", av) } }.padding(12)
                }
                Bloque(titulo: "Lista de espera", pie: "Cuando se libera un horario, se lo ofrecemos a quien espera de a una persona por vez.") {
                    RenglonInterruptor(titulo: "Ofrecer solos los huecos", activo: $es.automatica)
                    Numerito(titulo: "Tiempo para tomarlo", valor: $es.minutos, rango: 5...240, paso: 5) { "\($0) min" }
                    BotonGuardar { await estado.intentar("Guardamos la lista de espera") { try await estado.api.hacer("PUT", "/v1/panel/ajustes/espera", es) } }.padding(12)
                }
                if me != nil {
                    let mb = Binding(get: { me! }, set: { me = $0 })
                    Bloque(titulo: "Reporte mensual por mail") {
                        RenglonInterruptor(titulo: "Mandarlo el 1° de cada mes", activo: mb.activo)
                        ForEach(mb.encargados) { $e in RenglonInterruptor(titulo: e.nombre, detalle: e.email, activo: $e.recibe) }
                        BotonGuardar { await estado.intentar("Guardamos el reporte mensual") { try await estado.api.hacer("PUT", "/v1/panel/reportes/mensual/ajustes", me) } }.padding(12)
                    }
                }
                NavigationLink(value: DestinoMas.whatsapp) { FilaLista(simbolo: "message", titulo: "WhatsApp") }.buttonStyle(.plain).tarjeta()
                NavigationLink(value: DestinoMas.penalidades) { FilaLista(simbolo: "exclamationmark.shield", titulo: "Ausencias y penalidades") }.buttonStyle(.plain).tarjeta()
                NavigationLink(value: DestinoMas.auditoria) { FilaLista(simbolo: "list.bullet.rectangle", titulo: "Actividad") }.buttonStyle(.plain).tarjeta()
            }
        }
        .task { await cargar() }
    }

    private func cargar() async {
        do {
            a = try await estado.api.get("/v1/panel/ajustes")
            if let x: Avisos = try? await estado.api.get("/v1/panel/ajustes/avisos") { av = x }
            if let x: Espera = try? await estado.api.get("/v1/panel/ajustes/espera") { es = x }
            me = try? await estado.api.get("/v1/panel/reportes/mensual/ajustes")
            error = ""
        } catch let e as ErrorApi { error = e.detalle } catch { self.error = "Probá de nuevo." }
    }
}

extension Configuracion.Encargado: Identifiable {}

// MARK: - Recordatorios a clientes

struct AvisosClientes: View {
    struct Cup: Codable, Hashable { var id: String; var codigo: String; var descuento: String; var vigente: Bool }
    struct Semana: Codable, Hashable { var clienteId: String; var nombre: String; var fecha: String; var detalle: String; var enviado: Bool }
    struct Recordatorio: Codable, Identifiable { var tipo: String; var titulo: String; var activo: Bool; var texto: String; var cuponId: String?; var dias: Int; var regla: String; var semana: [Semana]; var semanaTexto: String; var id: String { tipo } }
    @Environment(Estado.self) private var estado
    @State private var lista: [Recordatorio] = []
    @State private var cupones: [Cupon] = []
    @State private var cargando = true

    var body: some View {
        PantallaMas(titulo: "Recordatorios", subtitulo: "Mails automáticos a tus clientas", cargando: cargando, recargar: cargar) {
            ForEach($lista) { $r in
                Bloque(titulo: r.titulo, pie: "\(r.regla) \(r.semanaTexto)") {
                    RenglonInterruptor(titulo: r.activo ? "Prendido" : "Apagado", activo: $r.activo)
                    if r.tipo == "extraniamos" { Numerito(titulo: "Si no viene hace", valor: $r.dias, rango: 15...365, paso: 15) { "\($0) días" } }
                    RenglonCampo(titulo: "Mensaje", texto: $r.texto, multilinea: true)
                    Renglon(titulo: "Con cupón") {
                        Picker("Cupón", selection: Binding(get: { r.cuponId ?? "" }, set: { r.cuponId = $0.isEmpty ? nil : $0 })) {
                            Text("Sin cupón").tag("")
                            ForEach(cupones.filter { $0.activo && $0.referidoId == nil }) { Text($0.codigo).tag($0.id) }
                        }
                        .tint(C.mentaTexto)
                    }
                    BotonGuardar { await guardar(r) }.padding(12)
                }
            }
        }
        .task { await cargar() }
    }

    private struct Rs: Decodable { var recordatorios: [Recordatorio] }
    private struct Cs: Decodable { var cupones: [Cupon] }
    private struct Cuerpo: Encodable { var activo: Bool; var texto: String; var cuponId: String; var dias: Int? }

    private func cargar() async {
        lista = (try? await estado.api.get("/v1/panel/recordatorios") as Rs)?.recordatorios ?? lista
        cupones = (try? await estado.api.get("/v1/panel/cupones") as Cs)?.cupones ?? cupones
        cargando = false
    }

    private func guardar(_ r: Recordatorio) async {
        await estado.intentar("Guardamos el recordatorio") {
            try await estado.api.hacer("PUT", "/v1/panel/recordatorios/\(r.tipo)", Cuerpo(activo: r.activo, texto: r.texto, cuponId: r.cuponId ?? "", dias: r.tipo == "extraniamos" ? r.dias : nil))
        }
    }
}

// MARK: - WhatsApp

struct WhatsAppVista: View {
    struct Ajustes: Codable { var disponible: Bool; var confirmacion: Bool; var hora: String; var sinRespuestaHoras: Int; var recordatorio: Bool }
    struct Mensaje: Codable, Identifiable { var id: String; var tipo: String; var cliente: String; var estado: String; var respuesta: String?; var creadoEn: Date; var ultimoError: String? }
    @Environment(Estado.self) private var estado
    @State private var a: Ajustes?
    @State private var mensajes: [Mensaje] = []
    @State private var error = ""

    var body: some View {
        PantallaMas(titulo: "WhatsApp", subtitulo: a?.disponible == false ? "Todavía no disponible" : "Confirmaciones automáticas", volver: "Más", cargando: a == nil && error.isEmpty, error: error, recargar: cargar) {
            if a != nil {
                let b = Binding(get: { a! }, set: { a = $0 })
                Bloque(pie: "El día antes le pedimos a cada clienta que confirme. Si no contesta, aparece en «Para atender» para que la llames.") {
                    RenglonInterruptor(titulo: "Pedir confirmación", activo: b.confirmacion)
                    Numerito(titulo: "A las", valor: Binding(get: { Int(b.wrappedValue.hora.prefix(2)) ?? 10 }, set: { b.wrappedValue.hora = String(format: "%02d:00", $0) }), rango: 7...21) { String(format: "%02d:00", $0) }
                    Numerito(titulo: "Si no contesta, avisarte a las", valor: b.sinRespuestaHoras, rango: 1...24) { "\($0) h" }
                    RenglonInterruptor(titulo: "Recordatorio el mismo día", activo: b.recordatorio)
                    BotonGuardar { await estado.intentar("Guardamos WhatsApp") { try await estado.api.hacer("PUT", "/v1/panel/ajustes/whatsapp", ["confirmacion": ValorJSON.booleano(b.wrappedValue.confirmacion), "hora": .texto(b.wrappedValue.hora), "sinRespuestaHoras": .numero(Double(b.wrappedValue.sinRespuestaHoras)), "recordatorio": .booleano(b.wrappedValue.recordatorio)]) } }.padding(12)
                }
                .disabled(a?.disponible == false)
                if !mensajes.isEmpty {
                    Bloque(titulo: "Últimos mensajes") {
                        ForEach(mensajes.prefix(30)) { m in
                            Renglon(titulo: m.cliente, detalle: "\(["confirmacion": "Confirmación", "recordatorio": "Recordatorio", "respuesta": "Respuesta"][m.tipo] ?? m.tipo) · \(Formato.hace(m.creadoEn))") {
                                Text(m.respuesta == "confirmo" ? "Confirmó" : m.respuesta == "cambiar" ? "Quiere cambiar" : ["enviado": "Enviado", "pendiente": "En cola", "fallido": "No salió", "descartado": "Descartado"][m.estado] ?? m.estado)
                                    .font(Tipo.lato(13, .bold)).foregroundStyle(m.estado == "fallido" ? C.error : C.mentaTexto)
                            }
                        }
                    }
                }
            }
        }
        .task { await cargar() }
    }

    private struct Ms: Decodable { var mensajes: [Mensaje] }
    private func cargar() async {
        do {
            a = try await estado.api.get("/v1/panel/ajustes/whatsapp")
            mensajes = (try? await estado.api.get("/v1/panel/whatsapp/mensajes") as Ms)?.mensajes ?? []
            error = ""
        } catch let e as ErrorApi { error = e.detalle } catch { self.error = "Probá de nuevo." }
    }
}

// MARK: - Actividad

struct Auditoria: View {
    struct Entrada: Codable, Identifiable { var id: String; var nombre: String; var accion: String; var entidad: String; var creadoEn: Date }
    @Environment(Estado.self) private var estado
    @State private var entradas: [Entrada] = []
    @State private var cargando = true

    var body: some View {
        PantallaMas(titulo: "Actividad", subtitulo: "Quién hizo qué en el panel", cargando: cargando, recargar: cargar) {
            let grupos = Dictionary(grouping: entradas) { Formato.seccion($0.creadoEn) }
            let orden = entradas.map { Formato.seccion($0.creadoEn) }.reduce(into: [String]()) { if !$0.contains($1) { $0.append($1) } }
            ForEach(orden, id: \.self) { s in
                Bloque(titulo: s) {
                    ForEach(grupos[s] ?? []) { e in
                        Renglon(titulo: "\(e.nombre.isEmpty ? "Alguien" : e.nombre) \(texto(e))", detalle: Formato.horaDe(e.creadoEn))
                    }
                }
            }
            if entradas.isEmpty && !cargando { Text("Todavía no hay actividad.").font(Tipo.lato(14)).foregroundStyle(C.sub) }
        }
        .task { await cargar() }
    }

    private func texto(_ e: Entrada) -> String {
        let verbo = ["crear": "creó", "editar": "editó", "borrar": "borró", "cancelar": "canceló", "cobrar": "cobró", "mover": "movió", "estado": "cambió el estado de", "invitar": "invitó a", "aprobar": "aprobó", "rechazar": "rechazó"]
        let partes = e.accion.split(separator: ".").map(String.init)
        let v = partes.last.flatMap { verbo[$0] } ?? e.accion
        let cosa = ["turno": "un turno", "servicio": "un servicio", "profesional": "una profesional", "cliente": "una clienta", "cupon": "un cupón", "ajustes": "los ajustes", "miembro": "a alguien del equipo", "membresia": "una membresía"][e.entidad] ?? e.entidad
        return "\(v) \(cosa)"
    }

    private struct Es: Decodable { var entradas: [Entrada] }
    private func cargar() async {
        entradas = (try? await estado.api.get("/v1/panel/auditoria", query: [URLQueryItem(name: "limite", value: "100")]) as Es)?.entradas ?? entradas
        cargando = false
    }
}

// MARK: - Sucursales

struct Sucursales: View {
    struct Sucursal: Codable, Hashable, Identifiable { var id: String; var slug: String; var nombre: String; var ciudad: String }
    struct Grupo: Codable, Hashable, Identifiable { var id: String; var nombre: String; var sucursales: [Sucursal] }
    struct Grupos: Codable { var grupos: [Grupo]; var sinGrupo: [Sucursal] }
    struct Numeros: Codable, Hashable { var turnos: Int; var atendidos: Int; var ausentes: Int; var cancelados: Int; var clientes: Int; var ingresos: Int }
    struct Reporte: Codable { var total: Numeros }
    @Environment(Estado.self) private var estado
    @State private var g: Grupos?
    @State private var totales: [String: Numeros] = [:]
    @State private var nombre = ""
    @State private var error = ""

    var body: some View {
        PantallaMas(titulo: "Sucursales", subtitulo: "Varios locales, una sola mirada", cargando: g == nil && error.isEmpty, error: error, recargar: cargar) {
            if let g {
                ForEach(g.grupos) { gr in
                    Bloque(titulo: gr.nombre, pie: totales[gr.id].map { "Este mes: \($0.atendidos) atendidos · \(Formato.pesos($0.ingresos))" } ?? "") {
                        ForEach(gr.sucursales) { s in
                            Renglon(titulo: s.nombre, detalle: s.ciudad) {
                                Button("Sacar") { Task { await estado.intentar { try await estado.api.hacer("DELETE", "/v1/grupos/\(gr.id)/sucursales/\(s.id)") }; await cargar() } }
                                    .buttonStyle(BotonTexto(color: C.error))
                            }
                        }
                        if !g.sinGrupo.isEmpty {
                            Menu {
                                ForEach(g.sinGrupo) { s in
                                    Button(s.nombre) { Task { await estado.intentar { try await estado.api.hacer("PUT", "/v1/grupos/\(gr.id)/sucursales/\(s.id)", [String: String]()) }; await cargar() } }
                                }
                            } label: { Renglon(titulo: "Sumar un local", simbolo: "plus.circle") }
                        }
                    }
                }
                Bloque(titulo: "Nuevo grupo", pie: "Para ver juntos los números de varios locales tuyos.") {
                    HStack {
                        TextField("Nombre del grupo", text: $nombre).font(Tipo.lato(16))
                        Button("Crear") { Task { await estado.intentar { try await estado.api.hacer("POST", "/v1/grupos", ["nombre": nombre]) }; nombre = ""; await cargar() } }
                            .buttonStyle(BotonMentaChico()).disabled(nombre.count < 2)
                    }
                    .padding(16)
                }
                Text("Para abrir otro local, entrá desde la compu a tengolibre.com.ar/empezar con esta misma cuenta.").font(Tipo.lato(13)).foregroundStyle(C.sub)
            }
        }
        .task { await cargar() }
    }

    private func cargar() async {
        do {
            g = try await estado.api.get("/v1/grupos")
            let cal = Formato.calendario
            let d = cal.date(from: cal.dateComponents([.year, .month], from: Date()))!
            for gr in g?.grupos ?? [] {
                if let r: Reporte = try? await estado.api.get("/v1/grupos/\(gr.id)/reporte", query: [URLQueryItem(name: "desde", value: Formato.diaISO(d)), URLQueryItem(name: "hasta", value: Formato.diaISO(Date()))]) { totales[gr.id] = r.total }
            }
            error = ""
        } catch let e as ErrorApi { error = e.detalle } catch { self.error = "Probá de nuevo." }
    }
}

// MARK: - Qué avisos llegan al teléfono

struct PreferenciasPush: View {
    struct Tipo_: Codable, Hashable { var tipo: String; var nombre: String; var activo: Bool }
    struct Prefs: Codable { var rol: String; var porDefecto: Bool; var apagados: [String]; var tipos: [Tipo_] }
    @Environment(Estado.self) private var estado
    @State private var p: Prefs?
    @State private var permitido = true

    var body: some View {
        Form {
            if !permitido {
                Section {
                    Button("Activar los avisos en Ajustes") {
                        if let u = URL(string: UIApplication.openNotificationSettingsURLString) { UIApplication.shared.open(u) }
                    }
                } footer: { Text("Los avisos están apagados para Tengo Libre en este iPhone.") }
            }
            if let p {
                Section {
                    ForEach(p.tipos, id: \.tipo) { t in
                        Toggle(t.nombre, isOn: Binding(get: { !p.apagados.contains(t.tipo) }, set: { v in Task { await cambiar(t.tipo, v) } })).tint(C.menta)
                    }
                } footer: { Text("Vale para este negocio. Lo que apagues lo seguís viendo en la campanita.") }
                if !p.porDefecto {
                    Section { Button("Volver a lo recomendado") { Task { await estado.intentar { try await estado.api.hacer("PUT", "/v1/panel/push/preferencias", ["porDefecto": true]) }; await cargar() } } }
                }
            } else {
                ProgressView()
            }
        }
        .navigationTitle("Qué te avisamos")
        .task {
            let a = await UNUserNotificationCenter.current().notificationSettings()
            permitido = a.authorizationStatus == .authorized || a.authorizationStatus == .provisional
            await cargar()
        }
    }

    private func cargar() async { p = try? await estado.api.get("/v1/panel/push/preferencias") }

    private func cambiar(_ tipo: String, _ on: Bool) async {
        guard var x = p else { return }
        if on { x.apagados.removeAll { $0 == tipo } } else if !x.apagados.contains(tipo) { x.apagados.append(tipo) }
        p = x
        await estado.intentar { try await estado.api.hacer("PUT", "/v1/panel/push/preferencias", ["apagados": x.apagados]) }
    }
}
