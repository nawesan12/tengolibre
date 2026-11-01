import { apiPublica } from '#lib/server/api.ts';
import { cobro, opcional, pagina, primerLugar } from '#lib/publico/api.server.ts';
import { vistaPrevia } from '#lib/publico/estilo.ts';
import type { Plan, ResenaPublica, Resumen } from '#lib/publico/tipos.ts';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params, fetch, url, setHeaders }) => {
	const api = apiPublica(fetch);
	// La vista previa de Plantillas (un iframe del panel, mismo origen): el borrador en la query, sólo
	// para mostrar. No se guarda nada, no se indexa y sólo se deja enmarcar desde el propio sitio.
	const vista = vistaPrevia(url.searchParams);
	if (vista)
		setHeaders({
			'cache-control': 'no-store',
			'x-robots-tag': 'noindex',
			'x-frame-options': 'SAMEORIGIN',
			'content-security-policy': "frame-ancestors 'self'"
		});
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
		proximo: lugar ? { fecha: lugar.fecha, hora: lugar.hora } : null,
		vista
	};
};
