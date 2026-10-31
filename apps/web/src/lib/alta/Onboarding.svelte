<!--
	El onboarding (Onboarding v2 del handoff): barra de progreso con el anillo, encabezado del paso
	(número en contorno, pastilla y consejo), la hoja del paso, «Tu página, en vivo» a la derecha en
	escritorio y la barra de abajo con «Lo completo después» y la acción principal.
	Cada paso es un formulario `#paso` que guarda en la API y redirige al siguiente.
-->
<script lang="ts">
	import { onDestroy } from 'svelte';
	import type { SubmitFunction } from '$app/forms';
	import Icono from '#lib/ui/Icono.svelte';
	import { Borrador } from './estado.svelte.ts';
	import { INFO, PASOS, anterior, siguiente, urlPaso, type Paso } from './pasos.ts';
	import { horasSemana, type Feriado } from './agenda.ts';
	import { pesos } from './rubros.ts';
	import type { Cargado } from './tipos.ts';
	import Previa from './Previa.svelte';
	import PasoRubro from './PasoRubro.svelte';
	import PasoNombre from './PasoNombre.svelte';
	import PasoServicios from './PasoServicios.svelte';
	import PasoEquipo from './PasoEquipo.svelte';
	import PasoAgenda from './PasoAgenda.svelte';
	import PasoCobro from './PasoCobro.svelte';
	import PasoWeb from './PasoWeb.svelte';
	import PasoListo from './PasoListo.svelte';

	type Datos = {
		paso: Paso;
		borrador: { nombre: string; slug: string; rubro: string };
		cargado: Cargado | null;
		persona: string;
		feriados: Feriado[];
		origen: string;
	};
	let { data, form }: { data: Datos; form: { error?: string } | null | undefined } = $props();

	const b = $derived(new Borrador(data));
	const paso = $derived(data.paso);
	const i = $derived(PASOS.indexOf(paso));
	const info = $derived(INFO[paso]);
	const creado = $derived(!!data.cargado);
	const ult = $derived(paso === 'listo');

	// «Guardado»: un momento después de cada paso que guardó.
	let guardado = $state(false);
	let enviando = $state(false);
	let t: ReturnType<typeof setTimeout> | undefined;
	onDestroy(() => clearTimeout(t));
	const alEnviar: SubmitFunction = () => {
		enviando = true;
		return async ({ result, update }) => {
			if (result.type === 'redirect') {
				guardado = true;
				clearTimeout(t);
				t = setTimeout(() => (guardado = false), 1600);
			}
			await update();
			enviando = false;
		};
	};

	// Cada paso dice si se puede seguir.
	let listo = $state(false);

	const extraNuevo = $derived(creado ? { n: '1' } : { rubro: b.rubro, nombre: b.nombre || undefined, slug: b.slugManual ?? undefined });
	const volverA = $derived(i > 0 && !ult ? urlPaso(anterior(paso), anterior(paso) === 'rubro' || anterior(paso) === 'nombre' ? extraNuevo : {}) : null);
	const saltarA = $derived(urlPaso(siguiente(paso)));
	const textoBoton = $derived(ult ? 'Ir a mi panel' : paso === 'web' ? 'Publicar mi página' : 'Seguir');
	const iconoBoton = $derived(ult ? 'dashboard' : 'arrow_forward');

	// Avisos que flotan sobre el teléfono a medida que se completa.
	const horas = $derived(
		horasSemana(b.dias.length, b.abre, b.cierra, b.personas.length, b.descOn ? { desde: b.descDesde, hasta: b.descHasta } : null)
	);
	const primero = $derived(b.activos[0]);
	const sena = $derived(
		primero ? (b.cobro === 'fija' ? b.senaFija : b.cobro === 'total' ? primero.precio : primero.precio * 0.3) : 0
	);
	const avisos = $derived(
		[
			primero && i >= 2 ? { ic: 'event_available', titulo: 'Nueva reserva', sub: `${primero.nombre} · 10:30`, clase: 'a1' } : null,
			i >= 5 && b.cobro !== 'sin_cobro' && primero ? { ic: 'paid', titulo: 'Seña pagada', sub: pesos(sena), clase: 'a2' } : null,
			i >= 3 && b.dias.length ? { ic: 'schedule', titulo: b.personas.length > 1 ? `${b.personas.length} agendas` : 'Tu agenda', sub: `${horas} h libres por semana`, clase: 'a3' } : null,
			ult ? { ic: 'star', titulo: 'Primera reseña', sub: '★★★★★', clase: 'a4' } : null
		].filter((x) => !!x)
	);
