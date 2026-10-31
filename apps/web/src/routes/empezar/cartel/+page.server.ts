import { redirect } from '@sveltejs/kit';
import { apiDe } from '#lib/server/api.ts';
import type { NegocioAlta, WebAlta } from '#lib/alta/tipos.ts';
import type { PageServerLoad } from './$types';

/** El cartel A5 para el mostrador: el negocio elegido, su color y el QR a su página. */
export const load: PageServerLoad = async (event) => {
	if (!event.locals.token || !event.locals.negocio) redirect(303, '/empezar');
	const api = apiDe(event);
	const [negocio, web] = await Promise.all([api.get<NegocioAlta>('/v1/panel/negocio'), api.get<WebAlta>('/v1/panel/mi-web')]);
	return { negocio, color: web.colorPrimario, origen: event.url.origin };
};
