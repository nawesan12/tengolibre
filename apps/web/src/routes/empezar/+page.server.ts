/**
 * /empezar: sin sesión, el alta («Landing y alta v3»): nombre, rubro y dirección, y se sigue con Google
 * o Apple. Con sesión, el onboarding de 8 pasos («Onboarding v2»). Rubro y nombre van en la URL hasta
 * que el paso del nombre crea el negocio; de ahí en adelante cada paso guarda en la API (los mismos
 * endpoints que el panel) y redirige al siguiente (`?paso=`), así se puede retomar donde quedó.
 */
import { fail, redirect } from '@sveltejs/kit';
import { dev } from '$app/env';
import { TENGOLIBRE_INGRESO_PRUEBA } from '$app/env/private';
import { PUBLIC_CLERK_PUBLICABLE } from '$app/env/public';
import { Api, apiDe, type Persona } from '#lib/server/api.ts';
import { intentar } from '#lib/server/acciones.ts';
import { COOKIE_VOLVER, elegirNegocio, volverValido } from '#lib/server/sesion.ts';
import { fapiDe } from '#lib/clerk.ts';
import { esPaso, conNegocio, siguiente, urlPaso, type Paso } from '#lib/alta/pasos.ts';
import { rubroDe } from '#lib/alta/rubros.ts';
import { franjas, minutos, proximosFeriados, hhmm } from '#lib/alta/agenda.ts';
import { ESTILO_BASE, type Ajustes, type BloqueoAlta, type Cargado, type CobroAlta, type NegocioAlta, type ProfesionalAlta, type ServicioAlta, type WebAlta } from '#lib/alta/tipos.ts';
import type { Actions, PageServerLoad, RequestEvent } from './$types';

const texto = (f: FormData, k: string) => String(f.get(k) ?? '').trim();
function json<T>(f: FormData, k: string, def: T): T {
	try {
		return JSON.parse(String(f.get(k) ?? '')) as T;
	} catch {
		return def;
	}
}

/** Lo que trae la URL antes de crear el negocio (también lo que vuelve de Google o Apple). */
const borradorDe = (url: URL) => ({
	nombre: (url.searchParams.get('nombre') ?? '').slice(0, 80),
	slug: (url.searchParams.get('slug') ?? '').slice(0, 48),
	rubro: rubroDe(url.searchParams.get('rubro'))?.clave ?? ''
});

const hoyAR = () => new Intl.DateTimeFormat('en-CA', { timeZone: 'America/Argentina/Buenos_Aires' }).format(new Date());

async function cargar(api: Api, paso: Paso): Promise<Cargado> {
	const [negocio, web, { servicios }, { profesionales }, cobro, ajustes, mp] = await Promise.all([
		api.get<NegocioAlta>('/v1/panel/negocio'),
		api.get<WebAlta>('/v1/panel/mi-web'),
		api.get<{ servicios: ServicioAlta[] }>('/v1/panel/servicios'),
		api.get<{ profesionales: ProfesionalAlta[] }>('/v1/panel/profesionales'),
		api.get<CobroAlta>('/v1/panel/cobros'),
		api.get<Ajustes>('/v1/panel/ajustes'),
		api.get<{ conectada: boolean }>('/v1/panel/mercadopago').catch(() => ({ conectada: false }))
	]);
	const activos = profesionales.filter((p) => p.activo);
	let bloqueos: BloqueoAlta[] = [];
	if (paso === 'agenda' && activos[0])
		bloqueos = (await api.get<{ bloqueos: BloqueoAlta[] }>(`/v1/panel/profesionales/${activos[0].id}/bloqueos`)).bloqueos;
	return { negocio, web, servicios, profesionales: activos, cobro, ajustes, bloqueos, mercadoPago: mp.conectada };
}

export const load: PageServerLoad = async (event) => {
	const { locals, url } = event;
	const borrador = borradorDe(url);
	if (!locals.token)
		return {
			modo: 'alta' as const,
			borrador,
			fapi: fapiDe(PUBLIC_CLERK_PUBLICABLE),
			prueba: dev || !!TENGOLIBRE_INGRESO_PRUEBA,
			origen: url.origin
		};

	const pedido = url.searchParams.get('paso');
	const paso: Paso = esPaso(pedido) ? pedido : borrador.rubro ? 'nombre' : 'rubro';
	const editando = conNegocio(paso) || url.searchParams.get('n') === '1';
	if (editando && !locals.negocio) redirect(303, '/empezar');

	const api = apiDe(event);
	const [persona, cargado] = await Promise.all([
		paso === 'equipo' ? api.get<Persona>('/v1/auth/persona').catch(() => null) : Promise.resolve(null),
		editando ? cargar(api, paso) : Promise.resolve(null)
	]);
	return {
		modo: 'onboarding' as const,
		paso,
		borrador,
		cargado,
		persona: persona?.nombre ?? '',
		feriados: paso === 'agenda' ? proximosFeriados(hoyAR()) : [],
		origen: url.origin
	};
};

