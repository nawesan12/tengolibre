import { apiDe } from '#lib/server/api.ts';
import { accion } from '#lib/configuracion/acciones.server.ts';
import type { Actions, PageServerLoad } from './$types';

export type Calendario = { id: string | null; profesionalId: string | null; nombre: string; url: string | null };

/** Los calendarios para suscribirse: el del negocio y uno por profesional (o sólo el propio). */
export const load: PageServerLoad = async (event) => {
	const { calendarios } = await apiDe(event).get<{ calendarios: Calendario[] }>('/v1/panel/calendarios');
	return { calendarios };
};

export const actions: Actions = {
	crear: async (event) => {
		const f = await event.request.formData();
		const profesionalId = String(f.get('profesionalId') ?? '');
		const nuevo = f.get('nuevo') === '1';
		return accion(`cal-${profesionalId}`, nuevo ? 'Listo: el link anterior dejó de andar.' : 'Listo, ya podés agregarlo.', () =>
			apiDe(event).post('/v1/panel/calendarios', { profesionalId })
		);
	},
	borrar: async (event) => {
		const f = await event.request.formData();
		return accion(`cal-${String(f.get('profesionalId') ?? '')}`, 'Desactivamos el link.', () =>
			apiDe(event).borrar(`/v1/panel/calendarios/${String(f.get('id') ?? '')}`)
		);
	}
};
