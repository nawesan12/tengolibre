<script lang="ts">
	import '#lib/gestion/gestion.css';
	import { fechaHora } from '#lib/formato.ts';
	import { cuandoCorto, describir } from '#lib/configuracion/actividad.ts';

	let { data } = $props();
	const f = $derived(data.filtros);
	const ENTIDADES = ['turno', 'cliente', 'invitacion', 'ajustes', 'aviso', 'espera', 'resena'];
	const ahora = new Date();

	/** El link de un filtro: cambia quién / plata y deja el resto (fechas, qué). */
	function con(cambios: Record<string, string>) {
		const q = new URLSearchParams();
		for (const [k, v] of Object.entries({ ...f, quien: '', plata: '', ...cambios })) if (v && k !== 'antes') q.set(k, v);
		const s = q.toString();
		return s ? `?${s}` : '?';
	}
	const todos = $derived(!f.quien && !f.plata);
	const nombreDe = (m: { nombre: string; email: string }) => (m.nombre || m.email).split(' ')[0];
</script>

<svelte:head><title>Registro de actividad · Panel</title></svelte:head>

<div class="g-pagina auditoria">
	<div class="g-col cabeza">
		<a class="g-kicker volver" href="/panel/configuracion">Configuración</a>
		<h1 class="g-titulo-2">Registro de actividad</h1>
	</div>

	<nav class="g-chips" aria-label="Filtrar por persona">
		<a class="g-chip" href={con({})} aria-current={todos ? 'page' : undefined}>Todos</a>
		{#each data.miembros as m (m.clerkId)}
			<a class="g-chip" href={con({ quien: m.clerkId })} aria-current={f.quien === m.clerkId ? 'page' : undefined}>{nombreDe(m)}</a>
		{/each}
		<a class="g-chip" href={con({ plata: '1' })} aria-current={f.plata ? 'page' : undefined}>Sólo cambios de&nbsp;plata</a>
	</nav>

	<details class="g-plegable mas" open={!!(f.desde || f.hasta || f.entidad)}>
		<summary>Fechas y qué cambió</summary>
		<form method="GET" class="g-form" aria-label="Filtros">
			{#if f.quien}<input type="hidden" name="quien" value={f.quien} />{/if}
			{#if f.plata}<input type="hidden" name="plata" value={f.plata} />{/if}
			<label class="g-campo">Desde <input type="date" name="desde" value={f.desde} /></label>
			<label class="g-campo">Hasta <input type="date" name="hasta" value={f.hasta} /></label>
			<label class="g-campo">
				Qué
				<input name="entidad" value={f.entidad} list="entidades" placeholder="turno, cliente…" autocapitalize="none" />
				<datalist id="entidades">{#each ENTIDADES as e (e)}<option value={e}></option>{/each}</datalist>
			</label>
			<div class="filtrar"><button class="g-boton oscuro">Filtrar</button></div>
		</form>
	</details>

	{#if data.entradas.length === 0}
		<div class="g-vacio"><span class="icono" aria-hidden="true">history</span>No hay cambios con esos&nbsp;filtros.</div>
	{:else}
		<ol class="g-lineas">
			{#each data.entradas as e (e.id)}
				{@const x = describir(e)}
				<li class="g-linea">
					<time datetime={e.creadoEn} title={fechaHora(e.creadoEn)}>{cuandoCorto(e.creadoEn, ahora)}</time>
					<span class="g-punto" style:background={x.color} aria-hidden="true"></span>
					<span><b>{x.quien}</b> {x.texto}<small>{x.area}</small></span>
				</li>
			{/each}
		</ol>
	{/if}

	{#if data.siguiente}
		<a class="g-boton" href={`?${data.consulta ? `${data.consulta}&` : ''}antes=${encodeURIComponent(data.siguiente)}`}>Ver más&nbsp;viejos</a>
	{/if}

	<a class="g-cta claro" href={`/panel/auditoria/exportar${data.consulta ? `?${data.consulta}` : ''}`} download>Bajar CSV</a>
</div>

<style>
	.auditoria {
		max-width: 620px;
	}
	.cabeza {
		gap: 8px;
	}
	.volver {
		text-decoration: none;
		align-self: flex-start;
	}
	.volver::before {
		content: '← ';
	}
	.mas {
		border-top: 1px solid #e6dcc8;
	}
	.mas form {
		padding: 4px 0 12px;
	}
	.filtrar {
		align-self: end;
	}
	.g-lineas {
		gap: 8px;
	}
	.g-linea {
		font-size: 14px;
	}
	.g-linea b {
		font-weight: 700;
	}
	.g-linea > span:last-child {
		overflow-wrap: anywhere;
	}
</style>
