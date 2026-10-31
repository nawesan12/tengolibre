import { apiDe, ErrorApi } from '#lib/server/api.ts';
import { esDia } from '#lib/operacion/fechas.ts';
import { param, siSePuede } from '#lib/operacion/servidor.ts';
import type { Reporte } from '#lib/operacion/tipos.ts';
import type { PageServerLoad } from './$types';

/** Los números de un período (por defecto, los últimos 30 días). Es de gestión: otro rol ve un aviso. */
export const load: PageServerLoad = async (event) => {
	event.depends('panel:turnos', 'panel:pagos');
	const q = new URLSearchParams();
	for (const k of ['desde', 'hasta']) if (esDia(param(event.url, k))) q.set(k, param(event.url, k));
	try {
		const reporte = await siSePuede(apiDe(event).get<Reporte>(`/v1/panel/reportes?${q}`));
		return { reporte, error: '' };
	} catch (e) {
		if (e instanceof ErrorApi && e.estado < 500) return { reporte: null, error: e.message };
		throw e;
	}
};
