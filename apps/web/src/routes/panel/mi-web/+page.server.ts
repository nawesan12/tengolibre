import { apiDe } from '#lib/server/api.ts';
import { accion, subirImagen } from '#lib/configuracion/acciones.server.ts';
import type { Actions, PageServerLoad } from './$types';

export type Negocio = {
	id: string;
	slug: string;
	nombre: string;
	descripcion: string;
	direccion: string;
	ciudad: string;
	telefono: string;
	whatsapp: string;
	instagram: string;
	logo: string | null;
	zonaHoraria: string;
};
export type Web = {
	rubro: string;
	provincia: string;
	acerca: string;
	portada: string | null;
	colorPrimario: string;
	colorAcento: string;
	email: string;
	sitioWeb: string;
	facebook: string;
	tiktok: string;
	latitud: number | null;
	longitud: number | null;
};
export type Foto = { id: string; url: string; epigrafe: string; orden: number };

export const load: PageServerLoad = async (event) => {
	const api = apiDe(event);
	const [negocio, web, { fotos }] = await Promise.all([
		api.get<Negocio>('/v1/panel/negocio'),
		api.get<Web>('/v1/panel/mi-web'),
		api.get<{ fotos: Foto[] }>('/v1/panel/mi-web/galeria')
	]);
	return { datos: negocio, web, fotos };
};

const texto = (f: FormData, k: string) => String(f.get(k) ?? '').trim();

export const actions: Actions = {
	negocio: async (event) => {
		const f = await event.request.formData();
		return accion('negocio', 'Guardamos los datos del negocio.', () =>
			apiDe(event).patch('/v1/panel/negocio', {
				nombre: texto(f, 'nombre'),
				descripcion: texto(f, 'descripcion'),
				direccion: texto(f, 'direccion'),
				ciudad: texto(f, 'ciudad'),
				telefono: texto(f, 'telefono'),
				whatsapp: texto(f, 'whatsapp'),
				instagram: texto(f, 'instagram').replace(/^@/, '')
			})
		);
	},
	web: async (event) => {
		const f = await event.request.formData();
		const lat = texto(f, 'latitud');
		const lng = texto(f, 'longitud');
		const ubicacion = lat && lng ? { latitud: Number(lat), longitud: Number(lng) } : { sinUbicacion: !lat && !lng };
		return accion('web', 'Guardamos tu web.', () =>
			apiDe(event).patch('/v1/panel/mi-web', {
				rubro: texto(f, 'rubro'),
				provincia: texto(f, 'provincia'),
				acerca: texto(f, 'acerca'),
				colorPrimario: texto(f, 'colorPrimario'),
				colorAcento: texto(f, 'colorAcento'),
				email: texto(f, 'email'),
				sitioWeb: texto(f, 'sitioWeb'),
				facebook: texto(f, 'facebook'),
				tiktok: texto(f, 'tiktok').replace(/^@/, ''),
				...ubicacion
			})
		);
	},
	logo: async (event) => {
		const f = await event.request.formData();
		return accion('logo', 'Cambiamos el logo.', async () => {
			const url = await subirImagen(event, f.get('archivo'));
			return apiDe(event).patch('/v1/panel/negocio', { logo: url });
		});
	},
	portada: async (event) => {
		const f = await event.request.formData();
		return accion('portada', 'Cambiamos la portada.', async () => {
			const url = await subirImagen(event, f.get('archivo'));
			return apiDe(event).patch('/v1/panel/mi-web', { portada: url });
		});
	},
	sacarPortada: async (event) =>
		accion('portada', 'Sacamos la portada.', () => apiDe(event).patch('/v1/panel/mi-web', { portada: '' })),
	foto: async (event) => {
		const f = await event.request.formData();
		return accion('galeria', 'Sumamos la foto a la galería.', async () => {
			const url = await subirImagen(event, f.get('archivo'));
			return apiDe(event).post('/v1/panel/mi-web/galeria', { url, epigrafe: texto(f, 'epigrafe') });
		});
	},
	epigrafe: async (event) => {
		const f = await event.request.formData();
		return accion('galeria', 'Guardamos el epígrafe.', () =>
			apiDe(event).patch(`/v1/panel/mi-web/galeria/${f.get('id')}`, { epigrafe: texto(f, 'epigrafe') })
		);
	},
	mover: async (event) => {
		const f = await event.request.formData();
		const api = apiDe(event);
		return accion('galeria', 'Cambiamos el orden.', async () => {
			// El orden nuevo sale de la galería de ahora (la API exige todas las fotos, una vez cada una).
			const { fotos } = await api.get<{ fotos: Foto[] }>('/v1/panel/mi-web/galeria');
			const ids = fotos.map((x) => x.id);
			const i = ids.indexOf(String(f.get('id')));
			const j = i + (f.get('hacia') === 'antes' ? -1 : 1);
			if (i >= 0 && j >= 0 && j < ids.length) [ids[i], ids[j]] = [ids[j], ids[i]];
			return api.put('/v1/panel/mi-web/galeria/orden', { ids });
		});
	},
	borrarFoto: async (event) => {
		const f = await event.request.formData();
		return accion('galeria', 'Sacamos la foto.', () => apiDe(event).borrar(`/v1/panel/mi-web/galeria/${f.get('id')}`));
	}
};
