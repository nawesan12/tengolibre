import { apiDe } from '#lib/server/api.ts';
import { intentar, numero } from '#lib/server/acciones.ts';
import type { Actions, PageServerLoad } from './$types';

export type Categoria = { id: string; nombre: string; orden: number };
export type Servicio = {
	id: string;
	categoriaId: string | null;
	nombre: string;
	descripcion: string;
	duracion: number;
	precio: number;
	foto: string | null;
	orden: number;
	activo: boolean;
};

export const load: PageServerLoad = async (event) => {
	const api = apiDe(event);
	const [{ servicios }, { categorias }] = await Promise.all([
		api.get<{ servicios: Servicio[] }>('/v1/panel/servicios'),
		api.get<{ categorias: Categoria[] }>('/v1/panel/categorias')
	]);
	return { servicios, categorias };
};

// Lo que viene del formulario de un servicio, como lo pide la API.
function servicioDe(f: FormData) {
	return {
		nombre: String(f.get('nombre') ?? ''),
		descripcion: String(f.get('descripcion') ?? ''),
		duracion: numero(f.get('duracion')) ?? 0,
		precio: numero(f.get('precio')) ?? 0,
		categoriaId: String(f.get('categoriaId') ?? '') || null,
		activo: f.get('activo') === 'on'
	};
}

export const actions: Actions = {
	crear: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).post('/v1/panel/servicios', servicioDe(f)));
	},
	editar: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).patch(`/v1/panel/servicios/${f.get('id')}`, servicioDe(f)));
	},
	borrar: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).borrar(`/v1/panel/servicios/${f.get('id')}`));
	},
	categoria: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).post('/v1/panel/categorias', { nombre: String(f.get('nombre') ?? '') }));
	},
	borrarCategoria: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).borrar(`/v1/panel/categorias/${f.get('id')}`));
	}
};
