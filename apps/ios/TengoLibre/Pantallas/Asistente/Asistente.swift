import SwiftUI

// MARK: Modelos (docs/api/asistente.md)

/// GET /v1/panel/asistente: si hay IA, los ejemplos y cuánto queda del cupo del mes.
struct EstadoAsistente: Decodable {
    var ia: Bool
    /// La plataforma tiene IA, la dueña la prendió para el negocio y quien mira la puede prender.
    var iaPlataforma: Bool?
    var iaNegocio: Bool?
    var puedePrender: Bool?
    var sugerencias: [String]
    var cupo: CupoAsistente?
}

struct CupoAsistente: Decodable {
    var cupoMensual: Int
    var usadasMes: Int
    var usadasHoy: Int
    var topeDiario: Int
}

/// Un cambio en la agenda que el asistente no hace solo: vuelve para que la persona lo confirme.
struct PropuestaAsistente: Decodable, Identifiable, Equatable {
    var id: String
    var accion: String
    var resumen: String
    var venceEn: String?
    /// Lo que dice el botón («Sí, cancelarlo»): un «Confirmar» / «Cancelar» al lado de «Cancelar el turno
    /// de Rosa» se leía como cancelar el turno.
    var boton: String?

    var textoBoton: String {
        if let boton, !boton.isEmpty { return boton }
        switch accion {
        case "cancelar_turno": return "Sí, cancelarlo"
        case "reprogramar_turno": return "Sí, moverlo"
        case "confirmar_turno": return "Sí, confirmarlo"
        case "crear_turno": return "Sí, cargarlo"
        case "ofrecer_a_la_lista": return "Sí, ofrecerlo"
        default: return "Sí, hacerlo"
        }
    }
}

/// Un turno de una lista del asistente: se muestra como fila que abre el turno.
struct ItemTurnoAsistente: Decodable, Identifiable, Equatable {
    var id: String
    var dia: String
    var hora: String
    var cliente: String
    var servicio: String
    var profesional: String?
    var estado: String?
}

/// No hay más IA por ahora (mes, día o plataforma): el intérprete sigue contestando.
struct TopeAsistente: Decodable, Equatable {
    var motivo: String
    var mensaje: String
}

/// POST /v1/panel/asistente (y lo que vuelve de /confirmar y /descartar, que sólo trae `respuesta`).
struct RespuestaAsistente: Decodable {
    var respuesta: String
    var propuestas: [PropuestaAsistente]?
    var origen: String?
    var tope: TopeAsistente?
    var encabezado: String?
    var items: [ItemTurnoAsistente]?
}

/// Un turno de la charla, como lo pide la API en `historial`.
struct TurnoChat: Codable, Equatable {
    var rol: String
    var texto: String
}

// MARK: La entrada en Hoy

/// El renglón de Hoy que abre el asistente: preguntar por la agenda o pedir un cambio, escrito o dictado.
/// La hoja la presenta Principal (nav.asistente). Sin conexión queda apagado.
struct EntradaAsistente: View {
    @Environment(Estado.self) private var estado
    @Environment(Navegacion.self) private var nav

    var body: some View {
        Button {
            Haptico.suave()
            nav.asistente = true
        } label: {
            HStack(spacing: 12) {
                Image(systemName: "sparkles").font(.system(size: 18, weight: .semibold)).foregroundStyle(C.petroleoHondo)
                    .frame(width: 40, height: 40).background(C.menta, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
                VStack(alignment: .leading, spacing: 2) {
                    Text("Preguntale al asistente").font(Tipo.lato(16, .bold)).foregroundStyle(C.tinta)
                    Text(estado.sinConexion ? "Sin conexión" : "«¿Qué tengo mañana?» o «Pasá a Juana al jueves»").font(Tipo.lato(13)).foregroundStyle(C.sub).lineLimit(2)
                }
                Spacer(minLength: 0)
                Image(systemName: "chevron.right").font(.system(size: 15, weight: .semibold)).foregroundStyle(C.sub)
            }
            .padding(.horizontal, 16).padding(.vertical, 12).frame(minHeight: 64)
            .tarjeta(22)
        }
        .buttonStyle(.plain)
        .disabled(estado.sinConexion)
        .opacity(estado.sinConexion ? 0.55 : 1)
    }
}

// MARK: La hoja

/// La charla con el asistente: los ejemplos, las burbujas (las listas de turnos como filas), «Pensando…»,
/// las propuestas con el botón de la acción y «No, dejalo así», el aviso del tope, el cupo del mes y, para
/// la dueña, prender la IA. La charla vive mientras la hoja está abierta.
struct Asistente: View {
    @Environment(Estado.self) private var estado
    @Environment(\.dismiss) private var cerrar

