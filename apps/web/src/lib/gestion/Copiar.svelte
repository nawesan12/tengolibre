<!-- Botón «Copiar» de un toque: copia el texto y avisa con ícono + palabra durante 2 s. -->
<script lang="ts">
	let { texto, etiqueta = 'Copiar' }: { texto: string; etiqueta?: string } = $props();
	let copiado = $state(false);

	async function copiar() {
		try {
			await navigator.clipboard.writeText(texto);
			copiado = true;
			setTimeout(() => (copiado = false), 2000);
		} catch {
			copiado = false;
		}
	}
</script>

<button type="button" class="g-boton chico copiar" onclick={copiar} aria-live="polite">
	{#if copiado}<span class="icono" aria-hidden="true">check</span>Copiado{:else}{etiqueta}{/if}
</button>

<style>
	.copiar {
		box-shadow: none;
		background: #fff;
	}
	.icono {
		font-size: 16px;
		color: var(--tl-menta-texto);
	}
</style>
