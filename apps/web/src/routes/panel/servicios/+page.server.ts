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
	/** La portada: siempre la primera de `fotos`. */
	foto: string | null;
	fotos: string[];
	orden: number;
	activo: boolean;
};
type Profesional = { id: string; nombre: string; servicios: string[]; activo: boolean };
/** Un producto del catálogo (docs/api/caja.md). `stock: null`: no se lleva la cuenta. */
export type Producto = { id: string; nombre: string; precio: number; stock: number | null; activo: boolean; orden: number };

export const load: PageServerLoad = async (event) => {
	const api = apiDe(event);
	const [{ servicios }, { categorias }, profesionales, productos] = await Promise.all([
		api.get<{ servicios: Servicio[] }>('/v1/panel/servicios'),
		api.get<{ categorias: Categoria[] }>('/v1/panel/categorias'),
		// Quién hace cada servicio: sólo para mostrarlo en la fila; si no se puede leer, no se muestra.
		api
			.get<{ profesionales: Profesional[] }>('/v1/panel/profesionales')
			.then((r) => r.profesionales.map((p) => ({ id: p.id, nombre: p.nombre, servicios: p.servicios ?? [] })))
			.catch(() => []),
		// El catálogo de productos (docs/api/caja.md): si no se puede leer, la carta no se muestra.
		api
			.get<{ productos: Producto[] }>('/v1/panel/productos')
			.then((r) => r.productos)
			.catch(() => null)
	]);
	return { servicios, categorias, profesionales, productos };
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
		.filter(Boolean)
		.map(encodeURIComponent);

/** El id del formulario, codificado para ir en la ruta de la API. */
const idDe = (f: FormData) => encodeURIComponent(String(f.get('id') ?? ''));

export const actions: Actions = {
	crear: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).post('/v1/panel/servicios', servicioDe(f)));
	},
	editar: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).patch(`/v1/panel/servicios/${idDe(f)}`, cambiosDe(f)));
	},
	// Fotos del servicio: el formulario manda las que ya tiene (`fotos`, en orden) y la API guarda la lista entera.
	foto: async (event) => {
		const f = await event.request.formData();
		const api = apiDe(event);
		return intentar(async () => {
			const fotos = fotosDe(f);
			const archivos = f.getAll('archivo').filter((a): a is File => a instanceof File && a.size > 0);
			if (!archivos.length) await subirImagen(event, null); // «Elegí una imagen.»
			for (const a of archivos.slice(0, MAX_FOTOS - fotos.length)) fotos.push(await subirImagen(event, a));
			return api.patch(`/v1/panel/servicios/${idDe(f)}`, { fotos });
		});
	},
	sacarFoto: async (event) => {
		const f = await event.request.formData();
		const sacar = String(f.get('url') ?? '');
		return intentar(() => apiDe(event).patch(`/v1/panel/servicios/${idDe(f)}`, { fotos: fotosDe(f).filter((u) => u !== sacar) }));
	},
	portada: async (event) => {
		const f = await event.request.formData();
		const elegida = String(f.get('url') ?? '');
		const fotos = fotosDe(f);
		const orden = fotos.includes(elegida) ? [elegida, ...fotos.filter((u) => u !== elegida)] : fotos;
		return intentar(() => apiDe(event).patch(`/v1/panel/servicios/${idDe(f)}`, { fotos: orden }));
	},
	borrar: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).borrar(`/v1/panel/servicios/${idDe(f)}`));
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
		return intentar(() => apiDe(event).patch(`/v1/panel/categorias/${idDe(f)}`, { nombre: String(f.get('nombre') ?? '') }));
	},
	borrarCategoria: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).borrar(`/v1/panel/categorias/${idDe(f)}`));
	},

	// Productos (docs/api/caja.md): POST/PATCH/DELETE /v1/panel/productos.
	crearProducto: async (event) => {
		const f = await event.request.formData();
		const stock = numero(f.get('stock'));
		return intentar(() =>
			apiDe(event).post('/v1/panel/productos', {
				nombre: String(f.get('nombre') ?? '').trim(),
				precio: precio(f.get('precio')) ?? 0,
				...(stock === undefined ? {} : { stock }),
				activo: true,
				orden: numero(f.get('orden')) ?? 0
			})
		);
	},
	editarProducto: async (event) => {
		const f = await event.request.formData();
		const c: Record<string, unknown> = {};
		if (f.has('nombre')) c.nombre = String(f.get('nombre') ?? '').trim();
		if (f.has('precio')) c.precio = precio(f.get('precio')) ?? 0;
		if (f.has('stock')) {
			// Vacío: deja de llevar la cuenta del stock.
			const stock = numero(f.get('stock'));
			if (stock === undefined) c.sinStock = true;
			else c.stock = stock;
		}
		if (f.has('conActivo')) c.activo = f.get('activo') === 'on';
		return intentar(() => apiDe(event).patch(`/v1/panel/productos/${idDe(f)}`, c));
	},
	borrarProducto: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).borrar(`/v1/panel/productos/${idDe(f)}`));
	},
	ordenarProductos: async (event) => {
		const f = await event.request.formData();
		const api = apiDe(event);
		return intentar(async () => {
			for (const [i, id] of ids(f).entries()) await api.patch(`/v1/panel/productos/${id}`, { orden: i + 1 });
			return null;
		});
	}
};

/** Cuántas fotos puede tener un servicio (catalogo.MaxFotosServicio). */
const MAX_FOTOS = 5;

function fotosDe(f: FormData): string[] {
	return f.getAll('fotos').map(String).filter(Boolean);
}
