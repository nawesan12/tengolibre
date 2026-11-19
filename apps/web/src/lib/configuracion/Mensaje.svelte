<script lang="ts">
	import '#lib/gestion/gestion.css';
	import { linkPlanes, nombrePlan } from '#lib/planes.ts';
	import { mensajeDe, type Form } from './mensajes.ts';

	/** El resultado de la acción (`form` de la página) y, si hay varios formularios, de cuál es. */
	let { form, que }: { form: Form; que?: string } = $props();
	const m = $derived(mensajeDe(form, que));

	/**
	 * Un aviso de tope o de error sale arriba del formulario: en el teléfono quedaba fuera de la
	 * pantalla y parecía que el botón no hacía nada. Al aparecer, se lleva a la vista y recibe el foco
	 * (los lectores de pantalla lo leen igual por role=alert).
	 */
	function aLaVista(nodo: HTMLElement) {
		const quieto = matchMedia('(prefers-reduced-motion: reduce)').matches;
		nodo.scrollIntoView({ block: 'center', behavior: quieto ? 'auto' : 'smooth' });
		nodo.focus({ preventScroll: true });
	}
</script>

{#if m?.tipo === 'suscripcion'}
	<p class="aviso error" role="alert" tabindex="-1" use:aLaVista>
		{m.texto}
		<a href="/panel/suscripcion">Ir a la suscripción</a>
	</p>
{:else if m?.tipo === 'tope'}
	<p class="g-aviso ambar tope" role="alert" tabindex="-1" use:aLaVista>
		<span class="icono" aria-hidden="true">workspace_premium</span>
		<span>{m.texto}</span>
		<a class="g-boton" href={linkPlanes(m.plan)}>{m.plan ? `Pasar a ${nombrePlan(m.plan)}` : 'Ver planes'}</a>
	</p>
{:else if m?.tipo === 'error'}
	<p class="aviso error" role="alert" tabindex="-1" use:aLaVista>{m.texto}</p>
{:else if m?.tipo === 'listo'}
	<p class="aviso" role="status">{m.texto}</p>
{/if}

<style>
	.tope:focus {
		outline: none;
	}
	.tope:focus-visible {
		outline: 2px solid var(--g-foco, currentColor);
		outline-offset: 2px;
	}
</style>
