import { expect, test } from './base.ts';
import { entrar, negocioNuevo } from './ayuda.ts';
import { negocioConAgenda, reservarConAna, sql } from './publico-ayuda.ts';

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

test('la sugerencia de la casa: al final del Inicio, se oculta con el teclado y vuelve con «Deshacer»', async ({ page }) => {
	// Una cuenta de 20 días, con la guía «Para recibir turnos» completa, rubro, un turno y la prueba vigente.
	const a = await negocioConAgenda();
	await reservarConAna(a);
	sql(
		`UPDATE negocios SET rubro = 'barberia', whatsapp = '2235550000', creado_en = now() - interval '20 days',
			prueba_hasta = now() + interval '10 days' WHERE id = :'p1'`,
		a.negocio
	);
	const cerradas = () => sql(`SELECT count(*) FROM sugerencias_cerradas WHERE negocio_id = :'p1'`, a.negocio);
	await entrar(page, a.duenia);

	const link = page.getByRole('link', { name: /Creá tu tienda en Tu.Local.Web/ });
	await expect(link).toBeVisible();
	await expect(link).toHaveAttribute('href', /^https:\/\/tulocalweb\.com\.ar\/empezar\?utm_source=tengolibre/);

	// La cruz abre el menú con las flechas y Escape devuelve el foco a la cruz.
	const cruz = page.getByRole('button', { name: 'Ocultar sugerencia' });
	await cruz.focus();
	await page.keyboard.press('ArrowDown');
	await expect(page.getByRole('menuitem', { name: 'Ocultar por 30 días' })).toBeFocused();
	await page.keyboard.press('Escape');
	await expect(page.getByRole('menu')).toHaveCount(0);
	await expect(cruz).toBeFocused();

	// Ocultarla: se va, se anuncia y queda guardado.
	await page.keyboard.press('Enter');
	await page.getByRole('menuitem', { name: 'Ocultar por 30 días' }).click();
	await expect(link).toHaveCount(0);
	await expect(page.getByRole('status').filter({ hasText: 'La ocultamos por 30 días.' })).toBeFocused();
	await expect.poll(cerradas).toBe('1');

	// «Deshacer» la trae de vuelta, con el foco en su link, y borra el cierre.
	await page.getByRole('button', { name: 'Deshacer' }).click();
	await expect(link).toBeFocused();
	await expect.poll(cerradas).toBe('0');
	await page.reload();
	await expect(link).toBeVisible();
});
