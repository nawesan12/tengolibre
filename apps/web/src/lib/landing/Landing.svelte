<!--
	Landing de Tengo Libre (la ruta /). Copia adaptada de la portada de Corredor, con los lineamientos de
	docs/lineamientos-landing.md: poco texto, sólo los colores del ícono (petróleo #183c44 del símbolo y
	menta #3ccc90 del fondo), pocas cajas, logos y fotos reales.
	Fotos de Unsplash (static/fotos/FUENTES.md); logos de terceros en static/logos/FUENTES.md.
	Las clases evitan los nombres globales de app.css (.boton, .tarjeta, .grilla, .etiqueta…).
-->
<script lang="ts">
	import { aparecer } from './aparecer.ts';
	import Panel from './Panel.svelte';
	import Telefono from './Telefono.svelte';

	// La barra fija del celular aparece recién cuando los botones del hero salieron de pantalla
	// (si no, se ven dos «Probar 7 días gratis» juntos), y se esconde mientras se ven los del cierre.
	let heroPasado = $state(false);
	let cierreVisible = $state(false);
	const barra = $derived(heroPasado && !cierreVisible);
	function vigilarBotones(nodo: HTMLElement) {
		if (typeof IntersectionObserver === 'undefined') return;
		const obs = new IntersectionObserver(([e]) => {
			heroPasado = !e.isIntersecting && e.boundingClientRect.top < 0;
		});
		obs.observe(nodo);
		return { destroy: () => obs.disconnect() };
	}
	function vigilarCierre(nodo: HTMLElement) {
		if (typeof IntersectionObserver === 'undefined') return;
		const obs = new IntersectionObserver(([e]) => {
			cierreVisible = e.isIntersecting;
		});
		obs.observe(nodo);
		return { destroy: () => obs.disconnect() };
	}

	// Logos reales de cada marca (static/logos): de sus sitios o de Wikimedia.
	const integraciones = [
		['mercadopago.svg', 'Mercado Pago'],
		['whatsapp.svg', 'WhatsApp'],
		['google.svg', 'Google'],
		['googlecalendar.svg', 'Google Calendar'],
		['apple.svg', 'Apple']
	];

	const funciones = [
		['event_available', 'Turnos online'],
		['calendar_month', 'Agenda'],
		['notifications_active', 'Recordatorios'],
		['payments', 'Señas'],
		['group', 'Clientes'],
		['card_membership', 'Membresías'],
		['confirmation_number', 'Cupones'],
		['redeem', 'Referidos'],
		['star', 'Reseñas'],
		['hourglass_top', 'Lista de espera'],
		['language', 'Tu página'],
		['storefront', 'Sucursales']
	] as const;

	const anio = new Date().getFullYear();
</script>

<svelte:head>
	<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin="anonymous" />
	<link
		href="https://fonts.googleapis.com/css2?family=Material+Symbols+Rounded:opsz,wght,FILL,GRAD@24,400,0..1,0&display=block"
		rel="stylesheet"
	/>
	<link rel="icon" href="/marca-icono.png" />
	<meta name="theme-color" content="#183c44" />
	<meta property="og:title" content="Tengo Libre · Que tus clientes saquen turno solos" />
	<meta property="og:image" content="/fotos/peluquera-con-clienta.jpg" />
</svelte:head>

<!-- El símbolo del ícono (un círculo grueso atravesado por una barra), como forma de fondo. -->
{#snippet simbolo(clase: string)}
	<svg class={clase} viewBox="0 0 200 220" aria-hidden="true">
		<circle cx="100" cy="118" r="64" fill="none" stroke="currentColor" stroke-width="28" />
		<rect x="86" y="6" width="28" height="208" rx="6" fill="currentColor" />
	</svg>
{/snippet}

<!-- Las apps nativas todavía no están: los badges oficiales con «Próximamente» arriba (sin link). -->
{#snippet tiendas(clase: string)}
	<div class="tiendas {clase}" style="--d: 480ms">
		<span>Próximamente</span>
		<div>
			<img src="/logos/appstore.svg" alt="Próximamente en App Store" width="120" height="40" />
			<img src="/logos/googleplay.png" alt="Próximamente en Google Play" width="135" height="40" />
		</div>
	</div>
{/snippet}

{#snippet marca(clase: string)}
	<span class="marca {clase}"><img src="/marca-icono.png" alt="" width="32" height="32" />Tengo Libre</span>
{/snippet}

<div class="pagina">
	<header class="hero grano">
		{@render simbolo('simbolo simbolo-hero')}
		<nav aria-label="Principal">
			<a href="/" class="inicio" aria-label="Tengo Libre, inicio">{@render marca('')}</a>
			<span class="acciones">
				<a class="link" href="/ingresar">Ingresar</a>
				<a class="cta menta chica" href="/empezar">Probar gratis</a>
			</span>
		</nav>

		<div class="hero-cuerpo">
			<div class="texto">
				<span class="pastilla entra"><i></i>Barberías, salones y estética</span>
				<h1 class="entra" style="--d: 120ms">
					Que tus clientes <span class="acento subrayado">saquen turno solos.</span>
				</h1>
				<div class="ctas entra" style="--d: 380ms" use:vigilarBotones>
					<a class="cta menta" href="/empezar">
						Probar 7 días gratis <span class="material-symbols-rounded flecha" aria-hidden="true">arrow_forward</span>
					</a>
					<a class="cta fantasma" href="/explorar">Buscar un negocio</a>
				</div>
				{@render tiendas('entra')}
			</div>
			<div class="foto entra" style="--d: 300ms">
				<img
					src="/fotos/peluquera-con-clienta.jpg"
					alt="Una peluquera sonríe mientras le seca el pelo a una clienta"
					width="1600"
					height="1068"
					fetchpriority="high"
				/>
				<div class="flotante turno-nuevo flota" aria-hidden="true">
					<span class="ic material-symbols-rounded">event_available</span>
					<span><strong>Nuevo turno</strong><small>Corte · 10:30</small></span>
				</div>
				<div class="flotante sena flota lento" aria-hidden="true">
					<span class="ic blanco"><img src="/logos/mercadopago.svg" alt="" /></span>
					<span><strong>Seña pagada</strong><small>$ 5.000</small></span>
				</div>
			</div>
		</div>

		<div class="cinta" aria-label="Integraciones">
			<div class="pista">
				{#each [0, 1] as vuelta (vuelta)}
					<ul aria-hidden={vuelta === 1}>
						{#each integraciones as [logo, nombre] (nombre)}
							<li><img src="/logos/{logo}" alt="" width="26" height="26" />{nombre}</li>
						{/each}
					</ul>
				{/each}
			</div>
		</div>
	</header>

	<main>
		<section id="sistema" class="sistema puntos-fondo">
			<div class="encabezado centrado" use:aparecer>
				<span class="ojo">El sistema</span>
				<h2>Tu agenda, a la vista.</h2>
			</div>
			<div class="maqueta" use:aparecer={150}>
				<Panel />
				<div class="tel flota lento"><Telefono /></div>
				<div class="flotante recordatorio flota" aria-hidden="true">
					<span class="ic blanco"><span class="material-symbols-rounded" aria-hidden="true">mail</span></span>
					<span><strong>Recordatorio por mail</strong><small>Mañana · 10:30</small></span>
				</div>
			</div>
		</section>

		<section class="bloque">
			<div class="bloque-foto" use:aparecer>
				<img
					class="retrato"
					src="/fotos/barbero-cortando.jpg"
					alt="Un barbero con delantal le seca el pelo a un cliente"
					width="1067"
					height="1600"
					loading="lazy"
				/>
				<div class="flotante chat" aria-hidden="true">
					<span class="chat-de"><span class="material-symbols-rounded" aria-hidden="true">mail</span>Barbería Norte</span>
					<span class="msj vos">Hola Juan, te esperamos mañana a las 10:30 ✂️</span>
					<span class="msj el">Ver mi turno</span>
				</div>
			</div>
			<div class="bloque-texto" use:aparecer={120}>
				<span class="num">01</span>
				<h2>Reservan solos, a toda hora.</h2>
				<p>Eligen servicio, profesional y horario desde tu página.</p>
				<ul class="puntos">
					<li><span class="material-symbols-rounded" aria-hidden="true">check_circle</span>Recordatorio automático</li>
					<li><span class="material-symbols-rounded" aria-hidden="true">check_circle</span>Reprograman solos</li>
					<li><span class="material-symbols-rounded" aria-hidden="true">check_circle</span>Lista de espera</li>
				</ul>
			</div>
		</section>

		<section class="bloque invertido">
			<div class="bloque-foto" use:aparecer>
				<img
					src="/fotos/recepcion.jpg"
					alt="Una clienta paga en el mostrador de un local y la encargada la atiende sonriendo"
					width="1600"
					height="1067"
					loading="lazy"
				/>
				<div class="flotante tarjeta-sena" aria-hidden="true">
					<span class="sena-de"><img src="/logos/mercadopago.svg" alt="" />Seña · Mercado Pago</span>
					<span class="montos"><b>$ 5.000</b><em>Pagada</em></span>
					<small>Corte y barba · Sábado 11:00</small>
				</div>
			</div>
			<div class="bloque-texto" use:aparecer={120}>
				<span class="num">02</span>
				<h2>La seña, cobrada al reservar.</h2>
				<p>Con Mercado Pago: menos ausencias y la plata en tu cuenta.</p>
				<ul class="puntos">
					<li><span class="material-symbols-rounded" aria-hidden="true">check_circle</span>Seña o total</li>
					<li><span class="material-symbols-rounded" aria-hidden="true">check_circle</span>Membresías</li>
					<li><span class="material-symbols-rounded" aria-hidden="true">check_circle</span>Cupones</li>
				</ul>
			</div>
		</section>

		<section class="bloque">
			<div class="bloque-foto" use:aparecer>
				<img
					src="/fotos/manicura.jpg"
					alt="Una manicura le pinta las uñas a una clienta"
					width="1600"
					height="1068"
					loading="lazy"
				/>
				<div class="flotante ficha" aria-hidden="true">
					<span class="avatar">SM</span>
					<span
						><strong>Sofía M.</strong><small>12 visitas · Semipermanente</small><em>Vuelve en 2 semanas</em></span
					>
				</div>
			</div>
			<div class="bloque-texto" use:aparecer={120}>
				<span class="num">03</span>
				<h2>Conocés a cada cliente.</h2>
				<p>Qué se hizo, cuándo vino y cuándo le toca volver.</p>
				<ul class="puntos">
					<li><span class="material-symbols-rounded" aria-hidden="true">check_circle</span>Historial</li>
					<li><span class="material-symbols-rounded" aria-hidden="true">check_circle</span>Reseñas</li>
					<li><span class="material-symbols-rounded" aria-hidden="true">check_circle</span>Referidos</li>
				</ul>
			</div>
		</section>

		<section class="funciones">
			<div class="encabezado centrado" use:aparecer>
				<span class="ojo">Y además</span>
				<h2>Todo lo que pasa entre turno y turno.</h2>
			</div>
			<ul>
				{#each funciones as [icono, titulo], i (titulo)}
					<li use:aparecer={i * 40}>
						<span class="material-symbols-rounded" aria-hidden="true">{icono}</span>
						{titulo}
					</li>
				{/each}
			</ul>
		</section>

		<section class="pasos grano">
			<div class="encabezado centrado" use:aparecer>
				<span class="ojo">Cómo empezás</span>
				<h2>En tres pasos.</h2>
			</div>
			<ol>
				<li use:aparecer>
					<div class="escena" aria-hidden="true">
						<div class="mini ingreso">
							<img src="/marca-icono.png" alt="" class="mini-icono" />
							<b>Creá tu negocio</b>
							<span class="boton-mini"><img src="/logos/google.svg" alt="" />Seguir con Google</span>
							<span class="boton-mini negro"><img src="/logos/apple.svg" alt="" />Seguir con Apple</span>
						</div>
					</div>
					<span class="numero">01</span>
					<b>Entrás</b>
					<span>Con Google o Apple.</span>
				</li>
				<li use:aparecer={150}>
					<div class="escena" aria-hidden="true">
						<div class="mini carga">
							<span class="linea"><b>Corte</b><small>30 min</small></span>
							<span class="linea"><b>Color</b><small>2 h</small></span>
							<span class="subiendo"><small>Armando tu agenda</small><small>3 de 3</small></span>
							<span class="barra-carga"><i></i></span>
						</div>
					</div>
					<span class="numero">02</span>
					<b>Cargás</b>
					<span>Servicios, horarios y equipo.</span>
				</li>
				<li use:aparecer={300}>
					<div class="escena" aria-hidden="true">
						<div class="mini listo">
							<span class="check material-symbols-rounded">check</span>
							<b>Turno confirmado</b>
							<span>Corte · Sábado 10:30</span>
							<em>tengolibre.com.ar/lumi</em>
						</div>
					</div>
					<span class="numero">03</span>
					<b>Compartís</b>
					<span>Tu link, y llegan los turnos.</span>
				</li>
			</ol>
		</section>
	</main>

	<!-- Cierre: de punta a punta, en el petróleo del ícono, con el teléfono inclinado. -->
	<section class="cierre grano">
		{@render simbolo('simbolo simbolo-cierre')}
		<div class="cierre-cuerpo">
			<div class="cierre-texto" use:aparecer>
				<span class="pastilla"><i></i>Empezá hoy</span>
				<h2>Tu agenda, <span class="acento">llena desde hoy.</span></h2>
				<p>7 días gratis con todo. Después elegís&nbsp;un&nbsp;plan.</p>
				<div class="ctas" use:vigilarCierre>
					<a class="cta menta" href="/empezar">
						Probar 7 días gratis <span class="material-symbols-rounded flecha" aria-hidden="true">arrow_forward</span>
					</a>
					<a class="cta fantasma" href="mailto:hola@tengolibre.com.ar">Hablá con nosotros</a>
				</div>
				{@render tiendas('')}
			</div>
			<div class="cierre-visual" use:aparecer={150}>
				<div class="halo"></div>
				<div class="tel-cierre flota lento"><Telefono /></div>
				<div class="flotante aviso-cierre flota" aria-hidden="true">
					<span class="ic material-symbols-rounded">check_circle</span>
					<span><strong>Turno confirmado</strong><small>Sábado · 11:00</small></span>
				</div>
				<div class="flotante resena-cierre flota" aria-hidden="true">
					<span class="ic material-symbols-rounded">star</span>
					<span><strong>Nueva reseña</strong><small>★★★★★</small></span>
				</div>
			</div>
		</div>
	</section>

	<footer>
		<div class="pie">
			<div class="pie-marca">
				{@render marca('')}
				<p>Que tus clientes saquen turno solos.</p>
			</div>
			<nav aria-label="Producto">
				<b>Producto</b>
				<a href="/empezar">Probar 7 días gratis</a>
				<a href="/explorar">Buscar un negocio</a>
				<a href="/ingresar">Ingresar</a>
			</nav>
			<nav aria-label="Grupo Vimo">
				<b>Grupo Vimo</b>
				<a href="https://grupovimo.com.ar">Quiénes somos</a>
				<a href="https://grupovimo.com.ar/tengo-libre">Tengo Libre en Grupo Vimo</a>
			</nav>
			<nav aria-label="Contacto">
				<b>Contacto</b>
				<a href="mailto:hola@tengolibre.com.ar">hola@tengolibre.com.ar</a>
			</nav>
		</div>
		<div class="pie-final">© {anio} Tengo Libre · Hecho en Argentina</div>
	</footer>

	<a class="cta menta barra" class:visible={barra} href="/empezar" tabindex={barra ? 0 : -1} aria-hidden={!barra}
		>Probar 7 días gratis</a
	>
</div>

<style>
	.pagina {
		/* Los colores del ícono de Tengo Libre: el petróleo del símbolo y la menta del fondo. */
		--petroleo: #183c44;
		--petroleo-hondo: #102b31;
		--petroleo-luz: #22535e;
		--menta: #3ccc90;
		--menta-claro: #6fdcae;
		--menta-suave: #eaf8f2;
		--menta-oscuro: #17805a;
		--crema: #f5efe2;
		--tinta: #0f2328;
		--gris: #4a5d61;
		--ancho: 1200px;
		--margen: clamp(20px, 4vw, 48px);
		/* Un solo ritmo vertical entre secciones. */
		--seccion: clamp(104px, 12vw, 168px);
		/* Grano fotográfico: ruido SVG muy suave sobre los fondos de color. */
		--ruido: url("data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='180' height='180'><filter id='n'><feTurbulence type='fractalNoise' baseFrequency='.85' numOctaves='3' stitchTiles='stitch'/><feColorMatrix values='0 0 0 0 0  0 0 0 0 0  0 0 0 0 0  0 0 0 .55 0'/></filter><rect width='100%' height='100%' filter='url(%23n)'/></svg>");
		min-height: 100vh;
		overflow-x: clip;
		background: #fff;
		font-family: Geist, system-ui, sans-serif;
		font-size: 16px;
		line-height: 1.5;
		color: var(--tinta);
		-webkit-font-smoothing: antialiased;
	}
	a {
		color: inherit;
		text-underline-offset: 3px;
	}
	a:focus-visible {
		outline: 3px solid var(--menta);
		outline-offset: 3px;
		border-radius: 6px;
	}
	.material-symbols-rounded {
		font-family: 'Material Symbols Rounded';
		font-weight: normal;
		font-style: normal;
		line-height: 1;
		font-variation-settings: 'FILL' 1;
		user-select: none;
	}
	.grano {
		position: relative;
		isolation: isolate;
	}
	.grano::after {
		content: '';
		position: absolute;
		inset: 0;
		z-index: -1;
		background-image: var(--ruido);
		opacity: 0.16;
		mix-blend-mode: overlay;
		pointer-events: none;
		border-radius: inherit;
	}
	:global(.simbolo) {
		position: absolute;
		z-index: -1;
		pointer-events: none;
	}

	/* Botones: uno solo por bloque es el principal. */
	.cta {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		min-height: 54px;
		padding: 0 28px;
		border-radius: 999px;
		font-size: 17px;
		font-weight: 600;
		text-decoration: none;
		white-space: nowrap;
		transition:
			background 0.15s,
			transform 0.15s,
			box-shadow 0.15s;
	}
	.cta:hover {
		transform: translateY(-2px);
	}
	.cta:hover .flecha {
		transform: translateX(3px);
	}
	.flecha {
		font-size: 20px;
		transition: transform 0.15s;
	}
	.cta:active {
		transform: scale(0.97);
	}
	.cta:focus-visible {
		outline: 3px solid var(--menta-claro);
		outline-offset: 3px;
		border-radius: 999px;
	}
	.cta.chica {
		min-height: 40px;
		padding: 0 18px;
		font-size: 15px;
	}
	.cta.menta {
		background: var(--menta);
		color: var(--petroleo-hondo);
		box-shadow:
			inset 0 1px 0 oklch(1 0 0 / 0.45),
			0 10px 24px -10px oklch(0.75 0.15 160 / 0.75);
	}
	.cta.menta:hover {
		background: var(--menta-claro);
	}
	.cta.fantasma {
		color: #fff;
		background: oklch(1 0 0 / 0.08);
		box-shadow: inset 0 0 0 1px oklch(1 0 0 / 0.16);
	}
	.cta.fantasma:hover {
		background: oklch(1 0 0 / 0.16);
	}

	.marca {
		display: flex;
		align-items: center;
		gap: 10px;
		font-size: 20px;
		font-weight: 650;
		letter-spacing: -0.02em;
		white-space: nowrap;
	}
	.marca img {
		width: 32px;
		height: 32px;
		border-radius: 8px;
	}
	.inicio {
		text-decoration: none;
	}

	.ojo {
		font-size: 13px;
		font-weight: 700;
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: var(--menta-oscuro);
	}
	h1 {
		margin: 0;
		font-size: clamp(44px, 4.45vw, 64px);
		line-height: 1;
		font-weight: 700;
		letter-spacing: -0.05em;
		text-wrap: balance;
	}
	h2 {
		margin: 0;
		font-size: clamp(34px, 4vw, 58px);
		line-height: 1.04;
		font-weight: 700;
		letter-spacing: -0.045em;
		text-wrap: balance;
	}
	p {
		text-wrap: pretty;
	}
	.acento {
		color: var(--menta);
	}
	/* El título va siempre en dos renglones: «Que tus clientes» / «saquen turno solos.». */
	h1 .acento {
		display: inline-block;
		white-space: nowrap;
	}
	@media (max-width: 640px) {
		h1 {
			font-size: min(9.6vw, 46px);
		}
	}
	/* Subrayado a mano alzada debajo de lo importante. */
	.subrayado {
		background: url("data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 300 20' preserveAspectRatio='none'><path d='M4 14 C80 4 180 4 296 10' fill='none' stroke='%233ccc90' stroke-width='5' stroke-linecap='round' opacity='.55'/></svg>")
			left 100% / 100% 0.28em no-repeat;
		padding-bottom: 0.08em;
	}

	/* Hero: el petróleo del ícono de borde a borde, con luz, grano y el símbolo de fondo. */
	.hero {
		background:
			radial-gradient(60% 70% at 88% 18%, var(--petroleo-luz) 0%, transparent 65%),
			radial-gradient(50% 60% at 0% 100%, #0c2227 0%, transparent 70%),
			var(--petroleo);
		color: #fff;
		overflow: hidden;
	}
	.hero :global(.simbolo-hero) {
		width: min(760px, 80vw);
		right: -12%;
		bottom: -22%;
		color: oklch(1 0 0 / 0.04);
	}
	nav {
		max-width: var(--ancho);
		margin: 0 auto;
		padding: 0 var(--margen);
		height: 76px;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
	}
	.acciones {
		display: flex;
		align-items: center;
		gap: clamp(16px, 2vw, 28px);
		font-size: 15px;
	}
	.link {
		color: oklch(1 0 0 / 0.85);
		text-decoration: none;
		font-weight: 550;
	}
	.link:hover {
		color: #fff;
		text-decoration: underline;
	}
	.hero-cuerpo {
		max-width: var(--ancho);
		margin: 0 auto;
		padding: clamp(28px, 5vw, 64px) var(--margen) 0;
		display: grid;
		grid-template-columns: 1.2fr 1fr;
		gap: clamp(40px, 5vw, 72px);
		align-items: center;
	}
	.texto {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 24px;
	}
	.pastilla {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		padding: 6px 14px 6px 10px;
		border-radius: 999px;
		background: oklch(1 0 0 / 0.08);
		box-shadow: inset 0 0 0 1px oklch(1 0 0 / 0.14);
		font-size: 13px;
		font-weight: 600;
		letter-spacing: 0.04em;
		color: var(--menta);
	}
	.pastilla i {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: var(--menta);
		box-shadow: 0 0 0 4px oklch(0.77 0.15 160 / 0.25);
		animation: latido 2.4s ease-in-out infinite;
	}
	@keyframes latido {
		50% {
			box-shadow: 0 0 0 7px oklch(0.77 0.15 160 / 0.08);
		}
	}
	.ctas {
		display: flex;
		flex-wrap: wrap;
		gap: 12px;
		margin-top: 4px;
	}
	.foto {
		position: relative;
	}
	.foto img {
		display: block;
		width: 100%;
		height: auto;
		aspect-ratio: 4 / 4.4;
		object-fit: cover;
		object-position: 62% 50%;
		border-radius: 32px;
		box-shadow: 0 50px 90px -30px oklch(0 0 0 / 0.6);
	}

	/* Tarjetitas flotantes de interfaz sobre las fotos. */
	.flotante {
		position: absolute;
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px 18px 12px 12px;
		border-radius: 18px;
		background: oklch(1 0 0 / 0.94);
		backdrop-filter: blur(10px);
		-webkit-backdrop-filter: blur(10px);
		color: var(--tinta);
		box-shadow:
			0 1px 0 oklch(1 0 0 / 0.6) inset,
			0 24px 44px -18px oklch(0 0 0 / 0.45);
		font-size: 15px;
		line-height: 1.3;
	}
	.flotante > span:last-child {
		display: flex;
		flex-direction: column;
	}
	.flotante small {
		font-size: 12.5px;
		color: var(--gris);
	}
	.ic {
		display: grid;
		place-items: center;
		flex: none;
		width: 40px;
		height: 40px;
		border-radius: 12px;
		background: var(--petroleo);
		color: var(--menta);
		font-size: 22px;
	}
	.ic.blanco {
		background: #fff;
		color: var(--petroleo);
		box-shadow: inset 0 0 0 1px oklch(0 0 0 / 0.06);
	}
	.ic img {
		width: 26px;
		height: 26px;
	}
	.turno-nuevo {
		left: clamp(-36px, -3vw, -12px);
		bottom: 32px;
	}
	.sena {
		right: clamp(-28px, -2vw, -10px);
		top: 30px;
	}

	/* Cinta de integraciones, en movimiento continuo. */
	.cinta {
		margin-top: clamp(56px, 7vw, 96px);
		padding: 22px 0;
		border-top: 1px solid oklch(1 0 0 / 0.08);
		overflow: hidden;
		mask: linear-gradient(90deg, transparent, #000 12%, #000 88%, transparent);
		-webkit-mask: linear-gradient(90deg, transparent, #000 12%, #000 88%, transparent);
	}
	.pista {
		display: flex;
		width: max-content;
		animation: correr 32s linear infinite;
	}
	.cinta ul {
		list-style: none;
		display: flex;
		gap: 18px;
		margin: 0;
		padding: 0 9px;
	}
	.cinta li {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 8px 16px 8px 8px;
		border-radius: 999px;
		background: #fff;
		color: var(--tinta);
		box-shadow: 0 10px 24px -14px oklch(0 0 0 / 0.5);
		font-size: 18px;
		font-weight: 650;
		letter-spacing: -0.01em;
		white-space: nowrap;
	}
	.cinta li img {
		width: 26px;
		height: 26px;
		object-fit: contain;
		border-radius: 6px;
	}
	@keyframes correr {
		to {
			transform: translateX(-50%);
		}
	}
	.tiendas {
		display: flex;
		flex-direction: column;
		gap: 10px;
		margin-top: 6px;
	}
	.tiendas > span {
		font-size: 12px;
		font-weight: 700;
		letter-spacing: 0.14em;
		text-transform: uppercase;
		color: oklch(1 0 0 / 0.65);
	}
	.tiendas div {
		display: flex;
		flex-wrap: wrap;
		gap: 10px;
	}
	.tiendas img {
		height: 44px;
		width: auto;
		border-radius: 8px;
		box-shadow: 0 0 0 1px oklch(1 0 0 / 0.18);
	}

	main {
		max-width: var(--ancho);
		margin: 0 auto;
		padding: 0 var(--margen);
	}
	.encabezado {
		display: flex;
		flex-direction: column;
		gap: 14px;
	}
	.encabezado.centrado {
		align-items: center;
		text-align: center;
	}

	/* El sistema, sobre una trama de puntos. */
	.sistema {
		position: relative;
		isolation: isolate;
		padding-top: var(--seccion);
	}
	.puntos-fondo::before {
		content: '';
		position: absolute;
		inset: 40px calc(-1 * var(--margen)) -40px;
		z-index: -1;
		background-image: radial-gradient(oklch(0.4 0.05 210 / 0.18) 1.2px, transparent 1.2px);
		background-size: 22px 22px;
		mask: radial-gradient(60% 60% at 50% 55%, #000 30%, transparent 75%);
		-webkit-mask: radial-gradient(60% 60% at 50% 55%, #000 30%, transparent 75%);
	}
	.maqueta {
		position: relative;
		margin: clamp(40px, 5vw, 64px) auto 0;
		max-width: 1040px;
		padding-bottom: 48px;
	}
	.tel {
		position: absolute;
		right: clamp(-150px, -9vw, -12px);
		bottom: 0;
		width: clamp(140px, 15vw, 200px);
	}
	.recordatorio {
		left: clamp(-60px, -4vw, -10px);
		bottom: 70px;
	}

	/* Bloques alternados con un número y una pieza de interfaz sobre la foto. */
	.bloque {
		display: grid;
		grid-template-columns: 1.1fr 1fr;
		gap: clamp(40px, 7vw, 110px);
		align-items: center;
		padding-top: var(--seccion);
	}
	.bloque.invertido .bloque-foto {
		order: 2;
	}
	.bloque-foto {
		position: relative;
		isolation: isolate;
	}
	.bloque-foto::before {
		content: '';
		position: absolute;
		inset: 10% -5% -6% 8%;
		z-index: -1;
		border-radius: 36px;
		background: var(--menta-suave);
	}
	.bloque.invertido .bloque-foto::before {
		inset: 10% 8% -6% -5%;
		background: #e7eff0;
	}
	.bloque-foto img {
		display: block;
		width: 100%;
		height: auto;
		aspect-ratio: 5 / 4;
		object-fit: cover;
		border-radius: 28px;
		box-shadow: 0 30px 60px -36px oklch(0.25 0.04 210 / 0.55);
	}
	.bloque-foto img.retrato {
		object-position: 50% 30%;
	}
	.bloque-texto {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	.num {
		font-size: 15px;
		font-weight: 700;
		letter-spacing: 0.1em;
		color: var(--menta-oscuro);
	}
	.num::after {
		content: '';
		display: inline-block;
		width: 40px;
		height: 2px;
		margin-left: 12px;
		vertical-align: middle;
		background: var(--menta);
	}
	.bloque-texto p {
		margin: 0;
		font-size: clamp(18px, 1.5vw, 20px);
		line-height: 1.5;
		color: var(--gris);
	}
	.puntos {
		list-style: none;
		margin: 6px 0 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.puntos li {
		display: flex;
		align-items: center;
		gap: 10px;
		font-size: 17px;
		font-weight: 550;
	}
	.puntos .material-symbols-rounded {
		color: var(--menta-oscuro);
		font-size: 22px;
	}
	/* El recordatorio, como un chat. */
	.chat {
		right: clamp(-36px, -3vw, -10px);
		bottom: 22px;
		flex-direction: column;
		align-items: stretch;
		gap: 8px;
		max-width: 280px;
		padding: 12px 14px 14px;
		background: #f1f6f5;
	}
	.chat-de {
		display: flex !important;
		flex-direction: row !important;
		align-items: center;
		gap: 8px;
		font-size: 12.5px;
		font-weight: 650;
	}
	.chat-de img {
		width: 18px;
		height: 18px;
	}
	.chat-de .material-symbols-rounded {
		font-size: 18px;
		color: var(--petroleo);
	}
	.msj {
		padding: 9px 12px;
		border-radius: 14px;
		font-size: 13.5px;
		line-height: 1.35;
		text-wrap: balance;
		box-shadow: 0 1px 1px oklch(0 0 0 / 0.08);
	}
	.msj.vos {
		align-self: flex-start;
		background: #fff;
		border-top-left-radius: 4px;
	}
	.msj.el {
		align-self: flex-end;
		background: #d4f3e5;
		border-top-right-radius: 4px;
	}
	/* La seña cobrada. */
	.tarjeta-sena {
		left: clamp(-40px, -3vw, -10px);
		bottom: 26px;
		flex-direction: column;
		align-items: stretch;
		gap: 6px;
		padding: 16px 18px;
		min-width: 240px;
	}
	.sena-de {
		display: flex !important;
		flex-direction: row !important;
		align-items: center;
		gap: 8px;
		font-size: 12.5px;
		color: var(--gris);
	}
	.sena-de img {
		width: 22px;
		height: 22px;
	}
	.montos {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
	}
	.montos b {
		font-size: 26px;
		letter-spacing: -0.03em;
	}
	.montos em {
		padding: 3px 10px;
		border-radius: 999px;
		background: var(--menta-suave);
		color: var(--menta-oscuro);
		font-style: normal;
		font-size: 13px;
		font-weight: 700;
	}
	/* La ficha de la clienta. */
	.ficha {
		right: clamp(-36px, -3vw, -10px);
		bottom: 26px;
	}
	.avatar {
		display: grid;
		place-items: center;
		flex: none;
		width: 44px;
		height: 44px;
		border-radius: 50%;
		background: var(--menta);
		color: var(--petroleo-hondo);
		font-size: 14px;
		font-weight: 700;
	}
	.ficha em {
		margin-top: 2px;
		font-style: normal;
		font-size: 12.5px;
		font-weight: 650;
		color: var(--menta-oscuro);
	}

	/* Funciones: íconos grandes, sin cajas ni texto descriptivo. */
	.funciones {
		padding-top: var(--seccion);
	}
	.funciones ul {
		list-style: none;
		margin: clamp(44px, 5vw, 68px) 0 0;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(6, 1fr);
		gap: clamp(28px, 4vw, 48px) 16px;
	}
	.funciones li {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 14px;
		font-size: 16px;
		font-weight: 600;
		text-align: center;
		line-height: 1.25;
	}
	.funciones li .material-symbols-rounded {
		display: grid;
		place-items: center;
		width: 68px;
		height: 68px;
		border-radius: 22px;
		background: linear-gradient(160deg, #eefaf4, #dcf3e8);
		box-shadow: inset 0 1px 0 #fff;
		color: var(--petroleo);
		font-size: 30px;
		transition:
			background 0.2s,
			color 0.2s,
			transform 0.2s;
	}
	.funciones li:hover .material-symbols-rounded {
		background: var(--petroleo);
		color: var(--menta);
		transform: translateY(-4px) rotate(-4deg);
	}

	/* Pasos: cada uno es una mini escena de la app, con el número grande y una flecha al siguiente. */
	.pasos {
		/* De punta a punta aunque esté dentro de main. */
		margin: var(--seccion) calc(50% - 50vw) 0;
		padding: clamp(80px, 9vw, 128px) calc(50vw - 50%);
		background:
			radial-gradient(60% 80% at 50% 0%, #fffaf0, transparent 70%),
			var(--crema);
	}
	.pasos ol {
		list-style: none;
		margin: clamp(48px, 6vw, 80px) 0 0;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: clamp(28px, 4vw, 64px);
	}
	.pasos li {
		position: relative;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	/* Flecha al paso siguiente. */
	.pasos li:not(:last-child)::after {
		content: 'arrow_forward';
		position: absolute;
		top: 120px;
		right: calc(-1 * clamp(28px, 4vw, 64px) / 2 - 18px);
		display: grid;
		place-items: center;
		width: 36px;
		height: 36px;
		border-radius: 50%;
		background: #fff;
		box-shadow: 0 8px 20px -8px oklch(0.3 0.05 210 / 0.45);
		font-family: 'Material Symbols Rounded';
		font-size: 20px;
		color: var(--petroleo);
		z-index: 2;
	}
	.escena {
		position: relative;
		display: grid;
		place-items: center;
		aspect-ratio: 5 / 4;
		margin-bottom: 18px;
		border-radius: 28px;
		background:
			radial-gradient(80% 70% at 50% 100%, oklch(0.88 0.08 160 / 0.5), transparent 70%),
			var(--menta-suave);
		overflow: hidden;
	}
	.pasos li:nth-child(2) .escena {
		background:
			radial-gradient(80% 70% at 50% 100%, oklch(0.88 0.08 160 / 0.6), transparent 70%),
			#fff;
	}
	.pasos li:nth-child(3) .escena {
		background:
			radial-gradient(70% 70% at 50% 40%, var(--petroleo-luz), transparent 75%),
			var(--petroleo);
	}
	.mini {
		display: flex;
		flex-direction: column;
		gap: 8px;
		width: 74%;
		padding: 18px;
		border-radius: 20px;
		background: #fff;
		box-shadow:
			0 1px 2px oklch(0 0 0 / 0.06),
			0 30px 50px -24px oklch(0.3 0.05 210 / 0.5);
		font-size: 14px;
		line-height: 1.35;
	}
	.mini b {
		font-size: 15px;
		letter-spacing: -0.01em;
	}
	.ingreso,
	.listo {
		align-items: center;
		text-align: center;
	}
	.mini-icono {
		width: 40px;
		height: 40px;
		border-radius: 11px;
	}
	.boton-mini {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		width: 100%;
		padding: 9px;
		border-radius: 999px;
		box-shadow: inset 0 0 0 1px oklch(0.88 0.01 210);
		font-size: 13px;
		font-weight: 600;
	}
	.boton-mini img {
		width: 16px;
		height: 16px;
	}
	.boton-mini.negro {
		background: #000;
		color: #fff;
		box-shadow: none;
	}
	.boton-mini.negro img {
		filter: invert(1);
	}
	.linea {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 9px 12px;
		border-radius: 12px;
		background: #f2f6f6;
	}
	.linea b {
		font-size: 14px;
	}
	.linea small,
	.subiendo small {
		font-size: 12px;
		color: var(--gris);
	}
	.subiendo {
		display: flex;
		justify-content: space-between;
		margin-top: 4px;
	}
	.barra-carga {
		height: 8px;
		border-radius: 4px;
		background: var(--menta-suave);
		overflow: hidden;
	}
	.barra-carga i {
		display: block;
		height: 100%;
		width: 100%;
		border-radius: 4px;
		background: var(--menta);
		transform-origin: left;
		animation: cargar 3.2s cubic-bezier(0.4, 0, 0.2, 1) infinite;
	}
	@keyframes cargar {
		0% {
			transform: scaleX(0.05);
		}
		70%,
		100% {
			transform: scaleX(1);
		}
	}
	.listo > span:not(.check) {
		font-size: 13px;
		color: var(--gris);
	}
	.listo em {
		font-style: normal;
		font-weight: 700;
		font-size: 14px;
		letter-spacing: -0.01em;
		color: var(--menta-oscuro);
	}
	.check {
		display: grid;
		place-items: center;
		width: 52px;
		height: 52px;
		margin-bottom: 4px;
		border-radius: 50%;
		background: var(--menta);
		color: var(--petroleo-hondo);
		font-size: 30px;
		font-variation-settings:
			'FILL' 1,
			'wght' 700;
		box-shadow: 0 0 0 8px oklch(0.77 0.15 160 / 0.3);
		animation: latir 2.6s ease-in-out infinite;
	}
	@keyframes latir {
		0%,
		100% {
			transform: scale(1);
			box-shadow: 0 0 0 8px oklch(0.77 0.15 160 / 0.3);
		}
		50% {
			transform: scale(1.06);
			box-shadow: 0 0 0 14px oklch(0.77 0.15 160 / 0.08);
		}
	}
	.numero {
		display: flex;
		align-items: center;
		gap: 12px;
		font-size: 48px;
		line-height: 1;
		font-weight: 800;
		letter-spacing: -0.04em;
		color: var(--petroleo);
	}
	.numero::after {
		content: '';
		width: 40px;
		height: 4px;
		border-radius: 2px;
		background: var(--menta);
	}
	.pasos li > b {
		font-size: clamp(24px, 2.4vw, 30px);
		letter-spacing: -0.03em;
	}
	.pasos li > span:last-child {
		font-size: 17px;
		color: #4d4a3f;
	}

	/* Cierre: petróleo de punta a punta, con luz menta, grano y el símbolo. */
	.cierre {
		background:
			radial-gradient(50% 80% at 85% 50%, oklch(0.77 0.15 160 / 0.2), transparent 70%),
			radial-gradient(60% 70% at 0% 0%, var(--petroleo-luz), transparent 70%),
			var(--petroleo-hondo);
		color: #fff;
		overflow: hidden;
	}
	.cierre :global(.simbolo-cierre) {
		width: min(820px, 90vw);
		left: -26%;
		bottom: -40%;
		color: oklch(1 0 0 / 0.035);
	}
	.cierre-cuerpo {
		max-width: var(--ancho);
		margin: 0 auto;
		padding: clamp(72px, 9vw, 120px) var(--margen) clamp(56px, 7vw, 96px);
		display: grid;
		grid-template-columns: 1.15fr 1fr;
		gap: clamp(40px, 6vw, 96px);
		align-items: center;
	}
	.cierre-texto {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 22px;
	}
	.cierre-texto h2 {
		font-size: clamp(38px, 4.6vw, 64px);
	}
	.cierre-texto h2 .acento {
		white-space: nowrap;
	}
	.cierre-texto p {
		margin: 0;
		max-width: 26em;
		font-size: clamp(18px, 1.5vw, 20px);
		line-height: 1.5;
		color: oklch(1 0 0 / 0.78);
	}
	.cierre-visual {
		position: relative;
		display: grid;
		place-items: center;
		min-height: 520px;
	}
	.halo {
		position: absolute;
		width: 380px;
		height: 380px;
		border-radius: 50%;
		background: radial-gradient(circle, oklch(0.77 0.15 160 / 0.4), transparent 65%);
		filter: blur(20px);
	}
	.tel-cierre {
		position: relative;
		width: 250px;
		transform: rotate(-4deg);
	}
	.tel-cierre.flota {
		animation-name: flotar-girado;
	}
	@keyframes flotar-girado {
		0%,
		100% {
			transform: rotate(-4deg) translateY(0);
		}
		50% {
			transform: rotate(-4deg) translateY(-12px);
		}
	}
	.aviso-cierre {
		top: 70px;
		left: 0;
	}
	.resena-cierre {
		bottom: 80px;
		right: 0;
		animation-delay: 1.2s;
	}
	.resena-cierre small {
		color: var(--menta-oscuro);
		letter-spacing: 0.1em;
	}

	/* Pie con columnas. */
	footer {
		background: #f8fbfa;
		color: var(--gris);
		font-size: 15px;
	}
	.pie {
		max-width: var(--ancho);
		margin: 0 auto;
		padding: clamp(48px, 6vw, 72px) var(--margen) 32px;
		display: grid;
		grid-template-columns: 1.6fr repeat(3, 1fr);
		gap: 32px;
	}
	.pie-marca .marca {
		color: var(--tinta);
	}
	.pie-marca p {
		margin: 12px 0 0;
		max-width: 18em;
	}
	.pie nav {
		max-width: none;
		height: auto;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		justify-content: flex-start;
		gap: 10px;
	}
	.pie nav b {
		margin-bottom: 4px;
		font-size: 13px;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: var(--tinta);
	}
	.pie a {
		color: var(--gris);
		text-decoration: none;
	}
	.pie a:hover {
		color: var(--petroleo);
		text-decoration: underline;
	}
	.pie-final {
		width: min(calc(var(--ancho) - 2 * var(--margen)), calc(100% - 2 * var(--margen)));
		margin: 0 auto;
		padding: 20px 0 32px;
		font-size: 13px;
		border-top: 1px solid oklch(0.92 0.005 210);
	}

	.barra {
		display: none;
	}

	/* Movimiento: lo que entra en pantalla aparece; lo que flota, flota. */
	:global(.aparece) {
		transition:
			opacity 0.8s cubic-bezier(0.2, 0.7, 0.2, 1),
			transform 0.8s cubic-bezier(0.2, 0.7, 0.2, 1);
	}
	:global(.aparece.oculto) {
		opacity: 0;
		transform: translateY(32px);
	}
	.flota {
		animation: flotar 5s ease-in-out infinite;
	}
	.flota.lento {
		animation-duration: 7s;
	}
	@keyframes flotar {
		50% {
			transform: translateY(-10px);
		}
	}
	.entra {
		animation: subir 0.8s cubic-bezier(0.2, 0.7, 0.2, 1) both;
		animation-delay: var(--d, 0ms);
	}
	@keyframes subir {
		from {
			opacity: 0;
			transform: translateY(18px);
		}
	}

	@media (max-width: 980px) {
		.hero-cuerpo,
		.bloque,
		.cierre-cuerpo {
			grid-template-columns: 1fr;
		}
		.pie {
			grid-template-columns: repeat(3, 1fr);
		}
		.pie-marca {
			grid-column: 1 / -1;
		}
		.bloque.invertido .bloque-foto {
			order: 0;
		}
		.funciones ul {
			grid-template-columns: repeat(4, 1fr);
		}
	}
	@media (max-width: 640px) {
		.acciones .cta {
			display: none;
		}
		.foto img {
			aspect-ratio: 4 / 3;
			border-radius: 22px;
		}
		.turno-nuevo {
			left: 12px;
			bottom: 12px;
			padding: 8px 14px 8px 8px;
			font-size: 14px;
		}
		.sena {
			right: 12px;
			top: 12px;
			padding: 8px 14px 8px 8px;
			font-size: 14px;
		}
		.ic {
			width: 32px;
			height: 32px;
			font-size: 18px;
		}
		.ic img {
			width: 20px;
			height: 20px;
		}
		.tel {
			right: -6px;
			width: 104px;
		}
		.recordatorio {
			display: none;
		}
		.bloque-foto img {
			border-radius: 22px;
		}
		.tarjeta-sena,
		.chat,
		.ficha {
			left: 12px;
			right: 12px;
			bottom: -24px;
			min-width: 0;
			max-width: none;
		}
		.chat .msj.el {
			display: none;
		}
		.bloque {
			padding-top: 96px;
		}
		.bloque-texto {
			padding-top: 16px;
		}
		.funciones ul {
			grid-template-columns: repeat(3, 1fr);
		}
		.funciones li {
			font-size: 13.5px;
			white-space: nowrap;
		}
		.pasos ol {
			grid-template-columns: 1fr;
			gap: 64px;
		}
		.pasos li:not(:last-child)::after {
			top: auto;
			bottom: -50px;
			right: 50%;
			transform: translateX(50%) rotate(90deg);
		}
		.cierre-visual {
			min-height: 420px;
		}
		.tel-cierre {
			width: 200px;
		}
		.aviso-cierre {
			top: 30px;
		}
		.resena-cierre {
			bottom: 40px;
		}
		.pie {
			grid-template-columns: 1fr 1fr;
		}
		.pie-final {
			padding-bottom: 104px;
		}
		/* Los cuatro botones, centrados: los dos llamados a lo ancho y los badges en una fila. */
		.texto,
		.cierre-texto {
			min-width: 0;
		}
		.ctas {
			flex-direction: column;
			align-self: stretch;
			width: 100%;
		}
		.ctas .cta {
			width: 100%;
		}
		.tiendas {
			align-self: stretch;
			align-items: center;
		}
		.tiendas div {
			justify-content: center;
		}
		.tiendas img {
			height: 48px;
		}
		/* En el celular, el llamado queda siempre a mano (cuando ya no se ve el del hero). */
		.barra {
			display: flex;
			opacity: 0;
			transform: translateY(24px);
			pointer-events: none;
			transition:
				opacity 0.25s,
				transform 0.25s;
			position: fixed;
			left: 16px;
			right: 16px;
			bottom: max(16px, env(safe-area-inset-bottom));
			z-index: 60;
			box-shadow: 0 14px 30px -10px oklch(0 0 0 / 0.45);
		}
		.barra.visible {
			opacity: 1;
			transform: none;
			pointer-events: auto;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.barra-carga i,
		.check,
		.entra,
		.flota,
		.pista,
		.pastilla i {
			animation: none;
		}
		.cta,
		.cta .flecha,
		.barra,
		.funciones li .material-symbols-rounded {
			transition: none;
		}
		.cta:hover,
		.funciones li:hover .material-symbols-rounded {
			transform: none;
		}
	}
</style>
