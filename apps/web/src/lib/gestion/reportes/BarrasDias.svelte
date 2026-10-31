<!--
	«¿Qué días se llena y qué días sobra lugar?»: una barra por día como un talón chico, lo hecho en
	petróleo abajo y lo que quedó libre, rayado, arriba. El libre sale de los minutos disponibles del
	día (si la API los da); si no, la barra es la cantidad de turnos y el resto queda liso, sin decir
	«libre». Con más de dos meses se agrupa por mes.
-->
<script lang="ts">
	import { nombreDia } from '#lib/operacion/fechas.ts';
	import type { DiaReporte } from '#lib/operacion/tipos.ts';

	type Dia = DiaReporte & { minutosReservados?: number; minutosDisponibles?: number };
	let { dias }: { dias: Dia[] } = $props();

	const MESES = ['ene', 'feb', 'mar', 'abr', 'may', 'jun', 'jul', 'ago', 'sep', 'oct', 'nov', 'dic'];
	const conLibre = $derived(dias.some((d) => typeof d.minutosDisponibles === 'number'));

	type Barra = { clave: string; nombre: string; etiqueta: string; turnos: number; hecho: number; libre: number; cerrado: boolean };
	const barras = $derived.by((): Barra[] => {
		let grupos: { clave: string; nombre: string; etiqueta: string; ds: Dia[] }[];
		if (dias.length > 62) {
			const m = new Map<string, Dia[]>();
			for (const d of dias) m.set(d.fecha.slice(0, 7), [...(m.get(d.fecha.slice(0, 7)) ?? []), d]);
			grupos = [...m].map(([k, ds]) => {
				const n = MESES[Number(k.slice(5, 7)) - 1];
				return { clave: k, nombre: n, etiqueta: n.toUpperCase(), ds };
			});
		} else {
			grupos = dias.map((d, i) => ({
				clave: d.fecha,
				nombre: nombreDia(d.fecha),
				etiqueta: i === 0 ? `${Number(d.fecha.slice(8))} ${MESES[Number(d.fecha.slice(5, 7)) - 1].toUpperCase()}` : String(Number(d.fecha.slice(8))),
				ds: [d]
			}));
		}
		const sum = (ds: Dia[], f: (d: Dia) => number) => ds.reduce((a, d) => a + f(d), 0);
		const maxTurnos = Math.max(1, ...grupos.map((g) => sum(g.ds, (d) => d.turnos)));
		return grupos.map((g) => {
			const turnos = sum(g.ds, (d) => d.turnos);
			if (conLibre) {
				const disp = sum(g.ds, (d) => d.minutosDisponibles ?? 0);
				const res = Math.min(disp, sum(g.ds, (d) => d.minutosReservados ?? 0));
				return { ...g, turnos, cerrado: disp === 0, hecho: disp ? (res / disp) * 100 : 0, libre: disp ? 100 - (res / disp) * 100 : 0 };
			}
			return { ...g, turnos, cerrado: turnos === 0, hecho: (turnos / maxTurnos) * 100, libre: 0 };
		});
	});
	// Rótulos abajo: el primero, cada 7 y el último (o todos si son meses).
	const conRotulo = (i: number) => dias.length > 62 || i === 0 || i === barras.length - 1 || (i % 7 === 0 && barras.length - 1 - i > 2);
	const descripcion = $derived(
		barras
			.map((b) => `${b.nombre}: ${b.turnos} ${b.turnos === 1 ? 'turno' : 'turnos'}${conLibre && !b.cerrado ? `, ${Math.round(b.libre)} % libre` : ''}`)
			.join('; ')
	);
</script>

<div class="grafico" role="img" aria-label={`Turnos por día. ${descripcion}`}>
	<div class="barras" style:--n={barras.length}>
		{#each barras as b (b.clave)}
			<div class="col" class:cerrado={b.cerrado} title={`${b.nombre}: ${b.turnos} turnos`}>
				{#if !b.cerrado}
					{#if conLibre}<span class="libre" style:height="{b.libre}%"></span>{/if}
					<span class="hecho" style:height="{b.hecho}%"></span>
				{:else}
					<span class="raya"></span>
				{/if}
			</div>
		{/each}
	</div>
	<div class="rotulos" style:--n={barras.length} aria-hidden="true">
		{#each barras as b, i (b.clave)}<span>{conRotulo(i) ? b.etiqueta : ''}</span>{/each}
	</div>
</div>
{#if conLibre}
	<p class="leyenda" aria-hidden="true"><span class="m hecho"></span>Turnos <span class="m libre"></span>Libre</p>
{/if}

<style>
	.grafico {
		overflow: hidden;
	}
	.barras,
	.rotulos {
		display: grid;
		grid-template-columns: repeat(var(--n), minmax(0, 1fr));
		gap: clamp(2px, 0.6vw, 6px);
	}
	.barras {
		height: 146px;
		align-items: end;
	}
	.col {
		height: 100%;
		display: flex;
		flex-direction: column;
		justify-content: flex-end;
		border-radius: 4px;
		overflow: hidden;
		background: var(--tl-crema);
	}
	.col.cerrado {
		background: none;
	}
	.hecho {
		display: block;
		background: var(--tl-petroleo);
		min-height: 2px;
	}
	.libre {
		display: block;
		background: repeating-linear-gradient(-45deg, rgba(60, 204, 144, 0.5) 0 2px, #f3fbf7 2px 7px);
		box-shadow: inset 0 0 0 1px rgba(23, 128, 90, 0.25);
		border-radius: 4px 4px 0 0;
	}
	.raya {
		display: block;
		height: 3px;
		background: var(--tl-petroleo);
		border-radius: 2px;
	}
	.rotulos {
		margin-top: 10px;
		font: 400 11px var(--tl-font);
		font-variant-numeric: tabular-nums;
		color: var(--tl-sub);
		white-space: nowrap;
	}
	.rotulos span {
		overflow: visible;
	}
	.rotulos span:last-child {
		direction: rtl;
	}
	.leyenda {
		display: flex;
		align-items: center;
		gap: 6px;
		margin: 8px 0 0;
		font-size: 12px;
		color: var(--tl-sub);
	}
	.m {
		width: 12px;
		height: 12px;
		border-radius: 3px;
		margin-left: 6px;
	}
	.m.hecho {
		margin-left: 0;
	}
</style>
