/** Las formas de la API que usa la operación diaria del panel (turnos, clientes, reportes, avisos). */

export type ClienteDelTurno = { id?: string; nombre: string; telefono: string; email: string };

export type Turno = {
	id: string;
	codigo: string;
	servicioId: string | null;
	servicio: string;
	profesionalId: string | null;
	profesional: string;
	cliente: ClienteDelTurno;
	inicio: string;
	fin: string;
	precio: number;
	estado: string;
	origen: string;
	nota: string;
	venceEn: string | null;
	creadoEn: string;
};

export type Franja = { dia: number; desde: string; hasta: string };

export type Profesional = {
	id: string;
	nombre: string;
	especialidad: string;
	activo: boolean;
	orden: number;
	servicios: string[];
	horario: Franja[];
};

export type Prestacion = {
	id: string;
	nombre: string;
	duracion: number;
	precio: number;
	activo: boolean;
};

export type HorarioLibre = { hora: string; inicio: string; profesionales: string[] };

export type Etiqueta = { id: string; nombre: string; color: string };
export type EtiquetaConUso = Etiqueta & { clientes: number };

export type Cliente = {
	id: string;
	nombre: string;
	telefono: string;
	email: string;
	creadoEn: string;
	visitas: number;
	ultimaVisita: string | null;
	proximoTurno: string | null;
	gastado: number;
	ausencias: number;
	penalizado: boolean;
	etiquetas: Etiqueta[];
};

export type Penalidad = {
	id: string;
	clienteId: string;
	cliente: string;
	telefono: string;
	motivo: string;
	nota: string;
	ausencias: number;
	hasta: string;
	creadoEn: string;
	levantadaEn: string | null;
	vigente: boolean;
};

export type TurnoDeCliente = Omit<Turno, 'cliente' | 'venceEn' | 'creadoEn'>;

export type Ficha = Cliente & { turnos: TurnoDeCliente[]; penalidad: Penalidad | null };

export type Nota = { id: string; texto: string; miembroId: string | null; autor: string; creadoEn: string };

export type Anotado = {
	id: string;
	servicioId: string;
	servicio: string;
	profesionalId: string | null;
	profesional: string | null;
	nombre: string;
	email: string;
	telefono: string;
	desde: string;
	hasta: string;
	avisadoEn: string | null;
	creadoEn: string;
};

export type Resena = {
	id: string;
	turnoId: string;
	nombre: string;
	servicio: string;
	profesional: string;
	puntaje: number;
	comentario: string;
	respuesta: string;
	respondidaEn: string | null;
	visible: boolean;
	creadoEn: string;
};

export type FilaReporte = {
	id: string | null;
	nombre: string;
	turnos: number;
	atendidos: number;
	ausentes: number;
	ingresos: number;
	minutosReservados: number;
	minutosDisponibles?: number;
	ocupacion?: number;
};

export type DiaReporte = { fecha: string; turnos: number; atendidos: number; ingresos: number };

export type Reporte = {
	desde: string;
	hasta: string;
	resumen: {
		turnos: number;
		atendidos: number;
		ingresos: number;
		ticketPromedio: number;
		clientes: number;
		tasaAusencias: number | null;
		ocupacion: number | null;
	};
	porEstado: Record<string, number>;
	porProfesional: FilaReporte[];
	porServicio: FilaReporte[];
	porDia: DiaReporte[];
	clientesNuevos: number;
	clientesRecurrentes: number;
};

export type Notificacion = {
	id: string;
	tipo: string;
	titulo: string;
	texto: string;
	turnoId: string | null;
	creadoEn: string;
	nueva: boolean;
};

export type RegistroAviso = {
	id: string;
	tipo: string;
	para: string;
	turnoId: string | null;
	cliente: string | null;
	estado: string;
	intentos: number;
	ultimoError: string | null;
	proximoIntento: string;
	enviadoEn: string | null;
	creadoEn: string;
};

export type Encontrado = {
	clientes: { id: string; nombre: string; telefono: string; email: string }[];
	turnos: { id: string; inicio: string; estado: string; servicio: string; profesional: string; clienteId: string; cliente: string }[];
	servicios: { id: string; nombre: string; precio: number; duracion: number; activo: boolean }[];
};
