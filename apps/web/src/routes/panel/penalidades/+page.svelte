<script lang="ts">
	import { enhance } from '$app/forms';
	import { fecha } from '#lib/formato.ts';
	import Pestanias from '#lib/operacion/Pestanias.svelte';
	import { CLIENTES, GESTION } from '#lib/operacion/secciones.ts';

	let { data, form } = $props();
	const gestiona = $derived(GESTION.includes(data.negocio.rol));
	const paginas = $derived(Math.max(1, Math.ceil(data.total / data.porPagina)));
	const enlace = (pagina: number) => `?${new URLSearchParams({ pagina: String(pagina), ...(data.todas ? { todas: '1' } : {}) })}`;
	const MOTIVOS: Record<string, string> = { ausencias: 'Por ausencias', manual: 'Puesta a mano' };
	const marcados = $derived(form && 'datos' in form && form.datos && typeof form.datos === 'object' && 'marcados' in form.datos ? (form.datos.marcados as number) : null);
</script>

<svelte:head><title>Penalidades · Clientes · Panel</title></svelte:head>

<div class="pila">
	<h1>Penalidades</h1>
	<Pestanias items={CLIENTES} etiqueta="Clientes" />
	{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}
	{#if marcados !== null}<p class="aviso" role="status">Se marcaron {marcados}&nbsp;{marcados === 1 ? 'turno' : 'turnos'} como no&nbsp;vino.</p>{/if}

	<div class="grilla">
		<div class="tarjeta"><div class="suave chico">Ausencias en total</div><strong>{data.ausencias.total}</strong></div>
		<div class="tarjeta"><div class="suave chico">Últimos 30 días</div><strong>{data.ausencias.ultimos30}</strong></div>
		<div class="tarjeta"><div class="suave chico">Penalidades vigentes</div><strong>{data.ausencias.penalidadesVigentes}</strong></div>
		<div class="tarjeta"><div class="suave chico">Reincidentes</div><strong>{data.ausencias.reincidentes}</strong></div>
	</div>

	<section class="tarjeta pila">
		<div class="fila" style="justify-content: space-between">
			<h2>{data.todas ? 'Todas las penalidades' : 'Vigentes'}</h2>
			<a class="chico" href={data.todas ? '?' : '?todas=1'}>{data.todas ? 'Ver sólo vigentes' : 'Ver también las anteriores'}</a>
		</div>
		{#if data.penalidades.length === 0}
			<p class="suave">No hay penalidades{data.todas ? '' : ' vigentes'}.</p>
		{:else}
			<ul class="lista">
				{#each data.penalidades as p (p.id)}
					<li class="fila" style="justify-content: space-between">
						<span>
							<a href={`/panel/clientes/${p.clienteId}`}><strong>{p.cliente}</strong></a>
							<span class="suave chico">
								· {MOTIVOS[p.motivo] ?? p.motivo}{p.motivo === 'ausencias' ? ` (${p.ausencias})` : ''} · hasta {fecha(p.hasta)}{p.levantadaEn ? ` · levantada el ${fecha(p.levantadaEn)}` : ''}
							</span>
							{#if p.nota}<div class="chico">{p.nota}</div>{/if}
						</span>
						{#if p.vigente && gestiona}
							<form method="POST" action="?/levantar" use:enhance>
								<input type="hidden" name="id" value={p.id} />
								<button class="boton chico" aria-label={`Levantar la penalidad de ${p.cliente}`}>Levantar</button>
							</form>
						{/if}
					</li>
				{/each}
			</ul>
		{/if}
		{#if paginas > 1}
			<nav class="fila" aria-label="Páginas">
				{#if data.pagina > 1}<a class="boton chico" href={enlace(data.pagina - 1)}>← Anterior</a>{/if}
				<span class="chico suave">Página {data.pagina} de {paginas}</span>
				{#if data.pagina < paginas}<a class="boton chico" href={enlace(data.pagina + 1)}>Siguiente →</a>{/if}
			</nav>
		{/if}
	</section>

	{#if data.ajustes && gestiona}
		<section class="tarjeta pila">
			<h2>Reglas de ausencias</h2>
			<form method="POST" action="?/ajustes" use:enhance={() => async ({ update }) => update({ reset: false })} class="grilla">
				<label class="campo">Penalizar solo con tantas ausencias en 90 días (0&nbsp;apaga) <input type="number" name="umbralAusencias" min="0" max="20" value={data.ajustes.umbralAusencias} /></label>
				<label class="campo">Días sin reservar <input type="number" name="diasPenalidad" min="1" max="365" value={data.ajustes.diasPenalidad} /></label>
				<label class="fila chico" style="align-self: end; min-height: 40px"><input type="checkbox" name="marcarAusentes" checked={data.ajustes.marcarAusentes} /> Marcar solos como «no vino» los turnos que&nbsp;pasaron</label>
				<div class="fila" style="align-self: end"><button class="boton primario">Guardar</button></div>
			</form>
			<form method="POST" action="?/marcar" use:enhance>
				<button class="boton">Marcar ausentes ahora</button>
			</form>
		</section>
	{/if}
</div>

<style>
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.lista li + li {
		border-top: 1px solid var(--borde);
		padding-top: 10px;
	}
</style>
