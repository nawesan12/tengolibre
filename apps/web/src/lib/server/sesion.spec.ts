import { describe, expect, it } from 'vitest';
import { volverValido } from './sesion.ts';

describe('volverValido', () => {
	it('acepta rutas propias y la reserva de un negocio', () => {
		for (const v of ['/panel', '/panel/turnos?fecha=2026-11-02', '/invitacion/abc', '/peluqueria-ana/reservar', '/peluqueria-ana/reservar?servicio=x&inicio=y', '/conectar?solicitud=abc_-123'])
			expect(volverValido(v), v).toBe(true);
	});
	it('rechaza lo de afuera y lo que no es una reserva', () => {
		for (const v of ['', 'https://otro.com', '//otro.com/panel', '/\\otro.com', '/peluqueria-ana', '/peluqueria-ana/reservarx', '/a/reservar', '/panelx', '/conectarx'])
			expect(volverValido(v), v).toBe(false);
	});
});
