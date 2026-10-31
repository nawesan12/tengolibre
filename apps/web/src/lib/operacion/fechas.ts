import { ZONA, fecha } from '#lib/formato.ts';

/**
 * Cuentas con días (AAAA-MM-DD) y horas del negocio. Los días se suman en UTC, así un cambio de
 * horario no corre la fecha; las horas se pasan a instantes con la zona del negocio.
 */
export const esDia = (s: string | null | undefined): s is string =>
	!!s && /^\d{4}-\d{2}-\d{2}$/.test(s) && !Number.isNaN(Date.parse(`${s}T00:00:00Z`));

export const esHora = (s: string | null | undefined): s is string => !!s && /^([01]\d|2[0-3]):[0-5]\d$/.test(s);

export function sumarDias(d: string, n: number) {
	const [y, m, dd] = d.split('-').map(Number);
	return new Date(Date.UTC(y, m - 1, dd + n)).toISOString().slice(0, 10);
}

/** 0 = domingo, como DIAS de formato y el horario de la API. */
export const diaDeSemana = (d: string) => new Date(`${d}T12:00:00Z`).getUTCDay();

/** El lunes de la semana de ese día. */
export const lunesDe = (d: string) => sumarDias(d, -((diaDeSemana(d) + 6) % 7));

/** Los días entre dos fechas, inclusive. */
export function diasEntre(desde: string, hasta: string) {
	const out: string[] = [];
	for (let d = desde; d <= hasta && out.length < 400; d = sumarDias(d, 1)) out.push(d);
	return out;
}

/** "lun 2 nov" de un AAAA-MM-DD (al mediodía, así ninguna zona lo corre de día). */
export const nombreDia = (d: string) => fecha(`${d}T15:00:00Z`);

function partes(ms: number, zona: string) {
	const p = Object.fromEntries(
		new Intl.DateTimeFormat('en-US', {
			timeZone: zona,
			hourCycle: 'h23',
			year: 'numeric',
			month: '2-digit',
			day: '2-digit',
			hour: '2-digit',
			minute: '2-digit'
		})
			.formatToParts(new Date(ms))
			.map((x) => [x.type, x.value])
	);
	return { y: +p.year, m: +p.month, d: +p.day, h: +p.hour % 24, mi: +p.minute };
}

/** Lo que la zona le suma a UTC en ese instante, en milisegundos. */
function desfase(ms: number, zona: string) {
	const p = partes(ms, zona);
	return Date.UTC(p.y, p.m - 1, p.d, p.h, p.mi) - Math.floor(ms / 60000) * 60000;
}

/** El instante (ISO) de un día y una hora en la zona del negocio. */
export function instante(d: string, h: string, zona = ZONA) {
	const [y, m, dd] = d.split('-').map(Number);
	const [hh, mi] = h.split(':').map(Number);
	const supuesto = Date.UTC(y, m - 1, dd, hh, mi);
	let t = supuesto - desfase(supuesto, zona);
	const otro = desfase(t, zona);
	if (otro !== desfase(supuesto, zona)) t = supuesto - otro;
	return new Date(t).toISOString();
}

/** Minutos desde la medianoche del negocio. */
export function minutosDelDia(iso: string, zona = ZONA) {
	const p = partes(Date.parse(iso), zona);
	return p.h * 60 + p.mi;
}

/** HH:MM de una cantidad de minutos. */
export const horaDe = (min: number) => `${String(Math.floor(min / 60)).padStart(2, '0')}:${String(min % 60).padStart(2, '0')}`;
