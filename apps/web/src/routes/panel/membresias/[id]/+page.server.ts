import { apiDe, ErrorApi } from '#lib/server/api.ts';
import { error } from '@sveltejs/kit';
import { numero } from '#lib/server/acciones.ts';
import { accion } from '#lib/configuracion/acciones.server.ts';
import type { Socio } from '../+page.server.ts';
import type { Actions, PageServerLoad } from './$types';

/** Un movimiento de visitas: negativo es un uso (con su turno) y positivo, una carga o un ajuste. */
export type Credito = { cantidad: number; motivo: string; turnoId: string | null; clerkId: string | null; creadoEn: string };

export const load: PageServerLoad = async (event) => {
	try {
		const socio = await apiDe(event).get<Socio & { creditos: Credito[] }>(ruta(event.params.id));
		return { socio };
	} catch (e) {
		if (e instanceof ErrorApi && [400, 403, 404].includes(e.estado)) error(404, 'No encontramos ese socio.');
		throw e;
	}
};

const ruta = (id: string) => `/v1/panel/membresias/socios/${encodeURIComponent(id)}`;

export const actions: Actions = {
	ajustar: async (event) => {
		const f = await event.request.formData();
		const ajustar = numero(f.get('ajustar')) ?? 0;
		return accion('ajustar', ajustar >= 0 ? 'Sumamos las visitas.' : 'Restamos las visitas.', () =>
			apiDe(event).patch(ruta(event.params.id), { ajustar, motivo: String(f.get('motivo') ?? '').trim() })
		);
	},
	renovar: async (event) =>
		accion('socio', 'Renovamos la membresía desde hoy.', () => apiDe(event).patch(ruta(event.params.id), { renovar: true })),
	estado: async (event) => {
		const f = await event.request.formData();
		const estado = String(f.get('estado'));
		return accion('socio', estado === 'cancelada' ? 'Cancelamos la membresía.' : 'Reactivamos la membresía.', () =>
			apiDe(event).patch(ruta(event.params.id), { estado })
		);
	}
};
