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

/** Cerrar, deshacer el cierre, contar la vista (la tarjeta quedó dibujada con su foto) o el clic (por /panel/sugerencia).
 *  Devuelve la respuesta, o null sin red: quien cierra mira `ok` (ver cerrarConReintento). */
export function avisar(cuerpo: { id: string; negocio: string; que: 'clic' | 'vista' | 'deshacer' } | { id: string; negocio: string; que: 'cerrar'; hasta: '30d' | 'siempre' }) {
	return fetch('/panel/sugerencia', {
		method: 'POST',
		headers: { 'content-type': 'application/json' },
		body: JSON.stringify(cuerpo),
		keepalive: true
	}).catch(() => null);
}

/** El cierre con un reintento al segundo: true si quedó guardado. Si no, la tarjeta vuelve y se avisa. */
export async function cerrarConReintento(cuerpo: { id: string; negocio: string; hasta: '30d' | 'siempre' }, espera = 1000) {
	for (let i = 0; i < 2; i++) {
		if (i) await new Promise((r) => setTimeout(r, espera));
		const r = await avisar({ ...cuerpo, que: 'cerrar' });
		if (r?.ok) return true;
		// Un 4xx (sin sesión, sin permiso) no se arregla reintentando.
		if (r && r.status < 500) return false;
	}
	return false;
}
