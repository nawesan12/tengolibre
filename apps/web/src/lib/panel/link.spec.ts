import { describe, expect, it } from 'vitest';
import { campoLink, ventanaMovimiento } from './link.ts';
import { rutaCanonica } from '#lib/publico/dominio.ts';

describe('el link', () => {
	it('un link viejo va al nuevo con la ruta y la query', () => {
		const u = (s: string) => new URL(`https://tengolibre.com.ar${s}`);
		expect(rutaCanonica(u('/lumi'), 'lumi', 'lumi-estetica')).toBe('/lumi-estetica');
		expect(rutaCanonica(u('/lumi/reservar?servicio=s1'), 'lumi', 'lumi-estetica')).toBe('/lumi-estetica/reservar?servicio=s1');
		expect(rutaCanonica(u('/lumi'), 'lumi', 'lumi')).toBeNull();
		// En un dominio propio la ruta no lleva el slug: no se mueve.
		expect(rutaCanonica(new URL('https://turnos.lumi.com/reservar'), 'lumi', 'lumi-estetica')).toBeNull();
	});
	it('el campo dice si el link sirve, nunca sólo con color', () => {
		expect(campoLink('lumi', 'propia')).toMatchObject({ ok: false, mensaje: 'Es tu link de ahora.' });
		expect(campoLink('lumi-2', 'libre')).toMatchObject({ ok: true, icono: 'check_circle' });
		expect(campoLink('dos', 'tomada').ok).toBe(false);
		expect(campoLink('ab', null).estado).toBe('corta');
	});
	it('cuenta los últimos 30 días', () => {
		expect(ventanaMovimiento('2026-11-18')).toEqual({ desde: '2026-10-20', hasta: '2026-11-18' });
	});
});
