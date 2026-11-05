import SwiftUI
import UIKit

/// Sugerencias en el momento justo: después de algo que pasó (un ausente, el primer cobro, cargar
/// turnos a mano…) la herramienta que lo resuelve, en un aviso con su botón. Cada una sale una sola vez
/// por negocio, nunca dos seguidas, y espera a que se vaya el aviso que esté a la vista (el «Deshacer»).
@MainActor
enum Momentos {
    private static var ultimo = Date.distantPast

    private static func clave(_ estado: Estado, _ k: String) -> String { "momento-\(k)-\(estado.negocio?.id ?? "")" }
    private static func visto(_ estado: Estado, _ k: String) -> Bool { UserDefaults.standard.bool(forKey: clave(estado, k)) }

    /// Muestra el aviso cuando no hay otro a la vista (hasta 12 s), si no se mostró antes.
    private static func mostrar(_ estado: Estado, _ k: String, simbolo: String, titulo: String, texto: String, accion: String, repetible: Bool = false, hacer: @escaping @MainActor () -> Void) {
        guard repetible || !visto(estado, k), Date().timeIntervalSince(ultimo) > 60 else { return }
        Task { @MainActor in
            for _ in 0..<24 where estado.toast != nil { try? await Task.sleep(for: .milliseconds(500)) }
            guard estado.toast == nil, Date().timeIntervalSince(ultimo) > 60 else { return }
            ultimo = Date()
            if !repetible { UserDefaults.standard.set(true, forKey: clave(estado, k)) }
            estado.avisar(Toast(simbolo: simbolo, titulo: titulo, texto: texto, accion: accion)) { hacer() }
        }
    }

    /// Se marcó un turno (deslizar, detalle o aviso).
    static func cambioDeEstado(_ estado: Estado, _ nav: Navegacion, _ t: Turno, a nuevo: String) {
        guard nuevo == "ausente" else { return }
        if estado.gestiona, let mp = estado.vivo.mp, mp.disponible, !mp.conectada || (estado.vivo.cobro?.modo ?? "sin_cobro") == "sin_cobro" {
            mostrar(estado, "ausente-sena", simbolo: "creditcard", titulo: "Que no falten: pedí seña", texto: "Con una seña al reservar, casi nadie falta.", accion: "Ver") {
                nav.pestana = .hoy; nav.pilaHoy.append(DestinoMas.cobros)
            }
            return
        }
        guard let id = t.cliente.id, estado.gestiona || estado.rol == "recepcion" else { return }
        Task {
            guard let f: Ficha = try? await estado.api.get("/v1/panel/clientes/\(id)"), f.ausencias >= 2, !f.penalizado else { return }
            let nombre = f.nombre.split(separator: " ").first.map(String.init) ?? f.nombre
            mostrar(estado, "penalizar-\(id)", simbolo: "exclamationmark.shield", titulo: "\(nombre) ya faltó \(f.ausencias) veces", texto: "Podés penalizarla para que no reserve por la web un tiempo.", accion: "Ver") {
                nav.pestana = .clientes; nav.pilaClientes.append(DestinoClientes.ficha(id))
            }
        }
    }

    /// El primer cobro desde la app.
    static func cobro(_ estado: Estado, _ nav: Navegacion) {
        mostrar(estado, "primer-cobro", simbolo: "banknote", titulo: "Tu caja, al día", texto: "Mirá cuánto entró hoy y por qué medio.", accion: "Ver la caja") {
            nav.pestana = .hoy; nav.pilaHoy.append(DestinoMas.caja)
        }
    }

    /// Cada turno cargado a mano: al tercero, que reserven solos.
    static func cargaManual(_ estado: Estado, _ nav: Navegacion) {
        let k = clave(estado, "manuales")
        let n = UserDefaults.standard.integer(forKey: k) + 1
        UserDefaults.standard.set(n, forKey: k)
        guard n >= 3 else { return }
        mostrar(estado, "reserven-solos", simbolo: "qrcode", titulo: "Que te reserven solos", texto: "Pasales tu link y se cargan los turnos sin vos.", accion: "Compartir") {
            nav.compartir = true
        }
    }

    /// Una reseña de 5 estrellas respondida: compartirla.
    static func resenaBuena(_ estado: Estado, _ r: Resena) {
        guard r.puntaje >= 5, !r.comentario.isEmpty else { return }
        mostrar(estado, "resena-\(r.id)", simbolo: "star.fill", titulo: "¡Qué linda reseña!", texto: "Compartila en tus historias.", accion: "Compartir", repetible: true) {
            let texto = "«\(r.comentario)» — \(r.nombre) ⭐️⭐️⭐️⭐️⭐️\nReservá en \(Web.textoPagina(estado.negocio?.slug ?? ""))"
            let a = UIActivityViewController(activityItems: [texto], applicationActivities: nil)
            var arriba = Ventana.principal()?.rootViewController
            while let p = arriba?.presentedViewController { arriba = p }
            arriba?.present(a, animated: true)
        }
    }
}

/// En la ficha: una clienta fiel sin membresía → ofrecérsela (o crear el primer plan).
struct SugerenciaMembresia: View {
    var ficha: Ficha
    var conMembresia: Bool
    @Environment(Estado.self) private var estado
    @Environment(Navegacion.self) private var nav
    @State private var vender = false
    @State private var planes: [PlanMembresia] = []

    var body: some View {
        if ficha.visitas >= 5 && !conMembresia && (estado.gestiona || estado.rol == "recepcion") {
            Button {
                if estado.vivo.planes == 0 { nav.pilaClientes.append(DestinoMas.membresias) } else { vender = true }
            } label: {
                HStack(spacing: 12) {
                    Image(systemName: "ticket").font(.system(size: 20, weight: .semibold)).foregroundStyle(C.petroleoHondo)
                        .frame(width: 40, height: 40).background(C.menta, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
                    VStack(alignment: .leading, spacing: 2) {
                        Text("Viene seguido: ofrecele una membresía").font(Tipo.lato(15, .bold))
                        Text(estado.vivo.planes == 0 ? "Creá tu primer plan con visitas incluidas" : "\(ficha.visitas) visitas · plata fija todos los meses").font(Tipo.lato(13)).foregroundStyle(C.sub)
                    }
                    Spacer(minLength: 0)
                    Image(systemName: "chevron.right").font(.system(size: 13, weight: .semibold)).foregroundStyle(C.sub)
                }
                .foregroundStyle(C.tinta).padding(14).tarjeta(18)
            }
            .buttonStyle(.plain)
            .sheet(isPresented: $vender) {
                NuevaSocia(planes: planes, nombreInicial: ficha.nombre, telefonoInicial: ficha.telefono, emailInicial: ficha.email) { estado.version += 1 }
            }
            .task {
                struct P: Decodable { var planes: [PlanMembresia] }
                planes = ((try? await estado.api.get("/v1/panel/membresias/planes") as P)?.planes ?? []).filter(\.activo)
            }
        }
    }
}
