import Observation
import SwiftUI
import UserNotifications
import WidgetKit

/// Cómo está el negocio por dentro: qué está conectado, prendido o cargado. Lo usan Más (el estado de
/// cada sección), Hoy («Descubrí» y la barra de avance) y «Potenciá tu negocio».
@Observable
@MainActor
final class EstadoNegocio {
    struct Paso: Identifiable, Hashable {
        var id: String
        var simbolo: String
        var titulo: String
        var detalle: String
        var listo: Bool
        var destino: DestinoMas?
        var basico: Bool
    }

    var cargado = false
    var mp: ConexionMP?
    var cobro: ConfigCobro?
    var iaActiva: Bool?
    var iaReservasMes = 0
    var clavesIA = 0
    var planes = 0
    var socias = 0
    var cupones = 0
    var recordatorios = 0
    var fotos = 0
    var logo = false
    var dominio: String?
    var dominioListo = false
    var arranque: Arranque?
    var whatsapp: Bool?
    var widget = false
    var avisos = false
    var equipo = 0
    var servicios = 0
    private var negocio = ""

    private struct IA: Decodable { var activo: Bool; var reservasMes: Int }
    private struct Mcp: Decodable { struct T: Decodable { var id: String }; var tokens: [T] }
    private struct Planes: Decodable { var planes: [PlanMembresia] }
    private struct Socios: Decodable { var socios: [Socio] }
    private struct Cups: Decodable { var cupones: [Cupon] }
    private struct Recs: Decodable { struct R: Decodable { var activo: Bool }; var recordatorios: [R] }
    private struct Fotos: Decodable { var fotos: [FotoGaleria] }
    private struct Dom: Decodable { struct D: Decodable { var nombre: String; var estado: String }; var dominio: D? }
    private struct WA: Decodable { var disponible: Bool; var confirmacion: Bool }
    private struct Pros: Decodable { var profesionales: [Profesional] }
    private struct Servs: Decodable { var servicios: [Servicio] }

    /// Trae todo (de a un pedido por vez: nada de `async let`, ver el crash de Tu Local Web).
    func cargar(_ estado: Estado) async {
        guard let n = estado.negocio?.id else { return }
        if n != negocio { cargado = false; negocio = n }
        let api = estado.api
        let g = estado.gestiona, mostrador = g || estado.rol == "recepcion"
        arranque = try? await api.get("/v1/panel/arranque")
        if g {
            mp = try? await api.get("/v1/panel/mercadopago")
            cobro = try? await api.get("/v1/panel/cobros")
            if let i: IA = try? await api.get("/v1/panel/ia/publico") { iaActiva = i.activo; iaReservasMes = i.reservasMes }
            clavesIA = (try? await api.get("/v1/panel/mcp") as Mcp)?.tokens.count ?? clavesIA
            fotos = (try? await api.get("/v1/panel/mi-web/galeria") as Fotos)?.fotos.count ?? fotos
            if let ng: NegocioDatos = try? await api.get("/v1/panel/negocio") { logo = !(ng.logo ?? "").isEmpty }
            if let d: Dom = try? await api.get("/v1/panel/dominio") { dominio = d.dominio?.nombre; dominioListo = d.dominio?.estado == "verificado" }
            if let w: WA = try? await api.get("/v1/panel/ajustes/whatsapp") { whatsapp = w.disponible ? w.confirmacion : nil }
            equipo = (try? await api.get("/v1/panel/profesionales") as Pros)?.profesionales.filter(\.activo).count ?? equipo
            servicios = (try? await api.get("/v1/panel/servicios") as Servs)?.servicios.filter(\.activo).count ?? servicios
        }
        if mostrador {
            planes = (try? await api.get("/v1/panel/membresias/planes") as Planes)?.planes.filter(\.activo).count ?? planes
            socias = (try? await api.get("/v1/panel/membresias/socios", query: [URLQueryItem(name: "estado", value: "activa")]) as Socios)?.socios.count ?? socias
            cupones = (try? await api.get("/v1/panel/cupones") as Cups)?.cupones.filter { $0.activo && $0.referidoId == nil }.count ?? cupones
            recordatorios = (try? await api.get("/v1/panel/recordatorios") as Recs)?.recordatorios.filter(\.activo).count ?? recordatorios
        }
        let a = await UNUserNotificationCenter.current().notificationSettings()
        avisos = a.authorizationStatus == .authorized || a.authorizationStatus == .provisional
        widget = await Self.hayWidget()
        cargado = true
    }

