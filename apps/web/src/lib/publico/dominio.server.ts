import { apiPublica, ErrorApi } from '#lib/server/api.ts';
import { cacheDeHosts, esHostPropio } from './dominio.ts';

// La misma caché corta que el reroute: un minuto por host.
const cache = cacheDeHosts();

/**
 * De qué negocio es este host (dominio propio verificado): su slug, o null si es de TengoLibre o
 * de nadie. Si la API no responde, null sin guardar (se reintenta en el próximo pedido).
 */
export async function slugDelHost(f: typeof fetch, host: string): Promise<string | null> {
	host = host.toLowerCase();
	if (esHostPropio(host) || !/^[a-z0-9.-]{1,253}$/.test(host)) return null;
	const enCache = cache.leer(host);
	if (enCache !== undefined) return enCache;
	try {
		const { slug } = await apiPublica(f).get<{ slug: string }>(`/v1/host/${encodeURIComponent(host)}`);
		cache.guardar(host, slug);
		return slug;
	} catch (e) {
		if (e instanceof ErrorApi && e.estado === 404) cache.guardar(host, null);
		return null;
	}
}
