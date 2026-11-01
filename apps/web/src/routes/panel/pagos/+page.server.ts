import { redirect } from '@sveltejs/kit';
import { apiDe } from '#lib/server/api.ts';
import { numero } from '#lib/server/acciones.ts';
import { accion } from '#lib/configuracion/acciones.server.ts';
import type { Actions, PageServerLoad } from './$types';

export type Conexion = {
	disponible: boolean;
	conectada: boolean;
	usuario?: number;
	desde?: string;
	venceEn?: string;
	reautorizar: boolean;
	ultimoError?: string;
	/** Supuesto (ver #lib/gestion/api-cobros.ts): el nombre de la cuenta conectada. */
	cuenta?: string;
};
export type ConfigCobro = { modo: string; senaPorcentaje: number; senaFija: number };
export type Pago = {
	id: string;
	turnoId: string;
	codigo: string;
	servicio: string;
	cliente: string;
	inicio: string;
	turnoEstado: string;
	monto: number;
	modo: string;
	estado: string;
	devuelto: number;
	mpPago: number | null;
	venceEn: string | null;
	vencido: boolean;
	creadoEn: string;
};

export const load: PageServerLoad = async (event) => {
	event.depends('panel:pagos');
	const api = apiDe(event);
	const estado = event.url.searchParams.get('estado') ?? '';
	const [conexion, config, { pagos }, { servicios }] = await Promise.all([
		api.get<Conexion>('/v1/panel/mercadopago'),
		api.get<ConfigCobro>('/v1/panel/cobros'),
		api.get<{ pagos: Pago[] }>(`/v1/panel/pagos${estado ? `?estado=${encodeURIComponent(estado)}` : ''}`),
		api.get<{ servicios: { nombre: string; precio: number; activo: boolean }[] }>('/v1/panel/servicios')
	]);
	// El ejemplo en plata real de cada forma de cobro: el servicio activo más caro.
	const ejemplo =
		servicios.filter((s) => s.activo && s.precio > 0).sort((a, b) => b.precio - a.precio)[0] ?? null;
	return { conexion, config, pagos, estado, ejemplo: ejemplo && { nombre: ejemplo.nombre, precio: ejemplo.precio } };
};

export const actions: Actions = {
	conectar: async (event) => {
		const r = await accion('mercadopago', '', () => apiDe(event).post<{ url: string }>('/v1/panel/mercadopago/conectar'));
		// A Mercado Pago a autorizar; vuelve a /panel/mercadopago/vuelta.
		if ('ok' in r && r.ok) redirect(303, r.datos.url, { external: true });
		return r;
	},
	desconectar: async (event) =>
		accion('mercadopago', 'Desconectamos Mercado Pago: los turnos nuevos ya no se cobran por adelantado.', () =>
			apiDe(event).borrar('/v1/panel/mercadopago')
		),
	cobro: async (event) => {
		const f = await event.request.formData();
		return accion('cobro', 'Guardamos cómo cobrás.', () =>
			apiDe(event).put('/v1/panel/cobros', {
				modo: String(f.get('modo') ?? 'sin_cobro'),
				senaPorcentaje: numero(f.get('senaPorcentaje')) ?? 0,
				senaFija: numero(f.get('senaFija')) ?? 0
			})
		);
	},
	devolver: async (event) => {
		const f = await event.request.formData();
		const parcial = f.get('cuanto') === 'parte';
		const monto = parcial ? (numero(f.get('monto')) ?? 0) : 0;
		return accion('pagos', parcial ? 'Devolvimos esa parte del pago.' : 'Devolvimos el pago entero y cancelamos el turno.', () =>
			apiDe(event).post(`/v1/panel/pagos/${f.get('id')}/devolver`, monto > 0 ? { monto } : {})
		);
	}
};
