import { fail } from '@sveltejs/kit';
import { apiDe, apiPublica, ErrorApi } from '#lib/server/api.ts';
import { accion } from '#lib/configuracion/acciones.server.ts';
import { RUTA_IA_PUBLICO, RUTAS_MCP, type AccionMcp, type IaPublica, type Mcp, type Token } from '#lib/gestion/api-integraciones.ts';
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
	event.depends('panel:mcp_registro');
	const api = apiDe(event);
	const [negocio, mp, calendarios, profesionales, mcp, registro, publico] = await Promise.all([
		api.get<Negocio>('/v1/panel/negocio'),
		opcional(api.get<Conexion>('/v1/panel/mercadopago')),
		opcional(api.get<{ calendarios: Calendario[] }>('/v1/panel/calendarios').then((r) => r.calendarios)),
		api.get<{ profesionales: Profesional[] }>('/v1/panel/profesionales').then((r) => r.profesionales.filter((p) => p.activo)),
		opcional(api.get<Mcp>(RUTAS_MCP.leer)),
		opcional(api.get<{ acciones: AccionMcp[] }>(`${RUTAS_MCP.registro}?limite=5`).then((r) => r.acciones)),
		opcional(api.get<IaPublica>(RUTA_IA_PUBLICO))
	]);
	const ejemplo = await huecosDeManana(event.fetch, negocio.slug, profesionales);
	// La clave que se mira: la de ?clave=<id>, o la primera.
	const elegida = mcp?.tokens.find((t) => t.id === event.url.searchParams.get('clave')) ?? mcp?.tokens[0] ?? null;
	return {
		negocio: { slug: negocio.slug, nombre: negocio.nombre, whatsapp: negocio.whatsapp, instagram: negocio.instagram },
		mp,
		calendarios: (calendarios ?? []).filter((c) => c.url).length,
		mcp,
		elegida,
		registro: registro ?? [],
		publico,
		ejemplo
	};
};

const texto = (f: FormData, k: string) => String(f.get(k) ?? '').trim();

export const actions: Actions = {
	/** Crea la clave de un asistente (con los permisos por defecto) y la muestra una sola vez. */
	crear: async (event) => {
		const f = await event.request.formData();
		const nombre = texto(f, 'nombre').slice(0, 60) || 'Mi asistente';
		return accion('clave', 'Copiá la clave ahora: no se vuelve a mostrar.', () =>
			apiDe(event).post<{ token: Token; clave: string; url: string }>(RUTAS_MCP.tokens, { nombre })
		);
	},
	/** Una clave nueva para el mismo asistente (mismo nombre y permisos); la vieja deja de andar. */
	regenerar: async (event) => {
		const f = await event.request.formData();
		const api = apiDe(event);
		const id = texto(f, 'id');
		const permisos = f.getAll('permiso').map(String);
		return accion('clave', 'Copiá la clave nueva ahora: la anterior dejó de andar.', async () => {
			const r = await api.post<{ token: Token; clave: string; url: string }>(RUTAS_MCP.tokens, { nombre: texto(f, 'nombre'), permisos });
			await api.borrar(`${RUTAS_MCP.tokens}/${id}`);
			return r;
		});
	},
	permisos: async (event) => {
		const f = await event.request.formData();
		const id = texto(f, 'id');
		if (!id) return fail(400, { que: 'permisos', error: 'Elegí un asistente.', codigo: 'invalido', campos: {} });
		const permisos = f.getAll('permiso').map(String);
		return accion('permisos', 'Guardamos qué puede hacer.', () => apiDe(event).patch(`${RUTAS_MCP.tokens}/${id}`, { permisos }));
	},
	revocar: async (event) => {
		const f = await event.request.formData();
		return accion('clave', 'Desconectamos ese asistente: su clave dejó de andar.', () =>
			apiDe(event).borrar(`${RUTAS_MCP.tokens}/${texto(f, 'id')}`)
		);
	},
	publico: async (event) => {
		const f = await event.request.formData();
		const activo = f.get('activo') === 'on';
		return accion('publico', activo ? 'Los asistentes de tus clientes ya te encuentran.' : 'Apagamos la reserva por IA.', () =>
			apiDe(event).put(RUTA_IA_PUBLICO, { activo })
		);
	}
};
