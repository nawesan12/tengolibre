<script lang="ts">
	import { page } from '$app/state';
	import Icono from '#lib/ui/Icono.svelte';
	import Logo from '#lib/ui/Logo.svelte';

	const noEsta = $derived(page.status === 404);
	// El mensaje propio de la ruta («Esa invitación no existe o ya no sirve.»), no el genérico.
	const mensaje = $derived.by(() => {
		const m = page.error?.message ?? '';
		return noEsta && m && !/^not found/i.test(m) && m !== 'No encontramos esta página.' ? m : '';
	});
</script>

<svelte:head><title>{page.status} · Tengo Libre</title><meta name="robots" content="noindex" /></svelte:head>

<div class="error-pagina">
	<main class="columna">
		<a href="/" class="inicio" aria-label="Tengo Libre, inicio"><Logo alto={24} /></a>
		{#if noEsta}
			<span class="rotulo tl-num">ERROR 404 · NO ESTÁ</span>
			<h1>No encontramos esta página</h1>
			<p>{mensaje || 'El link puede estar mal escrito o la página ya no existe.'}</p>
			<div class="tl-hueco hueco" aria-hidden="true"><b class="tl-num">404</b><span>Acá no hay nada, pero quedó el lugar&nbsp;libre</span></div>
			<div class="acciones">
				<a class="cta oscuro" href="/">Volver al inicio</a>
				<a class="cta2" href="/explorar">Buscar dónde sacar turno</a>
			</div>
		{:else}
			<span class="rotulo tl-num rojo">ERROR {page.status}</span>
			<h1>Algo salió mal</h1>
			<p>El servidor no respondió. Probá de nuevo en un&nbsp;momento.</p>
			<p class="aviso-rojo"><Icono nombre="error" lleno tam={20} /><span>Si sigue igual, escribinos a <a href="mailto:hola@tengolibre.com.ar">hola@tengolibre.com.ar</a></span></p>
			<div class="acciones">
				<button class="cta claro" onclick={() => location.reload()}>Reintentar</button>
				<a class="cta2" href="/">Volver al inicio</a>
			</div>
		{/if}
	</main>
</div>

<style>
	.error-pagina {
		min-height: 100dvh;
		background: var(--tl-crema-pagina);
	}
	.columna {
		width: min(100%, 440px);
		min-height: 100dvh;
		margin: 0 auto;
		padding: 16px 18px 22px;
		display: flex;
		flex-direction: column;
		gap: 14px;
	}
	.inicio {
		display: inline-flex;
		align-self: flex-start;
		align-items: center;
		min-height: 48px;
		margin-bottom: 16px;
	}
	.rotulo {
		font-size: 11px;
		font-weight: 700;
		letter-spacing: 0.08em;
		color: var(--tl-menta-texto);
	}
	.rotulo.rojo {
		color: #a3241a;
	}
	h1 {
		margin: 0;
		font-size: 28px;
		line-height: 1.02;
		font-weight: 700;
		letter-spacing: -0.045em;
	}
	p {
		margin: 0;
		font-size: 15px;
		line-height: 1.5;
		color: var(--tl-sub-2);
	}
	.hueco {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px 12px;
		min-height: 50px;
		padding: 8px 14px;
		border-radius: 14px;
	}
	.hueco b {
		font-size: 14px;
		color: var(--tl-menta-texto);
	}
	.hueco span {
		flex: 1;
		font-size: 13px;
		font-weight: 700;
	}
	.aviso-rojo {
		display: flex;
		gap: 10px;
		padding: 14px;
		border-radius: 16px;
		background: #fbe0dc;
		color: #5b1a13;
		font-size: 14px;
		line-height: 1.45;
	}
	.aviso-rojo a {
		color: inherit;
		font-weight: 700;
		overflow-wrap: anywhere;
	}
	.acciones {
		margin-top: auto;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}
	.cta {
		display: grid;
		place-items: center;
		height: 54px;
		border: 0;
		border-radius: 999px;
		font: 700 16px var(--tl-font);
		text-decoration: none;
		cursor: pointer;
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.cta:active {
		transform: scale(0.97);
	}
	.cta.oscuro {
		background: var(--tl-petroleo);
		color: #fff;
		box-shadow: var(--tl-luz-arriba-osc);
	}
	.cta.claro {
		background: var(--tl-papel);
		color: #a3241a;
		box-shadow: var(--tl-sh-tarjeta), inset 0 0 0 1px var(--tl-linea);
	}
	.cta2 {
		display: grid;
		place-items: center;
		height: 48px;
		font-size: 15px;
		font-weight: 700;
		color: var(--tl-menta-texto);
		text-decoration: none;
	}
	.cta2:hover {
		text-decoration: underline;
	}
	@media (min-width: 720px) {
		.columna {
			min-height: 0;
			padding-top: 64px;
		}
		.acciones {
			margin-top: 12px;
		}
	}
</style>
