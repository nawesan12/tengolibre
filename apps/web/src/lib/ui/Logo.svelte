<!-- Logotipo de Tengo Libre (marca/ del handoff). -->
<script lang="ts">
	let {
		variante = 'horizontal',
		alto = 28
	}: { variante?: 'horizontal' | 'negativo' | 'apilado' | 'simbolo' | 'simbolo-blanco'; alto?: number } = $props();
	const archivo = {
		horizontal: 'logotipo_horizontal.svg',
		negativo: 'logotipo_horizontal_negativo.svg',
		apilado: 'logotipo_apilado.svg',
		simbolo: 'simbolo_principal.svg',
		'simbolo-blanco': 'simbolo_una_tinta_blanco.svg'
	};
</script>

<img src="/marca/{archivo[variante]}" alt="Tengo Libre" height={alto} style:height="{alto}px" style:width="auto" />
