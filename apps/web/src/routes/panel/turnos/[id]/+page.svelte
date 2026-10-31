<script lang="ts">
	import { enhance } from '$app/forms';
	import { page } from '$app/state';
	import { fechaHora, hora, pesos } from '#lib/formato.ts';
	import { nombreDia } from '#lib/operacion/fechas.ts';
	import AccionesTurno from '#lib/operacion/AccionesTurno.svelte';
	import EstadoTurno from '#lib/operacion/EstadoTurno.svelte';

	let { data, form } = $props();
	const t = $derived(data.turno);
	let forzar = $state(false);
	let notaCambiada = $state(false);

	const ORIGENES: Record<string, string> = { web: 'Reserva online', panel: 'Cargado en el panel', app: 'App' };
	const ESTADO_AVISO: Record<string, string> = { pendiente: 'En cola', enviado: 'Enviado', fallido: 'Falló', descartado: 'Descartado' };
</script>

<svelte:head><title>Turno de {t.cliente.nombre} · Panel</title></svelte:head>

<div class="pila">
	<p><a href="/panel/turnos" class="chico">← Turnos</a></p>
	{#if page.url.searchParams.get('nuevo')}<p class="aviso" role="status">Listo, el turno quedó&nbsp;reservado.</p>{/if}
	{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}

	<section class="tarjeta pila">
		<div class="fila" style="justify-content: space-between">
			<h1>{t.cliente.nombre}</h1>
			<EstadoTurno estado={t.estado} />
		</div>
		<dl class="datos">
			<dt>Cuándo</dt>
			<dd>{fechaHora(t.inicio)} a {hora(t.fin)}</dd>
			<dt>Servicio</dt>
			<dd>{t.servicio} · {pesos(t.precio)}</dd>
			<dt>Profesional</dt>
			<dd>{t.profesional || 'Sin profesional'}</dd>
			<dt>Contacto</dt>
			<dd>
				{#if t.cliente.telefono}<a href={`tel:${t.cliente.telefono}`}>{t.cliente.telefono}</a>{/if}
				{#if t.cliente.telefono && t.cliente.email}·{/if}
				{#if t.cliente.email}<a href={`mailto:${t.cliente.email}`}>{t.cliente.email}</a>{/if}
				{#if !t.cliente.telefono && !t.cliente.email}<span class="suave">Sin datos</span>{/if}
			</dd>
			<dt>Origen</dt>
			<dd>{ORIGENES[t.origen] ?? t.origen}</dd>
		</dl>
		{#if t.cliente.id}<p><a href={`/panel/clientes/${t.cliente.id}`}>Ver la ficha del cliente</a></p>{/if}
		<AccionesTurno id={t.id} estado={t.estado} />
	</section>

	<section class="tarjeta pila">
		<h2>Nota interna</h2>
		<form
			method="POST"
			action="?/nota"
			class="pila"
			use:enhance={() =>
				async ({ update }) => {
					await update({ reset: false });
					notaCambiada = false;
				}}
		>
			<label class="campo"><span class="visualmente-oculto">Nota</span><textarea name="nota" maxlength="500" rows="3" oninput={() => (notaCambiada = true)}>{t.nota}</textarea></label>
			<div class="fila">
				<button class="boton">Guardar nota</button>
				{#if form && 'guardado' in form && form.guardado === 'nota' && !notaCambiada}<span class="suave chico" role="status">Guardada</span>{/if}
			</div>
		</form>
	</section>

	{#if t.estado !== 'cancelado' && t.estado !== 'atendido' && t.estado !== 'ausente'}
		<section class="tarjeta pila">
			<h2>Reprogramar</h2>
			<form method="GET" class="grilla" data-sveltekit-reset="false" aria-label="Día y profesional para reprogramar">
				<label class="campo">Día <input type="date" name="fecha" value={data.fecha} onchange={(e) => e.currentTarget.form?.requestSubmit()} /></label>
				<label class="campo">
					Profesional
					<select name="profesional" onchange={(e) => e.currentTarget.form?.requestSubmit()}>
						{#each data.profesionales as p (p.id)}
							<option value={p.id} selected={p.id === data.profesional}>{p.nombre}</option>
						{/each}
					</select>
				</label>
				<div class="fila" style="align-self: end"><button class="boton">Ver horarios</button></div>
			</form>
			<form method="POST" action="?/reprogramar" use:enhance class="pila">
				<input type="hidden" name="profesionalId" value={data.profesional} />
				<input type="hidden" name="fecha" value={data.fecha} />
				<label class="fila chico"><input type="checkbox" name="forzar" bind:checked={forzar} /> Sobreturno (a una hora que no figura&nbsp;libre)</label>
				{#if forzar}
					<label class="campo">Hora <input type="time" name="hora" required step="300" /></label>
				{:else if data.errorLibres}
					<p class="aviso error">{data.errorLibres}</p>
				{:else if data.horarios.length === 0}
					<p class="suave">No hay horarios libres el {nombreDia(data.fecha)}.</p>
				{:else}
					<label class="campo">
						Horario del {nombreDia(data.fecha)}
						<select name="inicio" required>
							{#each data.horarios as h (h.inicio)}<option value={h.inicio}>{h.hora}</option>{/each}
						</select>
					</label>
				{/if}
				<div class="fila"><button class="boton primario">Reprogramar</button></div>
			</form>
		</section>
	{/if}

	{#if data.avisos && data.avisos.length > 0}
		<section class="tarjeta pila">
			<h2>Avisos de este turno</h2>
			<ul class="avisos">
				{#each data.avisos as a (a.id)}
					<li class="fila" style="justify-content: space-between">
						<span>
							<strong>{a.tipo.replaceAll('_', ' ')}</strong>
							<span class="suave chico">· {a.para} · {ESTADO_AVISO[a.estado] ?? a.estado} · {fechaHora(a.enviadoEn ?? a.creadoEn)}</span>
							{#if a.ultimoError}<span class="chico" style="color: var(--peligro)"> · {a.ultimoError}</span>{/if}
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
		</section>
	{/if}
</div>

<style>
	.datos {
		display: grid;
		grid-template-columns: max-content 1fr;
		gap: 6px 16px;
		margin: 0;
	}
	dt {
		color: var(--suave);
	}
	dd {
		margin: 0;
		overflow-wrap: anywhere;
	}
	.avisos {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	@media (max-width: 420px) {
		.datos {
			grid-template-columns: 1fr;
		}
		dd {
			margin-bottom: 6px;
		}
	}
</style>
