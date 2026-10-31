/**
 * El registro de actividad en palabras: «Carla invitó a juli@… como encargada», con el área en un
 * rótulo y un punto de color por tipo (que nunca va solo: el área se lee al lado).
 */
import { ROLES } from '#lib/formato.ts';

export type EntradaAuditoria = {
	id: string;
	clerkId: string;
	nombre: string;
	accion: string;
	entidad: string;
	entidadId: string;
	detalle: unknown;
	creadoEn: string;
};

const AREAS: Record<string, string> = {
	turno: 'Turnos',
	cliente: 'Clientes',
	invitacion: 'Equipo',
	ajustes: 'Configuración',
	aviso: 'Avisos',
	espera: 'Lista de espera',
	resena: 'Reseñas',
	pago: 'Pagos',
	web: 'Mi web'
};

/** Petróleo: cambios del negocio; menta: lo del día a día; ámbar: plata; rojo: cancelaciones. */
const COLORES = { negocio: '#183c44', dia: '#3ccc90', plata: '#b26a00', baja: '#a3241a' };

const ACCIONES: Record<string, (d: Record<string, unknown>) => string> = {
	'equipo.invitado': (d) => `invitó a ${d.email ?? 'alguien'}${d.rol ? ` como ${(ROLES[String(d.rol)] ?? String(d.rol)).toLowerCase()}` : ''}`,
	'equipo.invitacion_borrada': (d) => `borró la invitación de ${d.email ?? 'alguien'}`,
	'equipo.invitacion_aceptada': (d) => `entró al equipo${d.rol ? ` como ${(ROLES[String(d.rol)] ?? String(d.rol)).toLowerCase()}` : ''}`,
	'turno.reservado': (d) => `cargó un turno${d.cliente ? ` de ${d.cliente}` : ''}`,
	'turno.cancelado': (d) => `canceló el turno${d.cliente ? ` de ${d.cliente}` : ''}`,
	'turno.cancelado_cliente': (d) => `canceló su turno`,
	'turno.reprogramado': (d) => `movió el turno${d.cliente ? ` de ${d.cliente}` : ''}`,
	'turno.ausente_automatico': () => 'quedó como ausencia: nadie cerró el turno',
	'espera.avisada': (d) => `avisó del hueco a ${d.email ?? 'alguien de la lista'}`,
	'espera.borrada': (d) => `sacó a ${d.email ?? 'alguien'} de la lista de espera`,
	'aviso.reenviado': (d) => `reenvió un mail${d.para ? ` a ${d.para}` : ''}`,
	'ajustes.avisos': () => 'cambió los avisos'
};

export function quienHizo(e: EntradaAuditoria) {
	return e.nombre || (e.clerkId ? 'Alguien del equipo' : 'Tengo Libre');
}

export function describir(e: EntradaAuditoria) {
	const d = e.detalle && typeof e.detalle === 'object' ? (e.detalle as Record<string, unknown>) : {};
	const texto = ACCIONES[e.accion]?.(d) ?? `hizo ${e.accion.replace(/[._]/g, ' ')}`;
	const area = AREAS[e.entidad] ?? e.entidad;
	const color = e.accion.includes('cancel') || e.accion.includes('borrad') || e.accion.includes('ausente')
		? COLORES.baja
		: e.entidad === 'pago'
			? COLORES.plata
			: e.entidad === 'turno' || e.entidad === 'espera' || e.entidad === 'aviso'
				? COLORES.dia
				: COLORES.negocio;
	return { quien: quienHizo(e), texto, area, color };
}

/** «10:08» si es de hoy, «SÁB» si es de esta semana, «12 OCT» si es más viejo. */
export function cuandoCorto(iso: string, ahora = new Date(), zona = 'America/Argentina/Buenos_Aires') {
	const d = new Date(iso);
	const dia = (x: Date) => new Intl.DateTimeFormat('en-CA', { timeZone: zona }).format(x);
	if (dia(d) === dia(ahora))
		return new Intl.DateTimeFormat('es-AR', { hour: '2-digit', minute: '2-digit', hour12: false, timeZone: zona }).format(d);
	const dias = (ahora.getTime() - d.getTime()) / 86_400_000;
	if (dias < 6) return new Intl.DateTimeFormat('es-AR', { weekday: 'short', timeZone: zona }).format(d).replace('.', '');
	return new Intl.DateTimeFormat('es-AR', { day: 'numeric', month: 'short', timeZone: zona }).format(d).replace('.', '');
}
