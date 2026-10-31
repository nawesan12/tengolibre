<script lang="ts">
	import type { Horario } from './tipos.ts';

	let { horarios, href }: { horarios: Horario[]; href: (h: Horario) => string } = $props();

	// Mañana, tarde y noche, como en Jiku: una lista larga de horas se lee mejor partida.
	const franjas = $derived(
		[
			{ nombre: 'Mañana', hs: horarios.filter((h) => h.hora < '12:00') },
			{ nombre: 'Tarde', hs: horarios.filter((h) => h.hora >= '12:00' && h.hora < '19:00') },
			{ nombre: 'Noche', hs: horarios.filter((h) => h.hora >= '19:00') }
		].filter((f) => f.hs.length)
	);
</script>

{#each franjas as f (f.nombre)}
	<div class="pila" style="gap: 6px">
		<h3 class="chico suave" style="margin: 0">{f.nombre}</h3>
		<ul class="horarios">
			{#each f.hs as h (h.inicio)}
				<li><a class="boton" href={href(h)}>{h.hora}</a></li>
			{/each}
		</ul>
	</div>
{/each}

<style>
	.horarios {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		gap: 8px;
		grid-template-columns: repeat(auto-fill, minmax(76px, 1fr));
	}
	.horarios .boton {
		width: 100%;
		min-height: 44px;
	}
</style>
