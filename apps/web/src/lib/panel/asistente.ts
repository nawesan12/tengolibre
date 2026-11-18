/** El asistente del panel (docs/api/asistente.md). */

export type TurnoChat = { rol: 'usuario' | 'asistente'; texto: string };

export type Propuesta = { id: string; accion: string; resumen: string; venceEn: string };

export type RespuestaAsistente = {
	respuesta: string;
	propuestas: Propuesta[];
	origen: 'motor' | 'ia' | 'cache';
	tope?: { motivo: string; mensaje: string };
};

export type EstadoAsistente = {
	ia: boolean;
	sugerencias: string[];
	cupo?: { cupoMensual: number; usadasMes: number; usadasHoy: number; topeDiario: number };
};

/** Lo que se le manda de historial: los últimos 8 turnos, sin los mensajes de error. */
export function historial(mensajes: { rol: 'usuario' | 'asistente'; texto: string; error?: boolean }[]): TurnoChat[] {
	return mensajes
		.filter((m) => !m.error && m.texto.trim())
		.slice(-8)
		.map((m) => ({ rol: m.rol, texto: m.texto }));
}

/** El cupo en una línea («IA: 12 de 150 respuestas este mes»); vacío si no hay IA o no hay cupo. */
export function lineaCupo(e: EstadoAsistente | null): string {
	if (!e?.ia || !e.cupo || !e.cupo.cupoMensual) return '';
	return `IA: ${e.cupo.usadasMes} de ${e.cupo.cupoMensual} respuestas este mes`;
}
