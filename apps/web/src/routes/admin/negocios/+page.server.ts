import { apiDe } from '#lib/server/api.ts';
import { intentar, numero } from '#lib/server/acciones.ts';
import { conSesion } from '#lib/cuenta/ingreso.server.ts';
import type { ListadoAdmin, NegocioAdmin } from '#lib/cuenta/tipos.ts';
import { ESTADOS_NEGOCIO, type ConteosAdmin, type EstadoNegocio, type NegocioAdminExtra } from '#lib/cuenta/contratos.ts';
import type { Actions, PageServerLoad } from './$types';

const ID = /^[0-9a-f-]{36}$/;

type Listado = Omit<ListadoAdmin, 'negocios'> & { negocios: (NegocioAdmin & NegocioAdminExtra)[]; conteos?: ConteosAdmin };

export const load: PageServerLoad = async (event) => {
	const q = (event.url.searchParams.get('q') ?? '').trim().slice(0, 120);
	const pagina = Math.max(1, Number(event.url.searchParams.get('pagina')) || 1);
	const e = event.url.searchParams.get('estado') ?? '';
	// Filtro por estado (contrato propuesto: estado=<EstadoNegocio>).
	const estado = (ESTADOS_NEGOCIO as string[]).includes(e) ? (e as EstadoNegocio) : '';
	const p = new URLSearchParams({ q, pagina: String(pagina) });
	if (estado) p.set('estado', estado);
	const listado = await conSesion(event, (api) => api.get<Listado>(`/v1/admin/negocios?${p}`), true);
	return { q, estado, listado };
};

export const actions: Actions = {
	activo: async (event) => {
		const f = await event.request.formData();
		const id = String(f.get('id') ?? '');
		if (!ID.test(id)) return { ok: false };
		const activo = f.get('activo') === 'true';
		return intentar(() => apiDe(event).patch(`/v1/admin/negocios/${id}`, { activo }));
	},
	extender: async (event) => {
		const f = await event.request.formData();
		const id = String(f.get('id') ?? '');
		if (!ID.test(id)) return { ok: false };
		return intentar(() => apiDe(event).post(`/v1/admin/negocios/${id}/extender-prueba`, { dias: numero(f.get('dias')) ?? 0 }));
	}
};
