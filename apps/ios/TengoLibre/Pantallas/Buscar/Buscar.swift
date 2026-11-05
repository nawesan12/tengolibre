import SwiftUI
import TipKit

/// «¿Qué querés hacer?»: encuentra secciones y acciones de la app por lo que la gente dice («plata»,
/// «seña», «IA», «vacaciones»…) y, con la API (`GET /v1/panel/buscar`), clientas, turnos y servicios.
struct Buscar: View {
    @Environment(Estado.self) private var estado
    @Environment(Navegacion.self) private var nav
    @Environment(\.dismiss) private var cerrar
    @State private var q = ""
    @State private var r: Encontrado?
    @State private var buscando = true

    struct Encontrado: Decodable {
        struct Cli: Decodable, Identifiable { var id: String; var nombre: String; var telefono: String }
        struct Tur: Decodable, Identifiable { var id: String; var inicio: Date; var estado: String; var servicio: String; var profesional: String; var cliente: String }
        struct Ser: Decodable, Identifiable { var id: String; var nombre: String; var precio: Int; var duracion: Int }
        var clientes: [Cli]?
        var turnos: [Tur]?
        var servicios: [Ser]?
    }

    /// Lo que se puede hacer, con las palabras con que la gente lo busca.
    struct Cosa: Identifiable {
        var simbolo: String
        var titulo: String
        var palabras: String
        var ir: Ir
        var id: String { titulo }
        enum Ir { case seccion(DestinoMas), pestana(Pestana), nuevo, compartir, avisos }
    }

