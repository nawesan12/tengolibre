import { apiDe } from '#lib/server/api.ts';
import { intentar, numero } from '#lib/server/acciones.ts';
import { conSesion } from '#lib/cuenta/ingreso.server.ts';
import type { ListadoAdmin } from '#lib/cuenta/tipos.ts';
import type { Actions, PageServerLoad } from './$types';

const ID = /^[0-9a-f-]{36}$/;

export const load: PageServerLoad = async (event) => {
	const q = (event.url.searchParams.get('q') ?? '').trim().slice(0, 120);
	const pagina = Math.max(1, Number(event.url.searchParams.get('pagina')) || 1);
	const listado = await conSesion(
		event,
		(api) => api.get<ListadoAdmin>(`/v1/admin/negocios?${new URLSearchParams({ q, pagina: String(pagina) })}`),
		true
	);
	return { q, listado };
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
