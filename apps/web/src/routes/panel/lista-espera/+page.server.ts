import { apiDe, type Api } from '#lib/server/api.ts';
import { intentar } from '#lib/server/acciones.ts';
import { dia } from '#lib/formato.ts';
import { esDia, sumarDias } from '#lib/operacion/fechas.ts';
import { enc, param, siSePuede } from '#lib/operacion/servidor.ts';
import type { Anotado, HorarioLibre } from '#lib/operacion/tipos.ts';
import type { Actions, PageServerLoad } from './$types';

const DIAS_A_MIRAR = 7;
const SUGERIDOS = 6;

/** Los primeros horarios libres dentro de lo que pidió el anotado (de hoy en adelante). */
async function sugeridos(api: Api, a: Anotado, hoy: string): Promise<HorarioLibre[]> {
	let d = a.desde > hoy ? a.desde : hoy;
	for (let i = 0; i < DIAS_A_MIRAR && d <= a.hasta; i++, d = sumarDias(d, 1)) {
		const q = new URLSearchParams({ servicio: a.servicioId, fecha: d });
		if (a.profesionalId) q.set('profesional', a.profesionalId);
		const r = await api.get<{ horarios: HorarioLibre[] }>(`/v1/panel/libres?${q}`).catch(() => null);
		if (r && r.horarios.length) return r.horarios.slice(0, SUGERIDOS);
	}
	return [];
}

export const load: PageServerLoad = async (event) => {
	event.depends('panel:espera', 'panel:turnos');
	const api = apiDe(event);
	const todos = param(event.url, 'todos') === '1';
	const r = await siSePuede(api.get<{ espera: Anotado[] }>(`/v1/panel/espera${todos ? '?todos=1' : ''}`));
	if (!r) return { prohibido: true as const, todos, espera: [], sugeridos: {} as Record<string, HorarioLibre[]> };
	const hoy = dia();
	const pendientes = r.espera.filter((a) => !a.avisadoEn && esDia(a.hasta) && a.hasta >= hoy).slice(0, 30);
	const pares = await Promise.all(pendientes.map(async (a) => [a.id, await sugeridos(api, a, hoy)] as const));
	return { prohibido: false as const, todos, espera: r.espera, sugeridos: Object.fromEntries(pares) as Record<string, HorarioLibre[]> };
};

export const actions: Actions = {
	avisar: async (event) => {
		const f = await event.request.formData();
		const inicio = String(f.get('inicio') ?? '');
		return intentar(() =>
			apiDe(event).post(`/v1/panel/espera/${enc(f.get('id'))}/avisar`, {
				inicio: inicio || undefined,
				nota: String(f.get('nota') ?? '')
			})
		);
	},
	borrar: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).borrar(`/v1/panel/espera/${enc(f.get('id'))}`));
	}
};
