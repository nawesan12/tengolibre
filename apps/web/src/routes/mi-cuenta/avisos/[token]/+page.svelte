<script lang="ts">
	import { enhance } from '$app/forms';

	let { data, form } = $props();
	const p = $derived(data.preferencias);
</script>

<svelte:head>
	<title>Tus avisos de {p.negocio} · Tengo Libre</title>
	<meta name="robots" content="noindex, nofollow" />
	<meta name="referrer" content="no-referrer" />
</svelte:head>

<main class="contenedor pila" style="max-width: 480px">
	<a class="suave chico" href={`/${p.slug}`}>{p.negocio}</a>
	<h1>Tus avisos por&nbsp;mail</h1>
	<p class="suave">Hola {p.cliente}: elegí qué mails querés recibir de {p.negocio}.</p>
	{#if form && 'guardado' in form}<p class="aviso" role="status">Guardamos tus preferencias.</p>{/if}
	{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}
	<form method="POST" use:enhance={() => async ({ update }) => update({ reset: false })} class="tarjeta pila">
		<label class="fila opcion">
			<input type="checkbox" name="email" checked={p.email} />
			<span><strong>Avisos de mis turnos</strong><br /><span class="suave chico">Confirmaciones, cambios y&nbsp;cancelaciones.</span></span>
		</label>
		<label class="fila opcion">
			<input type="checkbox" name="recordatorios" checked={p.recordatorios} />
			<span><strong>Recordatorios</strong><br /><span class="suave chico">Un mail antes de cada&nbsp;turno.</span></span>
		</label>
		<div class="fila">
			<button class="boton primario">Guardar</button>
			<button class="boton" name="nada" value="1">No quiero ningún mail</button>
		</div>
	</form>
</main>

<style>
	.opcion {
		flex-wrap: nowrap;
		align-items: flex-start;
		gap: 10px;
	}
	.opcion input {
		width: 20px;
		height: 20px;
		margin-top: 2px;
		accent-color: var(--marca);
		flex: none;
	}
</style>
