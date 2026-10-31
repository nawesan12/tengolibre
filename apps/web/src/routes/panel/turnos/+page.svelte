<!--
	Turnos (handoff «Panel v2»): la tira de días, filtros por profesional y estado, «Sólo huecos», y la
	lista de talones con la línea de ahora y los huecos que se liberaron (para ofrecer a la lista).
-->
<script lang="ts">
	import { onMount } from 'svelte';
	import { SvelteSet } from 'svelte/reactivity';
	import Icono from '#lib/ui/Icono.svelte';
	import { dia } from '#lib/formato.ts';
	import { diasEntre, minutosDelDia } from '#lib/operacion/fechas.ts';
	import FilaTurno from '#lib/operacion/FilaTurno.svelte';
	import FilaHueco from '#lib/operacion/FilaHueco.svelte';
	import LineaAhora from '#lib/operacion/LineaAhora.svelte';
	import Chip from '#lib/operacion/Chip.svelte';
	import Caja from '#lib/operacion/Caja.svelte';
	import { armarDia, cargarEn, duracion, franjasDe, hhmm, minutosAhora, primerNombre, profesionalesDelDia } from '#lib/panel/dia.ts';
	import type { Turno } from '#lib/operacion/tipos.ts';

	let { data, form } = $props();

	let ahoraMs = $state(Date.now());
	onMount(() => {
		const r = setInterval(() => (ahoraMs = Date.now()), 30_000);
		return () => clearInterval(r);
	});
	const hoy = $derived(dia(new Date(ahoraMs)));
	const ahora = $derived(minutosAhora(new Date(ahoraMs)));
	const rol = $derived(data.negocio.rol);
	const esProfesional = $derived(rol === 'profesional');
	const puedeActuar = $derived(!data.acceso?.bloqueado);
	const f = $derived(data.filtros);

	const enlace = (cambios: Record<string, string | boolean>) => {
		const q = new URLSearchParams();
		const v = { fecha: data.fecha, profesional: f.profesional, estado: f.estado, huecos: f.huecos, cancelados: f.cancelados, ...cambios };
		if (v.fecha && v.fecha !== hoy) q.set('fecha', String(v.fecha));
		if (v.profesional) q.set('profesional', String(v.profesional));
		if (v.estado) q.set('estado', String(v.estado));
		if (v.huecos) q.set('huecos', '1');
		if (v.cancelados) q.set('cancelados', '1');
		const s = q.toString();
		return s ? `?${s}` : '/panel/turnos';
	};

	const propio = $derived((data.negocio as { profesionalId?: string | null }).profesionalId ?? null);
	const delDia = $derived(data.turnos.filter((t) => dia(t.inicio) === data.fecha));
	const profes = $derived(profesionalesDelDia(data.profesionales, delDia, rol, propio));
	const corte = $derived(data.fecha === hoy ? ahora : data.fecha < hoy ? 24 * 60 : null);
	const dias = $derived(armarDia(profes, data.turnos, data.fecha, corte));

	// La tira de días.
	const tira = $derived(
		diasEntre(data.desde, data.hasta).map((d) => {
			const vivos = data.turnos.filter((t) => dia(t.inicio) === d && t.estado !== 'cancelado').length;
			const abre = profes.some((p) => franjasDe(p, d).length > 0) || vivos > 0;
			const lleno = abre && armarDia(profes, data.turnos, d, null).every((x) => x.libres === 0);
			const p = Object.fromEntries(
				new Intl.DateTimeFormat('es-AR', { weekday: 'short', day: '2-digit', timeZone: 'UTC' }).formatToParts(new Date(`${d}T12:00:00Z`)).map((x) => [x.type, x.value])
			);
			const tipo = d === data.fecha ? 'on' : d < hoy || !abre ? 'off' : lleno ? 'lleno' : 'normal';
			const k = !abre ? 'cerrado' : d < hoy && !vivos ? '—' : lleno ? 'lleno' : String(vivos);
			return { d, dow: String(p.weekday).replace('.', ''), n: p.day, k, tipo, aria: `${new Intl.DateTimeFormat('es-AR', { weekday: 'long', day: 'numeric', month: 'long', timeZone: 'UTC' }).format(new Date(`${d}T12:00:00Z`))}: ${k === 'cerrado' ? 'cerrado' : k === 'lleno' ? 'lleno' : `${k} turnos`}` };
		})
	);

	// Filtros.
	const ESTADOS: Record<string, string[]> = { sin: ['pendiente', 'pendiente_pago'], confirmado: ['confirmado'], atendido: ['atendido', 'ausente'] };
	const delProf = $derived(f.profesional && !esProfesional ? delDia.filter((t) => t.profesionalId === f.profesional) : delDia);
	const sinConf = $derived(delProf.filter((t) => ESTADOS.sin.includes(t.estado)).length);
	const filtrados = $derived.by(() => {
		const e = f.estado;
		if (!e) return delProf;
		const lista = ESTADOS[e] ?? [e];
		return delProf.filter((t) => lista.includes(t.estado));
	});
	const cancelOcultos = $derived(f.cancelados || f.estado === 'cancelado' ? 0 : filtrados.filter((t) => t.estado === 'cancelado').length);
	const visibles = $derived(f.cancelados || f.estado === 'cancelado' ? filtrados : filtrados.filter((t) => t.estado !== 'cancelado'));

	// Huecos que dejó una cancelación (sólo de acá en adelante y si el lugar sigue libre).
	const esperanPara = (t: Turno) =>
		data.espera.filter(
			(a) =>
				!a.avisadoEn &&
				a.desde <= data.fecha &&
				a.hasta >= data.fecha &&
				(!a.profesionalId || a.profesionalId === t.profesionalId) &&
				(!t.servicioId || a.servicioId === t.servicioId)
		).length;
	const liberados = $derived(
		f.estado
			? []
			: delProf.filter((t) => {
					if (t.estado !== 'cancelado' || !t.profesionalId || Date.parse(t.inicio) <= ahoraMs) return false;
					const m = minutosDelDia(t.inicio);
					return !!dias.find((d) => d.id === t.profesionalId)?.huecos.some((h) => h.desde <= m && h.hasta > m);
				})
	);
	const ofertaPara = (t: Turno) => data.ofertas.find((o) => o.profesionalId === t.profesionalId && Date.parse(o.inicio) === Date.parse(t.inicio)) ?? null;

	type Fila =
		| { tipo: 'turno'; k: number; t: Turno }
		| { tipo: 'liberado'; k: number; t: Turno }
		| { tipo: 'ahora'; k: number }
		| { tipo: 'hueco'; k: number; h: (typeof dias)[number]['huecos'][number] };
	const filas = $derived.by<Fila[]>(() => {
		if (f.huecos) {
			return dias
				.filter((d) => !f.profesional || d.id === f.profesional)
				.flatMap((d) => d.huecos)
				.sort((a, b) => a.desde - b.desde)
				.map((h) => ({ tipo: 'hueco', k: h.desde, h }));
		}
		const out: Fila[] = [
			...visibles.map((t) => ({ tipo: 'turno' as const, k: minutosDelDia(t.inicio) + 0.1, t })),
			...liberados.map((t) => ({ tipo: 'liberado' as const, k: minutosDelDia(t.inicio), t }))
		].sort((a, b) => a.k - b.k);
		if (data.fecha === hoy && out.length) {
			const i = out.findIndex((x) => x.k > ahora);
			out.splice(i < 0 ? out.length : i, 0, { tipo: 'ahora', k: ahora });
		}
		return out;
	});

	// Lo que entra en vivo (reservas por la web) se marca un rato con contorno menta.
	const recientes = new SvelteSet<string>();
	let vistos: Set<string> | null = null;
	$effect(() => {
		const ids = data.turnos.map((t) => t.id);
		if (vistos === null) {
			vistos = new Set(ids);
			return;
		}
		for (const id of ids)
			if (!vistos.has(id)) {
				vistos.add(id);
				recientes.add(id);
				setTimeout(() => recientes.delete(id), 6000);
			}
	});

	let caja = $state<Turno | null>(null);
	const ocupado = $derived(dias.reduce((s, d) => s + d.libres, 0));
	const nombreDia = $derived(new Intl.DateTimeFormat('es-AR', { weekday: 'long', day: 'numeric', month: 'long', timeZone: 'UTC' }).format(new Date(`${data.fecha}T12:00:00Z`)));
