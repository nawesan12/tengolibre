import { apiDe } from '#lib/server/api.ts';
import { numero } from '#lib/server/acciones.ts';
import { accion } from '#lib/configuracion/acciones.server.ts';
import { beneficiosDe } from '#lib/configuracion/dinero.ts';
import type { Actions, PageServerLoad } from './$types';

export type Plan = {
	id: string;
	nombre: string;
	descripcion: string;
	precio: number;
	visitas: number;
	duracionDias: number;
	prioridadDias: number;
	beneficios: string[];
	topeSocios: number | null;
	servicios: string[];
	activo: boolean;
	orden: number;
};
export type Socio = {
	id: string;
	planId: string;
	plan: string;
	visitasDelPlan: number;
	clienteId: string;
	cliente: string;
	telefono: string;
	desde: string;
	hasta: string;
	estado: string;
	quedan: number;
	nota: string;
};

const ESTADOS = ['activa', 'vencida', 'cancelada', 'todas'];

export const load: PageServerLoad = async (event) => {
	const api = apiDe(event);
	const pedido = event.url.searchParams.get('estado') ?? 'activa';
	const estado = ESTADOS.includes(pedido) ? pedido : 'activa';
	const [{ planes }, { socios }, { servicios }] = await Promise.all([
		api.get<{ planes: Plan[] }>('/v1/panel/membresias/planes'),
		api.get<{ socios: Socio[] }>(`/v1/panel/membresias/socios?estado=${estado}`),
		api.get<{ servicios: { id: string; nombre: string }[] }>('/v1/panel/servicios')
	]);
	return { planes, socios, servicios, estado };
};

function planDe(f: FormData) {
	const tope = numero(f.get('topeSocios'));
	return {
		nombre: String(f.get('nombre') ?? '').trim(),
		descripcion: String(f.get('descripcion') ?? '').trim(),
		precio: numero(f.get('precio')) ?? 0,
		visitas: numero(f.get('visitas')) ?? 0,
		duracionDias: numero(f.get('duracionDias')) ?? 30,
		prioridadDias: numero(f.get('prioridadDias')) ?? 0,
		beneficios: beneficiosDe(f.get('beneficios')),
		topeSocios: tope && tope > 0 ? tope : null,
		servicios: f.getAll('servicios').map(String),
		activo: f.get('activo') === 'on'
	};
}

const socio = (f: FormData) => `/v1/panel/membresias/socios/${f.get('id')}`;

export const actions: Actions = {
	crearPlan: async (event) => {
		const f = await event.request.formData();
		return accion('planes', 'Creamos el plan.', () => apiDe(event).post('/v1/panel/membresias/planes', planDe(f)));
	},
	editarPlan: async (event) => {
		const f = await event.request.formData();
		return accion('planes', 'Guardamos el plan.', () => apiDe(event).patch(`/v1/panel/membresias/planes/${f.get('id')}`, planDe(f)));
	},
	borrarPlan: async (event) => {
		const f = await event.request.formData();
		return accion('planes', 'Sacamos el plan.', () => apiDe(event).borrar(`/v1/panel/membresias/planes/${f.get('id')}`));
	},
	alta: async (event) => {
		const f = await event.request.formData();
		return accion('socios', 'Sumamos al socio.', () =>
			apiDe(event).post('/v1/panel/membresias/socios', {
				planId: String(f.get('planId') ?? ''),
				nombre: String(f.get('nombre') ?? '').trim(),
				telefono: String(f.get('telefono') ?? '').trim(),
				email: String(f.get('email') ?? '').trim(),
				nota: String(f.get('nota') ?? '').trim()
			})
		);
	},
	ajustar: async (event) => {
		const f = await event.request.formData();
		const ajustar = numero(f.get('ajustar')) ?? 0;
		return accion('socios', ajustar >= 0 ? 'Sumamos las visitas.' : 'Restamos las visitas.', () =>
			apiDe(event).patch(socio(f), { ajustar, motivo: String(f.get('motivo') ?? '') })
		);
	},
	renovar: async (event) => {
		const f = await event.request.formData();
		return accion('socios', 'Renovamos la membresía desde hoy.', () => apiDe(event).patch(socio(f), { renovar: true }));
	},
	estado: async (event) => {
		const f = await event.request.formData();
		const estado = String(f.get('estado'));
		return accion('socios', estado === 'cancelada' ? 'Cancelamos la membresía.' : 'Reactivamos la membresía.', () =>
			apiDe(event).patch(socio(f), { estado })
		);
	}
};
