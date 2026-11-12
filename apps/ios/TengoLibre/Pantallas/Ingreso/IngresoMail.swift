import SwiftUI

/// «Entrar con mail» (como Corredor y la web): según la cuenta, la contraseña o un código de 6 números;
/// si es la primera vez se crea la cuenta (con código, y la contraseña si la pide). «¿No podés
/// entrar?» lleva a recuperar el acceso. Termina igual que Apple y Google: Estado.ingresarCon.
struct IngresoMail: View {
    @Environment(Estado.self) private var estado
    @Environment(\.dismiss) private var cerrar
    @State private var email = ""
    @State private var clave = ""
    @State private var codigo = ""
    @State private var paso: ClerkMail.Paso?
    @State private var flujo = ClerkMail.configurado()
    @State private var ocupado = false
    @State private var error = ""
    @State private var recuperando = false

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    if paso == nil { pasoMail } else { volver; pasoActual }
                    if !error.isEmpty { Text(error).font(Tipo.lato(14, .bold)).foregroundStyle(C.error) }
                    if flujo == nil {
                        Text("El ingreso con mail no está disponible en este momento. Probá de nuevo en un rato.")
                            .font(Tipo.lato(14, .bold)).foregroundStyle(C.error)
                    }
                }
                .foregroundStyle(C.tinta).padding(20)
            }
            .scrollDismissesKeyboard(.interactively)
            .background(C.fondo)
            .navigationTitle("Entrar con mail").navigationBarTitleDisplayMode(.inline)
            .toolbar { ToolbarItem(placement: .cancellationAction) { Button("Cancelar") { cerrar() } } }
        }
        .presentationCornerRadius(Radio.hoja)
        .sheet(isPresented: $recuperando) { RecuperarAcceso(email: email).presentationDetents([.large]) }
    }

    private var pasoMail: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text("Tu mail").font(Tipo.lato(30, .black)).tracking(-0.8)
            Text("Si es tu primera vez, creamos tu cuenta con un código que te llega por mail.").font(Tipo.lato(16)).foregroundStyle(C.sub)
            Campo(titulo: "", texto: $email, ayuda: "tu@mail.com", teclado: .emailAddress)
                .textContentType(.emailAddress)
                .onSubmit { seguir() }
            Button(ocupado ? "Un segundo…" : "Seguir") { seguir() }
                .buttonStyle(BotonMenta(alto: 52, conSombra: false))
                .disabled(ocupado || !email.contains("@") || flujo == nil)
        }
    }

    private var volver: some View {
        Button { otroMail() } label: {
            Label(email, systemImage: "chevron.left").font(Tipo.lato(15, .bold)).foregroundStyle(C.mentaTexto).lineLimit(1)
        }
        .buttonStyle(.plain)
        .accessibilityLabel("Usar otro mail")
    }

    @ViewBuilder
    private var pasoActual: some View {
        switch paso {
        case .clave:
            Text("Tu contraseña").font(Tipo.lato(30, .black)).tracking(-0.8)
            Campo(titulo: "", texto: $clave, secreto: true).textContentType(.password).onSubmit { seguir() }
            Button(ocupado ? "Entrando…" : "Entrar") { seguir() }
                .buttonStyle(BotonMenta(alto: 52, conSombra: false)).disabled(ocupado || clave.isEmpty)
            Button("¿No podés entrar?") { recuperando = true }.buttonStyle(BotonTexto()).frame(maxWidth: .infinity)
        case .codigo(let alta):
            Text("Revisá tu mail").font(Tipo.lato(30, .black)).tracking(-0.8)
            Text(Formato.sinViudas("Te mandamos un código a \(email). \(alta ? "Con él creamos tu cuenta." : "Escribilo para entrar.")"))
                .font(Tipo.lato(16)).foregroundStyle(C.sub)
            CampoCodigo(codigo: $codigo) { if !ocupado { seguir() } }
            Button(ocupado ? "Revisando…" : alta ? "Crear mi cuenta" : "Entrar") { seguir() }
                .buttonStyle(BotonMenta(alto: 52, conSombra: false)).disabled(ocupado || codigo.count < 6)
        case .claveNueva:
            Text("Elegí una contraseña").font(Tipo.lato(30, .black)).tracking(-0.8)
            Text("La vas a usar para entrar con tu mail.").font(Tipo.lato(16)).foregroundStyle(C.sub)
            Campo(titulo: "", texto: $clave, secreto: true).textContentType(.newPassword)
            RequisitosClave(clave: clave)
            Button(ocupado ? "Guardando…" : "Guardar y entrar") { seguir() }
                .buttonStyle(BotonMenta(alto: 52, conSombra: false)).disabled(ocupado)
        default:
            EmptyView()
        }
    }

    private func otroMail() {
        paso = nil
        clave = ""; codigo = ""; error = ""
        flujo = ClerkMail.configurado()
    }

    private func seguir() {
        guard let flujo, !ocupado else { return }
        if case .claveNueva = paso {
            let r = ClerkMail.requisitos(clave)
            if !r.largo || !r.comun { error = "La contraseña todavía no cumple los requisitos."; return }
        }
        error = ""
        ocupado = true
        let actual = paso
        let mail = email.trimmingCharacters(in: .whitespaces).lowercased()
        Task {
            defer { ocupado = false }
            do {
                let sig: ClerkMail.Paso
                switch actual {
                case nil: sig = try await flujo.empezar(email: mail)
                case .clave: sig = try await flujo.entrarConClave(clave)
                case .codigo: sig = try await flujo.confirmarCodigo(codigo)
                case .claveNueva: sig = try await flujo.guardarClave(clave)
                case .listo: return
                }
                if case .listo(let token) = sig {
                    try await estado.ingresarCon(tokenClerk: token)
                    Haptico.exito()
                    cerrar()
                } else {
                    if case .claveNueva = sig { clave = "" }
                    if case .codigo = sig { codigo = "" }
                    paso = sig
                }
            } catch {
                self.error = ClerkMail.mensaje(de: error)
                Haptico.error()
            }
        }
    }
}

