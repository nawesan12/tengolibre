<!--
	El buscador global (handoff «Pantallas que faltaban», 1a): atajo «/», busca clientes, turnos y
	servicios mientras se escribe, agrupados; el primero queda elegido y Enter lo abre. Sin JS, el
	formulario va a /panel/buscar.
-->
<script lang="ts">
	import { goto } from '$app/navigation';
	import Icono from '#lib/ui/Icono.svelte';
	import { dia, hora, pesos } from '#lib/formato.ts';
	import { ESTILO } from '#lib/operacion/estados.ts';
	import type { Encontrado } from '#lib/operacion/tipos.ts';
	import { iniciales } from './dia.ts';

	let { pantalla = false, alCerrar }: { pantalla?: boolean; alCerrar?: () => void } = $props();

	let q = $state('');
	let res = $state<Encontrado>({ clientes: [], turnos: [], servicios: [] });
	let abierto = $state(false);
	let elegido = $state(0);
	let buscando = $state(false);
	let campo: HTMLInputElement | undefined = $state();
	let reloj: ReturnType<typeof setTimeout> | undefined;
	let pedido = 0;

	type Item = { href: string; clave: string };
	const items = $derived<Item[]>([
		...res.clientes.map((c) => ({ href: `/panel/clientes/${c.id}`, clave: `c${c.id}` })),
		...res.turnos.map((t) => ({ href: `/panel/turnos/${t.id}`, clave: `t${t.id}` })),
		...res.servicios.map((s) => ({ href: '/panel/servicios', clave: `s${s.id}` }))
	]);
	const indice = (clave: string) => items.findIndex((i) => i.clave === clave);
	const nada = $derived(q.trim().length >= 2 && !buscando && items.length === 0);

	$effect(() => {
		if (pantalla) campo?.focus();
	});

	function escribir() {
		clearTimeout(reloj);
		elegido = 0;
		const texto = q.trim();
		if (texto.length < 2) {
			res = { clientes: [], turnos: [], servicios: [] };
			return;
		}
		buscando = true;
		reloj = setTimeout(async () => {
			const n = ++pedido;
			try {
				const r = await fetch(`/panel/buscar/rapido?${new URLSearchParams({ q: texto })}`);
				if (n === pedido && r.ok) res = await r.json();
			} catch {
				/* sin conexión: queda lo anterior */
			} finally {
				if (n === pedido) buscando = false;
			}
		}, 200);
	}

	function cerrar() {
		abierto = false;
		alCerrar?.();
	}

	function tecla(e: KeyboardEvent) {
		if (e.key === 'ArrowDown' && items.length) {
			e.preventDefault();
			elegido = (elegido + 1) % items.length;
		} else if (e.key === 'ArrowUp' && items.length) {
			e.preventDefault();
			elegido = (elegido - 1 + items.length) % items.length;
		} else if (e.key === 'Escape') {
			e.preventDefault();
			if (q) q = '';
			else campo?.blur();
			cerrar();
		}
	}

	function enviar(e: SubmitEvent) {
		const it = items[elegido];
		if (it && abierto) {
			e.preventDefault();
			cerrar();
			q = '';
			goto(it.href);
		}
	}

	/** El atajo «/» desde cualquier lado que no sea un campo. */
	function atajo(e: KeyboardEvent) {
		if (pantalla || e.key !== '/' || e.metaKey || e.ctrlKey || e.altKey) return;
		const t = e.target as HTMLElement;
		if (t.closest('input, textarea, select, [contenteditable="true"]')) return;
		e.preventDefault();
		campo?.focus();
	}

	const cuando = (iso: string) => (dia(iso) === dia() ? 'Hoy' : new Intl.DateTimeFormat('es-AR', { weekday: 'short', day: '2-digit', timeZone: 'America/Argentina/Buenos_Aires' }).format(new Date(iso)).replace('.', ''));
</script>

<svelte:window onkeydown={atajo} />

<form
	method="GET"
	action="/panel/buscar"
	role="search"
	class="buscador"
	class:pantalla
	onsubmit={enviar}
	onfocusin={() => (abierto = true)}
	onfocusout={(e) => {
		if (!e.currentTarget.contains(e.relatedTarget as Node)) abierto = false;
	}}
