/**
 * Las formas de las funciones nuevas del diseño v2 (docs/api/): ofrecer el hueco a la lista
 * (espera-ofertas.md), confirmación por WhatsApp (whatsapp.md), turno fijo (turno-fijo.md), caja
 * (caja.md), días libres (dias-libres.md), recordatorios (recordatorios.md) y push web (push-web.md).
 */

// ── Ofrecer el hueco (espera-ofertas.md) ──
export type Intento = {
	id: string;
	esperaId: string | null;
	nombre: string;
	orden: number;
	estado: 'en_cola' | 'ofrecida' | 'tomada' | 'paso' | 'vencida' | 'cancelada' | 'salteada';
	ofrecidaEn: string | null;
	venceEn: string | null;
	respondidaEn: string | null;
};
export type Oferta = {
	id: string;
	estado: 'ofreciendo' | 'tomada' | 'cargada' | 'frenada' | 'agotada' | 'vencida' | 'caida';
	origen: 'automatica' | 'panel';
	servicioId: string;
	servicio: string;
	profesionalId: string;
	profesional: string;
	inicio: string;
	fin: string;
	precio: number;
	turnoId: string | null;
	creadoEn: string;
	cerradaEn: string | null;
	actual: Intento | null;
	sigue: Intento | null;
	cola: Intento[];
};

// ── Confirmación por WhatsApp (whatsapp.md): lo que suma cada turno del panel ──
export type ExtrasTurno = {
	serieId?: string | null;
	confirmadoPor?: 'whatsapp' | 'web' | 'panel' | null;
	confirmadoEn?: string | null;
	cambioPedidoEn?: string | null;
	llamarDesde?: string | null;
	llamarMotivo?: 'cambio' | 'sin_respuesta' | 'no_llego' | null;
};
export type ParaLlamar = {
	turnoId: string;
	cliente: string;
	telefono: string;
	servicio: string;
	profesional: string;
	inicio: string;
	estado: string;
	motivo: 'cambio' | 'sin_respuesta' | 'no_llego';
	desde: string;
};
export const MOTIVO_LLAMAR: Record<string, string> = {
	cambio: 'Pidió cambiarlo',
	sin_respuesta: 'No contestó el WhatsApp',
	no_llego: 'No le llegó el WhatsApp'
};

// ── Turno fijo (turno-fijo.md) ──
export type FechaSerie = {
	inicio: string;
	fecha: string;
	hora: string;
	estado: 'libre' | 'choca';
	motivo?: 'ocupado' | 'bloqueado' | 'no_atiende' | 'pasado';
	alternativa: { inicio: string; hora: string } | null;
};
export type PreviaSerie = {
	cadaSemanas: number;
	servicio: string;
	profesional: string;
	libres: number;
	chocan: number;
	fechas: FechaSerie[];
};
export const MOTIVO_CHOCA: Record<string, string> = {
	ocupado: 'Ocupado a esa hora',
	bloqueado: 'Está bloqueado ese día',
	no_atiende: 'No atiende ese día',
	pasado: 'Ya pasó'
};

// ── Caja (caja.md) ──
export type Producto = { id: string; nombre: string; precio: number; stock: number | null; activo: boolean; orden: number };
export type LineaCaja = { tipo: 'servicio' | 'cupon' | 'membresia' | 'sena' | 'producto' | 'descuento'; concepto: string; monto: number; cantidad?: number; productoId?: string };
export type CuentaCaja = {
	turnoId: string;
	cliente: string;
	servicio: string;
	estadoTurno: string;
	estado: 'sin_cobrar' | 'debe' | 'cobrado';
	medio: string | null;
	cobradoEn: string | null;
	lineas: LineaCaja[];
	total: number;
	medios: string[];
};
export type Medio = 'efectivo' | 'transferencia' | 'debito' | 'credito' | 'mercadopago';
export const MEDIOS: { clave: Medio; texto: string; icono: string }[] = [
	{ clave: 'efectivo', texto: 'Efectivo', icono: 'payments' },
	{ clave: 'transferencia', texto: 'Transfer.', icono: 'account_balance' },
	{ clave: 'mercadopago', texto: 'QR MP', icono: 'qr_code_2' },
	{ clave: 'debito', texto: 'Débito', icono: 'credit_card' },
	{ clave: 'credito', texto: 'Crédito', icono: 'credit_score' }
];

// ── Días libres (dias-libres.md) ──
export type PedidoDias = {
	id: string;
	profesionalId: string;
	profesional: string;
	desde: string;
	hasta: string;
	motivo: 'vacaciones' | 'tramite' | 'personal' | 'otro';
	nota: string;
	estado: 'pendiente' | 'aprobado' | 'rechazado' | 'cancelado';
	turnos: number;
	afectados: number;
	creadoEn: string;
};
export type AReprogramar = {
	id: string;
	pedidoId: string;
	motivo: string;
	resueltoEn: string | null;
	turno: { id: string; inicio: string; fin: string; servicio: string; profesional: string; estado: string; cliente: { nombre: string; telefono: string; email: string } };
};
export const MOTIVO_DIAS: Record<string, string> = { vacaciones: 'Vacaciones', tramite: 'Trámite', personal: 'Personal', otro: 'Otro' };

// ── Recordatorios (recordatorios.md) ──
export type Recordatorio = {
	tipo: 'cumpleanios' | 'extraniamos';
	titulo: string;
	activo: boolean;
	texto: string;
	cuponId: string | null;
	cupon: { id: string; codigo: string; descuento: string; vigente: boolean } | null;
	dias: number;
	regla: string;
	vistaPrevia: { para: string; asunto: string; html: string; texto: string };
	semana: { clienteId: string; nombre: string; email: string; fecha: string; detalle: string; enviado: boolean }[];
	semanaTexto: string;
};
