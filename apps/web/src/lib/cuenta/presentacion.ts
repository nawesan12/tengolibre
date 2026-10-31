/** Textos y estados de las pantallas de cuenta y accesos (diseño v2), sin Svelte para poder probarlos. */
import { dia, pesos, ZONA } from '#lib/formato.ts';
import type { Acceso, EstadoNegocio } from './contratos.ts';

/** Tono de una pastilla de estado: siempre va con ícono y palabra. */
export type Tono = 'ok' | 'aviso' | 'apagado' | 'peligro';
export type Estado = { texto: string; icono: string; tono: Tono; detalle?: string };

const DIAS3 = ['DOM', 'LUN', 'MAR', 'MIÉ', 'JUE', 'VIE', 'SÁB'];
const MESES = ['ene', 'feb', 'mar', 'abr', 'may', 'jun', 'jul', 'ago', 'sep', 'oct', 'nov', 'dic'];

/** Hora y minutos de un instante en la zona, «10:30». */
function hm(d: Date, zona: string) {
	return new Intl.DateTimeFormat('es-AR', { hour: '2-digit', minute: '2-digit', hour12: false, timeZone: zona }).format(d);
}

/** «LUN 02» de un instante, en la zona. */
export function diaCorto(iso: string | Date, zona = ZONA) {
	const d = new Date(iso);
	const [a, m, dd] = dia(d, zona).split('-').map(Number);
	const semana = new Date(Date.UTC(a, m - 1, dd)).getUTCDay();
	return `${DIAS3[semana]} ${String(dd).padStart(2, '0')}`;
}

/** «12 nov» de una fecha. */
export function diaMes(iso: string, zona = ZONA) {
	const [, m, dd] = dia(new Date(iso), zona).split('-').map(Number);
	return `${dd}\u00a0${MESES[m - 1]}`;
}

/** El próximo lugar libre como lo pide el diseño: «HOY 15:30» o «MAR 03 10:00». */
export function etiquetaLibre(iso: string, zona = ZONA, ahora = new Date()) {
	const d = new Date(iso);
	const mismo = dia(d, zona) === dia(ahora, zona);
	return `${mismo ? 'HOY' : diaCorto(d, zona)} ${hm(d, zona)}`;
}

/** El turno que el socio deja guardado mientras ingresa: «LIFTING · LUN 02 · 10:30». */
export function turnoGuardado(servicio: string | null, inicio: string | null, zona = ZONA) {
	const partes = [servicio?.toLocaleUpperCase('es-AR')];
	if (inicio && !Number.isNaN(Date.parse(inicio))) partes.push(diaCorto(inicio, zona), hm(new Date(inicio), zona));
	return partes.filter(Boolean).join(' · ');
}

/** Iniciales para el cuadrito del negocio o el avatar de la persona (hasta dos). */
export function iniciales(t: string, max = 2) {
	const p = t.trim().split(/\s+/).filter(Boolean);
	return (p.length > 1 ? p.slice(0, max).map((x) => x[0]) : [p[0]?.[0] ?? '?']).join('').toLocaleUpperCase('es-AR');
}

