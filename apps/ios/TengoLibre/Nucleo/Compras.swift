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

    /// Compra para el negocio elegido y le pasa la transacción a la API.
    static func comprar(_ p: Product, estado: Estado) async throws -> Resultado {
        guard let negocio = estado.negocio?.id, let cuenta = UUID(uuidString: negocio) else { return .cancelado }
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

    /// Manda a la API una transacción verificada por StoreKit y la cierra.
    static func registrar(_ v: VerificationResult<Transaction>, estado: Estado) async throws {
        guard case .verified(let t) = v else { throw ErrorApi.desconocido }
        guard productos[t.productID] != nil else { await t.finish(); return }
        try await estado.api.hacer("POST", "/v1/panel/suscripcion/apple", ["transaccion": v.jwsRepresentation])
        await t.finish()
        estado.version += 1
        await estado.refrescar()
    }

    /// «Restaurar compras»: sincroniza con la App Store y vuelve a mandar lo vigente.
    static func restaurar(estado: Estado) async throws -> Int {
        try? await AppStore.sync()
        var n = 0
        for await v in Transaction.currentEntitlements {
            if case .verified(let t) = v, productos[t.productID] != nil {
                try await registrar(v, estado: estado)
                n += 1
            }
        }
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
