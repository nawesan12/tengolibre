/** El asistente del panel (docs/api/asistente.md). */

export type TurnoChat = { rol: 'usuario' | 'asistente'; texto: string };

/** boton: lo que dice el botón de confirmar («Sí, cancelarlo»), según la acción. */
export type Propuesta = { id: string; accion: string; resumen: string; venceEn: string; boton?: string };

/** Un turno de una lista del asistente (se muestra como fila con link al turno). */
export type ItemTurno = { id: string; dia: string; hora: string; cliente: string; servicio: string; profesional?: string; estado?: string };

export type RespuestaAsistente = {
	respuesta: string;
	propuestas: Propuesta[];
	origen: 'motor' | 'ia' | 'cache';
	tope?: { motivo: string; mensaje: string };
	encabezado?: string;
	items?: ItemTurno[];
};

/**
 * ia: hay IA para este negocio. iaPlataforma: la plataforma tiene IA. iaNegocio: la prendió la dueña.
 * puedePrender: quien mira es la dueña.
 */
export type EstadoAsistente = {
	ia: boolean;
	iaPlataforma?: boolean;
	iaNegocio?: boolean;
	puedePrender?: boolean;
	sugerencias: string[];
	cupo?: { cupoMensual: number; usadasMes: number; usadasHoy: number; topeDiario: number };
};

/** El botón de confirmar de una propuesta: el de la API o uno que nombra la acción. */
export function botonDe(p: Propuesta): string {
	if (p.boton) return p.boton;
	return (
		{
			cancelar_turno: 'Sí, cancelarlo',
			reprogramar_turno: 'Sí, moverlo',
			confirmar_turno: 'Sí, confirmarlo',
			crear_turno: 'Sí, cargarlo',
			ofrecer_a_la_lista: 'Sí, ofrecerlo'
		}[p.accion] ?? 'Sí, hacerlo'
	);
}

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
