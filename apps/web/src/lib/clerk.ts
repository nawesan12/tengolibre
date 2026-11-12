/**
 * Ingreso con Google, Apple o mail y contraseña por Clerk, la identidad compartida de Grupo Vimo
 * (el mail y la contraseña, más abajo).
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

type Pedido = { metodo?: 'GET' | 'POST' | 'PATCH'; cuerpo?: Record<string, string>; cliente?: string };

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

// ——— Con mail y contraseña ———
//
// El mismo modo nativo, sin salir de /ingresar (calcado de Corredor). Según lo que tenga la cuenta:
// contraseña, o un código por mail. Alguien nuevo se da de alta con el código (y la contraseña, si la
// instancia la pide). Con contraseña desde un navegador nuevo, Clerk puede pedir confirmar el
// dispositivo (Device Trust: `needs_client_trust`) con un código por mail: es un segundo factor.
// Recuperar: Clerk manda un código (reset_password_email_code); con él se elige la contraseña nueva y
// se cierran las otras sesiones. El final es el mismo que con Google: el token va a /v1/auth/clerk.

type Factor = { strategy?: string; email_address_id?: string };
type RespuestaMail = Respuesta & {
	response?: Respuesta['response'] & {
		supported_first_factors?: Factor[] | null;
		supported_second_factors?: Factor[] | null;
		missing_fields?: string[];
		unverified_fields?: string[];
	};
};

/** El paso que sigue en la pantalla. `confiar`: el código por mail que confirma el dispositivo. */
export type PasoMail =
	| { paso: 'clave' }
	| { paso: 'codigo'; alta: boolean; confiar?: boolean }
	| { paso: 'clave_nueva'; alta: boolean }
	| { paso: 'listo'; token: string };

/** Lo que se guarda entre pasos (en memoria de la página; no hace falta sobrevivir a una recarga). */
export type FlujoMail = {
	fapi: string;
	cliente: string;
	id: string;
	alta: boolean;
	emailId?: string;
	factores: string[];
	/** El código que se espera es el segundo factor (confirmar el dispositivo). */
	segundo?: boolean;
};

async function sesionDe(fapi: string, sesion: string | null | undefined, cliente: string, f: typeof fetch) {
	if (!sesion || !/^sess_[A-Za-z0-9]+$/.test(sesion)) throw new ErrorClerk('cancelado');
	const t = await pedir(fapi, `/client/sessions/${sesion}/tokens`, { metodo: 'POST', cliente }, f);
	if (!t.json.jwt) throw new ErrorClerk('clerk_no_responde');
	await pedir(fapi, `/client/sessions/${sesion}/remove`, { metodo: 'POST', cliente: t.cliente }, f).catch(
		() => undefined
	);
	return t.json.jwt;
}

/**
 * Lo que sigue después de un intento de ingreso: la sesión, o confirmar el dispositivo con un código
 * por mail (needs_client_trust; las instancias viejas lo mandan como needs_second_factor).
 */
async function despuesDelIngreso(flujo: FlujoMail, json: RespuestaMail, f: typeof fetch): Promise<PasoMail> {
	const r = json.response;
	if (r?.status === 'needs_client_trust' || r?.status === 'needs_second_factor') {
		const factor = r.supported_second_factors?.find((x) => x.strategy === 'email_code');
		if (!factor) throw new ErrorClerk('segundo_factor');
		const cuerpo: Record<string, string> = { strategy: 'email_code' };
		if (factor.email_address_id) cuerpo.email_address_id = factor.email_address_id;
		const p = await pedir(flujo.fapi, `/client/sign_ins/${flujo.id}/prepare_second_factor`, {
			metodo: 'POST',
			cuerpo,
			cliente: flujo.cliente
		}, f);
		flujo.cliente = p.cliente;
		flujo.segundo = true;
		return { paso: 'codigo', alta: false, confiar: true };
	}
	return { paso: 'listo', token: await sesionDe(flujo.fapi, r?.created_session_id, flujo.cliente, f) };
}

