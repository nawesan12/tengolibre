<script lang="ts">
	import '#lib/gestion/gestion.css';
	import { enhance } from '$app/forms';
	import { ESTADOS, dia, hora, pesos } from '#lib/formato.ts';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';
	import { NOMBRE_MOTIVO } from '#lib/gestion/dias-libres.ts';
	import { fechaTalon } from '#lib/gestion/equipo/semana.ts';
	import { nombreDia, sumarDias } from '#lib/operacion/fechas.ts';
	import AccionesTurno from '#lib/operacion/AccionesTurno.svelte';
	import EstadoTurno from '#lib/operacion/EstadoTurno.svelte';

	let { data, form } = $props();

	// Agrupados por día, en el orden en que vienen (la API los da por hora).
	const porDia = $derived.by(() => {
		const g = new Map<string, typeof data.turnos>();
		for (const t of data.turnos) {
			const d = dia(t.inicio);
			g.set(d, [...(g.get(d) ?? []), t]);
		}
		return [...g];
	});

	const enlace = (desde: string, hasta: string) => {
		const q = new URLSearchParams({ desde, hasta });
		if (data.filtros.profesional) q.set('profesional', data.filtros.profesional);
		if (data.filtros.estado) q.set('estado', data.filtros.estado);
		return `?${q}`;
	};
	const largo = $derived(
		Math.round((Date.parse(data.filtros.hasta) - Date.parse(data.filtros.desde)) / 86_400_000)
	);
</script>

<svelte:head><title>Turnos · Panel</title></svelte:head>

