import SwiftUI

/// La caja de un turno (caja.md): lo que se cobra, lo pagado y lo que falta.
struct CuentaCaja: Codable, Equatable {
    struct Linea: Codable, Equatable, Hashable { var tipo: String; var concepto: String; var monto: Int; var cantidad: Int?; var productoId: String? }
    var turnoId: String
    var cliente: String
    var servicio: String
    var estadoTurno: String
    var estado: String
    var medio: String?
    var cobradoEn: Date?
    var lineas: [Linea]
    var total: Int
    var medios: [String]
}

struct Producto: Codable, Identifiable, Equatable, Hashable { var id: String; var nombre: String; var precio: Int; var stock: Int?; var activo: Bool; var orden: Int }

/// Un aviso que se le mandó al cliente por este turno (confirmación, recordatorio…).
struct RegistroAviso: Codable, Identifiable, Equatable, Hashable {
    var id: String
    var tipo: String
    var para: String
    var turnoId: String?
    var cliente: String?
    var estado: String
    var intentos: Int
    var ultimoError: String?
    var proximoIntento: Date
    var enviadoEn: Date?
    var creadoEn: Date
}

/// 09 Detalle del turno: hoja mediana y grande. Llamar y WhatsApp, pagos, membresía, cómo viene el
/// turno, los avisos que salieron y la nota interna. Abajo, «Marcar atendido y cobrar», Ausente,
/// Mover y Cancelar (y, si es fijo, cancelar los siguientes).
struct DetalleTurno: View {
    @State var turno: Turno
    @Environment(Estado.self) private var estado
    @Environment(Navegacion.self) private var nav
    @Environment(\.dismiss) private var cerrar
    @State private var caja: CuentaCaja?
    @State private var avisos: [RegistroAviso] = []
    @State private var nota = ""
    @State private var editandoNota = false
    @State private var cobrar = false
    @State private var mover = false
    @State private var confirmarCancelar = false
    @State private var confirmarSiguientes = false