    struct Mensaje: Identifiable, Equatable {
        var id = UUID()
        var rol: String
        var texto: String
        var propuestas: [PropuestaAsistente] = []
        var encabezado: String?
        var items: [ItemTurnoAsistente] = []
        /// Un error de la API: se muestra en la charla pero no viaja en el historial.
        var error = false
    }
    @State private var prendiendo = false

    @State private var info: EstadoAsistente?
    @State private var mensajes: [Mensaje] = []
    @State private var texto = ""
    @State private var pensando = false
    @State private var tope: TopeAsistente?
    /// Propuestas ya resueltas: id → «Hecho», «Cancelada»…
    @State private var resueltas: [String: String] = [:]
    @State private var resolviendo: String?
    @FocusState private var escribiendo: Bool

    private var puedeEnviar: Bool { !texto.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && !pensando }

    var body: some View {
        NavigationStack {
            ScrollViewReader { lector in
                ScrollView {
                    VStack(alignment: .leading, spacing: 12) {
                        if mensajes.isEmpty { inicio }
                        ForEach(mensajes) { m in burbuja(m) }
                        if pensando {
                            Text("Pensando…").font(Tipo.lato(15)).foregroundStyle(C.sub)
                                .padding(.horizontal, 14).padding(.vertical, 10)
                                .background(C.chip, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
                                .transition(.opacity)
                        }
                        Color.clear.frame(height: 1).id("fin")
                    }
                    .padding(16)
                    .animation(Movimiento.entrar, value: mensajes)
                    .animation(Movimiento.entrar, value: pensando)
                }
                .scrollDismissesKeyboard(.interactively)
                .onChange(of: mensajes.count) { _, _ in withAnimation(Movimiento.entrar) { lector.scrollTo("fin", anchor: .bottom) } }
                .onChange(of: pensando) { _, _ in withAnimation(Movimiento.entrar) { lector.scrollTo("fin", anchor: .bottom) } }
            }
            .safeAreaInset(edge: .bottom) { abajo }
            .background(C.hoja)
            .navigationTitle("Asistente").navigationBarTitleDisplayMode(.inline)
            .toolbar { ToolbarItem(placement: .cancellationAction) { Button("Cerrar") { cerrar() } } }
            .task { await cargar() }
        }
    }

    // MARK: Inicio y burbujas

    @ViewBuilder private var inicio: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text("¿En qué te ayudo?").font(Tipo.lato(22, .black)).foregroundStyle(C.tinta)
            Text("Preguntá por tu agenda o pedí un cambio. Antes de tocar un turno, te lo muestro para que lo confirmes.")
                .font(Tipo.lato(14)).foregroundStyle(C.sub)
        }
        if let s = info?.sugerencias, !s.isEmpty {
            Text("PROBÁ CON").rotulo().padding(.top, 4)
            VStack(alignment: .leading, spacing: 8) {
                ForEach(s, id: \.self) { q in
                    Button { Task { await preguntar(q) } } label: { Chip(texto: q, simbolo: "sparkles") }
                        .buttonStyle(.plain).disabled(pensando)
                }
            }
        }
    }

