<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { page } from '$app/state';
	import { completar, ErrorClerk } from '#lib/clerk.ts';

	// Termina el ingreso en el navegador y postea el token de Clerk al servidor.
	let token = $state('');
	let error = $state('');
	let form: HTMLFormElement;

	onMount(async () => {
		try {
			token = await completar(page.url.searchParams.get('rotating_token_nonce') ?? '');
		} catch (e) {
			error = e instanceof ErrorClerk ? e.codigo.replace(/[^a-z_]/g, '').slice(0, 40) : '';
			error ||= 'clerk_no_responde';
		}
		await tick();
		form.requestSubmit();
	});
</script>

<svelte:head><title>Ingresando · Tengo Libre</title><meta name="robots" content="noindex" /></svelte:head>

<main class="contenedor pila" role="status" aria-live="polite">
	<h1>Entrando a tu cuenta</h1>
	<p class="suave">Un segundo, estamos terminando de validar tu ingreso.</p>
	<form method="POST" bind:this={form} hidden>
		<input type="hidden" name="token" value={token} />
		<input type="hidden" name="error" value={error} />
	</form>
</main>
