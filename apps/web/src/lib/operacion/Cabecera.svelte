<!-- Encabezado de pantalla del panel (handoff v2): rótulo de la sección, título grande y acciones. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import Icono from '#lib/ui/Icono.svelte';
	let {
		titulo,
		rotulo,
		volver,
		children
	}: { titulo: string; rotulo?: string; volver?: { href: string; texto: string }; children?: Snippet } = $props();
</script>

{#if volver}<p class="volver"><a href={volver.href}><Icono nombre="arrow_back" tam={18} />{volver.texto}</a></p>{/if}
<div class="cabecera">
	<span class="tit">
		{#if rotulo}<span class="tl-rotulo rot">{rotulo}</span>{/if}
		<h1>{titulo}</h1>
	</span>
	{#if children}<span class="crece"></span>{@render children()}{/if}
</div>

<style>
	.volver {
		margin: 0 0 -8px;
	}
	.volver a {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		min-height: 40px;
		font-size: 14px;
		font-weight: 700;
		color: var(--tl-menta-texto);
		text-decoration: none;
	}
	.cabecera {
		display: flex;
		align-items: flex-end;
		flex-wrap: wrap;
		gap: 14px 16px;
	}
	.tit {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	.rot {
		font-size: 11px;
		color: var(--tl-menta-texto);
	}
	h1 {
		margin: 0;
		font-size: 50px;
		line-height: 0.95;
		font-weight: 700;
		letter-spacing: -0.055em;
	}
	.crece {
		flex: 1;
	}
	@media (max-width: 859px) {
		h1 {
			font-size: 34px;
		}
	}
</style>
