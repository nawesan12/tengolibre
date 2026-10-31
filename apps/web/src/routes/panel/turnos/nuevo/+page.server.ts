import { fail, redirect } from '@sveltejs/kit';
import { apiDe, ErrorApi } from '#lib/server/api.ts';
import { intentar } from '#lib/server/acciones.ts';
import { dia } from '#lib/formato.ts';
import { esDia, esHora, instante } from '#lib/operacion/fechas.ts';
import { catalogo, enc, param, siSePuede } from '#lib/operacion/servidor.ts';
import type { Ficha, HorarioLibre, Turno } from '#lib/operacion/tipos.ts';
import type { Actions, PageServerLoad } from './$types';

/**
 * Turno manual: se elige servicio, profesional y día (un formulario GET, así los horarios libres
 * salen de /v1/panel/libres con la página), y después el horario o un sobreturno a mano.
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

	return { servicios, profesionales, todos: cat.profesionales, servicio, profesional, fecha, horarios, errorLibres, cliente };
};

export const actions: Actions = {
	crear: async (event) => {
		const f = await event.request.formData();
		const s = (k: string) => String(f.get(k) ?? '').trim();
		const sobreturno = f.get('sobreturno') === 'on';
		let inicio = s('inicio');
		if (sobreturno) {
			if (!s('profesionalId')) return fail(400, { error: 'Para un sobreturno elegí el profesional.', codigo: 'invalido', campos: {} });
			if (!esDia(s('fecha')) || !esHora(s('hora')))
				return fail(400, { error: 'Para un sobreturno poné el día y la hora.', codigo: 'invalido', campos: { hora: 'requerido' } });
			inicio = instante(s('fecha'), s('hora'));
		}
		if (!inicio) return fail(400, { error: 'Elegí un horario.', codigo: 'invalido', campos: { inicio: 'requerido' } });
		const r = await intentar(() =>
			apiDe(event).post<Turno>('/v1/panel/turnos', {
				servicioId: s('servicioId'),
				profesionalId: s('profesionalId'),
				inicio,
				cliente: { nombre: s('nombre'), telefono: s('telefono'), email: s('email') },
				nota: s('nota'),
				forzar: sobreturno
			})
		);
		if ('datos' in r && r.ok) redirect(303, `/panel/turnos/${r.datos.id}?nuevo=1`);
		return r;
	}
};
