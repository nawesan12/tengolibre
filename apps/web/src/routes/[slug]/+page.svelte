<script lang="ts">
	import { fecha, pesos } from '#lib/formato.ts';
	import { duracion, estrellas, linkExterno, linkInstagram, linkMapa, linkWhatsapp, rubro, textoCobro } from '#lib/publico/textos.ts';

	let { data } = $props();
	const n = $derived(data.negocio);
	const w = $derived(data.web);
	const reservar = $derived(`/${n.slug}/reservar`);

	// Servicios agrupados por categoría (en el orden del negocio); los sueltos, al final.
	const grupos = $derived.by(() => {
		const out = data.categorias
			.map((c) => ({ id: c.id, nombre: c.nombre, servicios: data.servicios.filter((s) => s.categoriaId === c.id) }))
			.filter((g) => g.servicios.length);
		const sueltos = data.servicios.filter((s) => !s.categoriaId || !data.categorias.some((c) => c.id === s.categoriaId));
		if (sueltos.length) out.push({ id: '', nombre: out.length ? 'Otros servicios' : 'Servicios', servicios: sueltos });
		return out;
	});

	const mapa = $derived(linkMapa(w.latitud, w.longitud, [n.direccion, n.ciudad].filter(Boolean).join(', ')));
	const contacto = $derived(
		[
			n.whatsapp && { nombre: 'WhatsApp', href: linkWhatsapp(n.whatsapp) },
			n.telefono && { nombre: n.telefono, href: `tel:${n.telefono.replace(/[^\d+]/g, '')}` },
			w.email && { nombre: w.email, href: `mailto:${w.email}` },
			n.instagram && { nombre: 'Instagram', href: linkInstagram(n.instagram) },
			w.facebook && { nombre: 'Facebook', href: linkExterno(w.facebook) },
			w.tiktok && { nombre: 'TikTok', href: linkExterno(w.tiktok) },
			w.sitioWeb && { nombre: 'Sitio web', href: linkExterno(w.sitioWeb) }
		].filter((x): x is { nombre: string; href: string } => !!x && !!x.href)
	);
	const descripcion = $derived(
		(n.descripcion || w.acerca || `Sacá turno online en ${n.nombre}${n.ciudad ? `, ${n.ciudad}` : ''}.`).slice(0, 160)
	);
	const cobroTexto = $derived(textoCobro(data.cobro, pesos));
</script>

