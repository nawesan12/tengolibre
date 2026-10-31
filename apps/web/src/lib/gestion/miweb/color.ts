/** Contraste WCAG entre dos colores #rrggbb (1 a 21). */
function luminancia(hex: string) {
	const m = /^#?([0-9a-f]{6})$/i.exec(hex.trim());
	if (!m) return 0;
	const n = parseInt(m[1], 16);
	const canal = (c: number) => {
		const v = c / 255;
		return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
	};
	return 0.2126 * canal((n >> 16) & 255) + 0.7152 * canal((n >> 8) & 255) + 0.0722 * canal(n & 255);
}

export function contraste(a: string, b: string) {
	const [x, y] = [luminancia(a), luminancia(b)].sort((p, q) => q - p);
	return (x + 0.05) / (y + 0.05);
}

export const TINTA = '#0f2328';

/** El texto sobre el color del negocio: blanco o tinta, el que se lea mejor, con su contraste. */
export function textoSobre(fondo: string) {
	const blanco = contraste(fondo, '#ffffff');
	const tinta = contraste(fondo, TINTA);
	return blanco >= tinta ? { color: '#ffffff', nombre: 'blanco', ratio: blanco } : { color: TINTA, nombre: 'oscuro', ratio: tinta };
}

/** «7,9:1» */
export const ratioTexto = (r: number) => `${r.toLocaleString('es-AR', { maximumFractionDigits: 1 })}:1`;
