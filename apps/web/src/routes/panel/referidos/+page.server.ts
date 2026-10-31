import { apiDe } from '#lib/server/api.ts';
import { numero } from '#lib/server/acciones.ts';
import { accion } from '#lib/configuracion/acciones.server.ts';
import type { Actions, PageServerLoad } from './$types';

export type AjustesReferidos = { activo: boolean; tipo: string | null; valor: number | null; tope: number | null };
export type Referido = { id: string; codigo: string; cliente: string; usos: number; creadoEn: string };

export const load: PageServerLoad = async (event) => {
	const api = apiDe(event);
	const [config, { referidos }] = await Promise.all([
		api.get<AjustesReferidos>('/v1/panel/referidos/config'),
		api.get<{ referidos: Referido[] }>('/v1/panel/referidos')
	]);
	return { config, referidos };
};

export const actions: Actions = {
	config: async (event) => {
		const f = await event.request.formData();
		const conPremio = f.get('premio') === 'on';
		const tope = numero(f.get('tope'));
		return accion('config', 'Guardamos el programa de referidos.', () =>
			apiDe(event).put('/v1/panel/referidos/config', {
				activo: f.get('activo') === 'on',
				tipo: conPremio ? String(f.get('tipo') ?? 'porcentaje') : null,
				valor: conPremio ? (numero(f.get('valor')) ?? 0) : null,
				tope: tope && tope > 0 ? tope : null
			})
		);
	}
};
