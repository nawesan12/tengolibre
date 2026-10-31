/** Formatos de Argentina para todo el front. Las horas, en la zona del negocio. */
export const ZONA = 'America/Argentina/Buenos_Aires';

const pesosF = new Intl.NumberFormat('es-AR', { style: 'currency', currency: 'ARS', maximumFractionDigits: 0 });
export const pesos = (n: number | null | undefined) => pesosF.format(n ?? 0);

export const fecha = (d: string | Date, zona = ZONA) =>
	new Intl.DateTimeFormat('es-AR', { weekday: 'short', day: 'numeric', month: 'short', timeZone: zona }).format(new Date(d));

export const hora = (d: string | Date, zona = ZONA) =>
	new Intl.DateTimeFormat('es-AR', { hour: '2-digit', minute: '2-digit', hour12: false, timeZone: zona }).format(new Date(d));

export const fechaHora = (d: string | Date, zona = ZONA) => `${fecha(d, zona)} ${hora(d, zona)}`;

/** AAAA-MM-DD de un instante, en la zona del negocio. */
export const dia = (d: string | Date = new Date(), zona = ZONA) =>
	new Intl.DateTimeFormat('en-CA', { year: 'numeric', month: '2-digit', day: '2-digit', timeZone: zona }).format(new Date(d));

export const ESTADOS: Record<string, string> = {
	pendiente_pago: 'Esperando pago',
	pendiente: 'Pendiente',
	confirmado: 'Confirmado',
	atendido: 'Atendido',
	cancelado: 'Cancelado',
	ausente: 'No vino'
};

export const ROLES: Record<string, string> = {
	duenio: 'Dueño/a',
	encargado: 'Encargado/a',
	recepcion: 'Recepción',
	profesional: 'Profesional'
};

export const DIAS = ['Domingo', 'Lunes', 'Martes', 'Miércoles', 'Jueves', 'Viernes', 'Sábado'];
