<script lang="ts">
	import { page } from '$app/state';

	/** Pestañas entre pantallas hermanas (membresías y cupones): son links, no estado. */
	let { pestanias, etiqueta }: { pestanias: { nombre: string; href: string }[]; etiqueta: string } = $props();
</script>

<nav class="pestanias" aria-label={etiqueta}>
	{#each pestanias as p (p.href)}
		<a class="boton" href={p.href} aria-current={page.url.pathname === p.href ? 'page' : undefined}>{p.nombre}</a>
	{/each}
</nav>

<style>
	.pestanias {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}
	a[aria-current='page'] {
		background: var(--marca-suave);
		border-color: var(--marca);
	}
</style>
