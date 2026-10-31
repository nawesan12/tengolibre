import { apiDe } from '#lib/server/api.ts';
import { intentar, numero } from '#lib/server/acciones.ts';
import { enc, param, siSePuede } from '#lib/operacion/servidor.ts';
import type { Penalidad } from '#lib/operacion/tipos.ts';
import type { Actions, PageServerLoad } from './$types';

type Ajustes = { umbralAusencias: number; diasPenalidad: number; marcarAusentes: boolean };
type Ausencias = { total: number; ultimos30: number; penalidadesVigentes: number; reincidentes: number };

export const load: PageServerLoad = async (event) => {
	event.depends('panel:clientes', 'panel:turnos');
	const todas = param(event.url, 'todas') === '1';
	const pagina = Math.max(1, Number(param(event.url, 'pagina')) || 1);
	const q = new URLSearchParams({ pagina: String(pagina) });
	if (todas) q.set('todas', '1');
	const api = apiDe(event);
	const [lista, ausencias, ajustes] = await Promise.all([
		api.get<{ penalidades: Penalidad[]; total: number; pagina: number; porPagina: number }>(`/v1/panel/penalidades?${q}`),
		api.get<Ausencias>('/v1/panel/ausencias'),
		siSePuede(api.get<Ajustes>('/v1/panel/ausencias/ajustes'))
	]);
	return { ...lista, ausencias, ajustes, todas };
};

export const actions: Actions = {
	levantar: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).post(`/v1/panel/penalidades/${enc(f.get('id'))}/levantar`));
	},
	ajustes: async (event) => {
		const f = await event.request.formData();
		return intentar(() =>
			apiDe(event).put('/v1/panel/ausencias/ajustes', {
				umbralAusencias: numero(f.get('umbralAusencias')) ?? 0,
				diasPenalidad: numero(f.get('diasPenalidad')) ?? 30,
				marcarAusentes: f.get('marcarAusentes') === 'on'
			})
		);
	},
	marcar: async (event) => intentar(() => apiDe(event).post<{ marcados: number }>('/v1/panel/ausencias/marcar'))
};
