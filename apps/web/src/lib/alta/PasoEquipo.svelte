<!-- Paso 4: quién atiende, qué días y en qué horario. -->
<script lang="ts">
	import { enhance } from '$app/forms';
	import { tick, untrack } from 'svelte';
	import type { SubmitFunction } from '$app/forms';
	import Icono from '#lib/ui/Icono.svelte';
	import type { Borrador } from './estado.svelte.ts';
	import { DIAS, horasSemana, minutos } from './agenda.ts';

	let { b, alEnviar, form, listo = $bindable() }: { b: Borrador; alEnviar: SubmitFunction; form: { error?: string } | null | undefined; listo: boolean } = $props();

	const horaError = $derived(minutos(b.cierra) <= minutos(b.abre) ? 'La hora de cierre tiene que ser después de la de apertura.' : '');
	const nombres = $derived(b.personas.filter((p) => p.nombre.trim().length >= 2));
	$effect(() => {
		listo = nombres.length > 0 && nombres.length === b.personas.length && b.dias.length > 0 && !horaError;
	});
	const horas = $derived(horasSemana(b.dias.length, b.abre, b.cierra, b.personas.length));

	// Sumar a alguien: aparece un campo con forma de pastilla para escribir el nombre.
	let nuevo = $state('');
	let sumando = $state(false);
	let campo = $state<HTMLInputElement>();
	// La primera persona sin nombre (no sabemos cómo te llamás) se escribe en el lugar.
	let editando = $state(untrack(() => b.personas.findIndex((p) => !p.id && p.nombre.trim().length < 2)));
	function cerrar() {
		if ((b.personas[editando]?.nombre.trim().length ?? 0) >= 2) editando = -1;
	}
	async function abrir() {
		sumando = true;
		await tick();
		campo?.focus();
	}
	function sumar() {
		const n = nuevo.trim();
		if (n.length >= 2) b.personas.push({ nombre: n });
		nuevo = '';
		sumando = false;
	}
	function quitar(i: number) {
		const p = b.personas[i];
		if (p.id) b.quitar.push(p.id);
		b.personas.splice(i, 1);
		if (editando > i) editando--;
	}
	const toggle = (d: number) => (b.dias = b.dias.includes(d) ? b.dias.filter((x) => x !== d) : [...b.dias, d].sort((x, y) => x - y));
</script>

