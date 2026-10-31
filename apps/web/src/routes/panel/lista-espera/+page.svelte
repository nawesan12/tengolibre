<!--
	Lista de espera (handoff «Panel v2»): Esperando / Avisados / Vencidos; cada persona es un talón con
	los días que le sirven y, si hay un hueco que le sirve, el horario sugerido + Avisar / Quitar.
-->
<script lang="ts">
	import { enhance } from '$app/forms';
	import Icono from '#lib/ui/Icono.svelte';
	import { dia, fechaHora, hora } from '#lib/formato.ts';
	import { TABS_ESPERA, tabDe } from '#lib/operacion/espera.ts';
	import SinPermiso from '#lib/operacion/SinPermiso.svelte';
	import type { Anotado } from '#lib/operacion/tipos.ts';

	let { data, form } = $props();

	const cuenta = $derived(Object.fromEntries(TABS_ESPERA.map((t) => [t.clave, data.espera.filter((a) => tabDe(a, data.hoy) === t.clave).length])));
	const lista = $derived(data.espera.filter((a) => tabDe(a, data.hoy) === data.tab));

	const corto = (d: string) => {
		const p = Object.fromEntries(new Intl.DateTimeFormat('es-AR', { weekday: 'short', day: '2-digit', timeZone: 'UTC' }).formatToParts(new Date(`${d}T12:00:00Z`)).map((x) => [x.type, x.value]));
		return `${d === data.hoy ? 'Hoy' : String(p.weekday).replace('.', '')} ${p.day}`;
	};
	const franja = (a: Anotado) => (a.desde === a.hasta ? corto(a.desde) : `${corto(a.desde)} → ${corto(a.hasta)}`);
	const sugerido = (inicio: string) => `${dia(inicio) === data.hoy ? 'Hoy' : corto(dia(inicio))} ${hora(inicio)}`;
	const momento = (a: Anotado) => (a as Anotado & { momento?: string }).momento ?? '';
</script>

<svelte:head><title>Lista de espera · Panel</title></svelte:head>

