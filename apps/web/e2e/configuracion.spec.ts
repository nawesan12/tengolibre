import { expect, test, type Page } from '@playwright/test';
import { negocioNuevo, unico } from './ayuda.ts';

/** La API de las pruebas (la misma que levanta playwright.config.ts). */
const API = `http://127.0.0.1:${process.env.E2E_API_PUERTO ?? '4290'}`;

/** AAAA-MM-DD de dentro de `n` días, en Argentina. */
const enDias = (n: number) =>
	new Intl.DateTimeFormat('en-CA', { timeZone: 'America/Argentina/Buenos_Aires' }).format(new Date(Date.now() + n * 86_400_000));

/** Un PNG de 1×1 válido (la API mira los bytes, no el nombre). */
const PNG = Buffer.from(
	'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==',
	'base64'
);

const seccion = (page: Page, nombre: string) => page.getByRole('region', { name: nombre });

/**
 * Abre una pantalla del panel y espera a que esté hidratada (el layout abre el tiempo real en
 * onMount): lo que se escribe antes de hidratar lo pisan los valores iniciales de los campos.
 */
async function abrir(page: Page, ruta: string) {
	const sse = page.waitForRequest(/\/panel\/eventos/);
	await page.goto(ruta);
	await sse;
}

// Con el front recién levantado, vite compila cada ruta la primera vez que se pide y eso puede pasar
// el tiempo de espera de negocioNuevo: se calienta antes con un alta que no cuenta como prueba.
test.beforeAll(async ({ browser }) => {
	test.setTimeout(120_000);
	const page = await browser.newPage();
	await negocioNuevo(page).catch(() => {});
	await page.close();
});

test('profesional con horario y un día bloqueado: se refleja en los horarios libres', async ({ page, request }) => {
	const slug = await negocioNuevo(page);

	await abrir(page, '/panel/servicios');
	await page.getByLabel('Nombre', { exact: true }).fill('Corte');
	await page.getByLabel('Precio').fill('8000');
	await page.getByRole('button', { name: 'Agregar' }).first().click();
	await expect(page.getByText('Corte', { exact: true })).toBeVisible();

	await abrir(page, '/panel/equipo');
	await page.getByText('Sumar un profesional').click();
	await page.getByLabel('Nombre').fill('Ana Gómez');
	await page.getByLabel('Especialidad').fill('Barbera');
	await expect(page.getByLabel('Corte')).toBeChecked();
	await page.getByRole('button', { name: 'Sumar profesional' }).click();
	await expect(page).toHaveURL(/\/panel\/profesionales\/[0-9a-f-]+\?nuevo=1/);
	await expect(page.getByRole('heading', { name: 'Ana Gómez' })).toBeVisible();

	// Todos los días, de 9 a 13 y de 15 a 19 (horario cortado).
	const horario = seccion(page, 'Horario semanal');
	for (const d of [0, 1, 2, 3, 4, 5, 6]) {
		await horario.locator(`input[name="abre-${d}"]`).check();
		await horario.locator(`input[name="hasta-${d}-0"]`).fill('13:00');
		await horario.locator(`input[name="desde-${d}-1"]`).fill('15:00');
		await horario.locator(`input[name="hasta-${d}-1"]`).fill('19:00');
	}
	await horario.getByRole('button', { name: 'Guardar horario' }).click();
	await expect(horario.getByRole('status')).toHaveText('Guardamos el horario.');

	const bloqueada = enDias(3);
	const libre = enDias(4);
	const bloqueos = seccion(page, 'Días bloqueados');
	await bloqueos.getByLabel('Fecha').fill(bloqueada);
	await bloqueos.getByLabel('Motivo (opcional)').fill('Trámite');
	await bloqueos.getByRole('button', { name: 'Bloquear' }).click();
	await expect(bloqueos.getByRole('status')).toHaveText('Bloqueamos esa fecha.');
	await expect(bloqueos.getByText('Trámite', { exact: true })).toBeVisible();

	// Un bloqueo semanal también se guarda y se lista.
	const semanales = seccion(page, 'Horarios bloqueados todas las semanas');
	await semanales.getByLabel('Hora').fill('12:00');
	await semanales.getByRole('button', { name: 'Bloquear ese horario' }).click();
	await expect(semanales.getByText('Lunes 12:00')).toBeVisible();

	// La página pública: el día bloqueado no tiene horarios; el siguiente, sí (y sin el mediodía).
	const pub = await (await request.get(`${API}/v1/p/${slug}`)).json();
	const servicio = pub.servicios[0].id as string;
	const libres = async (fecha: string) => {
		const r = await request.get(`${API}/v1/p/${slug}/libres?servicio=${servicio}&fecha=${fecha}`);
		expect(r.ok()).toBe(true);
		return ((await r.json()).horarios ?? []) as { hora: string }[];
	};
	expect(await libres(bloqueada)).toEqual([]);
	const horas = (await libres(libre)).map((h) => h.hora);
	expect(horas).toContain('09:00');
	expect(horas).toContain('15:00');
	expect(horas).not.toContain('13:30');
});

