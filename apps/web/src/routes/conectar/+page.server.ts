import { fail, redirect } from '@sveltejs/kit';
import { apiDe, ErrorApi, type NegocioDeLaPersona } from '#lib/server/api.ts';
import type { Actions, PageServerLoad } from './$types';

/**
 * Consentimiento del OAuth del servidor MCP: un asistente (claude.ai, ChatGPT) pide acceso, la API
 * guarda el pedido y manda al navegador acá con ?solicitud=. La persona elige el negocio y los
 * permisos; «Permitir» devuelve la URL del asistente con el código. Errores del pedido en sí
 * (cliente desconocido, redirect_uri que no cierra) llegan como ?error=.
 */

export type Permiso = { clave: string; nombre: string; descripcion: string; herramientas: string[] };
export type Solicitud = {
	cliente: string;
	destino: string;
	negocios: NegocioDeLaPersona[];
	permisos: Permiso[];
	porDefecto: string[];
	venceEn: string;
};

const valida = (s: string | null): s is string => !!s && /^[A-Za-z0-9_-]{20,64}$/.test(s);
const ingresar = (s: string) => `/ingresar?volver=${encodeURIComponent(`/conectar?solicitud=${s}`)}`;

export const load: PageServerLoad = async (event) => {
	const { url, locals, setHeaders } = event;
	// Nada de cachear ni de mostrarla dentro de un iframe ajeno (clickjacking sobre «Permitir»).
	setHeaders({ 'cache-control': 'no-store', 'x-frame-options': 'DENY', 'content-security-policy': "frame-ancestors 'none'" });
	const error = url.searchParams.get('error');
	const solicitud = url.searchParams.get('solicitud');
	if (error || !valida(solicitud)) return { estado: 'error' as const, codigo: error ?? 'sin_solicitud' };
	if (!locals.token) redirect(303, ingresar(solicitud));
	try {
		const s = await apiDe(event).get<Solicitud>(`/v1/oauth/solicitudes/${solicitud}`);
		return { estado: 'pedido' as const, solicitud, s };
	} catch (e) {
		if (!(e instanceof ErrorApi)) throw e;
		if (e.estado === 401) redirect(303, ingresar(solicitud));
		if (e.estado === 404) return { estado: 'error' as const, codigo: 'vencida' };
		throw e;
	}
};

export const actions: Actions = {
	permitir: async (event) => {
		const f = await event.request.formData();
		const solicitud = String(f.get('solicitud') ?? '');
		if (!valida(solicitud)) return fail(400, { error: 'El pedido no es válido.' });
		if (!event.locals.token) redirect(303, ingresar(solicitud));
		const negocioId = String(f.get('negocio') ?? '');
		const permisos = f.getAll('permiso').map(String);
		let destino: string;
		try {
			destino = (await apiDe(event).post<{ redirect: string }>(`/v1/oauth/solicitudes/${solicitud}/permitir`, { negocioId, permisos }))
				.redirect;
		} catch (e) {
			if (!(e instanceof ErrorApi)) throw e;
			if (e.estado === 401) redirect(303, ingresar(solicitud));
			if (e.estado === 404) return fail(404, { error: 'El pedido venció. Volvé a conectar desde tu asistente.' });
			return fail(e.estado, { error: e.message });
		}
		redirect(303, destino, { external: true });
	},
	cancelar: async (event) => {
		const solicitud = String((await event.request.formData()).get('solicitud') ?? '');
		if (!valida(solicitud)) return fail(400, { error: 'El pedido no es válido.' });
		if (!event.locals.token) redirect(303, ingresar(solicitud));
		let destino: string;
		try {
			destino = (await apiDe(event).post<{ redirect: string }>(`/v1/oauth/solicitudes/${solicitud}/cancelar`)).redirect;
		} catch (e) {
			if (!(e instanceof ErrorApi)) throw e;
			if (e.estado === 404) redirect(303, '/conectar?error=vencida');
			return fail(e.estado, { error: e.message });
		}
		redirect(303, destino, { external: true });
	}
};
