<!--
	La campanita de la barra (handoff «Panel v2» y «Pantallas que faltaban», 1k): el número de sin ver
	y, al tocarla, lo último que pasó; cada aviso lleva a donde se resuelve.
-->
<script lang="ts">
	import { enhance } from '$app/forms';
	import Icono from '#lib/ui/Icono.svelte';
	import type { Notificacion } from '#lib/operacion/tipos.ts';
	import { cuandoAviso, destinoAviso, iconoAviso } from './avisos.ts';

	let {
		avisos,
		sinLeer,
		verEnvios,
		compacto = false
	}: { avisos: Notificacion[]; sinLeer: number; verEnvios: boolean; compacto?: boolean } = $props();

	let abierto = $state(false);
	let raiz: HTMLElement | undefined = $state();
	const etiqueta = $derived(sinLeer ? `Notificaciones, ${sinLeer} sin ver` : 'Notificaciones');

	function fuera(e: MouseEvent) {
		if (abierto && raiz && !raiz.contains(e.target as Node)) abierto = false;
	}
</script>

<svelte:window onclick={fuera} onkeydown={(e) => e.key === 'Escape' && (abierto = false)} />

<div class="campanita" class:compacto bind:this={raiz}>
	<button class="boton-c" aria-label={etiqueta} aria-expanded={abierto} aria-controls="avisos-panel" onclick={() => (abierto = !abierto)}>
		<Icono nombre="notifications" tam={compacto ? 24 : 22} />
		{#if sinLeer}
			{#if compacto}<span class="punto" aria-hidden="true"></span>{:else}<span class="numero tl-num" aria-hidden="true">{sinLeer > 99 ? '99+' : sinLeer}</span>{/if}
		{/if}
	</button>
	{#if abierto}
		<div id="avisos-panel" class="pop" role="dialog" aria-label="Avisos">
			<div class="cabeza">
				<b>Avisos</b>
				{#if sinLeer}
					<form method="POST" action="/panel/notificaciones?/vista" use:enhance>
						<button class="link">{compacto ? 'Marcar leídos' : 'Marcar todo como leído'}</button>
					</form>
				{/if}
			</div>
			{#each avisos as n (n.id)}
				{@const ic = iconoAviso(n.tipo)}
				<a class="item-aviso" class:nuevo={n.nueva} href={destinoAviso(n)} onclick={() => (abierto = false)}>
					<span style:color={ic.color}><Icono nombre={ic.icono} lleno tam={20} /></span>
					<span class="texto">
						<b>{#if n.nueva}<span class="visualmente-oculto">Nuevo: </span>{/if}{n.titulo}</b>
						{#if n.texto}<span>{n.texto}</span>{/if}
					</span>
					<span class="cuando tl-num">{cuandoAviso(n.creadoEn)}</span>
				</a>
			{:else}
				<p class="vacio">No pasó nada esta semana.</p>
			{/each}
			<a class="pie" href="/panel/notificaciones" onclick={() => (abierto = false)}><Icono nombre="notifications" tam={18} />Ver todos los avisos</a>
			{#if verEnvios}
				<a class="pie" href="/panel/notificaciones/envios" onclick={() => (abierto = false)}><Icono nombre="outgoing_mail" tam={18} />Mails enviados: ¿le llegó el&nbsp;recordatorio?</a>
			{/if}
		</div>
	{/if}
</div>

<style>
	.campanita {
		position: relative;
		flex: none;
	}
	.boton-c {
		position: relative;
		width: 44px;
		height: 44px;
		border-radius: 14px;
		border: 0;
		background: var(--tl-papel);
		box-shadow: inset 0 0 0 1px #e6dcc8;
		display: grid;
		place-items: center;
		color: var(--tl-tinta);
		cursor: pointer;
	}
	.boton-c :global(.icono) {
		font-variation-settings: 'FILL' 0, 'wght' 500, 'GRAD' 0, 'opsz' 24;
	}
	.compacto .boton-c {
		background: transparent;
		box-shadow: none;
		color: #fff;
		border-radius: 12px;
	}
	.numero {
		position: absolute;
		top: 5px;
		right: 5px;
		min-width: 18px;
		height: 18px;
		padding: 0 5px;
		border-radius: 999px;
		background: var(--tl-petroleo);
		color: var(--tl-menta);
		font: 700 10px var(--tl-font);
		display: grid;
		place-items: center;
	}
	.punto {
		position: absolute;
		top: 8px;
		right: 8px;
		width: 9px;
		height: 9px;
		border-radius: 50%;
		background: var(--tl-menta);
		box-shadow: 0 0 0 2px var(--tl-petroleo);
	}
	.pop {
		position: absolute;
		right: 0;
		top: 52px;
		z-index: 40;
		width: 380px;
		border-radius: 18px;
		background: var(--tl-papel);
		box-shadow: 0 0 0 1px #e6dcc8, 0 30px 60px -20px rgba(15, 35, 40, 0.45);
		overflow: hidden;
		color: var(--tl-tinta);
		animation: tl-gin 300ms var(--tl-ease) both;
	}
	.compacto .pop {
		position: fixed;
		inset: 60px 0 0;
		width: auto;
		border-radius: 0;
		overflow-y: auto;
		box-shadow: none;
	}
	.cabeza {
		display: flex;
		align-items: center;
		padding: 14px 16px 10px;
	}
	.cabeza b {
		flex: 1;
		font-size: 15px;
	}
	.compacto .cabeza b {
		font-size: 17px;
	}
	.link {
		border: 0;
		background: transparent;
		color: var(--tl-menta-texto);
		font: 700 13px var(--tl-font);
		min-height: 44px;
		cursor: pointer;
	}
	.item-aviso {
		display: grid;
		grid-template-columns: 28px 1fr auto;
		gap: 10px;
		padding: 12px 16px;
		border-top: 1px solid #f0e8d8;
		color: var(--tl-tinta);
		text-decoration: none;
	}
	.item-aviso:hover {
		background: #faf6ee;
	}
	.item-aviso.nuevo {
		background: #f3fbf7;
	}
	.texto {
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
	}
	.texto b {
		font-size: 14px;
		font-weight: 400;
	}
	.nuevo .texto b {
		font-weight: 700;
	}
	.texto span {
		font-size: 12px;
		color: var(--tl-sub);
		overflow-wrap: anywhere;
	}
	.compacto .texto b {
		font-size: 15px;
	}
	.compacto .texto span {
		font-size: 13px;
	}
	.cuando {
		font-size: 11px;
		color: var(--tl-sub);
	}
	.vacio {
		margin: 0;
		padding: 16px;
		border-top: 1px solid #f0e8d8;
		font-size: 14px;
		color: var(--tl-sub);
	}
	.pie {
		display: flex;
		align-items: center;
		gap: 8px;
		width: 100%;
		min-height: 44px;
		padding: 0 16px;
		border-top: 1px solid #f0e8d8;
		font-size: 13px;
		font-weight: 700;
		color: var(--tl-menta-texto);
		text-decoration: none;
	}
</style>
