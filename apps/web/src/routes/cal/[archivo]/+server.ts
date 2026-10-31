import { error } from '@sveltejs/kit';
import { apiPublica } from '#lib/server/api.ts';
import type { RequestHandler } from './$types';

/** El calendario para suscribirse (iCal): lo arma la API; acá sólo se reenvía con el mismo dominio. */
export const GET: RequestHandler = async ({ params, fetch }) => {
	if (!/^[A-Za-z0-9_-]{32,64}\.ics$/.test(params.archivo)) error(404, 'No encontramos ese calendario.');
	const r = await apiPublica(fetch).crudo(`/v1/cal/${params.archivo}`);
	if (r.status === 404) error(404, 'No encontramos ese calendario.');
	if (!r.ok) error(502, 'No pudimos leer el calendario.');
	return new Response(r.body, {
		headers: {
			'content-type': 'text/calendar; charset=utf-8',
			'cache-control': 'private, max-age=300',
			'x-robots-tag': 'noindex'
		}
	});
};
