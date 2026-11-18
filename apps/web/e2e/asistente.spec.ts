import { expect, test } from './base.ts';
import { negocioNuevo } from './ayuda.ts';

// El asistente de Inicio: pregunta, respuesta del intérprete (sin IA en local) y que entre en el
// teléfono sin correr la página de costado.
test('el asistente de Inicio contesta lo básico', async ({ page }, info) => {
	await negocioNuevo(page);
	const asis = page.getByRole('region', { name: 'Preguntale al asistente' });
	await expect(asis).toBeVisible();
	await asis.getByRole('button', { name: '¿Qué tengo mañana?' }).click();
	await expect(asis.getByText('Mañana no hay turnos.')).toBeVisible();

	await asis.getByLabel('Tu pregunta').fill('Pasá a Juana al jueves a las 5');
	await asis.getByRole('button', { name: 'Preguntar' }).click();
	await expect(asis.getByText(/No encontré un turno de Juana/)).toBeVisible();

	if (info.project.name === 'telefono') {
		const ancho = await page.evaluate(() => [document.documentElement.scrollWidth, window.innerWidth]);
		expect(ancho[0]).toBeLessThanOrEqual(ancho[1]);
	}
	await asis.screenshot({ path: info.outputPath('asistente.png') });
});
