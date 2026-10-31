import { expect, test } from './base.ts';
import { negocioNuevo } from './ayuda.ts';

test('alta del negocio y un servicio desde el panel', async ({ page }) => {
	await negocioNuevo(page);
	await expect(page.getByText('Para empezar a recibir turnos')).toBeVisible();

	await page.goto('/panel/servicios');
	await page.getByLabel('Nombre', { exact: true }).fill('Corte clásico');
	await page.getByLabel('Precio').fill('8000');
	await page.getByRole('button', { name: 'Agregar' }).first().click();
	await expect(page.getByText('Corte clásico')).toBeVisible();
	await expect(page.getByText(/30 min/)).toBeVisible();

	// Un servicio con duración inválida vuelve con el mensaje de la API, sin romper la página.
	await page.getByLabel('Nombre', { exact: true }).fill('Raro');
	await page.getByLabel('Duración (minutos)').fill('7');
	await page.getByLabel('Precio').fill('100');
	await page.getByLabel('Duración (minutos)').evaluate((e: HTMLInputElement) => e.removeAttribute('step'));
	await page.getByRole('button', { name: 'Agregar' }).first().click();
	await expect(page.getByRole('alert')).toContainText('duración');
});

test('sin sesión, el panel manda a ingresar', async ({ page }) => {
	await page.goto('/panel/servicios');
	await expect(page).toHaveURL(/\/ingresar\?volver=%2Fpanel%2Fservicios/);
});
