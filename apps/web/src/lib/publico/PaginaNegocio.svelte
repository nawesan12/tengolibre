<!--
	PaginaNegocio: la web pública de un negocio (handoff v2, «Reserva v2»). La usa /[slug] y
	/[slug]/reservar, y el panel la incrusta en Plantillas para la vista previa en vivo.

	Datos que recibe:
	- `pagina`: lo que devuelve GET /v1/p/{slug} (negocio, categorías, servicios, profesionales,
	  web con estilo y colores, galería, resumen de reseñas, política de cancelación).
	- `resenas`: la lista de GET /v1/p/{slug}/resenas (opcional; sin ella sólo va el promedio).
	- `planes`: los planes de membresía (GET /v1/p/{slug}/membresias), opcional.
	- `cobro`: la configuración de seña (GET /v1/p/{slug}/cobro), opcional.
	- `proximo`: el primer lugar libre { fecha: AAAA-MM-DD, hora: HH:MM } (barra del teléfono), opcional.

	Props del componente (README del handoff): `plantilla`, `portada`, `fondo`, `titulos`, `forma`,
	`primario`, `acento` y `secciones` (las que se ven y en qué orden: servicios, resenas, equipo,
	fotos, membresias) (por defecto, lo publicado en `pagina.web`), y para incrustarlo:
	`ancho` (fuerza el ancho de diseño en px: menos de 900 es el teléfono) e `incrustado` (la barra
	de abajo pasa de fixed a sticky).

	`vista`: 'pagina' (la página; en el teléfono, sin el panel de reserva) o 'reservar' (en el
	teléfono, sólo el panel). `reserva`: el panel de reserva a mostrar a la derecha (en escritorio) o
	como pantalla (en el teléfono). Sin él va el paso 1 (elegir servicio) con links a la reserva.
	`ruta`: base de los links ('/{slug}' por defecto).
