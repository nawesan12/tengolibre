import { apiDe, ErrorApi } from '#lib/server/api.ts';
import { dia } from '#lib/formato.ts';
import { diasEntre, esDia, sumarDias } from '#lib/operacion/fechas.ts';
import { param, siSePuede } from '#lib/operacion/servidor.ts';
import { ausentes, opcional } from '#lib/gestion/api-reportes.ts';
import { accion } from '#lib/configuracion/acciones.server.ts';
import type { HorasMuertasDatos } from '#lib/gestion/reportes/HorasMuertas.svelte';
import type { Reporte } from '#lib/operacion/tipos.ts';
import type { Actions, PageServerLoad } from './$types';

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
		const [reporte, previo, faltan, horas] = await Promise.all([
			siSePuede(api.get<Reporte>(`/v1/panel/reportes?${q}`)),
			siSePuede(api.get<Reporte>(`/v1/panel/reportes?${new URLSearchParams({ desde: antes.desde, hasta: antes.hasta })}`)).catch(() => null),
			ausentes(api, q).catch(() => null),
			opcional(api.get<HorasMuertasDatos>('/v1/panel/reportes/horas-muertas')).catch(() => null)
		]);
		return {
			reporte,
			previo: previo ? { ingresos: previo.resumen.ingresos, desde: antes.desde, mes: antes.mes } : null,
			ausentes: faltan,
			horasMuertas: horas,
			error: ''
		};
	} catch (e) {
		if (e instanceof ErrorApi && e.estado < 500) return { reporte: null, previo: null, ausentes: null, horasMuertas: null, error: e.message };
		throw e;
	}
};

export const actions: Actions = {
	/** El cupón sugerido de Horas muertas, tal cual lo propone la API. */
	cupon: async (event) =>
		accion('cupon', 'Creamos el cupón. Ya vale en esa franja.', () =>
			apiDe(event).post('/v1/panel/reportes/horas-muertas/cupon', {})
		)
};
