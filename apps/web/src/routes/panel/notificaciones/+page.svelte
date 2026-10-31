<script lang="ts">
	import { enhance } from '$app/forms';
	import { fechaHora } from '#lib/formato.ts';
	import Pestanias from '#lib/operacion/Pestanias.svelte';
	import { ATENCION, NOTIFICACIONES } from '#lib/operacion/secciones.ts';

	let { data, form } = $props();
	const verEnvios = $derived(ATENCION.includes(data.negocio.rol));
</script>

<svelte:head><title>Notificaciones · Panel</title></svelte:head>

<div class="pila">
	<div class="fila" style="justify-content: space-between">
		<h1>Notificaciones</h1>
		{#if data.sinLeer > 0}
			<form method="POST" action="?/vista" use:enhance>
				<button class="boton">Marcar todas como vistas</button>
			</form>
		{/if}
	</div>
	{#if verEnvios}<Pestanias items={NOTIFICACIONES} etiqueta="Notificaciones" />{/if}
	{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}

	<p class="suave chico" aria-live="polite">
		{data.sinLeer === 0 ? 'Estás al día.' : `${data.sinLeer} sin ver.`} Se muestran los últimos siete&nbsp;días.
	</p>

	{#if data.notificaciones.length === 0}
		<p class="tarjeta suave">No pasó nada esta semana.</p>
	{:else}
		<ul class="tarjeta lista">
			{#each data.notificaciones as n (n.id)}
				<li class:nueva={n.nueva}>
					<div class="fila" style="justify-content: space-between">
						<strong>{#if n.nueva}<span class="visualmente-oculto">Nueva: </span>{/if}{n.titulo}</strong>
						<span class="suave chico">{fechaHora(n.creadoEn)}</span>
					</div>
					{#if n.texto}<p class="chico">{n.texto}</p>{/if}
					{#if n.turnoId}<a class="chico" href={`/panel/turnos/${n.turnoId}`}>Ver el turno</a>{/if}
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
		gap: 0;
		padding: 0;
	}
	li {
		padding: 12px 16px;
		border-left: 3px solid transparent;
	}
	li + li {
		border-top: 1px solid var(--borde);
	}
	li.nueva {
		border-left-color: var(--marca);
		background: var(--marca-suave);
	}
	p {
		margin: 4px 0;
		overflow-wrap: anywhere;
	}
</style>
