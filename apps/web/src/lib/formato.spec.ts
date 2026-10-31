import { describe, expect, it } from 'vitest';
import { dia, hora, pesos } from './formato.ts';

describe('formato', () => {
	it('muestra pesos sin centavos', () => {
		expect(pesos(8000).replace(/\s/g, ' ')).toMatch(/\$ ?8\.000/);
	});
	it('usa la hora de Argentina', () => {
		expect(hora('2026-11-02T12:00:00Z')).toBe('09:00');
		expect(dia('2026-11-02T02:00:00Z')).toBe('2026-11-01');
	});
});
