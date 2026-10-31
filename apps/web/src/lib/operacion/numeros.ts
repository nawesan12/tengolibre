/** Cada valor como porcentaje del máximo (para el largo de una barra). */
export function escala(valores: number[]) {
	const max = Math.max(0, ...valores);
	return valores.map((v) => (max > 0 ? Math.round((v / max) * 100) : 0));
}

const unDecimal = new Intl.NumberFormat('es-AR', { maximumFractionDigits: 1 });

/** Un porcentaje que ya viene en 0..100 (la API los da así), o una raya si no hay dato. */
export const porcentaje = (v: number | null | undefined) => (v === null || v === undefined ? '—' : `${unDecimal.format(v)} %`);
