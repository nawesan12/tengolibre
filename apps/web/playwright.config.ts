import { defineConfig, devices } from '@playwright/test';

/**
 * E2E contra la API Go de verdad (Postgres de infra/compose.dev.yml, con las migraciones al día) y el
 * front en desarrollo. Se entra con el ingreso de prueba: la API sin Clerk acepta "dev:user_…".
 * Puertos propios (E2E_API_PUERTO y E2E_WEB_PUERTO; por defecto 4290 y 5193) para no chocar con una
 * API o un front que ya estén corriendo, ni con otra corrida en paralelo.
 */
const PA = process.env.E2E_API_PUERTO ?? '4290';
const PW = process.env.E2E_WEB_PUERTO ?? '5193';
const API = `http://127.0.0.1:${PA}`;
const WEB = `http://127.0.0.1:${PW}`;

export default defineConfig({
	testDir: 'e2e',
	fullyParallel: false,
	// Con la máquina cargada (agentes, builds) 5 s se quedan cortos.
	expect: { timeout: 10_000 },
	// Un reintento: la máquina la comparten varias sesiones y a veces está muy cargada. Lo que pasa al
	// segundo intento figura como `flaky` en el reporte, no se esconde.
	retries: 1,
	workers: 1,
	use: { baseURL: WEB, trace: 'retain-on-failure' },
	projects: [
		{ name: 'escritorio', use: { ...devices['Desktop Chrome'] } },
		{ name: 'telefono', use: { ...devices['iPhone 13'], defaultBrowserType: 'chromium' } }
	],
	webServer: [
		{
			command: 'cd ../api && go run ./cmd/tengolibrectl migrar && go run ./cmd/api',
			url: `${API}/salud/listo`,
			// TENGOLIBRE_ADMINS: una persona de prueba fija del equipo, para probar /admin (e2e/cuenta.spec.ts).
			env: {
				TENGOLIBRE_DIRECCION: `:${PA}`,
				TENGOLIBRE_ENTORNO: 'desarrollo',
				TENGOLIBRE_SITIO: WEB,
				TENGOLIBRE_ADMINS: 'user_e2eadmin00000001'
			},
			reuseExistingServer: !process.env.CI,
			timeout: 180_000
		},
		{
			command: `bunx vite dev --port ${PW} --strictPort --host 127.0.0.1`,
			url: WEB,
			env: { TENGOLIBRE_API: API, TENGOLIBRE_INGRESO_PRUEBA: '1' },
			reuseExistingServer: !process.env.CI,
			timeout: 120_000
		}
	]
});
