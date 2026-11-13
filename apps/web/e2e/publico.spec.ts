import { expect, hidratada, test, type Page } from './base.ts';
import { unico } from './ayuda.ts';
import { hoy, idPersona, invitar, negocioConAgenda, publica, reservarConAna, socio, sql } from './publico-ayuda.ts';

/** Elige en el calendario el primer día con lugar que no sea hoy (así hay plazo para cancelar). */
async function elegirDia(page: Page, distintoDe = '') {
	const dias = page.locator('a.dia[data-fecha]');
	await expect(dias.first()).toBeVisible();
	const fechas = await dias.evaluateAll((as) => as.map((a) => a.getAttribute('data-fecha') ?? ''));
	const f = fechas.find((x) => x > hoy() && x !== distintoDe)!;
	await page.locator(`a.dia[data-fecha="${f}"]`).click();
	return f;
}

/** En el teléfono no tiene que haber scroll de costado. */
async function sinDesborde(page: Page) {
	const [ancho, vista] = await page.evaluate(() => [document.documentElement.scrollWidth, window.innerWidth]);
	expect(ancho).toBeLessThanOrEqual(vista);
}

test('reserva completa, reprogramar, cancelar y reseña', async ({ page }) => {
	const a = await negocioConAgenda();

	// La página del negocio.
	await page.goto(`/${a.slug}`);
	await expect(page).toHaveTitle(new RegExp(a.nombre));
	await expect(page.getByRole('heading', { name: a.nombre, level: 1 })).toBeVisible();
	await expect(page.getByText('Corte y lavado').first()).toBeVisible();
	await expect(page.getByText('Ana', { exact: true })).toBeVisible();
	await expect(page.getByRole('link', { name: 'Cómo llegar' }).first()).toHaveAttribute('href', /google\.com\/maps/);
	await sinDesborde(page);

	// Servicio → profesional → día → horario → datos.
	await page.getByRole('link', { name: /Corte y lavado/ }).first().click();
	await expect(page.getByRole('heading', { name: /Con.quién/ })).toBeVisible();
	await page.getByRole('link', { name: /Cualquiera/ }).click();
	await expect(page.getByRole('heading', { name: 'Cuándo' })).toBeVisible();
	await sinDesborde(page);
	await elegirDia(page);
	await expect(page.getByText(/horarios? libres?$/)).toBeVisible();
	await page.getByRole('link', { name: '10:00' }).click();

	await expect(page.getByRole('heading', { name: 'Tus datos' })).toBeVisible();
	await sinDesborde(page);
	// El aviso del art. 6 (Ley 25.326) antes de pedir los datos, con el link a la política.
	await expect(page.getByText(/Tengo Libre los guarda y procesa por su cuenta/)).toBeVisible();
	await expect(page.getByRole('link', { name: /política de.privacidad/ })).toHaveAttribute('href', '/privacidad');
	// Un cupón que no existe avisa sin perder lo cargado.
	await page.getByLabel('Nombre y apellido').fill('Carla Gómez');
	await page.getByText('Cupón, nota o código de referido').click();
	await page.getByLabel('Cupón').fill('NOEXISTE');
	await page.getByRole('button', { name: 'Aplicar' }).click();
	await expect(page.getByRole('status')).toContainText(/no encontrado/i);
	await expect(page.getByLabel('Nombre y apellido')).toHaveValue('Carla Gómez');
	await page.getByLabel('Cupón').fill('');

	// Sin teléfono ni email, la API lo frena y el mensaje vuelve al formulario.
	await page.getByRole('button', { name: 'Reservar', exact: true }).click();
	await expect(page.getByRole('alert')).toContainText('teléfono o un email');

	const email = `carla${unico()}@prueba.com`;
	await page.getByLabel('Teléfono', { exact: true }).fill('11 5555-0101');
	await page.getByLabel('Email', { exact: true }).fill(email);
	await page.getByRole('button', { name: 'Reservar', exact: true }).click();

	// El link del turno.
	await expect(page).toHaveURL(/\/turno\/[\w-]+\?nuevo=1/);
	await expect(page.getByText('¡Listo! Guardá este link')).toBeVisible();
	await expect(page.locator('.hora-turno')).toHaveText('10:00');
	await expect(page.getByText(/hasta las 10:30/)).toBeVisible();
	await expect(page.getByText('Con Ana')).toBeVisible();
	await expect(page.locator('meta[name="robots"]')).toHaveAttribute('content', /noindex/);
	const codigo = new URL(page.url()).pathname.split('/').pop()!;

	// Confirmar desde el link (lo mismo que el botón de WhatsApp).
	await page.getByRole('button', { name: 'Confirmo que voy' }).click();
	await expect(page.getByText(/Le avisamos a .* que.venís/)).toBeVisible();
	await expect(page.getByText(/Confirmaste ·/)).toBeVisible();

	// Reprogramar a otro día.
	await page.getByRole('link', { name: 'Cambiar día u hora' }).click();
	await elegirDia(page);
	await page.locator('label.hora-op', { hasText: '11:00' }).click();
	await page.getByRole('button', { name: /^Cambiar a / }).click();
	await expect(page.getByText(/Cambiamos tu turno/)).toBeVisible();
	await expect(page.locator('.hora-turno')).toHaveText('11:00');

	// Cancelar.
	await page.getByRole('link', { name: 'Cancelar', exact: true }).click();
	await page.getByRole('button', { name: 'Sí, cancelar' }).click();
	await expect(page.getByRole('heading', { name: 'Turno cancelado' })).toBeVisible();
	await expect(page.getByRole('link', { name: 'Sacar otro turno' })).toBeVisible();

	// La reseña: el link lo manda el worker por mail; acá se crea el token a mano.
	const tk = `e2e${unico()}${unico()}`.slice(0, 32).padEnd(20, 'x');
	sql(`INSERT INTO resena_tokens (token, negocio_id, turno_id, vence_en)
		SELECT :'p1', negocio_id, id, now() + interval '1 day' FROM turnos WHERE codigo = :'p2'`, tk, codigo);
	await page.goto(`/resena/${tk}`);
	await expect(page.getByRole('heading', { name: '¿Cómo te fue, Carla?' })).toBeVisible();
	await page.getByRole('button', { name: 'Publicar reseña' }).click(); // sin estrellas: lo frena el navegador
	await expect(page.getByRole('heading', { name: '¿Cómo te fue, Carla?' })).toBeVisible();
	await page.locator('label[for="p5"]').click();
	await page.getByLabel('¿Qué te gustó? (opcional)').fill('Excelente atención.');
	await page.getByRole('button', { name: 'Publicar reseña' }).click();
	await expect(page.getByRole('heading', { name: '¡Gracias, Carla!' })).toBeVisible();
	await page.goto(`/resena/${tk}`);
	await expect(page.getByRole('heading', { name: 'Ya dejaste tu reseña' })).toBeVisible();

	// Visible en la página del negocio.
	await page.goto(`/${a.slug}`);
	await expect(page.getByText('Excelente atención.')).toBeVisible();
	await expect(page.getByRole('link', { name: '5 de 5 (1 reseña)' })).toBeVisible();

	// Preferencias de avisos del cliente.
	const pref = `pref${unico()}${unico()}`.padEnd(20, 'x');
	sql(`INSERT INTO preferencias (cliente_id, negocio_id, token)
		SELECT cliente_id, negocio_id, :'p1' FROM turnos WHERE codigo = :'p2' ON CONFLICT (cliente_id) DO UPDATE SET token = excluded.token`, pref, codigo);
	await page.goto(`/mi-cuenta/avisos/${pref}`);
	await expect(page.getByRole('heading', { name: '¿Qué mails querés recibir?' })).toBeVisible();
	await hidratada(page);
	// Cada llave guarda sola al tocarla.
	await page.getByLabel(/Recordatorio del turno/).uncheck();
	await expect(page.getByText('Guardamos tus preferencias.')).toBeVisible();
	await page.reload();
	await expect(page.getByLabel(/Recordatorio del turno/)).not.toBeChecked();
	await expect(page.getByLabel(/Cambios en tus turnos/)).toBeChecked();
});

