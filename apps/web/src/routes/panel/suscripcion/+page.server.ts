import { redirect } from '@sveltejs/kit';
import { apiDe } from '#lib/server/api.ts';
import { accion } from '#lib/configuracion/acciones.server.ts';
import type { Actions, PageServerLoad } from './$types';

export type PlanVenta = { clave: string; nombre: string; precio: number };
export type Suscripcion = {
	id: string;
	plan: string;
	estado: string;
	monto: number;
	link: string | null;
	proximoCobro: string | null;
	ultimoCobro: string | null;
	graciaHasta: string | null;
	canceladaEn: string | null;
	creadoEn: string;
};
export type Estado = {
	bloqueado: boolean;
	pruebaHasta: string | null;
	diasDePrueba: number;
	conSuscripcion: boolean;
	plan: string;
	nombrePlan: string;
	catalogo: PlanVenta[];
	suscripcion: Suscripcion | null;
};

export const load: PageServerLoad = async (event) => {
	const estado = await apiDe(event).get<Estado>('/v1/panel/suscripcion');
	return { estado, vuelta: event.url.searchParams.get('resultado') === 'vuelta' };
};

export const actions: Actions = {
	suscribir: async (event) => {
		const f = await event.request.formData();
		const r = await accion('suscribir', '', () =>
			apiDe(event).post<{ suscripcionId: string; url: string }>('/v1/panel/suscripcion', { plan: String(f.get('plan') ?? '') })
		);
		// A Mercado Pago a cargar la tarjeta; vuelve a /panel/suscripcion?resultado=vuelta.
		if ('ok' in r && r.ok) redirect(303, r.datos.url);
		return r;
	},
	cancelar: async (event) =>
		accion('cancelar', 'Cancelamos la suscripción.', () => apiDe(event).post('/v1/panel/suscripcion/cancelar'))
};
