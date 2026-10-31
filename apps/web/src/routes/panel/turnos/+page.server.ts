import { apiDe, ErrorApi } from '#lib/server/api.ts';
import { intentar } from '#lib/server/acciones.ts';
import { dia, hora } from '#lib/formato.ts';
import { esDia, sumarDias } from '#lib/operacion/fechas.ts';
import { cambiarEstado, catalogo, cobrar, enc, param, siExiste, siSePuede } from '#lib/operacion/servidor.ts';
import type { AReprogramar, Oferta, Producto } from '#lib/operacion/contrato.ts';
import { redirect } from '@sveltejs/kit';
import type { Anotado, Turno } from '#lib/operacion/tipos.ts';
import type { Actions, PageServerLoad } from './$types';

/**
 * Los turnos de un día (handoff «Panel v2», Turnos). Se trae la tira de días alrededor (dos antes y
 * cuatro después) para contar cuántos hay en cada uno; los filtros van en la URL.
 * Compatibilidad: `?desde=` elige el día.
 */
export const load: PageServerLoad = async (event) => {
	event.depends('panel:turnos', 'panel:espera', 'panel:ofertas', 'panel:caja', 'panel:reprogramar');
	const { url } = event;
	const fecha = esDia(param(url, 'fecha')) ? param(url, 'fecha') : esDia(param(url, 'desde')) ? param(url, 'desde') : dia();
	const desde = sumarDias(fecha, -2);
	const hasta = sumarDias(fecha, 4);
	const api = apiDe(event);
	const { negocio } = await event.parent();
	const mostrador = negocio.rol !== 'profesional';
	const [cat, rango, espera, ofertas, productos, reprogramar] = await Promise.all([
		catalogo(api),
		api
			.get<{ turnos: Turno[] }>(`/v1/panel/turnos?${new URLSearchParams({ desde, hasta })}`)
			.then((r) => ({ turnos: r.turnos, error: '' }))
			.catch((e) => {
				if (e instanceof ErrorApi && e.estado < 500) return { turnos: [] as Turno[], error: e.message };
				throw e;
			}),
		siSePuede(api.get<{ espera: Anotado[] }>('/v1/panel/espera')).catch(() => null),
		mostrador ? siExiste(api.get<{ ofertas: Oferta[] }>(`/v1/panel/espera/ofertas?fecha=${fecha}`)).catch(() => null) : null,
		siExiste(api.get<{ productos: Producto[] }>('/v1/panel/productos?activos=1')).catch(() => null),
		mostrador ? siExiste(api.get<{ turnos: AReprogramar[] }>('/v1/panel/reprogramar')).catch(() => null) : null
	]);
	// A cuántos de la lista se les ofrecería cada hueco que dejó una cancelación (de acá en adelante).
	const ahora = Date.now();
	const liberados = mostrador
		? rango.turnos.filter((t) => t.estado === 'cancelado' && t.servicioId && t.profesionalId && dia(t.inicio) === fecha && Date.parse(t.inicio) > ahora)
		: [];
	const candidatos = Object.fromEntries(
		await Promise.all(
			liberados.map(async (t) => {
				const q = new URLSearchParams({ servicio: t.servicioId!, profesional: t.profesionalId!, inicio: t.inicio });
				const r = await siExiste(api.get<{ espera: Anotado[] }>(`/v1/panel/espera/candidatos?${q}`)).catch(() => null);
				return [t.id, r?.espera.length ?? 0] as const;
			})
		)
	) as Record<string, number>;
	return {
		...cat,
		fecha,
		desde,
		hasta,
		turnos: rango.turnos,
		error: rango.error,
		espera: espera?.espera ?? [],
		ofertas: (ofertas?.ofertas ?? []).filter((o) => o.estado === 'ofreciendo'),
		candidatos,
		reprogramar: reprogramar?.turnos ?? [],
		productos: productos?.productos ?? [],
		filtros: {
			profesional: param(url, 'profesional'),
			estado: param(url, 'estado'),
			huecos: param(url, 'huecos') === '1',
			cancelados: param(url, 'cancelados') === '1'
		}
	};
};

export const actions: Actions = {
	estado: cambiarEstado,
	/** Ofrecer el hueco al primero de la lista de espera (espera-ofertas.md). */
	ofrecer: async (event) => {
		const f = await event.request.formData();
		const s = (k: string) => String(f.get(k) ?? '');
		return intentar(() =>
			apiDe(event).post<Oferta>('/v1/panel/espera/ofertas', { servicioId: s('servicioId'), profesionalId: s('profesionalId'), inicio: s('inicio') })
		);
	},
	frenar: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).post(`/v1/panel/espera/ofertas/${enc(f.get('id'))}/frenar`));
	},
	/** «Cargarlo yo»: deja de ofrecerlo y abre Nuevo turno en ese horario. */
	cargar: async (event) => {
		const f = await event.request.formData();
		const r = await intentar(() => apiDe(event).post<{ oferta: Oferta }>(`/v1/panel/espera/ofertas/${enc(f.get('id'))}/cargar`, {}));
		if ('ok' in r && r.ok) {
			const o = r.datos.oferta;
			redirect(303, `/panel/turnos/nuevo?${new URLSearchParams({ servicio: o.servicioId, profesional: o.profesionalId, fecha: dia(o.inicio), hora: hora(o.inicio) })}`);
		}
		return r;
	},
	/** Ya lo llamaron (whatsapp.md): sale de «para llamar». */
	llamado: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).post(`/v1/panel/turnos/${enc(f.get('id'))}/llamado`));
	},
	/** Un turno de la lista para reprogramar que ya se resolvió a mano (dias-libres.md). */
	resolver: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).post(`/v1/panel/reprogramar/${enc(f.get('id'))}/resolver`));
	},
	/** La caja: cobra lo que resta y lo marca atendido (caja.md). */
	cobrar
};
