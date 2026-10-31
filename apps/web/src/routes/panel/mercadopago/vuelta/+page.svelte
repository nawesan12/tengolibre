<script lang="ts">
	let { data } = $props();
</script>

<svelte:head><title>Mercado Pago · Panel</title></svelte:head>

<div class="pila">
	<h1>No pudimos conectar Mercado&nbsp;Pago</h1>
	<p class="aviso error" role="alert">{data.error}</p>
	<p><a class="boton primario" href="/panel/pagos">Volver a Pagos</a></p>
</div>
