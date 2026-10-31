<!-- Paso 3: los servicios típicos del rubro como talones; se sacan, se ajusta el precio y se suman otros. -->
<script lang="ts">
	import { enhance } from '$app/forms';
	import type { SubmitFunction } from '$app/forms';
	import Icono from '#lib/ui/Icono.svelte';
	import type { Borrador } from './estado.svelte.ts';
	import { duracion } from './rubros.ts';

	let { b, alEnviar, form, listo = $bindable() }: { b: Borrador; alEnviar: SubmitFunction; form: { error?: string } | null | undefined; listo: boolean } = $props();
	$effect(() => {
		listo = b.activos.length > 0 && b.activos.every((s) => s.nombre.trim().length >= 2);
	});
	const rubro = $derived(b.rubroInfo?.nombre.toLowerCase() ?? 'tu rubro');
	const DURACIONES = [15, 20, 30, 40, 45, 60, 75, 90, 120, 150, 180];
	const precioDe = (v: string) => parseInt(v.replace(/\D/g, '')) || 0;
	const json = $derived(JSON.stringify(b.servicios));
</script>

<form id="paso" method="POST" action="?/servicios" use:enhance={alEnviar} class="hoja" data-paso="servicios">
	<input type="hidden" name="servicios" value={json} />
	<h1>Te dejamos estos&nbsp;servicios</h1>
	<p class="bajada">Son los más pedidos en {rubro}. Tocá para sacar los que no hacés y ajustá el&nbsp;precio.</p>
	<div class="lista">
		{#each b.servicios as s, i (i)}
			<div class="srv" class:apagado={!s.activo} style:animation-delay="{i * 60}ms">
				<button
					type="button"
					class="tilde"
					aria-pressed={s.activo}
					aria-label={(s.activo ? 'Sacar ' : 'Volver a sumar ') + (s.nombre || 'el servicio')}
					onclick={() => (s.activo = !s.activo)}
					><span><Icono nombre={s.activo ? 'check' : 'remove'} tam={18} /></span></button
				>
				<span class="datos">
					{#if !s.nuevo}
						<b>{s.nombre}</b>
						<span>{duracion(s.duracion)}</span>
					{:else}
						<input class="nombre" bind:value={s.nombre} aria-label="Nombre del servicio" maxlength="80" placeholder="Nombre del servicio" />
						<select bind:value={s.duracion} aria-label="Duración de {s.nombre || 'el servicio'}">
							{#each DURACIONES as d (d)}<option value={d}>{duracion(d)}</option>{/each}
						</select>
					{/if}
				</span>
				<label class="precio">
					<span>$</span>
					<input
						value={s.precio.toLocaleString('es-AR')}
						oninput={(e) => (s.precio = precioDe(e.currentTarget.value))}
						onblur={(e) => (e.currentTarget.value = s.precio.toLocaleString('es-AR'))}
						inputmode="numeric"
						aria-label="Precio de {s.nombre || 'el servicio'}"
					/>
				</label>
			</div>
		{/each}
		<button type="button" class="agregar" onclick={() => b.servicios.push({ nombre: '', duracion: 30, precio: 10000, activo: true, nuevo: true })}>
			<Icono nombre="add" tam={22} />Agregar otro servicio
		</button>
	</div>
	{#if form?.error}<p class="error-paso" role="alert"><Icono nombre="error" lleno tam={16} />{form.error}</p>{/if}
</form>

<style>
	.lista {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.srv {
		display: grid;
		grid-template-columns: 52px minmax(0, 1fr) 120px;
		align-items: center;
		min-height: 64px;
		border-radius: 16px;
		background:
			radial-gradient(circle at 52px 0, #fffdf9 8px, transparent 8.5px),
			radial-gradient(circle at 52px 100%, #fffdf9 8px, transparent 8.5px),
			#f8f3ea;
		box-shadow: inset 0 0 0 1px #eee5d3;
		transition: opacity 0.2s;
		animation: tl-paso var(--tl-entrar) var(--tl-ease) both;
	}
	.apagado {
		opacity: 0.55;
	}
	.apagado .datos b {
		text-decoration: line-through;
	}
	.tilde {
		height: 100%;
		min-height: 64px;
		border: 0;
		border-right: 2px dashed #e2d6bf;
		background: transparent;
		display: grid;
		place-items: center;
		cursor: pointer;
	}
	.tilde > span {
		width: 24px;
		height: 24px;
		border-radius: 8px;
		background: #17805a;
		color: #fff;
		display: grid;
		place-items: center;
		transition: background var(--tl-tocar);
	}
	.apagado .tilde > span {
		background: #b5a98f;
	}
	.datos {
		display: flex;
		flex-direction: column;
		gap: 2px;
		padding: 8px 14px;
		min-width: 0;
	}
	.datos b {
		font-size: 15px;
		font-weight: 700;
	}
	.datos > span,
	.datos select {
		font-size: 13px;
		color: var(--tl-sub);
	}
	.nombre {
		border: 0;
		border-bottom: 1.5px dashed #d8ccb4;
		background: transparent;
		font: 700 16px var(--tl-font);
		color: var(--tl-tinta);
		padding: 2px 0;
		outline: none;
		min-width: 0;
	}
	.nombre:focus {
		border-bottom-color: #3ccc90;
	}
	.datos select {
		align-self: flex-start;
		border: 0;
		background: transparent;
		font-family: var(--tl-font);
		padding: 0;
	}
	.precio {
		display: flex;
		align-items: center;
		gap: 4px;
		margin-right: 12px;
		height: 42px;
		padding: 0 10px;
		border-radius: 12px;
		background: #f8f3ea;
		box-shadow: inset 0 0 0 1px #eee5d3;
		font-size: 15px;
		font-weight: 700;
		font-variant-numeric: tabular-nums;
	}
	.precio span {
		color: var(--tl-sub);
	}
	.precio input {
		width: 100%;
		min-width: 0;
		border: 0;
		background: transparent;
		font: 700 16px var(--tl-font);
		font-variant-numeric: tabular-nums;
		color: var(--tl-tinta);
		outline: none;
	}
	.agregar {
		display: flex;
		align-items: center;
		gap: 10px;
		min-height: 56px;
		padding: 0 16px;
		border-radius: 16px;
		border: 0;
		background: repeating-linear-gradient(-45deg, rgba(60, 204, 144, 0.22) 0 2px, transparent 2px 10px), #f3fbf7;
		background-size: 28.28px 28.28px;
		box-shadow: inset 0 0 0 1.5px rgba(23, 128, 90, 0.35);
		font: 700 15px var(--tl-font);
		color: #17805a;
		cursor: pointer;
		animation: tl-raya 3.5s linear infinite;
	}
	@media (max-width: 899px) {
		.srv {
			grid-template-columns: 48px minmax(0, 1fr) 108px;
		}
	}
</style>
