import { error, redirect } from '@sveltejs/kit';
import { apiPublica } from '#lib/server/api.ts';
import { intentar } from '#lib/server/acciones.ts';
import { codigoValido, o404 } from '#lib/publico/api.server.ts';
import type { OfertaHueco } from '#lib/publico/contrato.ts';
import type { Actions, PageServerLoad } from './$types';

/**
 * Se liberó un lugar y se le ofrece a quien esperaba (lista de espera): lo toma de un toque o lo
 * pasa al siguiente. La API de esto es un supuesto: ver #lib/publico/contrato.ts.
 */
export const load: PageServerLoad = async ({ params, fetch, setHeaders }) => {
	if (!codigoValido(params.token)) error(404, 'Ese link no existe o ya no sirve.');
	setHeaders({ 'cache-control': 'no-store' });
	const oferta = await o404(apiPublica(fetch).get<OfertaHueco>(`/v1/espera/ofertas/${params.token}`), 'Ese link no existe o ya no sirve.');
	// Si ya lo tomó, su turno.
	if (oferta.estado === 'tomada' && oferta.turnoCodigo) redirect(303, `/turno/${oferta.turnoCodigo}`);
	return { oferta };
};

export const actions: Actions = {
	tomar: async ({ params, fetch }) => {
		const r = await intentar(() =>
			apiPublica(fetch).post<{ codigo: string; pago?: { url?: string } }>(`/v1/espera/ofertas/${params.token}/tomar`)
		);
		if (!('ok' in r)) return r;
		if (r.datos.pago?.url && /^https:\/\//.test(r.datos.pago.url)) redirect(303, r.datos.pago.url);
		redirect(303, `/turno/${r.datos.codigo}?nuevo=1`);
	},
	pasar: async ({ params, fetch }) => {
		const r = await intentar(() => apiPublica(fetch).post(`/v1/espera/ofertas/${params.token}/pasar`));
		if (!('ok' in r)) return r;
		return { pasada: true };
	}
};
