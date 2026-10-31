<!-- Paso 5: «¿Cómo es tu día?»: descanso, tiempo entre turnos, anticipación y feriados. -->
<script lang="ts">
	import { enhance } from '$app/forms';
	import type { SubmitFunction } from '$app/forms';
	import Icono from '#lib/ui/Icono.svelte';
	import { ANTICIPACIONES, ENTRES, type Borrador } from './estado.svelte.ts';
	import { minutos, segmentosDia, type Feriado } from './agenda.ts';
	import type { BloqueoAlta } from './tipos.ts';

	let {
		b,
		feriados,
		bloqueados,
		alEnviar,
		form,
		listo = $bindable()
	}: {
		b: Borrador;
		feriados: Feriado[];
		bloqueados: BloqueoAlta[];
		alEnviar: SubmitFunction;
		form: { error?: string } | null | undefined;
		listo: boolean;
	} = $props();

	const descanso = $derived(b.descOn ? { desde: b.descDesde, hasta: b.descHasta } : null);
	const descError = $derived(
		b.descOn && !(minutos(b.descHasta) > minutos(b.descDesde) && minutos(b.descDesde) > minutos(b.abre) && minutos(b.descHasta) < minutos(b.cierra))
			? `El descanso tiene que caer entre las ${b.abre} y las ${b.cierra}.`
			: ''
	);
	const ferError = $derived(b.feriado === 'especial' && minutos(b.ferCierra) <= minutos(b.ferAbre) ? 'La hora de cierre tiene que ser después de la de apertura.' : '');
	$effect(() => {
		listo = !descError && !ferError;
	});

	const primero = $derived(b.activos[0]);
	const dia = $derived(segmentosDia(b.abre, b.cierra, primero?.duracion ?? 60, b.entre, descanso));
	const deServicio = $derived(primero ? primero.nombre.toLowerCase() : 'turnos de 1 h');

	const corta = (h: string) => h.replace(/^0/, '').replace(/:00$/, '');
	const yaBloqueado = $derived(new Set(bloqueados.map((x) => x.fecha.slice(0, 10))));
	const abreFeriado = (f: string) => (b.feriado !== 'cierro' ? !b.ferExcepciones.includes(f) : b.ferExcepciones.includes(f));
	const cerrados = $derived(feriados.filter((f) => !abreFeriado(f.fecha)).map((f) => f.fecha));
	const especiales = $derived(b.feriado === 'especial' ? feriados.filter((f) => abreFeriado(f.fecha)).map((f) => f.fecha) : []);
	const toggleFer = (f: string) =>
		(b.ferExcepciones = b.ferExcepciones.includes(f) ? b.ferExcepciones.filter((x) => x !== f) : [...b.ferExcepciones, f]);
</script>

