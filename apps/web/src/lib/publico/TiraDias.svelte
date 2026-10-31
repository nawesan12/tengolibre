<!-- La tira de días (handoff v2): uno al lado del otro, con scroll de costado; los días sin lugar,
	tachados y sin link. Cada día es un link real (funciona sin JavaScript). -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { tiraDia } from './calendario.ts';
	import type { Dia } from './tipos.ts';

	let {
		dias,
		href,
		elegida = '',
		chica = false
	}: { dias: Dia[]; href: (fecha: string) => string; elegida?: string; chica?: boolean } = $props();

	const etiqueta = (f: string, libres: number) => {
		const txt = new Date(f + 'T12:00:00Z').toLocaleDateString('es-AR', { weekday: 'long', day: 'numeric', month: 'long', timeZone: 'UTC' });
		return libres ? `${txt}: ${libres} ${libres === 1 ? 'horario libre' : 'horarios libres'}` : `${txt}: sin lugar`;
	};

	// El día elegido, a la vista (si está más allá del borde).
	let tira: HTMLDivElement;
	onMount(() => {
		const el = tira?.querySelector<HTMLElement>('[aria-current]');
		if (el && el.offsetLeft + el.offsetWidth > tira.clientWidth) tira.scrollLeft = el.offsetLeft - 18;
	});
</script>

<div class="tira" class:chica bind:this={tira} role="group" aria-label="Días">
	{#each dias as d (d.fecha)}
		{@const t = tiraDia(d.fecha)}
		{#if d.libres > 0}
			<a
				class="dia"
				href={href(d.fecha)}
				data-sveltekit-noscroll
				aria-label={etiqueta(d.fecha, d.libres)}
				aria-current={d.fecha === elegida ? 'date' : undefined}
				data-fecha={d.fecha}><span class="dow">{t.dow}</span><span class="num">{t.numero}</span></a
			>
		{:else}
			<span class="dia lleno" aria-label={etiqueta(d.fecha, 0)} data-fecha={d.fecha}><span class="dow">{t.dow}</span><span class="num">{t.numero}</span></span>
		{/if}
	{/each}
</div>

<style>
	.tira {
		display: flex;
		gap: 6px;
		overflow-x: auto;
		margin: 0 -18px;
		padding: 2px 18px;
		scrollbar-width: none;
	}
	.chica {
		margin: 0;
		padding: 2px 0;
	}
	.dia {
		flex: none;
		width: 54px;
		height: 66px;
		border-radius: var(--w-sr);
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 2px;
		background: var(--w-campo);
		color: var(--w-ink);
		box-shadow: inset 0 0 0 1.5px var(--w-line2);
		text-decoration: none;
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.chica .dia {
		width: 52px;
		height: 62px;
	}
	a.dia:active {
		transform: scale(0.94);
	}
	.dia[aria-current] {
		background: var(--w-pri);
		color: var(--w-on);
		box-shadow: none;
	}
	.lleno {
		background: transparent;
		color: var(--w-sub);
		box-shadow: inset 0 0 0 1px var(--w-line);
	}
	.dow {
		font: 400 11px var(--tl-font);
		text-transform: uppercase;
	}
	.num {
		font: 700 19px var(--tl-font);
		font-variant-numeric: tabular-nums;
	}
	.chica .num {
		font-size: 18px;
	}
	.lleno .num {
		text-decoration: line-through;
	}
</style>
