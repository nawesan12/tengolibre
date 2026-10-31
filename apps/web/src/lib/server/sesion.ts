import type { Cookies } from '@sveltejs/kit';
import { dev } from '$app/env';

/**
 * Cookies del panel. `__Host-` en producción: sólo HTTPS, sin Domain y con Path=/, así no las puede
 * pisar un subdominio. En desarrollo (http://localhost) van sin el prefijo.
 */
export const COOKIE_SESION = dev ? 'tl_sesion' : '__Host-tl_sesion';
export const COOKIE_NEGOCIO = dev ? 'tl_negocio' : '__Host-tl_negocio';

export function guardarSesion(cookies: Cookies, token: string, venceEn: string) {
	cookies.set(COOKIE_SESION, token, {
		path: '/',
		httpOnly: true,
		secure: !dev,
		sameSite: 'lax',
		expires: new Date(venceEn)
	});
}

export function borrarSesion(cookies: Cookies) {
	cookies.delete(COOKIE_SESION, { path: '/', secure: !dev });
	cookies.delete(COOKIE_NEGOCIO, { path: '/', secure: !dev });
}

export function elegirNegocio(cookies: Cookies, id: string) {
	cookies.set(COOKIE_NEGOCIO, id, {
		path: '/',
		httpOnly: true,
		secure: !dev,
		sameSite: 'lax',
		maxAge: 60 * 60 * 24 * 365
	});
}

/**
 * Volver a donde se pidió entrar: Google o Apple pasan por otro sitio y se pierde la consulta, así
 * que va en una cookie corta. Sólo rutas propias (nada de redirecciones abiertas).
 */
export const COOKIE_VOLVER = 'tl_volver';
export const volverValido = (v: string | null | undefined): v is string =>
	!!v && v.length < 2000 && !v.includes('\\') && /^\/(panel|admin|empezar|invitacion)(\/[^/]|\?|$)/.test(v);

export function destino(cookies: Cookies) {
	const v = cookies.get(COOKIE_VOLVER);
	cookies.delete(COOKIE_VOLVER, { path: '/' });
	return volverValido(v) ? v : '/panel';
}
