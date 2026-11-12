import { fail, redirect } from '@sveltejs/kit';
import { apiDe, ErrorApi, type NegocioDeLaPersona, type Persona } from '#lib/server/api.ts';
import { borrarSesion } from '#lib/server/sesion.ts';
import type { Actions, PageServerLoad } from './$types';

/**
 * Borrar la cuenta. Es la URL que va en Google Play (Seguridad de los datos) y la que abren las apps:
 * sin sesión explica qué pasa y manda a ingresar con la vuelta acá; con sesión, pide escribir el mail
 * para confirmar. La API desactiva la cuenta en el momento y borra los datos personales a los 30 días.
 */
export const load: PageServerLoad = async (event) => {
	const listo = event.url.searchParams.has('listo');
	if (!event.locals.token || listo) return { persona: null, duenia: [] as string[], listo };
	try {
		const api = apiDe(event);
		const [persona, { negocios }] = await Promise.all([
			api.get<Persona>('/v1/auth/persona'),
			api.get<{ negocios: NegocioDeLaPersona[] }>('/v1/yo')
		]);
		return { persona, duenia: negocios.filter((n) => n.rol === 'duenio').map((n) => n.nombre), listo };
	} catch (x) {
		if (!(x instanceof ErrorApi) || x.estado !== 401) throw x;
		borrarSesion(event.cookies);
		return { persona: null, duenia: [] as string[], listo };
	}
};

export const actions: Actions = {
	default: async (event) => {
		if (!event.locals.token) redirect(303, '/ingresar?volver=/mi-cuenta/borrar');
		const email = String((await event.request.formData()).get('email') ?? '').trim();
		const api = apiDe(event);
		try {
			const persona = await api.get<Persona>('/v1/auth/persona');
			if (!email || email.toLowerCase() !== persona.email.trim().toLowerCase())
				return fail(400, { email, error: 'Ese no es el mail de tu cuenta. Escribilo igual que arriba.' });
			await api.borrar('/v1/auth/cuenta');
		} catch (x) {
			if (!(x instanceof ErrorApi)) throw x;
			if (x.estado === 401) {
				borrarSesion(event.cookies);
				redirect(303, '/ingresar?volver=/mi-cuenta/borrar');
			}
			return fail(x.estado >= 500 ? 502 : x.estado, { email, error: x.message });
		}
		borrarSesion(event.cookies);
		redirect(303, '/mi-cuenta/borrar?listo=1');
	}
};
