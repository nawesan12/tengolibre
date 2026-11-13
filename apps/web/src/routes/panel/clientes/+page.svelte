<!--
	Clientes (handoff «Panel v2»): buscador, filtros por etiqueta y penalizadas, y la lista con cada
	cliente (iniciales, etiquetas, visitas y su próximo turno). Etiquetas y Penalidades arriba.
-->
<script lang="ts">
	import { page } from '$app/state';
	import { enhance } from '$app/forms';
	import Icono from '#lib/ui/Icono.svelte';
	import { fecha } from '#lib/formato.ts';
	import Chip from '#lib/operacion/Chip.svelte';
	import { ATENCION } from '#lib/operacion/secciones.ts';
	import { accion, colorEtiqueta, cuandoCorto } from '#lib/operacion/ui.ts';
	import { iniciales } from '#lib/panel/dia.ts';

	let { data, form } = $props();
	let alta = $state(false);
	const paginas = $derived(Math.max(1, Math.ceil(data.total / data.porPagina)));
	const puedeEditar = $derived(ATENCION.includes(data.negocio.rol));
	const gestiona = $derived(['duenio', 'encargado'].includes(data.negocio.rol));

	const enlace = (cambios: Record<string, string | number | boolean>) => {
		const v = { q: data.filtros.q, etiqueta: data.filtros.etiqueta, penalizados: data.filtros.penalizados, orden: data.filtros.orden, pagina: 1, ...cambios };
		const q = new URLSearchParams();
		if (v.q) q.set('q', String(v.q));
		if (v.etiqueta) q.set('etiqueta', String(v.etiqueta));
		if (v.penalizados) q.set('penalizados', '1');
		if (v.orden && v.orden !== 'nombre') q.set('orden', String(v.orden));
		if (Number(v.pagina) > 1) q.set('pagina', String(v.pagina));
		const s = q.toString();
		return s ? `?${s}` : '/panel/clientes';
	};
	const visitas = (c: (typeof data.clientes)[number]) =>
		c.visitas === 0
			? 'Todavía no vino'
			: c.visitas === 1
				? `Primera visita${c.ultimaVisita ? ` · ${fecha(c.ultimaVisita)}` : ''}`
				: `${c.visitas} visitas${c.penalizado && c.ausencias ? ` · ${c.ausencias} ausencias` : c.ultimaVisita ? ` · última ${fecha(c.ultimaVisita)}` : ''}`;
	const hoyEs = (iso: string | null) => !!iso && cuandoCorto(iso, false) === 'HOY';
</script>

<svelte:head><title>Clientes · Panel</title></svelte:head>

