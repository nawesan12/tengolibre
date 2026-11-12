import { describe, expect, it } from 'vitest';
import {
	completar,
	confirmarCodigo,
	empezarConMail,
	entrarConClave,
	ErrorClerk,
	fapiDe,
	iniciar,
	requisitosClave
} from './clerk.ts';

const FAPI = 'https://clerk.grupovimo.com.ar';

class Guardado implements Storage {
	m = new Map<string, string>();
	get length() {
		return this.m.size;
	}
	clear() {
		this.m.clear();
	}
	getItem(k: string) {
		return this.m.get(k) ?? null;
	}
	key(i: number) {
		return [...this.m.keys()][i] ?? null;
	}
	removeItem(k: string) {
		this.m.delete(k);
	}
	setItem(k: string, v: string) {
		this.m.set(k, v);
	}
}

type Pedido = {
	url: URL;
	metodo: string;
	auth: string | null;
	cuerpo: string;
	credenciales?: string;
};

/** Frontend API falsa: responde según la ruta y rota el token del cliente en cada respuesta. */
function fapiFalsa(rutas: Record<string, unknown>) {
	const pedidos: Pedido[] = [];
	let n = 0;
	const f = (async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = new URL(String(input));
		const h = new Headers(init?.headers);
		pedidos.push({
			url,
			metodo: init?.method ?? 'GET',
			auth: h.get('authorization'),
			cuerpo: String(init?.body ?? ''),
			credenciales: init?.credentials
		});
		const clave = `${init?.method ?? 'GET'} ${url.pathname}`;
		if (!(clave in rutas))
			return new Response(JSON.stringify({ errors: [{ code: 'resource_not_found' }] }), {
				status: 404
			});
		return new Response(JSON.stringify(rutas[clave]), {
			headers: { authorization: `cliente-${++n}` }
		});
	}) as typeof fetch;
	return { f, pedidos };
}

describe('fapiDe', () => {
	it('saca la Frontend API de la clave publicable', () => {
		expect(fapiDe('pk_live_Y2xlcmsuZ3J1cG92aW1vLmNvbS5hciQ')).toBe(FAPI);
		expect(fapiDe('pk_test_cHJlY2lzZS1zd2FuLTgwMjIuY2xlcmsuYWNjb3VudHMuZGV2JA')).toBe(
			'https://precise-swan-8022.clerk.accounts.dev'
		);
		expect(fapiDe('')).toBeNull();
		expect(fapiDe(undefined)).toBeNull();
		expect(fapiDe('sk_live_abc')).toBeNull();
		expect(fapiDe(`pk_live_${btoa('evil.com/x$')}`)).toBeNull();
	});
});

