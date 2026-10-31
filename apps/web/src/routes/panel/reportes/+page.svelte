<script lang="ts">
	import { ESTADOS, dia, pesos } from '#lib/formato.ts';
	import { nombreDia, sumarDias } from '#lib/operacion/fechas.ts';
	import { escala, porcentaje } from '#lib/operacion/numeros.ts';
	import SinPermiso from '#lib/operacion/SinPermiso.svelte';
	import type { FilaReporte } from '#lib/operacion/tipos.ts';

	let { data } = $props();
	const r = $derived(data.reporte);
	const hoy = dia();
	const rapidos = [
		{ nombre: '7 días', desde: sumarDias(hoy, -6) },
		{ nombre: '30 días', desde: sumarDias(hoy, -29) },
		{ nombre: '90 días', desde: sumarDias(hoy, -89) },
		{ nombre: 'Este mes', desde: `${hoy.slice(0, 8)}01` }
	];
	const rango = $derived(r ? `desde=${r.desde}&hasta=${r.hasta}` : '');
	const barrasDia = $derived(r ? escala(r.porDia.map((d) => d.turnos)) : []);
	const totalEstados = $derived(r ? Object.values(r.porEstado).reduce((a, b) => a + b, 0) : 0);
</script>

<svelte:head><title>Reportes · Panel</title></svelte:head>

