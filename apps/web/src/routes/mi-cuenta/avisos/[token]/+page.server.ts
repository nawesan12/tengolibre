import { error } from '@sveltejs/kit';
import { apiPublica } from '#lib/server/api.ts';
import { intentar } from '#lib/server/acciones.ts';
import { codigoValido, o404 } from '#lib/publico/api.server.ts';
import type { Actions, PageServerLoad } from './$types';

type Preferencias = { negocio: string; slug: string; cliente: string; email: boolean; recordatorios: boolean };

export const load: PageServerLoad = async ({ params, fetch, setHeaders }) => {
	if (!codigoValido(params.token)) error(404, 'Ese link no existe o ya no sirve.');
	setHeaders({ 'cache-control': 'no-store' });
	return {
		preferencias: await o404(apiPublica(fetch).get<Preferencias>(`/v1/preferencias/${params.token}`), 'Ese link no existe o ya no sirve.')
	};
};

export const actions: Actions = {
	default: async ({ params, fetch, request }) => {
		const f = await request.formData();
		// "Ninguno" apaga todo de una (el link de "no quiero más mails").
		const nada = f.get('nada') === '1';
		const r = await intentar(() =>
			apiPublica(fetch).put<Preferencias>(`/v1/preferencias/${params.token}`, {
				email: !nada && f.get('email') === 'on',
				recordatorios: !nada && f.get('recordatorios') === 'on'
			})
		);
		if (!('ok' in r)) return r;
		return { guardado: true };
	}
};
