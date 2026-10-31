import { apiDe } from '#lib/server/api.ts';
import { intentar } from '#lib/server/acciones.ts';
import { enc } from '#lib/operacion/servidor.ts';
import type { EtiquetaConUso } from '#lib/operacion/tipos.ts';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	event.depends('panel:clientes');
	const { etiquetas } = await apiDe(event).get<{ etiquetas: EtiquetaConUso[] }>('/v1/panel/etiquetas');
	return { etiquetas };
};

const datos = (f: FormData) => ({ nombre: String(f.get('nombre') ?? ''), color: String(f.get('color') ?? '') });

export const actions: Actions = {
	crear: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).post('/v1/panel/etiquetas', datos(f)));
	},
	editar: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).patch(`/v1/panel/etiquetas/${enc(f.get('id'))}`, datos(f)));
	},
	borrar: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).borrar(`/v1/panel/etiquetas/${enc(f.get('id'))}`));
	}
};
