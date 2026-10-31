<!-- Paso 8: ¡Listo! La página publicada, el link para compartir con su QR, lo hecho y lo que sigue. -->
<script lang="ts">
	import Icono from '#lib/ui/Icono.svelte';
	import Qr from './Qr.svelte';
	import type { Borrador } from './estado.svelte.ts';
	import { leerHorario, minutos } from './agenda.ts';
	import { pesos } from './rubros.ts';
	import { PLANTILLAS, type Cargado } from './tipos.ts';

	let { b, cargado, origen, horas }: { b: Borrador; cargado: Cargado; origen: string; horas: number } = $props();

	const slug = $derived(cargado.negocio.slug);
	const link = $derived(`${origen}/${slug}`);
	const host = $derived(new URL(origen).host);

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
	const whatsapp = $derived(`https://wa.me/?text=${encodeURIComponent(`Reservá tu turno en ${cargado.negocio.nombre}: ${link}`)}`);

	const servicios = $derived(cargado.servicios.filter((s) => s.activo));
	const h = $derived(cargado.profesionales[0] ? leerHorario(cargado.profesionales[0].horario) : null);
	const senas = $derived({ sin_cobro: 'Sin seña', porcentaje: `${cargado.cobro.senaPorcentaje}% al reservar`, fija: `${pesos(cargado.cobro.senaFija)} al reservar`, total: 'Pago total' });
	const hecho = $derived([
		{ ic: 'spa', t: 'Servicios', s: servicios.length ? `${servicios.length} ${servicios.length === 1 ? 'servicio' : 'servicios'} · desde ${pesos(Math.min(...servicios.map((x) => x.precio)))}` : 'Te falta cargarlos' },
		{ ic: 'schedule', t: 'Horario', s: h ? `${h.dias.length} días · ${h.abre} a ${h.cierra}` : 'Te falta cargarlo' },
		{ ic: 'payments', t: 'Seña', s: senas[cargado.cobro.modo] },
		{ ic: 'palette', t: 'Tu página', s: `Plantilla ${PLANTILLAS.find(([v]) => v === cargado.web.estilo.plantilla)?.[1] ?? 'Carta'}` }
	]);

	// El anillo del día: lo abierto en menta y el descanso en ámbar.
	const anillo = $derived.by(() => {
		if (!h) return 'conic-gradient(#3ccc90 0 360deg)';
		const A = minutos(h.abre);
		const tot = minutos(h.cierra) - A || 1;
		const g = (m: number) => ((m / tot) * 360).toFixed(1) + 'deg';
		if (!h.descanso) return 'conic-gradient(#3ccc90 0 360deg)';
		const d = minutos(h.descanso.desde) - A;
		const e = minutos(h.descanso.hasta) - A;
		return `conic-gradient(#3ccc90 0 ${g(d)}, rgba(240,196,106,.85) ${g(d)} ${g(e)}, #3ccc90 ${g(e)} 360deg)`;
	});
	const confeti = $derived(['#3ccc90', '#183c44', '#f0c46a', b.color, '#3ccc90', '#f0a98c', '#183c44', '#3ccc90']);

	const pideMP = $derived(cargado.cobro.modo !== 'sin_cobro' && !cargado.mercadoPago);
</script>

