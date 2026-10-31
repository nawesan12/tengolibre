import { expect, type Page } from '@playwright/test';

/** Un id de persona y un slug nuevos por prueba (la base local se reusa). */
export const unico = () => Date.now().toString(36) + Math.random().toString(36).slice(2, 6);

/** Entra con el ingreso de prueba como esa persona. */
export async function entrar(page: Page, id: string, volver = '/panel') {
	await page.goto(`/ingresar?volver=${encodeURIComponent(volver)}`);
	await page.getByLabel('Id de la persona').fill(id);
	await page.getByRole('button', { name: 'Entrar' }).click();
}

/** Entra y da de alta un negocio; devuelve su slug. */
export async function negocioNuevo(page: Page) {
	const u = unico();
	await entrar(page, `user_e2e${u}`, '/empezar');
	await expect(page.getByRole('heading', { name: 'Creá tu negocio' })).toBeVisible();
	await page.getByLabel('Nombre del negocio').fill(`Barbería ${u}`);
	await page.getByRole('button', { name: 'Crear' }).click();
	await expect(page.getByRole('heading', { name: /Hola/ })).toBeVisible();
	return `barberia-${u}`;
}
