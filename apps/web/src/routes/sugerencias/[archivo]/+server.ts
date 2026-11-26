import { error } from '@sveltejs/kit';
import { read } from '$app/server';
import mini from '#lib/sugerencias/tulocalweb-mini.jpg';
import banner from '#lib/sugerencias/tulocalweb-banner.jpg';
import type { RequestHandler } from './$types';

/**
 * Las imágenes de las sugerencias de la casa (el catálogo de la API apunta acá: la web, el iPhone y
 * Android leen la misma URL). No van en static/ porque ahí no llevan caché: estas se guardan un mes y
 * no se vuelven a pedir. Para cambiar una imagen, otro nombre de archivo (y el catálogo de la API).
 */
const IMAGENES: Record<string, string> = {
	'tulocalweb-mini.jpg': mini,
	'tulocalweb-banner.jpg': banner
};

export const GET: RequestHandler = async ({ params }) => {
	const fuente = IMAGENES[params.archivo];
	if (!fuente) error(404, 'No existe');
	const res = read(fuente);
	return new Response(res.body, {
		headers: { 'content-type': 'image/jpeg', 'cache-control': 'public, max-age=2592000, immutable' }
	});
};
