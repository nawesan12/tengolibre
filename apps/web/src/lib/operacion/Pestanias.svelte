<script lang="ts">
	import { page } from '$app/state';

	/** Pestañas entre pantallas hermanas (Clientes, Etiquetas, Penalidades; Campanita, Envíos). */
	let { items, etiqueta }: { items: { nombre: string; href: string }[]; etiqueta: string } = $props();
</script>

<nav class="pestanias" aria-label={etiqueta}>
	{#each items as it (it.href)}
		<a href={it.href} aria-current={page.url.pathname === it.href ? 'page' : undefined}>{it.nombre}</a>
	{/each}
</nav>

<style>
	.pestanias {
		display: flex;
		gap: 4px;
		overflow-x: auto;
		border-bottom: 1px solid var(--borde);
	}
	a {
		padding: 8px 12px;
		text-decoration: none;
		white-space: nowrap;
		border-bottom: 2px solid transparent;
		color: var(--suave);
	}
	a[aria-current='page'] {
		color: var(--tinta);
		border-bottom-color: var(--marca);
		font-weight: 600;
	}
</style>