    private var cosas: [Cosa] {
        let g = estado.gestiona, mostrador = g || estado.rol == "recepcion"
        var l: [Cosa] = [
            Cosa(simbolo: "plus.circle", titulo: "Cargar un turno", palabras: "nuevo turno reservar agendar anotar cargar sobreturno fijo", ir: .nuevo),
            Cosa(simbolo: "calendar", titulo: "Agenda", palabras: "semana dia calendario agenda horarios", ir: .pestana(.agenda)),
            Cosa(simbolo: "square.and.arrow.up", titulo: "Compartir mi link", palabras: "link qr instagram whatsapp compartir pagina reservas historia", ir: .compartir),
            Cosa(simbolo: "bell", titulo: "Avisos", palabras: "notificaciones campanita avisos novedades", ir: .avisos),
            Cosa(simbolo: "play.circle", titulo: "Qué podés hacer", palabras: "ayuda tutorial recorrido widget como se usa", ir: .seccion(.recorrido)),
            Cosa(simbolo: "person.crop.circle", titulo: "Tu cuenta", palabras: "cuenta salir cerrar sesion negocio cambiar modo", ir: .seccion(.cuenta))
        ]
        if estado.esProfesional {
            l += [Cosa(simbolo: "clock", titulo: "Mi horario", palabras: "horario dias libres vacaciones bloquear franco licencia", ir: .seccion(.miHorario)),
                  Cosa(simbolo: "star.bubble", titulo: "Reseñas", palabras: "reseñas opiniones estrellas comentarios", ir: .pestana(.resenas))]
            return l
        }
        l += [Cosa(simbolo: "banknote", titulo: "Caja", palabras: "plata dinero caja cobrar cobrado efectivo transferencia debe ingresos hoy", ir: .seccion(.caja)),
              Cosa(simbolo: "star.bubble", titulo: "Reseñas", palabras: "reseñas opiniones estrellas comentarios responder", ir: .seccion(.resenas)),
              Cosa(simbolo: "person.2.badge.gearshape", titulo: "Equipo y horarios", palabras: "equipo profesionales horario empleados invitar bloquear feriado", ir: .seccion(.equipo))]
        if mostrador {
            l += [Cosa(simbolo: "hourglass", titulo: "Lista de espera", palabras: "espera lista avisar lugar hueco liberado", ir: .pestana(.espera)),
                  Cosa(simbolo: "person.2", titulo: "Clientes", palabras: "clientas clientes contactos fichas", ir: .pestana(.clientes)),
                  Cosa(simbolo: "ticket", titulo: "Membresías", palabras: "membresias abono socias plan mensual suscripcion clientes", ir: .seccion(.membresias)),
                  Cosa(simbolo: "tag", titulo: "Cupones", palabras: "cupon descuento promo promocion oferta codigo", ir: .seccion(.cupones)),
                  Cosa(simbolo: "tag.circle", titulo: "Etiquetas", palabras: "etiquetas vip categorias clientas", ir: .seccion(.etiquetas)),
                  Cosa(simbolo: "exclamationmark.shield", titulo: "Penalidades y ausencias", palabras: "ausentes faltas penalidad no vino bloquear clienta", ir: .seccion(.penalidades)),
                  Cosa(simbolo: "envelope.badge", titulo: "Recordatorios a clientes", palabras: "cumpleaños extrañamos mails recordatorios volver", ir: .seccion(.avisosClientes))]
        }
        if g {
            l += [Cosa(simbolo: "sparkles", titulo: "Servicios", palabras: "servicios precios duracion productos categorias carta", ir: .seccion(.servicios)),
                  Cosa(simbolo: "creditcard", titulo: "Pagos y señas", palabras: "seña señas mercado pago mp cobrar adelantado devolver pagos", ir: .seccion(.cobros)),
                  Cosa(simbolo: "chart.bar", titulo: "Reportes", palabras: "reportes estadisticas ingresos ocupacion horas muertas numeros ventas", ir: .seccion(.reportes)),
                  Cosa(simbolo: "beach.umbrella", titulo: "Días libres del equipo", palabras: "vacaciones dias libres pedidos aprobar licencia", ir: .seccion(.diasLibres)),
                  Cosa(simbolo: "gift", titulo: "Referidos", palabras: "referidos recomendar amigas premio", ir: .seccion(.referidos)),
                  Cosa(simbolo: "globe", titulo: "Mi web", palabras: "web pagina logo fotos portada colores direccion redes instagram", ir: .seccion(.miWeb)),
                  Cosa(simbolo: "paintpalette", titulo: "Plantillas", palabras: "plantilla diseño estilo como se ve pagina", ir: .seccion(.plantillas)),
                  Cosa(simbolo: "link", titulo: "Dominio propio", palabras: "dominio .com direccion propia url", ir: .seccion(.dominio)),
                  Cosa(simbolo: "point.3.connected.trianglepath.dotted", titulo: "Integraciones e IA", palabras: "ia inteligencia artificial chatgpt claude gemini asistente calendario google mcp", ir: .seccion(.integraciones)),
                  Cosa(simbolo: "gearshape", titulo: "Configuración", palabras: "configuracion ajustes reservas anticipacion cancelar politica intervalo", ir: .seccion(.configuracion)),
                  Cosa(simbolo: "message", titulo: "WhatsApp", palabras: "whatsapp confirmar confirmacion mensajes recordatorio", ir: .seccion(.whatsapp)),
                  Cosa(simbolo: "list.bullet.rectangle", titulo: "Actividad", palabras: "auditoria actividad quien hizo historial cambios", ir: .seccion(.auditoria)),
                  Cosa(simbolo: "chart.line.uptrend.xyaxis", titulo: "Potenciá tu negocio", palabras: "crecer mas reservas primeros pasos avance", ir: .seccion(.primerosPasos))]
            if estado.esDuena {
                l += [Cosa(simbolo: "crown", titulo: "Suscripción", palabras: "suscripcion plan pagar tengo libre prueba", ir: .seccion(.suscripcion)),
                      Cosa(simbolo: "building.2", titulo: "Sucursales", palabras: "sucursales locales grupo otro local", ir: .seccion(.sucursales))]
            }
        }
        return l
    }

    private static func normal(_ s: String) -> String { s.folding(options: [.diacriticInsensitive, .caseInsensitive], locale: Formato.argentina) }

    private var encontradas: [Cosa] {
        let t = Self.normal(q).split(separator: " ").map(String.init).filter { $0.count >= 2 }
        guard !t.isEmpty else { return [] }
        return cosas.filter { c in
            let base = Self.normal(c.titulo + " " + c.palabras)
            return t.allSatisfy { p in base.split(separator: " ").contains { $0.hasPrefix(p) } }
        }
    }

