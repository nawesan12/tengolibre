import { describe, expect, it, vi } from 'vitest';
import { fechaLarga, semanas, sumarDias } from './calendario.ts';
import { cacheDeHosts, esHostPropio, rutaPara } from './dominio.ts';
import { urlDePaso } from './reserva.ts';
import { duracion, linkMapa, linkWhatsapp, sinViuda } from './textos.ts';

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
