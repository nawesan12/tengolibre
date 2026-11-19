import { error, redirect } from '@sveltejs/kit';
import { apiPublica, ErrorApi } from '#lib/server/api.ts';
import type { Cobro, Dia, Horario, Pagina } from './tipos.ts';
import { rutaCanonica } from './dominio.ts';

/** Un pedido público: un 404 de la API es la página de "no encontramos". */
export async function o404<T>(p: Promise<T>, mensaje = 'No encontramos esta página.'): Promise<T> {
	try {
		return await p;
	} catch (e) {
		if (e instanceof ErrorApi && e.estado === 404) error(404, mensaje);
		throw e;
	}
}

/** Lo que no es esencial para mostrar la página: si falla, sigue sin eso. */
export async function opcional<T>(p: Promise<T>, si: T): Promise<T> {
	try {
		return await p;
	} catch {
		return si;
	}
}

const slugValido = (s: string) => /^[a-z0-9]([a-z0-9-]{0,46}[a-z0-9])?$/.test(s);

export async function pagina(f: typeof fetch, slug: string, url?: URL) {
	// /Estudio-Lumi: el link escrito con mayúsculas lleva al de minúsculas (antes, 404).
	if (url && slug !== slug.toLowerCase() && slugValido(slug.toLowerCase())) {
		const a = rutaCanonica(url, slug, slug.toLowerCase());
		if (a) redirect(308, a);
	}
	if (!slugValido(slug)) error(404, 'No encontramos ese negocio.');
	const p = await o404(apiPublica(f).get<Pagina>(`/v1/p/${slug}`), 'No encontramos ese negocio.');
	if (url) canonica(url, slug, p.negocio.slug);
	return p;
}

/**
 * Un link viejo (el negocio cambió su dirección): la API lo sigue resolviendo y acá se manda al nuevo,
 * con la misma ruta y la query. Así los QR y carteles impresos siguen andando. En un dominio propio la
 * ruta no lleva el slug, y no hay nada que corregir.
 *
 * Temporal (307) y con caché acotada (hooks.server.ts): un 301 sin Cache-Control lo guarda el
 * navegador para siempre, y si el negocio volvía a su link anterior (o el viejo lo tomaba otro) la
 * clienta quedaba en un bucle de redirecciones o en el negocio equivocado. El canonical de la página
 * ya le dice a los buscadores cuál es la buena.
 */
export function canonica(url: URL, pedido: string, actual: string) {
	const a = rutaCanonica(url, pedido, actual);
	if (a) redirect(307, a);
}

export const cobro = (f: typeof fetch, slug: string) => opcional(apiPublica(f).get<Cobro>(`/v1/p/${slug}/cobro`), null);

/** El primer lugar libre de un servicio: { fecha, hora } (o null si no hay o si falla). */
export async function primerLugar(f: typeof fetch, slug: string, servicioId: string, profesional = '') {
	const api = apiPublica(f);
	const q = new URLSearchParams({ servicio: servicioId });
	if (profesional) q.set('profesional', profesional);
	const dias = await opcional(api.get<{ dias: Dia[] }>(`/v1/p/${slug}/dias?${q}`), { dias: [] });
	const dia = dias.dias.find((d) => d.libres > 0);
	if (!dia) return null;
	q.set('fecha', dia.fecha);
	const hs = await opcional(api.get<{ horarios: Horario[] }>(`/v1/p/${slug}/libres?${q}`), { horarios: [] });
	return hs.horarios[0] ? { fecha: dia.fecha, hora: hs.horarios[0].hora, dias: dias.dias } : null;
}

/** Un código (de turno, de reseña, de invitación) como para ir en la URL de la API. */
export const codigoValido = (c: string) => /^[A-Za-z0-9_-]{6,128}$/.test(c);
