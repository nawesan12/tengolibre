import { dia, hora } from '#lib/formato.ts';
import type { Notificacion } from '#lib/operacion/tipos.ts';

/** Ícono y color de cada aviso de la campanita (handoff «Panel v2»: siempre ícono + texto). */
export const ICONO_AVISO: Record<string, { icono: string; color: string }> = {
	reserva: { icono: 'event_available', color: '#17805a' },
	cancelacion: { icono: 'cancel', color: '#a3241a' },
	reprogramacion: { icono: 'event_repeat', color: '#183c44' },
	resena: { icono: 'star', color: '#b26a00' },
	espera: { icono: 'hourglass_top', color: '#17805a' },
	hueco: { icono: 'bolt', color: '#17805a' },
	equipo: { icono: 'group_add', color: '#183c44' },
	ausente: { icono: 'person_off', color: '#a3241a' },
	aviso_fallido: { icono: 'error', color: '#a3241a' },
	pago: { icono: 'paid', color: '#17805a' },
	dias_libres: { icono: 'beach_access', color: '#7a4a00' }
};

export const iconoAviso = (tipo: string) => ICONO_AVISO[tipo] ?? { icono: 'notifications', color: '#183c44' };

/** «09:58» si es de hoy, «sáb» si es de otro día. */
export function cuandoAviso(iso: string) {
	if (dia(iso) === dia()) return hora(iso);
	return new Intl.DateTimeFormat('es-AR', { weekday: 'short', timeZone: 'America/Argentina/Buenos_Aires' }).format(new Date(iso)).replace('.', '');
}

/** A dónde lleva un aviso: al turno, si tiene, o a la campanita. */
export const destinoAviso = (n: Notificacion) => (n.turnoId ? `/panel/turnos/${n.turnoId}` : '/panel/notificaciones');
