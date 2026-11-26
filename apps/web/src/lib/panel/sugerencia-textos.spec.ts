import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

// La ayuda del interruptor «Sugerencias» dice lo mismo en la web, iOS, Android y la lámina;
// sólo cambia «Hoy» (apps) por «Inicio» (web).
const raiz = fileURLToPath(new URL('../../../../../', import.meta.url));
const leer = (ruta: string) => readFileSync(raiz + ruta, 'utf8');
const AYUDA_APPS = 'Otras apps de Grupo Vimo según tu rubro, al final de Hoy. No les pasamos tus datos.';

describe('ayuda del interruptor de sugerencias', () => {
	it('es la misma en las apps y en la lámina', () => {
		for (const ruta of [
			'apps/ios/TengoLibre/Pantallas/Mas/MasWeb.swift',
			'apps/android/app/src/main/java/ar/com/tengolibre/app/pantallas/mas/MasWeb.kt',
			'docs/laminas/sugerencia.html'
		]) {
			expect(leer(ruta), ruta).toContain(AYUDA_APPS);
		}
	});

	it('en la web cambia sólo Hoy por Inicio', () => {
		const svelte = leer('apps/web/src/routes/panel/configuracion/+page.svelte').replaceAll('&nbsp;', ' ');
		expect(svelte).toContain(AYUDA_APPS.replace('al final de Hoy', 'al final del Inicio'));
	});
});
