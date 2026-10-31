import { describe, expect, it } from 'vitest';
import { fechaValida, paginas, prueba, ultimos30 } from './util.ts';

describe('cuenta', () => {
	it('cuenta las páginas', () => {
		expect(paginas(0, 25)).toBe(1);
		expect(paginas(25, 25)).toBe(1);
		expect(paginas(26, 25)).toBe(2);
	});
	it('valida fechas', () => {
		expect(fechaValida('2026-10-01')).toBe('2026-10-01');
		expect(fechaValida('2026-1-01')).toBeUndefined();
		expect(fechaValida('')).toBeUndefined();
		expect(fechaValida(null)).toBeUndefined();
	});
	it('arma los últimos 30 días en la hora de Argentina', () => {
		// 02:00 UTC del 1/11 todavía es 31/10 en Buenos Aires.
		expect(ultimos30(new Date('2026-11-01T02:00:00Z'))).toEqual({ desde: '2026-10-02', hasta: '2026-10-31' });
	});
	it('describe la prueba', () => {
		const ahora = new Date('2026-10-31T15:00:00Z');
		expect(prueba(null, ahora)).toBe('Sin prueba');
		expect(prueba('2026-10-01T00:00:00Z', ahora)).toBe('Prueba vencida');
		expect(prueba('2026-11-07T15:00:00Z', ahora)).toBe('Prueba hasta el 07/11/2026');
	});
});
