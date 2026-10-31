<!--
	«Verlo en tu calendario» (handoff «Panel v2», panel del calendario): un link privado por negocio y
	por profesional que se actualiza solo en Google Calendar, Apple u Outlook.
-->
<script lang="ts">
	import { enhance } from '$app/forms';
	import Icono from '#lib/ui/Icono.svelte';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';

	let { data, form } = $props();
	let copiado = $state('');

	const google = (url: string) => `https://calendar.google.com/calendar/render?cid=${encodeURIComponent(url.replace(/^https?:/, 'webcal:'))}`;
	const webcal = (url: string) => url.replace(/^https?:/, 'webcal:');
	async function copiar(url: string) {
		try {
			await navigator.clipboard.writeText(url);
			copiado = url;
		} catch {
			copiado = '';
		}
	}
</script>

<svelte:head><title>Verlo en tu calendario · Tengo Libre</title></svelte:head>

<p class="volver"><a href="/panel/calendario"><Icono nombre="arrow_back" tam={18} />Calendario</a></p>

<article class="hoja" aria-labelledby="t-ical">
	<header class="cabeza tl-petroleo">
		<h1 id="t-ical">Verlo en tu calendario</h1>
		<p>Un link privado que se actualiza solo en Google Calendar, Apple u Outlook (Apple y Outlook cada 15 minutos, Google cada algunas horas). Quien lo tenga ve los turnos: si se filtra, generá uno&nbsp;nuevo.</p>
	</header>
	<div class="cuerpo">
		{#if data.calendarios.length === 0}
			<p class="vacio">Todavía no tenés un profesional asociado a tu usuario. Pedile al dueño que te vincule en Equipo y&nbsp;horarios.</p>
		{/if}
		{#each data.calendarios as c (c.profesionalId ?? 'negocio')}
			{@const que = `cal-${c.profesionalId ?? ''}`}
			<section class="link" aria-labelledby={`t-${que}`}>
				<div class="fila-l">
					<Icono nombre={c.profesionalId ? 'person' : 'storefront'} tam={22} />
					<h2 id={`t-${que}`}>{c.nombre}</h2>
					<span class="tl-rotulo est" class:on={!!c.url}>{c.url ? 'Activo' : 'Sin link'}</span>
				</div>
				<Mensaje {form} {que} />
				{#if c.url}
					<label class="url">
						<span class="visualmente-oculto">Link del calendario de {c.nombre}</span>
						<input readonly value={c.url} class="tl-num" onfocus={(e) => e.currentTarget.select()} />
					</label>
					<div class="botones">
						<a class="b oscuro" href={google(c.url)} target="_blank" rel="noopener"><img src="/logos/google.svg" alt="" width="15" height="15" />Agregar a Google</a>
						<a class="b" href={webcal(c.url)}>Apple</a>
						<a class="b" href={webcal(c.url)}>Outlook</a>
						<button type="button" class="b" onclick={() => copiar(c.url!)}>{copiado === c.url ? 'Copiado' : 'Copiar'}</button>
					</div>
					<div class="links">
						<form method="POST" action="?/crear" use:enhance>
							<input type="hidden" name="profesionalId" value={c.profesionalId ?? ''} />
							<input type="hidden" name="nuevo" value="1" />
							<button class="verde">Generar uno nuevo</button>
						</form>
						<form method="POST" action="?/borrar" use:enhance>
							<input type="hidden" name="id" value={c.id} />
							<input type="hidden" name="profesionalId" value={c.profesionalId ?? ''} />
							<button class="rojo">Desactivar</button>
						</form>
					</div>
				{:else}
					<form method="POST" action="?/crear" use:enhance>
						<input type="hidden" name="profesionalId" value={c.profesionalId ?? ''} />
						<button class="b oscuro ancho">Crear link</button>
					</form>
				{/if}
			</section>
		{/each}
	</div>
</article>

<style>
	.volver {
		margin: 0;
	}
	.volver a {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		min-height: 40px;
		font-size: 14px;
		font-weight: 700;
		color: var(--tl-menta-texto);
		text-decoration: none;
	}
	.hoja {
		max-width: 560px;
		border-radius: 24px;
		background: var(--tl-papel);
		overflow: hidden;
		box-shadow: 0 30px 60px -36px rgba(15, 35, 40, 0.55);
		animation: tl-gin var(--tl-entrar) var(--tl-ease) both;
	}
	.cabeza {
		padding: 22px 20px;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	h1 {
		margin: 0;
		font-size: 22px;
		letter-spacing: -0.03em;
	}
	.cabeza p {
		margin: 0;
		font-size: 14px;
		line-height: 1.45;
		color: rgba(255, 255, 255, 0.8);
	}
	.cuerpo {
		padding: 16px 20px 24px;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.vacio {
		margin: 0;
		font-size: 14px;
		color: var(--tl-sub);
	}
	.link {
		display: flex;
		flex-direction: column;
		gap: 10px;
		padding: 14px 16px;
		border-radius: 16px;
		background: #f8f3ea;
	}
	.fila-l {
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.fila-l :global(.icono) {
		color: var(--tl-menta-texto);
	}
	h2 {
		flex: 1;
		margin: 0;
		font-size: 15px;
	}
	.est {
		font-size: 11px;
		color: var(--tl-sub);
	}
	.est.on {
		color: var(--tl-menta-texto);
	}
	.url input {
		width: 100%;
		height: 40px;
		padding: 0 12px;
		border: 0;
		border-radius: 10px;
		background: var(--tl-papel);
		box-shadow: inset 0 0 0 1px var(--tl-linea);
		font: 400 13px var(--tl-font);
		color: var(--tl-sub);
	}
	.botones {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
	}
	.b {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		min-height: 40px;
		padding: 0 14px;
		border: 0;
		border-radius: 999px;
		background: var(--tl-papel);
		box-shadow: inset 0 0 0 1px #d8ccb4;
		color: var(--tl-tinta);
		font: 700 13px var(--tl-font);
		text-decoration: none;
		cursor: pointer;
	}
	.b.oscuro {
		background: var(--tl-petroleo);
		color: #fff;
		box-shadow: none;
	}
	.b.oscuro img {
		background: #fff;
		border-radius: 50%;
		padding: 1px;
	}
	.ancho {
		width: 100%;
		justify-content: center;
		min-height: 48px;
	}
	.links {
		display: flex;
		gap: 14px;
	}
	.links button {
		min-height: 36px;
		border: 0;
		background: transparent;
		padding: 0;
		font: 700 13px var(--tl-font);
		cursor: pointer;
	}
	.verde {
		color: var(--tl-menta-texto);
	}
	.rojo {
		color: #a3241a;
	}
	@media (max-width: 859px) {
		.b {
			min-height: 44px;
		}
		.links button {
			min-height: 44px;
		}
	}
</style>
