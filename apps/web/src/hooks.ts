import type { Reroute } from '@sveltejs/kit/hooks';
import { cacheDeHosts, rutaPara, RUTA_HOST, type Destino } from '#lib/publico/dominio.ts';

// Dominio propio de un negocio: la raíz sirve /{slug}. El slug sale de /_host (que le pregunta a la
// API), con un minuto de caché en memoria, en el servidor y en el navegador.
const cache = cacheDeHosts();

export const reroute: Reroute = ({ url, fetch }) =>
	rutaPara(url, async (host) => {
		const enCache = cache.leer(host);
		if (enCache !== undefined) return enCache;
		const r = await fetch(`${RUTA_HOST}?host=${encodeURIComponent(host)}`).catch(() => null);
		if (!r || r.status >= 500) return { slug: null }; // sin guardar: se reintenta en el próximo pedido
		const d: Destino = r.ok ? ((await r.json()) as Destino) : { slug: null };
		cache.guardar(host, d);
		return d;
	});
