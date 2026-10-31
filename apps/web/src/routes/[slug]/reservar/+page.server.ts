import { fail, redirect } from '@sveltejs/kit';
import { apiDe, apiPublica } from '#lib/server/api.ts';
import { campos, intentar } from '#lib/server/acciones.ts';
import { cobro, opcional, pagina } from '#lib/publico/api.server.ts';
import { esFecha } from '#lib/publico/calendario.ts';
import { CUALQUIERA, type Paso } from '#lib/publico/reserva.ts';
import type { Dia, Horario, MiMembresia, Turno, Validacion } from '#lib/publico/tipos.ts';
import type { Actions, PageServerLoad } from './$types';


export const load: PageServerLoad = async (event) => {
	const { params, url, fetch, locals } = event;
	const api = apiPublica(fetch);
	const slug = params.slug;
	const p = await pagina(fetch, slug);
	const q = url.searchParams;

	const servicio = p.servicios.find((s) => s.id === q.get('servicio')) ?? null;
	const profesionales = servicio ? p.profesionales.filter((x) => x.servicios.includes(servicio.id)) : [];
	// Con un solo profesional para el servicio no hay nada que elegir.
	let profesionalId = q.get('profesional') ?? '';
	if (profesionales.length === 1) profesionalId = profesionales[0].id;
	if (profesionalId !== CUALQUIERA && !profesionales.some((x) => x.id === profesionalId)) profesionalId = '';
	const profesional = profesionales.find((x) => x.id === profesionalId) ?? null;
	const fecha = esFecha(q.get('fecha')) ? q.get('fecha')! : '';
	const inicio = q.get('inicio') ?? '';

	let paso: Paso = 'servicio';
	let dias: Dia[] = [];
	let horarios: Horario[] = [];
	let horario: Horario | null = null;
	let aviso = '';
	const filtro = (extra: Record<string, string>) => {
		const s = new URLSearchParams({ servicio: servicio!.id, ...extra });
		if (profesional) s.set('profesional', profesional.id);
		return s;
	};

	if (servicio) paso = 'profesional';
	if (servicio && profesionales.length === 0) aviso = 'Por ahora nadie del equipo hace este servicio online.';
	if (servicio && profesionalId) {
		paso = 'dia';
		dias = (await api.get<{ dias: Dia[] }>(`/v1/p/${slug}/dias?${filtro({})}`)).dias;
	}
	if (paso === 'dia' && fecha) {
		paso = 'horario';
		horarios = (await api.get<{ horarios: Horario[] }>(`/v1/p/${slug}/libres?${filtro({ fecha })}`)).horarios;
		if (inicio) {
			horario = horarios.find((h) => Date.parse(h.inicio) === Date.parse(inicio)) ?? null;
			if (horario) paso = 'datos';
			else aviso = 'Ese horario ya no está libre. Elegí otro.';
		}
	}

	const c = paso === 'datos' ? await cobro(fetch, slug) : null;
	// Socios: con sesión, su membresía (si cubre el servicio, la puede usar); sin sesión, si el negocio
	// tiene planes, la invitación a ingresar.
	let membresia: MiMembresia | null = null;
	let conPlanes = false;
	if (paso === 'datos' && servicio) {
		if (locals.token) {
			const m = await opcional(apiDe(event).get<{ membresia: MiMembresia | null }>(`/v1/p/${slug}/mi-membresia`), { membresia: null });
			if (m.membresia && m.membresia.saldo > 0 && (m.membresia.servicios.length === 0 || m.membresia.servicios.includes(servicio.id)))
				membresia = m.membresia;
		} else {
			const pl = await opcional(api.get<{ planes: unknown[] }>(`/v1/p/${slug}/membresias`), { planes: [] });
			conPlanes = pl.planes.length > 0;
		}
	}
	return {
		negocio: p.negocio,
		categorias: p.categorias,
		servicios: p.servicios,
		politicaCancelacion: p.politicaCancelacion,
		paso,
		servicio,
		profesionales,
		profesionalId,
		profesional,
		fecha,
		dias,
		horarios,
		horario,
		aviso,
		cobro: c,
		membresia,
		conPlanes,
		ingresado: !!locals.token,
		referido: (q.get('ref') ?? '').slice(0, 40)
	};
};

const texto = (f: FormData, k: string) => String(f.get(k) ?? '').trim();

export const actions: Actions = {
	reservar: async (event) => {
		const { request, params } = event;
		const f = await request.formData();
		const valores = campos(f);
		const profesionalId = texto(f, 'profesionalId');
		const r = await intentar(() =>
			// Con la sesión, si la hay: la API la usa sólo si se pide usar la membresía.
			apiDe(event).post<Turno>(`/v1/p/${params.slug}/turnos`, {
				servicioId: texto(f, 'servicioId'),
				profesionalId: profesionalId === CUALQUIERA ? '' : profesionalId,
				inicio: texto(f, 'inicio'),
				cliente: { nombre: texto(f, 'nombre'), telefono: texto(f, 'telefono'), email: texto(f, 'email') },
				nota: texto(f, 'nota'),
				cupon: texto(f, 'cupon') || undefined,
				referido: texto(f, 'referido') || undefined,
				usarMembresia: f.get('usarMembresia') === '1' || undefined
			})
		);
		if (!('ok' in r)) return fail(r.status, { ...r.data, valores });
		const t = r.datos;
		// Si el negocio cobra seña, derecho a Mercado Pago; sin link (MP no respondió), lo pide el link del turno.
		if (t.pago?.url && /^https:\/\//.test(t.pago.url)) redirect(303, t.pago.url);
		redirect(303, `/turno/${t.codigo}?nuevo=1`);
	},

	cupon: async ({ request, params, fetch }) => {
		const f = await request.formData();
		const valores = campos(f);
		const codigo = texto(f, 'cupon');
		if (!codigo) return fail(400, { valores, cupon: { valido: false, motivo: 'Escribí el código del cupón.' } as Validacion });
		const v = await opcional(
			apiPublica(fetch).post<Validacion>(`/v1/p/${params.slug}/cupones/validar`, { codigo, servicioId: texto(f, 'servicioId') }),
			{ valido: false, motivo: 'No pudimos revisar el cupón. Probá de nuevo.' } as Validacion
		);
		return { valores, cupon: v };
	},

	espera: async ({ request, params, fetch }) => {
		const f = await request.formData();
		const valores = campos(f);
		const profesionalId = texto(f, 'profesionalId');
		const r = await intentar(() =>
			apiPublica(fetch).post<{ servicio: string; desde: string; hasta: string }>(`/v1/p/${params.slug}/espera`, {
				servicioId: texto(f, 'servicioId'),
				profesionalId: profesionalId === CUALQUIERA ? '' : profesionalId,
				nombre: texto(f, 'nombre'),
				email: texto(f, 'email'),
				telefono: texto(f, 'telefono'),
				desde: texto(f, 'desde'),
				hasta: texto(f, 'hasta')
			})
		);
		if (!('ok' in r)) return fail(r.status, { ...r.data, valores, enEspera: true });
		return { anotado: r.datos };
	}
};
