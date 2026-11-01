import { pesos } from '#lib/formato.ts';

type Config = { activo: boolean; tipo: string | null; valor: number | null; tope: number | null };

/** El premio en partes, para armar la frase con los números resaltados. */
export function premioDe(c: Config) {
	if (!c.tipo || !c.valor) return null;
	return { descuento: c.tipo === 'fijo' ? pesos(c.valor) : `${c.valor}%`, tope: c.tope };
}
