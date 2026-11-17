import { esHostPropio } from '#lib/publico/dominio.ts';
import { slugDelHost } from '#lib/publico/dominio.server.ts';
import type { RequestHandler } from './$types';

/**
 * El robots.txt: la portada, el directorio y las páginas de los negocios se indexan; el panel, la
 * cuenta, los pasos del turno y el paso a paso de la reserva (/{slug}/reservar), no. En el dominio
 * propio de un negocio, lo mismo con el sitemap de ese negocio solo; en un host que no es de nadie,
 * nada.
 */
const PRIVADAS = ['/panel', '/admin', '/mi-cuenta', '/ingresar', '/salir', '/conectar', '/empezar', '/turno', '/resena', '/invitacion', '/espera', '/cal', '/_host', '/*/reservar'];

const texto = (lineas: string[]) =>
	new Response(lineas.join('\n'), { headers: { 'content-type': 'text/plain; charset=utf-8', 'cache-control': 'public, max-age=3600' } });

export const GET: RequestHandler = async ({ url, fetch }) => {
	if (!esHostPropio(url.hostname) && !(await slugDelHost(fetch, url.hostname))) return texto(['User-agent: *', 'Disallow: /', '']);
	return texto(['User-agent: *', 'Allow: /', ...PRIVADAS.map((p) => `Disallow: ${p}`), '', `Sitemap: ${url.origin}/sitemap.xml`, '']);
};