    private var est: EstadoTurno { turno.visible() }
    private var lectura: Bool { estado.soloLectura || estado.sinConexion }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 14) {
                HStack(spacing: 8) {
                    Text(cabeceraDia).rotulo().frame(maxWidth: .infinity, alignment: .leading)
                    Menu { menu } label: {
                        Image(systemName: "ellipsis").font(.system(size: 18, weight: .bold)).frame(width: 44, height: 44).background(C.chip, in: Circle())
                    }
                    .accessibilityLabel("Más acciones")
                    Button { cerrar() } label: {
                        Image(systemName: "xmark").font(.system(size: 16, weight: .bold)).frame(width: 44, height: 44).background(C.chip, in: Circle())
                    }
                    .accessibilityLabel("Cerrar")
                }
                .foregroundStyle(C.tinta)
                talonGrande
                contacto
                if let caja, estado.vePrecios { pagos(caja) }
                if let m = caja?.lineas.first(where: { $0.tipo == "membresia" }) {
                    HStack(spacing: 12) {
                        Image(systemName: "ticket").font(.system(size: 20)).foregroundStyle(C.mentaTexto)
                        VStack(alignment: .leading, spacing: 1) {
                            Text("Usa \(m.concepto)").font(Tipo.lato(15, .bold))
                            Text("Lo cubre la membresía").font(Tipo.lato(13)).foregroundStyle(C.sub)
                        }
                    }
                    .padding(.horizontal, 16).padding(.vertical, 12).frame(maxWidth: .infinity, alignment: .leading)
                    .background(C.fondo, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
                }
                comoViene
                notaInterna
            }
            .padding(.horizontal, 16).padding(.top, 10).padding(.bottom, 200)
        }
        .background(C.hoja)
        .safeAreaInset(edge: .bottom) { if !lectura { acciones } }
        .task { await cargar() }
        .sheet(isPresented: $cobrar) {
            Cobrar(turno: turno, caja: caja) { await cargar() }
                .presentationDetents([.medium, .large]).presentationCornerRadius(Radio.hoja)
        }
        .sheet(isPresented: $mover) {
            Reprogramar(turno: turno) { t in turno = t; estado.version += 1 }
                .presentationDetents([.large]).presentationCornerRadius(Radio.hoja)
        }
        .confirmationDialog("¿Cancelar el turno de \(turno.cliente.nombre)?", isPresented: $confirmarCancelar, titleVisibility: .visible) {
            Button("Cancelar el turno", role: .destructive) { Task { await cambiar("cancelado") } }
            Button("Volver", role: .cancel) {}
        } message: { Text("Le avisamos y el horario queda libre para otra persona.") }
        .confirmationDialog("¿Cancelar este y los siguientes del turno fijo?", isPresented: $confirmarSiguientes, titleVisibility: .visible) {
            Button("Cancelar los siguientes", role: .destructive) { Task { await cancelarSiguientes() } }
        }
    }

    private var cabeceraDia: String {
        let esHoy = Formato.diaISO(turno.inicio) == Formato.diaISO(Date())
        let f = Formato.fechaLarga(turno.inicio).split(separator: " ").prefix(2).joined(separator: " ")
        return esHoy ? "HOY · \(f)" : f
    }

    // MARK: Talón grande (88 pt de troquel)

    private var talonGrande: some View {
        HStack(spacing: 0) {
            VStack(spacing: 2) {
                Text(Formato.horaDe(turno.inicio)).font(Tipo.lato(24, .black)).tracking(-0.5).monospacedDigit()
                Text("a \(Formato.horaDe(turno.fin))").font(Tipo.lato(13)).foregroundStyle(C.sub).monospacedDigit()
            }
            .frame(width: 88)
            Troquel()
            VStack(alignment: .leading, spacing: 4) {
                Text(turno.cliente.nombre).font(Tipo.lato(22, .bold)).tracking(-0.4)
                Text(turno.profesional.isEmpty ? turno.servicio : "\(turno.servicio) · \(turno.profesional)").font(Tipo.lato(14)).foregroundStyle(C.sub)
                PastillaEstado(texto: textoEstado, simbolo: est.simbolo, fondo: est.fondo, tinta: est.tinta, alto: 26)
            }
            .padding(.horizontal, 14).padding(.vertical, 12).frame(maxWidth: .infinity, alignment: .leading)
        }
        .frame(minHeight: 104)
        .foregroundStyle(C.tinta)
        .background(C.fondo, in: TalonShape(troquelX: 88, radio: 20, muesca: 9))
        .overlay { if est == .enCurso { TalonShape(troquelX: 88, radio: 20, muesca: 9).stroke(C.seleccion, lineWidth: 2) } }
        .accessibilityElement(children: .combine)
    }

    private var textoEstado: String {
        if est == .enCurso { return "En curso · faltan \(max(1, Int(turno.fin.timeIntervalSinceNow / 60))) min" }
        if est == .pendientePago, let v = turno.venceEn { return "Seña pendiente · hasta las \(Formato.horaDe(v))" }
        return est.texto
    }

    // MARK: Contacto

    private var tel: String { turno.cliente.telefono.filter { "+0123456789".contains($0) } }

    private var contacto: some View {
        HStack(spacing: 8) {
            Button { if let u = URL(string: "tel:\(tel)") { UIApplication.shared.open(u) } } label: { Label("Llamar", systemImage: "phone") }
                .buttonStyle(BotonChip())
            Button {
                var d = tel.filter(\.isNumber)
                if d.hasPrefix("0") { d.removeFirst() }
                if !d.hasPrefix("54") { d = "549" + d }
                if let u = URL(string: "https://wa.me/\(d)") { UIApplication.shared.open(u) }
            } label: {
                HStack(spacing: 8) { Image("LogoWhatsApp").resizable().frame(width: 22, height: 22); Text("WhatsApp") }
            }
            .buttonStyle(BotonChip())
        }
        .disabled(tel.isEmpty)
    }

    // MARK: Pagos

    private func pagos(_ c: CuentaCaja) -> some View {
        VStack(spacing: 0) {
            Text("PAGOS").rotulo().frame(maxWidth: .infinity, alignment: .leading).padding(.bottom, 8)
            ForEach(Array(c.lineas.enumerated()), id: \.offset) { _, l in
                HStack {
                    if l.tipo == "sena" { Image("LogoMercadoPago").resizable().scaledToFit().frame(width: 20, height: 20) }
                    Text(l.tipo == "sena" ? "Seña pagada" : (l.cantidad.map { $0 > 1 ? "\(l.concepto) × \($0)" : l.concepto } ?? l.concepto))
                    Spacer()
                    Text(l.monto < 0 ? "− \(Formato.pesos(-l.monto))" : Formato.pesos(l.monto))
                }
                .font(Tipo.lato(15)).monospacedDigit().padding(.vertical, 6)
            }
            HStack {
                Text(c.estado == "cobrado" ? "Cobrado\(c.medio.map { " · \(medioTexto($0))" } ?? "")" : c.estado == "debe" ? "Debe" : "Falta cobrar")
                Spacer()
                Text(Formato.pesos(c.total))
            }
            .font(Tipo.lato(17, .black)).monospacedDigit()
            .padding(.top, 10).overlay(alignment: .top) { Troquel(vertical: false) }
            .padding(.top, 6)
        }
        .padding(.horizontal, 16).padding(.vertical, 14)
        .background(C.fondo, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
    }

    private func medioTexto(_ m: String) -> String {
        ["efectivo": "Efectivo", "transferencia": "Transferencia", "mercadopago": "QR de Mercado Pago", "debito": "Débito", "credito": "Crédito", "despues": "Después"][m] ?? m
    }

    // MARK: Cómo viene

    private struct Hito: Hashable { var simbolo: String; var texto: String; var hora: String; var hecho: Bool; var alerta = false }

    private var hitos: [Hito] {
        var out: [Hito] = []
        let origen = ["web": "Reservó por la web", "panel": "Lo cargaron en el panel", "app": "Reservó por la app", "whatsapp": "Reservó por WhatsApp", "espera": "Lo tomó de la lista de espera"][turno.origen] ?? "Reservó"
        out.append(Hito(simbolo: "checkmark.circle.fill", texto: origen, hora: corto(turno.creadoEn), hecho: true))
        if caja?.lineas.contains(where: { $0.tipo == "sena" }) == true {
            out.append(Hito(simbolo: "checkmark.circle.fill", texto: "Pagó la seña", hora: "", hecho: true))
        } else if turno.estado == "pendiente_pago" {
            out.append(Hito(simbolo: "hourglass", texto: "Esperando la seña\(turno.venceEn.map { " hasta las \(Formato.horaDe($0))" } ?? "")", hora: "", hecho: false, alerta: true))
        }
        for a in avisos {
            let que = ["confirmacion": "Le mandamos la confirmación", "recordatorio": "Le mandamos el recordatorio", "whatsapp_confirmar": "Le pedimos que confirme por WhatsApp", "cancelacion": "Le avisamos que se canceló", "reprogramacion": "Le avisamos el cambio de horario", "resena": "Le pedimos la reseña"][a.tipo] ?? "Aviso: \(a.tipo)"
            let falla = a.estado == "fallido" || a.estado == "error"
            out.append(Hito(simbolo: falla ? "exclamationmark.circle.fill" : (a.enviadoEn != nil ? "checkmark.circle.fill" : "clock"),
                            texto: falla ? "\(que) · no salió" : que, hora: a.enviadoEn.map(corto) ?? "", hecho: a.enviadoEn != nil, alerta: falla))
        }
        if let c = turno.confirmadoEn {
            let por = ["whatsapp": "Confirmó por WhatsApp", "web": "Confirmó por la web", "panel": "Lo confirmaron en el panel"][turno.confirmadoPor ?? ""] ?? "Confirmó"
            out.append(Hito(simbolo: "checkmark.circle.fill", texto: por, hora: corto(c), hecho: true))
        }
        if let c = turno.cambioPedidoEn { out.append(Hito(simbolo: "arrow.triangle.2.circlepath", texto: "Pidió cambiarlo", hora: corto(c), hecho: false, alerta: true)) }
        if let m = turno.llamarMotivo, turno.llamarDesde != nil {
            let t = ["cambio": "Hay que llamarla: pidió cambiarlo", "sin_respuesta": "Hay que llamarla: no contestó el WhatsApp", "no_llego": "Hay que llamarla: no le llegó el WhatsApp"][m] ?? "Hay que llamarla"
            out.append(Hito(simbolo: "phone.badge.waveform", texto: t, hora: "", hecho: false, alerta: true))
        }
        if est == .enCurso { out.append(Hito(simbolo: "timelapse", texto: "Está en el turno", hora: Formato.horaDe(turno.inicio), hecho: false)) }
        if turno.estado == "atendido" { out.append(Hito(simbolo: "checkmark.circle.fill", texto: "Atendido", hora: "", hecho: true)) }
        if turno.estado == "ausente" { out.append(Hito(simbolo: "person.slash", texto: "No vino", hora: "", hecho: false, alerta: true)) }
        return out
    }

    private func corto(_ f: Date) -> String {
        Formato.diaISO(f) == Formato.diaISO(Date()) ? "hoy · \(Formato.horaDe(f))" : "\(Formato.dia(f)) · \(Formato.horaDe(f))"
    }

    private var comoViene: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("CÓMO VIENE").rotulo()
            ForEach(hitos, id: \.self) { h in
                HStack(spacing: 10) {
                    Image(systemName: h.simbolo).font(.system(size: 18)).foregroundStyle(h.alerta ? C.pagoTinta : (h.hecho ? C.mentaTexto : C.tinta))
                    Text(h.texto).font(Tipo.lato(14)).frame(maxWidth: .infinity, alignment: .leading)
                    Text(h.hora).font(Tipo.lato(12)).foregroundStyle(C.sub).monospacedDigit()
                }
            }
            if let f = avisos.first(where: { $0.estado == "fallido" || $0.estado == "error" }), !lectura {
                Button("Volver a mandarlo") { Task { try? await estado.api.hacer("POST", "/v1/panel/avisos/\(f.id)/reenviar"); await cargar() } }
                    .buttonStyle(BotonTexto())
            }
        }
        .padding(.horizontal, 16).padding(.vertical, 14)
        .background(C.fondo, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
    }

    // MARK: Nota

    private struct CambioNota: Encodable { var nota: String }

    private var notaInterna: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .top, spacing: 10) {
                Image(systemName: "note.text").font(.system(size: 18)).foregroundStyle(C.sub)
                VStack(alignment: .leading, spacing: 2) {
                    Text("Nota interna · sólo la ve el equipo").font(Tipo.lato(13, .bold))
                    if editandoNota {
                        TextField("Escribí la nota", text: $nota, axis: .vertical).font(Tipo.lato(15)).lineLimit(2...6)
                    } else {
                        Text(turno.nota.isEmpty ? "Sin nota" : turno.nota).font(Tipo.lato(15)).foregroundStyle(turno.nota.isEmpty ? C.sub : C.tinta)
                    }
                }
                Spacer(minLength: 0)
                if !lectura {
                    Button(editandoNota ? "Guardar" : "Editar") {
                        if editandoNota {
                            Task {
                                do {
                                    try await estado.api.hacer("PATCH", "/v1/panel/turnos/\(turno.id)", CambioNota(nota: nota))
                                    turno.nota = nota
                                    editandoNota = false
                                } catch { estado.avisar(error) }
                            }
                        } else { nota = turno.nota; editandoNota = true }
                    }
                    .buttonStyle(BotonTexto())
                }
            }
        }
        .padding(.horizontal, 16).padding(.vertical, 12)
        .background(C.fondo, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
    }

    // MARK: Acciones

    private var acciones: some View {
        VStack(spacing: 10) {
            if turno.estado == "pendiente" || turno.estado == "pendiente_pago" {
                Button { Task { await cambiar("confirmado") } } label: { Label("Confirmar el turno", systemImage: "checkmark.circle") }
                    .buttonStyle(BotonMenta())
            } else if turno.estado != "cancelado" && caja?.estado != "cobrado" && estado.vePrecios {
                Button { cobrar = true } label: { Label(turno.estado == "atendido" ? "Cobrar" : "Marcar atendido y cobrar", systemImage: "checkmark.circle") }
                    .buttonStyle(BotonMenta())
            } else if turno.estado != "atendido" && turno.estado != "cancelado" {
                Button { Task { await cambiar("atendido") } } label: { Label("Marcar atendido", systemImage: "checkmark.circle") }
                    .buttonStyle(BotonMenta())
            }
            if turno.estado != "cancelado" {
                HStack(spacing: 8) {
                    if EstadoTurno.puedePasar(turno.estado, "ausente") && turno.estado != "ausente" {
                        Button { Task { await cambiar("ausente") } } label: { Label("Ausente", systemImage: "person.slash") }
                    }
                    Button { mover = true } label: { Label("Mover", systemImage: "calendar.badge.clock") }
                    Button { confirmarCancelar = true } label: { Label("Cancelar", systemImage: "calendar.badge.minus") }
                }
                .buttonStyle(BotonContorno(alto: 48, tam: 14))
                .labelStyle(.titleAndIcon)
            }
        }
        .padding(.horizontal, 16).padding(.top, 12).padding(.bottom, 8)
        .background(C.hoja.overlay(alignment: .top) { Rectangle().fill(C.linea).frame(height: 1) })
    }

    @ViewBuilder private var menu: some View {
        if !lectura {
            if turno.estado == "atendido" || turno.estado == "ausente" {
                Button("Volver a confirmado", systemImage: "arrow.uturn.backward") { Task { await cambiar("confirmado") } }
            }
            if turno.serieId != nil {
                Button("Cancelar este y los siguientes", systemImage: "repeat", role: .destructive) { confirmarSiguientes = true }
            }
            if turno.llamarDesde != nil {
                Button("Ya la llamé", systemImage: "phone.badge.checkmark") {
                    Task { try? await estado.api.hacer("POST", "/v1/panel/turnos/\(turno.id)/llamado"); await recargarTurno() }
                }
            }
        }
        if let id = turno.cliente.id {
            Button("Ver la ficha", systemImage: "person.text.rectangle") {
                cerrar()
                nav.pestana = .clientes
                nav.pilaClientes.append(DestinoClientes.ficha(id))
            }
        }
        ShareLink(item: "Tu turno en \(estado.negocio?.nombre ?? ""): \(Formato.diaYHora(turno.inicio)), \(turno.servicio). Código \(turno.codigo).") {
            Label("Compartir los datos", systemImage: "square.and.arrow.up")
        }
    }

    // MARK: Datos

    private struct Avisos: Decodable { var avisos: [RegistroAviso] }

    private func cargar() async {
        async let c: CuentaCaja? = try? estado.api.get("/v1/panel/turnos/\(turno.id)/caja")
        async let a: Avisos? = try? estado.api.get("/v1/panel/avisos", query: [URLQueryItem(name: "turno", value: turno.id)])
        let (cc, aa) = await (c, a)
        caja = cc
        avisos = aa?.avisos ?? []
        await recargarTurno()
    }

    private func recargarTurno() async {
        if let t: Turno = try? await estado.api.get("/v1/panel/turnos/\(turno.id)") { turno = t }
    }

    private func cambiar(_ nuevo: String) async {
        let antes = turno.estado
        do {
            try await estado.api.hacer("PATCH", "/v1/panel/turnos/\(turno.id)", AccionesTurno.CambioEstado(estado: nuevo))
            turno.estado = nuevo
            Haptico.exito()
            estado.version += 1
            if nuevo == "cancelado" || nuevo == "ausente" {
                let t = turno
                estado.avisar(Toast(simbolo: "checkmark", titulo: nuevo == "cancelado" ? "Turno cancelado" : "Marcado ausente", texto: "\(t.cliente.nombre) · \(Formato.horaDe(t.inicio))", accion: "Deshacer")) {
                    try? await estado.api.hacer("PATCH", "/v1/panel/turnos/\(t.id)", AccionesTurno.CambioEstado(estado: antes))
                }
                cerrar()
            }
        } catch {
            estado.avisar(error)
        }
    }

    private struct Cancelados: Decodable { var cancelados: [String] }

    private func cancelarSiguientes() async {
        do {
            let r: Cancelados = try await estado.api.enviar("POST", "/v1/panel/turnos/\(turno.id)/cancelar-siguientes", [String: String]())
            estado.version += 1
            estado.avisar("Cancelamos \(r.cancelados.count) \(r.cancelados.count == 1 ? "turno" : "turnos")", texto: "Les avisamos a todos.")
            cerrar()
        } catch { estado.avisar(error) }
    }
}

