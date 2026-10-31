<!-- Una sección que no le toca a este rol (handoff «Estados y teléfono»: sin culpar, con un camino). -->
<script lang="ts">
	import Icono from '#lib/ui/Icono.svelte';
	let { que = 'esta sección' }: { que?: string } = $props();
</script>

<section class="sin" role="status">
	<Icono nombre="lock" tam={28} />
	<h2>No tenés acceso a {que}</h2>
	<p>Es para quien gestiona el negocio. Si lo necesitás, pedíselo a la dueña o al&nbsp;encargado.</p>
	<a href="/panel">Volver al inicio</a>
</section>

<style>
	.sin {
		padding: 40px 24px;
		border-radius: 20px;
		background: var(--tl-papel);
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 10px;
	}
	.sin :global(.icono) {
		color: var(--tl-menta-texto);
	}
	h2 {
		margin: 0;
		font-size: 22px;
		letter-spacing: -0.02em;
	}
	p {
		margin: 0;
		font-size: 14px;
		color: var(--tl-sub);
		max-width: 32em;
	}
	a {
		min-height: 44px;
		display: inline-flex;
		align-items: center;
		font-weight: 700;
		color: var(--tl-menta-texto);
	}
</style>
