import { error } from '@sveltejs/kit';
import { apiPublica } from '#lib/server/api.ts';
import { codigoValido, o404 } from '#lib/publico/api.server.ts';
import type { TurnoCliente } from '#lib/publico/tipos.ts';
import type { RequestHandler } from './$types';

/** «Agendar» en Mi turno: el turno como evento de calendario (.ics), para el calendario del teléfono. */
const fechaIcs = (iso: string) => new Date(iso).toISOString().replace(/[-:]/g, '').replace(/\.\d{3}/, '');
const escapar = (s: string) => s.replace(/\\/g, '\\\\').replace(/\n/g, '\\n').replace(/([,;])/g, '\\$1');

export const GET: RequestHandler = async ({ params, fetch, url }) => {
	if (!codigoValido(params.codigo)) error(404, 'No encontramos ese turno.');
	const t = await o404(apiPublica(fetch).get<TurnoCliente>(`/v1/t/${params.codigo}`), 'No encontramos ese turno.');
	const n = t.negocio;
	const lugar = [n.direccion, n.ciudad].filter(Boolean).join(', ');
	const ics = [
		'BEGIN:VCALENDAR',
		'VERSION:2.0',
		'PRODID:-//Tengo Libre//Turnos//ES',
		'CALSCALE:GREGORIAN',
		'METHOD:PUBLISH',
		'BEGIN:VEVENT',
		`UID:${t.id}@tengolibre.com.ar`,
		`DTSTAMP:${fechaIcs(new Date().toISOString())}`,
		`DTSTART:${fechaIcs(t.inicio)}`,
		`DTEND:${fechaIcs(t.fin)}`,
		`SUMMARY:${escapar(`${t.servicio} en ${n.nombre}`)}`,
		lugar ? `LOCATION:${escapar(lugar)}` : '',
		`DESCRIPTION:${escapar(`${t.profesional ? `Con ${t.profesional}. ` : ''}Para cambiarlo o cancelarlo: ${url.origin}/turno/${t.codigo}`)}`,
		t.estado === 'cancelado' ? 'STATUS:CANCELLED' : 'STATUS:CONFIRMED',
		'END:VEVENT',
		'END:VCALENDAR'
	]
		.filter(Boolean)
		.join('\r\n');
	return new Response(ics + '\r\n', {
		headers: {
			'content-type': 'text/calendar; charset=utf-8',
			'content-disposition': 'attachment; filename="turno.ics"',
			'cache-control': 'no-store',
			'x-robots-tag': 'noindex'
		}
	});
};
