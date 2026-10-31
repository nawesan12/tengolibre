import { apiDe } from '#lib/server/api.ts';
import { numero } from '#lib/server/acciones.ts';
import { accion, siPuede } from '#lib/configuracion/acciones.server.ts';
import type { EntradaAuditoria } from '#lib/configuracion/actividad.ts';
import type { Actions, PageServerLoad } from './$types';

export type Ajustes = {
	intervalo: number;
	anticipacionMin: number;
	diasVista: number;
	margenMin: number;
	cancelarHastaHoras: number;
	politicaCancelacion: string;
};
export type AjustesAvisos = { resenaDemoraHoras: number; resumenDiario: boolean };
export type AjustesAusencias = { umbralAusencias: number; diasPenalidad: number; marcarAusentes: boolean };

export const load: PageServerLoad = async (event) => {
	const api = apiDe(event);
	const [ajustes, avisos, ausencias, actividad] = await Promise.all([
		api.get<Ajustes>('/v1/panel/ajustes'),
		api.get<AjustesAvisos>('/v1/panel/ajustes/avisos'),
		api.get<AjustesAusencias>('/v1/panel/ausencias/ajustes'),
		// El extracto del registro de actividad (lo último que cambió); si no le toca al rol, no se muestra.
		siPuede(api.get<{ entradas: EntradaAuditoria[] }>('/v1/panel/auditoria?limite=3'))
	]);
	return { ajustes, avisos, ausencias, actividad: actividad?.entradas ?? [] };
};

export const actions: Actions = {
	reserva: async (event) => {
		const f = await event.request.formData();
		// La anticipación se carga en horas (como la piensa un negocio) y la API la guarda en minutos.
		const horas = numero(f.get('anticipacionHoras')) ?? 0;
		return accion('reserva', 'Guardamos los ajustes de reserva.', () =>
			apiDe(event).put('/v1/panel/ajustes', {
				intervalo: numero(f.get('intervalo')) ?? 30,
				anticipacionMin: Math.round(horas * 60),
				diasVista: numero(f.get('diasVista')) ?? 30,
				margenMin: numero(f.get('margenMin')) ?? 0,
				cancelarHastaHoras: numero(f.get('cancelarHastaHoras')) ?? 0,
				politicaCancelacion: String(f.get('politicaCancelacion') ?? '').trim()
			})
		);
	},
	avisos: async (event) => {
		const f = await event.request.formData();
		return accion('avisos', 'Guardamos los avisos.', () =>
			apiDe(event).put('/v1/panel/ajustes/avisos', {
				resenaDemoraHoras: numero(f.get('resenaDemoraHoras')) ?? 0,
				resumenDiario: f.get('resumenDiario') === 'on'
			})
		);
	},
	ausencias: async (event) => {
		const f = await event.request.formData();
		return accion('ausencias', 'Guardamos las reglas de ausencias.', () =>
			apiDe(event).put('/v1/panel/ausencias/ajustes', {
				umbralAusencias: numero(f.get('umbralAusencias')) ?? 0,
				diasPenalidad: numero(f.get('diasPenalidad')) ?? 30,
				marcarAusentes: f.get('marcarAusentes') === 'on'
			})
		);
	}
};
