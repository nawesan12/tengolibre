import { json } from '@sveltejs/kit';
import { apiPublica, ErrorApi } from '#lib/server/api.ts';
import { alternativas, formatoValido, RESERVADOS, type Disponibilidad } from '#lib/alta/slug.ts';
import type { RequestHandler } from './$types';

/**
 * ¿Está libre esta dirección? La API no tiene un chequeo propio: la página pública de un slug que no
 * existe da 404 (la misma función, app.negocio_publico, que usa el alta para decir «ocupado»).
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
	let es: boolean;
	try {
		await apiPublica(f).get(`/v1/p/${slug}/web`);
		es = false;
	} catch (e) {
		if (!(e instanceof ErrorApi && e.estado === 404)) throw e;
		es = true;
	}
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
