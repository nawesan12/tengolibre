import { expect, hidratada } from './base.ts';
import type { Page } from './base.ts';

/** Un id de persona y un slug nuevos por prueba (la base local se reusa). */
export const unico = () => Date.now().toString(36) + Math.random().toString(36).slice(2, 6);

/** Entra con el ingreso de prueba como esa persona. */
export async function entrar(page: Page, id: string, volver = '/panel') {
	await page.goto(`/ingresar?volver=${encodeURIComponent(volver)}`);
	await page.getByLabel('Id de la persona').fill(id);
	await page.getByRole('button', { name: 'Entrar' }).click();
	await page.waitForURL((u) => !u.pathname.startsWith('/ingresar'));
	await hidratada(page);
}

/**
 * Con la sesión abierta, crea un negocio por el onboarding (/empezar): rubro en la URL, el nombre (la
 * dirección se valida al instante) y «Seguir»; el resto de los pasos se completa después. Termina en
 * el panel del negocio nuevo.
 */
export async function crearNegocio(page: Page, nombre: string) {
	await page.goto('/empezar?paso=nombre&rubro=barberia');
	await expect(page.getByRole('heading', { name: '¿Cómo se llama?' })).toBeVisible();
	await page.getByLabel('Nombre del negocio').fill(nombre);
	await expect(page.getByText('Está libre. Es tuya.')).toBeVisible();
	await page.getByRole('button', { name: 'Seguir' }).click();
	await page.waitForURL(/paso=servicios/);
	await page.goto('/panel');
	await expect(page.getByRole('heading', { name: /Hola/ })).toBeVisible();
}

/** Entra y da de alta un negocio; devuelve su slug. */
export async function negocioNuevo(page: Page) {
	const u = unico();
	await entrar(page, `user_e2e${u}`, '/empezar');
	await crearNegocio(page, `Barbería ${u}`);
	return `barberia-${u}`;
}
