import type { Handle, HandleFetch } from '@sveltejs/kit/hooks';
import { COOKIE_NEGOCIO, COOKIE_SESION, elegirNegocio } from '#lib/server/sesion.ts';
import { apiDe, BASE_API, type NegocioDeLaPersona } from '#lib/server/api.ts';

export const handle: Handle = async ({ event, resolve }) => {
	event.locals.token = event.cookies.get(COOKIE_SESION) ?? null;
	event.locals.negocio = event.cookies.get(COOKIE_NEGOCIO) ?? null;
	// Sin negocio elegido (recién ingresó o se borró la cookie), el primero, como el layout del panel: si
	// no, las cargas de la página corren en paralelo con el layout y piden sin X-Negocio (un 500).
	const ruta0 = event.url.pathname;
	if (event.locals.token && !event.locals.negocio && (ruta0 === '/panel' || ruta0.startsWith('/panel/'))) {
		const yo = await apiDe(event)
			.get<{ negocios: NegocioDeLaPersona[] }>('/v1/yo')
			.catch(() => null);
		const primero = yo?.negocios[0];
		if (primero) {
			elegirNegocio(event.cookies, primero.id);
			event.locals.negocio = primero.id;
		}
	}
	const res = await resolve(event);
	res.headers.set('x-content-type-options', 'nosniff');
	res.headers.set('referrer-policy', 'strict-origin-when-cross-origin');
	const ruta = event.url.pathname;
	const en = (base: string) => ruta === base || ruta.startsWith(base + '/');
	if (en('/panel') || en('/admin')) res.headers.set('x-frame-options', 'DENY');
	// Lo de la sesión nunca queda en una caché compartida: el panel, la cuenta, el alta, el ingreso y
	// cualquier respuesta a alguien con sesión (la reserva con su membresía, por ejemplo).
	if (en('/panel') || en('/admin') || en('/mi-cuenta') || en('/empezar') || en('/ingresar') || event.locals.token)
		res.headers.set('cache-control', 'private, no-store');
	return res;
};

/**
 * Los pedidos a la API salen de este servidor, así que para la API todos venían de la misma IP y los
 * topes «por IP» (reservar, lista de espera, cupones) eran uno solo para todo el sitio. Se reenvía la
 * IP del visitante que puso Cloudflare (Caddy sólo acepta a Cloudflare, que pisa ese encabezado si
 * lo manda el navegador).
 */
export const handleFetch: HandleFetch = async ({ event, request, fetch }) => {
	const ip = event.request.headers.get('cf-connecting-ip');
	if (ip && request.url.startsWith(BASE_API())) {
		request = new Request(request, { headers: new Headers(request.headers) });
		request.headers.set('cf-connecting-ip', ip);
	}
	return fetch(request);
};