    var body: some View {
        NavigationStack {
            List {
                if q.trimmingCharacters(in: .whitespaces).count < 2 {
                    Section("Lo más buscado") {
                        ForEach(cosas.prefix(estado.esProfesional ? 4 : 8)) { c in fila(c) }
                    }
                } else {
                    if !encontradas.isEmpty { Section("En la app") { ForEach(encontradas) { c in fila(c) } } }
                    if let cl = r?.clientes, !cl.isEmpty, !estado.esProfesional {
                        Section("Clientas") {
                            ForEach(cl) { c in
                                Button { ir { nav.pestana = .clientes; nav.pilaClientes.append(DestinoClientes.ficha(c.id)) } } label: {
                                    Label { VStack(alignment: .leading) { Text(c.nombre); Text(c.telefono).font(Tipo.lato(13)).foregroundStyle(C.sub) } } icon: { Iniciales(nombre: c.nombre, tam: 30) }
                                }
                            }
                        }
                    }
                    if let ts = r?.turnos, !ts.isEmpty {
                        Section("Turnos") {
                            ForEach(ts) { t in
                                Button {
                                    Task { if let x: Turno = try? await estado.api.get("/v1/panel/turnos/\(t.id)") { ir { nav.turno = x } } }
                                } label: {
                                    Label { VStack(alignment: .leading) { Text("\(t.cliente) · \(t.servicio)"); Text(Formato.diaYHora(t.inicio)).font(Tipo.lato(13)).foregroundStyle(C.sub) } } icon: { Image(systemName: "calendar") }
                                }
                            }
                        }
                    }
                    if let ss = r?.servicios, !ss.isEmpty, estado.gestiona {
                        Section("Servicios") {
                            ForEach(ss) { s in
                                Button { ir { nav.pestana = .mas; nav.pilaMas.append(DestinoMas.servicios) } } label: {
                                    Label("\(s.nombre) · \(Formato.pesos(s.precio))", systemImage: "sparkles")
                                }
                            }
                        }
                    }
                    if encontradas.isEmpty && (r?.clientes ?? []).isEmpty && (r?.turnos ?? []).isEmpty && (r?.servicios ?? []).isEmpty && r != nil {
                        Text("No encontramos «\(q)». Probá con otra palabra.").foregroundStyle(C.sub)
                    }
                }
            }
            .font(Tipo.lato(16)).tint(C.tinta)
            .searchable(text: $q, isPresented: $buscando, placement: .navigationBarDrawer(displayMode: .always), prompt: "¿Qué querés hacer?")
            .navigationTitle("Buscar").navigationBarTitleDisplayMode(.inline)
            .toolbar { ToolbarItem(placement: .cancellationAction) { Button("Cerrar") { cerrar() } } }
            .task(id: q) {
                let t = q.trimmingCharacters(in: .whitespaces)
                guard t.count >= 2 else { r = nil; return }
                try? await Task.sleep(for: .milliseconds(250))
                guard !Task.isCancelled else { return }
                r = try? await estado.api.get("/v1/panel/buscar", query: [URLQueryItem(name: "q", value: t)])
            }
        }
    }

    private func fila(_ c: Cosa) -> some View {
        Button { abrir(c) } label: { Label(c.titulo, systemImage: c.simbolo) }
    }

    private func abrir(_ c: Cosa) {
        switch c.ir {
        case .seccion(let d): ir { nav.pestana = .hoy; nav.pilaHoy.append(d) }
        case .pestana(let p): ir { nav.pestana = p }
        case .nuevo: ir { nav.nuevo = NuevoTurnoInicio() }
        case .compartir: ir { nav.compartir = true }
        case .avisos: ir { nav.avisos = true }
        }
    }

    /// Cierra el buscador y después navega (dos hojas no se abren a la vez).
    private func ir(_ hacer: @escaping @MainActor () -> Void) {
        cerrar()
        Task { @MainActor in
            try? await Task.sleep(for: .milliseconds(350))
            hacer()
        }
    }
}

/// El acceso al buscador: un campo de mentira que abre «¿Qué querés hacer?».
struct CampoBuscar: View {
    @Environment(Navegacion.self) private var nav
    var body: some View {
        Button { nav.buscar = true } label: {
            HStack(spacing: 10) {
                Image(systemName: "magnifyingglass").font(.system(size: 17, weight: .semibold))
                Text("¿Qué querés hacer?").font(Tipo.lato(16))
                Spacer()
            }
            .foregroundStyle(.white.opacity(0.75)).padding(.horizontal, 14).frame(height: 44)
            .background(.white.opacity(0.12), in: RoundedRectangle(cornerRadius: 14, style: .continuous))
        }
        .buttonStyle(.plain)
        .popoverTip(ConsejoBuscar())
        .accessibilityLabel("Buscar: ¿qué querés hacer?")
    }
}
