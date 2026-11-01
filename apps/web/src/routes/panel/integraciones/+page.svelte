<script lang="ts">
	import '#lib/gestion/gestion.css';
	import { enhance } from '$app/forms';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';
	import { confirmar, sinReset } from '#lib/configuracion/formularios.ts';
	import Copiar from '#lib/gestion/Copiar.svelte';
	import { ASISTENTES, ICONO_PERMISO } from '#lib/gestion/api-integraciones.ts';
	import { ZONA } from '#lib/formato.ts';

	let { data, form } = $props();
	const mcp = $derived(data.mcp);
	const elegida = $derived(data.elegida);
	const clave = $derived(
		form && 'datos' in form && form.que === 'clave' ? (form.datos as { clave?: string } | undefined)?.clave : undefined
	);
	/** Cada asistente del diseño es una clave con ese nombre; las de otro nombre van al final. */
	const chips = $derived([
		...ASISTENTES.map((n) => ({ nombre: n, token: mcp?.tokens.find((t) => t.nombre.toLowerCase() === n.toLowerCase()) ?? null })),
		...(mcp?.tokens ?? [])
			.filter((t) => !ASISTENTES.some((n) => n.toLowerCase() === t.nombre.toLowerCase()))
			.map((t) => ({ nombre: t.nombre, token: t }))
	]);

	/** Mandar el formulario apenas cambia algo (los interruptores se guardan solos). */
	const mandar = (e: Event) => (e.currentTarget as HTMLInputElement).form?.requestSubmit();

	const hoy = new Intl.DateTimeFormat('en-CA', { timeZone: ZONA }).format(new Date());
	const ayer = new Intl.DateTimeFormat('en-CA', { timeZone: ZONA }).format(new Date(Date.now() - 86_400_000));
	function cuando(iso: string) {
		const d = new Date(iso);
		const dia = new Intl.DateTimeFormat('en-CA', { timeZone: ZONA }).format(d);
		if (dia === hoy) return new Intl.DateTimeFormat('es-AR', { timeZone: ZONA, hour: '2-digit', minute: '2-digit' }).format(d);
		if (dia === ayer) return 'AYER';
		return new Intl.DateTimeFormat('es-AR', { timeZone: ZONA, weekday: 'short', day: 'numeric' }).format(d).toUpperCase();
	}

	const nombreCorto = $derived(data.negocio.nombre.split(' ')[0]);

	/** «Lo último que hizo la IA»: el asistente del negocio y los de las clientas, lo más nuevo arriba. */
	type Linea = { id: string; creadoEn: string; quien: string; texto: string; error?: string };
	const lineas = $derived<Linea[]>(
		[
			...data.registro.map((r) =>
				r.ok
					? { id: r.id, creadoEn: r.creadoEn, quien: r.token, texto: r.resumen || r.titulo.toLowerCase() }
					: { id: r.id, creadoEn: r.creadoEn, quien: r.token, texto: r.titulo.toLowerCase(), error: r.error }
			),
			...(data.publico?.ultimas ?? []).map((u) => ({ id: u.turnoId, creadoEn: u.creadoEn, quien: 'Asistente de una clienta', texto: u.resumen }))
		]
			.sort((a, b) => b.creadoEn.localeCompare(a.creadoEn))
			.slice(0, 5)
	);
	const cuantos = ['', 'un hueco', 'dos huecos', 'tres huecos'];

	type Otra = { t: string; logo?: string; icono?: string; ok: boolean; estado: string; accion: string; href: string; externo?: boolean };
	const otras = $derived<Otra[]>([
		{
			t: 'Google Calendar',
			logo: '/logos/googlecalendar.svg',
			ok: data.calendarios > 0,
			estado: data.calendarios > 0 ? `Conectado · ${data.calendarios} ${data.calendarios === 1 ? 'calendario' : 'calendarios'}` : 'Sin conectar',
			accion: data.calendarios > 0 ? 'Ver' : 'Conectar',
			href: '/panel/calendario/suscribirse'
		},
		{
			t: 'Mercado Pago',
			logo: '/logos/mercadopago.svg',
			ok: !!data.mp?.conectada,
			estado: data.mp?.conectada ? 'Conectado' : 'Sin conectar',
			accion: data.mp?.conectada ? 'Ver' : 'Conectar',
			href: '/panel/pagos'
		},
		{
			t: 'WhatsApp',
			logo: '/logos/whatsapp.svg',
			ok: !!data.negocio.whatsapp,
			estado: data.negocio.whatsapp ? 'Link en tu página' : 'Sin cargar',
			accion: data.negocio.whatsapp ? 'Editar' : 'Cargar',
			href: '/panel/mi-web'
		},
		{
			t: 'Apple y Outlook',
			logo: '/logos/apple.svg',
			ok: data.calendarios > 0,
			estado: data.calendarios > 0 ? 'Con tu link iCal' : 'Sin conectar',
			accion: data.calendarios > 0 ? 'Ver' : 'Conectar',
			href: '/panel/calendario/suscribirse'
		},
		{
			t: 'Botón en Instagram',
			icono: 'photo_camera',
			ok: !!data.negocio.instagram,
			estado: data.negocio.instagram ? `@${data.negocio.instagram}` : 'Sin conectar',
			accion: data.negocio.instagram ? 'Editar' : 'Conectar',
			href: '/panel/mi-web'
		}
	]);
