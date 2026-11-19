import { fail } from '@sveltejs/kit';
import { apiPublica, ErrorApi } from '#lib/server/api.ts';
import type { Actions } from './$types';

/** El botón de arrepentimiento (Res. SC 424/2020): sin cuenta, deja el pedido y devuelve un código. */
export const actions: Actions = {
	default: async ({ request, fetch }) => {
		const f = await request.formData();
		const valores = { email: String(f.get('email') ?? '').trim(), negocio: String(f.get('negocio') ?? '').trim(), detalle: String(f.get('detalle') ?? '').trim() };
		try {
			const r = await apiPublica(fetch).post<{ codigo: string }>('/v1/arrepentimiento', valores);
			return { codigo: r.codigo };
		} catch (e) {
			if (e instanceof ErrorApi) return fail(e.estado >= 400 && e.estado < 500 ? e.estado : 502, { error: e.message, valores });
			throw e;
		}
	}
};
