import { apiDe } from '#lib/server/api.ts';
import { dia } from '#lib/formato.ts';
import { esDia, lunesDe, sumarDias } from '#lib/operacion/fechas.ts';
import { catalogo, param } from '#lib/operacion/servidor.ts';
import type { Turno } from '#lib/operacion/tipos.ts';
import type { PageServerLoad } from './$types';

/**
 * El calendario (handoff «Panel v2»): un día con una columna por profesional (en el teléfono, una a
 * la vez) o la semana como mapa de horas, con una barrita por profesional.
 */
export const load: PageServerLoad = async (event) => {
	event.depends('panel:turnos');
	const { url } = event;
	const vista = param(url, 'vista') === 'semana' ? 'semana' : 'dia';
	const fecha = esDia(param(url, 'fecha')) ? param(url, 'fecha') : dia();
	const profesional = param(url, 'profesional');
	const cancelados = param(url, 'cancelados') === '1';
	const desde = vista === 'semana' ? lunesDe(fecha) : fecha;
	const hasta = vista === 'semana' ? sumarDias(desde, 6) : fecha;
	const api = apiDe(event);
	const [cat, { turnos }] = await Promise.all([catalogo(api), api.get<{ turnos: Turno[] }>(`/v1/panel/turnos?${new URLSearchParams({ desde, hasta })}`)]);
	return {
		vista,
		fecha,
		desde,
		hasta,
		profesional,
		cancelados,
		profesionales: cat.profesionales,
		turnos: cancelados ? turnos : turnos.filter((t) => t.estado !== 'cancelado')
	};
};
