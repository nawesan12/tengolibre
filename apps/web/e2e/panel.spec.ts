import { expect, test } from './base.ts';
import { negocioNuevo } from './ayuda.ts';

test('alta del negocio y un servicio desde el panel', async ({ page }) => {
	await negocioNuevo(page);
	await expect(page.getByRole('heading', { name: 'Para recibir turnos' })).toBeVisible();

	await page.goto('/panel/servicios');
	const alta = page.getByRole('form', { name: 'Agregar a Tus servicios' });
	await alta.getByLabel('Nombre', { exact: true }).fill('Corte clásico');
	await alta.getByLabel('Precio', { exact: true }).fill('8000');
	await alta.getByRole('button', { name: 'Agregar' }).click();
	await expect(page.getByLabel('Nombre de Corte clásico')).toBeVisible();
	await expect(page.getByLabel('Duración de Corte clásico en minutos')).toHaveValue('30');

	// Un servicio con duración inválida vuelve con el mensaje de la API, sin romper la página.
	await alta.getByLabel('Nombre', { exact: true }).fill('Raro');
	await alta.getByLabel('Duración (minutos)').fill('7');
	await alta.getByLabel('Precio', { exact: true }).fill('100');
	await alta.getByLabel('Duración (minutos)').evaluate((e: HTMLInputElement) => e.removeAttribute('step'));
	await alta.getByRole('button', { name: 'Agregar' }).click();
	await expect(page.getByRole('alert')).toContainText('duración');
});

test('sin sesión, el panel manda a ingresar', async ({ page }) => {
	await page.goto('/panel/servicios');
	await expect(page).toHaveURL(/\/ingresar\?volver=%2Fpanel%2Fservicios/);
});
