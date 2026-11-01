/** Cuentas del horario semanal para las pantallas de equipo (funciones puras). */
import { SEMANA, franjasDe, type Franja } from '#lib/configuracion/horario.ts';
import { DIAS } from '#lib/formato.ts';

export const minutos = (h: string) => {
	const [a, b] = h.slice(0, 5).split(':').map(Number);
	return a * 60 + (b || 0);
};

/** «10» o «10:30». */
export const horaCorta = (h: string) => {
	const [a, b] = h.slice(0, 5).split(':');
	return b === '00' ? a : `${a}:${b}`;
};

/** «10–14». */
export const rango = (f: { desde: string; hasta: string }) => `${horaCorta(f.desde)}–${horaCorta(f.hasta)}`;

/** Horas por semana del horario (redondeadas a la media hora). */
export function horasSemana(horario: Franja[]) {
	const m = horario.reduce((t, f) => t + Math.max(0, minutos(f.hasta) - minutos(f.desde)), 0);
	return Math.round((m / 60) * 2) / 2;
}

export const horasTexto = (h: number) => (Number.isInteger(h) ? `${h}` : h.toLocaleString('es-AR'));

export function iniciales(nombre: string) {
	const p = nombre.trim().split(/\s+/).filter(Boolean);
	return ((p[0]?.[0] ?? '') + (p.length > 1 ? p[p.length - 1][0] : (p[0]?.[1] ?? ''))).toUpperCase();
}

export const DIA_CORTO = ['DOM', 'LUN', 'MAR', 'MIÉ', 'JUE', 'VIE', 'SÁB'];

/** El eje de horas de la grilla: de 3 en 3, cubriendo todas las franjas (por defecto 9 a 21). */
export function eje(horario: Franja[]) {
	let ini = 9 * 60;
	let fin = 21 * 60;
	for (const f of horario) {
		ini = Math.min(ini, minutos(f.desde));
		fin = Math.max(fin, minutos(f.hasta));
	}
	const desde = Math.floor(ini / 60);
	const pasos = Math.max(2, Math.ceil((Math.ceil(fin / 60) - desde) / 3));
	const hasta = desde + pasos * 3;
	return { desde: desde * 60, hasta: hasta * 60, marcas: Array.from({ length: pasos + 1 }, (_, i) => desde + i * 3) };
}

/**
 * Para «Mi horario»: los días con el mismo horario seguidos se juntan («Jueves y viernes»), con sus
 * franjas y las horas que suman.
 */
export function diasAgrupados(horario: Franja[]) {
	const grupos: { dias: number[]; franjas: Franja[]; horas: number }[] = [];
	for (const d of SEMANA) {
		const fr = franjasDe(horario, d);
		if (fr.length === 0) continue;
		const clave = fr.map(rango).join(' · ');
		const ult = grupos.at(-1);
		const anterior = ult ? SEMANA[SEMANA.indexOf(ult.dias.at(-1)!) + 1] : -1;
		if (ult && anterior === d && ult.franjas.map(rango).join(' · ') === clave) {
			ult.dias.push(d);
			ult.horas += horasSemana(fr);
		} else grupos.push({ dias: [d], franjas: fr, horas: horasSemana(fr) });
	}
	return grupos.map((g) => ({
		...g,
		nombre:
			g.dias.length === 1
				? DIAS[g.dias[0]]
				: g.dias.length === 2
					? `${DIAS[g.dias[0]]} y ${DIAS[g.dias[1]].toLowerCase()}`
					: `${DIAS[g.dias[0]]} a ${DIAS[g.dias.at(-1)!].toLowerCase()}`,
		texto: g.franjas.map(rango).join(' · ')
	}));
}

/** Los días de `desde` a `hasta` (AAAA-MM-DD), como mucho `max`. */
export function diasEntre(desde: string, hasta: string, max = 31) {
	const out: string[] = [];
	const d = new Date(`${desde}T12:00:00Z`);
	const fin = new Date(`${(hasta || desde)}T12:00:00Z`);
	while (d <= fin && out.length < max) {
		out.push(d.toISOString().slice(0, 10));
		d.setUTCDate(d.getUTCDate() + 1);
	}
	return out;
}

const MESES = ['ENE', 'FEB', 'MAR', 'ABR', 'MAY', 'JUN', 'JUL', 'AGO', 'SEP', 'OCT', 'NOV', 'DIC'];

/** «SÁB 21 DIC» de un AAAA-MM-DD. */
export function fechaTalon(f: string) {
	const d = new Date(`${f}T12:00:00Z`);
	return `${DIA_CORTO[d.getUTCDay()]} ${d.getUTCDate()} ${MESES[d.getUTCMonth()]}`;
}

/** «14 → 18 NOV» o «14 NOV → 2 DIC». */
export function rangoTalon(desde: string, hasta: string) {
	const a = new Date(`${desde}T12:00:00Z`);
	const b = new Date(`${hasta}T12:00:00Z`);
	if (desde === hasta) return fechaTalon(desde);
	return a.getUTCMonth() === b.getUTCMonth()
		? `${a.getUTCDate()} → ${b.getUTCDate()} ${MESES[b.getUTCMonth()]}`
		: `${a.getUTCDate()} ${MESES[a.getUTCMonth()]} → ${b.getUTCDate()} ${MESES[b.getUTCMonth()]}`;
}
