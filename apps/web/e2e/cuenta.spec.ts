import { expect, test, type Page } from './base.ts';
import { entrar, negocioNuevo, unico } from './ayuda.ts';

/** La persona de prueba del equipo (TENGOLIBRE_ADMINS en playwright.config.ts). */
const ADMIN = 'user_e2eadmin00000001';

/** Con la sesión abierta, da de alta otro negocio; devuelve su nombre. */
async function otroNegocio(page: Page, nombre: string) {
	await page.goto('/empezar');
	await page.getByLabel('Nombre del negocio').fill(nombre);
	await page.getByRole('button', { name: 'Crear' }).click();
	await expect(page.getByRole('heading', { name: /Buen día|Buenas tardes|Buenas noches/ })).toBeVisible();
	return nombre;
}

test('mi cuenta: dos negocios y cambiar entre ellos', async ({ page }) => {
	const slug = await negocioNuevo(page);
	const primero = `Barbería ${slug.replace('barberia-', '')}`;
	const segundo = await otroNegocio(page, `Salón ${unico()}`);

	await page.goto('/mi-cuenta');
	await expect(page.getByRole('heading', { name: 'Mi cuenta' })).toBeVisible();
	await expect(page.getByText(primero)).toBeVisible();
	await expect(page.getByText(segundo)).toBeVisible();
	await expect(page.getByText('Dueño/a').first()).toBeVisible();

	await page.getByRole('button', { name: `Entrar a ${primero}` }).click();
	await expect(page).toHaveURL(/\/panel$/);
	await expect(page.locator('[data-negocio-actual]:visible')).toHaveText(primero);

	await page.goto('/mi-cuenta');
	await page.getByRole('button', { name: `Entrar a ${segundo}` }).click();
	await expect(page.locator('[data-negocio-actual]:visible')).toHaveText(segundo);

	await page.goto('/mi-cuenta');
	await expect(page.getByRole('link', { name: 'Crear otro negocio' })).toHaveAttribute('href', '/empezar');
	await page.getByRole('button', { name: 'Salir' }).click();
	await expect(page).toHaveURL(/\/$/);
	await page.goto('/mi-cuenta');
	await expect(page).toHaveURL(/\/ingresar\?volver=%2Fmi-cuenta/);
});

test('sucursales: un grupo con dos negocios y su reporte', async ({ page }) => {
	const slug = await negocioNuevo(page);
	const primero = `Barbería ${slug.replace('barberia-', '')}`;
	const segundo = await otroNegocio(page, `Sede ${unico()}`);

	await page.goto('/mi-cuenta');
	await page.getByRole('link', { name: 'Sucursales' }).click();
	await expect(page.getByRole('heading', { name: 'Sucursales', level: 1 })).toBeVisible();

	await page.getByLabel('Nombre del grupo').fill('Cadena');
	await page.getByRole('button', { name: 'Crear grupo' }).click();
	const grupo = page.getByRole('region', { name: 'Grupo Cadena' });
	await expect(grupo).toBeVisible();

	for (const n of [primero, segundo]) {
		await grupo.getByLabel('Sumar un negocio a Cadena').selectOption({ label: n });
		await grupo.getByRole('button', { name: 'Sumar' }).click();
		await expect(grupo.getByRole('button', { name: `Sacar ${n} de Cadena` })).toBeVisible();
	}

	// Renombrar.
	await grupo.getByRole('button', { name: 'Renombrar' }).click();
	await grupo.getByLabel('Nombre del grupo').fill('Cadena Norte');
	await grupo.getByRole('button', { name: 'Guardar' }).click();
	const norte = page.getByRole('region', { name: 'Grupo Cadena Norte' });
	await expect(norte).toBeVisible();

	// El reporte consolidado, con las dos sucursales y el total.
	await norte.getByRole('link', { name: 'Ver reporte' }).click();
	const tabla = page.getByRole('table', { name: 'Turnos e ingresos por sucursal' });
	await expect(tabla.getByRole('rowheader', { name: primero })).toBeVisible();
	await expect(tabla.getByRole('rowheader', { name: segundo })).toBeVisible();
	await expect(tabla.getByRole('rowheader', { name: 'Total' })).toBeVisible();

	// Un rango al revés vuelve con el mensaje de la API.
	await page.getByLabel('Desde').fill('2026-10-20');
	await page.getByLabel('Hasta').fill('2026-10-01');
	await page.getByRole('button', { name: 'Ver reporte' }).click();
	await expect(page.getByRole('alert')).toContainText('fechas');

	// Sacar uno y borrar el grupo: los negocios quedan sueltos.
	await norte.getByRole('button', { name: `Sacar ${segundo} de Cadena Norte` }).click();
	await expect(norte.getByRole('button', { name: `Sacar ${segundo} de Cadena Norte` })).toHaveCount(0);
	page.once('dialog', (d) => d.accept());
	await norte.getByRole('button', { name: 'Borrar' }).click();
	await expect(page.getByText('Todavía no armaste ningún grupo.')).toBeVisible();
});

