<!-- Un filtro en pastilla (handoff «Panel v2»): petróleo cuando está elegido, contorno si no. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	let {
		href,
		on = false,
		cuenta,
		cuentaColor,
		verde = false,
		children
	}: { href: string; on?: boolean; cuenta?: string | number; cuentaColor?: string; verde?: boolean; children: Snippet } = $props();
</script>

<a {href} class="chip" class:on class:verde aria-current={on ? 'true' : undefined} data-sveltekit-reset="false" data-sveltekit-replacestate>
	{@render children()}
	{#if cuenta !== undefined && cuenta !== '' && cuenta !== 0}<span class="k tl-num" style:color={cuentaColor}>{cuenta}</span>{/if}
</a>

<style>
	.chip {
		flex: none;
		display: inline-flex;
		align-items: center;
		gap: 6px;
		height: 38px;
		padding: 0 14px;
		border-radius: 999px;
		box-shadow: inset 0 0 0 1px #d8ccb4;
		color: var(--tl-tinta);
		font-size: 13px;
		font-weight: 700;
		white-space: nowrap;
		text-decoration: none;
		transition: background var(--tl-tocar) var(--tl-ease);
	}
	.chip:hover {
		background: rgba(255, 253, 249, 0.7);
	}
	.chip.on {
		background: var(--tl-petroleo);
		color: #fff;
		box-shadow: none;
	}
	.chip.verde.on {
		background: var(--tl-menta-texto);
	}
	.k {
		color: #7a4a00;
	}
	.on .k {
		color: #f0c46a;
	}
	@media (max-width: 859px) {
		.chip {
			height: 44px;
		}
	}
</style>
