<script lang="ts">
	import '#lib/gestion/gestion.css';
	import { dia, pesos } from '#lib/formato.ts';
	import { lunesDe } from '#lib/operacion/fechas.ts';
	import SinPermiso from '#lib/operacion/SinPermiso.svelte';
	import BarrasDias from '#lib/gestion/reportes/BarrasDias.svelte';
	import HorasMuertas from '#lib/gestion/reportes/HorasMuertas.svelte';

	let { data, form } = $props();
	const r = $derived(data.reporte);
	const hoy = dia();
	const MESES = ['enero', 'febrero', 'marzo', 'abril', 'mayo', 'junio', 'julio', 'agosto', 'septiembre', 'octubre', 'noviembre', 'diciembre'];
	const mesDe = (d: string) => MESES[Number(d.slice(5, 7)) - 1];
	const mayus = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);

	const periodos = [
		{ clave: 'semana', nombre: 'Semana', desde: lunesDe(hoy) },
		{ clave: 'mes', nombre: mayus(mesDe(hoy)), desde: `${hoy.slice(0, 8)}01` },
		{ clave: 'anio', nombre: 'Año', desde: `${hoy.slice(0, 5)}01-01` }
	];
	const actual = $derived(r && r.hasta === hoy ? periodos.find((p) => p.desde === r.desde)?.clave : undefined);
	const enPeriodo = $derived.by(() => {
		if (!r) return '';
		if (actual === 'semana') return 'esta semana';
		if (actual === 'anio') return `en ${hoy.slice(0, 4)}`;
		if (r.desde.endsWith('-01') && r.desde.slice(0, 7) === r.hasta.slice(0, 7)) return `en ${mesDe(r.desde)}`;
		return 'en el período';
	});
	const rango = $derived(r ? `desde=${r.desde}&hasta=${r.hasta}` : '');

	// Comparación con el período anterior (el mes anterior si se mira un mes).
	const cambio = $derived.by(() => {
		if (!r || !data.previo || data.previo.ingresos <= 0) return null;
		const p = Math.round(((r.resumen.ingresos - data.previo.ingresos) / data.previo.ingresos) * 100);
		return { p, contra: data.previo.mes ? mesDe(data.previo.desde) : 'el período anterior' };
	});

	const compacto = (n: number) =>
		n >= 1_000_000 ? `$ ${new Intl.NumberFormat('es-AR', { maximumFractionDigits: 1 }).format(n / 1_000_000)} M` : pesos(n);
	const pct = (n: number, total: number) => (total ? Math.round((n / total) * 100) : 0);

	const estados = $derived.by(() => {
		if (!r) return null;
		const e = r.porEstado;
		const total = Object.values(e).reduce((a, b) => a + b, 0);
		const porVenir = Object.entries(e)
			.filter(([k]) => !['atendido', 'ausente', 'cancelado'].includes(k))
			.reduce((a, [, v]) => a + v, 0);
		return {
			total,
			tramos: [
				{ clave: 'atendido', texto: 'Atendidos', n: e.atendido ?? 0 },
				{ clave: 'ausente', texto: 'No vinieron', n: e.ausente ?? 0 },
				{ clave: 'cancelado', texto: 'Cancelados', n: e.cancelado ?? 0 },
				{ clave: 'porvenir', texto: 'Por venir', n: porVenir }
			].filter((t) => t.n > 0)
		};
	});

	const servicios = $derived(r ? [...r.porServicio].sort((a, b) => b.turnos - a.turnos).slice(0, 4) : []);
	const maxServicio = $derived(Math.max(1, ...servicios.map((s) => s.turnos)));
	const profesionales = $derived(r ? [...r.porProfesional].sort((a, b) => b.atendidos - a.atendidos || b.turnos - a.turnos) : []);
	const vuelven = $derived(r && r.resumen.clientes ? pct(r.clientesRecurrentes, r.resumen.clientes) : null);
</script>

<svelte:head><title>Reportes · Panel</title></svelte:head>

