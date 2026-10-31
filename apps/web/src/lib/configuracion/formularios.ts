import type { SubmitFunction } from '$app/forms';

/**
 * Para los formularios que editan algo que ya existe: después de guardar se recargan los datos
 * pero no se vacían los campos (el reset volvería a lo que había antes de guardar).
 */
export const sinReset: SubmitFunction = () => async ({ update }) => update({ reset: false });

/** Pide confirmación antes de mandar (borrar, desconectar, cancelar). */
export const confirmar =
	(pregunta: string): SubmitFunction =>
	({ cancel }) => {
		if (!window.confirm(pregunta)) cancel();
		return async ({ update }) => update({ reset: false });
	};
