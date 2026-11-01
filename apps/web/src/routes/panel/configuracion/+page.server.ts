import { apiDe, ErrorApi } from '#lib/server/api.ts';
import { numero } from '#lib/server/acciones.ts';
import { accion, siPuede } from '#lib/configuracion/acciones.server.ts';
import type { EntradaAuditoria } from '#lib/configuracion/actividad.ts';
import type {
	AjustesEspera,
	AjustesMensual,
	AjustesWhatsapp,
	MensajeWhatsapp,
	Preferencias,
	Recordatorio,
	SuscripcionWeb
} from '#lib/configuracion/avisos.ts';
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
	// Lo nuevo (WhatsApp, lista de espera, recordatorios, avisos por persona, reporte mensual) se
	// tolera si falla: la página sigue con lo de siempre.
	const opcional = <T>(p: Promise<T>) => siPuede(p).catch(() => null);
	const [ajustes, avisos, ausencias, actividad, whatsapp, mensajes, espera, recordatorios, cupones, preferencias, navegador, mensual] =
		await Promise.all([
			api.get<Ajustes>('/v1/panel/ajustes'),
			api.get<AjustesAvisos>('/v1/panel/ajustes/avisos'),
			api.get<AjustesAusencias>('/v1/panel/ausencias/ajustes'),
			// El extracto del registro de actividad (lo último que cambió); si no le toca al rol, no se muestra.
			siPuede(api.get<{ entradas: EntradaAuditoria[] }>('/v1/panel/auditoria?limite=3')),
			opcional(api.get<AjustesWhatsapp>('/v1/panel/ajustes/whatsapp')),
			opcional(api.get<{ mensajes: MensajeWhatsapp[] }>('/v1/panel/whatsapp/mensajes')),
			opcional(api.get<AjustesEspera>('/v1/panel/ajustes/espera')),
			opcional(api.get<{ recordatorios: Recordatorio[] }>('/v1/panel/recordatorios')),
			opcional(api.get<{ cupones: { id: string; codigo: string; activo: boolean }[] }>('/v1/panel/cupones')),
			opcional(api.get<Preferencias>('/v1/panel/push/preferencias')),
			opcional(api.get<SuscripcionWeb>('/v1/yo/suscripciones-web')),
			opcional(api.get<AjustesMensual>('/v1/panel/reportes/mensual/ajustes'))
		]);
	return {
		ajustes,
		avisos,
		ausencias,
		actividad: actividad?.entradas ?? [],
		whatsapp,
		mensajes: (mensajes?.mensajes ?? []).slice(0, 5),
		espera,
		recordatorios: recordatorios?.recordatorios ?? [],
		cupones: (cupones?.cupones ?? []).filter((c) => c.activo).map((c) => ({ id: c.id, codigo: c.codigo })),
		preferencias,
		navegador,
		mensual
	};
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
	},
	whatsapp: async (event) => {
		const f = await event.request.formData();
		const hora = numero(f.get('hora')) ?? 10;
		return accion('whatsapp', 'Guardamos los avisos por WhatsApp.', () =>
			apiDe(event).put('/v1/panel/ajustes/whatsapp', {
				confirmacion: f.get('confirmacion') === 'on',
				hora: `${String(hora).padStart(2, '0')}:00`,
				sinRespuestaHoras: numero(f.get('sinRespuestaHoras')) ?? 4,
				recordatorio: f.get('recordatorio') === 'on'
			})
		);
	},
	mensual: async (event) => {
		const f = await event.request.formData();
		const encargados = f.getAll('encargado').map((id) => ({ id: String(id), recibe: f.get(`recibe-${id}`) === 'on' }));
		return accion('mensual', 'Guardamos el reporte mensual.', () =>
			apiDe(event).put('/v1/panel/reportes/mensual/ajustes', {
				...(f.has('hayActivo') ? { activo: f.get('activo') === 'on' } : {}),
				encargados
			})
		);
	},
	espera: async (event) => {
		const f = await event.request.formData();
		return accion('espera', 'Guardamos cómo se ofrece la lista de espera.', () =>
			apiDe(event).put('/v1/panel/ajustes/espera', {
				minutos: numero(f.get('minutos')) ?? 30,
				automatica: f.get('automatica') === 'on'
			})
		);
	},
	recordatorio: async (event) => {
		const f = await event.request.formData();
		const tipo = String(f.get('tipo') ?? '');
		const cuerpo: Record<string, unknown> = {
			activo: f.get('activo') === 'on',
			texto: String(f.get('texto') ?? '').trim(),
			cuponId: String(f.get('cuponId') ?? '')
		};
		if (tipo === 'extraniamos') cuerpo.dias = numero(f.get('dias')) ?? 60;
		return accion(`recordatorio-${tipo}`, 'Guardamos el recordatorio.', () =>
			apiDe(event).put(`/v1/panel/recordatorios/${encodeURIComponent(tipo)}`, cuerpo)
		);
	},
	push: async (event) => {
		const f = await event.request.formData();
		if (f.get('porDefecto') === '1')
			return accion('push', 'Volvimos a los avisos de tu rol.', () => apiDe(event).put('/v1/panel/push/preferencias', { porDefecto: true }));
		const tipos = f.getAll('tipo').map(String);
		const apagados = tipos.filter((t) => f.get(`activo-${t}`) !== 'on');
		return accion('push', 'Guardamos qué avisos te llegan.', () => apiDe(event).put('/v1/panel/push/preferencias', { apagados }));
	},
	suscribir: async (event) => {
		const f = await event.request.formData();
		let sub: Record<string, unknown>;
		try {
			sub = JSON.parse(String(f.get('suscripcion') ?? '')) as Record<string, unknown>;
		} catch {
			return accion('navegador', '', async () => {
				throw new ErrorApi(400, 'invalido', 'El navegador no devolvió la suscripción.');
			});
		}
		return accion('navegador', 'Listo: los avisos te llegan en este navegador.', () =>
			apiDe(event).post('/v1/yo/suscripciones-web', { ...sub, aparato: String(f.get('aparato') ?? '').slice(0, 60) })
		);
	},
	desuscribir: async (event) =>
		accion('navegador', 'Apagamos los avisos en este navegador.', () => apiDe(event).borrar('/v1/yo/suscripciones-web'))
};
