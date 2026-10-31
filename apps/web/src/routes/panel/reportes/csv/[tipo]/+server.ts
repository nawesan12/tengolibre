import { error } from '@sveltejs/kit';
import { apiDe } from '#lib/server/api.ts';
import { esDia } from '#lib/operacion/fechas.ts';
import type { RequestHandler } from './$types';

const TIPOS = new Set(['clientes', 'turnos']);

/**
 * Las descargas CSV: el navegador no tiene la sesión (cookie httpOnly), así que este servidor pide el
 * archivo a la API y lo pasa tal cual, con su nombre.
 */
export const GET: RequestHandler = async (event) => {
	const { tipo } = event.params;
	if (!TIPOS.has(tipo)) error(404, 'No existe ese archivo.');
	const q = new URLSearchParams();
	for (const k of ['desde', 'hasta']) {
		const v = event.url.searchParams.get(k);
		if (esDia(v)) q.set(k, v);
	}
	const res = await apiDe(event).crudo(`/v1/panel/reportes/${tipo}.csv?${q}`, { headers: { accept: 'text/csv' } });
	if (res.status === 403) error(403, 'Las descargas son para quien gestiona el negocio.');
	if (!res.ok || !res.body) {
		const p = (await res.json().catch(() => null)) as { detail?: string } | null;
		error(res.status >= 400 && res.status < 600 ? res.status : 502, p?.detail ?? 'No pudimos armar el archivo.');
	}
	return new Response(res.body, {
		headers: {
			'content-type': res.headers.get('content-type') ?? 'text/csv; charset=utf-8',
			'content-disposition': res.headers.get('content-disposition') ?? `attachment; filename="${tipo}.csv"`,
			'cache-control': 'no-store'
		}
	});
};
