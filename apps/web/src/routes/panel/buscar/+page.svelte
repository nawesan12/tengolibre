<!-- Buscar (handoff «Pantallas que faltaban», 1a) en página: lo mismo que el buscador de la barra. -->
<script lang="ts">
	import Icono from '#lib/ui/Icono.svelte';
	import { pesos } from '#lib/formato.ts';
	import Cabecera from '#lib/operacion/Cabecera.svelte';
	import { ESTILO } from '#lib/operacion/estados.ts';
	import { GESTION } from '#lib/operacion/secciones.ts';
	import { cuandoCorto } from '#lib/operacion/ui.ts';
	import { iniciales } from '#lib/panel/dia.ts';

	let { data } = $props();
	const nada = $derived(data.q.length >= 2 && !data.clientes.length && !data.turnos.length && !data.servicios.length);
</script>

<svelte:head><title>Buscar · Panel</title></svelte:head>

<Cabecera titulo="Buscar" />

<div class="hoja">
	<form method="GET" role="search" class="caja" data-sveltekit-reset="false" data-sveltekit-keepfocus>
		<Icono nombre="search" tam={22} />
		<label class="visualmente-oculto" for="q-buscar">Buscar en el panel</label>
		<input id="q-buscar" type="search" name="q" value={data.q} minlength="2" placeholder="Cliente, teléfono, email o servicio" />
		<button class="visualmente-oculto">Buscar</button>
	</form>

	{#if data.q.length < 2}
		<p class="nota">Escribí al menos dos letras o números. Desde cualquier pantalla, tocá «/» para&nbsp;buscar.</p>
	{:else if nada}
		<p class="nota">No encontramos nada con «{data.q}».</p>
	{/if}

	{#if data.clientes.length}
		<span class="tl-rotulo grupo">Clientes</span>
		{#each data.clientes as c (c.id)}
			<a class="res" href={`/panel/clientes/${c.id}`}>
				<span class="av">{iniciales(c.nombre)}</span>
				<span class="txt"><b>{c.nombre}</b><span class="tl-num">{[c.telefono, c.email].filter(Boolean).join(' · ')}</span></span>
			</a>
		{/each}
	{/if}

	{#if data.turnos.length}
		<span class="tl-rotulo grupo">Turnos</span>
		{#each data.turnos as t (t.id)}
			<a class="res talon" href={`/panel/turnos/${t.id}`}>
				<span class="cuando tl-num">{cuandoCorto(t.inicio)}</span>
				<span class="txt"><b>{t.cliente}</b><span>{t.servicio}{t.profesional ? ` · ${t.profesional}` : ''}</span></span>
				<span class="tl-rotulo est" style:color={t.estado === 'confirmado' ? '#17805a' : '#4a5d61'}>{ESTILO[t.estado]?.texto ?? t.estado}</span>
			</a>
		{/each}
	{/if}

	{#if data.servicios.length}
		<span class="tl-rotulo grupo">Servicios</span>
		{#each data.servicios as s (s.id)}
			<svelte:element this={GESTION.includes(data.negocio.rol) ? 'a' : 'div'} class="res" href={GESTION.includes(data.negocio.rol) ? '/panel/servicios' : undefined}>
				<Icono nombre="spa" tam={22} />
				<span class="txt"><b>{s.nombre}</b><span class="tl-num">{s.duracion} min · {pesos(s.precio)}{s.activo ? '' : ' · inactivo'}</span></span>
			</svelte:element>
		{/each}
	{/if}
</div>

<style>
	.hoja {
		max-width: 620px;
		padding: 24px;
		border-radius: 24px;
		background-color: var(--tl-crema);
		background-image: var(--tl-grano-osc), radial-gradient(rgba(24, 60, 68, 0.1) 1.2px, transparent 1.4px);
		background-size: auto, 18px 18px;
		box-shadow: var(--tl-sh-alta);
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.caja {
		display: flex;
		align-items: center;
		gap: 10px;
		height: 52px;
		padding: 0 16px;
		border-radius: 16px;
		background: var(--tl-papel);
		box-shadow: inset 0 0 0 2px var(--tl-menta);
		color: var(--tl-sub);
	}
	.caja input {
		flex: 1;
		min-width: 0;
		border: 0;
		outline: none;
		background: transparent;
		font: 400 16px var(--tl-font);
		color: var(--tl-tinta);
	}
	.nota {
		margin: 0;
		font-size: 14px;
		color: var(--tl-sub);
	}
	.grupo {
		font-size: 11px;
		color: var(--tl-sub);
		margin-top: 8px;
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
	}
	.res :global(.icono) {
		color: var(--tl-menta-texto);
	}
	.av {
		width: 36px;
		height: 36px;
		border-radius: 50%;
		background: var(--tl-crema);
		display: grid;
		place-items: center;
		font-size: 13px;
		font-weight: 700;
		flex: none;
	}
	.txt {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
	}
	.txt b {
		font-size: 15px;
	}
	.txt span {
		font-size: 12px;
		color: var(--tl-sub);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.talon {
		padding-left: 0;
		background:
			radial-gradient(circle at 96px 0, var(--tl-crema) 7px, transparent 7.5px),
			radial-gradient(circle at 96px 100%, var(--tl-crema) 7px, transparent 7.5px),
			var(--tl-papel);
	}
	.cuando {
		width: 96px;
		align-self: stretch;
		display: flex;
		align-items: center;
		padding: 0 12px;
		border-right: 2px dashed var(--tl-linea-2);
		font-size: 13px;
		font-weight: 700;
		flex: none;
	}
	.est {
		font-size: 11px;
	}
	@media (max-width: 859px) {
		.hoja {
			padding: 16px;
		}
	}
</style>
