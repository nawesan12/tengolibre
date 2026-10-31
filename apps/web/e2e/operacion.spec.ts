import { expect, test, type APIRequestContext, type Page } from './base.ts';
import { crearNegocio, entrar, unico } from './ayuda.ts';

/**
 * La operación diaria del panel contra la API de verdad: el negocio se da de alta por el panel y el
 * servicio y la profesional (con horario todos los días) por la API, con el mismo ingreso de prueba.
 */
// Cada prueba arma su negocio y recorre varias pantallas: más tiempo que el de una sola.
test.describe.configure({ timeout: 90_000 });

const API = `http://127.0.0.1:${process.env.E2E_API_PUERTO ?? '4290'}`;

const manana = () =>
	new Intl.DateTimeFormat('en-CA', { timeZone: 'America/Argentina/Buenos_Aires' }).format(new Date(Date.now() + 86_400_000));

/**
 * Una pantalla del panel ya hidratada: el layout abre el tiempo real (/panel/eventos) al montarse,
 * así los formularios ya andan con enhance y los clics no se adelantan.
 */
async function irA(page: Page, url: string) {
	await Promise.all([page.waitForRequest(/\/panel\/eventos/), page.goto(url)]);
}

type Armado = { slug: string; servicio: string; profesional: string; api: (metodo: string, ruta: string, cuerpo?: unknown) => Promise<any> };

async function armar(page: Page, request: APIRequestContext): Promise<Armado> {
	const u = unico();
	const quien = `user_e2e${u}`;
	await entrar(page, quien, '/empezar');
	await crearNegocio(page, `Barbería ${u}`);

	const auth = { authorization: `Bearer dev:${quien}` };
	const yo = await (await request.get(`${API}/v1/yo`, { headers: auth })).json();
	const negocio = yo.negocios[0];
	const api = async (metodo: string, ruta: string, cuerpo?: unknown) => {
		const r = await request.fetch(`${API}${ruta}`, { method: metodo, headers: { ...auth, 'x-negocio': negocio.id }, data: cuerpo });
		expect(r.ok(), `${metodo} ${ruta}: ${r.status()} ${await r.text()}`).toBeTruthy();
		return r.status() === 204 ? null : r.json();
	};
	const s = await api('POST', '/v1/panel/servicios', { nombre: 'Corte', duracion: 30, precio: 8000 });
	const horario = [0, 1, 2, 3, 4, 5, 6].map((dia) => ({ dia, desde: '09:00', hasta: '20:00' }));
	const p = await api('POST', '/v1/panel/profesionales', { nombre: 'Ana', servicios: [s.id], horario, orden: 1 });
	return { slug: negocio.slug, servicio: s.id, profesional: p.id, api };
}

