import { error, redirect } from '@sveltejs/kit';
import { apiPublica } from '#lib/server/api.ts';
import { intentar } from '#lib/server/acciones.ts';
import { codigoValido, o404, opcional } from '#lib/publico/api.server.ts';
import type { OfertaPublica, Turno, Web } from '#lib/publico/tipos.ts';
import type { Actions, PageServerLoad } from './$types';

/**
 * «Se liberó lo que esperabas» (docs/api/espera-ofertas.md): el lugar que se le ofrece a quien
 * esperaba en la lista; lo toma de un toque o lo pasa al siguiente. El mail linkea acá.
 */
export const load: PageServerLoad = async ({ params, fetch, setHeaders }) => {
	if (!codigoValido(params.token)) error(404, 'Ese link no existe o ya no sirve.');
	setHeaders({ 'cache-control': 'no-store' });
	const api = apiPublica(fetch);
	const oferta = await o404(api.get<OfertaPublica>(`/v1/espera/${params.token}`), 'Ese link no existe o ya no sirve.');
	// Si ya lo tomó, su turno.
	if (oferta.estado === 'tomada' && oferta.turno) redirect(303, `/turno/${oferta.turno}`);
	// Los colores del negocio (el talón va con su marca).
	const web = await opcional(api.get<Web>(`/v1/p/${oferta.negocio.slug}/web`), null);
	return { oferta, web };
};

export const actions: Actions = {
	tomar: async ({ params, fetch }) => {
		const r = await intentar(() => apiPublica(fetch).post<Turno>(`/v1/espera/${params.token}/tomar`));
		if (!('ok' in r)) return r;
		// Con seña, a Mercado Pago; sin link (MP no respondió), lo pide el link del turno.
		if (r.datos.pago?.url && /^https:\/\//.test(r.datos.pago.url)) redirect(303, r.datos.pago.url);
		redirect(303, `/turno/${r.datos.codigo}?nuevo=1`);
	},
	pasar: async ({ params, fetch }) => {
		const r = await intentar(() => apiPublica(fetch).post<OfertaPublica>(`/v1/espera/${params.token}/pasar`));
		if (!('ok' in r)) return r;
		return { pasada: true };
	}
};
