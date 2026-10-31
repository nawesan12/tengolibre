<script lang="ts">
	import Alta from '#lib/alta/Alta.svelte';
	import Onboarding from '#lib/alta/Onboarding.svelte';
	import { INFO } from '#lib/alta/pasos.ts';
	let { data, form } = $props();
</script>

<svelte:head>
	<title>{data.modo === 'alta' ? 'Empezar' : INFO[data.paso].nombre} · Tengo Libre</title>
	<meta name="robots" content="noindex" />
</svelte:head>

{#if data.modo === 'alta'}
	<Alta borrador={data.borrador} fapi={data.fapi} origen={data.origen} />
{:else}
	<Onboarding {data} form={form as { error?: string } | null} />
{/if}
