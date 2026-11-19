import { redirect } from '@sveltejs/kit';
import { apiDe } from '#lib/server/api.ts';
import { accion } from '#lib/configuracion/acciones.server.ts';
import {
	SUSCRIPCION_CAMBIAR,
	SUSCRIPCION_EXTRAS,
	SUSCRIPCION_PAGAR,
	SUSCRIPCION_REACTIVAR,
	SUSCRIPCION_TARJETA,
	SUSCRIPCION_WHATSAPP
} from '#lib/gestion/api-cobros.ts';
import type { Actions, PageServerLoad } from './$types';

/** Un plan como lo vende la API (apps/api/internal/dominio/planes): la única fuente de precios y topes. */
export type PlanVenta = {
	clave: string;
	nombre: string;
	precio: number;
	precioAnual: number;
	destacado: boolean;
	bajada: string;
	profesionales: number;
	sucursales: number;
	whatsappIncluidos: number;
	cupoIA: number;
	funciones: string[];
	incluye: string[];
};
export type Extras = { precioProfesional: number; packWhatsApp: { cantidad: number; precio: number }; mesesAnual: number; diasDePrueba: number };
export type Uso = {
	profesionalesActivos: number;
	topeProfesionales: number;
	profesionalesExtra: number;
	whatsappIncluidos: number;
	whatsappUsadosMes: number;
	whatsappSaldo: number;
	cupoIA: number;
};
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
	/** mercadopago (la web), apple (la App Store) o google (Google Play). */
	origen?: 'mercadopago' | 'apple' | 'google';
	periodo?: 'mensual' | 'anual';
	profesionalesExtra?: number;
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
	/** Respuestas con IA del asistente por mes del plan que rige. */
	cupoIA?: number;
	extras: Extras;
	uso: Uso;
};

export const load: PageServerLoad = async (event) => {
	const estado = await apiDe(event).get<Estado>('/v1/panel/suscripcion');
	// Cancelada con el mes pago, la API la deja «pausada» hasta el próximo cobro (con canceladaEn):
	// para la pantalla es una cancelada que todavía anda, no un cobro fallido.
	const s = estado.suscripcion;
	if (s?.estado === 'pausada' && s.canceladaEn) estado.suscripcion = { ...s, estado: 'cancelada', proximoCobro: s.graciaHasta ?? s.proximoCobro };
	const resultado = event.url.searchParams.get('resultado');
	return { estado, vuelta: resultado === 'vuelta', vueltaWhatsApp: resultado === 'whatsapp' };
};

export const actions: Actions = {
	suscribir: async (event) => {
		const f = await event.request.formData();
		const r = await accion('suscribir', '', () =>
			apiDe(event).post<{ suscripcionId: string; url: string }>('/v1/panel/suscripcion', {
				plan: String(f.get('plan') ?? ''),
				periodo: String(f.get('periodo') ?? 'mensual')
			})
		);
		// A Mercado Pago a cargar la tarjeta; vuelve a /panel/suscripcion?resultado=vuelta.
		if ('ok' in r && r.ok) redirect(303, r.datos.url, { external: true });
		return r;
	},
	cancelar: async (event) =>
		accion('cancelar', 'Cancelamos la suscripción.', () => apiDe(event).post('/v1/panel/suscripcion/cancelar')),
	// Ver #lib/gestion/api-cobros.ts: cambiar de plan, pagar la cuota que falló, cambiar la tarjeta y
	// reactivar una cancelada. Los tres últimos llevan a Mercado Pago.
	cambiar: async (event) => {
		const f = await event.request.formData();
		const r = await accion('suscribir', 'Cambiamos tu plan: el precio nuevo rige desde el próximo cobro.', () =>
			apiDe(event).post<{ url?: string }>(SUSCRIPCION_CAMBIAR, { plan: String(f.get('plan') ?? ''), periodo: String(f.get('periodo') ?? '') || undefined })
		);
		// Cambiar de período es un alta nueva en Mercado Pago.
		if ('ok' in r && r.ok && r.datos?.url) redirect(303, r.datos.url, { external: true });
		return r;
	},
	extras: async (event) => {
		const f = await event.request.formData();
		const cantidad = Math.max(0, Math.min(100, Math.trunc(Number(f.get('cantidad') ?? 0)) || 0));
		return accion('extras', 'Listo: el tope de profesionales ya cambió y el precio nuevo rige desde el próximo cobro.', () =>
			apiDe(event).put(SUSCRIPCION_EXTRAS, { cantidad })
		);
	},
	whatsapp: async (event) => irAMercadoPago(event, 'whatsapp', SUSCRIPCION_WHATSAPP),
	pagar: async (event) => irAMercadoPago(event, 'pagar', SUSCRIPCION_PAGAR),
	tarjeta: async (event) => irAMercadoPago(event, 'pagar', SUSCRIPCION_TARJETA),
	reactivar: async (event) => irAMercadoPago(event, 'cancelar', SUSCRIPCION_REACTIVAR)
};

async function irAMercadoPago(event: Parameters<Actions[string]>[0], que: string, ruta: string) {
	const r = await accion(que, '', () => apiDe(event).post<{ url: string }>(ruta, {}));
	if ('ok' in r && r.ok) redirect(303, r.datos.url, { external: true });
	return r;
}
