import { apiDe } from '#lib/server/api.ts';
import { accion, siPuede } from '#lib/configuracion/acciones.server.ts';
import { estiloDeForm, type Estilo } from '#lib/gestion/plantillas.ts';
import type { Actions, PageServerLoad } from './$types';

type Web = { portada: string | null; colorPrimario: string; colorAcento: string; rubro: string; estilo: Estilo };
type Negocio = { slug: string; nombre: string; ciudad: string; logo: string | null };
type Servicio = { nombre: string; precio: number; activo: boolean; orden: number };

/**
 * Plantillas en vivo: lo publicado viene de la API (`estilo` de mi-web) y el borrador vive en la
 * página. Las miniaturas se arman con lo real del negocio: nombre, colores, servicios y fotos.
 */
export const load: PageServerLoad = async (event) => {
	const api = apiDe(event);
	const [negocio, web, servicios, galeria, resenas] = await Promise.all([
		api.get<Negocio>('/v1/panel/negocio'),
		api.get<Web>('/v1/panel/mi-web'),
		api.get<{ servicios: Servicio[] }>('/v1/panel/servicios').then((r) => r.servicios),
		api.get<{ fotos: { url: string }[] }>('/v1/panel/mi-web/galeria').then((r) => r.fotos),
		siPuede(api.get<{ promedio: number; cantidad: number }>('/v1/panel/resenas?pagina=0')).catch(() => null)
	]);
	const fotos = [web.portada, ...galeria.map((f) => f.url)].filter((u): u is string => !!u);
	return {
		negocio: { slug: negocio.slug, nombre: negocio.nombre, ciudad: negocio.ciudad, logo: negocio.logo },
		colores: { primario: web.colorPrimario || '#183c44', acento: web.colorAcento || '#3ccc90' },
		rubro: web.rubro,
		publicado: web.estilo,
		// Las secciones son un contrato supuesto: hasta que la API las guarde no se publican.
		conSecciones: Array.isArray(web.estilo.secciones),
		servicios: servicios
			.filter((s) => s.activo)
			.sort((a, b) => a.orden - b.orden)
			.slice(0, 3)
			.map((s) => ({ nombre: s.nombre, precio: s.precio })),
		fotos: fotos.slice(0, 4),
		resenas: resenas && resenas.cantidad > 0 ? { promedio: resenas.promedio, cantidad: resenas.cantidad } : null
	};
};

export const actions: Actions = {
	publicar: async (event) => {
		const f = await event.request.formData();
		const estilo = estiloDeForm(f);
		const api = apiDe(event);
		const actual = await api.get<Web>('/v1/panel/mi-web').catch(() => null);
		if (!Array.isArray(actual?.estilo.secciones)) delete estilo.secciones;
		return accion('publicar', 'Publicada. Tus clientes ya la ven.', () => api.patch('/v1/panel/mi-web', { estilo }));
	}
};
