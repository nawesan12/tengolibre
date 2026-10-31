import { apiDe } from '#lib/server/api.ts';
import { intentar } from '#lib/server/acciones.ts';
import { enc, param } from '#lib/operacion/servidor.ts';
import { opcional, type DistribucionResenas } from '#lib/gestion/api-reportes.ts';
import type { Resena } from '#lib/operacion/tipos.ts';
import type { AjustesReferidos } from '../referidos/+page.server.ts';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	event.depends('panel:resenas');
	const { url } = event;
	const visible = ['si', 'no'].includes(param(url, 'visible')) ? param(url, 'visible') : '';
	const sinResponder = param(url, 'sinResponder') === '1';
	const pagina = Math.max(0, Number(param(url, 'pagina')) || 0);
	const q = new URLSearchParams({ pagina: String(pagina) });
	if (visible) q.set('visible', visible);
	if (sinResponder) q.set('sinResponder', '1');
	const api = apiDe(event);
	// Al lado, como en el diseño: cuándo se pide la reseña y el programa de referidos (sólo gestión).
	const [r, avisos, referidos] = await Promise.all([
		api.get<{ promedio: number; cantidad: number; resenas: Resena[]; distribucion?: DistribucionResenas }>(`/v1/panel/resenas?${q}`),
		opcional(api.get<{ resenaDemoraHoras: number }>('/v1/panel/ajustes/avisos')),
		opcional(api.get<AjustesReferidos>('/v1/panel/referidos/config'))
	]);
	return { ...r, demora: avisos?.resenaDemoraHoras ?? null, referidos, filtros: { visible, sinResponder, pagina } };
};

export const actions: Actions = {
	visibilidad: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).patch(`/v1/panel/resenas/${enc(f.get('id'))}`, { visible: f.get('visible') === '1' }));
	},
	responder: async (event) => {
		const f = await event.request.formData();
		return intentar(() =>
			apiDe(event).patch(`/v1/panel/resenas/${enc(f.get('id'))}`, { respuesta: String(f.get('respuesta') ?? '') })
		);
	}
};
