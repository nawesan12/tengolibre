import { describe, expect, it } from 'vitest';
import { estadoAdmin, estadoDeAcceso, etiquetaLibre, iniciales, textoSobre, turnoGuardado } from './presentacion.ts';

const ahora = new Date('2026-11-02T12:00:00Z'); // lunes 2, 9:00 en Buenos Aires

describe('presentación de cuenta', () => {
	it('el próximo libre: hoy o el día', () => {
		expect(etiquetaLibre('2026-11-02T18:30:00Z', undefined, ahora)).toBe('HOY 15:30');
		expect(etiquetaLibre('2026-11-03T13:00:00Z', undefined, ahora)).toBe('MAR 03 10:00');
	});

	it('el turno guardado del socio', () => {
		expect(turnoGuardado('Lifting', '2026-11-02T13:30:00Z')).toBe('LIFTING · LUN 02 · 10:30');
		expect(turnoGuardado('Corte', null)).toBe('CORTE');
	});

	it('iniciales y contraste sobre el color del negocio', () => {
		expect(iniciales('Carla Ruiz')).toBe('CR');
		expect(iniciales('lumi', 1)).toBe('L');
		expect(textoSobre('#7a3b5e')).toBe('#fff');
		expect(textoSobre('#3ccc90')).toBe('#0f2328');
	});

	it('el estado en Mi cuenta, con lo que necesita', () => {
		const base = { bloqueado: false, pruebaHasta: null, diasDePrueba: 0, conSuscripcion: false };
		expect(estadoDeAcceso(null, 'duenio')).toBeNull();
		expect(estadoDeAcceso({ ...base, pruebaHasta: '2026-11-05T12:00:00Z', diasDePrueba: 3 }, 'duenio')).toMatchObject({
			texto: 'Prueba · 3 días',
			icono: 'timer',
			detalle: 'Elegí un plan para no pasar a sólo lectura el 5\u00a0nov.'
		});
		expect(estadoDeAcceso({ ...base, bloqueado: true }, 'encargado')?.detalle).toBe('El dueño tiene que elegir un plan.');
		expect(estadoDeAcceso({ ...base, conSuscripcion: true, plan: 'completo', proximoCobro: { fecha: '2026-11-12T12:00:00Z', monto: 15000 } }, 'duenio'))
			.toMatchObject({ texto: 'Plan Completo', tono: 'ok' });
	});

	it('el estado en el admin', () => {
		const n = { activo: true, reservable: true, pruebaHasta: null };
		expect(estadoAdmin({ ...n, activo: false }).texto).toBe('Desactivado');
		expect(estadoAdmin({ ...n, pruebaHasta: '2026-11-05T12:00:00Z' }, ahora).texto).toBe('Prueba · 3 días');
		expect(estadoAdmin(n).texto).toBe('Activo');
		expect(estadoAdmin({ ...n, estado: 'cobro_fallido', graciaDias: 2 }).texto).toBe('Cobro fallido · gracia 2 d');
	});
});