test('el admin es un 404 para una persona común, y sin sesión manda a ingresar', async ({ page }) => {
	await page.goto('/admin');
	await expect(page).toHaveURL(/\/ingresar\?volver=%2Fadmin/);

	await entrar(page, `user_e2e${unico()}`, '/admin');
	// Al entrar vuelve a /admin y ya es un 404; después, cargada de cero, también.
	await expect(page.getByRole('heading', { name: 'No encontramos esta página' })).toBeVisible();
	const r = await page.goto('/admin');
	expect(r?.status()).toBe(404);
	await expect(page.getByRole('heading', { name: 'No encontramos esta página' })).toBeVisible();
	await expect(page.getByText('Administración')).toHaveCount(0);
	expect((await page.goto('/admin/negocios'))?.status()).toBe(404);
});

test('admin: estadísticas, búsqueda y desactivar o activar un negocio', async ({ page, context }) => {
	const slug = await negocioNuevo(page);
	const nombre = `Barbería ${slug.replace('barberia-', '')}`;
	await context.clearCookies();

	await entrar(page, ADMIN, '/admin');
	await expect(page.getByRole('heading', { name: 'Resumen de la plataforma' })).toBeVisible();
	await expect(page.locator('[data-numero="Negocios"]')).toHaveText(/^\d+$/);

	await page.getByRole('navigation', { name: 'Administración' }).getByRole('link', { name: 'Negocios' }).click();
	await page.getByRole('searchbox').fill(slug);
	await page.getByRole('button', { name: 'Buscar' }).click();
	await expect(page).toHaveURL(new RegExp(`q=${slug}`));
	await expect(page.getByText('Un negocio')).toBeVisible();
	const fila = page.getByRole('listitem', { name: nombre });
	await expect(fila.getByText('Activo', { exact: true })).toBeVisible();
	await expect(fila.getByText(/Prueba hasta el/)).toBeVisible();

	await fila.getByRole('button', { name: 'Desactivar' }).click();
	await expect(fila.getByText('Desactivado')).toBeVisible();
	await fila.getByRole('button', { name: 'Activar' }).click();
	await expect(fila.getByText('Activo', { exact: true })).toBeVisible();

	const antes = await fila.getByText(/Prueba hasta el/).textContent();
	await fila.getByLabel(/Días para sumar/).fill('30');
	await fila.getByRole('button', { name: 'Extender prueba' }).click();
	await expect(fila.getByText(/Prueba hasta el/)).not.toHaveText(antes ?? '');

	// Una búsqueda sin resultados.
	await page.getByRole('searchbox').fill(`nada-${unico()}`);
	await page.getByRole('button', { name: 'Buscar' }).click();
	await expect(page.getByText('No hay negocios que coincidan.')).toBeVisible();
});
