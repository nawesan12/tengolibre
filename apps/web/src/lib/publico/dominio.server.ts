import { apiPublica, ErrorApi } from '#lib/server/api.ts';
import { cacheDeHosts, esHostPropio } from './dominio.ts';

// La misma caché corta que el reroute: un minuto por host.
const cache = cacheDeHosts();

/**
 * De qué negocio es este host (dominio propio verificado): su slug, o null si es de TengoLibre o
 * de nadie (o si su plan ya no trae dominio propio). Si la API no responde, null sin guardar (se reintenta en el próximo pedido).
 */
export async function slugDelHost(f: typeof fetch, host: string): Promise<string | null> {
	host = host.toLowerCase();
	if (esHostPropio(host) || !/^[a-z0-9.-]{1,253}$/.test(host)) return null;
	const enCache = cache.leer(host);
	if (enCache !== undefined) return enCache.redirigir ? null : enCache.slug;
	try {
		const d = await apiPublica(f).get<{ slug: string; redirigir?: boolean }>(`/v1/host/${encodeURIComponent(host)}`);
		cache.guardar(host, d);
		return d.redirigir ? null : d.slug;
	} catch (e) {
		if (e instanceof ErrorApi && e.estado === 404) cache.guardar(host, { slug: null });
		return null;
	}
}
