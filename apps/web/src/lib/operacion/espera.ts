import { esDia } from './fechas.ts';
import type { Anotado } from './tipos.ts';

/** Las pestañas de la lista de espera (handoff «Panel v2»). */
export const TABS_ESPERA = [
	{ clave: 'esperando', nombre: 'Esperando' },
	{ clave: 'avisados', nombre: 'Avisados' },
	{ clave: 'vencidos', nombre: 'Vencidos' }
] as const;
export type TabEspera = (typeof TABS_ESPERA)[number]['clave'];

/** En qué pestaña va cada anotado: avisado, vencido (pasó su fecha sin lugar) o esperando. */
export const tabDe = (a: Anotado, hoy: string): TabEspera => (a.avisadoEn ? 'avisados' : esDia(a.hasta) && a.hasta < hoy ? 'vencidos' : 'esperando');
