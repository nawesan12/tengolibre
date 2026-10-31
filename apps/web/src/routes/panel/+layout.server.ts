import { redirect } from '@sveltejs/kit';
import { apiDe, ErrorApi, type NegocioDeLaPersona, type Persona } from '#lib/server/api.ts';
import { borrarSesion, elegirNegocio } from '#lib/server/sesion.ts';
import { dia } from '#lib/formato.ts';
import type { Anotado, Notificacion, Profesional, Turno } from '#lib/operacion/tipos.ts';
import type { LayoutServerLoad } from './$types';

export type Acceso = { bloqueado: boolean; pruebaHasta: string | null; diasDePrueba: number; conSuscripcion: boolean };

/**
 * El panel exige sesión y un negocio. El negocio elegido vive en una cookie; si no es de la persona
 * (o no hay), se toma el primero. Sin negocios, a darse de alta.
 *
 * Además trae lo que el armazón muestra en todas las pantallas: la campanita, el acceso (prueba o
 * sólo lectura), los turnos y profesionales de hoy («El día» del costado) y cuántos esperan en la
 * lista. Si algo de eso falla, el panel sigue sin ese dato.
 */
export const load: LayoutServerLoad = async (event) => {
	const { locals, url, cookies } = event;
	if (!locals.token) redirect(303, `/ingresar?volver=${encodeURIComponent(url.pathname + url.search)}`);
	const api = apiDe(event);
	let persona: Persona;
	let negocios: NegocioDeLaPersona[];
	try {
		[persona, { negocios }] = await Promise.all([
			api.get<Persona>('/v1/auth/persona'),
			api.get<{ negocios: NegocioDeLaPersona[] }>('/v1/yo')
		]);
	} catch (e) {
		if (e instanceof ErrorApi && e.estado === 401) {
			borrarSesion(cookies);
			redirect(303, `/ingresar?volver=${encodeURIComponent(url.pathname)}`);
		}
		throw e;
	}
	if (negocios.length === 0) redirect(303, '/empezar');
	let negocio = negocios.find((n) => n.id === locals.negocio);
	if (!negocio) {
		negocio = negocios[0];
		elegirNegocio(cookies, negocio.id);
		locals.negocio = negocio.id;
	}
	// Se recargan con el tiempo real.
	event.depends('panel:notificaciones', 'panel:turnos', 'panel:espera');
	const hoy = dia();
	const sinDato = <T>(p: Promise<T>) => p.catch(() => null);
	const a = apiDe(event);
	const [campanita, acceso, turnosHoy, profesionales, espera] = await Promise.all([
		sinDato(a.get<{ notificaciones: Notificacion[]; sinLeer: number }>('/v1/panel/campanita')),
		sinDato(a.get<Acceso>('/v1/panel/acceso')),
		sinDato(a.get<{ turnos: Turno[] }>(`/v1/panel/turnos?${new URLSearchParams({ desde: hoy, hasta: hoy })}`)),
		sinDato(a.get<{ profesionales: Profesional[] }>('/v1/panel/profesionales')),
		negocio.rol === 'profesional' ? null : sinDato(a.get<{ espera: Anotado[] }>('/v1/panel/espera'))
	]);
	return {
		persona,
		negocios,
		negocio,
		sinLeer: campanita?.sinLeer ?? 0,
		avisos: (campanita?.notificaciones ?? []).slice(0, 5),
		acceso,
		hoy: { fecha: hoy, turnos: turnosHoy?.turnos ?? [], profesionales: profesionales?.profesionales ?? [] },
		esperando: (espera?.espera ?? []).filter((x) => !x.avisadoEn && x.hasta >= hoy).length
	};
};
