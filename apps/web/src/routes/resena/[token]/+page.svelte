<script lang="ts">
	import { enhance } from '$app/forms';
	import { fecha } from '#lib/formato.ts';
	import { estrellas } from '#lib/publico/textos.ts';

	let { data, form } = $props();
	const p = $derived(data.pedido);
	const etiquetas = ['Muy mal', 'Mal', 'Bien', 'Muy bien', 'Excelente'];
</script>

<svelte:head>
	<title>Tu opinión sobre {p.negocio} · Tengo Libre</title>
	<meta name="robots" content="noindex, nofollow" />
	<meta name="referrer" content="no-referrer" />
</svelte:head>

<main class="contenedor pila" style="max-width: 520px">
	<a class="suave chico" href={`/${p.slug}`}>{p.negocio}</a>
	{#if form && 'gracias' in form}
		<h1>¡Gracias, {p.cliente}!</h1>
		<p>Tu reseña ayuda a {p.negocio} y a quien busca dónde sacar&nbsp;turno.</p>
		<p><a class="boton" href={`/${p.slug}`}>Ver {p.negocio}</a></p>
	{:else if p.resena}
		<h1>Ya dejaste tu reseña</h1>
		<div class="tarjeta pila">
			<span aria-label={`${p.resena.puntaje} de 5`}>{estrellas(p.resena.puntaje)}</span>
			{#if p.resena.comentario}<p style="margin: 0">{p.resena.comentario}</p>{/if}
		</div>
		<p class="suave">¡Gracias por contarnos cómo te&nbsp;fue!</p>
	{:else if p.vencido}
		<h1>Este link ya venció</h1>
		<p class="suave">Los pedidos de reseña sirven una semana. Si querés volver, <a href={`/${p.slug}/reservar`}>sacá otro turno</a>.</p>
	{:else}
		<h1>¿Cómo te fue, {p.cliente}?</h1>
		<p class="suave">{p.servicio}{p.profesional ? ` con ${p.profesional}` : ''}, el {fecha(p.inicio)}.</p>
		<form method="POST" use:enhance class="pila tarjeta">
			{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}
			<fieldset class="estrellas">
				<legend>Tu puntaje</legend>
				{#each [5, 4, 3, 2, 1] as n (n)}
					<input type="radio" id={`p${n}`} name="puntaje" value={n} required />
					<label for={`p${n}`} title={etiquetas[n - 1]}><span aria-hidden="true">★</span><span class="visualmente-oculto">{n} {n === 1 ? 'estrella' : 'estrellas'}: {etiquetas[n - 1]}</span></label>
				{/each}
			</fieldset>
			<label class="campo">
				Contanos más (opcional)
				<textarea name="comentario" maxlength="1000" rows="4">{form && 'comentario' in form ? form.comentario : ''}</textarea>
			</label>
			<button class="boton primario">Enviar reseña</button>
		</form>
	{/if}
</main>

<style>
	.estrellas {
		border: 0;
		padding: 0;
		margin: 0;
		display: flex;
		flex-direction: row-reverse;
		justify-content: flex-end;
		gap: 4px;
	}
	.estrellas legend {
		width: 100%;
		margin-bottom: 6px;
		font-size: 14px;
	}
	.estrellas input {
		position: absolute;
		opacity: 0;
		width: 1px;
		height: 1px;
	}
	.estrellas label {
		font-size: 36px;
		line-height: 1;
		cursor: pointer;
		color: var(--borde);
		padding: 2px;
	}
	/* Las elegidas: la marcada y las de su izquierda (el orden está invertido). */
	.estrellas input:checked ~ label,
	.estrellas label:hover,
	.estrellas label:hover ~ label {
		color: #f5a623;
	}
	.estrellas input:focus-visible + label {
		outline: 2px solid var(--marca);
		border-radius: 4px;
	}
</style>
