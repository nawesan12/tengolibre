import { describe, expect, it } from 'vitest';
import { historial, lineaCupo } from './asistente.ts';

describe('asistente', () => {
	it('manda los últimos 8 turnos, sin errores ni vacíos', () => {
		const ms = Array.from({ length: 12 }, (_, i) => ({ rol: (i % 2 ? 'asistente' : 'usuario') as 'usuario' | 'asistente', texto: `m${i}` }));
		ms.push({ rol: 'asistente', texto: 'falló', error: true } as never, { rol: 'usuario', texto: '  ' });
		const h = historial(ms);
		expect(h).toHaveLength(8);
		expect(h[0].texto).toBe('m4');
		expect(h.at(-1)?.texto).toBe('m11');
	});
	it('la línea del cupo sólo con IA', () => {
		expect(lineaCupo(null)).toBe('');
		expect(lineaCupo({ ia: false, sugerencias: [] })).toBe('');
		expect(lineaCupo({ ia: true, sugerencias: [], cupo: { cupoMensual: 150, usadasMes: 12, usadasHoy: 1, topeDiario: 30 } })).toBe(
			'IA: 12 de 150 respuestas este mes'
		);
	});
});