test('invitación al equipo y ajustes guardados', async ({ page }) => {
	await negocioNuevo(page);

	await abrir(page, '/panel/equipo');
	const invitaciones = seccion(page, 'Invitaciones');
	const email = `e2e-${unico()}@ejemplo.com`;
	await invitaciones.getByLabel('Email').fill(email);
	await invitaciones.getByLabel('Rol').selectOption('recepcion');
	await invitaciones.getByRole('button', { name: 'Invitar' }).click();
	await expect(invitaciones.getByRole('status')).toContainText(email);
	await expect(invitaciones.getByRole('listitem').filter({ hasText: email })).toContainText('Recepción');

	await abrir(page, '/panel/configuracion');
	const reservas = seccion(page, 'Reservas');
	await reservas.getByLabel('Se puede reservar hasta (días)').fill('45');
	await reservas.getByLabel('Los turnos arrancan cada').selectOption('15');
	await reservas.getByRole('button', { name: 'Guardar reservas' }).click();
	await expect(reservas.getByRole('status')).toHaveText('Guardamos los ajustes de reserva.');

	const ausencias = seccion(page, 'Ausencias y penalidades');
	await ausencias.getByLabel('Días de penalidad').fill('15');
	await ausencias.getByRole('button', { name: 'Guardar ausencias' }).click();
	await expect(ausencias.getByRole('status')).toBeVisible();

	// Un valor fuera de rango vuelve con el mensaje de la API al lado del formulario.
	const avisos = seccion(page, 'Avisos');
	await avisos.getByLabel(/Pedido de reseña/).evaluate((e: HTMLInputElement) => e.removeAttribute('max'));
	await avisos.getByLabel(/Pedido de reseña/).fill('500');
	await avisos.getByRole('button', { name: 'Guardar avisos' }).click();
	await expect(avisos.getByRole('alert')).toContainText('168');

	await page.reload();
	await expect(page.getByLabel('Se puede reservar hasta (días)')).toHaveValue('45');
	await expect(page.getByLabel('Los turnos arrancan cada')).toHaveValue('15');
	await expect(page.getByLabel('Días de penalidad')).toHaveValue('15');

	await abrir(page, '/panel/auditoria');
	await expect(page.getByRole('heading', { name: 'Registro de cambios' })).toBeVisible();
	await expect(page.getByText('equipo.invitado')).toBeVisible();
	const csv = await page.request.get('/panel/auditoria/exportar');
	expect(csv.ok()).toBe(true);
	expect(await csv.text()).toContain('equipo.invitado');
});

test('una foto sube a la galería y un cupón se crea', async ({ page }) => {
	await negocioNuevo(page);

	await abrir(page, '/panel/mi-web');
	const galeria = seccion(page, 'Galería');
	await galeria.getByLabel('Foto', { exact: true }).setInputFiles({ name: 'local.png', mimeType: 'image/png', buffer: PNG });
	await galeria.getByLabel('Epígrafe (opcional)').fill('El local');
	await galeria.getByRole('button', { name: 'Sumar a la galería' }).click();
	await expect(galeria.getByRole('status')).toHaveText('Sumamos la foto a la galería.');
	await expect(galeria.getByRole('img', { name: 'El local' })).toHaveAttribute('src', /\/archivos\/.+\.png$/);

	// Un archivo que no es imagen vuelve con el mensaje de la API.
	await galeria.getByLabel('Foto', { exact: true }).setInputFiles({ name: 'nota.png', mimeType: 'image/png', buffer: Buffer.from('hola') });
	await galeria.getByRole('button', { name: 'Sumar a la galería' }).click();
	await expect(galeria.getByRole('alert')).toContainText('JPG, PNG o WebP');

	const web = seccion(page, 'Tu página');
	await web.getByLabel('Provincia').fill('Córdoba');
	await web.getByRole('button', { name: 'Guardar tu página' }).click();
	await expect(web.getByRole('status')).toHaveText('Guardamos tu web.');

	await abrir(page, '/panel/cupones');
	const nuevo = seccion(page, 'Nuevo cupón');
	await nuevo.getByLabel('Código').fill('prueba20');
	await nuevo.getByLabel('Descuento').fill('20');
	await nuevo.getByRole('button', { name: 'Crear cupón' }).click();
	await expect(nuevo.getByRole('status')).toHaveText('Creamos el cupón.');
	await expect(seccion(page, 'Tus cupones').getByText('PRUEBA20')).toBeVisible();
	await expect(seccion(page, 'Tus cupones').getByText('20 %')).toBeVisible();
});

