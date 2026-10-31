<!-- Cartel A5 para imprimir y dejar en el mostrador: el QR lleva a la página del negocio. -->
<script lang="ts">
	import Icono from '#lib/ui/Icono.svelte';
	import Qr from '#lib/alta/Qr.svelte';
	let { data } = $props();
	const link = $derived(`${data.origen}/${data.negocio.slug}`);
	const host = $derived(new URL(data.origen).host);
</script>

<svelte:head><title>Cartel · {data.negocio.nombre}</title></svelte:head>

<div class="fondo">
	<button class="imprimir" onclick={() => print()}><Icono nombre="print" lleno tam={20} />Imprimir o guardar en PDF</button>
	<article class="cartel" style:--c={data.color}>
		<header class="tl-petroleo">
			<span class="rotulo">RESERVÁ TU TURNO</span>
			<h1>{data.negocio.nombre}</h1>
		</header>
		<div class="qr"><Qr texto={link} etiqueta="Código QR de {data.negocio.nombre}" /></div>
		<p class="escanea"><Icono nombre="photo_camera" lleno tam={22} />Escaneá con la cámara</p>
		<b class="url">{host}/<span>{data.negocio.slug}</span></b>
		<footer><img src="/marca/logotipo_horizontal.svg" alt="Tengo Libre" height="22" /></footer>
	</article>
</div>

<style>
	@page {
		size: A5;
		margin: 0;
	}
	.fondo {
		min-height: 100vh;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 16px;
		padding: 24px 16px;
		background: var(--tl-crema);
	}
	.imprimir {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		min-height: 48px;
		padding: 0 22px;
		border: 0;
		border-radius: 999px;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		font: 900 15px var(--tl-font);
		cursor: pointer;
	}
	.cartel {
		width: min(100%, 148mm);
		aspect-ratio: 148 / 210;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 5mm;
		border-radius: 24px;
		overflow: hidden;
		background: var(--tl-papel);
		box-shadow: var(--tl-sh-alta);
		text-align: center;
	}
	header {
		width: 100%;
		padding: 12mm 10mm 10mm;
		display: flex;
		flex-direction: column;
		gap: 3mm;
		border-bottom: 2px dashed rgba(255, 255, 255, 0.3);
	}
	.rotulo {
		font-size: 12px;
		font-weight: 900;
		letter-spacing: 0.08em;
		color: var(--tl-menta);
	}
	h1 {
		margin: 0;
		font-size: 36px;
		line-height: 0.95;
		font-weight: 900;
		letter-spacing: -0.04em;
		text-wrap: balance;
	}
	.qr {
		width: 62%;
		aspect-ratio: 1;
		padding: 4mm;
		border-radius: 16px;
		background: #fff;
		box-shadow: inset 0 0 0 1px #eee5d3;
	}
	.escanea {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		margin: 0;
		font-size: 16px;
		font-weight: 700;
		color: var(--tl-sub-2);
	}
	.escanea :global(.icono) {
		color: #17805a;
	}
	.url {
		font-size: 18px;
		overflow-wrap: anywhere;
		padding: 0 8mm;
	}
	.url span {
		color: #17805a;
	}
	footer {
		margin-top: auto;
		padding-bottom: 8mm;
	}
	footer img {
		height: 22px;
		width: auto;
	}
	@media print {
		.fondo {
			padding: 0;
			background: none;
		}
		.imprimir {
			display: none;
		}
		.cartel {
			width: 148mm;
			height: 210mm;
			border-radius: 0;
			box-shadow: none;
		}
	}
</style>
