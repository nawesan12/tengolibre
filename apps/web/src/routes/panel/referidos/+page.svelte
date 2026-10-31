<script lang="ts">
	import { enhance } from '$app/forms';
	import { fecha } from '#lib/formato.ts';
	import { sinReset } from '#lib/configuracion/formularios.ts';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';

	let { data, form } = $props();
	// svelte-ignore state_referenced_locally
	let premio = $state(data.config.tipo !== null);
</script>

<svelte:head><title>Referidos · Panel</title></svelte:head>

<div class="pila">
	<h1>Referidos</h1>
	<p class="suave">Cada cliente puede pedir su código desde el link de su turno y compartirlo. Quien lo usa reserva, y quien lo compartió se lleva un&nbsp;premio.</p>

	<section class="tarjeta pila" aria-labelledby="t-programa">
		<h2 id="t-programa">El programa</h2>
		<Mensaje {form} que="config" />
		<form method="POST" action="?/config" use:enhance={sinReset} class="grilla">
			<label class="fila ancho"><input type="checkbox" name="activo" checked={data.config.activo} /> Programa de referidos activo</label>
			<label class="fila ancho"><input type="checkbox" name="premio" bind:checked={premio} /> Premiar a quien refiere con un cupón de un&nbsp;uso</label>
			{#if premio}
				<label class="campo">
					Tipo de premio
					<select name="tipo">
						<option value="porcentaje" selected={data.config.tipo !== 'fijo'}>Porcentaje</option>
						<option value="fijo" selected={data.config.tipo === 'fijo'}>Monto fijo</option>
					</select>
				</label>
				<label class="campo">Descuento <input name="valor" type="number" min="1" value={data.config.valor ?? ''} required /></label>
			{/if}
			<label class="campo">
				Usos por código
				<input name="tope" type="number" min="1" value={data.config.tope ?? ''} placeholder="Sin límite" />
			</label>
			<div class="ancho"><button class="boton primario">Guardar</button></div>
		</form>
	</section>

	<section class="tarjeta pila" aria-labelledby="t-codigos">
		<h2 id="t-codigos">Códigos de tus clientes</h2>
		{#if data.referidos.length === 0}
			<p class="suave">Todavía ningún cliente pidió su código.</p>
		{:else}
			<div class="desplazable">
				<table class="tabla">
					<thead><tr><th>Cliente</th><th>Código</th><th>Usos</th><th>Desde</th></tr></thead>
					<tbody>
						{#each data.referidos as r (r.id)}
							<tr><td>{r.cliente}</td><td><code>{r.codigo}</code></td><td>{r.usos}</td><td>{fecha(r.creadoEn)}</td></tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>
</div>

<style>
	.ancho {
		grid-column: 1 / -1;
	}
</style>
