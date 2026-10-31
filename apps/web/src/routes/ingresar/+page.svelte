<script lang="ts">
	import { enhance } from '$app/forms';
	import { iniciar, ErrorClerk, type ProveedorClerk } from '#lib/clerk.ts';

	let { data, form } = $props();
	let error = $state('');
	let yendo = $state<ProveedorClerk | null>(null);

	const mensajes: Record<string, string> = {
		cancelado: 'Cancelaste el ingreso.',
		token_invalido: 'No pudimos verificar tu ingreso. Probá de nuevo.',
		sin_email: 'Tu cuenta no tiene un email verificado.',
		clerk_no_responde: 'El ingreso no responde. Probá en un rato.'
	};

	async function con(p: ProveedorClerk) {
		if (!data.fapi) return;
		yendo = p;
		error = '';
		try {
			location.href = await iniciar(data.fapi, p, `${location.origin}/ingresar/clerk`);
		} catch (e) {
			error = mensajes[e instanceof ErrorClerk ? e.codigo : ''] ?? mensajes.clerk_no_responde;
			yendo = null;
		}
	}
</script>

<svelte:head><title>Ingresar · Tengo Libre</title></svelte:head>

<main class="contenedor pila" style="max-width: 420px">
	<h1>Entrá a tu panel</h1>
	{#if data.error || error}
		<p class="aviso error" role="alert">{error || mensajes[data.error ?? ''] || 'No pudimos completar el ingreso.'}</p>
	{/if}
	{#if data.fapi}
		<button class="boton" disabled={!!yendo} onclick={() => con('google')}>Seguir con Google</button>
		<button class="boton" disabled={!!yendo} onclick={() => con('apple')}>Seguir con Apple</button>
	{:else}
		<p class="suave">El ingreso con Google y Apple no está configurado (falta PUBLIC_CLERK_PUBLICABLE).</p>
	{/if}

	{#if data.prueba}
		<form method="POST" action="?/prueba" use:enhance class="tarjeta pila">
			<h2>Ingreso de prueba</h2>
			<p class="suave chico">Sólo en desarrollo: entra como esa persona, sin Clerk.</p>
			<label class="campo">
				Id de la persona
				<input name="id" value="user_prueba00001" autocomplete="off" />
			</label>
			{#if form?.error}<p class="campo"><span class="error">{form.error}</span></p>{/if}
			<button class="boton primario">Entrar</button>
		</form>
	{/if}
</main>
