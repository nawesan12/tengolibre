<script lang="ts">
	import { enhance } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import { navigating, page } from '$app/state';
	import { dia as diaDe, pesos } from '#lib/formato.ts';
	import Icono from '#lib/ui/Icono.svelte';
	import PaginaNegocio from '#lib/publico/PaginaNegocio.svelte';
	import PanelReserva from '#lib/publico/PanelReserva.svelte';
	import Horarios from '#lib/publico/Horarios.svelte';
	import TiraDias from '#lib/publico/TiraDias.svelte';
	import { borrador } from '#lib/publico/borrador.svelte.ts';
	import { diaCorto, fechaLarga, mesesCortos, sumarDias } from '#lib/publico/calendario.ts';
	import { CUALQUIERA, urlDePaso, type Paso } from '#lib/publico/reserva.ts';
	import { duracion, linkWhatsapp, montoSena, sinViuda } from '#lib/publico/textos.ts';

	let { data, form } = $props();
	const n = $derived(data.negocio);
	const base = $derived(`/${n.slug}/reservar`);
	// Las acciones van a la misma URL con lo elegido: sin JavaScript, si fallan, la página vuelve al mismo paso.
	const accion = (nombre: string) => {
		const q = new URLSearchParams([...page.url.searchParams].filter(([k]) => !k.startsWith('/')));
		const s = q.toString();
		return `?${s ? s + '&' : ''}/${nombre}`;
	};
	// Los links llevan lo elegido explícito (lo que vino por defecto también: profesional y día).
	const ir = (paso: Paso, cambios: Record<string, string> = {}) => urlDePaso(base, page.url.searchParams, paso, cambios);
	const conProf = $derived<Record<string, string>>(data.profesionalId ? { profesional: data.profesionalId } : {});
	const aDia = (f: string) => ir('horario', { ...conProf, fecha: f });
	const aHora = (inicio: string, fecha = data.fecha) => ir('datos', { ...conProf, fecha, inicio });

	// Lo cargado vuelve si la acción falla; el cupón validado ajusta el total.
	const v = $derived((form?.valores ?? {}) as Record<string, string>);
	const cupon = $derived(form && 'cupon' in form ? form.cupon : null);
	const precio = $derived(data.servicio?.precio ?? 0);
	const descuento = $derived(cupon?.valido ? Math.min(cupon.descuento ?? 0, precio) : 0);
	const total = $derived(precio - descuento);
	const enEspera = $derived(!!form && ('anotado' in form || ('enEspera' in form && form.enEspera)));
	const sinDias = $derived(data.paso === 'dia' && data.dias.every((d) => d.libres === 0));
	const hoy = $derived(diaDe(new Date(), n.zonaHoraria));
	let enviando = $state(false);
	// El socio que ingresó reserva con su membresía (sus datos ya los tiene el negocio).
	let usar = $state(true);
	const conMembresia = $derived(!!data.membresia && usar);
	const sena = $derived(conMembresia ? 0 : montoSena(total, data.cobro));
	const resto = $derived(total - sena);
	const volver = $derived(encodeURIComponent(page.url.pathname + page.url.search));
	const abierto = (si: boolean | undefined) => (si ? { open: true } : {});
	const listo = $derived(data.paso === 'datos' && !!data.servicio && !!data.horario);
	const nombreProf = (ids: string[]) => {
		const ns = ids.map((id) => data.profesionales.find((p) => p.id === id)?.nombre.split(' ')[0]).filter(Boolean);
		return ns.length === 1 ? `con ${ns[0]}` : '';
	};
	const penalizado = $derived(!!form && 'codigo' in form && form.codigo === 'cliente_penalizado');
	// Mientras busca los horarios de otro día o de otra persona: el esqueleto con la forma real.
	const buscando = $derived(
		!!navigating.to &&
			navigating.to.url.pathname === page.url.pathname &&
			(navigating.to.url.searchParams.get('fecha') !== page.url.searchParams.get('fecha') ||
				navigating.to.url.searchParams.get('profesional') !== page.url.searchParams.get('profesional'))
	);
	// Las alternativas más cerca de la hora que se ocupó.
	const cerca = $derived.by(() => {
		if (!data.ocupado) return [];
		const m = (h: string) => Number(h.slice(0, 2)) * 60 + Number(h.slice(3, 5));
		const o = m(data.ocupado);
		return [...data.horarios].sort((a, b) => Math.abs(m(a.hora) - o) - Math.abs(m(b.hora) - o)).slice(0, 3).sort((a, b) => a.hora.localeCompare(b.hora));
	});
	const textoBoton = $derived(
		enviando ? 'Reservando…' : sena > 0 ? (sena >= total ? 'Reservar y pagar' : 'Reservar y pagar seña') : 'Reservar'
	);
	const valor = (k: 'nombre' | 'telefono' | 'email' | 'nota') => v[k] ?? borrador[k];
	const guardar = (k: 'nombre' | 'telefono' | 'email' | 'nota') => (e: Event) => {
		borrador[k] = (e.currentTarget as HTMLInputElement).value;
	};
	const etiquetaPlan = (plan: string) => (/^plan\b/i.test(plan) ? `Usar mi ${plan}` : `Usar mi membresía ${plan}`);