    private static func hayWidget() async -> Bool {
        await withCheckedContinuation { c in
            WidgetCenter.shared.getCurrentConfigurations { r in
                if case .success(let l) = r { c.resume(returning: !l.isEmpty) } else { c.resume(returning: false) }
            }
        }
    }

    // MARK: Lo que se ve en Más

    var textoCobros: (String, Bool) {
        guard let mp else { return ("", false) }
        if !mp.conectada { return ("Sin conectar · cobrá señas al reservar", true) }
        if mp.reautorizar { return ("Hay que volver a autorizar", true) }
        switch cobro?.modo {
        case "porcentaje": return ("Conectado · seña del \(cobro?.senaPorcentaje ?? 0) %", false)
        case "fija": return ("Conectado · seña de \(Formato.pesos(cobro?.senaFija ?? 0))", false)
        case "total": return ("Conectado · cobra el total al reservar", false)
        default: return ("Conectado · todavía no pide seña", true)
        }
    }

    var textoIA: (String, Bool) {
        guard let iaActiva else { return ("Calendarios, Claude, ChatGPT", false) }
        if !iaActiva && clavesIA == 0 { return ("Apagada · que te reserven desde ChatGPT", true) }
        var p: [String] = []
        if iaActiva { p.append(iaReservasMes > 0 ? "\(iaReservasMes) reservas por IA este mes" : "Reservas por IA prendidas") }
        if clavesIA > 0 { p.append(Formato.cantidad(clavesIA, "asistente conectado", "asistentes conectados")) }
        return (p.joined(separator: " · "), false)
    }

    var textoMembresias: (String, Bool) {
        cargado ? (planes == 0 ? ("Sin planes · plata fija todos los meses", true) : ("\(Formato.cantidad(socias, "socia activa", "socias activas")) · \(Formato.cantidad(planes, "plan", "planes"))", false)) : ("Planes y socias", false)
    }

    var textoCupones: (String, Bool) {
        cargado ? (cupones == 0 ? ("Ninguno activo", false) : (Formato.cantidad(cupones, "activo", "activos"), false)) : ("", false)
    }

    var textoRecordatorios: (String, Bool) {
        cargado ? (recordatorios == 0 ? ("Apagados · que vuelvan solas", true) : (Formato.cantidad(recordatorios, "prendido", "prendidos"), false)) : ("Cumpleaños y «te extrañamos»", false)
    }

    var textoWeb: (String, Bool) {
        guard cargado else { return ("Datos, fotos, logo y portada", false) }
        if !logo { return ("Falta el logo", true) }
        if fotos == 0 { return ("Sin fotos · sumá algunas", true) }
        return ("\(Formato.cantidad(fotos, "foto", "fotos")) · logo cargado", false)
    }

    var textoDominio: (String, Bool) {
        guard let dominio else { return (cargado ? "Usá tu propia dirección" : "", false) }
        return (dominioListo ? "\(dominio) · funcionando" : "\(dominio) · falta configurar", !dominioListo)
    }

    var textoWhatsApp: (String, Bool) {
        guard let whatsapp else { return ("Confirmaciones y recordatorios", false) }
        return whatsapp ? ("Pide confirmación el día antes", false) : ("Apagado · que confirmen solas", true)
    }

    // MARK: Potenciá tu negocio

