import SwiftUI

/// 05 Hoy (y «Tu día» de la profesional): la cabecera de petróleo con el negocio, compartir, avisos y
/// el día; abajo, «Para atender» y la pila con lo próximo primero. Estados 20–25: vacío que enseña,
/// cargando, error con salida, sin conexión y sólo lectura (dueña y equipo).
struct Hoy: View {
    @Environment(Estado.self) private var estado
    @Environment(Navegacion.self) private var nav
    @State private var datos = DatosDia()
    @State private var verTemprano = false
    @State private var ofrecer: Dia.Hueco?
    @State private var ahora = Date()
    @State private var avisados: Set<String> = []

    private var hoy: Date { Formato.calendario.startOfDay(for: ahora) }
    private var delDia: [Turno] { datos.turnos.filter { Formato.diaISO($0.inicio) == Formato.diaISO(hoy) } }
    private var profesionales: [Profesional] { Dia.profesionalesDelDia(datos.profesionales, delDia, rol: estado.rol) }
    private var dias: [Dia.DeProfesional] { Dia.armar(profesionales, datos.turnos, fecha: hoy, ahora: Dia.minutosAhora(ahora)) }

    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                Cabecera(titulo: estado.esProfesional ? "Tu día" : "Hoy",
                         subtitulo: estado.esProfesional ? "\(Formato.fechaLarga(hoy).split(separator: " ").prefix(2).joined(separator: " ")) · sólo tus turnos" : Formato.fechaLarga(hoy),
                         conNegocio: !estado.esProfesional,
                         acciones: [
                            AccionCabecera(simbolo: "square.and.arrow.up", etiqueta: "Compartir tu link") { nav.compartir = true },
                            AccionCabecera(simbolo: "bell", etiqueta: "Avisos", numero: estado.sinLeer) { nav.avisos = true }
                         ]) {
                    if datos.carga == .cargando && datos.turnos.isEmpty {
                        Esqueleto(alto: 118, radio: 20).opacity(0.4)
                    } else if !dias.isEmpty {
                        ElDia(dias: dias, esHoy: true)
                    }
                }
                Cuerpo { contenido }
            }
        }
        .scrollIndicators(.hidden)
        .background(C.fondo)
        .toolbar(.hidden, for: .navigationBar)
        .refreshable { await cargar() }
        .task(id: "\(estado.version)-\(estado.negocio?.id ?? "")") { await cargar() }
        .task {
            // El minuto corre: «Ahora», lo en curso y los huecos se acomodan solos.
            while !Task.isCancelled {
                try? await Task.sleep(for: .seconds(30))
                ahora = Date()
                if Int(ahora.timeIntervalSince1970) % 120 < 30 { await datos.cargar(estado, desde: hoy, hasta: hoy) }
            }
        }
        .sheet(item: $ofrecer) { h in
            OfrecerHueco(hueco: h, dia: hoy, datos: datos) { avisados.insert(h.id) }
                .presentationDetents([.medium, .large])
                .presentationCornerRadius(Radio.hoja)
        }
    }

    private func cargar() async {
        await datos.cargar(estado, desde: hoy, hasta: hoy)
        ActividadTurno.sincronizar(delDia)
    }

    @ViewBuilder private var contenido: some View {
        ForEach(banners, id: \.titulo) { b in Banner(b: b) }
        switch datos.carga {
        case .cargando where datos.turnos.isEmpty:
            ForEach([80, 72, 80, 80], id: \.self) { h in Esqueleto(alto: CGFloat(h)) }
            Text("Trayendo tu día…").font(Tipo.lato(13)).foregroundStyle(C.sub).frame(maxWidth: .infinity)
        case .error(let m):
            ErrorConSalida(titulo: "No pudimos traer tu día", texto: "Tus turnos están a salvo. \(m)") { Task { await cargar() } }
        default:
            if delDia.filter(\.ocupa).isEmpty {
                vacio
                AtajosHoy()
                DescubrirHoy()
            } else {
                paraAtender
                AtajosHoy()
                pila
                DescubrirHoy()
            }
        }
    }

    // MARK: Banners (sólo lectura y sin conexión)

    private var banners: [Banner.Datos] {
        var out: [Banner.Datos] = []
        if estado.soloLectura {
            if estado.esDuena {
                out.append(.init(simbolo: "lock.circle", titulo: "Tu prueba terminó", texto: "Podés ver todo, pero para cargar o cambiar turnos tenés que suscribirte. Tus clientes siguen viendo tu página.", accion: "Elegir un plan", petroleo: true) {
                    nav.pestana = .mas; nav.pilaMas.append(DestinoMas.suscripcion)
                })
            } else {
                out.append(.init(simbolo: "lock.circle", titulo: "Por ahora, sólo para mirar", texto: "La prueba de \(estado.negocio?.nombre ?? "tu negocio") terminó. La dueña tiene que activar la suscripción para que vuelvas a cargar turnos.", accion: "", petroleo: true) {})
            }
        }
        if estado.sinConexion {
            let h = estado.guardadoEn.map { " de las \(Formato.horaDe($0))" } ?? ""
            out.append(.init(simbolo: "icloud.slash", titulo: "Sin conexión · tu día\(h)", texto: "Podés ver los turnos y llamar. Para cargar o cambiar algo necesitás conexión.", accion: "", petroleo: false) {})
        }
        if let a = datos.arranque, !a.completo, estado.esDuena {
            let faltan = a.pasos.filter { !$0.listo }
            if let p = faltan.first {
                out.append(.init(simbolo: "flag.checkered", titulo: "Te faltan \(faltan.count) \(faltan.count == 1 ? "paso" : "pasos") para que te reserven", texto: p.titulo, accion: "Seguir", petroleo: true) {
                    nav.pestana = .mas; nav.pilaMas.append(DestinoMas.primerosPasos)
                })
            }
        }
        return out
    }

    // MARK: Para atender

    private struct Atender: Identifiable {
        var id: String
        var simbolo: String
        var titulo: String
        var texto: String
        var accion: String
        var hacer: () -> Void
    }

    private var atender: [Atender] {
        var out: [Atender] = []
        for t in delDia where t.estado == "pendiente_pago" && t.inicio > ahora.addingTimeInterval(-3600) {
            out.append(Atender(id: "sena-\(t.id)", simbolo: "creditcard", titulo: "Seña pendiente", texto: "\(t.cliente.nombre) · hoy \(Formato.horaDe(t.inicio))", accion: "Ver") { nav.turno = t })
        }
        let esperando = datos.espera.filter { $0.avisadoEn == nil }
        if !esperando.isEmpty && !estado.esProfesional {
            let match = dias.flatMap(\.huecos).first { !datos.esperanPara($0).isEmpty }
            let quien = match.flatMap { datos.esperanPara($0).first?.nombre.split(separator: " ").first }
            out.append(Atender(id: "espera", simbolo: "hourglass", titulo: "\(esperando.count) esperando un lugar",
                               texto: match != nil && quien != nil ? "A \(quien!) le sirve el hueco de las \(Dia.hhmm(match!.desde))" : "En la lista de espera", accion: "Ver") { nav.pestana = .espera })
        }
        for l in datos.paraLlamar {
            out.append(Atender(id: "llamar-\(l.turnoId)", simbolo: "phone", titulo: "Llamar a \(l.cliente)", texto: "\(l.motivoTexto) · \(Formato.horaDe(l.inicio))", accion: "Llamar") {
                if let u = URL(string: "tel:\(l.telefono.filter { "+0123456789".contains($0) })") { UIApplication.shared.open(u) }
                Task { try? await estado.api.hacer("POST", "/v1/panel/turnos/\(l.turnoId)/llamado"); await cargar() }
            })
        }
        for r in datos.reprogramar {
            out.append(Atender(id: "repro-\(r.id)", simbolo: "calendar.badge.clock", titulo: "Mover a \(r.turno.cliente.nombre)", texto: "\(r.turno.profesional) pidió días libres · \(Formato.dia(r.turno.inicio))", accion: "Mover") {
                Task { if let t: Turno = try? await estado.api.get("/v1/panel/turnos/\(r.turno.id)") { nav.turno = t } }
            })
        }
        for p in datos.pedidos {
            out.append(Atender(id: "pedido-\(p.id)", simbolo: "beach.umbrella", titulo: "\(p.profesional) pide días libres", texto: "\(p.motivoTexto) · \(p.turnos) \(p.turnos == 1 ? "turno" : "turnos") para mover", accion: "Ver") {
                nav.pestana = .mas; nav.pilaMas.append(DestinoMas.diasLibres)
            })
        }
        if let r = datos.resenaSinResponder {
            out.append(Atender(id: "resena", simbolo: "star.bubble", titulo: "Reseña sin responder", texto: "\(r.nombre) · \(r.puntaje) de 5", accion: "Responder") {
                if estado.esProfesional { nav.pestana = .resenas } else { nav.pestana = .mas; nav.pilaMas.append(DestinoMas.resenas) }
            })
        }
        if datos.mpConectado == false && estado.esDuena {
            out.append(Atender(id: "mp", simbolo: "creditcard.and.123", titulo: "Cobrá señas con Mercado Pago", texto: "Así nadie reserva y no viene", accion: "Conectar") {
                nav.pestana = .mas; nav.pilaMas.append(DestinoMas.cobros)
            })
        }
        return out
    }

    @ViewBuilder private var paraAtender: some View {
        let lista = atender
        if !lista.isEmpty {
            VStack(alignment: .leading, spacing: 0) {
                Text("PARA ATENDER").rotulo().padding(.horizontal, 16).padding(.top, 14).padding(.bottom, 4)
                ForEach(Array(lista.enumerated()), id: \.element.id) { i, a in
                    if i > 0 { Rectangle().fill(C.linea).frame(height: 1) }
                    HStack(spacing: 12) {
                        Image(systemName: a.simbolo).font(.system(size: 20, weight: .semibold)).foregroundStyle(C.mentaTexto).frame(width: 24)
                        VStack(alignment: .leading, spacing: 2) {
                            Text(a.titulo).font(Tipo.lato(15, .bold)).foregroundStyle(C.tinta)
                            Text(a.texto).font(Tipo.lato(13)).foregroundStyle(C.sub).lineLimit(2)
                        }
                        Spacer(minLength: 0)
                        Button(a.accion, action: a.hacer).buttonStyle(BotonContorno(alto: 44, tam: 15, lleno: false))
                    }
                    .padding(.leading, 16).padding(.trailing, 12).padding(.vertical, 8).frame(minHeight: 64)
                }
            }
            .tarjeta(22)
        }
    }

    // MARK: La pila

    private enum Item: Identifiable {
        case sep(String)
        case ahora(String)
        case talon(Turno)
        case hueco(Dia.Hueco)
        case plegado(String, [Turno])
        var id: String {
            switch self {
            case .sep(let s): "sep-\(s)"
            case .ahora: "ahora"
            case .talon(let t): t.id
            case .hueco(let h): h.id
            case .plegado: "plegado"
            }
        }
    }

    private var items: [Item] {
        let m = Dia.minutosAhora(ahora)
        let ocupan = delDia.filter(\.ocupa).sorted { $0.inicio < $1.inicio }
        let pasados = ocupan.filter { $0.fin <= ahora }
        let enCurso = ocupan.filter { $0.inicio <= ahora && $0.fin > ahora }
        let despues = ocupan.filter { $0.inicio > ahora }
        let huecos = estado.esProfesional ? dias.flatMap(\.huecos) : dias.flatMap(\.huecos)
        let proximoHueco = huecos.filter { $0.hasta > m }.min { $0.desde < $1.desde }
        var out: [Item] = [.sep("AHORA"), .ahora(Formato.horaDe(ahora))]
        out += enCurso.map { .talon($0) }
        if let h = proximoHueco, h.desde < m + 60 || enCurso.isEmpty { out.append(.hueco(h)) }
        if !despues.isEmpty || huecos.count > 1 {
            out.append(.sep("DESPUÉS"))
            var resto: [(Int, Item)] = despues.map { (Formato.minutosDelDia($0.inicio), .talon($0)) }
            for h in huecos where h != proximoHueco && h.desde > m { resto.append((h.desde, .hueco(h))) }
            out += resto.sorted { $0.0 < $1.0 }.map(\.1)
        }
        if !pasados.isEmpty {
            let at = pasados.filter { $0.estado == "atendido" }.count, au = pasados.filter { $0.estado == "ausente" }.count
            var t = "Más temprano: \(at) \(at == 1 ? "atendido" : "atendidos")"
            if au > 0 { t += " y \(au) \(au == 1 ? "ausente" : "ausentes")" }
            if at == 0 && au == 0 { t = "Más temprano: \(pasados.count) \(pasados.count == 1 ? "turno" : "turnos")" }
            out.append(.plegado(t, pasados))
        }
        return out
    }

    @ViewBuilder private var pila: some View {
        let lectura = estado.soloLectura || estado.sinConexion
        ForEach(items) { it in
            switch it {
            case .sep(let s): Rotulo(s)
            case .ahora(let h):
                HStack(spacing: 8) {
                    Text("AHORA · \(h)").font(Tipo.lato(12, .black)).tracking(0.5).monospacedDigit()
                        .foregroundStyle(C.petroleoHondo).padding(.horizontal, 10).frame(height: 26).background(C.menta, in: Capsule())
                    Capsule().fill(C.menta).frame(height: 2)
                }
                .padding(.horizontal, -4)
                .accessibilityElement(children: .combine)
            case .talon(let t):
                VistaTalon(turno: t, mostrarProfesional: !estado.esProfesional, ahora: ahora, alTocar: { nav.turno = t },
                           alDeslizar: lectura ? nil : { e in Task { await datos.cambiar(estado, t, a: e) } })
                    .transition(datos.recien.contains(t.id) ? .move(edge: .top).combined(with: .opacity) : .identity)
            case .hueco(let h):
                let n = datos.esperanPara(h).count
                let oferta = datos.ofertaPara(h, dia: hoy)
                VistaHueco(hueco: h, accion: lectura ? "Ver" : (oferta != nil ? "Ofreciendo" : (n > 0 ? "Avisar a \(n)" : "Cargar")),
                           simboloAccion: n > 0 && !lectura ? "megaphone" : "plus",
                           avisado: avisados.contains(h.id) || oferta != nil,
                           alTocar: { cargarEn(h) },
                           alAccion: {
                    if lectura { return }
                    if n > 0 { ofrecer = h } else { cargarEn(h) }
                })
            case .plegado(let t, let pasados):
                Button { withAnimation(Movimiento.entrar) { verTemprano.toggle() } } label: {
                    HStack(spacing: 10) {
                        Image(systemName: "clock.arrow.circlepath").font(.system(size: 18))
                        Text(t).font(Tipo.lato(15)).frame(maxWidth: .infinity, alignment: .leading)
                        Image(systemName: verTemprano ? "chevron.up" : "chevron.down").font(.system(size: 15, weight: .semibold))
                    }
                    .foregroundStyle(C.sub).padding(.horizontal, 16).frame(minHeight: 52)
                    .contorno(18)
                }
                .buttonStyle(.plain)
                if verTemprano {
                    ForEach(pasados) { t in
                        VistaTalon(turno: t, mostrarProfesional: !estado.esProfesional, ahora: ahora, alTocar: { nav.turno = t },
                                   alDeslizar: lectura ? nil : { e in Task { await datos.cambiar(estado, t, a: e) } })
                    }
                }
            }
        }
        .animation(Movimiento.entrar, value: datos.turnos)
    }

    private func cargarEn(_ h: Dia.Hueco) {
        guard !estado.soloLectura, !estado.sinConexion else { return }
        nav.nuevo = NuevoTurnoInicio(profesionalId: h.profesionalId, inicio: Dia.fecha(hoy, minutos: h.desde))
    }

    // MARK: Vacío que enseña

    private var vacio: some View {
        let (d, h) = Dia.rango(dias)
        return VacioQueEnsena(rayas: "\(Dia.hhmm(d)) a \(Dia.hhmm(h)) · todo libre", titulo: "Hoy no tenés turnos todavía",
                              texto: "Compartí tu link para que te reserven solos, o cargá el que te pidieron por teléfono.") {
            VStack(spacing: 10) {
                Button { nav.compartir = true } label: { Label("Compartir mi link", systemImage: "square.and.arrow.up") }
                    .buttonStyle(BotonMenta(alto: 52, conSombra: false))
                if !estado.soloLectura {
                    Button { nav.nuevo = NuevoTurnoInicio() } label: { Label("Cargar un turno", systemImage: "plus") }
                        .buttonStyle(BotonContorno())
                }
            }
        }
    }
}

