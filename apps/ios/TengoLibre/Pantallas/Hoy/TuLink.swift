import SwiftUI
import TipKit

/// El link de la página (tengolibre.com.ar/<slug>): es lo que trae las reservas, así que va arriba de
/// Hoy mientras el negocio tiene pocos turnos, con cuatro acciones grandes (Copiar, WhatsApp,
/// Instagram, QR) y «Cambiar link». La misma regla y los mismos textos que la web (TuLink.svelte).
enum LinkRegla {
    /// Con menos turnos que esto en los últimos 30 días, el link va primero y en grande.
    static let pocosTurnos = 20

    /// Cuántos turnos hubo en los últimos 30 días (hasta `pocosTurnos`: no hace falta contar más).
    static func movimiento(_ estado: Estado, hoy: Date) async -> Int? {
        guard let desde = Formato.calendario.date(byAdding: .day, value: -29, to: hoy) else { return nil }
        let q = [URLQueryItem(name: "desde", value: Formato.diaISO(desde)), URLQueryItem(name: "hasta", value: Formato.diaISO(hoy)),
                 URLQueryItem(name: "estado", value: "pendiente_pago,pendiente,confirmado,atendido,ausente"),
                 URLQueryItem(name: "limite", value: String(pocosTurnos))]
        struct Turnos: Decodable { var turnos: [Turno] }
        let t: Turnos? = try? await estado.api.get("/v1/panel/turnos", query: q)
        return t?.turnos.count
    }

    /// Mientras se escribe: minúsculas, sin acentos, espacios a guiones (el resto lo valida la API).
    static func limpiar(_ s: String) -> String {
        let base = s.folding(options: [.diacriticInsensitive, .caseInsensitive], locale: Formato.argentina).lowercased()
        var out = ""
        for c in base {
            if c.isASCII && (c.isLetter || c.isNumber) || c == "-" { out.append(c) }
            else if c == " " || c == "_" || c == "." { out.append("-") }
        }
        return String(out.prefix(48))
    }

    /// El formato que acepta la API (negocios.ValidarSlug).
    static func valido(_ s: String) -> Bool {
        s.range(of: "^[a-z0-9]([a-z0-9-]{0,46}[a-z0-9])?$", options: .regularExpression) != nil && !s.contains("--")
    }
}

/// El consejo de la primera vez: señala el link.
struct ConsejoLink: Tip {
    var title: Text { Text("Este es tu link") }
    var message: Text? { Text("Compartilo por WhatsApp o Instagram y te reservan solos.") }
    var image: Image? { Image(systemName: "link") }
}

struct TarjetaLink: View {
    @Environment(Estado.self) private var estado
    @Environment(Navegacion.self) private var nav
    var compacta = false
    @State private var copiado = false
    @State private var cambiar = false

