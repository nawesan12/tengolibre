<script lang="ts">
	import { enhance } from '$app/forms';
	import { ROLES, fecha } from '#lib/formato.ts';
	import { ROLES_INVITABLES } from '#lib/configuracion/textos.ts';
	import { confirmar } from '#lib/configuracion/formularios.ts';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';

	let { data, form } = $props();
	const esDuenio = $derived(data.negocio.rol === 'duenio');
	const pendientes = $derived((data.invitaciones ?? []).filter((i) => !i.aceptadaEn));
	const nombreServicio = (id: string) => data.servicios.find((s) => s.id === id)?.nombre;
</script>

<svelte:head><title>Equipo y horarios · Panel</title></svelte:head>

<div class="pila">
	<h1>Equipo y horarios</h1>

	<section class="tarjeta pila" aria-labelledby="t-profesionales">
		<h2 id="t-profesionales">Profesionales</h2>
		<p class="suave chico">Quienes atienden. Cada uno tiene su horario, sus servicios y sus días&nbsp;libres.</p>
		<Mensaje {form} que="profesionales" />
		{#if data.profesionales.length === 0}
			<p class="suave">Todavía no cargaste profesionales: sin al menos uno no se pueden reservar&nbsp;turnos.</p>
		{/if}
		<ul class="lista">
			{#each data.profesionales as p (p.id)}
				<li class="fila entre">
					<div>
						<a href={`/panel/profesionales/${p.id}`}><strong>{p.nombre}</strong></a>
						{#if !p.activo}<span class="etiqueta">Pausado</span>{/if}
						<div class="suave chico">
							{p.especialidad || 'Sin especialidad'} ·
							{p.horario.length === 0 ? 'Sin horario' : `${new Set(p.horario.map((h) => h.dia)).size} días por semana`} ·
							{p.servicios.length === 0 ? 'Sin servicios' : p.servicios.map(nombreServicio).filter(Boolean).join(', ')}
						</div>
					</div>
					<a class="boton" href={`/panel/profesionales/${p.id}`}>Horario y datos</a>
				</li>
			{/each}
		</ul>

		{#if data.miembros}
			<details>
				<summary class="boton">Sumar un profesional</summary>
				<form method="POST" action="?/profesional" use:enhance class="grilla" style="margin-top: 12px">
					<label class="campo">Nombre <input name="nombre" required minlength="2" maxlength="60" autocomplete="off" /></label>
					<label class="campo">Especialidad <input name="especialidad" maxlength="80" placeholder="Barbero, colorista…" /></label>
					<label class="campo">Email <input name="email" type="email" maxlength="120" /></label>
					<label class="campo">Teléfono <input name="telefono" type="tel" maxlength="30" /></label>
					{#if data.servicios.length > 0}
						<fieldset class="campo">
							<legend>Servicios que hace</legend>
							{#each data.servicios as s (s.id)}
								<label class="fila chico"><input type="checkbox" name="servicios" value={s.id} checked /> {s.nombre}</label>
							{/each}
						</fieldset>
					{:else}
						<p class="suave chico">Cuando cargues <a href="/panel/servicios">servicios</a> vas a poder elegir cuáles&nbsp;hace.</p>
					{/if}
					<div><button class="boton primario">Sumar profesional</button></div>
				</form>
			</details>
		{/if}
	</section>

	{#if data.miembros}
		<section class="tarjeta pila" aria-labelledby="t-miembros">
			<h2 id="t-miembros">Quiénes entran al panel</h2>
			<Mensaje {form} que="miembros" />
			<div class="desplazable">
				<table class="tabla">
					<thead><tr><th>Persona</th><th>Rol</th><th><span class="visualmente-oculto">Acciones</span></th></tr></thead>
					<tbody>
						{#each data.miembros as m (m.id)}
							<tr>
								<td>
									<strong>{m.nombre || m.email}</strong>
									{#if !m.activo}<span class="etiqueta">Sin acceso</span>{/if}
									<div class="suave chico">{m.email}</div>
								</td>
								<td>
									{#if esDuenio}
										<form method="POST" action="?/rol" use:enhance class="fila">
											<input type="hidden" name="id" value={m.id} />
											<label class="campo">
												<span class="visualmente-oculto">Rol de {m.nombre || m.email}</span>
												<select name="rol" onchange={(e) => e.currentTarget.form?.requestSubmit()}>
													{#each Object.entries(ROLES) as [clave, nombre] (clave)}
														<option value={clave} selected={clave === m.rol}>{nombre}</option>
													{/each}
												</select>
											</label>
											<noscript><button class="boton">Cambiar</button></noscript>
										</form>
									{:else}
										{ROLES[m.rol] ?? m.rol}
									{/if}
								</td>
								<td>
									{#if esDuenio}
										<form
											method="POST"
											action="?/activo"
											use:enhance={m.activo ? confirmar(`¿Sacarle el acceso al panel a ${m.nombre || m.email}?`) : undefined}
										>
											<input type="hidden" name="id" value={m.id} />
											<input type="hidden" name="activo" value={String(!m.activo)} />
											<button class="boton" class:peligro={m.activo}>{m.activo ? 'Sacar acceso' : 'Devolver acceso'}</button>
										</form>
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</section>

		<section class="tarjeta pila" aria-labelledby="t-invitaciones">
			<h2 id="t-invitaciones">Invitaciones</h2>
			<p class="suave chico">Le llega un mail con un link que sirve una semana. Al aceptarlo entra con el rol que&nbsp;elijas.</p>
			<Mensaje {form} que="invitaciones" />
			{#if esDuenio}
				<form method="POST" action="?/invitar" use:enhance class="grilla">
					<label class="campo">Email <input name="email" type="email" required maxlength="120" autocomplete="off" /></label>
					<label class="campo">
						Rol
						<select name="rol">
							{#each ROLES_INVITABLES as r (r)}<option value={r} selected={r === 'profesional'}>{ROLES[r]}</option>{/each}
						</select>
					</label>
					<div class="fila" style="align-self: end"><button class="boton primario">Invitar</button></div>
				</form>
			{/if}
			{#if pendientes.length === 0}
				<p class="suave">No hay invitaciones pendientes.</p>
			{:else}
				<ul class="lista">
					{#each pendientes as i (i.id)}
						<li class="fila entre">
							<div>
								<strong>{i.email}</strong>
								<div class="suave chico">{ROLES[i.rol] ?? i.rol} · vence el {fecha(i.venceEn)}</div>
							</div>
							{#if esDuenio}
								<form method="POST" action="?/revocar" use:enhance={confirmar(`¿Borrar la invitación de ${i.email}?`)}>
									<input type="hidden" name="id" value={i.id} />
									<button class="boton peligro">Borrar</button>
								</form>
							{/if}
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	{/if}
</div>

<style>
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.lista li {
		padding: 10px 0;
		border-bottom: 1px solid var(--borde);
	}
	.entre {
		justify-content: space-between;
	}
	fieldset {
		border: 1px solid var(--borde);
		border-radius: 8px;
		padding: 8px 10px;
		margin: 0;
	}
	summary {
		list-style: none;
		width: fit-content;
	}
	summary::-webkit-details-marker {
		display: none;
	}
</style>
