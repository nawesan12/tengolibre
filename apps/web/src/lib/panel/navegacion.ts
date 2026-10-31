/**
 * La navegación del panel, calcada de Jiku (components/layout/navigation.ts): una lista plana
 * agrupada por rayitas, con las pantallas que se visitan juntas plegadas en una sola entrada.
 */
export type Entrada = { nombre: string; href: string; cubre?: string[]; roles?: string[] };

const GESTION = ['duenio', 'encargado'];

export const GRUPOS: Entrada[][] = [
	[
		{ nombre: 'Inicio', href: '/panel' },
		{ nombre: 'Turnos', href: '/panel/turnos' },
		{ nombre: 'Calendario', href: '/panel/calendario' },
		{ nombre: 'Lista de espera', href: '/panel/lista-espera' },
		{ nombre: 'Clientes', href: '/panel/clientes', cubre: ['/panel/etiquetas', '/panel/penalidades'] }
	],
	[
		{ nombre: 'Mi web', href: '/panel/mi-web', cubre: ['/panel/dominio'], roles: GESTION },
		{ nombre: 'Plantillas', href: '/panel/plantillas', roles: GESTION },
		{ nombre: 'Integraciones e IA', href: '/panel/integraciones', roles: GESTION },
		{ nombre: 'Servicios', href: '/panel/servicios', roles: GESTION },
		{ nombre: 'Equipo y horarios', href: '/panel/equipo', cubre: ['/panel/profesionales', '/panel/invitaciones'] }
	],
	[
		{ nombre: 'Pagos', href: '/panel/pagos', cubre: ['/panel/mercadopago'], roles: GESTION },
		{ nombre: 'Membresías y cupones', href: '/panel/membresias', cubre: ['/panel/cupones'] }
	],
	[
		{ nombre: 'Reseñas', href: '/panel/resenas' },
		{ nombre: 'Referidos', href: '/panel/referidos', roles: GESTION },
		{ nombre: 'Reportes', href: '/panel/reportes', roles: GESTION }
	],
	[
		{
			nombre: 'Configuración',
			href: '/panel/configuracion',
			cubre: ['/panel/sucursales', '/panel/avisos', '/panel/auditoria'],
			roles: GESTION
		},
		{ nombre: 'Suscripción', href: '/panel/suscripcion', roles: GESTION }
	]
];

/** La entrada activa para una ruta: la propia o la que la cubre. */
export function activa(ruta: string, e: Entrada) {
	if (e.href === '/panel') return ruta === '/panel';
	return [e.href, ...(e.cubre ?? [])].some((h) => ruta === h || ruta.startsWith(h + '/'));
}

export const visible = (e: Entrada, rol: string) => !e.roles || e.roles.includes(rol);
