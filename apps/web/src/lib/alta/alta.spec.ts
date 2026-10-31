import { describe, expect, it } from 'vitest';
import { aSlug, alternativas, estadoCampo, formatoValido } from './slug.ts';
import { franjas, horasSemana, leerHorario, proximosFeriados, segmentosDia } from './agenda.ts';
import { duracion, pesos, rubroDeApi } from './rubros.ts';

describe('dirección', () => {
	it('arma la dirección desde el nombre', () => {
		expect(aSlug('Lumi Estética')).toBe('lumi-estetica');
		expect(aSlug('  Don Tito & Cía. ')).toBe('don-tito-cia');
		expect(aSlug('x'.repeat(60))).toHaveLength(48);
	});
	it('valida el formato de la API', () => {
		expect(formatoValido('lumi')).toBe(true);
		expect(formatoValido('-lumi')).toBe(false);
		expect(formatoValido('lu--mi')).toBe(false);
	});
	it('ofrece alternativas distintas a la pedida', () => {
		expect(alternativas('lumi', 'estetica')).toEqual(['lumi-estetica', 'lumi-turnos', 'soylumi']);
	});
	it('nunca dice el estado sólo con color', () => {
		for (const r of [null, { estado: 'libre' as const, sugerencias: [] }, { estado: 'tomada' as const, sugerencias: ['a'] }]) {
			const e = estadoCampo('lumi', r);
			expect(e.icono).not.toBe('');
			expect(e.mensaje).not.toBe('');
		}
		expect(estadoCampo('lu', null).estado).toBe('corta');
	});
});

describe('agenda', () => {
	it('arma franjas con domingo como 0 y el descanso partido', () => {
		expect(franjas([1, 7], '10:00', '19:00')).toEqual([
			{ dia: 1, desde: '10:00', hasta: '19:00' },
			{ dia: 0, desde: '10:00', hasta: '19:00' }
		]);
		expect(franjas([2], '10:00', '19:00', { desde: '13:00', hasta: '14:00' })).toEqual([
			{ dia: 2, desde: '10:00', hasta: '13:00' },
			{ dia: 2, desde: '14:00', hasta: '19:00' }
		]);
		expect(franjas([2], '19:00', '10:00')).toEqual([]);
	});
	it('cuenta las horas de la semana', () => {
		expect(horasSemana(6, '10:00', '19:00', 1)).toBe(54);
		expect(horasSemana(6, '10:00', '19:00', 2, { desde: '13:00', hasta: '14:00' })).toBe(96);
	});
	it('lee lo que ya estaba cargado', () => {
		const h = franjas([1, 2, 7], '09:00', '18:00', { desde: '13:00', hasta: '14:00' });
		expect(leerHorario(h)).toEqual({ dias: [1, 2, 7], abre: '09:00', cierra: '18:00', descanso: { desde: '13:00', hasta: '14:00' } });
		expect(leerHorario([])).toBeNull();
	});
	it('arma la tira del día', () => {
		const { turnos, seg } = segmentosDia('10:00', '12:00', 30, 10, null);
		expect(turnos).toBe(3);
		expect(seg.reduce((a, s) => a + s.f, 0)).toBe(120);
	});
	it('da los próximos feriados en orden', () => {
		const f = proximosFeriados('2026-10-31');
		expect(f.map((x) => x.fecha)).toEqual(['2026-12-08', '2026-12-25', '2027-01-01', '2027-02-08']);
		expect(f[0].rotulo).toBe('MAR 08 DIC');
	});
});

describe('formato', () => {
	it('plata y duración', () => {
		expect(pesos(8000)).toBe('$\u00a08.000');
		expect(duracion(90)).toBe('1 h 30 min');
		expect(duracion(60)).toBe('1 h');
		expect(rubroDeApi('salud')?.clave).toBe('consultorio');
	});
});
