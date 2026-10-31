import { fail, redirect } from '@sveltejs/kit';
import { dev } from '$app/env';
import { TENGOLIBRE_INGRESO_PRUEBA } from '$app/env/private';
import { PUBLIC_CLERK_PUBLICABLE } from '$app/env/public';
import { apiPublica, ErrorApi } from '#lib/server/api.ts';
import { COOKIE_VOLVER, destino, guardarSesion, volverValido } from '#lib/server/sesion.ts';
import { fapiDe } from '#lib/clerk.ts';
import type { Actions, PageServerLoad } from './$types';

/** El ingreso de prueba (sin Clerk) sólo existe en desarrollo o si se prende a propósito en local. */
const conPrueba = () => dev || TENGOLIBRE_INGRESO_PRUEBA;

export const load: PageServerLoad = ({ locals, url, cookies }) => {
	const volver = url.searchParams.get('volver');
	if (volverValido(volver)) cookies.set(COOKIE_VOLVER, volver, { path: '/', httpOnly: true, sameSite: 'lax', maxAge: 900 });
	if (locals.token) redirect(303, volverValido(volver) ? volver : '/panel');
	return {
		fapi: fapiDe(PUBLIC_CLERK_PUBLICABLE),
		prueba: conPrueba(),
		error: url.searchParams.get('error'),
		// Viene de la reserva de un negocio: es un socio, no alguien que entra a su panel.
		socio: volverValido(volver) && !/^\/(panel|admin|empezar|invitacion|mi-cuenta)/.test(volver)
	};
};

export const actions: Actions = {
	// Desarrollo: entra como una persona de prueba (la API acepta "dev:user_…" sin Clerk).
	prueba: async ({ request, cookies, fetch }) => {
		if (!conPrueba()) return fail(404);
		const id = String((await request.formData()).get('id') ?? '').trim();
		if (!/^user_[A-Za-z0-9]{10,64}$/.test(id)) return fail(400, { error: 'El id va como user_ y al menos 10 letras o números.' });
		try {
			const r = await apiPublica(fetch).post<{ token: string; venceEn: string }>('/v1/auth/clerk', { token: `dev:${id}` });
			guardarSesion(cookies, r.token, r.venceEn);
		} catch (e) {
			if (e instanceof ErrorApi) return fail(400, { error: e.message });
			throw e;
		}
		redirect(303, destino(cookies));
	}
};
