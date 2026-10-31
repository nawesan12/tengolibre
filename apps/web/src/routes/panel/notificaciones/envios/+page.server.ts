import { apiDe } from '#lib/server/api.ts';
import { intentar } from '#lib/server/acciones.ts';
import { enc, param, siSePuede } from '#lib/operacion/servidor.ts';
import type { RegistroAviso } from '#lib/operacion/tipos.ts';
import type { Actions, PageServerLoad } from './$types';

const ESTADOS = ['pendiente', 'enviado', 'fallido', 'descartado'];

/** El registro de mails que salieron (o fallaron), con reenviar. Es del mostrador, no del profesional. */
export const load: PageServerLoad = async (event) => {
	event.depends('panel:notificaciones', 'panel:turnos');
	const estado = ESTADOS.includes(param(event.url, 'estado')) ? param(event.url, 'estado') : '';
	const antes = param(event.url, 'antes');
	const q = new URLSearchParams();
	if (estado) q.set('estado', estado);
	if (antes && !Number.isNaN(Date.parse(antes))) q.set('antes', antes);
	const r = await siSePuede(apiDe(event).get<{ avisos: RegistroAviso[] }>(`/v1/panel/avisos?${q}`));
	return { avisos: r?.avisos ?? [], prohibido: !r, estado, antes: q.get('antes') ?? '' };
};

export const actions: Actions = {
	reenviar: async (event) => {
		const f = await event.request.formData();
		return intentar(() => apiDe(event).post(`/v1/panel/avisos/${enc(f.get('id'))}/reenviar`));
	}
};
