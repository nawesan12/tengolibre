<!-- Paso 6: ¿seña al reservar? Cuatro opciones con un ejemplo hecho con el primer servicio. -->
<script lang="ts">
	import { enhance } from '$app/forms';
	import type { SubmitFunction } from '$app/forms';
	import Icono from '#lib/ui/Icono.svelte';
	import type { Borrador, ModoCobro } from './estado.svelte.ts';
	import { pesos } from './rubros.ts';

	let { b, alEnviar, form, listo = $bindable() }: { b: Borrador; alEnviar: SubmitFunction; form: { error?: string } | null | undefined; listo: boolean } = $props();
	$effect(() => {
		listo = b.cobro !== 'fija' || b.senaFija > 0;
	});
	const p = $derived(b.activos[0]?.precio);
	const opciones = $derived<{ k: ModoCobro; t: string; s: string; ej: string; rec?: boolean }[]>([
		{ k: 'sin_cobro', t: 'Sin seña', s: 'Reservan sin pagar nada', ej: '—' },
		{ k: 'porcentaje', t: 'Seña del 30%', s: 'La más elegida', ej: p ? `${pesos(p * 0.3)} de ${pesos(p)}` : '30% del precio', rec: true },
		{ k: 'fija', t: 'Seña fija', s: 'El mismo monto siempre', ej: pesos(b.senaFija) },
		{ k: 'total', t: 'Pago total', s: 'Pagan todo al reservar', ej: p ? pesos(p) : 'El precio' }
	]);
	const precioDe = (v: string) => parseInt(v.replace(/\D/g, '')) || 0;
</script>

<form id="paso" method="POST" action="?/cobro" use:enhance={alEnviar} class="hoja" data-paso="cobro">
	<input type="hidden" name="modo" value={b.cobro} />
	<input type="hidden" name="senaFija" value={b.senaFija} />
	<h1>¿Pedís seña al&nbsp;reservar?</h1>
	<p class="bajada">Con seña bajan mucho las ausencias. Se cobra con Mercado Pago y la plata va directo a tu&nbsp;cuenta.</p>
	<div class="modos" role="radiogroup" aria-label="Seña al reservar">
		{#each opciones as o (o.k)}
			<button type="button" class="modo" role="radio" aria-checked={b.cobro === o.k} onclick={() => (b.cobro = o.k)}>
				<span class="modo-t"><span class="punto"></span><b>{o.t}</b>{#if o.rec}<span class="rec">Recomendado</span>{/if}</span>
				<span class="modo-s">{o.s}</span>
				<b class="modo-ej">{o.ej}</b>
			</button>
		{/each}
	</div>
	{#if b.cobro === 'fija'}
		<label class="fija">
			Monto de la seña
			<span class="fija-c"
				><span>$</span><input
					value={b.senaFija.toLocaleString('es-AR')}
					oninput={(e) => (b.senaFija = precioDe(e.currentTarget.value))}
					onblur={(e) => (e.currentTarget.value = b.senaFija.toLocaleString('es-AR'))}
					inputmode="numeric"
				/></span
			>
		</label>
	{/if}
	{#if b.cobro !== 'sin_cobro'}
		<div class="nota mp">
			<img src="/logos/mercadopago.svg" alt="" width="28" height="28" />
			<span>Conectás Mercado Pago al terminar, en un minuto. Hasta entonces, las reservas entran sin&nbsp;seña.</span>
		</div>
	{/if}
	{#if form?.error}<p class="error-paso" role="alert"><Icono nombre="error" lleno tam={16} />{form.error}</p>{/if}
</form>

<style>
	.modos {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
		gap: 10px;
	}
	.modo {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 6px;
		min-height: 120px;
		padding: 16px;
		border-radius: 20px;
		border: 0;
		background: #fffdf9;
		color: var(--tl-tinta);
		box-shadow: inset 0 0 0 1.5px #e0d5bf;
		text-align: left;
		font-family: var(--tl-font);
		cursor: pointer;
		transition: transform var(--tl-tocar);
	}
	.modo:hover {
		transform: translateY(-2px);
	}
	.modo[aria-checked='true'] {
		background: #183c44;
		color: #fff;
		box-shadow: 0 16px 30px -18px rgba(24, 60, 68, 0.8);
	}
	.modo-t {
		display: flex;
		align-items: center;
		gap: 8px;
		width: 100%;
	}
	.modo-t b {
		flex: 1;
		font-size: 16px;
	}
	.punto {
		width: 22px;
		height: 22px;
		flex: none;
		border-radius: 50%;
		box-shadow: inset 0 0 0 1.5px #a89c82;
	}
	.modo[aria-checked='true'] .punto {
		box-shadow: inset 0 0 0 7px #3ccc90;
	}
	.rec {
		font-size: 12px;
		font-weight: 700;
		padding: 3px 8px;
		border-radius: 999px;
		background: #3ccc90;
		color: #102b31;
	}
	.modo-s {
		font-size: 13px;
		opacity: 0.85;
		line-height: 1.4;
	}
	.modo-ej {
		margin-top: auto;
		font-size: 16px;
		font-weight: 700;
		font-variant-numeric: tabular-nums;
	}
	.fija {
		display: flex;
		flex-direction: column;
		gap: 8px;
		font-size: 14px;
		font-weight: 700;
	}
	.fija-c {
		display: flex;
		align-items: center;
		gap: 6px;
		max-width: 240px;
		height: 56px;
		padding: 0 16px;
		border-radius: 16px;
		background: #fff;
		box-shadow: inset 0 0 0 1.5px #e0d5bf;
		font-size: 18px;
	}
	.fija-c span {
		color: var(--tl-sub);
	}
	.fija-c input {
		flex: 1;
		min-width: 0;
		border: 0;
		background: transparent;
		font: 700 18px var(--tl-font);
		font-variant-numeric: tabular-nums;
		outline: none;
	}
	.mp {
		background: #fffdf9 !important;
		box-shadow: inset 0 0 0 1px #eee5d3;
	}
</style>
