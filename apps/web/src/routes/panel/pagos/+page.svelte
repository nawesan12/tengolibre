<script lang="ts">
	import '#lib/gestion/gestion.css';
	import { enhance } from '$app/forms';
	import { page } from '$app/state';
	import { fechaHora, hora, pesos } from '#lib/formato.ts';
	import { resta } from '#lib/configuracion/dinero.ts';
	import { ESTADOS_PAGO } from '#lib/configuracion/textos.ts';
	import { confirmar, sinReset } from '#lib/configuracion/formularios.ts';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';
	import { alReservar, cuandoCorto, nombreMes } from '#lib/gestion/cobros/formato.ts';
	import type { Pago } from './+page.server.ts';

	let { data, form } = $props();
	const c = $derived(data.conexion);
	const esDuenio = $derived(data.negocio.rol === 'duenio');
	// svelte-ignore state_referenced_locally
	let modo = $state(data.config.modo);
	// svelte-ignore state_referenced_locally
	let porcentaje = $state(data.config.senaPorcentaje || 30);
	// svelte-ignore state_referenced_locally
	let fija = $state(data.config.senaFija || 0);
	let devolviendo = $state<string | null>(null);
	let cuanto = $state<'todo' | 'parte'>('todo');
	const recienConectada = $derived(page.url.searchParams.get('mercadopago') === 'conectada');
	const ej = $derived(data.ejemplo);

	const MODOS = [
		{ clave: 'sin_cobro', titulo: 'Sin cobro', oculto: 'por adelantado', sub: 'Reservan sin pagar nada' },
		{ clave: 'porcentaje', titulo: 'Seña', simbolo: '%', oculto: 'en porcentaje', sub: 'Un porcentaje del precio' },
		{ clave: 'fija', titulo: 'Seña fija', oculto: 'de monto fijo', sub: 'El mismo monto siempre' },
		{ clave: 'total', titulo: 'Pago total', oculto: 'al reservar', sub: 'Pagan todo al reservar' }
	];

	function ejemplo(clave: string) {
		if (clave === 'sin_cobro') return '—';
		if (clave === 'porcentaje') return ej ? `${porcentaje}% · ${pesos(alReservar('porcentaje', ej.precio, porcentaje, 0))}` : `${porcentaje}%`;
		if (clave === 'fija') return fija > 0 ? pesos(fija) : 'Elegí el monto';
		return ej ? pesos(ej.precio) : 'El precio entero';
	}
	const cobra = $derived(ej ? alReservar(modo, ej.precio, porcentaje, fija) : 0);

	// El total cobrado: lo aprobado menos lo devuelto.
	const cobrado = $derived(data.pagos.filter((p) => p.estado === 'aprobado').reduce((t, p) => t + resta(p), 0));
	const mes = $derived(data.pagos.length ? nombreMes(data.pagos[0].creadoEn) : '');

	type Est = { texto: string; icono: string; tono: 'ok' | 'espera' | 'error' | 'neutro' };
	function estadoDe(p: Pago): Est {
		if (p.vencido) return { texto: 'Venció sin pagar', icono: 'timer_off', tono: 'error' };
		switch (p.estado) {
			case 'aprobado':
				return p.devuelto > 0
					? { texto: `Devueltos ${pesos(p.devuelto)}`, icono: 'currency_exchange', tono: 'ok' }
					: { texto: 'Aprobado', icono: 'check_circle', tono: 'ok' };
			case 'pendiente':
			case 'en_proceso':
				return { texto: p.venceEn ? `Esperando · vence ${hora(p.venceEn)}` : 'Esperando', icono: 'hourglass_top', tono: 'espera' };
			case 'devuelto':
				return { texto: 'Devuelto', icono: 'currency_exchange', tono: 'neutro' };
			case 'rechazado':
				return { texto: 'Rechazado', icono: 'block', tono: 'error' };
			default:
				return { texto: ESTADOS_PAGO[p.estado] ?? p.estado, icono: 'cancel', tono: 'neutro' };
		}
	}
	const apagado = (p: Pago) => p.vencido || p.estado === 'devuelto' || p.estado === 'cancelado' || p.estado === 'rechazado';
</script>

<svelte:head><title>Pagos · Panel</title></svelte:head>

