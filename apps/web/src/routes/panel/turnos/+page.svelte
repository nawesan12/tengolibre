<script lang="ts">
	import { ESTADOS, dia, hora, pesos } from '#lib/formato.ts';
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

	{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}
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
