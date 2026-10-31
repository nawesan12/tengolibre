<script lang="ts">
	import { enhance } from '$app/forms';
	import { fechaHora } from '#lib/formato.ts';
	import Pestanias from '#lib/operacion/Pestanias.svelte';
	import SinPermiso from '#lib/operacion/SinPermiso.svelte';
	import { NOTIFICACIONES } from '#lib/operacion/secciones.ts';

	let { data, form } = $props();
	const ESTADO: Record<string, string> = { pendiente: 'En cola', enviado: 'Enviado', fallido: 'Falló', descartado: 'Descartado' };
	const ultimo = $derived(data.avisos.at(-1));
</script>

<svelte:head><title>Mails enviados · Panel</title></svelte:head>

<div class="pila">
	<h1>Mails enviados</h1>
	<Pestanias items={NOTIFICACIONES} etiqueta="Notificaciones" />

	{#if data.prohibido}
		<SinPermiso que="el registro de mails" />
	{:else}
		{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}
		{#if form && 'ok' in form && form.ok}<p class="aviso" role="status">Listo, vuelve a salir en un&nbsp;momento.</p>{/if}
		<form method="GET" class="fila" aria-label="Filtrar por estado">
			<label class="campo">
				Estado
				<select name="estado" onchange={(e) => e.currentTarget.form?.requestSubmit()}>
					<option value="">Todos</option>
					{#each Object.entries(ESTADO) as [k, v] (k)}<option value={k} selected={k === data.estado}>{v}</option>{/each}
				</select>
			</label>
			<noscript><button class="boton">Filtrar</button></noscript>
		</form>

		{#if data.avisos.length === 0}
			<p class="tarjeta suave">No hay mails con ese filtro.</p>
		{:else}
			<ul class="tarjeta lista">
				{#each data.avisos as a (a.id)}
					<li class="fila" style="justify-content: space-between">
						<span class="que">
							<strong>{a.tipo.replaceAll('_', ' ')}</strong>
							<span class="suave chico">· {a.para}{a.cliente ? ` (${a.cliente})` : ''} · {fechaHora(a.enviadoEn ?? a.creadoEn)}</span>
							<span class="etiqueta">{ESTADO[a.estado] ?? a.estado}</span>
							{#if a.ultimoError}<span class="chico error-aviso">{a.ultimoError}</span>{/if}
							{#if a.turnoId}<a class="chico" href={`/panel/turnos/${a.turnoId}`}>Ver el turno</a>{/if}
						</span>
						{#if a.estado !== 'pendiente'}
							<form method="POST" action="?/reenviar" use:enhance>
								<input type="hidden" name="id" value={a.id} />
								<button class="boton chico">Reenviar</button>
							</form>
						{/if}
					</li>
				{/each}
			</ul>
			{#if data.avisos.length === 50 && ultimo}
				<a class="boton chico" href={`?${new URLSearchParams({ ...(data.estado ? { estado: data.estado } : {}), antes: ultimo.creadoEn })}`}>Más viejos →</a>
			{/if}
		{/if}
	{/if}
</div>

<style>
	.lista {
		list-style: none;
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.lista li + li {
		border-top: 1px solid var(--borde);
		padding-top: 10px;
	}
	.que {
		display: flex;
		flex-wrap: wrap;
		gap: 4px 8px;
		align-items: baseline;
		min-width: 0;
		overflow-wrap: anywhere;
	}
	.error-aviso {
		color: var(--peligro);
	}
</style>