<div class="g-pagina">
	{#if recienConectada}
		<section class="conectado" aria-labelledby="t-listo">
			<p class="banda" role="status"><span class="icono lleno" aria-hidden="true">check_circle</span>Mercado Pago conectado</p>
			<div class="conectado-cuerpo">
				{#if c.cuenta}<span class="g-kicker">{c.cuenta}</span>{/if}
				<h2 id="t-listo" class="g-titulo-2">Listo, ya podés cobrar&nbsp;señas</h2>
				<p class="g-bajada">La plata entra a tu cuenta de Mercado Pago. Elegí abajo cuánto cobrar al&nbsp;reservar.</p>
			</div>
		</section>
	{/if}

	<div class="g-cabeza">
		<h1 class="g-titulo">Pagos</h1>
		{#if c.disponible && c.conectada}
			<span class="mp-chip">
				<img src="/logos/mercadopago.svg" alt="" width="22" height="22" />
				<b>Mercado Pago conectado</b>
				<span class="g-meta">{c.cuenta ?? (c.usuario ? `Cuenta ${c.usuario}` : '')}</span>
				<span class="punto" aria-hidden="true"></span>
			</span>
		{:else if c.disponible && esDuenio}
			<!-- Sin enhance: la respuesta es una redirección a Mercado Pago. -->
			<form method="POST" action="?/conectar">
				<button class="g-boton menta"><img src="/logos/mercadopago.svg" alt="" width="22" height="22" />Conectar Mercado Pago</button>
			</form>
		{/if}
	</div>

	<Mensaje {form} que="mercadopago" />
	{#if !c.disponible}
		<p class="g-aviso ambar">
			<span class="icono" aria-hidden="true">info</span>
			<span>Los cobros con Mercado Pago todavía no están habilitados en TengoLibre. Mientras tanto, los turnos se reservan sin cobro por&nbsp;adelantado.</span>
		</p>
	{:else if c.conectada}
		{#if c.reautorizar}
			<div class="g-aviso rojo" role="alert">
				<span class="icono" aria-hidden="true">error</span>
				<span>Mercado Pago pide que vuelvas a conectar la cuenta para seguir&nbsp;cobrando.</span>
				{#if esDuenio}
					<form method="POST" action="?/conectar"><button class="g-boton">Volver a conectar</button></form>
				{/if}
			</div>
		{/if}
		{#if c.ultimoError}<p class="g-nota">Último problema con Mercado Pago: {c.ultimoError}</p>{/if}
	{:else}
		<p class="g-aviso socios">
			<span class="icono" aria-hidden="true">account_balance_wallet</span>
			<span>
				{esDuenio
					? 'Conectá tu cuenta para cobrar señas o el total al reservar. La plata entra directo a tu cuenta.'
					: 'La cuenta de Mercado Pago la conecta la dueña o el dueño del negocio.'}
			</span>
		</p>
	{/if}

	<section aria-labelledby="t-cobro" class="g-col">
		<h2 id="t-cobro" class="visualmente-oculto">Cómo cobrás al reservar</h2>
		<Mensaje {form} que="cobro" />
		{#if modo !== 'sin_cobro' && !c.conectada}
			<p class="g-aviso ambar">
				<span class="icono" aria-hidden="true">schedule</span>
				<span>Hasta que conectes Mercado Pago, los turnos se reservan sin&nbsp;cobro.</span>
			</p>
		{/if}
		<form method="POST" action="?/cobro" use:enhance={sinReset} class="g-col cobro">
			<fieldset class="modos">
				<legend class="visualmente-oculto">Modo de cobro</legend>
				{#each MODOS as m (m.clave)}
					<label class="modo" class:on={modo === m.clave}>
						<input type="radio" name="modo" value={m.clave} bind:group={modo} aria-describedby="m-{m.clave}" />
						<span class="modo-cabeza">
							<span class="radio" aria-hidden="true"></span>
							<b>{m.titulo}{#if m.simbolo}&nbsp;<span aria-hidden="true">{m.simbolo}</span>{/if}<span class="visualmente-oculto"> {m.oculto}</span></b>
						</span>
						<span class="modo-sub" id="m-{m.clave}">{m.sub}</span>
						<b class="modo-ej num">{ejemplo(m.clave)}</b>
					</label>
				{/each}
			</fieldset>

			<div class="cobro-pie">
				{#if modo === 'porcentaje'}
					<label class="g-campo corto">
						Porcentaje de la seña
						<input name="senaPorcentaje" type="number" min="1" max="100" bind:value={porcentaje} required />
					</label>
				{:else if modo === 'fija'}
					<label class="g-campo corto">
						Monto de la seña ($)
						<input name="senaFija" type="number" min="1" step="1" bind:value={fija} required />
					</label>
				{/if}
				<button class="g-boton oscuro">Guardar</button>
			</div>
			{#if ej && modo !== 'sin_cobro' && cobra > 0}
				<p class="g-aviso">
					<span class="icono lleno" aria-hidden="true">info</span>
					<span>
						Ejemplo con {ej.nombre.toLowerCase()} de {pesos(ej.precio)}: la clienta paga {pesos(cobra)} al reservar{#if ej.precio - cobra > 0}&nbsp;y {pesos(ej.precio - cobra)} en el&nbsp;local{/if}.
					</span>
				</p>
			{/if}
		</form>
	</section>

	<section class="g-bloque sin-pad" aria-labelledby="t-pagos">
		<div class="g-bloque-cabeza">
			<h2 id="t-pagos" class="g-bloque-titulo">Pagos recibidos</h2>
			<div class="pagos-meta">
				{#if cobrado > 0}<b class="num">{pesos(cobrado)} cobrados</b>{/if}
				<form method="GET">
					<label>
						<span class="visualmente-oculto">Estado</span>
						<select class="g-valor" name="estado" onchange={(e) => e.currentTarget.form?.requestSubmit()}>
							<option value="">Todos</option>
							{#each Object.entries(ESTADOS_PAGO) as [clave, nombre] (clave)}
								<option value={clave} selected={clave === data.estado}>{nombre}</option>
							{/each}
						</select>
					</label>
					<noscript><button class="g-boton chico">Filtrar</button></noscript>
				</form>
			</div>
		</div>
		<div class="pad"><Mensaje {form} que="pagos" /></div>
		{#if data.pagos.length === 0}
			<p class="g-vacio">
				<span class="icono" aria-hidden="true">payments</span>
				Todavía no hay pagos{data.estado ? ' con ese estado' : ''}.
			</p>
		{:else}
			<ul class="g-filas" aria-label={mes ? `Pagos de ${mes}` : undefined}>
				{#each data.pagos as p (p.id)}
					{@const e = estadoDe(p)}
					<li>
						<div class="pago" class:apagado={apagado(p)}>
							<span class="g-meta">{cuandoCorto(p.creadoEn)}</span>
							<span class="g-fila-texto">
								<b>{p.cliente}</b>
								<span>{p.modo === 'total' ? 'Pago total' : 'Seña'} · {p.servicio} {fechaHora(p.inicio)}</span>
							</span>
							<b class="monto num" class:tachado={p.estado === 'devuelto'}>{pesos(p.monto)}</b>
							<span class="g-estado {e.tono}"><span class="icono lleno" aria-hidden="true">{e.icono}</span>{e.texto}</span>
							<span class="accion">
								{#if p.estado === 'aprobado' && p.mpPago && resta(p) > 0 && devolviendo !== p.id}
									<button class="g-accion peligro" onclick={() => ((devolviendo = p.id), (cuanto = 'todo'))}>Devolver</button>
								{/if}
							</span>
						</div>
						{#if devolviendo === p.id}
							<form method="POST" action="?/devolver" use:enhance={confirmar('¿Devolver la plata? No se puede deshacer.')} class="devolucion">
								<input type="hidden" name="id" value={p.id} />
								<div class="g-chips" role="radiogroup" aria-label="Cuánto devolver">
									<label class="g-chip"><input type="radio" name="cuanto" value="todo" bind:group={cuanto} />Todo ({pesos(resta(p))}) y cancelar el turno</label>
									<label class="g-chip"><input type="radio" name="cuanto" value="parte" bind:group={cuanto} />Una parte</label>
								</div>
								{#if cuanto === 'parte'}
									<label class="g-campo corto">Monto ($) <input name="monto" type="number" min="1" max={resta(p) - 1} required /></label>
								{/if}
								<div class="fila">
									<button class="g-boton oscuro">Devolver</button>
									<button type="button" class="g-accion neutro" onclick={() => (devolviendo = null)}>Cancelar</button>
								</div>
							</form>
						{/if}
					</li>
				{/each}
			</ul>
		{/if}
	</section>

	{#if c.disponible && c.conectada && esDuenio}
		<form
			method="POST"
			action="?/desconectar"
			use:enhance={confirmar('¿Desconectar Mercado Pago? Los turnos nuevos no se van a cobrar por adelantado.')}
			class="desconectar"
		>
			<button class="g-accion peligro"><span class="icono" aria-hidden="true">link_off</span>Desconectar Mercado Pago</button>
		</form>
	{/if}
</div>

<style>
	.mp-chip {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		min-height: 44px;
		padding: 0 16px;
		border-radius: 999px;
		background: var(--tl-papel);
		font-size: 14px;
		flex-wrap: wrap;
	}
	.punto {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: var(--tl-menta);
	}
	.conectado {
		border-radius: var(--tl-r-m);
		overflow: hidden;
		background: var(--tl-papel);
	}
	.banda {
		margin: 0;
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 14px 18px;
		background-color: var(--tl-petroleo);
		background-image: var(--tl-grano);
		color: #fff;
		font-size: 15px;
	}
	.banda .icono {
		color: var(--tl-menta);
	}
	.conectado-cuerpo {
		padding: 16px 18px 20px;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.modos {
		border: 0;
		margin: 0;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		gap: 10px;
	}
	.modo {
		position: relative;
		display: flex;
		flex-direction: column;
		gap: 6px;
		padding: 16px;
		border-radius: var(--tl-r-m);
		background: var(--tl-papel);
		box-shadow: inset 0 0 0 1px #e6dcc8;
		color: var(--tl-tinta);
		cursor: pointer;
		transition: background var(--tl-tocar) var(--tl-ease);
	}
	.modo input {
		position: absolute;
		opacity: 0;
		width: 1px;
		height: 1px;
	}
	.modo:has(input:focus-visible) {
		outline: 3px solid var(--tl-menta);
		outline-offset: 2px;
	}
	.modo.on {
		background: var(--tl-petroleo);
		box-shadow: none;
		color: #fff;
	}
	.modo-cabeza {
		display: flex;
		align-items: center;
		gap: 8px;
	}
	.modo-cabeza b {
		font-size: 15px;
	}
	.radio {
		width: 20px;
		height: 20px;
		flex: none;
		border-radius: 50%;
		box-shadow: inset 0 0 0 1.5px #c9bda4;
	}
	.modo.on .radio {
		box-shadow: inset 0 0 0 6px var(--tl-menta);
		background: #fff;
	}
	.modo-sub {
		font-size: 13px;
		opacity: 0.85;
	}
	.modo-ej {
		margin-top: 4px;
		font-size: 15px;
	}
	.cobro-pie {
		display: flex;
		flex-wrap: wrap;
		align-items: flex-end;
		gap: 12px;
	}
	.corto {
		width: min(240px, 100%);
	}
	.pagos-meta {
		display: flex;
		align-items: center;
		gap: 12px;
		flex-wrap: wrap;
		font-size: 14px;
	}
	.pagos-meta form {
		display: flex;
		gap: 8px;
	}
	.pad:empty {
		display: none;
	}
	.pad {
		padding: 0 18px 12px;
	}
	.pago {
		display: grid;
		grid-template-columns: 110px minmax(0, 1fr) 120px 190px 90px;
		align-items: center;
		gap: 14px;
		min-height: 56px;
		padding: 6px 18px;
	}
	.pago.apagado {
		opacity: 0.7;
	}
	.monto {
		font-size: 15px;
	}
	.tachado {
		text-decoration: line-through;
	}
	.accion {
		justify-self: end;
	}
	.devolucion {
		display: flex;
		flex-direction: column;
		gap: 12px;
		margin: 0 18px 14px;
		padding: 14px;
		border-radius: 14px;
		background: var(--tl-crema);
	}
	.desconectar {
		display: flex;
		justify-content: flex-end;
	}
	@media (max-width: 900px) {
		.modos {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
		.pago {
			grid-template-columns: minmax(0, 1fr) auto;
			gap: 4px 12px;
			padding: 12px 14px;
		}
		.pago > .g-meta {
			grid-column: 1 / -1;
		}
		.pago .monto {
			grid-row: 2;
			grid-column: 2;
		}
		.pago .g-estado {
			white-space: normal;
		}
		.accion {
			justify-self: end;
		}
	}
	@media (max-width: 640px) {
		.modo {
			padding: 14px;
		}
		.modo-sub {
			font-size: 13px;
		}
	}
</style>
