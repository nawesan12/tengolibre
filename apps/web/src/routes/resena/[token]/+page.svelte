<script lang="ts">
	import { enhance } from '$app/forms';
	import { dia, hora } from '#lib/formato.ts';
	import { diaCorto } from '#lib/publico/calendario.ts';
	import { completar, contraste, mezclar, tema, variables } from '#lib/publico/estilo.ts';

	let { data, form } = $props();
	const p = $derived(data.pedido);
	const tm = $derived(tema(completar(data.web?.estilo), data.web?.colorPrimario, data.web?.colorAcento));
	// Las estrellas elegidas: el acento aclarado de fondo y oscurecido para la estrella (≥ 3:1).
	const estrellaBg = $derived(mezclar(tm.acc, tm.noche ? '#14221f' : '#fffdf9', 0.55));
	const estrellaC = $derived.by(() => {
		let c = tm.acc;
		for (let t = 0; contraste(c, estrellaBg) < 3 && t < 1; ) {
			t += 0.05;
			c = mezclar(tm.acc, tm.noche ? '#ffffff' : '#000000', t);
		}
		return c;
	});
	const vars = $derived(`${variables(tm)};--estrella-bg:${estrellaBg};--estrella:${estrellaC}`);
	const etiquetas = ['Muy mal', 'Mal', 'Bien', '¡Muy bien!', '¡Excelente!'];
	const f = $derived(dia(p.inicio));
	const detalle = $derived([p.servicio, p.profesional ? `con ${p.profesional.split(' ')[0]}` : ''].filter(Boolean).join(' '));
</script>

<svelte:head>
	<title>Tu opinión sobre {p.negocio} · Tengo Libre</title>
	<meta name="robots" content="noindex, nofollow" />
	<meta name="referrer" content="no-referrer" />
	<meta name="theme-color" content={tm.bg} />
</svelte:head>

