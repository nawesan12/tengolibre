/**
 * Cuentas del horario en el onboarding: de «días que abrís + abrís/cerrás + descanso» a las franjas
 * que pide la API (PUT /v1/panel/profesionales/{id}/horario, 0 es domingo), las horas libres por
 * semana, la tira de «un día con…» y los próximos feriados. Funciones puras, con pruebas.
 */
import type { Franja } from '#lib/configuracion/horario.ts';

/** Días como los muestra el paso: 1 lunes … 7 domingo. */
export const DIAS = [
	{ n: 1, letra: 'L', nombre: 'Lunes' },
	{ n: 2, letra: 'M', nombre: 'Martes' },
	{ n: 3, letra: 'M', nombre: 'Miércoles' },
	{ n: 4, letra: 'J', nombre: 'Jueves' },
	{ n: 5, letra: 'V', nombre: 'Viernes' },
	{ n: 6, letra: 'S', nombre: 'Sábado' },
	{ n: 7, letra: 'D', nombre: 'Domingo' }
];

export const minutos = (h: string) => {
	const [a, b] = (h || '0:0').split(':').map(Number);
	return (a || 0) * 60 + (b || 0);
};
export const hhmm = (m: number) => `${String(Math.floor(m / 60)).padStart(2, '0')}:${String(m % 60).padStart(2, '0')}`;

export type Descanso = { desde: string; hasta: string } | null;

/** El descanso sólo cuenta si cae adentro del horario y termina después de empezar. */
export function descansoValido(abre: string, cierra: string, d: Descanso): d is { desde: string; hasta: string } {
	if (!d) return false;
	const [a, b, x, y] = [minutos(abre), minutos(cierra), minutos(d.desde), minutos(d.hasta)];
	return y > x && x > a && y < b;
}

/** Las franjas de la semana para la API. */
export function franjas(dias: number[], abre: string, cierra: string, descanso: Descanso = null): Franja[] {
	if (minutos(cierra) <= minutos(abre)) return [];
	const tramos = descansoValido(abre, cierra, descanso)
		? [
				[abre, descanso.desde],
				[descanso.hasta, cierra]
			]
		: [[abre, cierra]];
	return [...dias]
		.sort((a, b) => a - b)
		.flatMap((d) => tramos.map(([desde, hasta]) => ({ dia: d % 7, desde, hasta })));
}

/** Horas por semana de todo el equipo (el número grande del paso). */
export function horasSemana(dias: number, abre: string, cierra: string, personas: number, descanso: Descanso = null) {
	const desc = descansoValido(abre, cierra, descanso) ? minutos(descanso.hasta) - minutos(descanso.desde) : 0;
	return Math.max(0, Math.round(((minutos(cierra) - minutos(abre) - desc) / 60) * dias * Math.max(1, personas)));
}

/** Lo que se lee del horario guardado de una persona, para volver al paso con lo que ya cargó. */
export function leerHorario(h: Franja[]): { dias: number[]; abre: string; cierra: string; descanso: Descanso } | null {
	if (!h.length) return null;
	const dias = [...new Set(h.map((f) => (f.dia === 0 ? 7 : f.dia)))].sort((a, b) => a - b);
	const primero = h.filter((f) => f.dia === h[0].dia).sort((a, b) => a.desde.localeCompare(b.desde));
	const abre = primero[0].desde.slice(0, 5);
	const cierra = primero[primero.length - 1].hasta.slice(0, 5);
	const descanso = primero.length > 1 ? { desde: primero[0].hasta.slice(0, 5), hasta: primero[1].desde.slice(0, 5) } : null;
	return { dias, abre, cierra, descanso };
}

export type Segmento = { f: number; tipo: 'turno' | 'entre' | 'descanso' | 'resto' };

