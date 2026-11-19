import { TENGOLIBRE_API } from '$app/env/private';
import type { RequestEvent } from '@sveltejs/kit';

/**
 * Cliente de la API Go desde el servidor de SvelteKit. El navegador nunca habla con la API: lo hace
 * este servidor, con la sesión de la cookie httpOnly y el negocio elegido. Los errores
 * problem+json vuelven como ErrorApi, listos para mostrar en el formulario.
 */
export const BASE_API = () => TENGOLIBRE_API;

export class ErrorApi extends Error {
	constructor(
		public estado: number,
		public codigo: string,
		message: string,
		public campos: Record<string, string> = {},
		/** Con tope_plan: el plan que lo habilita (inicial, equipo, centro). */
		public plan?: string
	) {
		super(message);
	}
}

type Problema = { detail?: string; title?: string; code?: string; plan?: string; issues?: { path: string; message: string }[] };

export type Persona = { clerkId: string; email: string; nombre: string; foto: string };
export type NegocioDeLaPersona = { id: string; slug: string; nombre: string; rol: string };

type Metodo = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';

export class Api {
	constructor(
		private token: string | null,
		private negocio: string | null,
		private f: typeof fetch = fetch
	) {}

	async pedir<T>(metodo: Metodo, ruta: string, cuerpo?: unknown): Promise<T> {
		const headers: Record<string, string> = { accept: 'application/json' };
		if (cuerpo !== undefined) headers['content-type'] = 'application/json';
		if (this.token) headers.authorization = `Bearer ${this.token}`;
		if (this.negocio && ruta.startsWith('/v1/panel/')) headers['x-negocio'] = this.negocio;
		// Con plazo: si la API se queda esperando (la base, Anthropic), la página muestra el error en vez de
		// quedar colgada. El asistente tiene más (la IA puede tardar unos 22 s).
		let res: Response;
		try {
			res = await this.f(BASE_API() + ruta, {
				method: metodo,
				headers,
				body: cuerpo === undefined ? undefined : JSON.stringify(cuerpo),
				signal: AbortSignal.timeout(ruta.startsWith('/v1/panel/asistente') ? 30_000 : 12_000)
			});
		} catch (e) {
			if (e instanceof DOMException && (e.name === 'TimeoutError' || e.name === 'AbortError'))
				throw new ErrorApi(504, 'sin_respuesta', 'Tardó demasiado. Probá de nuevo.');
			throw e;
		}
		if (res.status === 204 || res.status === 202) return undefined as T;
		const json = (await res.json().catch(() => null)) as (T & Problema) | null;
		if (!res.ok) {
			const campos = Object.fromEntries((json?.issues ?? []).map((i) => [i.path, i.message]));
			throw new ErrorApi(
				res.status,
				json?.code ?? 'error',
				json?.detail ?? json?.title ?? 'No pudimos completar el pedido.',
				campos,
				json?.plan || undefined
			);
		}
		return json as T;
	}

	get = <T>(ruta: string) => this.pedir<T>('GET', ruta);
	post = <T>(ruta: string, cuerpo?: unknown) => this.pedir<T>('POST', ruta, cuerpo ?? {});
	put = <T>(ruta: string, cuerpo: unknown) => this.pedir<T>('PUT', ruta, cuerpo);
	patch = <T>(ruta: string, cuerpo: unknown) => this.pedir<T>('PATCH', ruta, cuerpo);
	borrar = <T>(ruta: string) => this.pedir<T>('DELETE', ruta);

	/** Pedido crudo (CSV, archivos, el SSE del panel). Sin plazo general: cada llamada pasa el suyo en init.signal. */
	async crudo(ruta: string, init: RequestInit = {}): Promise<Response> {
		const headers = new Headers(init.headers);
		if (this.token) headers.set('authorization', `Bearer ${this.token}`);
		if (this.negocio && ruta.startsWith('/v1/panel/')) headers.set('x-negocio', this.negocio);
		return this.f(BASE_API() + ruta, { ...init, headers });
	}
}

/** El cliente para el pedido en curso (con la sesión y el negocio de las cookies). */
export const apiDe = (e: Pick<RequestEvent, 'locals' | 'fetch'>) => new Api(e.locals.token, e.locals.negocio, e.fetch);

/** Sin sesión y sin negocio: la página pública. */
export const apiPublica = (f: typeof fetch = fetch) => new Api(null, null, f);
