import type { RequestHandler } from './$types';

/**
 * El robots.txt de tengolibre.com.ar: la portada, el directorio y las páginas de los negocios se
 * indexan; el panel, la cuenta y los pasos del turno, no. (En el dominio propio de un negocio el
 * reroute lo manda a su página y no existe.)
 */
const PRIVADAS = ['/panel', '/admin', '/mi-cuenta', '/ingresar', '/salir', '/conectar', '/empezar', '/turno', '/resena', '/invitacion', '/espera', '/cal', '/_host'];

export const GET: RequestHandler = ({ url }) =>
	new Response(
		['User-agent: *', 'Allow: /', ...PRIVADAS.map((p) => `Disallow: ${p}`), '', `Sitemap: ${url.origin}/sitemap.xml`, ''].join('\n'),
		{ headers: { 'content-type': 'text/plain; charset=utf-8', 'cache-control': 'public, max-age=3600' } }
	);
