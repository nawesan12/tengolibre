import Foundation

/// El ingreso con mail (igual que Corredor, Nucleo/ClerkClave.swift): según la cuenta, contraseña o
/// código por mail; alguien nuevo se da de alta con el código (y la contraseña, si la instancia la
/// pide); y recuperar el acceso (código `reset_password_email_code`, contraseña nueva y cierre de las
/// otras sesiones). Si Clerk no conoce el teléfono (`needs_client_trust`), después de la contraseña
/// pide un código por mail. Usa los pedidos de `Clerk` (Nucleo/Clerk.swift) y termina igual que Apple y
/// Google: el JWT de la sesión de Clerk va a `POST /v1/auth/clerk` (Estado.ingresarCon).
final class ClerkMail {
    /// El paso que sigue en la pantalla.
    enum Paso: Equatable {
        case clave
        case codigo(alta: Bool)
        case claveNueva(alta: Bool)
        case listo(token: String)
    }

    private let clerk: Clerk
    private var cliente = ""
    private var id = ""
    private(set) var alta = false
    /// El código es el segundo paso del ingreso (teléfono nuevo para Clerk), no el primero.
    private var segundo = false

    init?(publicable: String?, http: URLSession = .shared) {
        guard let c = Clerk(publicable: publicable, http: http) else { return nil }
        clerk = c
    }

    /// Con la clave publicable del Info.plist (la misma que usa `Estado` para Apple y Google).
    static func configurado() -> ClerkMail? {
        ClerkMail(publicable: Bundle.main.object(forInfoDictionaryKey: "TLClerk") as? String)
    }

    private func pedir(_ ruta: String, _ metodo: String = "GET", cuerpo: [String: String]? = nil, conCliente: Bool = true) async throws -> Clerk.Respuesta {
        do {
            let (r, nuevo) = try await clerk.pedir(ruta, metodo, cuerpo: cuerpo, cliente: conCliente ? cliente : nil)
            if !nuevo.isEmpty { cliente = nuevo }
            return r
        } catch let e as Clerk.ErrorClerk {
            throw e
        } catch {
            throw Clerk.ErrorClerk(codigo: "sin_conexion")
        }
    }

    /// El JWT de la sesión de Clerk; la sesión de Clerk se cierra (Tengo Libre sigue con la propia).
    private func sesionDe(_ r: Clerk.Respuesta) async throws -> Paso {
        // Teléfono nuevo para Clerk: confirma con un código por mail antes de dar la sesión.
        if !alta, let estado = r.response?.status, estado == "needs_client_trust" || estado == "needs_second_factor" {
            guard let mail = r.response?.supported_second_factors?.first(where: { $0.strategy == "email_code" })?.email_address_id else {
                throw Clerk.ErrorClerk(codigo: "sin_sesion_" + estado)
            }
            _ = try await pedir("/client/sign_ins/\(id)/prepare_second_factor", "POST", cuerpo: ["strategy": "email_code", "email_address_id": mail])
            segundo = true
            return .codigo(alta: false)
        }
        guard let sesion = r.response?.created_session_id, sesion.range(of: "^sess_[A-Za-z0-9]+$", options: .regularExpression) != nil else {
            throw Clerk.ErrorClerk(codigo: "sin_sesion_" + (r.response?.status ?? "desconocido"))
        }
        let t = try await pedir("/client/sessions/\(sesion)/tokens", "POST", cuerpo: [:])
        guard let jwt = t.jwt, !jwt.isEmpty else { throw Clerk.ErrorClerk(codigo: "clerk_no_responde") }
        _ = try? await pedir("/client/sessions/\(sesion)/remove", "POST", cuerpo: [:])
        return .listo(token: jwt)
    }

    /// Arranca con el mail: si tiene cuenta, ingreso (contraseña o código); si no, alta con código.
    func empezar(email: String) async throws -> Paso {
        cliente = ""
        segundo = false
        do {
            let r = try await pedir("/client/sign_ins", "POST", cuerpo: ["identifier": email], conCliente: false)
            id = r.response?.id ?? ""
            alta = false
            let factores = r.response?.supported_first_factors ?? []
            if factores.contains(where: { $0.strategy == "password" }) { return .clave }
            guard let mail = factores.first(where: { $0.strategy == "email_code" })?.email_address_id else {
                throw Clerk.ErrorClerk(codigo: "sin_mail")
            }
            _ = try await pedir("/client/sign_ins/\(id)/prepare_first_factor", "POST", cuerpo: ["strategy": "email_code", "email_address_id": mail])
            return .codigo(alta: false)
        } catch let e as Clerk.ErrorClerk where e.codigo == "form_identifier_not_found" {
            // Alta: el mail, y verificarlo con un código.
            cliente = ""
            let a = try await pedir("/client/sign_ups", "POST", cuerpo: ["email_address": email], conCliente: false)
            id = a.response?.id ?? ""
            alta = true
            _ = try await pedir("/client/sign_ups/\(id)/prepare_verification", "POST", cuerpo: ["strategy": "email_code"])
            return .codigo(alta: true)
        }
    }

    /// Con la contraseña de la cuenta.
    func entrarConClave(_ clave: String) async throws -> Paso {
        let r = try await pedir("/client/sign_ins/\(id)/attempt_first_factor", "POST", cuerpo: ["strategy": "password", "password": clave])
        return try await sesionDe(r)
    }

