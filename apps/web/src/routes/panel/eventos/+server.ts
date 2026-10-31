import { error } from '@sveltejs/kit';
import { apiDe } from '#lib/server/api.ts';
import type { RequestHandler } from './$types';

/**
 * El stream de tiempo real: el navegador no tiene el token (cookie httpOnly), así que este servidor
 * abre el SSE de la API con la sesión y lo pasa tal cual. Si el navegador se va, se corta el de la API.
 */
export const GET: RequestHandler = async (event) => {
	if (!event.locals.token || !event.locals.negocio) error(401, 'Tenés que ingresar');
	const res = await apiDe(event).crudo('/v1/panel/eventos', {
		headers: { accept: 'text/event-stream' },
		signal: event.request.signal
	});
	if (!res.ok || !res.body) error(res.status === 200 ? 502 : res.status, 'Sin tiempo real');
	return new Response(res.body, {
		headers: {
			'content-type': 'text/event-stream',
			'cache-control': 'no-store',
			'x-accel-buffering': 'no'
		}
	});
};
