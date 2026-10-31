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
		public campos: Record<string, string> = {}
	) {
		super(message);
	}
}

type Problema = { detail?: string; title?: string; code?: string; issues?: { path: string; message: string }[] };

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
		const res = await this.f(BASE_API() + ruta, {
			method: metodo,
			headers,
			body: cuerpo === undefined ? undefined : JSON.stringify(cuerpo)
		});
		if (res.status === 204 || res.status === 202) return undefined as T;
		const json = (await res.json().catch(() => null)) as (T & Problema) | null;
		if (!res.ok) {
			const campos = Object.fromEntries((json?.issues ?? []).map((i) => [i.path, i.message]));
			throw new ErrorApi(
				res.status,
				json?.code ?? 'error',
				json?.detail ?? json?.title ?? 'No pudimos completar el pedido.',
				campos
			);
		}
		return json as T;
	}

	get = <T>(ruta: string) => this.pedir<T>('GET', ruta);
	post = <T>(ruta: string, cuerpo?: unknown) => this.pedir<T>('POST', ruta, cuerpo ?? {});
	put = <T>(ruta: string, cuerpo: unknown) => this.pedir<T>('PUT', ruta, cuerpo);
	patch = <T>(ruta: string, cuerpo: unknown) => this.pedir<T>('PATCH', ruta, cuerpo);
	borrar = <T>(ruta: string) => this.pedir<T>('DELETE', ruta);

	/** Pedido crudo (CSV, archivos). */
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
