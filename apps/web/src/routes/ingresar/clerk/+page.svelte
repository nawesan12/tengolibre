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

<div class="vuelta tl-petroleo">
	<img class="marca-agua" src="/marca/simbolo_una_tinta_blanco.svg" alt="" aria-hidden="true" />
	<main class="cuerpo" role="status" aria-live="polite">
		<img class="icono-app" src="/marca-icono.png" alt="" width="56" height="56" />
		<h1>Entrando a tu&nbsp;cuenta</h1>
		<p>Un segundo, estamos terminando de validar tu&nbsp;ingreso.</p>
		<div class="tl-hueco barra" aria-hidden="true"></div>
		<form method="POST" bind:this={form} hidden>
			<input type="hidden" name="token" value={token} />
			<input type="hidden" name="error" value={error} />
		</form>
	</main>
</div>

<style>
	.vuelta {
		position: relative;
		overflow: hidden;
		min-height: 100dvh;
		display: grid;
		place-items: center;
		padding: 48px 24px;
	}
	.marca-agua {
		position: absolute;
		left: -120px;
		bottom: -120px;
		width: 420px;
		height: 420px;
		opacity: 0.04;
		pointer-events: none;
	}
	.cuerpo {
		position: relative;
		width: min(100%, 420px);
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	.icono-app {
		border-radius: 15px;
	}
	h1 {
		margin: 8px 0 0;
		font-size: 36px;
		line-height: 1;
		font-weight: 700;
		letter-spacing: -0.05em;
	}
	p {
		margin: 0;
		font-size: 15px;
		color: rgba(255, 255, 255, 0.75);
	}
	.barra {
		height: 12px;
		border-radius: 999px;
		margin-top: 8px;
	}
</style>
