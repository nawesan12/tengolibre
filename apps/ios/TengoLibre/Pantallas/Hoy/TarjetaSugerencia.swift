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
/// uso / No me interesa». Mientras carga, con error o si la imagen no baja, no se dibuja nada. La misma
/// pieza que TarjetaSugerencia.svelte (web) y TarjetaSugerencia.kt; lámina en docs/laminas/sugerencia.html.
struct TarjetaSugerencia: View {
    @Environment(Estado.self) private var estado
    @AppStorage("recorridoVisto") private var recorridoVisto = false
    @State private var sugerencia: Sugerencia?
    @State private var cerrada = false
    @State private var fallo = false
    @State private var negocioCargado: String?

    private struct Respuesta: Decodable { var sugerencia: Sugerencia? }
    private struct Cierre: Encodable { var hasta: String }

    var body: some View {
        Group {
            if let s = sugerencia, !cerrada, !fallo, recorridoVisto, !estado.soloLectura,
               let destino = URL(string: s.url), let foto = URL(string: s.imagen) {
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
                        case .success(let i): i.resizable().scaledToFill()
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
                        Text(s.titulo).font(Tipo.lato(15, .bold)).foregroundStyle(C.tinta)
                            .fixedSize(horizontal: false, vertical: true)
                        HStack(spacing: 2) {
                            Text(s.cta)
                            Image(systemName: "arrow.up.right").font(.system(size: 12, weight: .bold))
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
            .accessibilityAddTraits(.isLink)

            Menu {
                Button { cerrar(s, hasta: "30d") } label: { Label("Ocultar por 30 días", systemImage: "clock") }
                Button { cerrar(s, hasta: "siempre") } label: { Label("Ya la uso / No me interesa", systemImage: "minus.circle") }
            } label: {
                Image(systemName: "xmark").font(.system(size: 14, weight: .bold)).foregroundStyle(C.sub)
                    .frame(width: 44, height: 44).contentShape(Rectangle())
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

    private func abrir(_ s: Sugerencia, _ destino: URL) {
        Haptico.suave()
        estado.webAbierta = destino
        Task { try? await estado.api.hacer("POST", "/v1/panel/sugerencia/\(s.id)/clic") }
    }

    private func cerrar(_ s: Sugerencia, hasta: String) {
        Haptico.suave()
        withAnimation(Movimiento.entrar) { cerrada = true }
        Task { try? await estado.api.hacer("POST", "/v1/panel/sugerencia/\(s.id)/cerrar", Cierre(hasta: hasta)) }
    }
}
