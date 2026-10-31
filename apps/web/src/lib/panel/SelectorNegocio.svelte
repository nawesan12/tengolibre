<!--
	El negocio arriba del costado (nombre y rol) y, al tocarlo, el selector (handoff «Pantallas que
	faltaban», 1b): cada negocio con su rol, el actual marcado, «Crear otro negocio», y la cuenta.
-->
<script lang="ts">
	import Icono from '#lib/ui/Icono.svelte';
	import { ROLES } from '#lib/formato.ts';
	import type { NegocioDeLaPersona, Persona } from '#lib/server/api.ts';

	let {
		negocio,
		negocios,
		persona,
		compacto = false,
		sub
	}: { negocio: NegocioDeLaPersona; negocios: NegocioDeLaPersona[]; persona: Persona; compacto?: boolean; sub?: string } = $props();

	let abierto = $state(false);
	let raiz: HTMLElement | undefined = $state();
	const inicial = (n: string) => n.trim()[0]?.toUpperCase() ?? '·';

	function fuera(e: MouseEvent) {
		if (abierto && raiz && !raiz.contains(e.target as Node)) abierto = false;
	}
	function tecla(e: KeyboardEvent) {
		if (abierto && e.key === 'Escape') abierto = false;
	}
</script>

<svelte:window onclick={fuera} onkeydown={tecla} />