-->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import { pesos } from '#lib/formato.ts';
	import Icono from '#lib/ui/Icono.svelte';
	import PanelReserva from './PanelReserva.svelte';
	import { diaCorto } from './calendario.ts';
	import {
		completar,
		secciones as seccionesDe,
		tema,
		variables,
		type Fondo,
		type Forma,
		type Plantilla,
		type Portada,
		type Seccion,
		type Titulos
	} from './estilo.ts';
	import {
		compartirWhatsapp,
		duracion,
		estadoAtencion,
		iniciales,
		linkExterno,
		linkInstagram,
		linkMapa,
		linkWhatsapp,
		renglonesAtencion,
		rubro,
		sinViuda,
		textoCobro
	} from './textos.ts';
	import type { Cobro, Pagina, Plan, ResenaPublica } from './tipos.ts';

	let {
		pagina,
		resenas = [],
		planes = [],
		cobro = null,
		proximo = null,
		plantilla,
		portada,
		fondo,
		titulos,
		forma,
		secciones,
		primario,
		acento,
		ancho,
		incrustado = false,
		vista = 'pagina',
		ruta,
		elegido = '',
		reserva
	}: {
		pagina: Pagina;
		resenas?: ResenaPublica[];
		planes?: Plan[];
		cobro?: Cobro | null;
		proximo?: { fecha: string; hora: string } | null;
		plantilla?: Plantilla;
		portada?: Portada;
		fondo?: Fondo;
		titulos?: Titulos;
		forma?: Forma;
		secciones?: Seccion[];
		primario?: string;
		acento?: string;
		ancho?: number;
		incrustado?: boolean;
		vista?: 'pagina' | 'reservar';
		ruta?: string;
		/** El servicio elegido en la reserva (la carta lo marca «Elegido»). */
		elegido?: string;
		reserva?: Snippet;
	} = $props();

	const n = $derived(pagina.negocio);
	const w = $derived(pagina.web);
	const base = $derived(ruta ?? `/${n.slug}`);
	const reservar = $derived(`${base}/reservar`);
	const estilo = $derived.by(() => {
		const e = completar(w.estilo);
		return completar({
			plantilla: plantilla ?? e.plantilla,
			portada: portada ?? e.portada,
			fondo: fondo ?? e.fondo,
			titulos: titulos ?? e.titulos,
			forma: forma ?? e.forma
		});
	});
	const t = $derived(tema(estilo, primario ?? w.colorPrimario, acento ?? w.colorAcento));
	/** Las secciones del medio de la página, en el orden elegido (sin elegir: todas). */
	const orden = $derived(seccionesDe(secciones ?? w.estilo?.secciones));
	const vars = $derived(variables(t));
	const pl = $derived(estilo.plantilla);
	// Con `ancho`, el teléfono o el escritorio lo decide la prop (la barra vive fuera del contenedor).
	const forzado = $derived(ancho == null ? null : ancho < 900 ? 'compacto' : 'amplio');

	const grupos = $derived.by(() => {
		const out = pagina.categorias
			.map((c) => ({ id: c.id, nombre: c.nombre, servicios: pagina.servicios.filter((s) => s.categoriaId === c.id) }))
			.filter((g) => g.servicios.length);
		const sueltos = pagina.servicios.filter((s) => !s.categoriaId || !pagina.categorias.some((c) => c.id === s.categoriaId));
		if (sueltos.length) out.push({ id: '', nombre: out.length ? 'Otros servicios' : '', servicios: sueltos });
		return out;
	});

	const fotos = $derived([w.portada, ...pagina.galeria.map((f) => f.url)].filter((x): x is string => !!x));
	const galeria = $derived(pagina.galeria.slice(0, 3));
	const lugar = $derived([n.ciudad, w.provincia].filter(Boolean).join(', '));
	const conPunto = $derived(/[.!?]$/.test(n.nombre) ? n.nombre : `${n.nombre}.`);
	const meta = $derived([w.rubro ? rubro(w.rubro) : '', lugar].filter(Boolean).join(' · '));
	const promedio = $derived(pagina.resenas.promedio.toLocaleString('es-AR', { minimumFractionDigits: 1, maximumFractionDigits: 1 }));
	const hayResenas = $derived(pagina.resenas.cantidad > 0);
	const nResenas = $derived(`${pagina.resenas.cantidad} ${pagina.resenas.cantidad === 1 ? 'reseña' : 'reseñas'}`);
	const mapa = $derived(linkMapa(w.latitud, w.longitud, [n.direccion, n.ciudad].filter(Boolean).join(', ')));
	const atencion = $derived(pagina.horarioAtencion?.length ? pagina.horarioAtencion : null);
	const abierto = $derived(atencion ? estadoAtencion(atencion, new Date(), n.zonaHoraria) : null);
	const acerca = $derived(w.acerca || n.descripcion);
	const cobroTexto = $derived(textoCobro(cobro, pesos));
	const destacada = $derived(resenas.find((r) => r.comentario && r.puntaje >= 4) ?? null);
	const otras = $derived(resenas.filter((r) => r !== destacada && r.comentario).slice(0, 3));
	const corto = (nombre: string) => nombre.split(/\s+/)[0] ?? nombre;
	const estrellas = (x: number) => '★'.repeat(Math.max(0, Math.min(5, Math.round(x))));

	type Chip = { nombre: string; href: string; logo?: string; icono?: string };
	const chips = $derived(
		(<(Chip | '' | false)[]>[
			n.whatsapp && { nombre: 'WhatsApp', href: linkWhatsapp(n.whatsapp), logo: '/logos/whatsapp.svg' },
			mapa && { nombre: 'Cómo llegar', href: mapa, icono: 'directions' },
			n.instagram && { nombre: 'Instagram', href: linkInstagram(n.instagram), icono: 'photo_camera' },
			n.telefono && { nombre: 'Llamar', href: `tel:${n.telefono.replace(/[^\d+]/g, '')}`, icono: 'call' },
			w.email && { nombre: 'Email', href: `mailto:${w.email}`, icono: 'mail' },
			w.facebook && { nombre: 'Facebook', href: linkExterno(w.facebook), icono: 'thumb_up' },
			w.tiktok && { nombre: 'TikTok', href: linkExterno(w.tiktok), icono: 'music_note' },
			w.sitioWeb && { nombre: 'Sitio web', href: linkExterno(w.sitioWeb), icono: 'language' }
		]).filter((x): x is Chip => !!x && !!x.href)
	);
	const textoCompartir = $derived(`Sacá turno en ${n.nombre}`);
	async function compartir(e: MouseEvent) {
		if (typeof navigator === 'undefined' || !navigator.share) return;
		e.preventDefault();
		try {
			await navigator.share({ title: n.nombre, text: textoCompartir, url: location.origin + base });
		} catch {
			/* la persona cerró el diálogo */
		}
	}

	const hace = (iso: string) => {
		const d = Math.max(0, Math.round((Date.now() - Date.parse(iso)) / 86_400_000));
		if (d < 1) return 'hoy';
		if (d < 7) return `hace ${d} ${d === 1 ? 'día' : 'días'}`;
		if (d < 31) return `hace ${Math.round(d / 7)} sem.`;
		if (d < 365) return `hace ${Math.round(d / 30)} ${Math.round(d / 30) === 1 ? 'mes' : 'meses'}`;
		return `hace ${Math.round(d / 365)} ${Math.round(d / 365) === 1 ? 'año' : 'años'}`;
	};
	const porPlan = (p: Plan) => (p.duracionDias === 30 || p.duracionDias === 31 ? 'por mes' : `cada ${p.duracionDias} días`);
	const textoPlan = (p: Plan) =>
		p.descripcion ||
		[`${p.visitas} ${p.visitas === 1 ? 'visita' : 'visitas'}`, ...p.beneficios].join(', ') + '.';
</script>

