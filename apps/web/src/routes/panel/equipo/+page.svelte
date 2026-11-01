<script lang="ts">
	import '#lib/gestion/gestion.css';
	import { enhance } from '$app/forms';
	import { page } from '$app/state';
	import { ROLES, fecha } from '#lib/formato.ts';
	import { ROLES_INVITABLES } from '#lib/configuracion/textos.ts';
	import { confirmar } from '#lib/configuracion/formularios.ts';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';
	import ListaEquipo from '#lib/gestion/equipo/ListaEquipo.svelte';
	import GrillaSemana from '#lib/gestion/equipo/GrillaSemana.svelte';
	import { horasSemana, horasTexto } from '#lib/gestion/equipo/semana.ts';

	let { data, form } = $props();
	const esDuenio = $derived(data.negocio.rol === 'duenio');
	const pendientes = $derived((data.invitaciones ?? []).filter((i) => !i.aceptadaEn));
	const primero = $derived(data.profesionales[0]);
	const nombreServicio = (id: string) => data.servicios.find((s) => s.id === id)?.nombre;
	let invitando = $state(false);
	const mio = (m: { clerkId: string }) => m.clerkId === data.persona.clerkId;
	const hace = (d: string) => {
		const dias = Math.floor((Date.now() - new Date(d).getTime()) / 86_400_000);
		return dias <= 0 ? 'hoy' : dias === 1 ? 'ayer' : `hace ${dias} días`;
	};
</script>

<svelte:head><title>Equipo y horarios · Panel</title></svelte:head>

