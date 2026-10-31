/**
 * Contrato supuesto de las funciones del diseño v2 que la API todavía está armando (decisión del
 * 31/10: se construye todo lo que muestra el handoff). Las pantallas ya están conectadas a esto; si la
 * API responde 404/405/501 (todavía no está), la pantalla sigue andando sin ese dato. Ajustar acá al
 * integrar.
 *
 * Endpoints que se suponen (todos con la sesión y el negocio del panel):
 *
 * Ofrecer el hueco al primero de la lista (Propuesta 01, Turnos):
 *   GET    /v1/panel/huecos/ofertas?fecha=AAAA-MM-DD         → { ofertas: Oferta[] }
 *   POST   /v1/panel/huecos/ofertas                           { profesionalId, inicio, fin, servicioId? } → Oferta
 *   DELETE /v1/panel/huecos/ofertas/{id}                      (frenar) → 204
 *   (el tiempo para tomarlo sale de la configuración del negocio; la API pasa al siguiente sola)
 *
 * Confirmación por WhatsApp (Propuesta 02): el turno trae `confirmacion` (ver Turno abajo).
 *
 * Turno fijo (Propuesta 03, Nuevo turno):
 *   POST /v1/panel/turnos/repeticion/previa  { servicioId, profesionalId, inicio, cadaSemanas, hasta } → { fechas: FechaRepetida[] }
 *   POST /v1/panel/turnos                    + { repetir?: { cadaSemanas, hasta, fechas?: { inicio }[] } } → Turno (el primero) & { repetidos?: number }
 *
 * Cobrar en el local (Propuestas 07 y 08, Caja al marcar «Atendido»):
 *   GET  /v1/panel/productos                 → { productos: Producto[] }
 *   POST /v1/panel/turnos/{id}/cobro          { medio: Medio, productos: string[] } → { total: number }
 *   (la seña pagada sale de `sena` del turno)
 *
 * Recordar el pago de la seña (Inicio, «Recordarle»):
 *   POST /v1/panel/turnos/{id}/recordar-pago  → 204
 *
 * Pedido de días libres (Propuesta 06, en Avisos): la notificación de tipo `dias_libres` trae `pedidoId`.
 *   POST /v1/panel/dias-libres/{id}/aprobar   → { turnos: number }   (los turnos a mover)
 *   POST /v1/panel/dias-libres/{id}/rechazar  → 204
 *
 * Recordatorios automáticos de clientes (Propuesta 12, Clientes):
 *   GET /v1/panel/clientes/recordatorios      → Recordatorios
 *   PUT /v1/panel/clientes/recordatorios      { cumpleanios: boolean, vuelve: boolean } → Recordatorios
 *
 * El profesional del usuario (para «El día» de quien es profesional): `profesionalId` en cada negocio
 * de GET /v1/yo.
 */

export type Oferta = {
	id: string;
	profesionalId: string;
	inicio: string;
	fin: string;
	/** A quién se le está ofreciendo ahora. */
	anotadoId: string;
	nombre: string;
	/** Hasta cuándo tiene para tomarlo. */
	vence: string;
	/** Quiénes siguen en la cola (nombres cortos), en orden. */
	siguen: string[];
	estado: 'ofreciendo' | 'tomado' | 'vencido' | 'frenado';
};

export type Confirmacion = { via: 'whatsapp' | 'mail' | 'panel'; respuesta: 'confirmo' | 'cambiar' | null; en: string | null };

export type Sena = { monto: number; estado: 'pagada' | 'pendiente' | 'devuelta' | 'vencida'; medio: string };

export type FechaRepetida = {
	inicio: string;
	/** libre, o por qué no se puede y la alternativa más cercana. */
	estado: 'libre' | 'ocupado' | 'bloqueado';
	motivo: string | null;
	alternativa: { inicio: string; texto: string } | null;
};

export type Producto = { id: string; nombre: string; precio: number };
export type Medio = 'efectivo' | 'transferencia' | 'qr_mp' | 'tarjeta';
export const MEDIOS: { clave: Medio; texto: string; icono: string }[] = [
	{ clave: 'efectivo', texto: 'Efectivo', icono: 'payments' },
	{ clave: 'transferencia', texto: 'Transfer.', icono: 'account_balance' },
	{ clave: 'qr_mp', texto: 'QR MP', icono: 'qr_code_2' },
	{ clave: 'tarjeta', texto: 'Tarjeta', icono: 'credit_card' }
];

export type Recordatorios = {
	cumpleanios: { activo: boolean; estaSemana: number; asunto: string; texto: string; cupon: string | null };
	vuelve: { activo: boolean; estaSemana: number; asunto: string; texto: string; cupon: string | null };
};

/** Lo que el turno suma con las funciones nuevas (opcionales hasta que la API las tenga). */
export type ExtrasTurno = { sena?: Sena | null; confirmacion?: Confirmacion | null; fijo?: { cadaSemanas: number } | null };
