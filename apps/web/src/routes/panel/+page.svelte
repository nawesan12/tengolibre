<script lang="ts">
	import { pesos, fechaHora, ESTADOS } from '#lib/formato.ts';
	let { data } = $props();
</script>

<svelte:head><title>Inicio · Panel</title></svelte:head>

<div class="pila">
	<h1>Hola{data.persona.nombre ? `, ${data.persona.nombre.split(' ')[0]}` : ''}</h1>

	{#if !data.arranque.completo}
		<section class="tarjeta pila">
			<h2>Para empezar a recibir turnos</h2>
			<ul>
				{#each data.arranque.pasos as p (p.clave)}
					<li>{p.listo ? '✓' : '○'} {p.titulo}{p.obligatorio ? '' : ' (opcional)'}</li>
				{/each}
			</ul>
		</section>
	{/if}

	{#if data.tablero}
		<div class="grilla">
			<div class="tarjeta"><div class="suave chico">Turnos hoy</div><strong>{data.tablero.turnosHoy}</strong></div>
			<div class="tarjeta"><div class="suave chico">Ingresos del mes</div><strong>{pesos(data.tablero.ingresosMes)}</strong></div>
			<div class="tarjeta"><div class="suave chico">Clientes activos</div><strong>{data.tablero.clientesActivos}</strong></div>
			<div class="tarjeta">
				<div class="suave chico">Ocupación hoy</div>
				<strong>{data.tablero.ocupacionHoy === null ? '—' : `${Math.round(data.tablero.ocupacionHoy * 100)}%`}</strong>
			</div>
		</div>
		<section class="tarjeta pila">
			<h2>Próximos turnos</h2>
			{#if data.tablero.proximos.length === 0}
				<p class="suave">No hay turnos próximos.</p>
			{:else}
				<div class="desplazable">
					<table class="tabla">
						<thead><tr><th>Cuándo</th><th>Cliente</th><th>Servicio</th><th>Profesional</th><th>Estado</th></tr></thead>
						<tbody>
							{#each data.tablero.proximos as t (t.id)}
								<tr>
									<td><a href={`/panel/turnos/${t.id}`}>{fechaHora(t.inicio)}</a></td>
									<td>{t.cliente}</td>
									<td>{t.servicio}</td>
									<td>{t.profesional}</td>
									<td><span class="etiqueta">{ESTADOS[t.estado] ?? t.estado}</span></td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</section>
	{/if}
</div>