/** Un color #rrggbb válido, o el de respaldo. */
export const colorValido = (c: string | null | undefined, si = '#183c44') => (c && /^#[0-9a-f]{6}$/i.test(c) ? c : si);

/** Texto blanco o tinta sobre el color del negocio, el que dé más contraste. */
export function textoSobre(color: string) {
	const lin = (i: number) => {
		const v = parseInt(color.slice(i, i + 2), 16) / 255;
		return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
	};
	const l = 0.2126 * lin(1) + 0.7152 * lin(3) + 0.0722 * lin(5);
	// Contra blanco (1,05/(l+,05)) y contra la tinta #0f2328 (l≈,014).
	return 1.05 / (l + 0.05) >= (l + 0.05) / 0.064 ? '#fff' : '#0f2328';
}

/** El estado de un negocio en Mi cuenta, con lo que necesita de la persona. */
export function estadoDeAcceso(a: Acceso | null, rol: string): Estado | null {
	if (!a) return null;
	const duenia = rol === 'duenio';
	if (a.cobroFallidoHasta)
		return {
			texto: 'Cobro fallido',
			icono: 'credit_card_off',
			tono: 'peligro',
			detalle: duenia
				? `Revisá el medio de pago antes del ${diaMes(a.cobroFallidoHasta)}.`
				: 'El dueño tiene que revisar el medio de pago.'
		};
	if (a.bloqueado)
		return {
			texto: 'Sólo lectura',
			icono: 'visibility',
			tono: 'apagado',
			detalle: duenia ? 'Elegí un plan para volver a cargar turnos.' : 'El dueño tiene que elegir un plan.'
		};
	if (a.conSuscripcion) {
		const plan = a.plan === 'inicial' ? 'Plan Inicial' : a.plan === 'completo' ? 'Plan Completo' : 'Suscripción activa';
		return {
			texto: plan,
			icono: 'check_circle',
			tono: 'ok',
			detalle: a.proximoCobro ? `Próximo cobro ${diaMes(a.proximoCobro.fecha)} · ${pesos(a.proximoCobro.monto)}` : 'La suscripción está al día.'
		};
	}
	if (a.pruebaHasta) {
		const n = a.diasDePrueba;
		return {
			texto: `Prueba · ${n} ${n === 1 ? 'día' : 'días'}`,
			icono: 'timer',
			tono: 'aviso',
			detalle: duenia
				? `Elegí un plan para no pasar a sólo lectura el ${diaMes(a.pruebaHasta)}.`
				: `La prueba termina el ${diaMes(a.pruebaHasta)}.`
		};
	}
	return null;
}

/** El estado de un negocio en el admin. Sin `estado` de la API, sale de activo y de la prueba. */
export function estadoAdmin(
	n: { activo: boolean; reservable: boolean; pruebaHasta: string | null; estado?: EstadoNegocio; graciaDias?: number | null },
	ahora = new Date()
): Estado {
	if (!n.activo) return { texto: 'Desactivado', icono: 'block', tono: 'peligro' };
	switch (n.estado) {
		case 'completo':
			return { texto: 'Completo', icono: 'check_circle', tono: 'ok' };
		case 'inicial':
			return { texto: 'Inicial', icono: 'check_circle', tono: 'ok' };
		case 'cobro_fallido':
			return {
				texto: n.graciaDias != null ? `Cobro fallido · gracia ${n.graciaDias} d` : 'Cobro fallido',
				icono: 'credit_card_off',
				tono: 'peligro'
			};
		case 'solo_lectura':
			return { texto: 'Sólo lectura', icono: 'visibility', tono: 'apagado' };
	}
	if (n.pruebaHasta && new Date(n.pruebaHasta) > ahora) {
		const dias = Math.ceil((new Date(n.pruebaHasta).getTime() - ahora.getTime()) / 86_400_000);
		return { texto: `Prueba · ${dias} ${dias === 1 ? 'día' : 'días'}`, icono: 'timer', tono: 'aviso' };
	}
	return { texto: 'Activo', icono: 'check_circle', tono: 'ok' };
}

export const NOMBRE_ESTADO: Record<EstadoNegocio, string> = {
	prueba: 'En prueba',
	inicial: 'Inicial',
	completo: 'Completo',
	cobro_fallido: 'Cobro fallido',
	solo_lectura: 'Sólo lectura'
};

/** Qué puede hacer cada rol en el panel (lo que dice la invitación). `si` false = sin acceso. */
export const PERMISOS: Record<string, { texto: string; si: boolean }[]> = {
	duenio: [
		{ texto: 'Todo el negocio: turnos, clientes y cobros', si: true },
		{ texto: 'La web, el equipo y los ajustes', si: true },
		{ texto: 'La suscripción', si: true }
	],
	encargado: [
		{ texto: 'Ver y cargar turnos, confirmar y cobrar', si: true },
		{ texto: 'Servicios, equipo, web y ajustes', si: true },
		{ texto: 'Sin acceso a la suscripción', si: false }
	],
	recepcion: [
		{ texto: 'Ver y cargar turnos, confirmar y cobrar', si: true },
		{ texto: 'Clientes y lista de espera', si: true },
		{ texto: 'Sin acceso a la web, el equipo ni la suscripción', si: false }
	],
	profesional: [
		{ texto: 'Tu agenda: ver y cargar tus turnos', si: true },
		{ texto: 'Tus clientes y su historial', si: true },
		{ texto: 'Sin acceso a la web, el equipo ni la suscripción', si: false }
	]
};
