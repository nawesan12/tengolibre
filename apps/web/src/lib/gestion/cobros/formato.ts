/** Formatos chicos de las pantallas de cobros (pagos, membresías, cupones, suscripción). */
import { dia, hora, ZONA } from '#lib/formato.ts';

const diaSemana = new Intl.DateTimeFormat('es-AR', { weekday: 'short', timeZone: ZONA });
const diaMes = new Intl.DateTimeFormat('es-AR', { day: '2-digit', timeZone: ZONA });
const corto = new Intl.DateTimeFormat('es-AR', { day: 'numeric', month: 'short', timeZone: ZONA });
const largo = new Intl.DateTimeFormat('es-AR', { weekday: 'long', day: 'numeric', month: 'long', timeZone: ZONA });
const mes = new Intl.DateTimeFormat('es-AR', { month: 'long', timeZone: ZONA });

const sinPunto = (s: string) => s.replace('.', '');

/** «HOY 09:12» si es de hoy; si no, «DOM 01». */
export function cuandoCorto(i: string, ahora = new Date()) {
	if (dia(i) === dia(ahora)) return `HOY ${hora(i)}`;
	return `${sinPunto(diaSemana.format(new Date(i)))} ${diaMes.format(new Date(i))}`.toUpperCase();
}

/** «28 nov». */
export const fechaCorta = (i: string) => sinPunto(corto.format(new Date(i)));

/** «martes 3 de noviembre». */
export const fechaLarga = (i: string) => largo.format(new Date(i)).replace(',', '');

/** «octubre». */
export const nombreMes = (i: string | Date) => mes.format(new Date(i));

/** Días enteros desde ahora hasta `i` (redondeado para arriba; 0 si ya pasó). */
export const diasHasta = (i: string, ahora = new Date()) => Math.max(0, Math.ceil((new Date(i).getTime() - ahora.getTime()) / 86_400_000));

/** Lo que paga una clienta al reservar con un modo de cobro, sobre un precio de ejemplo. */
export function alReservar(modo: string, precio: number, porcentaje: number, fija: number) {
	if (modo === 'porcentaje') return Math.round((precio * porcentaje) / 100);
	if (modo === 'fija') return Math.min(fija, precio);
	if (modo === 'total') return precio;
	return 0;
}
