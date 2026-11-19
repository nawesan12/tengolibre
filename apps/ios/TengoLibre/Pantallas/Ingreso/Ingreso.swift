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

/// 01 Ingreso: petróleo con el logotipo, «Tu agenda, en el bolsillo.» y la hoja con Apple, Google y
/// «Entrar con mail» (IngresoMail).
struct Ingreso: View {
    @Environment(Estado.self) private var estado
    @State private var entrando: String?
    @State private var error = ""
    @State private var conMail = false

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
                Spacer().frame(height: 360)
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
                Button { error = ""; conMail = true } label: {
                    HStack(spacing: 10) {
                        Image(systemName: "envelope").font(.system(size: 18, weight: .semibold))
                        Text("Entrar con mail").font(Tipo.lato(17, .bold))
                    }
                    .foregroundStyle(C.tinta).frame(maxWidth: .infinity, minHeight: 52)
                    .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                if !error.isEmpty {
                    Text(error).font(Tipo.lato(14, .bold)).foregroundStyle(C.error).multilineTextAlignment(.center)
                }
                Text("Usá la misma cuenta que en la web.").font(Tipo.lato(13)).foregroundStyle(C.sub).multilineTextAlignment(.center)
            }
            .disabled(entrando != nil)
            .padding(.horizontal, 20).padding(.top, 24).padding(.bottom, 12)
            .background(C.tarjeta, in: UnevenRoundedRectangle(topLeadingRadius: 28, topTrailingRadius: 28, style: .continuous))
            .ignoresSafeArea(edges: .bottom)
        }
        .sheet(isPresented: $conMail) { IngresoMail().presentationDetents([.large]) }
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

/// Alta: quien entra sin negocio abre el suyo (nombre y link).
struct Alta: View {
    @Environment(Estado.self) private var estado
    @State private var nombre = ""
    @State private var slug = ""
    /// Lo que muestra el campo del link: se normaliza en onChange (ver CambiarLink).
    @State private var campoSlug = ""
    @State private var slugTocado = false
    /// Si el link está libre (lo pregunta a la API con 300 ms de pausa): libre, tomada o invalida.
    @State private var libre: String?
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
                // La prueba es de 7 días (cobros: now() + 7 days), como dice la web.
                Text(Formato.sinViudas("7 días gratis. Después lo terminás de armar desde Más: servicios, equipo y horarios.")).font(Tipo.lato(16)).foregroundStyle(C.sub)
                Bloque {
                    RenglonCampo(titulo: "Nombre del negocio", texto: $nombre, placeholder: "Lumi Estética")
                    RenglonCampo(titulo: "Tu link", texto: $campoSlug, placeholder: "lumi", teclado: .URL)
                }
                // El link, grande: es lo que va a compartir (la gente no lee: que se vea antes de explicarlo).
                VStack(alignment: .leading, spacing: 8) {
                    Label("TU LINK", systemImage: "link").font(Tipo.rotulo).tracking(12 * 0.08).foregroundStyle(C.mentaTexto)
                    VStack(alignment: .leading, spacing: 0) {
                        Text("\(Web.base.host() ?? "tengolibre.com.ar")/").font(Tipo.lato(15, .bold)).foregroundStyle(C.sub)
                        Text(slug.isEmpty ? "tu-negocio" : slug).font(Tipo.lato(26, .black)).tracking(-0.6).foregroundStyle(C.mentaTexto)
                            .lineLimit(2).minimumScaleFactor(0.6)
                    }
                    if slug.count >= 3 {
                        Label(libreTexto.0, systemImage: libreTexto.1).font(Tipo.lato(13, .bold)).foregroundStyle(libreTexto.2)
                    }
                    Label("Este es tu link: lo compartís y te reservan\u{00a0}solos.", systemImage: "megaphone.fill")
                        .font(Tipo.lato(14, .bold)).foregroundStyle(C.tinta)
                        .padding(12).frame(maxWidth: .infinity, alignment: .leading)
                        .background(C.mentaSuave, in: RoundedRectangle(cornerRadius: 14, style: .continuous))
                }
                .padding(16).tarjeta(Radio.bloque)
                .overlay(RoundedRectangle(cornerRadius: Radio.bloque, style: .continuous).strokeBorder(libre == "libre" ? C.menta : C.linea, lineWidth: 2))
                if !error.isEmpty { Text(error).font(Tipo.lato(14, .bold)).foregroundStyle(C.error) }
                Button(creando ? "Creando…" : "Crear mi negocio") { Task { await crear() } }
                    .buttonStyle(BotonMenta()).disabled(creando || nombre.count < 2 || slug.count < 3 || libre == "tomada" || libre == "invalida" || libre == "escribiendo")
                // La app es para negocios: quien la bajó para reservar sigue por la web (y ve sus turnos en Mi cuenta).
                VStack(spacing: 8) {
                    Link(destination: Web.base.appending(path: "explorar")) {
                        Text(Formato.sinViudas("¿Querés reservar un turno? Hacelo desde la página del negocio."))
                            .font(Tipo.lato(14, .bold)).foregroundStyle(C.sub).multilineTextAlignment(.center)
                    }
                    Link(destination: Web.base.appending(path: "mi-cuenta")) {
                        Text("Tus turnos están en Mi\u{00a0}cuenta").font(Tipo.lato(14, .bold)).foregroundStyle(C.mentaTexto)
                    }
                }
                .frame(maxWidth: .infinity).padding(.top, 4)
                Button("Salir") { Task { await estado.salir() } }.buttonStyle(BotonTexto(color: C.sub)).frame(maxWidth: .infinity)
                // La cuenta ya existe (el ingreso la crea): se tiene que poder borrar sin abrir un negocio
                // (App Store, guía 5.1.1(v)).
                BorrarCuenta()
            }
            .foregroundStyle(C.tinta).padding(.horizontal, 20).padding(.bottom, 24)
        }
        .scrollDismissesKeyboard(.interactively)
        .background(C.fondo.ignoresSafeArea())
        .onChange(of: nombre) { _, n in if !slugTocado { slug = aSlug(n); campoSlug = slug } }
        .onChange(of: campoSlug) { _, v in
            // Si coincide con el slug, el cambio vino del nombre (o de la normalización): no es «tocado».
            guard v != slug else { return }
            slugTocado = true
            slug = LinkRegla.limpiar(v)
            if campoSlug != slug { campoSlug = slug }
        }
        .task(id: slug) {
            libre = nil
            guard slug.count >= 3 else { return }
            if LinkRegla.aMedias(slug) { libre = "escribiendo"; return }
            guard LinkRegla.valido(slug) else { libre = "invalida"; return }
            try? await Task.sleep(for: .milliseconds(300))
            if Task.isCancelled { return }
            struct D: Decodable { var estado: String }
            let d: D? = try? await estado.api.get("/v1/slugs/\(slug)")
            if !Task.isCancelled { libre = d?.estado }
        }
    }

    /// Texto, ícono y color del chequeo. Nunca sólo color.
    private var libreTexto: (String, String, Color) {
        switch libre {
        case "libre": ("Está libre. Es tuya.", "checkmark.circle.fill", C.mentaTexto)
        case "tomada": ("Ya la usa otro negocio.", "xmark.circle.fill", C.error)
        case "invalida": ("Esa dirección no se puede usar.", "xmark.circle.fill", C.error)
        case "escribiendo": ("Seguí escribiendo…", "ellipsis", C.sub)
        default: ("Fijándonos si está libre…", "ellipsis", C.sub)
        }
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
