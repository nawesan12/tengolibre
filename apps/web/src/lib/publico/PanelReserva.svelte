<!--
	El panel «Reservá tu turno» (handoff v2): cabecera con el color del negocio, los pasos uno abajo
	del otro y la barra con el resumen y el botón. Va dentro de PaginaNegocio (usa sus variables
	--w-* y su contenedor: en el teléfono ocupa la pantalla, en escritorio es la columna sticky).
	Los pasos usan las clases globales de acá: .paso, .paso-cab, .paso-n, .opciones, .opcion,
	.barra-texto, .boton-reserva.
-->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import { onMount } from 'svelte';
	import Icono from '#lib/ui/Icono.svelte';
	import { diaCorto } from './calendario.ts';

	let {
		primer = null,
		volver,
		nombre = '',
		children,
		barra
	}: {
		primer?: { fecha: string; hora: string } | null;
		volver: string;
		nombre?: string;
		children: Snippet;
		barra?: Snippet;
	} = $props();

	// Sin conexión: la banda avisa que lo elegido queda (está en la URL y en el borrador).
	let enLinea = $state(true);
	onMount(() => {
		const ver = () => (enLinea = navigator.onLine);
		ver();
		addEventListener('online', ver);
		addEventListener('offline', ver);
		return () => {
			removeEventListener('online', ver);
			removeEventListener('offline', ver);
		};
	});
</script>

<section class="panel" aria-label="Reservar">
	<div class="cabecera">
		<a class="atras" href={volver} aria-label={nombre ? `Volver a ${nombre}` : 'Volver'}><Icono nombre="arrow_back" tam={24} /></a>
		<span class="cab-texto">
			<b>Reservá tu turno</b>
			{#if primer}<span class="tl-num">PRIMER LUGAR · {diaCorto(primer.fecha)} · {primer.hora}</span>{/if}
		</span>
	</div>
	{#if !enLinea}
		<div class="banda" role="status"><Icono nombre="cloud_off" tam={20} lleno />Sin conexión. Lo que elegiste queda&nbsp;guardado.</div>
	{/if}
	<div class="pasos">
		{@render children()}
	</div>
	{#if barra}
		<div class="barra">{@render barra()}</div>
	{/if}
</section>

<style>
	.panel {
		display: flex;
		flex-direction: column;
		background: var(--w-papel);
		min-height: 100vh;
		color: var(--w-ink);
	}
	.cabecera {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 16px 18px;
		background-color: var(--w-pri);
		background-image: var(--tl-grano);
		color: var(--w-on);
	}
	.atras {
		width: 48px;
		height: 48px;
		margin-left: -10px;
		border-radius: var(--w-sr);
		color: var(--w-on);
		display: grid;
		place-items: center;
		text-decoration: none;
	}
	.cab-texto {
		flex: 1;
		display: flex;
		flex-direction: column;
		line-height: 1.2;
	}
	.cab-texto b {
		font-size: 19px;
		font-weight: var(--w-tw);
		letter-spacing: var(--w-tl);
	}
	.cab-texto span {
		font: 400 12px var(--tl-font);
		opacity: 0.85;
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
		line-height: 1.35;
	}
	.banda :global(.icono) {
		color: var(--tl-menta);
	}
	.pasos {
		flex: 1;
		display: flex;
		flex-direction: column;
	}
	.barra {
		position: sticky;
		bottom: 0;
		left: 0;
		right: 0;
		z-index: 10;
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px 18px calc(22px + env(safe-area-inset-bottom));
		background: var(--w-papel);
		box-shadow: 0 -1px 0 var(--w-line);
	}

	/* Piezas de los pasos (las arman las páginas). */
	.panel :global(.paso) {
		padding: 16px 18px;
		display: flex;
		flex-direction: column;
		gap: 10px;
		border-bottom: 1px solid var(--w-line);
	}
	.panel :global(.paso:last-child) {
		border-bottom: 0;
		padding-bottom: 20px;
	}
	.panel :global(.paso-cab) {
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.panel :global(.paso-cab h2) {
		flex: 1;
		margin: 0;
		font-size: 16px;
		font-weight: 700;
		letter-spacing: 0;
		line-height: 1.2;
	}
	.panel :global(.paso-n) {
		font: 700 12px var(--tl-font);
		font-variant-numeric: tabular-nums;
		color: var(--w-marca);
	}
	.panel :global(.paso-extra) {
		font: 400 12px var(--tl-font);
		font-variant-numeric: tabular-nums;
		color: var(--w-sub);
	}
	.panel :global(.cambiar) {
		display: inline-flex;
		align-items: center;
		min-height: 40px;
		color: var(--w-marca);
		font-size: 14px;
		font-weight: 700;
		text-underline-offset: 3px;
	}
	.panel :global(.opciones) {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
	}
	.panel :global(.opcion) {
		display: inline-flex;
		align-items: center;
		white-space: nowrap;
		min-height: 44px;
		padding: 0 14px;
		border-radius: var(--w-br);
		background: var(--w-campo);
		box-shadow: inset 0 0 0 1.5px var(--w-line2);
		color: var(--w-ink);
		font-size: 14px;
		font-weight: 700;
		text-decoration: none;
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.panel :global(.opcion:active) {
		transform: scale(0.96);
	}
	.panel :global(.opcion[aria-current]) {
		background: var(--w-pri);
		color: var(--w-on);
		box-shadow: none;
	}
	.panel :global(.barra-texto) {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		line-height: 1.3;
	}
	.panel :global(.barra-texto b) {
		font: 700 15px var(--tl-font);
		font-variant-numeric: tabular-nums;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.panel :global(.barra-texto span) {
		font-size: 12px;
		color: var(--w-sub);
	}
	.panel :global(.boton-reserva) {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-height: 52px;
		padding: 0 22px;
		border-radius: var(--w-br);
		border: 0;
		background: var(--w-pri);
		color: var(--w-on);
		font: 700 16px var(--tl-font);
		cursor: pointer;
		white-space: nowrap;
		box-shadow: 0 12px 24px -12px var(--w-pri);
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.panel :global(.boton-reserva:active) {
		transform: scale(0.96);
	}
	.panel :global(.boton-reserva.apagado),
	.panel :global(.boton-reserva:disabled) {
		background: #ece5d6;
		color: #4a5d61;
		cursor: not-allowed;
		box-shadow: none;
		transform: none;
	}

	@container web (min-width: 900px) {
		.panel {
			min-height: 0;
		}
		.atras {
			display: none;
		}
		.barra {
			padding-bottom: 16px;
		}
	}
	:global(.compacto) .panel {
		min-height: 100vh;
		border-radius: 0;
		box-shadow: none;
	}
	:global(.compacto) .atras {
		display: grid;
	}
	:global(.compacto) .barra {
		position: sticky;
		padding-bottom: 22px;
	}
</style>