/** La tira de «un día con…»: turnos del primer servicio, el tiempo entre turnos y el descanso. */
export function segmentosDia(abre: string, cierra: string, duracion: number, entre: number, descanso: Descanso) {
	const A = minutos(abre);
	const B = minutos(cierra);
	const dz = descansoValido(abre, cierra, descanso) ? [minutos(descanso.desde), minutos(descanso.hasta)] : null;
	const sd = Math.max(5, duracion);
	const seg: Segmento[] = [];
	let t = A;
	let turnos = 0;
	for (let i = 0; t < B && i < 400; i++) {
		if (dz && t >= dz[0] && t < dz[1]) {
			seg.push({ f: dz[1] - t, tipo: 'descanso' });
			t = dz[1];
			continue;
		}
		const fin = t + sd;
		if (fin > B || (dz && t < dz[0] && fin > dz[0])) {
			const lim = dz && t < dz[0] ? dz[0] : B;
			seg.push({ f: lim - t, tipo: 'resto' });
			t = lim;
			continue;
		}
		seg.push({ f: sd, tipo: 'turno' });
		turnos++;
		t = fin;
		if (entre && t < B) {
			const g = Math.min(entre, B - t);
			seg.push({ f: g, tipo: 'entre' });
			t += g;
		}
	}
	return { seg, turnos };
}

/**
 * Feriados nacionales inamovibles (y Carnaval y Viernes Santo, que dependen de la Pascua). Los
 * trasladables los fija un decreto cada año: no se adivinan.
 */
const FIJOS: [number, number, string][] = [
	[1, 1, 'Año Nuevo'],
	[3, 24, 'Día Nacional de la Memoria'],
	[4, 2, 'Día del Veterano y de los Caídos en Malvinas'],
	[5, 1, 'Día del Trabajador'],
	[5, 25, 'Revolución de Mayo'],
	[6, 20, 'Paso a la Inmortalidad del Gral. Belgrano'],
	[7, 9, 'Día de la Independencia'],
	[12, 8, 'Inmaculada Concepción de María'],
	[12, 25, 'Navidad']
];

/** Domingo de Pascua (algoritmo de Meeus/Jones/Butcher). */
function pascua(y: number) {
	const a = y % 19, b = Math.floor(y / 100), c = y % 100, d = Math.floor(b / 4), e = b % 4;
	const f = Math.floor((b + 8) / 25), g = Math.floor((b - f + 1) / 3), h = (19 * a + b - d - g + 15) % 30;
	const i = Math.floor(c / 4), k = c % 4, l = (32 + 2 * e + 2 * i - h - k) % 7, m = Math.floor((a + 11 * h + 22 * l) / 451);
	const mes = Math.floor((h + l - 7 * m + 114) / 31);
	const dia = ((h + l - 7 * m + 114) % 31) + 1;
	return new Date(Date.UTC(y, mes - 1, dia));
}

export type Feriado = { fecha: string; nombre: string; rotulo: string };

const DIA_CORTO = ['DOM', 'LUN', 'MAR', 'MIÉ', 'JUE', 'VIE', 'SÁB'];
const MES_CORTO = ['ENE', 'FEB', 'MAR', 'ABR', 'MAY', 'JUN', 'JUL', 'AGO', 'SEP', 'OCT', 'NOV', 'DIC'];
const iso = (d: Date) => d.toISOString().slice(0, 10);

/** Los próximos `n` feriados desde `hoy` (AAAA-MM-DD), con el rótulo «LUN 08 DIC». */
export function proximosFeriados(hoy: string, n = 4): Feriado[] {
	const y = Number(hoy.slice(0, 4));
	const todos: { d: Date; nombre: string }[] = [];
	for (const año of [y, y + 1]) {
		for (const [m, d, nombre] of FIJOS) todos.push({ d: new Date(Date.UTC(año, m - 1, d)), nombre });
		const p = pascua(año).getTime();
		const dia = 86_400_000;
		todos.push({ d: new Date(p - 48 * dia), nombre: 'Carnaval' }, { d: new Date(p - 47 * dia), nombre: 'Carnaval' });
		todos.push({ d: new Date(p - 2 * dia), nombre: 'Viernes Santo' });
	}
	return todos
		.filter((x) => iso(x.d) >= hoy)
		.sort((a, b) => a.d.getTime() - b.d.getTime())
		.slice(0, n)
		.map(({ d, nombre }) => ({
			fecha: iso(d),
			nombre,
			rotulo: `${DIA_CORTO[d.getUTCDay()]} ${String(d.getUTCDate()).padStart(2, '0')} ${MES_CORTO[d.getUTCMonth()]}`
		}));
}
