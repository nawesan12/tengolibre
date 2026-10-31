import { error, redirect, type RequestEvent } from '@sveltejs/kit';
import { apiDe, ErrorApi, type Api } from '#lib/server/api.ts';
import { borrarSesion } from '#lib/server/sesion.ts';

type Evento = Pick<RequestEvent, 'locals' | 'fetch' | 'url' | 'cookies'>;

const aIngresar = (e: Evento) => `/ingresar?volver=${encodeURIComponent(e.url.pathname + e.url.search)}`;

/**
 * Corre pedidos que necesitan la sesión de la persona. Sin sesión, o si venció (401), la borra y
 * manda a ingresar con la vuelta a esta página. Con `ocultar`, un 404 de la API (lo que responde el
 * admin a quien no es del equipo) es un 404 de la página, sin decir nada más.
 */
export async function conSesion<T>(e: Evento, fn: (api: Api) => Promise<T>, ocultar = false): Promise<T> {
	if (!e.locals.token) redirect(303, aIngresar(e));
	try {
		return await fn(apiDe(e));
	} catch (x) {
		if (x instanceof ErrorApi && x.estado === 401) {
			borrarSesion(e.cookies);
			redirect(303, aIngresar(e));
		}
		if (ocultar && x instanceof ErrorApi && x.estado === 404) error(404, 'Esa página no existe.');
		throw x;
	}
}
