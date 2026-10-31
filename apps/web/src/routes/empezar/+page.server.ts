import { redirect } from '@sveltejs/kit';
import { apiDe } from '#lib/server/api.ts';
import { intentar } from '#lib/server/acciones.ts';
import { elegirNegocio } from '#lib/server/sesion.ts';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = ({ locals }) => {
	if (!locals.token) redirect(303, '/ingresar?volver=/empezar');
};

export const actions: Actions = {
	default: async (event) => {
		const f = await event.request.formData();
		const r = await intentar(() =>
			apiDe(event).post<{ id: string }>('/v1/negocios', { nombre: String(f.get('nombre') ?? ''), slug: String(f.get('slug') ?? '') })
		);
		if (!('ok' in r)) return r;
		elegirNegocio(event.cookies, r.datos.id);
		redirect(303, '/panel');
	}
};
