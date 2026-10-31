import { json } from '@sveltejs/kit';
import { apiPublica, ErrorApi } from '#lib/server/api.ts';
import type { RequestHandler } from './$types';

/** De un host (dominio propio de un negocio) a su slug; null si no es de nadie. Lo usa el reroute. */
export const GET: RequestHandler = async ({ url, fetch }) => {
	const host = (url.searchParams.get('host') ?? '').toLowerCase();
	if (!/^[a-z0-9.-]{1,253}$/.test(host)) return json({ slug: null });
	try {
		const r = await apiPublica(fetch).get<{ slug: string }>(`/v1/host/${encodeURIComponent(host)}`);
		return json({ slug: r.slug }, { headers: { 'cache-control': 'public, max-age=60' } });
	} catch (e) {
		if (e instanceof ErrorApi && e.estado === 404) return json({ slug: null }, { headers: { 'cache-control': 'public, max-age=60' } });
		throw e;
	}
};