test('reservar funciona sin JavaScript', async ({ browser }) => {
	const a = await negocioConAgenda();
	const ctx = await browser.newContext({ javaScriptEnabled: false });
	const page = await ctx.newPage();
	await page.goto(`/${a.slug}/reservar?servicio=${a.servicio}`);
	await page.getByRole('link', { name: /^Ana/ }).click();
	await elegirDia(page);
	await page.getByRole('link', { name: '09:30' }).click();
	await page.getByLabel('Nombre y apellido').fill('Diego Sin JS');
	await page.getByLabel('Teléfono', { exact: true }).fill('11 4444-0000');
	await page.getByRole('button', { name: 'Reservar', exact: true }).click();
	await expect(page).toHaveURL(/\/turno\/[\w-]+\?nuevo=1/);
	await expect(page.locator('.hora-turno')).toHaveText('09:30');
	await expect(page.getByText('Con Ana')).toBeVisible();
	await ctx.close();
});

test('sin horarios, la lista de espera', async ({ page }) => {
	const a = await negocioConAgenda();
	// Un día sin lugar: ayer ya pasó, así que /libres viene vacío.
	const ayer = new Date(Date.parse(hoy() + 'T12:00:00Z') - 86_400_000).toISOString().slice(0, 10);
	await page.goto(`/${a.slug}/reservar?servicio=${a.servicio}&profesional=cualquiera&fecha=${ayer}`);
	await expect(page.getByRole('heading', { name: /Ese día ya no hay.lugar/ })).toBeVisible();
	await page.getByText(/^Anotarme para el/).click();
	await page.getByLabel('Nombre', { exact: true }).fill('Elena');
	await page.getByLabel('Email', { exact: true }).fill(`elena${unico()}@prueba.com`);
	await page.getByLabel('Desde').fill(hoy());
	await page.getByLabel('Hasta').fill(hoy());
	await page.getByRole('button', { name: 'Anotarme' }).click();
	await expect(page.getByRole('status')).toContainText('Te anotamos en la lista de espera');
});

