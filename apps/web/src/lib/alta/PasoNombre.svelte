<!-- Paso 2: el nombre y la dirección web, que se arma sola y se valida al instante. Acá se crea el negocio. -->
<script lang="ts">
	import { enhance } from '$app/forms';
	import { page } from '$app/state';
	import type { SubmitFunction } from '$app/forms';
	import Icono from '#lib/ui/Icono.svelte';
	import type { Borrador } from './estado.svelte.ts';

	let {
		b,
		creado,
		slug,
		alEnviar,
		form,
		listo = $bindable()
	}: { b: Borrador; creado: boolean; slug?: string; alEnviar: SubmitFunction; form: { error?: string } | null | undefined; listo: boolean } = $props();

	const host = $derived(page.url.host);
	$effect(() => {
		if (!creado) b.direccion.poner(b.slug);
	});
	const campo = $derived(b.direccion.campo);
	$effect(() => {
		listo = !!b.nombre.trim() && (creado || campo.ok);
	});
	const sugerencias = $derived(campo.estado === 'tomada' ? (b.direccion.resultado?.sugerencias ?? []) : []);
</script>

<form id="paso" method="POST" action="?/nombre" use:enhance={alEnviar} class="hoja" data-paso="nombre">
	<h1>¿Cómo se&nbsp;llama?</h1>
	{#if creado}<input type="hidden" name="editando" value="1" />{/if}
	<input type="hidden" name="rubro" value={b.rubro} />
	<input type="hidden" name="slug" value={b.slug} />
	<label class="campo-nombre">
		Nombre del negocio
		<input
			name="nombre"
			bind:value={b.nombre}
			oninput={() => (b.slugManual = null)}
			autocomplete="organization"
			maxlength="80"
			required
			placeholder="Lumi Estética"
		/>
	</label>
	<div class="dir">
		<span id="dir-t">Tu dirección web</span>
		{#if creado}
			<span class="dir-caja ok" aria-labelledby="dir-t">
				<span class="pre">{host}/</span><b>{slug}</b><Icono nombre="check_circle" lleno tam={22} />
			</span>
			<span class="msg ok" role="status">Es tuya. La dirección ya no cambia.</span>
		{:else}
			<span class="dir-caja" style:--borde={campo.borde} aria-labelledby="dir-t">
				<span class="pre">{host}/</span><b>{b.slug || '…'}</b><span style:color={campo.borde}><Icono nombre={campo.icono} lleno tam={22} /></span>
			</span>
			<span class="msg" style:color={campo.color} role="status">{campo.mensaje}</span>
		{/if}
	</div>
	{#if sugerencias.length}
		<div class="alts">
			{#each sugerencias as s (s)}
				<button type="button" class="alt" onclick={() => (b.slugManual = s)}>{s}</button>
			{/each}
		</div>
	{/if}
	{#if form?.error}<p class="error-paso" role="alert"><Icono nombre="error" lleno tam={16} />{form.error}</p>{/if}
</form>

<style>
	.campo-nombre {
		display: flex;
		flex-direction: column;
		gap: 8px;
		font-size: 14px;
		font-weight: 700;
	}
	.campo-nombre input {
		height: 56px;
		padding: 0 16px;
		border-radius: 16px;
		border: 0;
		background: #fff;
		box-shadow: inset 0 0 0 2px #3ccc90;
		font: 400 18px var(--tl-font);
		color: var(--tl-tinta);
		outline: none;
	}
	.dir {
		display: flex;
		flex-direction: column;
		gap: 8px;
		font-size: 14px;
		font-weight: 700;
	}
	.dir-caja {
		--borde: #17805a;
		display: flex;
		align-items: center;
		height: 56px;
		padding: 0 16px;
		border-radius: 16px;
		background: #fff;
		box-shadow: inset 0 0 0 2px var(--borde);
		font-size: 16px;
		font-weight: 400;
		overflow: hidden;
	}
	.dir-caja.ok {
		color: inherit;
	}
	.dir-caja.ok :global(.icono) {
		color: #17805a;
	}
	.pre {
		color: var(--tl-sub);
		white-space: nowrap;
	}
	.dir-caja b {
		font-weight: 700;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		margin-right: auto;
	}
	.msg {
		font-size: 13px;
		font-weight: 700;
	}
	.msg.ok {
		color: #17805a;
	}
	.alts {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}
	.alt {
		height: 44px;
		padding: 0 14px;
		border-radius: 999px;
		border: 0;
		background: repeating-linear-gradient(-45deg, rgba(60, 204, 144, 0.22) 0 2px, transparent 2px 9px), #f3fbf7;
		box-shadow: inset 0 0 0 1.5px rgba(23, 128, 90, 0.35);
		font: 700 14px var(--tl-font);
		color: #17805a;
		cursor: pointer;
	}
</style>
