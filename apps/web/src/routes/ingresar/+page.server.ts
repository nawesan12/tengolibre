import { fail, redirect } from '@sveltejs/kit';
import { dev } from '$app/env';
import { TENGOLIBRE_INGRESO_PRUEBA } from '$app/env/private';
import { PUBLIC_CLERK_PUBLICABLE } from '$app/env/public';
import { apiPublica, ErrorApi } from '#lib/server/api.ts';
import { COOKIE_VOLVER, destino, guardarSesion, volverValido } from '#lib/server/sesion.ts';
import { fapiDe } from '#lib/clerk.ts';
import { opcional } from '#lib/publico/api.server.ts';
import type { Pagina } from '#lib/publico/tipos.ts';
import { colorValido, turnoGuardado } from '#lib/cuenta/presentacion.ts';
import type { Actions, PageServerLoad } from './$types';

/** El ingreso de prueba (sin Clerk) sólo existe en desarrollo o si se prende a propósito en local. */
const conPrueba = () => dev || TENGOLIBRE_INGRESO_PRUEBA;

export const load: PageServerLoad = async ({ locals, url, cookies, fetch }) => {
	const volver = url.searchParams.get('volver');
	if (volverValido(volver)) cookies.set(COOKIE_VOLVER, volver, { path: '/', httpOnly: true, sameSite: 'lax', maxAge: 900 });
	if (locals.token) redirect(303, volverValido(volver) ? volver : '/panel');
	// Viene de la reserva de un negocio: es un socio, no alguien que entra a su panel.
	const socio = volverValido(volver) && !/^\/(panel|admin|empezar|invitacion|mi-cuenta)/.test(volver);
	return {
		fapi: fapiDe(PUBLIC_CLERK_PUBLICABLE),
		prueba: conPrueba(),
		error: url.searchParams.get('error'),
		socio,
		volver: socio ? volver : null,
		negocio: socio ? await negocioDeLaReserva(fetch, volver) : null
	};
};

/** El negocio de la reserva de donde viene el socio (sus colores) y el turno que deja guardado. */
async function negocioDeLaReserva(f: typeof fetch, volver: string) {
	const u = new URL(volver, 'http://x');
	const slug = u.pathname.split('/')[1] ?? '';
	if (!/^[a-z0-9]([a-z0-9-]{0,46}[a-z0-9])?$/.test(slug)) return null;
	const p = await opcional(apiPublica(f).get<Pagina>(`/v1/p/${slug}`), null);
	if (!p) return null;
	const servicio = p.servicios.find((s) => s.id === u.searchParams.get('servicio'))?.nombre ?? null;
	return {
		nombre: p.negocio.nombre,
		logo: p.negocio.logo,
		color: colorValido(p.web?.colorPrimario),
		turno: servicio ? turnoGuardado(servicio, u.searchParams.get('inicio'), p.negocio.zonaHoraria || undefined) : ''
	};
}

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
