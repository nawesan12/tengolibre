<script lang="ts">
	import { fecha, pesos } from '#lib/formato.ts';
	import Icono from '#lib/ui/Icono.svelte';

	let { data } = $props();
	const e = $derived(data.estadisticas);
	const n = (v: number) => v.toLocaleString('es-AR');
	const pct = (v: number) => `${Math.round(v)}%`;
	const variacion = $derived(
		e.turnosMesAnterior > 0 ? Math.round(((e.turnosDelMes - e.turnosMesAnterior) / e.turnosMesAnterior) * 100) : null
	);
	const notaTurnosMes = $derived(
		variacion === null
			? `${n(e.turnosMesAnterior)} el mes pasado`
			: `${variacion > 0 ? '▲ ' : variacion < 0 ? '▼ ' : ''}${Math.abs(variacion)}% contra el mes pasado`
	);

	type Kpi = { nombre: string; valor: string; nota: string; tono: 'petroleo' | 'papel' | 'ambar' };
	// El pulso, como en el diseño. Lo que la API todavía no manda cae a un número que sí existe.
	const kpis = $derived<Kpi[]>([
		{ nombre: 'Negocios activos', valor: n(e.activos), nota: `▲ ${n(e.negociosDelMes)} este mes`, tono: 'petroleo' },
		e.turnosHoy != null
			? { nombre: 'Turnos hoy', valor: n(e.turnosHoy), nota: e.porcentajeWeb != null ? `${pct(e.porcentajeWeb)} por la web` : '', tono: 'papel' }
			: { nombre: 'Turnos este mes', valor: n(e.turnosDelMes), nota: notaTurnosMes, tono: 'papel' },
		e.suscripcionesPagas != null
			? { nombre: 'Suscripciones pagas', valor: n(e.suscripcionesPagas), nota: e.mrr != null ? `MRR ${pesos(e.mrr)}` : '', tono: 'papel' }
			: { nombre: 'Ingresos este mes', valor: pesos(e.ingresosDelMes), nota: 'De los turnos atendidos', tono: 'papel' },
		e.pruebasPorVencer != null
			? {
					nombre: 'Pruebas que vencen en 3 días',
					valor: n(e.pruebasPorVencer),
					nota: e.conversion != null ? `Conversión ${pct(e.conversion)}` : '',
					tono: 'ambar'
				}
			: { nombre: 'En prueba', valor: n(e.enPrueba), nota: 'Con la prueba vigente', tono: 'ambar' }
	]);

	// Lo demás que la API ya sabe, en filas.
	const filas = $derived(
		[
			{ icono: 'storefront', nombre: 'Negocios en total', nota: `${n(e.activos)} activos`, valor: n(e.negocios), ver: true },
			{ icono: 'event', nombre: 'Turnos este mes', nota: notaTurnosMes, valor: n(e.turnosDelMes), ver: e.turnosHoy != null },
			{ icono: 'payments', nombre: 'Ingresos este mes', nota: 'De los turnos atendidos', valor: pesos(e.ingresosDelMes), ver: e.suscripcionesPagas != null },
			{ icono: 'timer', nombre: 'En prueba', nota: 'Con la prueba vigente', valor: n(e.enPrueba), ver: e.pruebasPorVencer != null },
			{ icono: 'group', nombre: 'Personas', nota: 'Con acceso a algún negocio', valor: n(e.personas), ver: true },
			{ icono: 'language', nombre: 'Dominios propios', nota: 'Verificados', valor: n(e.dominiosVerificados), ver: true }
		].filter((f) => f.ver)
	);
</script>

<svelte:head><title>Administración · Tengo Libre</title></svelte:head>