describe('ida y vuelta con Clerk', () => {
	const inicio = {
		'POST /v1/client/sign_ins': {
			response: {
				id: 'sia_1',
				status: 'needs_identifier',
				first_factor_verification: {
					status: 'unverified',
					external_verification_redirect_url: 'https://accounts.google.com/o/oauth2/auth?x=1'
				}
			}
		}
	};

	it('una persona que ya está en Clerk: recarga con el nonce, pide el token y cierra la sesión', async () => {
		const g = new Guardado();
		const { f, pedidos } = fapiFalsa({
			...inicio,
			'GET /v1/client/sign_ins/sia_1': {
				response: { id: 'sia_1', status: 'complete', created_session_id: 'sess_1' }
			},
			'POST /v1/client/sessions/sess_1/tokens': { jwt: 'jwt-de-clerk' },
			'POST /v1/client/sessions/sess_1/remove': { response: {} }
		});
		const url = await iniciar(FAPI, 'google', 'https://tengolibre.com.ar/ingresar/clerk', g, f);
		expect(url).toBe('https://accounts.google.com/o/oauth2/auth?x=1');
		expect(pedidos[0].cuerpo).toBe(
			'strategy=oauth_google&redirect_url=https%3A%2F%2Ftengolibre.com.ar%2Fingresar%2Fclerk'
		);
		// Modo nativo y sin cookies: el cliente va en Authorization.
		expect(pedidos[0].url.searchParams.get('_is_native')).toBe('1');
		expect(pedidos[0].credenciales).toBe('omit');

		expect(await completar('nonce-1', g, f)).toBe('jwt-de-clerk');
		const [, recarga, token, cierre] = pedidos;
		expect(recarga.url.searchParams.get('rotating_token_nonce')).toBe('nonce-1');
		expect(recarga.auth).toBe('cliente-1');
		// Vale siempre el último token del cliente que mandó Clerk.
		expect(token.auth).toBe('cliente-2');
		expect(cierre.url.pathname).toBe('/v1/client/sessions/sess_1/remove');
		// No queda nada guardado en el navegador.
		expect(g.length).toBe(0);
	});

	it('una persona nueva en Clerk: transfiere el ingreso a un alta', async () => {
		const g = new Guardado();
		const { f, pedidos } = fapiFalsa({
			...inicio,
			'GET /v1/client/sign_ins/sia_1': {
				response: {
					id: 'sia_1',
					status: 'needs_identifier',
					created_session_id: null,
					first_factor_verification: { status: 'transferable' }
				}
			},
			'POST /v1/client/sign_ups': {
				response: { id: 'sua_1', status: 'complete', created_session_id: 'sess_2' }
			},
			'POST /v1/client/sessions/sess_2/tokens': { jwt: 'jwt-nuevo' }
		});
		await iniciar(FAPI, 'apple', 'https://tengolibre.com.ar/ingresar/clerk', g, f);
		expect(pedidos[0].cuerpo).toContain('strategy=oauth_apple');
		// El cierre de la sesión de Clerk falla (404 en el falso): igual se entra.
		expect(await completar('nonce-2', g, f)).toBe('jwt-nuevo');
		expect(pedidos.find((p) => p.url.pathname === '/v1/client/sign_ups')?.cuerpo).toBe(
			'transfer=true'
		);
	});

	it('una persona nueva vuelve sin nonce: igual se consulta el ingreso y se hace el alta', async () => {
		const g = new Guardado();
		const { f, pedidos } = fapiFalsa({
			...inicio,
			'GET /v1/client/sign_ins/sia_1': {
				response: {
					id: 'sia_1',
					created_session_id: null,
					first_factor_verification: { status: 'transferable' }
				}
			},
			'POST /v1/client/sign_ups': {
				response: { id: 'sua_1', status: 'complete', created_session_id: 'sess_3' }
			},
			'POST /v1/client/sessions/sess_3/tokens': { jwt: 'jwt-sin-nonce' }
		});
		await iniciar(FAPI, 'google', 'https://tengolibre.com.ar/ingresar/clerk', g, f);
		expect(await completar('', g, f)).toBe('jwt-sin-nonce');
		const consulta = pedidos.find((p) => p.url.pathname === '/v1/client/sign_ins/sia_1');
		expect(consulta?.url.searchParams.has('rotating_token_nonce')).toBe(false);
	});

	it('sin el estado guardado o si canceló en el proveedor, no sigue', async () => {
		const g = new Guardado();
		const { f } = fapiFalsa({
			...inicio,
			'GET /v1/client/sign_ins/sia_1': {
				response: {
					id: 'sia_1',
					created_session_id: null,
					first_factor_verification: { status: 'failed' }
				}
			}
		});
		await expect(completar('n', g, f)).rejects.toEqual(new ErrorClerk('estado_invalido'));
		await iniciar(FAPI, 'google', 'https://tengolibre.com.ar/ingresar/clerk', g, f);
		await expect(completar('', g, f)).rejects.toEqual(new ErrorClerk('cancelado'));
		// El estado es de un solo uso.
		expect(g.length).toBe(0);
	});
});

