<script lang="ts">
	import Icono from '#lib/ui/Icono.svelte';

	/** Los requisitos de la contraseña nueva, que se tildan solos mientras se escribe (calcado de Corredor). */
	let { req }: { req: { largo: boolean; comun: boolean } } = $props();
</script>

<ul aria-live="polite">
	<li class:ok={req.largo}><Icono nombre={req.largo ? 'check_circle' : 'radio_button_unchecked'} lleno={req.largo} tam={18} />8 caracteres o más</li>
	<li class:ok={req.comun}><Icono nombre={req.comun ? 'check_circle' : 'radio_button_unchecked'} lleno={req.comun} tam={18} />No es una de las más&nbsp;usadas</li>
</ul>

<style>
	ul {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
		font-size: 14px;
	}
	li {
		display: flex;
		align-items: center;
		gap: 8px;
		opacity: 0.75;
	}
	li.ok {
		opacity: 1;
	}
	li.ok :global(.icono) {
		color: var(--tl-menta);
	}
</style>
