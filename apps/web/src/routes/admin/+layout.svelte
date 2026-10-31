<script lang="ts">
	import { page } from '$app/state';
	let { data, children } = $props();
	const enlaces = [
		{ href: '/admin', nombre: 'Resumen' },
		{ href: '/admin/negocios', nombre: 'Negocios' }
	];
</script>

<div class="admin">
	<header class="barra">
		<strong>Tengo Libre · Administración</strong>
		<nav aria-label="Administración" class="fila">
			{#each enlaces as e (e.href)}
				<a href={e.href} aria-current={page.url.pathname === e.href ? 'page' : undefined}>{e.nombre}</a>
			{/each}
		</nav>
		<div class="fila chico">
			<a href="/mi-cuenta" class="suave">{data.persona.email || 'Mi cuenta'}</a>
			<form method="POST" action="/salir"><button class="boton">Salir</button></form>
		</div>
	</header>
	<main class="contenedor">
		{@render children()}
	</main>
</div>

<style>
	.barra {
		display: flex;
		flex-wrap: wrap;
		gap: 8px 16px;
		align-items: center;
		justify-content: space-between;
		padding: 10px 16px;
		border-bottom: 1px solid var(--borde);
		background: var(--superficie);
	}
	nav a {
		padding: 8px 10px;
		border-radius: 8px;
		text-decoration: none;
	}
	nav a[aria-current='page'] {
		background: var(--marca-suave);
		font-weight: 600;
	}
	.barra .chico a {
		overflow-wrap: anywhere;
	}
</style>
