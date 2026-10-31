import { test as base, expect, type Page } from '@playwright/test';

/** Espera a que la página esté hidratada (lo marca el layout raíz en onMount). */
export const hidratada = (page: Page) => page.locator('html[data-hidratado="1"]').waitFor({ state: 'attached' });

/**
 * `test` con un `page.goto` que, además de cargar, espera la hidratación: así ninguna prueba escribe
 * en un campo que Svelte después pisa (pasa con la máquina cargada).
 */
export const test = base.extend({
	page: async ({ page }, usar) => {
		const goto = page.goto.bind(page);
		page.goto = async (url, opciones) => {
			const r = await goto(url, opciones);
			await hidratada(page);
			return r;
		};
		await usar(page);
	}
});

export { expect };
export type { Page };
