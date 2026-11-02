<!--
	El link de la página para compartir (la pieza «Tu link para compartir» del paso ¡Listo! del alta):
	Copiar, WhatsApp, Instagram (copia y abre el perfil), Compartir con el menú del teléfono cuando lo
	hay, el QR y el cartel A5. Lo usan el alta y el Inicio del panel, para que el link esté siempre a mano.
-->
<script lang="ts">
	import { onMount } from 'svelte';
	import Icono from '#lib/ui/Icono.svelte';
	import Qr from '#lib/alta/Qr.svelte';

	let { nombre, slug, origen, titulo = 'Tu link para compartir', ver = false }: { nombre: string; slug: string; origen: string; titulo?: string; ver?: boolean } =
		$props();

	const link = $derived(`${origen}/${slug}`);
	const host = $derived(new URL(origen).host);
	const mensaje = $derived(`Reservá tu turno en ${nombre}: ${link}`);
	const whatsapp = $derived(`https://wa.me/?text=${encodeURIComponent(mensaje)}`);

	let copiado = $state(false);
	async function copiar() {
		try {
			await navigator.clipboard.writeText(link);
			copiado = true;
			setTimeout(() => (copiado = false), 2000);
		} catch {
			// Sin permiso para el portapapeles: queda el link a la vista para copiarlo a mano.
		}
	}

	// El menú de compartir del teléfono (Instagram, Telegram, mail…), sólo donde existe.
	let puedeCompartir = $state(false);
	onMount(() => (puedeCompartir = typeof navigator.share === 'function'));
	async function compartir() {
		try {
			await navigator.share({ title: nombre, text: `Reservá tu turno en ${nombre}`, url: link });
		} catch {
			// Cerró el menú sin elegir: no pasa nada.
		}
	}
</script>

<div class="link">
	<div class="link-c">
		<span class="link-t">{titulo}</span>
		<b class="link-url">{host}/<span>{slug}</span></b>
		<div class="acciones">
			<button type="button" class="copiar" class:copiado onclick={copiar}><Icono nombre={copiado ? 'check' : 'content_copy'} tam={20} />{copiado ? 'Copiado' : 'Copiar'}</button>
			<a class="compartir" href={whatsapp} target="_blank" rel="noopener"><img src="/logos/whatsapp.svg" alt="" width="20" height="20" />WhatsApp</a>
			<a class="compartir" href="https://www.instagram.com/accounts/edit/" target="_blank" rel="noopener" onclick={copiar}
				><Icono nombre="photo_camera" lleno tam={20} />Instagram</a
			>
			{#if puedeCompartir}
				<button type="button" class="compartir" onclick={compartir}><Icono nombre="ios_share" tam={20} />Compartir</button>
			{/if}
			{#if ver}
				<a class="compartir" href={`/${slug}`} target="_blank" rel="noopener"><Icono nombre="north_east" tam={20} />Ver</a>
			{/if}
		</div>
	</div>
	<div class="link-qr">
		<span class="qr"><Qr texto={link} etiqueta="Código QR de tu página" /></span>
		<a href="/empezar/cartel" target="_blank" rel="noopener">Bajar cartel A5</a>
	</div>
</div>

<style>
	.link {
		display: grid;
		grid-template-columns: minmax(0, 1fr) 170px;
		border-radius: 24px;
		overflow: hidden;
		background: #fffdf9;
		box-shadow: 0 24px 44px -30px rgba(15, 35, 40, 0.5);
		animation: tl-paso 0.5s 0.1s both;
	}
	.link-c {
		display: flex;
		flex-direction: column;
		gap: 12px;
		padding: 20px 22px;
		border-right: 2px dashed #e2d6bf;
		min-width: 0;
	}
	.link-t {
		font-size: 13px;
		font-weight: 700;
		color: var(--tl-sub);
	}
	.link-url {
		font-size: 24px;
		font-weight: 700;
		letter-spacing: -0.02em;
		overflow-wrap: anywhere;
	}
	.link-url span {
		color: #17805a;
	}
	.acciones {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}
	.copiar,
	.compartir {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		height: 48px;
		padding: 0 18px;
		border-radius: 999px;
		border: 0;
		font: 700 15px var(--tl-font);
		text-decoration: none;
		cursor: pointer;
		transition:
			background 0.2s,
			transform var(--tl-tocar);
	}
	.copiar {
		background: #3ccc90;
		color: #102b31;
	}
	.copiar.copiado {
		background: #17805a;
		color: #fff;
	}
	.copiar:active {
		transform: scale(0.96);
	}
	.compartir {
		padding: 0 16px;
		border: 0;
		background: #f4eee2;
		color: var(--tl-tinta);
		font-size: 14px;
	}
	.compartir:hover {
		transform: translateY(-2px);
	}
	.compartir :global(.icono) {
		color: #17805a;
	}
	.link-qr {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 8px;
		padding: 18px;
	}
	.qr {
		width: 110px;
		height: 110px;
		border-radius: 14px;
		padding: 8px;
		background: #fff;
		box-shadow: inset 0 0 0 1px #eee5d3;
	}
	.link-qr a {
		min-height: 32px;
		display: inline-flex;
		align-items: center;
		font-size: 13px;
		font-weight: 700;
		color: #17805a;
	}
	@media (max-width: 899px) {
		.link {
			grid-template-columns: minmax(0, 1fr);
		}
		.link-c {
			border-right: 0;
			border-bottom: 2px dashed #e2d6bf;
		}
		.link-url {
			font-size: 19px;
		}
	}
</style>
