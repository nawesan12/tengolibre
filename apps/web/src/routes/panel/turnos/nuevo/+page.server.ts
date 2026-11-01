import { fail, redirect } from '@sveltejs/kit';
import { apiDe, ErrorApi } from '#lib/server/api.ts';
import { intentar } from '#lib/server/acciones.ts';
import { dia } from '#lib/formato.ts';
import { esDia, esHora, instante } from '#lib/operacion/fechas.ts';
import { catalogo, enc, param, siSePuede } from '#lib/operacion/servidor.ts';
import type { PreviaSerie } from '#lib/operacion/contrato.ts';
import type { Ficha, HorarioLibre, Turno } from '#lib/operacion/tipos.ts';
import type { Actions, PageServerLoad } from './$types';

/**
 * Nuevo turno (handoff «Panel v2», carga rápida): servicio, profesional y día van en la URL (los
 * horarios libres salen de /v1/panel/libres con la página); después el horario o un sobreturno, el
 * cliente y, si se repite, el turno fijo (turno-fijo.md: vista previa y reserva de la serie).
 */
export const load: PageServerLoad = async (event) => {
	event.depends('panel:turnos');
	const { url } = event;
	const api = apiDe(event);
	const cat = await catalogo(api);
	const servicios = cat.servicios.filter((s) => s.activo);
	const servicio = servicios.some((s) => s.id === param(url, 'servicio')) ? param(url, 'servicio') : (servicios[0]?.id ?? '');
	const profesionales = cat.profesionales.filter((p) => p.activo && (!servicio || p.servicios.includes(servicio)));
	const profesional = profesionales.some((p) => p.id === param(url, 'profesional')) ? param(url, 'profesional') : '';
	const fecha = esDia(param(url, 'fecha')) ? param(url, 'fecha') : dia();
	const hora = esHora(param(url, 'hora')) ? param(url, 'hora') : '';

	let horarios: HorarioLibre[] = [];
	let errorLibres = '';
	if (servicio) {
		const q = new URLSearchParams({ servicio, fecha });
		if (profesional) q.set('profesional', profesional);
		try {
			({ horarios } = await api.get<{ horarios: HorarioLibre[] }>(`/v1/panel/libres?${q}`));
		} catch (e) {
			if (!(e instanceof ErrorApi) || e.estado >= 500) throw e;
			errorLibres = e.message;
		}
	}

	// Desde la ficha de un cliente: sus datos ya cargados.
	let cliente = { id: '', nombre: '', telefono: '', email: '' };
	if (param(url, 'cliente')) {
		const f = await siSePuede(api.get<Ficha>(`/v1/panel/clientes/${enc(param(url, 'cliente'))}`)).catch(() => null);
		if (f) cliente = { id: f.id, nombre: f.nombre, telefono: f.telefono, email: f.email };
	}

	return { servicios, profesionales, todos: cat.profesionales, servicio, profesional, fecha, hora, horarios, errorLibres, cliente };
};

/** El inicio elegido: un horario libre, o día + hora del sobreturno. */
function inicioDe(f: FormData) {
	const s = (k: string) => String(f.get(k) ?? '').trim();
	if (f.get('sobreturno') === 'on') {
		if (!esDia(s('fecha')) || !esHora(s('hora'))) return null;
		return instante(s('fecha'), s('hora'));
	}
	return s('inicio') || null;
}

export const actions: Actions = {
	crear: async (event) => {
		const f = await event.request.formData();
		const s = (k: string) => String(f.get(k) ?? '').trim();
		const sobreturno = f.get('sobreturno') === 'on';
		const cada = Number(s('cadaSemanas')) || 0;
		const inicio = inicioDe(f);
		if (sobreturno && !s('profesionalId')) return fail(400, { error: 'Para un sobreturno elegí el profesional.', codigo: 'invalido', campos: {} });
		if (!inicio)
			return fail(400, {
				error: sobreturno ? 'Para un sobreturno poné el día y la hora.' : 'Elegí un horario.',
				codigo: 'invalido',
				campos: sobreturno ? { hora: 'requerido' } : { inicio: 'requerido' }
			});
		const cliente = { nombre: s('nombre'), telefono: s('telefono'), email: s('email') };

		// Turno fijo: primero la vista previa, después la serie con las fechas aceptadas.
		if (cada > 0) {
			if (!s('profesionalId')) return fail(400, { error: 'Para un turno fijo elegí quién lo atiende.', codigo: 'invalido', campos: {} });
			const api = apiDe(event);
			if (f.get('paso') !== 'reservar') {
				const cuerpo: Record<string, unknown> = { servicioId: s('servicioId'), profesionalId: s('profesionalId'), inicio, cadaSemanas: cada };
				if (esDia(s('hasta'))) cuerpo.hasta = s('hasta');
				else cuerpo.cantidad = Math.min(52, Math.max(2, Number(s('cantidad')) || 4));
				const r = await intentar(() => api.post<PreviaSerie>('/v1/panel/series/previa', cuerpo));
				return 'ok' in r && r.ok ? { previa: r.datos } : r;
			}
			const turnos = f.getAll('turnos').map((x) => ({ inicio: String(x) }));
			if (turnos.length === 0) return fail(400, { error: 'Elegí al menos una fecha.', codigo: 'invalido', campos: {} });
			const r = await intentar(() =>
				api.post<{ id: string; turnos: Turno[] }>('/v1/panel/series', {
					servicioId: s('servicioId'),
					profesionalId: s('profesionalId'),
					cadaSemanas: cada,
					...(s('clienteId') ? { clienteId: s('clienteId') } : { cliente }),
					nota: s('nota'),
					turnos
				})
			);
			if ('ok' in r && r.ok && r.datos.turnos[0]) redirect(303, `/panel/turnos/${r.datos.turnos[0].id}?nuevo=${r.datos.turnos.length}`);
			return r;
		}

		const r = await intentar(() =>
			apiDe(event).post<Turno>('/v1/panel/turnos', {
				servicioId: s('servicioId'),
				profesionalId: s('profesionalId'),
				inicio,
				cliente,
				nota: s('nota'),
				forzar: sobreturno
			})
		);
		if ('datos' in r && r.ok) redirect(303, `/panel/turnos/${r.datos.id}?nuevo=1`);
		return r;
	}
};
