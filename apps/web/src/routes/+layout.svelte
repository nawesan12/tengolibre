<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	let { children } = $props();
	// Marca que la página ya está hidratada: las e2e esperan esto antes de escribir en un campo (si
	// escriben antes, Svelte pisa lo escrito con el valor inicial).
	onMount(() => {
		document.documentElement.dataset.hidratado = '1';
	});
</script>

<svelte:head>
	<link rel="preconnect" href="https://fonts.googleapis.com" />
	<link href="https://fonts.googleapis.com/css2?family=Geist:wght@400..700&display=swap" rel="stylesheet" />
</svelte:head>

{@render children()}
