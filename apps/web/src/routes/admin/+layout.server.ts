import type { Persona } from '#lib/server/api.ts';
import { conSesion } from '#lib/cuenta/ingreso.server.ts';
import type { Estadisticas } from '#lib/cuenta/tipos.ts';
import type { LayoutServerLoad } from './$types';

/**
 * La administración no se anuncia: la API responde 404 a quien no es del equipo y acá eso es un 404
 * de la página. El chequeo va en el layout para que el error se muestre con el layout raíz, sin
 * la barra del admin.
 */
export const load: LayoutServerLoad = (event) =>
	conSesion(
		event,
		async (api) => {
			const [estadisticas, persona] = await Promise.all([
				api.get<Estadisticas>('/v1/admin/estadisticas'),
				api.get<Persona>('/v1/auth/persona')
			]);
			return { estadisticas, persona };
		},
		true
	);
