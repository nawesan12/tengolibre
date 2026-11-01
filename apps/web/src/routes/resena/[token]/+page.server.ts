import { error, fail } from '@sveltejs/kit';
import { apiPublica } from '#lib/server/api.ts';
import { intentar } from '#lib/server/acciones.ts';
import { codigoValido, o404, opcional } from '#lib/publico/api.server.ts';
import type { Web } from '#lib/publico/tipos.ts';
import type { Actions, PageServerLoad } from './$types';

export type PedidoResena = {
	negocio: string;
	slug: string;
	servicio: string;
	profesional: string;
	inicio: string;
	cliente: string;
	vencido: boolean;
	resena: { puntaje: number; comentario: string } | null;
};

export const load: PageServerLoad = async ({ params, fetch, setHeaders }) => {
	if (!codigoValido(params.token)) error(404, 'Ese link no existe o ya no sirve.');
	setHeaders({ 'cache-control': 'no-store' });
	const pedido = await o404(apiPublica(fetch).get<PedidoResena>(`/v1/resenas/${params.token}`), 'Ese link no existe o ya no sirve.');
	// Los colores del negocio (el talón y las estrellas van con su marca).
	const web = await opcional(apiPublica(fetch).get<Web>(`/v1/p/${pedido.slug}/web`), null);
	return { pedido, web };
};

export const actions: Actions = {
	default: async ({ params, fetch, request }) => {
		const f = await request.formData();
		const puntaje = Number(f.get('puntaje'));
		const comentario = String(f.get('comentario') ?? '');
		if (!(puntaje >= 1 && puntaje <= 5))
			return fail(400, { error: 'Elegí de 1 a 5 estrellas.', codigo: 'invalido', campos: { puntaje: 'requerido' }, comentario });
		const r = await intentar(() => apiPublica(fetch).post(`/v1/resenas/${params.token}`, { puntaje, comentario }));
		if (!('ok' in r)) return fail(r.status, { ...r.data, comentario });
		return { gracias: true };
	}
};
