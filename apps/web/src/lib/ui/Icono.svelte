<!-- Ícono de Material Symbols Rounded (el set del handoff). Decorativo salvo que lleve etiqueta. -->
<script lang="ts">
	let {
		nombre,
		lleno = false,
		tam = 22,
		etiqueta
	}: { nombre: string; lleno?: boolean; tam?: number; etiqueta?: string } = $props();
</script>

<span
	class="icono"
	class:lleno
	style:font-size="{tam}px"
	aria-hidden={etiqueta ? undefined : 'true'}
	role={etiqueta ? 'img' : undefined}
	aria-label={etiqueta}>{nombre}</span
>
