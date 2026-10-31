/**
 * Ingreso con Google y Apple por Clerk, la identidad compartida de Grupo Vimo.
 *
 * TengoLibre (tengolibre.com.ar) no vive en el dominio de la instancia (grupovimo.com.ar), así que
 * el SDK de navegador de Clerk no puede usar sus cookies sin pagar dominios satélite. En cambio se
 * habla con la Frontend API en modo nativo, igual que la app de iPhone: sin cookies, con el token del
 * cliente de Clerk en `Authorization`, guardado en sessionStorage sólo mientras dura el ida y vuelta.
 *
 * 1. `iniciar`: crea el intento de ingreso (`oauth_google` / `oauth_apple`) y manda al proveedor.
 * 2. El proveedor vuelve a Clerk y Clerk a `/ingresar/clerk?rotating_token_nonce=…` (la URL tiene que
 *    estar en la lista de redirect URLs de la instancia).
 * 3. `completar`: con el nonce recarga el intento; si la persona no tenía cuenta en Clerk, la crea
 *    (`transfer`). Pide el token de sesión, cierra la sesión de Clerk (TengoLibre usa la propia) y
 *    lo devuelve para mandarlo al servidor, que lo canjea en Go (`POST /v1/auth/clerk`). Go lo verifica:
 *    nada de lo que diga el navegador cuenta sin esa verificación.
 */

export type ProveedorClerk = 'google' | 'apple';

/** Versión de la Frontend API con la que se probó este flujo. */
export const VERSION_FAPI = '2025-11-10';
const CLAVE = 'tl-clerk';

/** La Frontend API sale de la clave publicable: pk_<entorno>_<base64("host$")>. */
export function fapiDe(publicable: string | undefined | null): string | null {
	const m = /^pk_(live|test)_([A-Za-z0-9+/=_-]+)$/.exec(publicable ?? '');
	if (!m) return null;
	try {
		const host = atob(m[2].replace(/-/g, '+').replace(/_/g, '/')).replace(/\$$/, '');
		return /^[a-z0-9.-]+\.[a-z]{2,}$/i.test(host) ? `https://${host}` : null;
	} catch {
		return null;
	}
}

/** Un error del ida y vuelta, con un código para el mensaje de /ingresar. */
export class ErrorClerk extends Error {
	constructor(public codigo: string) {
		super(codigo);
	}
}

type Respuesta = {
	response?: {
		id?: string;
		status?: string;
		created_session_id?: string | null;
		first_factor_verification?: {
			status?: string;
			external_verification_redirect_url?: string | null;
		} | null;
	};
	jwt?: string;
	errors?: { code?: string }[];
};

type Pedido = { metodo?: 'GET' | 'POST'; cuerpo?: Record<string, string>; cliente?: string };

async function pedir(fapi: string, ruta: string, p: Pedido, f: typeof fetch) {
	const u = new URL(`${fapi}/v1${ruta}`);
	u.searchParams.set('_is_native', '1');
	u.searchParams.set('__clerk_api_version', VERSION_FAPI);
	const headers: Record<string, string> = {};
	if (p.cliente) headers.authorization = p.cliente;
	if (p.cuerpo) headers['content-type'] = 'application/x-www-form-urlencoded';
	const res = await f(u, {
		method: p.metodo ?? 'GET',
		credentials: 'omit',
		headers,
		body: p.cuerpo ? new URLSearchParams(p.cuerpo) : undefined
	});
	const json = (await res.json().catch(() => ({}))) as Respuesta;
	if (!res.ok) throw new ErrorClerk(json.errors?.[0]?.code ?? 'clerk_no_responde');
	// Clerk rota el token del cliente: vale el último que mandó.
	return { json, cliente: res.headers.get('authorization') || p.cliente || '' };
}

type Estado = { fapi: string; cliente: string; intento: string };

/** Arranca el ingreso y devuelve la URL del proveedor (Google o Apple) a la que hay que ir. */
export async function iniciar(
	fapi: string,
	proveedor: ProveedorClerk,
	vuelta: string,
	guardar: Storage = sessionStorage,
	f: typeof fetch = fetch
): Promise<string> {
	const { json, cliente } = await pedir(
		fapi,
		'/client/sign_ins',
		{ metodo: 'POST', cuerpo: { strategy: `oauth_${proveedor}`, redirect_url: vuelta } },
		f
	);
	const url = json.response?.first_factor_verification?.external_verification_redirect_url;
	if (!url || !cliente || !json.response?.id) throw new ErrorClerk('clerk_no_responde');
	const e: Estado = { fapi, cliente, intento: json.response.id };
	guardar.setItem(CLAVE, JSON.stringify(e));
	return url;
}

/** Termina el ingreso con el nonce de la vuelta y devuelve el token de sesión de Clerk. */
export async function completar(
	nonce: string,
	guardar: Storage = sessionStorage,
	f: typeof fetch = fetch
): Promise<string> {
	let e: Estado | null;
	try {
		e = JSON.parse(guardar.getItem(CLAVE) ?? 'null') as Estado | null;
	} catch {
		e = null;
	}
	guardar.removeItem(CLAVE);
	if (!e?.fapi || !e.cliente || !/^sia_[A-Za-z0-9]+$/.test(e.intento ?? ''))
		throw new ErrorClerk('estado_invalido');
	// Con cuenta en Clerk la vuelta trae el nonce. Alguien nuevo vuelve sin él (el ingreso tiene que
	// pasar a alta): se consulta igual con el token del cliente que quedó guardado.
	const ruta = nonce
		? `/client/sign_ins/${e.intento}?rotating_token_nonce=${encodeURIComponent(nonce)}`
		: `/client/sign_ins/${e.intento}`;
	let ingreso: Awaited<ReturnType<typeof pedir>>;
	try {
		ingreso = await pedir(e.fapi, ruta, { cliente: e.cliente }, f);
	} catch (x) {
		// "No existe esa cuenta" no es un error: es alguien nuevo, va al alta.
		if (!(x instanceof ErrorClerk) || x.codigo !== 'external_account_not_found') throw x;
		ingreso = {
			json: { response: { first_factor_verification: { status: 'transferable' } } },
			cliente: e.cliente
		} as Awaited<ReturnType<typeof pedir>>;
	}
	let cliente = ingreso.cliente;
	let sesion = ingreso.json.response?.created_session_id ?? null;
	const verif = ingreso.json.response?.first_factor_verification;
	// Sin cuenta en Clerk: el ingreso se transfiere a un alta con los datos del proveedor.
	if (!sesion && verif?.status === 'transferable') {
		const alta = await pedir(
			e.fapi,
			'/client/sign_ups',
			{ metodo: 'POST', cuerpo: { transfer: 'true' }, cliente },
			f
		);
		cliente = alta.cliente;
		sesion = alta.json.response?.created_session_id ?? null;
	}
	if (!sesion || !/^sess_[A-Za-z0-9]+$/.test(sesion)) throw new ErrorClerk('cancelado');
	const t = await pedir(
		e.fapi,
		`/client/sessions/${sesion}/tokens`,
		{ metodo: 'POST', cliente },
		f
	);
	if (!t.json.jwt) throw new ErrorClerk('clerk_no_responde');
	// TengoLibre sigue con su propia sesión: la de Clerk se cierra (Go verifica la firma del token,
	// no necesita que siga abierta). Si falla, vence sola.
	await pedir(
		e.fapi,
		`/client/sessions/${sesion}/remove`,
		{ metodo: 'POST', cliente: t.cliente },
		f
	).catch(() => undefined);
	return t.json.jwt;
}
