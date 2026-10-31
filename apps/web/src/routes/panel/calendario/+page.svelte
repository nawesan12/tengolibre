<!--
	Calendario (handoff «Panel v2»). Día: una columna por profesional, lo libre rayado, la línea menta
	de ahora; tocar lo rayado abre Nuevo turno a esa hora. Semana: cada celda es una hora con una
	barrita por profesional (ocupado petróleo, libre rayado); tocar un día lo abre.
-->
<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import Icono from '#lib/ui/Icono.svelte';
	import { dia, hora } from '#lib/formato.ts';
	import { diasEntre, minutosDelDia, sumarDias } from '#lib/operacion/fechas.ts';
	import { carriles } from '#lib/operacion/calendario.ts';
	import { ESTILO, estadoVisible } from '#lib/operacion/estados.ts';
	import Chip from '#lib/operacion/Chip.svelte';
	import { armarDia, cargarEn, franjasDe, hhmm, iniciales, libresTexto, minutosAhora, primerNombre, profesionalesDelDia, rangoDelDia } from '#lib/panel/dia.ts';
	import type { Turno } from '#lib/operacion/tipos.ts';

	let { data } = $props();

	let ahoraMs = $state(Date.now());
	onMount(() => {
		const r = setInterval(() => (ahoraMs = Date.now()), 30_000);
		return () => clearInterval(r);
	});
	const hoy = $derived(dia(new Date(ahoraMs)));
	const ahora = $derived(minutosAhora(new Date(ahoraMs)));
	const rol = $derived(data.negocio.rol);
	const puedeCargar = $derived(!data.acceso?.bloqueado);
	const PX = 1.3;

	const enlace = (cambios: Record<string, string | boolean>) => {
		const v = { vista: data.vista, fecha: data.fecha, profesional: data.profesional, cancelados: data.cancelados, ...cambios };
		const q = new URLSearchParams();
		if (v.vista === 'semana') q.set('vista', 'semana');
		if (v.fecha && v.fecha !== hoy) q.set('fecha', String(v.fecha));
		if (v.profesional) q.set('profesional', String(v.profesional));
		if (v.cancelados) q.set('cancelados', '1');
		const s = q.toString();
		return s ? `?${s}` : '/panel/calendario';
	};

	const propio = $derived((data.negocio as { profesionalId?: string | null }).profesionalId ?? null);
	const profes = $derived(profesionalesDelDia(data.profesionales, data.turnos, rol, propio));
	const corteDe = (d: string) => (d === hoy ? ahora : d < hoy ? 24 * 60 : null);

	// ── Día ──
	const diasProf = $derived(armarDia(profes, data.turnos, data.fecha, corteDe(data.fecha)));
	const rango = $derived(rangoDelDia(diasProf));
	const elegido = $derived(data.profesional && profes.some((p) => p.id === data.profesional) ? data.profesional : (profes[0]?.id ?? ''));
	const sinProf = $derived(data.turnos.filter((t) => dia(t.inicio) === data.fecha && (!t.profesionalId || !profes.some((p) => p.id === t.profesionalId))));
	const delDia = $derived(data.turnos.filter((t) => dia(t.inicio) === data.fecha));
	const columnas = $derived([
		...diasProf.map((d) => ({ id: d.id, nombre: d.nombre, libres: d.libres, franjas: d.franjas, turnos: delDia.filter((t) => t.profesionalId === d.id) })),
		...(sinProf.length ? [{ id: 'sin', nombre: 'Sin profesional', libres: 0, franjas: [], turnos: sinProf }] : [])
	]);
	// Unos píxeles más abajo para que entre la última hora del eje.
	const alto = $derived((rango[1] - rango[0]) * PX + 10);
	const top = (m: number) => (m - rango[0]) * PX;
	const horasEje = $derived(Array.from({ length: (rango[1] - rango[0]) / 60 + 1 }, (_, i) => rango[0] + i * 60));
	const pasado = $derived(data.fecha < hoy ? alto : data.fecha === hoy ? Math.min(alto, Math.max(0, top(ahora))) : 0);
	const ubicar = (ts: Turno[]) =>
		carriles(
			ts.map((t) => {
				const inicio = minutosDelDia(t.inicio);
				const largo = Math.max(10, Math.round((Date.parse(t.fin) - Date.parse(t.inicio)) / 60000));
				return { id: t.id, inicio, fin: inicio + largo, t };
			})
		);
	const cerrados = (fr: { desde: number; hasta: number }[]) => {
		const out: [number, number][] = [];
		let ini = rango[0];
		for (const f of fr) {
			if (f.desde > ini) out.push([ini, f.desde]);
			ini = Math.max(ini, f.hasta);
		}
		if (ini < rango[1]) out.push([ini, rango[1]]);
		return out;
	};
	const look = (t: Turno) => {
		const e = estadoVisible(t.estado, t.inicio, t.fin, ahoraMs);
		return { e, clase: e === 'en_curso' ? 'en-curso' : t.estado };
	};
	function tocarHueco(ev: MouseEvent, c: (typeof columnas)[number]) {
		if (!puedeCargar || c.id === 'sin' || ev.target !== ev.currentTarget) return;
		const m = rango[0] + Math.floor(ev.offsetY / PX / 15) * 15;
		if (data.fecha === hoy && m < ahora) return;
		if (data.fecha < hoy) return;
		const libre = c.franjas.some((f) => f.desde <= m && f.hasta > m) && !c.turnos.some((t) => t.estado !== 'cancelado' && minutosDelDia(t.inicio) <= m && minutosDelDia(t.fin) > m);
		if (libre) goto(cargarEn(c.id, data.fecha, m));
	}

	// ── Semana ──
	const semana = $derived(
		diasEntre(data.desde, data.hasta).filter((d) => profes.some((p) => franjasDe(p, d).length) || data.turnos.some((t) => dia(t.inicio) === d))
	);
	const semDias = $derived(
		semana.map((d) => {
			const ds = armarDia(profes, data.turnos, d, null);
			const libres = ds.reduce((s, x) => s + x.libres, 0);
			const abre = ds.some((x) => x.franjas.length);
			const p = Object.fromEntries(new Intl.DateTimeFormat('es-AR', { weekday: 'short', day: '2-digit', timeZone: 'UTC' }).formatToParts(new Date(`${d}T12:00:00Z`)).map((x) => [x.type, x.value]));
			return { d, ds, dow: String(p.weekday).replace('.', ''), n: p.day, k: !abre ? 'cerrado' : libres === 0 ? 'lleno' : `${libresTexto(libres)} libres`, lleno: abre && libres === 0 };
		})
	);
	const rangoSem = $derived(rangoDelDia(semDias.flatMap((x) => x.ds)));
	const horasSem = $derived(Array.from({ length: (rangoSem[1] - rangoSem[0]) / 60 }, (_, i) => rangoSem[0] + i * 60));
	function celda(ds: ReturnType<typeof armarDia>, d: string, h: number) {
		return ds.map((x) => {
			const trabaja = x.franjas.some((f) => f.desde < h + 60 && f.hasta > h);
			const t = x.ocupado.find((o) => o.desde < h + 60 && o.hasta > h);
			const paso = d < hoy || (d === hoy && h + 60 <= ahora);
			return { nombre: x.nombre, estado: t ? 'ocupado' : trabaja ? 'libre' : 'cerrado', paso, cliente: t?.turno.cliente.nombre ?? '' };
		});
	}
	const nombresSem = $derived(profes.map((p) => primerNombre(p.nombre)).join(', '));

	const paso = $derived(data.vista === 'semana' ? 7 : 1);
	const titulo = $derived.by(() => {
		const f = (d: string, o: Intl.DateTimeFormatOptions) => new Intl.DateTimeFormat('es-AR', { ...o, timeZone: 'UTC' }).format(new Date(`${d}T12:00:00Z`)).replace('.', '').toUpperCase();
		if (data.vista === 'dia') return `${f(data.fecha, { weekday: 'short' })} ${f(data.fecha, { day: '2-digit' })} · ${f(data.fecha, { month: 'short' })}`;
		const fin = semana.at(-1) ?? data.hasta;
		return `SEMANA ${f(data.desde, { day: '2-digit' })}–${f(fin, { day: '2-digit' })} · ${f(fin, { month: 'short' })}`;
	});
	const cuantos = $derived(data.vista === 'dia' ? delDia.length : data.turnos.length);
