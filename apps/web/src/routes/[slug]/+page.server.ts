import { apiPublica } from '#lib/server/api.ts';
import { cobro, opcional, pagina } from '#lib/publico/api.server.ts';
import type { Plan, ResenaPublica, Resumen } from '#lib/publico/tipos.ts';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params, fetch }) => {
	const api = apiPublica(fetch);
	const [p, resenas, planes, c] = await Promise.all([
		pagina(fetch, params.slug),
		opcional(api.get<Resumen & { resenas: ResenaPublica[] }>(`/v1/p/${params.slug}/resenas`), null),
		opcional(api.get<{ planes: Plan[] }>(`/v1/p/${params.slug}/membresias`), { planes: [] }),
		cobro(fetch, params.slug)
	]);
	return { ...p, listaResenas: resenas?.resenas ?? [], planes: planes.planes, cobro: c };
};
