import { error, json } from '@sveltejs/kit';
import { apiDe, ErrorApi } from '#lib/server/api.ts';
import type { CuentaCaja } from '#lib/operacion/contrato.ts';
import type { RequestHandler } from './$types';

/** La cuenta de la caja de un turno (caja.md), para abrirla al marcar «Atendido». */
export const GET: RequestHandler = async (event) => {
	try {
		return json(await apiDe(event).get<CuentaCaja>(`/v1/panel/turnos/${encodeURIComponent(event.params.id)}/caja`));
	} catch (e) {
		if (e instanceof ErrorApi) error(e.estado, e.message);
		throw e;
	}
};
