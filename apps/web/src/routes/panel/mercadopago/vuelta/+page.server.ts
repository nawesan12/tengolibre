import { redirect } from '@sveltejs/kit';
import { apiDe, ErrorApi } from '#lib/server/api.ts';
import type { PageServerLoad } from './$types';

/**
 * Vuelta del OAuth de Mercado Pago: llega con `code` y `state` (o con `error` si la persona no
 * autorizó). Se canjean en la API con la sesión del panel y se vuelve a Pagos.
 */
export const load: PageServerLoad = async (event) => {
	const q = event.url.searchParams;
	const codigo = q.get('code');
	const estado = q.get('state');
	if (q.get('error') || !codigo || !estado)
		return { error: 'No se autorizó la conexión con Mercado Pago. Podés probar de nuevo cuando quieras.' };
	try {
		await apiDe(event).post('/v1/panel/mercadopago/vuelta', { codigo, estado });
	} catch (e) {
		if (e instanceof ErrorApi) return { error: e.message };
		throw e;
	}
	redirect(303, '/panel/pagos?mercadopago=conectada');
};
