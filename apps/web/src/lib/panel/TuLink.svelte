<!--
	El link de la página para compartir: es lo que trae las reservas, así que va con acciones grandes
	(Copiar, WhatsApp, Instagram, QR) y «Cambiar link» al lado. Lo usan el alta (¡Listo!), Inicio y el
	panel «Tu link» de la barra.
	- grande (por defecto): talón con el link enorme y el QR en el troquel (escritorio).
	- compacto: una franja con el link y las acciones, para cuando el negocio ya tiene movimiento.
	El QR se abre en grande con «QR» (para mostrarlo en el mostrador o bajar el cartel A5).
-->
<script lang="ts">
	import { onMount } from 'svelte';
	import Icono from '#lib/ui/Icono.svelte';
	import Qr from '#lib/alta/Qr.svelte';

	let {
		nombre,
		slug,
		origen,
		titulo = 'Tu link',
		bajada = 'Compartilo y te reservan&nbsp;solos.',
		ver = false,
		compacto = false,
		alCambiar
	}: {
		nombre: string;
		slug: string;
		origen: string;
		titulo?: string;
		/** HTML corto (lleva &nbsp; antes de la última palabra). */
		bajada?: string;
		ver?: boolean;
		compacto?: boolean;
		/** Si viene, aparece «Cambiar link» (sólo quien gestiona el negocio). */
		alCambiar?: () => void;
	} = $props();

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

	// El menú de compartir del teléfono (Telegram, mail…), sólo donde existe.
	let puedeCompartir = $state(false);
	onMount(() => (puedeCompartir = typeof navigator.share === 'function'));
	async function compartir() {
		try {
			await navigator.share({ title: nombre, text: `Reservá tu turno en ${nombre}`, url: link });
		} catch {
			// Cerró el menú sin elegir: no pasa nada.
		}
	}

	let verQr = $state(false);
</script>

