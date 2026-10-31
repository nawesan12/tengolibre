import { apiDe, ErrorApi } from '#lib/server/api.ts';
import { dia } from '#lib/formato.ts';
import { esDia, sumarDias } from '#lib/operacion/fechas.ts';
import { cambiarEstado, catalogo, param } from '#lib/operacion/servidor.ts';
import type { Turno } from '#lib/operacion/tipos.ts';
import type { Actions, PageServerLoad } from './$types';

/** Los turnos de un rango (por defecto, de hoy a una semana), con filtros de profesional y estado. */
export const load: PageServerLoad = async (event) => {
	event.depends('panel:turnos');
	const { url } = event;
	const desde = esDia(param(url, 'desde')) ? param(url, 'desde') : dia();
	let hasta = esDia(param(url, 'hasta')) ? param(url, 'hasta') : sumarDias(desde, 6);
	if (hasta < desde) hasta = desde;
	const profesional = param(url, 'profesional');
	const estado = param(url, 'estado');
	const q = new URLSearchParams({ desde, hasta });
	if (profesional) q.set('profesional', profesional);
	if (estado) q.set('estado', estado);
	const api = apiDe(event);
	const [cat, turnos] = await Promise.all([
		catalogo(api),
		api
			.get<{ turnos: Turno[] }>(`/v1/panel/turnos?${q}`)
			.then((r) => ({ turnos: r.turnos, error: '' }))
			.catch((e) => {
				if (e instanceof ErrorApi && e.estado < 500) return { turnos: [] as Turno[], error: e.message };
				throw e;
			})
	]);
	return { ...cat, ...turnos, filtros: { desde, hasta, profesional, estado } };
};

export const actions: Actions = { estado: cambiarEstado };
