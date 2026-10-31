<!--
	Un turno en la lista (handoff «Panel v2», Turnos): talón con la hora, el cliente, el servicio, el
	estado con ícono y palabra, cómo está la plata, la acción principal y el menú «⋯». En un lugar angosto
	(el teléfono) se apila en dos columnas. Toda acción tiene su botón.
-->
<script lang="ts">
	import { enhance } from '$app/forms';
	import Icono from '#lib/ui/Icono.svelte';
	import { hora, pesos } from '#lib/formato.ts';
	import EstadoTurno from './EstadoTurno.svelte';
	import { ICONO_ACCION, accionPrincipal, accionesRapidas, estadoVisible } from './estados.ts';
	import type { Turno } from './tipos.ts';

	let {
		t,
		ahora,
		puedeActuar = true,
		recien = false,
		alAtender
	}: { t: Turno; ahora: number; puedeActuar?: boolean; recien?: boolean; alAtender?: (t: Turno) => void } = $props();

	const est = $derived(estadoVisible(t.estado, t.inicio, t.fin, ahora));
	const principal = $derived(puedeActuar ? accionPrincipal(t.estado, t.inicio, ahora) : null);
	const menu = $derived(puedeActuar ? accionesRapidas(t.estado) : []);
	let abierto = $state(false);
	let enviando = $state(false);
	let raiz: HTMLElement | undefined = $state();

	const pago = $derived.by(() => {
		if (t.estado === 'pendiente_pago') return { icono: 'hourglass_top', color: '#7a4a00', texto: t.venceEn ? `Seña vence ${hora(t.venceEn)}` : 'Esperando la seña' };
		const s = t.sena;
		if (s === null) return { icono: 'money_off', color: '#4a5d61', texto: 'Sin seña' };
		if (!s) return { icono: 'sell', color: '#4a5d61', texto: pesos(t.precio) };
		if (s.estado === 'pagada') return { icono: 'paid', color: '#17805a', texto: `Seña ${pesos(s.monto)}` };
		if (s.estado === 'devuelta') return { icono: 'currency_exchange', color: '#a3241a', texto: 'Seña devuelta' };
		return { icono: 'hourglass_top', color: '#7a4a00', texto: `Seña ${pesos(s.monto)} pendiente` };
	});
	const confirmo = $derived(
		t.confirmacion?.via === 'whatsapp' && t.confirmacion.respuesta
			? t.confirmacion.respuesta === 'confirmo'
				? `Confirmó por WhatsApp${t.confirmacion.en ? ` · ${hora(t.confirmacion.en)}` : ''}`
				: 'Pidió cambiarlo por WhatsApp'
			: ''
	);

	function fuera(e: MouseEvent) {
		if (abierto && raiz && !raiz.contains(e.target as Node)) abierto = false;
	}
	const trabajar = () => {
		enviando = true;
		return async ({ update }: { update: (o?: { reset?: boolean }) => Promise<void> }) => {
			await update({ reset: false });
			enviando = false;
			abierto = false;
		};
	};
</script>

<svelte:window onclick={fuera} onkeydown={(e) => e.key === 'Escape' && (abierto = false)} />

