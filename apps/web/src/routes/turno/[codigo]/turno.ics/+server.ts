import { error } from '@sveltejs/kit';
import { apiPublica } from '#lib/server/api.ts';
import { codigoValido, o404 } from '#lib/publico/api.server.ts';
import type { TurnoCliente } from '#lib/publico/tipos.ts';
import type { RequestHandler } from './$types';

/**
 * «Agendar» en Mi turno: el turno como evento de calendario (.ics). Con el mismo criterio que el del
 * mail (avisos.ICSEvento): el mismo UID, SEQUENCE con los minutos del momento en que se arma (así gana
 * el más nuevo de los dos), la alarma de 1 h, CANCEL si se canceló y las líneas plegadas a 75 bytes.
 */
const fechaIcs = (iso: string) => new Date(iso).toISOString().replace(/[-:]/g, '').replace(/\.\d{3}/, '');
const escapar = (s: string) => s.replace(/\\/g, '\\\\').replace(/\n/g, '\\n').replace(/([,;])/g, '\\$1');
const bytes = new TextEncoder();
/** Pliega una línea a 75 bytes (RFC 5545 §3.1) sin cortar un carácter UTF-8 por la mitad. */
function plegar(linea: string): string {
	const partes: string[] = [];
	let actual = '';
	let largo = 0;
	for (const c of linea) {
		const n = bytes.encode(c).length;
		const tope = partes.length ? 74 : 75;
		if (largo + n > tope) {
			partes.push(actual);
			actual = '';
			largo = 0;
		}
		actual += c;
		largo += n;
	}
	partes.push(actual);
	return partes.join('\r\n ');
}

export const GET: RequestHandler = async ({ params, fetch, url }) => {
	if (!codigoValido(params.codigo)) error(404, 'No encontramos ese turno.');
	const t = await o404(apiPublica(fetch).get<TurnoCliente>(`/v1/t/${encodeURIComponent(params.codigo)}`), 'No encontramos ese turno.');
	const n = t.negocio;
	const lugar = [n.direccion, n.ciudad].filter(Boolean).join(', ');
	const cancelado = t.estado === 'cancelado';
	const ahora = new Date();
	const ics = [
		'BEGIN:VCALENDAR',
		'VERSION:2.0',
		'PRODID:-//TengoLibre//Turnos//ES',
		'CALSCALE:GREGORIAN',
		cancelado ? 'METHOD:CANCEL' : 'METHOD:PUBLISH',
		'BEGIN:VEVENT',
		`UID:${t.id}@tengolibre.com.ar`,
		`SEQUENCE:${Math.floor(ahora.getTime() / 60_000)}`,
		`DTSTAMP:${fechaIcs(ahora.toISOString())}`,
		`DTSTART:${fechaIcs(t.inicio)}`,
		`DTEND:${fechaIcs(t.fin)}`,
		`SUMMARY:${escapar(`${t.servicio} en ${n.nombre}`)}`,
		lugar ? `LOCATION:${escapar(lugar)}` : '',
		`DESCRIPTION:${escapar(`${t.profesional ? `Con ${t.profesional}. ` : ''}Para cambiarlo o cancelarlo: ${url.origin}/turno/${t.codigo}`)}`,
		cancelado ? 'STATUS:CANCELLED' : 'STATUS:CONFIRMED',
		...(cancelado ? [] : ['BEGIN:VALARM', 'TRIGGER:-PT1H', 'ACTION:DISPLAY', 'DESCRIPTION:Recordatorio de turno', 'END:VALARM']),
		'END:VEVENT',
		'END:VCALENDAR'
	]
		.filter(Boolean)
		.map(plegar)
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
