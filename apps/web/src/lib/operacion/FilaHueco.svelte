<!--
	Un hueco en la lista de turnos (firma 2, el rayado que corre): «Cargar acá» y, si alguien de la
	lista de espera le sirve, «Ofrecer a la lista». Mientras se ofrece (Propuesta 01), la cuenta
	regresiva, a quién y quién sigue, con «Frenar».
-->
<script lang="ts">
	import { enhance } from '$app/forms';
	import { onMount } from 'svelte';
	import Icono from '#lib/ui/Icono.svelte';
	import type { Oferta } from './contrato.ts';

	let {
		rango,
		titulo,
		sub = '',
		cargar,
		esperan = 0,
		oferta = null,
		ofrecer = null,
		puedeActuar = true
	}: {
		rango: string;
		titulo: string;
		sub?: string;
		cargar: string;
		esperan?: number;
		oferta?: Oferta | null;
		/** Lo que va en el formulario para ofrecerlo: profesional, inicio y fin. */
		ofrecer?: { profesionalId: string; inicio: string; servicioId: string | null } | null;
		puedeActuar?: boolean;
	} = $props();

	let ahora = $state(Date.now());
	onMount(() => {
		const r = setInterval(() => (ahora = Date.now()), 1000);
		return () => clearInterval(r);
	});
	const actual = $derived(oferta?.actual ?? null);
	const vence = $derived(actual?.venceEn ? Date.parse(actual.venceEn) : 0);
	const desde = $derived(actual?.ofrecidaEn ? Date.parse(actual.ofrecidaEn) : ahora);
	const restan = $derived(Math.max(0, vence - ahora));
	const cuenta = $derived(`${String(Math.floor(restan / 60000)).padStart(2, '0')}:${String(Math.floor(restan / 1000) % 60).padStart(2, '0')}`);
	// La barra se llena a medida que se le acaba el tiempo a quien lo tiene.
	const progreso = $derived(vence > desde ? Math.min(1, (ahora - desde) / (vence - desde)) : 0);
	const corto = (n: string) => {
		const p = n.trim().split(/\s+/);
		return p.length > 1 ? `${p[0]} ${p[1][0]}.` : p[0];
	};
</script>

