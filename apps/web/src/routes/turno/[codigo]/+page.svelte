<script lang="ts">
	import { enhance } from '$app/forms';
	import { page } from '$app/state';
	import { DIAS, ESTADOS, dia, fecha, fechaHora, hora, pesos } from '#lib/formato.ts';
	import Calendario from '#lib/publico/Calendario.svelte';
	import Horarios from '#lib/publico/Horarios.svelte';
	import { fechaLarga } from '#lib/publico/calendario.ts';
	import { linkWhatsapp } from '#lib/publico/textos.ts';

	let { data, form } = $props();
	const t = $derived(data.turno);
	const n = $derived(t.negocio);
	const z = $derived(n.zonaHoraria);
	const base = $derived(`/turno/${t.codigo}`);
	const activo = $derived(['pendiente', 'confirmado', 'pendiente_pago'].includes(t.estado));
	const cambiado = $derived(page.url.searchParams.has('cambiado'));
	const linkReferido = $derived(
		data.referido?.codigo ? `${page.url.origin}/${n.slug}/reservar?ref=${encodeURIComponent(data.referido.codigo)}` : ''
	);
	const mensajesPago: Record<string, string> = {
		ok: '¡Recibimos tu pago! En unos minutos queda confirmado el turno.',
		pendiente: 'Tu pago está en proceso. Te avisamos cuando se acredite.',
		error: 'El pago no se completó. Podés intentar de nuevo.'
	};
	let copiado = $state(false);
	async function copiar() {
		try {
			await navigator.clipboard.writeText(linkReferido);
			copiado = true;
		} catch {
			copiado = false;
		}
	}
</script>

<svelte:head>
	<title>Tu turno en {n.nombre} · Tengo Libre</title>
	<meta name="robots" content="noindex, nofollow" />
	<meta name="referrer" content="no-referrer" />
</svelte:head>