/** Arranca con el mail: si tiene cuenta, ingreso (clave o código); si no, alta con código. */
export async function empezarConMail(
	fapi: string,
	email: string,
	f: typeof fetch = fetch
): Promise<{ flujo: FlujoMail; sigue: PasoMail }> {
	try {
		const r = await pedir(fapi, '/client/sign_ins', { metodo: 'POST', cuerpo: { identifier: email } }, f);
		const json = r.json as RespuestaMail;
		const id = json.response?.id ?? '';
		const factores = (json.response?.supported_first_factors ?? []).map((x) => x.strategy ?? '');
		const emailId = json.response?.supported_first_factors?.find((x) => x.strategy === 'email_code')?.email_address_id;
		const flujo: FlujoMail = { fapi, cliente: r.cliente, id, alta: false, emailId, factores };
		if (factores.includes('password')) return { flujo, sigue: { paso: 'clave' } };
		if (!emailId) throw new ErrorClerk('sin_mail');
		const p = await pedir(fapi, `/client/sign_ins/${id}/prepare_first_factor`, {
			metodo: 'POST',
			cuerpo: { strategy: 'email_code', email_address_id: emailId },
			cliente: r.cliente
		}, f);
		flujo.cliente = p.cliente;
		return { flujo, sigue: { paso: 'codigo', alta: false } };
	} catch (x) {
		if (!(x instanceof ErrorClerk) || x.codigo !== 'form_identifier_not_found') throw x;
	}
	// Alta: el mail, y verificarlo con un código.
	const a = await pedir(fapi, '/client/sign_ups', { metodo: 'POST', cuerpo: { email_address: email } }, f);
	const id = (a.json as RespuestaMail).response?.id ?? '';
	const p = await pedir(fapi, `/client/sign_ups/${id}/prepare_verification`, {
		metodo: 'POST',
		cuerpo: { strategy: 'email_code' },
		cliente: a.cliente
	}, f);
	return { flujo: { fapi, cliente: p.cliente, id, alta: true, factores: [] }, sigue: { paso: 'codigo', alta: true } };
}

/** Con la contraseña de la cuenta. Desde un dispositivo nuevo puede pedir un código por mail. */
export async function entrarConClave(flujo: FlujoMail, clave: string, f: typeof fetch = fetch): Promise<PasoMail> {
	const r = await pedir(flujo.fapi, `/client/sign_ins/${flujo.id}/attempt_first_factor`, {
		metodo: 'POST',
		cuerpo: { strategy: 'password', password: clave },
		cliente: flujo.cliente
	}, f);
	flujo.cliente = r.cliente;
	return despuesDelIngreso(flujo, r.json as RespuestaMail, f);
}

/** El código que llegó por mail (ingreso, alta, confirmar el dispositivo o recuperar). */
export async function confirmarCodigo(
	flujo: FlujoMail,
	codigo: string,
	recuperar = false,
	f: typeof fetch = fetch
): Promise<PasoMail> {
	const code = codigo.replace(/\D/g, '');
	if (flujo.segundo) {
		const r = await pedir(flujo.fapi, `/client/sign_ins/${flujo.id}/attempt_second_factor`, {
			metodo: 'POST',
			cuerpo: { strategy: 'email_code', code },
			cliente: flujo.cliente
		}, f);
		flujo.cliente = r.cliente;
		flujo.segundo = false;
		return { paso: 'listo', token: await sesionDe(flujo.fapi, r.json.response?.created_session_id, r.cliente, f) };
	}
	const ruta = flujo.alta
		? `/client/sign_ups/${flujo.id}/attempt_verification`
		: `/client/sign_ins/${flujo.id}/attempt_first_factor`;
	const r = await pedir(flujo.fapi, ruta, {
		metodo: 'POST',
		cuerpo: { strategy: recuperar ? 'reset_password_email_code' : 'email_code', code },
		cliente: flujo.cliente
	}, f);
	flujo.cliente = r.cliente;
	const resp = (r.json as RespuestaMail).response;
	if (resp?.status === 'needs_new_password') return { paso: 'clave_nueva', alta: false };
	// La instancia pide contraseña para el alta.
	if (flujo.alta && resp?.status === 'missing_requirements' && resp.missing_fields?.includes('password'))
		return { paso: 'clave_nueva', alta: true };
	if (flujo.alta) return { paso: 'listo', token: await sesionDe(flujo.fapi, resp?.created_session_id, r.cliente, f) };
	return despuesDelIngreso(flujo, r.json as RespuestaMail, f);
}