<div class="selector" class:compacto bind:this={raiz}>
	<button class="actual" aria-expanded={abierto} aria-controls="selector-negocio" onclick={() => (abierto = !abierto)}>
		<span class="avatar" aria-hidden="true">{inicial(negocio.nombre)}</span>
		<span class="texto">
			<b data-negocio-actual>{negocio.nombre}</b>
			<span class="tl-num">{sub ?? ROLES[negocio.rol] ?? negocio.rol}</span>
		</span>
		{#if !compacto}<Icono nombre="unfold_more" tam={20} />{/if}
	</button>

	{#if abierto}
		<div id="selector-negocio" class="pop" role="dialog" aria-label="Cambiar de negocio">
			<span class="tl-rotulo rotulo">{negocios.length > 1 ? 'Cambiar de negocio' : 'Tu negocio'}</span>
			<form method="POST" action="/panel/negocio" class="lista">
				{#each negocios as n (n.id)}
					{@const es = n.id === negocio.id}
					<button class="item" class:es name="id" value={n.id} aria-current={es ? 'true' : undefined} disabled={es}>
						<span class="avatar chico" aria-hidden="true">{inicial(n.nombre)}</span>
						<span class="texto">
							<b>{n.nombre}</b>
							<span>{ROLES[n.rol] ?? n.rol}</span>
						</span>
						{#if es}<span class="tl-rotulo actual-tag">Actual</span>{/if}
					</button>
				{/each}
			</form>
			<a class="crear" href="/empezar">
				<Icono nombre="add_business" tam={22} />
				<span class="texto"><b>Crear otro negocio</b><span>7 días gratis</span></span>
			</a>
			<div class="cuenta">
				<span class="email" title={persona.email}>{persona.email}</span>
				<a href="/mi-cuenta">Mi cuenta</a>
				<form method="POST" action="/salir"><button class="salir">Salir</button></form>
			</div>
		</div>
	{/if}
</div>

<style>
	.selector {
		position: relative;
		margin: 0 14px 18px 0;
	}
	.actual {
		position: relative;
		display: flex;
		align-items: center;
		gap: 10px;
		width: 100%;
		min-height: 48px;
		padding: 6px 8px 6px 4px;
		border: 0;
		border-radius: 14px;
		background: transparent;
		color: #fff;
		text-align: left;
		font-family: var(--tl-font);
		cursor: pointer;
		transition: background var(--tl-tocar) var(--tl-ease);
	}
	.actual:hover {
		background: rgba(255, 255, 255, 0.07);
	}
	.actual :global(.icono) {
		color: rgba(255, 255, 255, 0.6);
	}
	.avatar {
		flex: none;
		width: 36px;
		height: 36px;
		border-radius: 10px;
		display: grid;
		place-items: center;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		font: 900 17px var(--tl-font);
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.4);
	}
	.avatar.chico {
		width: 36px;
		height: 36px;
		background: var(--tl-petroleo);
		color: #fff;
		font-size: 15px;
	}
	.texto {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		line-height: 1.2;
	}
	.texto b {
		font-size: 15px;
		font-weight: 700;
		letter-spacing: -0.01em;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.actual .texto span {
		font-size: 11px;
		color: rgba(255, 255, 255, 0.6);
	}
	.pop {
		position: absolute;
		left: 0;
		top: calc(100% + 8px);
		z-index: 60;
		width: 300px;
		max-width: calc(100vw - 32px);
		padding: 20px 16px 12px;
		border-radius: var(--tl-r-l);
		background-color: var(--tl-crema);
		background-image: var(--tl-grano-osc), radial-gradient(rgba(24, 60, 68, 0.1) 1.2px, transparent 1.4px);
		background-size: auto, 18px 18px;
		box-shadow: 0 0 0 1px #e6dcc8, 0 30px 60px -20px rgba(15, 35, 40, 0.55);
		color: var(--tl-tinta);
		display: flex;
		flex-direction: column;
		gap: 8px;
		animation: tl-gin var(--tl-entrar) var(--tl-ease) both;
	}
	.compacto {
		flex: 1;
		min-width: 0;
		margin: 0;
	}
	.compacto .actual {
		padding: 4px 6px 4px 0;
		min-height: 44px;
	}
	.compacto .actual > .avatar {
		width: 30px;
		height: 30px;
		border-radius: 8px;
		font-size: 15px;
	}
	.compacto .pop {
		position: fixed;
		left: 16px;
		top: 68px;
	}
	.rotulo {
		color: var(--tl-menta-texto);
		font-size: 11px;
		margin: 0 4px 4px;
	}
	.lista {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.item {
		display: flex;
		align-items: center;
		gap: 12px;
		min-height: 56px;
		padding: 8px 12px;
		border: 0;
		border-radius: 14px;
		background: var(--tl-papel);
		color: var(--tl-tinta);
		text-align: left;
		font-family: var(--tl-font);
		cursor: pointer;
		box-shadow: 0 1px 0 rgba(15, 35, 40, 0.05);
	}
	.item:disabled {
		cursor: default;
		opacity: 1;
	}
	.item.es {
		background: #f3fbf7;
		box-shadow: inset 0 0 0 2px var(--tl-menta);
	}
	.item .texto b {
		font-size: 14px;
	}
	.item .texto span {
		font-size: 12px;
		color: var(--tl-sub);
	}
	.actual-tag {
		color: var(--tl-menta-texto);
		font-size: 11px;
	}
	.crear {
		display: flex;
		align-items: center;
		gap: 12px;
		min-height: 52px;
		padding: 4px 12px;
		border-radius: 14px;
		color: var(--tl-tinta);
		text-decoration: none;
	}
	.crear:hover {
		background: rgba(255, 255, 255, 0.5);
	}
	.crear :global(.icono) {
		color: var(--tl-menta-texto);
	}
	.crear .texto b {
		font-size: 14px;
	}
	.crear .texto span {
		font-size: 12px;
		color: var(--tl-sub);
	}
	.cuenta {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 10px 4px 0;
		border-top: 1px solid var(--tl-linea-2);
		font-size: 13px;
	}
	.email {
		flex: 1;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		color: var(--tl-sub);
	}
	.cuenta a,
	.salir {
		min-height: 40px;
		display: inline-flex;
		align-items: center;
		border: 0;
		background: transparent;
		padding: 0;
		font: 700 13px var(--tl-font);
		color: var(--tl-menta-texto);
		text-decoration: none;
		cursor: pointer;
	}
	.salir {
		color: #a3241a;
	}
</style>
