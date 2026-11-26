import { describe, expect, it } from 'vitest';
import { origenDe } from './origen.ts';

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
