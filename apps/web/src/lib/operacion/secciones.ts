/** Las pantallas hermanas que comparten entrada en la navegación. */
export const CLIENTES = [
	{ nombre: 'Clientes', href: '/panel/clientes' },
	{ nombre: 'Etiquetas', href: '/panel/etiquetas' },
	{ nombre: 'Penalidades', href: '/panel/penalidades' }
];

export const NOTIFICACIONES = [
	{ nombre: 'Novedades', href: '/panel/notificaciones' },
	{ nombre: 'Mails enviados', href: '/panel/notificaciones/envios' }
];

/** Quienes atienden a los clientes (editarlos, etiquetarlos); el profesional sólo los ve. */
export const ATENCION = ['duenio', 'encargado', 'recepcion'];
export const GESTION = ['duenio', 'encargado'];
