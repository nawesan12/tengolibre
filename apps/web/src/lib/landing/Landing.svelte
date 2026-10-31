<!--
	Landing de Tengo Libre (la ruta /), fiel a «Landing y alta v3» del handoff v2. Las apps nativas
	todavía no están publicadas: van los badges oficiales con «Próximamente» aparte (sin link) y el QR
	es real (lleva a esta página). Fotos en static/fotos/FUENTES.md, logos en static/logos/FUENTES.md.
	Las clases evitan los nombres globales de app.css (.boton, .tarjeta, .campo…).
-->
<script lang="ts">
	import { page } from '$app/state';
	import Icono from '#lib/ui/Icono.svelte';
	import Qr from '#lib/alta/Qr.svelte';
	import { Direccion } from '#lib/alta/direccion.svelte.ts';
	import { aSlug } from '#lib/alta/slug.ts';
	import { pesos } from '#lib/alta/rubros.ts';
	import { aparecer } from './aparecer.ts';

	const sitio = $derived(page.url.origin);
	const host = $derived(page.url.host);

	// Barra flotante: aparece al pasar 760 px de scroll.
	let scrollY = $state(0);
	const barra = $derived(scrollY > 760);

	// Calculadora: turnos perdidos por semana × precio × 4,3 semanas.
	let perdidos = $state(4);
	let precio = $state(12000);
	const total = $derived(pesos(perdidos * precio * 4.3));

	// Cierre: la dirección se valida al instante.
	const dir = new Direccion();
	let escrito = $state('');
	$effect(() => dir.poner(aSlug(escrito)));
	const altaCon = $derived(dir.slug ? `/empezar?slug=${encodeURIComponent(dir.slug)}` : '/empezar');
	const mensajeCierre = $derived(
		!dir.slug
			? 'Escribí el nombre de tu negocio'
			: dir.campo.estado === 'libre'
				? `${host}/${dir.slug} está libre.`
				: dir.campo.mensaje.replace(' Probá una de estas:', '')
	);

	type Talon = { h: string; n: string; s: string; ic: string; k?: 'hueco' | 'nuevo' | 'osc' };
	const tira: Talon[] = [
		{ h: '09:30', n: 'Lucía · Corte', s: 'Confirmado', ic: 'check_circle' },
		{ h: '10:15', n: 'Libre 45 min', s: 'Avisando a la lista', ic: 'campaign', k: 'hueco' },
		{ h: '11:00', n: 'Martina · Lifting', s: 'Seña pagada', ic: 'paid', k: 'nuevo' },
		{ h: '12:30', n: 'Don Tito · Barba', s: 'Confirmado', ic: 'check_circle', k: 'osc' },
		{ h: '14:00', n: 'Paula · Uñas', s: 'Recordatorio enviado', ic: 'notifications' },
		{ h: '15:30', n: 'Libre 1 h 30', s: '2 personas esperan', ic: 'hourglass_top', k: 'hueco' },
		{ h: '16:00', n: 'Ana · Color', s: 'Reservó desde Instagram', ic: 'bolt', k: 'nuevo' },
		{ h: '17:30', n: 'Sofi · Masaje', s: 'Socia · 3 de 8', ic: 'workspace_premium', k: 'osc' }
	];

	// Logos reales de cada integración (static/logos).
	const integraciones = [
		['mercadopago.svg', 'Mercado Pago'],
		['whatsapp.svg', 'WhatsApp'],
		['google.svg', 'Google'],
		['googlecalendar.svg', 'Google Calendar'],
		['apple.svg', 'Apple']
	];

	const appPuntos = [
		['notifications_active', 'Te avisa cada turno nuevo'],
		['swipe', 'Movés turnos con un dedo'],
		['point_of_sale', 'La caja del día, a mano']
	];
	const pushes = [
		{ t: 'ahora', h: 'Nuevo turno · sáb 11:00', s: 'Martina, lifting. Seña de $ 6.600 pagada.', a: '.96', k: '1' },
		{ t: 'hace 6 min', h: 'Se llenó el hueco de las 15:30', s: 'Ana lo tomó desde la lista de espera.', a: '.86', k: '.97' },
		{ t: '9:00', h: 'Hoy tenés 12 turnos', s: 'Te queda 1 h 30 libre con Mica.', a: '.72', k: '.94' }
	];

	const plantillas = [
		{ n: 'Carta', nom: 'Lumi', bg: '#7a3b5e', tc: '#fff', ac: '#f0d2dc', fh: '52%', foto: 'manicura.jpg' },
		{ n: 'Vidriera', nom: 'Don Tito', bg: '#1f2a24', tc: '#f5c518', ac: '#f5c518', fh: '62%', foto: 'barbero-cortando.jpg' },
		{ n: 'Sobria', nom: 'Estudio Norte', bg: '#fffdf9', tc: '#0f2328', ac: '#24527a', fh: '0%', foto: 'recepcion.jpg' },
		{ n: 'Galería', nom: 'Bruma', bg: '#2f5d50', tc: '#fff', ac: '#cfe5d9', fh: '70%', foto: 'peluquera-con-clienta.jpg' }
	];

	const rubros = [
		{ n: 'Peluquerías', s: 'Color, corte y turnos largos', foto: 'peluquera-con-clienta.jpg', r: 'peluqueria' },
		{ n: 'Barberías', s: 'Turnos de 30 minutos sin huecos', foto: 'barbero-cortando.jpg', r: 'barberia' },
		{ n: 'Estética y uñas', s: 'Packs, socias y seña', foto: 'manicura.jpg', r: 'estetica' },
		{ n: 'Consultorios', s: 'Recepción y varios profesionales', foto: 'recepcion.jpg', r: 'consultorio' }
	];

	const precios = [
		{
			n: 'Inicial',
			precio: '$\u00a07.000',
			para: 'Para quien atiende solo o con una persona más.',
			inc: ['Página y turnos online', 'App para el celular', 'Recordatorios por mail', 'Hasta 2 profesionales'],
			clase: 'crema'
		},
		{
			n: 'Completo',
			precio: '$\u00a015.000',
			para: 'Para equipos que quieren llenar cada hueco.',
			inc: [
				'Todo lo del Inicial',
				'Seña con Mercado Pago',
				'Lista de espera que se ofrece sola',
				'Recordatorios por WhatsApp',
				'Membresías, cupones y referidos',
				'Sin límite de profesionales'
			],
			clase: 'oscuro',
			tag: 'EL MÁS ELEGIDO'
		}
	];

	const faq = [
		['¿Mis clientes tienen que bajarse una\u00a0app?', 'No. Reservan desde tu página en el navegador, sin crear cuenta. Les llega el comprobante por mail o WhatsApp.'],
		['¿Cómo cobro la\u00a0seña?', 'Conectás tu cuenta de Mercado Pago en un paso. La plata va directo a tu cuenta; nosotros no la tocamos.'],
		['¿Qué pasa cuando termina la\u00a0prueba?', 'Elegís un plan o la agenda queda en sólo lectura. No se borra nada y no te cobramos sin avisar.'],
		['¿Puedo usar mi propio\u00a0dominio?', 'Sí, en el plan Completo podés usar turnos.tunegocio.com.ar o el dominio que ya tengas.']
	];

	const pie = [
		{ t: 'PRODUCTO', l: [['Cómo funciona', '#como'], ['Precios', '#precios'], ['La app', '#app'], ['Plantillas', '#todo'], ['Integraciones', '#todo']] },
		{
			t: 'RUBROS',
			l: [
				['Peluquerías', '/empezar?rubro=peluqueria'],
				['Barberías', '/empezar?rubro=barberia'],
				['Estética', '/empezar?rubro=estetica'],
				['Uñas', '/empezar?rubro=unias'],
				['Consultorios', '/empezar?rubro=consultorio']
			]
		},
		{ t: 'AYUDA', l: [['Centro de ayuda', 'mailto:hola@tengolibre.com.ar'], ['Buscar un turno', '/explorar'], ['Ingresar', '/ingresar']] },
		{
			t: 'EMPRESA',
			l: [
				['Quiénes somos', 'https://grupovimo.com.ar'],
				['Trabajá con nosotros', 'mailto:hola@tengolibre.com.ar?subject=Trabajar%20en%20Tengo%20Libre'],
				['Prensa', 'mailto:hola@tengolibre.com.ar?subject=Prensa']
			]
		}
	];
	const anio = new Date().getFullYear();
