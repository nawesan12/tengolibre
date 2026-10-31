<script lang="ts">
	import { pesos } from '#lib/formato.ts';
	import { rubro } from '#lib/publico/textos.ts';
	import Icono from '#lib/ui/Icono.svelte';
	import Logo from '#lib/ui/Logo.svelte';
	import { etiquetaLibre, iniciales } from '#lib/cuenta/presentacion.ts';

	let { data } = $props();
	const d = $derived(data.directorio);
	const f = $derived(data.filtros);
	let formulario: HTMLFormElement;

	function con(cambios: Record<string, string>) {
		const q = new URLSearchParams();
		for (const [k, v] of Object.entries({ ...f, ...cambios })) if (v) q.set(k, v);
		const s = q.toString();
		return s ? `?${s}` : '?';
	}
	const conPagina = (p: number) => con({ pagina: p > 1 ? String(p) : '' });

	const titulo = $derived(
		[f.rubro ? rubro(f.rubro) : 'Negocios', f.ciudad ? `en ${f.ciudad}` : ''].filter(Boolean).join(' ')
	);
	const puntaje = (n: number) => n.toLocaleString('es-AR', { minimumFractionDigits: 1, maximumFractionDigits: 1 });
	const enviar = () => formulario.requestSubmit();
</script>

<svelte:head>
	<title>{titulo} · Tengo Libre</title>
	<meta name="description" content="Encontrá barberías, peluquerías y centros de estética y sacá turno online en el momento." />
</svelte:head>