/// El código de 6 números que llega por mail: grande, espaciado, y sigue solo al completarlo.
struct CampoCodigo: View {
    @Binding var codigo: String
    var completo: () -> Void

    var body: some View {
        TextField("123456", text: $codigo)
            .textContentType(.oneTimeCode).keyboardType(.numberPad)
            .font(Tipo.lato(28, .bold)).monospacedDigit().tracking(8).foregroundStyle(C.tinta)
            .padding(.horizontal, 14).padding(.vertical, 13)
            .background(C.campo, in: RoundedRectangle(cornerRadius: 14, style: .continuous))
            .onChange(of: codigo) { _, n in
                let solo = String(n.filter(\.isNumber).prefix(6))
                if solo != n { codigo = solo }
                if solo.count == 6 { completo() }
            }
            .accessibilityLabel("Código")
    }
}

/// Los requisitos de la contraseña nueva, que se tildan solos mientras se escribe.
struct RequisitosClave: View {
    var clave: String

    var body: some View {
        let r = ClerkMail.requisitos(clave)
        VStack(alignment: .leading, spacing: 6) {
            fila(r.largo, "8 caracteres o más")
            fila(r.comun, "No es una de las más usadas")
        }
        .accessibilityElement(children: .combine)
    }

    private func fila(_ ok: Bool, _ texto: String) -> some View {
        HStack(spacing: 8) {
            Image(systemName: ok ? "checkmark.circle.fill" : "circle").font(.system(size: 16, weight: .semibold))
                .foregroundStyle(ok ? C.mentaTexto : C.sub)
            Text(texto).font(Tipo.lato(14, ok ? .bold : .regular)).foregroundStyle(ok ? C.mentaTexto : C.sub)
        }
    }
}