<div class="link" class:compacto>
	<div class="link-c">
		<span class="tl-rotulo rot"><Icono nombre="link" tam={16} />{titulo}</span>
		<b class="link-url" data-link>{host}/<span>{slug}</span></b>
		{#if !compacto && bajada}<span class="bajada">{@html bajada}</span>{/if}
		<div class="acciones">
			<button type="button" class="acc principal" class:copiado onclick={copiar}>
				<Icono nombre={copiado ? 'check' : 'content_copy'} tam={22} /><span>{copiado ? 'Copiado' : 'Copiar'}</span>
			</button>
			<a class="acc" href={whatsapp} target="_blank" rel="noopener"><img src="/logos/whatsapp.svg" alt="" width="22" height="22" /><span>WhatsApp</span></a>
			<a class="acc" href="https://www.instagram.com/accounts/edit/" target="_blank" rel="noopener" onclick={copiar}
				><Icono nombre="photo_camera" lleno tam={22} /><span>Instagram</span></a
			>
			<button type="button" class="acc" class:on={verQr} aria-expanded={verQr} onclick={() => (verQr = !verQr)}>
				<Icono nombre="qr_code_2" tam={22} /><span>QR</span>
			</button>
		</div>
		<div class="extras">
			{#if alCambiar}
				<button type="button" class="extra" onclick={alCambiar}><Icono nombre="edit" tam={18} />Cambiar link</button>
			{/if}
			{#if puedeCompartir}
				<button type="button" class="extra" onclick={compartir}><Icono nombre="ios_share" tam={18} />Compartir</button>
			{/if}
			{#if ver}
				<a class="extra" href={`/${slug}`} target="_blank" rel="noopener"><Icono nombre="north_east" tam={18} />Ver mi página</a>
			{/if}
		</div>
		{#if verQr}
			<div class="qr-grande">
				<span class="qr"><Qr texto={link} etiqueta="Código QR de tu página" /></span>
				<span class="qr-t">
					<b>Mostralo en el mostrador</b>
					<span>Lo escanean con la cámara y&nbsp;reservan.</span>
					<a class="extra" href="/empezar/cartel" target="_blank" rel="noopener"><Icono nombre="print" tam={18} />Bajar cartel A5</a>
				</span>
			</div>
		{/if}
	</div>
	{#if !compacto}
		<div class="link-qr" aria-hidden="true">
			<span class="qr"><Qr texto={link} etiqueta="" /></span>
			<span class="tl-rotulo">Escaneá y reservá</span>
		</div>
	{/if}
</div>

<style>
	.link {
		position: relative;
		display: grid;
		grid-template-columns: minmax(0, 1fr) 200px;
		border-radius: 24px;
		background: #fffdf9;
		box-shadow:
			0 14px 28px -22px rgba(15, 35, 40, 0.45),
			inset 0 0 0 2px #3ccc90;
		animation: tl-paso 0.45s var(--tl-ease) both;
	}
	.link-c {
		display: flex;
		flex-direction: column;
		gap: 12px;
		padding: 22px 24px;
		min-width: 0;
	}
	.rot {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		color: #17805a;
	}
	.link-url {
		font-size: 34px;
		line-height: 1.05;
		font-weight: 900;
		letter-spacing: -0.03em;
		overflow-wrap: anywhere;
	}
	.link-url span {
		color: #17805a;
	}
	.bajada {
		font-size: 15px;
		color: var(--tl-sub);
		text-wrap: pretty;
	}
	.acciones {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		gap: 8px;
		margin-top: 4px;
	}
	/* Cuatro acciones del mismo tamaño: ícono arriba, palabra abajo, 64 px para el pulgar. */
	.acc {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 4px;
		min-height: 64px;
		padding: 8px 4px;
		border-radius: 18px;
		border: 0;
		background: #f4eee2;
		color: var(--tl-tinta);
		font: 700 14px var(--tl-font);
		text-decoration: none;
		cursor: pointer;
		transition:
			background 0.2s,
			transform var(--tl-tocar);
	}
	.acc :global(.icono) {
		color: #17805a;
	}
	.acc:hover {
		background: #efe6d4;
	}
	.acc:active {
		transform: scale(0.96);
	}
	.acc.on {
		box-shadow: inset 0 0 0 2px #17805a;
	}
	.acc.principal {
		background: #3ccc90;
		color: #102b31;
	}
	.acc.principal :global(.icono) {
		color: #102b31;
	}
	.acc.principal.copiado {
		background: #17805a;
		color: #fff;
	}
	.acc.principal.copiado :global(.icono) {
		color: #fff;
	}
	.extras {
		display: flex;
		flex-wrap: wrap;
		gap: 4px 16px;
	}
	.extras:empty {
		display: none;
	}
	.extra {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		min-height: 40px;
		padding: 0;
		border: 0;
		background: none;
		color: #17805a;
		font: 700 14px var(--tl-font);
		text-decoration: none;
		cursor: pointer;
	}
	.qr-grande {
		display: flex;
		align-items: center;
		gap: 16px;
		padding: 14px;
		border-radius: 18px;
		background: #f8f2e6;
		animation: tl-paso 0.3s var(--tl-ease) both;
	}
	.qr-grande .qr {
		width: 150px;
		height: 150px;
		flex: none;
	}
	.qr-t {
		display: flex;
		flex-direction: column;
		gap: 4px;
		font-size: 13px;
		color: var(--tl-sub);
	}
	.qr-t b {
		font-size: 15px;
		color: var(--tl-tinta);
	}
	/* El troquel del talón: línea punteada y dos medialunas del color de la página. */
	.link-qr {
		position: relative;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 10px;
		padding: 20px;
		border-left: 2px dashed #e2d6bf;
		color: var(--tl-sub);
	}
	.link-qr::before,
	.link-qr::after {
		content: '';
		position: absolute;
		left: -10px;
		width: 18px;
		height: 18px;
		border-radius: 50%;
		background: var(--tl-crema, #f4eee2);
	}
	.link-qr::before {
		top: -9px;
	}
	.link-qr::after {
		bottom: -9px;
	}
	.qr {
		display: block;
		width: 140px;
		height: 140px;
		border-radius: 14px;
		padding: 8px;
		background: #fff;
		box-shadow: inset 0 0 0 1px #eee5d3;
	}

	/* Compacto: franja de una línea en escritorio. */
	.compacto {
		grid-template-columns: minmax(0, 1fr);
		box-shadow:
			0 14px 28px -22px rgba(15, 35, 40, 0.45),
			inset 0 0 0 1px #e6dcc8;
	}
	.compacto .link-c {
		display: grid;
		grid-template-columns: minmax(0, 1fr) auto;
		align-items: center;
		gap: 4px 16px;
		padding: 16px 20px;
	}
	.compacto .rot {
		grid-column: 1;
	}
	.compacto .link-url {
		grid-column: 1;
		font-size: 22px;
	}
	.compacto .acciones {
		grid-column: 2;
		grid-row: 1 / span 2;
		display: flex;
		margin: 0;
	}
	.compacto .acc {
		flex-direction: row;
		min-height: 48px;
		padding: 0 16px;
		border-radius: 999px;
		gap: 8px;
	}
	.compacto .extras,
	.compacto .qr-grande {
		grid-column: 1 / -1;
	}

	@media (max-width: 899px) {
		.link {
			grid-template-columns: minmax(0, 1fr);
		}
		.link-qr {
			display: none;
		}
		.link-c {
			padding: 18px 16px;
		}
		.link-url {
			font-size: 26px;
		}
		.acc {
			font-size: 13px;
		}
		.compacto .link-c {
			display: flex;
		}
		.compacto .link-url {
			font-size: 20px;
		}
		.compacto .acciones {
			display: grid;
		}
		.compacto .acc {
			flex-direction: column;
			min-height: 60px;
			padding: 6px 2px;
			border-radius: 16px;
			gap: 2px;
		}
		.qr-grande .qr {
			width: 128px;
			height: 128px;
		}
	}
</style>