test('explorar encuentra el negocio por ciudad y texto', async ({ page }) => {
	const ciudad = `Ciudad${unico()}`;
	const a = await negocioConAgenda({ ciudad });
	await page.goto('/explorar');
	await expect(page.getByRole('heading', { name: /Buscá dónde sacar/ })).toBeVisible();
	await page.getByLabel('Ciudad', { exact: true }).fill(ciudad);
	await page.getByLabel('Buscar', { exact: true }).fill('lavado');
	await page.getByRole('button', { name: 'Buscar' }).click();
	await expect(page).toHaveURL(new RegExp(`ciudad=${ciudad}`));
	await expect(page.getByText('1 negocio')).toBeVisible();
	await page.getByRole('link', { name: new RegExp(a.nombre) }).click();
	await expect(page.getByRole('heading', { name: a.nombre, level: 1 })).toBeVisible();

	await page.goto(`/explorar?ciudad=${ciudad}&q=nada-que-ver`);
	await expect(page.getByText('No encontramos negocios')).toBeVisible();
});

test('un negocio que no existe y un link roto dan 404', async ({ page }) => {
	expect((await page.goto(`/no-existe-${unico()}`))?.status()).toBe(404);
	expect((await page.goto(`/turno/codigoquenoexiste123`))?.status()).toBe(404);
	// Las rutas fijas ganan al comodín del slug.
	await page.goto('/explorar');
	await expect(page.getByRole('heading', { name: /Buscá dónde sacar/ })).toBeVisible();
});

test('el socio ingresa desde la reserva y usa su membresía', async ({ page }) => {
	const a = await negocioConAgenda();
	const persona = idPersona();
	await socio(a, `${persona.slice(5)}@prueba.tengolibre.com.ar`);
	await page.goto(`/${a.slug}/reservar?servicio=${a.servicio}`);
	await page.getByRole('link', { name: /^Ana/ }).click();
	await elegirDia(page);
	await page.getByRole('link', { name: '10:30' }).click();
	await page.getByRole('link', { name: 'Ingresá' }).click();
	await page.getByLabel('Id de la persona').fill(persona);
	await page.getByRole('button', { name: 'Entrar' }).click();
	await expect(page).toHaveURL(/\/reservar\?/);
	await expect(page.getByLabel(/Usar mi membresía/)).toBeChecked();
	await expect(page.getByText('te queda 1 visita')).toBeVisible();
	await expect(page.getByLabel('Nombre y apellido')).toHaveCount(0);
	await sinDesborde(page);
	await page.getByRole('button', { name: 'Reservar', exact: true }).click();
	await expect(page).toHaveURL(/\/turno\/[\w-]+\?nuevo=1/);
	expect(sql(`SELECT count(*) FROM membresia_creditos WHERE negocio_id = :'p1' AND cantidad < 0`, a.negocio)).toBe('1');
});

