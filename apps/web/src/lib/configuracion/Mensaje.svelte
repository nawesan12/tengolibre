<script lang="ts">
	import { mensajeDe, type Form } from './mensajes.ts';

	/** El resultado de la acción (`form` de la página) y, si hay varios formularios, de cuál es. */
	let { form, que }: { form: Form; que?: string } = $props();
	const m = $derived(mensajeDe(form, que));
</script>

{#if m?.tipo === 'suscripcion'}
	<p class="aviso error" role="alert">
		{m.texto}
		<a href="/panel/suscripcion">Ir a la suscripción</a>
	</p>
{:else if m?.tipo === 'error'}
	<p class="aviso error" role="alert">{m.texto}</p>
{:else if m?.tipo === 'listo'}
	<p class="aviso" role="status">{m.texto}</p>
{/if}
