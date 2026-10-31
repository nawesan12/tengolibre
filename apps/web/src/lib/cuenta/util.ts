import { dia } from '#lib/formato.ts';

/** Cuántas páginas hay (al menos una). */
export const paginas = (total: number, limite: number) => Math.max(1, Math.ceil(total / Math.max(1, limite)));

/** Una fecha AAAA-MM-DD válida, o undefined. */
export const fechaValida = (v: string | null | undefined): string | undefined =>
	v && /^\d{4}-\d{2}-\d{2}$/.test(v) && !Number.isNaN(Date.parse(v)) ? v : undefined;

/** El rango por defecto del reporte: los últimos 30 días, hoy incluido, en la hora de Argentina. */
export function ultimos30(ahora = new Date()) {
	const hasta = dia(ahora);
	const d = new Date(`${hasta}T12:00:00Z`);
	d.setUTCDate(d.getUTCDate() - 29);
	return { desde: d.toISOString().slice(0, 10), hasta };
}

/** AAAA-MM-DD como DD/MM/AAAA. */
export const dma = (d: string) => d.slice(0, 10).split('-').reverse().join('/');

/** La prueba de un negocio, en palabras. */
export function prueba(hasta: string | null, ahora = new Date()) {
	if (!hasta) return 'Sin prueba';
	const fin = new Date(hasta);
	return fin > ahora ? `Prueba hasta el ${dma(dia(fin))}` : 'Prueba vencida';
}
