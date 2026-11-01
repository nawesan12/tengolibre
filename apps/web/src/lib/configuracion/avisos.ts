/**
 * Los ajustes de avisos nuevos de Configuración, con los contratos de la API (docs/api/):
 * whatsapp.md, espera-ofertas.md, recordatorios.md, push-web.md y reporte-mensual.md.
 */

/** GET/PUT /v1/panel/ajustes/whatsapp */
export type AjustesWhatsapp = { disponible: boolean; confirmacion: boolean; hora: string; sinRespuestaHoras: number; recordatorio: boolean };

/** GET /v1/panel/whatsapp/mensajes */
export type MensajeWhatsapp = {
	id: string;
	tipo: 'confirmacion' | 'recordatorio' | 'respuesta';
	para: string;
	turnoId: string | null;
	cliente: string;
	estado: 'pendiente' | 'enviado' | 'descartado' | 'fallido';
	intentos: number;
	ultimoError: string | null;
	enviadoEn: string | null;
	entregadoEn: string | null;
	leidoEn: string | null;
	respuesta: 'confirmo' | 'cambiar' | null;
	respondidoEn: string | null;
	sinRespuestaEn: string | null;
	creadoEn: string;
};

/** GET/PUT /v1/panel/ajustes/espera */
export type AjustesEspera = { minutos: number; automatica: boolean };

/** GET /v1/panel/recordatorios */
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

/** GET/PUT /v1/panel/push/preferencias */
export type Preferencias = {
	rol: string;
	porDefecto: boolean;
	apagados: string[];
	tipos: { tipo: string; nombre: string; activo: boolean }[];
};

/** GET /v1/yo/suscripciones-web */
export type SuscripcionWeb = { clave: string; suscripta: boolean };

/** GET/PUT /v1/panel/reportes/mensual/ajustes */
export type AjustesMensual = { activo: boolean; encargados: { id: string; nombre: string; email: string; recibe: boolean }[] };

/** Cómo se lee el estado de un WhatsApp: ícono + palabra (nunca sólo color). */
export function estadoWhatsapp(m: MensajeWhatsapp): { icono: string; texto: string; clase: 'ok' | 'espera' | 'error' | 'neutro' } {
	if (m.respuesta === 'confirmo') return { icono: 'check_circle', texto: 'Confirmó', clase: 'ok' };
	if (m.respuesta === 'cambiar') return { icono: 'edit_calendar', texto: 'Pidió cambiarlo', clase: 'espera' };
	if (m.estado === 'fallido') return { icono: 'error', texto: 'No le llegó', clase: 'error' };
	if (m.estado === 'descartado') return { icono: 'block', texto: m.ultimoError ? `No salió: ${m.ultimoError}` : 'No salió', clase: 'error' };
	if (m.estado === 'pendiente') return { icono: 'sync', texto: m.intentos > 1 ? `Reintentando · ${m.intentos}º intento` : 'En cola', clase: 'espera' };
	if (m.sinRespuestaEn) return { icono: 'call', texto: 'Sin respuesta: para llamar', clase: 'espera' };
	if (m.leidoEn) return { icono: 'done_all', texto: 'Leído', clase: 'ok' };
	if (m.entregadoEn) return { icono: 'check', texto: 'Entregado', clase: 'ok' };
	return { icono: 'send', texto: 'Enviado', clase: 'neutro' };
}

export const TIPO_WHATSAPP: Record<string, string> = { confirmacion: 'Confirmación', recordatorio: 'Recordatorio', respuesta: 'Respuesta' };