<li class="hueco" class:ofreciendo={!!oferta}>
	{#if oferta}
		<span class="rango tl-num">{rango}</span>
		<b class="titulo">Ofreciendo a {actual ? corto(actual.nombre) : 'la lista'}</b>
		<span class="reloj tl-num" role="timer" aria-label={`Le quedan ${Math.ceil(restan / 60000)} minutos para tomarlo`}><Icono nombre="timer" tam={16} />{cuenta}</span>
		<span class="barra" aria-hidden="true"><span style:width={`${(progreso * 100).toFixed(1)}%`}></span></span>
		<span class="sigue tl-num">
			{#if oferta.sigue}SIGUE: <span class="quien">{oferta.sigue.orden} · {corto(oferta.sigue.nombre).toUpperCase()}</span>{:else}ES LA ÚLTIMA DE LA LISTA{/if}
		</span>
		{#if puedeActuar}
			<span class="fin">
				<form method="POST" action="?/cargar" use:enhance>
					<input type="hidden" name="id" value={oferta.id} />
					<button class="yo">Cargarlo yo</button>
				</form>
				<form method="POST" action="?/frenar" use:enhance>
					<input type="hidden" name="id" value={oferta.id} />
					<button class="frenar">Frenar</button>
				</form>
			</span>
		{/if}
	{:else}
		<span class="rango tl-num">{rango}</span>
		<span class="texto"><b>{titulo}</b>{#if sub}<span>{sub}</span>{/if}</span>
		{#if puedeActuar}
			{#if ofrecer?.servicioId && esperan > 0}
				<form method="POST" action="?/ofrecer" use:enhance>
					<input type="hidden" name="profesionalId" value={ofrecer.profesionalId} />
					<input type="hidden" name="inicio" value={ofrecer.inicio} />
					{#if ofrecer.servicioId}<input type="hidden" name="servicioId" value={ofrecer.servicioId} />{/if}
					<button class="ofrecer"><Icono nombre="campaign" tam={16} />Ofrecer a la lista · {esperan}</button>
				</form>
			{/if}
			<a class="cargar" href={cargar}><Icono nombre="add" tam={16} />Cargar acá</a>
		{/if}
	{/if}
</li>

<style>
	.hueco {
		list-style: none;
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 8px 14px;
		min-height: 54px;
		padding: 8px 12px 8px 18px;
		border-radius: 14px;
		background: repeating-linear-gradient(-45deg, rgba(60, 204, 144, 0.22) 0 2px, transparent 2px 10px), rgba(60, 204, 144, 0.08);
		background-size: 28.28px 28.28px;
		box-shadow: inset 0 0 0 1.5px rgba(23, 128, 90, 0.35);
		animation:
			tl-gin var(--tl-entrar) var(--tl-ease) both,
			raya 3.5s linear infinite;
	}
	@keyframes raya {
		to {
			background-position: 28.28px 0;
		}
	}
	.hueco.ofreciendo {
		box-shadow: inset 0 0 0 2px var(--tl-menta-texto);
		padding: 12px 16px;
	}
	.rango {
		font-size: 14px;
		font-weight: 700;
		color: var(--tl-menta-texto);
		min-width: 94px;
	}
	.texto {
		flex: 1 1 160px;
		display: flex;
		flex-direction: column;
		line-height: 1.3;
	}
	.texto b,
	.titulo {
		font-size: 14px;
		font-weight: 700;
		color: #0f3d2b;
	}
	.titulo {
		flex: 1 1 140px;
	}
	.texto span {
		font-size: 12px;
		color: var(--tl-menta-texto);
	}
	.ofrecer,
	.cargar {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		height: 36px;
		padding: 0 13px;
		border-radius: 999px;
		border: 0;
		font: 700 13px var(--tl-font);
		text-decoration: none;
		cursor: pointer;
		white-space: nowrap;
	}
	.ofrecer {
		background: var(--tl-papel);
		color: var(--tl-tinta);
		box-shadow: 0 1px 0 rgba(15, 35, 40, 0.08);
	}
	.ofrecer :global(.icono) {
		color: var(--tl-menta-texto);
	}
	.cargar {
		background: var(--tl-menta-texto);
		color: #fff;
	}
	.reloj {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		height: 32px;
		padding: 0 12px;
		border-radius: 999px;
		background: var(--tl-petroleo);
		color: var(--tl-menta);
		font-size: 13px;
		font-weight: 700;
	}
	.barra {
		flex-basis: 100%;
		height: 4px;
		border-radius: 2px;
		background: rgba(23, 128, 90, 0.18);
		overflow: hidden;
	}
	.barra span {
		display: block;
		height: 100%;
		background: var(--tl-menta-texto);
		transition: width 1s linear;
	}
	.sigue {
		flex: 1;
		font-size: 11px;
		font-weight: 700;
		color: var(--tl-menta-texto);
	}
	.quien {
		padding: 2px 8px;
		border-radius: 6px;
		background: var(--tl-papel);
	}
	.fin {
		display: flex;
		align-items: center;
		gap: 12px;
	}
	.yo,
	.frenar {
		min-height: 36px;
		display: inline-flex;
		align-items: center;
		border: 0;
		background: transparent;
		padding: 0;
		font: 700 13px var(--tl-font);
		color: var(--tl-tinta);
		text-decoration: none;
		cursor: pointer;
	}
	.frenar {
		color: #a3241a;
	}
	@media (max-width: 859px) {
		.ofrecer,
		.cargar {
			height: 44px;
		}
		.yo,
		.frenar {
			min-height: 44px;
		}
	}
</style>
