<script lang="ts">
	import { fechaHora } from '#lib/formato.ts';

	let { data } = $props();
	const f = $derived(data.filtros);
	const ENTIDADES = ['turno', 'cliente', 'invitacion', 'ajustes', 'aviso', 'espera', 'resena'];
	const detalle = (d: unknown) => {
		if (!d || typeof d !== 'object') return '';
		return Object.entries(d as Record<string, unknown>)
			.map(([k, v]) => `${k}: ${typeof v === 'object' ? JSON.stringify(v) : String(v)}`)
			.join(' · ');
	};
</script>

<svelte:head><title>Registro de cambios · Panel</title></svelte:head>

<div class="pila">
	<p class="chico"><a href="/panel/configuracion">← Configuración</a></p>
	<h1>Registro de cambios</h1>
	<p class="suave">Quién hizo qué en el panel: turnos movidos, invitaciones, cambios de&nbsp;ajustes.</p>

	<form method="GET" class="tarjeta grilla" aria-label="Filtros">
		<label class="campo">Desde <input type="date" name="desde" value={f.desde} /></label>
		<label class="campo">Hasta <input type="date" name="hasta" value={f.hasta} /></label>
		<label class="campo">
			Qué
			<input name="entidad" value={f.entidad} list="entidades" placeholder="turno, cliente…" autocapitalize="none" />
			<datalist id="entidades">{#each ENTIDADES as e (e)}<option value={e}></option>{/each}</datalist>
		</label>
		{#if data.miembros.length > 0}
			<label class="campo">
				Quién
				<select name="quien">
					<option value="">Cualquiera</option>
					{#each data.miembros as m (m.clerkId)}<option value={m.clerkId} selected={m.clerkId === f.quien}>{m.nombre || m.email}</option>{/each}
				</select>
			</label>
		{/if}
		<div class="fila" style="align-self: end">
			<button class="boton primario">Filtrar</button>
			<a class="boton" href={`/panel/auditoria/exportar${data.consulta ? `?${data.consulta}` : ''}`} download>Descargar CSV</a>
		</div>
	</form>

	<section class="tarjeta">
		{#if data.entradas.length === 0}
			<p class="suave">No hay cambios con esos filtros.</p>
		{:else}
			<div class="desplazable">
				<table class="tabla">
					<thead><tr><th>Cuándo</th><th>Quién</th><th>Qué</th><th>Detalle</th></tr></thead>
					<tbody>
						{#each data.entradas as e (e.id)}
							<tr>
								<td class="sin-corte">{fechaHora(e.creadoEn)}</td>
								<td>{e.nombre || (e.clerkId ? 'Alguien del equipo' : 'TengoLibre')}</td>
								<td><code>{e.accion}</code></td>
								<td class="chico suave">{detalle(e.detalle)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
		{#if data.siguiente}
			<p><a class="boton" href={`?${data.consulta ? `${data.consulta}&` : ''}antes=${encodeURIComponent(data.siguiente)}`}>Ver más viejos</a></p>
		{/if}
	</section>
</div>

<style>
	.sin-corte {
		white-space: nowrap;
	}
</style>