</script>

<div class="onb">
	<header class="barra-arriba">
		{#if volverA}
			<a class="atras" href={volverA} aria-label="Volver al paso anterior"><Icono nombre="arrow_back" tam={24} /></a>
		{/if}
		<span class="anillo" style:--prog="{Math.round(((i + 1) / PASOS.length) * 360)}deg"
			><img src="/marca-icono.png" alt="Tengo Libre" width="30" height="30" /></span
		>
		<div class="progreso">
			<span class="progreso-t"><b>{info.nombre}</b> · paso {i + 1} de {PASOS.length}</span>
			<div class="segmentos" aria-hidden="true">
				{#each PASOS as _p, j (j)}<span class:hecho={j < i} class:actual={j === i}></span>{/each}
			</div>
		</div>
		<span class="guardado" role="status">{#if guardado}<Icono nombre="check" tam={16} />Guardado{/if}</span>
	</header>

	<div class="cuerpo">
		<main>
			{#key paso}
				<div class="encabezado">
					<span class="num" aria-hidden="true">{String(i + 1).padStart(2, '0')}</span>
					<span class="enc-c">
						<span class="pastilla-paso tl-petroleo"><span class="pp-ic"><Icono nombre={info.icono} lleno tam={17} /></span>{info.nombre}</span>
						<span class="consejo">{info.consejo}</span>
					</span>
				</div>

				{#if paso === 'rubro'}
					<PasoRubro {b} {creado} {alEnviar} bind:listo />
				{:else if paso === 'nombre'}
					<PasoNombre {b} {creado} slug={data.cargado?.negocio.slug} {alEnviar} {form} bind:listo />
				{:else if paso === 'servicios'}
					<PasoServicios {b} {alEnviar} {form} bind:listo />
				{:else if paso === 'equipo'}
					<PasoEquipo {b} {alEnviar} {form} bind:listo />
				{:else if paso === 'agenda'}
					<PasoAgenda {b} feriados={data.feriados} bloqueados={data.cargado?.bloqueos ?? []} {alEnviar} {form} bind:listo />
				{:else if paso === 'cobro'}
					<PasoCobro {b} {alEnviar} {form} bind:listo />
				{:else if paso === 'web'}
					<PasoWeb {b} {alEnviar} {form} bind:listo />
				{:else if data.cargado}
					<PasoListo {b} cargado={data.cargado} origen={data.origen} {horas} />
				{/if}
			{/key}
		</main>

		<aside class="en-vivo tl-petroleo" aria-label="Vista previa de tu página">
			<img class="simbolo" src="/marca/simbolo_una_tinta_blanco.svg" alt="" aria-hidden="true" />
			<span class="franja" aria-hidden="true"></span>
			<span class="en-vivo-t"><span class="punto"></span>Tu página, en vivo</span>
			{#each avisos as a (a.titulo)}
				<span class="aviso-flota {a.clase}" aria-hidden="true">
					<span class="af-ic"><Icono nombre={a.ic} lleno tam={17} /></span>
					<span class="af-c"><b>{a.titulo}</b><span>{a.sub}</span></span>
				</span>
			{/each}
			<div class="telefono" aria-hidden="true">
				<Previa {b} />
			</div>
		</aside>
	</div>

	<footer class="barra-abajo">
		<div class="barra-in">
			{#if info.saltable && creado}
				<a class="saltar" href={saltarA}>Lo completo después</a>
			{/if}
			<span class="espacio"></span>
			{#if ult}
				<a class="seguir" href="/panel">{textoBoton}<Icono nombre={iconoBoton} tam={20} /></a>
			{:else}
				<button class="seguir" form="paso" disabled={!listo || enviando} aria-busy={enviando}>
					{textoBoton}<Icono nombre={iconoBoton} tam={20} />
				</button>
			{/if}
		</div>
	</footer>
</div>

<style>
	.onb {
		min-height: 100vh;
		display: flex;
		flex-direction: column;
		background-color: var(--tl-crema);
		background-image: radial-gradient(rgba(24, 60, 68, 0.07) 1px, transparent 1.3px), var(--tl-grano-osc);
		background-size: 18px 18px, auto;
		color: var(--tl-tinta);
		--lado: 32px;
	}
	.barra-arriba {
		position: sticky;
		top: 0;
		z-index: 10;
		display: flex;
		align-items: center;
		gap: 12px;
		min-height: 64px;
		padding: 10px var(--lado);
		background: rgba(244, 238, 226, 0.9);
		backdrop-filter: blur(10px);
		-webkit-backdrop-filter: blur(10px);
	}
	.atras {
		width: 48px;
		height: 48px;
		margin-left: -8px;
		border-radius: 14px;
		display: grid;
		place-items: center;
		color: var(--tl-tinta);
	}
	.anillo {
		position: relative;
		width: 40px;
		height: 40px;
		flex: none;
		border-radius: 50%;
		background: conic-gradient(#17805a 0 var(--prog), #e2d6bf var(--prog) 360deg);
		transition: background 0.4s;
	}
	.anillo img {
		position: absolute;
		inset: 5px;
		border-radius: 50%;
	}
	.progreso {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 6px;
		max-width: 520px;
		min-width: 0;
	}
	.progreso-t {
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		font-size: 14px;
		font-weight: 700;
		color: var(--tl-sub-2);
	}
	.progreso-t b {
		color: var(--tl-tinta);
	}
	.segmentos {
		display: grid;
		grid-template-columns: repeat(8, 1fr);
		gap: 4px;
	}
	.segmentos span {
		height: 5px;
		border-radius: 3px;
		background: #e2d6bf;
		transition: background 0.3s;
	}
	.segmentos .hecho {
		background: #17805a;
	}
	.segmentos .actual {
		background: #3ccc90;
	}
	.guardado {
		flex: none;
		display: inline-flex;
		align-items: center;
		gap: 4px;
		font-size: 13px;
		font-weight: 700;
		color: var(--tl-sub);
		min-width: 80px;
	}
	.cuerpo {
		flex: 1;
		display: grid;
		grid-template-columns: minmax(0, 1fr) 320px;
		gap: 48px;
		width: 100%;
		max-width: 1120px;
		margin: 0 auto;
		padding: 40px 32px 48px;
		align-items: start;
	}
	main {
		display: flex;
		flex-direction: column;
		gap: 22px;
		min-width: 0;
		animation: tl-paso var(--tl-entrar) var(--tl-ease) both;
	}
	.encabezado {
		display: flex;
		align-items: center;
		gap: 18px;
	}
	.num {
		font-size: 112px;
		line-height: 0.78;
		font-weight: 700;
		letter-spacing: -0.07em;
		color: transparent;
		-webkit-text-stroke: 2px #183c44;
		font-variant-numeric: tabular-nums;
		animation: tl-pop 0.5s both;
	}
	.enc-c {
		display: flex;
		flex-direction: column;
		gap: 8px;
		min-width: 0;
	}
	.pastilla-paso {
		display: inline-flex;
		align-self: flex-start;
		align-items: center;
		gap: 8px;
		height: 36px;
		padding: 0 16px 0 6px;
		border-radius: 999px;
		font-size: 14px;
		font-weight: 700;
	}
	.pp-ic {
		width: 26px;
		height: 26px;
		border-radius: 50%;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		display: grid;
		place-items: center;
	}
	.consejo {
		font-size: 14px;
		line-height: 1.45;
		color: var(--tl-sub-2);
		max-width: 30em;
		text-wrap: pretty;
	}

	/* Tu página, en vivo */
	.en-vivo {
		position: sticky;
		top: 88px;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 14px;
		padding: 22px 0 28px;
		border-radius: 32px;
		overflow: hidden;
	}
	.simbolo {
		position: absolute;
		left: -90px;
		bottom: -90px;
		width: 320px;
		height: 320px;
		opacity: 0.04;
		pointer-events: none;
	}
	.franja {
		position: absolute;
		left: 46px;
		top: -40px;
		width: 48px;
		height: 760px;
		border-radius: 12px;
		background: rgba(255, 255, 255, 0.035);
	}
	.en-vivo-t {
		position: relative;
		display: inline-flex;
		align-items: center;
		gap: 8px;
		font-size: 14px;
		font-weight: 700;
		color: #fff;
	}
	.punto {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: var(--tl-menta);
		animation: tl-latido 1.6s infinite;
	}
	.aviso-flota {
		position: absolute;
		z-index: 3;
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 12px 8px 8px;
		border-radius: 14px;
		background: var(--tl-papel);
		color: var(--tl-tinta);
		box-shadow: 0 18px 30px -14px rgba(0, 0, 0, 0.55);
		font-size: 12px;
		line-height: 1.25;
		animation:
			tl-pop 0.5s both,
			tl-flota-y var(--dur, 6s) ease-in-out infinite;
	}
	.af-ic {
		width: 30px;
		height: 30px;
		flex: none;
		border-radius: 9px;
		display: grid;
		place-items: center;
	}
	.af-c {
		display: flex;
		flex-direction: column;
	}
	.af-c span {
		color: var(--tl-sub);
		font-variant-numeric: tabular-nums;
	}
	.a1 {
		left: -6px;
		top: 120px;
		--dur: 5s;
	}
	.a1 .af-ic {
		background: #183c44;
		color: #3ccc90;
	}
	.a2 {
		left: calc(100% - 150px);
		top: 300px;
		--dur: 6.5s;
	}
	.a2 .af-ic {
		background: #e3f6ec;
		color: #17805a;
	}
	.a3 {
		left: -10px;
		top: 440px;
		--dur: 7s;
	}
	.a3 .af-ic {
		background: #f3fbf7;
		color: #17805a;
	}
	.a4 {
		left: calc(100% - 140px);
		top: 70px;
		--dur: 5.5s;
	}
	.a4 .af-ic {
		background: #fbe9c2;
		color: #7a4a00;
	}
	.telefono {
		position: relative;
		width: 268px;
		height: 540px;
		border-radius: 38px;
		overflow: hidden;
		box-shadow: 0 0 0 8px #0f2328, 0 40px 70px -24px rgba(0, 0, 0, 0.6);
		transform: rotate(-2deg);
		transition: transform 0.4s var(--tl-ease);
	}
	.telefono:hover {
		transform: rotate(0);
	}

	/* Barra de abajo */
	.barra-abajo {
		position: sticky;
		bottom: 0;
		z-index: 10;
		padding: 12px var(--lado) 14px;
		background: rgba(255, 253, 249, 0.94);
		backdrop-filter: blur(10px);
		-webkit-backdrop-filter: blur(10px);
		box-shadow: 0 -1px 0 #e6dcc8;
	}
	.barra-in {
		display: flex;
		align-items: center;
		gap: 12px;
		max-width: 1120px;
		margin: 0 auto;
	}
	.espacio {
		flex: 1;
	}
	.saltar {
		white-space: nowrap;
		min-height: 48px;
		padding: 0 6px;
		display: inline-flex;
		align-items: center;
		font-size: 15px;
		font-weight: 700;
		color: var(--tl-sub);
		text-decoration: none;
	}
	.seguir {
		white-space: nowrap;
		display: inline-flex;
		align-items: center;
		gap: 8px;
		min-height: 54px;
		padding: 0 26px;
		border-radius: 999px;
		border: 0;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		font: 700 17px var(--tl-font);
		text-decoration: none;
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.45), 0 12px 24px -10px rgba(60, 204, 144, 0.8);
		cursor: pointer;
		transition:
			transform var(--tl-tocar) var(--tl-ease),
			background 0.2s;
	}
	.seguir:hover:not(:disabled) {
		transform: translateY(-2px);
	}
	.seguir:active:not(:disabled) {
		transform: scale(0.97);
	}
	.seguir:disabled {
		background: #ece5d6;
		color: var(--tl-sub);
		box-shadow: none;
		cursor: not-allowed;
	}

	/* Piezas comunes de los pasos */
	.onb :global(.hoja) {
		position: relative;
		display: flex;
		flex-direction: column;
		gap: 20px;
		padding: 28px 28px 30px;
		border-radius: 28px;
		background: var(--tl-papel);
		box-shadow: 0 1px 0 rgba(15, 35, 40, 0.04), 0 30px 60px -40px rgba(15, 35, 40, 0.45);
		animation: tl-paso var(--tl-entrar) var(--tl-ease) both;
	}
	.onb :global(.hoja h1) {
		margin: 0;
		font-size: 48px;
		line-height: 1;
		font-weight: 700;
		letter-spacing: -0.05em;
		text-wrap: balance;
	}
	.onb :global(.hoja .bajada) {
		margin: 0;
		font-size: 16px;
		line-height: 1.5;
		color: var(--tl-sub);
		text-wrap: pretty;
	}
	.onb :global(.rot) {
		font-size: 14px;
		font-weight: 700;
	}
	.onb :global(.opciones) {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}
	.onb :global(.opcion) {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		height: 46px;
		padding: 0 16px;
		border-radius: 999px;
		border: 0;
		background: var(--tl-papel);
		color: var(--tl-tinta);
		box-shadow: inset 0 0 0 1.5px #e0d5bf;
		font: 700 15px var(--tl-font);
		font-variant-numeric: tabular-nums;
		cursor: pointer;
		transition:
			transform var(--tl-tocar),
			background var(--tl-tocar);
	}
	.onb :global(.opcion:active) {
		transform: scale(0.96);
	}
	.onb :global(.opcion[aria-pressed='true']),
	.onb :global(.opcion[aria-checked='true']) {
		background: var(--tl-petroleo);
		color: #fff;
		box-shadow: none;
	}
	.onb :global(.titulo-campo) {
		display: flex;
		flex-direction: column;
		gap: 2px;
		margin-top: 6px;
	}
	.onb :global(.titulo-campo b) {
		font-size: 16px;
	}
	.onb :global(.titulo-campo span) {
		font-size: 13px;
		color: var(--tl-sub);
	}
	.onb :global(.horas) {
		display: flex;
		align-items: flex-end;
		gap: 10px;
	}
	.onb :global(.horas label) {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 6px;
		font-size: 13px;
		font-weight: 700;
		color: var(--tl-sub-2);
	}
	.onb :global(.horas input) {
		width: 100%;
		height: 58px;
		padding: 0 16px;
		border-radius: 16px;
		border: 0;
		background: #fff;
		box-shadow: inset 0 0 0 1.5px #e0d5bf;
		font: 700 22px var(--tl-font);
		font-variant-numeric: tabular-nums;
		color: var(--tl-tinta);
		outline: none;
		transition: box-shadow var(--tl-tocar);
	}
	.onb :global(.horas input:focus) {
		box-shadow: inset 0 0 0 2px #3ccc90;
	}
	.onb :global(.horas .flecha) {
		height: 58px;
		display: grid;
		place-items: center;
		color: #6b7a7d;
	}
	.onb :global(.error-paso) {
		display: flex;
		align-items: center;
		gap: 6px;
		margin: 0;
		font-size: 13px;
		font-weight: 700;
		color: #a3241a;
	}
	.onb :global(.nota) {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 14px 16px;
		border-radius: 18px;
		background: #f8f3ea;
		font-size: 14px;
		line-height: 1.4;
	}
	.onb :global(.nota > .icono) {
		color: var(--tl-menta-texto);
	}

	@keyframes -global-tl-paso {
		from {
			opacity: 0;
			transform: translateX(24px);
		}
		to {
			opacity: 1;
			transform: none;
		}
	}
	@keyframes -global-tl-pop {
		0% {
			opacity: 0;
			transform: scale(0.7);
		}
		70% {
			transform: scale(1.05);
		}
		100% {
			opacity: 1;
			transform: none;
		}
	}
	@keyframes tl-flota-y {
		50% {
			transform: translateY(-8px);
		}
	}
	@keyframes tl-latido {
		0% {
			box-shadow: 0 0 0 0 rgba(60, 204, 144, 0.6);
		}
		100% {
			box-shadow: 0 0 0 10px rgba(60, 204, 144, 0);
		}
	}

	@media (max-width: 899px) {
		.onb {
			--lado: 16px;
		}
		.cuerpo {
			grid-template-columns: minmax(0, 1fr);
			padding: 20px 16px 32px;
		}
		.en-vivo {
			display: none;
		}
		.num {
			font-size: 72px;
		}
		.barra-abajo {
			padding-bottom: max(22px, env(safe-area-inset-bottom));
		}
		.guardado {
			min-width: 0;
		}
		.saltar {
			font-size: 14px;
		}
		.seguir {
			padding: 0 20px;
			font-size: 16px;
		}
		.onb :global(.hoja) {
			padding: 20px 16px;
		}
		.onb :global(.hoja h1) {
			font-size: 34px;
		}
	}
</style>
