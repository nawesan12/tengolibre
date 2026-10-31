<script lang="ts">
	import { pesos } from '#lib/formato.ts';
	import { rubro } from '#lib/publico/textos.ts';

	let { data } = $props();
	const d = $derived(data.directorio);
	const f = $derived(data.filtros);

	function conPagina(p: number) {
		const q = new URLSearchParams();
		for (const [k, v] of Object.entries(f)) if (v) q.set(k, v);
		if (p > 1) q.set('pagina', String(p));
		const s = q.toString();
		return s ? `?${s}` : '?';
	}

	const titulo = $derived(
		[f.rubro ? rubro(f.rubro) : 'Negocios', f.ciudad ? `en ${f.ciudad}` : ''].filter(Boolean).join(' ')
	);
</script>

<svelte:head>
	<title>{titulo} · Tengo Libre</title>
	<meta name="description" content="Encontrá barberías, peluquerías y centros de estética y sacá turno online en el momento." />
</svelte:head>

<main class="contenedor pila">
	<header class="pila">
		<a href="/" class="suave chico">Tengo Libre</a>
		<h1>Encontrá dónde sacar&nbsp;turno</h1>
	</header>

	<form method="GET" class="tarjeta filtros" role="search">
		<label class="campo">
			Buscar
			<input name="q" type="search" defaultValue={f.q} placeholder="Nombre o servicio" maxlength="80" />
		</label>
		<label class="campo">
			Ciudad
			<input name="ciudad" defaultValue={f.ciudad} list="ciudades" maxlength="80" autocomplete="address-level2" />
			<datalist id="ciudades">
				{#each data.ciudades as c (c.ciudad)}<option value={c.ciudad}>{c.provincia}</option>{/each}
			</datalist>
		</label>
		<label class="campo">
			Rubro
			<select name="rubro">
				<option value="">Todos</option>
				{#each data.rubros as r (r)}<option value={r} selected={r === f.rubro}>{rubro(r)}</option>{/each}
			</select>
		</label>
		<button class="boton primario">Buscar</button>
	</form>

	{#if data.ciudades.length && !f.ciudad}
		<nav aria-label="Ciudades" class="fila chico">
			{#each data.ciudades.slice(0, 12) as c (c.ciudad)}
				<a class="etiqueta" href={`?ciudad=${encodeURIComponent(c.ciudad)}`}>{c.ciudad} ({c.cantidad})</a>
			{/each}
		</nav>
	{/if}

	<p class="suave" aria-live="polite">
		{d.total === 0 ? 'No encontramos negocios con esa búsqueda.' : d.total === 1 ? '1 negocio' : `${d.total} negocios`}
	</p>

	<ul class="grilla lista" aria-label="Resultados">
		{#each d.negocios as n (n.slug)}
			<li>
				<a class="tarjeta ficha" href={`/${n.slug}`}>
					{#if n.portada}<img src={n.portada} alt="" loading="lazy" class="portada" />{/if}
					<div class="fila" style="flex-wrap: nowrap">
						{#if n.logo}<img src={n.logo} alt="" class="logo" width="44" height="44" loading="lazy" />{/if}
						<div>
							<h2>{n.nombre}</h2>
							<p class="suave chico">{[n.rubro ? rubro(n.rubro) : '', n.ciudad].filter(Boolean).join(' · ')}</p>
						</div>
					</div>
					{#if n.servicios.length}<p class="chico">{n.servicios.join(', ')}</p>{/if}
					{#if n.desde != null}<p class="chico"><strong>Desde {pesos(n.desde)}</strong></p>{/if}
				</a>
			</li>
		{/each}
	</ul>

	{#if d.paginas > 1}
		<nav class="fila" aria-label="Páginas" style="justify-content: center">
			{#if data.pagina > 1}<a class="boton" href={conPagina(data.pagina - 1)} rel="prev">Anterior</a>{/if}
			<span class="suave chico">Página {data.pagina} de {d.paginas}</span>
			{#if data.pagina < d.paginas}<a class="boton" href={conPagina(data.pagina + 1)} rel="next">Siguiente</a>{/if}
		</nav>
	{/if}
</main>

<style>
	.filtros {
		display: grid;
		gap: 12px;
		grid-template-columns: repeat(auto-fit, minmax(min(200px, 100%), 1fr));
		align-items: end;
	}
	.lista {
		list-style: none;
		padding: 0;
		margin: 0;
	}
	.ficha {
		display: flex;
		flex-direction: column;
		gap: 8px;
		height: 100%;
		text-decoration: none;
	}
	.ficha h2 {
		margin: 0;
		font-size: 17px;
	}
	.ficha p {
		margin: 0;
	}
	.portada {
		width: calc(100% + 32px);
		margin: -16px -16px 0;
		aspect-ratio: 16 / 7;
		object-fit: cover;
		border-radius: var(--radio) var(--radio) 0 0;
	}
	.logo {
		border-radius: 50%;
		object-fit: cover;
		flex: none;
	}
</style>
