import { apiDe } from '#lib/server/api.ts';
import { siPuede } from '#lib/configuracion/acciones.server.ts';
import { FILTROS, filtrosDe } from '#lib/configuracion/auditoria.ts';
import type { PageServerLoad } from './$types';

export type Entrada = {
	id: string;
	clerkId: string;
	nombre: string;
	accion: string;
	entidad: string;
	entidadId: string;
	detalle: unknown;
	creadoEn: string;
};
type Miembro = { clerkId: string; nombre: string; email: string };

export const load: PageServerLoad = async (event) => {
	const api = apiDe(event);
	const q = filtrosDe(event.url);
	q.set('limite', '50');
	const [{ entradas }, equipo] = await Promise.all([
		api.get<{ entradas: Entrada[] }>(`/v1/panel/auditoria?${q}`),
		siPuede(api.get<{ miembros: Miembro[] }>('/v1/panel/equipo'))
	]);
	q.delete('limite');
	q.delete('antes');
	return {
		entradas,
		miembros: equipo?.miembros ?? [],
		filtros: Object.fromEntries(FILTROS.map((k) => [k, event.url.searchParams.get(k) ?? ''])),
		consulta: q.toString(),
		// Hay más si vino la página llena: el cursor es el instante del último.
		siguiente: entradas.length === 50 ? entradas[entradas.length - 1].creadoEn : null
	};
};
