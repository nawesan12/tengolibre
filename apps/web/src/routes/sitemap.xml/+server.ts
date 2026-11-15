import { apiPublica } from '#lib/server/api.ts';
import type { Directorio } from '#lib/publico/tipos.ts';
import type { RequestHandler } from './$types';

/** Las páginas fijas y las de los negocios del directorio (hasta 40 páginas de a 50, el tope de la API). */
const FIJAS = ['/', '/explorar', '/privacidad', '/terminos'];

export const GET: RequestHandler = async ({ url, fetch }) => {
	const api = apiPublica(fetch);
	const slugs: string[] = [];
	for (let pagina = 1; pagina <= 40; pagina++) {
		const d = await api.get<Directorio>(`/v1/explorar?pagina=${pagina}&limite=50`).catch(() => null);
		if (!d) break;
		slugs.push(...d.negocios.map((n) => n.slug));
		if (pagina >= d.paginas) break;
	}
	const urls = [...FIJAS, ...slugs.map((s) => `/${encodeURIComponent(s)}`)];
	return new Response(
		'<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n' +
			urls.map((p) => `\t<url><loc>${url.origin}${p}</loc></url>`).join('\n') +
			'\n</urlset>\n',
		{ headers: { 'content-type': 'application/xml; charset=utf-8', 'cache-control': 'public, max-age=3600' } }
	);
};
