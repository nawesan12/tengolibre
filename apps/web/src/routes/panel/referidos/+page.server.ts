import { apiDe } from '#lib/server/api.ts';
import { numero } from '#lib/server/acciones.ts';
import { accion } from '#lib/configuracion/acciones.server.ts';
import { usosReferidos } from '#lib/gestion/api-reportes.ts';
import type { Actions, PageServerLoad } from './$types';

export type AjustesReferidos = { activo: boolean; tipo: string | null; valor: number | null; tope: number | null };
export type Referido = { id: string; codigo: string; cliente: string; usos: number; creadoEn: string };

export const load: PageServerLoad = async (event) => {
	const api = apiDe(event);
	const [config, { referidos }, usos] = await Promise.all([
		api.get<AjustesReferidos>('/v1/panel/referidos/config'),
		api.get<{ referidos: Referido[] }>('/v1/panel/referidos'),
		// Quién trajo a quién: contrato supuesto (src/lib/gestion/api-reportes.ts); sin él, no se muestra.
		usosReferidos(api).catch(() => null)
	]);
	return { config, referidos, usos };
};

export const actions: Actions = {
	config: async (event) => {
		const f = await event.request.formData();
		// El premio: «porcentaje» o «fijo»; sin tipo (o con el viejo tilde apagado), no hay premio.
		const tipo = String(f.get('tipo') ?? '');
		const conPremio = (tipo === 'porcentaje' || tipo === 'fijo') && f.get('premio') !== 'off';
		const tope = numero(f.get('tope'));
		return accion('config', 'Guardamos el programa de referidos.', () =>
			apiDe(event).put('/v1/panel/referidos/config', {
				activo: f.get('activo') === 'on',
				tipo: conPremio ? tipo : null,
				valor: conPremio ? (numero(f.get('valor')) ?? 0) : null,
				tope: tope && tope > 0 ? tope : null
			})
		);
	}
};
