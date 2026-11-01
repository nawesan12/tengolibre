<script lang="ts">
	import { enhance } from '$app/forms';
	import { onMount } from 'svelte';
	import { dia, hora, pesos } from '#lib/formato.ts';
	import Icono from '#lib/ui/Icono.svelte';
	import { diaCorto } from '#lib/publico/calendario.ts';
	import { completar, tema, variables } from '#lib/publico/estilo.ts';

	let { data, form } = $props();
	const o = $derived(data.oferta);
	const n = $derived(o.negocio);
	const z = $derived(n.zonaHoraria);
	const tm = $derived(tema(completar(data.web?.estilo), data.web?.colorPrimario, data.web?.colorAcento));
	const vars = $derived(variables(tm));
	const f = $derived(dia(o.inicio, z));
	const esHoy = $derived(f === dia(new Date(), z));
	// La cuenta regresiva: se calcula al cargar y, con JavaScript, minuto a minuto.
	let ahora = $state(Date.now());
	onMount(() => {
		const id = setInterval(() => (ahora = Date.now()), 15_000);
		return () => clearInterval(id);
	});
	const quedan = $derived(o.venceEn ? Math.max(0, Math.ceil((Date.parse(o.venceEn) - ahora) / 60_000)) : null);
	const vencida = $derived(o.estado !== 'ofrecida' || quedan === 0);
	const pasada = $derived((!!form && 'pasada' in form) || o.estado === 'paso');
	const tomadaPorOtro = $derived(!!form && 'codigo' in form && (form.codigo === 'oferta_cerrada' || form.codigo === 'horario_ocupado' || form.codigo === 'oferta_vencida'));
</script>

<svelte:head>
	<title>Se liberó un lugar en {n.nombre} · Tengo Libre</title>
	<meta name="robots" content="noindex, nofollow" />
	<meta name="referrer" content="no-referrer" />
	<meta name="theme-color" content={tm.bg} />
</svelte:head>

