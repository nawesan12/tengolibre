<script lang="ts">
	import { enhance } from '$app/forms';
	import { ROLES } from '#lib/formato.ts';
	import Icono from '#lib/ui/Icono.svelte';
	import TarjetaEstado from '#lib/cuenta/TarjetaEstado.svelte';
	import { iniciales, PERMISOS, textoSobre } from '#lib/cuenta/presentacion.ts';

	let { data, form } = $props();
	const i = $derived(data.invitacion);
	const n = $derived(data.negocio);
	const error = $derived(form?.error || data.falla);
	const rol = $derived(ROLES[i.rol] ?? i.rol);
	const dias = $derived(i.venceEn ? Math.max(1, Math.ceil((Date.parse(i.venceEn) - Date.now()) / 86_400_000)) : null);
	const pie = $derived(
		[dias ? `Vence en ${dias} ${dias === 1 ? 'día' : 'días'}` : '', `Para ${i.email}`].filter(Boolean).join(' · ').toLocaleUpperCase('es-AR')
	);
	let enviando = $state(false);
</script>

<svelte:head>
	<title>Invitación a {i.negocio} · Tengo Libre</title>
	<meta name="robots" content="noindex, nofollow" />
	<meta name="referrer" content="no-referrer" />
</svelte:head>

{#snippet cuadro(tam: number)}
	{#if n.logo}
		<img class="cuadro" src={n.logo} alt="" width={tam} height={tam} style:width="{tam}px" style:height="{tam}px" />
	{:else}
		<span class="cuadro" style:width="{tam}px" style:height="{tam}px" style:background={n.color} style:color={textoSobre(n.color)} aria-hidden="true"
			>{iniciales(i.negocio, 1)}</span
		>
	{/if}
{/snippet}

<div class="fondo">
<main class="invitacion">
	{#if i.aceptada}
		<span class="tl-rotulo rotulo">Invitación a {i.negocio}</span>
		<TarjetaEstado icono="task_alt" titulo="Ya la usaste">
			<p>Esta invitación ya se aceptó con <b>{i.email}</b>. Entrá con esa&nbsp;cuenta.</p>
			{#snippet accion()}<a class="pastilla" href="/panel">Ir al panel</a>{/snippet}
		</TarjetaEstado>
	{:else if i.vencida}
		<span class="tl-rotulo rotulo">Invitación a {i.negocio}</span>
		<TarjetaEstado icono="schedule" color="#7a4a00" titulo="Esta invitación venció">
			<p>Las invitaciones duran 7 días. Pedile a {i.invitadoPor ?? 'quien te invitó'} que te mande una nueva desde&nbsp;Equipo.</p>
			{#snippet accion()}
				{#if n.whatsapp}
					<a class="pastilla" href={n.whatsapp} target="_blank" rel="noopener">Escribirle a {i.negocio}</a>
				{:else if n.publicada}
					<a class="pastilla" href={`/${i.slug}`}>Ver su página</a>
				{/if}
			{/snippet}
		</TarjetaEstado>
	{:else}
		<span class="tl-rotulo rotulo">Te invitaron</span>
		<h1 class="visualmente-oculto">Te invitaron a {i.negocio}</h1>
		<article class="talon" aria-label={`Invitación a ${i.negocio}`}>
			<div class="cabeza tl-petroleo">
				<span class="negocio">{@render cuadro(40)}<b>{i.negocio}</b></span>
				<span class="quien">{i.invitadoPor ? `${i.invitadoPor} te invita` : 'Te invitan'} a sumarte&nbsp;como</span>
				<b class="rol">{rol}</b>
			</div>
			<div class="troquel" aria-hidden="true"><span></span></div>
			<ul class="permisos">
				{#each PERMISOS[i.rol] ?? [] as p (p.texto)}
					<li class:no={!p.si}><Icono nombre={p.si ? 'check' : 'remove'} tam={18} />{p.texto}</li>
				{/each}
				<li class="pie tl-num">{pie}</li>
			</ul>
		</article>

		{#if error}
			<p class="error" role="alert"><Icono nombre="error" lleno tam={20} /><span>{error}</span></p>
		{/if}

		<div class="acciones">
			<form
				method="POST"
				use:enhance={() => {
					enviando = true;
					return async ({ update }) => {
						await update();
						enviando = false;
					};
				}}
			>
				<button class="tl-cta aceptar" disabled={enviando}>Aceptar la invitación</button>
			</form>
			{#if error}
				<form method="POST" action="/salir">
					<button class="secundario">Salir y entrar con otra cuenta</button>
				</form>
			{/if}
		</div>
	{/if}
</main>
</div>

<style>
	.fondo {
		min-height: 100dvh;
		background: var(--tl-crema);
	}
	.invitacion {
		width: min(100%, 420px);
		min-height: 100dvh;
		margin: 0 auto;
		padding: 48px 20px 28px;
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	.rotulo {
		font-size: 12px;
		color: var(--tl-menta-texto);
	}
	.talon {
		border-radius: 22px;
		overflow: hidden;
		background: var(--tl-papel);
		box-shadow: 0 24px 40px -24px rgba(15, 35, 40, 0.5);
		animation: tl-gin var(--tl-entrar) var(--tl-ease) both;
	}
	.cabeza {
		padding: 18px 20px;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.negocio {
		display: flex;
		align-items: center;
		gap: 10px;
		font-size: 18px;
	}
	.cuadro {
		border-radius: 11px;
		display: grid;
		place-items: center;
		font-weight: 700;
		object-fit: cover;
		flex: none;
	}
	.quien {
		font-size: 14px;
		color: rgba(255, 255, 255, 0.8);
	}
	.rol {
		font-size: 28px;
		line-height: 1.05;
		letter-spacing: -0.04em;
		color: var(--tl-menta);
	}
	/* Troquel horizontal: mitad petróleo, mitad papel, medialunas del fondo y línea punteada. */
	.troquel {
		height: 18px;
		position: relative;
		background: linear-gradient(var(--tl-petroleo) 0 50%, var(--tl-papel) 50%);
	}
	.troquel::before,
	.troquel::after {
		content: '';
		position: absolute;
		top: 0;
		width: 18px;
		height: 18px;
		border-radius: 50%;
		background: var(--tl-crema);
	}
	.troquel::before {
		left: -9px;
	}
	.troquel::after {
		right: -9px;
	}
	.permisos {
		list-style: none;
		margin: 0;
		padding: 6px 20px 18px;
		display: flex;
		flex-direction: column;
		gap: 8px;
		font-size: 14px;
	}
	.troquel span {
		position: absolute;
		left: 16px;
		right: 16px;
		top: 8px;
		border-top: 2px dashed var(--tl-linea-2);
	}
	.permisos li {
		display: flex;
		gap: 8px;
	}
	.permisos li :global(.icono) {
		color: var(--tl-menta-texto);
	}
	.permisos li.no {
		color: var(--tl-sub);
	}
	.permisos li.no :global(.icono) {
		color: inherit;
	}
	.pie {
		margin-top: 4px;
		font-size: 11px;
		font-weight: 700;
		color: var(--tl-sub);
		overflow-wrap: anywhere;
	}
	.error {
		display: flex;
		gap: 10px;
		margin: 0;
		padding: 14px;
		border-radius: 16px;
		background: #fbe0dc;
		color: #5b1a13;
		font-size: 14px;
		line-height: 1.45;
	}
	.acciones {
		margin-top: auto;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}
	.aceptar {
		width: 100%;
		justify-content: center;
		font-size: 16px;
	}
	.secundario {
		width: 100%;
		height: 48px;
		border: 0;
		background: none;
		color: var(--tl-tinta);
		font: 700 15px var(--tl-font);
		cursor: pointer;
	}
	@media (min-width: 720px) {
		.invitacion {
			min-height: 0;
			padding-top: 64px;
		}
		.acciones {
			margin-top: 8px;
		}
	}
</style>
