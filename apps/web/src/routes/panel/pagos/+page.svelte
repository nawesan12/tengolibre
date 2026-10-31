<script lang="ts">
	import { enhance } from '$app/forms';
	import { page } from '$app/state';
	import { ESTADOS, fechaHora, fecha, pesos } from '#lib/formato.ts';
	import { MODOS_COBRO, resta } from '#lib/configuracion/dinero.ts';
	import { ESTADOS_PAGO } from '#lib/configuracion/textos.ts';
	import { confirmar, sinReset } from '#lib/configuracion/formularios.ts';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';

	let { data, form } = $props();
	const c = $derived(data.conexion);
	const esDuenio = $derived(data.negocio.rol === 'duenio');
	// svelte-ignore state_referenced_locally
	let modo = $state(data.config.modo);
	let devolviendo = $state<string | null>(null);
	let cuanto = $state<'todo' | 'parte'>('todo');
	const recienConectada = $derived(page.url.searchParams.get('mercadopago') === 'conectada');
</script>

<svelte:head><title>Pagos · Panel</title></svelte:head>

<div class="pila">
	<h1>Pagos</h1>

	<section class="tarjeta pila" aria-labelledby="t-mp">
		<h2 id="t-mp">Mercado Pago</h2>
		{#if recienConectada}<p class="aviso" role="status">Listo: conectaste tu cuenta de Mercado&nbsp;Pago.</p>{/if}
		<Mensaje {form} que="mercadopago" />
		{#if !c.disponible}
			<p class="aviso">
				Los cobros con Mercado Pago todavía no están habilitados en TengoLibre. Mientras tanto, los turnos se reservan
				sin cobro por&nbsp;adelantado.
			</p>
		{:else if c.conectada}
			<p>
				<span class="etiqueta">Conectada</span>
				{#if c.desde}<span class="suave chico">desde el {fecha(c.desde)}</span>{/if}
			</p>
			{#if c.reautorizar}
				<p class="aviso error">Mercado Pago pide que vuelvas a conectar la cuenta para seguir&nbsp;cobrando.</p>
			{/if}
			{#if c.ultimoError}<p class="suave chico">Último problema: {c.ultimoError}</p>{/if}
			{#if esDuenio}
				<div class="fila">
					{#if c.reautorizar}
						<form method="POST" action="?/conectar"><button class="boton primario">Volver a conectar</button></form>
					{/if}
					<form method="POST" action="?/desconectar" use:enhance={confirmar('¿Desconectar Mercado Pago? Los turnos nuevos no se van a cobrar por adelantado.')}>
						<button class="boton peligro">Desconectar</button>
					</form>
				</div>
			{/if}
		{:else}
			<p class="suave">Conectá tu cuenta para cobrar señas o el total al reservar. La plata entra directo a tu&nbsp;cuenta.</p>
			{#if esDuenio}
				<!-- Sin enhance: la respuesta es una redirección a Mercado Pago. -->
				<form method="POST" action="?/conectar"><button class="boton primario">Conectar Mercado Pago</button></form>
			{:else}
				<p class="suave chico">La cuenta la conecta la dueña o el dueño del negocio.</p>
			{/if}
		{/if}
	</section>

	<section class="tarjeta pila" aria-labelledby="t-cobro">
		<h2 id="t-cobro">Cómo cobrás al reservar</h2>
		<Mensaje {form} que="cobro" />
		{#if modo !== 'sin_cobro' && !c.conectada}
			<p class="aviso">Hasta que conectes Mercado Pago, los turnos se reservan sin&nbsp;cobro.</p>
		{/if}
		<form method="POST" action="?/cobro" use:enhance={sinReset} class="pila">
			<fieldset class="opciones">
				<legend class="visualmente-oculto">Modo de cobro</legend>
				{#each Object.entries(MODOS_COBRO) as [clave, nombre] (clave)}
					<label class="fila"><input type="radio" name="modo" value={clave} bind:group={modo} /> {nombre}</label>
				{/each}
			</fieldset>
			{#if modo === 'porcentaje'}
				<label class="campo corto">
					Porcentaje de la seña
					<input name="senaPorcentaje" type="number" min="1" max="100" value={data.config.senaPorcentaje} required />
				</label>
			{:else if modo === 'fija'}
				<label class="campo corto">
					Monto de la seña ($)
					<input name="senaFija" type="number" min="1" step="1" value={data.config.senaFija || ''} required />
				</label>
			{/if}
			<div><button class="boton primario">Guardar</button></div>
		</form>
	</section>

	<section class="tarjeta pila" aria-labelledby="t-pagos">
		<div class="fila entre">
			<h2 id="t-pagos" style="margin: 0">Pagos recibidos</h2>
			<form method="GET" class="fila">
				<label class="campo">
					<span class="visualmente-oculto">Estado</span>
					<select name="estado" onchange={(e) => e.currentTarget.form?.requestSubmit()}>
						<option value="">Todos</option>
						{#each Object.entries(ESTADOS_PAGO) as [clave, nombre] (clave)}
							<option value={clave} selected={clave === data.estado}>{nombre}</option>
						{/each}
					</select>
				</label>
				<noscript><button class="boton">Filtrar</button></noscript>
			</form>
		</div>
		<Mensaje {form} que="pagos" />
		{#if data.pagos.length === 0}
			<p class="suave">Todavía no hay pagos{data.estado ? ' con ese estado' : ''}.</p>
		{:else}
			<ul class="lista">
				{#each data.pagos as p (p.id)}
					<li class="pila">
						<div class="fila entre">
							<div>
								<strong>{pesos(p.monto)}</strong>
								<span class="etiqueta">{p.vencido ? 'Vencido' : (ESTADOS_PAGO[p.estado] ?? p.estado)}</span>
								{#if p.devuelto > 0}<span class="suave chico">devueltos {pesos(p.devuelto)}</span>{/if}
								<div class="suave chico">
									{p.cliente} · {p.servicio} · turno {fechaHora(p.inicio)} ({ESTADOS[p.turnoEstado] ?? p.turnoEstado})
								</div>
							</div>
							{#if p.estado === 'aprobado' && p.mpPago && resta(p) > 0 && devolviendo !== p.id}
								<button class="boton" onclick={() => ((devolviendo = p.id), (cuanto = 'todo'))}>Devolver</button>
							{/if}
						</div>
						{#if devolviendo === p.id}
							<form
								method="POST"
								action="?/devolver"
								use:enhance={confirmar('¿Devolver la plata? No se puede deshacer.')}
								class="pila devolucion"
							>
								<input type="hidden" name="id" value={p.id} />
								<label class="fila"><input type="radio" name="cuanto" value="todo" bind:group={cuanto} /> Todo ({pesos(resta(p))}) y cancelar el turno</label>
								<label class="fila"><input type="radio" name="cuanto" value="parte" bind:group={cuanto} /> Una parte</label>
								{#if cuanto === 'parte'}
									<label class="campo corto">Monto ($) <input name="monto" type="number" min="1" max={resta(p) - 1} required /></label>
								{/if}
								<div class="fila">
									<button class="boton primario">Devolver</button>
									<button type="button" class="boton" onclick={() => (devolviendo = null)}>Cancelar</button>
								</div>
							</form>
						{/if}
					</li>
				{/each}
			</ul>
		{/if}
	</section>
</div>

<style>
	.entre {
		justify-content: space-between;
	}
	.opciones {
		border: 0;
		padding: 0;
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.corto {
		max-width: 260px;
	}
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.lista li {
		padding: 10px 0;
		border-bottom: 1px solid var(--borde);
	}
	.devolucion {
		padding: 12px;
		border: 1px solid var(--borde);
		border-radius: 8px;
	}
</style>
