import { apiPublica } from '#lib/server/api.ts';
import { esHostPropio } from '#lib/publico/dominio.ts';
import { slugDelHost } from '#lib/publico/dominio.server.ts';
import type { Directorio } from '#lib/publico/tipos.ts';
import type { RequestHandler } from './$types';

/** Las páginas fijas y las de los negocios del directorio (hasta 40 páginas de a 50, el tope de la API). */
const FIJAS = ['/', '/explorar', '/privacidad', '/terminos'];

// El directorio, 10 minutos en memoria: cada visita al sitemap no son 40 pedidos a la API.
const DIEZ_MINUTOS = 10 * 60_000;
let guardado: { slugs: string[]; vence: number } | null = null;
let enCurso: Promise<string[]> | null = null;

async function directorio(f: typeof fetch): Promise<string[]> {
	if (guardado && guardado.vence > Date.now()) return guardado.slugs;
	enCurso ??= (async () => {
		const api = apiPublica(f);
		const slugs: string[] = [];
		let completo = true;
		for (let pagina = 1; pagina <= 40; pagina++) {
			const d = await api.get<Directorio>(`/v1/explorar?pagina=${pagina}&limite=50`).catch(() => null);
			if (!d) {
				completo = pagina > 1;
				break;
			}
			slugs.push(...d.negocios.map((n) => n.slug));
			if (pagina >= d.paginas) break;
		}
		// Si la API no respondió de entrada, no se guarda (se reintenta en la próxima visita).
		if (completo) guardado = { slugs, vence: Date.now() + DIEZ_MINUTOS };
		return slugs;
	})().finally(() => (enCurso = null));
	return enCurso;
}

const xml = (urls: string[]) =>
	new Response(
		'<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n' +
			urls.map((u) => `\t<url><loc>${u}</loc></url>`).join('\n') +
			'\n</urlset>\n',
		{ headers: { 'content-type': 'application/xml; charset=utf-8', 'cache-control': 'public, max-age=3600' } }
	);

export const GET: RequestHandler = async ({ url, fetch }) => {
	// En el dominio propio de un negocio, sólo su página (nunca los negocios de otros).
	if (!esHostPropio(url.hostname)) return xml((await slugDelHost(fetch, url.hostname)) ? [`${url.origin}/`] : []);
	const slugs = await directorio(fetch);
	return xml([...FIJAS, ...slugs.map((s) => `/${encodeURIComponent(s)}`)].map((p) => url.origin + p));
};