</script>

<svelte:window bind:scrollY />

<svelte:head>
	<meta property="og:title" content="Tengo Libre · Llená cada hueco" />
	<meta property="og:description" content="Turnos online para salones, barberías y consultorios. Seña con Mercado Pago." />
	<meta property="og:image" content="{sitio}/og/inicio.png" />
	<meta property="og:image:width" content="1200" />
	<meta property="og:image:height" content="630" />
	<meta property="og:url" content={sitio} />
	<meta name="twitter:card" content="summary_large_image" />
</svelte:head>

<!-- Las apps todavía no están: badges oficiales, sin link, con «Próximamente» como texto aparte. -->
{#snippet tiendas(clase = '')}
	<div class="tiendas {clase}">
		<span class="proximamente"><Icono nombre="schedule" lleno tam={16} />Próximamente</span>
		<div>
			<img src="/logos/appstore.svg" alt="App Store, próximamente" width="168" height="56" />
			<img src="/logos/googleplay.png" alt="Google Play, próximamente" width="188" height="56" />
		</div>
	</div>
{/snippet}

<div class="landing">
	<header class="cabecera">
		<nav>
			<a href="/" class="logo" aria-label="Tengo Libre, inicio"
				><img src="/marca/logotipo_horizontal_negativo.svg" alt="Tengo Libre" height="28" /></a
			>
			<span class="enlaces">
				<a href="#como">Cómo funciona</a>
				<a href="#precios">Precios</a>
				<a href="#app"><Icono nombre="smartphone" lleno tam={18} />Bajá la app</a>
				<a href="/ingresar">Ingresar</a>
			</span>
			<a class="pastilla menta chica" href="/empezar">Empezar gratis</a>
		</nav>

		<div class="hero-marco">
			<div class="hero-caja">
				<a class="pestana" href="/empezar">
					<span class="rotulo">7 DÍAS GRATIS · SIN TARJETA</span>
					<span class="pestana-cta">Empezar ahora<Icono nombre="arrow_forward" tam={26} /></span>
				</a>
				<div class="hero">
					<h1>Llená cada <span class="subrayado"><span class="raya" aria-hidden="true"></span><span class="palabra">hueco.</span></span></h1>
					<p class="bajada">Tus clientes reservan y pagan la seña solos. Vos&nbsp;atendés.</p>
					{@render tiendas('centradas')}
					<div class="marquesina" aria-hidden="true">
						<div class="tira">
							{#each [...tira, ...tira] as t, i (i)}
								<span class="tal {t.k ?? 'claro'}">
									<b class="tal-h">{t.h}</b>
									<span class="tal-c">
										<b>{t.n}</b>
										<span class="tal-s"><Icono nombre={t.ic} lleno tam={15} />{t.s}</span>
									</span>
								</span>
							{/each}
						</div>
					</div>
				</div>
			</div>
			<div class="funciona">
				<span class="rotulo">FUNCIONA CON</span>
				{#each integraciones as [logo, nombre] (nombre)}
					<span class="integ"><span class="integ-logo"><img src="/logos/{logo}" alt="" width="22" height="22" /></span>{nombre}</span>
				{/each}
			</div>
		</div>
	</header>
	<div class="troquel" aria-hidden="true"><span></span><span></span><span></span></div>

	<section id="como" class="seccion">
		<div class="titulo-fila" use:aparecer>
			<div class="titulo-col">
				<span class="rotulo verde">CÓMO FUNCIONA</span>
				<h2 class="h2">Tres pasos. La agenda se ocupa&nbsp;sola.</h2>
			</div>
			<a class="pastilla hondo grande" href="/empezar">Crear mi página<Icono nombre="arrow_forward" tam={22} /></a>
		</div>
		<div class="pasos">
			<article class="paso crema-puntos" use:aparecer>
				<span class="numero">01</span>
				<div class="paso-txt">
					<h3>Compartís tu&nbsp;link</h3>
					<p>En Instagram, en WhatsApp y con un QR en el&nbsp;mostrador.</p>
				</div>
				<div class="link-qr">
					<span class="qr-chico"><Qr texto={sitio} color="#183c44" fondo="#fffdf9" etiqueta="Código QR de Tengo Libre" /></span>
					<span class="link-qr-txt"><span class="rotulo verde chico">TU PÁGINA</span><b>{host}/lumi</b></span>
				</div>
			</article>
			<article class="paso petroleo" use:aparecer={80}>
				<span class="numero">02</span>
				<div class="paso-txt">
					<h3>Reservan y pagan la&nbsp;seña</h3>
					<p>Sin crear cuenta, en menos de un&nbsp;minuto.</p>
				</div>
				<div class="talon-sena">
					<span class="ts-h"><span class="rotulo verde chico">SÁB 7</span><b>11:00</b></span>
					<span class="ts-c"><b>Lifting con Abril</b><span class="ok"><Icono nombre="check_circle" lleno tam={16} />Seña de $ 6.600 pagada</span></span>
				</div>
			</article>
			<article class="paso menta" use:aparecer={160}>
				<span class="numero">03</span>
				<div class="paso-txt">
					<h3>Si cancelan, se llena&nbsp;solo</h3>
					<p>Le ofrecemos el lugar a quien espera. Vos no escribís a&nbsp;nadie.</p>
				</div>
				<div class="se-lleno">
					<span class="liberado"><b>15:30</b><span>Se liberó con Mica</span></span>
					<span class="tomado"><img src="/logos/whatsapp.svg" alt="" width="22" height="22" /><span><b>Ana lo tomó</b> en 6 minutos</span></span>
				</div>
			</article>
		</div>
	</section>

	<section id="app" class="seccion">
		<div class="app" use:aparecer>
			<img class="simbolo-fondo" src="/marca/simbolo_una_tinta_blanco.svg" alt="" aria-hidden="true" />
			<div class="app-txt">
				<span class="rotulo menta-txt"><Icono nombre="smartphone" lleno tam={18} />LA APP</span>
				<h2 class="h2 app-h2">Tu agenda, en el&nbsp;bolsillo.</h2>
				<div class="app-puntos">
					{#each appPuntos as [ic, t] (t)}
						<span><span class="ic-caja"><Icono nombre={ic} lleno tam={20} /></span>{t}</span>
					{/each}
				</div>
				<div class="app-tiendas">
					{@render tiendas('sobre-oscuro')}
					<span class="qr-app">
						<span class="qr-56"><Qr texto={sitio} color="#102b31" etiqueta="Código QR para abrir Tengo Libre en el teléfono" /></span>
						<span>Abrila en<br />tu teléfono</span>
					</span>
				</div>
			</div>
			<div class="telefono-col" aria-hidden="true">
				<img class="icono-app" src="/marca-icono.png" alt="" />
				<div class="telefono">
					<div class="pantalla">
						<span class="isla"></span>
						<span class="fecha">lunes 2 de noviembre</span>
						<b class="hora">10:12</b>
						<div class="pushes">
							{#each pushes as p (p.h)}
								<div class="push" style:background="rgba(255,253,249,{p.a})" style:transform="scale({p.k})">
									<img src="/marca-icono.png" alt="" width="34" height="34" />
									<span class="push-c">
										<span class="push-t"><span>TENGO LIBRE</span><span>{p.t}</span></span>
										<b>{p.h}</b>
										<span class="push-s">{p.s}</span>
									</span>
								</div>
							{/each}
						</div>
					</div>
				</div>
			</div>
		</div>
	</section>

	<section class="seccion">
		<div class="calc" use:aparecer>
			<div class="calc-foto">
				<img src="/fotos/peluquera-con-clienta.jpg" alt="Una peluquera atiende a una clienta" loading="lazy" />
				<span class="silla"><b>15:30</b><span>Silla vacía</span></span>
			</div>
			<div class="calc-txt">
				<span class="rotulo verde">HACÉ LA CUENTA</span>
				<h2 class="h2-calc">¿Cuánto te cuestan los&nbsp;huecos?</h2>
				<label class="rango">
					<span><span>Turnos perdidos por semana</span><b>{perdidos}</b></span>
					<input type="range" min="1" max="15" bind:value={perdidos} aria-label="Turnos perdidos por semana" />
				</label>
				<label class="rango">
					<span><span>Precio promedio</span><b>{pesos(precio)}</b></span>
					<input type="range" min="4000" max="40000" step="1000" bind:value={precio} aria-label="Precio promedio del turno" />
				</label>
				<div class="total">
					<span class="total-c"><span>Por mes se te escapan</span><b aria-live="polite">{total}</b></span>
					<a class="pastilla menta media" href="/empezar">Recuperalos<Icono nombre="arrow_forward" tam={20} /></a>
				</div>
			</div>
		</div>
	</section>

	<section id="todo" class="seccion">
		<div class="titulo-col" use:aparecer>
			<span class="rotulo verde">TODO EN UN LUGAR</span>
			<h2 class="h2">Chau libreta, Excel y mil&nbsp;audios.</h2>
		</div>
		<div class="bento">
			<article class="b-card crema-grano ancha" use:aparecer>
				<span class="b-tit">
					<h3>Tu página, con tu&nbsp;marca</h3>
					<span>Seis armados y tus colores. Se ve como vos, no como&nbsp;nosotros.</span>
				</span>
				<div class="miniaturas">
					{#each plantillas as p (p.n)}
						<div class="mini">
							<span class="mini-hoja" style:background={p.bg}>
								<span class="mini-foto" style:height={p.fh} style:background-image="url(/fotos/{p.foto})"></span>
								<span class="mini-pie"><b style:color={p.tc}>{p.nom}</b><span style:background={p.ac}></span></span>
							</span>
							<span class="mini-n">{p.n}</span>
						</div>
					{/each}
				</div>
			</article>
			<article class="b-card petroleo-brillo" use:aparecer={80}>
				<Icono nombre="auto_awesome" lleno tam={34} />
				<h3>Preguntale a tu&nbsp;asistente</h3>
				<span class="b-txt">Conectás Claude, ChatGPT o Gemini a tu&nbsp;agenda.</span>
				<div class="chat">
					<span class="yo">¿Qué tengo libre mañana?</span>
					<span class="ia">Mica, de 15:30 a 17:00. Hay 2 personas&nbsp;esperando.</span>
				</div>
			</article>
			<article class="b-card papel" use:aparecer>
				<Icono nombre="notifications_active" lleno tam={34} />
				<h3>Recordatorios&nbsp;solos</h3>
				<span class="b-txt">Le llega el aviso con un botón para confirmar o&nbsp;mover.</span>
				<div class="wa"><img src="/logos/whatsapp.svg" alt="" width="26" height="26" /><span><b>Mañana 11:00 con Abril.</b> ¿Venís?</span></div>
			</article>
			<article class="b-card papel" use:aparecer={80}>
				<Icono nombre="workspace_premium" lleno tam={34} />
				<h3>Socias y&nbsp;cupones</h3>
				<span class="b-txt">Packs de visitas, descuentos y referidas que traen&nbsp;clientas.</span>
				<div class="pack"><b>3 de 8</b><span>visitas del pack de&nbsp;Sofi</span></div>
			</article>
			<article class="b-card menta-grano" use:aparecer={160}>
				<Icono nombre="insert_chart" lleno tam={34} />
				<h3>Un reporte cada&nbsp;mes</h3>
				<span class="b-txt">Qué facturaste, dónde quedan huecos y cómo&nbsp;llenarlos.</span>
				<a class="pastilla hondo chica ejemplo" href="/panel/reportes">Ver un ejemplo<Icono nombre="arrow_forward" tam={20} /></a>
			</article>
		</div>
		<div class="centro">
			<a class="pastilla menta grande sombra" href="/empezar">Probalo 7 días gratis<Icono nombre="arrow_forward" tam={22} /></a>
		</div>
	</section>

	<section class="seccion rubros-sec">
		<h2 class="h2-chico" use:aparecer>Hecho para tu&nbsp;rubro</h2>
		<div class="rubros">
			{#each rubros as r, i (r.n)}
				<a class="rubro" href="/empezar?rubro={r.r}" style:--foto="url(/fotos/{r.foto})" use:aparecer={i * 60}>
					<span class="rubro-c">
						<b>{r.n}</b>
						<span>{r.s}</span>
						<span class="rubro-cta">Empezar<Icono nombre="arrow_forward" tam={18} /></span>
					</span>
				</a>
			{/each}
		</div>
	</section>

	<section id="precios" class="seccion">
		<div class="titulo-centro" use:aparecer>
			<span class="rotulo verde">PRECIOS</span>
			<h2 class="h2">7 días con todo. Después&nbsp;elegís.</h2>
			<span class="sub">Sin permanencia. Cancelás cuando&nbsp;quieras.</span>
		</div>
		<div class="planes">
			{#each precios as p, i (p.n)}
				<div class="plan {p.clase}" use:aparecer={i * 80}>
					<div class="plan-arriba">
						<span class="plan-n"><b>{p.n}</b>{#if p.tag}<span class="tag">{p.tag}</span>{/if}</span>
						<span class="plan-precio"><b>{p.precio}</b><span>por mes</span></span>
						<span class="plan-para">{p.para}</span>
					</div>
					<div class="plan-troquel" aria-hidden="true"><span></span><span></span><span></span></div>
					<div class="plan-abajo">
						<ul>
							{#each p.inc as x (x)}<li><Icono nombre="check_circle" lleno tam={20} />{x}</li>{/each}
						</ul>
						<a class="plan-cta" href="/empezar">Empezar gratis</a>
					</div>
				</div>
			{/each}
		</div>
	</section>

	<section class="seccion faq">
		<h2 class="h2-chico">Preguntas&nbsp;frecuentes</h2>
		{#each faq as [q, a] (q)}
			<details>
				<summary><span>{q}</span><Icono nombre="add_circle" lleno tam={26} /></summary>
				<p>{a}</p>
			</details>
		{/each}
	</section>

	<section class="seccion cierre-sec">
		<div class="cierre-marco" use:aparecer>
			<span class="cierre-pestana" aria-hidden="true"><img src="/marca/simbolo_una_tinta_blanco.svg" alt="" width="54" height="54" /></span>
			<div class="cierre">
				<h2>Tu dirección todavía está&nbsp;libre.</h2>
				<form class="cierre-form" method="GET" action="/empezar" onsubmit={(e) => { e.preventDefault(); location.href = altaCon; }}>
					<label class="cierre-campo" style:--borde={dir.slug && !dir.campo.ok && dir.campo.estado !== 'buscando' ? '#a3241a' : '#fffdf9'}>
						<span class="prefijo">{host}/</span>
						<input bind:value={escrito} name="slug" aria-label="Tu dirección" placeholder="tu-negocio" autocomplete="off" autocapitalize="none" spellcheck="false" />
						<button class="reservarla">Reservarla</button>
					</label>
					<span class="cierre-msg" role="status"><Icono nombre={dir.slug ? dir.campo.icono : 'edit'} lleno tam={20} />{mensajeCierre}</span>
					<button class="reservarla-m">Reservarla y empezar</button>
				</form>
			</div>
		</div>
	</section>

	<footer class="pie">
		<div class="pie-in">
			<div class="pie-arriba">
				<div class="pie-tit">
					<h2>¿Tenés un hueco? <span>Llenalo.</span></h2>
					<span>7 días gratis, sin tarjeta. En 5 minutos tenés tu&nbsp;página.</span>
				</div>
				<div class="pie-cta">
					<a class="pastilla menta grande sombra" href="/empezar">Empezar gratis<Icono nombre="arrow_forward" tam={22} /></a>
					{@render tiendas('sobre-oscuro')}
				</div>
			</div>
			<div class="pie-cols">
				{#each pie as c (c.t)}
					<span class="pie-col">
						<b>{c.t}</b>
						{#each c.l as [n, href] (n)}<a {href}>{n}</a>{/each}
					</span>
				{/each}
				<span class="pie-col">
					<b>CONTACTO</b>
					<span class="redes"><a href="mailto:hola@tengolibre.com.ar">Escribinos</a><a href="https://grupovimo.com.ar">Grupo Vimo</a></span>
					<span>hola@tengolibre.com.ar</span>
				</span>
			</div>
			<div class="legal">
				<span>© {anio} Tengo Libre · Hecho en Argentina</span>
				<span><a href="https://www.argentina.gob.ar/produccion/defensadelconsumidor">Defensa del consumidor</a></span>
			</div>
		</div>
		<img class="logo-gigante" src="/marca/logotipo_horizontal_negativo.svg" alt="" aria-hidden="true" />
	</footer>

	<div class="barra" class:visible={barra} aria-hidden={!barra}>
		<img src="/marca/simbolo_una_tinta_blanco.svg" alt="" width="22" height="22" />
		<span>7 días gratis, sin tarjeta</span>
		<a class="pastilla menta media" href="/empezar" tabindex={barra ? 0 : -1}>Empezar</a>
	</div>
</div>

<style>
	.landing {
		overflow-x: clip;
		background: var(--tl-papel);
		color: var(--tl-tinta);
		--lado: clamp(20px, 4vw, 48px);
		--sep: clamp(64px, 9vw, 120px);
	}
	.landing :global(h1),
	.landing :global(h2),
	.landing :global(h3) {
		text-wrap: balance;
		margin: 0;
		font-weight: 900;
	}
	.landing :global(p) {
		text-wrap: pretty;
		margin: 0;
	}
	.landing :global(a) {
		text-decoration: none;
	}
	.rotulo {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		font-size: 13px;
		font-weight: 900;
		letter-spacing: 0.08em;
	}
	.rotulo.chico {
		font-size: 11px;
	}
	.verde {
		color: var(--tl-menta-texto);
	}
	.menta-txt {
		color: var(--tl-menta);
	}

	/* Pastillas de acción */
	.pastilla {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 10px;
		border: 0;
		border-radius: 999px;
		font: 900 15px var(--tl-font);
		white-space: nowrap;
		cursor: pointer;
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.pastilla:hover {
		transform: translateY(-2px);
	}
	.pastilla:active {
		transform: scale(0.96);
	}
	.pastilla.menta {
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
	}
	.pastilla.hondo {
		background: var(--tl-petroleo-hondo);
		color: #fff;
	}
	.pastilla.chica {
		height: 48px;
		padding: 0 22px;
	}
	.pastilla.media {
		height: 52px;
		padding: 0 22px;
		font-size: 16px;
		gap: 8px;
	}
	.pastilla.grande {
		height: 60px;
		padding: 0 30px;
		font-size: 18px;
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.4), 0 16px 32px -14px rgba(15, 35, 40, 0.8);
	}
	.pastilla.menta.sombra {
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.4), 0 16px 32px -14px rgba(60, 204, 144, 0.9);
	}

	/* Tiendas: badges oficiales sin tocar + «Próximamente» aparte */
	.tiendas {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.tiendas.centradas {
		align-items: center;
		margin-top: 28px;
	}
	.tiendas > div {
		display: flex;
		flex-wrap: wrap;
		gap: 12px;
	}
	.tiendas.centradas > div {
		justify-content: center;
	}
	.tiendas img {
		height: 56px;
		width: auto;
		display: block;
	}
	.proximamente {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		font-size: 12px;
		font-weight: 900;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--tl-sub-2);
	}
	.sobre-oscuro .proximamente {
		color: rgba(255, 255, 255, 0.8);
	}
	.sobre-oscuro img {
		border-radius: 10px;
		box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.2);
	}

	/* Cabecera */
	.cabecera {
		position: relative;
		background-color: var(--tl-petroleo);
		background-image: var(--tl-grano), radial-gradient(50% 60% at 50% 0%, rgba(60, 204, 144, 0.22), transparent 70%);
		color: #fff;
		padding-bottom: clamp(40px, 6vw, 72px);
	}
	nav {
		position: relative;
		z-index: 3;
		max-width: 1320px;
		margin: 0 auto;
		padding: 0 var(--lado);
		height: 84px;
		display: flex;
		align-items: center;
		gap: 28px;
	}
	.logo {
		flex: 1;
		display: flex;
	}
	.logo img {
		height: 28px;
		width: auto;
	}
	.enlaces {
		display: flex;
		gap: 28px;
		font-size: 15px;
		font-weight: 700;
	}
	.enlaces a {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		color: rgba(255, 255, 255, 0.85);
	}
	.enlaces a:hover {
		color: #fff;
	}
	.enlaces :global(.icono) {
		color: var(--tl-menta);
	}
	.hero-marco {
		position: relative;
		max-width: 1320px;
		margin: 0 auto;
		padding: clamp(16px, 3vw, 32px) clamp(12px, 3vw, 40px) 0;
	}
	.hero-caja {
		position: relative;
		padding-top: clamp(44px, 5vw, 58px);
	}
	.pestana {
		position: absolute;
		left: 50%;
		top: 0;
		z-index: 3;
		transform: translateX(-50%);
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 2px;
		width: clamp(220px, 26vw, 320px);
		height: clamp(88px, 9vw, 116px);
		border-radius: clamp(22px, 2.4vw, 32px);
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		box-shadow:
			0 0 0 clamp(10px, 1.2vw, 16px) var(--tl-petroleo),
			0 30px 50px -10px rgba(0, 0, 0, 0.45),
			inset 0 1px 0 rgba(255, 255, 255, 0.5);
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.pestana:hover {
		transform: translateX(-50%) translateY(-3px);
	}
	.pestana .rotulo {
		font-size: 12px;
	}
	.pestana-cta {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		font-size: clamp(20px, 2.2vw, 26px);
		font-weight: 900;
		letter-spacing: -0.02em;
	}
	.hero {
		position: relative;
		overflow: hidden;
		border-radius: clamp(32px, 4.4vw, 64px);
		background-color: var(--tl-crema);
		background-image: var(--tl-grano-osc), radial-gradient(rgba(24, 60, 68, 0.12) 1.2px, transparent 1.5px);
		background-size: auto, 20px 20px;
		color: var(--tl-tinta);
		padding: clamp(88px, 10vw, 140px) clamp(20px, 4vw, 56px) 0;
		text-align: center;
		box-shadow: inset 0 2px 0 rgba(255, 255, 255, 0.7), 0 60px 120px -60px rgba(0, 0, 0, 0.8);
	}
	h1 {
		margin: 0 auto;
		max-width: 9.5em;
		font-size: clamp(58px, 11vw, 176px);
		line-height: 0.86;
		letter-spacing: -0.06em;
	}
	.subrayado {
		position: relative;
		white-space: nowrap;
	}
	.raya {
		position: absolute;
		left: -0.06em;
		right: -0.06em;
		top: 0.52em;
		height: 0.36em;
		border-radius: 0.12em;
		background: repeating-linear-gradient(-45deg, #3ccc90 0 4px, rgba(60, 204, 144, 0.35) 4px 12px);
		background-size: 34px 34px;
		animation: tl-raya 2.4s linear infinite;
	}
	.palabra {
		position: relative;
	}
	.bajada {
		margin: clamp(20px, 2.4vw, 32px) auto 0 !important;
		max-width: 26em;
		font-size: clamp(18px, 1.7vw, 22px);
		line-height: 1.45;
		color: var(--tl-sub-2);
	}
	.marquesina {
		position: relative;
		margin: clamp(48px, 6vw, 80px) calc(-1 * clamp(20px, 4vw, 56px)) 0;
		padding: 0 0 clamp(28px, 3vw, 40px);
		-webkit-mask: linear-gradient(90deg, transparent, #000 10%, #000 90%, transparent);
		mask: linear-gradient(90deg, transparent, #000 10%, #000 90%, transparent);
	}
	.tira {
		display: flex;
		gap: 14px;
		width: max-content;
		animation: tl-marquee 48s linear infinite;
	}
	@keyframes tl-marquee {
		to {
			transform: translateX(-50%);
		}
	}
	.tal {
		--tro: var(--tl-linea-2);
		flex: none;
		display: grid;
		grid-template-columns: 88px auto;
		min-height: 78px;
		border-radius: 20px;
		background:
			radial-gradient(circle at 88px 0, #f4eee2 9px, transparent 9.5px),
			radial-gradient(circle at 88px 100%, #f4eee2 9px, transparent 9.5px),
			var(--fondo-t);
		text-align: left;
	}
	.tal.claro {
		--fondo-t: #fffdf9;
		box-shadow: 0 18px 30px -20px rgba(15, 35, 40, 0.55);
		color: var(--tl-tinta);
		--sc: var(--tl-sub);
	}
	.tal.hueco {
		--fondo-t: repeating-linear-gradient(-45deg, rgba(60, 204, 144, 0.4) 0 2px, rgba(255, 253, 249, 0.9) 2px 10px);
		box-shadow: inset 0 0 0 1.5px rgba(23, 128, 90, 0.45);
		color: #0f3d2b;
		--sc: var(--tl-menta-texto);
		--tro: rgba(23, 128, 90, 0.4);
	}
	.tal.nuevo {
		--fondo-t: #3ccc90;
		box-shadow: 0 18px 30px -18px rgba(23, 128, 90, 0.9);
		color: var(--tl-petroleo-hondo);
		--sc: var(--tl-petroleo-hondo);
		--tro: rgba(16, 43, 49, 0.3);
	}
	.tal.osc {
		--fondo-t: #183c44;
		box-shadow: 0 18px 30px -18px rgba(15, 35, 40, 0.9);
		color: #fff;
		--sc: var(--tl-menta);
		--tro: rgba(255, 255, 255, 0.22);
	}
	.tal-h {
		display: flex;
		align-items: center;
		justify-content: center;
		border-right: 2px dashed var(--tro);
		font-size: 22px;
		font-weight: 900;
		font-variant-numeric: tabular-nums;
	}
	.tal-c {
		display: flex;
		flex-direction: column;
		justify-content: center;
		gap: 2px;
		padding: 0 22px 0 16px;
	}
	.tal-c b {
		font-size: 15px;
		white-space: nowrap;
	}
	.tal-s {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		font-size: 13px;
		white-space: nowrap;
		color: var(--sc);
	}
	.funciona {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: center;
		gap: 12px 32px;
		padding-top: clamp(32px, 4vw, 48px);
	}
	.funciona .rotulo {
		font-size: 12px;
		color: rgba(255, 255, 255, 0.72);
	}
	.integ {
		display: inline-flex;
		align-items: center;
		gap: 10px;
		font-size: 16px;
		font-weight: 700;
		color: rgba(255, 255, 255, 0.9);
	}
	.integ-logo {
		width: 38px;
		height: 38px;
		border-radius: 11px;
		background: #fff;
		display: grid;
		place-items: center;
		box-shadow: 0 8px 18px -10px rgba(0, 0, 0, 0.6);
	}
	.troquel {
		position: relative;
		height: 28px;
		background: linear-gradient(#183c44 0 50%, #fffdf9 50%);
	}
	.troquel span:nth-child(1),
	.troquel span:nth-child(2) {
		position: absolute;
		top: 0;
		width: 28px;
		height: 28px;
		border-radius: 50%;
		background: #fffdf9;
	}
	.troquel span:nth-child(1) {
		left: -14px;
	}
	.troquel span:nth-child(2) {
		right: -14px;
	}
	.troquel span:nth-child(3) {
		position: absolute;
		left: 40px;
		right: 40px;
		top: 13px;
		border-top: 2px dashed rgba(127, 150, 150, 0.35);
	}

	/* Secciones */
	.seccion {
		max-width: 1320px;
		margin: 0 auto;
		padding: var(--sep) var(--lado) 0;
		display: flex;
		flex-direction: column;
		gap: 40px;
	}
	#como {
		gap: 44px;
	}
	.titulo-fila {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		align-items: flex-end;
		gap: 24px;
	}
	.titulo-col {
		display: flex;
		flex-direction: column;
		gap: 14px;
		max-width: 760px;
	}
	.h2 {
		font-size: clamp(38px, 5.4vw, 72px);
		line-height: 0.95;
		letter-spacing: -0.045em;
	}
	.h2-chico {
		font-size: clamp(32px, 4vw, 52px);
		line-height: 1;
		letter-spacing: -0.04em;
	}
	.pasos {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 240px), 1fr));
		gap: 16px;
	}
	.paso {
		position: relative;
		overflow: hidden;
		display: flex;
		flex-direction: column;
		gap: 20px;
		padding: 32px;
		border-radius: 36px;
	}
	.paso h3 {
		font-size: 28px;
		line-height: 1.05;
		letter-spacing: -0.025em;
	}
	.paso p {
		font-size: 16px;
		line-height: 1.5;
	}
	.paso-txt {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.numero {
		font-size: 88px;
		line-height: 0.8;
		font-weight: 900;
		letter-spacing: -0.06em;
		color: transparent;
		-webkit-text-stroke: 2px var(--trazo);
		font-variant-numeric: tabular-nums;
	}
	.crema-puntos {
		--trazo: #183c44;
		background-color: var(--tl-crema);
		background-image: var(--tl-grano-osc), radial-gradient(rgba(24, 60, 68, 0.1) 1.2px, transparent 1.4px);
		background-size: auto, 18px 18px;
		box-shadow: inset 0 2px 0 rgba(255, 255, 255, 0.7);
	}
	.crema-puntos p {
		color: var(--tl-sub-2);
	}
	.paso.petroleo {
		--trazo: #3ccc90;
		background-color: var(--tl-petroleo);
		background-image: var(--tl-grano), radial-gradient(80% 60% at 100% 0%, rgba(60, 204, 144, 0.25), transparent 70%);
		color: #fff;
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.12);
	}
	.paso.petroleo p {
		color: rgba(255, 255, 255, 0.8);
	}
	.paso.menta {
		--trazo: #102b31;
		background-color: var(--tl-menta);
		background-image: var(--tl-grano-osc);
		color: var(--tl-petroleo-hondo);
		box-shadow: inset 0 2px 0 rgba(255, 255, 255, 0.45);
	}
	.link-qr {
		margin-top: auto;
		display: flex;
		align-items: center;
		gap: 14px;
		padding: 14px;
		border-radius: 22px;
		background: var(--tl-papel);
		box-shadow: 0 18px 36px -24px rgba(15, 35, 40, 0.6);
	}
	.qr-chico {
		width: 68px;
		height: 68px;
		flex: none;
		border-radius: 12px;
		overflow: hidden;
	}
	.link-qr-txt {
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
	}
	.link-qr-txt b {
		font-size: 17px;
		font-variant-numeric: tabular-nums;
		overflow-wrap: anywhere;
	}
	.talon-sena {
		margin-top: auto;
		display: grid;
		grid-template-columns: 92px minmax(0, 1fr);
		min-height: 80px;
		border-radius: 20px;
		background:
			radial-gradient(circle at 92px 0, #183c44 9px, transparent 9.5px),
			radial-gradient(circle at 92px 100%, #183c44 9px, transparent 9.5px),
			#fffdf9;
		color: var(--tl-tinta);
		box-shadow: 0 20px 40px -20px rgba(0, 0, 0, 0.6);
	}
	.ts-h {
		display: flex;
		flex-direction: column;
		justify-content: center;
		padding: 0 16px;
		border-right: 2px dashed var(--tl-linea-2);
	}
	.ts-h b {
		font-size: 26px;
		font-weight: 900;
		font-variant-numeric: tabular-nums;
	}
	.ts-c {
		display: flex;
		flex-direction: column;
		justify-content: center;
		padding: 0 16px;
		min-width: 0;
	}
	.ts-c b {
		font-size: 15px;
	}
	.ok {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		font-size: 13px;
		color: var(--tl-menta-texto);
		font-weight: 700;
	}
	.se-lleno {
		margin-top: auto;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.liberado,
	.tomado {
		display: flex;
		align-items: center;
		gap: 10px;
		min-height: 52px;
		padding: 0 16px;
		border-radius: 16px;
	}
	.liberado {
		background: repeating-linear-gradient(-45deg, rgba(16, 43, 49, 0.2) 0 2px, transparent 2px 10px);
		background-size: 28px 28px;
		animation: tl-raya 3.5s linear infinite;
		box-shadow: inset 0 0 0 1.5px rgba(16, 43, 49, 0.35);
	}
	.liberado b {
		font-size: 16px;
		font-variant-numeric: tabular-nums;
	}
	.liberado span {
		font-size: 14px;
		font-weight: 700;
	}
	.tomado {
		background: var(--tl-petroleo-hondo);
		color: #fff;
		font-size: 14px;
		box-shadow: 0 18px 30px -18px rgba(0, 0, 0, 0.6);
	}

	/* La app */
	.app {
		position: relative;
		overflow: hidden;
		display: grid;
		grid-template-columns: 1fr 1.05fr;
		gap: clamp(32px, 5vw, 64px);
		align-items: center;
		border-radius: clamp(32px, 4vw, 56px);
		background-color: var(--tl-petroleo-hondo);
		background-image: var(--tl-grano), radial-gradient(60% 80% at 85% 60%, rgba(60, 204, 144, 0.3), transparent 70%);
		color: #fff;
		padding: clamp(36px, 6vw, 80px);
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.1);
	}
	.simbolo-fondo {
		position: absolute;
		left: -80px;
		bottom: -120px;
		width: 420px;
		opacity: 0.03;
		pointer-events: none;
	}
	.app-txt {
		position: relative;
		display: flex;
		flex-direction: column;
		gap: 24px;
	}
	.app-h2 {
		font-size: clamp(38px, 5vw, 68px);
	}
	.app-puntos {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.app-puntos > span {
		display: flex;
		align-items: center;
		gap: 12px;
		font-size: 18px;
		font-weight: 700;
	}
	.ic-caja {
		width: 36px;
		height: 36px;
		flex: none;
		border-radius: 11px;
		background: rgba(60, 204, 144, 0.16);
		display: grid;
		place-items: center;
		color: var(--tl-menta);
	}
	.app-tiendas {
		display: flex;
		flex-wrap: wrap;
		align-items: flex-end;
		gap: 12px;
	}
	.qr-app {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 8px 14px 8px 8px;
		border-radius: 16px;
		background: rgba(255, 255, 255, 0.06);
		box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.12);
		font-size: 13px;
		line-height: 1.35;
		color: rgba(255, 255, 255, 0.8);
		white-space: nowrap;
	}
	.qr-56 {
		width: 56px;
		height: 56px;
		border-radius: 10px;
		overflow: hidden;
	}
	.telefono-col {
		position: relative;
		justify-self: center;
		width: min(100%, 340px);
		padding: 20px 0;
	}
	.icono-app {
		position: absolute;
		z-index: 3;
		right: -34px;
		top: -6px;
		width: 112px;
		height: 112px;
		border-radius: 28px;
		transform: rotate(10deg);
		box-shadow: 0 30px 50px -16px rgba(0, 0, 0, 0.6);
		animation: tl-flota 6s ease-in-out infinite;
	}
	@keyframes tl-flota {
		50% {
			transform: rotate(10deg) translateY(-10px);
		}
	}
	.telefono {
		position: relative;
		width: 300px;
		max-width: 100%;
		margin: 0 auto;
		aspect-ratio: 300 / 620;
		border-radius: 50px;
		padding: 10px;
		background: linear-gradient(145deg, #2a3a3e, #0b1214);
		box-shadow: 0 60px 90px -40px rgba(0, 0, 0, 0.9), inset 0 0 0 1.5px rgba(255, 255, 255, 0.12);
		transform: rotate(-5deg);
	}
	.pantalla {
		position: relative;
		height: 100%;
		overflow: hidden;
		border-radius: 40px;
		background-color: var(--tl-petroleo);
		background-image: var(--tl-grano), radial-gradient(90% 50% at 50% 0%, rgba(60, 204, 144, 0.35), transparent 70%);
		display: flex;
		flex-direction: column;
		align-items: center;
		padding: 20px 12px;
	}
	.isla {
		width: 96px;
		height: 28px;
		border-radius: 999px;
		background: #0b1214;
	}
	.fecha {
		margin-top: 28px;
		font-size: 15px;
		font-weight: 700;
		opacity: 0.85;
	}
	.hora {
		font-size: 76px;
		line-height: 1;
		font-weight: 900;
		letter-spacing: -0.04em;
		font-variant-numeric: tabular-nums;
	}
	.pushes {
		margin-top: auto;
		width: 100%;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.push {
		display: flex;
		gap: 10px;
		padding: 12px;
		border-radius: 20px;
		color: var(--tl-tinta);
		box-shadow: 0 14px 28px -16px rgba(0, 0, 0, 0.6);
	}
	.push img {
		border-radius: 9px;
		flex: none;
	}
	.push-c {
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
		text-align: left;
		flex: 1;
	}
	.push-t {
		display: flex;
		justify-content: space-between;
		font-size: 11px;
		font-weight: 700;
		color: var(--tl-sub);
	}
	.push-c b {
		font-size: 14px;
	}
	.push-s {
		font-size: 13px;
		line-height: 1.3;
		color: var(--tl-sub-2);
	}

	/* Calculadora */
	.calc {
		position: relative;
		overflow: hidden;
		display: grid;
		grid-template-columns: 1fr 1.05fr;
		border-radius: clamp(32px, 4vw, 56px);
		background-color: var(--tl-crema);
		background-image: var(--tl-grano-osc);
		box-shadow: inset 0 2px 0 rgba(255, 255, 255, 0.7);
	}
	.calc-foto {
		position: relative;
		min-height: 360px;
	}
	.calc-foto img {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		object-fit: cover;
		object-position: 60% 50%;
	}
	.silla {
		position: absolute;
		left: 20px;
		bottom: 20px;
		display: grid;
		grid-template-columns: 84px auto;
		min-height: 64px;
		border-radius: 18px;
		background: repeating-linear-gradient(-45deg, rgba(60, 204, 144, 0.45) 0 2px, rgba(255, 253, 249, 0.92) 2px 10px);
		background-size: 34px 34px;
		animation: tl-raya 3s linear infinite;
		color: #0f3d2b;
		box-shadow: 0 20px 40px -16px rgba(0, 0, 0, 0.6);
	}
	.silla b {
		display: flex;
		align-items: center;
		justify-content: center;
		border-right: 2px dashed rgba(23, 128, 90, 0.4);
		font-size: 22px;
		font-variant-numeric: tabular-nums;
	}
	.silla span {
		display: flex;
		align-items: center;
		padding: 0 18px;
		font-size: 15px;
		font-weight: 900;
	}
	.calc-txt {
		padding: clamp(32px, 5vw, 64px);
		display: flex;
		flex-direction: column;
		gap: 24px;
		justify-content: center;
	}
	.h2-calc {
		font-size: clamp(34px, 4.4vw, 58px);
		line-height: 0.98;
		letter-spacing: -0.045em;
	}
	.rango {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.rango > span {
		display: flex;
		justify-content: space-between;
		font-size: 16px;
		font-weight: 700;
	}
	.rango b {
		color: var(--tl-menta-texto);
		font-variant-numeric: tabular-nums;
	}
	.rango input {
		width: 100%;
		accent-color: #17805a;
		height: 32px;
		margin: 0;
	}
	.total {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
		padding: 20px 20px 20px 24px;
		border-radius: 24px;
		background-color: var(--tl-petroleo);
		background-image: var(--tl-grano);
		color: #fff;
	}
	.total-c {
		display: flex;
		flex-direction: column;
		gap: 4px;
		font-size: 14px;
		color: rgba(255, 255, 255, 0.8);
	}
	.total-c b {
		font-size: clamp(34px, 4vw, 52px);
		line-height: 1;
		font-weight: 900;
		letter-spacing: -0.035em;
		font-variant-numeric: tabular-nums;
		color: var(--tl-menta);
	}

	/* Bento */
	.bento {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 16px;
	}
	.b-card {
		display: flex;
		flex-direction: column;
		gap: 16px;
		padding: 32px;
		border-radius: 36px;
	}
	.b-card h3 {
		font-size: 26px;
		letter-spacing: -0.025em;
	}
	.b-txt,
	.b-tit > span {
		font-size: 16px;
		line-height: 1.5;
		text-wrap: pretty;
	}
	.ancha {
		grid-column: span 2;
		gap: 20px;
	}
	.b-tit {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	.crema-grano {
		background-color: var(--tl-crema);
		background-image: var(--tl-grano-osc);
		box-shadow: inset 0 2px 0 rgba(255, 255, 255, 0.7);
	}
	.crema-grano .b-tit > span,
	.papel .b-txt {
		color: var(--tl-sub-2);
	}
	.miniaturas {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
		gap: 12px;
	}
	.mini {
		display: flex;
		flex-direction: column;
		gap: 8px;
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.mini:hover {
		transform: translateY(-4px) rotate(-1deg);
	}
	.mini-hoja {
		position: relative;
		display: block;
		aspect-ratio: 3 / 4;
		border-radius: 18px;
		overflow: hidden;
		box-shadow: 0 18px 30px -18px rgba(15, 35, 40, 0.7);
	}
	.mini-foto {
		position: absolute;
		inset: 0 0 auto;
		background-size: cover;
		background-position: center;
	}
	.mini-pie {
		position: absolute;
		left: 10px;
		right: 10px;
		bottom: 10px;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	.mini-pie b {
		font-size: 15px;
		line-height: 1;
		letter-spacing: -0.02em;
	}
	.mini-pie span {
		height: 22px;
		border-radius: 999px;
	}
	.mini-n {
		font-size: 13px;
		font-weight: 700;
		color: var(--tl-sub-2);
	}
	.petroleo-brillo {
		background-color: var(--tl-petroleo);
		background-image: var(--tl-grano), radial-gradient(80% 60% at 100% 0%, rgba(60, 204, 144, 0.22), transparent 70%);
		color: #fff;
	}
	.petroleo-brillo > :global(.icono) {
		color: var(--tl-menta);
	}
	.petroleo-brillo .b-txt {
		color: rgba(255, 255, 255, 0.8);
	}
	.chat {
		margin-top: auto;
		display: flex;
		flex-direction: column;
		gap: 8px;
		font-size: 14px;
	}
	.yo {
		align-self: flex-end;
		max-width: 88%;
		padding: 10px 14px;
		border-radius: 18px 18px 4px 18px;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		font-weight: 700;
	}
	.ia {
		max-width: 92%;
		padding: 10px 14px;
		border-radius: 18px 18px 18px 4px;
		background: rgba(255, 255, 255, 0.1);
		line-height: 1.4;
	}
	.papel {
		background: var(--tl-papel);
		box-shadow: inset 0 0 0 1.5px var(--tl-linea), 0 24px 48px -36px rgba(15, 35, 40, 0.5);
	}
	.papel > :global(.icono) {
		color: var(--tl-menta-texto);
	}
	.wa {
		margin-top: auto;
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 14px;
		border-radius: 18px;
		background: var(--tl-crema);
		font-size: 14px;
		line-height: 1.35;
	}
	.pack {
		margin-top: auto;
		display: grid;
		grid-template-columns: auto 1fr;
		min-height: 62px;
		border-radius: 18px;
		background:
			radial-gradient(circle at 0 50%, #fffdf9 9px, transparent 9.5px),
			radial-gradient(circle at 100% 50%, #fffdf9 9px, transparent 9.5px),
			#183c44;
		color: #fff;
	}
	.pack b {
		display: flex;
		align-items: center;
		padding: 0 16px 0 22px;
		border-right: 2px dashed rgba(255, 255, 255, 0.22);
		font-size: 20px;
		color: var(--tl-menta);
		font-variant-numeric: tabular-nums;
		white-space: nowrap;
	}
	.pack span {
		display: flex;
		align-items: center;
		padding: 0 22px 0 16px;
		font-size: 14px;
	}
	.menta-grano {
		background-color: var(--tl-menta);
		background-image: var(--tl-grano-osc);
		color: var(--tl-petroleo-hondo);
	}
	.ejemplo {
		margin-top: auto;
		align-self: flex-start;
		gap: 6px;
		padding: 0 18px;
	}
	.centro {
		display: flex;
		justify-content: center;
	}

	/* Rubros */
	.rubros-sec {
		gap: 28px;
	}
	.rubros {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 200px), 1fr));
		gap: 14px;
	}
	.rubro {
		position: relative;
		display: flex;
		align-items: flex-end;
		min-height: 260px;
		padding: 22px;
		border-radius: 32px;
		overflow: hidden;
		background-color: var(--tl-petroleo);
		background-image: linear-gradient(180deg, transparent 30%, rgba(16, 43, 49, 0.94)), var(--foto);
		background-size: cover;
		background-position: center;
		color: #fff;
		box-shadow: 0 30px 50px -36px rgba(15, 35, 40, 0.8);
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.rubro:hover {
		transform: translateY(-4px);
	}
	.rubro-c {
		display: flex;
		flex-direction: column;
		gap: 4px;
		width: 100%;
	}
	.rubro-c b {
		font-size: 26px;
		letter-spacing: -0.02em;
	}
	.rubro-c > span:not(.rubro-cta) {
		font-size: 14px;
		color: rgba(255, 255, 255, 0.85);
	}
	.rubro-cta {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		margin-top: 8px;
		font-size: 14px;
		font-weight: 900;
		color: var(--tl-menta);
	}

	/* Precios */
	.titulo-centro {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 14px;
		text-align: center;
	}
	.sub {
		font-size: 17px;
		color: var(--tl-sub-2);
	}
	.planes {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 400px), 1fr));
		gap: 20px;
		max-width: 920px;
		width: 100%;
		margin: 0 auto;
	}
	.plan {
		display: flex;
		flex-direction: column;
		border-radius: 36px;
		overflow: hidden;
	}
	.plan.crema {
		background-color: var(--tl-crema);
		background-image: var(--tl-grano-osc);
		color: var(--tl-tinta);
		box-shadow: inset 0 0 0 1.5px var(--tl-linea);
		--tro: var(--tl-linea-2);
		--ic: var(--tl-menta-texto);
	}
	.plan.oscuro {
		background-color: var(--tl-petroleo);
		background-image: var(--tl-grano);
		color: #fff;
		box-shadow: 0 40px 70px -40px rgba(15, 35, 40, 0.8);
		--tro: rgba(255, 255, 255, 0.2);
		--ic: var(--tl-menta);
	}
	.plan-arriba {
		padding: 32px 32px 24px;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.plan-n {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 8px;
	}
	.plan-n b {
		font-size: 24px;
		letter-spacing: -0.02em;
	}
	.tag {
		height: 28px;
		padding: 0 12px;
		border-radius: 999px;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		display: grid;
		place-items: center;
		font-size: 12px;
		font-weight: 900;
		letter-spacing: 0.08em;
	}
	.plan-precio {
		display: flex;
		align-items: baseline;
		gap: 8px;
		flex-wrap: wrap;
	}
	.plan-precio b {
		font-size: 56px;
		line-height: 1;
		letter-spacing: -0.045em;
		font-variant-numeric: tabular-nums;
	}
	.plan-precio span {
		font-size: 15px;
		opacity: 0.8;
	}
	.plan-para {
		font-size: 15px;
		opacity: 0.9;
	}
	.plan-troquel {
		height: 22px;
		position: relative;
	}
	.plan-troquel span:nth-child(1),
	.plan-troquel span:nth-child(2) {
		position: absolute;
		top: 0;
		width: 22px;
		height: 22px;
		border-radius: 50%;
		background: var(--tl-papel);
	}
	.plan-troquel span:nth-child(1) {
		left: -11px;
	}
	.plan-troquel span:nth-child(2) {
		right: -11px;
	}
	.plan-troquel span:nth-child(3) {
		position: absolute;
		left: 24px;
		right: 24px;
		top: 10px;
		border-top: 2px dashed var(--tro);
	}
	.plan-abajo {
		padding: 16px 32px 32px;
		display: flex;
		flex-direction: column;
		gap: 12px;
		flex: 1;
	}
	.plan ul {
		list-style: none;
		margin: 0 0 12px;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.plan li {
		display: flex;
		gap: 10px;
		font-size: 15px;
		line-height: 1.4;
	}
	.plan li :global(.icono) {
		color: var(--ic);
	}
	.plan-cta {
		margin-top: auto;
		height: 56px;
		border-radius: 999px;
		display: grid;
		place-items: center;
		font-size: 17px;
		font-weight: 900;
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.plan-cta:hover {
		transform: translateY(-2px);
	}
	.crema .plan-cta {
		background: var(--tl-petroleo);
		color: #fff;
	}
	.oscuro .plan-cta {
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
	}

	/* Preguntas */
	.faq {
		max-width: 880px;
		gap: 12px;
	}
	.faq h2 {
		margin-bottom: 12px;
	}
	details {
		border-radius: 22px;
		background: #f8f3ea;
		padding: 0 22px;
	}
	summary {
		display: flex;
		align-items: center;
		gap: 16px;
		min-height: 68px;
		cursor: pointer;
		font-size: 18px;
		font-weight: 900;
		list-style: none;
	}
	summary::-webkit-details-marker {
		display: none;
	}
	summary span {
		flex: 1;
	}
	summary :global(.icono) {
		color: var(--tl-menta-texto);
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	details[open] summary :global(.icono) {
		transform: rotate(45deg);
	}
	details p {
		margin: 0 0 22px;
		font-size: 16px;
		line-height: 1.55;
		color: var(--tl-sub-2);
		max-width: 42em;
	}

	/* Cierre */
	.cierre-sec {
		padding-bottom: var(--sep);
	}
	.cierre-marco {
		position: relative;
		padding-top: clamp(40px, 4vw, 52px);
	}
	.cierre-pestana {
		position: absolute;
		left: 50%;
		top: 0;
		z-index: 2;
		width: clamp(160px, 18vw, 220px);
		height: clamp(80px, 8vw, 104px);
		transform: translateX(-50%);
		border-radius: 28px;
		background: var(--tl-petroleo-hondo);
		box-shadow: 0 0 0 14px var(--tl-papel);
		display: grid;
		place-items: center;
	}
	.cierre {
		position: relative;
		overflow: hidden;
		border-radius: clamp(32px, 4vw, 56px);
		background-color: var(--tl-menta);
		background-image: var(--tl-grano-osc), radial-gradient(rgba(16, 43, 49, 0.12) 1.2px, transparent 1.5px);
		background-size: auto, 20px 20px;
		color: var(--tl-petroleo-hondo);
		padding: clamp(80px, 9vw, 120px) clamp(24px, 5vw, 72px) clamp(40px, 6vw, 72px);
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 24px;
		text-align: center;
		box-shadow: inset 0 2px 0 rgba(255, 255, 255, 0.45);
	}
	.cierre h2 {
		font-size: clamp(40px, 6.4vw, 92px);
		line-height: 0.9;
		letter-spacing: -0.055em;
	}
	.cierre-form {
		width: 100%;
		max-width: 620px;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.cierre-campo {
		display: flex;
		align-items: center;
		min-height: 68px;
		padding: 0 8px 0 22px;
		border-radius: 999px;
		background: var(--tl-papel);
		box-shadow: inset 0 0 0 2px var(--borde), 0 24px 40px -20px rgba(16, 43, 49, 0.6);
		font-size: 18px;
		text-align: left;
	}
	.prefijo {
		color: var(--tl-sub);
		white-space: nowrap;
	}
	.cierre-campo input {
		flex: 1;
		min-width: 0;
		height: 56px;
		border: 0;
		background: transparent;
		font: 900 18px var(--tl-font);
		color: var(--tl-tinta);
		outline: none;
	}
	.reservarla,
	.reservarla-m {
		border: 0;
		border-radius: 999px;
		background: var(--tl-petroleo-hondo);
		color: #fff;
		font: 900 16px var(--tl-font);
		cursor: pointer;
	}
	.reservarla {
		height: 54px;
		padding: 0 24px;
	}
	.reservarla-m {
		display: none;
		height: 56px;
		font-size: 17px;
	}
	.cierre-msg {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 6px;
		font-size: 15px;
		font-weight: 700;
	}

	/* Pie */
	.pie {
		position: relative;
		overflow: hidden;
		background-color: var(--tl-petroleo-hondo);
		background-image: var(--tl-grano);
		color: rgba(255, 255, 255, 0.78);
	}
	.pie-in {
		max-width: 1320px;
		margin: 0 auto;
		padding: clamp(56px, 7vw, 88px) var(--lado) 0;
		display: flex;
		flex-direction: column;
		gap: 56px;
	}
	.pie-arriba {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		align-items: flex-end;
		gap: 28px;
		padding-bottom: 48px;
		border-bottom: 2px dashed rgba(255, 255, 255, 0.12);
	}
	.pie-tit {
		display: flex;
		flex-direction: column;
		gap: 16px;
		max-width: 620px;
		font-size: 17px;
	}
	.pie-tit h2 {
		font-size: clamp(36px, 5vw, 68px);
		line-height: 0.92;
		letter-spacing: -0.05em;
		color: #fff;
	}
	.pie-tit h2 span {
		color: var(--tl-menta);
	}
	.pie-cta {
		display: flex;
		flex-direction: column;
		gap: 16px;
		align-items: flex-start;
	}
	.pie-cols {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 170px), 1fr));
		gap: 32px;
		font-size: 15px;
	}
	.pie-col {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.pie-col b {
		font-size: 12px;
		font-weight: 900;
		letter-spacing: 0.08em;
		color: #fff;
	}
	.pie-col a {
		color: rgba(255, 255, 255, 0.78);
		min-height: 24px;
	}
	.pie-col a:hover {
		color: var(--tl-menta);
	}
	.redes {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}
	.redes a {
		height: 40px;
		padding: 0 14px;
		border-radius: 999px;
		background: rgba(255, 255, 255, 0.08);
		box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.12);
		color: #fff;
		display: grid;
		place-items: center;
		font-size: 14px;
		font-weight: 700;
	}
	.legal {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		gap: 12px 24px;
		font-size: 13px;
		color: rgba(255, 255, 255, 0.72);
	}
	.legal a {
		color: inherit;
	}
	.logo-gigante {
		display: block;
		width: calc(100% - 2 * var(--lado));
		max-width: 1320px;
		margin: 40px auto -4%;
		opacity: 0.07;
		pointer-events: none;
	}

	/* Barra flotante */
	.barra {
		position: fixed;
		left: 50%;
		bottom: 16px;
		z-index: 30;
		transform: translate(-50%, 140%);
		width: min(calc(100% - 24px), 460px);
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 8px 8px 8px 20px;
		border-radius: 999px;
		background: rgba(16, 43, 49, 0.92);
		backdrop-filter: blur(12px);
		-webkit-backdrop-filter: blur(12px);
		color: #fff;
		box-shadow: 0 20px 44px -14px rgba(0, 0, 0, 0.6), inset 0 0 0 1px rgba(255, 255, 255, 0.1);
		transition: transform var(--tl-entrar) var(--tl-ease);
	}
	.barra.visible {
		transform: translate(-50%, 0);
	}
	.barra span {
		flex: 1;
		font-size: 14px;
		font-weight: 700;
	}

	:global(.aparece) {
		transition:
			opacity var(--tl-entrar) var(--tl-ease),
			transform var(--tl-entrar) var(--tl-ease);
	}
	:global(.aparece.oculto) {
		opacity: 0;
		transform: translateY(32px);
	}

	@media (max-width: 1039px) {
		.app,
		.calc {
			grid-template-columns: minmax(0, 1fr);
		}
		.bento {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}
	@media (max-width: 759px) {
		.enlaces,
		.qr-app,
		.reservarla {
			display: none;
		}
		.reservarla-m {
			display: block;
		}
		.bento {
			grid-template-columns: minmax(0, 1fr);
		}
		.ancha {
			grid-column: auto;
		}
		.cierre-campo {
			font-size: 16px;
			padding-right: 22px;
		}
		.cierre-campo input {
			font-size: 16px;
		}
		.tiendas img {
			height: 48px;
		}
		.icono-app {
			right: 0;
			width: 88px;
			height: 88px;
			border-radius: 22px;
		}
		.pie-cta,
		.pie-cta :global(.pastilla) {
			width: 100%;
		}
	}
</style>
