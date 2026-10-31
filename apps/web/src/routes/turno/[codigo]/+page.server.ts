import { error, fail, redirect } from '@sveltejs/kit';
import { apiPublica } from '#lib/server/api.ts';
import { intentar } from '#lib/server/acciones.ts';
import { cobro, codigoValido, o404, opcional } from '#lib/publico/api.server.ts';
import { esFecha } from '#lib/publico/calendario.ts';
import type { Dia, Horario, MiReferido, Sugerencia, TurnoCliente, Web } from '#lib/publico/tipos.ts';
import type { Actions, PageServerLoad } from './$types';

const ACTIVOS = ['pendiente', 'confirmado', 'pendiente_pago'];
const activo = (e: string) => ACTIVOS.includes(e);

export const load: PageServerLoad = async ({ params, url, fetch, setHeaders }) => {
	if (!codigoValido(params.codigo)) error(404, 'No encontramos ese turno.');
	setHeaders({ 'cache-control': 'no-store' });
	const api = apiPublica(fetch);
	const c = params.codigo;
	const [turno, referido, sugerencias] = await Promise.all([
		o404(api.get<TurnoCliente>(`/v1/t/${c}`), 'No encontramos ese turno. Revisá el link.'),
		opcional(api.get<MiReferido>(`/v1/t/${c}/referido`), null),
		opcional(api.get<{ sugerencias: Sugerencia[] }>(`/v1/t/${c}/sugerencias`), { sugerencias: [] })
	]);

	// Reprogramar: el mismo servicio y profesional, en otro día y horario (las mismas reglas que cancelar).
	const puedeCambiar = turno.puedeCancelar && !!turno.servicioId && ACTIVOS.includes(turno.estado);
	const cambiar = puedeCambiar && url.searchParams.has('cambiar');
	let fecha = esFecha(url.searchParams.get('fecha')) ? url.searchParams.get('fecha')! : '';
	let dias: Dia[] = [];
	let horarios: Horario[] = [];
	if (cambiar) {
		const q = new URLSearchParams({ servicio: turno.servicioId! });
		if (turno.profesionalId) q.set('profesional', turno.profesionalId);
		const slug = turno.negocio.slug;
		dias = (await opcional(api.get<{ dias: Dia[] }>(`/v1/p/${slug}/dias?${q}`), { dias: [] })).dias;
		// Sin día elegido: el primero con lugar (las horas se ven de una, como en el diseño).
		if (!fecha) fecha = dias.find((d) => d.libres > 0)?.fecha ?? '';
		if (fecha) {
			q.set('fecha', fecha);
			horarios = (await opcional(api.get<{ horarios: Horario[] }>(`/v1/p/${slug}/libres?${q}`), { horarios: [] })).horarios;
		}
	}

	// El color y el estilo del negocio (el talón se pinta con su marca) y la seña si falta pagarla.
	const [web, conf] = await Promise.all([
		opcional(api.get<Web>(`/v1/p/${turno.negocio.slug}/web`), null),
		turno.estado === 'pendiente_pago' ? cobro(fetch, turno.negocio.slug) : null
	]);

	return {
		turno,
		web,
		cobro: conf,
		cancelar: activo(turno.estado) && turno.puedeCancelar && url.searchParams.has('cancelar') && !cambiar,
		referido,
		sugerencias: sugerencias.sugerencias,
		puedeCambiar,
		cambiar,
		fecha,
		dias,
		horarios,
		nuevo: url.searchParams.has('nuevo'),
		pago: url.searchParams.get('pago')
	};
};

export const actions: Actions = {
	cancelar: async ({ params, fetch }) => {
		const r = await intentar(() => apiPublica(fetch).post<TurnoCliente>(`/v1/t/${params.codigo}/cancelar`));
		if (!('ok' in r)) return r;
		return { cancelado: true };
	},
	reprogramar: async ({ params, fetch, request }) => {
		const f = await request.formData();
		const inicio = String(f.get('inicio') ?? '');
		if (isNaN(Date.parse(inicio))) return fail(400, { error: 'Elegí un horario.', codigo: 'invalido', campos: {} });
		const r = await intentar(() => apiPublica(fetch).post<TurnoCliente>(`/v1/t/${params.codigo}/reprogramar`, { inicio }));
		if (!('ok' in r)) return r;
		redirect(303, `/turno/${params.codigo}?cambiado=1`);
	},
	pagar: async ({ params, fetch }) => {
		const r = await intentar(() => apiPublica(fetch).post<{ url: string; monto: number }>(`/v1/t/${params.codigo}/pagar`));
		if (!('ok' in r)) return r;
		if (!/^https:\/\//.test(r.datos.url)) return fail(502, { error: 'No pudimos armar el link de pago. Probá en un rato.', codigo: 'sin_link', campos: {} });
		redirect(303, r.datos.url);
	},
	referido: async ({ params, fetch }) => {
		const r = await intentar(() => apiPublica(fetch).post<MiReferido>(`/v1/t/${params.codigo}/referido`));
		if (!('ok' in r)) return r;
		return { referidoCreado: true };
	}
};
