import { apiPublica } from '#lib/server/api.ts';
import { opcional } from '#lib/publico/api.server.ts';
import type { Ciudad, Directorio } from '#lib/publico/tipos.ts';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ url, fetch }) => {
	const api = apiPublica(fetch);
	const filtros = {
		q: (url.searchParams.get('q') ?? '').trim().slice(0, 80),
		ciudad: (url.searchParams.get('ciudad') ?? '').trim().slice(0, 80),
		rubro: (url.searchParams.get('rubro') ?? '').trim().slice(0, 40)
	};
	const pagina = Math.max(1, Math.min(1000, Number(url.searchParams.get('pagina')) || 1));
	const q = new URLSearchParams({ ...filtros, pagina: String(pagina), limite: '24' });
	for (const [k, v] of [...q]) if (!v) q.delete(k);
	const [directorio, { ciudades }, { rubros }] = await Promise.all([
		api.get<Directorio>(`/v1/explorar?${q}`),
		opcional(api.get<{ ciudades: Ciudad[] }>('/v1/explorar/ciudades'), { ciudades: [] }),
		opcional(api.get<{ rubros: string[] }>('/v1/explorar/rubros'), { rubros: [] })
	]);
	return { directorio, ciudades, rubros, filtros, pagina };
};
