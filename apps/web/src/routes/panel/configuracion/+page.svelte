<script lang="ts">
	import { enhance } from '$app/forms';
	import { INTERVALOS } from '#lib/configuracion/textos.ts';
	import { sinReset } from '#lib/configuracion/formularios.ts';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';

	let { data, form } = $props();
	const a = $derived(data.ajustes);
</script>

<svelte:head><title>Configuración · Panel</title></svelte:head>

<div class="pila">
	<h1>Configuración</h1>
	<nav class="fila" aria-label="Más configuración">
		<a class="boton" href="/panel/auditoria">Registro de cambios</a>
		<a class="boton" href="/panel/sucursales">Sucursales</a>
		<a class="boton" href="/panel/pagos">Cobros</a>
		<a class="boton" href="/panel/suscripcion">Suscripción</a>
	</nav>

	<section class="tarjeta pila" aria-labelledby="t-reserva">
		<h2 id="t-reserva">Reservas</h2>
		<Mensaje {form} que="reserva" />
		<form method="POST" action="?/reserva" use:enhance={sinReset} class="grilla">
			<label class="campo">
				Los turnos arrancan cada
				<select name="intervalo">
					{#each INTERVALOS as i (i)}<option value={i} selected={i === a.intervalo}>{i} minutos</option>{/each}
				</select>
			</label>
			<label class="campo">
				Anticipación mínima (horas)
				<input name="anticipacionHoras" type="number" min="0" max="168" step="0.5" value={a.anticipacionMin / 60} required />
				<span class="suave chico">Con cuánto tiempo antes se puede reservar.</span>
			</label>
			<label class="campo">
				Se puede reservar hasta (días)
				<input name="diasVista" type="number" min="1" max="180" value={a.diasVista} required />
			</label>
			<label class="campo">
				Margen entre turnos (minutos)
				<input name="margenMin" type="number" min="0" max="120" step="5" value={a.margenMin} required />
				<span class="suave chico">Tiempo libre después de cada turno para preparar.</span>
			</label>
			<label class="campo">
				Se puede cancelar hasta (horas antes)
				<input name="cancelarHastaHoras" type="number" min="0" max="168" value={a.cancelarHastaHoras} required />
			</label>
			<label class="campo ancho">
				Política de cancelación
				<textarea name="politicaCancelacion" rows="3" maxlength="2000">{a.politicaCancelacion}</textarea>
				<span class="suave chico">La ve el cliente al reservar.</span>
			</label>
			<div><button class="boton primario">Guardar reservas</button></div>
		</form>
	</section>

	<section class="tarjeta pila" aria-labelledby="t-avisos">
		<h2 id="t-avisos">Avisos</h2>
		<Mensaje {form} que="avisos" />
		<form method="POST" action="?/avisos" use:enhance={sinReset} class="grilla">
			<label class="campo">
				Pedido de reseña (horas después del turno)
				<input name="resenaDemoraHoras" type="number" min="0" max="168" value={data.avisos.resenaDemoraHoras} required />
			</label>
			<label class="fila chico" style="align-self: end">
				<input type="checkbox" name="resumenDiario" checked={data.avisos.resumenDiario} />
				Mandarme a la tarde el resumen de los turnos de&nbsp;mañana
			</label>
			<div class="ancho"><button class="boton primario">Guardar avisos</button></div>
		</form>
	</section>

	<section class="tarjeta pila" aria-labelledby="t-ausencias">
		<h2 id="t-ausencias">Ausencias y penalidades</h2>
		<p class="suave chico">Quien falta sin avisar varias veces puede quedar un tiempo sin poder reservar&nbsp;online.</p>
		<Mensaje {form} que="ausencias" />
		<form method="POST" action="?/ausencias" use:enhance={sinReset} class="grilla">
			<label class="campo">
				Ausencias en 90 días para penalizar
				<input name="umbralAusencias" type="number" min="0" max="20" value={data.ausencias.umbralAusencias} required />
				<span class="suave chico">0: nunca se penaliza solo.</span>
			</label>
			<label class="campo">
				Días de penalidad
				<input name="diasPenalidad" type="number" min="1" max="365" value={data.ausencias.diasPenalidad} required />
			</label>
			<label class="fila chico" style="align-self: end">
				<input type="checkbox" name="marcarAusentes" checked={data.ausencias.marcarAusentes} />
				Marcar como ausente un turno que nadie cerró
			</label>
			<div class="ancho"><button class="boton primario">Guardar ausencias</button></div>
		</form>
	</section>
</div>

<style>
	.ancho {
		grid-column: 1 / -1;
	}
</style>