</script>

<svelte:head><title>Calendario · Panel</title></svelte:head>

<div class="cabeza">
	<h1>Calendario</h1>
	<div class="pastilla nav-fecha">
		<a href={enlace({ fecha: sumarDias(data.fecha, -paso) })} aria-label={data.vista === 'semana' ? 'Semana anterior' : 'Día anterior'} data-sveltekit-noscroll><Icono nombre="chevron_left" tam={20} /></a>
		<a class="hoy tl-num" href={enlace({ fecha: hoy })} title="Ir a hoy" data-sveltekit-noscroll>{titulo}</a>
		<a href={enlace({ fecha: sumarDias(data.fecha, paso) })} aria-label={data.vista === 'semana' ? 'Semana siguiente' : 'Día siguiente'} data-sveltekit-noscroll><Icono nombre="chevron_right" tam={20} /></a>
	</div>
	<div class="pastilla vistas" role="group" aria-label="Vista">
		<a href={enlace({ vista: 'dia' })} class:on={data.vista === 'dia'} aria-current={data.vista === 'dia' ? 'page' : undefined}>Día</a>
		<a href={enlace({ vista: 'semana' })} class:on={data.vista === 'semana'} aria-current={data.vista === 'semana' ? 'page' : undefined}>Semana</a>
	</div>
	<span class="crece"></span>
	<a class="cancelados" class:on={data.cancelados} href={enlace({ cancelados: !data.cancelados })} aria-current={data.cancelados ? "true" : undefined} data-sveltekit-noscroll>
		<Icono nombre={data.cancelados ? 'visibility' : 'visibility_off'} tam={18} />Cancelados
	</a>
	<a class="ical" href="/panel/calendario/suscribirse"><img src="/logos/google.svg" alt="" width="16" height="16" />Verlo en tu calendario</a>
