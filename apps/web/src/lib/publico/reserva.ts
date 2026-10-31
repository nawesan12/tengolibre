/** "cualquiera": que la agenda elija al primero libre. */
export const CUALQUIERA = 'cualquiera';

export type Paso = 'servicio' | 'profesional' | 'dia' | 'horario' | 'datos';

/** Los pasos de la reserva, en orden; cada uno guarda en la URL lo elegido hasta ahí. */
export const PASOS: Paso[] = ['servicio', 'profesional', 'dia', 'horario', 'datos'];

/** Qué parámetros de la URL quedan al volver a un paso (lo de los pasos siguientes se borra). */
const DE_PASO: Record<Paso, string[]> = {
	servicio: [],
	profesional: ['servicio'],
	dia: ['servicio', 'profesional'],
	horario: ['servicio', 'profesional', 'fecha'],
	datos: ['servicio', 'profesional', 'fecha', 'inicio']
};

/**
 * La URL de un paso con lo elegido: `actual` son los parámetros de ahora (lo que se conserva, como
 * el código de referido), `cambios` lo nuevo.
 */
export function urlDePaso(base: string, actual: Pick<URLSearchParams, 'get'>, paso: Paso, cambios: Record<string, string> = {}) {
	const q = new URLSearchParams();
	for (const k of DE_PASO[paso]) {
		const v = actual.get(k);
		if (v) q.set(k, v);
	}
	const ref = actual.get('ref');
	if (ref) q.set('ref', ref);
	for (const [k, v] of Object.entries(cambios)) {
		if (v) q.set(k, v);
		else q.delete(k);
	}
	const s = q.toString();
	return s ? `${base}?${s}` : base;
}