/** La contraseña nueva: al recuperar (cierra las otras sesiones de Clerk) o la que pide el alta. */
export async function guardarClave(flujo: FlujoMail, clave: string, f: typeof fetch = fetch): Promise<PasoMail> {
	const r = flujo.alta
		? await pedir(flujo.fapi, `/client/sign_ups/${flujo.id}`, {
				metodo: 'PATCH',
				cuerpo: { password: clave },
				cliente: flujo.cliente
			}, f)
		: await pedir(flujo.fapi, `/client/sign_ins/${flujo.id}/reset_password`, {
				metodo: 'POST',
				cuerpo: { password: clave, sign_out_of_other_sessions: 'true' },
				cliente: flujo.cliente
			}, f);
	flujo.cliente = r.cliente;
	if (flujo.alta) return { paso: 'listo', token: await sesionDe(flujo.fapi, r.json.response?.created_session_id, r.cliente, f) };
	return despuesDelIngreso(flujo, r.json as RespuestaMail, f);
}

/** Recuperar: arranca el ingreso con el mail y pide el código para la contraseña nueva. */
export async function pedirRecuperar(fapi: string, email: string, f: typeof fetch = fetch): Promise<FlujoMail | null> {
	let r: Awaited<ReturnType<typeof pedir>>;
	try {
		r = await pedir(fapi, '/client/sign_ins', { metodo: 'POST', cuerpo: { identifier: email } }, f);
	} catch (x) {
		// Sin cuenta: la pantalla dice lo mismo que con cuenta (no se revela quién está registrado).
		if (x instanceof ErrorClerk && x.codigo === 'form_identifier_not_found') return null;
		throw x;
	}
	const json = r.json as RespuestaMail;
	const factor = json.response?.supported_first_factors?.find((x) => x.strategy === 'reset_password_email_code');
	const id = json.response?.id ?? '';
	if (!factor?.email_address_id) return null;
	const p = await pedir(fapi, `/client/sign_ins/${id}/prepare_first_factor`, {
		metodo: 'POST',
		cuerpo: { strategy: 'reset_password_email_code', email_address_id: factor.email_address_id },
		cliente: r.cliente
	}, f);
	return { fapi, cliente: p.cliente, id, alta: false, factores: [] };
}

/** Los requisitos de la contraseña nueva, que se tildan solos mientras se escribe. */
const COMUNES = new Set([
	'1234567890', '12345678910', 'qwertyuiop', 'contraseña', 'contrasena', 'password12', 'password123', 'argentina1',
	'0123456789', '1111111111', 'abcdefghij', 'tengolibre', 'tengolibre1', 'peluqueria', 'barberia1', 'iloveyou12',
	'boca123456', 'river12345'
]);
export function requisitosClave(clave: string) {
	return { largo: clave.length >= 8, comun: clave.length > 0 && !esComun(clave) };
}
function esComun(c: string) {
	const x = c.toLowerCase();
	return COMUNES.has(x) || /^(.)\1+$/.test(x) || /^(0123456789|1234567890|9876543210)/.test(x);
}

/** El error de Clerk como lo lee la persona. */
export function mensajeClerk(codigo: string): string {
	const m: Record<string, string> = {
		form_password_incorrect: 'La contraseña no es esa. Probá de nuevo o recuperala.',
		form_code_incorrect: 'El código no es ese. Revisalo y probá de nuevo.',
		verification_expired: 'El código venció. Pedí uno nuevo.',
		form_password_pwned: 'Esa contraseña apareció en filtraciones. Elegí otra.',
		form_password_length_too_short: 'La contraseña es muy corta.',
		form_password_validation_failed: 'La contraseña no cumple los requisitos.',
		form_param_format_invalid: 'Revisá el mail: no parece válido.',
		form_identifier_exists: 'Ese mail ya tiene cuenta. Ingresá.',
		too_many_requests: 'Hubo demasiados intentos. Probá de nuevo en unos minutos.',
		strategy_for_user_invalid: 'Esa cuenta entra con Google o Apple: usá ese botón.',
		sin_mail: 'Esa cuenta entra con Google o Apple: usá ese botón.',
		segundo_factor: 'Tu cuenta pide un segundo paso que todavía no está en la web. Entrá con Google o Apple.'
	};
	return m[codigo] ?? 'No pudimos completar el ingreso. Probá de nuevo.';
}
