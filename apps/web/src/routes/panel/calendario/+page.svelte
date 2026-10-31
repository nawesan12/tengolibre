<script lang="ts">
	import { dia, hora } from '#lib/formato.ts';
	import { diasEntre, minutosDelDia, nombreDia, sumarDias } from '#lib/operacion/fechas.ts';
	import { carriles, horasVisibles } from '#lib/operacion/calendario.ts';
	import type { Turno } from '#lib/operacion/tipos.ts';

	let { data } = $props();

	const PX = 1.4; // píxeles por minuto: un turno de 30' mide 42 px

	type Columna = { clave: string; titulo: string; turnos: Turno[] };

	const tramo = (t: Turno) => {
		const inicio = minutosDelDia(t.inicio);
		const largo = Math.max(10, Math.round((Date.parse(t.fin) - Date.parse(t.inicio)) / 60000));
		return { id: t.id, inicio, fin: Math.min(24 * 60, inicio + largo), turno: t };
	};

	const columnas = $derived.by<Columna[]>(() => {
		if (data.vista === 'semana')
			return diasEntre(data.desde, data.hasta).map((d) => ({
				clave: d,
				titulo: nombreDia(d),
				turnos: data.turnos.filter((t) => dia(t.inicio) === d)
			}));
		const activos = data.profesionales.filter((p) => p.activo || data.turnos.some((t) => t.profesionalId === p.id));
		const cols: Columna[] = activos.map((p) => ({
			clave: p.id,
			titulo: p.nombre,
			turnos: data.turnos.filter((t) => t.profesionalId === p.id)
		}));
		const sueltos = data.turnos.filter((t) => !t.profesionalId || !activos.some((p) => p.id === t.profesionalId));
		if (sueltos.length) cols.push({ clave: 'sin', titulo: 'Sin profesional', turnos: sueltos });
		// Quien es profesional ve sólo lo suyo: sin columnas vacías de los demás.
		return data.negocio.rol === 'profesional' ? cols.filter((c) => c.turnos.length > 0) : cols;
	});

	const ubicados = $derived(columnas.map((c) => carriles(c.turnos.map(tramo))));
	const horas = $derived(horasVisibles(ubicados.flat()));
	const alto = $derived((horas[1] - horas[0]) * 60 * PX);

	const paso = $derived(data.vista === 'semana' ? 7 : 1);
	const enlace = (cambios: Record<string, string>) => {
		const q = new URLSearchParams({ vista: data.vista, fecha: data.fecha });
		if (data.profesional) q.set('profesional', data.profesional);
		if (data.cancelados) q.set('cancelados', '1');
		for (const [k, v] of Object.entries(cambios)) if (v) q.set(k, v);
		else q.delete(k);
		return `?${q}`;
	};
	const titulo = $derived(data.vista === 'semana' ? `Semana del ${nombreDia(data.desde)} al ${nombreDia(data.hasta)}` : nombreDia(data.fecha));
</script>

<svelte:head><title>Calendario · Panel</title></svelte:head>

