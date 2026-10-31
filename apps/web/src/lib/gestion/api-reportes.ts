/**
 * Contratos SUPUESTOS de la API para lo que muestran Reseñas, Referidos, Reportes y el Registro de
 * actividad del diseño v2 y la API todavía no da. La UI ya está conectada: mientras el endpoint no
 * exista (404/405/501) la sección no se muestra o cae a lo que se puede calcular. Al integrar,
 * ajustar acá la ruta o la forma y nada más.
 *
 * - GET /v1/panel/reportes/ausentes?desde=AAAA-MM-DD&hasta=AAAA-MM-DD
 *     → { clientes: [{ id, nombre, ausencias }] }  (los que más faltaron sin avisar, de mayor a menor, hasta 5)
 * - GET /v1/panel/reportes  (el de siempre) con un campo opcional más en cada día:
 *     porDia[].minutosReservados, porDia[].minutosDisponibles  (para la barra hecho / libre)
 * - GET /v1/panel/resenas  (el de siempre) con un campo opcional más:
 *     distribucion: { "1": n, "2": n, "3": n, "4": n, "5": n }  (reseñas visibles por puntaje)
 * - GET /v1/panel/referidos/usos
 *     → { usos: [{ id, refiere, referido, estado: 'premio_usado' | 'premio_disponible' | 'turno', turnoInicio: string | null }] }
 * - GET /v1/panel/auditoria?plata=1  (filtro nuevo: sólo cambios de plata — cobros, señas, precios, devoluciones)
 */

/** Lo mínimo del cliente de la API que hace falta (así este archivo no importa nada del servidor). */
type ApiGet = { get: <T>(ruta: string) => Promise<T> };

export type Ausente = { id: string; nombre: string; ausencias: number };
export type DistribucionResenas = Partial<Record<'1' | '2' | '3' | '4' | '5', number>>;
export type UsoReferido = {
	id: string;
	refiere: string;
	referido: string;
	estado: 'premio_usado' | 'premio_disponible' | 'turno';
	turnoInicio: string | null;
};

/** Un pedido que todavía puede no existir en la API: null si no está (o si no le toca al rol). */
export async function opcional<T>(p: Promise<T>): Promise<T | null> {
	try {
		return await p;
	} catch (e) {
		const estado = (e as { estado?: number })?.estado;
		if (estado === 404 || estado === 405 || estado === 501 || estado === 403) return null;
		throw e;
	}
}

export const ausentes = (api: ApiGet, q: URLSearchParams) =>
	opcional(api.get<{ clientes: Ausente[] }>(`/v1/panel/reportes/ausentes?${q}`)).then((r) => r?.clientes ?? null);

export const usosReferidos = (api: ApiGet) =>
	opcional(api.get<{ usos: UsoReferido[] }>('/v1/panel/referidos/usos')).then((r) => r?.usos ?? null);
