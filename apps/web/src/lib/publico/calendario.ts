import type { Dia } from './tipos.ts';

/** Una celda del calendario: un día de la vista o un hueco para alinear la semana. */
export type Celda = (Dia & { numero: number; mes: number }) | null;

/** Las semanas del calendario (de lunes a domingo) para los días que devuelve /dias. */
export function semanas(dias: Dia[]): Celda[][] {
	if (dias.length === 0) return [];
	const celdas: Celda[] = [];
	const primero = new Date(dias[0].fecha + 'T12:00:00Z');
	const hueco = (primero.getUTCDay() + 6) % 7; // lunes = 0
	for (let i = 0; i < hueco; i++) celdas.push(null);
	for (const d of dias) {
		const f = new Date(d.fecha + 'T12:00:00Z');
		celdas.push({ ...d, numero: f.getUTCDate(), mes: f.getUTCMonth() });
	}
	while (celdas.length % 7) celdas.push(null);
	const out: Celda[][] = [];
	for (let i = 0; i < celdas.length; i += 7) out.push(celdas.slice(i, i + 7));
	return out;
}

export const MESES = ['enero', 'febrero', 'marzo', 'abril', 'mayo', 'junio', 'julio', 'agosto', 'septiembre', 'octubre', 'noviembre', 'diciembre'];
const DIAS = ['Domingo', 'Lunes', 'Martes', 'Miércoles', 'Jueves', 'Viernes', 'Sábado'];

/** "Sábado 7 de noviembre" para un AAAA-MM-DD (es una fecha: no depende de la zona). */
export function fechaLarga(f: string) {
	const d = new Date(f + 'T12:00:00Z');
	return `${DIAS[d.getUTCDay()]} ${d.getUTCDate()} de ${MESES[d.getUTCMonth()]}`;
}

/** Un AAAA-MM-DD válido (lo que viene en la URL). */
export const esFecha = (f: string | null): f is string => !!f && /^\d{4}-\d{2}-\d{2}$/.test(f) && !isNaN(Date.parse(f));

/** Sumar días a un AAAA-MM-DD. */
export function sumarDias(f: string, n: number) {
	const d = new Date(f + 'T12:00:00Z');
	d.setUTCDate(d.getUTCDate() + n);
	return d.toISOString().slice(0, 10);
}

export const DIAS3 = ['dom', 'lun', 'mar', 'mié', 'jue', 'vie', 'sáb'];
export const MESES3 = ['ene', 'feb', 'mar', 'abr', 'may', 'jun', 'jul', 'ago', 'sep', 'oct', 'nov', 'dic'];

const partes = (f: string) => {
	const d = new Date(f + 'T12:00:00Z');
	return { dow: d.getUTCDay(), dia: d.getUTCDate(), mes: d.getUTCMonth() };
};

/** "LUN 02" para un AAAA-MM-DD (rótulos del talón y de la barra). */
export function diaCorto(f: string) {
	const p = partes(f);
	return `${DIAS3[p.dow].toUpperCase()} ${String(p.dia).padStart(2, '0')}`;
}

/** "lun 2" para un AAAA-MM-DD (en texto corrido). */
export function diaTexto(f: string) {
	const p = partes(f);
	return `${DIAS3[p.dow]} ${p.dia}`;
}

/** "NOV" o "NOV · DIC": los meses que abarcan los días. */
export function mesesCortos(fechas: string[]) {
	const ms = [...new Set(fechas.map((f) => partes(f).mes))];
	return ms.map((m) => MESES3[m].toUpperCase()).join(' · ');
}

/** "MES" en mayúsculas de un AAAA-MM-DD ("NOV"). */
export const mesCorto = (f: string) => MESES3[partes(f).mes].toUpperCase();

/** Lo que muestra un día en la tira: "lun" y "02". */
export const tiraDia = (f: string) => {
	const p = partes(f);
	return { dow: DIAS3[p.dow], numero: String(p.dia).padStart(2, '0') };
};
