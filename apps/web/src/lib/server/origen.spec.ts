import { describe, expect, it } from 'vitest';
import type { Cookies } from '@sveltejs/kit';
import { borrarOrigen, guardarOrigen, leerOrigen, origenDe } from './origen.ts';

/** Un frasco de cookies en memoria, con lo justo que usa origen.ts. */
function frasco() {
	const m = new Map<string, string>();
	return {
		get: (k: string) => m.get(k),
		set: (k: string, v: string) => void m.set(k, v),
		delete: (k: string) => void m.delete(k)
	} as unknown as Cookies;
}

describe('origenDe', () => {
	it('toma los utm válidos', () => {
		const o = origenDe(new URL('https://tengolibre.com.ar/empezar?utm_source=tulocalweb&utm_medium=sugerencia&utm_content=tlw-tl'), new Date('2026-12-01T10:00:00Z'));
		expect(o).toEqual({ fuente: 'tulocalweb', medio: 'sugerencia', contenido: 'tlw-tl', llego: '2026-12-01T10:00:00.000Z' });
	});
	it('descarta lo raro y sin utm devuelve null', () => {
		expect(origenDe(new URL('https://tengolibre.com.ar/empezar?utm_source=%3Cscript%3E'))).toBeNull();
		expect(origenDe(new URL('https://tengolibre.com.ar/empezar?ref=x'))).toBeNull();
	});
});

describe('la cookie del origen', () => {
	it('se lee lo guardado y, tras el alta, se borra: otro negocio del mismo navegador no lo hereda', () => {
		const c = frasco();
		guardarOrigen(c, { fuente: 'tulocalweb', medio: 'sugerencia', llego: '2026-12-01T10:00:00.000Z' });
		expect(leerOrigen(c)).toEqual({ fuente: 'tulocalweb', medio: 'sugerencia', llego: '2026-12-01T10:00:00.000Z' });
		borrarOrigen(c);
		expect(leerOrigen(c)).toBeNull();
	});
});
