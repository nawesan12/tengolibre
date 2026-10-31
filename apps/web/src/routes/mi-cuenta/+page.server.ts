import type { NegocioDeLaPersona, Persona } from '#lib/server/api.ts';
import { conSesion } from '#lib/cuenta/ingreso.server.ts';
import type { PageServerLoad } from './$types';

/** La cuenta de la persona: quién es y sus negocios. Entrar a uno usa la acción de /panel/negocio. */
export const load: PageServerLoad = (event) =>
	conSesion(event, async (api) => {
		const [persona, { negocios }] = await Promise.all([
			api.get<Persona>('/v1/auth/persona'),
			api.get<{ negocios: NegocioDeLaPersona[] }>('/v1/yo')
		]);
		return { persona, negocios, elegido: event.locals.negocio };
	});
