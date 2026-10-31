<!--
	Talón = un turno (firma 1 del handoff): troquel vertical con dos medialunas del color del fondo,
	la hora a la izquierda y el resto a la derecha. Estados: nuevo, encurso, pasado (atendido),
	pendiente (borde punteado), cancelado. `fondo` es el color de lo que hay detrás (para las medialunas).
-->
<script lang="ts">
	import type { Snippet } from 'svelte';
	let {
		hora,
		estado,
		fondo = 'var(--tl-crema-pagina)',
		ancho = 88,
		href,
		children
	}: {
		hora: Snippet | string;
		estado?: 'nuevo' | 'encurso' | 'pasado' | 'pendiente' | 'cancelado';
		fondo?: string;
		ancho?: number;
		href?: string;
		children: Snippet;
	} = $props();
</script>

<svelte:element
	this={href ? 'a' : 'div'}
	{href}
	class="tl-talon talon {estado ? `tl-talon--${estado}` : ''}"
	class:pendiente={estado === 'pendiente'}
	style:--talon-x="{ancho}px"
	style:--talon-fondo={fondo}
>
	<div class="hora">
		{#if typeof hora === 'string'}{hora}{:else}{@render hora()}{/if}
	</div>
	<div class="cuerpo">{@render children()}</div>
</svelte:element>

<style>
	.talon {
		text-decoration: none;
		color: inherit;
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	a.talon:hover {
		transform: translateY(-2px);
	}
	.hora {
		font-size: 20px;
		letter-spacing: -0.02em;
		padding: 0 8px;
	}
	.cuerpo {
		display: flex;
		flex-direction: column;
		justify-content: center;
		gap: 2px;
		padding: 12px 18px;
		min-width: 0;
	}
	.pendiente {
		outline: 2px dashed var(--tl-linea-2);
		outline-offset: -2px;
	}
</style>
