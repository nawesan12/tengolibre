import { apiDe } from '#lib/server/api.ts';
import { intentar } from '#lib/server/acciones.ts';
import type { Notificacion } from '#lib/operacion/tipos.ts';
import type { Actions, PageServerLoad } from './$types';

/** La campanita: lo que pasó en los últimos siete días; las nuevas, marcadas. */
export const load: PageServerLoad = async (event) => {
	event.depends('panel:notificaciones');
	return apiDe(event).get<{ notificaciones: Notificacion[]; sinLeer: number }>('/v1/panel/campanita');
};

export const actions: Actions = {
	vista: async (event) => intentar(() => apiDe(event).post('/v1/panel/campanita/vista'))
};
