/*
 * Lo que el diseño v2 de Cuenta y accesos muestra y la API todavía no manda (o manda a medias).
 * Todo lo nuevo es opcional: mientras no venga, la pantalla lo omite (nada de datos inventados) y no
 * se envía (la API rechaza campos desconocidos). Cada campo documenta el contrato propuesto.
 */

/** GET /v1/admin/estadisticas: además de lo que ya manda (Estadisticas), el pulso de hoy y la plata. */
export type EstadisticasExtra = {
	turnosHoy?: number;
	/** 0–100: qué parte de los turnos de hoy entró por la web pública. */
	porcentajeWeb?: number;
	suscripcionesPagas?: number;
	/** Ingreso mensual recurrente de las suscripciones, en pesos. */
	mrr?: number;
	/** Negocios con la prueba que vence en los próximos 3 días. */
	pruebasPorVencer?: number;
	/** 0–100: pruebas que terminaron en una suscripción (últimos 90 días). */
	conversion?: number;
};

/** Estado de la cuenta de un negocio, para el admin. */
export type EstadoNegocio = 'prueba' | 'inicial' | 'completo' | 'cobro_fallido' | 'solo_lectura';
export const ESTADOS_NEGOCIO: EstadoNegocio[] = ['prueba', 'cobro_fallido', 'solo_lectura'];

/**
 * GET /v1/admin/negocios: cada fila con rubro y estado. Acepta `estado=<EstadoNegocio>` como filtro
 * y devuelve `conteos` para las pastillas.
 */
export type NegocioAdminExtra = {
	rubro?: string;
	estado?: EstadoNegocio;
	/** Días de gracia que quedan con el cobro fallido. */
	graciaDias?: number | null;
};
export type ConteosAdmin = Partial<Record<EstadoNegocio, number>>;

/** GET /v1/panel/acceso, por negocio (Mi cuenta). Los opcionales nuevos son el contrato propuesto. */
export type Acceso = {
	bloqueado: boolean;
	pruebaHasta: string | null;
	diasDePrueba: number;
	conSuscripcion: boolean;
	plan?: string;
	/** Propuesto: el próximo cobro de la suscripción. */
	proximoCobro?: { fecha: string; monto: number } | null;
	/** Propuesto: el último cobro falló y corre la gracia hasta esa fecha (ISO). */
	cobroFallidoHasta?: string | null;
};

/** GET /v1/invitaciones/{token}. Los opcionales son el contrato propuesto. */
export type Invitacion = {
	negocio: string;
	slug: string;
	email: string;
	rol: string;
	vencida: boolean;
	aceptada: boolean;
	/** Quién la mandó («Carla Ruiz»). */
	invitadoPor?: string;
	/** Hasta cuándo sirve (ISO). */
	venceEn?: string;
};

/**
 * GET/PUT /v1/preferencias/{token}. `email` (cambios en los turnos) y `recordatorios` existen; el
 * resto es el contrato propuesto: si la API no los manda, no se muestran ni se envían.
 */
export type Preferencias = {
	negocio: string;
	slug: string;
	cliente: string;
	email: boolean;
	recordatorios: boolean;
	/** El mail del cliente, para el encabezado. */
	correo?: string;
	resena?: boolean;
	listaEspera?: boolean;
	novedades?: boolean;
};
export const PREFERENCIAS_NUEVAS = ['resena', 'listaEspera', 'novedades'] as const;

/** GET /v1/explorar: cada ficha puede traer su puntaje y el próximo lugar libre; acepta `conLugarHoy=1`. */
export type FichaExtra = {
	promedio?: number | null;
	resenas?: number;
	/** ISO del próximo horario libre. */
	proximoLibre?: string | null;
	zonaHoraria?: string;
};
