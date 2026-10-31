<script lang="ts">
	import '#lib/gestion/gestion.css';
	import { enhance } from '$app/forms';
	import { fecha } from '#lib/formato.ts';
	import { GESTION } from '#lib/operacion/secciones.ts';
	import { premioDe } from '#lib/configuracion/referidos.ts';

	let { data, form } = $props();
	const gestiona = $derived(GESTION.includes(data.negocio.rol));
	// Las respondidas se editan a pedido; las sin responder ya vienen abiertas.
	let editando = $state<string | null>(null);
	const unDecimal = new Intl.NumberFormat('es-AR', { minimumFractionDigits: 1, maximumFractionDigits: 1 });
	const enlace = (pagina: number) => {
		const q = new URLSearchParams({ pagina: String(pagina) });
		if (data.filtros.visible) q.set('visible', data.filtros.visible);
		if (data.filtros.sinResponder) q.set('sinResponder', '1');
		return `?${q}`;
	};
	const f = $derived(data.filtros);
	const filtros = $derived([
		{ nombre: 'Todas', href: '?', activo: !f.visible && !f.sinResponder },
		{ nombre: 'Sin responder', href: '?sinResponder=1', activo: f.sinResponder && !f.visible },
		{ nombre: 'Visibles', href: '?visible=si', activo: f.visible === 'si' && !f.sinResponder },
		{ nombre: 'Ocultas', href: '?visible=no', activo: f.visible === 'no' && !f.sinResponder }
	]);

	// Cuántas hay de cada puntaje: lo da la API (distribucion) o, si están todas en la página, se cuenta.
	const distribucion = $derived.by(() => {
		if (data.distribucion) return [5, 4, 3, 2, 1].map((p) => data.distribucion?.[String(p) as '1'] ?? 0);
		const visibles = data.resenas.filter((r) => r.visible);
		if (f.pagina > 0 || f.visible || f.sinResponder || visibles.length < data.cantidad) return null;
		return [5, 4, 3, 2, 1].map((p) => visibles.filter((r) => r.puntaje === p).length);
	});
	const maxDist = $derived(distribucion ? Math.max(1, ...distribucion) : 1);
	const estrellas = (n: number) => '★'.repeat(n);
	const premio = $derived(data.referidos ? premioDe(data.referidos) : null);
</script>

<svelte:head><title>Reseñas · Panel</title></svelte:head>