    @ViewBuilder private func burbuja(_ m: Mensaje) -> some View {
        let mio = m.rol == "usuario"
        VStack(alignment: mio ? .trailing : .leading, spacing: 8) {
            if !m.items.isEmpty {
                // Una lista de turnos: una fila por turno (antes, renglones de texto que se partían en dos).
                VStack(alignment: .leading, spacing: 0) {
                    Text(m.encabezado ?? m.texto).font(Tipo.lato(15, .bold)).foregroundStyle(C.tinta).padding(.bottom, 6)
                    ForEach(m.items) { t in
                        Button { abrirTurno(t.id) } label: {
                            HStack(spacing: 12) {
                                Text(t.hora).font(Tipo.lato(15, .bold)).monospacedDigit().frame(width: 48, alignment: .leading)
                                VStack(alignment: .leading, spacing: 1) {
                                    Text(t.cliente).font(Tipo.lato(15, .bold)).lineLimit(1)
                                    Text([t.servicio, t.profesional, t.estado].compactMap { $0 }.filter { !$0.isEmpty }.joined(separator: " · "))
                                        .font(Tipo.lato(13)).foregroundStyle(C.sub).lineLimit(1)
                                }
                                Spacer(minLength: 0)
                                Image(systemName: "chevron.right").font(.system(size: 13, weight: .semibold)).foregroundStyle(C.sub)
                            }
                            .foregroundStyle(C.tinta).frame(minHeight: 48).contentShape(Rectangle())
                        }
                        .buttonStyle(.plain)
                        .accessibilityLabel("\(t.hora), \(t.cliente), \(t.servicio)")
                        if t.id != m.items.last?.id { Divider() }
                    }
                }
                .padding(.horizontal, 14).padding(.vertical, 10)
                .background(C.chip, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
            } else {
                Text(m.texto).font(Tipo.lato(15)).lineSpacing(2)
                    .foregroundStyle(mio ? C.petroleoHondo : (m.error ? C.ausenteTinta : C.tinta))
                    .padding(.horizontal, 14).padding(.vertical, 10)
                    .background(mio ? C.menta : (m.error ? C.ausenteFondo : C.chip), in: RoundedRectangle(cornerRadius: 18, style: .continuous))
                    .textSelection(.enabled)
            }
            ForEach(m.propuestas) { p in tarjetaPropuesta(p) }
        }
        .frame(maxWidth: .infinity, alignment: mio ? .trailing : .leading)
        .padding(.leading, mio ? 48 : 0)
    }

    /// Una propuesta: el resumen y los dos botones; resuelta, queda apagada con lo que pasó.
    private func tarjetaPropuesta(_ p: PropuestaAsistente) -> some View {
        let fin = resueltas[p.id]
        return VStack(alignment: .leading, spacing: 12) {
            Text("PARA CONFIRMAR").rotulo()
            Text(p.resumen).font(Tipo.lato(15)).foregroundStyle(C.tinta).lineSpacing(2)
            if let fin {
                PastillaEstado(texto: fin, simbolo: fin == "Hecho" ? "checkmark" : "minus",
                               fondo: fin == "Hecho" ? C.mentaSuave : C.chip, tinta: fin == "Hecho" ? C.mentaTexto : C.sub, alto: 28)
            } else {
                HStack(spacing: 8) {
                    Button(resolviendo == p.id ? "Un momento…" : p.textoBoton) { Task { await confirmar(p) } }
                        .buttonStyle(BotonMenta(alto: 44, tam: 15, conSombra: false))
                    Button("No, dejalo así") { Task { await descartar(p) } }
                        .buttonStyle(BotonTexto(color: C.sub))
                }
                .disabled(resolviendo != nil)
            }
        }
        .padding(14).frame(maxWidth: .infinity, alignment: .leading)
        .background(C.fondo, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
        .contorno(18)
        .opacity(fin == nil ? 1 : 0.7)
    }

    // MARK: Abajo: tope, campo y cupo

    private var abajo: some View {
        VStack(spacing: 8) {
            if let tope {
                HStack(alignment: .top, spacing: 8) {
                    Image(systemName: "exclamationmark.triangle").foregroundStyle(C.pagoTinta)
                    Text(tope.mensaje).font(Tipo.lato(13)).foregroundStyle(C.pagoTinta)
                }
                .padding(10).frame(maxWidth: .infinity, alignment: .leading)
                .background(C.pagoFondo, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
            }
            HStack(alignment: .bottom, spacing: 8) {
                // El dictado es el del teclado: no hace falta pedir el micrófono.
                TextField("Escribí o dictá tu pregunta", text: $texto, axis: .vertical)
                    .lineLimit(1...4)
                    .focused($escribiendo)
                    .submitLabel(.send)
                    // Con varios renglones, la tecla «Enviar» escribe un salto: se toma como enviar.
                    .onChange(of: texto) { _, nuevo in
                        if nuevo.hasSuffix("\n") { texto = String(nuevo.dropLast()); enviar() }
                    }
                    .font(Tipo.lato(16)).foregroundStyle(C.tinta)
                    .padding(.horizontal, 14).padding(.vertical, 12)
                    .background(C.campo, in: RoundedRectangle(cornerRadius: 22, style: .continuous))
                Button(action: enviar) {
                    Image(systemName: "arrow.up").font(.system(size: 18, weight: .bold)).foregroundStyle(C.petroleoHondo)
                        .frame(width: 44, height: 44).background(C.menta, in: Circle())
                }
                .buttonStyle(.plain)
                .opacity(puedeEnviar ? 1 : 0.45)
                .disabled(!puedeEnviar)
                .accessibilityLabel("Enviar")
            }
            if info?.iaPlataforma == true, info?.puedePrender == true, info?.iaNegocio == false {
                HStack(alignment: .center, spacing: 10) {
                    Text(Formato.sinViudas("Con IA (Claude) entiende preguntas más libres. Nunca le mandamos notas ni datos de salud."))
                        .font(Tipo.lato(13)).foregroundStyle(C.tinta)
                    Spacer(minLength: 0)
                    Button(prendiendo ? "Prendiendo…" : "Prender la IA") { Task { await prenderIA() } }
                        .buttonStyle(BotonContorno(alto: 40, tam: 14)).fixedSize().disabled(prendiendo)
                }
                .padding(10).frame(maxWidth: .infinity, alignment: .leading)
                .background(C.chip, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
            }
            if let c = info?.cupo, info?.ia == true, c.cupoMensual > 0 {
                Text("IA: \(min(c.usadasMes, c.cupoMensual)) de \(c.cupoMensual) respuestas este mes")
                    .font(Tipo.lato(12)).foregroundStyle(C.sub).monospacedDigit()
            }
        }
        .padding(.horizontal, 16).padding(.top, 8).padding(.bottom, 8)
        .background(C.hoja)
    }

    // MARK: Datos

    private func cargar() async {
        info = try? await estado.api.get("/v1/panel/asistente") as EstadoAsistente
    }

    private struct Activa: Encodable { var activa: Bool }

    /// La dueña prende la IA del negocio (Anthropic es un proveedor nuevo: lo acepta ella).
    private func prenderIA() async {
        prendiendo = true
        defer { prendiendo = false }
        struct R: Decodable { var iaNegocio: Bool }
        do {
            let _: R = try await estado.api.enviar("PATCH", "/v1/panel/asistente/ia", Activa(activa: true))
            Haptico.exito()
            await cargar()
        } catch {
            mensajes.append(Mensaje(rol: "asistente", texto: (error as? ErrorApi ?? .desconocido).detalle, error: true))
        }
    }

    /// Abre el turno de una fila: se cierra la hoja y Principal lo abre (como un push).
    private func abrirTurno(_ id: String) {
        cerrar()
        // El detalle es otra hoja: se pide cuando esta terminó de bajar (con la API rápida, presentarla
        // mientras se cerraba ésta no la mostraba).
        let estado = estado
        Task { @MainActor in
            try? await Task.sleep(for: .milliseconds(450))
            estado.rutaPendiente = .turno(id)
        }
    }

    /// VoiceOver lee lo que contestó el asistente (la web lo hace con aria-live).
    private func anunciar(_ texto: String) {
        AccessibilityNotification.Announcement(texto).post()
    }

    private func enviar() {
        let q = texto.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !q.isEmpty, !pensando else { return }
        texto = ""
        Task { await preguntar(q) }
    }

    private struct Pregunta: Encodable { var pregunta: String; var historial: [TurnoChat] }

    private func preguntar(_ q: String) async {
        guard !pensando else { return }
        let pregunta = String(q.prefix(500))
        // Los últimos 8 turnos de esta hoja, sin los errores (la API recorta cada uno a 600).
        let historial = mensajes.filter { !$0.error }.suffix(8).map { TurnoChat(rol: $0.rol, texto: String($0.texto.prefix(600))) }
        mensajes.append(Mensaje(rol: "usuario", texto: pregunta))
        pensando = true
        defer { pensando = false }
        do {
            let r: RespuestaAsistente = try await estado.api.enviar("POST", "/v1/panel/asistente", Pregunta(pregunta: pregunta, historial: Array(historial)))
            if let t = r.tope { tope = t }
            // Con tope, la respuesta es el mismo aviso: va arriba del campo y no se repite en la charla.
            if r.tope?.mensaje != r.respuesta || !(r.propuestas ?? []).isEmpty {
                mensajes.append(Mensaje(rol: "asistente", texto: r.respuesta, propuestas: r.propuestas ?? [],
                                        encabezado: r.encabezado, items: r.items ?? []))
            }
            anunciar(r.respuesta)
            if r.origen == "ia" { await cargar() }
        } catch is CancellationError {
        } catch {
            let detalle = (error as? ErrorApi ?? .desconocido).detalle
            mensajes.append(Mensaje(rol: "asistente", texto: detalle, error: true))
            anunciar(detalle)
        }
    }

    private struct Cual: Encodable { var propuestaId: String }

    private func confirmar(_ p: PropuestaAsistente) async {
        resolviendo = p.id
        defer { resolviendo = nil }
        do {
            let _: RespuestaAsistente = try await estado.api.enviar("POST", "/v1/panel/asistente/confirmar", Cual(propuestaId: p.id))
            // La tarjeta con «Hecho» alcanza: antes se repetía la misma oración en una burbuja.
            resueltas[p.id] = "Hecho"
            Haptico.exito()
            anunciar("Hecho")
            // La agenda cambió: Hoy y las demás listas se vuelven a pedir.
            estado.version += 1
        } catch is CancellationError {
        } catch {
            fallo(p, error)
        }
    }

    private func descartar(_ p: PropuestaAsistente) async {
        resolviendo = p.id
        defer { resolviendo = nil }
        do {
            let _: RespuestaAsistente = try await estado.api.enviar("POST", "/v1/panel/asistente/descartar", Cual(propuestaId: p.id))
            resueltas[p.id] = "Sin cambios"
            anunciar("Sin cambios")
        } catch is CancellationError {
        } catch {
            fallo(p, error)
        }
    }

    /// El detalle del error va a la charla; si la propuesta ya no se puede usar, queda apagada.
    private func fallo(_ p: PropuestaAsistente, _ error: Error) {
        let e = error as? ErrorApi ?? .desconocido
        Haptico.error()
        if e.codigo == "propuesta_vencida" { resueltas[p.id] = "Vencida" }
        if e.codigo == "propuesta_resuelta" { resueltas[p.id] = "Ya resuelta" }
        mensajes.append(Mensaje(rol: "asistente", texto: e.detalle, error: true))
        anunciar(e.detalle)
    }
}
