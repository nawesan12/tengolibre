import { apiDe, ErrorApi } from '#lib/server/api.ts';
import { intentar } from '#lib/server/acciones.ts';
import type { Grupos, Reporte } from '#lib/cuenta/tipos.ts';
import { fechaValida, ultimos30 } from '#lib/cuenta/util.ts';
import type { Actions, PageServerLoad } from './$types';

/**
 * Sucursales: los grupos son de la persona (no del negocio elegido), van por /v1/grupos sin
 * X-Negocio. El reporte consolidado es del grupo elegido en la URL (o el primero) y el rango, por
 * defecto los últimos 30 días.
 */
export const load: PageServerLoad = async (event) => {
	const api = apiDe(event);
	const grupos = await api.get<Grupos>('/v1/grupos');
	const q = event.url.searchParams;
	const def = ultimos30();
	const desde = fechaValida(q.get('desde')) ?? def.desde;
	const hasta = fechaValida(q.get('hasta')) ?? def.hasta;
	const pedido = q.get('grupo');
	const grupo = grupos.grupos.find((g) => g.id === pedido) ?? grupos.grupos[0];
	let reporte: Reporte | null = null;
	let errorReporte = '';
	if (grupo) {
		try {
			reporte = await api.get<Reporte>(`/v1/grupos/${grupo.id}/reporte?${new URLSearchParams({ desde, hasta })}`);
		} catch (e) {
			if (!(e instanceof ErrorApi) || e.estado >= 500) throw e;
			errorReporte = e.message;
		}
	}
	return { ...grupos, grupoId: grupo?.id ?? null, desde, hasta, reporte, errorReporte };
};

const ID = /^[0-9a-f-]{36}$/;
const ids = (f: FormData, ...k: string[]) => k.map((x) => String(f.get(x) ?? '')).every((v) => ID.test(v));

export const actions: Actions = {
	crear: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).post('/v1/grupos', { nombre: String(f.get('nombre') ?? '') }));
	},
	renombrar: async (event) => {
		const f = await event.request.formData();
		if (!ids(f, 'id')) return { ok: false };
		return intentar(() => apiDe(event).patch(`/v1/grupos/${f.get('id')}`, { nombre: String(f.get('nombre') ?? '') }));
	},
	borrar: async (event) => {
		const f = await event.request.formData();
		if (!ids(f, 'id')) return { ok: false };
		return intentar(() => apiDe(event).borrar(`/v1/grupos/${f.get('id')}`));
	},
	sumar: async (event) => {
		const f = await event.request.formData();
		if (!ids(f, 'id', 'negocio')) return { ok: false };
		return intentar(() => apiDe(event).put(`/v1/grupos/${f.get('id')}/sucursales/${f.get('negocio')}`, {}));
	},
	sacar: async (event) => {
		const f = await event.request.formData();
		if (!ids(f, 'id', 'negocio')) return { ok: false };
		return intentar(() => apiDe(event).borrar(`/v1/grupos/${f.get('id')}/sucursales/${f.get('negocio')}`));
	}
};