    func pasos(_ estado: Estado) -> [Paso] {
        var out: [Paso] = []
        for p in arranque?.pasos ?? [] {
            out.append(Paso(id: "a-\(p.clave)", simbolo: p.listo ? "checkmark" : "circle.dashed", titulo: p.titulo,
                            detalle: p.obligatorio ? "Hace falta para que te reserven" : "Recomendado", listo: p.listo, destino: destinoArranque(p.clave), basico: true))
        }
        if estado.gestiona {
            out.append(Paso(id: "mp", simbolo: "creditcard", titulo: "Cobrá señas con Mercado Pago", detalle: "Menos ausencias, plata directo a tu cuenta",
                            listo: mp?.conectada == true && (cobro?.modo ?? "sin_cobro") != "sin_cobro", destino: .cobros, basico: false))
            out.append(Paso(id: "ia", simbolo: "sparkles", titulo: "Que te reserven desde ChatGPT y Claude", detalle: "Tus horarios a mano de los asistentes",
                            listo: iaActiva == true, destino: .integraciones, basico: false))
            out.append(Paso(id: "mcp", simbolo: "bubble.left.and.text.bubble.right", titulo: "Manejá tu agenda hablando", detalle: "Conectá Claude o ChatGPT a tu panel",
                            listo: clavesIA > 0, destino: .integraciones, basico: false))
            out.append(Paso(id: "fotos", simbolo: "photo.on.rectangle", titulo: "Logo y fotos en tu página", detalle: "Las páginas con fotos reciben más reservas",
                            listo: logo && fotos > 0, destino: .miWeb, basico: false))
            if whatsapp != nil {
                out.append(Paso(id: "wa", simbolo: "message", titulo: "Que confirmen por WhatsApp", detalle: "El día antes, solas",
                                listo: whatsapp == true, destino: .whatsapp, basico: false))
            }
        }
        if estado.gestiona || estado.rol == "recepcion" {
            out.append(Paso(id: "memb", simbolo: "ticket", titulo: "Vendé membresías", detalle: "Plata fija todos los meses", listo: planes > 0, destino: .membresias, basico: false))
            out.append(Paso(id: "rec", simbolo: "envelope.badge", titulo: "Recordatorios automáticos", detalle: "Cumpleaños y «te extrañamos»", listo: recordatorios > 0, destino: .avisosClientes, basico: false))
        }
        out.append(Paso(id: "avisos", simbolo: "bell.badge", titulo: "Avisos en este iPhone", detalle: "Cada reserva y cancelación al toque", listo: avisos, destino: .cuenta, basico: false))
        out.append(Paso(id: "widget", simbolo: "apps.iphone", titulo: "El widget en tu pantalla de inicio", detalle: "Tu próximo turno sin abrir la app", listo: widget, destino: .recorrido, basico: false))
        return out
    }

    func avance(_ estado: Estado) -> Double {
        let p = pasos(estado)
        guard !p.isEmpty else { return 1 }
        return Double(p.filter(\.listo).count) / Double(p.count)
    }

    private func destinoArranque(_ clave: String) -> DestinoMas? {
        switch clave {
        case "servicios", "servicio": .servicios
        case "profesionales", "profesional", "horario", "horarios", "equipo": .equipo
        case "web", "mi_web", "miweb", "logo", "fotos", "datos", "direccion": .miWeb
        case "cobros", "mercadopago", "pagos", "sena": .cobros
        case "suscripcion": .suscripcion
        case "plantilla": .plantillas
        default: nil
        }
    }
}

/// Más › «Potenciá tu negocio»: el anillo de avance, lo básico y lo que hace crecer, cada uno con su botón.
struct Potencia: View {
    @Environment(Estado.self) private var estado

