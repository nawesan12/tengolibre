import { defineEnvVars } from '@sveltejs/kit/env';

export const variables = defineEnvVars({
	TENGOLIBRE_API: {
		description: 'La API Go (desde el servidor de SvelteKit). En el VPS: http://127.0.0.1:4200.',
		schema: (v) => (v || 'http://127.0.0.1:4200').replace(/\/$/, '')
	},
	TENGOLIBRE_INGRESO_PRUEBA: {
		description: 'Con "1", /ingresar ofrece entrar como una persona de prueba (sólo local: la API lo rechaza en producción).',
		schema: (v) => v === '1'
	},
	PUBLIC_CLERK_PUBLICABLE: {
		description: 'Clave publicable de Clerk (pk_live_…): de ahí sale la Frontend API para Google y Apple.',
		public: true,
		schema: (v) => v ?? ''
	}
});
