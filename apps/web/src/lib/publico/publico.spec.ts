import { describe, expect, it, vi } from 'vitest';
import { diaCorto, fechaLarga, mesesCortos, semanas, sumarDias } from './calendario.ts';
import { completar, contraste, tema, textoSobre, variables } from './estilo.ts';
import { cacheDeHosts, esHostPropio, rutaPara } from './dominio.ts';
import { urlDePaso } from './reserva.ts';
import { duracion, estadoAtencion, iniciales, linkMapa, linkWhatsapp, montoSena, renglonesAtencion, sinViuda } from './textos.ts';

describe('calendario', () => {
	it('arma semanas de lunes a domingo con huecos', () => {
		// 2026-11-04 es miércoles.
		const s = semanas([
			{ fecha: '2026-11-04', libres: 3 },
			{ fecha: '2026-11-05', libres: 0 }
		]);
		expect(s).toHaveLength(1);
		expect(s[0].slice(0, 2)).toEqual([null, null]);
		expect(s[0][2]).toMatchObject({ fecha: '2026-11-04', numero: 4 });
		expect(s[0]).toHaveLength(7);
	});
	it('escribe la fecha larga y suma días', () => {
		expect(fechaLarga('2026-11-07')).toBe('Sábado 7 de noviembre');
		expect(sumarDias('2026-11-30', 1)).toBe('2026-12-01');
	});
});

describe('dominio propio', () => {
	it('no resuelve los hosts de TengoLibre ni otras rutas que la raíz', async () => {
		const buscar = vi.fn(async () => 'la-barberia');
		expect(esHostPropio('tengolibre.com.ar')).toBe(true);
		expect(await rutaPara(new URL('https://www.tengolibre.com.ar/'), buscar)).toBeUndefined();
		expect(await rutaPara(new URL('https://turnos.labarberia.com/turno/abc'), buscar)).toBeUndefined();
		expect(buscar).not.toHaveBeenCalled();
		expect(await rutaPara(new URL('https://Turnos.LaBarberia.com/'), buscar)).toBe('/la-barberia');
		expect(buscar).toHaveBeenCalledWith('turnos.labarberia.com');
	});
	it('sin negocio para el host, no cambia la ruta', async () => {
		expect(await rutaPara(new URL('https://nadie.com/'), async () => null)).toBeUndefined();
	});
	it('la caché vence', () => {
		let t = 0;
		const c = cacheDeHosts(1000, () => t);
		c.guardar('a.com', 'a');
		c.guardar('b.com', null);
		expect(c.leer('a.com')).toBe('a');
		expect(c.leer('b.com')).toBeNull();
		expect(c.leer('c.com')).toBeUndefined();
		t = 1000;
		expect(c.leer('a.com')).toBeUndefined();
	});
});

describe('pasos de la reserva', () => {
	it('al volver a un paso borra lo elegido después y conserva el referido', () => {
		const q = new URLSearchParams({ servicio: 's1', profesional: 'p1', fecha: '2026-11-04', inicio: 'x', ref: 'ANA123' });
		expect(urlDePaso('/b/reservar', q, 'dia')).toBe('/b/reservar?servicio=s1&profesional=p1&ref=ANA123');
		expect(urlDePaso('/b/reservar', q, 'profesional', { profesional: '' })).toBe('/b/reservar?servicio=s1&ref=ANA123');
		expect(urlDePaso('/b/reservar', new URLSearchParams(), 'servicio')).toBe('/b/reservar');
	});
});

describe('textos', () => {
	it('duración, mapa, whatsapp y renglones viudos', () => {
		expect(duracion(45)).toBe('45 min');
		expect(duracion(90)).toBe('1 h 30 min');
		expect(duracion(120)).toBe('2 h');
		expect(linkMapa(-34.6, -58.4)).toContain('query=-34.6,-58.4');
		expect(linkMapa(null, null)).toBe('');
		expect(linkWhatsapp('11 2345-6789')).toBe('https://wa.me/541123456789');
		expect(sinViuda('Sacá tu turno')).toBe('Sacá tu turno');
	});
});

describe('estilo de la web', () => {
	it('completa el estilo con los valores por defecto y descarta lo que no existe', () => {
		expect(completar(null)).toEqual({ plantilla: 'carta', portada: 'foto', fondo: 'crema', titulos: 'grandes', forma: 'redondos' });
		expect(completar({ plantilla: 'galeria', fondo: 'otro' })).toMatchObject({ plantilla: 'galeria', fondo: 'crema' });
	});

	it('el texto sobre el color del negocio es blanco o tinta según el contraste', () => {
		expect(textoSobre('#7a3b5e')).toBe('#ffffff');
		expect(textoSobre('#f5c518')).toBe('#14181a');
	});

	it('el color de marca como texto llega a 4,5:1 y en Noche se aclara', () => {
		const claro = tema(completar({}), '#f5c518', '#1d3557');
		expect(contraste(claro.marca, '#ffffff')).toBeGreaterThanOrEqual(4.5);
		const noche = tema(completar({ fondo: 'noche' }), '#7a3b5e', null);
		expect(noche.bg).toBe('#14221f');
		expect(contraste(noche.marca, noche.bg)).toBeGreaterThanOrEqual(4.5);
		expect(variables(noche)).toContain('--w-pri:#7a3b5e');
	});

	it('un color inválido usa el de por defecto', () => {
		expect(tema(completar({}), 'rojo', '').pri).toBe('#7a3b5e');
	});
});

describe('textos de la web', () => {
	it('horario de atención en renglones cortos', () => {
		const f = [1, 2, 3, 4, 5].map((dia) => ({ dia, desde: '10:00', hasta: '20:00' }));
		expect(renglonesAtencion([...f, { dia: 6, desde: '10:00', hasta: '14:00' }])).toEqual(['LUN–VIE 10–20', 'SÁB 10–14']);
		// Martes 3 de noviembre de 2026, 15:00 en Buenos Aires.
		expect(estadoAtencion(f, new Date('2026-11-03T18:00:00Z'), 'America/Argentina/Buenos_Aires').texto).toBe('Abierto hasta las 20');
		expect(estadoAtencion(f, new Date('2026-11-03T01:00:00Z'), 'America/Argentina/Buenos_Aires').abierto).toBe(false);
	});

	it('la seña, como la cuenta la API', () => {
		expect(montoSena(22000, { modo: 'porcentaje', senaPorcentaje: 30, senaFija: 0 })).toBe(6600);
		expect(montoSena(5000, { modo: 'fija', senaPorcentaje: 0, senaFija: 8000 })).toBe(5000);
		expect(montoSena(9000, { modo: 'total', senaPorcentaje: 0, senaFija: 0 })).toBe(9000);
		expect(montoSena(9000, { modo: 'sin_cobro', senaPorcentaje: 0, senaFija: 0 })).toBe(0);
		expect(montoSena(9000, null)).toBe(0);
	});

	it('rótulos cortos de días e iniciales', () => {
		expect(diaCorto('2026-11-02')).toBe('LUN 02');
		expect(mesesCortos(['2026-10-31', '2026-11-02'])).toBe('OCT · NOV');
		expect(iniciales('Carla Ruiz')).toBe('CR');
	});
});
