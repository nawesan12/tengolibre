<script lang="ts">
	import { onMount } from 'svelte';
	import { enhance } from '$app/forms';
	import Icono from '#lib/ui/Icono.svelte';
	import Llave from '#lib/cuenta/Llave.svelte';
	import { PREFERENCIAS_NUEVAS } from '#lib/cuenta/contratos.ts';

	let { data, form } = $props();
	const p = $derived(data.preferencias);
	// Los avisos nuevos se muestran sólo si la API ya los manda.
	const nuevos = $derived(PREFERENCIAS_NUEVAS.filter((k) => typeof p[k] === 'boolean'));
	const conNuevos = $derived(nuevos.length > 0);
	let formulario: HTMLFormElement;
	// Con JavaScript cada llave guarda sola; sin él, queda el botón.
	let vivo = $state(false);
	onMount(() => (vivo = true));
	const guardar = () => formulario.requestSubmit();

	const TEXTOS = {
		resena: { titulo: 'Pedido de reseña', detalle: 'Unas horas después del turno' },
		listaEspera: { titulo: 'Lugar libre en la lista de espera', detalle: 'Cuando se libera lo que esperabas' },
		novedades: { titulo: 'Novedades del negocio', detalle: 'Promos y membresías, como mucho 1 por mes' }
	} as const;
</script>

<svelte:head>
	<title>Tus avisos de {p.negocio} · Tengo Libre</title>
	<meta name="robots" content="noindex, nofollow" />
	<meta name="referrer" content="no-referrer" />
</svelte:head>

<main class="avisos">
	<span class="quien tl-num">{(p.correo || p.cliente).toLocaleUpperCase('es-AR')}</span>
	<h1>¿Qué mails querés&nbsp;recibir?</h1>
	<p class="de">Los avisos de <a href={`/${p.slug}`}>{p.negocio}</a>.</p>

	{#if form && 'guardado' in form}
		<p class="estado ok" role="status">
			<Icono nombre="check_circle" lleno tam={20} />
			<span>{form.nada ? `Listo: no te vamos a mandar más mails de ${p.negocio}.` : 'Guardamos tus preferencias.'}</span>
		</p>
	{/if}
	{#if form?.error}
		<p class="estado error" role="alert"><Icono nombre="error" lleno tam={20} /><span>{form.error}</span></p>
	{/if}

	<form method="POST" bind:this={formulario} use:enhance={() => async ({ update }) => update({ reset: false })} class="form">
		<input type="hidden" name="campos" value={nuevos.join(',')} />
		<div class="lista">
			<Llave nombre="recordatorios" titulo="Recordatorio del turno" detalle="Un mail antes de cada turno" checked={p.recordatorios} onchange={guardar} />
			<Llave
				nombre="email"
				titulo="Cambios en tus turnos"
				detalle={conNuevos ? 'Si el negocio lo mueve o lo cancela' : 'La confirmación, y si el negocio lo mueve o lo cancela'}
				checked={p.email}
				onchange={guardar}
			/>
			{#each nuevos as k (k)}
				<Llave nombre={k} titulo={TEXTOS[k].titulo} detalle={TEXTOS[k].detalle} checked={!!p[k]} onchange={guardar} />
			{/each}
		</div>
		{#if conNuevos}
			<p class="nota">La confirmación de un turno nuevo siempre llega: es tu&nbsp;comprobante.</p>
		{/if}
		{#if !vivo}<button class="boton oscuro guardar">Guardar</button>{/if}
		<button class="todo" name="nada" value="1">Dejar de recibir todo</button>
	</form>
</main>

<style>
	.avisos {
		width: min(100%, 440px);
		min-height: 100dvh;
		margin: 0 auto;
		padding: 48px 20px 24px;
		display: flex;
		flex-direction: column;
		gap: 14px;
	}
	.quien {
		font-size: 12px;
		font-weight: 700;
		color: var(--tl-sub);
		overflow-wrap: anywhere;
	}
	h1 {
		margin: 0;
		font-size: 30px;
		line-height: 1.02;
		font-weight: 700;
		letter-spacing: -0.045em;
	}
	.de {
		margin: -6px 0 0;
		font-size: 14px;
		color: var(--tl-sub);
	}
	.de a {
		color: var(--tl-menta-texto);
		font-weight: 700;
	}
	.form {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 14px;
	}
	.lista {
		border-radius: 18px;
		background: var(--tl-papel);
		overflow: hidden;
		box-shadow: var(--tl-sh-tarjeta);
	}
	.nota {
		margin: 0;
		font-size: 13px;
		line-height: 1.45;
		color: var(--tl-sub);
	}
	.estado {
		display: flex;
		gap: 10px;
		margin: 0;
		padding: 14px;
		border-radius: 16px;
		font-size: 14px;
		line-height: 1.45;
		animation: tl-gin var(--tl-entrar) var(--tl-ease) both;
	}
	.estado.ok {
		background: var(--tl-menta-suave);
		color: #0f3d2b;
	}
	.estado.error {
		background: #fbe0dc;
		color: #5b1a13;
	}
	.guardar {
		width: 100%;
		min-height: 48px;
	}
	.todo {
		margin-top: auto;
		height: 48px;
		border: 0;
		background: none;
		color: #a3241a;
		font: 700 15px var(--tl-font);
		cursor: pointer;
		border-radius: 999px;
	}
	.todo:hover {
		background: rgba(163, 36, 26, 0.06);
	}
	@media (min-width: 720px) {
		.avisos {
			min-height: 0;
			padding-top: 64px;
		}
		.todo {
			margin-top: 8px;
		}
	}
</style>