/** Sigue al próximo paso del negocio ya creado. */
const ir = (p: Paso): never => redirect(303, urlPaso(siguiente(p)));

async function ajustesCon(event: RequestEvent, cambios: Partial<Ajustes>) {
	const api = apiDe(event);
	const actual = await api.get<Ajustes>('/v1/panel/ajustes');
	return api.put('/v1/panel/ajustes', { ...actual, ...cambios });
}

export const actions: Actions = {
	/** Alta sin sesión: guarda el borrador para volver de Google o Apple al paso del nombre. */
	borrador: async ({ request, cookies }) => {
		const f = await request.formData();
		const q = new URLSearchParams({ paso: 'nombre' });
		for (const k of ['nombre', 'slug', 'rubro']) if (texto(f, k)) q.set(k, texto(f, k).slice(0, 80));
		const volver = `/empezar?${q}`;
		if (!volverValido(volver)) return fail(400, { error: 'Revisá los datos.' });
		cookies.set(COOKIE_VOLVER, volver, { path: '/', httpOnly: true, sameSite: 'lax', maxAge: 900 });
		return { volver };
	},

	/** Paso 1 con el negocio ya creado: cambia el rubro. */
	rubro: async (event) => {
		const f = await event.request.formData();
		const r = rubroDe(texto(f, 'rubro'));
		if (!r) return fail(400, { error: 'Elegí un rubro.' });
		const res = await intentar(() => apiDe(event).patch('/v1/panel/mi-web', { rubro: r.api }));
		if (!('ok' in res)) return res;
		redirect(303, urlPaso('nombre', { n: '1' }));
	},

	/** Paso 2: crea el negocio (o le cambia el nombre si ya estaba creado). */
	nombre: async (event) => {
		const f = await event.request.formData();
		const nombre = texto(f, 'nombre');
		if (f.get('editando') === '1') {
			const r = await intentar(() => apiDe(event).patch('/v1/panel/negocio', { nombre }));
			if (!('ok' in r)) return r;
			ir('nombre');
		}
		const slug = texto(f, 'slug').toLowerCase();
		const rubro = rubroDe(texto(f, 'rubro'));
		const r = await intentar(() => apiDe(event).post<{ id: string }>('/v1/negocios', { nombre, slug }));
		if (!('ok' in r)) return r;
		elegirNegocio(event.cookies, r.datos.id);
		// La cookie recién puesta no llega a este pedido: el negocio va a mano.
		if (rubro) await new Api(event.locals.token, r.datos.id, event.fetch).patch('/v1/panel/mi-web', { rubro: rubro.api }).catch(() => null);
		redirect(303, urlPaso('servicios', rubro ? { rubro: rubro.clave } : {}));
	},

	/** Paso 3: los servicios (los nuevos se crean, los de antes se actualizan o se apagan). */
	servicios: async (event) => {
		const f = await event.request.formData();
		const lista = json<{ id?: string; nombre: string; duracion: number; precio: number; activo: boolean }[]>(f, 'servicios', []);
		if (!lista.some((s) => s.activo)) return fail(400, { error: 'Dejá al menos un servicio.' });
		const api = apiDe(event);
		const r = await intentar(async () => {
			for (const s of lista) {
				const cuerpo = { nombre: s.nombre.trim(), descripcion: '', duracion: s.duracion, precio: Math.max(0, Math.round(s.precio)), categoriaId: null, activo: s.activo };
				if (s.id) await api.patch(`/v1/panel/servicios/${s.id}`, cuerpo);
				else if (s.activo) await api.post('/v1/panel/servicios', cuerpo);
			}
		});
		if (!('ok' in r)) return r;
		ir('servicios');
	},

	/** Paso 4: quién atiende, qué días y en qué horario (con todos los servicios activos). */
	equipo: async (event) => {
		const f = await event.request.formData();
		const personas = json<{ id?: string; nombre: string }[]>(f, 'personas', []).filter((p) => p.nombre.trim());
		const quitar = json<string[]>(f, 'quitar', []);
		const dias = json<number[]>(f, 'dias', []);
		const abre = texto(f, 'abre');
		const cierra = texto(f, 'cierra');
		if (!personas.length) return fail(400, { error: 'Sumá al menos a una persona.' });
		if (!dias.length) return fail(400, { error: 'Elegí al menos un día.' });
		if (minutos(cierra) <= minutos(abre)) return fail(400, { error: 'La hora de cierre tiene que ser después de la de apertura.' });
		const api = apiDe(event);
		const r = await intentar(async () => {
			const { servicios } = await api.get<{ servicios: ServicioAlta[] }>('/v1/panel/servicios');
			const ids = servicios.filter((s) => s.activo).map((s) => s.id);
			const { profesionales } = await api.get<{ profesionales: ProfesionalAlta[] }>('/v1/panel/profesionales');
			for (const id of quitar) await api.patch(`/v1/panel/profesionales/${id}`, { activo: false }).catch(() => null);
			for (const p of personas) {
				let id = p.id;
				if (!id) id = (await api.post<{ id: string }>('/v1/panel/profesionales', { nombre: p.nombre.trim(), especialidad: '', email: '', telefono: '', servicios: ids, activo: true })).id;
				else if (profesionales.find((x) => x.id === id)?.nombre !== p.nombre.trim())
					await api.patch(`/v1/panel/profesionales/${id}`, { nombre: p.nombre.trim() });
				// Si ya tenía descanso cargado (paso «Tu día»), se respeta al volver.
				const antes = profesionales.find((x) => x.id === id)?.horario ?? [];
				const conDescanso = antes.filter((x) => x.dia === antes[0]?.dia).length > 1;
				const desc = conDescanso ? { desde: antes[0].hasta.slice(0, 5), hasta: antes[1].desde.slice(0, 5) } : null;
				await api.put(`/v1/panel/profesionales/${id}/horario`, { horario: franjas(dias, abre, cierra, desc) });
				await api.put(`/v1/panel/profesionales/${id}/servicios`, { servicios: ids });
			}
		});
		if (!('ok' in r)) return r;
		ir('equipo');
	},

	/** Paso 5: descanso, tiempo entre turnos, anticipación y feriados. */
	agenda: async (event) => {
		const f = await event.request.formData();
		const descanso = f.get('descanso') === 'on' ? { desde: texto(f, 'descDesde'), hasta: texto(f, 'descHasta') } : null;
		const entre = Number(texto(f, 'entre')) || 0;
		const antic = Number(texto(f, 'antic')) || 0;
		const cerrados = json<string[]>(f, 'cerrados', []).filter((x) => /^\d{4}-\d{2}-\d{2}$/.test(x));
		const especiales = json<string[]>(f, 'especiales', []).filter((x) => /^\d{4}-\d{2}-\d{2}$/.test(x));
		const ferAbre = texto(f, 'ferAbre');
		const ferCierra = texto(f, 'ferCierra');
		const api = apiDe(event);
		const r = await intentar(async () => {
			await ajustesCon(event, { margenMin: entre, anticipacionMin: antic });
			const { profesionales } = await api.get<{ profesionales: ProfesionalAlta[] }>('/v1/panel/profesionales');
			for (const p of profesionales.filter((x) => x.activo && x.horario.length)) {
				const dias = [...new Set(p.horario.map((x) => (x.dia === 0 ? 7 : x.dia)))];
				const delDia = p.horario.filter((x) => x.dia === p.horario[0].dia).sort((a, b) => a.desde.localeCompare(b.desde));
				const abre = delDia[0].desde.slice(0, 5);
				const cierra = delDia[delDia.length - 1].hasta.slice(0, 5);
				await api.put(`/v1/panel/profesionales/${p.id}/horario`, { horario: franjas(dias, abre, cierra, descanso) });
				// Feriados: los cerrados bloquean el día; los de horario especial, lo de antes y lo de después.
				const { bloqueos } = await api.get<{ bloqueos: BloqueoAlta[] }>(`/v1/panel/profesionales/${p.id}/bloqueos`);
				const ya = new Set(bloqueos.map((x) => x.fecha.slice(0, 10)));
				const bloquear = (fecha: string, desde: string | null, hasta: string | null) =>
					api.post(`/v1/panel/profesionales/${p.id}/bloqueos`, { fecha, desde, hasta, motivo: 'Feriado' });
				for (const fecha of cerrados) if (!ya.has(fecha)) await bloquear(fecha, null, null);
				if (minutos(ferCierra) > minutos(ferAbre))
					for (const fecha of especiales) {
						if (ya.has(fecha)) continue;
						for (const [desde, hasta] of [
							[abre, ferAbre],
							[ferCierra, cierra]
						])
							if (minutos(hasta) > minutos(desde)) await bloquear(fecha, hhmm(minutos(desde)), hhmm(minutos(hasta)));
					}
			}
		});
		if (!('ok' in r)) return r;
		ir('agenda');
	},

	/** Paso 6: la seña. */
	cobro: async (event) => {
		const f = await event.request.formData();
		const modo = texto(f, 'modo');
		const r = await intentar(() =>
			apiDe(event).put('/v1/panel/cobros', {
				modo,
				// Siempre 30: con «sin cobro» marca que se eligió en este paso (la base trae 50).
				senaPorcentaje: 30,
				senaFija: modo === 'fija' ? Math.round(Number(texto(f, 'senaFija')) || 0) : 0
			})
		);
		if (!('ok' in r)) return r;
		ir('cobro');
	},

	/** Paso 7: color y plantilla. Publica el estilo entero. */
	web: async (event) => {
		const f = await event.request.formData();
		const estilo = { ...ESTILO_BASE, ...json<Partial<typeof ESTILO_BASE>>(f, 'estilo', {}) };
		const r = await intentar(() => apiDe(event).patch('/v1/panel/mi-web', { colorPrimario: texto(f, 'color'), estilo }));
		if (!('ok' in r)) return r;
		ir('web');
	}
};
