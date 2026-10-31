import { apiDe, ErrorApi } from '#lib/server/api.ts';
import { dia } from '#lib/formato.ts';
import { diasEntre, esDia, sumarDias } from '#lib/operacion/fechas.ts';
import { param, siSePuede } from '#lib/operacion/servidor.ts';
import { ausentes } from '#lib/gestion/api-reportes.ts';
import type { Reporte } from '#lib/operacion/tipos.ts';
import type { PageServerLoad } from './$types';

/** El período anterior para comparar: el mes anterior entero si se mira un mes, si no, los mismos días justo antes. */
function anterior(desde: string, hasta: string) {
	if (desde.endsWith('-01') && hasta.slice(0, 7) === desde.slice(0, 7)) {
		const fin = sumarDias(desde, -1);
		return { desde: `${fin.slice(0, 8)}01`, hasta: fin, mes: true };
	}
	const n = diasEntre(desde, hasta).length;
	return { desde: sumarDias(desde, -n), hasta: sumarDias(desde, -1), mes: false };
}

/** Los números de un período (por defecto, el mes en curso). Es de gestión: otro rol ve un aviso. */
export const load: PageServerLoad = async (event) => {
	event.depends('panel:turnos', 'panel:pagos');
	const hoy = dia();
	let desde = param(event.url, 'desde');
	let hasta = param(event.url, 'hasta');
	if (!esDia(desde)) desde = `${hoy.slice(0, 8)}01`;
	if (!esDia(hasta)) hasta = hoy;
	if (hasta < desde) [desde, hasta] = [hasta, desde];
	const q = new URLSearchParams({ desde, hasta });
	const antes = anterior(desde, hasta);
	const api = apiDe(event);
	try {
		const [reporte, previo, faltan] = await Promise.all([
			siSePuede(api.get<Reporte>(`/v1/panel/reportes?${q}`)),
			siSePuede(api.get<Reporte>(`/v1/panel/reportes?${new URLSearchParams({ desde: antes.desde, hasta: antes.hasta })}`)).catch(() => null),
			ausentes(api, q).catch(() => null)
		]);
		return {
			reporte,
			previo: previo ? { ingresos: previo.resumen.ingresos, desde: antes.desde, mes: antes.mes } : null,
			ausentes: faltan,
			error: ''
		};
	} catch (e) {
		if (e instanceof ErrorApi && e.estado < 500) return { reporte: null, previo: null, ausentes: null, error: e.message };
		throw e;
	}
};
