<script lang="ts">
	import { enhance } from '$app/forms';
	import { ROLES } from '#lib/formato.ts';

	let { data, form } = $props();
	const i = $derived(data.invitacion);
	const error = $derived(form?.error || data.falla);
</script>

<svelte:head>
	<title>Invitación a {i.negocio} · Tengo Libre</title>
	<meta name="robots" content="noindex, nofollow" />
	<meta name="referrer" content="no-referrer" />
</svelte:head>

<main class="contenedor pila" style="max-width: 480px">
	<h1>Te invitaron a {i.negocio}</h1>
	{#if i.aceptada}
		<p>Esta invitación ya se usó.</p>
		<p><a class="boton primario" href="/panel">Ir al panel</a></p>
	{:else if i.vencida}
		<p>Esta invitación venció. Pedile a quien te invitó que te mande&nbsp;otra.</p>
	{:else}
		<p>Vas a sumarte al equipo como <strong>{ROLES[i.rol] ?? i.rol}</strong>, con la cuenta <strong>{i.email}</strong>.</p>
		{#if error}
			<p class="aviso error" role="alert">{error}</p>
			<form method="POST" action="/salir">
				<button class="boton">Salir y entrar con otra cuenta</button>
			</form>
		{/if}
		<form method="POST" use:enhance>
			<button class="boton primario">Aceptar la invitación</button>
		</form>
	{/if}
</main>
