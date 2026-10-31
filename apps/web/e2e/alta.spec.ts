import { expect, test } from './base.ts';
import { entrar, negocioNuevo, unico } from './ayuda.ts';

test('landing: hero, calculadora y la dirección que se valida al instante', async ({ page }) => {
	await page.goto('/');
	await expect(page.getByRole('heading', { level: 1 })).toHaveText(/Llená cada\s*hueco\./);
	await expect(page.getByText('Próximamente').first()).toBeVisible();
	await expect(page.getByRole('img', { name: 'Código QR de Tengo Libre' })).toBeAttached();

	// Calculadora: 4 turnos × $ 12.000 × 4,3.
	await expect(page.getByText('$ 206.400')).toBeVisible();

	// Cierre: una dirección libre y otra tomada.
	const libre = `lugar-${unico()}`;
	const campo = page.getByLabel('Tu dirección');
	await campo.fill(libre);
	await expect(page.getByText(`${libre} está libre.`)).toBeVisible();
	const tomada = await negocioNuevoSlug(page);
	await page.goto('/');
	await page.getByLabel('Tu dirección').fill(tomada);
	await expect(page.getByText('Ya la usa otro negocio.')).toBeVisible();
	await page.getByLabel('Tu dirección').fill(libre);
	await expect(page.getByText(`${libre} está libre.`)).toBeVisible();
	await page.getByRole('button', { name: /Reservarla/ }).filter({ visible: true }).click();
	await expect(page).toHaveURL(new RegExp(`/empezar\\?slug=${libre}`));
});

// Crea un negocio con otra persona y vuelve a salir: su dirección queda tomada.
async function negocioNuevoSlug(page: import('./base.ts').Page) {
	const slug = await negocioNuevo(page);
	await page.context().clearCookies();
	return slug;
}

test('alta sin sesión: nombre, rubro y dirección; sigue al ingreso con lo cargado', async ({ page }) => {
	await page.goto('/empezar?rubro=unias');
	await expect(page.getByRole('heading', { name: 'Empecemos por lo básico.' })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Uñas' })).toHaveAttribute('aria-pressed', 'true');
	const crear = page.getByRole('button', { name: /Crear con Google/ });
	await expect(crear).toBeDisabled();
	const u = unico();
	await page.getByLabel('Nombre del negocio').fill(`Uñas ${u}`);
	await expect(page.getByLabel('Tu dirección web')).toHaveValue(`unas-${u}`);
	await expect(page.getByText('Está libre. Es tuya.')).toBeVisible();
	await expect(crear).toBeEnabled();
});

test('onboarding: los ocho pasos crean un negocio que se puede reservar', async ({ page }) => {
	const u = unico();
	await entrar(page, `user_e2e${u}`, '/empezar');

	// 1. Rubro: al elegir, sigue solo.
	await expect(page.getByRole('heading', { name: '¿Qué tipo de negocio tenés?' })).toBeVisible();
	await page.getByRole('button', { name: 'Barbería' }).click();

	// 2. Nombre y dirección.
	await expect(page.getByRole('heading', { name: '¿Cómo se llama?' })).toBeVisible();
	await page.getByLabel('Nombre del negocio').fill(`Barbería ${u}`);
	await expect(page.getByText('Está libre. Es tuya.')).toBeVisible();
	await page.getByRole('button', { name: 'Seguir' }).click();

	// 3. Servicios sugeridos por el rubro: se saca uno y se cambia un precio.
	await expect(page.getByRole('heading', { name: 'Te dejamos estos servicios' })).toBeVisible();
	await page.getByRole('button', { name: 'Sacar Color' }).click();
	await page.getByLabel('Precio de Corte', { exact: true }).fill('9500');
	await page.getByRole('button', { name: 'Seguir' }).click();

	// 4. Quién atiende y cuándo.
	await expect(page.getByRole('heading', { name: '¿Quién atiende y cuándo?' })).toBeVisible();
	const tuNombre = page.getByLabel('Tu nombre');
	if (await tuNombre.isVisible()) await tuNombre.fill('Tito');
	await page.getByRole('button', { name: 'Domingo, cerrado' }).click();
	await expect(page.getByRole('button', { name: 'Domingo, abierto' })).toBeVisible();
	await page.getByRole('button', { name: 'Seguir' }).click();

	// 5. Tu día.
	await expect(page.getByRole('heading', { name: '¿Cómo es tu día?' })).toBeVisible();
	await page.getByRole('button', { name: '15 min' }).click();
	await page.getByRole('button', { name: 'Seguir' }).click();

	// 6. Seña.
	await expect(page.getByRole('heading', { name: '¿Pedís seña al reservar?' })).toBeVisible();
	await page.getByRole('radio', { name: /Sin seña/ }).click();
	await page.getByRole('button', { name: 'Seguir' }).click();

	// 7. Color y plantilla.
	await expect(page.getByRole('heading', { name: 'Elegí cómo se ve tu página' })).toBeVisible();
	await page.getByRole('radio', { name: 'Terracota' }).click();
	await page.getByRole('button', { name: /^Vidriera/ }).click();
	await page.getByRole('button', { name: 'Publicar mi página' }).click();

	// 8. ¡Listo!
	await expect(page.getByRole('heading', { name: new RegExp(`Barbería ${u} ya toma`) })).toBeVisible();
	await expect(page.getByRole('img', { name: 'Código QR de tu página' })).toBeVisible();
	await expect(page.getByText('Plantilla Vidriera')).toBeVisible();
	await expect(page.getByText('Sin seña')).toBeVisible();

	// La página pública ya muestra lo cargado.
	await page.goto(`/barberia-${u}`);
	await expect(page.getByText('Corte y barba').first()).toBeVisible();

	// Volver a un paso trae lo guardado.
	await page.goto('/empezar?paso=servicios');
	await expect(page.getByLabel('Precio de Corte', { exact: true })).toHaveValue('9.500');
	// Lo que se sacó antes de guardar no se creó.
	await expect(page.getByRole('button', { name: /Color/ })).toHaveCount(0);
	await expect(page.getByRole('link', { name: 'Lo completo después' })).toBeVisible();
});
