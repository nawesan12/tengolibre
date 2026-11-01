/** Lo de los días libres que también usa el navegador (sin llamar a la API). */
export const MOTIVOS_LIBRES = ['vacaciones', 'tramite', 'salud', 'otro'] as const;
export type MotivoLibre = (typeof MOTIVOS_LIBRES)[number];
export const NOMBRE_MOTIVO: Record<MotivoLibre, string> = {
	vacaciones: 'Vacaciones',
	tramite: 'Trámite',
	salud: 'Salud',
	otro: 'Otro'
};

export type PedidoLibre = {
	id: string;
	profesionalId: string;
	profesional: string;
	desde: string;
	hasta: string;
	motivo: MotivoLibre;
	nota: string;
	estado: 'pendiente' | 'aprobado' | 'rechazado' | 'cancelado';
	/** Los turnos activos que hay ahora esos días («3 turnos para mover»). */
	afectados: number;
	turnos: number;
	respuesta: string;
	respondidoEn: string | null;
	creadoEn: string;
};
