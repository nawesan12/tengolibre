<!-- Hueco = tiempo libre (firma 2): rayado que corre despacio, siempre con texto. `color` cambia la
	raya (en la web pública, el color del negocio). -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	let { color, children }: { color?: string; children: Snippet } = $props();
</script>

<div
	class="tl-hueco hueco"
	style:background={color
		? `repeating-linear-gradient(-45deg, color-mix(in srgb, ${color} 45%, transparent) 0 2px, rgba(255,253,249,.9) 2px 10px)`
		: undefined}
	style:background-size={color ? '34px 34px' : undefined}
>
	{@render children()}
</div>

<style>
	.hueco {
		border-radius: 18px;
		padding: 14px 18px;
		font-weight: 700;
	}
</style>