test('turno manual, sobreturno, nota y cambios de estado', async ({ page, request }) => {
	await armar(page, request);
	const d = manana();

	await irA(page, `/panel/turnos/nuevo?fecha=${d}`);
	await expect(page.getByRole('radio', { name: /09:00/ })).toBeChecked();
	await page.getByLabel('Nombre').fill('Lucía Pérez');
	await page.getByLabel('Teléfono').fill('1155550001');
	await page.getByRole('button', { name: 'Reservar turno' }).click();
	await expect(page.getByRole('heading', { name: 'Lucía Pérez' })).toBeVisible();
	await expect(page.getByText('el turno quedó reservado')).toBeVisible();
	// Lo que carga el panel nace confirmado.
	await expect(page.getByText('Confirmado', { exact: true })).toBeVisible();

	await page.getByLabel('Nota', { exact: true }).fill('Trae a su hijo');
	await page.getByRole('button', { name: 'Guardar nota' }).click();
	await expect(page.getByText('Guardada')).toBeVisible();

	// Sobreturno a la misma hora con la misma profesional.
	await irA(page, `/panel/turnos/nuevo?fecha=${d}`);
	await page.getByLabel(/Sobreturno/).check();
	await page.getByLabel('Profesional').last().selectOption({ label: 'Ana' });
	await page.getByLabel('Hora', { exact: true }).fill('09:00');
	await page.getByLabel('Nombre').fill('Marcos Gil');
	await page.getByLabel('Teléfono').fill('1155550002');
	await page.getByRole('button', { name: 'Reservar turno' }).click();
	await expect(page.getByRole('heading', { name: 'Marcos Gil' })).toBeVisible();
	// Cancelarlo desde el detalle (pide confirmación).
	page.once('dialog', (d) => d.accept());
	await page.getByRole('button', { name: 'Cancelar' }).click();
	await expect(page.getByText('Cancelado', { exact: true })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Reprogramar' })).toBeHidden();

	// En el listado, los dos; atendido desde la acción rápida.
	await irA(page, `/panel/turnos?desde=${d}&hasta=${d}`);
	const lucia = page.locator('li.turno', { hasText: 'Lucía Pérez' });
	await expect(lucia.getByText('Trae a su hijo')).toBeVisible();
	await expect(page.locator('li.turno', { hasText: 'Marcos Gil' }).getByText('Cancelado', { exact: true })).toBeVisible();
	await lucia.getByRole('button', { name: 'Atendido: Lucía Pérez' }).click();
	await expect(lucia.getByText('Atendido', { exact: true })).toBeVisible();

	// Filtrado por estado.
	await irA(page, `/panel/turnos?desde=${d}&hasta=${d}&estado=atendido`);
	await expect(page.locator('li.turno')).toHaveCount(1);
});

test('ficha del cliente con nota y etiqueta', async ({ page, request }) => {
	const a = await armar(page, request);
	await a.api('POST', '/v1/panel/turnos', {
		servicioId: a.servicio,
		profesionalId: a.profesional,
		inicio: `${manana()}T10:00:00-03:00`,
		cliente: { nombre: 'Rocío Díaz', telefono: '1155550003' }
	});

	await irA(page, '/panel/etiquetas');
	await page.getByLabel('Nombre').fill('VIP');
	await page.getByRole('button', { name: 'Crear etiqueta' }).click();
	await expect(page.locator('.lista').getByText('VIP')).toBeVisible();

	await irA(page, '/panel/clientes');
	await page.getByLabel('Buscar', { exact: true }).fill('Rocío');
	await page.getByRole('button', { name: 'Buscar' }).click();
	await page.getByRole('link', { name: 'Rocío Díaz' }).click();
	await expect(page.getByRole('heading', { name: 'Rocío Díaz' })).toBeVisible();
	await expect(page.getByText('Historial de turnos')).toBeVisible();
	await expect(page.locator('.historial li')).toHaveCount(1);

	await page.getByLabel('Nota nueva').fill('Prefiere a la tarde');
	await page.getByRole('button', { name: 'Agregar nota' }).click();
	await expect(page.locator('.notas').getByText('Prefiere a la tarde')).toBeVisible();

	await page.getByLabel('Etiqueta para agregar').selectOption({ label: 'VIP' });
	await page.getByRole('button', { name: 'Poner etiqueta' }).click();
	await expect(page.getByRole('button', { name: 'Quitar VIP' })).toBeVisible();

	// Penalidad a mano y levantarla.
	await page.getByRole('button', { name: 'Penalizar' }).click();
	await expect(page.getByText(/No puede reservar online/)).toBeVisible();
	await page.getByRole('button', { name: 'Levantar penalidad' }).click();
	await expect(page.getByText('Sin penalidad vigente.')).toBeVisible();

	// El listado la encuentra por la etiqueta; la búsqueda de la barra, por el teléfono.
	await irA(page, '/panel/clientes');
	await page.locator('select[name=etiqueta]').selectOption({ label: 'VIP' });
	await page.getByRole('button', { name: 'Buscar' }).click();
	await expect(page.getByRole('link', { name: 'Rocío Díaz' })).toBeVisible();
	await expect(page.getByText('1 cliente', { exact: true })).toBeVisible();

	await irA(page, '/panel/buscar?q=55550003');
	await expect(page.getByRole('link', { name: 'Rocío Díaz' })).toBeVisible();
});

test('el calendario y la campanita se actualizan solos con una reserva online', async ({ page, request }) => {
	const a = await armar(page, request);
	const d = manana();
	await irA(page, `/panel/calendario?fecha=${d}`);
	await expect(page.getByRole('columnheader', { name: 'Ana' })).toBeVisible();
	await expect(page.getByText('0 turnos')).toBeVisible();
	// Que el tiempo real ya esté escuchando antes de reservar.
	await page.waitForTimeout(1500);

	const r = await request.post(`${API}/v1/p/${a.slug}/turnos`, {
		data: { servicioId: a.servicio, profesionalId: a.profesional, inicio: `${d}T11:00:00-03:00`, cliente: { nombre: 'Cliente Web', telefono: '1155550004' } }
	});
	expect(r.status(), await r.text()).toBe(201);

	const bloque = page.locator('.bloque', { hasText: 'Cliente Web' });
	await expect(bloque).toBeVisible({ timeout: 15_000 });
	await expect(bloque).toContainText('11:00');
	await expect(page.getByRole('link', { name: /Notificaciones, \d+ sin ver/ })).toBeVisible({ timeout: 15_000 });

	// Vista de semana con el mismo turno; y la campanita lo lista y se marca vista.
	await irA(page, `/panel/calendario?vista=semana&fecha=${d}`);
	await expect(page.locator('.bloque', { hasText: 'Cliente Web' })).toBeVisible();
	await irA(page, '/panel/notificaciones');
	await expect(page.locator('li.nueva').first()).toBeVisible();
	const vista = page.waitForResponse((r) => r.request().method() === 'POST' && r.url().includes('/panel/notificaciones'));
	await page.getByRole('button', { name: 'Marcar todas como vistas' }).click();
	expect((await vista).ok()).toBeTruthy();
	await expect(page.getByText('Estás al día.')).toBeVisible({ timeout: 15_000 });
	await expect(page.getByRole('link', { name: 'Notificaciones', exact: true })).toBeVisible();
});

test('reportes con descarga CSV, lista de espera y reseñas', async ({ page, request }) => {
	const a = await armar(page, request);
	await irA(page, '/panel/reportes');
	await expect(page.getByRole('heading', { name: 'Por profesional' })).toBeVisible();
	await expect(page.getByText('Ticket promedio')).toBeVisible();
	const csv = await page.request.get('/panel/reportes/csv/turnos');
	expect(csv.status()).toBe(200);
	expect(csv.headers()['content-type']).toContain('text/csv');
	expect((await page.request.get('/panel/reportes/csv/otra')).status()).toBe(404);

	await irA(page, '/panel/lista-espera');
	await expect(page.getByText('No hay nadie esperando.')).toBeVisible();
	const d = manana();
	const r = await request.post(`${API}/v1/p/${a.slug}/espera`, {
		data: { servicioId: a.servicio, nombre: 'Juana Ríos', email: 'juana@example.com', desde: d, hasta: d }
	});
	expect(r.status(), await r.text()).toBe(201);
	await irA(page, '/panel/lista-espera');
	const anotado = page.locator('[data-anotado]', { hasText: 'Juana Ríos' });
	await expect(anotado.getByLabel('Horario sugerido')).toContainText('09:00');
	await anotado.getByRole('button', { name: 'Avisar' }).click();
	await expect(page.getByText('No hay nadie esperando.')).toBeVisible();
	await irA(page, '/panel/lista-espera?todos=1');
	await expect(page.locator('[data-anotado]', { hasText: 'Juana Ríos' }).getByText(/Avisado/)).toBeVisible();
	await irA(page, '/panel/resenas');
	await expect(page.getByRole('heading', { name: 'Reseñas' })).toBeVisible();
	await irA(page, '/panel/penalidades');
	await expect(page.getByText('Reglas de ausencias')).toBeVisible();

	// A 375 px ninguna pantalla se corre de costado: el calendario se desplaza adentro de su marco.
	await page.setViewportSize({ width: 375, height: 812 });
	for (const ruta of ['/panel/turnos', '/panel/turnos/nuevo', `/panel/calendario?fecha=${d}`, '/panel/calendario?vista=semana', '/panel/clientes', '/panel/reportes', '/panel/notificaciones', '/panel/buscar?q=ju']) {
		await irA(page, ruta);
		const ancho = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
		expect(ancho, ruta).toBeLessThanOrEqual(0);
	}
});

test('los turnos en Google Calendar: link para suscribirse y regenerarlo', async ({ page, request }) => {
	const a = await armar(page, request);
	await a.api('POST', '/v1/panel/turnos', {
		servicioId: a.servicio,
		profesionalId: a.profesional,
		inicio: `${manana()}T10:00:00-03:00`,
		cliente: { nombre: 'Lucía Calendario', telefono: '1155550009' }
	});
	await irA(page, '/panel/calendario');
	await page.getByRole('link', { name: 'Verlo en Google Calendar' }).click();
	await expect(page.getByRole('heading', { name: 'Verlo en tu calendario' })).toBeVisible();
	const negocio = page.getByRole('region', { name: 'Todo el negocio' });
	await negocio.getByRole('button', { name: 'Crear link' }).click();
	const link = negocio.getByLabel(/Link del calendario/);
	await expect(link).toHaveValue(/\/cal\/[\w-]+\.ics$/);
	const viejo = await link.inputValue();
	await expect(negocio.getByRole('link', { name: 'Agregar a Google Calendar' })).toHaveAttribute('href', /calendar\.google\.com.*cid=webcal/);

	const ics = await request.get(viejo);
	expect(ics.headers()['content-type']).toContain('text/calendar');
	const cuerpo = await ics.text();
	expect(cuerpo).toContain('BEGIN:VCALENDAR');
	expect(cuerpo).toContain('Corte · Lucía Calendario');

	// En el teléfono, sin scroll de costado.
	const [ancho, vista] = await page.evaluate(() => [document.documentElement.scrollWidth, window.innerWidth]);
	expect(ancho).toBeLessThanOrEqual(vista);

	await negocio.getByRole('button', { name: 'Generar link nuevo' }).click();
	await expect(negocio.getByRole('status')).toContainText('el link anterior dejó de andar');
	await expect(link).not.toHaveValue(viejo);
	expect((await request.get(viejo)).status()).toBe(404);
	await expect(page.getByRole('region', { name: 'Ana' }).getByRole('button', { name: 'Crear link' })).toBeVisible();
});
