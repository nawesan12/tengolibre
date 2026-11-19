/**
 * Dominio propio: un negocio puede apuntar su dominio (turnos.labarberia.com) al front, y la raíz
 * de ese dominio sirve su página. Los hosts de TengoLibre (y los locales) no se resuelven.
 */
const PROPIOS = new Set(['tengolibre.com.ar', 'www.tengolibre.com.ar', 'localhost', '127.0.0.1', '[::1]', '::1']);

export const esHostPropio = (host: string) => PROPIOS.has(host.toLowerCase());

/** El endpoint que traduce el host al slug (no choca con ningún slug: llevan sólo a-z, 0-9 y -). */
export const RUTA_HOST = '/_host';

/** Lo que dice la API de un host: el slug del negocio y si hay que llevar a la dirección de TengoLibre
 * (el dominio está guardado pero el plan ya no trae dominio propio). */
export type Destino = { slug: string | null; redirigir?: boolean };

type Entrada = { destino: Destino; vence: number };

/** Caché corta en memoria: un minuto por host, así una visita no pega siempre en la API. */
export function cacheDeHosts(ttl = 60_000, ahora = () => Date.now()) {
	const m = new Map<string, Entrada>();
	return {
		leer(host: string): Destino | undefined {
			const e = m.get(host);
			if (!e) return undefined;
			if (e.vence <= ahora()) {
				m.delete(host);
				return undefined;
			}
			return e.destino;
		},
		guardar(host: string, destino: Destino) {
			if (m.size > 1000) m.clear();
			m.set(host, { destino, vence: ahora() + ttl });
		}
	};
}

/**
 * A qué ruta va un pedido: en un dominio ajeno, la raíz es la página del negocio. Lo demás (el
 * link del turno, /_app, /{slug}/reservar) sigue igual. Un dominio fuera del plan no se reescribe:
 * lo redirige el handle (redireccionFueraDelPlan).
 */
export async function rutaPara(url: URL, buscar: (host: string) => Promise<Destino>): Promise<string | undefined> {
	if (url.pathname !== '/' || esHostPropio(url.hostname)) return undefined;
	const d = await buscar(url.hostname.toLowerCase());
	return d.slug && !d.redirigir ? `/${d.slug}` : undefined;
}

/** Un dominio propio fuera del plan lleva, en cualquier ruta, a la página en TengoLibre. */
export function redireccionFueraDelPlan(url: URL, d: Destino): string | null {
	if (!d.slug || !d.redirigir) return null;
	const resto = url.pathname === '/' ? '' : url.pathname;
	return `${SITIO}/${d.slug}${resto}${url.search}`;
}

/** La dirección de TengoLibre: la de los canonical de las páginas de los negocios. */
export const SITIO = 'https://tengolibre.com.ar';

/** El origen para los links absolutos: TengoLibre en producción; en local, el de la URL. */
export const sitioDe = (url: URL) => (/^(localhost|127\.0\.0\.1|\[::1\]|::1)$/.test(url.hostname) ? url.origin : SITIO);

/** Adónde va un link viejo (`/<pedido>/…` → `/<actual>/…`, con la query); null si no hay que moverse. */
export function rutaCanonica(url: URL, pedido: string, actual: string): string | null {
	if (pedido === actual) return null;
	const pre = `/${pedido}`;
	if (url.pathname !== pre && !url.pathname.startsWith(`${pre}/`)) return null;
	return `/${actual}${url.pathname.slice(pre.length)}${url.search}`;
}
