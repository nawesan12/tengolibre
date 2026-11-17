import type { Handle } from '@sveltejs/kit/hooks';
import { COOKIE_NEGOCIO, COOKIE_SESION } from '#lib/server/sesion.ts';

export const handle: Handle = async ({ event, resolve }) => {
	event.locals.token = event.cookies.get(COOKIE_SESION) ?? null;
	event.locals.negocio = event.cookies.get(COOKIE_NEGOCIO) ?? null;
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
