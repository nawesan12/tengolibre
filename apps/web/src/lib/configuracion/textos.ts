/** Textos fijos de la configuración del negocio. */
export const RUBROS: Record<string, string> = {
	barberia: 'Barbería',
	peluqueria: 'Peluquería',
	estetica: 'Estética',
	unias: 'Uñas',
	'cejas-pestanias': 'Cejas y pestañas',
	maquillaje: 'Maquillaje',
	depilacion: 'Depilación',
	masajes: 'Masajes',
	spa: 'Spa',
	tatuajes: 'Tatuajes',
	salud: 'Salud',
	entrenamiento: 'Entrenamiento',
	otro: 'Otro'
};

/** Los roles que se pueden invitar (la dueña no se invita). */
export const ROLES_INVITABLES = ['encargado', 'recepcion', 'profesional'];

/** Cada cuánto arranca un turno (lo que acepta la base). */
export const INTERVALOS = [5, 10, 15, 20, 30, 45, 60];

export const ESTADOS_PAGO: Record<string, string> = {
	pendiente: 'Pendiente',
	aprobado: 'Aprobado',
	rechazado: 'Rechazado',
	devuelto: 'Devuelto',
	cancelado: 'Cancelado',
	en_proceso: 'En proceso'
};

export const ESTADOS_SOCIO: Record<string, string> = {
	activa: 'Activa',
	vencida: 'Vencida',
	cancelada: 'Cancelada'
};

export const ESTADOS_DOMINIO: Record<string, string> = {
	pendiente: 'Falta verificar',
	verificado: 'Conectado',
	error: 'Todavía no anda'
};

export const ESTADOS_SUSCRIPCION: Record<string, string> = {
	pendiente: 'Esperando el alta en Mercado Pago',
	autorizada: 'Activa',
	pausada: 'Pausada',
	cancelada: 'Cancelada'
};
