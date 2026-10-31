<!-- Paso 1: el rubro. Tarjetas con foto; al elegir, tilde y sigue solo. -->
<script lang="ts">
	import { enhance } from '$app/forms';
	import type { SubmitFunction } from '$app/forms';
	import Icono from '#lib/ui/Icono.svelte';
	import { RUBROS } from './rubros.ts';
	import type { Borrador } from './estado.svelte.ts';

	let { b, creado, alEnviar, listo = $bindable() }: { b: Borrador; creado: boolean; alEnviar: SubmitFunction; listo: boolean } = $props();
	$effect(() => {
		listo = !!b.rubro;
	});

	let formulario = $state<HTMLFormElement>();
	let t: ReturnType<typeof setTimeout> | undefined;
	function elegir(clave: string) {
		b.rubro = clave;
		b.direccion.rubro = clave;
		clearTimeout(t);
		t = setTimeout(() => formulario?.requestSubmit(), 380);
	}
</script>

{#snippet contenido()}
	<h1>¿Qué tipo de negocio&nbsp;tenés?</h1>
	<p class="bajada">Con esto te dejamos cargados servicios y horarios típicos. Después cambiás lo que&nbsp;quieras.</p>
	<div class="rubros">
		{#each RUBROS as r, i (r.clave)}
			<button
				type="button"
				class="rubro"
				class:sin-foto={!r.foto}
				aria-pressed={b.rubro === r.clave}
				onclick={() => elegir(r.clave)}
				style:animation-delay="{i * 50}ms"
			>
				{#if r.foto}
					<span class="foto" style:background-image="url(/fotos/{r.foto})" style:background-position={r.pos}></span>
				{:else}
					<span class="rayado" aria-hidden="true"><Icono nombre={r.icono} lleno tam={56} /></span>
				{/if}
				<span class="velo"></span>
				<span class="pie">
					<span class="chip"><Icono nombre={r.icono} lleno tam={20} /></span>
					<b lang="es">{r.nombre}</b>
				</span>
				{#if b.rubro === r.clave}
					<span class="tilde"><Icono nombre="check" tam={20} /><span class="visualmente-oculto">Elegido</span></span>
				{/if}
			</button>
		{/each}
	</div>
{/snippet}

{#if creado}
	<form id="paso" bind:this={formulario} method="POST" action="?/rubro" use:enhance={alEnviar} class="hoja" data-paso="rubro">
		<input type="hidden" name="rubro" value={b.rubro} />
		{@render contenido()}
	</form>
{:else}
	<!-- Sin negocio todavía: el rubro viaja en la URL hasta el paso del nombre. -->
	<form id="paso" bind:this={formulario} method="GET" action="/empezar" class="hoja" data-paso="rubro">
		<input type="hidden" name="paso" value="nombre" />
		<input type="hidden" name="rubro" value={b.rubro} />
		{#if b.nombre}<input type="hidden" name="nombre" value={b.nombre} />{/if}
		{#if b.slugManual}<input type="hidden" name="slug" value={b.slugManual} />{/if}
		{@render contenido()}
	</form>
{/if}

<style>
	.rubros {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(168px, 1fr));
		gap: 10px;
	}
	.rubro {
		position: relative;
		overflow: hidden;
		min-height: 168px;
		padding: 0;
		border-radius: 22px;
		border: 0;
		background: #183c44;
		box-shadow: 0 10px 22px -18px rgba(15, 35, 40, 0.5);
		text-align: left;
		cursor: pointer;
		font-family: var(--tl-font);
		transition:
			transform 0.25s var(--tl-ease),
			box-shadow 0.25s;
		animation: tl-pop 0.5s both;
	}
	.rubro:hover {
		transform: translateY(-3px);
		box-shadow: 0 20px 36px -20px rgba(24, 60, 68, 0.6);
	}
	.rubro:active {
		transform: scale(0.97);
	}
	.rubro[aria-pressed='true'] {
		transform: translateY(-3px);
		box-shadow: 0 0 0 3px #3ccc90, 0 20px 36px -18px rgba(24, 60, 68, 0.7);
	}
	.sin-foto {
		background: repeating-linear-gradient(-45deg, #ece5d6 0 10px, #e4dccb 10px 20px);
	}
	.foto {
		position: absolute;
		inset: 0;
		background-size: cover;
		transition: transform 0.6s var(--tl-ease);
	}
	.rubro:hover .foto {
		transform: scale(1.07);
	}
	.rayado {
		position: absolute;
		inset: 0 0 40px;
		display: grid;
		place-items: center;
		color: rgba(24, 60, 68, 0.28);
	}
	.velo {
		position: absolute;
		inset: 0;
		background: linear-gradient(to top, rgba(15, 35, 40, 0.88) 0%, rgba(15, 35, 40, 0.35) 45%, rgba(15, 35, 40, 0) 75%);
	}
	.pie {
		position: absolute;
		left: 10px;
		right: 10px;
		bottom: 10px;
		display: flex;
		align-items: center;
		gap: 8px;
		min-width: 0;
		color: #fff;
	}
	.chip {
		width: 36px;
		height: 36px;
		flex: none;
		border-radius: 12px;
		background: rgba(255, 255, 255, 0.18);
		backdrop-filter: blur(8px);
		-webkit-backdrop-filter: blur(8px);
		display: grid;
		place-items: center;
		color: #fff;
		transition: background 0.2s;
	}
	.rubro[aria-pressed='true'] .chip {
		background: #3ccc90;
		color: #102b31;
	}
	.pie b {
		flex: 1;
		min-width: 0;
		font-size: 16px;
		font-weight: 700;
		letter-spacing: -0.01em;
		line-height: 1.15;
		overflow-wrap: anywhere;
		hyphens: auto;
	}
	.tilde {
		position: absolute;
		right: 10px;
		top: 10px;
		width: 32px;
		height: 32px;
		border-radius: 50%;
		background: #3ccc90;
		color: #102b31;
		display: grid;
		place-items: center;
		animation: tl-pop 0.35s both;
	}
	@media (max-width: 899px) {
		.rubros {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
		.rubro {
			min-height: 150px;
		}
		.pie b {
			font-size: 15px;
		}
	}
</style>
