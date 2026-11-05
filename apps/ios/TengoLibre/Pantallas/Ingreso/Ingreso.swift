import SwiftUI
import UserNotifications

/// 00 Arranque: el símbolo sobre petróleo mientras se lee la sesión.
struct PantallaArranque: View {
    var body: some View {
        ZStack {
            C.petroleo.ignoresSafeArea()
            Image("SimboloBlanco").resizable().scaledToFit().frame(width: 72, height: 72)
        }
        .accessibilityLabel("Tengo Libre")
    }
}

/// 01 Ingreso: petróleo con el logotipo, «Tu agenda, en el bolsillo.» y la hoja con Apple y Google.
struct Ingreso: View {
    @Environment(Estado.self) private var estado
    @State private var entrando: String?
    @State private var error = ""

    var body: some View {
        ZStack(alignment: .bottom) {
            C.petroleo.ignoresSafeArea()
            Image(uiImage: Grano.imagen).resizable(resizingMode: .tile).opacity(0.5).ignoresSafeArea()
            VStack(alignment: .leading, spacing: 18) {
                Image("LogotipoNegativo").resizable().scaledToFit().frame(height: 30).padding(.top, 24)
                Spacer()
                Text("Tu agenda,\nen el bolsillo.").font(Tipo.lato(44, .black)).tracking(-1.5).foregroundStyle(.white).lineSpacing(-4)
                    .minimumScaleFactor(0.7)
                Text("Mirá el día, cargá turnos y enterate de cada reserva al toque.").font(Tipo.lato(17)).foregroundStyle(.white.opacity(0.8))
                Spacer().frame(height: 300)
            }
            .padding(.horizontal, 24)
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)

            VStack(spacing: 12) {
                Button { Task { await entrar("apple") } } label: {
                    HStack(spacing: 10) {
                        if entrando == "apple" { ProgressView().tint(.white) } else { Image(systemName: "apple.logo").font(.system(size: 20, weight: .semibold)) }
                        Text("Continuar con Apple").font(Tipo.lato(17, .bold))
                    }
                    .foregroundStyle(.white).frame(maxWidth: .infinity, minHeight: 56)
                    .background(.black, in: RoundedRectangle(cornerRadius: 16, style: .continuous))
                }
                .buttonStyle(.plain)
                Button { Task { await entrar("google") } } label: {
                    HStack(spacing: 10) {
                        if entrando == "google" { ProgressView() } else { Image("LogoGoogle").resizable().scaledToFit().frame(width: 20, height: 20) }
                        Text("Continuar con Google").font(Tipo.lato(17, .bold))
                    }
                    .foregroundStyle(Color(hex: 0x0F2328)).frame(maxWidth: .infinity, minHeight: 56)
                    .background(.white, in: RoundedRectangle(cornerRadius: 16, style: .continuous))
                    .overlay(RoundedRectangle(cornerRadius: 16, style: .continuous).strokeBorder(Color(hex: 0xE2D6BF), lineWidth: 1.5))
                }
                .buttonStyle(.plain)
                if !error.isEmpty {
                    Text(error).font(Tipo.lato(14, .bold)).foregroundStyle(C.error).multilineTextAlignment(.center)
                }
                Text("Sin contraseñas. Usá la misma cuenta que en la web.").font(Tipo.lato(13)).foregroundStyle(C.sub).multilineTextAlignment(.center)
            }
            .disabled(entrando != nil)
            .padding(.horizontal, 20).padding(.top, 24).padding(.bottom, 12)
            .background(C.tarjeta, in: UnevenRoundedRectangle(topLeadingRadius: 28, topTrailingRadius: 28, style: .continuous))
            .ignoresSafeArea(edges: .bottom)
        }
    }

    private func entrar(_ con: String) async {
        entrando = con
        error = ""
        defer { entrando = nil }
        do {
            if con == "apple" { try await estado.ingresarConApple() } else { try await estado.ingresarConGoogle() }
            Haptico.exito()
        } catch let e as Clerk.ErrorClerk where e.codigo == "cancelado" {
            return
        } catch let e as ErrorApi {
            error = e.detalle
            Haptico.error()
        } catch {
            self.error = (error as? LocalizedError)?.errorDescription ?? "No pudimos entrar. Probá de nuevo."
            Haptico.error()
        }
    }
}

