<script lang="ts">
	import { enhance } from '$app/forms';
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

<div class="pila" style="max-width: 720px">
	<p class="chico"><a href="/panel/calendario">← Calendario</a></p>
	<h1>Verlo en tu calendario</h1>
	<p class="suave">
		Sumá los turnos a Google Calendar, al calendario del iPhone o a Outlook. Se actualizan solos: Apple y Outlook cada 15
		minutos, Google cada algunas&nbsp;horas.
	</p>
	<p class="suave chico">Es un link privado: quien lo tenga ve los turnos con los datos de los clientes. Si se filtra, generá uno&nbsp;nuevo.</p>

	{#if data.calendarios.length === 0}
		<p class="tarjeta">Todavía no tenés un profesional asociado a tu usuario. Pedile al dueño que te vincule en Equipo y&nbsp;horarios.</p>
	{/if}

	{#each data.calendarios as c (c.profesionalId ?? 'negocio')}
		{@const que = `cal-${c.profesionalId ?? ''}`}
		<section class="tarjeta pila" aria-labelledby={`t-${que}`}>
			<h2 id={`t-${que}`} class="chico-titulo">{c.nombre}</h2>
			<Mensaje {form} {que} />
			{#if c.url}
				<div class="fila enlace">
					<label class="campo" style="flex: 1; min-width: 0">
						<span class="visualmente-oculto">Link del calendario de {c.nombre}</span>
						<input readonly value={c.url} onfocus={(e) => e.currentTarget.select()} />
					</label>
					<button type="button" class="boton" onclick={() => copiar(c.url!)}>{copiado === c.url ? 'Copiado' : 'Copiar'}</button>
				</div>
				<div class="fila acciones">
					<a class="boton primario" href={google(c.url)} target="_blank" rel="noopener">Agregar a Google Calendar</a>
					<a class="boton" href={webcal(c.url)}>Apple u Outlook</a>
				</div>
				<div class="fila acciones">
					<form method="POST" action="?/crear" use:enhance>
						<input type="hidden" name="profesionalId" value={c.profesionalId ?? ''} />
						<input type="hidden" name="nuevo" value="1" />
						<button class="boton chico">Generar link nuevo</button>
					</form>
					<form method="POST" action="?/borrar" use:enhance>
						<input type="hidden" name="id" value={c.id} />
						<input type="hidden" name="profesionalId" value={c.profesionalId ?? ''} />
						<button class="boton chico">Desactivar</button>
					</form>
				</div>
			{:else}
				<form method="POST" action="?/crear" use:enhance>
					<input type="hidden" name="profesionalId" value={c.profesionalId ?? ''} />
					<button class="boton primario">Crear link</button>
				</form>
			{/if}
		</section>
	{/each}
</div>

<style>
	.chico-titulo {
		font-size: 1.05rem;
		margin: 0;
	}
	.enlace {
		align-items: flex-end;
		flex-wrap: nowrap;
	}
	.enlace input {
		width: 100%;
		font-size: 0.85rem;
	}
	.acciones {
		flex-wrap: wrap;
		gap: 8px;
	}
</style>
