import { Api, apiPublica, type NegocioDeLaPersona, type Persona } from '#lib/server/api.ts';
import { conSesion } from '#lib/cuenta/ingreso.server.ts';
import { opcional } from '#lib/publico/api.server.ts';
import type { Pagina } from '#lib/publico/tipos.ts';
import type { Acceso } from '#lib/cuenta/contratos.ts';
import { colorValido } from '#lib/cuenta/presentacion.ts';
import type { PageServerLoad } from './$types';

type TurnoDeLaPersona = {
	codigo: string;
	inicio: string;
	fin: string;
	estado: string;
	servicio: string;
	profesional: string;
	negocio: string;
	slug: string;
	zonaHoraria: string;
};

/** La cuenta de la persona: quién es y sus negocios. Entrar a uno usa la acción de /panel/negocio. */
export const load: PageServerLoad = (event) =>
	conSesion(event, async (api) => {
		const [persona, { negocios }, { turnos }] = await Promise.all([
			api.get<Persona>('/v1/auth/persona'),
			api.get<{ negocios: NegocioDeLaPersona[] }>('/v1/yo'),
			// Los turnos que reservó con su mail, en cualquier negocio. Si falla, la cuenta se ve igual.
			opcional(api.get<{ turnos: TurnoDeLaPersona[] }>('/v1/yo/turnos'), { turnos: [] })
		]);
		// Cada tarjeta, con su estado (prueba, plan, sólo lectura) y el color de su página. Si algo
		// falla, la tarjeta sigue sin eso.
		const conEstado = await Promise.all(
			negocios.map(async (n) => {
				const [acceso, p] = await Promise.all([
					opcional(new Api(event.locals.token, n.id, event.fetch).get<Acceso>('/v1/panel/acceso'), null),
					opcional(apiPublica(event.fetch).get<Pagina>(`/v1/p/${n.slug}`), null)
				]);
				return { ...n, acceso, color: colorValido(p?.web?.colorPrimario), logo: p?.negocio.logo ?? null };
			})
		);
		return { persona, negocios: conEstado, turnos, elegido: event.locals.negocio };
	});
