<script lang="ts">
	import { page } from '$app/state';
	import Icono from '#lib/ui/Icono.svelte';
	let { data, children } = $props();
	const enlaces = [
		{ href: '/admin', nombre: 'Resumen', icono: 'monitoring' },
		{ href: '/admin/negocios', nombre: 'Negocios', icono: 'storefront' }
	];
</script>

<div class="admin">
	<aside class="riel tl-petroleo">
		<a href="/admin" class="marca" aria-label="Tengo Libre, resumen">
			<img src="/marca-icono.png" alt="" width="38" height="38" />
		</a>
		<span class="tl-num rotulo">ADMIN</span>
		<nav aria-label="Administración">
			{#each enlaces as e (e.href)}
				<a href={e.href} aria-current={page.url.pathname === e.href ? 'page' : undefined}>
					<Icono nombre={e.icono} lleno={page.url.pathname === e.href} tam={22} />
					<span>{e.nombre}</span>
				</a>
			{/each}
		</nav>
		<div class="abajo">
			<a href="/mi-cuenta" title={data.persona.email || 'Mi cuenta'}>
				<Icono nombre="account_circle" tam={22} />
				<span>Mi cuenta</span>
			</a>
			<form method="POST" action="/salir">
				<button><Icono nombre="logout" tam={22} /><span>Salir</span></button>
			</form>
		</div>
	</aside>
	<main class="contenido">
		{@render children()}
	</main>
</div>

<style>
	.admin {
		min-height: 100dvh;
		display: grid;
		grid-template-columns: 72px minmax(0, 1fr);
		background: var(--tl-crema);
	}
	.riel {
		position: sticky;
		top: 0;
		height: 100dvh;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		padding: 18px 0;
		background-color: var(--tl-petroleo-hondo);
	}
	.marca img {
		display: block;
		border-radius: 10px;
	}
	.rotulo {
		margin: 8px 0 12px;
		font-size: 11px;
		font-weight: 700;
		letter-spacing: 0.1em;
		color: var(--tl-menta);
	}
	nav,
	.abajo {
		display: flex;
		flex-direction: column;
		gap: 4px;
		width: 100%;
	}
	.abajo {
		margin-top: auto;
	}
	nav a,
	.abajo a,
	.abajo button {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 2px;
		width: 100%;
		min-height: 56px;
		padding: 8px 0;
		border: 0;
		background: none;
		color: rgba(255, 255, 255, 0.78);
		font: 700 11px var(--tl-font);
		text-decoration: none;
		cursor: pointer;
		transition: color var(--tl-tocar) var(--tl-ease);
	}
	nav a:hover,
	.abajo a:hover,
	.abajo button:hover {
		color: #fff;
	}
	/* Pestaña: el ítem activo se funde con la página como una solapa. */
	nav a[aria-current='page'] {
		position: relative;
		margin-left: 8px;
		width: calc(100% - 8px);
		border-radius: 16px 0 0 16px;
		background: var(--tl-crema);
		color: var(--tl-tinta);
	}
	nav a[aria-current='page'] :global(.icono) {
		color: var(--tl-menta-texto);
	}
	.contenido {
		padding: 32px clamp(16px, 3vw, 36px) 48px;
		min-width: 0;
	}

	@media (max-width: 719px) {
		.admin {
			grid-template-columns: minmax(0, 1fr);
			grid-template-rows: auto 1fr;
		}
		.riel {
			position: static;
			height: auto;
			flex-direction: row;
			padding: 0 8px 0 16px;
			gap: 4px;
		}
		.rotulo {
			margin: 0 8px;
		}
		nav,
		.abajo {
			flex-direction: row;
			width: auto;
		}
		.abajo {
			margin-left: auto;
			margin-top: 0;
		}
		nav a,
		.abajo a,
		.abajo button {
			width: 56px;
			min-height: 64px;
		}
		nav a[aria-current='page'] {
			margin: 8px 0 0;
			width: 64px;
			border-radius: 16px 16px 0 0;
		}
		.abajo a span,
		.abajo button span {
			position: absolute;
			width: 1px;
			height: 1px;
			overflow: hidden;
			clip: rect(0 0 0 0);
		}
		.abajo a,
		.abajo button {
			width: 48px;
		}
		.contenido {
			padding-top: 24px;
		}
	}
</style>
