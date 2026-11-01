<!--
	Avisos (la campanita, handoff «Pantallas que faltaban», 1k): lo nuevo arriba con fondo menta, cada
	aviso con su ícono y lleva a donde se resuelve; «Marcar todo como leído» y los mails enviados.
-->
<script lang="ts">
	import { page } from '$app/state';
	import { accion } from '#lib/operacion/ui.ts';
	import { enhance } from '$app/forms';
	import Icono from '#lib/ui/Icono.svelte';
	import Cabecera from '#lib/operacion/Cabecera.svelte';
	import { ATENCION } from '#lib/operacion/secciones.ts';
	import { cuandoAviso, destinoAviso, iconoAviso } from '#lib/panel/avisos.ts';

	let { data, form } = $props();
	const verEnvios = $derived(ATENCION.includes(data.negocio.rol));
</script>

<svelte:head><title>Avisos · Panel</title></svelte:head>

<Cabecera titulo="Lo que pasó" rotulo={data.sinLeer ? `Avisos · ${data.sinLeer} sin leer` : 'Avisos'} />

<div class="hoja">
	{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}
	<p class="nota" aria-live="polite">{data.sinLeer === 0 ? 'Estás al día.' : `${data.sinLeer} sin ver.`} Se muestran los últimos siete&nbsp;días.</p>

	{#if data.notificaciones.length === 0}
		<p class="vacio">No pasó nada esta semana.</p>
	{:else}
		<ul class="lista">
			{#each data.notificaciones as n (n.id)}
				{@const ic = iconoAviso(n.tipo)}
				<li class:nueva={n.nueva}>
					<a href={destinoAviso(n)}>
						<span style:color={ic.color}><Icono nombre={ic.icono} lleno tam={20} /></span>
						<span class="txt">
							<b>{#if n.nueva}<span class="visualmente-oculto">Nuevo: </span>{/if}{n.titulo}</b>
							<span>{n.texto ? `${n.texto} · ` : ''}<span class="tl-num">{cuandoAviso(n.creadoEn)}</span></span>
						</span>
					</a>
				</li>
			{/each}
		</ul>
	{/if}

	{#if data.sinLeer > 0}
		<form method="POST" action={accion(page.url, 'vista')} use:enhance>
			<button class="leido">Marcar todo como leído</button>
		</form>
	{/if}
	{#if verEnvios}<a class="mails" href="/panel/notificaciones/envios"><Icono nombre="outgoing_mail" tam={18} />Mails enviados</a>{/if}
</div>

<style>
	.hoja {
		max-width: 560px;
		padding: 24px;
		border-radius: 24px;
		background-color: var(--tl-crema);
		background-image: var(--tl-grano-osc), radial-gradient(rgba(24, 60, 68, 0.1) 1.2px, transparent 1.4px);
		background-size: auto, 18px 18px;
		box-shadow: var(--tl-sh-alta);
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.nota,
	.vacio {
		margin: 0;
		font-size: 13px;
		color: var(--tl-sub);
	}
	.vacio {
		font-size: 15px;
	}
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.lista a {
		display: flex;
		align-items: flex-start;
		gap: 12px;
		min-height: 56px;
		padding: 12px 14px;
		border-radius: 14px;
		background: var(--tl-papel);
		color: var(--tl-tinta);
		text-decoration: none;
	}
	.nueva a {
		background: #f3fbf7;
	}
	.txt {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}
	.txt b {
		font-size: 15px;
		font-weight: 400;
	}
	.nueva .txt b {
		font-weight: 700;
	}
	.txt > span {
		font-size: 12px;
		color: var(--tl-sub);
		overflow-wrap: anywhere;
	}
	.leido {
		width: 100%;
		height: 52px;
		border: 0;
		border-radius: 999px;
		background: var(--tl-papel);
		font: 700 16px var(--tl-font);
		color: var(--tl-tinta);
		cursor: pointer;
	}
	.mails {
		align-self: center;
		display: inline-flex;
		align-items: center;
		gap: 6px;
		min-height: 44px;
		font-weight: 700;
		color: var(--tl-menta-texto);
		text-decoration: none;
	}
	@media (max-width: 859px) {
		.hoja {
			padding: 16px;
		}
	}
</style>
