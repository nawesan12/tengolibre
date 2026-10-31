import { invalidate } from '$app/navigation';

/**
 * Tiempo real del panel: escucha /panel/eventos (SSE, que el servidor de SvelteKit trae de la API) y
 * cuando algo cambia invalida las cargas que dependen de ese tipo (`depends('panel:turnos')`, etc.).
 * Varios avisos seguidos se juntan en una sola recarga. EventSource se reconecta solo.
 */
export type Aviso = { negocio: string; tipo: string; id: string; op: string };

const TIPOS = ['turnos', 'pagos', 'notificaciones', 'espera', 'resenas', 'clientes'] as const;

export function escucharPanel(alAviso?: (a: Aviso) => void) {
	const fuente = new EventSource('/panel/eventos');
	const pendientes = new Set<string>();
	let reloj: ReturnType<typeof setTimeout> | undefined;
	const recargar = () => {
		for (const t of pendientes) invalidate(`panel:${t}`);
		pendientes.clear();
	};
	for (const tipo of TIPOS) {
		fuente.addEventListener(tipo, (ev) => {
			try {
				alAviso?.(JSON.parse((ev as MessageEvent).data) as Aviso);
			} catch {
				/* aviso ilegible: igual se recarga */
			}
			pendientes.add(tipo);
			clearTimeout(reloj);
			reloj = setTimeout(recargar, 300);
		});
	}
	return () => {
		clearTimeout(reloj);
		fuente.close();
	};
}
