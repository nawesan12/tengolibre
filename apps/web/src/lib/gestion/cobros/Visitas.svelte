<!-- Las visitas de una membresía como puntos (llenos: las que quedan) + «3 DE 4». Para lectores de
	pantalla, una sola frase: «le quedan 3 de 4 visitas». -->
<script lang="ts">
	let { quedan, total }: { quedan: number; total: number } = $props();
	const pocas = $derived(total > 0 && quedan <= Math.max(1, Math.floor(total / 4)));
</script>

{#if total === 0}
	<span class="valor">VISITAS LIBRES</span>
{:else}
	<span class="puntos" aria-hidden="true">
		{#each Array.from({ length: Math.min(total, 12) }, (_, i) => i) as i (i)}
			<span class="punto" class:lleno={i < quedan}></span>
		{/each}
	</span>
	<span class="valor" class:pocas aria-hidden="true">{quedan} DE {total}</span>
	<span class="visualmente-oculto">le quedan {quedan} de {total} visitas</span>
{/if}

<style>
	.puntos {
		display: inline-flex;
		flex-wrap: wrap;
		gap: 3px;
	}
	.punto {
		width: 12px;
		height: 12px;
		border-radius: 50%;
		box-shadow: inset 0 0 0 1.5px var(--tl-petroleo);
	}
	.punto.lleno {
		background: var(--tl-petroleo);
	}
	.valor {
		font: 400 12px var(--tl-font);
		font-variant-numeric: tabular-nums;
		color: var(--tl-menta-texto);
		white-space: nowrap;
	}
	.valor.pocas {
		color: #7a4a00;
	}
</style>
