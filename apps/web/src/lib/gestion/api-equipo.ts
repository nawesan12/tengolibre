/**
 * Pedidos de días libres de un profesional (docs/api/dias-libres.md):
 *   GET  /v1/panel/dias-libres?estado=pendiente,aprobado → { pedidos: PedidoLibre[] }
 *   POST /v1/panel/dias-libres { desde, hasta?, motivo, nota? } → PedidoLibre (sólo quien tiene un profesional vinculado)
 *   POST /v1/panel/dias-libres/{id}/aprobar  { respuesta? } → { pedido, reprogramar }
 *   POST /v1/panel/dias-libres/{id}/rechazar { respuesta? } → PedidoLibre
 *   POST /v1/panel/dias-libres/{id}/cancelar → PedidoLibre
 *   GET  /v1/panel/dias-libres/turnos?desde&hasta&profesional → { cantidad, turnos }
 */
import type { Api } from '#lib/server/api.ts';
import { ErrorApi } from '#lib/server/api.ts';

import type { MotivoLibre, PedidoLibre } from './dias-libres.ts';
export { MOTIVOS_LIBRES, NOMBRE_MOTIVO, type MotivoLibre, type PedidoLibre } from './dias-libres.ts';

/** Los pedidos pendientes de un profesional. Si el rol no puede verlos, lista vacía. */
export async function pedidosLibres(api: Api, profesional: string): Promise<PedidoLibre[]> {
	try {
		const { pedidos } = await api.get<{ pedidos: PedidoLibre[] }>('/v1/panel/dias-libres?estado=pendiente');
		return (pedidos ?? []).filter((p) => p.profesionalId === profesional);
	} catch (e) {
		if (e instanceof ErrorApi && e.estado < 500) return [];
		throw e;
	}
}

export const pedirLibres = (api: Api, cuerpo: { desde: string; hasta: string; motivo: MotivoLibre; nota: string }) =>
	api.post<PedidoLibre>('/v1/panel/dias-libres', cuerpo);

export const resolverPedido = (api: Api, pedido: string, que: 'aprobar' | 'rechazar' | 'cancelar') =>
	api.post(`/v1/panel/dias-libres/${pedido}/${que}`, {});