<svelte:head>
	<title>{n.nombre} · Turnos online</title>
	<meta name="description" content={descripcion} />
	<meta property="og:title" content={n.nombre} />
	<meta property="og:description" content={descripcion} />
	{#if w.portada || n.logo}<meta property="og:image" content={w.portada ?? n.logo} />{/if}
</svelte:head>

<main class="pila">
	{#if w.portada}<img class="portada" src={w.portada} alt="" />{/if}
	<div class="contenedor pila">
		<header class="fila cabecera">
			{#if n.logo}<img class="logo" src={n.logo} alt={`Logo de ${n.nombre}`} width="72" height="72" />{/if}
			<div class="pila" style="gap: 4px">
				<h1>{n.nombre}</h1>
				<p class="suave chico" style="margin: 0">
					{[w.rubro ? rubro(w.rubro) : '', n.ciudad, w.provincia].filter(Boolean).join(' · ')}
				</p>
				{#if data.resenas.cantidad > 0}
					<p class="chico" style="margin: 0">
						<span aria-hidden="true">{estrellas(data.resenas.promedio)}</span>
						<a href="#resenas">{data.resenas.promedio.toLocaleString('es-AR')} de 5 ({data.resenas.cantidad}
							{data.resenas.cantidad === 1 ? 'reseña' : 'reseñas'})</a>
					</p>
				{/if}
			</div>
		</header>

		{#if n.descripcion}<p>{n.descripcion}</p>{/if}

		<div class="fila">
			<a class="boton primario grande" href={reservar}>Reservar turno</a>
			{#if mapa}<a class="boton" href={mapa} target="_blank" rel="noopener">Cómo llegar</a>{/if}
		</div>

		<section class="pila" aria-labelledby="t-servicios">
			<h2 id="t-servicios">Servicios</h2>
			{#if grupos.length === 0}<p class="suave">Este negocio todavía no cargó servicios.</p>{/if}
			{#each grupos as g (g.id)}
				<div class="pila">
					{#if grupos.length > 1}<h3>{g.nombre}</h3>{/if}
					<ul class="lista tarjeta">
						{#each g.servicios as s (s.id)}
							<li>
								<a class="servicio" href={`${reservar}?servicio=${s.id}`}>
									<span class="pila" style="gap: 2px">
										<strong>{s.nombre}</strong>
										{#if s.descripcion}<span class="suave chico">{s.descripcion}</span>{/if}
										<span class="suave chico">{duracion(s.duracion)}</span>
									</span>
									<span class="precio">{s.precio > 0 ? pesos(s.precio) : 'Sin cargo'}</span>
								</a>
							</li>
						{/each}
					</ul>
				</div>
			{/each}
		</section>

		{#if data.profesionales.length}
			<section class="pila" aria-labelledby="t-equipo">
				<h2 id="t-equipo">Equipo</h2>
				<ul class="grilla lista-suelta">
					{#each data.profesionales as p (p.id)}
						<li class="tarjeta fila" style="flex-wrap: nowrap; align-items: flex-start">
							{#if p.foto}
								<img class="logo" src={p.foto} alt="" width="48" height="48" loading="lazy" />
							{:else}
								<span class="inicial" aria-hidden="true">{p.nombre.slice(0, 1)}</span>
							{/if}
							<div>
								<strong>{p.nombre}</strong>
								{#if p.especialidad}<div class="suave chico">{p.especialidad}</div>{/if}
								{#if p.bio}<p class="chico" style="margin: 4px 0 0">{p.bio}</p>{/if}
							</div>
						</li>
					{/each}
				</ul>
			</section>
		{/if}

		{#if data.galeria.length}
			<section class="pila" aria-labelledby="t-galeria">
				<h2 id="t-galeria">Galería</h2>
				<ul class="galeria lista-suelta">
					{#each data.galeria as f (f.id)}
						<li>
							<figure>
								<img src={f.url} alt={f.epigrafe || `Foto de ${n.nombre}`} loading="lazy" />
								{#if f.epigrafe}<figcaption class="suave chico">{f.epigrafe}</figcaption>{/if}
							</figure>
						</li>
					{/each}
				</ul>
			</section>
		{/if}

		{#if data.planes.length}
			<section class="pila" aria-labelledby="t-planes">
				<h2 id="t-planes">Membresías</h2>
				<ul class="grilla lista-suelta">
					{#each data.planes as pl (pl.id)}
						<li class="tarjeta pila" style="gap: 6px">
							<strong>{pl.nombre}</strong>
							<span>{pesos(pl.precio)} · {pl.visitas} {pl.visitas === 1 ? 'visita' : 'visitas'} en {pl.duracionDias}&nbsp;días</span>
							{#if pl.descripcion}<span class="suave chico">{pl.descripcion}</span>{/if}
							{#if pl.beneficios.length}
								<ul class="chico" style="margin: 0; padding-left: 18px">
									{#each pl.beneficios as b (b)}<li>{b}</li>{/each}
								</ul>
							{/if}
						</li>
					{/each}
				</ul>
				<p class="suave chico">Para sumarte a una membresía, consultá en el&nbsp;local.</p>
			</section>
		{/if}

		{#if data.resenas.cantidad > 0}
			<section class="pila" id="resenas" aria-labelledby="t-resenas">
				<h2 id="t-resenas">Reseñas</h2>
				<ul class="pila lista-suelta">
					{#each data.listaResenas as r, i (i)}
						<li class="tarjeta pila" style="gap: 4px">
							<div class="fila" style="justify-content: space-between">
								<strong>{r.nombre}</strong>
								<span aria-label={`${r.puntaje} de 5`}>{estrellas(r.puntaje)}</span>
							</div>
							<span class="suave chico">{r.servicio} · {fecha(r.creadoEn, n.zonaHoraria)}</span>
							{#if r.comentario}<p style="margin: 0">{r.comentario}</p>{/if}
							{#if r.respuesta}<p class="chico respuesta">Respuesta de {n.nombre}: {r.respuesta}</p>{/if}
						</li>
					{/each}
				</ul>
			</section>
		{/if}

		<section class="pila" aria-labelledby="t-info">
			<h2 id="t-info">Información</h2>
			<div class="tarjeta pila">
				{#if w.acerca}<p style="margin: 0">{w.acerca}</p>{/if}
				{#if n.direccion}
					<p style="margin: 0">
						<strong>Dirección:</strong>
						{#if mapa}<a href={mapa} target="_blank" rel="noopener">{n.direccion}{n.ciudad ? `, ${n.ciudad}` : ''}</a>
						{:else}{n.direccion}{/if}
					</p>
				{/if}
				{#if contacto.length}
					<ul class="fila lista-suelta" aria-label="Contacto">
						{#each contacto as c (c.href)}<li><a class="boton" href={c.href} target="_blank" rel="noopener">{c.nombre}</a></li>{/each}
					</ul>
				{/if}
			</div>
			{#if data.politicaCancelacion || cobroTexto}
				<div class="tarjeta pila">
					<h3 style="margin: 0">Políticas</h3>
					{#if cobroTexto}<p style="margin: 0">{cobroTexto}</p>{/if}
					{#if data.politicaCancelacion}<p style="margin: 0; white-space: pre-line">{data.politicaCancelacion}</p>{/if}
				</div>
			{/if}
		</section>

		<footer class="suave chico">
			Turnos con <a href="/">Tengo Libre</a> · <a href="/explorar">Más negocios</a>
		</footer>
	</div>

	<div class="barra">
		<a class="boton primario grande" href={reservar}>Reservar turno</a>
	</div>
</main>

<style>
	.portada {
		width: 100%;
		max-height: 280px;
		object-fit: cover;
		display: block;
	}
	.cabecera {
		flex-wrap: nowrap;
		align-items: center;
		gap: 14px;
	}
	.cabecera h1 {
		margin: 0;
	}
	.logo {
		border-radius: 50%;
		object-fit: cover;
		flex: none;
	}
	.inicial {
		display: grid;
		place-items: center;
		width: 48px;
		height: 48px;
		border-radius: 50%;
		background: var(--marca-suave);
		font-weight: 600;
		flex: none;
	}
	.grande {
		min-height: 48px;
		padding: 0 20px;
	}
	.lista,
	.lista-suelta {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.lista {
		padding: 0;
	}
	.lista li + li {
		border-top: 1px solid var(--borde);
	}
	.servicio {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 12px;
		padding: 14px 16px;
		text-decoration: none;
		min-height: 56px;
	}
	.servicio:hover {
		background: var(--fondo);
	}
	.precio {
		font-weight: 600;
		white-space: nowrap;
	}
	.galeria {
		display: grid;
		gap: 8px;
		grid-template-columns: repeat(auto-fill, minmax(min(160px, 45%), 1fr));
	}
	.galeria figure {
		margin: 0;
	}
	.galeria img {
		width: 100%;
		aspect-ratio: 1;
		object-fit: cover;
		border-radius: var(--radio);
		display: block;
	}
	.respuesta {
		margin: 4px 0 0;
		padding: 8px 10px;
		border-radius: 8px;
		background: var(--fondo);
	}
	.barra {
		display: none;
	}
	@media (max-width: 640px) {
		.barra {
			display: block;
			position: sticky;
			bottom: 0;
			padding: 12px 16px calc(12px + env(safe-area-inset-bottom));
			background: var(--superficie);
			border-top: 1px solid var(--borde);
		}
		.barra .boton {
			width: 100%;
		}
	}
</style>
