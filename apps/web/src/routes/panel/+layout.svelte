<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { GRUPOS, activa, visible } from '#lib/panel/navegacion.ts';
	import { escucharPanel } from '#lib/tiempo-real.svelte.ts';

	let { data, children } = $props();
	let menu = $state(false);

	// Tiempo real mientras el panel está abierto.
	onMount(() => escucharPanel());
</script>

<div class="panel">
	<header class="barra">
		<button class="boton menu" aria-expanded={menu} aria-controls="nav-panel" onclick={() => (menu = !menu)}>Menú</button>
		<strong>{data.negocio.nombre}</strong>
		<form method="GET" action="/panel/buscar" role="search" class="buscar">
			<label for="buscar-panel" class="visualmente-oculto">Buscar en el panel</label>
			<input id="buscar-panel" type="search" name="q" placeholder="Buscar" minlength="2" />
		</form>
		<a class="boton campanita" href="/panel/notificaciones" aria-label={data.sinLeer ? `Notificaciones, ${data.sinLeer} sin ver` : 'Notificaciones'}>
			Avisos{#if data.sinLeer}<span class="contador" aria-hidden="true">{data.sinLeer > 99 ? '99+' : data.sinLeer}</span>{/if}
		</a>
		<a class="boton" href={`/${data.negocio.slug}`} target="_blank" rel="noopener">Ver mi página</a>
	</header>

	<nav id="nav-panel" class:abierto={menu} aria-label="Panel">
		{#if data.negocios.length > 1}
			<form method="POST" action="/panel/negocio" class="campo">
				<label for="cambiar-negocio" class="chico suave">Negocio</label>
				<select id="cambiar-negocio" name="id" onchange={(e) => e.currentTarget.form?.requestSubmit()}>
					{#each data.negocios as n (n.id)}
						<option value={n.id} selected={n.id === data.negocio.id}>{n.nombre}</option>
					{/each}
				</select>
			</form>
		{/if}
		{#each GRUPOS as grupo, i (i)}
			<ul>
				{#each grupo.filter((e) => visible(e, data.negocio.rol)) as e (e.href)}
					<li>
						<a href={e.href} aria-current={activa(page.url.pathname, e) ? 'page' : undefined} onclick={() => (menu = false)}>
							{e.nombre}
						</a>
					</li>
				{/each}
			</ul>
		{/each}
		<div class="persona chico">
			<span class="suave">{data.persona.email}</span>
			<form method="POST" action="/salir"><button class="boton">Salir</button></form>
		</div>
	</nav>

	<main class="contenido">
		{#if data.acceso?.bloqueado}
			<div class="aviso-lectura" role="status">
				<strong>Tu prueba gratis terminó.</strong>
				{#if data.negocio.rol === 'duenio'}
					Podés ver todo, pero para hacer cambios <a href="/panel/suscripcion">elegí un plan</a>.
				{:else}
					Podés ver todo; para hacer cambios, el dueño tiene que elegir un plan.
				{/if}
				Tu página sigue tomando reservas.
			</div>
		{:else if data.acceso && !data.acceso.conSuscripcion && data.acceso.pruebaHasta && data.acceso.diasDePrueba <= 3 && data.negocio.rol === 'duenio'}
			<div class="aviso-lectura suave" role="status">
				{data.acceso.diasDePrueba === 1 ? 'Te queda 1 día' : `Te quedan ${data.acceso.diasDePrueba} días`} de prueba.
				<a href="/panel/suscripcion">Elegí un plan</a> para no perder los cambios.
			</div>
		{/if}
		{@render children()}
	</main>
</div>

<style>
	.panel {
		min-height: 100dvh;
		display: grid;
		grid-template-columns: 240px 1fr;
		grid-template-rows: auto 1fr;
	}
	.barra {
		grid-column: 1 / -1;
		display: flex;
		gap: 12px;
		align-items: center;
		justify-content: space-between;
		padding: 10px 16px;
		border-bottom: 1px solid var(--borde);
		background: var(--superficie);
	}
	.menu {
		display: none;
	}
	.aviso-lectura {
		margin-bottom: 16px;
		padding: 10px 14px;
		border: 1px solid var(--borde);
		border-radius: 8px;
		background: var(--superficie);
		text-wrap: pretty;
	}
	.barra > strong {
		margin-right: auto;
	}
	.buscar input {
		font: inherit;
		min-height: 40px;
		width: 180px;
		padding: 8px 10px;
		border: 1px solid var(--borde);
		border-radius: 8px;
	}
	.contador {
		min-width: 20px;
		padding: 0 6px;
		border-radius: 999px;
		background: var(--peligro);
		color: #fff;
		font-size: 12px;
		line-height: 20px;
		text-align: center;
	}
	nav {
		padding: 12px;
		border-right: 1px solid var(--borde);
		background: var(--superficie);
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	ul {
		list-style: none;
		margin: 0;
		padding: 0 0 8px;
		border-bottom: 1px solid var(--borde);
	}
	li a {
		display: block;
		padding: 8px 10px;
		border-radius: 8px;
		text-decoration: none;
	}
	li a[aria-current='page'] {
		background: var(--marca-suave);
		font-weight: 600;
	}
	.persona {
		margin-top: auto;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	.contenido {
		padding: 20px 16px 48px;
		min-width: 0;
	}
	@media (max-width: 800px) {
		.panel {
			grid-template-columns: 1fr;
		}
		.menu {
			display: inline-flex;
		}
		.barra {
			flex-wrap: wrap;
		}
		.buscar {
			order: 10;
			flex-basis: 100%;
		}
		.buscar input {
			width: 100%;
		}
		nav {
			display: none;
			border-right: 0;
			border-bottom: 1px solid var(--borde);
		}
		nav.abierto {
			display: flex;
		}
	}
</style>
