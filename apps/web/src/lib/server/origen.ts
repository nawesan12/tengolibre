import type { Cookies } from '@sveltejs/kit';

/**
 * De dónde viene quien crea su negocio: los utm del link con el que llegó a /empezar (una sugerencia
 * de otra app de Grupo Vimo, un anuncio, un QR). Va en una cookie de 90 días y se manda con el alta
 * (POST /v1/negocios), que lo guarda en el negocio. Lo mismo que lib/server/origen.ts de Tu Local Web,
 * sin ref ni cupón (Tengo Libre no tiene partners).
 */
export const COOKIE_ORIGEN = 'tl-origen';

export type Origen = { fuente?: string; medio?: string; campana?: string; contenido?: string; llego?: string };

const CAMPOS = ['fuente', 'medio', 'campana', 'contenido'] as const;
const valido = (v: unknown) => (typeof v === 'string' && /^[A-Za-z0-9_.-]{1,40}$/.test(v) ? v : undefined);

/** Los utm de la URL, o null si no trae ninguno válido. */
export function origenDe(url: URL, ahora = new Date()): Origen | null {
	const q = url.searchParams;
	const o: Origen = {
		fuente: valido(q.get('utm_source')),
		medio: valido(q.get('utm_medium')),
		campana: valido(q.get('utm_campaign')),
		contenido: valido(q.get('utm_content'))
	};
	const limpio = Object.fromEntries(Object.entries(o).filter(([, v]) => v)) as Origen;
	return Object.keys(limpio).length ? { ...limpio, llego: ahora.toISOString() } : null;
}

export function guardarOrigen(cookies: Cookies, o: Origen) {
	cookies.set(COOKIE_ORIGEN, JSON.stringify(o), { path: '/', httpOnly: true, sameSite: 'lax', maxAge: 90 * 86_400 });
}

/** El origen guardado (sólo los campos conocidos y válidos), o null. */
export function leerOrigen(cookies: Cookies): Origen | null {
	try {
		const r = JSON.parse(cookies.get(COOKIE_ORIGEN) ?? 'null') as Record<string, unknown> | null;
		if (!r || typeof r !== 'object' || Array.isArray(r)) return null;
		const limpio = Object.fromEntries(CAMPOS.map((k) => [k, valido(r[k])]).filter(([, v]) => v)) as Origen;
		if (!Object.keys(limpio).length) return null;
		const llego = typeof r.llego === 'string' && r.llego.length <= 40 ? r.llego : undefined;
		return llego ? { ...limpio, llego } : limpio;
	} catch {
		return null;
	}
}
