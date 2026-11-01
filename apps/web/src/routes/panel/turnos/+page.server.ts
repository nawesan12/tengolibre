import { apiDe, ErrorApi } from '#lib/server/api.ts';
import { dia } from '#lib/formato.ts';
import { esDia, sumarDias } from '#lib/operacion/fechas.ts';
import { cambiarEstado, catalogo, enc, param } from '#lib/operacion/servidor.ts';
import { ATENCION } from '#lib/operacion/secciones.ts';
import { accion, siPuede } from '#lib/configuracion/acciones.server.ts';
import type { Reprogramar } from '#lib/gestion/dias-libres.ts';
import type { Turno } from '#lib/operacion/tipos.ts';
import type { Actions, PageServerLoad } from './$types';

/** Los turnos de un rango (por defecto, de hoy a una semana), con filtros de profesional y estado. */
export const load: PageServerLoad = async (event) => {
	event.depends('panel:turnos');
	event.depends('panel:reprogramar');
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
	// «Para reprogramar»: los turnos de días libres aprobados, para quien atiende (no el profesional).
	const { negocio } = await event.parent();
	const [cat, turnos, reprogramar] = await Promise.all([
		catalogo(api),
		api
			.get<{ turnos: Turno[] }>(`/v1/panel/turnos?${q}`)
			.then((r) => ({ turnos: r.turnos, error: '' }))
			.catch((e) => {
				if (e instanceof ErrorApi && e.estado < 500) return { turnos: [] as Turno[], error: e.message };
				throw e;
			}),
		ATENCION.includes(negocio.rol)
			? siPuede(api.get<{ turnos: Reprogramar[] }>('/v1/panel/reprogramar').then((r) => r.turnos)).catch(() => null)
			: null
	]);
	return { ...cat, ...turnos, reprogramar: reprogramar ?? [], filtros: { desde, hasta, profesional, estado } };
};

export const actions: Actions = {
	estado: cambiarEstado,
	/** Lo habló con la clienta (o lo deja así): sale de la lista sin mover el turno. */
	resolver: async (event) => {
		const f = await event.request.formData();
		return accion('reprogramar', 'Listo: salió de la lista.', () => apiDe(event).post(`/v1/panel/reprogramar/${enc(f.get('id'))}/resolver`));
	}
};