</script>

<svelte:head>
	<title>Reservar turno · {n.nombre}</title>
	<meta name="description" content={`Elegí servicio, día y horario y reservá tu turno en ${n.nombre}.`} />
	<meta name="theme-color" content={data.pagina.web.colorPrimario} />
</svelte:head>

{#snippet espera(desde: string, hasta: string, cta: string)}
	{#if form && 'anotado' in form && form.anotado}
		<p class="aviso-w verde" role="status">
			<Icono nombre="hourglass_top" tam={20} lleno />
			<span>¡Listo! Te anotamos en la lista de espera para {form.anotado.servicio}. Si se libera un lugar, te lo ofrecemos a vos&nbsp;primero.</span>
		</p>
	{:else}
		<p class="aviso-w verde">
			<Icono nombre="hourglass_top" tam={20} lleno />
			<span>Anotate en la lista de espera: si alguien cancela, te ofrecemos el lugar a vos primero, con 30 minutos para&nbsp;tomarlo.</span>
		</p>
		<!-- Sólo se abre solo (sin cerrar lo que la persona abrió antes de que cargue la página). -->
		<details class="espera" {...abierto(enEspera)}>
			<summary class="cta-w">{cta}</summary>
			<form method="POST" action={accion('espera')} use:enhance class="form-w">
				<input type="hidden" name="servicioId" value={data.servicio?.id} />
				<input type="hidden" name="profesionalId" value={data.profesionalId} />
				{#if form && 'enEspera' in form && form.error}<p class="aviso-w rojo" role="alert"><Icono nombre="error" tam={20} lleno /><span>{form.error}</span></p>{/if}
				<label class="campo-w">Nombre <input name="nombre" required maxlength="80" autocomplete="name" value={v.nombre ?? borrador.nombre} /></label>
				<label class="campo-w">Email <input name="email" type="email" required maxlength="120" autocomplete="email" value={v.email ?? borrador.email} /></label>
				<label class="campo-w">Teléfono (opcional) <input name="telefono" type="tel" maxlength="30" autocomplete="tel" value={v.telefono ?? borrador.telefono} /></label>
				<fieldset class="momento">
					<legend>¿Qué momento te sirve?</legend>
					{#each [['cualquiera', 'Cualquiera'], ['manana', 'Mañana'], ['tarde', 'Tarde']] as [valor, nombre] (valor)}
						<label class="opcion"><input type="radio" name="momento" value={valor} checked={(v.momento ?? 'cualquiera') === valor} />{nombre}</label>
					{/each}
				</fieldset>
				<div class="dos">
					<label class="campo-w">Desde <input name="desde" type="date" required min={hoy} value={v.desde ?? desde} /></label>
					<label class="campo-w">Hasta <input name="hasta" type="date" required min={hoy} value={v.hasta ?? hasta} /></label>
				</div>
				<button class="boton-reserva">Anotarme</button>
			</form>
		</details>
	{/if}
{/snippet}

{#snippet hueco(href: string, h: string, texto: string)}
	<a class="hueco-w" {href} data-sveltekit-reset="false"><b class="tl-num">{h}</b><span>{texto}</span><Icono nombre="arrow_forward" tam={18} /></a>
{/snippet}

<main>
	<PaginaNegocio
		pagina={data.pagina}
		resenas={data.listaResenas}
		planes={data.planes}
		cobro={data.cobro}
		proximo={data.primer}
		vista="reservar"
		elegido={data.servicio?.id ?? ''}
	>
		{#snippet reserva()}
			<PanelReserva primer={data.primer} volver={`/${n.slug}`} nombre={n.nombre}>
				{#if data.aviso}<p class="aviso-w rojo paso-aviso" role="alert"><Icono nombre="error" tam={20} lleno /><span>{data.aviso}</span></p>{/if}

				<!-- 01 · Servicio -->
				<section class="paso" aria-labelledby="t-p1">
					<div class="paso-cab">
						<span class="paso-n">01</span>
						<h2 id="t-p1">Qué te hacés</h2>
						{#if data.servicio}<a class="cambiar" href={ir('servicio')} aria-label="Cambiar servicio">Cambiar</a>{/if}
					</div>
					{#if data.servicio}
						<div class="elegido">
							<span>{data.servicio.nombre}</span><span class="puntos" aria-hidden="true"></span><b class="tl-num">{pesos(precio)}</b>
						</div>
						<span class="chico sub tl-num">{duracion(data.servicio.duracion)}</span>
					{:else}
						{#if data.servicios.length === 0}<p class="sub">Este negocio todavía no cargó&nbsp;servicios.</p>{/if}
						<div class="opciones">
							{#each data.servicios as s (s.id)}
								<a class="opcion" href={ir('profesional', { servicio: s.id })} data-sveltekit-reset="false">{s.nombre}</a>
							{/each}
						</div>
					{/if}
				</section>

				{#if data.servicio && data.profesionales.length}
					<!-- 02 · Con quién -->
					<section class="paso" aria-labelledby="t-p2">
						<div class="paso-cab"><span class="paso-n">02</span><h2 id="t-p2">Con quién</h2></div>
						{#if data.profesionales.length === 1}
							<span class="texto">{sinViuda(`Con ${data.profesionales[0].nombre}, que es quien lo hace.`)}</span>
						{:else}
							<div class="opciones">
								<a class="opcion" href={ir('dia', { profesional: CUALQUIERA })} data-sveltekit-reset="false" aria-current={data.profesionalId === CUALQUIERA ? 'true' : undefined}>Cualquiera</a>
								{#each data.profesionales as p (p.id)}
									<a class="opcion" href={ir('dia', { profesional: p.id })} data-sveltekit-reset="false" aria-current={data.profesionalId === p.id ? 'true' : undefined}>
										{#if p.foto}<img src={p.foto} alt="" width="28" height="28" />{/if}{p.nombre}
									</a>
								{/each}
							</div>
						{/if}
					</section>
				{/if}

				{#if data.servicio && data.profesionalId}
					<!-- 03 · Cuándo -->
					<section class="paso" aria-labelledby="t-p3">
						<div class="paso-cab">
							<span class="paso-n">03</span>
							<h2 id="t-p3">Cuándo</h2>
							{#if data.dias.length}<span class="paso-extra">{mesesCortos(data.dias.map((d) => d.fecha))}</span>{/if}
						</div>
						{#if sinDias}
							<span class="kicker tl-num">SIN LUGAR</span>
							<h3 class="titulo-estado">No quedan lugares en los próximos&nbsp;días</h3>
							{@render espera(hoy, sumarDias(hoy, 14), 'Anotarme en la lista de espera')}
						{:else}
							<TiraDias dias={data.dias} elegida={data.fecha} href={aDia} />
							{#if buscando}
								<span class="chico sub">Buscando horarios…</span>
								<div class="esqueletos" aria-hidden="true">
									{#each [1, 0.8, 0.6, 0.4] as o (o)}<span style:opacity={o}></span>{/each}
								</div>
							{:else if data.horarios.length}
								{#if data.ocupado}
									<div class="estado" role="alert">
										<span class="kicker tl-num">PASO 04 · CUÁNDO</span>
										<h3 class="titulo-estado">Las {data.ocupado} se ocuparon&nbsp;recién</h3>
										<p class="texto">Tus datos quedan guardados. Estos siguen libres, cerca de lo que&nbsp;elegiste:</p>
										{#each cerca as h (h.inicio)}
											{@render hueco(aHora(h.inicio), h.hora, [fechaLarga(data.fecha), nombreProf(h.profesionales)].filter(Boolean).join(' · '))}
										{/each}
									</div>
								{/if}
								<span class="chico sub">{fechaLarga(data.fecha)} · {data.horarios.length} {data.horarios.length === 1 ? 'horario libre' : 'horarios libres'}</span>
								<Horarios horarios={data.horarios} href={(h) => aHora(h.inicio)} actual={data.horario?.inicio ?? ''} />
							{:else if data.fecha}
								<span class="kicker tl-num">{diaCorto(data.fecha)} · LLENO</span>
								<h3 class="titulo-estado">Ese día ya no hay&nbsp;lugar</h3>
								{#if data.cercano}
									{@render hueco(
										aHora(data.cercano.horario.inicio, data.cercano.fecha),
										`${diaCorto(data.cercano.fecha)} · ${data.cercano.horario.hora}`,
										['Lo más cerca', nombreProf(data.cercano.horario.profesionales)].filter(Boolean).join(', ')
									)}
								{/if}
								{@render espera(data.fecha < hoy ? hoy : data.fecha, data.fecha < hoy ? hoy : data.fecha, `Anotarme para el ${fechaLarga(data.fecha < hoy ? hoy : data.fecha).split(' ')[0].toLowerCase()}`)}
							{/if}
						{/if}
					</section>
				{/if}

				{#if listo && data.servicio && data.horario}
					<!-- 04 · Tus datos -->
					<section class="paso" aria-labelledby="t-p4">
						<div class="paso-cab"><span class="paso-n">04</span><h2 id="t-p4">Tus datos</h2></div>
						<form
							id="f-reserva"
							method="POST"
							action={accion('reservar')}
							class="form-w"
							use:enhance={({ action }) => {
								enviando = action.searchParams.has('/reservar');
								return async ({ update, result }) => {
									await update({ reset: false });
									// Se ocupó mientras completaba: se vuelve a pedir la agenda (y lo cargado queda en el borrador).
									if (result.type === 'failure' && result.data?.codigo === 'horario_ocupado') await invalidateAll();
									enviando = false;
								};
							}}
						>
							<input type="hidden" name="servicioId" value={data.servicio.id} />
							<input type="hidden" name="profesionalId" value={data.profesionalId} />
							<input type="hidden" name="inicio" value={data.horario.inicio} />
							{#if penalizado}
								<div class="estado" role="alert">
									<h3 class="titulo-estado">Por ahora no podés reservar&nbsp;online</h3>
									<p class="aviso-w ambar"><Icono nombre="block" tam={20} lleno /><span>{form?.error}</span></p>
									{#if n.whatsapp}<a class="cta-w" href={linkWhatsapp(n.whatsapp)} target="_blank" rel="noopener">Escribir a {n.nombre} por WhatsApp</a>{/if}
									{#if n.telefono}<a class="cta-2" href={`tel:${n.telefono.replace(/[^\d+]/g, '')}`}>Llamar al {n.telefono}</a>{/if}
								</div>
							{:else if form && 'error' in form && form.error && !('enEspera' in form) && form.codigo !== 'horario_ocupado'}
								<p class="aviso-w rojo" role="alert"><Icono nombre="error" tam={20} lleno /><span>{form.error}</span></p>
							{/if}
							{#if data.membresia}
								<label class="socio">
									<input type="checkbox" name="usarMembresia" value="1" bind:checked={usar} />
									<span class="pila2">
										<b>{etiquetaPlan(data.membresia.plan)}</b>
										<span>{data.membresia.saldo === 1 ? 'Te queda 1 visita' : `Te quedan ${data.membresia.saldo} visitas`} · sin&nbsp;seña</span>
									</span>
								</label>
							{:else if data.conPlanes && !data.ingresado}
								<div class="invitar">
									<Icono nombre="card_membership" tam={20} lleno />
									<span>¿Sos socio?</span>
									<a href={`/ingresar?volver=${volver}`}>Ingresá</a>
								</div>
							{/if}
							{#if !conMembresia}
								<label class="visualmente-oculto" for="r-nombre">Nombre y apellido</label>
								<input
									id="r-nombre"
									class="entrada"
									name="nombre"
									required
									maxlength="80"
									autocomplete="name"
									placeholder="Nombre y apellido"
									value={valor('nombre')}
									oninput={guardar('nombre')}
									aria-invalid={!!form?.campos?.['cliente.nombre']}
								/>
								<label class="visualmente-oculto" for="r-telefono">Teléfono</label>
								<input
									id="r-telefono"
									class="entrada"
									name="telefono"
									type="tel"
									maxlength="30"
									autocomplete="tel"
									inputmode="tel"
									placeholder="Teléfono"
									value={valor('telefono')}
									oninput={guardar('telefono')}
									aria-invalid={!!form?.campos?.['cliente.telefono']}
									aria-describedby="ayuda-contacto"
								/>
								<label class="visualmente-oculto" for="r-email">Email</label>
								<input
									id="r-email"
									class="entrada"
									name="email"
									type="email"
									maxlength="120"
									autocomplete="email"
									placeholder="Email"
									value={valor('email')}
									oninput={guardar('email')}
									aria-invalid={!!form?.campos?.['cliente.email']}
									aria-describedby="ayuda-contacto"
								/>
								<span id="ayuda-contacto" class="chico sub">Teléfono o email: con uno alcanza. Ahí te llega la&nbsp;confirmación.</span>
							{/if}
							<details {...abierto(!!(v.cupon || v.referido || v.nota || v.nacimiento || data.referido || form?.campos?.["cliente.nacimiento"]))}>
								<summary class="extra">
									<Icono nombre="sell" tam={18} lleno />Cupón, nota o código de referido
									{#if descuento > 0}<span class="ok tl-num">− {pesos(descuento)}</span>{/if}
								</summary>
								<div class="form-w extra-cuerpo">
									<div class="fila-cupon">
										<label class="visualmente-oculto" for="r-cupon">Cupón</label>
										<input id="r-cupon" class="entrada cupon" name="cupon" maxlength="40" autocomplete="off" autocapitalize="characters" placeholder="Cupón" value={v.cupon ?? ''} />
										<button class="aplicar" formaction={accion('cupon')} formnovalidate>Aplicar</button>
									</div>
									{#if cupon}
										<p class={cupon.valido ? 'aviso-w verde' : 'aviso-w rojo'} role="status">
											<Icono nombre={cupon.valido ? 'check_circle' : 'error'} tam={20} lleno />
											<span>{cupon.valido ? `Cupón aplicado: ${pesos(descuento)} de descuento.` : cupon.motivo || 'Ese cupón no sirve.'}</span>
										</p>
									{/if}
									<label class="campo-w">
										Código de referido
										<input name="referido" maxlength="40" autocomplete="off" autocapitalize="characters" value={v.referido ?? data.referido} />
										<span class="chico sub">Si te recomendó alguien, poné su&nbsp;código.</span>
									</label>
									<label class="campo-w">
										Tu cumpleaños (opcional)
										<input name="nacimiento" type="date" min="1900-01-01" max={hoy} value={v.nacimiento ?? ''} aria-describedby="ayuda-cumple" />
										<span id="ayuda-cumple" class="chico sub">Para saludarte ese&nbsp;día.</span>
									</label>
									<label class="campo-w">
										Algo que {n.nombre} tenga que saber
										<textarea name="nota" maxlength="500" rows="2" oninput={guardar('nota')}>{valor('nota')}</textarea>
									</label>
								</div>
							</details>

							<div class="ticket">
								<div class="ticket-arriba">
									<div class="renglon"><span>{data.servicio.nombre}</span><span class="tl-num">{pesos(precio)}</span></div>
									{#if descuento > 0}
										<div class="renglon ok"><span>{cupon?.codigo ?? 'Cupón'}</span><span class="tl-num">− {pesos(descuento)}</span></div>
									{/if}
								</div>
								<div class="troquel" aria-hidden="true"><span></span></div>
								<div class="ticket-abajo">
									<div class="renglon total"><b>Total</b><b class="tl-num">{conMembresia ? 'Con tu membresía' : pesos(total)}</b></div>
									{#if sena > 0}
										<span class="chico sub">
											{#if sena >= total}
												Hoy pagás el total con Mercado&nbsp;Pago.
											{:else}
												Hoy pagás la seña de <b class="tl-num">{pesos(sena)}</b> con Mercado Pago; el resto, en el&nbsp;local.
											{/if}
										</span>
									{/if}
									{#if data.politicaCancelacion}<span class="chico sub pre">{data.politicaCancelacion}</span>{/if}
								</div>
							</div>
						</form>
					</section>
				{/if}

				{#snippet barra()}
					<span class="barra-texto">
						{#if listo && data.horario}
							<b>{diaCorto(data.fecha)} · {data.horario.hora}</b>
							<span>{conMembresia ? 'Con tu membresía' : sena > 0 && sena < total ? `Total ${pesos(total)} · seña ${pesos(sena)}` : `Total ${pesos(total)}`}</span>
						{:else if data.servicio}
							<b>{data.servicio.nombre}</b>
							<span>Elegí día y horario</span>
						{:else}
							<b>Elegí un servicio</b>
							<span>{data.servicios.length} {data.servicios.length === 1 ? 'servicio' : 'servicios'} para&nbsp;elegir</span>
						{/if}
					</span>
					{#if listo}
						<button class="boton-reserva" form="f-reserva" disabled={enviando}>{textoBoton}</button>
					{:else}
						<button class="boton-reserva" type="button" disabled>Reservar</button>
					{/if}
				{/snippet}
			</PanelReserva>
		{/snippet}
	</PaginaNegocio>
</main>

<style>
	.sub {
		color: var(--w-sub);
	}
	.chico {
		font-size: 13px;
	}
	.pre {
		white-space: pre-line;
	}
	.pila2 {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}
	.texto {
		margin: 0;
		font-size: 15px;
		line-height: 1.5;
		color: var(--w-sub2);
		text-wrap: pretty;
	}
	.paso-aviso {
		margin: 12px 18px 0;
	}
	.elegido {
		display: flex;
		align-items: baseline;
		gap: 8px;
		font-size: 15px;
	}
	.elegido b {
		font: 700 15px var(--tl-font);
		white-space: nowrap;
	}
	.puntos {
		flex: 1;
		min-width: 16px;
		border-bottom: 2px dotted var(--w-line2);
		transform: translateY(-4px);
	}
	.opcion img {
		width: 28px;
		height: 28px;
		border-radius: 50%;
		object-fit: cover;
		margin: 0 8px 0 -6px;
	}
	.kicker {
		font: 700 11px var(--tl-font);
		letter-spacing: 0.08em;
		color: var(--w-ok);
	}
	.titulo-estado {
		margin: 0;
		font-size: 24px;
		line-height: 1.05;
		font-weight: var(--w-tw);
		letter-spacing: -0.04em;
	}
	.estado {
		display: flex;
		flex-direction: column;
		gap: 10px;
		padding-bottom: 4px;
	}
	.esqueletos {
		display: grid;
		gap: 6px;
	}
	.esqueletos span {
		height: 48px;
		border-radius: 12px;
		background: linear-gradient(90deg, var(--w-chip) 0, var(--w-papel) 50%, var(--w-chip) 100%);
		background-size: 400px 100%;
		animation: tl-brillo 1.4s linear infinite;
	}
	@keyframes tl-brillo {
		from {
			background-position: -200px 0;
		}
		to {
			background-position: 200px 0;
		}
	}
	.hueco-w {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px 12px;
		min-height: 50px;
		padding: 8px 14px;
		border-radius: 14px;
		background: repeating-linear-gradient(-45deg, var(--w-raya) 0 2px, transparent 2px 10px), var(--w-tint);
		background-size: 28.28px 28.28px, auto;
		box-shadow: inset 0 0 0 1.5px var(--w-troquel);
		color: var(--w-ink);
		text-decoration: none;
		animation: tl-raya-w 4s linear infinite;
	}
	.hueco-w b {
		font: 700 14px var(--tl-font);
		color: var(--w-marca);
	}
	.hueco-w span {
		flex: 1;
		font-size: 13px;
		font-weight: 700;
	}
	@keyframes tl-raya-w {
		to {
			background-position: 28.28px 0, 0 0;
		}
	}
	.aviso-w {
		display: flex;
		gap: 10px;
		margin: 0;
		padding: 14px;
		border-radius: 16px;
		font-size: 14px;
		line-height: 1.45;
		text-wrap: pretty;
	}
	.aviso-w.verde {
		background: #e3f6ec;
		color: #0f3d2b;
	}
	.aviso-w.ambar {
		background: #fbe9c2;
		color: #5c3800;
	}
	.aviso-w.rojo {
		background: #fbe0dc;
		color: #5b1a13;
	}
	.cta-w {
		display: grid;
		place-items: center;
		min-height: 54px;
		padding: 0 20px;
		border-radius: var(--w-br);
		background: var(--w-pri);
		color: var(--w-on);
		font-size: 16px;
		font-weight: 700;
		text-align: center;
		text-decoration: none;
		cursor: pointer;
		list-style: none;
	}
	.cta-w::-webkit-details-marker {
		display: none;
	}
	.cta-2 {
		display: grid;
		place-items: center;
		min-height: 48px;
		font-size: 15px;
		font-weight: 700;
		color: var(--w-sub);
		text-decoration: none;
	}
	.espera[open] > .cta-w {
		background: transparent;
		color: var(--w-ink);
		box-shadow: inset 0 0 0 1.5px var(--w-line2);
	}
	.espera .form-w {
		margin-top: 12px;
	}
	.form-w {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.momento {
		border: 0;
		margin: 0;
		padding: 0;
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
		min-width: 0;
	}
	.momento legend {
		width: 100%;
		margin-bottom: 6px;
		font-size: 14px;
		font-weight: 700;
	}
	.momento .opcion {
		position: relative;
		cursor: pointer;
	}
	.momento input {
		position: absolute;
		opacity: 0;
		width: 1px;
		height: 1px;
	}
	.momento .opcion:has(input:checked) {
		background: var(--w-pri);
		color: var(--w-on);
		box-shadow: none;
	}
	.momento .opcion:has(input:focus-visible) {
		outline: 3px solid var(--tl-menta);
		outline-offset: 2px;
	}
	.dos {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 8px;
	}
	.campo-w {
		display: flex;
		flex-direction: column;
		gap: 6px;
		font-size: 14px;
		font-weight: 700;
		min-width: 0;
	}
	.campo-w input,
	.campo-w textarea,
	.entrada {
		min-height: 50px;
		padding: 0 14px;
		border-radius: var(--w-sr);
		border: 0;
		background: var(--w-campo);
		color: var(--w-ink);
		box-shadow: inset 0 0 0 1.5px var(--w-line2);
		font: 400 16px var(--tl-font);
		width: 100%;
		min-width: 0;
	}
	.campo-w textarea {
		padding: 12px 14px;
		resize: vertical;
	}
	.entrada::placeholder {
		color: var(--w-sub);
	}
	.campo-w input:focus,
	.campo-w textarea:focus,
	.entrada:focus {
		outline: none;
		box-shadow: inset 0 0 0 2px var(--w-marca);
	}
	.entrada[aria-invalid='true'] {
		box-shadow: inset 0 0 0 2px #a3241a;
	}
	.socio {
		display: flex;
		align-items: flex-start;
		gap: 12px;
		padding: 14px;
		border-radius: var(--w-sr);
		background: var(--w-socios);
		cursor: pointer;
	}
	.socio input {
		width: 22px;
		height: 22px;
		margin: 0;
		accent-color: #183c44;
		flex: none;
	}
	.socio b {
		font-size: 15px;
	}
	.socio span span {
		font-size: 13px;
		color: var(--w-socios-txt);
	}
	.invitar {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 8px 8px 8px 12px;
		border-radius: var(--w-sr);
		background: var(--w-socios);
		font-size: 14px;
		color: var(--w-socios-txt);
	}
	.invitar span {
		flex: 1;
	}
	.invitar a {
		display: inline-flex;
		align-items: center;
		min-height: 44px;
		padding: 0 16px;
		border-radius: var(--w-br);
		background: var(--w-papel);
		color: var(--w-ink);
		font-size: 13px;
		font-weight: 700;
		text-decoration: none;
	}
	.extra {
		display: flex;
		align-items: center;
		gap: 8px;
		min-height: 48px;
		font-size: 14px;
		font-weight: 700;
		cursor: pointer;
		list-style: none;
	}
	.extra::-webkit-details-marker {
		display: none;
	}
	.extra > :global(.icono) {
		color: var(--w-sub);
	}
	.ok {
		color: var(--w-ok);
	}
	.extra .ok {
		margin-left: auto;
		font: 400 12px var(--tl-font);
	}
	.extra-cuerpo {
		padding-bottom: 6px;
	}
	.fila-cupon {
		display: flex;
		gap: 8px;
	}
	.cupon {
		flex: 1;
		border-radius: 12px;
		letter-spacing: 0.08em;
		font-variant-numeric: tabular-nums;
	}
	.aplicar {
		min-height: 50px;
		padding: 0 16px;
		border-radius: 12px;
		border: 0;
		box-shadow: inset 0 0 0 1.5px var(--w-line2);
		background: var(--w-campo);
		color: var(--w-ink);
		font: 700 14px var(--tl-font);
		cursor: pointer;
	}
	.ticket {
		border-radius: var(--w-sr);
		overflow: hidden;
		background: var(--w-chip);
	}
	.ticket-arriba {
		padding: 14px 16px;
		display: flex;
		flex-direction: column;
		gap: 6px;
		font-size: 14px;
	}
	.renglon {
		display: flex;
		justify-content: space-between;
		gap: 12px;
	}
	.troquel {
		height: 14px;
		position: relative;
		background:
			radial-gradient(circle at 0 50%, var(--w-papel) 7px, transparent 7.5px),
			radial-gradient(circle at 100% 50%, var(--w-papel) 7px, transparent 7.5px);
	}
	.troquel span {
		position: absolute;
		left: 12px;
		right: 12px;
		top: 6px;
		border-top: 2px dashed var(--w-line2);
	}
	.ticket-abajo {
		padding: 10px 16px 14px;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}
	.total {
		align-items: baseline;
	}
	.total b:first-child {
		font-size: 15px;
	}
	.total b:last-child {
		font: 700 20px var(--tl-font);
		letter-spacing: -0.03em;
	}
	.ticket-abajo .sub {
		font-size: 12px;
		line-height: 1.45;
	}
	.ticket-abajo .sub b {
		color: var(--w-ink);
	}
</style>
