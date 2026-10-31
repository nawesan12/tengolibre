<!--
	Nuevo turno (handoff «Panel v2», carga rápida; «Propuestas integradas» 03, turno fijo): cliente,
	servicio, con quién y cuándo, si se repite, y abajo el resumen con «Reservar».
-->
<script lang="ts">
	import { enhance } from '$app/forms';
	import Icono from '#lib/ui/Icono.svelte';
	import { dia, pesos } from '#lib/formato.ts';
	import { nombreDia, sumarDias } from '#lib/operacion/fechas.ts';
	import { MOTIVO_CHOCA, type FechaSerie } from '#lib/operacion/contrato.ts';
	import { iniciales, primerNombre } from '#lib/panel/dia.ts';
	import type { Encontrado } from '#lib/operacion/tipos.ts';

	let { data, form } = $props();

	let sobreturno = $state(false);
	let cada = $state(0);
	let hastaModo = $state<'cantidad' | 'fecha'>('cantidad');
	let enviando = $state(false);
	const servicio = $derived(data.servicios.find((s) => s.id === data.servicio));
	const nombreDe = (id: string) => data.todos.find((p) => p.id === id)?.nombre ?? '';
	const profNombre = $derived(nombreDe(data.profesional));

	const enlace = (cambios: Record<string, string>) => {
		const v = { servicio: data.servicio, profesional: data.profesional, fecha: data.fecha, cliente: data.cliente.id, ...cambios };
		const q = new URLSearchParams();
		for (const [k, x] of Object.entries(v)) if (x) q.set(k, x);
		return `?${q}`;
	};
	const dias = $derived(Array.from({ length: 7 }, (_, i) => sumarDias(dia(), i)));
	const dow = (d: string) => new Intl.DateTimeFormat('es-AR', { weekday: 'short', timeZone: 'UTC' }).format(new Date(`${d}T12:00:00Z`)).replace('.', '');
	const preelegido = $derived(Math.max(0, data.horarios.findIndex((h) => h.hora === data.hora)));
	let elegido = $state('');
	$effect(() => {
		elegido = data.horarios[preelegido]?.inicio ?? '';
	});
	const horaElegida = $derived(data.horarios.find((h) => h.inicio === elegido)?.hora ?? '');

	// El cliente: búsqueda rápida o uno nuevo.
	let nombre = $state('');
	let telefono = $state('');
	let email = $state('');
	let clienteId = $state('');
	$effect(() => {
		nombre = data.cliente.nombre;
		telefono = data.cliente.telefono;
		email = data.cliente.email;
		clienteId = data.cliente.id;
	});
	let encontrados = $state<Encontrado['clientes']>([]);
	let reloj: ReturnType<typeof setTimeout> | undefined;
	function buscar() {
		clienteId = '';
		clearTimeout(reloj);
		const q = nombre.trim();
		if (q.length < 2) {
			encontrados = [];
			return;
		}
		reloj = setTimeout(async () => {
			try {
				const r = await fetch(`/panel/buscar/rapido?${new URLSearchParams({ q })}`);
				if (r.ok) encontrados = ((await r.json()) as Encontrado).clientes.slice(0, 5);
			} catch {
				encontrados = [];
			}
		}, 250);
	}
	function elegir(c: Encontrado['clientes'][number]) {
		nombre = c.nombre;
		telefono = c.telefono;
		email = c.email;
		clienteId = c.id;
		encontrados = [];
	}

	// Turno fijo: la vista previa vuelve en `form.previa`.
	const previa = $derived(form && 'previa' in form ? form.previa : null);
	let usar = $state<Record<string, string>>({});
	$effect(() => {
		const u: Record<string, string> = {};
		for (const x of previa?.fechas ?? []) if (x.estado === 'libre') u[x.inicio] = x.inicio;
		usar = u;
	});
	const aceptadas = $derived(Object.values(usar).filter(Boolean));
	const corto = (iso: string) =>
		new Intl.DateTimeFormat('es-AR', { weekday: 'short', day: '2-digit', month: 'short', timeZone: 'America/Argentina/Buenos_Aires' })
			.format(new Date(iso))
			.replace(/\./g, '')
			.replace(' de ', ' ')
			.toUpperCase();
	const alternar = (x: FechaSerie, alt = false) => {
		const v = alt && x.alternativa ? x.alternativa.inicio : x.inicio;
		usar = { ...usar, [x.inicio]: usar[x.inicio] === v ? '' : v };
	};
	const resumen = $derived(
		[servicio?.nombre, profNombre ? `con ${primerNombre(profNombre)}` : '', `${nombreDia(data.fecha)}${horaElegida && !sobreturno ? ` ${horaElegida}` : ''}`].filter(Boolean).join(' · ')
	);