    private var slug: String { estado.negocio?.slug ?? "" }
    private var url: URL { Web.pagina(slug) }
    private var host: String { "\(Web.base.host() ?? "tengolibre.com.ar")/" }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Label("TU LINK", systemImage: "link").font(Tipo.rotulo).tracking(12 * 0.08).foregroundStyle(C.mentaTexto)
            VStack(alignment: .leading, spacing: 0) {
                Text(host).font(Tipo.lato(compacta ? 13 : 15, .bold)).foregroundStyle(C.sub)
                Text(slug).font(Tipo.lato(compacta ? 22 : 28, .black)).tracking(-0.6).foregroundStyle(C.mentaTexto)
                    .lineLimit(2).minimumScaleFactor(0.6)
            }
            .popoverTip(ConsejoLink())
            if !compacta { Text("Compartilo y te reservan solos.").font(Tipo.lato(15)).foregroundStyle(C.sub) }
            HStack(spacing: 8) {
                accion(copiado ? "Copiado" : "Copiar", copiado ? "checkmark" : "doc.on.doc", principal: true) { copiar() }
                accion("WhatsApp", "message.fill") { whatsapp() }
                accion("Instagram", "camera.fill") { instagram() }
                accion("QR", "qrcode") { nav.compartir = true }
            }
            HStack(spacing: 18) {
                if estado.gestiona {
                    Button { cambiar = true } label: { Label("Cambiar link", systemImage: "pencil") }
                }
                Link(destination: url) { Label("Ver mi página", systemImage: "arrow.up.right") }
            }
            .font(Tipo.lato(14, .bold)).foregroundStyle(C.mentaTexto).frame(minHeight: 40)
        }
        .padding(compacta ? 16 : 18)
        .frame(maxWidth: .infinity, alignment: .leading)
        .tarjeta(Radio.bloque)
        .overlay(RoundedRectangle(cornerRadius: Radio.bloque, style: .continuous).strokeBorder(compacta ? C.linea : C.menta, lineWidth: compacta ? 1 : 2))
        .sheet(isPresented: $cambiar) {
            NavigationStack { CambiarLink() }
                .presentationDetents([.large])
                .presentationCornerRadius(Radio.hoja)
        }
    }

    /// Cuatro acciones del mismo tamaño: ícono arriba, palabra abajo, 64 pt para el pulgar.
    private func accion(_ t: String, _ s: String, principal: Bool = false, _ a: @escaping () -> Void) -> some View {
        Button(action: a) {
            VStack(spacing: 4) {
                Image(systemName: s).font(.system(size: 20, weight: .semibold))
                    .foregroundStyle(principal ? C.petroleoHondo : C.mentaTexto)
                Text(t).font(Tipo.lato(13, .bold)).lineLimit(1).minimumScaleFactor(0.8)
            }
            .foregroundStyle(principal ? C.petroleoHondo : C.tinta)
            .frame(maxWidth: .infinity, minHeight: 64)
            .background(principal ? C.menta : C.chip, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
        }
        .buttonStyle(.plain)
    }

    private func copiar() {
        UIPasteboard.general.url = url
        Haptico.exito()
        withAnimation { copiado = true }
        Task { try? await Task.sleep(for: .seconds(2)); withAnimation { copiado = false } }
    }

    private func whatsapp() {
        let t = "Reservá tu turno en \(estado.negocio?.nombre ?? ""): \(url.absoluteString)"
        let q = t.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? ""
        if let u = URL(string: "whatsapp://send?text=\(q)"), UIApplication.shared.canOpenURL(u) { UIApplication.shared.open(u) }
        else if let u = URL(string: "https://wa.me/?text=\(q)") { UIApplication.shared.open(u) }
    }

    /// Instagram: el link copiado para pegarlo en la bio, y se abre la edición del perfil.
    private func instagram() {
        copiar()
        estado.avisar("Copiamos tu link", texto: "Pegalo en tu bio de Instagram.")
        if let u = URL(string: "https://www.instagram.com/accounts/edit/") { UIApplication.shared.open(u) }
    }
}

/// Cambiar el link: vista previa, si está libre mientras se escribe y el aviso de que el viejo sigue
/// andando (la API lo guarda y la web redirige al nuevo). El mismo flujo que la web (EditorLink.svelte).
struct CambiarLink: View {
    @Environment(Estado.self) private var estado
    @Environment(\.dismiss) private var cerrar
    @State private var nuevo = ""
    @State private var disponible: String?
    @State private var link: LinkDelNegocio?
    @State private var guardando = false
    @State private var error = ""

    struct LinkDelNegocio: Decodable { var slug: String; var anteriores: [Anterior]; var restantes: Int }
    struct Anterior: Decodable { var slug: String }
    private struct Disponibilidad: Decodable { var estado: String }

    private var actual: String { estado.negocio?.slug ?? "" }
    private var host: String { "\(Web.base.host() ?? "tengolibre.com.ar")/" }