/// Marcar atendido y cobrar (caja.md): medio de pago, productos vendidos y «Cobrar después».
struct Cobrar: View {
    var turno: Turno
    var caja: CuentaCaja?
    var listo: () async -> Void
    @Environment(Estado.self) private var estado
    @Environment(\.dismiss) private var cerrar
    @State private var medio = "efectivo"
    @State private var productos: [Producto] = []
    @State private var elegidos: Set<String> = []
    @State private var enviando = false

    private let medios: [(String, String, String)] = [
        ("efectivo", "Efectivo", "banknote"), ("transferencia", "Transferencia", "building.columns"),
        ("mercadopago", "QR de Mercado Pago", "qrcode"), ("debito", "Débito", "creditcard"), ("credito", "Crédito", "creditcard.fill")
    ]

    private var total: Int {
        (caja?.total ?? turno.precio) + productos.filter { elegidos.contains($0.id) }.reduce(0) { $0 + $1.precio }
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    VStack(alignment: .leading, spacing: 2) {
                        Text("Falta cobrar").rotulo()
                        Text(Formato.pesos(total)).font(Tipo.lato(40, .black)).monospacedDigit().foregroundStyle(C.tinta)
                        Text("\(turno.cliente.nombre) · \(turno.servicio)").font(Tipo.lato(14)).foregroundStyle(C.sub)
                    }
                    Text("CÓMO PAGA").rotulo()
                    LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 8) {
                        ForEach(medios, id: \.0) { m in
                            let on = medio == m.0
                            Button { medio = m.0; Haptico.seleccion() } label: {
                                Label(m.1, systemImage: m.2).font(Tipo.lato(15, .bold)).frame(maxWidth: .infinity, minHeight: 56)
                                    .foregroundStyle(on ? C.seleccionTexto : C.tinta)
                                    .background(on ? C.seleccion : C.fondo, in: RoundedRectangle(cornerRadius: 16, style: .continuous))
                                    .contorno(16, color: on ? .clear : C.linea)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                    if !productos.isEmpty {
                        Text("SUMAR PRODUCTOS").rotulo()
                        ForEach(productos) { p in
                            Button {
                                if elegidos.contains(p.id) { elegidos.remove(p.id) } else { elegidos.insert(p.id) }
                            } label: {
                                HStack {
                                    Image(systemName: elegidos.contains(p.id) ? "checkmark.circle.fill" : "circle").font(.system(size: 22)).foregroundStyle(C.mentaTexto)
                                    Text(p.nombre).font(Tipo.lato(16)).foregroundStyle(C.tinta)
                                    Spacer()
                                    Text(Formato.pesos(p.precio)).font(Tipo.lato(15)).monospacedDigit().foregroundStyle(C.sub)
                                }
                                .frame(minHeight: 48)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                }
                .padding(16)
            }
            .background(C.hoja)
            .safeAreaInset(edge: .bottom) {
                VStack(spacing: 8) {
                    Button(enviando ? "Cobrando…" : "Cobrar \(Formato.pesos(total))") { Task { await cobrar(medio) } }
                        .buttonStyle(BotonMenta()).disabled(enviando)
                    Button("Cobrar después") { Task { await cobrar("despues") } }.buttonStyle(BotonTexto(color: C.tinta))
                }
                .padding(16).background(C.hoja)
            }
            .navigationTitle("Cobrar").navigationBarTitleDisplayMode(.inline)
            .toolbar { ToolbarItem(placement: .cancellationAction) { Button("Cancelar") { cerrar() } } }
            .task {
                struct P: Decodable { var productos: [Producto] }
                productos = (try? await estado.api.get("/v1/panel/productos", query: [URLQueryItem(name: "activos", value: "1")]) as P)?.productos ?? []
            }
        }
    }

    private struct Pedido: Encodable {
        struct L: Encodable { var productoId: String; var cantidad: Int }
        var medio: String
        var productos: [L]
    }

    private func cobrar(_ m: String) async {
        enviando = true
        defer { enviando = false }
        do {
            if turno.estado != "atendido" {
                try await estado.api.hacer("PATCH", "/v1/panel/turnos/\(turno.id)", AccionesTurno.CambioEstado(estado: "atendido"))
            }
            try await estado.api.hacer("POST", "/v1/panel/turnos/\(turno.id)/caja", Pedido(medio: m, productos: elegidos.map { .init(productoId: $0, cantidad: 1) }))
            Haptico.exito()
            estado.version += 1
            estado.avisar(m == "despues" ? "Atendido · queda debiendo" : "Atendido y cobrado", texto: m == "despues" ? "Lo cobrás desde la caja cuando pague." : Formato.pesos(total))
            await listo()
            cerrar()
        } catch {
            estado.avisar(error)
        }
    }
}

/// Mover un turno (reprogramar): otro día, otra profesional, un horario libre o encimado.
struct Reprogramar: View {
    var turno: Turno
    var listo: (Turno) -> Void
    @Environment(Estado.self) private var estado
    @Environment(\.dismiss) private var cerrar
    @State private var fecha: Date
    @State private var profesional: String?
    @State private var profesionales: [Profesional] = []
    @State private var libres: [HorarioLibre] = []
    @State private var elegido: HorarioLibre?
    @State private var forzar = false
    @State private var hora = Date()

    init(turno: Turno, listo: @escaping (Turno) -> Void) {
        self.turno = turno
        self.listo = listo
        _fecha = State(initialValue: Formato.calendario.startOfDay(for: max(turno.inicio, Date())))
        _profesional = State(initialValue: turno.profesionalId)
        _hora = State(initialValue: turno.inicio)
    }

    var body: some View {
        NavigationStack {
            Form {
                Section("Día") {
                    DatePicker("Día", selection: $fecha, in: Formato.calendario.startOfDay(for: Date())..., displayedComponents: .date)
                        .datePickerStyle(.graphical).tint(C.mentaTexto)
                }
                Section("Con quién") {
                    Picker("Profesional", selection: $profesional) {
                        ForEach(profesionales.filter { $0.activo && (turno.servicioId == nil || $0.servicios.isEmpty || $0.servicios.contains(turno.servicioId!)) }) { p in
                            Text(p.nombre).tag(Optional(p.id))
                        }
                    }
                }
                Section(forzar ? "Hora" : "Horarios libres") {
                    if forzar {
                        DatePicker("Hora", selection: $hora, displayedComponents: .hourAndMinute)
                    } else if libres.isEmpty {
                        Text("No hay horarios libres ese día.").foregroundStyle(C.sub)
                    } else {
                        LazyVGrid(columns: Array(repeating: GridItem(.flexible()), count: 4), spacing: 8) {
                            ForEach(libres, id: \.inicio) { h in
                                Button(h.hora) { elegido = h }
                                    .font(Tipo.lato(16, .black)).monospacedDigit()
                                    .frame(maxWidth: .infinity, minHeight: 44)
                                    .foregroundStyle(elegido?.inicio == h.inicio ? C.seleccionTexto : C.huecoTinta)
                                    .background(elegido?.inicio == h.inicio ? C.seleccion : C.mentaSuave, in: RoundedRectangle(cornerRadius: 12))
                                    .buttonStyle(.plain)
                            }
                        }
                    }
                    Toggle("Encimarlo aunque esté ocupado", isOn: $forzar).tint(C.menta)
                }
            }
            .navigationTitle("Mover el turno").navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancelar") { cerrar() } }
                ToolbarItem(placement: .confirmationAction) { Button("Mover") { Task { await mover() } }.disabled(!forzar && elegido == nil) }
            }
            .task { await cargar() }
            .onChange(of: fecha) { _, _ in Task { await traerLibres() } }
            .onChange(of: profesional) { _, _ in Task { await traerLibres() } }
        }
    }

    private struct Pros: Decodable { var profesionales: [Profesional] }
    private struct Libres: Decodable { var horarios: [HorarioLibre] }

    private func cargar() async {
        profesionales = (try? await estado.api.get("/v1/panel/profesionales") as Pros)?.profesionales ?? []
        await traerLibres()
    }

    private func traerLibres() async {
        elegido = nil
        guard let s = turno.servicioId else { return }
        var q = [URLQueryItem(name: "servicio", value: s), URLQueryItem(name: "fecha", value: Formato.diaISO(fecha))]
        if let profesional { q.append(URLQueryItem(name: "profesional", value: profesional)) }
        libres = (try? await estado.api.get("/v1/panel/libres", query: q) as Libres)?.horarios ?? []
    }

    private struct Pedido: Encodable { var inicio: Date; var profesionalId: String; var forzar: Bool }

    private func mover() async {
        let c = Formato.calendario.dateComponents([.hour, .minute], from: hora)
        let inicio = forzar ? Dia.fecha(fecha, minutos: (c.hour ?? 0) * 60 + (c.minute ?? 0)) : (elegido?.inicio ?? turno.inicio)
        do {
            let t: Turno = try await estado.api.enviar("POST", "/v1/panel/turnos/\(turno.id)/reprogramar",
                Pedido(inicio: inicio, profesionalId: profesional ?? turno.profesionalId ?? "", forzar: forzar))
            Haptico.exito()
            estado.avisar("Turno movido", texto: "\(Formato.diaYHora(t.inicio)). Le avisamos a \(t.cliente.nombre).")
            listo(t)
            cerrar()
        } catch {
            estado.avisar(error)
        }
    }
}
