import { apiDe } from '#lib/server/api.ts';
import { intentar, numero } from '#lib/server/acciones.ts';
import { subirImagen } from '#lib/configuracion/acciones.server.ts';
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
type Profesional = { id: string; nombre: string; servicios: string[]; activo: boolean };

export const load: PageServerLoad = async (event) => {
	const api = apiDe(event);
	const [{ servicios }, { categorias }, profesionales] = await Promise.all([
		api.get<{ servicios: Servicio[] }>('/v1/panel/servicios'),
		api.get<{ categorias: Categoria[] }>('/v1/panel/categorias'),
		// Quién hace cada servicio: sólo para mostrarlo en la fila; si no se puede leer, no se muestra.
		api
			.get<{ profesionales: Profesional[] }>('/v1/panel/profesionales')
			.then((r) => r.profesionales.map((p) => ({ id: p.id, nombre: p.nombre, servicios: p.servicios ?? [] })))
			.catch(() => [])
	]);
	return { servicios, categorias, profesionales };
};

// Lo que viene del formulario de un servicio nuevo, como lo pide la API.
function servicioDe(f: FormData) {
	return {
		nombre: String(f.get('nombre') ?? ''),
		descripcion: String(f.get('descripcion') ?? ''),
		duracion: numero(f.get('duracion')) ?? 30,
		precio: precio(f.get('precio')) ?? 0,
		categoriaId: String(f.get('categoriaId') ?? '') || null,
		activo: f.get('activo') === 'on'
	};
}

/**
 * Lo que se editó en una fila: la API lee el servicio, lo pisa con lo que vino y lo valida entero,
 * así que se manda sólo lo que trae el formulario (la fila no pisa la descripción, ni el detalle la
 * duración).
 */
function cambiosDe(f: FormData) {
	const c: Record<string, unknown> = {};
	if (f.has('nombre')) c.nombre = String(f.get('nombre') ?? '');
	if (f.has('duracion')) c.duracion = numero(f.get('duracion')) ?? 0;
	if (f.has('precio')) c.precio = precio(f.get('precio')) ?? 0;
	if (f.has('descripcion')) c.descripcion = String(f.get('descripcion') ?? '');
	if (f.has('categoriaId')) c.categoriaId = String(f.get('categoriaId') ?? '') || null;
	if (f.has('conActivo')) c.activo = f.get('activo') === 'on';
	return c;
}

// El precio se escribe con puntos de miles («22.000»): se guardan sólo los dígitos.
const precio = (v: FormDataEntryValue | null) => numero(v === null ? null : String(v).replace(/\./g, ''));

const ids = (f: FormData) =>
	String(f.get('ids') ?? '')
		.split(',')
		.filter(Boolean);

export const actions: Actions = {
	crear: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).post('/v1/panel/servicios', servicioDe(f)));
	},
	editar: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).patch(`/v1/panel/servicios/${f.get('id')}`, cambiosDe(f)));
	},
	foto: async (event) => {
		const f = await event.request.formData();
		const api = apiDe(event);
		return intentar(async () => {
			const url = await subirImagen(event, f.get('archivo'));
			return api.patch(`/v1/panel/servicios/${f.get('id')}`, { foto: url });
		});
	},
	sacarFoto: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).patch(`/v1/panel/servicios/${f.get('id')}`, { foto: null }));
	},
	borrar: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).borrar(`/v1/panel/servicios/${f.get('id')}`));
	},
	// El orden nuevo de los servicios de una categoría (o de las categorías): uno por uno, con su lugar.
	ordenar: async (event) => {
		const f = await event.request.formData();
		const api = apiDe(event);
		const que = f.get('que') === 'categorias' ? 'categorias' : 'servicios';
		return intentar(async () => {
			for (const [i, id] of ids(f).entries()) await api.patch(`/v1/panel/${que}/${id}`, { orden: i + 1 });
			return null;
		});
	},
	categoria: async (event) => {
		const f = await event.request.formData();
		return intentar(() =>
			apiDe(event).post('/v1/panel/categorias', { nombre: String(f.get('nombre') ?? ''), orden: numero(f.get('orden')) ?? 0 })
		);
	},
	renombrarCategoria: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).patch(`/v1/panel/categorias/${f.get('id')}`, { nombre: String(f.get('nombre') ?? '') }));
	},
	borrarCategoria: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).borrar(`/v1/panel/categorias/${f.get('id')}`));
	}
};
