import { dia as diaDe, hora } from '#lib/formato.ts';
import { diaDeSemana, minutosDelDia } from '#lib/operacion/fechas.ts';
import type { Profesional, Turno } from '#lib/operacion/tipos.ts';

/**
 * El día de cada profesional (firma 3 del handoff, «El día»): sus franjas de trabajo, lo ocupado y
 * los huecos que quedan. Todo en minutos desde la medianoche del negocio. Es una cuenta del front
 * con el horario semanal y los turnos: no mira bloqueos ni excepciones (eso lo sabe /v1/panel/libres).
 */
export type Tramo = { desde: number; hasta: number };
export type Hueco = { profesionalId: string; profesional: string; desde: number; hasta: number };
export type DiaProfesional = {
	id: string;
	nombre: string;
	franjas: Tramo[];
	ocupado: (Tramo & { turno: Turno })[];
	huecos: Hueco[];
	/** Minutos libres que quedan (sólo huecos de al menos MINIMO). */
	libres: number;
};

/** Un hueco más corto que esto no se ofrece (el handoff muestra desde media hora). */
export const MINIMO = 30;

const aMin = (h: string) => {
	const [a, b] = h.split(':').map(Number);
	return a * 60 + (b || 0);
};

/** «21 h 15», «45 min», «2 h» (como el prototipo). */
export function duracion(min: number) {
	const h = Math.floor(min / 60);
	const r = Math.round(min % 60);
	return h ? `${h} h${r ? ` ${r}` : ''}` : `${r} min`;
}

/** Igual que `duracion` pero redondeado a cuartos de hora hacia abajo. */
export const libresTexto = (min: number) => duracion(min - (min % 15));

/** HH:MM de minutos. */
export const hhmm = (m: number) => `${String(Math.floor(m / 60)).padStart(2, '0')}:${String(m % 60).padStart(2, '0')}`;

/** Lo que ocupa un turno en el día (los cancelados no ocupan). */
export const ocupa = (t: Turno) => t.estado !== 'cancelado';

/** Las franjas del día (AAAA-MM-DD) de un profesional, ordenadas y sin pisarse. */
export function franjasDe(p: Profesional, fecha: string): Tramo[] {
	const d = diaDeSemana(fecha);
	return p.horario
		.filter((f) => f.dia === d)
		.map((f) => ({ desde: aMin(f.desde), hasta: aMin(f.hasta) }))
		.filter((f) => f.hasta > f.desde)
		.sort((a, b) => a.desde - b.desde);
}

/** Resta los ocupados a las franjas: lo que queda libre, desde `desde` en adelante. */
export function restar(franjas: Tramo[], ocupados: Tramo[], desde = 0): Tramo[] {
	const out: Tramo[] = [];
	const orden = [...ocupados].sort((a, b) => a.desde - b.desde);
	for (const f of franjas) {
		let ini = Math.max(f.desde, desde);
		for (const o of orden) {
			if (o.hasta <= ini || o.desde >= f.hasta) continue;
			if (o.desde > ini) out.push({ desde: ini, hasta: Math.min(o.desde, f.hasta) });
			ini = Math.max(ini, o.hasta);
			if (ini >= f.hasta) break;
		}
		if (ini < f.hasta) out.push({ desde: ini, hasta: f.hasta });
	}
	return out;
}

/**
 * El día de cada profesional. `ahora` (minutos) recorta lo que ya pasó: los huecos arrancan en el
 * próximo cuarto de hora. Con `ahora` null se cuenta el día entero (otro día que no es hoy).
 */
export function armarDia(profesionales: Profesional[], turnos: Turno[], fecha: string, ahora: number | null): DiaProfesional[] {
	const delDia = turnos.filter((t) => diaDe(t.inicio) === fecha);
	return profesionales.map((p) => {
		const franjas = franjasDe(p, fecha);
		const ocupado = delDia
			.filter((t) => t.profesionalId === p.id && ocupa(t))
			.map((t) => {
				const desde = minutosDelDia(t.inicio);
				const largo = Math.max(5, Math.round((Date.parse(t.fin) - Date.parse(t.inicio)) / 60000));
				return { desde, hasta: Math.min(24 * 60, desde + largo), turno: t };
			})
			.sort((a, b) => a.desde - b.desde);
		const corte = ahora === null ? 0 : Math.ceil(ahora / 15) * 15;
		const huecos = restar(franjas, ocupado, corte)
			.filter((l) => l.hasta - l.desde >= MINIMO)
			.map((l) => ({ profesionalId: p.id, profesional: p.nombre, ...l }));
		return { id: p.id, nombre: p.nombre, franjas, ocupado, huecos, libres: huecos.reduce((s, h) => s + h.hasta - h.desde, 0) };
	});
}

/** De qué hora a qué hora va el día (para dibujar la franja): de la primera a la última. */
export function rangoDelDia(dias: DiaProfesional[], base: [number, number] = [9 * 60, 20 * 60]): [number, number] {
	const fr = dias.flatMap((d) => [...d.franjas, ...d.ocupado]);
	if (fr.length === 0) return base;
	const desde = Math.floor(Math.min(...fr.map((f) => f.desde)) / 60) * 60;
	const hasta = Math.ceil(Math.max(...fr.map((f) => f.hasta)) / 60) * 60;
	return [desde, Math.max(hasta, desde + 60)];
}

/**
 * Quiénes entran en «El día»: los activos (o con turnos ese día). Quien es profesional ve sólo lo
 * suyo: su profesional si la API lo dice (`profesionalId` del negocio) o los que tienen turnos suyos.
 */
export function profesionalesDelDia(profesionales: Profesional[], turnos: Turno[], rol: string, propio?: string | null) {
	if (rol === 'profesional') {
		if (propio) return profesionales.filter((p) => p.id === propio);
		const con = new Set(turnos.map((t) => t.profesionalId));
		return profesionales.filter((p) => con.has(p.id));
	}
	return profesionales.filter((p) => p.activo || turnos.some((t) => t.profesionalId === p.id));
}

/** El link para cargar un turno en un hueco (Nuevo turno con profesional, día y hora elegidos). */
export const cargarEn = (profesionalId: string, fecha: string, minutos: number) =>
	`/panel/turnos/nuevo?${new URLSearchParams({ profesional: profesionalId, fecha, hora: hhmm(minutos) })}`;

/** Los minutos de ahora en la zona del negocio. */
export const minutosAhora = (d = new Date()) => minutosDelDia(d.toISOString());

/** «10:12» de ahora. */
export const horaAhora = (d = new Date()) => hora(d);

/** Primera palabra de un nombre (para el saludo y las etiquetas cortas). */
export const primerNombre = (n: string) => n.trim().split(/\s+/)[0] ?? '';

/** Iniciales (dos letras; con un solo nombre, sus dos primeras: «Carla» → «CA»). */
export function iniciales(n: string) {
	const p = n.trim().split(/\s+/).filter(Boolean);
	if (p.length === 1) return p[0].slice(0, 2).toUpperCase();
	return p
		.slice(0, 2)
		.map((x) => x[0]!.toUpperCase())
		.join('');
}
