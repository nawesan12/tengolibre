import { redirect } from '@sveltejs/kit';
import { elegirNegocio, volverValido } from '#lib/server/sesion.ts';
import type { Actions, PageServerLoad } from './$types';

const reId = /^[0-9a-f-]{36}$/;

/**
 * Cambiar de negocio por URL: /panel/negocio?ir=<id>&volver=<ruta del panel>. Lo usa el aviso push
 * de otro negocio (static/sw-panel.js), para abrir la pantalla con ese negocio activo. Sólo si la
 * persona es del negocio (la lista que trae el layout); si no, sigue con el que tenía.
 */
export const load: PageServerLoad = async ({ url, cookies, parent }) => {
	const ir = url.searchParams.get('ir') ?? '';
	const pedido = url.searchParams.get('volver');
	const volver = volverValido(pedido) && /^\/panel(\/|\?|$)/.test(pedido) ? pedido : '/panel';
	if (reId.test(ir)) {
		const { negocios } = await parent();
		if (negocios.some((n) => n.id === ir)) elegirNegocio(cookies, ir);
	}
	redirect(303, volver);
};

/** Cambiar de negocio: el layout del panel verifica que sea de la persona. */
export const actions: Actions = {
	default: async ({ request, cookies }) => {
		const id = String((await request.formData()).get('id') ?? '');
		if (reId.test(id)) elegirNegocio(cookies, id);
		redirect(303, '/panel');
	}
};