describe('con mail y contraseña', () => {
	const conClave = {
		'POST /v1/client/sign_ins': {
			response: {
				id: 'sia_2',
				status: 'needs_first_factor',
				supported_first_factors: [{ strategy: 'password' }, { strategy: 'email_code', email_address_id: 'idn_1' }]
			}
		}
	};

	it('entra con la contraseña y cierra la sesión de Clerk', async () => {
		const { f, pedidos } = fapiFalsa({
			...conClave,
			'POST /v1/client/sign_ins/sia_2/attempt_first_factor': {
				response: { id: 'sia_2', status: 'complete', created_session_id: 'sess_2' }
			},
			'POST /v1/client/sessions/sess_2/tokens': { jwt: 'jwt-mail' },
			'POST /v1/client/sessions/sess_2/remove': { response: {} }
		});
		const { flujo, sigue } = await empezarConMail(FAPI, 'ana@x.com', f);
		expect(sigue).toEqual({ paso: 'clave' });
		expect(await entrarConClave(flujo, 'una-clave-larga', f)).toEqual({ paso: 'listo', token: 'jwt-mail' });
		expect(pedidos[1].cuerpo).toBe('strategy=password&password=una-clave-larga');
		expect(pedidos.at(-1)?.url.pathname).toBe('/v1/client/sessions/sess_2/remove');
	});

	it('desde un dispositivo nuevo (needs_client_trust) pide el código por mail y entra', async () => {
		const { f, pedidos } = fapiFalsa({
			...conClave,
			'POST /v1/client/sign_ins/sia_2/attempt_first_factor': {
				response: {
					id: 'sia_2',
					status: 'needs_client_trust',
					supported_second_factors: [{ strategy: 'email_code', email_address_id: 'idn_1' }]
				}
			},
			'POST /v1/client/sign_ins/sia_2/prepare_second_factor': { response: { id: 'sia_2', status: 'needs_client_trust' } },
			'POST /v1/client/sign_ins/sia_2/attempt_second_factor': {
				response: { id: 'sia_2', status: 'complete', created_session_id: 'sess_3' }
			},
			'POST /v1/client/sessions/sess_3/tokens': { jwt: 'jwt-confiado' },
			'POST /v1/client/sessions/sess_3/remove': { response: {} }
		});
		const { flujo } = await empezarConMail(FAPI, 'ana@x.com', f);
		expect(await entrarConClave(flujo, 'una-clave-larga', f)).toEqual({ paso: 'codigo', alta: false, confiar: true });
		const prep = pedidos.find((p) => p.url.pathname.endsWith('/prepare_second_factor'));
		expect(prep?.cuerpo).toBe('strategy=email_code&email_address_id=idn_1');
		expect(await confirmarCodigo(flujo, '12 34 56', false, f)).toEqual({ paso: 'listo', token: 'jwt-confiado' });
		const intento = pedidos.find((p) => p.url.pathname.endsWith('/attempt_second_factor'));
		expect(intento?.cuerpo).toBe('strategy=email_code&code=123456');
	});

	it('sin código por mail como segundo factor, avisa', async () => {
		const { f } = fapiFalsa({
			...conClave,
			'POST /v1/client/sign_ins/sia_2/attempt_first_factor': {
				response: { id: 'sia_2', status: 'needs_client_trust', supported_second_factors: [{ strategy: 'phone_code' }] }
			}
		});
		const { flujo } = await empezarConMail(FAPI, 'ana@x.com', f);
		await expect(entrarConClave(flujo, 'una-clave-larga', f)).rejects.toThrow(ErrorClerk);
	});

	it('alguien nuevo se da de alta con el código', async () => {
		const { f } = fapiFalsa({
			'POST /v1/client/sign_ups': { response: { id: 'sua_1', status: 'missing_requirements' } },
			'POST /v1/client/sign_ups/sua_1/prepare_verification': { response: { id: 'sua_1' } }
		});
		// sign_ins responde 404 con resource_not_found: se fuerza el código de «no existe».
		const f2 = (async (input: RequestInfo | URL, init?: RequestInit) => {
			if (String(input).includes('/client/sign_ins'))
				return new Response(JSON.stringify({ errors: [{ code: 'form_identifier_not_found' }] }), { status: 422 });
			return f(input, init);
		}) as typeof fetch;
		const { flujo, sigue } = await empezarConMail(FAPI, 'nueva@x.com', f2);
		expect(sigue).toEqual({ paso: 'codigo', alta: true });
		expect(flujo.alta).toBe(true);
	});

	it('la contraseña nueva pide 8 caracteres y que no sea de las más usadas', () => {
		expect(requisitosClave('corta')).toEqual({ largo: false, comun: true });
		expect(requisitosClave('1234567890')).toEqual({ largo: true, comun: false });
		expect(requisitosClave('turnos-en-la-plaza')).toEqual({ largo: true, comun: true });
	});
});
