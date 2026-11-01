import { apiPublica } from '#lib/server/api.ts';
import { cobro, opcional, pagina, primerLugar } from '#lib/publico/api.server.ts';
import type { Plan, ResenaPublica, Resumen } from '#lib/publico/tipos.ts';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params, fetch }) => {
	const api = apiPublica(fetch);
	const p = await pagina(fetch, params.slug);
	// El primer lugar libre (barra del teléfono y cabecera de la reserva): el del primer servicio.
	const primero = p.servicios.find((s) => s.activo !== false);
	const [resenas, planes, c, lugar] = await Promise.all([
		opcional(api.get<Resumen & { resenas: ResenaPublica[] }>(`/v1/p/${params.slug}/resenas`), null),
		opcional(api.get<{ planes: Plan[] }>(`/v1/p/${params.slug}/membresias`), { planes: [] }),
		cobro(fetch, params.slug),
		primero ? primerLugar(fetch, params.slug, primero.id) : null
	]);
	return {
		...p,
		listaResenas: resenas?.resenas ?? [],
		planes: planes.planes,
		cobro: c,
		proximo: lugar ? { fecha: lugar.fecha, hora: lugar.hora } : null
	};
};