<div class="fondo" style={vars}>
	<main class="resena">
		<a class="talon" href={`/${p.slug}`} aria-label={`${p.negocio}: ${detalle}, ${diaCorto(f)} ${hora(p.inicio)}`}>
			<span class="t-cuando"><span class="tl-num">{diaCorto(f)}</span><b class="tl-num">{hora(p.inicio)}</b></span>
			<span class="t-que"><b>{p.negocio}</b><span>{detalle}</span></span>
		</a>

		{#if form && 'gracias' in form}
			<h1>¡Gracias, {p.cliente}!</h1>
			<p class="texto">Tu reseña ayuda a {p.negocio} y a quien busca dónde sacar&nbsp;turno.</p>
			<a class="cta" href={`/${p.slug}`}>Ver {p.negocio}</a>
		{:else if p.resena}
			<h1>Ya dejaste tu reseña</h1>
			<div class="estrellas quietas" aria-label={`${p.resena.puntaje} de 5`}>
				{#each [1, 2, 3, 4, 5] as n (n)}<span class="estrella" class:prendida={n <= p.resena.puntaje} aria-hidden="true">★</span>{/each}
			</div>
			{#if p.resena.comentario}<p class="texto">«{p.resena.comentario}»</p>{/if}
			<p class="texto">¡Gracias por contarnos cómo te&nbsp;fue!</p>
		{:else if p.vencido}
			<h1>Este link ya venció</h1>
			<p class="texto">Los pedidos de reseña sirven una semana. Si querés volver, sacá otro&nbsp;turno.</p>
			<a class="cta" href={`/${p.slug}/reservar`}>Sacar otro turno</a>
		{:else}
			<h1>¿Cómo te fue, {p.cliente}?</h1>
			<form method="POST" use:enhance class="pila">
				{#if form?.error}<p class="aviso-w" role="alert">{form.error}</p>{/if}
				<fieldset class="estrellas">
					<legend class="visualmente-oculto">Tu puntaje</legend>
					{#each [5, 4, 3, 2, 1] as n (n)}
						<input type="radio" id={`p${n}`} name="puntaje" value={n} required />
						<label for={`p${n}`} class="estrella"
							><span aria-hidden="true">★</span><span class="visualmente-oculto">{n} {n === 1 ? 'estrella' : 'estrellas'}: {etiquetas[n - 1]}</span></label
						>
					{/each}
				</fieldset>
				<span class="elegida" aria-hidden="true">
					{#each [1, 2, 3, 4, 5] as n (n)}<b data-n={n}>{etiquetas[n - 1]}</b>{/each}
				</span>
				<label class="campo-w">
					¿Qué te gustó? (opcional)
					<textarea name="comentario" maxlength="1000" rows="4">{form && 'comentario' in form ? form.comentario : ''}</textarea>
				</label>
				<span class="chico">Se publica con tu nombre y la inicial del&nbsp;apellido.</span>
				<button class="cta">Publicar reseña</button>
			</form>
		{/if}
	</main>
</div>

<style>
	.fondo {
		min-height: 100vh;
		background: var(--w-bg);
		color: var(--w-ink);
	}
	.resena {
		max-width: 460px;
		margin: 0 auto;
		padding: 24px 20px 48px;
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	.talon {
		display: grid;
		grid-template-columns: 88px 1fr;
		min-height: 60px;
		border-radius: 16px;
		background:
			radial-gradient(circle at 88px 0, var(--w-bg) 8px, transparent 8.5px),
			radial-gradient(circle at 88px 100%, var(--w-bg) 8px, transparent 8.5px),
			var(--w-pri);
		color: var(--w-on);
		text-decoration: none;
		animation: tl-gin var(--tl-entrar) var(--tl-ease) both;
	}
	.t-cuando {
		display: flex;
		flex-direction: column;
		justify-content: center;
		padding: 0 14px;
		border-right: 2px dashed var(--w-on-a);
	}
	.t-cuando span {
		font: 400 11px var(--tl-font);
		opacity: 0.85;
	}
	.t-cuando b {
		font: 700 18px var(--tl-font);
	}
	.t-que {
		display: flex;
		flex-direction: column;
		justify-content: center;
		padding: 8px 14px;
		font-size: 13px;
		min-width: 0;
	}
	.t-que span {
		opacity: 0.85;
	}
	h1 {
		margin: 0;
		font-size: 30px;
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
	.pila {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	.chico {
		font-size: 12px;
		color: var(--w-sub);
	}
	.estrellas {
		border: 0;
		padding: 0;
		margin: 0;
		display: flex;
		flex-direction: row-reverse;
		gap: 6px;
		min-width: 0;
	}
	.estrellas.quietas {
		flex-direction: row;
	}
	.estrellas input {
		position: absolute;
		opacity: 0;
		width: 1px;
		height: 1px;
	}
	.estrella {
		flex: 1;
		height: 60px;
		border-radius: 16px;
		background: var(--w-campo);
		box-shadow: inset 0 0 0 1.5px var(--w-line2);
		color: var(--w-line2);
		display: grid;
		place-items: center;
		font-size: 30px;
		line-height: 1;
		cursor: pointer;
		transition: transform var(--tl-tocar) var(--tl-ease), background var(--tl-tocar) var(--tl-ease);
	}
	.estrella:active {
		transform: scale(0.94);
	}
	/* Las elegidas: la marcada y las de su izquierda (el orden está invertido). */
	.estrellas input:checked ~ .estrella,
	.estrellas .estrella:hover,
	.estrellas .estrella:hover ~ .estrella,
	.estrella.prendida {
		background: var(--estrella-bg);
		color: var(--estrella);
		box-shadow: none;
	}
	.estrellas input:focus-visible + .estrella {
		outline: 3px solid var(--tl-menta);
		outline-offset: 2px;
	}
	.elegida {
		margin-top: -6px;
		min-height: 22px;
		font-size: 15px;
	}
	.elegida b {
		display: none;
	}
	.pila:has(#p1:checked) .elegida b[data-n='1'],
	.pila:has(#p2:checked) .elegida b[data-n='2'],
	.pila:has(#p3:checked) .elegida b[data-n='3'],
	.pila:has(#p4:checked) .elegida b[data-n='4'],
	.pila:has(#p5:checked) .elegida b[data-n='5'] {
		display: inline;
	}
	.campo-w {
		display: flex;
		flex-direction: column;
		gap: 6px;
		font-size: 14px;
		font-weight: 700;
	}
	textarea {
		padding: 12px 14px;
		border-radius: 14px;
		border: 0;
		background: var(--w-campo);
		color: var(--w-ink);
		box-shadow: inset 0 0 0 1.5px var(--w-line2);
		font: 400 16px var(--tl-font);
		resize: none;
	}
	textarea:focus {
		outline: none;
		box-shadow: inset 0 0 0 2px var(--w-marca);
	}
	.aviso-w {
		margin: 0;
		padding: 12px 14px;
		border-radius: 14px;
		background: #fbe0dc;
		color: #5b1a13;
		font-size: 14px;
	}
	.cta {
		display: grid;
		place-items: center;
		min-height: 54px;
		padding: 0 20px;
		border: 0;
		border-radius: var(--w-br);
		background: var(--w-pri);
		color: var(--w-on);
		font: 700 16px var(--tl-font);
		text-decoration: none;
		cursor: pointer;
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.cta:active {
		transform: scale(0.97);
	}
</style>
