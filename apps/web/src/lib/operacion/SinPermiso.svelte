<script lang="ts">
	let { que = 'esta sección' }: { que?: string } = $props();
</script>

<section class="tarjeta pila" role="status">
	<h2>No tenés acceso a {que}</h2>
	<p class="suave">Es para quien gestiona el negocio. Si lo necesitás, pedíselo a la dueña o al&nbsp;encargado.</p>
</section>
