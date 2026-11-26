import { error } from '@sveltejs/kit';
import { Api, ErrorApi } from '#lib/server/api.ts';
import type { RequestHandler } from './$types';

/**
 * La tarjeta de la sugerencia de la casa (Inicio): cerrarla («Ocultar por 30 días» o «Ya la uso») y
 * contar el clic. El navegador no habla con la API: pasa por acá con la sesión de la cookie. El negocio
 * es el de la página (no el de la cookie, por si se cambió en otra pestaña); la API igual lo verifica.
 */
export const POST: RequestHandler = async (event) => {
	const c = (await event.request.json().catch(() => null)) as { id?: unknown; que?: unknown; hasta?: unknown; negocio?: unknown } | null;
	const id = typeof c?.id === 'string' && /^[a-z0-9-]{2,40}$/.test(c.id) ? c.id : null;
	if (!id) error(400, 'Falta la sugerencia.');
	const negocio = typeof c?.negocio === 'string' && /^[0-9a-f-]{36}$/.test(c.negocio) ? c.negocio : event.locals.negocio;
	const api = new Api(event.locals.token, negocio, event.fetch);
	try {
		if (c?.que === 'clic') await api.post(`/v1/panel/sugerencia/${id}/clic`);
		else if (c?.que === 'cerrar' && (c.hasta === '30d' || c.hasta === 'siempre')) await api.post(`/v1/panel/sugerencia/${id}/cerrar`, { hasta: c.hasta });
		else error(400, 'Pedido inválido.');
	} catch (e) {
		if (e instanceof ErrorApi) error(e.estado, e.message);
		throw e;
	}
	return new Response(null, { status: 204 });
};