<div class="cabeza">
	<h1>Clientes</h1>
	<span class="total tl-num" aria-live="polite">{data.total} en total</span>
	<span class="crece"></span>
	<a href="/panel/etiquetas">Etiquetas</a>
	<a href="/panel/penalidades">Penalidades</a>
	{#if puedeEditar}<button class="nuevo" aria-expanded={alta} onclick={() => (alta = !alta)}><Icono nombre="person_add" tam={20} />Nuevo cliente</button>{/if}
</div>

{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}
{#if page.url.searchParams.get('suprimido') === 'borrado'}
	<p class="aviso" role="status">Listo: borramos el cliente y sus&nbsp;datos.</p>
{:else if page.url.searchParams.get('suprimido') === 'anonimizado'}
	<p class="aviso" role="status">Listo: borramos sus datos. Sus turnos pasados quedan como «Cliente&nbsp;borrado».</p>
{/if}

{#if alta}
	<form method="POST" action={accion(page.url, 'crear')} use:enhance class="alta" aria-label="Nuevo cliente">
		<label class="campo">Nombre <input name="nombre" required maxlength="80" /></label>
		<label class="campo">Teléfono <input name="telefono" type="tel" maxlength="30" /></label>
		<label class="campo">Email <input name="email" type="email" maxlength="120" /></label>
		<button class="boton oscuro">Agregar</button>
	</form>
{/if}

<form method="GET" role="search" class="buscar" data-sveltekit-reset="false">
	<label class="caja">
		<Icono nombre="search" tam={22} />
		<span class="visualmente-oculto">Buscar clientes</span>
		<input type="search" name="q" value={data.filtros.q} placeholder="Nombre, teléfono o email" />
	</label>
	{#if data.filtros.etiqueta}<input type="hidden" name="etiqueta" value={data.filtros.etiqueta} />{/if}
	{#if data.filtros.penalizados}<input type="hidden" name="penalizados" value="1" />{/if}
	<label class="orden">
		<span class="visualmente-oculto">Orden</span>
		<select name="orden" onchange={(e) => e.currentTarget.form?.requestSubmit()}>
			<option value="nombre" selected={data.filtros.orden === 'nombre'}>Por nombre</option>
			<option value="reciente" selected={data.filtros.orden === 'reciente'}>Más nuevos</option>
			<option value="ultima" selected={data.filtros.orden === 'ultima'}>Última visita</option>
			<option value="visitas" selected={data.filtros.orden === 'visitas'}>Más visitas</option>
			<option value="gastado" selected={data.filtros.orden === 'gastado'}>Más gastado</option>
		</select>
	</label>
	<button class="visualmente-oculto">Buscar</button>
</form>

<div class="chips" aria-label="Filtrar">
	<Chip href={enlace({ etiqueta: '', penalizados: false })} on={!data.filtros.etiqueta && !data.filtros.penalizados}>Todas</Chip>
	{#each data.etiquetas as e (e.id)}
		<Chip href={enlace({ etiqueta: e.id, penalizados: false })} on={data.filtros.etiqueta === e.id} cuenta={e.clientes} cuentaColor="currentColor">{e.nombre}</Chip>
	{/each}
	<Chip href={enlace({ penalizados: !data.filtros.penalizados, etiqueta: '' })} on={data.filtros.penalizados}>Penalizadas</Chip>
</div>

{#if data.clientes.length === 0}
	<div class="vacio">
		<Icono nombre="person_search" tam={28} />
		<b>No encontramos clientes con esos filtros</b>
		<a class="boton" href="/panel/clientes">Ver todos</a>
	</div>
{:else}
	<ul class="lista">
		{#each data.clientes as c (c.id)}
			<li>
				<a class="fila-c" href={`/panel/clientes/${c.id}`}>
					<span class="avatar">{iniciales(c.nombre)}</span>
					<span class="quien">
						<span class="nombre">
							<b>{c.nombre}</b>
							{#each c.etiquetas as e (e.id)}
								{@const col = colorEtiqueta(e.color)}
								<span class="tag" style:background={col.fondo} style:color={col.texto}>{e.nombre}</span>
							{/each}
							{#if c.visitas === 0 && c.proximoTurno && !c.penalizado}<span class="tag nueva">1ª vez</span>{/if}
							{#if c.penalizado}<span class="tag penal">Penalizada</span>{/if}
						</span>
						<span class="tel tl-num">{c.telefono || c.email || 'Sin contacto'}</span>
					</span>
					<span class="visitas">{visitas(c)}</span>
					<span class="prox tl-num" class:hoy={hoyEs(c.proximoTurno)}>{c.proximoTurno ? cuandoCorto(c.proximoTurno) : '—'}</span>
				</a>
			</li>
		{/each}
	</ul>
{/if}

{#if paginas > 1}
	<nav class="paginas" aria-label="Páginas">
		{#if data.pagina > 1}<a class="boton" href={enlace({ pagina: data.pagina - 1 })}><Icono nombre="chevron_left" tam={20} />Anterior</a>{/if}
		<span class="tl-num">Página {data.pagina} de {paginas}</span>
		{#if data.pagina < paginas}<a class="boton" href={enlace({ pagina: data.pagina + 1 })}>Siguiente<Icono nombre="chevron_right" tam={20} /></a>{/if}
	</nav>
{/if}

{#if data.recordatorios.length}
	<section class="recs" aria-labelledby="t-recs">
		<h2 id="t-recs">Recordatorios automáticos</h2>
		<div class="recs-g">
			{#each data.recordatorios as r (r.tipo)}
				<article class="rec">
					<div class="rec-cab">
						<Icono nombre={r.tipo === 'cumpleanios' ? 'cake' : 'history'} tam={24} />
						<h3>{r.titulo}</h3>
						{#if gestiona}
							<form method="POST" action={accion(page.url, 'recordatorio')} use:enhance>
								<input type="hidden" name="tipo" value={r.tipo} />
								<input type="hidden" name="activo" value={r.activo ? '0' : '1'} />
								<button class="interruptor" role="switch" aria-checked={r.activo} aria-label={`${r.titulo}: ${r.activo ? 'prendido' : 'apagado'}`}>
									<span></span>
								</button>
							</form>
						{:else}
							<span class="estado-r">{r.activo ? 'Prendido' : 'Apagado'}</span>
						{/if}
					</div>
					<p class="regla">{r.regla}</p>
					<div class="mail">
						<span class="tl-rotulo asunto">Asunto · {r.vistaPrevia.asunto}</span>
						<p>{r.texto}</p>
						{#if r.cupon}<span class="cupon tl-num">{r.cupon.codigo}</span>{/if}
					</div>
					<span class="tl-rotulo semana">{r.semanaTexto}</span>
				</article>
			{/each}
		</div>
	</section>
{/if}

<style>
	.recs {
		display: flex;
		flex-direction: column;
		gap: 12px;
		margin-top: 12px;
	}
	.recs h2 {
		margin: 0;
		font-size: 18px;
	}
	.recs-g {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(320px, 100%), 1fr));
		gap: 14px;
		padding: 20px;
		border-radius: 24px;
		background: var(--tl-crema);
		background-image: var(--tl-grano-osc);
	}
	.rec {
		display: flex;
		flex-direction: column;
		gap: 10px;
		padding: 18px;
		border-radius: 20px;
		background: var(--tl-papel);
	}
	.rec-cab {
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.rec-cab :global(.icono) {
		color: var(--tl-menta-texto);
	}
	.rec-cab h3 {
		flex: 1;
		margin: 0;
		font-size: 16px;
	}
	.interruptor {
		width: 52px;
		height: 32px;
		padding: 3px;
		border: 0;
		border-radius: 999px;
		background: #c9bda4;
		cursor: pointer;
		display: flex;
	}
	.interruptor span {
		width: 26px;
		height: 26px;
		border-radius: 50%;
		background: #fff;
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.interruptor[aria-checked='true'] {
		background: var(--tl-menta-texto);
	}
	.interruptor[aria-checked='true'] span {
		transform: translateX(20px);
	}
	.estado-r {
		font-size: 13px;
		font-weight: 700;
		color: var(--tl-sub);
	}
	.regla {
		margin: 0;
		font-size: 13px;
		color: var(--tl-sub-2);
	}
	.mail {
		padding: 12px 14px;
		border-radius: 14px;
		background: #f8f3ea;
		box-shadow: inset 0 0 0 1px var(--tl-linea);
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	.asunto {
		font-size: 11px;
		color: var(--tl-sub);
		letter-spacing: 0.04em;
	}
	.mail p {
		margin: 0;
		font-size: 14px;
		line-height: 1.5;
	}
	.cupon {
		align-self: flex-start;
		padding: 4px 10px;
		border-radius: 6px;
		background: repeating-linear-gradient(-45deg, rgba(60, 204, 144, 0.22) 0 2px, transparent 2px 9px), #f3fbf7;
		font-size: 13px;
		font-weight: 900;
		letter-spacing: 0.08em;
		color: var(--tl-menta-texto);
	}
	.semana {
		font-size: 11px;
		color: var(--tl-menta-texto);
	}
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
	.total {
		font-size: 13px;
		color: var(--tl-sub);
		padding-bottom: 4px;
	}
	.crece {
		flex: 1;
	}
	.cabeza > a {
		font-size: 14px;
		font-weight: 700;
		color: var(--tl-menta-texto);
		min-height: 36px;
		display: inline-flex;
		align-items: center;
	}
	.nuevo {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		height: 44px;
		padding: 0 18px;
		border: 0;
		border-radius: 999px;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		font: 700 14px var(--tl-font);
		cursor: pointer;
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.45), 0 12px 24px -12px rgba(60, 204, 144, 0.8);
	}
	.alta {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(200px, 100%), 1fr));
		gap: 12px;
		align-items: end;
		padding: 16px;
		border-radius: 20px;
		background: var(--tl-papel);
		box-shadow: 0 10px 20px -16px rgba(15, 35, 40, 0.45);
	}
	.buscar {
		display: flex;
		gap: 8px;
	}
	.caja {
		flex: 1;
		display: flex;
		align-items: center;
		gap: 10px;
		height: 52px;
		padding: 0 16px;
		border-radius: 16px;
		background: var(--tl-papel);
		box-shadow: inset 0 0 0 1px #e6dcc8;
		color: var(--tl-sub);
	}
	.caja:focus-within {
		box-shadow: inset 0 0 0 2px var(--tl-menta);
	}
	.caja input {
		flex: 1;
		min-width: 0;
		border: 0;
		outline: none;
		background: transparent;
		font: 400 16px var(--tl-font);
		color: var(--tl-tinta);
	}
	.orden select {
		height: 52px;
		padding: 0 14px;
		border: 0;
		border-radius: 16px;
		background: var(--tl-papel);
		box-shadow: inset 0 0 0 1px #e6dcc8;
		font: 700 13px var(--tl-font);
		color: var(--tl-tinta);
	}
	.chips {
		display: flex;
		gap: 6px;
		overflow-x: auto;
		scrollbar-width: none;
		margin-top: -6px;
		padding: 2px;
	}
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		border-radius: 20px;
		background: var(--tl-papel);
		overflow: hidden;
		box-shadow: 0 20px 40px -30px rgba(15, 35, 40, 0.5);
	}
	.lista li + li {
		border-top: 1px solid #f0e8d8;
	}
	.fila-c {
		display: grid;
		grid-template-columns: 42px minmax(0, 1.4fr) minmax(0, 1fr) 150px;
		align-items: center;
		gap: 14px;
		min-height: 68px;
		padding: 10px 18px;
		color: var(--tl-tinta);
		text-decoration: none;
		transition: background var(--tl-tocar) var(--tl-ease);
	}
	.fila-c:hover {
		background: #faf6ee;
	}
	.avatar {
		width: 42px;
		height: 42px;
		border-radius: 50%;
		background: var(--tl-crema);
		display: grid;
		place-items: center;
		font-size: 13px;
		font-weight: 700;
	}
	.quien {
		display: flex;
		flex-direction: column;
		gap: 3px;
		min-width: 0;
	}
	.nombre {
		display: flex;
		align-items: center;
		gap: 6px;
		flex-wrap: wrap;
	}
	.nombre b {
		font-size: 15px;
	}
	.tag {
		font-size: 11px;
		font-weight: 700;
		letter-spacing: 0.05em;
		text-transform: uppercase;
		padding: 2px 6px;
		border-radius: 5px;
	}
	.tag.nueva {
		background: var(--tl-menta-suave);
		color: var(--tl-menta-texto);
	}
	.tag.penal {
		background: #fbe9c2;
		color: #7a4a00;
	}
	.tel {
		font-size: 12px;
		color: var(--tl-sub);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.visitas {
		font-size: 13px;
		color: var(--tl-sub);
	}
	.prox {
		justify-self: end;
		font-size: 12px;
		font-weight: 700;
		color: var(--tl-sub);
	}
	.prox.hoy {
		color: var(--tl-menta-texto);
	}
	.vacio {
		padding: 40px 20px;
		border-radius: 20px;
		background: var(--tl-papel);
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 10px;
		text-align: center;
	}
	.vacio :global(.icono) {
		color: var(--tl-menta-texto);
	}
	.paginas {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 12px;
		font-size: 13px;
		color: var(--tl-sub);
	}
	@media (max-width: 859px) {
		h1 {
			font-size: 34px;
		}
		.cabeza > a {
			min-height: 44px;
		}
		.fila-c {
			grid-template-columns: 42px minmax(0, 1fr) auto;
			padding: 10px 14px;
		}
		.visitas {
			display: none;
		}
		.buscar {
			flex-wrap: wrap;
		}
		.caja {
			flex-basis: 100%;
		}
		.orden select {
			height: 44px;
		}
	}
</style>