    var body: some View {
        let vivo = estado.vivo
        let pasos = vivo.pasos(estado)
        let avance = vivo.avance(estado)
        PantallaMas(titulo: "Potenciá tu negocio", subtitulo: "\(pasos.filter(\.listo).count) de \(pasos.count) listos",
                    cargando: !vivo.cargado, recargar: { await vivo.cargar(estado) }) {
            HStack(spacing: 18) {
                ZStack {
                    Circle().stroke(C.chip, lineWidth: 10)
                    Circle().trim(from: 0, to: avance).stroke(C.menta, style: StrokeStyle(lineWidth: 10, lineCap: .round)).rotationEffect(.degrees(-90))
                    Text("\(Int((avance * 100).rounded())) %").font(Tipo.lato(20, .black)).monospacedDigit()
                }
                .frame(width: 88, height: 88)
                .animation(Movimiento.entrar, value: avance)
                VStack(alignment: .leading, spacing: 4) {
                    Text(avance >= 1 ? "Estás usando todo" : "Sacale todo el jugo").font(Tipo.lato(19, .black))
                    Text(avance >= 1 ? "Tu negocio tiene todo prendido. ¡Bien ahí!" : "Cada paso te trae más reservas o más plata.").font(Tipo.lato(14)).foregroundStyle(C.sub)
                }
            }
            .foregroundStyle(C.tinta).padding(16).frame(maxWidth: .infinity, alignment: .leading).tarjeta()
            grupo("Lo básico", pasos.filter(\.basico))
            grupo("Para crecer", pasos.filter { !$0.basico })
        }
        .task(id: estado.version) { await vivo.cargar(estado) }
    }

    @ViewBuilder private func grupo(_ t: String, _ l: [EstadoNegocio.Paso]) -> some View {
        if !l.isEmpty {
            Bloque(titulo: t) {
                ForEach(l) { p in
                    if let d = p.destino, !p.listo {
                        NavigationLink(value: d) { fila(p) }.buttonStyle(.plain)
                    } else {
                        fila(p)
                    }
                }
            }
        }
    }

    private func fila(_ p: EstadoNegocio.Paso) -> some View {
        HStack(spacing: 12) {
            Image(systemName: p.listo ? "checkmark" : p.simbolo).font(.system(size: 16, weight: .bold))
                .foregroundStyle(p.listo ? C.petroleoHondo : C.mentaTexto)
                .frame(width: 34, height: 34)
                .background(p.listo ? C.menta : C.mentaSuave, in: Circle())
            VStack(alignment: .leading, spacing: 2) {
                Text(p.titulo).font(Tipo.lato(15, .bold)).strikethrough(p.listo, color: C.sub)
                Text(p.listo ? "Listo" : p.detalle).font(Tipo.lato(13)).foregroundStyle(C.sub)
            }
            Spacer(minLength: 0)
            if !p.listo && p.destino != nil { Text("Hacerlo").font(Tipo.lato(13, .bold)).foregroundStyle(C.mentaTexto) }
        }
        .foregroundStyle(C.tinta).padding(.horizontal, 16).padding(.vertical, 10).frame(minHeight: 60)
        .opacity(p.listo ? 0.7 : 1)
        .contentShape(Rectangle())
    }
}

/// En Hoy, mientras falte algo: «Tu negocio al 60 % · Seguir».
struct BarraAvance: View {
    @Environment(Estado.self) private var estado
    @Environment(Navegacion.self) private var nav

    var body: some View {
        let vivo = estado.vivo
        let avance = vivo.avance(estado)
        if vivo.cargado && avance < 1 && estado.gestiona {
            let siguiente = vivo.pasos(estado).first { !$0.listo }
            Button { nav.pilaHoy.append(DestinoMas.primerosPasos) } label: {
                VStack(alignment: .leading, spacing: 8) {
                    HStack {
                        Text("Tu negocio al \(Int((avance * 100).rounded())) %").font(Tipo.lato(15, .black))
                        Spacer()
                        Text("Seguir").font(Tipo.lato(14, .bold)).foregroundStyle(C.mentaTexto)
                        Image(systemName: "chevron.right").font(.system(size: 12, weight: .bold)).foregroundStyle(C.mentaTexto)
                    }
                    GeometryReader { g in
                        ZStack(alignment: .leading) {
                            Capsule().fill(C.chip)
                            Capsule().fill(C.menta).frame(width: g.size.width * avance)
                        }
                    }
                    .frame(height: 8)
                    if let s = siguiente { Text("Lo próximo: \(s.titulo)").font(Tipo.lato(13)).foregroundStyle(C.sub).lineLimit(1) }
                }
                .foregroundStyle(C.tinta).padding(14).tarjeta(18)
            }
            .buttonStyle(.plain)
            .accessibilityLabel("Tu negocio al \(Int((avance * 100).rounded())) por ciento. Seguir")
        }
    }
}
