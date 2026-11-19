/**
 * Lo que la web sabe de los planes sin pedirlo: los nombres (para los botones «Pasar a …») y los días
 * de prueba (los textos públicos). Precios, topes y lo que incluye cada plan salen de la API
 * (GET /v1/planes y /v1/panel/suscripcion): no se repiten acá.
 */
import catalogo from './planes-catalogo.json';

export const DIAS_DE_PRUEBA = catalogo.diasDePrueba;

export const NOMBRE_PLAN: Record<string, string> = { inicial: 'Inicial', equipo: 'Equipo', centro: 'Centro' };

export const nombrePlan = (clave?: string | null) => (clave ? (NOMBRE_PLAN[clave] ?? '') : '');

/** El link a la pantalla de planes con el plan sugerido ya elegido. */
export const linkPlanes = (plan?: string | null) => (plan && NOMBRE_PLAN[plan] ? `/panel/suscripcion?plan=${plan}#planes` : '/panel/suscripcion#planes');

/** Funciones de un plan (planes.Funcion de la API). */
export type Funcion =
	| 'sena'
	| 'espera'
	| 'membresias'
	| 'cupones'
	| 'referidos'
	| 'penalidades'
	| 'reportes_completos'
	| 'sucursales'
	| 'dominio'
	| 'plantillas'
	| 'csv';

/** Lo que dice el aviso de cada función que el plan no trae (sujeto + verbo). */
export const QUE_VIENE: Record<Funcion, string> = {
	sena: 'La seña con Mercado Pago viene',
	espera: 'La lista de espera viene',
	membresias: 'Las membresías vienen',
	cupones: 'Los cupones vienen',
	referidos: 'Los referidos vienen',
	penalidades: 'Las penalidades por ausencias vienen',
	reportes_completos: 'Los reportes completos vienen',
	sucursales: 'Las sucursales vienen',
	dominio: 'El dominio propio viene',
	plantillas: 'Las plantillas de página vienen',
	csv: 'Exportar en CSV viene'
};

/** El acceso del panel (GET /v1/panel/acceso): qué funciones trae el plan que rige. */
export type AccesoPlan = { plan?: string; funciones?: string[]; viene?: Record<string, string> } | null | undefined;

/** El plan que trae f, si el plan que rige no la trae (null: la trae, o no se sabe). */
export function vieneCon(acceso: AccesoPlan, f: Funcion): string | null {
	if (!acceso?.viene) return null;
	return acceso.viene[f] ?? null;
}
