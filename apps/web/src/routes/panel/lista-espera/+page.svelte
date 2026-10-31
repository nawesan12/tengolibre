<script lang="ts">
	import { enhance } from '$app/forms';
	import { fechaHora } from '#lib/formato.ts';
	import { nombreDia } from '#lib/operacion/fechas.ts';
	import SinPermiso from '#lib/operacion/SinPermiso.svelte';

	let { data, form } = $props();
</script>

<svelte:head><title>Lista de espera · Panel</title></svelte:head>

<div class="pila">
	<h1>Lista de espera</h1>

	{#if data.prohibido}
		<SinPermiso que="la lista de espera" />
	{:else}
		<p class="suave">Quienes se anotaron porque no encontraron lugar. Cuando se libera un horario se les avisa solos; desde acá podés avisarle a alguien con un horario&nbsp;puntual.</p>
		{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}
		<p><a class="chico" href={data.todos ? '?' : '?todos=1'}>{data.todos ? 'Ver sólo los que esperan' : 'Ver también avisados y vencidos'}</a></p>

		{#if data.espera.length === 0}
			<p class="tarjeta suave">No hay nadie esperando.</p>
		{/if}
		<ul class="lista">
			{#each data.espera as a (a.id)}
				<li class="tarjeta pila" data-anotado={a.id}>
					<div class="fila" style="justify-content: space-between">
						<div>
							<strong>{a.nombre}</strong>
							<div class="suave chico">
								{a.servicio}{a.profesional ? ` con ${a.profesional}` : ''} · del {nombreDia(a.desde)} al {nombreDia(a.hasta)}
							</div>
							<div class="chico">
								<a href={`mailto:${a.email}`}>{a.email}</a>{#if a.telefono} · <a href={`tel:${a.telefono}`}>{a.telefono}</a>{/if}
							</div>
						</div>
						{#if a.avisadoEn}<span class="etiqueta">Avisado {fechaHora(a.avisadoEn)}</span>{/if}
					</div>
					<div class="fila acciones">
						{#if !a.avisadoEn}
							<form method="POST" action="?/avisar" use:enhance class="fila avisar">
								<input type="hidden" name="id" value={a.id} />
								<label class="campo">
									Horario sugerido
									<select name="inicio">
										{#each data.sugeridos[a.id] ?? [] as h (h.inicio)}<option value={h.inicio}>{fechaHora(h.inicio)}</option>{/each}
										<option value="">Sin horario puntual</option>
									</select>
								</label>
								<label class="campo">Nota <input name="nota" maxlength="300" placeholder="Opcional" /></label>
								<button class="boton primario">Avisar</button>
							</form>
						{/if}
						<form
							method="POST"
							action="?/borrar"
							use:enhance={({ cancel }) => {
								if (!confirm(`¿Sacar a ${a.nombre} de la lista?`)) cancel();
							}}
						>
							<input type="hidden" name="id" value={a.id} />
							<button class="boton peligro" aria-label={`Borrar a ${a.nombre}`}>Borrar</button>
						</form>
					</div>
				</li>
			{/each}
		</ul>
	{/if}
</div>

<style>
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.acciones {
		justify-content: space-between;
		align-items: end;
	}
	.avisar {
		align-items: end;
	}
</style>
