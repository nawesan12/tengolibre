/**
 * La dirección web del negocio (tengolibre.com.ar/<slug>): cómo se arma desde el nombre, qué formato
 * acepta la API (negocios.ValidarSlug) y qué decirle a la persona mientras escribe.
 */

/** Las rutas del sitio que no pueden ser la dirección de un negocio (negocios.Reservados en la API). */
export const RESERVADOS = new Set([
	'admin', 'api', 'explorar', 'iniciar-sesion', 'invitacion', 'mi-cuenta', 'mis-turnos', 'panel', 'presentacion',
	'recuperar-contrasena', 'registrarse', 'review', 'resena', 'sin-conexion', 'turno', 'ingresar', 'precios', 'ayuda',
	'legal', 'terminos', 'privacidad', 'www', 'cal', 'empezar', 'og', 'salir'
]);

/** Del nombre (o lo que se escriba) a una dirección válida: minúsculas, sin acentos, guiones. */
export const aSlug = (s: string) =>
	s
		.normalize('NFD')
		.replace(/[̀-ͯ]/g, '')
		.toLowerCase()
		.replace(/[^a-z0-9]+/g, '-')
		.replace(/^-+|-+$/g, '')
		.slice(0, 48)
		.replace(/-+$/, '');

const reSlug = /^[a-z0-9]([a-z0-9-]{0,46}[a-z0-9])?$/;

export const formatoValido = (s: string) => reSlug.test(s) && !s.includes('--');

/** Otras direcciones para ofrecer cuando la que se pidió está tomada. */
export function alternativas(base: string, rubro?: string): string[] {
	const out = [`${base}-${rubro && rubro !== 'otro' ? rubro : 'estudio'}`, `${base}-turnos`, `soy${base}`];
	return [...new Set(out.map(aSlug))].filter((x) => formatoValido(x) && x !== base);
}

/** Lo que devuelve el chequeo de /empezar/direccion. */
export type Disponibilidad = { estado: 'libre' | 'tomada' | 'invalida'; sugerencias: string[] };

/** Cómo se ve el campo: ícono, color del borde, color y texto del mensaje. Nunca sólo color. */
export type EstadoCampo = {
	estado: 'vacio' | 'corta' | 'buscando' | 'libre' | 'tomada' | 'invalida';
	icono: string;
	borde: string;
	color: string;
	mensaje: string;
	ok: boolean;
};

export function estadoCampo(slug: string, r: Disponibilidad | null, vacio = 'Escribí el nombre de tu negocio'): EstadoCampo {
	if (!slug) return { estado: 'vacio', icono: 'edit', borde: '#ddd1b9', color: '#4a5d61', mensaje: vacio, ok: false };
	if (slug.length < 3)
		return { estado: 'corta', icono: 'info', borde: '#b26a00', color: '#8a5200', mensaje: 'Tiene que tener al menos 3 letras', ok: false };
	if (!r) return { estado: 'buscando', icono: 'progress_activity', borde: '#ddd1b9', color: '#4a5d61', mensaje: 'Fijándonos si está libre…', ok: false };
	if (r.estado === 'tomada')
		return {
			estado: 'tomada',
			icono: 'cancel',
			borde: '#a3241a',
			color: '#a3241a',
			mensaje: r.sugerencias.length ? 'Ya la usa otro negocio. Probá una de estas:' : 'Ya la usa otro negocio.',
			ok: false
		};
	if (r.estado === 'invalida')
		return { estado: 'invalida', icono: 'cancel', borde: '#a3241a', color: '#a3241a', mensaje: 'Esa dirección no se puede usar.', ok: false };
	return { estado: 'libre', icono: 'check_circle', borde: '#17805a', color: '#17805a', mensaje: 'Está libre. Es tuya.', ok: true };
}
