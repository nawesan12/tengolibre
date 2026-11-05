<script lang="ts">
	import type { Snippet } from 'svelte';
	let { titulo, actualizado, children }: { titulo: string; actualizado: string; children: Snippet } = $props();
</script>

<svelte:head><title>{titulo} · Tengo Libre</title></svelte:head>

<main class="legal">
	<a class="marca" href="/">Tengo Libre</a>
	<h1>{titulo}</h1>
	<p class="fecha">Actualizado el {actualizado}</p>
	{@render children()}
	<p class="contacto">¿Dudas? Escribinos a <a href="mailto:hola@tengolibre.com.ar">hola@tengolibre.com.ar</a>.</p>
</main>

<style>
	.legal {
		max-width: 720px;
		margin: 0 auto;
		padding: 40px 20px 80px;
		font: 400 17px/1.6 var(--tl-font);
		color: var(--tl-tinta);
		background: var(--tl-papel);
		min-height: 100vh;
	}
	.marca {
		font-weight: var(--tl-w-black);
		color: var(--tl-petroleo);
		text-decoration: none;
	}
	h1 {
		font-weight: var(--tl-w-black);
		font-size: clamp(32px, 6vw, 48px);
		letter-spacing: -0.03em;
		line-height: 1.05;
		margin: 24px 0 6px;
		text-wrap: balance;
	}
	.fecha {
		color: var(--tl-sub);
		margin: 0 0 28px;
	}
	.legal :global(h2) {
		font-weight: var(--tl-w-black);
		font-size: 22px;
		margin: 32px 0 8px;
	}
	.legal :global(p),
	.legal :global(li) {
		color: var(--tl-sub-2);
		text-wrap: pretty;
	}
	.legal :global(a) {
		color: var(--tl-menta-texto);
	}
	.contacto {
		margin-top: 40px;
		padding-top: 20px;
		border-top: 1px solid var(--tl-linea);
	}
</style>