<li class="turno" data-turno={t.id} class:cancelado={t.estado === 'cancelado'} class:atendido={t.estado === 'atendido'}>
	<div class="talon" class:en-curso={est === 'en_curso'} class:recien>
		<a class="stub tl-num" href={`/panel/turnos/${t.id}`} aria-label={`${hora(t.inicio)}, ${t.cliente.nombre}: abrir el turno`}>
			<b>{hora(t.inicio)}</b><span>{hora(t.fin)}</span>
			{#if t.estado === 'atendido'}<span class="picado" title="Atendido"></span>{/if}
		</a>
		<a class="quien" href={`/panel/turnos/${t.id}`} tabindex="-1">
			<span class="nombre"><b>{t.cliente.nombre}</b></span>
			{#if t.nota}<span class="nota"><Icono nombre="sticky_note_2" lleno tam={14} />{t.nota}</span>{/if}
			{#if confirmo}<span class="wa"><img src="/logos/whatsapp.svg" alt="" width="13" height="13" />{confirmo}</span>{/if}
			{#if !t.nota && !confirmo}<span class="tel tl-num">{t.cliente.telefono || t.cliente.email}</span>{/if}
		</a>
		<span class="que"><span class="srv">{t.servicio}</span><span class="con">con {t.profesional || 'cualquiera'}</span></span>
		<span class="estado-pago">
			<span class="e"><EstadoTurno estado={est} /></span>
			<span class="pago tl-num" style:color={pago.color}><Icono nombre={pago.icono} lleno tam={14} />{pago.texto}</span>
		</span>
		<span class="acciones" bind:this={raiz}>
			{#if principal}
				{#if principal.estado === 'atendido' && alAtender}
					<button class="accion" onclick={() => alAtender(t)}><Icono nombre={principal.icono} tam={16} />{principal.texto}</button>
				{:else}
					<form method="POST" action="?/estado" use:enhance={trabajar}>
						<input type="hidden" name="id" value={t.id} />
						<button class="accion" name="estado" value={principal.estado} disabled={enviando}><Icono nombre={principal.icono} tam={16} />{principal.texto}</button>
					</form>
				{/if}
			{/if}
			<button class="mas" aria-label={`Más acciones para ${t.cliente.nombre}, ${hora(t.inicio)}`} aria-expanded={abierto} onclick={() => (abierto = !abierto)}>
				<Icono nombre="more_horiz" tam={20} />
			</button>
			{#if abierto}
				<div class="menu" role="menu">
					<a role="menuitem" href={`/panel/turnos/${t.id}`}><Icono nombre="open_in_full" tam={20} />Abrir detalle</a>
					{#if menu.length}
						<form method="POST" action="?/estado" use:enhance={trabajar}>
							<input type="hidden" name="id" value={t.id} />
							{#each menu as a (a.estado)}
								{#if a.estado === 'atendido' && alAtender}
									<button type="button" role="menuitem" aria-label={`${a.texto}: ${t.cliente.nombre}`} onclick={() => ((abierto = false), alAtender(t))}>
										<Icono nombre={ICONO_ACCION[a.estado]} tam={20} />{a.texto}
									</button>
								{:else}
									<button
										role="menuitem"
										name="estado"
										value={a.estado}
										class:peligro={a.estado === 'cancelado'}
										aria-label={`${a.texto}: ${t.cliente.nombre}`}
										disabled={enviando}
										onclick={(e) => {
											if (a.estado === 'cancelado' && !confirm('¿Cancelar este turno? Se le avisa al cliente y queda el hueco.')) e.preventDefault();
										}}><Icono nombre={ICONO_ACCION[a.estado]} tam={20} />{a.estado === 'cancelado' ? 'Cancelar turno' : a.texto}</button
									>
								{/if}
							{/each}
						</form>
						{#if t.estado !== 'atendido' && t.estado !== 'ausente'}
							<a role="menuitem" href={`/panel/turnos/${t.id}#reprogramar`}><Icono nombre="event_repeat" tam={20} />Reprogramar</a>
						{/if}
					{/if}
				</div>
			{/if}
		</span>
	</div>
</li>

<style>
	.turno {
		list-style: none;
		animation: tl-gin var(--tl-entrar) var(--tl-ease) both;
	}
	.talon {
		--x: 112px;
		--fondo: #fffdf9;
		display: grid;
		grid-template-columns: var(--x) minmax(0, 1.25fr) minmax(0, 1fr) 168px 152px;
		align-items: stretch;
		min-height: 64px;
		border-radius: 14px;
		background:
			radial-gradient(circle at var(--x) 0, var(--tl-crema) 9px, transparent 9.5px),
			radial-gradient(circle at var(--x) 100%, var(--tl-crema) 9px, transparent 9.5px),
			var(--fondo);
		box-shadow: 0 1px 0 rgba(15, 35, 40, 0.05), 0 10px 20px -16px rgba(15, 35, 40, 0.45);
		transition: transform 200ms var(--tl-ease), box-shadow 200ms;
	}
	.talon:hover {
		transform: translateY(-2px);
	}
	.atendido .talon {
		--fondo: #f8f2e6;
	}
	.cancelado .talon {
		opacity: 0.55;
	}
	.cancelado .stub b {
		text-decoration: line-through;
	}
	.talon.en-curso {
		box-shadow: 0 0 0 2px var(--tl-petroleo), 0 10px 20px -16px rgba(15, 35, 40, 0.45);
	}
	.talon.recien {
		--fondo: #effaf4;
		box-shadow: 0 0 0 2px var(--tl-menta), 0 12px 24px -14px rgba(23, 128, 90, 0.6);
	}
	.stub {
		position: relative;
		display: flex;
		flex-direction: column;
		justify-content: center;
		padding: 0 18px;
		border-right: 2px dashed var(--tl-linea-2);
		color: var(--tl-tinta);
		text-decoration: none;
	}
	.stub b {
		font-size: 21px;
		font-weight: 700;
		letter-spacing: -0.02em;
	}
	.stub > span:not(.picado) {
		font-size: 11px;
		color: var(--tl-sub);
	}
	.picado {
		position: absolute;
		right: 12px;
		top: 50%;
		margin-top: -7px;
		width: 14px;
		height: 14px;
		border-radius: 50%;
		background: var(--tl-crema);
		box-shadow: inset 0 1px 2px rgba(15, 35, 40, 0.35);
	}
	.quien {
		display: flex;
		flex-direction: column;
		justify-content: center;
		gap: 1px;
		padding: 8px 18px;
		min-width: 0;
		color: var(--tl-tinta);
		text-decoration: none;
	}
	.nombre {
		display: flex;
		align-items: center;
		gap: 8px;
	}
	.nombre b {
		font-size: 15px;
		font-weight: 700;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.nota,
	.wa {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 12px;
		color: #8a5300;
		min-width: 0;
	}
	.wa {
		color: var(--tl-menta-texto);
	}
	.tel {
		font-size: 12px;
		color: var(--tl-sub);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.que {
		display: flex;
		flex-direction: column;
		justify-content: center;
		min-width: 0;
	}
	.srv {
		font-size: 14px;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.con {
		font-size: 12px;
		color: var(--tl-sub);
	}
	.estado-pago {
		display: flex;
		flex-direction: column;
		justify-content: center;
		gap: 4px;
		align-items: flex-start;
	}
	.pago {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		font-size: 11px;
		font-weight: 700;
	}
	.acciones {
		position: relative;
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: 6px;
		padding: 0 12px 0 8px;
	}
	.accion {
		display: inline-flex;
		align-items: center;
		gap: 5px;
		height: 36px;
		padding: 0 13px;
		border-radius: 999px;
		border: 0;
		background: var(--tl-petroleo);
		color: #fff;
		font: 700 13px var(--tl-font);
		cursor: pointer;
		white-space: nowrap;
		transition: transform var(--tl-tocar), box-shadow var(--tl-tocar);
	}
	.accion :global(.icono) {
		color: var(--tl-menta);
	}
	.accion:hover {
		transform: translateY(-1px);
		box-shadow: 0 10px 18px -10px rgba(24, 60, 68, 0.8);
	}
	.accion:active {
		transform: scale(0.95);
	}
	.mas {
		width: 36px;
		height: 36px;
		border-radius: 50%;
		border: 0;
		background: transparent;
		display: grid;
		place-items: center;
		color: var(--tl-tinta);
		cursor: pointer;
		opacity: 0.45;
		transition: opacity 180ms, background 180ms;
	}
	.talon:hover .mas,
	.mas:focus-visible,
	.mas[aria-expanded='true'] {
		opacity: 1;
	}
	.mas[aria-expanded='true'] {
		background: #ece5d6;
	}
	.menu {
		position: absolute;
		right: 8px;
		top: calc(100% - 6px);
		z-index: 6;
		width: 210px;
		padding: 6px;
		border-radius: 16px;
		background: var(--tl-papel);
		box-shadow: 0 0 0 1px #e6dcc8, 0 20px 40px -14px rgba(15, 35, 40, 0.45);
		display: flex;
		flex-direction: column;
	}
	.menu form {
		display: contents;
	}
	.menu a,
	.menu button {
		display: flex;
		align-items: center;
		gap: 10px;
		min-height: 40px;
		padding: 0 10px;
		border-radius: 10px;
		border: 0;
		background: transparent;
		color: var(--tl-tinta);
		font: 700 14px var(--tl-font);
		text-align: left;
		text-decoration: none;
		cursor: pointer;
	}
	.menu a:hover,
	.menu button:hover {
		background: var(--tl-crema);
	}
	.menu .peligro {
		color: #a3241a;
	}

	/* Angosto: talón de dos columnas, todo apilado a la derecha. */
	@container lista (max-width: 760px) {
		.talon {
			--x: 76px;
			grid-template-columns: var(--x) minmax(0, 1fr) auto;
			grid-template-areas: 'stub quien e' 'stub que que' 'stub pago acc';
			border-radius: 16px;
		}
		.stub {
			grid-area: stub;
			padding: 0 12px;
		}
		.stub b {
			font-size: 18px;
			letter-spacing: -0.03em;
		}
		.picado {
			top: auto;
			bottom: 10px;
			right: 8px;
			margin: 0;
			width: 11px;
			height: 11px;
		}
		.tel {
			display: none;
		}
		.quien {
			grid-area: quien;
			padding: 12px 6px 0 14px;
		}
		.e {
			grid-area: e;
			padding: 12px 12px 0 0;
		}
		.e :global(.estado) {
			height: 22px;
			font-size: 11px;
		}
		.que {
			grid-area: que;
			padding: 2px 12px 0 14px;
			flex-direction: row;
			flex-wrap: wrap;
			justify-content: flex-start;
			gap: 0 4px;
		}
		.srv,
		.con {
			font-size: 13px;
			color: var(--tl-sub);
		}
		.srv::after {
			content: ' ·';
		}
		.con {
			white-space: nowrap;
		}
		.estado-pago {
			display: contents;
		}
		.pago {
			grid-area: pago;
			align-self: center;
			padding: 8px 0 12px 14px;
		}
		.acciones {
			grid-area: acc;
			padding: 6px 8px 10px 0;
		}
		.accion {
			height: 44px;
			padding: 0 16px;
		}
		.mas {
			width: 44px;
			height: 44px;
			opacity: 1;
		}
		.menu {
			right: 8px;
		}
	}
</style>