<div class="g-pagina equipo">
	<div class="izquierda g-col">
		<h1 class="g-titulo">Equipo</h1>
		<Mensaje {form} que="profesionales" />
		{#if data.profesionales.length === 0}
			<div class="g-bloque">
				<p class="g-nota">Todavía no cargaste profesionales: sin al menos uno no se pueden reservar&nbsp;turnos.</p>
			</div>
		{/if}
		<ListaEquipo profesionales={data.profesionales} />

		{#if data.miembros}
			<details class="sumar" id="sumar" open={page.url.searchParams.has('sumar') || data.profesionales.length === 0}>
				<summary><span class="icono" aria-hidden="true">person_add</span>Sumar un profesional</summary>
				<form method="POST" action="?/profesional" use:enhance class="g-bloque">
					<label class="g-campo">Nombre <input name="nombre" required minlength="2" maxlength="60" autocomplete="off" /></label>
					<label class="g-campo">Especialidad <input name="especialidad" maxlength="80" placeholder="Barbero, colorista…" /></label>
					<label class="g-campo">Email <input name="email" type="email" maxlength="120" /></label>
					<label class="g-campo">Teléfono <input name="telefono" type="tel" maxlength="30" /></label>
					{#if data.servicios.length > 0}
						<fieldset class="g-campo servicios">
							<legend>Qué hace</legend>
							<div class="g-chips">
								{#each data.servicios as s (s.id)}
									<label class="g-chip"><input type="checkbox" name="servicios" value={s.id} checked />{s.nombre}</label>
								{/each}
							</div>
						</fieldset>
					{:else}
						<p class="g-nota">Cuando cargues <a href="/panel/servicios">servicios</a> vas a poder elegir cuáles&nbsp;hace.</p>
					{/if}
					<button class="g-cta">Sumar profesional</button>
				</form>
			</details>
		{/if}
	</div>

	<div class="derecha g-col">
		{#if primero}
			<section class="g-bloque" aria-labelledby="t-horario">
				<div class="g-bloque-cabeza">
					<h2 id="t-horario" class="g-bloque-titulo grande">Horario de {primero.nombre}</h2>
					<span class="g-meta">{horasTexto(horasSemana(primero.horario))} h por semana</span>
				</div>
				<GrillaSemana horario={primero.horario} />
				<p class="g-nota">
					{primero.servicios.length > 0
						? `Hace ${primero.servicios.map(nombreServicio).filter(Boolean).join(', ')}.`
						: 'Todavía no tiene servicios.'}
					<a class="g-accion" href={`/panel/profesionales/${primero.id}`}>Horario, bloqueos y&nbsp;datos&nbsp;→</a>
				</p>
			</section>
		{/if}

		{#if data.miembros}
			<section class="g-bloque" aria-labelledby="t-accesos">
				<div class="g-bloque-cabeza">
					<h2 id="t-accesos" class="g-bloque-titulo">Quiénes entran al panel</h2>
					{#if esDuenio}
						<button type="button" class="g-boton oscuro chico" aria-expanded={invitando} onclick={() => (invitando = !invitando)}>
							{invitando ? 'Cerrar' : 'Invitar por mail'}
						</button>
					{/if}
				</div>
				<Mensaje {form} que="miembros" />
				<Mensaje {form} que="invitaciones" />
				{#if esDuenio && invitando}
					<form method="POST" action="?/invitar" use:enhance class="invitar">
						<label class="g-campo">Email <input name="email" type="email" required maxlength="120" autocomplete="off" /></label>
						<label class="g-campo">
							Rol
							<select name="rol">
								{#each ROLES_INVITABLES as r (r)}<option value={r} selected={r === 'profesional'}>{ROLES[r]}</option>{/each}
							</select>
						</label>
						<button class="g-boton menta">Invitar</button>
						<p class="g-nota g-ancho">Le llega un mail con un link que sirve una semana. Al aceptarlo entra con el rol que&nbsp;elijas.</p>
					</form>
				{/if}
				<ul class="accesos">
					{#each data.miembros as m (m.id)}
						<li class="acceso">
							<span class="quien">
								<b>{m.nombre || m.email}</b>
								<span>{m.email}{#if !m.activo} · <span class="sin">sin acceso</span>{/if}</span>
							</span>
							{#if esDuenio && !mio(m)}
								<form method="POST" action="?/rol" use:enhance class="rol">
									<input type="hidden" name="id" value={m.id} />
									<label>
										<span class="visualmente-oculto">Rol de {m.nombre || m.email}</span>
										<select class="g-valor" name="rol" onchange={(e) => e.currentTarget.form?.requestSubmit()}>
											{#each Object.entries(ROLES) as [clave, nombre] (clave)}
												<option value={clave} selected={clave === m.rol}>{nombre}</option>
											{/each}
										</select>
									</label>
									<noscript><button class="g-boton chico">Cambiar</button></noscript>
								</form>
							{:else}
								<span class="g-valor fijo">{ROLES[m.rol] ?? m.rol}</span>
							{/if}
							<span class="accion">
								{#if mio(m)}
									<span class="g-sub">Vos</span>
								{:else if esDuenio}
									<form
										method="POST"
										action="?/activo"
										use:enhance={m.activo ? confirmar(`¿Sacarle el acceso al panel a ${m.nombre || m.email}?`) : undefined}
									>
										<input type="hidden" name="id" value={m.id} />
										<input type="hidden" name="activo" value={String(!m.activo)} />
										<button class="g-accion" class:peligro={m.activo}>{m.activo ? 'Quitar acceso' : 'Devolver acceso'}</button>
									</form>
								{/if}
							</span>
						</li>
					{/each}
					{#each pendientes as i (i.id)}
						<li class="acceso">
							<span class="quien">
								<b>{i.email}</b>
								<span><span class="icono" aria-hidden="true">schedule_send</span>Invitación enviada {hace(i.creadoEn)} · vence el {fecha(i.venceEn)}</span>
							</span>
							<span class="g-valor fijo">{ROLES[i.rol] ?? i.rol}</span>
							<span class="accion">
								{#if esDuenio}
									<form method="POST" action="?/reenviar" use:enhance>
										<input type="hidden" name="id" value={i.id} />
										<input type="hidden" name="email" value={i.email} />
										<input type="hidden" name="rol" value={i.rol} />
										<button class="g-accion">Reenviar</button>
									</form>
									<form method="POST" action="?/revocar" use:enhance={confirmar(`¿Borrar la invitación de ${i.email}?`)}>
										<input type="hidden" name="id" value={i.id} />
										<button class="g-accion peligro" aria-label={`Borrar la invitación de ${i.email}`}>Borrar</button>
									</form>
								{/if}
							</span>
						</li>
					{/each}
				</ul>
			</section>
		{/if}
	</div>
</div>

<style>
	.equipo {
		display: grid;
		grid-template-columns: 300px minmax(0, 1fr);
		gap: 24px;
		align-items: start;
	}
	.g-titulo {
		font-size: 40px;
		margin-bottom: 4px;
	}
	.grande {
		font-size: 17px;
	}
	.sumar > summary {
		list-style: none;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 6px;
		min-height: 48px;
		border-radius: 16px;
		box-shadow: inset 0 0 0 1.5px #d8ccb4;
		font-size: 14px;
		font-weight: 700;
		cursor: pointer;
	}
	.sumar > summary::-webkit-details-marker {
		display: none;
	}
	.sumar > summary .icono {
		font-size: 20px;
	}
	.sumar[open] > summary {
		margin-bottom: 12px;
	}
	.servicios {
		border: 0;
		padding: 0;
		margin: 0;
	}
	.servicios legend {
		padding: 0;
		margin-bottom: 6px;
	}
	.invitar {
		display: grid;
		grid-template-columns: minmax(0, 1fr) 180px auto;
		gap: 12px;
		align-items: end;
		padding: 14px;
		border-radius: 14px;
		background: var(--tl-crema);
	}
	.accesos {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.acceso {
		display: grid;
		grid-template-columns: minmax(0, 1fr) 160px 150px;
		align-items: center;
		gap: 12px;
		min-height: 52px;
		padding: 6px 0;
		border-top: 1px solid #f0e8d8;
		font-size: 14px;
	}
	.quien {
		display: flex;
		flex-direction: column;
		min-width: 0;
	}
	.quien b {
		font-weight: 700;
		overflow-wrap: anywhere;
	}
	.quien > span {
		font-size: 12px;
		color: var(--tl-sub);
		font-variant-numeric: tabular-nums;
		overflow-wrap: anywhere;
	}
	.quien .icono {
		font-size: 14px;
		vertical-align: -2px;
		margin-right: 4px;
		color: var(--tl-menta-texto);
	}
	.sin {
		font-weight: 700;
		color: #a3241a;
	}
	.rol select,
	.g-valor.fijo {
		width: 100%;
		height: 36px;
		border-radius: 10px;
		font-size: 13px;
		text-transform: none;
	}
	.g-valor.fijo {
		padding: 0 12px;
	}
	.accion {
		display: flex;
		justify-content: flex-end;
		gap: 12px;
		font-size: 13px;
	}
	.accion .g-accion {
		font-size: 13px;
	}
	@media (max-width: 1100px) {
		.equipo {
			grid-template-columns: minmax(0, 1fr);
		}
	}
	@media (max-width: 640px) {
		.g-titulo {
			font-size: 36px;
		}
		.invitar {
			grid-template-columns: minmax(0, 1fr);
		}
		.acceso {
			grid-template-columns: minmax(0, 1fr) auto;
			padding: 10px 0;
		}
		.quien {
			grid-column: 1 / -1;
		}
		.rol select,
		.g-valor.fijo {
			height: 48px;
			min-width: 150px;
		}
	}
</style>
