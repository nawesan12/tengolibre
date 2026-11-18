import { expect, test } from './base.ts';
import { negocioNuevo, unico } from './ayuda.ts';

test('el link: primero en Inicio, se cambia en Mi web y el viejo sigue andando', async ({ page }) => {
	const viejo = await negocioNuevo(page);

	// Sin turnos, el link va arriba de todo y en grande.
	await page.goto('/panel');
	await expect(page.locator('[data-link]').first()).toHaveText(new RegExp(`/${viejo}$`));
	await expect(page.getByRole('button', { name: 'Copiar' }).first()).toBeVisible();

	// Mi web: «Cambiar link» abre el editor con la vista previa y el aviso.
	await page.goto('/panel/mi-web');
	await page.getByRole('button', { name: 'Cambiar link' }).click();
	await expect(page.getByText('Es tu link de ahora.')).toBeVisible();
	await expect(page.getByText('Tu link de antes sigue andando.')).toBeVisible();
	const campo = page.getByLabel('Tu link nuevo');
	await campo.fill('panel');
	await expect(page.getByText('Esa dirección no se puede usar.')).toBeVisible();
	const nuevo = `link-${unico()}`;
	await campo.fill(nuevo);
	await expect(page.getByText('Está libre.')).toBeVisible();
	await page.getByRole('button', { name: 'Guardar link' }).click();
	await expect(page.getByText('Tu link nuevo ya anda.')).toBeVisible();
	await expect(page.getByText(`/${viejo}`).last()).toBeVisible();

	// El viejo (y su reserva) redirige al nuevo.
	await page.goto(`/${viejo}/reservar`);
	await expect(page).toHaveURL(new RegExp(`/${nuevo}/reservar$`));
});
