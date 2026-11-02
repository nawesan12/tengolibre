<!--
	El armazón del panel (handoff v2, «Panel v2»): costado petróleo con grano y la pestaña activa
	fundida con la página, el negocio arriba, «El día» abajo; barra con el buscador, la campanita y
	«Ver mi página». En el teléfono, barra petróleo arriba, pestañas abajo y «Nuevo turno» a mano.

	El contenido de cada página va en <main class="contenido">: ancho máximo 1180, laterales de 36 px
	en escritorio y 14 px en el teléfono, y 20 px entre bloques (flex en columna).
-->
<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import Icono from '#lib/ui/Icono.svelte';
	import { escucharPanel } from '#lib/tiempo-real.svelte.ts';
	import { PESTANIAS, PESTANIAS_PROFESIONAL, PRINCIPAL, SECUNDARIO, activa, nombreDe, visible } from '#lib/panel/navegacion.ts';
	import { armarDia, horaAhora, minutosAhora, profesionalesDelDia, rangoDelDia } from '#lib/panel/dia.ts';
	import ElDia from '#lib/panel/ElDia.svelte';
	import SelectorNegocio from '#lib/panel/SelectorNegocio.svelte';
	import Campanita from '#lib/panel/Campanita.svelte';
	import Buscador from '#lib/panel/Buscador.svelte';

	let { data, children } = $props();

	const rol = $derived(data.negocio.rol);
	const esProfesional = $derived(rol === 'profesional');
	const principal = $derived(PRINCIPAL.filter((e) => visible(e, rol) && (!esProfesional || ['inicio', 'turnos', 'calendario'].includes(e.clave))));
	const secundario = $derived(SECUNDARIO.filter((e) => visible(e, rol)));
	const pestanias = $derived(esProfesional ? PESTANIAS_PROFESIONAL : PESTANIAS.filter((e) => visible(e, rol)));
	const enMas = $derived([...principal.filter((e) => !pestanias.some((p) => p.clave === e.clave)), ...secundario]);
	const ruta = $derived(page.url.pathname);

	// La hora: se mueve sola mientras el panel está abierto.
	let ahoraMs = $state(Date.now());
	const ahora = $derived(minutosAhora(new Date(ahoraMs)));
	const ahoraTexto = $derived(horaAhora(new Date(ahoraMs)));
	const fechaCorta = $derived.by(() => {
		const p = Object.fromEntries(
			new Intl.DateTimeFormat('es-AR', { weekday: 'short', day: '2-digit', month: '2-digit', timeZone: 'America/Argentina/Buenos_Aires' })
				.formatToParts(new Date(ahoraMs))
				.map((x) => [x.type, x.value])
		);
		return `${String(p.weekday).replace('.', '').toUpperCase()} ${p.day}·${p.month}`;
	});

	const propio = $derived((data.negocio as { profesionalId?: string | null }).profesionalId ?? null);
	const dias = $derived(armarDia(profesionalesDelDia(data.hoy.profesionales, data.hoy.turnos, rol, propio), data.hoy.turnos, data.hoy.fecha, ahora));
	const rango = $derived(rangoDelDia(dias));
	const vivosHoy = $derived(data.hoy.turnos.filter((t) => t.estado !== 'cancelado').length);

	let secAbierto = $state(false);
	const secActivo = $derived(secundario.some((e) => activa(ruta, e)));
	let mas = $state(false);
	let buscando = $state(false);

	const bloqueado = $derived(!!data.acceso?.bloqueado);
	const avisoPrueba = $derived(
		!bloqueado && !!data.acceso && !data.acceso.conSuscripcion && !!data.acceso.pruebaHasta && data.acceso.diasDePrueba <= 3 && rol === 'duenio'
	);
	const verEnvios = $derived(['duenio', 'encargado', 'recepcion'].includes(rol));
	// «Nuevo turno» flotante sólo donde es la acción principal (un foco por pantalla).
	const conFab = $derived(['/panel', '/panel/turnos', '/panel/calendario', '/panel/clientes', '/panel/lista-espera'].includes(ruta));

	onMount(() => {
		const parar = escucharPanel();
		const reloj = setInterval(() => (ahoraMs = Date.now()), 30_000);
		return () => {
			parar();
			clearInterval(reloj);
		};
	});

	// Al navegar se cierran las capas del teléfono.
	$effect(() => {
		void ruta;
		mas = false;
		buscando = false;
	});

	// El link de la página desde la barra del teléfono: el menú de compartir del sistema o, si no hay, copiarlo.
	async function compartirPagina() {
		const url = `${location.origin}/${data.negocio.slug}`;
		if (typeof navigator.share === 'function') {
			try {
				await navigator.share({ title: data.negocio.nombre, text: `Reservá tu turno en ${data.negocio.nombre}`, url });
			} catch {
				// Cerró el menú sin elegir.
			}
			return;
		}
		try {
			await navigator.clipboard.writeText(url);
		} catch {
			window.open(`/${data.negocio.slug}`, '_blank', 'noopener');
		}
	}
