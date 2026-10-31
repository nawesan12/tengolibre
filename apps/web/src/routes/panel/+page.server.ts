import { apiDe } from '#lib/server/api.ts';
import { intentar } from '#lib/server/acciones.ts';
import { enc, siExiste, siSePuede } from '#lib/operacion/servidor.ts';
import type { Anotado, Resena } from '#lib/operacion/tipos.ts';
import type { Actions, PageServerLoad } from './$types';

type Paso = { clave: string; titulo: string; listo: boolean; obligatorio: boolean };

/**
 * Inicio: los turnos y profesionales de hoy ya vienen del layout («El día»); acá se suma lo que
 * necesita atención (reseñas sin responder, la lista de espera) y los primeros pasos mientras falten.
 */
export const load: PageServerLoad = async (event) => {
	event.depends('panel:turnos', 'panel:espera', 'panel:resenas', 'panel:pagos');
	const { negocio } = await event.parent();
	const api = apiDe(event);
	const gestiona = ['duenio', 'encargado'].includes(negocio.rol);
	const [arranque, resenas, espera, mp] = await Promise.all([
		api.get<{ completo: boolean; reservable: boolean; pasos: Paso[] }>('/v1/panel/arranque'),
		negocio.rol === 'recepcion' ? null : siSePuede(api.get<{ resenas: Resena[] }>('/v1/panel/resenas')).catch(() => null),
		negocio.rol === 'profesional' ? null : siSePuede(api.get<{ espera: Anotado[] }>('/v1/panel/espera')).catch(() => null),
		gestiona ? siExiste(api.get<{ disponible: boolean; conectada: boolean }>('/v1/panel/mercadopago')).catch(() => null) : null
	]);
	const sinResponder = (resenas?.resenas ?? []).filter((r) => !r.respondidaEn && !r.respuesta);
	return {
		arranque,
		resena: sinResponder[0] ?? null,
		espera: espera?.espera ?? [],
		// null: no se sabe o Mercado Pago no está disponible (no se ofrece conectarlo).
		mpConectado: mp && mp.disponible ? mp.conectada : null,
		gestiona
	};
};

export const actions: Actions = {
	/** «Recordarle» la seña a quien todavía no pagó (contrato nuevo). */
	recordar: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).post(`/v1/panel/turnos/${enc(f.get('id'))}/recordar-pago`));
	}
};