/// Un banner arriba del día (sólo lectura, sin conexión, primeros pasos).
struct Banner: View {
    struct Datos {
        var simbolo: String
        var titulo: String
        var texto: String
        var accion: String
        var petroleo: Bool
        var hacer: () -> Void
    }
    var b: Datos
    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(alignment: .top, spacing: 12) {
                Image(systemName: b.simbolo).font(.system(size: 22, weight: .semibold)).foregroundStyle(b.petroleo ? C.menta : C.tinta)
                VStack(alignment: .leading, spacing: 4) {
                    Text(b.titulo).font(Tipo.lato(16, .bold))
                    Text(b.texto).font(Tipo.lato(14)).foregroundStyle(b.petroleo ? .white.opacity(0.8) : C.sub).lineSpacing(2)
                }
            }
            if !b.accion.isEmpty { Button(b.accion, action: b.hacer).buttonStyle(BotonMenta(alto: 48, tam: 16, conSombra: false)) }
        }
        .foregroundStyle(b.petroleo ? .white : C.tinta)
        .padding(16)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background {
            if b.petroleo {
                ZStack { C.bloquePetroleo; Image(uiImage: Grano.imagen).resizable(resizingMode: .tile).opacity(0.5) }
            } else { C.tarjeta }
        }
        .clipShape(RoundedRectangle(cornerRadius: 22, style: .continuous))
        .shadow(color: Color(hex: 0x0F2328, alfa: 0.18), radius: 14, y: 10)
    }
}

