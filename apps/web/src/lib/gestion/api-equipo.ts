/**
 * Contratos SUPUESTOS de equipo que la API todavía no tiene (los usa la UI del handoff v2). Al
 * integrar, ajustar acá las rutas y formas; las pantallas no cambian.
 *
 * Pedidos de días libres de un profesional («Mi horario» → «Pedir días libres»):
 *   GET  /v1/panel/profesionales/{id}/pedidos-libres
 *        → { pedidos: PedidoLibre[] }   (los de hoy en adelante, más nuevos primero)
 *   POST /v1/panel/profesionales/{id}/pedidos-libres
 *        { desde: 'AAAA-MM-DD', hasta: 'AAAA-MM-DD', motivo: string } → PedidoLibre (201)
 *   Quien gestiona lo aprueba (y se convierte en bloqueos) o lo rechaza:
 *   PATCH /v1/panel/profesionales/{id}/pedidos-libres/{pedido} { estado: 'aprobado' | 'rechazado' }
 */
import type { Api } from '#lib/server/api.ts';
import { ErrorApi } from '#lib/server/api.ts';

export type PedidoLibre = {
	id: string;
	desde: string;
	hasta: string;
	motivo: string;
	estado: 'pendiente' | 'aprobado' | 'rechazado';
	/** Nombre de quien lo aprobó o rechazó, si ya se resolvió. */
	resolvio?: string | null;
	creadoEn: string;
};

const base = (id: string) => `/v1/panel/profesionales/${id}/pedidos-libres`;

/** Los pedidos; mientras la API no lo tenga (404/405/501), lista vacía en vez de romper. */
export async function pedidosLibres(api: Api, profesional: string): Promise<PedidoLibre[]> {
	try {
		return (await api.get<{ pedidos: PedidoLibre[] }>(base(profesional))).pedidos ?? [];
	} catch (e) {
		if (e instanceof ErrorApi && e.estado < 500) return [];
		throw e;
	}
}

export const pedirLibres = (api: Api, profesional: string, cuerpo: { desde: string; hasta: string; motivo: string }) =>
	api.post<PedidoLibre>(base(profesional), cuerpo);

export const resolverPedido = (api: Api, profesional: string, pedido: string, estado: 'aprobado' | 'rechazado') =>
	api.patch(`${base(profesional)}/${pedido}`, { estado });
