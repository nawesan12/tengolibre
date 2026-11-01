import { redirect } from '@sveltejs/kit';
import { apiDe } from '#lib/server/api.ts';
import { accion } from '#lib/configuracion/acciones.server.ts';
import { SUSCRIPCION_CAMBIAR, SUSCRIPCION_PAGAR, SUSCRIPCION_REACTIVAR, SUSCRIPCION_TARJETA } from '#lib/gestion/api-cobros.ts';
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
	// Cancelada con el mes pago, la API la deja «pausada» hasta el próximo cobro (con canceladaEn):
	// para la pantalla es una cancelada que todavía anda, no un cobro fallido.
	const s = estado.suscripcion;
	if (s?.estado === 'pausada' && s.canceladaEn) estado.suscripcion = { ...s, estado: 'cancelada', proximoCobro: s.graciaHasta ?? s.proximoCobro };
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
		accion('cancelar', 'Cancelamos la suscripción.', () => apiDe(event).post('/v1/panel/suscripcion/cancelar')),
	// Ver #lib/gestion/api-cobros.ts: cambiar de plan, pagar la cuota que falló, cambiar la tarjeta y
	// reactivar una cancelada. Los tres últimos llevan a Mercado Pago.
	cambiar: async (event) => {
		const f = await event.request.formData();
		return accion('suscribir', 'Cambiamos tu plan: el precio nuevo rige desde el próximo cobro.', () =>
			apiDe(event).post(SUSCRIPCION_CAMBIAR, { plan: String(f.get('plan') ?? '') })
		);
	},
	pagar: async (event) => irAMercadoPago(event, 'pagar', SUSCRIPCION_PAGAR),
	tarjeta: async (event) => irAMercadoPago(event, 'pagar', SUSCRIPCION_TARJETA),
	reactivar: async (event) => irAMercadoPago(event, 'cancelar', SUSCRIPCION_REACTIVAR)
};

async function irAMercadoPago(event: Parameters<Actions[string]>[0], que: string, ruta: string) {
	const r = await accion(que, '', () => apiDe(event).post<{ url: string }>(ruta, {}));
	if ('ok' in r && r.ok) redirect(303, r.datos.url);
	return r;
}
