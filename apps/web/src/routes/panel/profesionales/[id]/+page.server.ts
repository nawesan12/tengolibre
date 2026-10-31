import { error, redirect } from '@sveltejs/kit';
import { apiDe, type Api } from '#lib/server/api.ts';
import { numero } from '#lib/server/acciones.ts';
import { accion, siPuede } from '#lib/configuracion/acciones.server.ts';
import { horarioDeForm, sacarSemanal, sumarSemanal, type Semanal } from '#lib/configuracion/horario.ts';
import { dia } from '#lib/formato.ts';
import { sumarDias } from '#lib/operacion/fechas.ts';
import type { Turno } from '#lib/operacion/tipos.ts';
import { diasEntre } from '#lib/gestion/equipo/semana.ts';
import { pedidosLibres, pedirLibres, resolverPedido } from '#lib/gestion/api-equipo.ts';
import type { Miembro, Profesional } from '../../equipo/+page.server.ts';
import type { Actions, PageServerLoad } from './$types';

type Bloqueo = { id: string; fecha: string; desde: string | null; hasta: string | null; motivo?: string };
type Excepcion = { fecha: string; hora: string; tipo: 'bloqueado' | 'libre' };

const semanalesDe = async (api: Api, id: string) =>
	(await api.get<{ semanales: Semanal[] }>(`/v1/panel/profesionales/${id}/semanales`)).semanales;

export const load: PageServerLoad = async (event) => {
	const api = apiDe(event);
	const id = event.params.id;
	const hoy = dia();
	const [{ profesionales }, { servicios }, { bloqueos }, semanales, { excepciones }, equipo, turnos, pedidos] = await Promise.all([
		api.get<{ profesionales: Profesional[] }>('/v1/panel/profesionales'),
		api.get<{ servicios: { id: string; nombre: string; activo: boolean }[] }>('/v1/panel/servicios'),
		api.get<{ bloqueos: Bloqueo[] }>(`/v1/panel/profesionales/${id}/bloqueos`),
		semanalesDe(api, id),
		api.get<{ excepciones: Excepcion[] }>(`/v1/panel/profesionales/${id}/excepciones`),
		siPuede(api.get<{ miembros: Miembro[] }>('/v1/panel/equipo')),
		// Los turnos que vienen (hasta 90 días): para avisar cuáles toca un bloqueo antes de guardarlo.
		api
			.get<{ turnos: Turno[] }>(`/v1/panel/turnos?${new URLSearchParams({ desde: hoy, hasta: sumarDias(hoy, 90), profesional: id })}`)
			.then((r) => r.turnos.filter((t) => t.estado !== 'cancelado'))
			.catch(() => [] as Turno[]),
		pedidosLibres(api, id)
	]);
	const profesional = profesionales.find((p) => p.id === id);
	if (!profesional) error(404, 'No encontramos ese profesional.');
	const proximos = turnos.map((t) => ({ id: t.id, inicio: t.inicio, cliente: t.cliente.nombre, servicio: t.servicio }));
	return { profesional, profesionales, servicios, bloqueos, semanales, excepciones, miembros: equipo?.miembros ?? null, proximos, pedidos };
};

const base = (id: string) => `/v1/panel/profesionales/${id}`;

