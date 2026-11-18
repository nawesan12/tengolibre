import { json } from '@sveltejs/kit';
import { apiDe, ErrorApi } from '#lib/server/api.ts';
import type { EstadoLink, Link } from '#lib/panel/link.ts';
import type { RequestHandler } from './$types';

/**
 * El link del negocio para el editor (Mi web y el panel «Tu link» de la barra): sin ?slug, el link de
 * ahora con los anteriores; con ?slug, si está libre para este negocio (mientras se escribe).
 */
export const GET: RequestHandler = async (event) => {
	const api = apiDe(event);
	const slug = (event.url.searchParams.get('slug') ?? '').trim().toLowerCase();
	if (!slug) return json(await api.get<Link>('/v1/panel/negocio/link'));
	if (!/^[a-z0-9-]{1,48}$/.test(slug)) return json({ estado: 'invalida' satisfies EstadoLink });
	return json(await api.get<{ estado: EstadoLink }>(`/v1/panel/negocio/link/${slug}`), { headers: { 'cache-control': 'no-store' } });
};

/** Cambiar el link: los viejos siguen llevando acá (la API los guarda y la web redirige con 301). */
export const POST: RequestHandler = async (event) => {
	const { slug } = (await event.request.json().catch(() => ({}))) as { slug?: string };
	try {
		return json(await apiDe(event).patch<Link>('/v1/panel/negocio/link', { slug: String(slug ?? '') }));
	} catch (e) {
		if (e instanceof ErrorApi && e.estado < 500) return json({ error: e.message, codigo: e.codigo }, { status: e.estado });
		throw e;
	}
};
