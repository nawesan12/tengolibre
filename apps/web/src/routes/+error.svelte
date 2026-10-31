<script lang="ts">
	import { page } from '$app/state';
</script>

<svelte:head><title>{page.status} · Tengo Libre</title></svelte:head>

<main class="contenedor pila">
	<h1>{page.status === 404 ? 'No encontramos esta página' : 'Algo salió mal'}</h1>
	<p class="suave">{page.error?.message}</p>
	<p><a class="boton" href="/">Volver al inicio</a></p>
</main>
