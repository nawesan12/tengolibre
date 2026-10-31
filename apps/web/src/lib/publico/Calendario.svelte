<script lang="ts">
	import { MESES, semanas } from './calendario.ts';
	import type { Dia } from './tipos.ts';

	let { dias, href, elegida = '' }: { dias: Dia[]; href: (fecha: string) => string; elegida?: string } = $props();

	const filas = $derived(semanas(dias));
	const titulo = $derived.by(() => {
		if (!dias.length) return '';
		const a = new Date(dias[0].fecha + 'T12:00:00Z').getUTCMonth();
		const b = new Date(dias[dias.length - 1].fecha + 'T12:00:00Z').getUTCMonth();
		return a === b ? MESES[a] : `${MESES[a]} y ${MESES[b]}`;
	});
	const etiqueta = (f: string, libres: number) => {
		const d = new Date(f + 'T12:00:00Z');
		const txt = d.toLocaleDateString('es-AR', { weekday: 'long', day: 'numeric', month: 'long', timeZone: 'UTC' });
		return libres ? `${txt}: ${libres} ${libres === 1 ? 'horario libre' : 'horarios libres'}` : `${txt}: sin horarios`;
	};
</script>

<div class="calendario" role="group" aria-label={`Días de ${titulo}`}>
	<p class="mes">{titulo}</p>
	<div class="semana encabezado" aria-hidden="true">
		{#each ['L', 'M', 'M', 'J', 'V', 'S', 'D'] as l, i (i)}<span>{l}</span>{/each}
	</div>
	{#each filas as semana, i (i)}
		<div class="semana">
			{#each semana as c, j (j)}
				{#if !c}
					<span></span>
				{:else if c.libres > 0}
					<a
						class="dia"
						class:elegido={c.fecha === elegida}
						href={href(c.fecha)}
						aria-label={etiqueta(c.fecha, c.libres)}
						aria-current={c.fecha === elegida ? 'date' : undefined}
						data-fecha={c.fecha}>{c.numero}</a
					>
				{:else}
					<span class="dia lleno" aria-label={etiqueta(c.fecha, 0)} data-fecha={c.fecha}>{c.numero}</span>
				{/if}
			{/each}
		</div>
	{/each}
</div>

<style>
	.calendario {
		display: flex;
		flex-direction: column;
		gap: 4px;
		max-width: 420px;
	}
	.mes {
		margin: 0 0 4px;
		font-weight: 600;
	}
	.mes::first-letter {
		text-transform: uppercase;
	}
	.semana {
		display: grid;
		grid-template-columns: repeat(7, 1fr);
		gap: 4px;
	}
	.encabezado span {
		text-align: center;
		font-size: 12px;
		color: var(--suave);
	}
	.dia {
		display: grid;
		place-items: center;
		min-height: 44px;
		border-radius: 8px;
		text-decoration: none;
		font-weight: 550;
		border: 1px solid var(--borde);
		background: var(--superficie);
	}
	a.dia:hover {
		border-color: var(--marca);
	}
	.dia.elegido {
		background: var(--marca);
		border-color: var(--marca);
		color: var(--marca-tinta);
	}
	.dia.lleno {
		color: var(--suave);
		background: transparent;
		border-color: transparent;
		text-decoration: line-through;
		font-weight: 400;
	}
</style>