    /// Ícono, color y texto del estado. Nunca sólo color.
    private var campo: (simbolo: String, color: Color, texto: String, ok: Bool) {
        if nuevo == actual { return ("checkmark.circle.fill", C.mentaTexto, "Es tu link de ahora.", false) }
        if nuevo.count < 3 { return ("info.circle.fill", C.pagoTinta, "Tiene que tener al menos 3 letras", false) }
        if !LinkRegla.valido(nuevo) || disponible == "invalida" { return ("xmark.circle.fill", C.error, "Esa dirección no se puede usar.", false) }
        switch disponible {
        case "libre": return ("checkmark.circle.fill", C.mentaTexto, "Está libre.", true)
        case "tomada": return ("xmark.circle.fill", C.error, "Ya la usa otro negocio.", false)
        case "propia": return ("checkmark.circle.fill", C.mentaTexto, "Es tu link de ahora.", false)
        default: return ("ellipsis", C.sub, "Fijándonos si está libre…", false)
        }
    }
    private var sinCambios: Bool { link?.restantes == 0 }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 14) {
                VStack(alignment: .leading, spacing: 0) {
                    Text(host).font(Tipo.lato(15, .bold)).foregroundStyle(C.sub)
                    Text(nuevo.isEmpty ? "…" : nuevo).font(Tipo.lato(28, .black)).tracking(-0.6).foregroundStyle(C.mentaTexto)
                        .lineLimit(2).minimumScaleFactor(0.6)
                }
                HStack(spacing: 8) {
                    TextField("tu-negocio", text: Binding(get: { nuevo }, set: { nuevo = LinkRegla.limpiar($0) }))
                        .keyboardType(.URL).textInputAutocapitalization(.never).autocorrectionDisabled()
                        .font(Tipo.lato(17, .bold)).foregroundStyle(C.tinta)
                        .accessibilityLabel("Tu link nuevo")
                    Image(systemName: campo.simbolo).foregroundStyle(campo.color).font(.system(size: 22))
                }
                .padding(.horizontal, 14).frame(minHeight: 56)
                .background(C.campo, in: RoundedRectangle(cornerRadius: 16, style: .continuous))
                .contorno(16, color: campo.color, ancho: 2)
                Text(campo.texto).font(Tipo.lato(13, .bold)).foregroundStyle(campo.color)

                HStack(alignment: .top, spacing: 10) {
                    Image(systemName: "checkmark.seal.fill").foregroundStyle(C.mentaTexto).font(.system(size: 20))
                    (Text("Tu link de antes sigue andando. ").bold() + Text("Los QR y carteles impresos llevan al\u{00a0}nuevo."))
                        .font(Tipo.lato(14)).foregroundStyle(C.tinta)
                }
                .padding(14).frame(maxWidth: .infinity, alignment: .leading)
                .background(C.mentaSuave, in: RoundedRectangle(cornerRadius: 18, style: .continuous))

                if !error.isEmpty { Text(error).font(Tipo.lato(14, .bold)).foregroundStyle(C.error) }
                Button(guardando ? "Guardando…" : "Guardar link") { Task { await guardar() } }
                    .buttonStyle(BotonMenta()).disabled(!campo.ok || guardando || sinCambios)
                if let link {
                    Text(sinCambios ? "Ya lo cambiaste 5 veces este\u{00a0}mes." : link.restantes == 1 ? "Te queda 1 cambio este\u{00a0}mes." : "Te quedan \(link.restantes) cambios este\u{00a0}mes.")
                        .font(Tipo.lato(13)).foregroundStyle(C.sub).frame(maxWidth: .infinity)
                    if !link.anteriores.isEmpty {
                        Rotulo("SIGUEN LLEVANDO A TU PÁGINA")
                        ForEach(link.anteriores, id: \.slug) { a in
                            Label("\(host)\(a.slug)", systemImage: "link").font(Tipo.lato(14)).foregroundStyle(C.sub)
                        }
                    }
                }
            }
            .padding(16)
        }
        .background(C.hoja)
        .navigationTitle("Cambiar tu link").navigationBarTitleDisplayMode(.inline)
        .toolbar { ToolbarItem(placement: .cancellationAction) { Button("Cerrar") { cerrar() } } }
        .task {
            nuevo = actual
            link = try? await estado.api.get("/v1/panel/negocio/link")
        }
        .task(id: nuevo) {
            // Con 300 ms de pausa: no se pregunta letra por letra.
            disponible = nil
            guard nuevo != actual, nuevo.count >= 3, LinkRegla.valido(nuevo) else { return }
            try? await Task.sleep(for: .milliseconds(300))
            if Task.isCancelled { return }
            let d: Disponibilidad? = try? await estado.api.get("/v1/panel/negocio/link/\(nuevo)")
            if !Task.isCancelled { disponible = d?.estado }
        }
    }

    private func guardar() async {
        guardando = true
        defer { guardando = false }
        error = ""
        do {
            let l: LinkDelNegocio = try await estado.api.enviar("PATCH", "/v1/panel/negocio/link", ["slug": nuevo])
            link = l
            // El negocio trae el slug: se vuelve a pedir para que todo use el nuevo.
            try? await estado.cargarSesion()
            Haptico.exito()
            estado.avisar("Listo. Tu link nuevo ya anda.")
            cerrar()
        } catch let e as ErrorApi {
            error = e.detalle
        } catch {
            self.error = "No se pudo cambiar. Probá de nuevo."
        }
    }
}