/// «Avisar a la lista»: qué servicio se ofrece en ese hueco y a quiénes les llega, en orden.
struct OfrecerHueco: View {
    var hueco: Dia.Hueco
    var dia: Date
    var datos: DatosDia
    var listo: () -> Void
    @Environment(Estado.self) private var estado
    @Environment(\.dismiss) private var cerrar
    @State private var servicio: String?
    @State private var candidatos: [Anotado] = []
    @State private var enviando = false

    private var opciones: [Servicio] {
        let pide = Set(datos.esperanPara(hueco).map(\.servicioId))
        let delPro = datos.profesionales.first { $0.id == hueco.profesionalId }?.servicios ?? []
        return datos.servicios.filter { $0.activo && pide.contains($0.id) && (delPro.isEmpty || delPro.contains($0.id)) && $0.duracion <= hueco.minutos }
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("Libre \(Formato.duracion(hueco.minutos)) · \(Dia.hhmm(hueco.desde)) con \(hueco.profesional)").font(Tipo.lato(20, .black))
                        Text("Se lo ofrecemos de a una persona por vez, en el orden en que se anotaron. Cada una tiene un rato para tomarlo.")
                            .font(Tipo.lato(14)).foregroundStyle(C.sub)
                    }
                    Text("QUÉ SE OFRECE").rotulo()
                    if opciones.isEmpty {
                        Text("Nadie de la lista espera un servicio que entre en este hueco.").font(Tipo.lato(15)).foregroundStyle(C.sub)
                    }
                    LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 8) {
                        ForEach(opciones) { s in
                            let on = servicio == s.id
                            Button { servicio = s.id; Task { await traerCandidatos() } } label: {
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(s.nombre).font(Tipo.lato(15, .bold)).multilineTextAlignment(.leading)
                                    Text(Formato.duracion(s.duracion)).font(Tipo.lato(13)).opacity(0.8)
                                }
                                .frame(maxWidth: .infinity, minHeight: 64, alignment: .leading).padding(.horizontal, 12)
                                .foregroundStyle(on ? C.seleccionTexto : C.tinta)
                                .background(on ? C.seleccion : C.fondo, in: RoundedRectangle(cornerRadius: 16, style: .continuous))
                                .contorno(16, color: on ? .clear : C.linea)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                    if !candidatos.isEmpty {
                        Text("A QUIÉNES, EN ORDEN").rotulo()
                        ForEach(Array(candidatos.enumerated()), id: \.element.id) { i, a in
                            HStack(spacing: 12) {
                                Text("\(i + 1)").font(Tipo.lato(15, .black)).monospacedDigit().frame(width: 24)
                                Iniciales(nombre: a.nombre, tam: 36)
                                VStack(alignment: .leading) {
                                    Text(a.nombre).font(Tipo.lato(15, .bold))
                                    Text("Espera desde el \(Formato.dia(a.creadoEn))").font(Tipo.lato(12)).foregroundStyle(C.sub)
                                }
                            }
                        }
                    }
                }
                .padding(16)
            }
            .safeAreaInset(edge: .bottom) {
                Button {
                    Task { await lanzar() }
                } label: { Label(enviando ? "Avisando…" : "Avisar a la lista", systemImage: "megaphone") }
                .buttonStyle(BotonMenta())
                .disabled(servicio == nil || enviando)
                .padding(16).background(C.hoja)
            }
            .background(C.hoja)
            .navigationTitle("Avisar a la lista").navigationBarTitleDisplayMode(.inline)
            .toolbar { ToolbarItem(placement: .cancellationAction) { Button("Cancelar") { cerrar() } } }
            .onAppear { if opciones.count == 1 { servicio = opciones[0].id; Task { await traerCandidatos() } } }
        }
    }

    private struct Esperan: Decodable { var espera: [Anotado] }

    private func traerCandidatos() async {
        guard let servicio else { return }
        let q = [URLQueryItem(name: "servicio", value: servicio), URLQueryItem(name: "profesional", value: hueco.profesionalId),
                 URLQueryItem(name: "inicio", value: ISO8601DateFormatter().string(from: Dia.fecha(dia, minutos: hueco.desde)))]
        candidatos = (try? await estado.api.get("/v1/panel/espera/candidatos", query: q) as Esperan)?.espera ?? []
    }

    private func lanzar() async {
        guard let servicio else { return }
        enviando = true
        defer { enviando = false }
        do {
            try await AccionesTurno.ofrecer(estado, servicioId: servicio, profesionalId: hueco.profesionalId, inicio: Dia.fecha(dia, minutos: hueco.desde))
            listo()
            cerrar()
        } catch {
            estado.avisar(error)
        }
    }
}
