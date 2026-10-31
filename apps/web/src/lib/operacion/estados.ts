/**
 * Los pasos de estado que el panel puede dar (los mismos de la API, turnos.PuedePasar), y las
 * acciones rápidas que se muestran en cada turno.
 */
const PASOS: Record<string, string[]> = {
	pendiente_pago: ['pendiente', 'confirmado', 'cancelado'],
	pendiente: ['confirmado', 'cancelado', 'atendido', 'ausente'],
	confirmado: ['pendiente', 'cancelado', 'atendido', 'ausente'],
	atendido: ['ausente', 'confirmado'],
	ausente: ['atendido', 'confirmado']
};

export const puedePasar = (de: string, a: string) => de === a || (PASOS[de] ?? []).includes(a);

const RAPIDAS = [
	{ estado: 'confirmado', texto: 'Confirmar' },
	{ estado: 'atendido', texto: 'Atendido' },
	{ estado: 'ausente', texto: 'No vino' },
	{ estado: 'cancelado', texto: 'Cancelar' }
];

/** Las acciones rápidas que tienen sentido desde ese estado (sin la de quedarse igual). */
export const accionesRapidas = (estado: string) => RAPIDAS.filter((a) => a.estado !== estado && puedePasar(estado, a.estado));

/** Los estados que ocupan la agenda (un cancelado no). */
export const ocupa = (estado: string) => estado !== 'cancelado';