<form id="paso" method="POST" action="?/agenda" use:enhance={alEnviar} class="hoja agenda" data-paso="agenda">
	<input type="hidden" name="descanso" value={b.descOn ? 'on' : ''} />
	<input type="hidden" name="entre" value={b.entre} />
	<input type="hidden" name="antic" value={b.antic} />
	<input type="hidden" name="feriado" value={b.feriado} />
	<input type="hidden" name="cerrados" value={JSON.stringify(cerrados)} />
	<input type="hidden" name="especiales" value={JSON.stringify(especiales)} />
	<h1>¿Cómo es tu&nbsp;día?</h1>

	<div class="dia">
		<span class="dia-t"><span>{b.abre}</span><span>Un día con {deServicio}</span><span>{b.cierra}</span></span>
		<div class="tira" aria-hidden="true">
			{#each dia.seg as g, i (i)}
				<span class="seg {g.tipo}" style:flex={g.f}>{g.tipo === 'descanso' ? 'Descanso' : ''}</span>
			{/each}
		</div>
		<span class="leyenda">
			<b>{dia.turnos} turnos por día</b>
			<span><i class="turno"></i>Turno</span>
			<span><i class="entre"></i>Entre turnos</span>
			<span><i class="descanso"></i>Descanso</span>
		</span>
	</div>

	<span class="titulo-campo"><b>Descanso</b><span>En ese horario nadie puede reservar.</span></span>
	<label class="interruptor">
		<button type="button" role="switch" aria-checked={b.descOn} onclick={() => (b.descOn = !b.descOn)} aria-label="Tengo descanso"><span></span></button>
		<span>Me tomo un descanso todos los días</span>
	</label>
	{#if b.descOn}
		<div class="horas aparece-paso">
			<label>Desde<input type="time" step="900" name="descDesde" bind:value={b.descDesde} /></label>
			<span class="flecha" aria-hidden="true"><Icono nombre="arrow_forward" tam={22} /></span>
			<label>Hasta<input type="time" step="900" name="descHasta" bind:value={b.descHasta} /></label>
		</div>
		{#if descError}<span class="error-paso" role="alert"><Icono nombre="error" lleno tam={16} />{descError}</span>{/if}
	{/if}

	<span class="titulo-campo"><b>Tiempo entre turnos</b><span>Para limpiar, preparar o atrasarte un poco sin que se pise el&nbsp;siguiente.</span></span>
	<div role="group" aria-label="Tiempo entre turnos" class="opciones">
		{#each ENTRES as v (v)}
			<button type="button" class="opcion" aria-pressed={b.entre === v} onclick={() => (b.entre = v)}>{v ? `${v} min` : 'Sin pausa'}</button>
		{/each}
	</div>

	<span class="titulo-campo"><b>Anticipación mínima</b><span>Cuánto antes tienen que reservar. Evita que te caiga alguien en 10&nbsp;minutos.</span></span>
	<div role="group" aria-label="Anticipación mínima" class="opciones">
		{#each ANTICIPACIONES as [m, t] (m)}
			<button type="button" class="opcion" aria-pressed={b.antic === m} onclick={() => (b.antic = m)}>{t} antes</button>
		{/each}
	</div>

	<span class="titulo-campo"><b>Feriados</b><span>Los próximos feriados nacionales ya están&nbsp;cargados.</span></span>
	<div role="group" aria-label="Feriados" class="opciones">
		<button type="button" class="opcion" aria-pressed={b.feriado === 'cierro'} onclick={() => ((b.feriado = 'cierro'), (b.ferExcepciones = []))}>Cierro</button>
		<button type="button" class="opcion" aria-pressed={b.feriado === 'abro'} onclick={() => ((b.feriado = 'abro'), (b.ferExcepciones = []))}>Abro igual</button>
		<button type="button" class="opcion" aria-pressed={b.feriado === 'especial'} onclick={() => ((b.feriado = 'especial'), (b.ferExcepciones = []))}
			>Abro de {corta(b.ferAbre)} a {corta(b.ferCierra)}</button
		>
	</div>
	{#if b.feriado === 'especial'}
		<div class="horas aparece-paso">
			<label>Los feriados abrís<input type="time" step="900" name="ferAbre" bind:value={b.ferAbre} /></label>
			<span class="flecha" aria-hidden="true"><Icono nombre="arrow_forward" tam={22} /></span>
			<label>y cerrás<input type="time" step="900" name="ferCierra" bind:value={b.ferCierra} /></label>
		</div>
		{#if ferError}<span class="error-paso" role="alert"><Icono nombre="error" lleno tam={16} />{ferError}</span>{/if}
	{/if}
	<div class="feriados">
		{#each feriados as f (f.fecha)}
			{@const abre = abreFeriado(f.fecha)}
			<div class="feriado" class:cerrado={!abre}>
				<b>{f.rotulo}</b>
				<span>{f.nombre}</span>
				{#if yaBloqueado.has(f.fecha)}
					<span class="estado cerrado"><Icono nombre="block" lleno tam={16} />Ya bloqueado</span>
				{:else}
					<button type="button" class="estado" class:cerrado={!abre} onclick={() => toggleFer(f.fecha)} aria-label="{f.nombre}: {abre ? 'abierto' : 'cerrado'}. Tocá para cambiar">
						<Icono nombre={abre ? 'storefront' : 'block'} lleno tam={16} />{abre ? (b.feriado === 'especial' ? `${corta(b.ferAbre)} a ${corta(b.ferCierra)} h` : 'Abierto') : 'Cerrado'}
					</button>
				{/if}
			</div>
		{/each}
	</div>
	<div class="nota">
		<Icono nombre="beach_access" lleno tam={22} />
		<span>¿Vacaciones o un día libre? Los cargás después en Equipo y horarios, y te avisamos qué turnos&nbsp;toca.</span>
	</div>
	{#if form?.error}<p class="error-paso" role="alert"><Icono nombre="error" lleno tam={16} />{form.error}</p>{/if}
</form>

<style>
	.agenda {
		gap: 16px !important;
	}
	.dia {
		display: flex;
		flex-direction: column;
		gap: 8px;
		padding: 16px;
		border-radius: 20px;
		background: #f8f3ea;
	}
	.dia-t {
		display: flex;
		justify-content: space-between;
		gap: 8px;
		font-size: 12px;
		font-weight: 700;
		color: var(--tl-sub);
		font-variant-numeric: tabular-nums;
	}
	.tira {
		display: flex;
		height: 48px;
		border-radius: 12px;
		overflow: hidden;
		background: #fff;
		box-shadow: inset 0 0 0 1px #eee5d3;
	}
	.seg {
		min-width: 0;
		display: grid;
		place-items: center;
		font-size: 11px;
		font-weight: 700;
		overflow: hidden;
		white-space: nowrap;
		transition: flex 0.35s var(--tl-ease);
	}
	.seg.turno {
		background: #183c44;
	}
	.seg.entre {
		background: #e2d6bf;
	}
	.seg.descanso {
		background: repeating-linear-gradient(-45deg, #f0c46a 0 2px, #fbe9c2 2px 6px);
		color: #5c3800;
	}
	.leyenda {
		display: flex;
		flex-wrap: wrap;
		gap: 6px 16px;
		font-size: 13px;
		color: var(--tl-sub);
	}
	.leyenda b {
		font-size: 15px;
		color: #17805a;
		font-variant-numeric: tabular-nums;
	}
	.leyenda span {
		display: inline-flex;
		align-items: center;
		gap: 6px;
	}
	.leyenda i {
		width: 12px;
		height: 12px;
		border-radius: 3px;
	}
	.leyenda .turno {
		background: #183c44;
	}
	.leyenda .entre {
		background: #e2d6bf;
	}
	.leyenda .descanso {
		background: repeating-linear-gradient(-45deg, #f0c46a 0 2px, #fbe9c2 2px 5px);
	}
	.interruptor {
		display: flex;
		align-items: center;
		gap: 12px;
		min-height: 48px;
		cursor: pointer;
		font-size: 15px;
	}
	.interruptor button {
		width: 52px;
		height: 30px;
		flex: none;
		border-radius: 999px;
		border: 0;
		background: #b5a98f;
		position: relative;
		cursor: pointer;
		transition: background var(--tl-tocar);
	}
	.interruptor button[aria-checked='true'] {
		background: #17805a;
	}
	.interruptor button span {
		position: absolute;
		top: 3px;
		left: 3px;
		width: 24px;
		height: 24px;
		border-radius: 50%;
		background: #fff;
		box-shadow: 0 1px 3px rgba(15, 35, 40, 0.3);
		transition: left 0.2s cubic-bezier(0.2, 0.9, 0.25, 1.2);
	}
	.interruptor button[aria-checked='true'] span {
		left: 25px;
	}
	.aparece-paso {
		animation: tl-paso 0.35s both;
	}
	.feriados {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	.feriado {
		display: grid;
		grid-template-columns: 96px minmax(0, 1fr) auto;
		min-height: 54px;
		border-radius: 14px;
		background:
			radial-gradient(circle at 96px 0, #fffdf9 7px, transparent 7.5px),
			radial-gradient(circle at 96px 100%, #fffdf9 7px, transparent 7.5px),
			#fffdf9;
		box-shadow: inset 0 0 0 1px #eee5d3;
		transition: background 0.2s;
	}
	.feriado.cerrado {
		background:
			radial-gradient(circle at 96px 0, #fffdf9 7px, transparent 7.5px),
			radial-gradient(circle at 96px 100%, #fffdf9 7px, transparent 7.5px),
			#f8f3ea;
	}
	.feriado > b {
		display: flex;
		align-items: center;
		padding: 0 10px;
		border-right: 2px dashed #e2d6bf;
		font-size: 13px;
		font-weight: 700;
		font-variant-numeric: tabular-nums;
		white-space: nowrap;
	}
	.feriado > span:not(.estado) {
		display: flex;
		align-items: center;
		padding: 0 12px;
		font-size: 13px;
		min-width: 0;
	}
	.estado {
		align-self: center;
		margin: 8px 10px;
		height: 38px;
		padding: 0 12px;
		border-radius: 999px;
		border: 0;
		background: #e3f6ec;
		color: #17805a;
		font: 700 13px var(--tl-font);
		display: inline-flex;
		align-items: center;
		gap: 6px;
		white-space: nowrap;
		cursor: pointer;
	}
	.estado.cerrado {
		background: #ece5d6;
		color: var(--tl-sub);
	}
	@media (max-width: 899px) {
		.estado {
			min-height: 44px;
			margin: 5px 8px;
		}
	}
</style>
