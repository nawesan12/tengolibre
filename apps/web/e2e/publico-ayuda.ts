import { execFileSync } from 'node:child_process';
import { unico } from './ayuda.ts';

/**
 * Lo que arma la parte pública por la API (más rápido que el panel, que ya tiene sus pruebas) y lo
 * que en producción llega por mail (tokens de reseña, preferencias, invitación), leído de la base
 * local con psql.
 */
const API = `http://127.0.0.1:${process.env.E2E_API_PUERTO ?? '4290'}`;
const PG = process.env.E2E_PG ?? 'postgres://tengolibre@127.0.0.1:5446/tengolibre_dev';

export const idPersona = () => `user_e2e${unico()}`;

export async function token(persona: string) {
	const r = await pedir<{ token: string }>('POST', '/v1/auth/clerk', { token: `dev:${persona}` });
	return r.token;
}

async function pedir<T>(metodo: string, ruta: string, cuerpo?: unknown, tk?: string, negocio?: string): Promise<T> {
	const headers: Record<string, string> = { 'content-type': 'application/json', accept: 'application/json' };
	if (tk) headers.authorization = `Bearer ${tk}`;
	if (negocio) headers['x-negocio'] = negocio;
	const res = await fetch(API + ruta, { method: metodo, headers, body: cuerpo === undefined ? undefined : JSON.stringify(cuerpo) });
	if (!res.ok) throw new Error(`${metodo} ${ruta}: ${res.status} ${await res.text()}`);
	return (res.status === 204 ? undefined : await res.json()) as T;
}

/** SQL contra la base local (el superusuario de desarrollo); devuelve la primera columna de la primera fila. */
export function sql(q: string, ...args: string[]) {
	const vars = args.flatMap((a, i) => ['-v', `p${i + 1}=${a}`]);
	return execFileSync('psql', [PG, '-qtAX', '-v', 'ON_ERROR_STOP=1', ...vars], { input: q, encoding: 'utf8' }).trim().split('\n')[0] ?? '';
}

export type Armado = { slug: string; nombre: string; negocio: string; servicio: string; tk: string; duenia: string };

/** Un negocio con un servicio y dos profesionales que atienden todos los días de 8 a 20. */
export async function negocioConAgenda(opciones: { ciudad?: string } = {}): Promise<Armado> {
	const u = unico();
	const duenia = `user_e2e${u}`;
	const tk = await token(duenia);
	const slug = `peluqueria-${u}`;
	const nombre = `Peluquería ${u}`;
	const n = await pedir<{ id: string }>('POST', '/v1/negocios', { slug, nombre }, tk);
	const p = <T>(metodo: string, ruta: string, cuerpo?: unknown) => pedir<T>(metodo, ruta, cuerpo, tk, n.id);
	await p('PATCH', '/v1/panel/negocio', { ciudad: opciones.ciudad ?? 'Rosario', direccion: 'Córdoba 1234', descripcion: 'Cortes y color.' });
	const s = await p<{ id: string }>('POST', '/v1/panel/servicios', { nombre: 'Corte y lavado', duracion: 30, precio: 9000 });
	const horario = [0, 1, 2, 3, 4, 5, 6].map((dia) => ({ dia, desde: '08:00', hasta: '20:00' }));
	await p('POST', '/v1/panel/profesionales', { nombre: 'Ana', servicios: [s.id], horario, orden: 1 });
	await p('POST', '/v1/panel/profesionales', { nombre: 'Beto', servicios: [s.id], horario, orden: 2 });
	return { slug, nombre, negocio: n.id, servicio: s.id, tk, duenia };
}

export const invitar = (a: Armado, email: string) =>
	pedir('POST', '/v1/panel/invitaciones', { email, rol: 'recepcion' }, a.tk, a.negocio);

/** Hoy en Argentina, AAAA-MM-DD. */
export const hoy = () =>
	new Intl.DateTimeFormat('en-CA', { timeZone: 'America/Argentina/Buenos_Aires', year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date());

/** Un plan de una visita para el servicio y un socio con ese email (cargado por el negocio). */
export async function socio(a: Armado, email: string) {
	const p = <T>(metodo: string, ruta: string, cuerpo?: unknown) => pedir<T>(metodo, ruta, cuerpo, a.tk, a.negocio);
	const plan = await p<{ id: string }>('POST', '/v1/panel/membresias/planes', { nombre: 'Un corte', precio: 9000, visitas: 1, duracionDias: 30 });
	await p('POST', '/v1/panel/membresias/socios', { planId: plan.id, nombre: 'Socia', telefono: '11 5555 0000', email });
}

/** Un pedido a la API pública (sin sesión), como lo hace el navegador de un cliente. */
export const publica = <T>(metodo: string, ruta: string, cuerpo?: unknown) => pedir<T>(metodo, ruta, cuerpo);

/** Un turno reservado por la página pública con Ana, el primer horario de un día después de hoy. */
export async function reservarConAna(a: Armado, nombre = 'Cliente E2E') {
	const ana = sql(`SELECT id FROM profesionales WHERE negocio_id = :'p1' AND nombre = 'Ana'`, a.negocio);
	const q = new URLSearchParams({ servicio: a.servicio, profesional: ana });
	const { dias } = await publica<{ dias: { fecha: string; libres: number }[] }>('GET', `/v1/p/${a.slug}/dias?${q}`);
	const manana = new Date(Date.parse(hoy() + 'T12:00:00Z') + 86_400_000).toISOString().slice(0, 10);
	const fecha = dias.find((d) => d.libres > 0 && d.fecha > manana)!.fecha;
	q.set('fecha', fecha);
	const { horarios } = await publica<{ horarios: { inicio: string }[] }>('GET', `/v1/p/${a.slug}/libres?${q}`);
	const t = await publica<{ codigo: string; inicio: string }>('POST', `/v1/p/${a.slug}/turnos`, {
		servicioId: a.servicio,
		profesionalId: ana,
		inicio: horarios[0].inicio,
		cliente: { nombre, telefono: '11 5555 0202', email: `cli${unico()}@prueba.com` }
	});
	return { ...t, fecha, ana };
}