/// 02 Modo: «¿Cómo vas a usar Tengo Libre?» — atiendo un negocio / reservo turnos.
struct ElegirModo: View {
    @Environment(Estado.self) private var estado

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text("¿Cómo vas a usar Tengo Libre?").font(Tipo.lato(30, .black)).tracking(-0.8).foregroundStyle(C.tinta).padding(.top, 40)
            Text("Lo podés cambiar cuando quieras desde tu cuenta.").font(Tipo.lato(16)).foregroundStyle(C.sub)
            Button { Task { await estado.elegirModo("negocio") } } label: {
                VStack(alignment: .leading, spacing: 10) {
                    Image(systemName: "storefront").font(.system(size: 28, weight: .semibold)).foregroundStyle(C.menta)
                    Text("Atiendo un negocio").font(Tipo.lato(22, .black))
                    Text("Tu agenda, tu equipo y tus clientas. Cargás turnos y te enterás de cada reserva.").font(Tipo.lato(15)).foregroundStyle(.white.opacity(0.8)).multilineTextAlignment(.leading)
                }
                .foregroundStyle(.white).padding(20).frame(maxWidth: .infinity, alignment: .leading)
                .background(C.bloquePetroleo, in: RoundedRectangle(cornerRadius: 24, style: .continuous))
            }
            .buttonStyle(.plain)
            Button { Task { await estado.elegirModo("cliente") } } label: {
                VStack(alignment: .leading, spacing: 10) {
                    Image(systemName: "calendar.badge.checkmark").font(.system(size: 28, weight: .semibold)).foregroundStyle(C.mentaTexto)
                    Text("Reservo turnos").font(Tipo.lato(22, .black))
                    Text("Tus turnos en un solo lugar, recordatorios y lugares cerca tuyo con horario libre.").font(Tipo.lato(15)).foregroundStyle(C.sub).multilineTextAlignment(.leading)
                }
                .foregroundStyle(C.tinta).padding(20).frame(maxWidth: .infinity, alignment: .leading)
                .tarjeta(24)
            }
            .buttonStyle(.plain)
            Spacer()
        }
        .padding(.horizontal, 20)
        .background(C.fondo.ignoresSafeArea())
    }
}

/// 03 Elegir negocio: «¿Dónde trabajás hoy?».
struct ElegirNegocio: View {
    @Environment(Estado.self) private var estado
    @State private var elegido: NegocioDeLaPersona?

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("¿Dónde trabajás hoy?").font(Tipo.lato(30, .black)).tracking(-0.8).foregroundStyle(C.tinta).padding(.top, 40)
            Text("Tenés más de un lugar. Después cambiás desde arriba de Hoy.").font(Tipo.lato(16)).foregroundStyle(C.sub).padding(.bottom, 8)
            ScrollView {
                VStack(spacing: 10) {
                    ForEach(estado.negocios) { n in
                        let on = elegido?.id == n.id
                        Button { elegido = n; Haptico.seleccion() } label: {
                            HStack(spacing: 12) {
                                Iniciales(nombre: n.nombre, tam: 44, fondo: on ? C.menta : C.chip, tinta: on ? C.petroleoHondo : C.tinta)
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(n.nombre).font(Tipo.lato(17, .bold))
                                    Text(n.rolTexto).font(Tipo.lato(13)).foregroundStyle(C.sub)
                                }
                                Spacer()
                                Image(systemName: on ? "checkmark.circle.fill" : "circle").font(.system(size: 22)).foregroundStyle(on ? C.mentaTexto : C.linea)
                            }
                            .foregroundStyle(C.tinta).padding(.horizontal, 16).frame(minHeight: 72)
                            .tarjeta(20)
                            .overlay { if on { RoundedRectangle(cornerRadius: 20, style: .continuous).strokeBorder(C.menta, lineWidth: 2) } }
                        }
                        .buttonStyle(.plain)
                    }
                }
            }
            Button { if let elegido { Task { await estado.entrarANegocio(elegido) } } } label: {
                Text(elegido.map { "Entrar a \($0.nombre)" } ?? "Elegí un lugar").lineLimit(1)
            }
            .buttonStyle(BotonMenta()).disabled(elegido == nil)
            .padding(.bottom, 8)
        }
        .padding(.horizontal, 20)
        .background(C.fondo.ignoresSafeArea())
    }
}

/// 04 Permiso de avisos: dos avisos de ejemplo, «Enterate al toque de cada turno.» y los dos botones.
struct PermisoAvisos: View {
    @Environment(Estado.self) private var estado

    var body: some View {
        VStack(spacing: 0) {
            ZStack {
                C.petroleo
                Image(uiImage: Grano.imagen).resizable(resizingMode: .tile).opacity(0.5)
                VStack(spacing: 10) {
                    aviso("Nueva reserva", "Sofía · Corte y brushing · mañana 10:30", "ahora")
                    aviso("Se liberó un horario", "Hoy 16:00 con Carla · 3 en la lista de espera", "hace 2 min").opacity(0.85).scaleEffect(0.96)
                }
                .padding(.horizontal, 20)
            }
            .frame(maxHeight: .infinity)
            .ignoresSafeArea(edges: .top)
            VStack(alignment: .leading, spacing: 14) {
                Text("Enterate al toque de cada turno.").font(Tipo.lato(30, .black)).tracking(-0.8).foregroundStyle(C.tinta)
                Text("Reservas, cancelaciones y huecos que se liberan. Desde el aviso confirmás o le avisás a la lista sin abrir la app.")
                    .font(Tipo.lato(16)).foregroundStyle(C.sub)
                Button("Activar avisos") {
                    Task {
                        _ = await estado.pedirPermisoPush()
                        await estado.terminarPermisos()
                    }
                }
                .buttonStyle(BotonMenta())
                Button("Ahora no") { Task { await estado.terminarPermisos() } }.buttonStyle(BotonTexto(color: C.tinta)).frame(maxWidth: .infinity).frame(height: 44)
            }
            .padding(20)
            .background(C.fondo)
        }
        .background(C.fondo.ignoresSafeArea())
    }

