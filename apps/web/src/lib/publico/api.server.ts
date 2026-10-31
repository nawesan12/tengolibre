import { error } from '@sveltejs/kit';
import { apiPublica, ErrorApi } from '#lib/server/api.ts';
import type { Cobro, Pagina } from './tipos.ts';

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

export async function pagina(f: typeof fetch, slug: string) {
	if (!slugValido(slug)) error(404, 'No encontramos ese negocio.');
	return o404(apiPublica(f).get<Pagina>(`/v1/p/${slug}`), 'No encontramos ese negocio.');
}

export const cobro = (f: typeof fetch, slug: string) => opcional(apiPublica(f).get<Cobro>(`/v1/p/${slug}/cobro`), null);

/** Un código (de turno, de reseña, de invitación) como para ir en la URL de la API. */
export const codigoValido = (c: string) => /^[A-Za-z0-9_-]{6,128}$/.test(c);
