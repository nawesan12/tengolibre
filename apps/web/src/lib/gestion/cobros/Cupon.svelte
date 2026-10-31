<!-- Un cupón como en el diseño (Membresías y cupones): el código que se lee, el descuento, la barra
	de usos (sólo si tiene tope) y una línea con usos, vigencia y estado. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import { pesos } from '#lib/formato.ts';
	import { descuento } from '#lib/configuracion/dinero.ts';
	import { fechaCorta } from './formato.ts';

	type Cupon = {
		codigo: string;
		tipo: string;
		valor: number;
		minimo: number | null;
		maxUsos: number | null;
		usos: number;
		hasta: string | null;
		activo: boolean;
	};
	let { c, acciones }: { c: Cupon; acciones?: Snippet } = $props();

	const agotado = $derived(c.maxUsos !== null && c.usos >= c.maxUsos);
	const vencido = $derived(c.hasta !== null && new Date(c.hasta).getTime() < Date.now());
	const info = $derived(
		[
			!c.activo ? 'Pausado' : agotado ? 'Agotado' : vencido ? 'Vencido' : '',
			c.maxUsos !== null ? `${c.usos} de ${c.maxUsos} usos` : `${c.usos} ${c.usos === 1 ? 'uso' : 'usos'} · sin límite`,
			c.hasta ? `hasta ${fechaCorta(c.hasta)}` : '',
			c.minimo ? `desde ${pesos(c.minimo)}` : ''
		]
			.filter(Boolean)
			.join(' · ')
	);
</script>

<div class="cupon" class:apagado={!c.activo || agotado || vencido}>
	<span class="cabeza">
		<b class="codigo">{c.codigo}</b>
		<span class="valor">{descuento(c.tipo, c.valor, pesos)}</span>
	</span>
	{#if c.maxUsos !== null}
		<span class="g-barra" aria-hidden="true"><span style:width="{Math.min(100, (c.usos / c.maxUsos) * 100)}%"></span></span>
	{/if}
	<span class="info">
		{#if !c.activo}<span class="icono" aria-hidden="true">pause_circle</span>{:else if agotado || vencido}<span class="icono" aria-hidden="true">block</span>{/if}
		{info}
	</span>
	{#if acciones}<span class="acciones">{@render acciones()}</span>{/if}
</div>

<style>
	.cupon {
		display: flex;
		flex-direction: column;
		gap: 6px;
		padding: 12px 18px;
	}
	.apagado .cabeza,
	.apagado .g-barra {
		opacity: 0.6;
	}
	.cabeza {
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.codigo {
		flex: 1;
		min-width: 0;
		font: 700 15px var(--tl-font);
		font-variant-numeric: tabular-nums;
		letter-spacing: 0.08em;
		overflow-wrap: anywhere;
	}
	.valor {
		font: 700 13px var(--tl-font);
		font-variant-numeric: tabular-nums;
		color: var(--tl-menta-texto);
		white-space: nowrap;
	}
	.g-barra {
		height: 5px;
		background: var(--tl-crema);
	}
	.info {
		display: flex;
		align-items: center;
		gap: 4px;
		font: 400 11px var(--tl-font);
		font-variant-numeric: tabular-nums;
		text-transform: uppercase;
		color: var(--tl-sub);
	}
	.info .icono {
		font-size: 14px;
	}
	.acciones {
		display: flex;
		flex-wrap: wrap;
		gap: 4px 16px;
	}
</style>
