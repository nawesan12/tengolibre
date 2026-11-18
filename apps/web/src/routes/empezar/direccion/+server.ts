import { json } from '@sveltejs/kit';
import { apiPublica } from '#lib/server/api.ts';
import { alternativas, formatoValido, RESERVADOS, type Disponibilidad } from '#lib/alta/slug.ts';
import type { RequestHandler } from './$types';

/**
 * ¿Está libre esta dirección? Le pregunta a la API (/v1/slugs/{slug}), que también da por tomada la
 * dirección anterior de otro negocio (sus QR impresos siguen andando 90 días como mínimo).
 * Si está tomada, devuelve alternativas que sí están libres.
 */
// El alta pregunta sin sesión (se elige la dirección antes de entrar con Google o Apple), así que no
// se puede exigir: cada respuesta queda 30 segundos en memoria, y escribir letra por letra (o
// repetir el pedido a propósito) no pega siempre en la API.
const TTL = 30_000;
const vistos = new Map<string, { libre: boolean; vence: number }>();

async function libre(f: typeof fetch, slug: string): Promise<boolean> {
	if (!formatoValido(slug) || RESERVADOS.has(slug)) return false;
	const ya = vistos.get(slug);
	if (ya && ya.vence > Date.now()) return ya.libre;
	const { estado } = await apiPublica(f).get<{ estado: string }>(`/v1/slugs/${slug}`);
	const es = estado === 'libre';
	if (vistos.size > 2000) vistos.clear();
	vistos.set(slug, { libre: es, vence: Date.now() + TTL });
	return es;
}

export const GET: RequestHandler = async ({ url, fetch }) => {
	const slug = url.searchParams.get('slug') ?? '';
	const rubro = url.searchParams.get('rubro') ?? undefined;
	let r: Disponibilidad;
	if (!formatoValido(slug) || RESERVADOS.has(slug)) r = { estado: 'invalida', sugerencias: [] };
	else if (await libre(fetch, slug)) r = { estado: 'libre', sugerencias: [] };
	else {
		const alts = alternativas(slug, rubro);
		const ok = await Promise.all(alts.map((a) => libre(fetch, a).catch(() => false)));
		r = { estado: 'tomada', sugerencias: alts.filter((_, i) => ok[i]) };
	}
	return json(r, { headers: { 'cache-control': 'no-store' } });
};
