import { redirect } from '@sveltejs/kit';
import { apiDe, ErrorApi, type NegocioDeLaPersona, type Persona } from '#lib/server/api.ts';
import { borrarSesion, elegirNegocio } from '#lib/server/sesion.ts';
import type { LayoutServerLoad } from './$types';

/**
 * El panel exige sesión y un negocio. El negocio elegido vive en una cookie; si no es de la persona
 * (o no hay), se toma el primero. Sin negocios, a darse de alta.
 */
export const load: LayoutServerLoad = async (event) => {
	const { locals, url, cookies } = event;
	if (!locals.token) redirect(303, `/ingresar?volver=${encodeURIComponent(url.pathname + url.search)}`);
	const api = apiDe(event);
	let persona: Persona;
	let negocios: NegocioDeLaPersona[];
	try {
		[persona, { negocios }] = await Promise.all([
			api.get<Persona>('/v1/auth/persona'),
			api.get<{ negocios: NegocioDeLaPersona[] }>('/v1/yo')
		]);
	} catch (e) {
		if (e instanceof ErrorApi && e.estado === 401) {
			borrarSesion(cookies);
			redirect(303, `/ingresar?volver=${encodeURIComponent(url.pathname)}`);
		}
		throw e;
	}
	if (negocios.length === 0) redirect(303, '/empezar');
	let negocio = negocios.find((n) => n.id === locals.negocio);
	if (!negocio) {
		negocio = negocios[0];
		elegirNegocio(cookies, negocio.id);
		locals.negocio = negocio.id;
	}
	return { persona, negocios, negocio };
};
