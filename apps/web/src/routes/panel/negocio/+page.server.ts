import { redirect } from '@sveltejs/kit';
import { elegirNegocio } from '#lib/server/sesion.ts';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = () => redirect(303, '/panel');

/** Cambiar de negocio: el layout del panel verifica que sea de la persona. */
export const actions: Actions = {
	default: async ({ request, cookies }) => {
		const id = String((await request.formData()).get('id') ?? '');
		if (/^[0-9a-f-]{36}$/.test(id)) elegirNegocio(cookies, id);
		redirect(303, '/panel');
	}
};
