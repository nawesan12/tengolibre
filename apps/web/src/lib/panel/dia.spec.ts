import { describe, expect, it } from 'vitest';
import { armarDia, duracion, iniciales, libresTexto, rangoDelDia, restar } from './dia.ts';
import type { Profesional, Turno } from '#lib/operacion/tipos.ts';

const prof = (id: string, desde = '10:00', hasta = '20:00'): Profesional => ({
	id,
	nombre: id,
	especialidad: '',
	activo: true,
	orden: 0,
	servicios: [],
	horario: [{ dia: 1, desde, hasta }] // lunes
});
const turno = (id: string, p: string, ini: string, fin: string, estado = 'confirmado') =>
	({ id, profesionalId: p, inicio: `2026-11-02T${ini}:00-03:00`, fin: `2026-11-02T${fin}:00-03:00`, estado }) as Turno;

describe('el día', () => {
	it('resta lo ocupado a las franjas', () => {
		expect(restar([{ desde: 600, hasta: 1200 }], [{ desde: 660, hasta: 720 }])).toEqual([
			{ desde: 600, hasta: 660 },
			{ desde: 720, hasta: 1200 }
		]);
		expect(restar([{ desde: 600, hasta: 700 }], [], 650)).toEqual([{ desde: 650, hasta: 700 }]);
	});
	it('arma los huecos de cada profesional sin los cancelados ni lo que pasó', () => {
		const ts = [turno('a', 'Ana', '10:00', '11:00'), turno('b', 'Ana', '11:00', '11:45', 'cancelado'), turno('c', 'Ana', '12:00', '12:10')];
		const [ana] = armarDia([prof('Ana')], ts, '2026-11-02', 10 * 60 + 12);
		expect(ana.huecos.map((h) => [h.desde, h.hasta])).toEqual([
			[11 * 60, 12 * 60],
			[12 * 60 + 10, 20 * 60]
		]);
		expect(ana.libres).toBe(60 + 470);
		expect(rangoDelDia([ana])).toEqual([600, 1200]);
		// Otro día que no trabaja: sin franjas ni huecos.
		expect(armarDia([prof('Ana')], [], '2026-11-03', null)[0].huecos).toEqual([]);
	});
	it('textos', () => {
		expect(duracion(405)).toBe('6 h 45');
		expect(duracion(45)).toBe('45 min');
		expect(libresTexto(1282)).toBe('21 h 15');
		expect(iniciales('Carla')).toBe('CA');
		expect(iniciales('Valeria Ortiz Paz')).toBe('VO');
	});
});
