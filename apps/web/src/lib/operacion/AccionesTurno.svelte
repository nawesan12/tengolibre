<script lang="ts">
	import { enhance } from '$app/forms';
	import { accionesRapidas } from './estados.ts';

	/**
	 * Los botones de estado de un turno (confirmar, atendido, no vino, cancelar). Mandan a la acción
	 * `estado` de la página donde están (o a la que se indique).
	 */
	let { id, estado, accion = '?/estado', quien = '' }: { id: string; estado: string; accion?: string; quien?: string } = $props();
	const acciones = $derived(accionesRapidas(estado));
	let enviando = $state(false);
</script>

{#if acciones.length > 0}
	<form
		method="POST"
		action={accion}
		class="fila"
		use:enhance={() => {
			enviando = true;
			return async ({ update }) => {
				await update({ reset: false });
				enviando = false;
			};
		}}
	>
		<input type="hidden" name="id" value={id} />
		{#each acciones as a (a.estado)}
			<button
				class="boton chico"
				class:peligro={a.estado === 'cancelado'}
				name="estado"
				value={a.estado}
				disabled={enviando}
				aria-label={quien ? `${a.texto}: ${quien}` : undefined}
				onclick={(e) => {
					if (a.estado === 'cancelado' && !confirm('¿Cancelar este turno? Se le avisa al cliente.')) e.preventDefault();
				}}>{a.texto}</button
			>
		{/each}
	</form>
{/if}
