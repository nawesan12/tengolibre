import type { Persona } from '#lib/server/api.ts';

declare global {
	namespace App {
		interface Locals {
			/** Token de la sesión propia de TengoLibre (cookie httpOnly), si hay. */
			token: string | null;
			/** El negocio elegido en el panel (cookie), si hay. */
			negocio: string | null;
		}
		interface PageData {
			persona?: Persona | null;
		}
		interface Error {
			codigo?: string;
		}
	}
}

export {};
