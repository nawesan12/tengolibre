import type { Handle } from '@sveltejs/kit/hooks';
import { COOKIE_NEGOCIO, COOKIE_SESION } from '#lib/server/sesion.ts';

export const handle: Handle = async ({ event, resolve }) => {
	event.locals.token = event.cookies.get(COOKIE_SESION) ?? null;
	event.locals.negocio = event.cookies.get(COOKIE_NEGOCIO) ?? null;
	const res = await resolve(event);
	res.headers.set('x-content-type-options', 'nosniff');
	res.headers.set('referrer-policy', 'strict-origin-when-cross-origin');
	if (event.url.pathname.startsWith('/panel') || event.url.pathname.startsWith('/admin')) {
		res.headers.set('x-frame-options', 'DENY');
		res.headers.set('cache-control', 'no-store');
	}
	return res;
};