<div class="g-pagina">
	<div class="g-cabeza">
		<h1 class="g-titulo">Reportes</h1>
		{#if r}
			<nav class="g-segmento" aria-label="Período">
				{#each periodos as p (p.clave)}
					<a class="g-chip" href={`?desde=${p.desde}&hasta=${hoy}`} aria-current={actual === p.clave ? 'page' : undefined}>{p.nombre}</a>
				{/each}
			</nav>
		{/if}
		<a class="g-boton mensual" href="/panel/reportes/mensual"><span class="icono" aria-hidden="true">description</span>Reporte mensual</a>
	</div>

	{#if data.error}<p class="aviso error" role="alert">{data.error}</p>{/if}

	{#if !r && !data.error}
		<SinPermiso que="los reportes" />
	{/if}

	{#if r}
		<details class="g-plegable otro" open={!actual}>
			<summary>Otro período</summary>
			<form method="GET" class="fechas" aria-label="Elegir fechas">
				<label class="g-campo">Desde <input type="date" name="desde" value={r.desde} /></label>
				<label class="g-campo">Hasta <input type="date" name="hasta" value={r.hasta} /></label>
				<button class="g-boton oscuro">Ver</button>
			</form>
		</details>

		<div class="g-tres arriba">
			<section class="g-oscuro facture" aria-labelledby="t-facture">
				<h2 class="pregunta claro" id="t-facture">¿Cuánto facturé {enPeriodo}?</h2>
				<b class="plata">{pesos(r.resumen.ingresos)}</b>
				{#if cambio}
					<span class="cambio">
						<span class="icono" aria-hidden="true">{cambio.p >= 0 ? 'arrow_upward' : 'arrow_downward'}</span>
						{Math.abs(cambio.p)} % {cambio.p >= 0 ? 'más' : 'menos'} que {cambio.contra}
					</span>
				{/if}
				<span class="pie">{r.resumen.atendidos} turnos atendidos · ticket promedio {pesos(r.resumen.ticketPromedio)}</span>
			</section>

			<section class="g-bloque" aria-labelledby="t-terminaron">
				<h2 class="pregunta" id="t-terminaron">¿Cómo terminaron los turnos?</h2>
				{#if !estados || estados.total === 0}
					<p class="g-nota">Sin turnos en el&nbsp;período.</p>
				{:else}
					<div class="apilada" aria-hidden="true">
						{#each estados.tramos as t (t.clave)}<span class="tramo {t.clave}" style:flex={t.n}></span>{/each}
					</div>
					<ul class="leyenda">
						{#each estados.tramos as t (t.clave)}
							<li class={t.clave}>{pct(t.n, estados.total)} % {t.texto} · {t.n}</li>
						{/each}
					</ul>
				{/if}
			</section>

			<section class="g-bloque" aria-labelledby="t-faltan">
				<h2 class="pregunta" id="t-faltan">¿Quién falta más?</h2>
				{#if data.ausentes && data.ausentes.length > 0}
					<ul class="faltan">
						{#each data.ausentes.slice(0, 3) as c (c.id)}
							<li>
								<b>{c.nombre}</b>
								<span class="veces" aria-label={`${c.ausencias} ${c.ausencias === 1 ? 'ausencia' : 'ausencias'}`}>
									{c.ausencias}<span class="icono" aria-hidden="true">close</span>
								</span>
							</li>
						{/each}
					</ul>
				{:else if data.ausentes}
					<p class="g-nota"><span class="icono ok" aria-hidden="true">check_circle</span> Nadie faltó sin avisar en el&nbsp;período.</p>
				{:else}
					<p class="faltaron"><b>{r.porEstado.ausente ?? 0}</b> {(r.porEstado.ausente ?? 0) === 1 ? 'turno' : 'turnos'} sin aviso</p>
				{/if}
				<a class="g-accion chica" href="/panel/penalidades">Ver penalidades →</a>
			</section>
		</div>

		<section class="g-bloque dias" aria-labelledby="t-dias">
			<div class="g-bloque-cabeza">
				<h2 class="pregunta" id="t-dias">¿Qué días se llena y qué días sobra&nbsp;lugar?</h2>
				<a class="g-accion chica" href={`/panel/reportes/csv/turnos?${rango}`} download data-sveltekit-reload>CSV <span class="icono" aria-hidden="true">arrow_downward</span></a>
			</div>
			{#if r.porDia.length === 0}
				<p class="g-nota">Sin días en el&nbsp;período.</p>
			{:else}
				<BarrasDias dias={r.porDia} />
			{/if}
		</section>

		{#if data.horasMuertas}<HorasMuertas datos={data.horasMuertas} {form} />{/if}

		<div class="g-tres">
			<section class="g-bloque" aria-labelledby="t-servicio">
				<h2 class="pregunta" id="t-servicio">¿Qué servicio se pide más?</h2>
				{#if servicios.length === 0}
					<p class="g-nota">Sin turnos en el&nbsp;período.</p>
				{:else}
					<ul class="servicios">
						{#each servicios as s (s.id ?? s.nombre)}
							<li>
								<span class="pista"><span class="relleno" style:width="{(s.turnos / maxServicio) * 100}%"></span><span class="nombre">{s.nombre || 'Sin nombre'}</span></span>
								<b>{s.turnos}</b>
							</li>
						{/each}
					</ul>
				{/if}
			</section>

			<section class="g-bloque" aria-labelledby="t-quien">
				<h2 class="pregunta" id="t-quien">¿Quién atiende más?</h2>
				{#if profesionales.length === 0}
					<p class="g-nota">Sin turnos en el&nbsp;período.</p>
				{:else}
					<ul class="quienes">
						{#each profesionales as p (p.id ?? p.nombre)}
							<li><b>{p.nombre || 'Sin nombre'}</b><span>{p.atendidos} · {compacto(p.ingresos)}</span></li>
						{/each}
					</ul>
				{/if}
			</section>

			<section class="g-bloque" aria-labelledby="t-vuelven">
				<div class="g-bloque-cabeza">
					<h2 class="pregunta" id="t-vuelven">¿Vuelven?</h2>
					<a class="g-accion chica" href={`/panel/reportes/csv/clientes?${rango}`} download data-sveltekit-reload>CSV <span class="icono" aria-hidden="true">arrow_downward</span></a>
				</div>
				{#if vuelven === null}
					<p class="g-nota">Sin clientes en el&nbsp;período.</p>
				{:else}
					<b class="grande">{vuelven}%</b>
					<p class="g-nota">de tus clientes del período ya habían venido. {r.clientesNuevos}&nbsp;{r.clientesNuevos === 1 ? 'nuevo' : 'nuevos'}.</p>
				{/if}
			</section>
		</div>
	{/if}
</div>

<style>
	.g-cabeza {
		align-items: center;
	}
	.otro {
		margin-top: -8px;
	}
	.fechas {
		display: flex;
		flex-wrap: wrap;
		align-items: end;
		gap: 12px;
		padding-bottom: 8px;
	}
	.pregunta {
		margin: 0;
		font-size: 14px;
		font-weight: 400;
		line-height: 1.35;
		letter-spacing: normal;
		color: var(--tl-sub);
	}
	.pregunta.claro {
		color: rgba(255, 255, 255, 0.82);
	}
	.g-oscuro.facture {
		border-radius: var(--tl-r-l);
		padding: 24px 20px;
		gap: 6px;
	}
	.plata {
		font: 700 clamp(34px, 4vw, 44px) / 1 var(--tl-font);
		font-variant-numeric: tabular-nums;
		letter-spacing: -0.05em;
		margin: 6px 0 4px;
	}
	.cambio {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		font-size: 13px;
		font-variant-numeric: tabular-nums;
		text-transform: uppercase;
		color: var(--tl-menta);
	}
	.cambio .icono {
		font-size: 16px;
	}
	.pie {
		font-size: 13px;
		font-variant-numeric: tabular-nums;
		color: rgba(255, 255, 255, 0.82);
	}
	.apilada {
		display: flex;
		height: 14px;
		border-radius: 999px;
		overflow: hidden;
		background: var(--tl-crema);
	}
	.tramo.atendido {
		background: var(--tl-petroleo);
	}
	.tramo.ausente {
		background: #a3241a;
	}
	.tramo.cancelado {
		background: #cfc3a8;
	}
	.tramo.porvenir {
		background: var(--tl-menta);
	}
	.leyenda {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 4px;
		font-size: 12px;
		font-variant-numeric: tabular-nums;
		text-transform: uppercase;
		color: var(--tl-sub-2);
	}
	.leyenda .ausente {
		color: #a3241a;
	}
	.leyenda .porvenir {
		color: var(--tl-menta-texto);
	}
	.faltan,
	.quienes,
	.servicios {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	.faltan li,
	.quienes li {
		display: flex;
		justify-content: space-between;
		gap: 12px;
		font-size: 14px;
	}
	.faltan b,
	.quienes b {
		font-weight: 700;
		overflow-wrap: anywhere;
	}
	.veces {
		display: inline-flex;
		align-items: center;
		color: #a3241a;
		font-weight: 700;
		font-variant-numeric: tabular-nums;
	}
	.veces .icono {
		font-size: 16px;
	}
	.quienes span {
		font-size: 13px;
		font-variant-numeric: tabular-nums;
		color: var(--tl-sub-2);
		white-space: nowrap;
	}
	.faltaron {
		margin: 0;
		font-size: 14px;
	}
	.faltaron b {
		font-size: 22px;
		font-variant-numeric: tabular-nums;
	}
	.ok {
		font-size: 16px;
		color: var(--tl-menta-texto);
		vertical-align: -3px;
	}
	.chica {
		font-size: 12px;
		min-height: 32px;
		align-self: flex-start;
	}
	.chica .icono {
		font-size: 14px;
	}
	.servicios li {
		display: grid;
		grid-template-columns: minmax(0, 1fr) 36px;
		align-items: center;
		gap: 12px;
	}
	.pista {
		position: relative;
		min-height: 28px;
		display: flex;
		align-items: center;
	}
	.relleno {
		position: absolute;
		inset: 0 auto 0 0;
		border-radius: 8px;
		background: var(--tl-menta-suave);
	}
	.nombre {
		position: relative;
		padding: 4px 8px;
		font-size: 13px;
		overflow-wrap: anywhere;
	}
	.servicios b {
		text-align: right;
		font-variant-numeric: tabular-nums;
	}
	.grande {
		font: 700 40px/1 var(--tl-font);
		font-variant-numeric: tabular-nums;
		letter-spacing: -0.04em;
	}
	@media (max-width: 640px) {
		.chica {
			min-height: 44px;
		}
	}
</style>