    /// El código que llegó por mail (ingreso, alta, teléfono nuevo o recuperar).
    func confirmarCodigo(_ codigo: String, recuperar: Bool = false) async throws -> Paso {
        let solo = codigo.filter(\.isNumber)
        let ruta: String
        if alta {
            ruta = "/client/sign_ups/\(id)/attempt_verification"
        } else if segundo {
            ruta = "/client/sign_ins/\(id)/attempt_second_factor"
        } else {
            ruta = "/client/sign_ins/\(id)/attempt_first_factor"
        }
        let r = try await pedir(ruta, "POST", cuerpo: ["strategy": recuperar ? "reset_password_email_code" : "email_code", "code": solo])
        if r.response?.status == "needs_new_password" { return .claveNueva(alta: false) }
        // La instancia pide contraseña para el alta.
        if alta, r.response?.status == "missing_requirements", r.response?.missing_fields?.contains("password") == true {
            return .claveNueva(alta: true)
        }
        segundo = false
        return try await sesionDe(r)
    }

    /// La contraseña nueva: al recuperar (cierra las otras sesiones de Clerk) o la que pide el alta.
    func guardarClave(_ clave: String) async throws -> Paso {
        let r: Clerk.Respuesta
        if alta {
            r = try await pedir("/client/sign_ups/\(id)", "PATCH", cuerpo: ["password": clave])
        } else {
            r = try await pedir("/client/sign_ins/\(id)/reset_password", "POST", cuerpo: ["password": clave, "sign_out_of_other_sessions": "true"])
        }
        return try await sesionDe(r)
    }

    /// Recuperar: arranca el ingreso con el mail y pide el código para la contraseña nueva. `false` si el
    /// mail no tiene cuenta o no usa contraseña (la pantalla dice lo mismo: no se revela quién está
    /// registrado).
    func pedirRecuperar(email: String) async throws -> Bool {
        cliente = ""
        alta = false
        segundo = false
        let r: Clerk.Respuesta
        do {
            r = try await pedir("/client/sign_ins", "POST", cuerpo: ["identifier": email], conCliente: false)
        } catch let e as Clerk.ErrorClerk where e.codigo == "form_identifier_not_found" {
            return false
        }
        id = r.response?.id ?? ""
        guard let mail = r.response?.supported_first_factors?.first(where: { $0.strategy == "reset_password_email_code" })?.email_address_id else {
            return false
        }
        _ = try await pedir("/client/sign_ins/\(id)/prepare_first_factor", "POST",
                            cuerpo: ["strategy": "reset_password_email_code", "email_address_id": mail])
        return true
    }

    // MARK: Requisitos y mensajes (los mismos de la web)

    private static let comunes: Set<String> = [
        "1234567890", "12345678910", "qwertyuiop", "contraseña", "contrasena", "password12", "password123", "argentina1",
        "0123456789", "1111111111", "abcdefghij", "tengolibre", "tengolibre1", "turnos123", "iloveyou12", "boca123456", "river12345",
        "12345678", "87654321", "123456789", "password", "qwertyui", "abcdefgh", "asdfghjk", "argentina", "iloveyou",
    ]

    /// Los requisitos de la contraseña nueva que se tildan solos: 8 caracteres o más, y que no sea de las
    /// más usadas.
    static func requisitos(_ clave: String) -> (largo: Bool, comun: Bool) {
        (clave.count >= 8, !clave.isEmpty && !esComun(clave))
    }

    private static func esComun(_ c: String) -> Bool {
        let x = c.lowercased()
        if comunes.contains(x) { return true }
        if x.count > 1, let p = x.first, x.allSatisfy({ $0 == p }) { return true }
        return x.hasPrefix("01234567") || x.hasPrefix("12345678") || x.hasPrefix("98765432")
    }

    /// El error de Clerk como lo lee la persona.
    static func mensaje(_ codigo: String) -> String {
        let m: [String: String] = [
            "form_password_incorrect": "La contraseña no es esa. Probá de nuevo o recuperala.",
            "form_code_incorrect": "El código no es ese. Revisalo y probá de nuevo.",
            "verification_expired": "El código venció. Pedí uno nuevo.",
            "verification_failed": "El código no es ese. Revisalo y probá de nuevo.",
            "form_password_pwned": "Esa contraseña apareció en filtraciones. Elegí otra.",
            "form_password_length_too_short": "La contraseña es muy corta: tiene que tener 8 caracteres o más.",
            "form_password_validation_failed": "La contraseña no cumple los requisitos.",
            "form_param_format_invalid": "Revisá el mail: no parece válido.",
            "form_identifier_exists": "Ese mail ya tiene cuenta. Ingresá.",
            "too_many_requests": "Hubo demasiados intentos. Probá de nuevo en unos minutos.",
            "strategy_for_user_invalid": "Esa cuenta entra con Google o Apple: usá ese botón.",
            "sin_mail": "Esa cuenta entra con Google o Apple: usá ese botón.",
            "sin_conexion": "No pudimos conectar. Probá de nuevo.",
            "sin_clave": "El ingreso no está disponible en este momento.",
        ]
        return m[codigo] ?? "No pudimos completar el ingreso. Probá de nuevo."
    }

    /// Cualquier error del ingreso con mail: el de Clerk, o el que manda la API al canjear la sesión
    /// (por ejemplo, 409 `cuenta_borrada` con su detalle).
    static func mensaje(de error: Error) -> String {
        if let c = error as? Clerk.ErrorClerk { return mensaje(c.codigo) }
        if let a = error as? ErrorApi { return a.detalle }
        return "No pudimos completar el ingreso. Probá de nuevo."
    }
}
