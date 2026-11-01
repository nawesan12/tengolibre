import { error, fail } from '@sveltejs/kit';
import { apiDe, ErrorApi } from '#lib/server/api.ts';
import { intentar } from '#lib/server/acciones.ts';
import { dia } from '#lib/formato.ts';
import { esDia, esHora, instante } from '#lib/operacion/fechas.ts';
import { cambiarEstado, catalogo, cobrar, enc, param, siExiste, siSePuede } from '#lib/operacion/servidor.ts';
import type { CuentaCaja, Producto } from '#lib/operacion/contrato.ts';
import type { HorarioLibre, RegistroAviso, Turno } from '#lib/operacion/tipos.ts';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	event.depends('panel:turnos', 'panel:notificaciones', 'panel:caja');
	const api = apiDe(event);
	const id = enc(event.params.id);
	let turno: Turno;
	try {
		turno = await api.get<Turno>(`/v1/panel/turnos/${id}`);
	} catch (e) {
		if (e instanceof ErrorApi && (e.estado === 404 || e.estado === 400)) error(404, 'No encontramos ese turno.');
		throw e;
	}
	const [cat, avisos, productos, caja] = await Promise.all([
		catalogo(api),
		siSePuede(api.get<{ avisos: RegistroAviso[] }>(`/v1/panel/avisos?turno=${id}`)).then((r) => r?.avisos ?? null),
		siExiste(api.get<{ productos: Producto[] }>('/v1/panel/productos?activos=1')).catch(() => null),
		siExiste(api.get<CuentaCaja>(`/v1/panel/turnos/${id}/caja`)).catch(() => null)
	]);

	// Reprogramar: los libres del día elegido, para el mismo servicio.
	const fecha = esDia(param(event.url, 'fecha')) ? param(event.url, 'fecha') : dia(turno.inicio);
	const profesional = param(event.url, 'profesional') || turno.profesionalId || '';
	let horarios: HorarioLibre[] = [];
	let errorLibres = '';
	if (turno.servicioId) {
		const q = new URLSearchParams({ servicio: turno.servicioId, fecha });
		if (profesional) q.set('profesional', profesional);
		try {
			({ horarios } = await api.get<{ horarios: HorarioLibre[] }>(`/v1/panel/libres?${q}`));
		} catch (e) {
			if (!(e instanceof ErrorApi) || e.estado >= 500) throw e;
			errorLibres = e.message;
		}
	}
	const profesionales = cat.profesionales.filter((p) => p.activo && (!turno.servicioId || p.servicios.includes(turno.servicioId)));
	return { turno, avisos, profesionales, fecha, profesional, horarios, errorLibres, productos: productos?.productos ?? [], caja };
};

export const actions: Actions = {
	estado: cambiarEstado,
	cobrar,
	/** Ya lo llamaron (whatsapp.md). */
	llamado: async (event) => intentar(() => apiDe(event).post(`/v1/panel/turnos/${enc(event.params.id)}/llamado`)),
	/** Turno fijo: cancela éste y los que siguen de la serie (turno-fijo.md). */
	cancelarSiguientes: async (event) =>
		intentar(() => apiDe(event).post<{ cancelados: string[] }>(`/v1/panel/turnos/${enc(event.params.id)}/cancelar-siguientes`)),
	nota: async (event) => {
		const f = await event.request.formData();
		const r = await intentar(() => apiDe(event).patch(`/v1/panel/turnos/${enc(event.params.id)}`, { nota: String(f.get('nota') ?? '') }));
		return 'datos' in r ? { ...r, guardado: 'nota' as const } : r;
	},
	reprogramar: async (event) => {
		const f = await event.request.formData();
		const s = (k: string) => String(f.get(k) ?? '').trim();
		const forzar = f.get('forzar') === 'on';
		let inicio = s('inicio');
		if (forzar) {
			if (!esDia(s('fecha')) || !esHora(s('hora'))) return fail(400, { error: 'Poné el día y la hora del sobreturno.', codigo: 'invalido', campos: {} });
			inicio = instante(s('fecha'), s('hora'));
		}
		if (!inicio) return fail(400, { error: 'Elegí el horario nuevo.', codigo: 'invalido', campos: {} });
		return intentar(() =>
			apiDe(event).post(`/v1/panel/turnos/${enc(event.params.id)}/reprogramar`, { inicio, profesionalId: s('profesionalId'), forzar })
		);
	},
	reenviar: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).post(`/v1/panel/avisos/${enc(f.get('id'))}/reenviar`));
	}
};
