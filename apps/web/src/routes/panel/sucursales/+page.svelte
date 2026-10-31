<script lang="ts">
	import { enhance } from '$app/forms';
	import { pesos } from '#lib/formato.ts';
	import { dma } from '#lib/cuenta/util.ts';
	let { data, form } = $props();
	let renombrando = $state<string | null>(null);
	const esDuenia = $derived(data.grupos.some((g) => g.sucursales.length > 0) || data.sinGrupo.length > 0);
	const columnas = [
		{ k: 'turnos', nombre: 'Turnos' },
		{ k: 'atendidos', nombre: 'Atendidos' },
		{ k: 'ausentes', nombre: 'Ausentes' },
		{ k: 'cancelados', nombre: 'Cancelados' },
		{ k: 'clientes', nombre: 'Clientes' }
	] as const;
</script>

<svelte:head><title>Sucursales · Panel</title></svelte:head>

<div class="pila">
	<h1>Sucursales</h1>
	<p class="suave">
		Agrupá los negocios de los que sos dueña o dueño para verlos juntos y sacar un reporte de todas
		las&nbsp;sedes.
	</p>
	{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}

	<section class="tarjeta pila">
		<h2>Nuevo grupo</h2>
		<form method="POST" action="?/crear" use:enhance class="fila">
			<label class="campo crece">Nombre del grupo <input name="nombre" required maxlength="80" /></label>
			<button class="boton primario">Crear grupo</button>
		</form>
	</section>

	{#if data.grupos.length === 0}
		<p class="tarjeta suave">Todavía no armaste ningún grupo.</p>
	{/if}

	{#each data.grupos as g (g.id)}
		<section class="tarjeta pila" aria-label={`Grupo ${g.nombre}`}>
			{#if renombrando === g.id}
				<form
					method="POST"
					action="?/renombrar"
					use:enhance={() => async ({ update, result }) => {
						await update();
						if (result.type === 'success') renombrando = null;
					}}
					class="fila"
				>
					<input type="hidden" name="id" value={g.id} />
					<label class="campo crece">Nombre del grupo <input name="nombre" value={g.nombre} required maxlength="80" /></label>
					<button class="boton primario">Guardar</button>
					<button type="button" class="boton" onclick={() => (renombrando = null)}>Cancelar</button>
				</form>
			{:else}
				<div class="fila" style="justify-content: space-between">
					<h2>{g.nombre}</h2>
					<div class="fila">
						<a class="boton" href={`?grupo=${g.id}#reporte`}>Ver reporte</a>
						<button class="boton" onclick={() => (renombrando = g.id)}>Renombrar</button>
						<form
							method="POST"
							action="?/borrar"
							use:enhance={({ cancel }) => {
								if (!confirm(`¿Borrar el grupo ${g.nombre}? Los negocios quedan, sólo salen del grupo.`)) cancel();
							}}
						>
							<input type="hidden" name="id" value={g.id} />
							<button class="boton peligro">Borrar</button>
						</form>
					</div>
				</div>
			{/if}

			{#if g.sucursales.length === 0}
				<p class="suave chico">Sin sucursales todavía.</p>
			{:else}
				<ul class="lista">
					{#each g.sucursales as s (s.id)}
						<li>
							<span><strong>{s.nombre}</strong>{#if s.ciudad}<span class="suave chico"> · {s.ciudad}</span>{/if}</span>
							<form method="POST" action="?/sacar" use:enhance>
								<input type="hidden" name="id" value={g.id} />
								<input type="hidden" name="negocio" value={s.id} />
								<button class="boton" aria-label={`Sacar ${s.nombre} de ${g.nombre}`}>Sacar</button>
							</form>
						</li>
					{/each}
				</ul>
			{/if}

			{#if data.sinGrupo.length > 0}
				<form method="POST" action="?/sumar" use:enhance class="fila">
					<input type="hidden" name="id" value={g.id} />
					<label class="campo crece">
						Sumar un negocio a {g.nombre}
						<select name="negocio" required>
							{#each data.sinGrupo as s (s.id)}<option value={s.id}>{s.nombre}</option>{/each}
						</select>
					</label>
					<button class="boton">Sumar</button>
				</form>
			{/if}
		</section>
	{/each}

	{#if !esDuenia}
		<p class="aviso">Sólo se pueden agrupar los negocios de los que sos dueña o&nbsp;dueño.</p>
	{:else if data.sinGrupo.length > 0}
		<p class="suave chico">Sin grupo: {data.sinGrupo.map((s) => s.nombre).join(', ')}.</p>
	{/if}

	{#if data.grupos.length > 0}
		<section id="reporte" class="tarjeta pila">
			<h2>Reporte consolidado</h2>
			<form method="GET" class="grilla">
				<label class="campo">
					Grupo
					<select name="grupo">
						{#each data.grupos as g (g.id)}<option value={g.id} selected={g.id === data.grupoId}>{g.nombre}</option>{/each}
					</select>
				</label>
				<label class="campo">Desde <input type="date" name="desde" value={data.desde} required /></label>
				<label class="campo">Hasta <input type="date" name="hasta" value={data.hasta} required /></label>
				<button class="boton">Ver reporte</button>
			</form>
			{#if data.errorReporte}
				<p class="aviso error" role="alert">{data.errorReporte}</p>
			{:else if data.reporte}
				<p class="suave chico">{data.reporte.grupo}, del {dma(data.reporte.desde)} al {dma(data.reporte.hasta)}.</p>
				{#if data.reporte.sucursales.length === 0}
					<p class="suave">Este grupo no tiene sucursales.</p>
				{:else}
					<div class="desplazable">
						<table class="tabla">
							<caption class="visualmente-oculto">Turnos e ingresos por sucursal</caption>
							<thead>
								<tr>
									<th scope="col">Sucursal</th>
									{#each columnas as c (c.k)}<th scope="col" class="num">{c.nombre}</th>{/each}
									<th scope="col" class="num">Ingresos</th>
								</tr>
							</thead>
							<tbody>
								{#each data.reporte.sucursales as s (s.id)}
									<tr>
										<th scope="row">{s.nombre}</th>
										{#each columnas as c (c.k)}<td class="num">{s[c.k]}</td>{/each}
										<td class="num">{pesos(s.ingresos)}</td>
									</tr>
								{/each}
							</tbody>
							<tfoot>
								<tr>
									<th scope="row">Total</th>
									{#each columnas as c (c.k)}<td class="num">{data.reporte.total[c.k]}</td>{/each}
									<td class="num">{pesos(data.reporte.total.ingresos)}</td>
								</tr>
							</tfoot>
						</table>
					</div>
					<p class="suave chico">Turnos sin cancelados ni los que esperan el pago; ingresos de los&nbsp;atendidos.</p>
				{/if}
			{/if}
		</section>
	{/if}
</div>

<style>
	.crece {
		flex: 1 1 200px;
	}
	h2 {
		margin: 0;
	}
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.lista li {
		display: flex;
		gap: 12px;
		align-items: center;
		justify-content: space-between;
		padding: 8px 0;
		border-bottom: 1px solid var(--borde);
	}
	.num {
		text-align: right;
		font-variant-numeric: tabular-nums;
		white-space: nowrap;
	}
	tfoot th,
	tfoot td {
		font-weight: 650;
	}
	.grilla {
		align-items: end;
	}
</style>
