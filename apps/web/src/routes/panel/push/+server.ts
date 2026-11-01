import { error } from '@sveltejs/kit';
import { apiDe, ErrorApi } from '#lib/server/api.ts';
import type { RequestHandler } from './$types';

/** Guarda la suscripción push de este navegador (push-web.md): POST /v1/yo/suscripciones-web. */
export const POST: RequestHandler = async (event) => {
	const cuerpo = await event.request.json().catch(() => null);
	if (!cuerpo || typeof cuerpo !== 'object') error(400, 'Falta la suscripción.');
	try {
		await apiDe(event).post('/v1/yo/suscripciones-web', cuerpo);
	} catch (e) {
		if (e instanceof ErrorApi) error(e.estado, e.message);
		throw e;
	}
	return new Response(null, { status: 204 });
};
