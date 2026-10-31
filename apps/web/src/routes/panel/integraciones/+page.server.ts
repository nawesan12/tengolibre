import { apiDe, apiPublica, ErrorApi } from '#lib/server/api.ts';
import { accion } from '#lib/configuracion/acciones.server.ts';
import { ASISTENTES, PERMISOS, RUTAS_IA, type AsistenteIA, type Permisos } from '#lib/gestion/api-integraciones.ts';
import { ZONA } from '#lib/formato.ts';
import type { Actions, PageServerLoad } from './$types';

type Negocio = { slug: string; nombre: string; whatsapp: string; instagram: string };
type Conexion = { disponible: boolean; conectada: boolean; cuenta?: string };
type Calendario = { id: string | null; profesionalId: string | null; nombre: string; url: string | null };
type Profesional = { id: string; nombre: string; activo: boolean };

/** Lo que no existe todavía en la API (404/501) se muestra como «sin activar» en vez de romper. */
async function opcional<T>(p: Promise<T>): Promise<T | null> {
	try {
		return await p;
	} catch (e) {
		if (e instanceof ErrorApi && [403, 404, 405, 501].includes(e.estado)) return null;
		throw e;
	}
}

/** El día de mañana en Argentina (AAAA-MM-DD) y su nombre («martes 3»). */
function manana() {
	const d = new Date(Date.now() + 86_400_000);
	const fecha = new Intl.DateTimeFormat('en-CA', { timeZone: ZONA }).format(d);
	const nombre = new Intl.DateTimeFormat('es-AR', { timeZone: ZONA, weekday: 'long', day: 'numeric' }).format(d);
	return { fecha, nombre };
}

/**
 * El ejemplo de «Así se usa» con los huecos reales de mañana a la tarde: el primer servicio y hasta
 * tres horarios libres de profesionales distintos. Sin huecos, el ejemplo no lleva horarios.
 */
async function huecosDeManana(fetch: typeof globalThis.fetch, slug: string, profesionales: Profesional[]) {
	const m = manana();
	try {
		const pub = await apiPublica(fetch).get<{ servicios: { id: string }[] }>(`/v1/p/${slug}`);
		const servicio = pub.servicios[0]?.id;
		if (!servicio) return { dia: m.nombre, huecos: [] };
		const { horarios } = await apiPublica(fetch).get<{ horarios: { hora: string; profesionales: string[] }[] }>(
			`/v1/p/${slug}/libres?servicio=${servicio}&fecha=${m.fecha}`
		);
		const nombres = new Map(profesionales.map((p) => [p.id, p.nombre.split(' ')[0]]));
		const usados = new Set<string>();
		const huecos: { hora: string; quien: string }[] = [];
		for (const h of horarios ?? []) {
			if (h.hora < '14:00') continue;
			const p = h.profesionales.find((x) => !usados.has(x)) ?? h.profesionales[0];
			if (!p || usados.has(p)) continue;
			usados.add(p);
			huecos.push({ hora: h.hora, quien: nombres.get(p) ?? '' });
			if (huecos.length === 3) break;
		}
		return { dia: m.nombre, huecos };
	} catch {
		return { dia: m.nombre, huecos: [] };
	}
}

export const load: PageServerLoad = async (event) => {
	const api = apiDe(event);
	const [negocio, mp, calendarios, profesionales, ia] = await Promise.all([
		api.get<Negocio>('/v1/panel/negocio'),
		opcional(api.get<Conexion>('/v1/panel/mercadopago')),
		opcional(api.get<{ calendarios: Calendario[] }>('/v1/panel/calendarios').then((r) => r.calendarios)),
		api.get<{ profesionales: Profesional[] }>('/v1/panel/profesionales').then((r) => r.profesionales.filter((p) => p.activo)),
		opcional(api.get<AsistenteIA>(RUTAS_IA.leer))
	]);
	const ejemplo = await huecosDeManana(event.fetch, negocio.slug, profesionales);
	return {
		negocio: { slug: negocio.slug, nombre: negocio.nombre, whatsapp: negocio.whatsapp, instagram: negocio.instagram },
		mp,
		calendarios: (calendarios ?? []).filter((c) => c.url).length,
		ia,
		ejemplo
	};
};

export const actions: Actions = {
	activar: async (event) => accion('ia', 'Listo: tu asistente ya tiene su dirección.', () => apiDe(event).post(RUTAS_IA.activar)),
	asistente: async (event) => {
		const f = await event.request.formData();
		const asistente = String(f.get('asistente'));
		return accion('ia', '', () =>
			apiDe(event).put(RUTAS_IA.asistente, { asistente: (ASISTENTES as readonly string[]).includes(asistente) ? asistente : 'otro' })
		);
	},
	permisos: async (event) => {
		const f = await event.request.formData();
		const permisos = Object.fromEntries(PERMISOS.map((p) => [p, f.get(p) === 'on'])) as Permisos;
		return accion('permisos', 'Guardamos qué puede hacer.', () => apiDe(event).put(RUTAS_IA.permisos, { permisos }));
	},
	publico: async (event) => {
		const f = await event.request.formData();
		const activo = f.get('activo') === 'on';
		return accion('publico', activo ? 'Los asistentes de tus clientes ya te encuentran.' : 'Apagamos la reserva por IA.', () =>
			apiDe(event).put(RUTAS_IA.publico, { activo })
		);
	},
	clave: async (event) =>
		accion('clave', 'Copiá la clave ahora: no se vuelve a mostrar.', () => apiDe(event).post<{ clave: string }>(RUTAS_IA.clave))
};