<div class="g-pagina pagina">
	<div class="g-col izq">
		<div class="g-cabeza">
			<h1 class="g-titulo">Reseñas</h1>
			{#if data.demora !== null}
				<span class="g-meta">Se pide {data.demora === 0 ? 'al terminar el turno' : `${data.demora} h después del turno`}</span>
			{/if}
		</div>

		<div class="g-bloque resumen">
			<div class="promedio">
				<b class="numero">{data.cantidad ? unDecimal.format(data.promedio) : '—'}</b>
				<span class="g-sub">{data.cantidad}&nbsp;{data.cantidad === 1 ? 'reseña' : 'reseñas'}</span>
			</div>
			{#if distribucion}
				<ul class="distribucion" aria-label="Reseñas por puntaje">
					{#each distribucion as n, i (i)}
						<li>
							<span class="p" aria-label={`${5 - i} estrellas`}>{5 - i}★</span>
							<span class="pista" aria-hidden="true"><span style:width="{(n / maxDist) * 100}%"></span></span>
							<span class="n">{n}</span>
						</li>
					{/each}
				</ul>
			{:else}
				<p class="g-nota">Las reseñas visibles salen en tu página. Se piden solas por mail después de cada turno&nbsp;atendido.</p>
			{/if}
		</div>

		<nav class="g-chips" aria-label="Filtrar reseñas">
			{#each filtros as x (x.nombre)}
				<a class="g-chip" href={x.href} aria-current={x.activo ? 'page' : undefined}>{x.nombre}</a>
			{/each}
		</nav>

		{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}

		{#if data.resenas.length === 0}
			<div class="g-bloque g-vacio">
				<span class="icono lleno" aria-hidden="true">star</span>
				Todavía no hay reseñas con esos filtros. Se piden solas por mail después de cada turno&nbsp;atendido.
			</div>
		{/if}

		<ul class="lista">
			{#each data.resenas as r (r.id)}
				{@const abierta = gestiona && (editando === r.id || !r.respuesta)}
				<li class="g-bloque resena" class:pendiente={!r.respuesta && r.visible} class:oculta={!r.visible}>
					<div class="cab">
						<b class="nombre">{r.nombre}</b>
						<span class="estrellas" role="img" aria-label={`${r.puntaje} de 5 estrellas`}>{estrellas(r.puntaje)}</span>
						<span class="g-sub">{r.servicio}{r.profesional ? ` con ${r.profesional}` : ''} · {fecha(r.creadoEn)}</span>
						<span class="estado">
							{#if !r.visible}
								<span class="g-meta"><span class="icono" aria-hidden="true">visibility_off</span>Oculta</span>
							{:else if r.respuesta}
								<span class="g-meta ok"><span class="icono" aria-hidden="true">check_circle</span>Respondida</span>
							{:else}
								<span class="g-meta espera"><span class="icono" aria-hidden="true">schedule</span>Sin responder</span>
							{/if}
						</span>
					</div>
					{#if r.comentario}<p class="texto">{r.comentario}</p>{/if}

					{#if abierta}
						<form
							method="POST"
							action="?/responder"
							class="responder"
							use:enhance={() =>
								async ({ result, update }) => {
									await update({ reset: false });
									if (result.type === 'success') editando = null;
								}}
						>
							<input type="hidden" name="id" value={r.id} />
							<label class="visualmente-oculto" for={`resp-${r.id}`}>Respuesta pública a {r.nombre}</label>
							<textarea id={`resp-${r.id}`} name="respuesta" maxlength="1000" rows="2" placeholder={`¡Gracias, ${r.nombre.split(' ')[0]}! …`}>{r.respuesta}</textarea>
							<div class="acciones">
								<button class="g-boton menta">{r.respuesta ? 'Guardar respuesta' : 'Publicar respuesta'}</button>
								{#if editando === r.id}
									<button type="button" class="g-accion neutro" onclick={() => (editando = null)}>Cancelar</button>
								{/if}
								<button class="g-accion neutro" formaction="?/visibilidad" name="visible" value={r.visible ? '0' : '1'} formnovalidate>
									{r.visible ? 'Ocultar reseña' : 'Mostrar reseña'}
								</button>
							</div>
							{#if r.respuesta}<p class="g-nota">Dejala vacía para sacar la&nbsp;respuesta.</p>{/if}
						</form>
					{:else}
						{#if r.respuesta}
							<p class="respuesta"><b>Tu respuesta:</b> {r.respuesta}</p>
						{/if}
						{#if gestiona}
							<div class="acciones">
								<button type="button" class="g-accion" onclick={() => (editando = r.id)}>{r.respuesta ? 'Editar respuesta' : 'Responder'}</button>
								<form method="POST" action="?/visibilidad" use:enhance>
									<input type="hidden" name="id" value={r.id} />
									<input type="hidden" name="visible" value={r.visible ? '0' : '1'} />
									<button class="g-accion neutro">{r.visible ? 'Ocultar reseña' : 'Mostrar reseña'}</button>
								</form>
							</div>
						{/if}
					{/if}
				</li>
			{/each}
		</ul>

		{#if f.pagina > 0 || data.resenas.length === 50}
			<nav class="g-chips" aria-label="Páginas">
				{#if f.pagina > 0}<a class="g-boton" href={enlace(f.pagina - 1)}>← Más nuevas</a>{/if}
				{#if data.resenas.length === 50}<a class="g-boton" href={enlace(f.pagina + 1)}>Más viejas →</a>{/if}
			</nav>
		{/if}
	</div>

	{#if data.referidos}
		<aside class="g-col der" aria-labelledby="t-referidos">
			<div class="g-cabeza">
				<h2 class="g-titulo-2" id="t-referidos">Referidos</h2>
				{#if data.referidos.activo}
					<span class="g-pildora"><span class="icono lleno" aria-hidden="true">check_circle</span>Activo</span>
				{:else}
					<span class="g-pildora neutro"><span class="icono" aria-hidden="true">pause_circle</span>Apagado</span>
				{/if}
			</div>
			<div class="g-oscuro premio">
				<span class="g-kicker">El premio</span>
				{#if premio}
					<p>
						Quien recomienda gana <b>{premio.descuento}</b> de descuento en un cupón de un uso, cuando su referido reserva por primera&nbsp;vez.
					</p>
				{:else}
					<p>Tus clientes comparten su código y quien lo usa reserva. Todavía no hay premio para quien&nbsp;recomienda.</p>
				{/if}
				<a class="g-boton chico menta" href="/panel/referidos">Editar el programa</a>
			</div>
		</aside>
	{/if}
</div>

<style>
	.pagina {
		display: grid;
		grid-template-columns: minmax(0, 1.3fr) minmax(0, 1fr);
		gap: 24px;
		align-items: start;
	}
	.izq {
		gap: 14px;
	}
	.resumen {
		flex-direction: row;
		align-items: center;
		gap: 24px;
		padding: 22px 18px;
	}
	.promedio {
		display: flex;
		flex-direction: column;
		min-width: 120px;
	}
	.numero {
		font: 700 64px/0.95 var(--tl-font);
		font-variant-numeric: tabular-nums;
		letter-spacing: -0.06em;
	}
	.distribucion {
		list-style: none;
		margin: 0;
		padding: 0;
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}
	.distribucion li {
		display: grid;
		grid-template-columns: 28px minmax(0, 1fr) 32px;
		align-items: center;
		gap: 8px;
		font-size: 11px;
		font-variant-numeric: tabular-nums;
		color: var(--tl-sub-2);
	}
	.pista {
		height: 8px;
		border-radius: 999px;
		background: #f0e8d8;
		overflow: hidden;
	}
	.pista span {
		display: block;
		height: 100%;
		border-radius: inherit;
		background: var(--tl-petroleo);
		min-width: 4px;
	}
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.resena {
		gap: 10px;
	}
	.resena.pendiente {
		box-shadow: 0 0 0 2px #f0c46a;
	}
	.resena.oculta {
		opacity: 0.8;
	}
	.cab {
		display: flex;
		flex-wrap: wrap;
		align-items: baseline;
		gap: 4px 10px;
	}
	.nombre {
		font-size: 15px;
	}
	.estrellas {
		color: #b26a00;
		letter-spacing: 0.04em;
	}
	.estado {
		margin-left: auto;
	}
	.estado .g-meta {
		display: inline-flex;
		align-items: center;
		gap: 4px;
	}
	.estado .icono {
		font-size: 15px;
	}
	.g-meta.espera {
		color: #7a4a00;
	}
	.texto {
		margin: 0;
		font-size: 14px;
		line-height: 1.5;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}
	.responder {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	textarea {
		width: 100%;
		min-height: 56px;
		padding: 12px 14px;
		border: 0;
		border-radius: 12px;
		background: #f8f3ea;
		box-shadow: inset 0 0 0 1.5px var(--tl-menta);
		font: 400 14px/1.45 var(--tl-font);
		color: var(--tl-tinta);
		resize: vertical;
	}
	textarea:focus {
		outline: none;
		box-shadow: inset 0 0 0 2px var(--tl-menta-texto);
	}
	.acciones {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px 20px;
	}
	.respuesta {
		margin: 0;
		padding: 10px 12px;
		border-radius: 12px;
		background: #f8f3ea;
		font-size: 13px;
		line-height: 1.45;
		color: var(--tl-sub-2);
		overflow-wrap: anywhere;
	}
	.der {
		gap: 14px;
	}
	.der .g-cabeza {
		align-items: center;
	}
	.premio {
		border-radius: var(--tl-r-l);
		padding: 20px;
		gap: 10px;
	}
	.premio .g-kicker {
		color: var(--tl-menta);
	}
	.premio p {
		margin: 0;
		font-size: 15px;
		line-height: 1.5;
	}
	.premio b {
		color: var(--tl-menta);
	}
	.premio .g-boton {
		align-self: flex-start;
	}
	@media (max-width: 1000px) {
		.pagina {
			grid-template-columns: minmax(0, 1fr);
		}
	}
	@media (max-width: 640px) {
		.resumen {
			gap: 16px;
		}
		.promedio {
			min-width: 84px;
		}
		.numero {
			font-size: 52px;
		}
		textarea {
			font-size: 16px;
		}
	}
</style>
