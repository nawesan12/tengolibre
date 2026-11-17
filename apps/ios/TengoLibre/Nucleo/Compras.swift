import Foundation
import StoreKit

/// La suscripción desde el iPhone (StoreKit 2): los planes son suscripciones mensuales de la App Store
/// (grupo «Tengo Libre»). Cada compra lleva el id del negocio (appAccountToken) y su transacción
/// firmada va a la API (`POST /v1/panel/suscripcion/apple`), que además recibe las renovaciones y
/// cancelaciones directo de la App Store. Desde la web se sigue pagando con Mercado Pago.
@MainActor
enum Compras {
    static let productos: [String: String] = [
        "ar.com.tengolibre.inicial.mensual": "inicial",
        "ar.com.tengolibre.completo.mensual": "completo"
    ]

    /// Los planes de la App Store, del más barato al más caro (vacío si la tienda no los da).
    static func planes() async -> [Product] {
        let p = (try? await Product.products(for: Array(productos.keys))) ?? []
        return p.sorted { $0.price < $1.price }
    }

    enum Resultado { case listo, cancelado, pendiente }

    /// Este Apple ID ya paga la suscripción de otro negocio: comprar acá la cambiaría de plan, no sumaría otra.
    static let yaPagaOtro = ErrorApi(estado: 409, codigo: "otro_negocio_tienda", titulo: "Este Apple ID ya paga otro negocio",
                                     detalle: "Para pagar este negocio, usá otro Apple ID en la App Store o pagá desde la web.")

    /// Si este Apple ID tiene vigente un plan de otro negocio (su appAccountToken es otro).
    static func pagaOtroNegocio(_ cuenta: UUID) async -> Bool {
        for await v in Transaction.currentEntitlements {
            if case .verified(let t) = v, productos[t.productID] != nil, t.revocationDate == nil, let otra = t.appAccountToken, otra != cuenta { return true }
        }
        return false
    }

    /// Compra para el negocio elegido y le pasa la transacción a la API.
    static func comprar(_ p: Product, estado: Estado) async throws -> Resultado {
        guard let negocio = estado.negocio?.id, let cuenta = UUID(uuidString: negocio) else { return .cancelado }
        // Se avisa antes de abrir la compra.
        if await pagaOtroNegocio(cuenta) { throw yaPagaOtro }
        let r = try await p.purchase(options: [.appAccountToken(cuenta)])
        switch r {
        case .success(let v):
            try await registrar(v, estado: estado)
            return .listo
        case .pending:
            return .pendiente
        default:
            return .cancelado
        }
    }

    /// Manda a la API una transacción verificada por StoreKit y la cierra. Va con el negocio de la compra
    /// (su appAccountToken), no con el elegido ahora: una renovación o una compra de otro dispositivo
    /// puede ser de otro negocio de la persona.
    static func registrar(_ v: VerificationResult<Transaction>, estado: Estado) async throws {
        guard case .verified(let t) = v else { throw ErrorApi.desconocido }
        guard productos[t.productID] != nil else { await t.finish(); return }
        _ = try await estado.api.pedir("POST", "/v1/panel/suscripcion/apple", cuerpo: Api.codificador.encode(["transaccion": v.jwsRepresentation]),
                                       negocio: t.appAccountToken?.uuidString.lowercased())
        await t.finish()
        estado.version += 1
        await estado.refrescar()
    }

    /// «Restaurar compras»: sincroniza con la App Store y vuelve a mandar lo vigente. Una que falla no
    /// corta las demás; si no se pudo ninguna, se dice por qué.
    static func restaurar(estado: Estado) async throws -> Int {
        try? await AppStore.sync()
        var n = 0
        var primerError: Error?
        for await v in Transaction.currentEntitlements {
            if case .verified(let t) = v, productos[t.productID] != nil {
                do {
                    try await registrar(v, estado: estado)
                    n += 1
                } catch {
                    if primerError == nil { primerError = error }
                }
            }
        }
        if n == 0, let primerError { throw primerError }
        return n
    }

    /// Renovaciones, compras pendientes que se aprobaron y compras hechas en otro dispositivo: se
    /// escuchan mientras la app está abierta.
    static func escuchar(estado: Estado) {
        Task.detached {
            for await v in Transaction.updates {
                await MainActor.run {
                    Task { try? await registrar(v, estado: estado) }
                }
            }
        }
    }
}
