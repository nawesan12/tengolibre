import { redirect } from '@sveltejs/kit';
import { apiDe } from '#lib/server/api.ts';
import { intentar } from '#lib/server/acciones.ts';
import { enc, siExiste, siSePuede } from '#lib/operacion/servidor.ts';
import type { ParaLlamar, PedidoDias } from '#lib/operacion/contrato.ts';
import type { Anotado, Resena } from '#lib/operacion/tipos.ts';
import type { Actions, PageServerLoad } from './$types';

type Paso = { clave: string; titulo: string; listo: boolean; obligatorio: boolean };

/**
 * Inicio: los turnos y profesionales de hoy ya vienen del layout («El día»); acá se suma lo que
 * necesita atención (reseñas sin responder, la lista de espera, a quién llamar, pedidos de días
 * libres), los primeros pasos mientras falten y si este navegador ya recibe los avisos.
 */
export const load: PageServerLoad = async (event) => {
	event.depends('panel:turnos', 'panel:espera', 'panel:resenas', 'panel:pagos', 'panel:dias_libres');
	const { negocio } = await event.parent();
	const api = apiDe(event);
	const gestiona = ['duenio', 'encargado'].includes(negocio.rol);
	const nada = <T>(p: Promise<T>) => siExiste(p).catch(() => null);
	const [arranque, resenas, espera, mp, llamar, pedidos, push] = await Promise.all([
		api.get<{ completo: boolean; reservable: boolean; pasos: Paso[] }>('/v1/panel/arranque'),
		negocio.rol === 'recepcion' ? null : siSePuede(api.get<{ resenas: Resena[] }>('/v1/panel/resenas')).catch(() => null),
		negocio.rol === 'profesional' ? null : siSePuede(api.get<{ espera: Anotado[] }>('/v1/panel/espera')).catch(() => null),
		gestiona ? nada(api.get<{ disponible: boolean; conectada: boolean }>('/v1/panel/mercadopago')) : null,
		nada(api.get<{ turnos: ParaLlamar[] }>('/v1/panel/para-llamar')),
		gestiona ? nada(api.get<{ pedidos: PedidoDias[] }>('/v1/panel/dias-libres?estado=pendiente')) : null,
		nada(api.get<{ clave: string; suscripta: boolean }>('/v1/yo/suscripciones-web'))
	]);
	const sinResponder = (resenas?.resenas ?? []).filter((r) => !r.respondidaEn && !r.respuesta);
	return {
		arranque,
		resena: sinResponder[0] ?? null,
		espera: espera?.espera ?? [],
		// null: no se sabe o Mercado Pago no está disponible (no se ofrece conectarlo).
		mpConectado: mp && mp.disponible ? mp.conectada : null,
		paraLlamar: llamar?.turnos ?? [],
		pedidos: pedidos?.pedidos ?? [],
		// La tarjeta «Enterate en el momento» sólo si el push web está prendido y este navegador no está.
		push: push && push.clave && !push.suscripta ? { clave: push.clave } : null,
		gestiona
	};
};

export const actions: Actions = {
	/** «Aprobar y ver turnos» un pedido de días libres (dias-libres.md). */
	aprobar: async (event) => {
		const f = await event.request.formData();
		const r = await intentar(() => apiDe(event).post<{ reprogramar: unknown[] }>(`/v1/panel/dias-libres/${enc(f.get('id'))}/aprobar`, {}));
		if ('ok' in r && r.ok) redirect(303, '/panel/turnos');
		return r;
	},
	rechazar: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).post(`/v1/panel/dias-libres/${enc(f.get('id'))}/rechazar`, {}));
	}
};