<div class="fondo" style={vars}>
	<main class="oferta">
		{#if pasada}
			<span class="kicker tl-num">LISTO</span>
			<h1>Lo pasamos a la&nbsp;siguiente</h1>
			<p class="texto">Seguís en la lista de espera: si se libera otro lugar, te&nbsp;avisamos.</p>
			<a class="cta" href={`/${n.slug}`}>Ver {n.nombre}</a>
		{:else if vencida || tomadaPorOtro}
			<span class="kicker gris tl-num">{diaCorto(f)} · {hora(o.inicio, z)}</span>
			<h1>Este lugar ya no está&nbsp;disponible</h1>
			<p class="texto">Pasó el tiempo para tomarlo o lo tomó otra persona. Mirá si hay otro horario que te&nbsp;sirva.</p>
			<a class="cta" href={`/${n.slug}/reservar`}>Ver horarios libres</a>
		{:else}
			<span class="kicker tl-num">SE LIBERÓ LO QUE ESPERABAS</span>
			<h1 class="visualmente-oculto">Se liberó un lugar en {n.nombre}</h1>
			<section class="talon" aria-label="El lugar que se liberó">
				<div class="arriba">
					<b>{n.nombre}</b>
					<span class="tl-num">{esHoy ? 'HOY · ' : ''}{diaCorto(f)}</span>
					<b class="hora tl-num">{hora(o.inicio, z)}</b>
				</div>
				<div class="troquel" aria-hidden="true"><span class="izq"></span><span class="der"></span><span class="linea"></span></div>
				<div class="abajo">
					<span>{o.servicio}{o.profesional ? ` · ${o.profesional.split(' ')[0]}` : ''}</span>
					<b class="tl-num">{pesos(o.precio)}</b>
				</div>
			</section>
			{#if form && 'error' in form && form.error}<p class="aviso rojo" role="alert"><Icono nombre="error" tam={20} lleno /><span>{form.error}</span></p>{/if}
			{#if quedan !== null}
				<p class="aviso ambar">
					<Icono nombre="timer" tam={20} lleno />
					<span>Es tuyo si lo tomás en los próximos <b class="tl-num">{quedan} min</b>. Después pasa a la&nbsp;siguiente.</span>
				</p>
			{/if}
			<div class="acciones">
				<form method="POST" action="?/tomar" use:enhance>
					<button class="cta">Tomar el lugar</button>
				</form>
				<form method="POST" action="?/pasar" use:enhance>
					<button class="cta-2">No me sirve, que pase</button>
				</form>
			</div>
		{/if}
	</main>
</div>

<style>
	.fondo {
		min-height: 100vh;
		background: var(--w-bg);
		color: var(--w-ink);
	}
	.oferta {
		max-width: 460px;
		min-height: 100vh;
		margin: 0 auto;
		padding: 24px 18px 32px;
		display: flex;
		flex-direction: column;
		gap: 14px;
	}
	.kicker {
		font: 700 12px var(--tl-font);
		letter-spacing: 0.04em;
		color: var(--w-ok);
	}
	.kicker.gris {
		color: var(--w-sub);
	}
	h1 {
		margin: 0;
		font-size: 28px;
		line-height: 1.02;
		font-weight: var(--w-tw);
		letter-spacing: -0.045em;
	}
	.texto {
		margin: 0;
		font-size: 15px;
		line-height: 1.5;
		color: var(--w-sub2);
	}
	.talon {
		border-radius: 22px;
		overflow: hidden;
		background: var(--w-papel);
		box-shadow: 0 24px 40px -24px rgba(15, 35, 40, 0.5);
		animation: tl-gin var(--tl-entrar) var(--tl-ease) both;
	}
	.arriba {
		padding: 16px 18px 20px;
		background-color: var(--w-pri);
		background-image: var(--tl-grano);
		color: var(--w-on);
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.arriba > b:first-child {
		font-size: 15px;
	}
	.arriba span {
		font: 400 12px var(--tl-font);
		opacity: 0.85;
	}
	.hora {
		font: 700 60px var(--tl-font);
		letter-spacing: -0.07em;
		line-height: 0.95;
	}
	.troquel {
		height: 18px;
		position: relative;
		background: linear-gradient(var(--w-pri) 0 50%, var(--w-papel) 50%);
	}
	.troquel .izq,
	.troquel .der {
		position: absolute;
		top: 0;
		width: 18px;
		height: 18px;
		border-radius: 50%;
		background: var(--w-bg);
	}
	.izq {
		left: -9px;
	}
	.der {
		right: -9px;
	}
	.linea {
		position: absolute;
		left: 16px;
		right: 16px;
		top: 8px;
		border-top: 2px dashed var(--w-line2);
	}
	.abajo {
		padding: 4px 18px 16px;
		display: flex;
		justify-content: space-between;
		gap: 10px;
		font-size: 14px;
	}
	.aviso {
		display: flex;
		align-items: center;
		gap: 10px;
		margin: 0;
		padding: 12px 14px;
		border-radius: 14px;
		font-size: 14px;
		line-height: 1.4;
	}
	.ambar {
		background: #fbe9c2;
		color: #5c3800;
	}
	.rojo {
		background: #fbe0dc;
		color: #5b1a13;
	}
	.acciones {
		margin-top: auto;
		display: flex;
		flex-direction: column;
	}
	.cta {
		display: grid;
		place-items: center;
		width: 100%;
		min-height: 56px;
		border: 0;
		border-radius: var(--w-br);
		background: var(--w-pri);
		color: var(--w-on);
		font: 700 17px var(--tl-font);
		text-decoration: none;
		cursor: pointer;
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.cta:active {
		transform: scale(0.97);
	}
	.cta-2 {
		display: grid;
		place-items: center;
		width: 100%;
		min-height: 48px;
		border: 0;
		background: transparent;
		font: 700 15px var(--tl-font);
		color: var(--w-sub);
		cursor: pointer;
	}
</style>
