<script lang="ts">
	import { enhance } from '$app/forms';
	let { form } = $props();
	let nombre = $state('');
	let slug = $state('');
	let tocado = $state(false);
	const aSlug = (s: string) =>
		s.normalize('NFD').replace(/[̀-ͯ]/g, '').toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '').slice(0, 48);
	$effect(() => {
		if (!tocado) slug = aSlug(nombre);
	});
</script>

<svelte:head><title>Empezar · Tengo Libre</title></svelte:head>

<main class="contenedor pila" style="max-width: 480px">
	<h1>Creá tu negocio</h1>
	<p class="suave">Tenés 7 días gratis con todo. Después elegís un plan.</p>
	<form method="POST" use:enhance class="tarjeta pila">
		<label class="campo">
			Nombre del negocio
			<input name="nombre" bind:value={nombre} required maxlength="80" />
		</label>
		<label class="campo">
			Tu dirección: tengolibre.com.ar/<strong>{slug || '…'}</strong>
			<input name="slug" bind:value={slug} oninput={() => (tocado = true)} required pattern="[a-z0-9-]+" maxlength="48" />
			{#if form?.campos?.slug}<span class="error">{form.campos.slug}</span>{/if}
		</label>
		{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}
		<button class="boton primario">Crear</button>
	</form>
</main>