<form id="paso" method="POST" action="?/equipo" use:enhance={alEnviar} class="hoja" data-paso="equipo">
	<input type="hidden" name="personas" value={JSON.stringify(b.personas)} />
	<input type="hidden" name="quitar" value={JSON.stringify(b.quitar)} />
	<input type="hidden" name="dias" value={JSON.stringify(b.dias)} />
	<h1>¿Quién atiende y&nbsp;cuándo?</h1>
	<div role="group" aria-label="Quién atiende" class="personas">
		{#each b.personas as p, i (i)}
			{#if i !== editando}
				<span class="persona">
					{p.nombre}
					<button type="button" onclick={() => quitar(i)} aria-label="Quitar a {p.nombre}"><Icono nombre="close" tam={18} /></button>
				</span>
			{:else}
				<input
					class="persona-campo"
					bind:value={p.nombre}
					aria-label="Tu nombre"
					placeholder="Tu nombre"
					maxlength="60"
					onblur={cerrar}
					onkeydown={(e) => {
						if (e.key === 'Enter') {
							e.preventDefault();
							cerrar();
						}
					}}
				/>
			{/if}
		{/each}
		{#if sumando}
			<input
				class="persona-campo"
				bind:this={campo}
				bind:value={nuevo}
				aria-label="Nombre de quien atiende"
				placeholder="Nombre"
				maxlength="60"
				onkeydown={(e) => {
					if (e.key === 'Enter') {
						e.preventDefault();
						sumar();
					}
				}}
				onblur={sumar}
			/>
		{:else}
			<button type="button" class="sumar" onclick={abrir}><Icono nombre="person_add" lleno tam={20} />Sumar a alguien</button>
		{/if}
	</div>

	<span class="rot">Días que abrís</span>
	<div class="dias">
		{#each DIAS as d (d.n)}
			<button type="button" class="dia" aria-pressed={b.dias.includes(d.n)} aria-label="{d.nombre}, {b.dias.includes(d.n) ? 'abierto' : 'cerrado'}" onclick={() => toggle(d.n)}>{d.letra}</button>
		{/each}
	</div>

	<span class="rot">Horario</span>
	<div class="horas">
		<label>Abrís<input type="time" step="900" name="abre" bind:value={b.abre} required /></label>
		<span class="flecha" aria-hidden="true"><Icono nombre="arrow_forward" tam={22} /></span>
		<label>Cerrás<input type="time" step="900" name="cierra" bind:value={b.cierra} required /></label>
	</div>
	{#if horaError}<span class="error-paso" role="alert"><Icono nombre="error" lleno tam={16} />{horaError}</span>{/if}

	<div class="resumen">
		<div class="res-num tl-petroleo"><b>{horas}</b><span>horas libres por semana</span></div>
		<div class="res-dias">
			<div class="mini-dias">
				{#each DIAS as d (d.n)}
					<span><span class="barra" class:abre={b.dias.includes(d.n)}></span><span class="l">{d.letra}</span></span>
				{/each}
			</div>
			<span class="res-nota">Lo afinás por persona en Equipo y&nbsp;horarios.</span>
		</div>
	</div>
	{#if form?.error}<p class="error-paso" role="alert"><Icono nombre="error" lleno tam={16} />{form.error}</p>{/if}
</form>

<style>
	.personas {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}
	.persona {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		height: 48px;
		padding: 0 8px 0 16px;
		border-radius: 999px;
		background: #183c44;
		color: #fff;
		font-size: 15px;
		font-weight: 700;
		animation: tl-pop 0.3s both;
	}
	.persona button {
		width: 32px;
		height: 32px;
		border-radius: 50%;
		border: 0;
		background: rgba(255, 255, 255, 0.12);
		color: #fff;
		display: grid;
		place-items: center;
		cursor: pointer;
	}
	.persona-campo {
		height: 48px;
		width: 180px;
		padding: 0 16px;
		border-radius: 999px;
		border: 0;
		background: #fff;
		box-shadow: inset 0 0 0 2px #3ccc90;
		font: 700 16px var(--tl-font);
		color: var(--tl-tinta);
		outline: none;
	}
	.sumar {
		white-space: nowrap;
		flex: none;
		display: inline-flex;
		align-items: center;
		gap: 6px;
		height: 48px;
		padding: 0 16px;
		border-radius: 999px;
		border: 0;
		box-shadow: inset 0 0 0 1.5px #d8ccb4;
		background: transparent;
		font: 700 15px var(--tl-font);
		color: var(--tl-tinta);
		cursor: pointer;
	}
	.rot {
		margin-top: 8px;
	}
	.dias {
		display: grid;
		grid-template-columns: repeat(7, 1fr);
		gap: 6px;
	}
	.dia {
		height: 56px;
		border-radius: 14px;
		border: 0;
		background: #fffdf9;
		color: var(--tl-tinta);
		box-shadow: inset 0 0 0 1.5px #e0d5bf;
		font: 700 15px var(--tl-font);
		cursor: pointer;
		transition: background var(--tl-tocar);
	}
	.dia[aria-pressed='true'] {
		background: #183c44;
		color: #fff;
		box-shadow: none;
	}
	.resumen {
		display: grid;
		grid-template-columns: 128px minmax(0, 1fr);
		border-radius: 20px;
		overflow: hidden;
		background: #f8f3ea;
		box-shadow: inset 0 0 0 1px #eee5d3;
	}
	.res-num {
		display: flex;
		flex-direction: column;
		justify-content: center;
		gap: 4px;
		padding: 16px;
		border-right: 2px dashed rgba(255, 255, 255, 0.28);
	}
	.res-num b {
		font-size: 44px;
		font-weight: 700;
		letter-spacing: -0.05em;
		line-height: 0.9;
		color: #3ccc90;
		font-variant-numeric: tabular-nums;
	}
	.res-num span {
		font-size: 12px;
		line-height: 1.3;
		color: rgba(255, 255, 255, 0.82);
	}
	.res-dias {
		display: flex;
		flex-direction: column;
		justify-content: center;
		gap: 10px;
		padding: 14px 16px;
		min-width: 0;
	}
	.mini-dias {
		display: grid;
		grid-template-columns: repeat(7, minmax(0, 1fr));
		gap: 4px;
	}
	.mini-dias > span {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 4px;
	}
	.barra {
		width: 100%;
		height: 26px;
		border-radius: 7px;
		background: #ece5d6;
		transition: background 0.2s;
	}
	.barra.abre {
		background: repeating-linear-gradient(-45deg, #3ccc90 0 2px, #dff4e9 2px 6px);
	}
	.l {
		font-size: 11px;
		font-weight: 700;
		color: var(--tl-sub);
	}
	.res-nota {
		font-size: 13px;
		line-height: 1.4;
		color: var(--tl-sub-2);
		text-wrap: pretty;
	}
	@media (max-width: 899px) {
		.dia {
			height: 48px;
		}
		.resumen {
			grid-template-columns: 104px minmax(0, 1fr);
		}
	}
</style>
