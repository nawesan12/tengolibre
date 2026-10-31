import { apiDe, ErrorApi } from '#lib/server/api.ts';
import { intentar } from '#lib/server/acciones.ts';
import { dia } from '#lib/formato.ts';
import { esDia, sumarDias } from '#lib/operacion/fechas.ts';
import { cambiarEstado, catalogo, cobrar, enc, param, siExiste, siSePuede } from '#lib/operacion/servidor.ts';
import type { Oferta, Producto } from '#lib/operacion/contrato.ts';
import type { Anotado, Turno } from '#lib/operacion/tipos.ts';
import type { Actions, PageServerLoad } from './$types';

/**
 * Los turnos de un día (handoff «Panel v2», Turnos). Se trae la tira de días alrededor (dos antes y
 * cuatro después) para contar cuántos hay en cada uno; los filtros van en la URL.
 * Compatibilidad: `?desde=` elige el día.
 */
export const load: PageServerLoad = async (event) => {
	event.depends('panel:turnos', 'panel:espera');
	const { url } = event;
	const fecha = esDia(param(url, 'fecha')) ? param(url, 'fecha') : esDia(param(url, 'desde')) ? param(url, 'desde') : dia();
	const desde = sumarDias(fecha, -2);
	const hasta = sumarDias(fecha, 4);
	const api = apiDe(event);
	const [cat, rango, espera, ofertas, productos] = await Promise.all([
		catalogo(api),
		api
			.get<{ turnos: Turno[] }>(`/v1/panel/turnos?${new URLSearchParams({ desde, hasta })}`)
			.then((r) => ({ turnos: r.turnos, error: '' }))
			.catch((e) => {
				if (e instanceof ErrorApi && e.estado < 500) return { turnos: [] as Turno[], error: e.message };
				throw e;
			}),
		siSePuede(api.get<{ espera: Anotado[] }>('/v1/panel/espera')).catch(() => null),
		siExiste(api.get<{ ofertas: Oferta[] }>(`/v1/panel/huecos/ofertas?fecha=${fecha}`)).catch(() => null),
		siExiste(api.get<{ productos: Producto[] }>('/v1/panel/productos')).catch(() => null)
	]);
	return {
		...cat,
		fecha,
		desde,
		hasta,
		turnos: rango.turnos,
		error: rango.error,
		espera: espera?.espera ?? [],
		ofertas: (ofertas?.ofertas ?? []).filter((o) => o.estado === 'ofreciendo'),
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
	/** Ofrecer el hueco al primero de la lista de espera (contrato nuevo, Propuesta 01). */
	ofrecer: async (event) => {
		const f = await event.request.formData();
		const s = (k: string) => String(f.get(k) ?? '');
		return intentar(() =>
			apiDe(event).post<Oferta>('/v1/panel/huecos/ofertas', {
				profesionalId: s('profesionalId'),
				inicio: s('inicio'),
				fin: s('fin'),
				servicioId: s('servicioId') || undefined
			})
		);
	},
	frenar: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).borrar(`/v1/panel/huecos/ofertas/${enc(f.get('id'))}`));
	},
	/** La caja: cobra lo que resta y lo marca atendido (contrato nuevo, Propuestas 07 y 08). */
	cobrar
};