export const actions: Actions = {
	datos: async (event) => {
		const f = await event.request.formData();
		return accion('datos', 'Guardamos los datos.', () =>
			apiDe(event).patch(base(event.params.id), {
				nombre: String(f.get('nombre') ?? ''),
				especialidad: String(f.get('especialidad') ?? ''),
				bio: String(f.get('bio') ?? ''),
				email: String(f.get('email') ?? ''),
				telefono: String(f.get('telefono') ?? ''),
				orden: numero(f.get('orden')) ?? 0,
				activo: f.get('activo') === 'on',
				...(f.has('miembroId') ? { miembroId: String(f.get('miembroId')) } : {})
			})
		);
	},
	borrar: async (event) => {
		const r = await accion('datos', '', () => apiDe(event).borrar(base(event.params.id)));
		if ('ok' in r && r.ok) redirect(303, '/panel/equipo');
		return r;
	},
	servicios: async (event) => {
		const f = await event.request.formData();
		return accion('servicios', 'Guardamos los servicios que hace.', () =>
			apiDe(event).put(`${base(event.params.id)}/servicios`, { servicios: f.getAll('servicios').map(String) })
		);
	},
	horario: async (event) => {
		const f = await event.request.formData();
		return accion('horario', 'Guardamos el horario.', () =>
			apiDe(event).put(`${base(event.params.id)}/horario`, { horario: horarioDeForm(f) })
		);
	},
	bloqueo: async (event) => {
		const f = await event.request.formData();
		const entero = f.get('entero') === 'on';
		const fecha = String(f.get('fecha') ?? '');
		// Con «hasta el día», un bloqueo por cada fecha del rango (la API los guarda de a uno).
		const hastaDia = String(f.get('hastaDia') ?? '');
		const fechas = hastaDia && hastaDia > fecha ? diasEntre(fecha, hastaDia) : [fecha];
		const api = apiDe(event);
		return accion('bloqueos', fechas.length > 1 ? `Bloqueamos esos ${fechas.length} días.` : 'Bloqueamos esa fecha.', async () => {
			for (const d of fechas)
				await api.post(`${base(event.params.id)}/bloqueos`, {
					fecha: d,
					desde: entero ? null : String(f.get('desde') ?? '') || null,
					hasta: entero ? null : String(f.get('hasta') ?? '') || null,
					motivo: String(f.get('motivo') ?? '')
				});
			return { fechas };
		});
	},
	borrarBloqueo: async (event) => {
		const f = await event.request.formData();
		// Un talón puede juntar varios días seguidos: se sacan todos.
		const ids = f.getAll('bloqueo').map(String);
		const api = apiDe(event);
		return accion('bloqueos', ids.length > 1 ? 'Sacamos esos días bloqueados.' : 'Sacamos el bloqueo.', async () => {
			for (const id of ids) await api.borrar(`${base(event.params.id)}/bloqueos/${id}`);
			return null;
		});
	},
	semanal: async (event) => {
		const f = await event.request.formData();
		const api = apiDe(event);
		return accion('semanales', 'Sumamos el bloqueo semanal.', async () => {
			const lista = sumarSemanal(await semanalesDe(api, event.params.id), {
				dia: Number(f.get('dia')),
				hora: String(f.get('hora') ?? '').slice(0, 5)
			});
			return api.put(`${base(event.params.id)}/semanales`, { semanales: lista });
		});
	},
	borrarSemanal: async (event) => {
		const f = await event.request.formData();
		const api = apiDe(event);
		return accion('semanales', 'Sacamos el bloqueo semanal.', async () => {
			const lista = sacarSemanal(await semanalesDe(api, event.params.id), Number(f.get('dia')), String(f.get('hora')));
			return api.put(`${base(event.params.id)}/semanales`, { semanales: lista });
		});
	},
	excepcion: async (event) => {
		const f = await event.request.formData();
		return accion('excepciones', 'Guardamos la excepción.', () =>
			apiDe(event).put(`${base(event.params.id)}/excepciones`, {
				fecha: String(f.get('fecha') ?? ''),
				hora: String(f.get('hora') ?? '').slice(0, 5),
				tipo: String(f.get('tipo') ?? 'bloqueado')
			})
		);
	},
	pedirLibres: async (event) => {
		const f = await event.request.formData();
		const desde = String(f.get('desde') ?? '');
		return accion('pedidos', 'Mandamos el pedido. Te avisamos cuando lo respondan.', () =>
			pedirLibres(apiDe(event), event.params.id, {
				desde,
				hasta: String(f.get('hasta') ?? '') || desde,
				motivo: String(f.get('motivo') ?? '').trim()
			})
		);
	},
	resolverPedido: async (event) => {
		const f = await event.request.formData();
		const estado = f.get('estado') === 'aprobado' ? 'aprobado' : 'rechazado';
		return accion('pedidos', estado === 'aprobado' ? 'Aprobamos el pedido: esos días quedan bloqueados.' : 'Rechazamos el pedido.', () =>
			resolverPedido(apiDe(event), event.params.id, String(f.get('pedido')), estado)
		);
	},
	borrarExcepcion: async (event) => {
		const f = await event.request.formData();
		return accion('excepciones', 'Sacamos la excepción.', () =>
			apiDe(event).borrar(`${base(event.params.id)}/excepciones/${f.get('fecha')}/${f.get('hora')}`)
		);
	}
};
