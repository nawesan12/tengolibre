import CoreImage.CIFilterBuiltins
import UserNotifications
import SwiftUI

/// 17 Avisos (la campanita): lo de los últimos siete días agrupado HOY · AYER · antes, con su ícono,
/// «SIN LEER» en las nuevas y el atajo a lo que corresponde. Al abrir se marcan como vistas.
struct Avisos: View {
    @Environment(Estado.self) private var estado
    @Environment(Navegacion.self) private var nav
    @Environment(\.dismiss) private var cerrar
    @State private var lista: [Notificacion] = []
    @State private var cargando = true

    private var grupos: [(String, [Notificacion])] {
        var orden: [String] = []
        var por: [String: [Notificacion]] = [:]
        for n in lista {
            let s = Formato.seccion(n.creadoEn).uppercased()
            if por[s] == nil { orden.append(s) }
            por[s, default: []].append(n)
        }
        return orden.map { ($0, por[$0]!) }
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 12) {
                    if cargando && lista.isEmpty {
                        ForEach(0..<5, id: \.self) { _ in Esqueleto(alto: 72) }
                    } else if lista.isEmpty {
                        VacioQueEnsena(rayas: "Todo tranquilo", titulo: "No hay avisos", texto: "Acá vas a ver las reservas, cancelaciones, reseñas y todo lo que pase en tu negocio.") { EmptyView() }
                    }
                    ForEach(grupos, id: \.0) { g in
                        Text(g.0).rotulo().padding(.top, 6)
                        VStack(spacing: 0) {
                            ForEach(Array(g.1.enumerated()), id: \.element.id) { i, n in
                                if i > 0 { Rectangle().fill(C.linea).frame(height: 1).padding(.leading, 68) }
                                fila(n)
                            }
                        }
                        .tarjeta()
                    }
                }
                .padding(16)
            }
            .background(C.fondo)
            .navigationTitle("Avisos").navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) { Button("Listo") { cerrar() } }
                ToolbarItem(placement: .topBarLeading) {
                    NavigationLink { PreferenciasPush() } label: { Image(systemName: "gearshape") }.accessibilityLabel("Qué avisos te llegan")
                }
            }
            .refreshable { await cargar() }
            .task { await cargar() }
        }
    }

    private func fila(_ n: Notificacion) -> some View {
        Button { abrir(n) } label: {
            HStack(alignment: .top, spacing: 12) {
                Image(systemName: n.simbolo).font(.system(size: 18, weight: .semibold))
                    .foregroundStyle(n.nueva ? C.petroleoHondo : C.tinta)
                    .frame(width: 40, height: 40)
                    .background(n.nueva ? C.menta : C.chip, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
                VStack(alignment: .leading, spacing: 2) {
                    HStack(spacing: 6) {
                        Text(n.titulo).font(Tipo.lato(15, .bold)).lineLimit(2)
                        Spacer(minLength: 0)
                        Text(Formato.horaDe(n.creadoEn)).font(Tipo.lato(12)).foregroundStyle(C.sub).monospacedDigit()
                    }
                    if !n.texto.isEmpty { Text(n.texto).font(Tipo.lato(14)).foregroundStyle(C.sub).lineLimit(3) }
                    if n.nueva {
                        HStack(spacing: 4) { Circle().fill(C.menta).frame(width: 6, height: 6); Text("SIN LEER").font(Tipo.lato(10, .black)).tracking(0.8) }
                            .foregroundStyle(C.mentaTexto).padding(.top, 2)
                    }
                }
            }
            .foregroundStyle(C.tinta).padding(14).contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }

    private func abrir(_ n: Notificacion) {
        if let id = n.turnoId {
            Task {
                if let t: Turno = try? await estado.api.get("/v1/panel/turnos/\(id)") {
                    cerrar()
                    try? await Task.sleep(for: .milliseconds(350))
                    nav.turno = t
                }
            }
            return
        }
        cerrar()
        switch n.tipo {
        case "resena": if estado.esProfesional { nav.pestana = .resenas } else { nav.pestana = .mas; nav.pilaMas.append(DestinoMas.resenas) }
        case "espera": nav.pestana = estado.esProfesional ? .hoy : .espera
        case "dias_libres", "equipo": nav.pestana = .mas; nav.pilaMas.append(DestinoMas.diasLibres)
        default: break
        }
    }

    private struct Campanita: Decodable { var notificaciones: [Notificacion]; var sinLeer: Int }

    private func cargar() async {
        do {
            let r: Campanita = try await estado.api.get("/v1/panel/campanita")
            lista = r.notificaciones
            if r.sinLeer > 0 {
                _ = try? await estado.api.pedir("POST", "/v1/panel/campanita/vista")
                estado.sinLeer = 0
                try? await UNUserNotificationCenter.current().setBadgeCount(0)
            }
        } catch { estado.avisar(error) }
        cargando = false
    }
}

/// 18 Compartir: el QR sobre crema con el logo, el link con «Copiar» y las cuatro formas de mandarlo.
struct Compartir: View {
    @Environment(Estado.self) private var estado
    @Environment(\.dismiss) private var cerrar
    @Environment(\.displayScale) private var escala
    @State private var copiado = false

    private var slug: String { estado.negocio?.slug ?? "" }
    private var url: URL { Web.pagina(slug) }
    private var texto: String { Web.textoPagina(slug) }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 16) {
                    tarjetaQR
                    HStack(spacing: 8) {
                        Text(url.absoluteString.replacingOccurrences(of: "https://", with: "")).font(Tipo.lato(16, .bold)).lineLimit(1).minimumScaleFactor(0.7)
                        Spacer(minLength: 0)
                        Button(copiado ? "Copiado" : "Copiar") {
                            UIPasteboard.general.url = url
                            Haptico.exito()
                            withAnimation { copiado = true }
                        }
                        .buttonStyle(BotonMentaChico())
                    }
                    .foregroundStyle(C.tinta).padding(.leading, 16).padding(.trailing, 6).frame(height: 56)
                    .background(C.chip, in: Capsule())
                    LazyVGrid(columns: [GridItem(.flexible(), spacing: 10), GridItem(.flexible(), spacing: 10)], spacing: 10) {
                        tile("Historia de Instagram", "camera") { historia() }
                        tile("Estado de WhatsApp", "message") {
                            let t = texto.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? ""
                            if let u = URL(string: "whatsapp://send?text=\(t)") { UIApplication.shared.open(u) }
                        }
                        tile("Guardar el QR", "square.and.arrow.down") {
                            if let i = imagenQR() { UIImageWriteToSavedPhotosAlbum(i, nil, nil, nil); estado.avisar("Guardamos el QR en tus fotos") }
                        }
                        ShareLink(item: url, message: Text(texto)) {
                            tileContenido("Más opciones", "square.and.arrow.up")
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding(16)
            }
            .background(C.hoja)
            .navigationTitle("Tu link").navigationBarTitleDisplayMode(.inline)
            .toolbar { ToolbarItem(placement: .confirmationAction) { Button("Listo") { cerrar() } } }
        }
    }

    private var tarjetaQR: some View {
        VStack(spacing: 14) {
            Image("Logotipo").resizable().scaledToFit().frame(height: 26)
            if let i = imagenQR() {
                Image(uiImage: i).interpolation(.none).resizable().scaledToFit().frame(width: 200, height: 200)
                    .accessibilityLabel("Código QR de tu página")
            }
            Text(estado.negocio?.nombre ?? "").font(Tipo.lato(20, .black)).foregroundStyle(C.petroleo)
            Text("Escaneá y reservá tu turno").font(Tipo.lato(14)).foregroundStyle(Color(hex: 0x4A5D61))
        }
        .padding(24).frame(maxWidth: .infinity)
        .background(C.crema, in: RoundedRectangle(cornerRadius: 24, style: .continuous))
        .environment(\.colorScheme, .light)
    }

    private func tile(_ t: String, _ s: String, _ a: @escaping () -> Void) -> some View {
        Button(action: a) { tileContenido(t, s) }.buttonStyle(.plain)
    }

    private func tileContenido(_ t: String, _ s: String) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Image(systemName: s).font(.system(size: 22, weight: .semibold)).foregroundStyle(C.mentaTexto)
            Text(t).font(Tipo.lato(15, .bold)).multilineTextAlignment(.leading)
        }
        .foregroundStyle(C.tinta).padding(14).frame(maxWidth: .infinity, minHeight: 96, alignment: .topLeading)
        .background(C.fondo, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
        .contorno(18)
    }

    func imagenQR() -> UIImage? {
        let f = CIFilter.qrCodeGenerator()
        f.message = Data(url.absoluteString.utf8)
        f.correctionLevel = "M"
        guard let ci = f.outputImage?.transformed(by: CGAffineTransform(scaleX: 12, y: 12)) else { return nil }
        let color = CIFilter.falseColor()
        color.inputImage = ci
        color.color0 = CIColor(red: 0x18 / 255, green: 0x3C / 255, blue: 0x44 / 255)
        color.color1 = CIColor(red: 0xF4 / 255, green: 0xEE / 255, blue: 0xE2 / 255)
        guard let out = color.outputImage, let cg = CIContext().createCGImage(out, from: out.extent) else { return nil }
        return UIImage(cgImage: cg)
    }

    /// La historia de Instagram: una imagen vertical con el QR (si Instagram no está, la hoja de compartir).
    private func historia() {
        let r = ImageRenderer(content: tarjetaQR.frame(width: 360).padding(40).background(C.petroleo))
        r.scale = escala
        guard let img = r.uiImage, let png = img.pngData() else { return }
        if let u = URL(string: "instagram-stories://share?source_application=ar.com.tengolibre.app"), UIApplication.shared.canOpenURL(u) {
            UIPasteboard.general.setItems([["com.instagram.sharedSticker.backgroundImage": png]], options: [.expirationDate: Date().addingTimeInterval(300)])
            UIApplication.shared.open(u)
        } else {
            let a = UIActivityViewController(activityItems: [img, url], applicationActivities: nil)
            Ventana.principal()?.rootViewController?.presentedViewController?.present(a, animated: true)
        }
    }
}
