import { describe, expect, it } from 'vitest';
import { horarioDeForm, sacarSemanal, sumarSemanal, franjasDe } from './horario.ts';
import { beneficiosDe, cuponDeForm, diaComoInstante, instanteComoDia, resta } from './dinero.ts';
import { mensajeDe } from './mensajes.ts';

const form = (o: Record<string, string>) => {
	const f = new FormData();
	for (const [k, v] of Object.entries(o)) f.set(k, v);
	return f;
};

describe('horario', () => {
	it('arma las franjas de los días que abre, con horario cortado', () => {
		const f = form({
			'abre-1': 'on',
			'desde-1-0': '09:00',
			'hasta-1-0': '13:00',
			'desde-1-1': '16:00',
			'hasta-1-1': '20:00',
			// El martes tiene horas cargadas pero no abre.
			'desde-2-0': '09:00',
			'hasta-2-0': '13:00',
			// El sábado tiene una franja a medio llenar.
			'abre-6': 'on',
			'desde-6-0': '10:00',
			'hasta-6-0': '14:00',
			'desde-6-1': '15:00'
		});
		expect(horarioDeForm(f)).toEqual([
			{ dia: 1, desde: '09:00', hasta: '13:00' },
			{ dia: 1, desde: '16:00', hasta: '20:00' },
			{ dia: 6, desde: '10:00', hasta: '14:00' }
		]);
	});

	it('ordena las franjas de un día', () => {
		const h = [
			{ dia: 3, desde: '16:00', hasta: '20:00' },
			{ dia: 3, desde: '09:00', hasta: '12:00' },
			{ dia: 4, desde: '09:00', hasta: '12:00' }
		];
		expect(franjasDe(h, 3).map((x) => x.desde)).toEqual(['09:00', '16:00']);
	});

	it('suma bloqueos semanales sin repetir, de lunes a domingo', () => {
		let l = sumarSemanal([], { dia: 0, hora: '13:00' });
		l = sumarSemanal(l, { dia: 1, hora: '13:00' });
		l = sumarSemanal(l, { dia: 1, hora: '12:00' });
		l = sumarSemanal(l, { dia: 1, hora: '12:00' });
		expect(l).toEqual([
			{ dia: 1, hora: '12:00' },
			{ dia: 1, hora: '13:00' },
			{ dia: 0, hora: '13:00' }
		]);
		expect(sacarSemanal(l, 1, '13:00')).toHaveLength(2);
	});
});

describe('cupones y plata', () => {
	it('lee un cupón nuevo con fechas en hora de Argentina', () => {
		const c = cuponDeForm(
			form({ codigo: ' verano25 ', tipo: 'porcentaje', valor: '25', minimo: '', maxUsos: '10', desde: '2026-11-01', hasta: '2026-11-30', activo: 'on' }),
			true
		);
		expect(c).toEqual({
			codigo: 'VERANO25',
			tipo: 'porcentaje',
			valor: 25,
			minimo: null,
			maxUsos: 10,
			desde: '2026-11-01T00:00:00-03:00',
			hasta: '2026-11-30T23:59:59-03:00',
			activo: true
		});
	});

	it('en la edición no manda el código', () => {
		const c = cuponDeForm(form({ codigo: 'X', tipo: 'fijo', valor: '1500' }), false);
		expect(c.codigo).toBeUndefined();
		expect(c.activo).toBe(false);
		expect(c.hasta).toBeUndefined();
	});

	it('convierte días e instantes', () => {
		expect(diaComoInstante('mal')).toBeNull();
		expect(instanteComoDia('2026-11-30T23:59:59-03:00')).toBe('2026-11-30');
		expect(instanteComoDia(null)).toBe('');
	});

	it('beneficios, uno por renglón, y lo que queda por devolver', () => {
		expect(beneficiosDe('Corte\n\n  Barba \n')).toEqual(['Corte', 'Barba']);
		expect(resta({ monto: 5000, devuelto: 2000 })).toBe(3000);
	});
});

describe('mensajes', () => {
	it('el 402 invita a suscribirse en vez de mostrar el error', () => {
		const m = mensajeDe({ que: 'x', error: 'Suscribite para seguir operando.', codigo: 'suscripcion_requerida' }, 'x');
		expect(m?.tipo).toBe('suscripcion');
	});

	it('sólo muestra el mensaje del formulario que se mandó', () => {
		expect(mensajeDe({ que: 'a', mensaje: 'Listo.' }, 'b')).toBeNull();
		expect(mensajeDe({ que: 'a', mensaje: 'Listo.' }, 'a')).toEqual({ tipo: 'listo', texto: 'Listo.' });
		expect(mensajeDe({ que: 'a', error: 'Mal.', codigo: 'invalido' })).toEqual({ tipo: 'error', texto: 'Mal.' });
	});
});
