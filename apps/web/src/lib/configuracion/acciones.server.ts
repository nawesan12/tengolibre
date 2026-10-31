import type { ActionFailure, RequestEvent } from '@sveltejs/kit';
import { intentar, type FallaForm } from '#lib/server/acciones.ts';
import { apiDe, ErrorApi } from '#lib/server/api.ts';

/**
 * Una página de configuración tiene muchos formularios: cada acción dice de cuál viene (`que`), así
 * el mensaje (de error o de "listo") sale al lado del formulario que se mandó y no arriba de todo.
 */
export type Resultado<T> =
	| { ok: true; datos: T; que: string; mensaje: string; error?: undefined; campos?: undefined }
	| ActionFailure<FallaForm & { que: string }>;

export async function accion<T>(que: string, mensaje: string, fn: () => Promise<T>): Promise<Resultado<T>> {
	const r = await intentar(fn);
	if ('ok' in r && r.ok) return { ok: true, datos: r.datos, que, mensaje };
	const falla = r as ActionFailure<FallaForm & { que: string }>;
	falla.data.que = que;
	return falla;
}

/** El 403 de una lectura que no le toca al rol: la sección no se muestra en vez de romper. */
export async function siPuede<T>(p: Promise<T>): Promise<T | null> {
	try {
		return await p;
	} catch (e) {
		if (e instanceof ErrorApi && e.estado === 403) return null;
		throw e;
	}
}

/** Mismo criterio que Api.pedir para un pedido crudo: problem+json a ErrorApi. */
async function leer<T>(res: Response): Promise<T> {
	const json = (await res.json().catch(() => null)) as
		| (T & { detail?: string; title?: string; code?: string; issues?: { path: string; message: string }[] })
		| null;
	if (!res.ok) {
		throw new ErrorApi(
			res.status,
			json?.code ?? 'error',
			json?.detail ?? json?.title ?? 'No pudimos subir la imagen.',
			Object.fromEntries((json?.issues ?? []).map((i) => [i.path, i.message]))
		);
	}
	return json as T;
}

export const MAX_IMAGEN = 5 << 20;

/**
 * Sube una imagen a la API (`POST /v1/panel/archivos`, multipart con el campo `archivo`) con la
 * sesión del pedido y devuelve su URL pública. El tipo y las medidas los valida la API mirando los
 * bytes; acá sólo se corta lo que seguro no pasa, para no mandarlo.
 */
export async function subirImagen(event: Pick<RequestEvent, 'locals' | 'fetch'>, archivo: FormDataEntryValue | null) {
	if (!(archivo instanceof File) || archivo.size === 0) throw new ErrorApi(400, 'sin_archivo', 'Elegí una imagen.');
	if (archivo.size > MAX_IMAGEN) throw new ErrorApi(422, 'muy_grande', 'La imagen pasa de 5 MB.');
	const cuerpo = new FormData();
	cuerpo.set('archivo', archivo, archivo.name || 'imagen');
	const res = await apiDe(event).crudo('/v1/panel/archivos', {
		method: 'POST',
		body: cuerpo,
		headers: { accept: 'application/json' }
	});
	const a = await leer<{ id: string; url: string }>(res);
	return a.url;
}