</script>

<div class="panel">
	<nav class="costado" aria-label="Panel">
		<img class="marca-agua" src="/marca/simbolo_una_tinta_blanco.svg" alt="" aria-hidden="true" />
		<SelectorNegocio negocio={data.negocio} negocios={data.negocios} persona={data.persona} />
		<ul class="principal">
			{#each principal as e (e.clave)}
				{@const on = activa(ruta, e)}
				<li>
					<a href={e.href} class:on aria-current={on ? 'page' : undefined}>
						<Icono nombre={e.icono} lleno={on} tam={21} />
						<span class="nombre">{nombreDe(e, rol)}</span>
						{#if e.clave === 'turnos' && on}<span class="k tl-num">{vivosHoy}</span>{/if}
						{#if e.clave === 'espera' && data.esperando && !on}<span class="badge tl-num" aria-label={`${data.esperando} esperando`}>{data.esperando}</span>{/if}
					</a>
				</li>
			{/each}
		</ul>
		{#if secundario.length}
			<div class="secundario">
				<button class="toggle" aria-expanded={secAbierto || secActivo} aria-controls="nav-ajustes" onclick={() => (secAbierto = !secAbierto)}>
					<Icono nombre="apps" tam={20} />
					<span>Tu negocio y ajustes</span>
					<Icono nombre={secAbierto || secActivo ? 'expand_less' : 'expand_more'} tam={20} />
				</button>
				{#if secAbierto || secActivo}
					<ul id="nav-ajustes" class="grilla-sec">
						{#each secundario as e (e.clave)}
							{@const on = activa(ruta, e)}
							<li>
								<a href={e.href} class:on aria-current={on ? 'page' : undefined}><Icono nombre={e.icono} tam={16} />{nombreDe(e, rol)}</a>
							</li>
						{/each}
					</ul>
				{/if}
			</div>
		{/if}
		<span class="aire" aria-hidden="true"></span>
		<ElDia {dias} {rango} {ahora} {ahoraTexto} {esProfesional} />
	</nav>

	<div class="columna">
		<header class="barra">
			<Buscador />
			<span class="crece"></span>
			<Campanita avisos={data.avisos} sinLeer={data.sinLeer} {verEnvios} />
			<a class="ver-pagina" href={`/${data.negocio.slug}`} target="_blank" rel="noopener">Ver mi página<Icono nombre="north_east" tam={18} /></a>
		</header>

		<header class="barra-tel tl-petroleo">
			<SelectorNegocio negocio={data.negocio} negocios={data.negocios} persona={data.persona} compacto sub={`${fechaCorta} · ${ahoraTexto}`} />
			<button class="icono-tel" aria-label="Compartir tu página" onclick={compartirPagina}><Icono nombre="ios_share" tam={24} /></button>
			<button class="icono-tel" aria-label="Buscar" onclick={() => (buscando = true)}><Icono nombre="search" tam={24} /></button>
			<Campanita avisos={data.avisos} sinLeer={data.sinLeer} {verEnvios} compacto />
		</header>
		{#if buscando}<Buscador pantalla alCerrar={() => (buscando = false)} />{/if}

		<main class="contenido" id="contenido">
			{#if bloqueado}
				<div class="aviso-lectura tl-petroleo" role="status">
					<Icono nombre="visibility" tam={22} />
					{#if rol === 'duenio'}
						<span class="crece-txt"><b>Tu prueba gratis terminó.</b> Podés ver todo, pero para hacer cambios tenés que elegir un plan. Tu página sigue tomando&nbsp;reservas.</span>
						<a class="elegir" href="/panel/suscripcion">Elegir un plan</a>
					{:else}
						<span class="crece-txt"><b>El panel está en sólo lectura.</b> Para volver a hacer cambios, el dueño tiene que elegir un plan. La página sigue tomando&nbsp;reservas.</span>
					{/if}
				</div>
			{:else if avisoPrueba && data.acceso}
				<div class="aviso-prueba" role="status">
					<b class="tl-num">{data.acceso.diasDePrueba === 1 ? '1 día' : `${data.acceso.diasDePrueba} días`}</b>
					<span class="crece-txt">de prueba. Elegí un plan para no perder los&nbsp;cambios.</span>
					<a href="/panel/suscripcion">Ver planes</a>
				</div>
			{/if}
			{@render children()}
		</main>
	</div>

	{#if conFab && !bloqueado}
		<a class="fab" href="/panel/turnos/nuevo"><Icono nombre="add" tam={24} />Nuevo turno</a>
	{/if}
	<nav class="tabs" aria-label="Panel" style:--n={pestanias.length + 1}>
		{#each pestanias as e (e.clave)}
			{@const on = !mas && activa(ruta, e)}
			<a href={e.href} aria-current={on ? 'page' : undefined} class:on>
				<span class="pill"><Icono nombre={e.icono} lleno={on} tam={22} /></span>{e.nombre}
			</a>
		{/each}
		<button class:on={mas} aria-expanded={mas} aria-controls="hoja-mas" onclick={() => (mas = !mas)}>
			<span class="pill"><Icono nombre={esProfesional ? 'person' : 'menu'} lleno={mas} tam={22} /></span>{esProfesional ? 'Yo' : 'Más'}
		</button>
	</nav>
	{#if mas}
		<button class="velo" aria-label="Cerrar el menú" onclick={() => (mas = false)}></button>
		<div id="hoja-mas" class="hoja" role="dialog" aria-label="Más secciones">
			<span class="manija" aria-hidden="true"></span>
			{#each enMas as e (e.clave)}
				<a href={e.href} aria-current={activa(ruta, e) ? 'page' : undefined}>
					<Icono nombre={e.icono} tam={22} />
					<span class="crece-txt">{nombreDe(e, rol)}</span>
					{#if e.clave === 'espera' && data.esperando}<span class="badge tl-num">{data.esperando}</span>{/if}
				</a>
			{/each}
			<a href="/mi-cuenta"><Icono nombre="account_circle" tam={22} /><span class="crece-txt">Mi cuenta</span></a>
			<form method="POST" action="/salir"><button class="salir"><Icono nombre="logout" tam={22} />Salir</button></form>
		</div>
	{/if}
</div>

<style>
	.panel {
		min-height: 100dvh;
		display: grid;
		grid-template-columns: 264px minmax(0, 1fr);
		background-color: var(--tl-crema);
		background-image: var(--tl-grano-osc);
	}
	/* ── Costado ── */
	.costado {
		position: sticky;
		top: 0;
		height: 100vh;
		height: 100dvh;
		overflow-x: hidden;
		overflow-y: auto;
		scrollbar-width: none;
		background-color: var(--tl-petroleo);
		background-image: var(--tl-grano);
		color: #fff;
		display: flex;
		flex-direction: column;
		padding: 18px 0 16px 14px;
	}
	.marca-agua {
		position: absolute;
		left: -90px;
		top: 150px;
		width: 360px;
		height: 360px;
		opacity: 0.035;
		pointer-events: none;
	}
	ul {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.principal {
		position: relative;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}
	.principal a {
		position: relative;
		display: flex;
		align-items: center;
		gap: 12px;
		height: 42px;
		padding: 0 12px;
		margin-right: 14px;
		border-radius: 12px;
		color: rgba(255, 255, 255, 0.8);
		font-size: 14px;
		text-decoration: none;
		transition: background var(--tl-tocar) var(--tl-ease);
	}
	.principal a:hover {
		background: rgba(255, 255, 255, 0.07);
	}
	.principal a :global(.icono) {
		color: rgba(255, 255, 255, 0.55);
	}
	.principal a.on {
		height: 46px;
		margin-right: 0;
		border-radius: 14px 0 0 14px;
		background: var(--tl-crema);
		color: var(--tl-tinta);
		font-weight: 700;
	}
	.principal a.on :global(.icono) {
		color: var(--tl-menta-texto);
	}
	/* La pestaña: dos curvas del color de la página la funden con el contenido. */
	.principal a.on::before,
	.principal a.on::after {
		content: '';
		position: absolute;
		right: 0;
		width: 14px;
		height: 14px;
		pointer-events: none;
	}
	.principal a.on::before {
		top: -14px;
		background: radial-gradient(circle at 0 0, transparent 14px, var(--tl-crema) 14.5px);
	}
	.principal a.on::after {
		bottom: -14px;
		background: radial-gradient(circle at 0 100%, transparent 14px, var(--tl-crema) 14.5px);
	}
	.nombre {
		flex: 1;
	}
	.k {
		margin-right: 12px;
		font-size: 12px;
		font-weight: 700;
		color: var(--tl-sub);
	}
	.badge {
		min-width: 20px;
		height: 20px;
		padding: 0 6px;
		border-radius: 999px;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		font-size: 11px;
		font-weight: 700;
		display: grid;
		place-items: center;
	}
	.secundario {
		position: relative;
		margin: 12px 14px 0 0;
		padding-top: 8px;
		border-top: 1px solid rgba(255, 255, 255, 0.1);
	}
	.toggle {
		display: flex;
		align-items: center;
		gap: 12px;
		width: 100%;
		height: 40px;
		padding: 0 12px;
		border: 0;
		border-radius: 12px;
		background: transparent;
		color: rgba(255, 255, 255, 0.72);
		font: 400 13px var(--tl-font);
		text-align: left;
		cursor: pointer;
	}
	.toggle:hover {
		background: rgba(255, 255, 255, 0.07);
	}
	.toggle span {
		flex: 1;
	}
	.toggle :global(.icono) {
		color: rgba(255, 255, 255, 0.55);
	}
	.grilla-sec {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 2px;
		padding: 4px 0 2px;
	}
	.grilla-sec a {
		display: flex;
		align-items: center;
		gap: 6px;
		min-height: 36px;
		padding: 4px 8px;
		border-radius: 9px;
		color: rgba(255, 255, 255, 0.85);
		font-size: 12px;
		line-height: 1.2;
		text-decoration: none;
	}
	.grilla-sec a:hover {
		background: rgba(255, 255, 255, 0.07);
	}
	.grilla-sec a.on {
		background: rgba(60, 204, 144, 0.18);
		color: #fff;
		font-weight: 700;
	}
	.grilla-sec a :global(.icono) {
		color: rgba(255, 255, 255, 0.5);
	}
	/* Siempre algo de aire entre el menú y «El día», aunque la pantalla sea baja. */
	.aire {
		flex: 0 0 24px;
	}

	/* ── Columna y barra ── */
	.columna {
		display: flex;
		flex-direction: column;
		min-width: 0;
		min-height: 100dvh;
		background-image: radial-gradient(rgba(24, 60, 68, 0.08) 1px, transparent 1.3px);
		background-size: 18px 18px;
	}
	.barra {
		position: sticky;
		top: 0;
		z-index: 20;
		height: 72px;
		flex: none;
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 0 36px;
		background: rgba(244, 238, 226, 0.86);
		backdrop-filter: blur(10px);
		-webkit-backdrop-filter: blur(10px);
	}
	.crece {
		flex: 1;
	}
	.ver-pagina {
		display: flex;
		align-items: center;
		gap: 6px;
		height: 44px;
		padding: 0 16px;
		border-radius: 14px;
		background: var(--tl-papel);
		box-shadow: inset 0 0 0 1px #e6dcc8;
		color: var(--tl-tinta);
		font-size: 14px;
		font-weight: 700;
		text-decoration: none;
		white-space: nowrap;
		flex: none;
	}
	.ver-pagina:hover {
		background: #fff;
	}
	.barra-tel {
		display: none;
		position: sticky;
		top: 0;
		z-index: 20;
		align-items: center;
		gap: 4px;
		min-height: 60px;
		padding: 8px 8px 8px 16px;
	}
	.icono-tel {
		width: 48px;
		height: 48px;
		border: 0;
		border-radius: 12px;
		background: transparent;
		color: #fff;
		display: grid;
		place-items: center;
		cursor: pointer;
	}

	/* ── Contenido ── */
	.contenido {
		flex: 1;
		width: 100%;
		max-width: 1180px;
		padding: 8px 36px 64px;
		display: flex;
		flex-direction: column;
		gap: 20px;
		min-width: 0;
	}
	.aviso-lectura {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 12px 14px;
		padding: 14px 16px;
		border-radius: 16px;
		font-size: 14px;
		line-height: 1.4;
	}
	.aviso-lectura :global(.icono) {
		color: var(--tl-menta);
	}
	.crece-txt {
		flex: 1 1 220px;
		text-wrap: pretty;
	}
	.elegir {
		display: inline-flex;
		align-items: center;
		min-height: 44px;
		padding: 0 18px;
		border-radius: 999px;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		font-weight: 700;
		text-decoration: none;
	}
	.aviso-prueba {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 8px 12px;
		padding: 10px 16px;
		border-radius: 14px;
		background: #fbe9c2;
		color: #5c3800;
		font-size: 14px;
	}
	.aviso-prueba a {
		color: #5c3800;
		font-weight: 700;
		min-height: 32px;
		display: inline-flex;
		align-items: center;
	}

	/* ── Teléfono ── */
	.fab,
	.tabs,
	.velo,
	.hoja {
		display: none;
	}
	@media (max-width: 859px) {
		.panel {
			grid-template-columns: minmax(0, 1fr);
		}
		.costado,
		.barra {
			display: none;
		}
		.barra-tel {
			display: flex;
		}
		.contenido {
			padding: 16px 14px 160px;
		}
		.fab {
			position: fixed;
			right: 16px;
			bottom: 96px;
			z-index: 15;
			display: inline-flex;
			align-items: center;
			gap: 8px;
			height: 56px;
			padding: 0 20px 0 16px;
			border-radius: 999px;
			background: var(--tl-menta);
			color: var(--tl-petroleo-hondo);
			font-size: 15px;
			font-weight: 700;
			text-decoration: none;
			box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.45), 0 12px 24px -10px rgba(60, 204, 144, 0.8);
			animation: tl-gin 320ms var(--tl-ease) both;
		}
		.tabs {
			position: fixed;
			left: 0;
			right: 0;
			bottom: 0;
			z-index: 15;
			display: grid;
			grid-template-columns: repeat(var(--n), 1fr);
			height: 80px;
			padding: 6px 4px 18px;
			background: var(--tl-papel);
			box-shadow: 0 -1px 0 #e6dcc8;
		}
		.tabs a,
		.tabs button {
			display: flex;
			flex-direction: column;
			align-items: center;
			justify-content: center;
			gap: 3px;
			border: 0;
			background: transparent;
			color: var(--tl-sub);
			font: 700 11px var(--tl-font);
			text-decoration: none;
			cursor: pointer;
			min-height: 48px;
		}
		.tabs .on {
			color: var(--tl-petroleo);
		}
		.pill {
			padding: 3px 14px;
			border-radius: 999px;
		}
		.tabs .on .pill {
			background: var(--tl-menta-suave);
		}
		.velo {
			display: block;
			position: fixed;
			inset: 0;
			z-index: 40;
			border: 0;
			background: rgba(15, 35, 40, 0.4);
		}
		.hoja {
			position: fixed;
			left: 0;
			right: 0;
			bottom: 0;
			z-index: 41;
			max-height: 85dvh;
			overflow-y: auto;
			padding: 10px 16px 28px;
			border-radius: 24px 24px 0 0;
			background: var(--tl-papel);
			display: flex;
			flex-direction: column;
			gap: 4px;
			animation: tl-gin 320ms var(--tl-ease) both;
		}
		.manija {
			align-self: center;
			width: 40px;
			height: 4px;
			border-radius: 2px;
			background: #d8ccb4;
			margin-bottom: 8px;
		}
		.hoja a,
		.salir {
			display: flex;
			align-items: center;
			gap: 14px;
			width: 100%;
			min-height: 52px;
			padding: 0 6px;
			border: 0;
			border-bottom: 1px solid #f0e8d8;
			background: transparent;
			color: var(--tl-tinta);
			font: 400 16px var(--tl-font);
			text-align: left;
			text-decoration: none;
			cursor: pointer;
		}
		.hoja a[aria-current='page'] {
			font-weight: 700;
		}
		.hoja :global(.icono) {
			color: var(--tl-menta-texto);
		}
		.salir {
			color: #a3241a;
			border-bottom: 0;
		}
		.salir :global(.icono) {
			color: #a3241a;
		}
	}
</style>