<main class="contenedor pila turno">
	<header class="pila" style="gap: 4px">
		<a class="suave chico" href={`/${n.slug}`}>{n.nombre}</a>
		<h1>{t.estado === 'cancelado' ? 'Turno cancelado' : 'Tu turno'}</h1>
	</header>

	{#if data.nuevo && t.estado !== 'cancelado'}
		<p class="aviso" role="status">
			{t.estado === 'pendiente_pago' ? 'Reservamos tu lugar. Falta el pago para confirmarlo.' : '¡Listo! Reservaste tu turno. Guardá este link para verlo o cambiarlo.'}
		</p>
	{/if}
	{#if cambiado}<p class="aviso" role="status">Cambiamos tu turno al nuevo&nbsp;horario.</p>{/if}
	{#if data.pago && mensajesPago[data.pago]}
		<p class={data.pago === 'error' ? 'aviso error' : 'aviso'} role="status">{mensajesPago[data.pago]}</p>
	{/if}
	{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}
	{#if form && 'cancelado' in form}<p class="aviso" role="status">Cancelaste el turno. Avisamos al&nbsp;negocio.</p>{/if}

	<section class="tarjeta pila" aria-label="Detalle del turno">
		<div class="fila" style="justify-content: space-between">
			<strong class="cuando">{fechaLarga(dia(t.inicio, z))}</strong>
			<span class="etiqueta" class:cancelado={t.estado === 'cancelado'}>{ESTADOS[t.estado] ?? t.estado}</span>
		</div>
		<p class="hora">{hora(t.inicio, z)} a {hora(t.fin, z)}&nbsp;h</p>
		<dl class="datos">
			<div><dt>Servicio</dt><dd>{t.servicio}</dd></div>
			{#if t.profesional}<div><dt>Con</dt><dd>{t.profesional}</dd></div>{/if}
			<div><dt>Precio</dt><dd>{pesos(t.precio)}</dd></div>
			<div><dt>A nombre de</dt><dd>{t.cliente.nombre}</dd></div>
			{#if n.direccion}<div><dt>Dónde</dt><dd>{n.direccion}{n.ciudad ? `, ${n.ciudad}` : ''}</dd></div>{/if}
		</dl>
		{#if t.estado === 'pendiente_pago'}
			<form method="POST" action="?/pagar" use:enhance class="pila">
				{#if t.venceEn}<p class="chico" style="margin: 0">Pagá antes de las {fechaHora(t.venceEn, z)} para no perder el&nbsp;lugar.</p>{/if}
				<button class="boton primario">Pagar con Mercado Pago</button>
			</form>
		{/if}
	</section>

	{#if activo}
		<section class="pila" aria-labelledby="t-cambios">
			<h2 id="t-cambios">¿Necesitás cambiarlo?</h2>
			{#if data.puedeCambiar}
				{#if data.cambiar}
					<div class="tarjeta pila">
						<h3 style="margin: 0">Elegí el nuevo día</h3>
						<Calendario dias={data.dias} elegida={data.fecha} href={(f) => `${base}?cambiar=1&fecha=${f}`} />
						{#if data.fecha}
							<h3 style="margin: 0">Horarios del {fechaLarga(data.fecha)}</h3>
							{#if data.horarios.length}
								<form method="POST" action="?/reprogramar" use:enhance class="pila">
									<fieldset class="horas">
										<legend class="visualmente-oculto">Nuevo horario</legend>
										{#each data.horarios as h (h.inicio)}
											<label class="boton hora-op"><input type="radio" name="inicio" value={h.inicio} required />{h.hora}</label>
										{/each}
									</fieldset>
									<button class="boton primario">Cambiar a este horario</button>
								</form>
							{:else}
								<p class="suave">Ese día no quedan horarios. Probá con&nbsp;otro.</p>
							{/if}
						{/if}
						<a class="boton" href={base}>No cambiar</a>
					</div>
				{:else}
					<div class="fila">
						<a class="boton" href={`${base}?cambiar=1`}>Cambiar día u horario</a>
					</div>
				{/if}
				<details class="tarjeta">
					<summary>Cancelar el turno</summary>
					<form method="POST" action="?/cancelar" use:enhance class="pila" style="margin-top: 12px">
						<p style="margin: 0">¿Seguro? El horario queda libre para otra&nbsp;persona.</p>
						<button class="boton peligro">Sí, cancelar el turno</button>
					</form>
				</details>
				<p class="suave chico">Podés cambiarlo o cancelarlo hasta {t.cancelarHastaHoras} {t.cancelarHastaHoras === 1 ? 'hora' : 'horas'}&nbsp;antes.</p>
			{:else}
				<p class="suave">
					Ya pasó el plazo para cambiarlo online ({t.cancelarHastaHoras}&nbsp;h antes). Escribile al&nbsp;negocio.
				</p>
				{#if n.whatsapp}<a class="boton" href={linkWhatsapp(n.whatsapp)} target="_blank" rel="noopener">Escribir por WhatsApp</a>{/if}
			{/if}
		</section>
	{:else}
		<p><a class="boton primario" href={`/${n.slug}/reservar`}>Sacar otro turno</a></p>
	{/if}

	{#if data.sugerencias.length && t.servicioId}
		<section class="pila" aria-labelledby="t-sug">
			<h2 id="t-sug">Tu próximo turno</h2>
			<ul class="pila lista">
				{#each data.sugerencias.slice(0, 3) as s, i (i)}
					<li class="tarjeta fila" style="justify-content: space-between">
						<span>
							<strong>{DIAS[s.dia]} {s.hora}&nbsp;h</strong>
							{#if s.profesional}con {s.profesional}{/if}
							<span class="suave chico" style="display: block">{s.motivo}</span>
						</span>
						<a
							class="boton"
							href={`/${n.slug}/reservar?servicio=${t.servicioId}&profesional=${s.profesionalId || 'cualquiera'}&fecha=${s.proxima}`}
							>Reservar el {fecha(s.proxima + 'T12:00:00Z', 'UTC')}</a
						>
					</li>
				{/each}
			</ul>
		</section>
	{/if}

	{#if data.referido?.programaActivo}
		<section class="tarjeta pila" aria-labelledby="t-ref">
			<h2 id="t-ref">Recomendá {n.nombre}</h2>
			{#if data.referido.codigo}
				<p style="margin: 0">Tu código es <strong class="codigo">{data.referido.codigo}</strong>. Cuando alguien reserva con él, te ganás un&nbsp;descuento.</p>
				<label class="campo">
					Link para compartir
					<input readonly value={linkReferido} onfocus={(e) => e.currentTarget.select()} />
				</label>
				<div class="fila">
					<button type="button" class="boton" onclick={copiar}>{copiado ? 'Copiado' : 'Copiar link'}</button>
				</div>
				{#if data.referido.referidos > 0}
					<p class="suave chico" style="margin: 0">Ya reservaron {data.referido.referidos} con tu&nbsp;código.</p>
				{/if}
				{#each data.referido.recompensas.filter((c) => c.activo) as c (c.id)}
					<p class="aviso" style="margin: 0">
						Tu premio: cupón <strong>{c.codigo}</strong> ({c.tipo === 'porcentaje' ? `${c.valor}%` : pesos(c.valor)} de&nbsp;descuento).
					</p>
				{/each}
			{:else}
				<p style="margin: 0">Pedí tu código y compartilo: cuando alguien reserva con él, te ganás un&nbsp;descuento.</p>
				<form method="POST" action="?/referido" use:enhance>
					<button class="boton">Quiero mi código</button>
				</form>
			{/if}
		</section>
	{/if}

	<footer class="suave chico">
		<a href={`/${n.slug}`}>Ver {n.nombre}</a>
		{#if n.telefono} · <a href={`tel:${n.telefono.replace(/[^\d+]/g, '')}`}>{n.telefono}</a>{/if}
	</footer>
</main>

<style>
	.turno {
		max-width: 640px;
	}
	.cuando {
		font-size: 17px;
	}
	.hora {
		margin: 0;
		font-size: 22px;
		font-weight: 600;
	}
	.datos {
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	.datos div {
		display: grid;
		grid-template-columns: 100px 1fr;
		gap: 8px;
	}
	.datos dt {
		color: var(--suave);
		font-size: 13px;
	}
	.datos dd {
		margin: 0;
	}
	.etiqueta.cancelado {
		background: #fdecea;
		color: var(--peligro);
	}
	.horas {
		border: 0;
		margin: 0;
		padding: 0;
		display: grid;
		gap: 8px;
		grid-template-columns: repeat(auto-fill, minmax(76px, 1fr));
	}
	.hora-op {
		min-height: 44px;
		gap: 4px;
	}
	.hora-op:has(input:checked) {
		border-color: var(--marca);
		background: var(--marca-suave);
	}
	.hora-op input {
		accent-color: var(--marca);
		margin: 0;
	}
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.codigo {
		font-family: ui-monospace, monospace;
		letter-spacing: 0.05em;
	}
	summary {
		cursor: pointer;
		min-height: 28px;
	}
</style>
