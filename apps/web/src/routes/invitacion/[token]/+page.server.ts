import { error, redirect, type RequestEvent } from '@sveltejs/kit';
import { apiDe, apiPublica, ErrorApi } from '#lib/server/api.ts';
import { intentar } from '#lib/server/acciones.ts';
import { elegirNegocio } from '#lib/server/sesion.ts';
import { codigoValido, o404, opcional } from '#lib/publico/api.server.ts';
import type { Pagina } from '#lib/publico/tipos.ts';
import type { Invitacion } from '#lib/cuenta/contratos.ts';
import { colorValido } from '#lib/cuenta/presentacion.ts';
import { linkWhatsapp } from '#lib/publico/textos.ts';
import type { Actions, PageServerLoad } from './$types';

type Aceptada = { negocio: { id: string; slug: string; nombre: string; rol: string } };

const aceptar = (e: Pick<RequestEvent, 'locals' | 'fetch'>, token: string) => apiDe(e).post<Aceptada>(`/v1/invitaciones/${token}/aceptar`);

export const load: PageServerLoad = async (event) => {
	const { params, fetch, locals, url, cookies, setHeaders } = event;
	if (!codigoValido(params.token)) error(404, 'Esa invitación no existe o ya no sirve.');
	setHeaders({ 'cache-control': 'no-store' });
	const invitacion = await o404(apiPublica(fetch).get<Invitacion>(`/v1/invitaciones/${params.token}`), 'Esa invitación no existe o ya no sirve.');
	const usable = !invitacion.vencida && !invitacion.aceptada;
	// Sin sesión: a ingresar, y de vuelta acá para aceptarla sola.
	if (usable && !locals.token) redirect(303, `/ingresar?volver=${encodeURIComponent(`/invitacion/${params.token}?aceptar=1`)}`);
	// Volviendo del ingreso: se acepta y al panel de ese negocio.
	let falla = '';
	if (usable && url.searchParams.has('aceptar')) {
		try {
			const r = await aceptar(event, params.token);
			elegirNegocio(cookies, r.negocio.id);
		} catch (e) {
			if (!(e instanceof ErrorApi)) throw e;
			falla = e.message;
		}
		if (!falla) redirect(303, '/panel');
	}
	// La página pública del negocio da su color y cómo escribirle (si no está publicada, sin eso).
	const p = await opcional(apiPublica(fetch).get<Pagina>(`/v1/p/${invitacion.slug}`), null);
	const negocio = {
		color: colorValido(p?.web?.colorPrimario, '#3ccc90'),
		logo: p?.negocio.logo ?? null,
		whatsapp: p ? linkWhatsapp(p.negocio.whatsapp || '') : '',
		publicada: !!p
	};
	return { invitacion, usable, falla, negocio };
};

export const actions: Actions = {
	default: async (event) => {
		if (!event.locals.token) redirect(303, `/ingresar?volver=${encodeURIComponent(`/invitacion/${event.params.token}?aceptar=1`)}`);
		const r = await intentar(() => aceptar(event, event.params.token));
		if (!('ok' in r)) return r;
		elegirNegocio(event.cookies, r.datos.negocio.id);
		redirect(303, '/panel');
	}
};
