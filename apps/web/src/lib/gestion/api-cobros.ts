/**
 * Lo que el diseño v2 de cobros pide además de lo de siempre (apps/api/internal/dominio/suscripcion
 * y cobros). Sin las credenciales de Mercado Pago de la plataforma responden 409 sin_suscripciones.
 *
 * Suscripción (todos con la sesión de la dueña; responden problem+json si no se puede):
 * - POST /v1/panel/suscripcion          { plan, periodo?: 'mensual' | 'anual' } → 201 { url }
 * - POST /v1/panel/suscripcion/cambiar   { plan: 'inicial' | 'equipo' | 'centro', periodo? } → 200 Suscripcion | { url }
 *     Mismo período: el precio nuevo rige desde el próximo cobro. Otro período: un alta nueva en
 *     Mercado Pago (responde { url }) que empieza cuando vence lo pagado.
 * - PUT  /v1/panel/suscripcion/extras    { cantidad: 0..100 } → 200 Suscripcion
 *     Profesionales extra de Centro ($1.900 por mes cada uno), con la suscripción de Mercado Pago.
 * - POST /v1/panel/suscripcion/whatsapp  {} → 201 { url }
 *     Un pack de 100 recordatorios por WhatsApp ($5.900, pago único con Mercado Pago).
 *
 * Los topes del plan responden 402 { code: 'tope_plan', plan: '<el que lo habilita>', limit? }.
 * - POST /v1/panel/suscripcion/pagar     {} → 200 { url: string }
 *     Con el cobro fallido (estado 'pausada' sin canceladaEn): un alta nueva que cobra ya y, al
 *     autorizarse, reemplaza a la pausada (Mercado Pago no deja pagar una cuota suelta).
 * - POST /v1/panel/suscripcion/tarjeta   {} → 200 { url: string }
 *     Un alta nueva con la tarjeta nueva; con el mes pago empieza a cobrar al vencer.
 * - POST /v1/panel/suscripcion/reactivar {} → 200 { url: string }
 *     Con la suscripción cancelada: la vuelve a dar de alta con el mismo plan (link de Mercado Pago).
 *     GET /v1/panel/suscripcion trae la última cancelada si no hay otra.
 *
 * Mercado Pago:
 * - GET /v1/panel/mercadopago agrega `cuenta?: string` (el nombre o razón social de la cuenta
 *   conectada, de /users/me al conectar, para el chip «Mercado Pago conectado · LUMI ESTÉTICA SRL»).
 *   Si no viene (Mercado Pago no lo dio), se muestra el número de usuario.
 */
export const SUSCRIPCION_CAMBIAR = '/v1/panel/suscripcion/cambiar';
export const SUSCRIPCION_PAGAR = '/v1/panel/suscripcion/pagar';
export const SUSCRIPCION_TARJETA = '/v1/panel/suscripcion/tarjeta';
export const SUSCRIPCION_REACTIVAR = '/v1/panel/suscripcion/reactivar';
export const SUSCRIPCION_EXTRAS = '/v1/panel/suscripcion/extras';
export const SUSCRIPCION_WHATSAPP = '/v1/panel/suscripcion/whatsapp';
