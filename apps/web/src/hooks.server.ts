import type { Handle, HandleFetch } from '@sveltejs/kit/hooks';
import { COOKIE_NEGOCIO, COOKIE_SESION, elegirNegocio } from '#lib/server/sesion.ts';
import { apiDe, BASE_API, type NegocioDeLaPersona } from '#lib/server/api.ts';
import { cacheDeHosts, esHostPropio, redireccionFueraDelPlan, RUTA_HOST, type Destino } from '#lib/publico/dominio.ts';

// Dominio propio fuera del plan (bajó de Centro): cualquier ruta de ese host lleva a la página del
// negocio en TengoLibre, en vez de mostrar la portada de TengoLibre en su dominio.
const hosts = cacheDeHosts();

async function destinoDe(event: Parameters<Handle>[0]['event']): Promise<Destino | null> {
	const host = event.url.hostname.toLowerCase();
	if (esHostPropio(host) || event.url.pathname === RUTA_HOST || event.url.pathname.startsWith('/_app/')) return null;
	const enCache = hosts.leer(host);
	if (enCache !== undefined) return enCache;
	const r = await event.fetch(`${RUTA_HOST}?host=${encodeURIComponent(host)}`).catch(() => null);
	if (!r || r.status >= 500) return null;
	const d: Destino = r.ok ? ((await r.json()) as Destino) : { slug: null };
	hosts.guardar(host, d);
	return d;
}

export const handle: Handle = async ({ event, resolve }) => {
	const destino = await destinoDe(event);
	const ir = destino ? redireccionFueraDelPlan(event.url, destino) : null;
	if (ir) return new Response(null, { status: 302, headers: { location: ir, 'cache-control': 'public, max-age=60' } });
	event.locals.token = event.cookies.get(COOKIE_SESION) ?? null;
	event.locals.negocio = event.cookies.get(COOKIE_NEGOCIO) ?? null;
	// Sin negocio elegido (recién ingresó o se borró la cookie), el primero, como el layout del panel: si
	// no, las cargas de la página corren en paralelo con el layout y piden sin X-Negocio (un 500).
	const ruta0 = event.url.pathname;
	if (event.locals.token && !event.locals.negocio && (ruta0 === '/panel' || ruta0.startsWith('/panel/'))) {
		const yo = await apiDe(event)
			.get<{ negocios: NegocioDeLaPersona[] }>('/v1/yo')
			.catch(() => null);
		const primero = yo?.negocios[0];
		if (primero) {
			elegirNegocio(event.cookies, primero.id);
			event.locals.negocio = primero.id;
		}
	}
	// Las acciones del panel guardan en el negocio que muestra la página (el campo __negocio que suma
	// el layout), no en el de la cookie: si se cambió de negocio en otra pestaña, la vieja no escribe en
	// el nuevo. La API igual verifica que la persona sea del negocio.
	if (event.request.method === 'POST' && (ruta0 === '/panel' || ruta0.startsWith('/panel/'))) {
		const tipo = event.request.headers.get('content-type') ?? '';
		if (tipo.startsWith('application/x-www-form-urlencoded') || tipo.startsWith('multipart/form-data')) {
			const pagina = await event.request
				.clone()
				.formData()
				.then((f) => String(f.get('__negocio') ?? ''))
				.catch(() => '');
			if (/^[0-9a-f-]{36}$/.test(pagina)) event.locals.negocio = pagina;
		}
	}
	const res = await resolve(event);
	res.headers.set('x-content-type-options', 'nosniff');
	res.headers.set('referrer-policy', 'strict-origin-when-cross-origin');
	const ruta = event.url.pathname;
	const en = (base: string) => ruta === base || ruta.startsWith(base + '/');
	if (en('/panel') || en('/admin')) res.headers.set('x-frame-options', 'DENY');
	// Nadie enmarca las páginas en un sitio ajeno (clickjacking en la reserva, Mi turno, Mi cuenta o el
	// ingreso). La vista previa de Plantillas es del mismo origen. La CSP completa necesita relevar
	// antes Clerk, Mercado Pago y los demás externos: por ahora, sólo esto.
	if (!res.headers.has('content-security-policy')) res.headers.set('content-security-policy', "frame-ancestors 'self'");
	// Una redirección nunca queda guardada para siempre: los links de un negocio cambian (y vuelven).
	if (res.status >= 300 && res.status < 400 && !res.headers.has('cache-control'))
		res.headers.set('cache-control', 'public, max-age=600');
	// Lo de la sesión nunca queda en una caché compartida: el panel, la cuenta, el alta, el ingreso y
	// cualquier respuesta a alguien con sesión (la reserva con su membresía, por ejemplo).
	if (en('/panel') || en('/admin') || en('/mi-cuenta') || en('/empezar') || en('/ingresar') || event.locals.token)
		res.headers.set('cache-control', 'private, no-store');
	return res;
};

/**
 * Los pedidos a la API salen de este servidor, así que para la API todos venían de la misma IP y los
 * topes «por IP» (reservar, lista de espera, cupones) eran uno solo para todo el sitio. Se reenvía la
 * IP del visitante que puso Cloudflare (Caddy sólo acepta a Cloudflare, que pisa ese encabezado si
 * lo manda el navegador).
 */
export const handleFetch: HandleFetch = async ({ event, request, fetch }) => {
	const ip = event.request.headers.get('cf-connecting-ip');
	if (ip && request.url.startsWith(BASE_API())) {
		request = new Request(request, { headers: new Headers(request.headers) });
		request.headers.set('cf-connecting-ip', ip);
	}
	return fetch(request);
};
