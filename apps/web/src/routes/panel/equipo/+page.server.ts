import { redirect } from '@sveltejs/kit';
import { apiDe } from '#lib/server/api.ts';
import { accion, siPuede } from '#lib/configuracion/acciones.server.ts';
import type { Actions, PageServerLoad } from './$types';

export type Miembro = { id: string; clerkId: string; email: string; nombre: string; rol: string; activo: boolean; creadoEn: string };
export type Invitacion = { id: string; email: string; rol: string; invito: string; venceEn: string; aceptadaEn: string | null; creadoEn: string };
export type Franja = { dia: number; desde: string; hasta: string };
export type Profesional = {
	id: string;
	miembroId: string | null;
	nombre: string;
	especialidad: string;
	bio: string;
	foto: string | null;
	email: string;
	telefono: string;
	orden: number;
	activo: boolean;
	servicios: string[];
	horario: Franja[];
};
type Servicio = { id: string; nombre: string; activo: boolean };

export const load: PageServerLoad = async (event) => {
	const api = apiDe(event);
	// Equipo e invitaciones son de quien gestiona; los profesionales los ve todo el panel.
	const [equipo, invitaciones, { profesionales }, { servicios }] = await Promise.all([
		siPuede(api.get<{ miembros: Miembro[] }>('/v1/panel/equipo')),
		siPuede(api.get<{ invitaciones: Invitacion[] }>('/v1/panel/invitaciones')),
		api.get<{ profesionales: Profesional[] }>('/v1/panel/profesionales'),
		api.get<{ servicios: Servicio[] }>('/v1/panel/servicios')
	]);
	return {
		miembros: equipo?.miembros ?? null,
		invitaciones: invitaciones?.invitaciones ?? null,
		profesionales,
		servicios
	};
};

export const actions: Actions = {
	rol: async (event) => {
		const f = await event.request.formData();
		return accion('miembros', 'Cambiamos el rol.', () =>
			apiDe(event).patch(`/v1/panel/equipo/${f.get('id')}`, { rol: String(f.get('rol')) })
		);
	},
	activo: async (event) => {
		const f = await event.request.formData();
		const activo = f.get('activo') === 'true';
		return accion('miembros', activo ? 'Volvió a entrar al panel.' : 'Ya no entra al panel.', () =>
			apiDe(event).patch(`/v1/panel/equipo/${f.get('id')}`, { activo })
		);
	},
	invitar: async (event) => {
		const f = await event.request.formData();
		const email = String(f.get('email') ?? '').trim();
		return accion('invitaciones', `Le mandamos la invitación a ${email}.`, () =>
			apiDe(event).post('/v1/panel/invitaciones', { email, rol: String(f.get('rol') ?? '') })
		);
	},
	revocar: async (event) => {
		const f = await event.request.formData();
		return accion('invitaciones', 'Borramos la invitación: el link ya no sirve.', () =>
			apiDe(event).borrar(`/v1/panel/invitaciones/${f.get('id')}`)
		);
	},
	profesional: async (event) => {
		const f = await event.request.formData();
		const r = await accion('profesionales', 'Sumamos al profesional.', () =>
			apiDe(event).post<Profesional>('/v1/panel/profesionales', {
				nombre: String(f.get('nombre') ?? ''),
				especialidad: String(f.get('especialidad') ?? ''),
				email: String(f.get('email') ?? ''),
				telefono: String(f.get('telefono') ?? ''),
				servicios: f.getAll('servicios').map(String),
				activo: true
			})
		);
		// Recién creado, lo próximo es cargarle el horario.
		if ('ok' in r && r.ok) redirect(303, `/panel/profesionales/${r.datos.id}?nuevo=1`);
		return r;
	}
};