</script>

<svelte:head><title>Turnos · Panel</title></svelte:head>

<div class="cabeza">
	<h1>Turnos</h1>
	<nav class="tira" aria-label="Días">
		{#each tira as d (d.d)}
			<a href={enlace({ fecha: d.d })} class="dia {d.tipo}" aria-current={d.tipo === 'on' ? 'date' : undefined} aria-label={d.aria} data-sveltekit-noscroll>
				<span class="dow">{d.dow}</span><b class="tl-num">{d.n}</b><span class="k tl-num">{d.k}</span>
			</a>
		{/each}
	</nav>
	<span class="crece"></span>
	{#if puedeActuar}
		<a class="nuevo" href={`/panel/turnos/nuevo?fecha=${data.fecha}`}><Icono nombre="add" tam={22} />Nuevo turno</a>
	{/if}
</div>

<div class="filtros" aria-label="Filtros">
	{#if !esProfesional && profes.length > 1}
		<Chip href={enlace({ profesional: '' })} on={!f.profesional}>Todas</Chip>
		{#each profes as p (p.id)}<Chip href={enlace({ profesional: p.id })} on={f.profesional === p.id}>{primerNombre(p.nombre)}</Chip>{/each}
		<span class="sep" aria-hidden="true"></span>
	{/if}
	<Chip href={enlace({ estado: '', huecos: false })} on={!f.estado && !f.huecos}>Todos</Chip>
	<Chip href={enlace({ estado: 'sin', huecos: false })} on={f.estado === 'sin' && !f.huecos} cuenta={sinConf}>Sin confirmar</Chip>
	<Chip href={enlace({ estado: 'confirmado', huecos: false })} on={f.estado === 'confirmado' && !f.huecos}>Confirmados</Chip>
	<Chip href={enlace({ estado: 'atendido', huecos: false })} on={f.estado === 'atendido' && !f.huecos}>Atendidos</Chip>
	<Chip href={enlace({ huecos: !f.huecos, estado: '' })} on={f.huecos} verde><span class="mini-raya" aria-hidden="true"></span>Sólo huecos</Chip>
</div>

{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}
{#if data.error}<p class="aviso error" role="alert">{data.error}</p>{/if}

<h2 class="visualmente-oculto" aria-live="polite">{nombreDia}: {visibles.length}&nbsp;{visibles.length === 1 ? 'turno' : 'turnos'}</h2>
<ul class="lista">
	{#each filas as x (x.tipo === 'ahora' ? 'ahora' : x.tipo === 'hueco' ? `h${x.h.profesionalId}${x.h.desde}` : `${x.tipo}${x.t.id}`)}
		{#if x.tipo === 'turno'}
			<FilaTurno t={x.t} ahora={ahoraMs} {puedeActuar} recien={recientes.has(x.t.id)} alAtender={(t) => (caja = t)} />
		{:else if x.tipo === 'ahora'}
			<li class="ahora"><LineaAhora hora={hhmm(ahora)} rotulo /></li>
		{:else if x.tipo === 'liberado'}
			{@const t = x.t}
			<FilaHueco
				rango={`${hhmm(minutosDelDia(t.inicio))} – ${hhmm(minutosDelDia(t.fin))}`}
				titulo={`Se liberó: ${primerNombre(t.cliente.nombre)} canceló`}
				sub={`${primerNombre(t.profesional)} · ${duracion(Math.round((Date.parse(t.fin) - Date.parse(t.inicio)) / 60000))}`}
				cargar={cargarEn(t.profesionalId ?? '', data.fecha, minutosDelDia(t.inicio))}
				esperan={rol === 'profesional' ? 0 : esperanPara(t)}
				oferta={ofertaPara(t)}
				ofrecer={{ profesionalId: t.profesionalId ?? '', inicio: t.inicio, fin: t.fin, servicioId: t.servicioId }}
				{puedeActuar}
			/>
		{:else}
			<FilaHueco
				rango={`${hhmm(x.h.desde)} – ${hhmm(x.h.hasta)}`}
				titulo={`${primerNombre(x.h.profesional)} libre ${duracion(x.h.hasta - x.h.desde)}`}
				cargar={cargarEn(x.h.profesionalId, data.fecha, x.h.desde)}
				{puedeActuar}
			/>
		{/if}
	{/each}
</ul>

{#if filas.filter((x) => x.tipo !== 'ahora').length === 0}
	{#if f.huecos}
		<div class="vacio">
			<Icono nombre="event_busy" tam={28} />
			<b>No quedan huecos {data.fecha === hoy ? 'hoy' : 'ese día'}</b>
			<a class="boton" href={enlace({ huecos: false })}>Ver los turnos</a>
		</div>
	{:else if delDia.length === 0}
		<div class="vacio">
			<span class="tl-rotulo fecha">{nombreDia}</span>
			<b class="grande">Todavía no hay turnos</b>
			<span>Tu página ya está tomando reservas. Compartí el link o cargá uno&nbsp;vos.</span>
			{#if ocupado > 0}
				{@const fr = dias.flatMap((d) => d.franjas)}
				<span class="todo-libre tl-num"><b>{hhmm(Math.min(...fr.map((x) => x.desde)))} – {hhmm(Math.max(...fr.map((x) => x.hasta)))}</b>Todo el día libre · {dias.filter((d) => d.franjas.length).length}&nbsp;{dias.filter((d) => d.franjas.length).length === 1 ? 'profesional' : 'profesionales'}</span>
			{/if}
			{#if puedeActuar}<a class="boton oscuro" href={`/panel/turnos/nuevo?fecha=${data.fecha}`}>Cargar un turno</a>{/if}
		</div>
	{:else}
		<div class="vacio centro">
			<Icono nombre="filter_alt_off" tam={28} />
			<b>No hay turnos con este filtro</b>
			<a class="boton" href={enlace({ estado: '', profesional: '', huecos: false })}>Ver todos</a>
		</div>
	{/if}
{/if}

{#if cancelOcultos > 0}
	<a class="ver-cancelados" href={enlace({ cancelados: true })} data-sveltekit-noscroll>
		<Icono nombre="visibility" tam={18} />Mostrar {cancelOcultos}&nbsp;{cancelOcultos === 1 ? 'cancelado' : 'cancelados'}
	</a>
{:else if f.cancelados}
	<a class="ver-cancelados" href={enlace({ cancelados: false })} data-sveltekit-noscroll><Icono nombre="visibility_off" tam={18} />Ocultar los cancelados</a>
{/if}

{#if caja}<Caja t={caja} productos={data.productos} alCerrar={() => (caja = null)} />{/if}

<style>
	.cabeza {
		display: flex;
		align-items: flex-end;
		flex-wrap: wrap;
		gap: 16px 20px;
	}
	h1 {
		margin: 0;
		font-size: 50px;
		line-height: 0.9;
		font-weight: 700;
		letter-spacing: -0.055em;
	}
	.tira {
		display: flex;
		gap: 6px;
		overflow-x: auto;
		scrollbar-width: none;
		max-width: 100%;
		padding: 2px;
	}
	.dia {
		flex: none;
		width: 60px;
		height: 62px;
		border-radius: 14px;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 1px;
		background: var(--tl-papel);
		color: var(--tl-tinta);
		box-shadow: 0 1px 0 rgba(15, 35, 40, 0.06), 0 6px 14px -10px rgba(15, 35, 40, 0.4);
		text-decoration: none;
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.dia:hover {
		transform: translateY(-2px);
	}
	.dow {
		font-size: 11px;
		text-transform: uppercase;
		opacity: 0.75;
	}
	.dia b {
		font-size: 19px;
		font-weight: 700;
		line-height: 1.1;
	}
	.k {
		font-size: 11px;
		color: var(--tl-sub);
	}
	.dia.on {
		background: var(--tl-petroleo);
		color: #fff;
		box-shadow: none;
	}
	.dia.on .k {
		color: var(--tl-menta);
	}
	.dia.off {
		background: transparent;
		color: #5f6f72;
		box-shadow: inset 0 0 0 1px var(--tl-linea-2);
	}
	.dia.off .k {
		color: #5f6f72;
	}
	.dia.lleno .k {
		color: #a3241a;
	}
	.crece {
		flex: 1;
	}
	.nuevo {
		white-space: nowrap;
		flex: none;
		display: inline-flex;
		align-items: center;
		gap: 8px;
		height: 52px;
		padding: 0 22px;
		border-radius: 999px;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		font-size: 16px;
		font-weight: 700;
		text-decoration: none;
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.45), 0 12px 24px -10px rgba(60, 204, 144, 0.8);
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.nuevo:hover {
		transform: translateY(-2px);
	}
	.filtros {
		display: flex;
		gap: 6px;
		align-items: center;
		overflow-x: auto;
		scrollbar-width: none;
		padding: 2px 2px 4px;
		margin: -6px -2px 0;
	}
	.sep {
		flex: none;
		width: 1px;
		height: 24px;
		background: #d8ccb4;
		margin: 0 6px;
	}
	.mini-raya {
		width: 12px;
		height: 12px;
		border-radius: 3px;
		background: repeating-linear-gradient(-45deg, #3ccc90 0 2px, #dff4e9 2px 5px);
	}
	.lista {
		container: lista / inline-size;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.ahora {
		list-style: none;
	}
	.vacio {
		padding: 40px 24px;
		border-radius: 20px;
		background: var(--tl-papel);
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 10px;
	}
	.vacio.centro {
		align-items: center;
		text-align: center;
	}
	.vacio :global(.icono) {
		color: var(--tl-menta-texto);
	}
	.vacio .fecha {
		color: var(--tl-menta-texto);
		font-size: 11px;
	}
	.vacio .grande {
		font-size: 28px;
		letter-spacing: -0.03em;
		line-height: 1.05;
	}
	.vacio > span:not(.tl-rotulo):not(.todo-libre) {
		font-size: 14px;
		color: var(--tl-sub);
		max-width: 32em;
	}
	.todo-libre {
		display: flex;
		align-items: center;
		gap: 12px;
		min-height: 48px;
		padding: 0 14px;
		border-radius: 14px;
		background: repeating-linear-gradient(-45deg, rgba(60, 204, 144, 0.22) 0 2px, transparent 2px 10px), rgba(60, 204, 144, 0.08);
		box-shadow: inset 0 0 0 1.5px rgba(23, 128, 90, 0.35);
		font-size: 13px;
		font-weight: 700;
		color: #0f3d2b;
	}
	.todo-libre b {
		color: var(--tl-menta-texto);
		font-size: 15px;
	}
	.ver-cancelados {
		align-self: center;
		display: flex;
		align-items: center;
		gap: 6px;
		min-height: 44px;
		padding: 0 14px;
		border-radius: 999px;
		color: var(--tl-sub);
		font-size: 13px;
		font-weight: 700;
		text-decoration: none;
	}
	@media (max-width: 859px) {
		h1 {
			font-size: 34px;
		}
		.nuevo {
			display: none;
		}
		.cabeza {
			gap: 12px;
		}
	}
</style>
