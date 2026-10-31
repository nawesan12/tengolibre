/**
 * Contratos SUPUESTOS de la API para lo que el diseño v2 de cobros muestra y la API todavía no tiene.
 * La UI ya los llama; al integrar, ajustar acá (y en apps/api) si la forma real es otra.
 *
 * Suscripción (todos con la sesión de la dueña; responden problem+json si no se puede):
 * - POST /v1/panel/suscripcion/cambiar   { plan: 'inicial' | 'completo' } → 200 Suscripcion
 *     Cambia el plan de una suscripción autorizada; el precio nuevo rige desde el próximo cobro.
 * - POST /v1/panel/suscripcion/pagar     {} → 200 { url: string }
 *     Con el cobro fallido (estado 'pausada'): link de Mercado Pago para pagar ya la cuota adeudada.
 * - POST /v1/panel/suscripcion/tarjeta   {} → 200 { url: string }
 *     Link de Mercado Pago para cambiar el medio de pago del débito.
 * - POST /v1/panel/suscripcion/reactivar {} → 200 { url: string }
 *     Con la suscripción cancelada: la vuelve a dar de alta con el mismo plan (link de Mercado Pago).
 *
 * Mercado Pago:
 * - GET /v1/panel/mercadopago agrega `cuenta?: string` (el nombre o razón social de la cuenta
 *   conectada, para el chip «Mercado Pago conectado · LUMI ESTÉTICA SRL»). Si no viene, se muestra
 *   el número de usuario.
 */
export const SUSCRIPCION_CAMBIAR = '/v1/panel/suscripcion/cambiar';
export const SUSCRIPCION_PAGAR = '/v1/panel/suscripcion/pagar';
export const SUSCRIPCION_TARJETA = '/v1/panel/suscripcion/tarjeta';
export const SUSCRIPCION_REACTIVAR = '/v1/panel/suscripcion/reactivar';
