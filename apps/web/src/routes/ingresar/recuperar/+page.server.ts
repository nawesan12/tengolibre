import { PUBLIC_CLERK_PUBLICABLE } from '$app/env/public';
import { fapiDe } from '#lib/clerk.ts';
import type { PageServerLoad } from './$types';

/** Recuperar la contraseña: la Frontend API de Clerk y el mail, si vino de /ingresar. */
export const load: PageServerLoad = ({ url }) => ({
	fapi: fapiDe(PUBLIC_CLERK_PUBLICABLE),
	email: (url.searchParams.get('email') ?? '').slice(0, 200)
});