{#snippet tabla(titulo: string, filas: FilaReporte[], conOcupacion: boolean)}
	{@const barras = escala(filas.map((f) => f.ingresos))}
	<section class="tarjeta pila">
		<h2>{titulo}</h2>
		{#if filas.length === 0}
			<p class="suave">Sin turnos en el período.</p>
		{:else}
			<div class="desplazable">
				<table class="tabla">
					<thead>
						<tr>
							<th scope="col">Nombre</th>
							<th scope="col">Turnos</th>
							<th scope="col">Atendidos</th>
							<th scope="col">No vino</th>
							{#if conOcupacion}<th scope="col">Ocupación</th>{/if}
							<th scope="col">Ingresos</th>
						</tr>
					</thead>
					<tbody>
						{#each filas as f, i (f.id ?? f.nombre)}
							<tr>
								<th scope="row">{f.nombre || 'Sin nombre'}</th>
								<td>{f.turnos}</td>
								<td>{f.atendidos}</td>
								<td>{f.ausentes}</td>
								{#if conOcupacion}<td>{porcentaje(f.ocupacion)}</td>{/if}
								<td class="con-barra">
									{pesos(f.ingresos)}
									<span class="barra" style={`width: ${barras[i]}%`} aria-hidden="true"></span>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>
{/snippet}

<div class="pila">
	<h1>Reportes</h1>

	{#if data.error}<p class="aviso error" role="alert">{data.error}</p>{/if}

	{#if !r && !data.error}
		<SinPermiso que="los reportes" />
	{:else}
		<form method="GET" class="tarjeta fila" aria-label="Período">
			<label class="campo">Desde <input type="date" name="desde" value={r?.desde ?? ''} /></label>
			<label class="campo">Hasta <input type="date" name="hasta" value={r?.hasta ?? ''} /></label>
			<button class="boton" style="align-self: end">Ver</button>
			<div class="fila" style="align-self: end">
				{#each rapidos as x (x.nombre)}<a class="boton chico" href={`?desde=${x.desde}&hasta=${hoy}`}>{x.nombre}</a>{/each}
			</div>
		</form>
	{/if}

	{#if r}
		<p class="suave">Del {nombreDia(r.desde)} al {nombreDia(r.hasta)}. Ingresos es lo facturado: el precio de los turnos&nbsp;atendidos.</p>
		<div class="grilla numeros">
			<div class="tarjeta"><div class="suave chico">Ingresos</div><strong>{pesos(r.resumen.ingresos)}</strong></div>
			<div class="tarjeta"><div class="suave chico">Turnos</div><strong>{r.resumen.turnos}</strong></div>
			<div class="tarjeta"><div class="suave chico">Atendidos</div><strong>{r.resumen.atendidos}</strong></div>
			<div class="tarjeta"><div class="suave chico">Ticket promedio</div><strong>{pesos(r.resumen.ticketPromedio)}</strong></div>
			<div class="tarjeta"><div class="suave chico">Clientes</div><strong>{r.resumen.clientes}</strong> <span class="chico suave">({r.clientesNuevos} nuevos, {r.clientesRecurrentes}&nbsp;recurrentes)</span></div>
			<div class="tarjeta"><div class="suave chico">No vinieron</div><strong>{porcentaje(r.resumen.tasaAusencias)}</strong></div>
			<div class="tarjeta"><div class="suave chico">Ocupación</div><strong>{porcentaje(r.resumen.ocupacion)}</strong></div>
		</div>

		<section class="tarjeta pila">
			<h2>Turnos por día</h2>
			{#if r.porDia.every((d) => d.turnos === 0)}
				<p class="suave">Sin turnos en el período.</p>
			{:else}
				<div class="columnas" role="img" aria-label={`Turnos por día: ${r.porDia.map((d) => `${nombreDia(d.fecha)}, ${d.turnos}`).join('; ')}`}>
					{#each r.porDia as d, i (d.fecha)}
						<div class="col" title={`${nombreDia(d.fecha)}: ${d.turnos} turnos, ${pesos(d.ingresos)}`}>
							<span class="valor" style={`height: ${barrasDia[i]}%`}></span>
						</div>
					{/each}
				</div>
				<div class="fila chico suave" style="justify-content: space-between">
					<span>{nombreDia(r.porDia[0].fecha)}</span><span>{nombreDia(r.porDia[r.porDia.length - 1].fecha)}</span>
				</div>
			{/if}
		</section>

		<section class="tarjeta pila">
			<h2>Por estado</h2>
			{#if totalEstados === 0}
				<p class="suave">Sin turnos en el período.</p>
			{:else}
				<div class="apilada" aria-hidden="true">
					{#each Object.entries(r.porEstado) as [k, v] (k)}
						<span class="tramo {k}" style={`flex: ${v}`}></span>
					{/each}
				</div>
				<ul class="fila leyenda chico">
					{#each Object.entries(r.porEstado) as [k, v] (k)}
						<li><span class="punto {k}" aria-hidden="true"></span>{ESTADOS[k] ?? k}: {v}</li>
					{/each}
				</ul>
			{/if}
		</section>

		{@render tabla('Por profesional', r.porProfesional, true)}
		{@render tabla('Por servicio', r.porServicio, false)}

		<section class="tarjeta pila">
			<h2>Descargas</h2>
			<p class="suave chico">Planillas CSV (se abren en Excel o Google&nbsp;Sheets).</p>
			<div class="fila">
				<a class="boton" href={`/panel/reportes/csv/turnos?${rango}`} download data-sveltekit-reload>Turnos del período</a>
				<a class="boton" href={`/panel/reportes/csv/clientes?${rango}`} download data-sveltekit-reload>Clientes</a>
			</div>
		</section>
	{/if}
</div>

<style>
	.numeros {
		grid-template-columns: repeat(auto-fill, minmax(min(160px, 100%), 1fr));
	}
	.numeros strong {
		font-size: 20px;
	}
	.con-barra {
		position: relative;
		min-width: 120px;
	}
	.barra {
		display: block;
		height: 4px;
		margin-top: 4px;
		border-radius: 2px;
		background: var(--marca);
	}
	.columnas {
		display: flex;
		align-items: flex-end;
		gap: 2px;
		height: 140px;
		border-bottom: 1px solid var(--borde);
	}
	.col {
		flex: 1;
		height: 100%;
		display: flex;
		align-items: flex-end;
	}
	.valor {
		display: block;
		width: 100%;
		min-height: 1px;
		background: var(--marca);
		border-radius: 3px 3px 0 0;
	}
	.apilada {
		display: flex;
		height: 14px;
		border-radius: 7px;
		overflow: hidden;
	}
	.leyenda {
		list-style: none;
		margin: 0;
		padding: 0;
		gap: 12px;
	}
	.punto {
		display: inline-block;
		width: 10px;
		height: 10px;
		border-radius: 50%;
		margin-right: 4px;
	}
	.tramo,
	.punto {
		background: var(--marca);
	}
	.pendiente,
	.pendiente_pago {
		background: var(--aviso);
	}
	.atendido {
		background: var(--suave);
	}
	.ausente {
		background: var(--peligro);
	}
	.cancelado {
		background: var(--borde);
	}
</style>