/// Recuperar el acceso con Clerk: un código por mail, con él la contraseña nueva, se cierran las otras
/// sesiones y se entra. Si el mail no tiene cuenta, dice lo mismo (no se revela quién está registrado).
struct RecuperarAcceso: View {
    @Environment(Estado.self) private var estado
    @Environment(\.dismiss) private var cerrar
    @State var email: String
    @State private var codigo = ""
    @State private var clave = ""
    @State private var paso = "mail"
    @State private var enviado = false
    @State private var enviando = false
    @State private var error = ""
    @State private var flujo = ClerkMail.configurado()

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    if paso == "nueva" { nueva } else { pedir }
                }
                .foregroundStyle(C.tinta).padding(20)
            }
            .scrollDismissesKeyboard(.interactively)
            .background(C.fondo)
            .navigationTitle("Recuperar el acceso").navigationBarTitleDisplayMode(.inline)
            .toolbar { ToolbarItem(placement: .cancellationAction) { Button("Cancelar") { cerrar() } } }
        }
        .presentationCornerRadius(Radio.hoja)
    }

    @ViewBuilder
    private var pedir: some View {
        Image(systemName: "envelope.badge").font(.system(size: 24, weight: .semibold)).foregroundStyle(C.mentaTexto)
            .frame(width: 56, height: 56).background(C.mentaSuave, in: Circle())
        Text("¿No podés entrar?").font(Tipo.lato(30, .black)).tracking(-0.8)
        Text(Formato.sinViudas("Si entrás con Google o Apple, usá ese botón: no necesitás contraseña. Si usás mail, te mandamos un código para crear una nueva."))
            .font(Tipo.lato(16)).foregroundStyle(C.sub)
        if paso == "mail" {
            Campo(titulo: "", texto: $email, ayuda: "tu@mail.com", teclado: .emailAddress).textContentType(.emailAddress)
            Button(enviando ? "Mandando…" : "Mandarme el código") { enviar() }
                .buttonStyle(BotonMenta(alto: 52, conSombra: false)).disabled(enviando || flujo == nil || !email.contains("@"))
        } else {
            Text("Código que te llegó a \(email)").font(Tipo.lato(13, .bold)).foregroundStyle(C.sub)
            CampoCodigo(codigo: $codigo) {}
            Button(enviando ? "Revisando…" : "Seguir") { enviar() }
                .buttonStyle(BotonMenta(alto: 52, conSombra: false)).disabled(enviando || codigo.count < 6)
        }
        if !error.isEmpty { Text(error).font(Tipo.lato(14, .bold)).foregroundStyle(C.error) }
        if enviado {
            Label(Formato.sinViudas("Listo: si ese mail tiene cuenta, te llega un código en un minuto. Revisá también spam."), systemImage: "envelope")
                .font(Tipo.lato(14)).foregroundStyle(C.tinta)
                .padding(12).frame(maxWidth: .infinity, alignment: .leading)
                .background(C.mentaSuave, in: RoundedRectangle(cornerRadius: 14, style: .continuous))
        }
        Button("Volver a ingresar") { cerrar() }.buttonStyle(BotonTexto()).frame(maxWidth: .infinity)
    }

    @ViewBuilder
    private var nueva: some View {
        Text("Elegí una contraseña nueva").font(Tipo.lato(30, .black)).tracking(-0.8)
        Campo(titulo: "", texto: $clave, secreto: true).textContentType(.newPassword)
        RequisitosClave(clave: clave)
        if !error.isEmpty { Text(error).font(Tipo.lato(14, .bold)).foregroundStyle(C.error) }
        Button(enviando ? "Guardando…" : "Guardar y entrar") { enviar() }
            .buttonStyle(BotonMenta(alto: 52, conSombra: false)).disabled(enviando)
        Text("Cerramos las otras sesiones abiertas por seguridad.").font(Tipo.lato(13)).foregroundStyle(C.sub).frame(maxWidth: .infinity)
    }

    private func enviar() {
        guard let flujo, !enviando else { return }
        if paso == "nueva" {
            let r = ClerkMail.requisitos(clave)
            if !r.largo || !r.comun { error = "La contraseña todavía no cumple los requisitos."; return }
        }
        error = ""
        enviando = true
        Task {
            defer { enviando = false }
            do {
                switch paso {
                case "mail":
                    let hay = try await flujo.pedirRecuperar(email: email.trimmingCharacters(in: .whitespaces).lowercased())
                    enviado = true
                    if hay { paso = "codigo" }
                case "codigo", "confianza":
                    let sig = try await flujo.confirmarCodigo(codigo, recuperar: paso == "codigo")
                    if case .claveNueva = sig { paso = "nueva" }
                    if case .listo(let token) = sig { try await entrar(token) }
                default:
                    let sig = try await flujo.guardarClave(clave)
                    if case .listo(let token) = sig { try await entrar(token) }
                    // Teléfono nuevo para Clerk: un código más por mail para terminar.
                    if case .codigo = sig { codigo = ""; enviado = false; paso = "confianza" }
                }
            } catch {
                self.error = ClerkMail.mensaje(de: error)
                Haptico.error()
            }
        }
    }

    private func entrar(_ token: String) async throws {
        try await estado.ingresarCon(tokenClerk: token)
        Haptico.exito()
        cerrar()
    }
}
