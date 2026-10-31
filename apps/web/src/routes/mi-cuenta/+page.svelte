<script lang="ts">
	import { ROLES } from '#lib/formato.ts';
	import Icono from '#lib/ui/Icono.svelte';
	import Logo from '#lib/ui/Logo.svelte';
	import EstadoPastilla from '#lib/cuenta/EstadoPastilla.svelte';
	import { estadoDeAcceso, iniciales, textoSobre } from '#lib/cuenta/presentacion.ts';

	let { data } = $props();
	const p = $derived(data.persona);
	const duenia = $derived(data.negocios.some((n) => n.rol === 'duenio'));
	const primerNombre = $derived(p.nombre.trim().split(/\s+/)[0] ?? '');
</script>

<svelte:head><title>Mi cuenta · Tengo Libre</title></svelte:head>

<div class="cuenta">
	<main class="ancho">
		<a href="/" class="inicio" aria-label="Tengo Libre, inicio"><Logo alto={24} /></a>

		<header class="persona" aria-label="Tus datos">
			{#if p.foto}
				<img class="avatar" src={p.foto} alt="" width="56" height="56" />
			{:else}
				<span class="avatar" aria-hidden="true">{iniciales(p.nombre || p.email || '?')}</span>
			{/if}
			<span class="quien">
				<h1>{primerNombre ? `Hola, ${primerNombre}` : 'Hola'}</h1>
				<span class="mail tl-num">{p.email}</span>
			</span>
			<form method="POST" action="/salir"><button class="salir">Salir</button></form>
		</header>

		<section aria-labelledby="tus-negocios" class="seccion">
			<h2 id="tus-negocios" class="tl-rotulo rotulo">Tus negocios</h2>
			{#if data.negocios.length === 0}
				<p class="vacio">Todavía no tenés un negocio. Creá el tuyo y probalo 7&nbsp;días gratis.</p>
			{/if}
			<ul class="negocios">
				{#each data.negocios as n, i (n.id)}
					{@const estado = estadoDeAcceso(n.acceso, n.rol)}
					<li class="negocio" style:animation-delay="{Math.min(i, 6) * 50}ms">
						<div class="cabeza" style:background-color={n.color} style:color={textoSobre(n.color)}>
							{#if n.logo}
								<img class="cuadro" src={n.logo} alt="" width="40" height="40" />
							{:else}
								<span class="cuadro" style:color={n.color} aria-hidden="true">{iniciales(n.nombre, 1)}</span>
							{/if}
							<span class="nombre">
								<b>{n.nombre}</b>
								<span>{ROLES[n.rol] ?? n.rol}</span>
							</span>
						</div>
						<div class="cuerpo">
							<span class="estados">
								{#if estado}<EstadoPastilla {estado} />{/if}
								{#if n.id === data.elegido}<EstadoPastilla estado={{ texto: 'El último que abriste', icono: 'history', tono: 'apagado' }} />{/if}
							</span>
							{#if estado?.detalle}<p class="detalle">{estado.detalle}</p>{/if}
							<a class="slug tl-num" href={`/${n.slug}`} target="_blank" rel="noopener">/{n.slug}</a>
							<form method="POST" action="/panel/negocio" class="entrar">
								<input type="hidden" name="id" value={n.id} />
								<button aria-label={`Entrar a ${n.nombre}`}>Entrar <Icono nombre="arrow_forward" tam={18} /></button>
							</form>
						</div>
					</li>
				{/each}
				<li class="crear-li">
					<a class="crear" href="/empezar">
						<Icono nombre="add_business" lleno tam={28} />
						<b>Crear otro negocio</b>
						<span>Con 7 días gratis</span>
					</a>
				</li>
			</ul>
			{#if duenia}
				<p class="mas">
					<a class="pastilla" href="/panel/sucursales"><Icono nombre="hub" tam={18} />Sucursales</a>
				</p>
			{/if}
		</section>
	</main>
</div>

<style>
	.cuenta {
		min-height: 100dvh;
		background-color: var(--tl-crema);
		background-image: radial-gradient(rgba(24, 60, 68, 0.08) 1px, transparent 1.3px);
		background-size: 18px 18px;
	}
	.ancho {
		width: min(980px, 100%);
		margin: 0 auto;
		padding: 16px 16px 48px;
		display: flex;
		flex-direction: column;
		gap: 18px;
	}
	.inicio {
		display: inline-flex;
		align-self: flex-start;
		align-items: center;
		min-height: 48px;
	}
	.persona {
		display: flex;
		align-items: center;
		gap: 14px;
	}
	.avatar {
		width: 56px;
		height: 56px;
		border-radius: 50%;
		flex: none;
		object-fit: cover;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		display: grid;
		place-items: center;
		font-size: 18px;
		font-weight: 700;
	}
	.quien {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
	}
	h1 {
		margin: 0;
		font-size: clamp(30px, 4vw, 38px);
		line-height: 1;
		font-weight: 700;
		letter-spacing: -0.05em;
	}
	.mail {
		font-size: 13px;
		color: var(--tl-sub);
		overflow-wrap: anywhere;
	}
	.salir {
		min-height: 48px;
		padding: 0 12px;
		border: 0;
		border-radius: 999px;
		background: none;
		color: #a3241a;
		font: 700 14px var(--tl-font);
		cursor: pointer;
	}
	.salir:hover {
		background: rgba(163, 36, 26, 0.06);
	}
	.seccion {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.rotulo {
		margin: 8px 0 0;
		font-size: 12px;
		color: var(--tl-sub);
	}
	.vacio {
		margin: 0;
		color: var(--tl-sub-2);
	}
	.negocios {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		gap: 14px;
		grid-template-columns: repeat(auto-fill, minmax(min(260px, 100%), 1fr));
	}
	.negocio {
		display: flex;
		flex-direction: column;
		border-radius: 20px;
		overflow: hidden;
		background: var(--tl-papel);
		box-shadow: 0 14px 28px -20px rgba(15, 35, 40, 0.5);
		animation: tl-gin var(--tl-entrar) var(--tl-ease) both;
	}
	.cabeza {
		padding: 16px;
		background-image: var(--tl-grano);
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.cuadro {
		width: 40px;
		height: 40px;
		border-radius: 11px;
		background: #fff;
		display: grid;
		place-items: center;
		font-weight: 700;
		object-fit: cover;
		flex: none;
	}
	.nombre {
		display: flex;
		flex-direction: column;
		min-width: 0;
	}
	.nombre b {
		font-size: 16px;
		overflow-wrap: anywhere;
	}
	.nombre span {
		font-size: 13px;
		opacity: 0.85;
	}
	.cuerpo {
		flex: 1;
		padding: 14px 16px 6px;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.estados {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
	}
	.detalle {
		margin: 0;
		font-size: 13px;
		line-height: 1.4;
		color: var(--tl-sub);
	}
	.slug {
		align-self: flex-start;
		font-size: 13px;
		color: var(--tl-sub);
		overflow-wrap: anywhere;
	}
	.entrar {
		margin-top: auto;
	}
	.entrar button {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		min-height: 48px;
		padding: 0;
		border: 0;
		background: none;
		color: var(--tl-menta-texto);
		font: 700 14px var(--tl-font);
		cursor: pointer;
	}
	.entrar button:hover {
		text-decoration: underline;
		text-underline-offset: 3px;
	}
	.entrar button :global(.icono) {
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.entrar button:hover :global(.icono) {
		transform: translateX(3px);
	}
	.crear-li {
		display: flex;
	}
	.crear {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 8px;
		min-height: 200px;
		padding: 16px;
		border-radius: 20px;
		box-shadow: inset 0 0 0 2px #d8ccb4;
		background: repeating-linear-gradient(-45deg, rgba(60, 204, 144, 0.12) 0 2px, transparent 2px 10px);
		color: var(--tl-tinta);
		text-decoration: none;
		text-align: center;
		transition: transform var(--tl-tocar) var(--tl-ease), box-shadow var(--tl-tocar) var(--tl-ease);
	}
	.crear:hover {
		transform: translateY(-2px);
		box-shadow: inset 0 0 0 2px var(--tl-menta-texto);
	}
	.crear :global(.icono) {
		color: var(--tl-menta-texto);
	}
	.crear b {
		font-size: 15px;
	}
	.crear span {
		font-size: 13px;
		color: var(--tl-sub);
	}
	.mas {
		margin: 0;
	}
	.pastilla {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		min-height: 48px;
		padding: 0 18px;
		border-radius: 999px;
		background: var(--tl-petroleo);
		color: #fff;
		font-size: 14px;
		font-weight: 700;
		text-decoration: none;
		box-shadow: var(--tl-luz-arriba-osc);
	}
	@media (max-width: 719px) {
		.crear {
			min-height: 140px;
		}
	}
	@media (min-width: 720px) {
		.ancho {
			padding-top: 32px;
		}
		.negocios {
			grid-template-columns: repeat(3, minmax(0, 1fr));
		}
	}
</style>
