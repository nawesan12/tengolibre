import { redirect } from '@sveltejs/kit';
import { apiDe } from '#lib/server/api.ts';
import { borrarSesion } from '#lib/server/sesion.ts';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = () => redirect(303, '/');

export const actions: Actions = {
	default: async (event) => {
		await apiDe(event).post('/v1/auth/salir').catch(() => undefined);
		borrarSesion(event.cookies);
		redirect(303, '/');
	}
};
