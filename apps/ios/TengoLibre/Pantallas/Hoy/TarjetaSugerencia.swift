import SwiftUI

/// Una sugerencia de la casa (GET /v1/panel/sugerencia): otra app de Grupo Vimo, con los textos del
/// catálogo de la API. Todo opcional menos lo que se muestra: si falta algo o llega null, no se dibuja.
struct Sugerencia: Decodable, Equatable {
    var id: String
    var marca: String
    var sello: String
    var titulo: String
    var cta: String
    var url: String
    var imagen: String
    var banner: String?
    var color: String?
}

/// Al final de Hoy, sólo para la dueña o la encargada: la foto de la otra app, el sello «De Grupo Vimo»,
/// el título y el link (se abre en el navegador de la app). La cruz abre «Ocultar por 30 días» o «Ya la
/// uso / No me interesa», con un aviso que permite deshacerlo. Mientras carga, con error o si la imagen no baja, no se dibuja nada. La misma
/// pieza que TarjetaSugerencia.svelte (web) y TarjetaSugerencia.kt; lámina en docs/laminas/sugerencia.html.
struct TarjetaSugerencia: View {
    @Environment(Estado.self) private var estado
    @AppStorage("recorridoVisto") private var recorridoVisto = false
    @State private var sugerencia: Sugerencia?
    @State private var cerrada = false
    @State private var fallo = false
    @State private var negocioCargado: String?
    /// La vista se cuenta una vez por tarjeta, cuando la foto quedó dibujada (la API además cuenta una
    /// por negocio y por día: pedirla con cada cambio de estado.version no infla nada).
    @State private var vistaDe: String?
    // Los íconos crecen con el texto (Dynamic Type), como en Android con sp.
    @ScaledMetric(relativeTo: .body) private var tamCruz: CGFloat = 14
    @ScaledMetric(relativeTo: .body) private var tamFlecha: CGFloat = 12

    private struct Respuesta: Decodable { var sugerencia: Sugerencia? }
    private struct Cierre: Encodable { var hasta: String }

    var body: some View {
        // Un VStack y no un Group: los modificadores de un Group van a cada hijo, y sin sugerencia no hay
        // ninguno, así que el .task no corría nunca y la tarjeta no llegaba a pedirse.
        VStack(spacing: 0) {
            // El mismo filtro que Android: link https y foto.
            if let s = sugerencia, !cerrada, !fallo, recorridoVisto, !estado.soloLectura, s.url.hasPrefix("https://"),
               !s.imagen.isEmpty, let destino = URL(string: s.url), let foto = URL(string: s.imagen) {
                tarjeta(s, destino: destino, foto: foto)
            }
        }
        .task(id: "\(estado.version)-\(estado.negocio?.id ?? "")") { await cargar() }
    }

    private func tarjeta(_ s: Sugerencia, destino: URL, foto: URL) -> some View {
        ZStack(alignment: .topTrailing) {
            Button { abrir(s, destino) } label: {
                HStack(spacing: 12) {
                    AsyncImage(url: foto) { fase in
                        switch fase {
                        case .success(let i): i.resizable().scaledToFill().onAppear { contarVista(s) }
                        case .failure: Color.clear.onAppear { fallo = true }
                        default: Color(hexTexto: s.color ?? "") ?? C.chip
                        }
                    }
                    .frame(width: 72, height: 72, alignment: .bottom)
                    .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
                    .accessibilityHidden(true)
                    VStack(alignment: .leading, spacing: 3) {
                        Text(s.sello).font(Tipo.lato(11, .black, .caption)).tracking(11 * 0.08).textCase(.uppercase)
                            .foregroundStyle(C.sub).padding(.trailing, 36)
                        Text(Formato.sinViudas(s.titulo)).font(Tipo.lato(15, .bold)).foregroundStyle(C.tinta)
                            .fixedSize(horizontal: false, vertical: true)
                        HStack(spacing: 2) {
                            Text(Formato.sinViudas(s.cta))
                            Image(systemName: "arrow.up.right").font(.system(size: tamFlecha, weight: .bold))
                        }
                        .font(Tipo.lato(14, .bold)).foregroundStyle(C.mentaTexto)
                    }
                    Spacer(minLength: 0)
                }
                .padding(12)
                .frame(maxWidth: .infinity, alignment: .leading)
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .accessibilityElement(children: .ignore)
            .accessibilityLabel("\(s.sello). \(s.titulo). \(s.cta)")
            // Abre el navegador: es un enlace, no un botón (Android lo lee con «abrir en el navegador»).
            .accessibilityRemoveTraits(.isButton)
            .accessibilityAddTraits(.isLink)
            .accessibilityHint("Se abre en el navegador.")

            Menu {
                Button { cerrar(s, hasta: "30d") } label: { Label("Ocultar por 30 días", systemImage: "clock") }
                Button { cerrar(s, hasta: "siempre") } label: { Label("Ya la uso / No me interesa", systemImage: "minus.circle") }
            } label: {
                Image(systemName: "xmark").font(.system(size: tamCruz, weight: .bold)).foregroundStyle(C.sub)
                    .frame(minWidth: 44, minHeight: 44).contentShape(Rectangle())
            }
            .accessibilityLabel("Ocultar sugerencia")
        }
        .background(C.tarjeta, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: 18, style: .continuous).strokeBorder(C.linea, lineWidth: 1))
        .transition(.opacity)
    }

    /// Sin async let (en Release se cae, ver ios-crash-async-let): un pedido y listo.
    private func cargar() async {
        let negocio = estado.negocio?.id
        if negocio != negocioCargado { cerrada = false; fallo = false; negocioCargado = negocio }
        guard estado.gestiona, negocio != nil else { sugerencia = nil; return }
        do {
            let r: Respuesta = try await estado.api.get("/v1/panel/sugerencia")
            sugerencia = r.sugerencia
        } catch is CancellationError {
            // La reemplazó otra carga.
        } catch {
            sugerencia = nil
        }
    }

    private func contarVista(_ s: Sugerencia) {
        guard vistaDe != s.id else { return }
        vistaDe = s.id
        Task { try? await estado.api.hacer("POST", "/v1/panel/sugerencia/\(s.id)/vista") }
    }

    private func abrir(_ s: Sugerencia, _ destino: URL) {
        Haptico.suave()
        estado.webAbierta = destino
        Task { try? await estado.api.hacer("POST", "/v1/panel/sugerencia/\(s.id)/clic") }
    }

    /// Se va en el acto y sale el aviso con «Deshacer» (que la trae de vuelta), anunciado a VoiceOver.
    private func cerrar(_ s: Sugerencia, hasta: String) {
        Haptico.suave()
        withAnimation(Movimiento.entrar) { cerrada = true }
        let titulo = hasta == "30d" ? "La ocultamos por 30 días" : "Listo, no la vas a ver más"
        Task {
            try? await estado.api.hacer("POST", "/v1/panel/sugerencia/\(s.id)/cerrar", Cierre(hasta: hasta))
            estado.avisar(Toast(simbolo: "checkmark", titulo: titulo, accion: "Deshacer")) {
                do {
                    try await estado.api.hacer("DELETE", "/v1/panel/sugerencia/\(s.id)/cerrar")
                    withAnimation(Movimiento.entrar) { cerrada = false }
                } catch {
                    estado.avisar(error)
                }
            }
            AccessibilityNotification.Announcement(titulo).post()
        }
    }
}
