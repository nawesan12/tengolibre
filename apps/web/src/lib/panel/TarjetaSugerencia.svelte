<!--
	La sugerencia de la casa (planes/promos-cruzadas-produccion-corregido.md, lámina en
	docs/laminas/sugerencia.html): una fila discreta al final del Inicio con la foto de la otra app, el
	sello «De Grupo Vimo», el título y el link. La cruz abre «Ocultar por 30 días» o «Ya la uso / No me
	interesa». Al cerrar se va en el acto; si la imagen no carga, no se dibuja nada. La misma pieza que
	TarjetaSugerencia.swift y TarjetaSugerencia.kt.
-->
<script lang="ts">
	import { onMount } from 'svelte';
	import Icono from '#lib/ui/Icono.svelte';
	import { avisar, type Sugerencia } from './sugerencia.ts';

	let { sugerencia, negocio }: { sugerencia: Sugerencia; negocio: string } = $props();

	let cerrada = $state(false);
	let fallo = $state(false);
	let menu = $state(false);
	let img = $state<HTMLImageElement>();
	let raiz = $state<HTMLElement>();

	// Sin palabra sola en el último renglón (con texto grande puede partirse).
	const pegar = (s: string) => s.replace(/ (\S+)$/, ' $1');

	// El CTA: si se parte, la flecha baja pegada a la última palabra (y la marca, que viene con
	// espacios duros de la API, no se corta).
	const cta = $derived.by(() => {
		const i = sugerencia.cta.lastIndexOf(' ');
		return i < 0 ? { inicio: '', fin: sugerencia.cta } : { inicio: sugerencia.cta.slice(0, i + 1), fin: sugerencia.cta.slice(i + 1) };
	});

	onMount(() => {
		// Si la imagen falló antes de que la página se hidratara, onerror no llega.
		if (img?.complete && img.naturalWidth === 0) fallo = true;
		const fuera = (e: PointerEvent) => {
			if (menu && raiz && !raiz.contains(e.target as Node)) menu = false;
		};
		document.addEventListener('pointerdown', fuera);
		return () => document.removeEventListener('pointerdown', fuera);
	});

	function cerrar(hasta: '30d' | 'siempre') {
		menu = false;
		cerrada = true;
		avisar({ id: sugerencia.id, negocio, que: 'cerrar', hasta });
	}
</script>

{#if !cerrada && !fallo}
	<section class="sug" aria-label="Sugerencia de Grupo Vimo" bind:this={raiz}>
		<a class="cuerpo" href={sugerencia.url} target="_blank" rel="noopener" onclick={() => avisar({ id: sugerencia.id, negocio, que: 'clic' })}>
			<picture class="foto" style:background-color={sugerencia.color}>
				<source media="(min-width: 768px)" srcset={sugerencia.banner} />
				<img bind:this={img} src={sugerencia.imagen} alt="" width="72" height="72" loading="lazy" decoding="async" onerror={() => (fallo = true)} />
			</picture>
			<span class="textos">
				<span class="sello">{sugerencia.sello}</span>
				<b class="titulo">{pegar(sugerencia.titulo)}</b>
				<span class="cta">{cta.inicio}<span class="fin">{cta.fin}<Icono nombre="arrow_outward" tam={16} /></span></span>
			</span>
		</a>
		<button
			class="cruz"
			type="button"
			aria-label="Ocultar sugerencia"
			aria-haspopup="menu"
			aria-expanded={menu}
			onclick={() => (menu = !menu)}
			onkeydown={(e) => e.key === 'Escape' && (menu = false)}
		>
			<Icono nombre="close" tam={18} />
		</button>
		{#if menu}
			<div class="menu" role="menu" tabindex="-1" onkeydown={(e) => e.key === 'Escape' && (menu = false)}>
				<button type="button" role="menuitem" onclick={() => cerrar('30d')}><Icono nombre="schedule" tam={18} />Ocultar por 30&nbsp;días</button>
				<button type="button" role="menuitem" onclick={() => cerrar('siempre')}><Icono nombre="do_not_disturb_on" tam={18} />Ya la uso / No me&nbsp;interesa</button>
			</div>
		{/if}
	</section>
{/if}

<style>
	.sug {
		position: relative;
		border: 1px solid var(--tl-linea);
		border-radius: 18px;
		background: var(--tl-papel);
		animation: tl-gin var(--tl-entrar) var(--tl-ease) both;
	}
	.cuerpo {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px 12px 12px 12px;
		color: var(--tl-tinta);
		text-decoration: none;
		border-radius: 18px;
	}
	.cuerpo:focus-visible {
		outline: 2px solid var(--tl-menta-texto);
		outline-offset: 2px;
	}
	.foto {
		flex: none;
		width: 72px;
		height: 72px;
		border-radius: 14px;
		overflow: hidden;
	}
	.foto img {
		display: block;
		width: 100%;
		height: 100%;
		object-fit: cover;
		object-position: 50% 100%;
	}
	.textos {
		display: flex;
		flex-direction: column;
		gap: 3px;
		min-width: 0;
	}
	.sello {
		/* El renglón del sello deja lugar a la cruz. */
		padding-right: 36px;
		font-size: 11px;
		font-weight: 900;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--tl-sub);
	}
	.titulo {
		font-size: 15px;
		line-height: 1.25;
		text-wrap: balance;
	}
	.cta {
		font-size: 14px;
		line-height: 1.35;
		font-weight: 700;
		color: var(--tl-menta-texto);
	}
	.fin {
		white-space: nowrap;
	}
	/* inline-block: el subrayado del hover no pasa por la flecha. */
	.fin :global(.icono) {
		display: inline-block;
		margin-left: 2px;
		vertical-align: -3px;
	}
	.cuerpo:hover .cta {
		text-decoration: underline;
	}
	.cruz {
		position: absolute;
		top: 0;
		right: 0;
		width: 44px;
		height: 44px;
		display: grid;
		place-items: center;
		border: 0;
		border-radius: 0 18px 0 14px;
		background: transparent;
		color: var(--tl-sub);
		cursor: pointer;
	}
	.cruz:hover,
	.cruz[aria-expanded='true'] {
		color: var(--tl-tinta);
	}
	.cruz:focus-visible {
		outline: 2px solid var(--tl-menta-texto);
		outline-offset: -4px;
	}
	.menu {
		position: absolute;
		z-index: 5;
		/* Sobre la tarjeta, hacia arriba (pisa sólo su borde): no corta el título ni el CTA, y como la
		   tarjeta es lo último del Inicio, abajo quedarían la barra y el botón «Nuevo turno». */
		bottom: calc(100% - 8px);
		right: 8px;
		display: flex;
		flex-direction: column;
		min-width: 260px;
		padding: 4px 0;
		border-radius: 14px;
		background: var(--tl-papel);
		box-shadow: 0 14px 28px -12px rgba(15, 35, 40, 0.45);
		border: 1px solid var(--tl-linea);
	}
	.menu button {
		display: flex;
		align-items: center;
		gap: 10px;
		min-height: 48px;
		padding: 0 16px;
		border: 0;
		background: transparent;
		color: var(--tl-tinta);
		font: 400 16px var(--tl-font);
		text-align: left;
		cursor: pointer;
	}
	.menu button + button {
		border-top: 1px solid var(--tl-linea);
	}
	.menu button:hover,
	.menu button:focus-visible {
		background: var(--tl-crema);
		outline: none;
	}
	.menu :global(.icono) {
		color: var(--tl-sub);
	}
	@media (min-width: 768px) {
		.foto {
			width: 144px;
		}
	}
</style>
