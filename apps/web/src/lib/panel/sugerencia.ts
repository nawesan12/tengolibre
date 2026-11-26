/** Una sugerencia de la casa (GET /v1/panel/sugerencia): otra app de Grupo Vimo, con sus textos del catálogo de la API. */
export type Sugerencia = {
	id: string;
	marca: string;
	sello: string;
	titulo: string;
	cta: string;
	url: string;
	imagen: string;
	banner: string;
	color: string;
};

/** Cerrar o contar el clic (por /panel/sugerencia). Si falla no se avisa nada: la tarjeta ya se fue. */
export function avisar(cuerpo: { id: string; negocio: string; que: 'clic' } | { id: string; negocio: string; que: 'cerrar'; hasta: '30d' | 'siempre' }) {
	return fetch('/panel/sugerencia', {
		method: 'POST',
		headers: { 'content-type': 'application/json' },
		body: JSON.stringify(cuerpo),
		keepalive: true
	}).catch(() => null);
}
