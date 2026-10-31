import { apiDe, ErrorApi } from '#lib/server/api.ts';
import { dia } from '#lib/formato.ts';
import type { ReporteMensual } from '#lib/gestion/reportes/mensual.ts';
import type { PageServerLoad } from './$types';

type Profesional = { id: string; foto: string | null };

/**
 * El reporte mensual imprimible (el mail del primer día de cada mes linkea acá). Sin `mes`, la API
 * devuelve el mes pasado. Las fotos del equipo salen de los profesionales.
 */
export const load: PageServerLoad = async (event) => {
	const mes = event.url.searchParams.get('mes') ?? '';
	const valido = /^\d{4}-\d{2}$/.test(mes);
	const api = apiDe(event);
	try {
		const [reporte, fotos] = await Promise.all([
			api.get<ReporteMensual>(`/v1/panel/reportes/mensual${valido ? `?mes=${mes}` : ''}`),
			api
				.get<{ profesionales: Profesional[] }>('/v1/panel/profesionales')
				.then((r) => Object.fromEntries(r.profesionales.filter((p) => p.foto).map((p) => [p.id, p.foto as string])))
				.catch(() => ({}) as Record<string, string>)
		]);
		return { reporte, fotos, error: '', mesActual: dia().slice(0, 7) };
	} catch (e) {
		if (e instanceof ErrorApi && e.estado < 500)
			return { reporte: null, fotos: {} as Record<string, string>, error: e.message, mesActual: dia().slice(0, 7) };
		throw e;
	}
};
