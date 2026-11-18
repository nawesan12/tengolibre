<!--
	«Tu link» de la barra del panel: desde cualquier pantalla, compartir el link y (quien gestiona)
	cambiarlo. En el teléfono es una hoja desde abajo; en escritorio, una tarjeta centrada.
-->
<script lang="ts">
	import Icono from '#lib/ui/Icono.svelte';
	import TuLink from './TuLink.svelte';
	import EditorLink from './EditorLink.svelte';

	let {
		abierto = $bindable(false),
		nombre,
		slug,
		origen,
		gestiona,
		cambiar = $bindable(false)
	}: { abierto: boolean; nombre: string; slug: string; origen: string; gestiona: boolean; cambiar?: boolean } = $props();

	let dialogo = $state<HTMLDialogElement>();
	$effect(() => {
		if (!dialogo) return;
		if (abierto && !dialogo.open) dialogo.showModal();
		if (!abierto && dialogo.open) dialogo.close();
	});
	const host = $derived(new URL(origen).host);
</script>

<dialog
	bind:this={dialogo}
	class="panel-link"
	aria-label="Tu link"
	onclose={() => {
		abierto = false;
		cambiar = false;
	}}
	onclick={(e) => {
		// Tocar afuera de la hoja la cierra.
		if (e.target === dialogo) abierto = false;
	}}
>
	<div class="hoja">
		<span class="manija" aria-hidden="true"></span>
		<div class="cab">
			<h2>{cambiar ? 'Cambiar tu link' : 'Compartí tu link'}</h2>
			<button class="cerrar" aria-label="Cerrar" onclick={() => (abierto = false)}><Icono nombre="close" tam={22} /></button>
		</div>
		{#if cambiar}
			<EditorLink {slug} {host} />
			<button class="volver" onclick={() => (cambiar = false)}><Icono nombre="arrow_back" tam={18} />Volver a compartir</button>
		{:else}
			<TuLink {nombre} {slug} {origen} compacto ver alCambiar={gestiona ? () => (cambiar = true) : undefined} />
		{/if}
	</div>
</dialog>

<style>
	.panel-link {
		width: min(560px, calc(100vw - 32px));
		max-height: calc(100dvh - 48px);
		padding: 0;
		border: 0;
		border-radius: 28px;
		background: var(--tl-crema);
		color: var(--tl-tinta);
		box-shadow: 0 30px 50px -36px rgba(15, 35, 40, 0.8);
	}
	.panel-link::backdrop {
		background: rgba(16, 43, 49, 0.55);
	}
	.panel-link[open] {
		animation: tl-paso 0.3s var(--tl-ease) both;
	}
	.hoja {
		display: flex;
		flex-direction: column;
		gap: 14px;
		padding: 16px 20px 24px;
	}
	.manija {
		display: none;
	}
	.cab {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 8px;
	}
	h2 {
		margin: 0;
		font-size: 22px;
		letter-spacing: -0.03em;
	}
	.cerrar {
		width: 44px;
		height: 44px;
		border: 0;
		border-radius: 12px;
		background: transparent;
		color: var(--tl-tinta);
		display: grid;
		place-items: center;
		cursor: pointer;
	}
	.volver {
		align-self: flex-start;
		display: inline-flex;
		align-items: center;
		gap: 6px;
		min-height: 44px;
		padding: 0;
		border: 0;
		background: none;
		color: #17805a;
		font: 700 14px var(--tl-font);
		cursor: pointer;
	}
	@media (max-width: 899px) {
		.panel-link {
			width: 100vw;
			max-width: 100vw;
			margin: auto 0 0;
			border-radius: 28px 28px 0 0;
		}
		.hoja {
			padding: 8px 16px calc(24px + env(safe-area-inset-bottom));
		}
		.manija {
			display: block;
			align-self: center;
			width: 40px;
			height: 5px;
			border-radius: 3px;
			background: #ddd1b9;
		}
	}
</style>
