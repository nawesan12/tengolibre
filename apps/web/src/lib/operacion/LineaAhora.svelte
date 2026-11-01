<!-- La línea menta de «ahora» entre los turnos (handoff «Panel v2»). -->
<script lang="ts">
	let { hora, rotulo = false }: { hora: string; rotulo?: boolean } = $props();
</script>

<div class="ahora" role="separator" aria-label={`Ahora, ${hora}`}>
	<span class="pastilla tl-num"><span class="punto"></span>{hora}</span>
	<span class="raya"></span>
	{#if rotulo}<span class="txt">ahora</span>{/if}
</div>

<style>
	.ahora {
		display: flex;
		align-items: center;
		gap: 10px;
		height: 16px;
	}
	.pastilla {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		height: 22px;
		padding: 0 8px;
		border-radius: 999px;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		font-size: 11px;
		font-weight: 700;
	}
	.punto {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: var(--tl-petroleo-hondo);
		animation: tl-pulse 1.8s infinite;
	}
	.raya {
		flex: 1;
		height: 2px;
		border-radius: 1px;
		background: var(--tl-menta);
	}
	.txt {
		font-size: 11px;
		font-weight: 700;
		color: var(--tl-menta-texto);
	}
</style>