test('invitación: sin sesión pide ingresar y al volver la acepta', async ({ page }) => {
	const a = await negocioConAgenda();
	const persona = idPersona();
	const email = `${persona.slice(5)}@prueba.tengolibre.com.ar`.toLowerCase();
	await invitar(a, email);
	const tk = sql(`SELECT token FROM invitaciones WHERE email = :'p1'`, email);
	await page.goto(`/invitacion/${tk}`);
	await expect(page).toHaveURL(/\/ingresar\?volver=/);
	await page.getByLabel('Id de la persona').fill(persona);
	await page.getByRole('button', { name: 'Entrar' }).click();
	await expect(page).toHaveURL(/\/panel/);
	// En el teléfono el nombre del negocio va en la barra de arriba, no en el menú lateral.
	await expect(page.locator('[data-negocio-actual]:visible')).toHaveText(a.nombre);
	await page.goto(`/invitacion/${tk}`);
	await expect(page.getByRole('heading', { name: 'Ya la usaste' })).toBeVisible();
});

test('dominio propio: /_host traduce el host al slug', async ({ request }) => {
	const a = await negocioConAgenda();
	const host = `turnos-${unico()}.ejemplo.com`;
	expect(await (await request.get(`/_host?host=${host}`)).json()).toEqual({ slug: null });
	sql(`INSERT INTO dominios (negocio_id, nombre, token, estado, verificado_en) VALUES (:'p1', :'p2', :'p3', 'verificado', now())`,
		a.negocio, host, 'a'.repeat(24));
	expect(await (await request.get(`/_host?host=${host}`)).json()).toEqual({ slug: a.slug });
});

test('se libera un lugar: la persona de la lista lo toma desde su link', async ({ page }) => {
	const a = await negocioConAgenda();
	const t = await reservarConAna(a);
	await publica('POST', `/v1/p/${a.slug}/espera`, {
		servicioId: a.servicio,
		profesionalId: '',
		nombre: 'Martina',
		email: `martina${unico()}@prueba.com`,
		desde: t.fecha,
		hasta: t.fecha,
		momento: 'cualquiera'
	});
	await publica('POST', `/v1/t/${t.codigo}/cancelar`);
	// El link lo manda el mail; acá se lee de la base.
	const tk = sql(`SELECT token FROM oferta_intentos WHERE negocio_id = :'p1' AND estado = 'ofrecida'`, a.negocio);
	await page.goto(`/espera/${tk}`);
	await expect(page.getByText('SE LIBERÓ LO QUE ESPERABAS')).toBeVisible();
	await expect(page.getByText(/Es tuyo si lo tomás en los próximos/)).toBeVisible();
	await sinDesborde(page);
	await page.getByRole('button', { name: 'Tomar el lugar' }).click();
	await expect(page).toHaveURL(/\/turno\/[\w-]+\?nuevo=1/);
	// El talón muestra la referencia para decir en el local; el turno quedó a nombre de Martina.
	const codigo = new URL(page.url()).pathname.split('/').pop() ?? '';
	const id = sql(`SELECT id FROM turnos WHERE codigo = :'p1'`, codigo);
	await expect(page.getByText(id.slice(0, 6).toUpperCase(), { exact: true })).toBeVisible();
	expect(sql(`SELECT c.nombre FROM turnos t JOIN clientes c ON c.id = t.cliente_id WHERE t.codigo = :'p1'`, codigo)).toBe('Martina');
	// El link ya usado lleva a su turno.
	await page.goto(`/espera/${tk}`);
	await expect(page).toHaveURL(/\/turno\/[\w-]+/);
});

test('turno atendido: repetir en un toque', async ({ page }) => {
	const a = await negocioConAgenda();
	const t = await reservarConAna(a, 'Sofía Méndez');
	sql(`UPDATE turnos SET estado = 'atendido' WHERE codigo = :'p1'`, t.codigo);
	await page.goto(`/turno/${t.codigo}`);
	await expect(page.getByRole('heading', { name: /¿Repetimos en \d+ semanas?\?/ })).toBeVisible();
	await expect(page.getByText('Tus datos ya los tenemos.')).toBeVisible();
	await sinDesborde(page);
	await page.getByRole('button', { name: 'Reservar', exact: true }).click();
	await expect(page).toHaveURL(/\/turno\/[\w-]+\?nuevo=1/);
	await expect(page.getByText('¡Listo! Guardá este link')).toBeVisible();
	await expect(page.getByText('Con Ana')).toBeVisible();
});
