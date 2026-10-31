<script lang="ts">
	import { enhance } from '$app/forms';
	import { fecha } from '#lib/formato.ts';
	import { GESTION } from '#lib/operacion/secciones.ts';

	let { data, form } = $props();
	const gestiona = $derived(GESTION.includes(data.negocio.rol));
	let respondiendo = $state<string | null>(null);
	const unDecimal = new Intl.NumberFormat('es-AR', { maximumFractionDigits: 1 });
	const enlace = (pagina: number) => {
		const q = new URLSearchParams({ pagina: String(pagina) });
		if (data.filtros.visible) q.set('visible', data.filtros.visible);
		if (data.filtros.sinResponder) q.set('sinResponder', '1');
		return `?${q}`;
	};
</script>

<svelte:head><title>Reseñas · Panel</title></svelte:head>

<div class="pila">
	<h1>Reseñas</h1>
	<p>
		<strong class="promedio">{data.cantidad ? unDecimal.format(data.promedio) : '—'}</strong>
		<span class="suave">de 5 · {data.cantidad}&nbsp;{data.cantidad === 1 ? 'reseña visible' : 'reseñas visibles'} en tu página</span>
	</p>
	{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}

	<form method="GET" class="tarjeta fila" aria-label="Filtros">
		<label class="campo">
			Mostrar
			<select name="visible">
				<option value="" selected={!data.filtros.visible}>Todas</option>
				<option value="si" selected={data.filtros.visible === 'si'}>Visibles</option>
				<option value="no" selected={data.filtros.visible === 'no'}>Ocultas</option>
			</select>
		</label>
		<label class="fila chico" style="align-self: end; min-height: 40px"><input type="checkbox" name="sinResponder" value="1" checked={data.filtros.sinResponder} /> Sin responder</label>
		<button class="boton" style="align-self: end">Filtrar</button>
	</form>

	{#if data.resenas.length === 0}
		<p class="tarjeta suave">Todavía no hay reseñas con esos filtros. Se piden solas por mail después de cada turno&nbsp;atendido.</p>
	{/if}

	<ul class="lista">
		{#each data.resenas as r (r.id)}
			<li class="tarjeta pila" class:oculta={!r.visible}>
				<div class="fila" style="justify-content: space-between">
					<div>
						<span class="estrellas" role="img" aria-label={`${r.puntaje} de 5 estrellas`}>{'★'.repeat(r.puntaje)}{'☆'.repeat(5 - r.puntaje)}</span>
						<strong>{r.nombre}</strong>
						<div class="suave chico">{r.servicio}{r.profesional ? ` con ${r.profesional}` : ''} · {fecha(r.creadoEn)}</div>
					</div>
					{#if !r.visible}<span class="etiqueta">Oculta</span>{/if}
				</div>
				{#if r.comentario}<p class="texto">{r.comentario}</p>{/if}
				{#if r.respuesta && respondiendo !== r.id}
					<blockquote class="respuesta chico"><strong>Tu respuesta:</strong> {r.respuesta}</blockquote>
				{/if}
				{#if gestiona}
					{#if respondiendo === r.id}
						<form
							method="POST"
							action="?/responder"
							class="pila"
							use:enhance={() =>
								async ({ result, update }) => {
									await update();
									if (result.type === 'success') respondiendo = null;
								}}
						>
							<input type="hidden" name="id" value={r.id} />
							<label class="campo">Respuesta pública <textarea name="respuesta" maxlength="1000" rows="3">{r.respuesta}</textarea></label>
							<p class="suave chico">Dejala vacía para sacar la respuesta.</p>
							<div class="fila">
								<button class="boton primario">Guardar respuesta</button>
								<button type="button" class="boton" onclick={() => (respondiendo = null)}>Cancelar</button>
							</div>
						</form>
					{:else}
						<div class="fila">
							<button class="boton chico" onclick={() => (respondiendo = r.id)}>{r.respuesta ? 'Editar respuesta' : 'Responder'}</button>
							<form method="POST" action="?/visibilidad" use:enhance>
								<input type="hidden" name="id" value={r.id} />
								<input type="hidden" name="visible" value={r.visible ? '0' : '1'} />
								<button class="boton chico">{r.visible ? 'Ocultar' : 'Mostrar'}</button>
							</form>
						</div>
					{/if}
				{/if}
			</li>
		{/each}
	</ul>

	<nav class="fila" aria-label="Páginas">
		{#if data.filtros.pagina > 0}<a class="boton chico" href={enlace(data.filtros.pagina - 1)}>← Más nuevas</a>{/if}
		{#if data.resenas.length === 50}<a class="boton chico" href={enlace(data.filtros.pagina + 1)}>Más viejas →</a>{/if}
	</nav>
</div>

<style>
	.promedio {
		font-size: 28px;
	}
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.oculta {
		opacity: 0.75;
	}
	.estrellas {
		color: var(--aviso);
		margin-right: 6px;
	}
	.texto {
		margin: 0;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}
	.respuesta {
		margin: 0;
		padding: 8px 12px;
		border-left: 3px solid var(--marca);
		background: var(--fondo);
		overflow-wrap: anywhere;
	}
</style>
