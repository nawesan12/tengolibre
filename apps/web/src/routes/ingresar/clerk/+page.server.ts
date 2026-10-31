import { redirect } from '@sveltejs/kit';
import { apiPublica, ErrorApi } from '#lib/server/api.ts';
import { destino, guardarSesion } from '#lib/server/sesion.ts';
import type { Actions } from './$types';

/**
 * Vuelta de Google o Apple por Clerk. La página termina el ida y vuelta en el navegador y postea acá
 * el token de sesión de Clerk; la API lo verifica y devuelve la sesión de TengoLibre, que queda en
 * la cookie httpOnly.
 */
export const actions: Actions = {
	default: async ({ request, cookies, fetch }) => {
		const f = await request.formData();
		const error = String(f.get('error') ?? '');
		const token = String(f.get('token') ?? '');
		if (error || !token || token.length > 8192)
			redirect(303, `/ingresar?error=${encodeURIComponent(/^[a-z_]{1,40}$/.test(error) ? error : 'cancelado')}`);
		try {
			const r = await apiPublica(fetch).post<{ token: string; venceEn: string }>('/v1/auth/clerk', { token });
			guardarSesion(cookies, r.token, r.venceEn);
		} catch (e) {
			if (!(e instanceof ErrorApi)) throw e;
			redirect(303, `/ingresar?error=${encodeURIComponent(e.codigo)}`);
		}
		redirect(303, destino(cookies));
	}
};