<div class="cabeza">
	<h1>Lista de espera</h1>
	<span class="crece"></span>
	{#if !data.prohibido}
		<nav class="tabs" aria-label="Estado">
			{#each TABS_ESPERA as t (t.clave)}
				<a href={t.clave === 'esperando' ? '?' : `?tab=${t.clave}`} class:on={data.tab === t.clave} aria-current={data.tab === t.clave ? 'page' : undefined} data-sveltekit-noscroll
					>{t.nombre}<span class="tl-num">{cuenta[t.clave]}</span></a
				>
			{/each}
		</nav>
	{/if}
</div>

{#if data.prohibido}
	<SinPermiso que="la lista de espera" />
{:else}
	{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}

	<ul class="lista">
		{#each lista as a (a.id)}
			{@const sug = data.sugeridos[a.id] ?? []}
			<li class="talon" class:vencido={data.tab === 'vencidos'} data-anotado={a.id}>
				<div class="stub">
					<span class="tl-rotulo r">Le sirve</span>
					<b class="tl-num tl-rotulo-b">{franja(a)}</b>
					{#if momento(a)}<span class="tl-rotulo m">{momento(a)}</span>{/if}
				</div>
				<div class="quien">
					<b>{a.nombre}</b>
					<span class="srv">{a.servicio} · {a.profesional ?? 'Cualquiera'}</span>
					<span class="contacto tl-num">
						<a href={`mailto:${a.email}`}>{a.email}</a>{#if a.telefono}{' · '}<a href={`tel:${a.telefono}`}>{a.telefono}</a>{/if}
					</span>
				</div>
				<div class="acciones">
					{#if data.tab === 'esperando'}
						<form method="POST" action="?/avisar" use:enhance class="avisar">
							<input type="hidden" name="id" value={a.id} />
							{#if sug.length}
								<label class="match">
									<span class="visualmente-oculto">Horario sugerido</span>
									<select name="inicio" class="tl-num">
										{#each sug as h (h.inicio)}<option value={h.inicio}>{sugerido(h.inicio).toUpperCase()}</option>{/each}
										<option value="">SIN HORARIO PUNTUAL</option>
									</select>
								</label>
							{:else}
								<input type="hidden" name="inicio" value="" />
							{/if}
							<button class="btn-avisar"><Icono nombre="campaign" tam={16} />Avisar</button>
						</form>
						<form
							method="POST"
							action="?/borrar"
							use:enhance={({ cancel }) => {
								if (!confirm(`¿Sacar a ${a.nombre} de la lista?`)) cancel();
							}}
						>
							<input type="hidden" name="id" value={a.id} />
							<button class="quitar" aria-label={`Quitar a ${a.nombre} de la lista`}>Quitar</button>
						</form>
					{:else if a.avisadoEn}
						<span class="tl-rotulo nota verde">Avisado el {fechaHora(a.avisadoEn)}</span>
					{:else}
						<span class="tl-rotulo nota">Venció sin lugar</span>
					{/if}
				</div>
			</li>
		{/each}
	</ul>

	{#if lista.length === 0}
		<div class="vacio">
			<Icono nombre="hourglass_empty" tam={30} />
			<b>{data.tab === 'esperando' ? 'Nadie esperando acá' : data.tab === 'avisados' ? 'Todavía no avisaste a nadie' : 'Nada vencido'}</b>
			<span>Cuando un día se llena, tu página ofrece anotarse. Si alguien cancela, te mostramos a quién avisar&nbsp;primero.</span>
		</div>
	{/if}
	<p class="pie">El primero que toma el lugar se lo queda. Avisamos por mail y, si dejó teléfono, con un link a&nbsp;WhatsApp.</p>
{/if}

<style>
	.cabeza {
		display: flex;
		align-items: flex-end;
		flex-wrap: wrap;
		gap: 14px 16px;
	}
	h1 {
		margin: 0;
		font-size: 50px;
		line-height: 0.9;
		font-weight: 700;
		letter-spacing: -0.055em;
	}
	.crece {
		flex: 1;
	}
	.tabs {
		display: flex;
		padding: 3px;
		border-radius: 999px;
		background: var(--tl-papel);
		box-shadow: inset 0 0 0 1px #e6dcc8;
		max-width: 100%;
		overflow-x: auto;
	}
	.tabs a {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		height: 38px;
		padding: 0 14px;
		border-radius: 999px;
		font-size: 13px;
		font-weight: 700;
		color: var(--tl-tinta);
		text-decoration: none;
		white-space: nowrap;
	}
	.tabs a span {
		opacity: 0.75;
	}
	.tabs a.on {
		background: var(--tl-petroleo);
		color: #fff;
	}
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.talon {
		--x: 150px;
		display: grid;
		grid-template-columns: var(--x) minmax(0, 1fr) auto;
		border-radius: 16px;
		background:
			radial-gradient(circle at var(--x) 0, var(--tl-crema) 9px, transparent 9.5px),
			radial-gradient(circle at var(--x) 100%, var(--tl-crema) 9px, transparent 9.5px),
			var(--tl-papel);
		box-shadow: 0 10px 20px -16px rgba(15, 35, 40, 0.45);
		animation: tl-gin var(--tl-entrar) var(--tl-ease) both;
	}
	.talon.vencido {
		opacity: 0.65;
	}
	.stub {
		display: flex;
		flex-direction: column;
		justify-content: center;
		gap: 2px;
		padding: 12px 16px;
		border-right: 2px dashed var(--tl-linea-2);
	}
	.r {
		font-size: 11px;
		font-weight: 700;
		color: var(--tl-sub);
		letter-spacing: 0.04em;
	}
	.tl-rotulo-b {
		font-size: 14px;
		font-weight: 700;
		line-height: 1.25;
		text-transform: uppercase;
	}
	.m {
		font-size: 11px;
		color: var(--tl-menta-texto);
	}
	.quien {
		display: flex;
		flex-direction: column;
		justify-content: center;
		gap: 3px;
		padding: 12px 16px;
		min-width: 0;
	}
	.quien b {
		font-size: 15px;
	}
	.srv {
		font-size: 13px;
		color: var(--tl-sub);
	}
	.contacto {
		font-size: 12px;
		color: var(--tl-sub);
		overflow-wrap: anywhere;
	}
	.contacto a {
		color: inherit;
		text-decoration: none;
	}
	.contacto a:hover {
		text-decoration: underline;
	}
	.acciones {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		justify-content: flex-end;
		gap: 8px;
		padding: 10px 14px;
	}
	.avisar {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-wrap: wrap;
	}
	.match select {
		appearance: none;
		height: 34px;
		padding: 0 10px;
		border: 0;
		border-radius: 10px;
		background: repeating-linear-gradient(-45deg, rgba(60, 204, 144, 0.22) 0 2px, transparent 2px 9px), #f3fbf7;
		box-shadow: inset 0 0 0 1px rgba(23, 128, 90, 0.3);
		font: 700 12px var(--tl-font);
		letter-spacing: 0.04em;
		color: var(--tl-menta-texto);
		cursor: pointer;
	}
	.btn-avisar {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		height: 38px;
		padding: 0 14px;
		border-radius: 999px;
		border: 0;
		background: var(--tl-petroleo);
		color: #fff;
		font: 700 13px var(--tl-font);
		cursor: pointer;
	}
	.btn-avisar :global(.icono) {
		color: var(--tl-menta);
	}
	.quitar {
		height: 38px;
		padding: 0 10px;
		border: 0;
		background: transparent;
		font: 700 13px var(--tl-font);
		color: var(--tl-sub);
		cursor: pointer;
	}
	.nota {
		font-size: 12px;
		color: var(--tl-sub);
		letter-spacing: 0.04em;
	}
	.nota.verde {
		color: var(--tl-menta-texto);
	}
	.vacio {
		padding: 40px 24px;
		border-radius: 20px;
		background: var(--tl-papel);
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 10px;
	}
	.vacio :global(.icono) {
		color: var(--tl-menta-texto);
	}
	.vacio b {
		font-size: 18px;
	}
	.vacio span,
	.pie {
		font-size: 14px;
		color: var(--tl-sub);
		max-width: 32em;
		text-wrap: pretty;
	}
	.pie {
		margin: 0;
		font-size: 13px;
		max-width: none;
	}
	@media (max-width: 859px) {
		h1 {
			font-size: 34px;
		}
		.talon {
			--x: 112px;
			grid-template-columns: var(--x) minmax(0, 1fr);
		}
		.acciones {
			grid-column: 1 / -1;
			justify-content: flex-start;
			border-top: 1px solid #f0e8d8;
		}
		.btn-avisar,
		.quitar,
		.match select {
			height: 44px;
		}
	}
</style>
