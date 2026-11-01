<script lang="ts">
	import '#lib/gestion/gestion.css';
	import { enhance } from '$app/forms';
	import { fecha, pesos } from '#lib/formato.ts';
	import { sinReset } from '#lib/configuracion/formularios.ts';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';

	let { data, form } = $props();
	// svelte-ignore state_referenced_locally
	let tipo = $state(data.config.tipo ?? '');
	// svelte-ignore state_referenced_locally
	let valor = $state<number | null>(data.config.valor);
	// svelte-ignore state_referenced_locally
	let activo = $state(data.config.activo);
	let formulario: HTMLFormElement | undefined = $state();

	const descuento = $derived(valor ? (tipo === 'fijo' ? pesos(valor) : `${valor}%`) : tipo === 'fijo' ? '$ …' : '…%');
	const ESTADOS = {
		premio_usado: { texto: 'Premio usado', clase: 'neutro', icono: 'redeem' },
		premio_disponible: { texto: 'Premio disponible', clase: 'espera', icono: 'featured_seasonal_and_gifts' },
		turno: { texto: 'Con turno', clase: 'ok', icono: 'event_available' }
	} as const;
</script>

<svelte:head><title>Referidos · Panel</title></svelte:head>

<div class="g-pagina pagina">
	<div class="g-cabeza cabeza">
		<h1 class="g-titulo">Referidos</h1>
		<label class="activo">
			<span class="visualmente-oculto">Programa de referidos activo</span>
			<input
				class="g-interruptor"
				type="checkbox"
				role="switch"
				name="activo"
				form="f-referidos"
				bind:checked={activo}
				onchange={() => formulario?.requestSubmit()}
			/>
			<span class="palabra" aria-hidden="true">{activo ? 'Activo' : 'Apagado'}</span>
		</label>
	</div>
	<p class="g-bajada">
		Cada cliente pide su código desde el link de su turno y lo comparte. Quien lo usa reserva, y quien lo compartió se lleva un&nbsp;premio.
	</p>

	<section class="g-oscuro premio" aria-labelledby="t-premio">
		<h2 class="g-kicker" id="t-premio">El premio</h2>
		<Mensaje {form} que="config" />
		<p class="frase">
			{#if tipo}
				Quien recomienda gana <b>{descuento}</b> de descuento en un cupón de un uso, cuando su referido reserva por primera&nbsp;vez.
			{:else}
				Sin premio: tus clientes comparten su código igual, pero quien recomienda no se lleva&nbsp;nada.
			{/if}
		</p>
		<form id="f-referidos" method="POST" action="?/config" use:enhance={sinReset} bind:this={formulario} class="campos">
			<fieldset class="g-chips tipos">
				<legend class="visualmente-oculto">Tipo de premio</legend>
				<label class="g-chip"><input type="radio" name="tipo" value="porcentaje" bind:group={tipo} />Porcentaje</label>
				<label class="g-chip"><input type="radio" name="tipo" value="fijo" bind:group={tipo} />Monto fijo</label>
				<label class="g-chip"><input type="radio" name="tipo" value="" bind:group={tipo} />Sin premio</label>
			</fieldset>
			{#if tipo}
				<label class="g-campo">
					Descuento {tipo === 'fijo' ? '($)' : '(%)'}
					<input name="valor" type="number" min="1" max={tipo === 'fijo' ? undefined : 100} bind:value={valor} required inputmode="numeric" />
				</label>
			{/if}
			<label class="g-campo">
				Usos por código
				<input name="tope" type="number" min="1" value={data.config.tope ?? ''} placeholder="Sin límite" inputmode="numeric" />
			</label>
			<div class="guardar"><button class="g-boton menta">Guardar</button></div>
		</form>
	</section>

	{#if data.usos && data.usos.length > 0}
		<section class="g-bloque sin-pad" aria-labelledby="t-usos">
			<h2 class="visualmente-oculto" id="t-usos">Quién trajo a quién</h2>
			<ul class="g-filas sin-borde-arriba">
				{#each data.usos as u (u.id)}
					{@const e = ESTADOS[u.estado]}
					<li class="g-fila uso">
						<b class="quien">{u.refiere}</b>
						<span class="flecha icono" aria-label="trajo a">arrow_forward</span>
						<span class="quien">{u.referido}</span>
						<span class="g-estado {e.clase}">
							<span class="icono" aria-hidden="true">{e.icono}</span>
							{u.estado === 'turno' && u.turnoInicio ? `Turno el ${fecha(u.turnoInicio)}` : e.texto}
						</span>
					</li>
				{/each}
			</ul>
		</section>
	{/if}

	<section class="g-bloque sin-pad" aria-labelledby="t-codigos">
		<div class="g-bloque-cabeza"><h2 class="g-bloque-titulo" id="t-codigos">Códigos de tus clientes</h2></div>
		{#if data.referidos.length === 0}
			<div class="g-vacio"><span class="icono" aria-hidden="true">group_add</span>Todavía ningún cliente pidió su&nbsp;código.</div>
		{:else}
			<ul class="g-filas">
				{#each data.referidos as r (r.id)}
					<li class="g-fila">
						<span class="g-fila-texto"><b>{r.cliente}</b><span>Desde {fecha(r.creadoEn)}</span></span>
						<span class="codigo">{r.codigo}</span>
						<span class="g-meta">{r.usos}&nbsp;{r.usos === 1 ? 'uso' : 'usos'}</span>
					</li>
				{/each}
			</ul>
		{/if}
	</section>
</div>

<style>
	.pagina {
		max-width: 620px;
		gap: 14px;
	}
	.cabeza {
		align-items: center;
	}
	.activo {
		display: inline-flex;
		flex-direction: row-reverse;
		align-items: center;
		gap: 10px;
		min-height: 48px;
		cursor: pointer;
	}
	.palabra {
		font: 700 13px var(--tl-font);
		color: var(--tl-sub-2);
	}
	.activo .g-interruptor {
		width: 48px;
		height: 28px;
	}
	.activo .g-interruptor::after {
		width: 22px;
		height: 22px;
	}
	.activo .g-interruptor:checked::after {
		transform: translateX(20px);
	}
	.premio {
		border-radius: var(--tl-r-l);
		padding: 20px;
		gap: 12px;
	}
	.premio .g-kicker {
		margin: 0;
		color: var(--tl-menta);
	}
	.frase {
		margin: 0;
		font-size: 15px;
		line-height: 1.5;
	}
	.frase b {
		color: var(--tl-menta);
		font-variant-numeric: tabular-nums;
	}
	.campos {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 12px;
	}
	.tipos {
		grid-column: 1 / -1;
		border: 0;
		margin: 0;
		padding: 0;
	}
	.tipos .g-chip {
		position: relative;
		background: rgba(255, 255, 255, 0.1);
		color: #fff;
		box-shadow: none;
	}
	.tipos .g-chip:has(input:checked) {
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
	}
	.premio .g-campo {
		color: rgba(255, 255, 255, 0.86);
	}
	.guardar {
		grid-column: 1 / -1;
	}
	.premio :global(.aviso) {
		margin: 0;
	}
	.uso {
		gap: 10px;
	}
	.sin-borde-arriba > li:first-child {
		border-top: 0;
	}
	.quien {
		font-size: 14px;
		min-width: 0;
		overflow-wrap: anywhere;
	}
	b.quien {
		flex: 0 1 30%;
	}
	span.quien {
		flex: 1;
	}
	.flecha {
		font-size: 16px;
		color: var(--tl-menta-texto);
	}
	.uso .g-estado {
		font-size: 11px;
		text-transform: uppercase;
	}
	.codigo {
		font: 700 14px var(--tl-font);
		font-variant-numeric: tabular-nums;
		letter-spacing: 0.08em;
	}
	@media (max-width: 640px) {
		.campos {
			grid-template-columns: minmax(0, 1fr);
		}
		.uso {
			flex-wrap: wrap;
		}
		.uso .g-estado {
			flex-basis: 100%;
		}
	}
</style>
