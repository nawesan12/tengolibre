import { error } from '@sveltejs/kit';
import { apiDe } from '#lib/server/api.ts';
import { filtrosDe } from '#lib/configuracion/auditoria.ts';
import type { RequestHandler } from './$types';

/** El CSV de la auditoría con los mismos filtros de la pantalla, pasado tal cual desde la API. */
export const GET: RequestHandler = async (event) => {
	if (!event.locals.token) error(401, 'Entrá de nuevo para descargar.');
	const q = filtrosDe(event.url);
	q.delete('antes');
	const res = await apiDe(event).crudo(`/v1/panel/auditoria/exportar?${q}`, { headers: { accept: 'text/csv' } });
	if (!res.ok) {
		const p = (await res.json().catch(() => null)) as { detail?: string } | null;
		error(res.status, p?.detail ?? 'No pudimos armar el archivo.');
	}
	return new Response(res.body, {
		headers: {
			'content-type': res.headers.get('content-type') ?? 'text/csv; charset=utf-8',
			'content-disposition': res.headers.get('content-disposition') ?? 'attachment; filename="auditoria.csv"',
			'cache-control': 'no-store'
		}
	});
};
