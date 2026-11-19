<script lang="ts">
	import '#lib/gestion/gestion.css';
	import { linkPlanes, nombrePlan, QUE_VIENE, vieneCon, type AccesoPlan, type Funcion } from '#lib/planes.ts';

	/**
	 * Arriba de una pantalla cuya función el plan que rige no trae: lo dice antes de que se complete el
	 * formulario (el servidor igual lo frena), sin esconder nada. Lo ya configurado queda guardado.
	 */
	let { acceso, funcion, detalle }: { acceso: AccesoPlan; funcion: Funcion; detalle?: string } = $props();
	const plan = $derived(vieneCon(acceso, funcion));
</script>

{#if plan}
	<p class="g-aviso ambar" role="status">
		<span class="icono" aria-hidden="true">workspace_premium</span>
		<span
			>{QUE_VIENE[funcion]} con {nombrePlan(plan)}. {detalle ?? 'Podés ver lo que ya está; para crear o cambiar, pasá de\u00a0plan.'}</span
		>
		<a class="g-boton" href={linkPlanes(plan)}>Pasar a {nombrePlan(plan)}</a>
	</p>
{/if}
