<script lang="ts">
	import { enhance } from '$app/forms';
	import { page } from '$app/state';
	import { DIAS, ROLES, dia as hoy, fecha } from '#lib/formato.ts';
	import { FRANJAS_POR_DIA, SEMANA, franjasDe } from '#lib/configuracion/horario.ts';
	import { confirmar, sinReset } from '#lib/configuracion/formularios.ts';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';

	let { data, form } = $props();
	const p = $derived(data.profesional);
	const esGestion = $derived(['duenio', 'encargado'].includes(data.negocio.rol));
	let entero = $state(true);
	// Fecha corta y legible para un AAAA-MM-DD (a mediodía, para que la zona no cambie el día).
	const diaLegible = (d: string) => fecha(`${d}T12:00:00-03:00`);
</script>

<svelte:head><title>{p.nombre} · Panel</title></svelte:head>

<div class="pila">
	<p class="chico"><a href="/panel/equipo">← Equipo y horarios</a></p>
	<h1>{p.nombre}</h1>
	{#if page.url.searchParams.has('nuevo')}
		<p class="aviso" role="status">Listo, ya está cargado. Ahora poné los días y horarios en que&nbsp;atiende.</p>
	{/if}

	<section class="tarjeta pila" aria-labelledby="t-horario">
		<h2 id="t-horario">Horario semanal</h2>
		<p class="suave chico">Si corta al mediodía, cargá la mañana y la tarde como dos&nbsp;franjas.</p>
		<Mensaje {form} que="horario" />
		<form method="POST" action="?/horario" use:enhance={sinReset} class="pila">
			{#each SEMANA as d (d)}
				{@const franjas = franjasDe(p.horario, d)}
				<fieldset class="dia">
					<legend><label class="fila"><input type="checkbox" name={`abre-${d}`} checked={franjas.length > 0} /> {DIAS[d]}</label></legend>
					<div class="franjas">
						{#each Array.from({ length: FRANJAS_POR_DIA }, (_, i) => i) as i (i)}
							<div class="fila">
								<label class="campo">
									<span>{i === 0 ? 'Desde' : 'Vuelve a las'}</span>
									<input type="time" name={`desde-${d}-${i}`} value={franjas[i]?.desde ?? (i === 0 ? '09:00' : '')} step="300" />
								</label>
								<label class="campo">
									<span>{i === 0 ? 'Hasta' : 'Hasta (tarde)'}</span>
									<input type="time" name={`hasta-${d}-${i}`} value={franjas[i]?.hasta ?? (i === 0 ? '18:00' : '')} step="300" />
								</label>
							</div>
						{/each}
					</div>
				</fieldset>
			{/each}
			<div><button class="boton primario">Guardar horario</button></div>
		</form>
	</section>

	<section class="tarjeta pila" aria-labelledby="t-bloqueos">
		<h2 id="t-bloqueos">Días bloqueados</h2>
		<p class="suave chico">Vacaciones, un trámite, un día de estudio: esas fechas no se pueden&nbsp;reservar.</p>
		<Mensaje {form} que="bloqueos" />
		<form method="POST" action="?/bloqueo" use:enhance class="grilla">
			<label class="campo">Fecha <input type="date" name="fecha" required min={hoy()} /></label>
			<label class="fila chico" style="align-self: end"><input type="checkbox" name="entero" bind:checked={entero} /> Todo el día</label>
			{#if !entero}
				<label class="campo">Desde <input type="time" name="desde" required step="300" /></label>
				<label class="campo">Hasta <input type="time" name="hasta" required step="300" /></label>
			{/if}
			<label class="campo">Motivo (opcional) <input name="motivo" maxlength="120" /></label>
			<div class="fila" style="align-self: end"><button class="boton primario">Bloquear</button></div>
		</form>
		{#if data.bloqueos.length === 0}
			<p class="suave">No hay fechas bloqueadas de acá en adelante.</p>
		{:else}
			<ul class="lista">
				{#each data.bloqueos as b (b.id)}
					<li class="fila entre">
						<div>
							<strong>{diaLegible(b.fecha)}</strong>
							<span class="suave chico">{b.desde ? `de ${b.desde.slice(0, 5)} a ${b.hasta?.slice(0, 5)}` : 'todo el día'}</span>
							{#if b.motivo}<div class="suave chico">{b.motivo}</div>{/if}
						</div>
						<form method="POST" action="?/borrarBloqueo" use:enhance>
							<input type="hidden" name="bloqueo" value={b.id} />
							<button class="boton" aria-label={`Sacar el bloqueo del ${diaLegible(b.fecha)}`}>Sacar</button>
						</form>
					</li>
				{/each}
			</ul>
		{/if}
	</section>

	<section class="tarjeta pila" aria-labelledby="t-semanales">
		<h2 id="t-semanales">Horarios bloqueados todas las semanas</h2>
		<p class="suave chico">Un turno puntual que nunca se da, como el almuerzo de los&nbsp;martes.</p>
		<Mensaje {form} que="semanales" />
		<form method="POST" action="?/semanal" use:enhance class="grilla">
			<label class="campo">
				Día
				<select name="dia">{#each SEMANA as d (d)}<option value={d}>{DIAS[d]}</option>{/each}</select>
			</label>
			<label class="campo">Hora <input type="time" name="hora" required step="300" /></label>
			<div class="fila" style="align-self: end"><button class="boton primario">Bloquear ese horario</button></div>
		</form>
		{#if data.semanales.length > 0}
			<ul class="fila lista-chips">
				{#each data.semanales as s (`${s.dia}-${s.hora}`)}
					<li>
						<form method="POST" action="?/borrarSemanal" use:enhance class="fila">
							<input type="hidden" name="dia" value={s.dia} />
							<input type="hidden" name="hora" value={s.hora} />
							<span class="etiqueta">{DIAS[s.dia]} {s.hora}</span>
							<button class="boton chico" aria-label={`Sacar ${DIAS[s.dia]} ${s.hora}`}>×</button>
						</form>
					</li>
				{/each}
			</ul>
		{/if}
	</section>

	<section class="tarjeta pila" aria-labelledby="t-excepciones">
		<h2 id="t-excepciones">Excepciones de un día</h2>
		<p class="suave chico">Bloquear un horario puntual de una fecha, o liberar ese día uno de los bloqueados&nbsp;semanales.</p>
		<Mensaje {form} que="excepciones" />
		<form method="POST" action="?/excepcion" use:enhance class="grilla">
			<label class="campo">Fecha <input type="date" name="fecha" required min={hoy()} /></label>
			<label class="campo">Hora <input type="time" name="hora" required step="300" /></label>
			<label class="campo">
				Qué pasa
				<select name="tipo">
					<option value="bloqueado">No se puede reservar</option>
					<option value="libre">Se puede reservar</option>
				</select>
			</label>
			<div class="fila" style="align-self: end"><button class="boton primario">Guardar excepción</button></div>
		</form>
		{#if data.excepciones.length > 0}
			<ul class="lista">
				{#each data.excepciones as e (`${e.fecha}-${e.hora}`)}
					<li class="fila entre">
						<span><strong>{diaLegible(e.fecha)} {e.hora}</strong> <span class="suave chico">{e.tipo === 'libre' ? 'se puede reservar' : 'bloqueado'}</span></span>
						<form method="POST" action="?/borrarExcepcion" use:enhance>
							<input type="hidden" name="fecha" value={e.fecha} />
							<input type="hidden" name="hora" value={e.hora} />
							<button class="boton">Sacar</button>
						</form>
					</li>
				{/each}
			</ul>
		{/if}
	</section>

	{#if esGestion}
		<section class="tarjeta pila" aria-labelledby="t-servicios">
			<h2 id="t-servicios">Servicios que hace</h2>
			<Mensaje {form} que="servicios" />
			{#if data.servicios.length === 0}
				<p class="suave">Todavía no hay <a href="/panel/servicios">servicios</a> cargados.</p>
			{:else}
				<form method="POST" action="?/servicios" use:enhance={sinReset} class="pila">
					<div class="grilla">
						{#each data.servicios as s (s.id)}
							<label class="fila chico">
								<input type="checkbox" name="servicios" value={s.id} checked={p.servicios.includes(s.id)} />
								{s.nombre}{s.activo ? '' : ' (pausado)'}
							</label>
						{/each}
					</div>
					<div><button class="boton primario">Guardar servicios</button></div>
				</form>
			{/if}
		</section>

		<section class="tarjeta pila" aria-labelledby="t-datos">
			<h2 id="t-datos">Datos</h2>
			<Mensaje {form} que="datos" />
			<form method="POST" action="?/datos" use:enhance={sinReset} class="grilla">
				<label class="campo">Nombre <input name="nombre" value={p.nombre} required minlength="2" maxlength="60" /></label>
				<label class="campo">Especialidad <input name="especialidad" value={p.especialidad} maxlength="80" /></label>
				<label class="campo">Email <input name="email" type="email" value={p.email} maxlength="120" /></label>
				<label class="campo">Teléfono <input name="telefono" type="tel" value={p.telefono} maxlength="30" /></label>
				<label class="campo">Orden en la página <input name="orden" type="number" min="0" max="999" value={p.orden} /></label>
				{#if data.miembros}
					<label class="campo">
						Su cuenta del panel
						<select name="miembroId">
							<option value="">Sin cuenta</option>
							{#each data.miembros as m (m.id)}
								<option value={m.id} selected={m.id === p.miembroId}>{m.nombre || m.email} ({ROLES[m.rol] ?? m.rol})</option>
							{/each}
						</select>
					</label>
				{/if}
				<label class="campo" style="grid-column: 1 / -1">Sobre su trabajo <textarea name="bio" rows="3" maxlength="1000">{p.bio}</textarea></label>
				<label class="fila chico"><input type="checkbox" name="activo" checked={p.activo} /> Se le pueden reservar turnos</label>
				<div class="fila"><button class="boton primario">Guardar datos</button></div>
			</form>
			<form method="POST" action="?/borrar" use:enhance={confirmar(`¿Borrar a ${p.nombre}? Si tiene turnos, mejor pausalo.`)}>
				<button class="boton peligro">Borrar profesional</button>
			</form>
		</section>
	{/if}
</div>

<style>
	.dia {
		border: 1px solid var(--borde);
		border-radius: 8px;
		padding: 8px 12px 12px;
		margin: 0;
	}
	.dia legend {
		font-weight: 600;
		padding: 0 4px;
	}
	.franjas {
		display: flex;
		flex-wrap: wrap;
		gap: 8px 24px;
	}
	.franjas .campo {
		min-width: 120px;
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
	.lista-chips {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.entre {
		justify-content: space-between;
	}
</style>