</div>

<h2 class="visualmente-oculto" aria-live="polite">{titulo}: {cuantos}&nbsp;{cuantos === 1 ? 'turno' : 'turnos'}</h2>

{#if data.vista === 'dia'}
	{#if columnas.length > 1}
		<div class="chips-tel" aria-label="Profesional">
			{#each columnas as c (c.id)}<Chip href={enlace({ profesional: c.id === 'sin' ? '' : c.id })} on={c.id === elegido}>{primerNombre(c.nombre)}</Chip>{/each}
		</div>
	{/if}
	{#if columnas.length === 0}
		<p class="vacio">No hay profesionales ni turnos para mostrar. Cargá el equipo en <a href="/panel/equipo">Equipo y&nbsp;horarios</a>.</p>
	{:else}
		<div class="marco">
			<div class="calgrid" style:--n={columnas.length}>
				<span class="esquina"></span>
				{#each columnas as c (c.id)}
					<div class="col-cab" class:fuera={c.id !== elegido} role="columnheader">
						<span class="avatar">{iniciales(c.nombre)}</span>
						<span class="cab-txt"><b>{c.nombre}</b><span class="tl-num">{c.id === 'sin' ? `${c.turnos.length} turnos` : c.libres ? `${libresTexto(c.libres)} libres` : 'Sin huecos'}</span></span>
					</div>
				{/each}
			</div>
			<div class="calgrid cuerpo" style:--n={columnas.length} style:height={`${alto}px`}>
				<div class="eje" aria-hidden="true">
					{#each horasEje as h (h)}<span class="tl-num" style:top={`${top(h)}px`}>{hhmm(h)}</span>{/each}
				</div>
				{#each columnas as c (c.id)}
					<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
					<div class="columna" class:fuera={c.id !== elegido} class:carga={puedeCargar} onclick={(e) => tocarHueco(e, c)} title={puedeCargar ? 'Tocá lo rayado para cargar un turno' : undefined}>
						{#each cerrados(c.franjas) as [x, y] (x)}<span class="cerrado" style:top={`${top(x)}px`} style:height={`${(y - x) * PX}px`}></span>{/each}
						<span class="pasado" style:height={`${pasado}px`}></span>
						<ul aria-label={c.nombre}>
							{#each ubicar(c.turnos) as u (u.id)}
								{@const l = look(u.t)}
								{@const est = ESTILO[l.e]}
								<li
									class="bloque {l.clase}"
									class:corto={(u.fin - u.inicio) * PX < 44}
									style:top={`${top(u.inicio) + 1}px`}
									style:height={`${Math.max(24, (u.fin - u.inicio) * PX - 2)}px`}
									style:left={`calc(${(u.carril / u.carriles) * 100}% + 6px)`}
									style:width={`calc(${100 / u.carriles}% - 12px)`}
								>
									<a href={`/panel/turnos/${u.t.id}`} aria-label={`${hora(u.t.inicio)}, ${u.t.cliente.nombre}, ${u.t.servicio}, ${est?.texto ?? u.t.estado}`}>
										<span class="b-linea"><b class="tl-num">{hora(u.t.inicio)}</b><span class="b-cli">{u.t.cliente.nombre}</span><span class="b-ic" style:color={est?.color === '#fff' ? '#183c44' : est?.color}><Icono nombre={est?.icono ?? 'circle'} lleno tam={15} /></span></span>
										<span class="b-srv">{u.t.servicio}</span>
									</a>
								</li>
							{/each}
						</ul>
					</div>
				{/each}
				{#if data.fecha === hoy && ahora >= rango[0] && ahora <= rango[1]}
					<div class="linea-ahora" style:top={`${top(ahora)}px`} aria-hidden="true"><span class="tl-num">{hhmm(ahora)}</span><i></i></div>
				{/if}
			</div>
		</div>
		<p class="nota">Lo rayado está libre: tocalo para cargar un turno a esa hora. Los turnos que entran por la web aparecen solos, con contorno&nbsp;menta.</p>
	{/if}
{:else}
	<div class="marco semana">
		<div class="sem" style:--n={semDias.length}>
			<span class="sem-esq"></span>
			{#each semDias as d (d.d)}
				<a class="sem-dia" class:hoy={d.d === hoy} href={enlace({ vista: 'dia', fecha: d.d })} aria-label={`Abrir el ${d.dow} ${d.n}: ${d.k}`}>
					<span class="dow">{d.dow}</span><b class="tl-num">{d.n}</b><span class="k tl-num" class:lleno={d.lleno}>{d.k}</span>
				</a>
			{/each}
			{#each horasSem as h (h)}
				<span class="sem-h tl-num">{String(h / 60).padStart(2, '0')}</span>
				{#each semDias as d (d.d)}
					{@const cs = celda(d.ds, d.d, h)}
					<a
						class="celda"
						href={enlace({ vista: 'dia', fecha: d.d })}
						style:--p={cs.length || 1}
						aria-label={`${d.dow} ${d.n}, ${h / 60} h: ${cs.map((x) => `${x.nombre} ${x.estado === 'ocupado' ? `ocupado (${x.cliente})` : x.estado}`).join(', ')}`}
					>
						{#each cs as x, i (i)}<span class="barrita {x.estado}" class:paso={x.paso}></span>{/each}
					</a>
				{/each}
			{/each}
		</div>
	</div>
	<div class="leyenda">
		<span><i class="cuadro ocupado"></i>Ocupado</span>
		<span><i class="cuadro libre"></i>Libre</span>
		<span>Cada celda es una hora: una barrita por profesional{nombresSem ? ` (${nombresSem})` : ''}. Tocá un día para&nbsp;abrirlo.</span>
	</div>
{/if}

<style>
	.cabeza {
		display: flex;
		align-items: flex-end;
		flex-wrap: wrap;
		gap: 14px 16px;
	}
	h1 {
		margin: 0;
		font-size: 50px;
		line-height: 0.9;
		font-weight: 700;
		letter-spacing: -0.055em;
	}
	.pastilla {
		display: flex;
		align-items: center;
		gap: 2px;
		padding: 3px;
		border-radius: 999px;
		background: var(--tl-papel);
		box-shadow: inset 0 0 0 1px #e6dcc8;
	}
	.nav-fecha a {
		text-decoration: none;
		width: 38px;
		height: 38px;
		border-radius: 50%;
		display: grid;
		place-items: center;
		color: var(--tl-tinta);
	}
	.nav-fecha a:hover {
		background: var(--tl-crema);
	}
	.nav-fecha .hoy {
		width: auto;
		padding: 0 8px;
		border-radius: 999px;
		font-size: 14px;
		font-weight: 700;
		text-decoration: none;
		white-space: nowrap;
	}
	.vistas a {
		height: 38px;
		padding: 0 16px;
		border-radius: 999px;
		display: inline-flex;
		align-items: center;
		font-size: 13px;
		font-weight: 700;
		color: var(--tl-tinta);
		text-decoration: none;
	}
	.vistas a.on {
		background: var(--tl-petroleo);
		color: #fff;
	}
	.crece {
		flex: 1;
	}
	.cancelados,
	.ical {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		height: 40px;
		padding: 0 14px;
		border-radius: 999px;
		font-size: 13px;
		font-weight: 700;
		text-decoration: none;
		white-space: nowrap;
	}
	.cancelados {
		box-shadow: inset 0 0 0 1px #d8ccb4;
		color: var(--tl-tinta);
	}
	.cancelados.on {
		background: var(--tl-petroleo);
		color: #fff;
		box-shadow: none;
	}
	.ical {
		gap: 8px;
		padding: 0 16px;
		background: var(--tl-petroleo);
		color: #fff;
	}
	.ical img {
		background: #fff;
		border-radius: 50%;
		padding: 1px;
	}
	.chips-tel {
		display: none;
		gap: 6px;
		overflow-x: auto;
		scrollbar-width: none;
	}
	.marco {
		border-radius: 20px;
		background: var(--tl-papel);
		box-shadow: 0 1px 0 rgba(15, 35, 40, 0.05), 0 20px 40px -30px rgba(15, 35, 40, 0.5);
		overflow: hidden;
	}
	.calgrid {
		display: grid;
		grid-template-columns: 56px repeat(var(--n), minmax(0, 1fr));
	}
	.calgrid:first-child {
		border-bottom: 1px solid #eee5d3;
	}
	.col-cab {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 12px;
		border-left: 1px solid #eee5d3;
		min-width: 0;
	}
	.avatar {
		flex: none;
		width: 34px;
		height: 34px;
		border-radius: 50%;
		background: var(--tl-crema);
		display: grid;
		place-items: center;
		font-size: 12px;
		font-weight: 700;
	}
	.cab-txt {
		display: flex;
		flex-direction: column;
		line-height: 1.2;
		min-width: 0;
	}
	.cab-txt b {
		font-size: 14px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.cab-txt span {
		font-size: 11px;
		color: var(--tl-menta-texto);
	}
	.cuerpo {
		position: relative;
	}
	.eje {
		position: relative;
	}
	.eje span {
		position: absolute;
		right: 8px;
		transform: translateY(-6px);
		font-size: 11px;
		color: var(--tl-sub);
	}
	.columna {
		position: relative;
		border-left: 1px solid #eee5d3;
		background:
			repeating-linear-gradient(to bottom, transparent 0 77px, rgba(15, 35, 40, 0.07) 77px 78px),
			repeating-linear-gradient(-45deg, rgba(60, 204, 144, 0.16) 0 2px, transparent 2px 10px),
			#f5fbf8;
	}
	.columna.carga {
		cursor: copy;
	}
	.cerrado {
		position: absolute;
		left: 0;
		right: 0;
		background: var(--tl-papel);
		pointer-events: none;
	}
	.pasado {
		position: absolute;
		left: 0;
		right: 0;
		top: 0;
		background: rgba(250, 246, 238, 0.82);
		pointer-events: none;
	}
	.columna ul {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.bloque {
		position: absolute;
		border-radius: 10px;
		overflow: hidden;
		background: var(--tl-papel);
		color: var(--tl-tinta);
		box-shadow: 0 6px 14px -10px rgba(15, 35, 40, 0.5);
	}
	.bloque a {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 1px;
		height: 100%;
		padding: 5px 10px;
		color: inherit;
		text-decoration: none;
	}
	.b-linea {
		display: flex;
		align-items: center;
		gap: 6px;
		width: 100%;
	}
	.b-linea b {
		font-size: 12px;
		font-weight: 700;
	}
	.b-cli {
		flex: 1;
		min-width: 0;
		font-size: 13px;
		font-weight: 700;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.b-ic {
		display: inline-flex;
	}
	.b-srv {
		font-size: 12px;
		opacity: 0.75;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		max-width: 100%;
	}
	.bloque.corto .b-srv {
		display: none;
	}
	.bloque.pendiente {
		outline: 1.5px dashed #b9ab8e;
		outline-offset: -2px;
	}
	.bloque.pendiente_pago {
		background: #fbe9c2;
		color: #5c3800;
		box-shadow: none;
	}
	.bloque.atendido {
		background: #efe7d6;
		box-shadow: none;
	}
	.bloque.ausente {
		background: #fbe0dc;
		color: #a3241a;
		box-shadow: none;
	}
	.bloque.cancelado {
		background: #ece5d6;
		color: var(--tl-sub);
		box-shadow: none;
		opacity: 0.6;
	}
	.bloque.cancelado b {
		text-decoration: line-through;
	}
	.bloque.en-curso {
		outline: 2px solid var(--tl-petroleo);
		outline-offset: -2px;
	}
	.linea-ahora {
		position: absolute;
		left: 44px;
		right: 0;
		display: flex;
		align-items: center;
		pointer-events: none;
		transform: translateY(-11px);
		z-index: 3;
	}
	.linea-ahora span {
		display: inline-flex;
		align-items: center;
		height: 22px;
		padding: 0 7px;
		border-radius: 999px;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		font-size: 11px;
		font-weight: 700;
	}
	.linea-ahora i {
		flex: 1;
		height: 2px;
		background: var(--tl-menta);
	}
	.nota,
	.vacio {
		margin: 0;
		font-size: 13px;
		color: var(--tl-sub);
		text-wrap: pretty;
	}
	.vacio {
		padding: 32px 24px;
		border-radius: 20px;
		background: var(--tl-papel);
		font-size: 15px;
	}
	/* Semana */
	.semana {
		overflow-x: auto;
	}
	.sem {
		display: grid;
		grid-template-columns: 56px repeat(var(--n), minmax(96px, 1fr));
		min-width: calc(56px + var(--n) * 96px);
	}
	.sem-esq {
		border-bottom: 1px solid #eee5d3;
	}
	.sem-dia {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 2px;
		padding: 12px;
		border-left: 1px solid #eee5d3;
		border-bottom: 1px solid #eee5d3;
		color: var(--tl-tinta);
		text-decoration: none;
	}
	.sem-dia.hoy {
		background: #f3fbf7;
	}
	.sem-dia .dow {
		font-size: 11px;
		text-transform: uppercase;
		opacity: 0.75;
	}
	.sem-dia b {
		font-size: 20px;
		font-weight: 700;
		line-height: 1;
	}
	.sem-dia .k {
		font-size: 11px;
		color: var(--tl-menta-texto);
	}
	.sem-dia .k.lleno {
		color: #a3241a;
	}
	.sem-h {
		padding: 8px 8px 0 0;
		text-align: right;
		font-size: 11px;
		color: var(--tl-sub);
	}
	.celda {
		display: grid;
		grid-template-columns: repeat(var(--p), 1fr);
		gap: 3px;
		padding: 6px;
		height: 44px;
		border-left: 1px solid #eee5d3;
		border-top: 1px solid var(--tl-crema);
	}
	.celda:hover {
		background: var(--tl-crema-pagina);
	}
	.barrita {
		border-radius: 4px;
	}
	.barrita.ocupado {
		background: var(--tl-petroleo);
	}
	.barrita.ocupado.paso {
		background: rgba(24, 60, 68, 0.35);
	}
	.barrita.libre {
		background: repeating-linear-gradient(-45deg, #3ccc90 0 2px, #dff4e9 2px 5px);
	}
	.barrita.libre.paso {
		opacity: 0.4;
	}
	.leyenda {
		display: flex;
		flex-wrap: wrap;
		gap: 16px;
		font-size: 13px;
		color: var(--tl-sub);
	}
	.leyenda span {
		display: flex;
		align-items: center;
		gap: 6px;
	}
	.cuadro {
		width: 14px;
		height: 14px;
		border-radius: 4px;
	}
	.cuadro.ocupado {
		background: var(--tl-petroleo);
	}
	.cuadro.libre {
		background: repeating-linear-gradient(-45deg, #3ccc90 0 2px, #dff4e9 2px 5px);
	}
	@media (max-width: 859px) {
		h1 {
			font-size: 34px;
		}
		.cabeza {
			gap: 10px;
		}
		.cancelados,
		.ical {
			height: 44px;
		}
		.chips-tel {
			display: flex;
		}
		/* En el teléfono, una columna por vez (la elegida en los chips). */
		.calgrid {
			grid-template-columns: 56px minmax(0, 1fr);
		}
		.fuera {
			display: none;
		}
		.linea-ahora {
			left: 44px;
		}
	}
</style>
