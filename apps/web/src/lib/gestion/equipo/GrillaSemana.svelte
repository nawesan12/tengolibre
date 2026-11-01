<!--
	El horario de la semana como una grilla de franjas (handoff v2, «Equipo y horarios»): una fila
	por día, de lunes a domingo, con lo que atiende rayado como un hueco y el eje de horas abajo.
-->
<script lang="ts">
	import { SEMANA, franjasDe, type Franja } from '#lib/configuracion/horario.ts';
	import { DIAS } from '#lib/formato.ts';
	import { DIA_CORTO, eje, minutos, rango } from './semana.ts';

	let { horario }: { horario: Franja[] } = $props();
	const e = $derived(eje(horario));
	const pos = (h: string) => ((minutos(h) - e.desde) / (e.hasta - e.desde)) * 100;
</script>

<div class="grilla" role="list" aria-label="Franjas de la semana">
	{#each SEMANA as d (d)}
		{@const fr = franjasDe(horario, d)}
		<div class="dia" role="listitem">
			<b class:cerrado={fr.length === 0} aria-hidden="true">{DIA_CORTO[d]}</b>
			<span class="visualmente-oculto">{DIAS[d]}: {fr.length === 0 ? 'no atiende' : fr.map(rango).join(' y ')}</span>
			<div class="pista" aria-hidden="true">
				{#if fr.length === 0}
					<span class="vacio">Cerrado</span>
				{/if}
				{#each fr as f (f.desde)}
					{@const l = pos(f.desde)}
					{@const w = Math.max(0, pos(f.hasta) - l)}
					<span class="franja" style:left="{l}%" style:width="{w}%">
						{#if w >= 9}{rango(f)}{/if}
					</span>
				{/each}
			</div>
		</div>
	{/each}
	<div class="dia eje" aria-hidden="true">
		<span></span>
		<span class="marcas">
			{#each e.marcas as m (m)}<span>{String(m).padStart(2, '0')}</span>{/each}
		</span>
	</div>
</div>

<style>
	.grilla {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.dia {
		display: grid;
		grid-template-columns: 44px minmax(0, 1fr);
		gap: 12px;
		align-items: center;
	}
	.dia > b {
		font: 700 12px var(--tl-font);
		font-variant-numeric: tabular-nums;
		color: var(--tl-tinta);
	}
	.dia > b.cerrado {
		color: var(--tl-sub);
	}
	.pista {
		position: relative;
		height: 30px;
		border-radius: 8px;
		background: var(--tl-crema);
	}
	.vacio {
		position: absolute;
		inset: 0;
		display: flex;
		align-items: center;
		padding-left: 10px;
		font: 400 11px var(--tl-font);
		text-transform: uppercase;
		letter-spacing: 0.04em;
		color: var(--tl-sub);
	}
	.franja {
		position: absolute;
		top: 0;
		bottom: 0;
		border-radius: 8px;
		background: repeating-linear-gradient(-45deg, rgba(60, 204, 144, 0.3) 0 2px, transparent 2px 9px), #dff4e9;
		background-size: 28px 28px;
		box-shadow: inset 0 0 0 1.5px rgba(23, 128, 90, 0.4);
		display: flex;
		align-items: center;
		justify-content: center;
		overflow: hidden;
		white-space: nowrap;
		font: 700 11px var(--tl-font);
		font-variant-numeric: tabular-nums;
		color: var(--tl-menta-texto);
		animation: tl-raya 3.5s linear infinite;
		transition:
			left var(--tl-tocar) var(--tl-ease),
			width var(--tl-tocar) var(--tl-ease);
	}
	.eje {
		font: 400 11px var(--tl-font);
		font-variant-numeric: tabular-nums;
		color: var(--tl-sub);
	}
	.marcas {
		display: flex;
		justify-content: space-between;
	}
	@keyframes tl-raya {
		to {
			background-position: 28px 0;
		}
	}
</style>
