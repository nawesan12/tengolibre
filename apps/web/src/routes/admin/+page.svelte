<script lang="ts">
	import { fecha, pesos } from '#lib/formato.ts';
	let { data } = $props();
	const e = $derived(data.estadisticas);
	const variacion = $derived(
		e.turnosMesAnterior > 0 ? Math.round(((e.turnosDelMes - e.turnosMesAnterior) / e.turnosMesAnterior) * 100) : null
	);
	const numeros = $derived([
		{ nombre: 'Negocios', valor: e.negocios, nota: `${e.activos} activos · ${e.negociosDelMes} nuevos este mes` },
		{ nombre: 'En prueba', valor: e.enPrueba, nota: 'Activos con la prueba vigente' },
		{ nombre: 'Personas', valor: e.personas, nota: 'Con acceso a algún negocio' },
		{
			nombre: 'Turnos este mes',
			valor: e.turnosDelMes,
			nota: variacion === null ? `${e.turnosMesAnterior} el mes pasado` : `${variacion > 0 ? '+' : ''}${variacion}% contra el mes pasado`
		},
		{ nombre: 'Ingresos este mes', valor: pesos(e.ingresosDelMes), nota: 'De los turnos atendidos' },
		{ nombre: 'Dominios propios', valor: e.dominiosVerificados, nota: 'Verificados' }
	]);
</script>

<svelte:head><title>Administración · Tengo Libre</title></svelte:head>

<div class="pila">
	<h1>Resumen de la plataforma</h1>
	<section class="grilla" aria-label="Números">
		{#each numeros as n (n.nombre)}
			<article class="tarjeta">
				<h2 class="chico suave">{n.nombre}</h2>
				<p class="valor" data-numero={n.nombre}>{n.valor}</p>
				<p class="chico suave">{n.nota}</p>
			</article>
		{/each}
	</section>

	<section class="tarjeta pila">
		<div class="fila" style="justify-content: space-between">
			<h2>Negocios recientes</h2>
			<a class="boton" href="/admin/negocios">Ver todos</a>
		</div>
		{#if e.recientes.length === 0}
			<p class="suave">Todavía no hay negocios.</p>
		{:else}
			<ul class="lista">
				{#each e.recientes as r (r.id)}
					<li>
						<a href={`/admin/negocios?q=${encodeURIComponent(r.slug)}`}><strong>{r.nombre}</strong></a>
						<span class="suave chico">/{r.slug} · {fecha(r.creadoEn)}</span>
					</li>
				{/each}
			</ul>
		{/if}
	</section>
</div>

<style>
	.valor {
		font-size: 26px;
		font-weight: 650;
		margin: 0;
		font-variant-numeric: tabular-nums;
	}
	h2.chico {
		font-size: 13px;
		margin: 0 0 4px;
	}
	.tarjeta p.chico {
		margin: 4px 0 0;
	}
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.lista li {
		display: flex;
		flex-wrap: wrap;
		gap: 4px 10px;
		align-items: baseline;
		padding: 8px 0;
		border-bottom: 1px solid var(--borde);
	}
</style>