>
	<label class="caja">
		<Icono nombre="search" tam={20} />
		<span class="visualmente-oculto">Buscar en el panel</span>
		<input
			bind:this={campo}
			bind:value={q}
			oninput={escribir}
			onkeydown={tecla}
			name="q"
			type="search"
			minlength="2"
			autocomplete="off"
			placeholder="Buscar cliente, teléfono o turno"
			aria-controls="resultados-busqueda"
			aria-activedescendant={abierto && items[elegido] ? `r-${items[elegido].clave}` : undefined}
		/>
		{#if pantalla}
			<button type="button" class="cerrar" aria-label="Cerrar el buscador" onclick={cerrar}><Icono nombre="close" tam={20} /></button>
		{:else}
			<kbd class="tl-num">{abierto && q ? 'ESC' : '/'}</kbd>
		{/if}
	</label>

	{#if (abierto || pantalla) && q.trim().length >= 2}
		<div id="resultados-busqueda" class="resultados" role="listbox" aria-label="Resultados">
			{#if res.clientes.length}
				<span class="tl-rotulo grupo">Clientes</span>
				{#each res.clientes as c (c.id)}
					{@const i = indice(`c${c.id}`)}
					<a id={`r-c${c.id}`} role="option" aria-selected={i === elegido} class="res" class:elegido={i === elegido} href={`/panel/clientes/${c.id}`} onclick={cerrar}>
						<span class="avatar">{iniciales(c.nombre)}</span>
						<span class="texto"><b>{c.nombre}</b><span class="tl-num">{[c.telefono, c.email].filter(Boolean).join(' · ')}</span></span>
						{#if i === elegido}<span class="tl-rotulo enter">Enter ↵</span>{/if}
					</a>
				{/each}
			{/if}
			{#if res.turnos.length}
				<span class="tl-rotulo grupo">Turnos</span>
				{#each res.turnos as t (t.id)}
					{@const i = indice(`t${t.id}`)}
					<a id={`r-t${t.id}`} role="option" aria-selected={i === elegido} class="res talon" class:elegido={i === elegido} href={`/panel/turnos/${t.id}`} onclick={cerrar}>
						<span class="cuando tl-num"><b>{cuando(t.inicio)}</b><b>{hora(t.inicio)}</b></span>
						<span class="texto"><b>{t.cliente}</b><span>{t.servicio}{t.profesional ? ` · ${t.profesional}` : ''}</span></span>
						<span class="tl-rotulo est" style:color={t.estado === 'confirmado' ? '#17805a' : '#4a5d61'}>{ESTILO[t.estado]?.texto ?? t.estado}</span>
					</a>
				{/each}
			{/if}
			{#if res.servicios.length}
				<span class="tl-rotulo grupo">Servicios</span>
				{#each res.servicios as s (s.id)}
					{@const i = indice(`s${s.id}`)}
					<a id={`r-s${s.id}`} role="option" aria-selected={i === elegido} class="res" class:elegido={i === elegido} href="/panel/servicios" onclick={cerrar}>
						<Icono nombre="spa" tam={22} />
						<span class="texto"><b>{s.nombre}</b><span class="tl-num">{s.duracion} min · {pesos(s.precio)}{s.activo ? '' : ' · inactivo'}</span></span>
					</a>
				{/each}
			{/if}
			{#if nada}<p class="nada">No encontramos nada con «{q.trim()}».</p>{/if}
			{#if buscando && items.length === 0}<p class="nada">Buscando…</p>{/if}
		</div>
	{/if}
</form>

<style>
	.buscador {
		position: relative;
		flex: 1 1 440px;
		min-width: 0;
		max-width: 440px;
	}
	.caja {
		display: flex;
		align-items: center;
		gap: 10px;
		height: 44px;
		padding: 0 8px 0 14px;
		border-radius: 14px;
		background: var(--tl-papel);
		box-shadow: 0 1px 0 rgba(15, 35, 40, 0.05), inset 0 0 0 1px #e6dcc8;
		color: var(--tl-sub);
		transition: box-shadow var(--tl-tocar) var(--tl-ease);
	}
	.caja:focus-within {
		box-shadow: inset 0 0 0 2px var(--tl-menta);
	}
	input {
		flex: 1;
		min-width: 0;
		border: 0;
		outline: none;
		background: transparent;
		font: 400 14px var(--tl-font);
		color: var(--tl-tinta);
	}
	input::-webkit-search-cancel-button {
		display: none;
	}
	kbd {
		font: 700 11px var(--tl-font);
		padding: 3px 7px;
		border-radius: 6px;
		background: var(--tl-crema);
		color: var(--tl-sub);
	}
	.resultados {
		position: absolute;
		left: 0;
		top: 52px;
		z-index: 40;
		width: max(100%, 480px);
		max-height: min(70vh, 560px);
		overflow-y: auto;
		padding: 12px 16px 16px;
		border-radius: 24px;
		background-color: var(--tl-crema);
		background-image: var(--tl-grano-osc), radial-gradient(rgba(24, 60, 68, 0.1) 1.2px, transparent 1.4px);
		background-size: auto, 18px 18px;
		box-shadow: 0 0 0 1px #e6dcc8, 0 30px 60px -20px rgba(15, 35, 40, 0.45);
		display: flex;
		flex-direction: column;
		gap: 8px;
		animation: tl-gin 300ms var(--tl-ease) both;
	}
	.grupo {
		font-size: 11px;
		color: var(--tl-sub);
		margin: 8px 0 0;
	}
	.res {
		display: flex;
		align-items: center;
		gap: 12px;
		min-height: 56px;
		padding: 8px 14px;
		border-radius: 14px;
		background: var(--tl-papel);
		color: var(--tl-tinta);
		text-decoration: none;
		box-shadow: 0 1px 0 rgba(15, 35, 40, 0.05);
	}
	.res :global(.icono) {
		color: var(--tl-menta-texto);
	}
	.res.elegido {
		background: #f3fbf7;
		box-shadow: inset 0 0 0 2px var(--tl-menta);
	}
	.avatar {
		flex: none;
		width: 36px;
		height: 36px;
		border-radius: 50%;
		background: var(--tl-crema);
		display: grid;
		place-items: center;
		font-size: 13px;
		font-weight: 700;
	}
	.texto {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		line-height: 1.3;
	}
	.texto b {
		font-size: 15px;
		font-weight: 700;
	}
	.texto span {
		font-size: 12px;
		color: var(--tl-sub);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.enter {
		font-size: 12px;
		color: var(--tl-menta-texto);
	}
	.talon {
		padding: 0 14px 0 0;
		background:
			radial-gradient(circle at 84px 0, var(--tl-crema) 7px, transparent 7.5px),
			radial-gradient(circle at 84px 100%, var(--tl-crema) 7px, transparent 7.5px),
			var(--tl-papel);
	}
	.talon.elegido {
		background:
			radial-gradient(circle at 84px 0, var(--tl-crema) 7px, transparent 7.5px),
			radial-gradient(circle at 84px 100%, var(--tl-crema) 7px, transparent 7.5px),
			#f3fbf7;
	}
	.cuando {
		flex: none;
		width: 84px;
		align-self: stretch;
		display: flex;
		flex-direction: column;
		justify-content: center;
		padding: 0 12px;
		border-right: 2px dashed var(--tl-linea-2);
		font-size: 14px;
		line-height: 1.15;
		text-transform: uppercase;
	}
	.est {
		font-size: 11px;
	}
	.nada {
		margin: 8px 4px;
		font-size: 14px;
		color: var(--tl-sub);
	}
	.cerrar {
		width: 40px;
		height: 40px;
		border: 0;
		border-radius: 10px;
		background: transparent;
		display: grid;
		place-items: center;
		color: var(--tl-tinta);
		cursor: pointer;
	}
	/* En el teléfono: pantalla completa debajo de la barra. */
	.pantalla {
		position: fixed;
		inset: 0;
		z-index: 70;
		max-width: none;
		padding: 12px 14px;
		background: var(--tl-crema);
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.pantalla .caja {
		height: 52px;
	}
	.pantalla input {
		font-size: 16px;
	}
	.pantalla .resultados {
		position: static;
		width: 100%;
		max-height: none;
		flex: 1;
		box-shadow: none;
		padding: 0;
		background: transparent;
		animation: none;
	}
</style>