<div class="explorar">
	<main class="ancho">
		<header class="cabeza">
			<a href="/" class="inicio" aria-label="Tengo Libre, inicio"><Logo alto={24} /></a>
			<h1>Buscá dónde sacar&nbsp;turno</h1>
		</header>

		<form method="GET" class="filtros" role="search" bind:this={formulario}>
			<div class="fila-buscar">
				<label class="buscar">
					<Icono nombre="search" tam={22} />
					<input aria-label="Buscar" name="q" type="search" value={f.q} placeholder="Nombre o servicio" maxlength="80" />
				</label>
				<button class="ir">Buscar</button>
			</div>
			<div class="chips">
				<label class="chip campo-chip" class:activo={!!f.ciudad}>
					<Icono nombre="location_on" tam={18} lleno={!!f.ciudad} />
					<input aria-label="Ciudad" name="ciudad" value={f.ciudad} list="ciudades" maxlength="80" autocomplete="address-level2" placeholder="Ciudad" />
				</label>
				<datalist id="ciudades">
					{#each data.ciudades as c (c.ciudad)}<option value={c.ciudad}>{c.provincia}</option>{/each}
				</datalist>
				<label class="chip campo-chip" class:activo={!!f.rubro}>
					<Icono nombre="storefront" tam={18} lleno={!!f.rubro} />
					<select aria-label="Rubro" name="rubro" onchange={enviar}>
						<option value="">Todos los rubros</option>
						{#each data.rubros as r (r)}<option value={r} selected={r === f.rubro}>{rubro(r)}</option>{/each}
					</select>
				</label>
				<label class="chip" class:activo={!!f.conLugarHoy}>
					<input class="oculto" type="checkbox" name="conLugarHoy" value="1" checked={!!f.conLugarHoy} onchange={enviar} />
					<Icono nombre={f.conLugarHoy ? 'check' : 'event_available'} tam={18} />
					Hoy con lugar
				</label>
			</div>
		</form>

		{#if data.ciudades.length && !f.ciudad}
			<nav aria-label="Ciudades" class="ciudades">
				{#each data.ciudades.slice(0, 12) as c (c.ciudad)}
					<a class="chip" href={con({ ciudad: c.ciudad, pagina: '' })}>{c.ciudad} <span class="tl-num cuenta">{c.cantidad}</span></a>
				{/each}
			</nav>
		{/if}

		<p class="total" aria-live="polite">
			{d.total === 0 ? 'No encontramos negocios con esa búsqueda.' : d.total === 1 ? '1 negocio' : `${d.total} negocios`}
		</p>
		{#if d.total === 0 && (f.q || f.ciudad || f.rubro || f.conLugarHoy)}
			<p class="limpiar"><a href="/explorar">Ver todos los negocios</a></p>
		{/if}

		<ul class="resultados" aria-label="Resultados">
			{#each d.negocios as n, i (n.slug)}
				<li style:animation-delay="{Math.min(i, 8) * 40}ms">
					<a class="ficha" href={`/${n.slug}`}>
						<span class="foto">
							{#if n.portada}
								<img src={n.portada} alt="" loading="lazy" />
							{:else if n.logo}
								<img class="logo" src={n.logo} alt="" loading="lazy" width="56" height="56" />
							{:else}
								<span class="inicial" aria-hidden="true">{iniciales(n.nombre, 1)}</span>
							{/if}
						</span>
						<span class="datos">
							<b class="nombre">{n.nombre}</b>
							<span class="sub">{[n.rubro ? rubro(n.rubro) : '', n.ciudad].filter(Boolean).join(' · ')}</span>
							{#if n.promedio || n.desde != null}
								<span class="precio">
									{#if n.promedio}<b class="tl-num" aria-label={`${puntaje(n.promedio)} de 5`}>{puntaje(n.promedio)} ★</b>{/if}
									{#if n.desde != null}<span>desde <span class="tl-num">{pesos(n.desde)}</span></span>{/if}
								</span>
							{/if}
							{#if n.proximoLibre}
								<span class="libre tl-num">LIBRE {etiquetaLibre(n.proximoLibre, n.zonaHoraria || undefined)}</span>
							{:else if n.servicios.length}
								<span class="servicios">{n.servicios.join(', ')}</span>
							{/if}
						</span>
					</a>
				</li>
			{/each}
		</ul>

		{#if d.paginas > 1}
			<nav class="paginas" aria-label="Páginas">
				{#if data.pagina > 1}<a class="chip" href={conPagina(data.pagina - 1)} rel="prev"><Icono nombre="arrow_back" tam={18} />Anterior</a>{/if}
				<span class="tl-num">Página {data.pagina} de {d.paginas}</span>
				{#if data.pagina < d.paginas}<a class="chip" href={conPagina(data.pagina + 1)} rel="next">Siguiente<Icono nombre="arrow_forward" tam={18} /></a>{/if}
			</nav>
		{/if}
	</main>
</div>

<style>
	.explorar {
		min-height: 100dvh;
		background: var(--tl-crema);
	}
	.ancho {
		width: min(1100px, 100%);
		margin: 0 auto;
		padding: 24px 16px 48px;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.cabeza {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	.inicio {
		display: inline-flex;
		align-self: flex-start;
		min-height: 48px;
		align-items: center;
	}
	h1 {
		margin: 0;
		font-size: clamp(30px, 5vw, 48px);
		line-height: 1;
		font-weight: 700;
		letter-spacing: -0.045em;
	}
	.filtros {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.fila-buscar {
		display: flex;
		gap: 8px;
	}
	.buscar {
		flex: 1;
		min-width: 0;
		display: flex;
		align-items: center;
		gap: 8px;
		height: 52px;
		padding: 0 14px;
		border-radius: 16px;
		background: #fff;
		box-shadow: inset 0 0 0 1.5px #e0d5bf;
		color: var(--tl-sub);
	}
	.buscar:focus-within {
		box-shadow: inset 0 0 0 2px var(--tl-petroleo), 0 0 0 3px rgba(60, 204, 144, 0.35);
	}
	.buscar input {
		flex: 1;
		min-width: 0;
		height: 100%;
		border: 0;
		background: none;
		font: 400 16px var(--tl-font);
		color: var(--tl-tinta);
		outline: none;
	}
	.ir {
		flex: none;
		height: 52px;
		padding: 0 20px;
		border: 0;
		border-radius: 999px;
		background: var(--tl-petroleo);
		color: #fff;
		font: 700 15px var(--tl-font);
		cursor: pointer;
		box-shadow: var(--tl-luz-arriba-osc);
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.ir:active {
		transform: scale(0.97);
	}
	.chips,
	.ciudades {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
	}
	.chip {
		flex: none;
		display: inline-flex;
		align-items: center;
		gap: 4px;
		height: 40px;
		padding: 0 14px;
		border-radius: 999px;
		box-shadow: inset 0 0 0 1px #d8ccb4;
		font-size: 13px;
		font-weight: 700;
		color: var(--tl-tinta);
		text-decoration: none;
		cursor: pointer;
		transition: background var(--tl-tocar) var(--tl-ease);
	}
	.chip:hover {
		background: rgba(255, 255, 255, 0.6);
	}
	.chip.activo {
		background: var(--tl-petroleo);
		color: #fff;
		box-shadow: none;
	}
	.chip:focus-within {
		outline: 3px solid var(--tl-menta);
		outline-offset: 2px;
	}
	.campo-chip input,
	.campo-chip select {
		border: 0;
		background: none;
		color: inherit;
		font: 700 13px var(--tl-font);
		outline: none;
		min-width: 0;
		height: 100%;
	}
	.campo-chip input {
		width: 13ch;
	}
	.campo-chip input::placeholder {
		color: var(--tl-sub);
	}
	.campo-chip select {
		cursor: pointer;
		max-width: 22ch;
	}
	.campo-chip select option {
		color: var(--tl-tinta);
	}
	.oculto {
		position: absolute;
		opacity: 0;
		width: 1px;
		height: 1px;
		pointer-events: none;
	}
	.cuenta {
		color: var(--tl-sub);
		font-weight: 400;
	}
	.total {
		margin: 4px 0 0;
		font-size: 14px;
		color: var(--tl-sub);
	}
	.limpiar {
		margin: 0;
		font-size: 14px;
	}
	.limpiar a {
		color: var(--tl-menta-texto);
		font-weight: 700;
	}
	.resultados {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		gap: 10px;
		grid-template-columns: repeat(auto-fill, minmax(min(340px, 100%), 1fr));
	}
	.resultados li {
		animation: tl-gin var(--tl-entrar) var(--tl-ease) both;
	}
	.ficha {
		display: grid;
		grid-template-columns: 92px minmax(0, 1fr);
		height: 100%;
		min-height: 112px;
		border-radius: 18px;
		overflow: hidden;
		background: var(--tl-papel);
		color: inherit;
		text-decoration: none;
		box-shadow: var(--tl-sh-tarjeta);
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.ficha:hover {
		transform: translateY(-2px);
	}
	.foto {
		position: relative;
		display: grid;
		place-items: center;
		background-color: var(--tl-petroleo);
		background-image: var(--tl-grano);
	}
	.foto img {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		object-fit: cover;
		opacity: 0.9;
	}
	.foto img.logo {
		position: static;
		width: 56px;
		height: 56px;
		border-radius: 14px;
		opacity: 1;
		background: #fff;
	}
	.inicial {
		font-size: 32px;
		font-weight: 900;
		color: var(--tl-menta);
	}
	.datos {
		display: flex;
		flex-direction: column;
		gap: 4px;
		padding: 12px 14px;
		min-width: 0;
	}
	.nombre {
		font-size: 16px;
		overflow-wrap: anywhere;
	}
	.sub,
	.servicios {
		font-size: 13px;
		color: var(--tl-sub);
	}
	.servicios {
		display: -webkit-box;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
	.precio {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 13px;
	}
	.precio > span {
		color: var(--tl-sub);
	}
	.libre {
		align-self: flex-start;
		margin-top: 2px;
		display: inline-flex;
		align-items: center;
		height: 28px;
		padding: 0 10px;
		border-radius: 8px;
		background: repeating-linear-gradient(-45deg, rgba(60, 204, 144, 0.22) 0 2px, transparent 2px 9px), #f3fbf7;
		font-size: 11px;
		font-weight: 700;
		color: var(--tl-menta-texto);
	}
	.paginas {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 12px;
		margin-top: 12px;
		font-size: 13px;
		color: var(--tl-sub);
	}
	@media (max-width: 719px) {
		/* En el teléfono, filtros y ciudades en una tira que se desliza (como el diseño). */
		.chips,
		.ciudades {
			flex-wrap: nowrap;
			overflow-x: auto;
			margin: 0 -16px;
			padding: 2px 16px 4px;
			scrollbar-width: none;
		}
		.chips::-webkit-scrollbar,
		.ciudades::-webkit-scrollbar {
			display: none;
		}
		.chip,
		.paginas .chip {
			height: 48px;
		}
	}
	@media (min-width: 720px) {
		.ancho {
			padding-top: 32px;
			gap: 16px;
		}
	}
</style>
