<script lang="ts">
	import { enhance } from '$app/forms';
	import Pestanias from '#lib/operacion/Pestanias.svelte';
	import { CLIENTES, GESTION } from '#lib/operacion/secciones.ts';

	let { data, form } = $props();
	const gestiona = $derived(GESTION.includes(data.negocio.rol));
	let editando = $state<string | null>(null);
</script>

<svelte:head><title>Etiquetas · Clientes · Panel</title></svelte:head>

<div class="pila">
	<h1>Etiquetas</h1>
	<Pestanias items={CLIENTES} etiqueta="Clientes" />
	<p class="suave">Para agrupar clientes (frecuentes, VIP, a reactivar) y filtrarlos en el&nbsp;listado.</p>
	{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}

	{#if gestiona}
		<form method="POST" action="?/crear" use:enhance class="tarjeta fila" aria-label="Nueva etiqueta">
			<label class="campo">Nombre <input name="nombre" required maxlength="40" /></label>
			<label class="campo">Color <input name="color" type="color" value="#0F9D58" /></label>
			<button class="boton primario" style="align-self: end">Crear etiqueta</button>
		</form>
	{/if}

	{#if data.etiquetas.length === 0}
		<p class="tarjeta suave">Todavía no hay etiquetas.</p>
	{:else}
		<ul class="tarjeta lista">
			{#each data.etiquetas as e (e.id)}
				<li>
					{#if editando === e.id}
						<form
							method="POST"
							action="?/editar"
							class="fila"
							use:enhance={() =>
								async ({ result, update }) => {
									await update();
									if (result.type === 'success') editando = null;
								}}
						>
							<input type="hidden" name="id" value={e.id} />
							<label class="campo">Nombre <input name="nombre" value={e.nombre} required maxlength="40" /></label>
							<label class="campo">Color <input name="color" type="color" value={e.color} /></label>
							<div class="fila" style="align-self: end">
								<button class="boton primario">Guardar</button>
								<button type="button" class="boton" onclick={() => (editando = null)}>Cancelar</button>
							</div>
						</form>
					{:else}
						<div class="fila" style="justify-content: space-between">
							<span class="fila">
								<span class="etiqueta" style={`background: ${e.color}22; color: ${e.color}`}>{e.nombre}</span>
								<a class="chico suave" href={`/panel/clientes?etiqueta=${e.id}`}>{e.clientes}&nbsp;{e.clientes === 1 ? 'cliente' : 'clientes'}</a>
							</span>
							{#if gestiona}
								<span class="fila">
									<button class="boton chico" onclick={() => (editando = e.id)}>Editar</button>
									<form
										method="POST"
										action="?/borrar"
										use:enhance={({ cancel }) => {
											if (!confirm(`¿Borrar la etiqueta ${e.nombre}? Se les saca a todos los clientes.`)) cancel();
										}}
									>
										<input type="hidden" name="id" value={e.id} />
										<button class="boton chico peligro" aria-label={`Borrar ${e.nombre}`}>Borrar</button>
									</form>
								</span>
							{/if}
						</div>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
</div>

<style>
	.lista {
		list-style: none;
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.lista li + li {
		border-top: 1px solid var(--borde);
		padding-top: 10px;
	}
	input[type='color'] {
		width: 56px;
		padding: 4px;
	}
</style>
