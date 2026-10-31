/**
 * La navegación del panel (handoff v2, «Panel v2»): arriba lo del día a día y, plegado, «Tu negocio y
 * ajustes». El menú se recorta según el rol; las pantallas que se visitan juntas comparten entrada.
 */
export type Entrada = {
	clave: string;
	nombre: string;
	href: string;
	icono: string;
	cubre?: string[];
	roles?: string[];
	/** Nombre para quien es profesional (ve sólo lo suyo). */
	paraProfesional?: string;
};

const G = ['duenio', 'encargado'];
const MOSTRADOR = [...G, 'recepcion'];

export const PRINCIPAL: Entrada[] = [
	{ clave: 'inicio', nombre: 'Inicio', href: '/panel', icono: 'home', cubre: ['/panel/buscar', '/panel/notificaciones'] },
	{ clave: 'turnos', nombre: 'Turnos', href: '/panel/turnos', icono: 'event_note' },
	{ clave: 'calendario', nombre: 'Calendario', href: '/panel/calendario', icono: 'calendar_month' },
	{ clave: 'espera', nombre: 'Lista de espera', href: '/panel/lista-espera', icono: 'hourglass_top', roles: MOSTRADOR },
	{ clave: 'clientes', nombre: 'Clientes', href: '/panel/clientes', icono: 'group', cubre: ['/panel/etiquetas', '/panel/penalidades'], roles: MOSTRADOR }
];

export const SECUNDARIO: Entrada[] = [
	{ clave: 'web', nombre: 'Mi web', href: '/panel/mi-web', icono: 'language', cubre: ['/panel/dominio'], roles: G },
	{ clave: 'plantillas', nombre: 'Plantillas', href: '/panel/plantillas', icono: 'dashboard_customize', roles: G },
	{ clave: 'integ', nombre: 'Integraciones e IA', href: '/panel/integraciones', icono: 'hub', roles: G },
	{ clave: 'srv', nombre: 'Servicios', href: '/panel/servicios', icono: 'spa', roles: G },
	{
		clave: 'equipo',
		nombre: 'Equipo y horarios',
		paraProfesional: 'Mi horario',
		href: '/panel/equipo',
		icono: 'badge',
		cubre: ['/panel/profesionales', '/panel/invitaciones']
	},
	{ clave: 'pagos', nombre: 'Pagos', href: '/panel/pagos', icono: 'payments', cubre: ['/panel/mercadopago'], roles: G },
	{ clave: 'memb', nombre: 'Membresías y cupones', href: '/panel/membresias', icono: 'card_membership', cubre: ['/panel/cupones'], roles: MOSTRADOR },
	{ clave: 'resenas', nombre: 'Reseñas', href: '/panel/resenas', icono: 'star' },
	{ clave: 'ref', nombre: 'Referidos', href: '/panel/referidos', icono: 'redeem', roles: G },
	{ clave: 'rep', nombre: 'Reportes', href: '/panel/reportes', icono: 'bar_chart', roles: G },
	{ clave: 'conf', nombre: 'Configuración', href: '/panel/configuracion', icono: 'settings', cubre: ['/panel/avisos', '/panel/auditoria'], roles: G },
	{ clave: 'sus', nombre: 'Suscripción', href: '/panel/suscripcion', icono: 'workspace_premium', roles: ['duenio'] },
	{ clave: 'sucursales', nombre: 'Sucursales', href: '/panel/sucursales', icono: 'store', roles: ['duenio'] }
];

/** Las pestañas de abajo en el teléfono (la última abre «Más»). */
export const PESTANIAS: Entrada[] = [
	{ clave: 'inicio', nombre: 'Inicio', href: '/panel', icono: 'home' },
	{ clave: 'turnos', nombre: 'Turnos', href: '/panel/turnos', icono: 'event_note' },
	{ clave: 'calendario', nombre: 'Agenda', href: '/panel/calendario', icono: 'calendar_month' },
	{ clave: 'clientes', nombre: 'Clientes', href: '/panel/clientes', icono: 'group', cubre: ['/panel/etiquetas', '/panel/penalidades'], roles: MOSTRADOR }
];
export const PESTANIAS_PROFESIONAL: Entrada[] = [
	{ clave: 'inicio', nombre: 'Hoy', href: '/panel', icono: 'today' },
	{ clave: 'turnos', nombre: 'Turnos', href: '/panel/turnos', icono: 'event_note' },
	{ clave: 'calendario', nombre: 'Semana', href: '/panel/calendario?vista=semana', icono: 'calendar_month' }
];

/** La entrada activa para una ruta: la propia o la que la cubre. */
export function activa(ruta: string, e: Entrada) {
	const href = e.href.split('?')[0];
	if (href === '/panel') return ruta === '/panel' || (e.cubre ?? []).some((h) => ruta === h || ruta.startsWith(h + '/'));
	return [href, ...(e.cubre ?? [])].some((h) => ruta === h || ruta.startsWith(h + '/'));
}

export const visible = (e: Entrada, rol: string) => !e.roles || e.roles.includes(rol);

/** El nombre de la entrada para ese rol. */
export const nombreDe = (e: Entrada, rol: string) => (rol === 'profesional' && e.paraProfesional) || e.nombre;
