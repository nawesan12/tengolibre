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

/**
 * Cómo se ve cada estado (handoff «Panel v2»): ícono + palabra siempre, nunca sólo el color.
 * `en_curso` es un confirmado que ya empezó y todavía no terminó.
 */
export const ESTILO: Record<string, { texto: string; icono: string; fondo: string; color: string; borde: string }> = {
	confirmado: { texto: 'Confirmado', icono: 'check_circle', fondo: '#e3f6ec', color: '#17805a', borde: 'none' },
	en_curso: { texto: 'En curso', icono: 'check_circle', fondo: '#e3f6ec', color: '#17805a', borde: 'none' },
	pendiente: { texto: 'Pendiente', icono: 'radio_button_unchecked', fondo: '#fffdf9', color: '#0f2328', borde: 'inset 0 0 0 1px #c9bda4' },
	pendiente_pago: { texto: 'Esperando pago', icono: 'hourglass_top', fondo: '#fbe9c2', color: '#7a4a00', borde: 'none' },
	atendido: { texto: 'Atendido', icono: 'done_all', fondo: '#183c44', color: '#fff', borde: 'none' },
	ausente: { texto: 'No vino', icono: 'person_off', fondo: '#fbe0dc', color: '#a3241a', borde: 'none' },
	cancelado: { texto: 'Cancelado', icono: 'cancel', fondo: '#ece5d6', color: '#4a5d61', borde: 'none' }
};

/** El estado a mostrar: un confirmado que está pasando ahora es «En curso». */
export function estadoVisible(estado: string, inicio: string, fin: string, ahora = Date.now()) {
	if (estado === 'confirmado' && Date.parse(inicio) <= ahora && Date.parse(fin) > ahora) return 'en_curso';
	return estado;
}

/**
 * La acción principal de un turno (el botón petróleo de la fila): confirmar lo pendiente y marcar
 * atendido lo que ya empezó. El resto va en el menú «⋯».
 */
export function accionPrincipal(estado: string, inicio: string, ahora = Date.now()) {
	if (estado === 'pendiente' || estado === 'pendiente_pago') return { estado: 'confirmado', texto: 'Confirmar', icono: 'check' };
	if (estado === 'confirmado' && Date.parse(inicio) <= ahora) return { estado: 'atendido', texto: 'Atendido', icono: 'done_all' };
	return null;
}

/** Íconos de las acciones del menú. */
export const ICONO_ACCION: Record<string, string> = { confirmado: 'check', atendido: 'done_all', ausente: 'person_off', cancelado: 'cancel', pendiente: 'undo' };
