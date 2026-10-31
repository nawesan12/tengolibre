/** Los filtros de la auditoría que entiende la API, tal cual vienen en la URL de la página. */
export const FILTROS = ['desde', 'hasta', 'entidad', 'quien', 'antes'] as const;

export function filtrosDe(url: URL) {
	const q = new URLSearchParams();
	for (const k of FILTROS) {
		const v = url.searchParams.get(k)?.trim();
		if (v) q.set(k, v);
	}
	return q;
}