test('suscripción en prueba y Mercado Pago sin configurar', async ({ page }) => {
	await negocioNuevo(page);

	await abrir(page, '/panel/suscripcion');
	await expect(page.getByText('Prueba gratis')).toBeVisible();
	await expect(page.getByText(/Te quedan? \d+ días?/)).toBeVisible();
	// En local no hay token de la plataforma: lo dice sin romperse.
	await page.getByRole('button', { name: 'Suscribirme al Inicial' }).click();
	await expect(page.getByRole('alert')).toContainText('todavía no están habilitadas');

	await abrir(page, '/panel/pagos');
	await expect(page.getByText('Los cobros con Mercado Pago todavía no están habilitados')).toBeVisible();
	await expect(page.getByRole('button', { name: 'Conectar Mercado Pago' })).toHaveCount(0);
	const cobro = seccion(page, 'Cómo cobrás al reservar');
	await cobro.getByLabel('Seña en porcentaje').check();
	await cobro.getByLabel('Porcentaje de la seña').fill('30');
	await cobro.getByRole('button', { name: 'Guardar' }).click();
	await expect(cobro.getByRole('status')).toHaveText('Guardamos cómo cobrás.');
	await expect(cobro.getByText('Hasta que conectes Mercado Pago')).toBeVisible();
	await expect(seccion(page, 'Pagos recibidos').getByText('Todavía no hay pagos')).toBeVisible();

	// Una vuelta de Mercado Pago que no sirve muestra el problema y el camino de vuelta.
	await abrir(page, '/panel/mercadopago/vuelta?code=x&state=y');
	await expect(page.getByRole('heading', { name: /No pudimos conectar/ })).toBeVisible();
	await expect(page.getByRole('alert')).toBeVisible();
	await abrir(page, '/panel/mercadopago/vuelta?error=access_denied');
	await expect(page.getByRole('alert')).toContainText('No se autorizó');
});

test('membresías, referidos y dominio, sin desbordar en el teléfono', async ({ page }) => {
	// Recorre todas las pantallas: la primera vez vite compila cada una.
	test.setTimeout(120_000);
	await negocioNuevo(page);
	const sinDesborde = async () =>
		expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);

	await abrir(page, '/panel/membresias');
	const planes = seccion(page, 'Planes');
	await planes.getByLabel('Nombre').fill('Club mensual');
	await planes.getByLabel('Precio por período ($)').fill('20000');
	await planes.getByRole('button', { name: 'Crear plan' }).click();
	await expect(planes.getByRole('status')).toHaveText('Creamos el plan.');
	const socios = seccion(page, 'Socios');
	await socios.getByText('Sumar un socio').click();
	await socios.getByLabel('Nombre').fill('Juan Pérez');
	await socios.getByLabel('Teléfono').fill('1155550000');
	await socios.getByRole('button', { name: 'Sumar socio' }).click();
	await expect(socios.getByRole('status')).toHaveText('Sumamos al socio.');
	await expect(socios.getByText(/le quedan 4/)).toBeVisible();
	await sinDesborde();

	await abrir(page, '/panel/referidos');
	await page.getByLabel('Programa de referidos activo').check();
	await page.getByRole('button', { name: 'Guardar' }).click();
	await expect(page.getByRole('status')).toHaveText('Guardamos el programa de referidos.');
	await sinDesborde();

	await abrir(page, '/panel/dominio');
	await page.getByLabel('Dominio').fill(`turnos.e2e${unico()}.com`);
	await page.getByRole('button', { name: 'Conectar' }).click();
	await expect(page.getByRole('status')).toContainText('Anotamos el dominio');
	await expect(page.getByRole('cell', { name: 'TXT' })).toBeVisible();
	await sinDesborde();

	for (const ruta of ['/panel/equipo', '/panel/mi-web', '/panel/configuracion', '/panel/pagos', '/panel/suscripcion', '/panel/cupones', '/panel/auditoria']) {
		await abrir(page, ruta);
		await sinDesborde();
	}
});
