import { apiDe } from '#lib/server/api.ts';
import { numero } from '#lib/server/acciones.ts';
import { accion } from '#lib/configuracion/acciones.server.ts';
import type { Socio } from '../+page.server.ts';
import type { Actions, PageServerLoad } from './$types';

/** Un movimiento de visitas: negativo es un uso (con su turno) y positivo, una carga o un ajuste. */
export type Credito = { cantidad: number; motivo: string; turnoId: string | null; clerkId: string | null; creadoEn: string };

export const load: PageServerLoad = async (event) => {
	const socio = await apiDe(event).get<Socio & { creditos: Credito[] }>(`/v1/panel/membresias/socios/${event.params.id}`);
	return { socio };
};

const ruta = (id: string) => `/v1/panel/membresias/socios/${id}`;

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
