import { apiDe } from '#lib/server/api.ts';
import { accion } from '#lib/configuracion/acciones.server.ts';
import { cuponDeForm } from '#lib/configuracion/dinero.ts';
import type { Actions, PageServerLoad } from './$types';

export type Cupon = {
	id: string;
	codigo: string;
	tipo: 'porcentaje' | 'fijo';
	valor: number;
	minimo: number | null;
	maxUsos: number | null;
	usos: number;
	desde: string;
	hasta: string | null;
	activo: boolean;
	referidoId?: string;
	creadoEn: string;
};

export const load: PageServerLoad = async (event) => {
	const { cupones } = await apiDe(event).get<{ cupones: Cupon[] }>('/v1/panel/cupones');
	return { cupones };
};

export const actions: Actions = {
	crear: async (event) => {
		const f = await event.request.formData();
		return accion('crear', 'Creamos el cupón.', () => apiDe(event).post('/v1/panel/cupones', cuponDeForm(f, true)));
	},
	editar: async (event) => {
		const f = await event.request.formData();
		return accion('cupones', 'Guardamos el cupón.', () => apiDe(event).patch(`/v1/panel/cupones/${f.get('id')}`, cuponDeForm(f, false)));
	},
	activo: async (event) => {
		const f = await event.request.formData();
		const activo = f.get('activo') === 'true';
		return accion('cupones', activo ? 'Activamos el cupón.' : 'Pausamos el cupón.', () =>
			apiDe(event).patch(`/v1/panel/cupones/${f.get('id')}`, { activo })
		);
	},
	borrar: async (event) => {
		const f = await event.request.formData();
		return accion('cupones', 'Sacamos el cupón (si ya se usó, queda pausado).', () =>
			apiDe(event).borrar(`/v1/panel/cupones/${f.get('id')}`)
		);
	}
};
