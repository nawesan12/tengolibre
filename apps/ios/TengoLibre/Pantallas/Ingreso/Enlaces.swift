import SwiftUI

/// El turno de una clienta abierto desde un link (tengolibre.com.ar/turno/<código>) o un aviso: se trae,
/// se guarda en «Tus turnos» y se muestra.
struct TurnoClienteCargar: View {
    var codigo: String
    @Environment(Estado.self) private var estado
    @Environment(\.dismiss) private var cerrar
    @State private var turno: TurnoCli?
    @State private var error = ""

    var body: some View {
        Group {
            if let turno {
                TurnoClienteHoja(turno: turno) { estado.version += 1 }
            } else {
                NavigationStack {
                    VStack(spacing: 16) {
                        if error.isEmpty {
                            ProgressView()
                            Text("Trayendo tu turno…").font(Tipo.lato(15)).foregroundStyle(C.sub)
                        } else {
                            ErrorConSalida(titulo: "No encontramos ese turno", texto: error) { Task { await cargar() } }
                        }
                    }
                    .padding(24).frame(maxWidth: .infinity, maxHeight: .infinity).background(C.hoja)
                    .toolbar { ToolbarItem(placement: .cancellationAction) { Button("Cerrar") { cerrar() } } }
                }
            }
        }
        .task { await cargar() }
    }

    private func cargar() async {
        do {
            let t: TurnoCli = try await estado.api.get("/v1/t/\(codigo)")
            MisTurnos.sumar(t.codigo)
            await estado.seguirTurnos([t.codigo])
            turno = t
        } catch let e as ErrorApi { error = e.detalle } catch { self.error = "Revisá el link." }
    }
}

/// Una invitación al equipo (tengolibre.com.ar/invitacion/<token>): quién invita, a qué negocio y con
/// qué rol, y «Sumarme». Al aceptar, entra a ese negocio.
struct InvitacionVista: View {
    var token: String
    @Environment(Estado.self) private var estado
    @Environment(\.dismiss) private var cerrar
    @State private var inv: Invitacion?
    @State private var error = ""
    @State private var aceptando = false

    struct Invitacion: Decodable {
        var negocio: String
        var slug: String
        var email: String
        var rol: String
        var vencida: Bool
        var aceptada: Bool
        var invitadoPor: String?
    }

    var body: some View {
        NavigationStack {
            VStack(alignment: .leading, spacing: 16) {
                if let inv {
                    Image(systemName: "person.2.badge.plus").font(.system(size: 30, weight: .semibold)).foregroundStyle(C.petroleoHondo)
                        .frame(width: 64, height: 64).background(C.menta, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
                    Text("Te invitaron a \(inv.negocio)").font(Tipo.lato(28, .black)).tracking(-0.6)
                    Text("\(inv.invitadoPor.map { "\($0) te sumó" } ?? "Te sumaron") como \(nombreRol(inv.rol).lowercased()). Vas a ver la agenda y los avisos de ese negocio en esta app.")
                        .font(Tipo.lato(16)).foregroundStyle(C.sub)
                    if inv.vencida || inv.aceptada {
                        Label(inv.aceptada ? "Esta invitación ya se usó." : "Esta invitación venció. Pedile otra a quien te invitó.", systemImage: "exclamationmark.circle")
                            .font(Tipo.lato(15, .bold)).foregroundStyle(C.ausenteTinta)
                    } else if !inv.email.isEmpty, let mio = estado.persona?.email, inv.email.lowercased() != mio.lowercased() {
                        Text("La invitación es para \(inv.email) y entraste como \(mio). Igual podés aceptarla.").font(Tipo.lato(14)).foregroundStyle(C.sub)
                    }
                    Spacer()
                    if !inv.vencida && !inv.aceptada {
                        Button(aceptando ? "Sumándote…" : "Sumarme a \(inv.negocio)") { Task { await aceptar(inv) } }
                            .buttonStyle(BotonMenta()).disabled(aceptando)
                    }
                } else if !error.isEmpty {
                    ErrorConSalida(titulo: "No encontramos la invitación", texto: error) { Task { await cargar() } }
                    Spacer()
                } else {
                    ProgressView().frame(maxWidth: .infinity, maxHeight: .infinity)
                }
            }
            .foregroundStyle(C.tinta).padding(20).background(C.fondo)
            .toolbar { ToolbarItem(placement: .cancellationAction) { Button("Ahora no") { cerrar() } } }
            .task { await cargar() }
        }
    }

    private func cargar() async {
        do { inv = try await estado.api.get("/v1/invitaciones/\(token)"); error = "" }
        catch let e as ErrorApi { error = e.detalle } catch { self.error = "Probá de nuevo." }
    }

    private func aceptar(_ i: Invitacion) async {
        aceptando = true
        defer { aceptando = false }
        do {
            _ = try await estado.api.pedir("POST", "/v1/invitaciones/\(token)/aceptar", cuerpo: Data("{}".utf8))
            try await estado.cargarSesion()
            Haptico.exito()
            cerrar()
            if let n = estado.negocios.first(where: { $0.slug == i.slug }) {
                if estado.modo != "negocio" { await estado.elegirModo("negocio") }
                await estado.entrarANegocio(n)
            }
            estado.avisar("Ya sos parte de \(i.negocio)")
        } catch { estado.avisar(error) }
    }
}
