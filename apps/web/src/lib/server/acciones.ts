import { fail, type ActionFailure } from '@sveltejs/kit';
import { ErrorApi } from './api.ts';

/** Lo que devuelve una acción que falló: el mensaje y los errores por campo, para el formulario. */
export type FallaForm = { error: string; codigo: string; campos: Record<string, string> };

/**
 * Corre una acción que habla con la API: un ErrorApi vuelve como fail() con el estado de la API,
 * así el formulario muestra el mensaje sin perder lo cargado. Cualquier otro error sigue de largo.
 */
export async function intentar<T>(
	fn: () => Promise<T>
): Promise<{ ok: true; datos: T; error?: undefined; campos?: undefined } | ActionFailure<FallaForm>> {
	try {
		return { ok: true, datos: await fn() };
	} catch (e) {
		if (e instanceof ErrorApi)
			return fail(e.estado >= 400 && e.estado < 600 ? e.estado : 400, { error: e.message, codigo: e.codigo, campos: e.campos });
		throw e;
	}
}

/** Los campos de un formulario como objeto plano de strings. */
export const campos = (f: FormData) => Object.fromEntries([...f.entries()].map(([k, v]) => [k, String(v)]));

/** Un número del formulario (vacío: undefined). */
export const numero = (v: FormDataEntryValue | null) => (v === null || String(v).trim() === '' ? undefined : Number(v));
