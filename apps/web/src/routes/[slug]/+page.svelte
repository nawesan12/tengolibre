<script lang="ts">
	import PaginaNegocio from '#lib/publico/PaginaNegocio.svelte';

	let { data } = $props();
	const n = $derived(data.negocio);
	const w = $derived(data.web);
	const descripcion = $derived(
		(n.descripcion || w.acerca || `Sacá turno online en ${n.nombre}${n.ciudad ? `, ${n.ciudad}` : ''}.`).slice(0, 160)
	);
</script>

<svelte:head>
	<title>{n.nombre} · Turnos online</title>
	<meta name="description" content={descripcion} />
	<link rel="canonical" href={data.canonical} />
	<meta property="og:type" content="website" />
	<meta property="og:url" content={data.canonical} />
	<meta property="og:title" content={n.nombre} />
	<meta property="og:description" content={descripcion} />
	<!-- Sin portada ni logo, la imagen de marca de Tengo Libre (nunca un link sin imagen). -->
	<meta property="og:image" content={w.portada || n.logo || data.ogImagen} />
	<meta name="twitter:card" content={n.logo && !w.portada ? 'summary' : 'summary_large_image'} />
	<meta name="theme-color" content={w.colorPrimario} />
	{#if data.vista}<meta name="robots" content="noindex" />{/if}
</svelte:head>

<main>
	<PaginaNegocio
		pagina={data}
		resenas={data.listaResenas}
		planes={data.planes}
		cobro={data.cobro}
		proximo={data.proximo}
		plantilla={data.vista?.plantilla}
		portada={data.vista?.portada}
		fondo={data.vista?.fondo}
		titulos={data.vista?.titulos}
		forma={data.vista?.forma}
		secciones={data.vista?.secciones}
	/>
</main>
