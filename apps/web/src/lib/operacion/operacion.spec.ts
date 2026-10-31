import { describe, expect, it } from 'vitest';
import { diasEntre, instante, lunesDe, minutosDelDia, sumarDias, esDia, esHora, horaDe } from './fechas.ts';
import { carriles, horasVisibles } from './calendario.ts';
import { accionesRapidas, puedePasar } from './estados.ts';
import { escala, porcentaje } from './numeros.ts';

describe('fechas', () => {
	it('suma días cruzando meses y años', () => {
		expect(sumarDias('2026-12-31', 1)).toBe('2027-01-01');
		expect(sumarDias('2026-03-01', -1)).toBe('2026-02-28');
	});
	it('el lunes de la semana', () => {
		expect(lunesDe('2026-11-01')).toBe('2026-10-26'); // domingo
		expect(lunesDe('2026-11-02')).toBe('2026-11-02');
		expect(diasEntre('2026-11-02', '2026-11-04')).toEqual(['2026-11-02', '2026-11-03', '2026-11-04']);
	});
	it('pasa día y hora de Argentina a un instante y vuelta', () => {
		expect(instante('2026-11-02', '09:30')).toBe('2026-11-02T12:30:00.000Z');
		expect(instante('2026-11-02', '22:00')).toBe('2026-11-03T01:00:00.000Z');
		expect(minutosDelDia('2026-11-03T01:00:00Z')).toBe(22 * 60);
		expect(instante('2026-07-01', '10:00', 'Europe/Madrid')).toBe('2026-07-01T08:00:00.000Z');
	});
	it('valida días y horas', () => {
		expect(esDia('2026-11-02')).toBe(true);
		expect(esDia('2026-1-2')).toBe(false);
		expect(esHora('24:00')).toBe(false);
		expect(horaDe(605)).toBe('10:05');
	});
});

describe('calendario', () => {
	it('agranda las horas para que entren todos', () => {
		expect(horasVisibles([])).toEqual([8, 21]);
		expect(
			horasVisibles([
				{ id: 'a', inicio: 7 * 60 + 30, fin: 8 * 60 },
				{ id: 'b', inicio: 21 * 60, fin: 22 * 60 + 10 }
			])
		).toEqual([7, 23]);
	});
	it('reparte en carriles sólo los que se pisan', () => {
		const r = carriles([
			{ id: 'a', inicio: 540, fin: 570 },
			{ id: 'b', inicio: 550, fin: 580 },
			{ id: 'c', inicio: 570, fin: 600 },
			{ id: 'd', inicio: 700, fin: 730 }
		]);
		const por = Object.fromEntries(r.map((x) => [x.id, [x.carril, x.carriles]]));
		expect(por).toEqual({ a: [0, 2], b: [1, 2], c: [0, 2], d: [0, 1] });
	});
});

describe('estados', () => {
	it('sigue los pasos de la API', () => {
		expect(puedePasar('atendido', 'cancelado')).toBe(false);
		expect(accionesRapidas('pendiente').map((a) => a.estado)).toEqual(['confirmado', 'atendido', 'ausente', 'cancelado']);
		expect(accionesRapidas('cancelado')).toEqual([]);
	});
});

describe('números', () => {
	it('escala las barras sobre el máximo', () => {
		expect(escala([2, 4, 0])).toEqual([50, 100, 0]);
		expect(escala([0, 0])).toEqual([0, 0]);
		expect(porcentaje(null)).toBe('—');
		expect(porcentaje(12.345)).toBe('12,3 %');
	});
});