<div class="pila">
	<div class="fila" style="justify-content: space-between">
		<h1>Calendario</h1>
		<a class="boton primario" href={`/panel/turnos/nuevo?fecha=${data.vista === 'dia' ? data.fecha : dia()}`}>Nuevo turno</a>
	</div>

	<div class="fila barra-cal">
		<div class="fila" role="group" aria-label="Vista">
			<a class="boton chico" href={enlace({ vista: 'dia' })} aria-current={data.vista === 'dia' ? 'page' : undefined}>Día</a>
			<a class="boton chico" href={enlace({ vista: 'semana' })} aria-current={data.vista === 'semana' ? 'page' : undefined}>Semana</a>
		</div>
		<div class="fila" role="group" aria-label="Fechas">
			<a class="boton chico" href={enlace({ fecha: sumarDias(data.fecha, -paso) })} aria-label={data.vista === 'semana' ? 'Semana anterior' : 'Día anterior'}>←</a>
			<a class="boton chico" href={enlace({ fecha: dia() })}>Hoy</a>
			<a class="boton chico" href={enlace({ fecha: sumarDias(data.fecha, paso) })} aria-label={data.vista === 'semana' ? 'Semana siguiente' : 'Día siguiente'}>→</a>
		</div>
		<form method="GET" class="fila" data-sveltekit-reset="false">
			<input type="hidden" name="vista" value={data.vista} />
			{#if data.cancelados}<input type="hidden" name="cancelados" value="1" />{/if}
			<label class="campo"><span class="visualmente-oculto">Ir al día</span><input type="date" name="fecha" value={data.fecha} onchange={(e) => e.currentTarget.form?.requestSubmit()} /></label>
			{#if data.vista === 'semana'}
				<label class="campo">
					<span class="visualmente-oculto">Profesional</span>
					<select name="profesional" onchange={(e) => e.currentTarget.form?.requestSubmit()}>
						<option value="">Todos</option>
						{#each data.profesionales as p (p.id)}<option value={p.id} selected={p.id === data.profesional}>{p.nombre}</option>{/each}
					</select>
				</label>
			{/if}
			<noscript><button class="boton chico">Ir</button></noscript>
		</form>
		<a class="chico" href={enlace({ cancelados: data.cancelados ? '' : '1' })}>{data.cancelados ? 'Ocultar cancelados' : 'Mostrar cancelados'}</a>
	</div>

	<h2 aria-live="polite">{titulo} <span class="suave chico">· {data.turnos.length}&nbsp;{data.turnos.length === 1 ? 'turno' : 'turnos'}</span></h2>

	{#if columnas.length === 0}
		<p class="tarjeta suave">No hay profesionales ni turnos para mostrar. Cargá el equipo en <a href="/panel/equipo">Equipo y&nbsp;horarios</a>.</p>
	{:else}
		<div class="tarjeta marco">
			<div class="cal" style={`--columnas: ${columnas.length}; --alto: ${alto}px; --hora: ${60 * PX}px`} data-vista={data.vista}>
				<div class="esquina"></div>
				{#each columnas as c (c.clave)}
					<div class="cabeza" role="columnheader">
						{#if data.vista === 'semana'}
							<a href={enlace({ vista: 'dia', fecha: c.clave })} aria-current={c.clave === dia() ? 'date' : undefined}>{c.titulo}</a>
						{:else}
							{c.titulo}
						{/if}
					</div>
				{/each}
				<div class="horas" aria-hidden="true">
					{#each Array.from({ length: horas[1] - horas[0] }, (_, i) => horas[0] + i) as h (h)}
						<span style={`top: ${(h - horas[0]) * 60 * PX}px`}>{String(h).padStart(2, '0')}:00</span>
					{/each}
				</div>
				{#each columnas as c, i (c.clave)}
					<ul class="columna" aria-label={c.titulo}>
						{#each ubicados[i] as u (u.id)}
							<li
								class="bloque {u.turno.estado}"
								style={`top: ${(u.inicio - horas[0] * 60) * PX}px; height: ${Math.max(22, (u.fin - u.inicio) * PX - 2)}px; left: calc(${(u.carril / u.carriles) * 100}% + 2px); width: calc(${100 / u.carriles}% - 4px)`}
							>
								<a href={`/panel/turnos/${u.turno.id}`}>
									<strong>{hora(u.turno.inicio)}</strong>
									{u.turno.cliente.nombre}
									<span class="detalle">{u.turno.servicio}{data.vista === 'semana' && u.turno.profesional ? ` · ${u.turno.profesional}` : ''}</span>
								</a>
							</li>
						{/each}
					</ul>
				{/each}
			</div>
		</div>
	{/if}
</div>

<style>
	.barra-cal {
		justify-content: space-between;
	}
	.barra-cal a[aria-current='page'] {
		background: var(--marca-suave);
		border-color: var(--marca);
	}
	.marco {
		padding: 0;
		overflow: auto;
		max-height: 75dvh;
	}
	.cal {
		display: grid;
		grid-template-columns: 52px repeat(var(--columnas), minmax(132px, 1fr));
		grid-template-rows: auto var(--alto);
		min-width: min-content;
	}
	.esquina,
	.cabeza {
		position: sticky;
		top: 0;
		z-index: 2;
		background: var(--superficie);
		border-bottom: 1px solid var(--borde);
		padding: 8px;
		font-weight: 600;
		font-size: 14px;
		text-align: center;
		overflow-wrap: anywhere;
	}
	.esquina {
		left: 0;
		z-index: 3;
	}
	.cabeza a[aria-current='date'] {
		color: var(--marca);
	}
	.horas {
		position: sticky;
		left: 0;
		z-index: 1;
		background: var(--superficie);
		border-right: 1px solid var(--borde);
	}
	.horas span {
		position: absolute;
		right: 6px;
		font-size: 11px;
		color: var(--suave);
		transform: translateY(-1px);
	}
	.columna {
		position: relative;
		list-style: none;
		margin: 0;
		padding: 0;
		border-left: 1px solid var(--borde);
		background-image: repeating-linear-gradient(to bottom, var(--borde) 0 1px, transparent 1px var(--hora));
	}
	.bloque {
		position: absolute;
		overflow: hidden;
		border-radius: 6px;
		background: var(--marca-suave);
		border-left: 3px solid var(--marca);
		font-size: 12px;
		line-height: 1.25;
	}
	.bloque a {
		display: block;
		height: 100%;
		padding: 3px 6px;
		text-decoration: none;
	}
	.bloque .detalle {
		display: block;
		color: var(--suave);
	}
	.bloque.pendiente,
	.bloque.pendiente_pago {
		background: #fff4e0;
		border-left-color: var(--aviso);
	}
	.bloque.atendido {
		background: var(--borde);
		border-left-color: var(--suave);
	}
	.bloque.ausente,
	.bloque.cancelado {
		background: #fdecea;
		border-left-color: var(--peligro);
		text-decoration: line-through;
	}
</style>
