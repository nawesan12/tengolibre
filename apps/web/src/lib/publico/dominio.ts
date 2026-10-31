/**
 * Dominio propio: un negocio puede apuntar su dominio (turnos.labarberia.com) al front, y la raíz
 * de ese dominio sirve su página. Los hosts de TengoLibre (y los locales) no se resuelven.
 */
const PROPIOS = new Set(['tengolibre.com.ar', 'www.tengolibre.com.ar', 'localhost', '127.0.0.1', '[::1]', '::1']);

export const esHostPropio = (host: string) => PROPIOS.has(host.toLowerCase());

/** El endpoint que traduce el host al slug (no choca con ningún slug: llevan sólo a-z, 0-9 y -). */
export const RUTA_HOST = '/_host';

type Entrada = { slug: string | null; vence: number };

/** Caché corta en memoria: un minuto por host, así una visita no pega siempre en la API. */
export function cacheDeHosts(ttl = 60_000, ahora = () => Date.now()) {
	const m = new Map<string, Entrada>();
	return {
		leer(host: string): string | null | undefined {
			const e = m.get(host);
			if (!e) return undefined;
			if (e.vence <= ahora()) {
				m.delete(host);
				return undefined;
			}
			return e.slug;
		},
		guardar(host: string, slug: string | null) {
			if (m.size > 1000) m.clear();
			m.set(host, { slug, vence: ahora() + ttl });
		}
	};
}

/**
 * A qué ruta va un pedido: en un dominio ajeno, la raíz es la página del negocio. Lo demás (el
 * link del turno, /_app, /{slug}/reservar) sigue igual.
 */
export async function rutaPara(url: URL, buscar: (host: string) => Promise<string | null>): Promise<string | undefined> {
	if (url.pathname !== '/' || esHostPropio(url.hostname)) return undefined;
	const slug = await buscar(url.hostname.toLowerCase());
	return slug ? `/${slug}` : undefined;
}