<div class="resumen">
	<h1>La plataforma</h1>

	<section class="kpis" aria-label="Números">
		{#each kpis as k, i (k.nombre)}
			<article class="kpi {k.tono}" style:animation-delay="{i * 50}ms">
				<h2>{k.nombre}</h2>
				<p class="valor tl-num" data-numero={k.nombre}>{k.valor}</p>
				{#if k.nota}<p class="nota tl-num">{k.nota.toLocaleUpperCase('es-AR')}</p>{/if}
			</article>
		{/each}
	</section>

	<section class="filas" aria-label="Más números">
		{#each filas as f (f.nombre)}
			<div class="fila">
				<Icono nombre={f.icono} lleno tam={20} />
				<span class="que"><b>{f.nombre}</b><span>{f.nota}</span></span>
				<b class="tl-num" data-numero={f.nombre}>{f.valor}</b>
			</div>
		{/each}
	</section>

	<section class="recientes" aria-labelledby="recientes">
		<div class="titulo">
			<h2 id="recientes">Negocios recientes</h2>
			<a class="ver" href="/admin/negocios">Ver todos <Icono nombre="arrow_forward" tam={18} /></a>
		</div>
		{#if e.recientes.length === 0}
			<p class="vacio">Todavía no hay negocios.</p>
		{:else}
			<ul class="lista">
				{#each e.recientes as r (r.id)}
					<li>
						<span class="quien"><b>{r.nombre}</b><span class="tl-num">/{r.slug} · alta {fecha(r.creadoEn)}</span></span>
						<a class="abrir" href={`/admin/negocios?q=${encodeURIComponent(r.slug)}`} aria-label={`Abrir ${r.nombre}`}>Abrir</a>
					</li>
				{/each}
			</ul>
		{/if}
	</section>
</div>

<style>
	.resumen {
		display: flex;
		flex-direction: column;
		gap: 18px;
	}
	h1 {
		margin: 0;
		font-size: clamp(34px, 4.4vw, 44px);
		line-height: 0.95;
		font-weight: 700;
		letter-spacing: -0.055em;
	}
	.kpis {
		display: grid;
		gap: 12px;
		grid-template-columns: repeat(4, minmax(0, 1fr));
	}
	.kpi {
		padding: 18px;
		border-radius: 20px;
		display: flex;
		flex-direction: column;
		gap: 4px;
		animation: tl-gin var(--tl-entrar) var(--tl-ease) both;
	}
	.kpi h2 {
		margin: 0;
		font-size: 13px;
		font-weight: 400;
		letter-spacing: 0;
		opacity: 0.8;
	}
	.kpi p {
		margin: 0;
	}
	.valor {
		font-size: 34px;
		font-weight: 700;
		letter-spacing: -0.05em;
		line-height: 1;
	}
	.nota {
		font-size: 12px;
		font-weight: 700;
	}
	.petroleo {
		background-color: var(--tl-petroleo);
		background-image: var(--tl-grano);
		color: #fff;
	}
	.petroleo .nota {
		color: var(--tl-menta);
	}
	.papel {
		background: var(--tl-papel);
		color: var(--tl-tinta);
	}
	.papel .nota {
		color: var(--tl-menta-texto);
	}
	.ambar {
		background: #fbe9c2;
		color: #5c3800;
	}
	.ambar h2 {
		opacity: 1;
	}
	.ambar .nota {
		color: #7a4a00;
	}
	.filas {
		display: grid;
		gap: 8px;
		grid-template-columns: repeat(auto-fill, minmax(min(260px, 100%), 1fr));
	}
	.fila {
		display: flex;
		align-items: center;
		gap: 12px;
		min-height: 56px;
		padding: 8px 14px;
		border-radius: 14px;
		background: var(--tl-papel);
	}
	.fila :global(.icono) {
		color: var(--tl-menta-texto);
	}
	.que {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
	}
	.que b {
		font-size: 14px;
	}
	.que span {
		font-size: 13px;
		color: var(--tl-sub);
	}
	.fila > b {
		font-size: 15px;
	}
	.titulo {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		margin-bottom: 8px;
	}
	.titulo h2 {
		margin: 0;
		font-size: 22px;
		font-weight: 700;
	}
	.ver,
	.abrir {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		min-height: 40px;
		font-size: 13px;
		font-weight: 700;
		color: var(--tl-menta-texto);
		text-decoration: none;
	}
	.ver:hover,
	.abrir:hover {
		text-decoration: underline;
	}
	.vacio {
		margin: 0;
		color: var(--tl-sub);
	}
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
		border-radius: 18px;
		background: var(--tl-papel);
		overflow: hidden;
	}
	.lista li {
		display: flex;
		align-items: center;
		gap: 14px;
		min-height: 58px;
		padding: 6px 18px;
		border-top: 1px solid #f0e8d8;
	}
	.lista li:first-child {
		border-top: 0;
	}
	.quien {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
	}
	.quien b {
		font-size: 14px;
		overflow-wrap: anywhere;
	}
	.quien span {
		font-size: 13px;
		color: var(--tl-sub);
		overflow-wrap: anywhere;
	}
	@media (max-width: 1023px) {
		.kpis {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}
	@media (max-width: 719px) {
		.valor {
			font-size: 28px;
		}
		.ver,
		.abrir {
			min-height: 48px;
		}
	}
</style>