<div class="pila">
	<div class="fila" style="justify-content: space-between">
		<h1>Turnos</h1>
		<a class="boton primario" href="/panel/turnos/nuevo">Nuevo turno</a>
	</div>

	{#if data.reprogramar.length}
		<!-- «Para reprogramar» (propuesta 06): los turnos de días libres aprobados, como talones. -->
		<section class="g-bloque reprogramar" aria-labelledby="t-reprogramar">
			<span class="cabeza">
				<span class="icono lleno" aria-hidden="true">event_busy</span>
				<h2 id="t-reprogramar" class="g-bloque-titulo">Para reprogramar</h2>
				<span class="g-pildora ambar num">{data.reprogramar.length}</span>
			</span>
			<p class="g-nota">Caen en días libres aprobados. Movelos con la clienta: al moverlos o cancelarlos salen solos de la&nbsp;lista.</p>
			<ul class="talones">
				{#each data.reprogramar as r (r.id)}
					<li class="g-talon ambar" style:--tx="96px">
						<span>{fechaTalon(dia(r.turno.inicio))}<span class="num">{hora(r.turno.inicio)}</span></span>
						<span>
							<b>{r.turno.cliente.nombre}</b>
							<span>{r.turno.servicio}{r.turno.profesional ? ` · con ${r.turno.profesional}` : ''} · {NOMBRE_MOTIVO[r.motivo] ?? r.motivo}</span>
						</span>
						<span class="acciones-talon">
							<a class="g-accion" href={`/panel/turnos/${r.turno.id}`}>Mover</a>
							<form method="POST" action="?/resolver" use:enhance>
								<input type="hidden" name="id" value={r.id} />
								<button class="g-accion" aria-label={`Sacar a ${r.turno.cliente.nombre} de la lista sin moverlo`}>Ya está</button>
							</form>
						</span>
					</li>
				{/each}
			</ul>
		</section>
	{/if}
	<Mensaje {form} que="reprogramar" />

	{#if form?.error && !('que' in form)}<p class="aviso error" role="alert">{form.error}</p>{/if}
	{#if data.error}<p class="aviso error" role="alert">{data.error}</p>{/if}

	<form method="GET" class="tarjeta grilla filtros" aria-label="Filtros">
		<label class="campo">Desde <input type="date" name="desde" value={data.filtros.desde} /></label>
		<label class="campo">Hasta <input type="date" name="hasta" value={data.filtros.hasta} /></label>
		<label class="campo">
			Profesional
			<select name="profesional">
				<option value="">Todos</option>
				{#each data.profesionales as p (p.id)}
					<option value={p.id} selected={p.id === data.filtros.profesional}>{p.nombre}</option>
				{/each}
			</select>
		</label>
		<label class="campo">
			Estado
			<select name="estado">
				<option value="">Todos</option>
				{#each Object.entries(ESTADOS) as [k, v] (k)}
					<option value={k} selected={k === data.filtros.estado}>{v}</option>
				{/each}
			</select>
		</label>
		<div class="fila" style="align-self: end">
			<button class="boton">Filtrar</button>
		</div>
	</form>

	<div class="fila">
		<a class="boton chico" href={enlace(sumarDias(data.filtros.desde, -largo - 1), sumarDias(data.filtros.desde, -1))}>← Anteriores</a>
		<a class="boton chico" href={enlace(dia(), dia())}>Hoy</a>
		<a class="boton chico" href={enlace(sumarDias(data.filtros.hasta, 1), sumarDias(data.filtros.hasta, largo + 1))}>Siguientes →</a>
	</div>

	{#if data.turnos.length === 0}
		<p class="tarjeta suave">No hay turnos en estas fechas con esos filtros.</p>
	{/if}

	{#each porDia as [d, turnos] (d)}
		<section class="pila" aria-labelledby={`dia-${d}`}>
			<h2 id={`dia-${d}`}>{nombreDia(d)} <span class="suave chico">· {turnos.length}&nbsp;{turnos.length === 1 ? 'turno' : 'turnos'}</span></h2>
			<ul class="lista">
				{#each turnos as t (t.id)}
					<li class="tarjeta turno" data-turno={t.id}>
						<div class="cuando">
							<strong>{hora(t.inicio)}</strong>
							<span class="suave chico">a {hora(t.fin)}</span>
						</div>
						<div class="que">
							<a href={`/panel/turnos/${t.id}`}><strong>{t.cliente.nombre}</strong></a>
							<div class="suave chico">{t.servicio} · {t.profesional || 'Sin profesional'} · {pesos(t.precio)}</div>
							{#if t.nota}<div class="chico">{t.nota}</div>{/if}
						</div>
						<div class="estado"><EstadoTurno estado={t.estado} /></div>
						<div class="acciones"><AccionesTurno id={t.id} estado={t.estado} quien={t.cliente.nombre} /></div>
					</li>
				{/each}
			</ul>
		</section>
	{/each}
</div>

<style>
	.reprogramar .cabeza {
		display: flex;
		align-items: center;
		gap: 8px;
	}
	.reprogramar .cabeza .icono {
		color: #b26a00;
	}
	.reprogramar h2 {
		margin: 0;
	}
	.talones {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.talones .g-talon > :first-child {
		font-size: 12px;
		gap: 2px;
	}
	.acciones-talon {
		flex-wrap: wrap;
	}
	.acciones-talon form {
		display: flex;
	}
	@media (max-width: 480px) {
		/* En el teléfono las acciones bajan a una segunda fila, debajo del texto. */
		.talones .g-talon {
			grid-template-columns: var(--tx) minmax(0, 1fr);
		}
		.talones .g-talon > :first-child {
			grid-row: span 2;
		}
		.talones .g-talon > :nth-child(3) {
			padding: 0 14px 8px;
			gap: 16px;
		}
	}
	.filtros {
		grid-template-columns: repeat(auto-fill, minmax(min(130px, 100%), 1fr));
	}
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.turno {
		display: grid;
		grid-template-columns: 64px 1fr auto;
		grid-template-areas: 'cuando que estado' 'cuando acciones acciones';
		gap: 6px 12px;
		align-items: start;
	}
	.cuando {
		grid-area: cuando;
		display: flex;
		flex-direction: column;
	}
	.que {
		grid-area: que;
		min-width: 0;
		overflow-wrap: anywhere;
	}
	.estado {
		grid-area: estado;
	}
	.acciones {
		grid-area: acciones;
	}
	@media (max-width: 480px) {
		.turno {
			grid-template-columns: 56px 1fr;
			grid-template-areas: 'cuando que' 'cuando estado' 'acciones acciones';
		}
	}
</style>
