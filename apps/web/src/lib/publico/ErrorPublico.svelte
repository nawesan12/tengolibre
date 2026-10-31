<!-- Error de una página pública (negocio, reserva, Mi turno, reseña), con la forma de los estados del
	handoff v2 («Estados y teléfono»): rótulo, título, una línea y una sola acción. -->
<script lang="ts">
	import { page } from '$app/state';
	import Icono from '#lib/ui/Icono.svelte';

	let { titulo = 'No encontramos esta página' }: { titulo?: string } = $props();
	const noEsta = $derived(page.status === 404);
	let enLinea = $state(true);
	$effect(() => {
		enLinea = navigator.onLine;
	});
</script>

<svelte:head><title>{noEsta ? 'No encontramos esta página' : 'Algo salió mal'} · Tengo Libre</title></svelte:head>

<div class="fondo">
	{#if !enLinea}
		<div class="banda" role="status"><Icono nombre="cloud_off" tam={20} lleno />Sin conexión. Cuando vuelva, probá de&nbsp;nuevo.</div>
	{/if}
	<main class="estado">
		<span class="kicker" class:rojo={!noEsta}>{noEsta ? 'NO ENCONTRADO' : `ERROR ${page.status}`}</span>
		<h1>{noEsta ? titulo : 'No pudimos traer esta página'}</h1>
		<p>{noEsta ? page.error?.message || 'Revisá el link.' : 'El servidor no respondió. Probá de nuevo en un momento.'}</p>
		{#if noEsta}
			<a class="cta" href="/explorar">Buscar dónde sacar turno</a>
			<a class="cta-2" href="/">Ir a Tengo Libre</a>
		{:else}
			<a class="cta claro" href={page.url.pathname + page.url.search} data-sveltekit-reload>Reintentar</a>
			<p class="chico">Si sigue igual, escribinos a <a href="mailto:hola@tengolibre.com.ar">hola@tengolibre.com.ar</a></p>
		{/if}
	</main>
</div>

<style>
	.fondo {
		min-height: 100vh;
		background: var(--tl-crema-pagina);
	}
	.banda {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 10px 16px;
		background-color: var(--tl-petroleo-hondo);
		background-image: var(--tl-grano);
		color: #fff;
		font-size: 13px;
	}
	.banda :global(.icono) {
		color: var(--tl-menta);
	}
	.estado {
		max-width: 460px;
		margin: 0 auto;
		padding: 48px 18px;
		display: flex;
		flex-direction: column;
		gap: 14px;
	}
	.kicker {
		font: 700 11px var(--tl-font);
		letter-spacing: 0.08em;
		color: var(--tl-menta-texto);
	}
	.kicker.rojo {
		color: #a3241a;
	}
	h1 {
		margin: 0;
		font-size: 28px;
		line-height: 1.02;
		font-weight: 700;
		letter-spacing: -0.045em;
	}
	p {
		margin: 0;
		font-size: 15px;
		line-height: 1.5;
		color: var(--tl-sub-2);
	}
	.chico {
		font-size: 13px;
	}
	.cta {
		display: grid;
		place-items: center;
		min-height: 54px;
		margin-top: 12px;
		border-radius: 999px;
		background: var(--tl-petroleo);
		color: #fff;
		font-size: 16px;
		font-weight: 700;
		text-decoration: none;
	}
	.cta.claro {
		background: var(--tl-papel);
		color: var(--tl-tinta);
		box-shadow: inset 0 0 0 1.5px var(--tl-linea-2);
	}
	.cta-2 {
		display: grid;
		place-items: center;
		min-height: 44px;
		font-size: 15px;
		font-weight: 700;
		color: var(--tl-sub);
		text-decoration: none;
	}
</style>
