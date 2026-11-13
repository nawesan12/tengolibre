import { error, redirect } from '@sveltejs/kit';
import { apiDe, ErrorApi } from '#lib/server/api.ts';
import { intentar, numero } from '#lib/server/acciones.ts';
import { cambiarEstado, enc } from '#lib/operacion/servidor.ts';
import type { EtiquetaConUso, Ficha, Nota, Penalidad } from '#lib/operacion/tipos.ts';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	event.depends('panel:clientes', 'panel:turnos');
	const api = apiDe(event);
	const id = enc(event.params.id);
	try {
		const [ficha, { notas }, { etiquetas }, { penalidades }] = await Promise.all([
			api.get<Ficha>(`/v1/panel/clientes/${id}`),
			api.get<{ notas: Nota[] }>(`/v1/panel/clientes/${id}/notas`),
			api.get<{ etiquetas: EtiquetaConUso[] }>('/v1/panel/etiquetas'),
			api.get<{ penalidades: Penalidad[] }>(`/v1/panel/clientes/${id}/penalidades`)
		]);
		return { ficha, notas, etiquetas, penalidades };
	} catch (e) {
		if (e instanceof ErrorApi && (e.estado === 404 || e.estado === 400)) error(404, 'No encontramos ese cliente.');
		throw e;
	}
};

const base = (event: { params: { id: string } }) => `/v1/panel/clientes/${enc(event.params.id)}`;

export const actions: Actions = {
	editar: async (event) => {
		const f = await event.request.formData();
		return intentar(() =>
			apiDe(event).patch(base(event), {
				nombre: String(f.get('nombre') ?? ''),
				telefono: String(f.get('telefono') ?? ''),
				email: String(f.get('email') ?? '')
			})
		);
	},
	/** Pedido de supresión: sin turnos se borra; con turnos pasados queda anónimo (la API decide). */
	suprimir: async (event) => {
		const r = await intentar(() => apiDe(event).borrar<{ resultado: 'borrado' | 'anonimizado' }>(base(event)));
		if ('ok' in r && r.ok) redirect(303, `/panel/clientes?suprimido=${r.datos.resultado}`);
		return r;
	},
	etiqueta: async (event) => {
		const f = await event.request.formData();
		const ruta = `${base(event)}/etiquetas/${enc(f.get('etiqueta'))}`;
		return intentar(() => (f.get('quitar') ? apiDe(event).borrar(ruta) : apiDe(event).put(ruta, {})));
	},
	nota: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).post(`${base(event)}/notas`, { texto: String(f.get('texto') ?? '') }));
	},
	borrarNota: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).borrar(`${base(event)}/notas/${enc(f.get('nota'))}`));
	},
	penalizar: async (event) => {
		const f = await event.request.formData();
		return intentar(() =>
			apiDe(event).post(`${base(event)}/penalidades`, { dias: numero(f.get('dias')) ?? 0, nota: String(f.get('nota') ?? '') })
		);
	},
	levantar: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).post(`/v1/panel/penalidades/${enc(f.get('id'))}/levantar`));
	},
	estado: cambiarEstado
};
