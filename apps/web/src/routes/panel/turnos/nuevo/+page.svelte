<script lang="ts">
	import { enhance } from '$app/forms';
	import { pesos } from '#lib/formato.ts';
	import { nombreDia } from '#lib/operacion/fechas.ts';

	let { data, form } = $props();
	let sobreturno = $state(false);
	const servicio = $derived(data.servicios.find((s) => s.id === data.servicio));
	const nombreDe = (id: string) => data.todos.find((p) => p.id === id)?.nombre ?? '';
</script>

<svelte:head><title>Nuevo turno · Panel</title></svelte:head>

<div class="pila">
	<p><a href="/panel/turnos" class="chico">← Turnos</a></p>
	<h1>Nuevo turno</h1>

	{#if data.servicios.length === 0}
		<p class="tarjeta">Primero cargá un servicio en <a href="/panel/servicios">Servicios</a>.</p>
	{:else}
		<form method="GET" class="tarjeta grilla" aria-label="Servicio, profesional y día" data-sveltekit-reset="false">
			{#if data.cliente.id}<input type="hidden" name="cliente" value={data.cliente.id} />{/if}
			<label class="campo">
				Servicio
				<select name="servicio" onchange={(e) => e.currentTarget.form?.requestSubmit()}>
					{#each data.servicios as s (s.id)}
						<option value={s.id} selected={s.id === data.servicio}>{s.nombre} · {s.duracion} min · {pesos(s.precio)}</option>
					{/each}
				</select>
			</label>
			<label class="campo">
				Profesional
				<select name="profesional" onchange={(e) => e.currentTarget.form?.requestSubmit()}>
					<option value="">Cualquiera</option>
					{#each data.profesionales as p (p.id)}
						<option value={p.id} selected={p.id === data.profesional}>{p.nombre}</option>
					{/each}
				</select>
			</label>
			<label class="campo">Día <input type="date" name="fecha" value={data.fecha} onchange={(e) => e.currentTarget.form?.requestSubmit()} /></label>
			<div class="fila" style="align-self: end"><button class="boton">Ver horarios</button></div>
		</form>

		{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}

		<form method="POST" action="?/crear" use:enhance class="pila">
			<input type="hidden" name="servicioId" value={data.servicio} />
			<input type="hidden" name="fecha" value={data.fecha} />

			<section class="tarjeta pila">
				<h2>Horario del {nombreDia(data.fecha)}</h2>
				<label class="fila chico"><input type="checkbox" name="sobreturno" bind:checked={sobreturno} /> Sobreturno (a una hora que no figura&nbsp;libre)</label>
				{#if sobreturno}
					<div class="grilla">
						<label class="campo">
							Profesional
							<select name="profesionalId" required>
								<option value="" disabled selected={!data.profesional}>Elegí</option>
								{#each data.profesionales as p (p.id)}
									<option value={p.id} selected={p.id === data.profesional}>{p.nombre}</option>
								{/each}
							</select>
						</label>
						<label class="campo">Hora <input type="time" name="hora" required step="300" /></label>
					</div>
				{:else}
					<input type="hidden" name="profesionalId" value={data.profesional} />
					{#if data.errorLibres}
						<p class="aviso error">{data.errorLibres}</p>
					{:else if data.horarios.length === 0}
						<p class="suave">No quedan horarios libres{servicio ? ` de ${servicio.nombre}` : ''} ese día. Probá otro día u otro profesional, o cargalo como sobreturno.</p>
					{:else}
						<fieldset class="horarios">
							<legend class="visualmente-oculto">Horarios libres</legend>
							{#each data.horarios as h, i (h.inicio)}
								<label class="horario">
									<input type="radio" name="inicio" value={h.inicio} required checked={i === 0} />
									<span>{h.hora}</span>
									{#if !data.profesional}<span class="suave chico">{h.profesionales.map(nombreDe).filter(Boolean).join(', ')}</span>{/if}
								</label>
							{/each}
						</fieldset>
					{/if}
				{/if}
			</section>

			<section class="tarjeta pila">
				<h2>Cliente</h2>
				<div class="grilla">
					<label class="campo">Nombre <input name="nombre" required maxlength="80" value={data.cliente.nombre} autocomplete="off" /></label>
					<label class="campo">Teléfono <input name="telefono" type="tel" maxlength="30" value={data.cliente.telefono} autocomplete="off" /></label>
					<label class="campo">Email <input name="email" type="email" maxlength="120" value={data.cliente.email} autocomplete="off" /></label>
				</div>
				<p class="suave chico">Con un teléfono o un email le llegan los avisos. Si el teléfono ya está cargado, el turno va a ese&nbsp;cliente.</p>
				<label class="campo">Nota interna <textarea name="nota" maxlength="500" rows="2"></textarea></label>
			</section>

			<div class="fila"><button class="boton primario">Reservar turno</button></div>
		</form>
	{/if}
</div>

<style>
	.horarios {
		border: 0;
		padding: 0;
		margin: 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(92px, 1fr));
		gap: 8px;
	}
	.horario {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 2px;
		padding: 8px;
		border: 1px solid var(--borde);
		border-radius: 8px;
		cursor: pointer;
		text-align: center;
	}
	.horario:has(input:checked) {
		border-color: var(--marca);
		background: var(--marca-suave);
	}
	.horario input {
		position: absolute;
		opacity: 0;
		pointer-events: none;
	}
	.horario:has(input:focus-visible) {
		outline: 2px solid var(--marca);
		outline-offset: 2px;
	}
</style>