<div class="listo" data-paso="listo">
	<div class="hero tl-petroleo">
		<img class="simbolo" src="/marca/simbolo_una_tinta_blanco.svg" alt="" aria-hidden="true" />
		<div class="hero-t">
			<span class="publicada"><span class="punto"></span>Publicada recién</span>
			<h1>{cargado.negocio.nombre} ya toma <span>turnos.</span></h1>
			<p>Tenés 7 días con todo. Compartí tu link: el primer turno puede entrar&nbsp;hoy.</p>
		</div>
		<span class="anillo" aria-hidden="true">
			<span class="marcas"></span>
			<span class="dia" style:background={anillo}></span>
			<span class="aguja"></span>
			<span class="centro"><b>{horas} h</b><span>libres por semana</span></span>
			{#each confeti as c, i (i)}<span class="conf" style:left="{10 + i * 11}%" style:background={c} style:animation-delay="{i * 80}ms"></span>{/each}
		</span>
	</div>

	<div class="link">
		<div class="link-c">
			<span class="link-t">Tu link para compartir</span>
			<b class="link-url">{host}/<span>{slug}</span></b>
			<div class="acciones">
				<button type="button" class="copiar" class:copiado onclick={copiar}><Icono nombre={copiado ? 'check' : 'content_copy'} tam={20} />{copiado ? 'Copiado' : 'Copiar'}</button>
				<a class="compartir" href={whatsapp} target="_blank" rel="noopener"><img src="/logos/whatsapp.svg" alt="" width="20" height="20" />WhatsApp</a>
				<a class="compartir" href="https://www.instagram.com/accounts/edit/" target="_blank" rel="noopener" onclick={copiar}
					><Icono nombre="photo_camera" lleno tam={20} />Instagram</a
				>
			</div>
		</div>
		<div class="link-qr">
			<span class="qr"><Qr texto={link} etiqueta="Código QR de tu página" /></span>
			<a href="/empezar/cartel" target="_blank" rel="noopener">Bajar cartel A5</a>
		</div>
	</div>

	<div class="hecho">
		{#each hecho as x, i (x.t)}
			<div class="hecho-t" style:animation-delay="{150 + i * 70}ms">
				<span class="hecho-ic"><Icono nombre={x.ic} lleno tam={22} /><span class="picado"></span></span>
				<span class="hecho-c"><b>{x.t}</b><span>{x.s}</span></span>
			</div>
		{/each}
	</div>

	<b class="sigue-t">Lo que sigue</b>
	<div class="sigue">
		{#if pideMP}
			<form method="POST" action="/panel/pagos?/conectar" class="fila mp">
				<span class="fila-ic blanco"><img src="/logos/mercadopago.svg" alt="" width="26" height="26" /></span>
				<span class="fila-c"><b>Conectá Mercado&nbsp;Pago</b><span>Para cobrar la seña. Tarda un&nbsp;minuto.</span></span>
				<button class="fila-b oscuro">Conectar</button>
			</form>
		{/if}
		<a class="fila" href="/{slug}/reservar" target="_blank" rel="noopener">
			<span class="fila-ic"><Icono nombre="event_available" lleno tam={22} /></span>
			<span class="fila-c"><b>Hacé una reserva de&nbsp;prueba</b><span>Reservá vos como si fueras una clienta y mirá qué le&nbsp;llega.</span></span>
			<span class="fila-b">Probar</span>
		</a>
		{#if cargado.profesionales.length > 1}
			<a class="fila" href="/panel/equipo">
				<span class="fila-ic"><Icono nombre="group_add" lleno tam={22} /></span>
				<span class="fila-c"><b>Invitá a tu&nbsp;equipo</b><span>Cada una ve su agenda en su&nbsp;teléfono.</span></span>
				<span class="fila-b">Invitar</span>
			</a>
		{/if}
		<a class="fila" href="/panel/integraciones">
			<span class="fila-ic"><Icono nombre="smart_toy" lleno tam={22} /></span>
			<span class="fila-c"><b>Conectá tu asistente de&nbsp;IA</b><span>Preguntale «¿qué huecos tengo mañana?» desde Claude o&nbsp;ChatGPT.</span></span>
			<span class="fila-b">Ver cómo</span>
		</a>
	</div>
</div>

<style>
	.listo {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	.hero {
		position: relative;
		overflow: hidden;
		border-radius: 32px;
		padding: 34px 36px;
		background-image: var(--tl-grano), radial-gradient(60% 80% at 85% 20%, rgba(60, 204, 144, 0.28), transparent 70%);
		display: grid;
		grid-template-columns: minmax(0, 1fr) 220px;
		gap: 24px;
		align-items: center;
		animation: tl-paso 0.5s both;
	}
	.simbolo {
		position: absolute;
		left: -120px;
		bottom: -140px;
		width: 420px;
		height: 420px;
		opacity: 0.04;
		pointer-events: none;
	}
	.hero-t {
		position: relative;
		display: flex;
		flex-direction: column;
		gap: 14px;
	}
	.publicada {
		display: inline-flex;
		align-self: flex-start;
		align-items: center;
		gap: 8px;
		height: 34px;
		padding: 0 14px 0 10px;
		border-radius: 999px;
		background: rgba(60, 204, 144, 0.16);
		color: #3ccc90;
		font-size: 14px;
		font-weight: 700;
	}
	.punto {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: #3ccc90;
		animation: tl-latido 1.6s infinite;
	}
	h1 {
		margin: 0;
		font-size: 56px;
		line-height: 0.95;
		font-weight: 700;
		letter-spacing: -0.055em;
		text-wrap: balance;
	}
	h1 span {
		color: #3ccc90;
		white-space: nowrap;
	}
	.hero-t p {
		margin: 0;
		font-size: 16px;
		line-height: 1.5;
		color: rgba(255, 255, 255, 0.8);
		max-width: 28em;
		text-wrap: pretty;
	}
	.anillo {
		--r: 210px;
		position: relative;
		justify-self: center;
		width: var(--r);
		height: var(--r);
		display: block;
		animation: tl-ring 1.1s var(--tl-ease) both;
	}
	.marcas {
		position: absolute;
		inset: -10px;
		border-radius: 50%;
		background: repeating-conic-gradient(rgba(255, 255, 255, 0.55) 0 0.9deg, transparent 0.9deg 36deg);
		-webkit-mask: radial-gradient(farthest-side, transparent calc(100% - 5px), #000 calc(100% - 4px));
		mask: radial-gradient(farthest-side, transparent calc(100% - 5px), #000 calc(100% - 4px));
	}
	.dia {
		position: absolute;
		inset: 0;
		border-radius: 50%;
		-webkit-mask: radial-gradient(farthest-side, transparent calc(100% - 20px), #000 calc(100% - 19px));
		mask: radial-gradient(farthest-side, transparent calc(100% - 20px), #000 calc(100% - 19px));
	}
	.aguja {
		position: absolute;
		left: 50%;
		top: -12px;
		width: 10px;
		height: 62px;
		margin-left: -5px;
		border-radius: 5px;
		background: linear-gradient(#3ccc90 0 14px, #fff 14px);
		transform-origin: 50% calc(12px + var(--r) / 2);
		transform: rotate(30deg);
	}
	.centro {
		position: absolute;
		inset: 26%;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		text-align: center;
	}
	.centro b {
		font-size: 30px;
		font-weight: 700;
		letter-spacing: -0.04em;
		font-variant-numeric: tabular-nums;
	}
	.centro span {
		font-size: 12px;
		color: #3ccc90;
		font-weight: 700;
	}
	.conf {
		position: absolute;
		top: 60%;
		width: 8px;
		height: 12px;
		border-radius: 2px;
		animation: tl-conf 1.8s var(--tl-ease) both;
	}
	@keyframes tl-conf {
		0% {
			opacity: 0;
			transform: translateY(0) rotate(0);
		}
		15% {
			opacity: 1;
		}
		100% {
			opacity: 0;
			transform: translateY(-140px) rotate(220deg);
		}
	}
	@keyframes tl-ring {
		from {
			opacity: 0;
			transform: rotate(-160deg) scale(0.8);
		}
		to {
			opacity: 1;
			transform: none;
		}
	}
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
	.hecho {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
		gap: 10px;
	}
	.hecho-t {
		position: relative;
		display: grid;
		grid-template-columns: 56px minmax(0, 1fr);
		min-height: 66px;
		border-radius: 16px;
		background: #f8f2e6;
		animation: tl-paso var(--tl-entrar) both;
	}
	.hecho-ic {
		position: relative;
		display: grid;
		place-items: center;
		border-right: 2px dashed #e2d6bf;
		color: #17805a;
	}
	.picado {
		position: absolute;
		right: 6px;
		top: 7px;
		width: 10px;
		height: 10px;
		border-radius: 50%;
		background: #f4eee2;
		box-shadow: inset 0 1px 2px rgba(15, 35, 40, 0.35);
	}
	.hecho-c {
		display: flex;
		flex-direction: column;
		justify-content: center;
		gap: 2px;
		padding: 8px 14px;
		min-width: 0;
	}
	.hecho-c b {
		font-size: 14px;
	}
	.hecho-c span {
		font-size: 12px;
		color: var(--tl-sub);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.sigue-t {
		font-size: 18px;
		letter-spacing: -0.02em;
		margin-top: 6px;
	}
	.sigue {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.fila {
		display: flex;
		align-items: center;
		gap: 14px;
		min-height: 68px;
		margin: 0;
		padding: 10px 12px 10px 14px;
		border-radius: 18px;
		background: #fffdf9;
		box-shadow: inset 0 0 0 1px #eee5d3;
		color: var(--tl-tinta);
		text-decoration: none;
		transition: transform 0.2s var(--tl-ease);
		animation: tl-paso var(--tl-entrar) both;
	}
	.fila:hover {
		transform: translateX(4px);
	}
	.fila.mp {
		background: #fbe9c2;
		box-shadow: none;
	}
	.fila-ic {
		width: 44px;
		height: 44px;
		flex: none;
		border-radius: 14px;
		background: #e3f6ec;
		color: #17805a;
		display: grid;
		place-items: center;
	}
	.fila-ic.blanco {
		background: #fff;
	}
	.fila-c {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
	}
	.fila-c b {
		font-size: 15px;
	}
	.fila-c span {
		font-size: 13px;
		color: var(--tl-sub);
		text-wrap: pretty;
	}
	.fila-b {
		flex: none;
		min-height: 40px;
		padding: 0 14px;
		border-radius: 999px;
		border: 0;
		background: #f4eee2;
		color: var(--tl-tinta);
		display: grid;
		place-items: center;
		font: 700 13px var(--tl-font);
		white-space: nowrap;
		cursor: pointer;
	}
	.fila-b.oscuro {
		background: #183c44;
		color: #fff;
	}
	@media (max-width: 899px) {
		.hero {
			grid-template-columns: minmax(0, 1fr);
			padding: 26px 20px;
		}
		h1 {
			font-size: 40px;
		}
		.anillo {
			--r: 180px;
		}
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
		.fila-b {
			min-height: 44px;
		}
	}
</style>
