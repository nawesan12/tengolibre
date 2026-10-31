import { error } from '@sveltejs/kit';
import { apiPublica } from '#lib/server/api.ts';
import { intentar } from '#lib/server/acciones.ts';
import { codigoValido, o404 } from '#lib/publico/api.server.ts';
import { PREFERENCIAS_NUEVAS, type Preferencias } from '#lib/cuenta/contratos.ts';
import type { Actions, PageServerLoad } from './$types';

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
		// "Dejar de recibir todo" apaga todo de una (el link de "no quiero más mails").
		const nada = f.get('nada') === '1';
		const cuerpo: Record<string, boolean> = {
			email: !nada && f.get('email') === 'on',
			recordatorios: !nada && f.get('recordatorios') === 'on'
		};
		// Los avisos nuevos van sólo si la API los mandó al cargar (si no, los rechaza).
		const hay = new Set(String(f.get('campos') ?? '').split(','));
		for (const k of PREFERENCIAS_NUEVAS) if (hay.has(k)) cuerpo[k] = !nada && f.get(k) === 'on';
		const r = await intentar(() => apiPublica(fetch).put<Preferencias>(`/v1/preferencias/${params.token}`, cuerpo));
		if (!('ok' in r)) return r;
		return { guardado: true, nada };
	}
};