    private func aviso(_ t: String, _ x: String, _ h: String) -> some View {
        HStack(alignment: .top, spacing: 10) {
            Image("Simbolo").resizable().scaledToFit().frame(width: 36, height: 36).clipShape(RoundedRectangle(cornerRadius: 9))
            VStack(alignment: .leading, spacing: 2) {
                HStack { Text("TENGO LIBRE").font(Tipo.lato(11, .bold)).foregroundStyle(.secondary); Spacer(); Text(h).font(Tipo.lato(11)).foregroundStyle(.secondary) }
                Text(t).font(Tipo.lato(15, .bold))
                Text(x).font(Tipo.lato(14))
            }
        }
        .foregroundStyle(Color(hex: 0x0F2328)).padding(12)
        .background(.white.opacity(0.92), in: RoundedRectangle(cornerRadius: 20, style: .continuous))
        .accessibilityHidden(true)
    }
}

/// Alta: quien entra sin negocio abre el suyo (nombre y link), o pasa a reservar.
struct Alta: View {
    @Environment(Estado.self) private var estado
    @State private var nombre = ""
    @State private var slug = ""
    @State private var slugTocado = false
    @State private var creando = false
    @State private var error = ""

    private func aSlug(_ s: String) -> String {
        let base = s.folding(options: [.diacriticInsensitive, .caseInsensitive], locale: Formato.argentina).lowercased()
        var out = "", guion = false
        for c in base {
            if c.isLetter && c.isASCII || c.isNumber { out.append(c); guion = false }
            else if !guion && !out.isEmpty { out.append("-"); guion = true }
        }
        return String(out.trimmingCharacters(in: CharacterSet(charactersIn: "-")).prefix(40))
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                Text("Abrí tu negocio en Tengo Libre").font(Tipo.lato(30, .black)).tracking(-0.8).padding(.top, 40)
                Text("Gratis los primeros 30 días. Después lo terminás de armar desde Más: servicios, equipo y horarios.").font(Tipo.lato(16)).foregroundStyle(C.sub)
                Bloque {
                    RenglonCampo(titulo: "Nombre del negocio", texto: $nombre, placeholder: "Lumi Estética")
                    RenglonCampo(titulo: "Tu link", texto: Binding(get: { slug }, set: { slug = aSlug($0); slugTocado = true }), placeholder: "lumi", teclado: .URL)
                }
                Text("tengolibre.com.ar/\(slug.isEmpty ? "tu-negocio" : slug)").font(Tipo.lato(15, .bold)).foregroundStyle(C.mentaTexto)
                if !error.isEmpty { Text(error).font(Tipo.lato(14, .bold)).foregroundStyle(C.error) }
                Button(creando ? "Creando…" : "Crear mi negocio") { Task { await crear() } }
                    .buttonStyle(BotonMenta()).disabled(creando || nombre.count < 2 || slug.count < 3)
                Button("En realidad, quiero reservar turnos") { Task { await estado.elegirModo("cliente") } }
                    .buttonStyle(BotonTexto(color: C.tinta)).frame(maxWidth: .infinity)
                Button("Salir") { Task { await estado.salir() } }.buttonStyle(BotonTexto(color: C.sub)).frame(maxWidth: .infinity)
            }
            .foregroundStyle(C.tinta).padding(.horizontal, 20)
        }
        .scrollDismissesKeyboard(.interactively)
        .background(C.fondo.ignoresSafeArea())
        .onChange(of: nombre) { _, n in if !slugTocado { slug = aSlug(n) } }
    }

    private struct Creado: Decodable { var id: String }

    private func crear() async {
        creando = true
        defer { creando = false }
        error = ""
        do {
            let r: Creado = try await estado.api.enviar("POST", "/v1/negocios", ["nombre": nombre.trimmingCharacters(in: .whitespaces), "slug": slug])
            try await estado.cargarSesion()
            if let n = estado.negocios.first(where: { $0.id == r.id }) { await estado.entrarANegocio(n) }
            Haptico.exito()
        } catch let e as ErrorApi {
            error = e.detalle
        } catch {
            self.error = "No pudimos crearlo. Probá de nuevo."
        }
    }
}
