/**
 * Qué mensaje mostrar al lado de un formulario. El 402 `suscripcion_requerida` (la prueba venció y
 * el panel quedó de sólo lectura) no es un error del formulario: invita a ir a la suscripción.
 */
export type Form = { que?: string; error?: string; codigo?: string; mensaje?: string } | null | undefined;

export const SUSCRIPCION_REQUERIDA = 'suscripcion_requerida';

export type Mensaje =
	| { tipo: 'error'; texto: string }
	| { tipo: 'suscripcion'; texto: string }
	| { tipo: 'listo'; texto: string }
	| null;

export function mensajeDe(form: Form, que?: string): Mensaje {
	if (!form || (que !== undefined && form.que !== que)) return null;
	if (form.codigo === SUSCRIPCION_REQUERIDA)
		return { tipo: 'suscripcion', texto: 'Tu prueba gratis terminó: podés ver todo, pero para hacer cambios hay que suscribirse.' };
	if (form.error) return { tipo: 'error', texto: form.error };
	if (form.mensaje) return { tipo: 'listo', texto: form.mensaje };
	return null;
}
