import { redirect } from '@sveltejs/kit';
import { apiDe } from '#lib/server/api.ts';
import { intentar } from '#lib/server/acciones.ts';
import { param } from '#lib/operacion/servidor.ts';
import type { Cliente, EtiquetaConUso } from '#lib/operacion/tipos.ts';
import type { Actions, PageServerLoad } from './$types';

const ORDENES = ['nombre', 'reciente', 'ultima', 'gastado', 'visitas'];

export const load: PageServerLoad = async (event) => {
	event.depends('panel:clientes', 'panel:turnos');
	const { url } = event;
	const filtros = {
		q: param(url, 'q'),
		etiqueta: param(url, 'etiqueta'),
		penalizados: param(url, 'penalizados') === '1',
		orden: ORDENES.includes(param(url, 'orden')) ? param(url, 'orden') : 'nombre',
		pagina: Math.max(1, Number(param(url, 'pagina')) || 1)
	};
	const q = new URLSearchParams({ orden: filtros.orden, pagina: String(filtros.pagina), porPagina: '20' });
	if (filtros.q) q.set('q', filtros.q);
	if (filtros.etiqueta) q.set('etiqueta', filtros.etiqueta);
	if (filtros.penalizados) q.set('penalizados', '1');
	const api = apiDe(event);
	const [lista, { etiquetas }] = await Promise.all([
		api.get<{ clientes: Cliente[]; total: number; pagina: number; porPagina: number }>(`/v1/panel/clientes?${q}`),
		api.get<{ etiquetas: EtiquetaConUso[] }>('/v1/panel/etiquetas')
	]);
	return { ...lista, etiquetas, filtros };
};

export const actions: Actions = {
	crear: async (event) => {
		const f = await event.request.formData();
		const r = await intentar(() =>
			apiDe(event).post<Cliente>('/v1/panel/clientes', {
				nombre: String(f.get('nombre') ?? ''),
				telefono: String(f.get('telefono') ?? ''),
				email: String(f.get('email') ?? '')
			})
		);
		if ('datos' in r && r.ok) redirect(303, `/panel/clientes/${r.datos.id}`);
		return r;
	}
};
