import { apiDe } from '#lib/server/api.ts';
import { accion } from '#lib/configuracion/acciones.server.ts';
import type { Actions, PageServerLoad } from './$types';

export type Registro = { tipo: string; nombre: string; valor: string; alternativa?: boolean };
export type Dominio = {
	nombre: string;
	estado: 'pendiente' | 'verificado' | 'error';
	detalle: string;
	creadoEn: string;
	chequeadoEn: string | null;
	verificadoEn: string | null;
	registros: Registro[];
};

export const load: PageServerLoad = async (event) => {
	const r = await apiDe(event).get<{ dominio: Dominio | null; destino: string; ip: string }>('/v1/panel/dominio');
	return { dominio: r.dominio, destino: r.destino, ip: r.ip };
};

export const actions: Actions = {
	poner: async (event) => {
		const f = await event.request.formData();
		return accion('dominio', 'Anotamos el dominio. Cargá los registros en tu proveedor y verificalo.', () =>
			apiDe(event).put('/v1/panel/dominio', { dominio: String(f.get('dominio') ?? '').trim() })
		);
	},
	verificar: async (event) => {
		const r = await accion('dominio', '', () => apiDe(event).post<{ dominio: Dominio }>('/v1/panel/dominio/verificar'));
		if ('ok' in r && r.ok)
			r.mensaje =
				r.datos.dominio.estado === 'verificado'
					? 'Listo: tu dominio ya muestra tu página.'
					: 'Todavía no lo vemos. Los cambios de DNS pueden tardar unas horas.';
		return r;
	},
	quitar: async (event) =>
		accion('dominio', 'Sacamos el dominio: tu página sigue en la dirección de TengoLibre.', () => apiDe(event).borrar('/v1/panel/dominio'))
};