</script>

<svelte:head><title>Integraciones e IA · Panel</title></svelte:head>

<div class="g-pagina integraciones">
	<div class="g-col cabeza">
		<h1 class="g-titulo">Integraciones e&nbsp;IA</h1>
		<p class="g-bajada bajada">
			Conectá Tengo Libre con las herramientas que ya usás y con los asistentes de inteligencia artificial, los tuyos y los de tus&nbsp;clientes.
		</p>
	</div>

	<div class="dos">
		<section class="asistente" aria-labelledby="t-asistente">
			<div class="asistente-cabeza tl-petroleo">
				<span class="fila-titulo">
					<span class="icono lleno robot" aria-hidden="true">smart_toy</span>
					<h2 id="t-asistente">Tu asistente de&nbsp;IA</h2>
					{#if elegida?.cliente}
						<span class="estado-oscuro"><span class="icono lleno" aria-hidden="true">check_circle</span>Conectado</span>
					{:else if elegida}
						<span class="estado-oscuro espera"><span class="icono" aria-hidden="true">pending</span>Esperando conexión</span>
					{:else}
						<span class="estado-oscuro apagado"><span class="icono" aria-hidden="true">radio_button_unchecked</span>Sin activar</span>
					{/if}
				</span>
				<p>Preguntale «¿qué huecos tengo mañana?» o pedile «pasá a Sofía al jueves» desde el asistente que&nbsp;uses.</p>
				{#if mcp}
					<nav class="asistentes" aria-label="Tus asistentes">
						{#each chips as c (c.nombre)}
							{#if c.token}
								<a class="asist" class:on={elegida?.id === c.token.id} href="?clave={c.token.id}" aria-current={elegida?.id === c.token.id ? 'true' : undefined}>
									{c.nombre}
									{#if elegida?.id === c.token.id}<span class="icono" aria-hidden="true">check</span>{/if}
								</a>
							{:else}
								<form method="POST" action="?/crear" use:enhance={sinReset}>
									<input type="hidden" name="nombre" value={c.nombre} />
									<button class="asist" aria-label="Sumar {c.nombre}">{#if elegida}<span class="icono" aria-hidden="true">add</span>{/if}{c.nombre}</button>
								</form>
							{/if}
						{/each}
					</nav>
				{/if}
				<Mensaje {form} que="ia" />
			</div>

			{#if clave}
				<div class="direccion">
					<div class="g-aviso ambar" role="status">
						<span class="icono" aria-hidden="true">key</span>
						<span class="clave-nueva"><b>Copiá la clave ahora: no se vuelve a&nbsp;mostrar.</b><span class="url">{clave}</span></span>
						<Copiar texto={clave} />
					</div>
				</div>
			{:else}
				<div class="mensaje-clave"><Mensaje {form} que="clave" /></div>
			{/if}

			{#if mcp && elegida}
				<div class="direccion">
					<span class="g-rotulo">Dirección del servidor MCP</span>
					<span class="caja-direccion"><span class="url">{mcp.url}</span><Copiar texto={mcp.url} /></span>
					<form method="POST" action="?/regenerar" use:enhance={confirmar('¿Generar una clave nueva? La anterior deja de andar al instante.')} class="clave">
						<input type="hidden" name="id" value={elegida.id} />
						<input type="hidden" name="nombre" value={elegida.nombre} />
						{#each elegida.permisos as p (p)}<input type="hidden" name="permiso" value={p} />{/each}
						<span>La clave (termina en <b class="num">{elegida.pista}</b>) va aparte y sólo se muestra una&nbsp;vez.</span>
						<button class="g-accion">Generar una nueva</button>
					</form>
				</div>

				<form method="POST" action="?/permisos" use:enhance={sinReset} class="permisos" aria-labelledby="t-permisos">
					<span id="t-permisos" class="g-rotulo">Qué puede hacer</span>
					<input type="hidden" name="id" value={elegida.id} />
					{#each mcp.permisos as p (p.clave)}
						{@const on = elegida.permisos.includes(p.clave)}
						<label class="permiso">
							<span class="icono lleno" class:apagado={!on} aria-hidden="true">{ICONO_PERMISO[p.clave] ?? 'tune'}</span>
							<span class="permiso-texto"><b>{p.nombre}</b><span>{on ? p.descripcion : 'Apagado'}</span></span>
							<input type="checkbox" role="switch" class="g-interruptor" name="permiso" value={p.clave} checked={on} onchange={mandar} />
						</label>
					{/each}
					<button class="visualmente-oculto">Guardar permisos</button>
					<Mensaje {form} que="permisos" />
				</form>
				<form method="POST" action="?/revocar" use:enhance={confirmar(`¿Desconectar ${elegida.nombre}? Su clave deja de andar al instante.`)} class="revocar">
					<input type="hidden" name="id" value={elegida.id} />
					<button class="g-accion peligro">Desconectar {elegida.nombre}</button>
				</form>
			{:else}
				<div class="direccion">
					<span class="g-rotulo">Cómo funciona</span>
					<ol class="pasos">
						<li class="g-paso ahora"><span class="g-paso-num">1</span><span class="g-paso-cuerpo"><b>Activalo</b><span>Te damos la dirección de tu servidor MCP y una&nbsp;clave.</span></span></li>
						<li class="g-paso"><span class="g-paso-num">2</span><span class="g-paso-cuerpo"><b>Sumalo a tu asistente</b><span>Claude, ChatGPT, Gemini u otro que acepte&nbsp;MCP.</span></span></li>
						<li class="g-paso"><span class="g-paso-num">3</span><span class="g-paso-cuerpo"><b>Elegí qué puede hacer</b><span>Cada permiso se prende o se apaga con un&nbsp;interruptor.</span></span></li>
					</ol>
					{#if !mcp}
						<p class="g-nota">El asistente de IA no está disponible en este momento. Probá de nuevo más&nbsp;tarde.</p>
					{/if}
				</div>
			{/if}
		</section>

		<div class="g-col derecha">
			<section class="g-bloque uso" aria-labelledby="t-uso">
				<h2 id="t-uso" class="g-rotulo">Así se usa</h2>
				<span class="burbuja yo">¿Qué huecos tengo mañana a la&nbsp;tarde?</span>
				<span class="burbuja ia-burbuja">
					{#if data.ejemplo.huecos.length}
						Mañana {data.ejemplo.dia} tenés {cuantos[data.ejemplo.huecos.length]} de&nbsp;tarde:
						<span class="huecos">
							{#each data.ejemplo.huecos as h (h.hora + h.quien)}<span class="hueco">{h.hora}{h.quien ? ` · ${h.quien}` : ''}</span>{/each}
						</span>
						¿Querés que se los ofrezca a la lista de&nbsp;espera?
					{:else}
						Mañana {data.ejemplo.dia} no te quedan huecos de tarde. ¿Querés que mire&nbsp;pasado?
					{/if}
				</span>
				<span class="g-sub">Lo que cambia algo (cargar, mover, cancelar) siempre te pide confirmación&nbsp;antes.</span>
			</section>

			<section class="g-bloque sin-pad publico" aria-labelledby="t-publico">
				<form method="POST" action="?/publico" use:enhance={sinReset} class="publico-cabeza">
					<span class="icono lleno" aria-hidden="true">travel_explore</span>
					<span class="publico-texto">
						<h2 id="t-publico">La IA de tus clientes</h2>
						<span>Que sus asistentes encuentren {nombreCorto} y&nbsp;reserven</span>
					</span>
					<input
						type="checkbox"
						role="switch"
						class="g-interruptor grande"
						name="activo"
						aria-label="La IA de tus clientes puede reservar"
						checked={!!data.publico?.activo}
						disabled={!data.publico}
						onchange={mandar}
					/>
					<button class="visualmente-oculto" disabled={!data.publico}>Guardar</button>
				</form>
				<ul class="puede">
					<li><span class="icono lleno" aria-hidden="true">spa</span>Ver servicios y precios</li>
					<li><span class="icono lleno" aria-hidden="true">event_available</span>Ver horarios libres</li>
					<li><span class="icono lleno" aria-hidden="true">bookmark_added</span>Reservar con seña por link</li>
				</ul>
				<div class="publico-pie">
					{#if data.publico}
						<b class="num">{data.publico.reservasMes}</b>
						<span>
							{data.publico.reservasMes === 1 ? 'reserva llegó' : 'reservas llegaron'} por un asistente de IA este mes. Pagaron la seña por el link de Mercado Pago, como cualquier&nbsp;reserva.
						</span>
					{:else}
						<span class="icono" aria-hidden="true">info</span>
						<span>Todavía no está disponible para tu negocio. Cuando lo esté, las reservas van a pagar la seña por el link de Mercado Pago, como cualquier&nbsp;reserva.</span>
					{/if}
				</div>
				<div class="mensaje-publico"><Mensaje {form} que="publico" /></div>
			</section>

			<section class="g-bloque registro" aria-labelledby="t-registro">
				<h2 id="t-registro" class="g-rotulo">Lo último que hizo la IA</h2>
				{#if lineas.length}
					<ul class="lineas">
						{#each lineas as r (r.id)}
							<li>
								<span class="num">{cuando(r.creadoEn)}</span>
								{#if r.error === undefined}
									<span><b>{r.quien}</b> {r.texto}</span>
								{:else}
									<span class="fallo"><span class="icono" aria-hidden="true">error</span><span><b>{r.quien}</b> no pudo {r.texto}: {r.error}</span></span>
								{/if}
							</li>
						{/each}
					</ul>
				{:else}
					<p class="g-nota">Todavía no hizo nada. Acá vas a ver cada cosa que haga, con la hora y quién la&nbsp;pidió.</p>
				{/if}
			</section>
		</div>
	</div>

	<h2 class="g-rotulo otras-titulo">Otras conexiones</h2>
	<ul class="otras">
		{#each otras as o (o.t)}
			<li>
				<a class="otra" href={o.href}>
					<span class="logo">
						{#if o.logo}<img src={o.logo} alt="" width="22" height="22" />{:else}<span class="icono lleno" aria-hidden="true">{o.icono}</span>{/if}
					</span>
					<span class="otra-texto">
						<b>{o.t}</b>
						<span class="estado" class:ok={o.ok}>
							<span class="icono" class:lleno={o.ok} aria-hidden="true">{o.ok ? 'check_circle' : 'radio_button_unchecked'}</span>{o.estado}
						</span>
					</span>
					<span class="otra-accion">{o.accion}</span>
				</a>
			</li>
		{/each}
	</ul>
</div>

<style>
	.integraciones {
		gap: 20px;
	}
	.cabeza {
		gap: 8px;
	}
	.bajada {
		max-width: 44em;
		color: var(--tl-sub);
	}
	.dos {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 16px;
		align-items: start;
	}
	.asistente {
		border-radius: 22px;
		background: var(--tl-papel);
		overflow: hidden;
		box-shadow: 0 20px 40px -30px rgba(15, 35, 40, 0.5);
		min-width: 0;
	}
	.asistente-cabeza {
		padding: 20px;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.fila-titulo {
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.robot {
		font-size: 26px;
		color: var(--tl-menta);
	}
	.fila-titulo h2 {
		flex: 1;
		margin: 0;
		font-size: 18px;
		font-weight: 700;
		letter-spacing: normal;
	}
	.estado-oscuro {
		display: inline-flex;
		align-items: center;
		gap: 5px;
		height: 26px;
		padding: 0 10px 0 8px;
		border-radius: 999px;
		background: rgba(60, 204, 144, 0.18);
		color: var(--tl-menta);
		font-size: 12px;
		font-weight: 700;
		white-space: nowrap;
	}
	.estado-oscuro .icono {
		font-size: 15px;
	}
	.estado-oscuro.espera {
		background: rgba(251, 233, 194, 0.18);
		color: #fbe9c2;
	}
	.estado-oscuro.apagado {
		background: rgba(255, 255, 255, 0.12);
		color: rgba(255, 255, 255, 0.85);
	}
	.asistente-cabeza p {
		margin: 0;
		font-size: 14px;
		line-height: 1.45;
		color: rgba(255, 255, 255, 0.8);
	}
	.asistente-cabeza form {
		margin: 0;
	}
	.asistentes {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
	}
	.asist {
		position: relative;
		display: inline-flex;
		align-items: center;
		gap: 6px;
		min-height: 34px;
		padding: 0 12px;
		border-radius: 999px;
		background: rgba(255, 255, 255, 0.1);
		color: #fff;
		font-size: 13px;
		font-weight: 700;
		cursor: pointer;
	}
	.asist.on {
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
	}
	.asist .icono {
		font-size: 15px;
	}
	.direccion {
		padding: 16px 20px;
		display: flex;
		flex-direction: column;
		gap: 10px;
		border-bottom: 1px solid #f0e8d8;
	}
	.caja-direccion {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 6px 6px 6px 14px;
		border-radius: 12px;
		background: var(--tl-crema);
	}
	.url {
		flex: 1;
		min-width: 0;
		font-size: 14px;
		font-weight: 700;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		font-variant-numeric: tabular-nums;
	}
	.clave {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0 6px;
		font-size: 12px;
		color: var(--tl-sub);
	}
	.clave .g-accion {
		font-size: 12px;
		min-height: 32px;
	}
	.clave-nueva {
		display: flex;
		flex-direction: column;
		gap: 4px;
		min-width: 0;
	}
	.clave-nueva .url {
		white-space: normal;
		overflow-wrap: anywhere;
	}
	.pasos {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.permisos {
		padding: 8px 20px 12px;
		display: flex;
		flex-direction: column;
	}
	.permisos > .g-rotulo {
		padding: 8px 0;
	}
	.permiso {
		display: flex;
		align-items: center;
		gap: 12px;
		min-height: 50px;
		border-top: 1px solid #f0e8d8;
		cursor: pointer;
	}
	.permiso > .icono {
		font-size: 20px;
		color: var(--tl-menta-texto);
	}
	.permiso > .icono.apagado {
		color: #8a9794;
	}
	.permiso-texto {
		flex: 1;
		display: flex;
		flex-direction: column;
		padding: 6px 0;
	}
	.permiso-texto b {
		font-size: 14px;
	}
	.permiso-texto span {
		font-size: 12px;
		color: var(--tl-sub);
	}
	.derecha {
		gap: 16px;
	}
	.uso {
		padding: 18px;
		border-radius: 22px;
		gap: 10px;
	}
	.uso h2,
	.registro h2,
	.otras-titulo {
		margin: 0;
	}
	.burbuja {
		padding: 10px 14px;
		font-size: 14px;
		line-height: 1.45;
		text-wrap: pretty;
	}
	.burbuja.yo {
		align-self: flex-end;
		max-width: 80%;
		border-radius: 16px 16px 4px 16px;
		background: var(--tl-petroleo);
		color: #fff;
	}
	.ia-burbuja {
		align-self: flex-start;
		max-width: 90%;
		padding: 12px 14px;
		border-radius: 16px 16px 16px 4px;
		background: var(--tl-crema);
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.huecos {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
	}
	.hueco {
		display: inline-flex;
		align-items: center;
		height: 30px;
		padding: 0 10px;
		border-radius: 8px;
		background: repeating-linear-gradient(-45deg, rgba(60, 204, 144, 0.25) 0 2px, transparent 2px 8px), #f3fbf7;
		box-shadow: inset 0 0 0 1px rgba(23, 128, 90, 0.35);
		font-size: 12px;
		font-weight: 700;
		color: var(--tl-menta-texto);
		font-variant-numeric: tabular-nums;
	}
	.publico {
		border-radius: 22px;
	}
	.publico-cabeza {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 18px 18px 12px;
		margin: 0;
	}
	.publico-cabeza > .icono {
		font-size: 24px;
		color: var(--tl-menta-texto);
	}
	.publico-texto {
		flex: 1;
		display: flex;
		flex-direction: column;
		min-width: 0;
	}
	.publico-texto h2 {
		margin: 0;
		font-size: 16px;
		font-weight: 700;
		letter-spacing: normal;
	}
	.publico-texto span {
		font-size: 13px;
		color: var(--tl-sub);
	}
	.g-interruptor.grande {
		width: 48px;
		height: 28px;
	}
	.g-interruptor.grande::after {
		width: 22px;
		height: 22px;
	}
	.g-interruptor.grande:checked::after {
		transform: translateX(20px);
	}
	.puede {
		list-style: none;
		margin: 0;
		padding: 0 18px 14px;
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 8px;
	}
	.puede li {
		display: flex;
		flex-direction: column;
		gap: 6px;
		padding: 12px;
		border-radius: 14px;
		background: var(--tl-crema);
		font-size: 12px;
		line-height: 1.35;
	}
	.puede .icono {
		font-size: 20px;
		color: var(--tl-menta-texto);
	}
	.publico-pie {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 14px 18px;
		border-top: 1px solid #f0e8d8;
		font-size: 13px;
		line-height: 1.4;
		color: var(--tl-sub-2);
	}
	.publico-pie b {
		font-size: 26px;
		font-weight: 700;
		letter-spacing: -0.04em;
		color: var(--tl-menta-texto);
	}
	.publico-pie .icono {
		color: var(--tl-menta-texto);
	}
	.mensaje-publico:not(:empty) {
		padding: 0 18px 14px;
	}
	.registro {
		padding: 14px 18px;
		gap: 8px;
	}
	.lineas {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.lineas li {
		display: grid;
		grid-template-columns: 56px minmax(0, 1fr);
		gap: 8px;
		font-size: 13px;
		line-height: 1.4;
	}
	.lineas li > .num {
		color: var(--tl-sub);
	}
	.fallo {
		display: flex;
		gap: 4px;
		color: #a3241a;
	}
	.fallo .icono {
		font-size: 16px;
	}
	.revocar {
		padding: 0 20px 14px;
	}
	.mensaje-clave:not(:empty) {
		padding: 12px 20px 0;
	}
	.asistentes form {
		margin: 0;
	}
	.asistentes button.asist {
		border: 0;
		font-family: var(--tl-font);
	}
	a.asist {
		text-decoration: none;
	}
	.otras {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
		gap: 12px;
	}
	.otra {
		display: flex;
		align-items: center;
		gap: 12px;
		min-height: 72px;
		padding: 14px 16px;
		border-radius: 18px;
		background: var(--tl-papel);
		color: var(--tl-tinta);
		text-decoration: none;
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.otra:hover {
		transform: translateY(-2px);
	}
	.logo {
		width: 40px;
		height: 40px;
		flex: none;
		border-radius: 12px;
		background: var(--tl-crema);
		display: grid;
		place-items: center;
	}
	.logo .icono {
		font-size: 22px;
		color: var(--tl-menta-texto);
	}
	.otra-texto {
		flex: 1;
		display: flex;
		flex-direction: column;
		min-width: 0;
	}
	.otra-texto b {
		font-size: 14px;
	}
	.estado {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 12px;
		font-weight: 700;
		color: var(--tl-sub);
		overflow-wrap: anywhere;
	}
	.estado.ok {
		color: var(--tl-menta-texto);
	}
	.estado .icono {
		font-size: 14px;
	}
	.otra-accion {
		font-size: 13px;
		font-weight: 700;
		color: var(--tl-menta-texto);
	}

	@media (max-width: 960px) {
		.dos {
			grid-template-columns: minmax(0, 1fr);
		}
	}
	@media (max-width: 640px) {
		.asist {
			min-height: 44px;
		}
		.permiso {
			min-height: 56px;
		}
		.puede {
			grid-template-columns: minmax(0, 1fr);
		}
		.puede li {
			flex-direction: row;
			align-items: center;
		}
		.clave .g-accion {
			min-height: 44px;
		}
	}
</style>
