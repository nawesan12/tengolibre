<script lang="ts">
	import { enhance } from '$app/forms';
	import { fecha } from '#lib/formato.ts';
	import { rubro } from '#lib/publico/textos.ts';
	import Icono from '#lib/ui/Icono.svelte';
	import EstadoPastilla from '#lib/cuenta/EstadoPastilla.svelte';
	import { paginas, prueba } from '#lib/cuenta/util.ts';
	import { ESTADOS_NEGOCIO } from '#lib/cuenta/contratos.ts';
	import { estadoAdmin, NOMBRE_ESTADO } from '#lib/cuenta/presentacion.ts';

	let { data, form } = $props();
	const l = $derived(data.listado);
	const total = $derived(paginas(l.total, l.limite));
	const enlace = (cambios: { pagina?: number; estado?: string }) => {
		const p = new URLSearchParams();
		if (data.q) p.set('q', data.q);
		const estado = cambios.estado ?? data.estado;
		if (estado) p.set('estado', estado);
		if (cambios.pagina && cambios.pagina > 1) p.set('pagina', String(cambios.pagina));
		const s = p.toString();
		return s ? `?${s}` : '/admin/negocios';
	};
</script>

<svelte:head><title>Negocios · Administración</title></svelte:head>

<div class="negocios">
	<h1>Negocios</h1>

	<div class="barra">
		<form method="GET" class="buscar" role="search">
			{#if data.estado}<input type="hidden" name="estado" value={data.estado} />{/if}
			<label class="campo-buscar">
				<Icono nombre="search" tam={20} />
				<span class="visualmente-oculto">Buscar</span>
				<input type="search" name="q" value={data.q} placeholder="Buscar negocio, dueño o slug" maxlength="120" />
			</label>
			<button class="ir">Buscar</button>
			{#if data.q}<a class="limpiar" href={data.estado ? `?estado=${data.estado}` : '/admin/negocios'}>Limpiar</a>{/if}
		</form>
		{#if l.conteos}
			<nav class="filtros" aria-label="Filtrar por estado">
				<a class="chip" class:activo={!data.estado} href={enlace({ estado: '' })} aria-current={!data.estado ? 'true' : undefined}>Todos</a>
				{#each ESTADOS_NEGOCIO as e (e)}
					<a class="chip" class:activo={data.estado === e} href={enlace({ estado: e })} aria-current={data.estado === e ? 'true' : undefined}>
						{NOMBRE_ESTADO[e]} <span class="tl-num">{l.conteos[e] ?? 0}</span>
					</a>
				{/each}
			</nav>
		{/if}
	</div>

	<p class="total" aria-live="polite">
		{l.total === 1 ? 'Un negocio' : `${l.total} negocios`}{data.q ? ` para “${data.q}”` : ''}
	</p>
	{#if form?.error}<p class="error" role="alert"><Icono nombre="error" lleno tam={20} /><span>{form.error}</span></p>{/if}

	{#if l.negocios.length === 0}
		<p class="vacio">No hay negocios que coincidan.</p>
	{:else}
		<ul class="lista">
			{#each l.negocios as n (n.id)}
				{@const estado = estadoAdmin(n)}
				<li aria-label={n.nombre}>
					<details>
						<summary>
							<span class="quien">
								<b>{n.nombre}</b>
								<span class="tl-num">/{n.slug}{n.ciudad ? ` · ${n.ciudad}` : ''}{n.rubro ? ` · ${rubro(n.rubro)}` : ''}</span>
							</span>
							<span class="duenia">{n.duenia ?? '—'}</span>
							<span class="turnos tl-num">{n.turnos30Dias.toLocaleString('es-AR')} / 30 DÍAS</span>
							<span class="estado">
								<EstadoPastilla {estado} plano />
								{#if n.activo && !n.reservable}<EstadoPastilla estado={{ texto: 'No recibe turnos', icono: 'event_busy', tono: 'apagado' }} plano />{/if}
							</span>
							<span class="abrir">Abrir <Icono nombre="expand_more" tam={18} /></span>
						</summary>
						<div class="detalle">
							<dl class="datos">
								<div><dt>Dueña</dt><dd>{n.duenia ?? '—'}</dd></div>
								<div><dt>Equipo</dt><dd class="tl-num">{n.miembros}</dd></div>
								<div><dt>Alta</dt><dd>{fecha(n.creadoEn)}</dd></div>
								<div><dt>Prueba</dt><dd>{prueba(n.pruebaHasta)}</dd></div>
								<div><dt>Cuenta</dt><dd>{n.activo ? 'Activa' : 'Desactivada'}</dd></div>
							</dl>
							<div class="acciones">
								<a class="pastilla clara" href={`/${n.slug}`} target="_blank" rel="noopener"><Icono nombre="open_in_new" tam={18} />Ver la página</a>
								<form method="POST" action="?/activo" use:enhance={() => async ({ update }) => update({ reset: false })}>
									<input type="hidden" name="id" value={n.id} />
									<input type="hidden" name="activo" value={String(!n.activo)} />
									<button class="pastilla" class:peligro={n.activo}>
										<Icono nombre={n.activo ? 'block' : 'check_circle'} tam={18} />{n.activo ? 'Desactivar' : 'Activar'}
									</button>
								</form>
								<form method="POST" action="?/extender" use:enhance={() => async ({ update }) => update({ reset: false })} class="extender">
									<input type="hidden" name="id" value={n.id} />
									<label class="dias">
										<span class="visualmente-oculto">Días para sumar a la prueba de {n.nombre}</span>
										<input name="dias" type="number" min="1" max="365" value="7" required />
										<span aria-hidden="true">días</span>
									</label>
									<button class="pastilla oscura"><Icono nombre="more_time" tam={18} />Extender prueba</button>
								</form>
							</div>
						</div>
					</details>
				</li>
			{/each}
		</ul>
	{/if}

	{#if total > 1}
		<nav class="paginas" aria-label="Páginas">
			{#if l.pagina > 1}<a class="chip" href={enlace({ pagina: l.pagina - 1 })}><Icono nombre="arrow_back" tam={18} />Anterior</a>{:else}<span></span>{/if}
			<span class="tl-num">Página {l.pagina} de {total}</span>
			{#if l.pagina < total}<a class="chip" href={enlace({ pagina: l.pagina + 1 })}>Siguiente<Icono nombre="arrow_forward" tam={18} /></a>{:else}<span></span>{/if}
		</nav>
	{/if}
</div>

<style>
	.negocios {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	h1 {
		margin: 0;
		font-size: clamp(34px, 4.4vw, 44px);
		line-height: 0.95;
		font-weight: 700;
		letter-spacing: -0.055em;
	}
	.barra {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		align-items: center;
	}
	.buscar {
		flex: 1 1 320px;
		display: flex;
		gap: 8px;
		align-items: center;
	}
	.campo-buscar {
		flex: 1;
		min-width: 0;
		display: flex;
		align-items: center;
		gap: 8px;
		height: 46px;
		padding: 0 14px;
		border-radius: 14px;
		background: var(--tl-papel);
		box-shadow: inset 0 0 0 1px #e6dcc8;
		color: var(--tl-sub);
	}
	.campo-buscar:focus-within {
		box-shadow: inset 0 0 0 2px var(--tl-petroleo), 0 0 0 3px rgba(60, 204, 144, 0.35);
	}
	.campo-buscar input {
		flex: 1;
		min-width: 0;
		height: 100%;
		border: 0;
		background: none;
		font: 400 16px var(--tl-font);
		color: var(--tl-tinta);
		outline: none;
	}
	.ir,
	.pastilla {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		height: 40px;
		padding: 0 16px;
		border: 0;
		border-radius: 999px;
		background: var(--tl-papel);
		box-shadow: inset 0 0 0 1px #d8ccb4;
		color: var(--tl-tinta);
		font: 700 13px var(--tl-font);
		text-decoration: none;
		cursor: pointer;
		white-space: nowrap;
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.ir {
		height: 46px;
		background: var(--tl-petroleo);
		color: #fff;
		box-shadow: var(--tl-luz-arriba-osc);
	}
	.ir:active,
	.pastilla:active {
		transform: scale(0.97);
	}
	.pastilla.oscura {
		background: var(--tl-petroleo);
		color: #fff;
		box-shadow: var(--tl-luz-arriba-osc);
	}
	.pastilla.peligro {
		color: #a3241a;
		box-shadow: inset 0 0 0 1px #f0c9c4;
	}
	.limpiar {
		font-size: 13px;
		font-weight: 700;
		color: var(--tl-menta-texto);
		min-height: 40px;
		display: inline-flex;
		align-items: center;
	}
	.filtros {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}
	.chip {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		height: 40px;
		padding: 0 14px;
		border-radius: 999px;
		box-shadow: inset 0 0 0 1px #d8ccb4;
		color: var(--tl-tinta);
		font-size: 13px;
		font-weight: 700;
		text-decoration: none;
	}
	.chip.activo {
		background: var(--tl-petroleo);
		color: #fff;
		box-shadow: none;
	}
	.total {
		margin: 0;
		font-size: 13px;
		color: var(--tl-sub);
	}
	.error {
		display: flex;
		gap: 10px;
		margin: 0;
		padding: 14px;
		border-radius: 16px;
		background: #fbe0dc;
		color: #5b1a13;
		font-size: 14px;
	}
	.vacio {
		margin: 0;
		padding: 20px;
		border-radius: 18px;
		background: var(--tl-papel);
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
		border-top: 1px solid #f0e8d8;
	}
	.lista li:first-child {
		border-top: 0;
	}
	summary {
		display: grid;
		grid-template-columns: minmax(0, 1.4fr) minmax(0, 1fr) 130px 190px 80px;
		align-items: center;
		gap: 14px;
		min-height: 58px;
		padding: 6px 18px;
		cursor: pointer;
		list-style: none;
	}
	summary::-webkit-details-marker {
		display: none;
	}
	summary:hover {
		background: rgba(244, 238, 226, 0.5);
	}
	.quien {
		display: flex;
		flex-direction: column;
		min-width: 0;
	}
	.quien b {
		font-size: 14px;
		overflow-wrap: anywhere;
	}
	.quien span,
	.duenia {
		font-size: 13px;
		color: var(--tl-sub);
		overflow-wrap: anywhere;
	}
	.turnos {
		font-size: 13px;
		font-weight: 700;
	}
	.estado {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}
	.abrir {
		justify-self: end;
		display: inline-flex;
		align-items: center;
		gap: 2px;
		font-size: 13px;
		font-weight: 700;
		color: var(--tl-menta-texto);
	}
	.abrir :global(.icono) {
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	details[open] .abrir :global(.icono) {
		transform: rotate(180deg);
	}
	.detalle {
		display: flex;
		flex-direction: column;
		gap: 14px;
		padding: 4px 18px 18px;
		animation: tl-gin var(--tl-entrar) var(--tl-ease) both;
	}
	.datos {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(150px, 100%), 1fr));
		gap: 8px 16px;
		margin: 0;
		padding: 14px 16px;
		border-radius: 14px;
		background: var(--tl-crema-pagina);
	}
	.datos dt {
		font-size: 11px;
		font-weight: 900;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--tl-sub);
	}
	.datos dd {
		margin: 0;
		font-size: 14px;
		overflow-wrap: anywhere;
	}
	.acciones {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		align-items: center;
	}
	.extender {
		display: flex;
		gap: 8px;
		align-items: center;
	}
	.dias {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		height: 40px;
		padding: 0 12px 0 4px;
		border-radius: 999px;
		box-shadow: inset 0 0 0 1px #d8ccb4;
		font-size: 13px;
		color: var(--tl-sub);
	}
	.dias input {
		width: 56px;
		height: 32px;
		border: 0;
		border-radius: 999px;
		background: none;
		text-align: center;
		font: 700 16px var(--tl-font);
		font-variant-numeric: tabular-nums;
		color: var(--tl-tinta);
	}
	.paginas {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		font-size: 13px;
		color: var(--tl-sub);
	}
	@media (max-width: 899px) {
		summary {
			grid-template-columns: minmax(0, 1fr) auto;
			grid-template-areas: 'quien abrir' 'estado turnos';
			gap: 6px 12px;
			padding: 12px 16px;
		}
		.quien {
			grid-area: quien;
		}
		.duenia {
			display: none;
		}
		.estado {
			grid-area: estado;
		}
		.turnos {
			grid-area: turnos;
			justify-self: end;
		}
		.abrir {
			grid-area: abrir;
			min-height: 48px;
		}
		.detalle {
			padding: 4px 16px 16px;
		}
	}
	@media (max-width: 719px) {
		.ir,
		.pastilla,
		.chip,
		.dias,
		.limpiar {
			height: 48px;
			min-height: 48px;
		}
		.campo-buscar,
		.ir {
			height: 48px;
		}
	}
</style>
