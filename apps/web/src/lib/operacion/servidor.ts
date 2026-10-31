import type { RequestEvent } from '@sveltejs/kit';
import { apiDe, ErrorApi, type Api } from '#lib/server/api.ts';
import { intentar } from '#lib/server/acciones.ts';
import type { Prestacion, Profesional } from './tipos.ts';

/**
 * Ayudas de servidor para las pantallas de la operación. La API ya aplica los roles: lo que un rol
 * no puede ver vuelve 403, y la pantalla lo muestra con un mensaje en vez de romperse.
 */
export async function siSePuede<T>(p: Promise<T>): Promise<T | null> {
	try {
		return await p;
	} catch (e) {
		if (e instanceof ErrorApi && e.estado === 403) return null;
		throw e;
	}
}

/**
 * Una lectura de las funciones nuevas (lib/operacion/contrato.ts): mientras la API no las tenga
 * (404, 405, 501) o el rol no las vea (403), vuelve null y la pantalla sigue sin ese dato.
 */
export async function siExiste<T>(p: Promise<T>): Promise<T | null> {
	try {
		return await p;
	} catch (e) {
		if (e instanceof ErrorApi && [403, 404, 405, 501].includes(e.estado)) return null;
		throw e;
	}
}

/** Profesionales y servicios para los selectores (los activos primero, en su orden). */
export async function catalogo(api: Api) {
	const [{ profesionales }, { servicios }] = await Promise.all([
		api.get<{ profesionales: Profesional[] }>('/v1/panel/profesionales'),
		api.get<{ servicios: Prestacion[] }>('/v1/panel/servicios')
	]);
	return { profesionales, servicios };
}

/** La acción `estado` de cualquier pantalla con turnos: id y estado nuevo. */
export async function cambiarEstado(event: RequestEvent) {
	const f = await event.request.formData();
	return intentar(() => apiDe(event).patch(`/v1/panel/turnos/${enc(f.get('id'))}`, { estado: String(f.get('estado') ?? '') }));
}

/** Un id de la ruta o del formulario, seguro para ir en la URL de la API. */
export const enc = (v: FormDataEntryValue | string | null | undefined) => encodeURIComponent(String(v ?? ''));

/** Un parámetro de la consulta, o vacío. */
export const param = (url: URL, k: string) => url.searchParams.get(k)?.trim() ?? '';