</script>

<svelte:head><title>Nuevo turno · Panel</title></svelte:head>

<p class="volver"><a href="/panel/turnos"><Icono nombre="arrow_back" tam={18} />Turnos</a></p>

<article class="hoja" aria-labelledby="t-nuevo">
	<header class="cabeza">
		<span class="tit">
			<h1 id="t-nuevo">Nuevo turno</h1>
			{#if data.hora && data.profesional}<span class="desde tl-num">Hueco de {primerNombre(profNombre)} · {data.hora}</span>{/if}
		</span>
		<a class="cerrar" href="/panel/turnos" aria-label="Cerrar"><Icono nombre="close" tam={22} /></a>
	</header>

	{#if data.servicios.length === 0}
		<div class="cuerpo"><p class="vacio">Primero cargá un servicio en <a href="/panel/servicios">Servicios</a>.</p></div>
	{:else}
		<div class="cuerpo">
			<section class="sec">
				<span class="tl-rotulo rot">Servicio</span>
				<div class="chips">
					{#each data.servicios as s (s.id)}
						<a class="chip" class:on={s.id === data.servicio} href={enlace({ servicio: s.id, profesional: '' })} data-sveltekit-noscroll data-sveltekit-replacestate>{s.nombre}</a>
					{/each}
				</div>
				{#if servicio}<span class="nota tl-num">{servicio.duracion} min · {pesos(servicio.precio)}</span>{/if}
			</section>

			<section class="sec">
				<span class="tl-rotulo rot">Con quién y cuándo</span>
				<div class="chips">
					<a class="chip" class:on={!data.profesional} href={enlace({ profesional: '' })} data-sveltekit-noscroll data-sveltekit-replacestate>Cualquiera</a>
					{#each data.profesionales as p (p.id)}
						<a class="chip" class:on={p.id === data.profesional} href={enlace({ profesional: p.id })} data-sveltekit-noscroll data-sveltekit-replacestate>{p.nombre}</a>
					{/each}
				</div>
				<div class="dias">
					{#each dias as d (d)}
						<a class="dia" class:on={d === data.fecha} href={enlace({ fecha: d })} data-sveltekit-noscroll data-sveltekit-replacestate>
							<span>{d === dia() ? 'hoy' : dow(d)}</span><b class="tl-num">{d.slice(8)}</b>
						</a>
					{/each}
					<form method="GET" class="otro-dia" data-sveltekit-noscroll>
						{#each Object.entries({ servicio: data.servicio, profesional: data.profesional, cliente: data.cliente.id }) as [k, v] (k)}
							{#if v}<input type="hidden" name={k} value={v} />{/if}
						{/each}
						<label class="campo"><span class="visualmente-oculto">Otro día</span><input type="date" name="fecha" value={data.fecha} onchange={(e) => e.currentTarget.form?.requestSubmit()} /></label>
					</form>
				</div>
			</section>

			{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}

			<form
				method="POST"
				action="?/crear"
				id="form-nuevo"
				class="pila-f"
				use:enhance={() => {
					enviando = true;
					return async ({ update }) => {
						await update({ reset: false });
						enviando = false;
					};
				}}
			>
				<input type="hidden" name="servicioId" value={data.servicio} />
				<input type="hidden" name="fecha" value={data.fecha} />
				{#if !sobreturno}<input type="hidden" name="profesionalId" value={data.profesional} />{/if}

				<section class="sec">
					<span class="tl-rotulo rot">Horario del {nombreDia(data.fecha)}</span>
					{#if !sobreturno}
						{#if data.errorLibres}
							<p class="aviso error">{data.errorLibres}</p>
						{:else if data.horarios.length === 0}
							<p class="nota">No quedan horarios libres{servicio ? ` de ${servicio.nombre}` : ''} ese día. Probá otro día u otro profesional, o cargalo como&nbsp;sobreturno.</p>
						{:else}
							<fieldset class="horas">
								<legend class="visualmente-oculto">Horarios libres</legend>
								{#each data.horarios as h (h.inicio)}
									<label class="h">
										<input type="radio" name="inicio" value={h.inicio} required bind:group={elegido} />
										<span class="tl-num">{h.hora}</span>
										{#if !data.profesional}<small>{h.profesionales.map((x) => primerNombre(nombreDe(x))).filter(Boolean).join(', ')}</small>{/if}
									</label>
								{/each}
							</fieldset>
						{/if}
					{/if}
					<label class="check"><input type="checkbox" name="sobreturno" bind:checked={sobreturno} />Sobreturno: a una hora que no figura&nbsp;libre</label>
					{#if sobreturno}
						<div class="sobre">
							<label class="campo">
								Profesional
								<select name="profesionalId" required>
									<option value="" disabled selected={!data.profesional}>Elegí</option>
									{#each data.profesionales as p (p.id)}<option value={p.id} selected={p.id === data.profesional}>{p.nombre}</option>{/each}
								</select>
							</label>
							<label class="campo">Hora <input type="time" name="hora" required step="300" value={data.hora} /></label>
							<span class="ambar"><Icono nombre="warning" tam={18} />Se va a superponer con otro turno. Avisale a quien&nbsp;atiende.</span>
						</div>
					{/if}
				</section>

				<section class="sec">
					<span class="tl-rotulo rot">Cliente</span>
					<input type="hidden" name="clienteId" value={clienteId} />
					<div class="cliente">
						<label class="campo crece">
							Nombre
							<input name="nombre" required maxlength="80" autocomplete="off" bind:value={nombre} oninput={buscar} />
						</label>
						{#if encontrados.length}
							<ul class="encontrados" aria-label="Clientes que ya están">
								{#each encontrados as c (c.id)}
									<li>
										<button type="button" onclick={() => elegir(c)}>
											<span class="av">{iniciales(c.nombre)}</span>
											<span class="txt"><b>{c.nombre}</b><span class="tl-num">{[c.telefono, c.email].filter(Boolean).join(' · ')}</span></span>
										</button>
									</li>
								{/each}
							</ul>
						{/if}
						<div class="dos">
							<label class="campo">Teléfono <input name="telefono" type="tel" maxlength="30" autocomplete="off" bind:value={telefono} /></label>
							<label class="campo">Email <input name="email" type="email" maxlength="120" autocomplete="off" bind:value={email} /></label>
						</div>
						<p class="nota">Con un teléfono o un email le llegan los avisos. Si el teléfono ya está cargado, el turno va a ese&nbsp;cliente.</p>
					</div>
					<label class="campo">Nota interna <textarea name="nota" maxlength="500" rows="2" placeholder="Sólo la ve el equipo"></textarea></label>
				</section>

				<section class="sec">
					<span class="tl-rotulo rot">Se repite</span>
					<div class="chips" role="radiogroup" aria-label="Se repite">
						{#each [[0, 'Una vez'], [1, 'Cada semana'], [2, 'Cada 2'], [3, 'Cada 3'], [4, 'Cada 4']] as [v, t] (v)}
							<label class="chip" class:on={cada === v}><input type="radio" name="cadaSemanas" value={v} bind:group={cada} />{t}</label>
						{/each}
					</div>
					{#if cada > 0}
						{#if !data.profesional && !sobreturno}<p class="ambar"><Icono nombre="info" tam={18} />Para un turno fijo elegí quién lo atiende.</p>{/if}
						<div class="hasta">
							<span>Hasta</span>
							<div class="seg">
								<label class:on={hastaModo === 'cantidad'}><input type="radio" bind:group={hastaModo} value="cantidad" />Cantidad</label>
								<label class:on={hastaModo === 'fecha'}><input type="radio" bind:group={hastaModo} value="fecha" />Una fecha</label>
							</div>
							{#if hastaModo === 'cantidad'}
								<label class="campo chico-c"><span class="visualmente-oculto">Cuántos turnos</span><input type="number" name="cantidad" min="2" max="52" value="4" />turnos</label>
							{:else}
								<label class="campo chico-c"><span class="visualmente-oculto">Hasta qué día</span><input type="date" name="hasta" min={data.fecha} /></label>
							{/if}
						</div>
						{#if previa}
							<ul class="fechas" aria-label="Las fechas del turno fijo">
								{#each previa.fechas as x (x.inicio)}
									{@const usada = usar[x.inicio] ?? ''}
									<li class:choca={x.estado === 'choca'} class:fuera={!usada}>
										<span class="f tl-num">{corto(x.inicio)}</span>
										{#if x.estado === 'libre'}
											<span class="e verde">Libre · {x.hora}</span>
											<button type="button" class="alt" onclick={() => alternar(x)}>{usada ? 'Sacar' : 'Sumar'}</button>
										{:else}
											<span class="e">{MOTIVO_CHOCA[x.motivo ?? ''] ?? 'No se puede'}</span>
											{#if x.alternativa}
												<button type="button" class="alt" onclick={() => alternar(x, true)}>{usada ? `Pasada a ${x.alternativa.hora} · Sacar` : `Pasar a ${x.alternativa.hora}`}</button>
											{:else}
												<span class="sin">Sin lugar ese día</span>
											{/if}
										{/if}
										{#if usada}<input type="hidden" name="turnos" value={usada} />{/if}
									</li>
								{/each}
							</ul>
						{/if}
					{/if}
				</section>
			</form>
		</div>

		<footer class="pie">
			<span class="res">
				<b class="tl-num">{cada > 0 && previa ? `${aceptadas.length} turnos · ${resumen}` : resumen}</b>
				Le llega la confirmación por&nbsp;mail.
			</span>
			{#if cada > 0 && !previa}
				<button class="reservar" form="form-nuevo" name="paso" value="previa" disabled={enviando}>Ver las fechas</button>
			{:else if cada > 0}
				<button class="reservar" form="form-nuevo" name="paso" value="reservar" disabled={enviando || aceptadas.length === 0}>Reservar los {aceptadas.length}</button>
			{:else}
				<button class="reservar" form="form-nuevo" disabled={enviando}>Reservar turno</button>
			{/if}
		</footer>
	{/if}
</article>

<style>
	.volver {
		margin: 0;
	}
	.volver a {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		min-height: 40px;
		font-size: 14px;
		font-weight: 700;
		color: var(--tl-menta-texto);
		text-decoration: none;
	}
	.hoja {
		width: 100%;
		max-width: 560px;
		border-radius: 24px;
		background: var(--tl-papel);
		box-shadow: 0 30px 60px -36px rgba(15, 35, 40, 0.55);
		display: flex;
		flex-direction: column;
		animation: tl-gin var(--tl-entrar) var(--tl-ease) both;
	}
	.cabeza {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 16px 20px;
		border-bottom: 1px solid #f0e8d8;
	}
	.tit {
		flex: 1;
		display: flex;
		flex-direction: column;
	}
	h1 {
		margin: 0;
		font-size: 22px;
		font-weight: 700;
		letter-spacing: -0.03em;
	}
	.desde {
		font-size: 12px;
		font-weight: 700;
		color: var(--tl-menta-texto);
	}
	.cerrar {
		width: 44px;
		height: 44px;
		border-radius: 14px;
		background: var(--tl-crema);
		display: grid;
		place-items: center;
		color: var(--tl-tinta);
	}
	.cuerpo {
		padding: 16px 20px 24px;
		display: flex;
		flex-direction: column;
		gap: 22px;
	}
	.pila-f {
		display: flex;
		flex-direction: column;
		gap: 22px;
	}
	.sec {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.rot {
		font-size: 11px;
		color: var(--tl-sub);
	}
	.chips {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
	}
	.chip {
		position: relative;
		display: inline-flex;
		align-items: center;
		min-height: 40px;
		padding: 0 14px;
		border-radius: 999px;
		box-shadow: inset 0 0 0 1.5px #d8ccb4;
		background: #fff;
		color: var(--tl-tinta);
		font-size: 14px;
		font-weight: 700;
		text-decoration: none;
		cursor: pointer;
	}
	.chip.on {
		background: var(--tl-petroleo);
		color: #fff;
		box-shadow: none;
	}
	.chip input {
		position: absolute;
		opacity: 0;
		pointer-events: none;
	}
	.chip:has(input:focus-visible) {
		outline: 3px solid var(--tl-menta);
		outline-offset: 2px;
	}
	.nota {
		margin: 0;
		font-size: 13px;
		color: var(--tl-sub);
		text-wrap: pretty;
	}
	.dias {
		display: flex;
		gap: 6px;
		flex-wrap: wrap;
		align-items: center;
	}
	.dia {
		width: 56px;
		height: 56px;
		border-radius: 14px;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		background: var(--tl-crema);
		color: var(--tl-tinta);
		text-decoration: none;
	}
	.dia span {
		font-size: 11px;
		text-transform: uppercase;
	}
	.dia b {
		font-size: 17px;
		font-weight: 700;
	}
	.dia.on {
		background: var(--tl-petroleo);
		color: #fff;
	}
	.otro-dia input {
		min-height: 48px;
	}
	.horas {
		border: 0;
		padding: 0;
		margin: 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(76px, 1fr));
		gap: 6px;
	}
	.h {
		position: relative;
		cursor: pointer;
	}
	.h input {
		position: absolute;
		opacity: 0;
		pointer-events: none;
	}
	.h span {
		display: grid;
		place-items: center;
		height: 44px;
		border-radius: 12px;
		background: repeating-linear-gradient(-45deg, rgba(60, 204, 144, 0.18) 0 2px, transparent 2px 9px), #f3fbf7;
		font-size: 15px;
		font-weight: 700;
		color: var(--tl-menta-texto);
	}
	.h small {
		display: block;
		margin-top: 2px;
		font-size: 11px;
		text-align: center;
		color: var(--tl-sub);
	}
	.h:has(input:checked) span {
		background: var(--tl-petroleo);
		color: #fff;
	}
	.h:has(input:focus-visible) span {
		outline: 3px solid var(--tl-menta);
		outline-offset: 2px;
	}
	.check {
		display: flex;
		align-items: center;
		gap: 10px;
		min-height: 44px;
		font-size: 14px;
		cursor: pointer;
	}
	.check input {
		width: 20px;
		height: 20px;
		accent-color: var(--tl-petroleo);
	}
	.sobre {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 10px;
	}
	.ambar {
		grid-column: 1 / -1;
		margin: 0;
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 12px;
		border-radius: 12px;
		background: #fbe9c2;
		color: #5c3800;
		font-size: 13px;
	}
	.cliente {
		position: relative;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.dos {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
		gap: 10px;
	}
	.encontrados {
		list-style: none;
		margin: 0;
		padding: 4px;
		border-radius: 14px;
		background: var(--tl-crema);
	}
	.encontrados button {
		display: flex;
		align-items: center;
		gap: 10px;
		width: 100%;
		min-height: 48px;
		padding: 4px 8px;
		border: 0;
		border-radius: 10px;
		background: transparent;
		text-align: left;
		font-family: var(--tl-font);
		cursor: pointer;
	}
	.encontrados button:hover {
		background: var(--tl-papel);
	}
	.av {
		width: 34px;
		height: 34px;
		border-radius: 50%;
		background: var(--tl-papel);
		display: grid;
		place-items: center;
		font-size: 12px;
		font-weight: 700;
	}
	.txt {
		display: flex;
		flex-direction: column;
	}
	.txt b {
		font-size: 15px;
	}
	.txt span {
		font-size: 12px;
		color: var(--tl-sub);
	}
	.hasta {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 8px 12px;
		font-size: 14px;
	}
	.seg {
		display: flex;
		padding: 3px;
		border-radius: 999px;
		background: var(--tl-crema);
	}
	.seg label {
		position: relative;
		min-height: 36px;
		padding: 0 12px;
		display: inline-flex;
		align-items: center;
		border-radius: 999px;
		font-size: 13px;
		font-weight: 700;
		cursor: pointer;
	}
	.seg label.on {
		background: var(--tl-petroleo);
		color: #fff;
	}
	.seg input {
		position: absolute;
		opacity: 0;
		pointer-events: none;
	}
	.chico-c {
		flex-direction: row;
		align-items: center;
		gap: 8px;
	}
	.chico-c input[type='number'] {
		width: 80px;
	}
	.fechas {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	.fechas li {
		display: grid;
		grid-template-columns: 112px minmax(0, 1fr) auto;
		align-items: center;
		min-height: 46px;
		border-radius: 12px;
		font-size: 13px;
	}
	.fechas li.choca {
		background: #fbe9c2;
	}
	.fechas li.fuera:not(.choca) {
		opacity: 0.55;
	}
	.f {
		align-self: stretch;
		display: flex;
		align-items: center;
		padding: 0 12px;
		border-right: 2px dashed var(--tl-linea-2);
		font-weight: 700;
	}
	.e {
		padding: 0 12px;
		color: #5c3800;
	}
	.e.verde {
		color: var(--tl-menta-texto);
	}
	.alt,
	.sin {
		margin-right: 10px;
		min-height: 36px;
		border: 0;
		background: transparent;
		font: 700 12px var(--tl-font);
		color: var(--tl-menta-texto);
		cursor: pointer;
	}
	.sin {
		display: inline-flex;
		align-items: center;
		color: #5c3800;
	}
	.pie {
		position: sticky;
		bottom: 0;
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 14px 20px 18px;
		border-top: 1px solid #f0e8d8;
		background: var(--tl-papel);
		border-radius: 0 0 24px 24px;
	}
	.res {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		font-size: 12px;
		color: var(--tl-sub);
		line-height: 1.35;
	}
	.res b {
		font-size: 15px;
		font-weight: 700;
		color: var(--tl-tinta);
	}
	.reservar {
		flex: none;
		height: 52px;
		padding: 0 22px;
		border: 0;
		border-radius: 999px;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		font: 700 16px var(--tl-font);
		cursor: pointer;
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.45), 0 12px 24px -10px rgba(60, 204, 144, 0.8);
	}
	.reservar:disabled {
		opacity: 0.55;
		cursor: not-allowed;
	}
	.vacio {
		margin: 0;
	}
	@media (max-width: 859px) {
		.hoja {
			border-radius: 20px;
		}
		.pie {
			bottom: 80px;
			border-radius: 0;
		}
		.sobre {
			grid-template-columns: 1fr;
		}
		.fechas li {
			grid-template-columns: 96px minmax(0, 1fr);
		}
		.alt,
		.sin {
			grid-column: 2;
			justify-self: start;
			margin-left: 12px;
		}
	}
</style>
