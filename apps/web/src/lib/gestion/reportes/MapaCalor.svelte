<!--
	La semana como mapa de calor (handoff «Reporte mensual v2», página 2): una fila por día y una celda
	por hora. Lleno en petróleo, a medias en gris petróleo, lo que quedó libre rayado (el hueco) con su
	ocupación, y lo cerrado en crema con una raya. Sirve para Horas muertas y para el reporte mensual.
-->
<script lang="ts" module>
	export type Celda = {
		dia: number;
		hora: number;
		capacidadMinutos: number;
		ocupadoMinutos: number;
		ocupacion: number | null;
		vacio: number | null;
		estado: string;
	};
	export type Mapa = { dias: number[]; horas: number[]; celdas: Celda[] };
</script>

<script lang="ts">
	let { mapa, etiqueta, leyenda = true }: { mapa: Mapa; etiqueta: string; leyenda?: boolean } = $props();

	const CORTO = ['Dom', 'Lun', 'Mar', 'Mié', 'Jue', 'Vie', 'Sáb'];
	const LARGO = ['Domingo', 'Lunes', 'Martes', 'Miércoles', 'Jueves', 'Viernes', 'Sábado'];

	/** El estado del mapa de horas muertas se pasa al de tres tonos del mensual. */
	function tono(c: Celda | undefined): 'lleno' | 'medias' | 'libre' | 'cerrado' {
		if (!c || c.estado === 'cerrado' || c.ocupacion === null) return 'cerrado';
		if (c.estado === 'lleno') return 'lleno';
		if (c.estado === 'a_medias') return 'medias';
		if (c.estado === 'libre' || c.estado === 'suele_quedar_vacio') return 'libre';
		return c.ocupacion >= 85 ? 'lleno' : 'medias';
	}
	const celda = (d: number, h: number) => mapa.celdas.find((c) => c.dia === d && c.hora === h);
	const pct = (n: number | null) => `${Math.round(n ?? 0)}%`;
	const TEXTO = { lleno: 'lleno', medias: 'a medias', libre: 'quedó libre', cerrado: 'cerrado' };
</script>

<div class="marco">
	<table class="mapa" style:--cols={mapa.horas.length}>
		<caption class="visualmente-oculto">{etiqueta}</caption>
		<thead>
			<tr>
				<td></td>
				{#each mapa.horas as h (h)}<th scope="col" class="num">{String(h).padStart(2, '0')}</th>{/each}
			</tr>
		</thead>
		<tbody>
			{#each mapa.dias as d (d)}
				<tr>
					<th scope="row"><abbr title={LARGO[d]}>{CORTO[d]}</abbr></th>
					{#each mapa.horas as h (h)}
						{@const c = celda(d, h)}
						{@const t = tono(c)}
						<td>
							<span class="c {t}">
								{#if t === 'libre'}<span aria-hidden="true">{pct(c?.ocupacion ?? 0)}</span>{:else if t === 'cerrado'}<span aria-hidden="true">–</span>{/if}
								<span class="visualmente-oculto">{LARGO[d]} {h} h: {TEXTO[t]}{t !== 'cerrado' ? `, ocupado ${pct(c?.ocupacion ?? 0)}` : ''}</span>
							</span>
						</td>
					{/each}
				</tr>
			{/each}
		</tbody>
	</table>
</div>
{#if leyenda}
	<ul class="leyenda">
		<li><span class="m lleno" aria-hidden="true"></span>Lleno (más de&nbsp;85%)</li>
		<li><span class="m medias" aria-hidden="true"></span>A medias</li>
		<li><span class="m libre" aria-hidden="true"></span>Quedó libre (menos de&nbsp;50%)</li>
	</ul>
{/if}

<style>
	.marco {
		max-width: 100%;
		overflow-x: auto;
	}
	.mapa {
		width: 100%;
		min-width: calc(44px + var(--cols) * 26px);
		border-collapse: separate;
		border-spacing: 4px;
		margin: -4px;
		table-layout: fixed;
		font-variant-numeric: tabular-nums;
	}
	.mapa thead th {
		font-size: 12px;
		font-weight: 700;
		color: var(--tl-sub);
		text-align: center;
	}
	.mapa thead td,
	.mapa tbody th {
		width: 44px;
	}
	.mapa tbody th {
		text-align: left;
		font-size: 13px;
		font-weight: 700;
	}
	abbr {
		text-decoration: none;
	}
	td {
		padding: 0;
	}
	.c {
		display: grid;
		place-items: center;
		height: 34px;
		border-radius: 7px;
		font-size: 11px;
		font-weight: 700;
	}
	.lleno {
		background: var(--tl-petroleo);
	}
	.medias {
		background: #7c9aa0;
	}
	.libre {
		background: repeating-linear-gradient(-45deg, #3ccc90 0 2px, #dff4e9 2px 6px);
		box-shadow: inset 0 0 0 1px rgba(23, 128, 90, 0.35);
		color: var(--tl-petroleo-hondo);
	}
	.libre span[aria-hidden] {
		padding: 0 3px;
		border-radius: 4px;
		background: rgba(223, 244, 233, 0.85);
	}
	.cerrado {
		background: var(--tl-crema);
		color: #8a9794;
	}
	.leyenda {
		list-style: none;
		margin: 4px 0 0;
		padding: 0;
		display: flex;
		flex-wrap: wrap;
		gap: 8px 16px;
		font-size: 13px;
		color: var(--tl-sub-2);
	}
	.leyenda li {
		display: inline-flex;
		align-items: center;
		gap: 6px;
	}
	.m {
		width: 14px;
		height: 14px;
		border-radius: 4px;
	}
	@media (max-width: 640px) {
		.c {
			height: 30px;
			font-size: 10px;
		}
		/* En el teléfono el porcentaje no entra: queda para el lector de pantalla. */
		.libre span[aria-hidden] {
			display: none;
		}
		.mapa tbody th,
		.mapa thead td {
			width: 34px;
		}
		.mapa {
			min-width: calc(34px + var(--cols) * 24px);
		}
	}
	@media print {
		.marco {
			overflow: visible;
		}
	}
</style>
