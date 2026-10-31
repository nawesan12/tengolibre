<!--
	Un estado que corta el camino (invitación vencida o usada, error): ícono grande con su color,
	título, explicación y una sola salida en pastilla petróleo (Cuenta y accesos v2, 1b).
-->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import Icono from '#lib/ui/Icono.svelte';

	let {
		icono,
		color = 'var(--tl-menta-texto)',
		titulo,
		nivel = 1,
		children,
		accion
	}: { icono: string; color?: string; titulo: string; nivel?: 1 | 2; children: Snippet; accion?: Snippet } = $props();
</script>

<section class="estado">
	<span class="ic" style:color><Icono nombre={icono} lleno tam={32} /></span>
	<svelte:element this={`h${nivel}`} class="titulo">{titulo}</svelte:element>
	<div class="texto">{@render children()}</div>
	{#if accion}<div class="accion">{@render accion()}</div>{/if}
</section>

<style>
	.estado {
		display: flex;
		flex-direction: column;
		gap: 10px;
		padding: 22px;
		border-radius: 24px;
		background: var(--tl-papel);
		box-shadow: var(--tl-sh-tarjeta);
		animation: tl-gin var(--tl-entrar) var(--tl-ease) both;
	}
	.ic {
		display: inline-flex;
	}
	.titulo {
		margin: 0;
		font-size: 20px;
		line-height: 1.2;
		font-weight: 700;
		letter-spacing: -0.02em;
	}
	.texto {
		font-size: 14px;
		line-height: 1.5;
		color: var(--tl-sub-2);
	}
	.texto :global(p) {
		margin: 0;
	}
	.accion {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		align-items: center;
	}
	.accion :global(.pastilla) {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		min-height: 44px;
		padding: 0 18px;
		border: 0;
		border-radius: 999px;
		background: var(--tl-petroleo);
		color: #fff;
		font: 700 14px var(--tl-font);
		text-decoration: none;
		cursor: pointer;
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.accion :global(.pastilla:active) {
		transform: scale(0.97);
	}
	@media (max-width: 719px) {
		.accion :global(.pastilla) {
			min-height: 48px;
		}
	}
</style>