{#snippet logo(tam: number, invertido = false)}
	{#if n.logo}
		<img class="logo" src={n.logo} alt="" width={tam} height={tam} style:width="{tam}px" style:height="{tam}px" />
	{:else}
		<span class="logo inicial" class:invertido style:width="{tam}px" style:height="{tam}px" style:font-size="{Math.round(tam * 0.4)}px" aria-hidden="true"
			>{n.nombre.trim().slice(0, 1).toUpperCase()}</span
		>
	{/if}
{/snippet}

{#snippet foto(i: number, clase = '', alt = '')}
	{#if fotos[i % Math.max(1, fotos.length)]}
		<img class={clase} src={fotos[i % fotos.length]} {alt} loading={i ? 'lazy' : undefined} />
	{:else}
		<span class="{clase} sin-foto" aria-hidden="true"></span>
	{/if}
{/snippet}

{#snippet sServicios()}
	<section class="carta-servicios lado" aria-labelledby="t-servicios">
		<div class="titulo-fila">
			<h2 id="t-servicios">Servicios</h2>
			<span class="sub chico-num">{pagina.servicios.length} · precios finales</span>
		</div>
		{#if grupos.length === 0}<p class="sub">Este negocio todavía no cargó&nbsp;servicios.</p>{/if}
		{#each grupos as g (g.id)}
			{#if g.nombre}<h3 class="categoria">{g.nombre}</h3>{/if}
			{#each g.servicios as s (s.id)}
				<a class="servicio" href={`${reservar}?servicio=${s.id}`} aria-current={elegido === s.id ? 'true' : undefined}>
					<span class="srv-fila">
						<span class="srv-nombre">{s.nombre}</span>
						<span class="puntos" aria-hidden="true"></span>
						<b class="tl-num">{s.precio > 0 ? pesos(s.precio) : 'Sin cargo'}</b>
					</span>
					<span class="srv-sub">
						<span class="tl-num">{duracion(s.duracion)}{s.descripcion ? ` · ${s.descripcion}` : ''}</span>
						<span class="srv-cta">{elegido === s.id ? 'Elegido' : 'Reservar'}</span>
					</span>
				</a>
			{/each}
		{/each}
	</section>
{/snippet}

{#snippet sMembresias()}
	{#each planes as p (p.id)}
		<div class="plan lado-m">
			<div class="plan-texto">
				<span class="plan-rotulo"><Icono nombre="card_membership" tam={16} lleno />MEMBRESÍA</span>
				<b class="plan-nombre">{p.nombre}</b>
				<span class="plan-desc">{sinViuda(textoPlan(p))}</span>
			</div>
			<div class="plan-precio">
				<b class="tl-num">{pesos(p.precio)}</b>
				<span>{porPlan(p)}</span>
			</div>
		</div>
	{/each}
{/snippet}

{#snippet sResenas()}
	{#if hayResenas}
		<section id="resenas" class="resenas lado" aria-labelledby="t-res">
			<div class="titulo-fila">
				<h2 id="t-res">Reseñas</h2>
				<b class="tl-num prom">{promedio}</b><span class="sub chico">de {pagina.resenas.cantidad}</span>
			</div>
			{#if destacada}
				<figure>
					<blockquote>«{sinViuda(destacada.comentario)}»</blockquote>
					<figcaption>
						<span class="estrellas" aria-label={`${destacada.puntaje} de 5`}>{estrellas(destacada.puntaje)}</span>{destacada.nombre}{destacada.servicio
							? ` · ${destacada.servicio}`
							: ''}
					</figcaption>
				</figure>
				{#if destacada.respuesta}<span class="respuesta"><b>{n.nombre} respondió:</b> {destacada.respuesta}</span>{/if}
			{/if}
			{#each otras as r, i (i)}
				<div class="resena">
					<span class="resena-cab">
						<b>{r.nombre}</b><span class="estrellas" aria-label={`${r.puntaje} de 5`}>{estrellas(r.puntaje)}</span>
						<span class="cuando tl-num">{hace(r.creadoEn)}</span>
					</span>
					{r.comentario}
					{#if r.respuesta}<span class="respuesta"><b>{n.nombre} respondió:</b> {r.respuesta}</span>{/if}
				</div>
			{/each}
		</section>
	{/if}
{/snippet}

{#snippet sEquipo()}
	{#if pagina.profesionales.length}
		<section class="equipo lado" aria-labelledby="t-eq">
			<h2 id="t-eq">Equipo</h2>
			<ul>
				{#each pagina.profesionales as p (p.id)}
					<li>
						{#if p.foto}<img src={p.foto} alt="" width="72" height="72" loading="lazy" />{:else}<span class="ini" aria-hidden="true">{iniciales(p.nombre)}</span>{/if}
						<b>{corto(p.nombre)}</b>
						{#if p.especialidad}<span>{p.especialidad}</span>{/if}
					</li>
				{/each}
			</ul>
		</section>
	{/if}
{/snippet}

{#snippet sFotos()}
	{#if galeria.length}
		<div class="fotos lado-m" style:--cols={galeria.length === 1 ? '1fr' : galeria.length === 2 ? '2fr 1fr' : '2fr 1fr 1fr'}>
			{#each galeria as f (f.id)}<img src={f.url} alt={f.epigrafe || `Foto de ${n.nombre}`} loading="lazy" />{/each}
		</div>
	{/if}
{/snippet}

<div class="web-tema" class:incrustado style={vars} style:width={ancho ? `${ancho}px` : undefined}>
	<div class="web">
		<div class="lienzo {forzado ?? ''}" class:reservando={vista === 'reservar'}>
			<div class="contenido">
				<!-- Portada según la plantilla -->
				{#if pl === 'vidriera'}
					<header class="vidriera">
						{#if fotos[0]}<img class="foto-op" src={fotos[0]} alt={`El local de ${n.nombre}`} />{/if}
						<div class="sombra-abajo"></div>
						<div class="vid-texto">
							<span class="vid-logo">{@render logo(64)}</span>
							<h1>{n.nombre}</h1>
							<span class="vid-meta">{[w.rubro ? rubro(w.rubro) : '', n.ciudad, hayResenas ? `★ ${promedio}` : ''].filter(Boolean).join(' · ')}</span>
							<a class="vid-cta" href={reservar}>Reservar turno</a>
						</div>
					</header>
				{:else if pl === 'sobria'}
					<header class="sobria">
						<span class="aro" aria-hidden="true"></span>
						{@render logo(72, true)}
						<h1>{n.nombre}</h1>
						{#if n.descripcion || lugar}<span class="sobria-desc">{sinViuda([n.descripcion, lugar].filter(Boolean).join(' '))}</span>{/if}
					</header>
				{:else if pl === 'galeria'}
					<header class="galeria-cab">
						<div class="gal-fotos">
							{@render foto(0, 'g1', `El local de ${n.nombre}`)}
							{@render foto(1, '')}
							{@render foto(2, '')}
						</div>
						<div class="gal-titulo lado">
							{@render logo(52)}
							<span class="pila0">
								<h1>{n.nombre}</h1>
								{#if meta}<span class="sub">{meta}</span>{/if}
							</span>
						</div>
					</header>
				{:else if pl === 'editorial'}
					<header class="editorial">
						<div class="pila14">
							{#if meta}<span class="ed-meta">{meta}</span>{/if}
							<h1>{conPunto}</h1>
							{#if n.descripcion}<span class="ed-desc">{sinViuda(n.descripcion)}</span>{/if}
						</div>
						{@render foto(0, 'ed-foto', `El local de ${n.nombre}`)}
					</header>
				{:else if pl === 'bloques'}
					<header class="bloques lado">
						<div class="blq-marca">
							{@render logo(48, true)}
							<h1>{n.nombre}</h1>
						</div>
						{@render foto(0, 'blq-foto')}
						<div class="blq-acento">
							{#if hayResenas}
								<b class="tl-num">{promedio} ★</b><span>{nResenas}</span>
							{:else}
								<b class="tl-num">{pagina.servicios.length}</b><span>{pagina.servicios.length === 1 ? 'servicio' : 'servicios'}</span>
							{/if}
						</div>
						{@render foto(1, 'blq-foto1')}
					</header>
				{:else}
					<header class="carta">
						{#if fotos[0]}<img class="foto-op" src={fotos[0]} alt={`El local de ${n.nombre}`} />{/if}
						<div class="degrade"></div>
						<div class="grano"></div>
						<div class="carta-texto">
							<span class="carta-logo">{@render logo(64, true)}</span>
							<span class="pila4">
								<h1>{n.nombre}</h1>
								{#if meta}<span class="carta-meta">{meta}</span>{/if}
							</span>
						</div>
					</header>
				{/if}

				{#if hayResenas || abierto}
					<div class="puntaje lado">
						{#if hayResenas}
							<a href="#resenas" aria-label={`${pagina.resenas.promedio.toLocaleString('es-AR')} de 5 (${nResenas})`}>
								<b class="tl-num">{promedio}</b><span class="estrellas" aria-hidden="true">{estrellas(pagina.resenas.promedio)}</span><span class="sub subrayado">{nResenas}</span>
							</a>
						{/if}
						{#if abierto}
							<span class="abierto" class:cerrado={!abierto.abierto}><Icono nombre={abierto.abierto ? 'schedule' : 'do_not_disturb_on'} tam={16} />{abierto.texto}</span>
						{/if}
					</div>
				{/if}

				<div class="chips lado">
					{#each chips as c (c.href)}
						<a class="chip" href={c.href} target={c.href.startsWith('http') ? '_blank' : undefined} rel="noopener">
							{#if c.logo}<img src={c.logo} alt="" width="20" height="20" />{:else}<Icono nombre={c.icono ?? 'link'} tam={20} lleno />{/if}{c.nombre}
						</a>
					{/each}
					<a class="chip" href={compartirWhatsapp(`${textoCompartir}: ${base}`)} onclick={compartir} target="_blank" rel="noopener">
						<Icono nombre="ios_share" tam={20} lleno />Compartir
					</a>
				</div>

				{#each orden as x (x)}
					{#if x === 'servicios'}{@render sServicios()}{:else if x === 'membresias'}{@render sMembresias()}{:else if x === 'resenas'}{@render sResenas()}{:else if x === 'equipo'}{@render sEquipo()}{:else}{@render sFotos()}{/if}
				{/each}

				<section id="donde" class="donde lado" aria-label="Dónde y cuándo">
					{#if n.direccion}
						<div class="donde-fila">
							<Icono nombre="location_on" lleno />
							<span class="pila2">
								<b>{n.direccion}{n.ciudad ? `, ${n.ciudad}` : ''}</b>
								{#if w.provincia}<span class="sub chico">{w.provincia}</span>{/if}
							</span>
							{#if mapa}<a class="enlace-marca" href={mapa} target="_blank" rel="noopener">Cómo llegar</a>{/if}
						</div>
					{/if}
					{#if atencion}
						<div class="donde-fila">
							<Icono nombre="schedule" lleno />
							<span class="atencion tl-num">{#each renglonesAtencion(atencion) as r (r)}{r}<br />{/each}</span>
						</div>
					{/if}
					{#if acerca}
						<details>
							<summary><Icono nombre="info" lleno /><b>Acerca de {n.nombre}</b><span class="mas"><Icono nombre="expand_more" /></span></summary>
							<p>{acerca}</p>
						</details>
					{/if}
					{#if pagina.politicaCancelacion || cobroTexto}
						<details>
							<summary><Icono nombre="policy" lleno /><b>Cancelaciones</b><span class="mas"><Icono nombre="expand_more" /></span></summary>
							{#if cobroTexto}<p>{cobroTexto}</p>{/if}
							{#if pagina.politicaCancelacion}<p class="pre">{pagina.politicaCancelacion}</p>{/if}
						</details>
					{/if}
				</section>

				<a class="hecho" href="/"><img src="/marca-icono.png" alt="" width="18" height="18" />Turnos con <b>Tengo&nbsp;Libre</b></a>
			</div>

			<div class="columna">
				{#if reserva}
					{@render reserva()}
				{:else}
					<PanelReserva primer={proximo} volver={base}>
						<section class="paso" aria-labelledby="t-p1">
							<div class="paso-cab"><span class="paso-n">01</span><h2 id="t-p1">Qué te hacés</h2></div>
							<div class="opciones">
								{#each pagina.servicios as s (s.id)}<a class="opcion" href={`${reservar}?servicio=${s.id}`}>{s.nombre}</a>{/each}
							</div>
						</section>
						{#snippet barra()}
							<span class="barra-texto">
								<b>Elegí un servicio</b>
								<span>{pagina.servicios.length} {pagina.servicios.length === 1 ? 'servicio' : 'servicios'} para&nbsp;elegir</span>
							</span>
							<span class="boton-reserva apagado" aria-disabled="true">Reservar</span>
						{/snippet}
					</PanelReserva>
				{/if}
			</div>
		</div>
	</div>

	{#if vista === 'pagina'}
		<div class="barra-pagina {forzado ?? ''}">
			<span class="bp-texto">
				{#if proximo}
					<span class="bp-rotulo tl-num">PRÓXIMO LUGAR</span>
					<b class="tl-num">{diaCorto(proximo.fecha)} · {proximo.hora}</b>
				{:else}
					<span class="bp-rotulo tl-num">TURNOS ONLINE</span>
					<b>{n.nombre}</b>
				{/if}
			</span>
			<a class="bp-cta" href={reservar}>Reservar turno</a>
		</div>
	{/if}
</div>

<style>
	.web-tema {
		--grano: var(--tl-grano);
		background: var(--w-bg);
		color: var(--w-ink);
		min-height: 100vh;
		transition: background 0.3s, color 0.3s;
		max-width: 100%;
	}
	.web-tema.incrustado {
		min-height: 100%;
	}
	.web {
		container: web / inline-size;
	}
	.lienzo {
		display: grid;
		grid-template-columns: minmax(0, 1fr);
		max-width: 1240px;
		margin: 0 auto;
		align-items: start;
	}
	.contenido {
		display: flex;
		flex-direction: column;
		gap: 28px;
		min-width: 0;
		padding-bottom: 96px;
	}
	.columna {
		display: none;
		min-width: 0;
	}
	.lienzo.reservando .contenido {
		display: none;
	}
	.lienzo.reservando .columna {
		display: block;
	}
	.lado {
		padding-left: 16px;
		padding-right: 16px;
	}
	.lado-m {
		margin-left: 16px;
		margin-right: 16px;
	}
	h1,
	h2 {
		margin: 0;
		font-weight: var(--w-tw);
		letter-spacing: var(--w-tl);
	}
	h2 {
		font-size: 28px;
		line-height: 1.1;
	}
	.sub {
		color: var(--w-sub);
	}
	.chico {
		font-size: 13px;
	}
	.chico-num {
		font: 500 12px var(--tl-font);
		font-variant-numeric: tabular-nums;
	}
	.pila0,
	.pila2,
	.pila4,
	.pila14 {
		display: flex;
		flex-direction: column;
		min-width: 0;
	}
	.pila2 {
		gap: 2px;
	}
	.pila4 {
		gap: 4px;
	}
	.pila14 {
		gap: 14px;
	}
	.logo {
		flex: none;
		border-radius: var(--w-cr);
		object-fit: cover;
	}
	.inicial {
		display: grid;
		place-items: center;
		font-weight: 700;
		background: var(--w-pri);
		color: var(--w-on);
	}
	.inicial.invertido {
		background: var(--w-on);
		color: var(--w-pri);
	}
	.sin-foto {
		display: block;
		background-color: var(--w-pri);
		background-image: var(--grano);
	}
	.foto-op {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		object-fit: cover;
		display: block;
		opacity: var(--w-foto-op);
		transition: opacity 0.3s;
	}

	/* Carta */
	.carta {
		position: relative;
		height: 300px;
		overflow: hidden;
		background: var(--w-pri);
	}
	.degrade {
		position: absolute;
		inset: 0;
		background: linear-gradient(to top, var(--w-pri) 4%, var(--w-pri-a) 38%, transparent 72%);
	}
	.grano {
		position: absolute;
		inset: 0;
		background-image: var(--grano);
		mix-blend-mode: overlay;
		opacity: 0.9;
	}
	.carta-texto {
		position: absolute;
		left: 16px;
		right: 16px;
		bottom: 16px;
		display: flex;
		align-items: flex-end;
		gap: 14px;
		color: var(--w-on);
	}
	.carta-texto h1 {
		font-size: 38px;
		line-height: 0.95;
	}
	.carta-meta {
		font-size: 14px;
		opacity: 0.9;
	}
	.carta-logo :global(.logo) {
		font-size: 28px !important;
	}

	/* Vidriera */
	.vidriera {
		position: relative;
		height: 520px;
		overflow: hidden;
		background: var(--w-pri);
	}
	.sombra-abajo {
		position: absolute;
		inset: 0;
		background: linear-gradient(to top, rgba(15, 35, 40, 0.82) 0%, rgba(15, 35, 40, 0.2) 50%, transparent 70%);
	}
	.vid-texto {
		position: absolute;
		left: 0;
		right: 0;
		bottom: 16px;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 12px;
		text-align: center;
		color: #fff;
		padding: 0 16px;
	}
	.vid-logo :global(.logo) {
		box-shadow: 0 0 0 4px rgba(255, 255, 255, 0.9);
	}
	.vid-texto h1 {
		font-size: 38px;
		line-height: 0.95;
	}
	.vid-meta {
		font-size: 14px;
		opacity: 0.9;
	}
	.vid-cta {
		display: inline-flex;
		align-items: center;
		height: 48px;
		padding: 0 24px;
		border-radius: var(--w-br);
		background: var(--w-pri);
		color: var(--w-on);
		font-size: 15px;
		font-weight: 700;
		text-decoration: none;
	}

	/* Sobria */
	.sobria {
		position: relative;
		overflow: hidden;
		padding: 64px 20px 28px;
		background-color: var(--w-pri);
		background-image: var(--grano);
		color: var(--w-on);
		display: flex;
		flex-direction: column;
		gap: 18px;
	}
	.aro {
		position: absolute;
		right: -80px;
		top: -80px;
		width: 280px;
		height: 280px;
		border-radius: 50%;
		box-shadow: inset 0 0 0 44px var(--w-on-a);
	}
	.sobria > :global(.logo) {
		position: relative;
		font-size: 30px !important;
	}
	.sobria h1 {
		position: relative;
		font-size: 64px;
		line-height: 0.92;
	}
	.sobria-desc {
		position: relative;
		font-size: 15px;
		opacity: 0.9;
		max-width: 28em;
		text-wrap: pretty;
	}

	/* Galería */
	.galeria-cab {
		display: flex;
		flex-direction: column;
		gap: 14px;
	}
	.gal-fotos {
		display: grid;
		grid-template-columns: 2fr 1fr;
		grid-template-rows: 150px 120px;
		gap: 6px;
		overflow: hidden;
	}
	.gal-fotos > :global(*) {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.gal-fotos > :global(.g1) {
		grid-row: span 2;
	}
	.gal-titulo {
		display: flex;
		align-items: center;
		gap: 12px;
	}
	.gal-titulo :global(.logo) {
		border-radius: 15px;
	}
	.gal-titulo h1 {
		font-size: 30px;
		line-height: 1;
	}
	.gal-titulo .sub {
		font-size: 14px;
	}

	/* Editorial */
	.editorial {
		display: grid;
		grid-template-columns: minmax(0, 1fr) 120px;
		gap: 24px;
		align-items: end;
		padding: 28px 16px 0;
	}
	.ed-meta {
		font-size: 14px;
		font-weight: 700;
		color: var(--w-marca);
	}
	.editorial h1 {
		font-size: 64px;
		line-height: 0.84;
		overflow-wrap: anywhere;
	}
	.ed-desc {
		font-size: 16px;
		line-height: 1.45;
		color: var(--w-sub2);
		max-width: 24em;
		text-wrap: pretty;
	}
	.editorial :global(.ed-foto) {
		width: 100%;
		aspect-ratio: 1;
		object-fit: cover;
		border-radius: 50%;
		box-shadow: 0 0 0 10px var(--w-tint);
	}

	/* Bloques */
	.bloques {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		grid-auto-rows: 110px;
		gap: 8px;
	}
	.blq-marca {
		grid-column: span 2;
		grid-row: span 2;
		border-radius: var(--w-cr);
		background-color: var(--w-pri);
		background-image: var(--grano);
		color: var(--w-on);
		padding: 20px;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		min-width: 0;
	}
	.blq-marca > :global(.logo) {
		border-radius: var(--w-sr);
		font-size: 22px !important;
	}
	.blq-marca h1 {
		font-size: 28px;
		line-height: 0.92;
		overflow-wrap: anywhere;
	}
	.bloques :global(.blq-foto) {
		grid-column: span 2;
	}
	.bloques :global(.blq-foto),
	.bloques :global(.blq-foto1) {
		width: 100%;
		height: 100%;
		object-fit: cover;
		border-radius: var(--w-cr);
		min-width: 0;
	}
	.blq-acento {
		border-radius: var(--w-cr);
		background: var(--w-acc);
		padding: 14px;
		display: flex;
		flex-direction: column;
		justify-content: flex-end;
		color: #14181a;
		min-width: 0;
	}
	.blq-acento b {
		font-size: 28px;
		letter-spacing: -0.04em;
		line-height: 1;
	}
	.blq-acento span {
		font-size: 13px;
	}

	/* Puntaje y chips */
	.puntaje {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px 16px;
	}
	.puntaje a {
		display: flex;
		align-items: baseline;
		gap: 6px;
		text-decoration: none;
		min-height: 32px;
	}
	.puntaje a b {
		font: 700 20px var(--tl-font);
		letter-spacing: -0.03em;
	}
	.estrellas {
		color: var(--w-acc);
		font-size: 16px;
		letter-spacing: 1px;
	}
	.subrayado {
		font-size: 13px;
		text-decoration: underline;
		text-underline-offset: 3px;
	}
	.abierto {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		font-size: 13px;
		font-weight: 700;
		color: var(--w-ok);
	}
	.abierto.cerrado {
		color: var(--w-sub);
	}
	.chips {
		display: flex;
		gap: 8px;
		overflow-x: auto;
		scrollbar-width: none;
	}
	.chip {
		flex: none;
		display: inline-flex;
		align-items: center;
		gap: 8px;
		height: 48px;
		padding: 0 16px;
		border-radius: var(--w-br);
		background: var(--w-papel);
		box-shadow: inset 0 0 0 1px var(--w-line);
		font-size: 14px;
		font-weight: 700;
		text-decoration: none;
		color: var(--w-ink);
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.chip:active {
		transform: scale(0.96);
	}

	/* Carta de servicios */
	.carta-servicios {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	.titulo-fila {
		display: flex;
		align-items: baseline;
		gap: 12px;
		margin-bottom: 6px;
	}
	.titulo-fila h2 {
		flex: 1;
	}
	.categoria {
		margin: 0;
		font: 700 11px var(--tl-font);
		font-variant-numeric: tabular-nums;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: var(--w-marca);
		padding: 14px 0 4px;
	}
	.servicio {
		display: flex;
		flex-direction: column;
		gap: 2px;
		width: 100%;
		padding: 12px 0;
		border-bottom: 1px solid var(--w-line);
		border-radius: 10px;
		color: var(--w-ink);
		text-decoration: none;
		transition: padding 0.2s var(--tl-ease), background 0.2s;
	}
	.servicio:hover {
		padding-left: 10px;
		padding-right: 10px;
		background: var(--w-tint);
	}
	.servicio:active {
		transform: scale(0.99);
	}
	.srv-fila {
		display: flex;
		align-items: baseline;
		gap: 8px;
	}
	.srv-nombre {
		font-size: 17px;
		font-weight: 700;
		min-width: 0;
	}
	.puntos {
		flex: 1;
		min-width: 20px;
		border-bottom: 2px dotted var(--w-line2);
		transform: translateY(-4px);
	}
	.srv-fila b {
		font: 700 16px var(--tl-font);
		letter-spacing: -0.02em;
		white-space: nowrap;
	}
	.srv-sub {
		display: flex;
		justify-content: space-between;
		gap: 12px;
		font-size: 13px;
		color: var(--w-sub);
	}
	.srv-cta {
		font-weight: 700;
		color: var(--w-marca);
		flex: none;
	}

	/* Membresía */
	.plan {
		border-radius: var(--w-cr);
		overflow: hidden;
		background: var(--w-socios);
		display: grid;
		grid-template-columns: minmax(0, 1fr) 120px;
	}
	.plan-texto {
		padding: 18px 20px;
		display: flex;
		flex-direction: column;
		gap: 6px;
		border-right: 2px dashed #dccaa2;
	}
	.plan-rotulo {
		display: flex;
		gap: 6px;
		align-items: center;
		font: 700 11px var(--tl-font);
		letter-spacing: 0.1em;
		color: var(--w-socios-txt);
	}
	.plan-nombre {
		font-size: 19px;
		letter-spacing: -0.02em;
	}
	.plan-desc {
		font-size: 14px;
		line-height: 1.45;
		color: var(--w-socios-txt);
		text-wrap: pretty;
	}
	.plan-precio {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 2px;
		padding: 12px;
		text-align: center;
	}
	.plan-precio b {
		font: 700 20px var(--tl-font);
		letter-spacing: -0.03em;
	}
	.plan-precio span {
		font-size: 12px;
		color: var(--w-socios-txt);
	}

	/* Reseñas */
	.resenas {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	.resenas .titulo-fila {
		margin: 0;
	}
	.prom {
		font: 700 20px var(--tl-font);
	}
	figure {
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	blockquote {
		margin: 0;
		font-size: 22px;
		line-height: 1.3;
		font-weight: 400;
		letter-spacing: -0.02em;
		text-wrap: pretty;
	}
	figcaption {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		align-items: center;
		font-size: 13px;
		color: var(--w-sub);
	}
	.resena {
		display: flex;
		flex-direction: column;
		gap: 8px;
		padding: 16px;
		border-radius: var(--w-cr);
		background: var(--w-papel);
		font-size: 14px;
		line-height: 1.5;
	}
	.resena-cab {
		display: flex;
		gap: 8px;
		align-items: baseline;
	}
	.cuando {
		margin-left: auto;
		font: 400 11px var(--tl-font);
		color: var(--w-sub);
	}
	.respuesta {
		padding: 10px 12px;
		border-radius: 12px;
		background: var(--w-tint);
		font-size: 13px;
	}

	/* Equipo */
	.equipo {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.equipo ul {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		gap: 18px;
		overflow-x: auto;
	}
	.equipo li {
		flex: none;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 6px;
		width: 92px;
		text-align: center;
	}
	.equipo img,
	.ini {
		width: 72px;
		height: 72px;
		border-radius: 50%;
		object-fit: cover;
	}
	.ini {
		background: var(--w-tint);
		display: grid;
		place-items: center;
		font-weight: 700;
		color: var(--w-marca);
	}
	.equipo b {
		font-size: 14px;
	}
	.equipo li span:not(.ini) {
		font-size: 12px;
		color: var(--w-sub);
		line-height: 1.2;
	}

	/* Fotos */
	.fotos {
		display: grid;
		grid-template-columns: var(--cols);
		grid-template-rows: 140px;
		gap: 6px;
		border-radius: var(--w-cr);
		overflow: hidden;
	}
	.fotos img {
		width: 100%;
		height: 100%;
		object-fit: cover;
		min-width: 0;
	}

	/* Dónde */
	.donde {
		display: flex;
		flex-direction: column;
	}
	.donde-fila {
		display: flex;
		align-items: flex-start;
		gap: 14px;
		padding: 16px 0;
		border-top: 1px solid var(--w-line);
	}
	.donde-fila > :global(.icono),
	summary > :global(.icono:first-child) {
		color: var(--w-marca);
	}
	.donde-fila .pila2 {
		flex: 1;
	}
	.donde-fila b {
		font-size: 16px;
	}
	.atencion {
		flex: 1;
		font-size: 14px;
		line-height: 1.6;
	}
	.enlace-marca {
		font-size: 14px;
		font-weight: 700;
		color: var(--w-marca);
		text-underline-offset: 3px;
	}
	details {
		border-top: 1px solid var(--w-line);
	}
	details:last-child {
		border-bottom: 1px solid var(--w-line);
	}
	summary {
		display: flex;
		align-items: center;
		gap: 14px;
		min-height: 56px;
		cursor: pointer;
		list-style: none;
	}
	summary::-webkit-details-marker {
		display: none;
	}
	summary b {
		flex: 1;
		font-size: 16px;
	}
	.mas {
		color: var(--w-sub);
		display: grid;
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	details[open] .mas {
		transform: rotate(180deg);
	}
	details p {
		margin: 0 0 16px 36px;
		font-size: 14px;
		line-height: 1.55;
		color: var(--w-sub2);
	}
	.pre {
		white-space: pre-line;
	}
	.hecho {
		display: flex;
		justify-content: center;
		gap: 8px;
		align-items: center;
		font-size: 12px;
		color: var(--w-sub);
		padding-bottom: 12px;
		text-decoration: none;
	}
	.hecho img {
		border-radius: 5px;
	}
	.hecho b {
		color: var(--w-ink);
	}

	/* Barra del teléfono */
	.barra-pagina {
		position: fixed;
		left: 0;
		right: 0;
		bottom: 0;
		z-index: 20;
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px 16px calc(22px + env(safe-area-inset-bottom));
		background: color-mix(in srgb, var(--w-papel) 94%, transparent);
		backdrop-filter: blur(10px);
		-webkit-backdrop-filter: blur(10px);
		box-shadow: 0 -1px 0 var(--w-line);
	}
	.incrustado .barra-pagina {
		position: sticky;
	}
	.bp-texto {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		line-height: 1.25;
	}
	.bp-texto b {
		font: 700 16px var(--tl-font);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.bp-rotulo {
		font: 400 11px var(--tl-font);
		color: var(--w-sub);
		letter-spacing: 0.02em;
	}
	.bp-cta {
		display: inline-flex;
		align-items: center;
		min-height: 52px;
		padding: 0 24px;
		border-radius: var(--w-br);
		background: var(--w-pri);
		color: var(--w-on);
		font-size: 16px;
		font-weight: 700;
		text-decoration: none;
		white-space: nowrap;
		box-shadow: 0 12px 24px -12px var(--w-pri);
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.bp-cta:hover {
		transform: translateY(-2px);
	}
	.bp-cta:active {
		transform: scale(0.96);
	}
	@media (min-width: 900px) {
		.barra-pagina:not(.compacto) {
			display: none;
		}
	}
	.barra-pagina.amplio {
		display: none;
	}
	.lienzo.compacto .columna {
		display: none;
	}
	.lienzo.compacto.reservando .columna {
		display: block;
	}

	/* Escritorio: página a la izquierda y la reserva a la derecha, sticky. */
	@container web (min-width: 900px) {
		.lienzo:not(.compacto) {
			grid-template-columns: minmax(0, 1fr) 410px;
			gap: 48px;
			padding: 28px 32px 64px;
		}
		.lienzo:not(.compacto) .contenido {
			display: flex;
			padding-bottom: 0;
		}
		.lienzo:not(.compacto) .columna {
			display: block;
			position: sticky;
			top: 24px;
			max-height: calc(100vh - 48px);
			overflow-y: auto;
			overscroll-behavior: contain;
			border-radius: 24px;
			box-shadow: 0 0 0 1px #eee5d3, 0 30px 60px -30px rgba(15, 35, 40, 0.4);
		}
		.lienzo:not(.compacto) .lado {
			padding-left: 0;
			padding-right: 0;
		}
		.lienzo:not(.compacto) .lado-m {
			margin-left: 0;
			margin-right: 0;
		}
		.lienzo:not(.compacto) .carta {
			height: 380px;
			border-radius: 28px;
		}
		.lienzo:not(.compacto) .carta-texto {
			left: 28px;
			right: 28px;
			bottom: 28px;
		}
		.lienzo:not(.compacto) .carta-texto h1,
		.lienzo:not(.compacto) .vid-texto h1 {
			font-size: 56px;
		}
		.lienzo:not(.compacto) .carta-logo :global(.logo) {
			width: 80px !important;
			height: 80px !important;
		}
		.lienzo:not(.compacto) .vidriera {
			height: 560px;
			border-radius: 28px;
		}
		.lienzo:not(.compacto) .vid-texto {
			bottom: 28px;
		}
		.lienzo:not(.compacto) .sobria {
			padding: 56px 44px 40px;
			border-radius: 28px;
		}
		.lienzo:not(.compacto) .sobria h1 {
			font-size: 88px;
		}
		.lienzo:not(.compacto) .gal-fotos {
			grid-template-rows: 220px 180px;
			border-radius: 28px;
		}
		.lienzo:not(.compacto) .editorial {
			grid-template-columns: minmax(0, 1fr) 260px;
			padding: 12px 0 0;
		}
		.lienzo:not(.compacto) .editorial h1 {
			font-size: 112px;
		}
		.lienzo:not(.compacto) .bloques {
			grid-auto-rows: 150px;
		}
		.lienzo:not(.compacto) .blq-marca h1 {
			font-size: 44px;
		}
		.lienzo:not(.compacto) .fotos {
			grid-template-rows: 200px;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.servicio:hover {
			padding-left: 0;
			padding-right: 0;
		}
	}
</style>
