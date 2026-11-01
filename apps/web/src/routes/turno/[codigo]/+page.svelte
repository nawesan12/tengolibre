<script lang="ts">
	import { enhance } from '$app/forms';
	import { page } from '$app/state';
	import { DIAS, ESTADOS, dia, fecha, hora, pesos } from '#lib/formato.ts';
	import Icono from '#lib/ui/Icono.svelte';
	import Horarios from '#lib/publico/Horarios.svelte';
	import TiraDias from '#lib/publico/TiraDias.svelte';
	import { DIAS3, diaCorto, diaTexto, mesCorto } from '#lib/publico/calendario.ts';
	import { completar, tema, variables } from '#lib/publico/estilo.ts';
	import { compartirWhatsapp, duracion, linkMapa, linkWhatsapp, montoSena, sinViuda } from '#lib/publico/textos.ts';

	let { data, form } = $props();
	const t = $derived(data.turno);
	const n = $derived(t.negocio);
	const z = $derived(n.zonaHoraria);
	const base = $derived(`/turno/${t.codigo}`);
	const tm = $derived(tema(completar(data.web?.estilo), data.web?.colorPrimario, data.web?.colorAcento));
	const vars = $derived(variables(tm));
	const activo = $derived(['pendiente', 'confirmado', 'pendiente_pago'].includes(t.estado));
	const cancelado = $derived(t.estado === 'cancelado');
	const cambiado = $derived(page.url.searchParams.has('cambiado'));
	const f = $derived(dia(t.inicio, z));
	const minutos = $derived(Math.round((Date.parse(t.fin) - Date.parse(t.inicio)) / 60000));
	const mapa = $derived(linkMapa(null, null, [n.direccion, n.ciudad].filter(Boolean).join(', ')));
	const sena = $derived(t.sena?.estado === 'aprobado' ? t.sena.monto : 0);
	const falta = $derived(t.estado === 'pendiente_pago' ? (t.sena?.monto ?? montoSena(t.precio, data.cobro)) : 0);
	const limite = $derived(new Date(Date.parse(t.inicio) - t.cancelarHastaHoras * 3_600_000));
	const limiteTexto = $derived(`${DIAS3[new Date(dia(limite, z) + 'T12:00:00Z').getUTCDay()]} ${Number(dia(limite, z).slice(8))}, ${hora(limite, z)}`);
	const cuandoCorto = $derived(`${diaTexto(f)}, ${hora(t.inicio, z)}`);
	const linkReferido = $derived(
		data.referido?.codigo ? `${page.url.origin}/${n.slug}/reservar?ref=${encodeURIComponent(data.referido.codigo)}` : ''
	);
	const contacto = $derived(n.whatsapp ? linkWhatsapp(n.whatsapp) : n.telefono ? `tel:${n.telefono.replace(/[^\d+]/g, '')}` : '');
	const reservarDeNuevo = $derived(
		t.servicioId ? `/${n.slug}/reservar?servicio=${t.servicioId}&profesional=${t.profesionalId || 'cualquiera'}` : `/${n.slug}/reservar`
	);
	const mensajesPago: Record<string, string> = {
		ok: '¡Recibimos tu pago! En unos minutos queda confirmado el turno.',
		pendiente: 'Tu pago está en proceso. Te avisamos cuando se acredite.'
	};
	const ICONOS: Record<string, string> = {
		confirmado: 'check_circle',
		pendiente: 'schedule',
		pendiente_pago: 'hourglass_top',
		cancelado: 'cancel',
		atendido: 'task_alt',
		ausente: 'person_off'
	};
	const sello = $derived(cancelado ? 'CANCELADO' : sena > 0 ? 'SEÑA OK' : t.conMembresia ? 'MEMBRESÍA' : '');
	const minutosQuedan = $derived(t.venceEn ? Math.max(0, Math.round((Date.parse(t.venceEn) - Date.now()) / 60000)) : null);
	let nueva = $state('');
	let copiado = $state(false);
	async function compartir(e: MouseEvent) {
		e.preventDefault();
		const texto = `Reservá en ${n.nombre} con mi código ${data.referido?.codigo}`;
		try {
			if (navigator.share) await navigator.share({ title: n.nombre, text: texto, url: linkReferido });
			else {
				await navigator.clipboard.writeText(linkReferido);
				copiado = true;
			}
		} catch {
			/* la persona cerró el diálogo */
		}
	}
</script>

<svelte:head>
	<title>Tu turno en {n.nombre} · Tengo Libre</title>
	<meta name="robots" content="noindex, nofollow" />
	<meta name="referrer" content="no-referrer" />
	<meta name="theme-color" content={tm.bg} />
</svelte:head>

{#snippet logo(tam: number)}
	{#if n.logo}<img class="logo" src={n.logo} alt="" width={tam} height={tam} />{:else}<span class="logo inicial" aria-hidden="true">{n.nombre.trim().slice(0, 1).toUpperCase()}</span>{/if}
{/snippet}

<div class="fondo" style={vars}>
	<main class="turno">
		<h1 class="visualmente-oculto">{cancelado ? 'Turno cancelado' : `Tu turno en ${n.nombre}`}</h1>

		{#if data.nuevo && activo && t.estado !== 'pendiente_pago'}
			<p class="aviso-w tinte" role="status"><Icono nombre="celebration" lleno /><span>¡Listo! Guardá este link: desde acá lo cambiás o lo&nbsp;cancelás.</span></p>
		{/if}
		{#if cambiado && activo}<p class="aviso-w tinte" role="status"><Icono nombre="event_repeat" lleno /><span>Cambiamos tu turno al nuevo&nbsp;horario.</span></p>{/if}
		{#if data.pago && mensajesPago[data.pago]}
			<p class="aviso-w tinte" role="status"><Icono nombre="paid" lleno /><span>{mensajesPago[data.pago]}</span></p>
		{/if}
		{#if form && 'confirmado' in form}<p class="aviso-w tinte" role="status"><Icono nombre="check_circle" lleno /><span>¡Gracias! Le avisamos a {n.nombre} que&nbsp;venís.</span></p>{/if}
		{#if form?.error}<p class="aviso-w rojo" role="alert"><Icono nombre="error" lleno /><span>{form.error}</span></p>{/if}

		{#if t.estado === 'pendiente_pago'}
			{#if data.pago === 'error'}
				<section class="estado" aria-labelledby="t-pago">
					<span class="kicker ambar tl-num">ESPERANDO PAGO</span>
					<h2 id="t-pago">El pago no se&nbsp;completó</h2>
					<p class="texto">
						No se te cobró nada. Tu lugar del {cuandoCorto} sigue guardado{t.venceEn ? ` hasta las ${hora(t.venceEn, z)}` : ''}.
					</p>
					<div class="talon-chico ambar">
						<b class="tl-num">{hora(t.inicio, z)}</b>
						<span class="tc-medio"><b>{t.servicio}</b>{#if falta}<span class="tl-num">Seña {pesos(falta)}</span>{/if}</span>
						{#if minutosQuedan !== null}<span class="tc-fin tl-num">{minutosQuedan} MIN</span>{/if}
					</div>
					<form method="POST" action="?/pagar" use:enhance class="pila">
						<button class="cta mp">Intentar de nuevo</button>
					</form>
					{#if data.puedeCambiar}<a class="cta-2" href={`${base}?cambiar=1`}>Elegir otro horario</a>{/if}
				</section>
			{:else}
				<section class="estado mp-bloque" aria-labelledby="t-pago">
					<span class="mp-logo"><img src="/logos/mercadopago.svg" alt="" width="38" height="38" /></span>
					<h2 id="t-pago" class="grande">
						{#if falta}Falta la seña de <span class="tl-num">{pesos(falta)}</span>{:else}Falta pagar la&nbsp;seña{/if}
					</h2>
					<p class="texto">
						Te guardamos el lugar del <b>{cuandoCorto}</b>{t.venceEn ? ` hasta las ${hora(t.venceEn, z)}` : ' un rato'}. Pagás en Mercado Pago y volvés&nbsp;sola.
					</p>
					{#if falta}
						<div class="renglon chip"><span>Seña · {t.servicio}</span><b class="tl-num">{pesos(falta)}</b></div>
					{/if}
					<form method="POST" action="?/pagar" use:enhance class="pila">
						<button class="cta mp">Pagar con Mercado Pago</button>
					</form>
				</section>
			{/if}
		{/if}

		{#if t.estado === 'atendido' && data.repetir && data.repetirElegido}
			{@const r = data.repetir}
			{@const e = data.repetirElegido}
			<section class="repetir" aria-labelledby="t-repetir">
				<span class="atendido"><Icono nombre="done_all" tam={15} lleno />Atendido · {diaTexto(f)}</span>
				<h2 id="t-repetir" class="grande">¿Repetimos en {r.semanas} {r.semanas === 1 ? 'semana' : 'semanas'}?</h2>
				<div class="talon-hueco grande-h">
					<span class="th-cuando"><span class="tl-num">{diaCorto(e.fecha)} {mesCorto(e.fecha)}</span><b class="tl-num">{e.hora}</b></span>
					<span class="th-datos">
						<b>{r.servicio.nombre}</b>
						<span class="sub">Con {r.profesional.nombre} · {pesos(r.servicio.precio)}</span>
						<span class="nota-marca">{e.mismoHorario ? (e.fecha.length && new Date(e.fecha + 'T12:00:00Z').getUTCDay() === new Date(f + 'T12:00:00Z').getUTCDay() ? 'Mismo día y hora' : 'Misma hora') : 'La hora más parecida'}</span>
					</span>
				</div>
				{#if data.opcionesRepetir.length > 1}
					<div class="fechas" role="group" aria-label="Otras fechas">
						{#each data.opcionesRepetir as o (o.inicio)}
							<a class="fecha-op tl-num" href={`${base}?repetir=${o.fecha}`} data-sveltekit-reset="false" aria-current={o.fecha === e.fecha ? 'true' : undefined}>{diaCorto(o.fecha)}</a>
						{/each}
					</div>
				{/if}
				<span class="chico sub">
					Tus datos ya los tenemos.{#if r.pago?.monto}{' '}{r.pago.modo === 'total' ? 'Se paga por adelantado' : 'La seña es la misma'}: <span class="tl-num">{pesos(r.pago.monto)}</span>.{/if}
				</span>
				<form method="POST" action="?/repetir" use:enhance class="pila">
					<input type="hidden" name="inicio" value={e.inicio} />
					<button class="cta">{r.pago?.monto ? (r.pago.modo === 'total' ? 'Reservar y pagar' : 'Reservar y pagar seña') : 'Reservar'}</button>
				</form>
				<a class="cta-2" href={reservarDeNuevo}>Elegir otro día u hora</a>
			</section>
		{/if}

		<!-- El talón del turno -->
		<section class="talon" class:apagado={cancelado} aria-label="Detalle del turno">
			{#if sello}<span class="sello tl-num" aria-hidden="true">{sello}</span>{/if}
			<div class="talon-arriba">
				<div class="talon-cab">
					{@render logo(34)}
					<b>{n.nombre}</b>
					<span class="estado-pastilla"><Icono nombre={ICONOS[t.estado] ?? 'info'} tam={15} lleno />{ESTADOS[t.estado] ?? t.estado}</span>
				</div>
				<div class="cuando">
					<span class="tl-num">{diaCorto(f)} · {mesCorto(f)}</span>
					<b class="hora-turno tl-num" class:tachado={cancelado}>{hora(t.inicio, z)}</b>
				</div>
			</div>
			<div class="troquel" aria-hidden="true"><span class="izq"></span><span class="der"></span><span class="linea"></span></div>
			<div class="talon-abajo">
				<div class="renglon"><span>{t.servicio}</span><span class="tl-num">{duracion(minutos)} · hasta las {hora(t.fin, z)}</span></div>
				<div class="renglon"><span>{t.profesional ? `Con ${t.profesional}` : 'Con quien esté libre'}</span>{#if n.direccion}<span class="sub">{n.direccion}</span>{/if}</div>
				<div class="renglon raya">
					{#if sena > 0}
						<span class="con-icono ok"><Icono nombre="paid" tam={18} lleno />Seña pagada</span><span class="tl-num">{pesos(sena)}</span>
					{:else if t.conMembresia}
						<span class="con-icono ok"><Icono nombre="card_membership" tam={18} lleno />Con tu membresía</span><span class="tl-num">1 visita</span>
					{:else if falta}
						<span class="con-icono ambar-txt"><Icono nombre="hourglass_top" tam={18} lleno />Falta la seña</span><span class="tl-num">{pesos(falta)}</span>
					{:else}
						<span class="con-icono"><Icono nombre="payments" tam={18} lleno />Precio</span><span class="tl-num">{pesos(t.precio)}</span>
					{/if}
				</div>
				{#if sena > 0 && t.precio > sena}
					<div class="renglon"><span class="sub">En el local</span><b class="tl-num">{pesos(t.precio - sena)}</b></div>
				{/if}
				<div class="renglon raya codigo">
					{#if t.referencia}
						<b class="tl-num">{t.referencia}</b><span class="sub chico">mostralo al llegar</span>
					{:else}
						<span>A nombre de <b>{t.cliente.nombre}</b></span>
					{/if}
				</div>
			</div>
		</section>

		{#if activo}
			{#if t.estado === 'pendiente'}
				<form method="POST" action="?/confirmar" use:enhance class="pila">
					<button class="cta">Confirmo que voy</button>
				</form>
			{:else if t.confirmadoEn && (t.confirmadoPor === 'web' || t.confirmadoPor === 'whatsapp')}
				<span class="confirmado"><Icono nombre="check_circle" tam={18} lleno />Confirmaste{t.confirmadoPor === 'whatsapp' ? ' por WhatsApp' : ''} · <span class="tl-num">{hora(t.confirmadoEn, z)}</span></span>
			{/if}
			<div class="dos">
				<a class="boton-w" href={`${base}/turno.ics`} download="turno.ics"><Icono nombre="calendar_add_on" tam={20} lleno />Agendar</a>
				{#if mapa}<a class="boton-w" href={mapa} target="_blank" rel="noopener"><Icono nombre="directions" tam={20} lleno />Cómo llegar</a>{/if}
			</div>

			{#if data.puedeCambiar}
				{#if data.cambiar}
					<section class="caja" aria-labelledby="t-cambio">
						<h2 id="t-cambio">¿Para cuándo lo&nbsp;pasamos?</h2>
						<TiraDias dias={data.dias} elegida={data.fecha} chica href={(x) => `${base}?cambiar=1&fecha=${x}`} />
						{#if data.fecha && data.horarios.length}
							<form method="POST" action="?/reprogramar" use:enhance class="pila">
								<fieldset>
									<legend class="visualmente-oculto">Nuevo horario del {data.fecha}</legend>
									<Horarios horarios={data.horarios} nombre="inicio" chica bind:elegida={nueva} />
								</fieldset>
								<span class="chico sub">{sinViuda(`${sena > 0 ? 'La seña pasa al turno nuevo. ' : ''}Mismo servicio${t.profesional ? ` y con ${t.profesional}` : ''}.`)}</span>
								<div class="fila">
									<button class="cta">{nueva ? `Cambiar a las ${hora(nueva, z)}` : 'Cambiar a este horario'}</button>
									<a class="cta-texto" href={base}>No</a>
								</div>
							</form>
						{:else if data.fecha}
							<p class="texto">Ese día no quedan horarios. Probá con&nbsp;otro.</p>
							<a class="cta-texto" href={base}>No cambiar</a>
						{:else}
							<p class="texto">No quedan horarios libres en los próximos&nbsp;días.</p>
							<a class="cta-texto" href={base}>No cambiar</a>
						{/if}
					</section>
				{:else if data.cancelar}
					<section class="caja peligro" aria-labelledby="t-cancelar">
						<h2 id="t-cancelar">¿Cancelás el&nbsp;turno?</h2>
						<p class="texto">Estás a tiempo. El horario queda libre para otra&nbsp;persona.</p>
						<form method="POST" action="?/cancelar" use:enhance class="fila">
							<button class="cta rojo">Sí, cancelar</button>
							<a class="cta-texto" href={base}>No, lo mantengo</a>
						</form>
					</section>
				{:else}
					<div class="fila">
						<a class="boton-w contorno ancho" href={`${base}?cambiar=1`}>Cambiar día u hora</a>
						<a class="cta-texto rojo-txt" href={`${base}?cancelar=1`}>Cancelar</a>
					</div>
					<span class="chico sub centro">Cambios y cancelaciones hasta el <span class="tl-num">{limiteTexto}</span></span>
				{/if}
			{:else}
				<section class="estado" aria-labelledby="t-plazo">
					<span class="kicker tl-num">{f === dia(new Date(), z) ? 'HOY · ' : ''}{diaCorto(f)} · {hora(t.inicio, z)}</span>
					<h2 id="t-plazo">Ya no se puede cambiar desde&nbsp;acá</h2>
					<p class="texto">
						El plazo era hasta {t.cancelarHastaHoras}&nbsp;h antes. Si te surgió algo, escribile a {n.nombre}: lo ve&nbsp;enseguida.
					</p>
					{#if contacto}<a class="cta" href={contacto} target="_blank" rel="noopener">Escribir a {n.nombre}</a>{/if}
				</section>
			{/if}
		{:else if cancelado}
			<div class="estado">
				<p class="texto">Cancelaste el turno. Le avisamos a {n.nombre}.</p>
				<a class="cta" href={`/${n.slug}/reservar`}>Sacar otro turno</a>
			</div>
		{:else if t.estado === 'ausente'}
			<section class="estado" aria-labelledby="t-ausente">
				<span class="kicker rojo-txt tl-num">{diaCorto(f)} · {hora(t.inicio, z)}</span>
				<h2 id="t-ausente">Te esperamos y no&nbsp;llegaste</h2>
				<p class="texto">
					{sena > 0 ? `La seña de ${pesos(sena)} quedó para ${n.nombre}, como dice la política. ` : ''}¿Lo volvemos a&nbsp;agendar?
				</p>
				<a class="cta" href={reservarDeNuevo}>Reservar de nuevo</a>
			</section>
		{:else if !data.repetir}
			<a class="cta" href={`/${n.slug}/reservar`}>Sacar otro turno</a>
		{/if}

		{#if data.sugerencias.length && t.servicioId && !data.repetir}
			<section class="proximo" aria-labelledby="t-sug">
				<h2 id="t-sug">Tu próximo turno</h2>
				{#each data.sugerencias.slice(0, 3) as s, i (i)}
					<a
						class="talon-hueco"
						href={`/${n.slug}/reservar?servicio=${t.servicioId}&profesional=${s.profesionalId || 'cualquiera'}&fecha=${s.proxima}`}
						aria-label={`Reservar el ${fecha(s.proxima + 'T12:00:00Z', 'UTC')} a las ${s.hora}`}
					>
						<span class="th-cuando"><span class="tl-num">{diaCorto(s.proxima)} {mesCorto(s.proxima)}</span><b class="tl-num">{s.hora}</b></span>
						<span class="th-texto"><span>{s.motivo || `${DIAS[s.dia]} ${s.hora}`}{s.profesional ? ` · con ${s.profesional}` : ''}</span><b>Reservar</b></span>
					</a>
				{/each}
			</section>
		{/if}

		{#if data.referido?.programaActivo}
			<section class="recomenda tl-petroleo" aria-labelledby="t-ref">
				<span class="ref-cab"><Icono nombre="redeem" lleno /><h2 id="t-ref">Recomendá {n.nombre}</h2></span>
				{#if data.referido.codigo}
					<span class="ref-texto">Cuando alguien reserva con tu código, te llega un cupón de&nbsp;descuento.</span>
					<div class="ref-codigo">
						<span class="tl-num">{data.referido.codigo}</span>
						<a class="ref-boton" href={compartirWhatsapp(`Reservá en ${n.nombre} con mi código ${data.referido.codigo}: ${linkReferido}`)} target="_blank" rel="noopener" onclick={compartir}
							>{copiado ? 'Copiado' : 'Compartir'}</a
						>
					</div>
					<label class="visualmente-oculto" for="link-ref">Link para compartir</label>
					<input id="link-ref" class="visualmente-oculto" readonly value={linkReferido} tabindex="-1" />
					{#if data.referido.referidos > 0}
						<span class="ref-texto">Ya reservaron {data.referido.referidos} con tu&nbsp;código.</span>
					{/if}
					{#each data.referido.recompensas.filter((c) => c.activo) as c (c.id)}
						<span class="ref-premio"><Icono nombre="sell" tam={18} lleno />Tu premio: cupón <b class="tl-num">{c.codigo}</b> ({c.tipo === 'porcentaje' ? `${c.valor}%` : pesos(c.valor)} de&nbsp;descuento)</span>
					{/each}
				{:else}
					<span class="ref-texto">Pedí tu código y compartilo: cuando alguien reserva con él, te ganás un&nbsp;descuento.</span>
					<form method="POST" action="?/referido" use:enhance>
						<button class="ref-boton">Quiero mi código</button>
					</form>
				{/if}
			</section>
		{/if}

		<a class="hecho" href={`/${n.slug}`}>Ver {n.nombre}</a>
	</main>
</div>

<style>
	.fondo {
		min-height: 100vh;
		background: var(--w-bg);
		color: var(--w-ink);
	}
	.turno {
		max-width: 460px;
		margin: 0 auto;
		padding: 24px 18px 48px;
		display: flex;
		flex-direction: column;
		gap: 14px;
	}
	h2 {
		margin: 0;
		font-size: 28px;
		line-height: 1.02;
		font-weight: var(--w-tw);
		letter-spacing: -0.045em;
	}
	.sub {
		color: var(--w-sub);
	}
	.chico {
		font-size: 13px;
	}
	.centro {
		text-align: center;
		margin-top: -4px;
	}
	.pila {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.fila {
		display: flex;
		gap: 8px;
		align-items: center;
	}
	.texto {
		margin: 0;
		font-size: 15px;
		line-height: 1.5;
		color: var(--w-sub2);
		text-wrap: pretty;
	}
	.aviso-w {
		display: flex;
		align-items: center;
		gap: 10px;
		margin: 0;
		padding: 12px 14px;
		border-radius: var(--w-sr);
		font-size: 14px;
		line-height: 1.4;
	}
	.aviso-w.tinte {
		background: var(--w-tint);
	}
	.aviso-w.tinte :global(.icono) {
		color: var(--w-marca);
	}
	.aviso-w.rojo {
		background: #fbe0dc;
		color: #5b1a13;
	}
	.estado {
		display: flex;
		flex-direction: column;
		gap: 14px;
	}
	.kicker {
		font: 700 11px var(--tl-font);
		letter-spacing: 0.08em;
		color: var(--w-ok);
	}
	.kicker.ambar {
		color: #7a4a00;
	}
	.rojo-txt {
		color: #a3241a;
	}
	.ambar-txt {
		color: #7a4a00;
	}
	.mp-bloque {
		padding: 24px 0 8px;
	}
	.mp-logo {
		width: 60px;
		height: 60px;
		border-radius: var(--w-cr);
		background: var(--w-campo);
		box-shadow: inset 0 0 0 1px var(--w-line);
		display: grid;
		place-items: center;
	}
	h2.grande {
		font-size: 32px;
		line-height: 1.05;
		letter-spacing: var(--w-tl);
	}
	.renglon {
		display: flex;
		justify-content: space-between;
		gap: 10px;
	}
	.renglon > :last-child {
		text-align: right;
	}
	.chip {
		padding: 14px 16px;
		border-radius: var(--w-sr);
		background: var(--w-chip);
		font-size: 14px;
	}
	.cta {
		flex: 1;
		display: grid;
		place-items: center;
		min-height: 52px;
		padding: 0 20px;
		border: 0;
		border-radius: var(--w-br);
		background: var(--w-pri);
		color: var(--w-on);
		font: 700 16px var(--tl-font);
		text-align: center;
		text-decoration: none;
		cursor: pointer;
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.cta:active {
		transform: scale(0.97);
	}
	.cta.mp {
		background: #009ee3;
		color: #fff;
		min-height: 54px;
	}
	.cta.rojo {
		background: #a3241a;
		color: #fff;
	}
	.cta-2,
	.cta-texto {
		display: grid;
		place-items: center;
		min-height: 50px;
		padding: 0 16px;
		font-size: 15px;
		font-weight: 700;
		color: var(--w-ink);
		text-decoration: none;
	}
	.cta-2 {
		color: var(--w-sub);
	}
	.cta-texto.rojo-txt {
		color: #a3241a;
	}
	.talon-chico {
		display: grid;
		grid-template-columns: 86px minmax(0, 1fr) auto;
		min-height: 58px;
		border-radius: 14px;
		background:
			radial-gradient(circle at 86px 0, var(--w-bg) 8px, transparent 8.5px),
			radial-gradient(circle at 86px 100%, var(--w-bg) 8px, transparent 8.5px),
			#fbe9c2;
		color: #3a2600;
	}
	.talon-chico > b {
		display: flex;
		align-items: center;
		padding: 0 12px;
		border-right: 2px dashed #e2d6bf;
		font: 700 17px var(--tl-font);
	}
	.tc-medio {
		display: flex;
		flex-direction: column;
		justify-content: center;
		padding: 0 12px;
		min-width: 0;
		font-size: 12px;
	}
	.tc-medio b {
		font-size: 14px;
	}
	.tc-fin {
		display: flex;
		align-items: center;
		padding-right: 12px;
		font: 700 11px var(--tl-font);
		color: #7a4a00;
	}

	/* El talón */
	.talon {
		position: relative;
		border-radius: var(--w-cr);
		overflow: hidden;
		background: var(--w-papel);
		box-shadow: 0 30px 50px -30px rgba(15, 35, 40, 0.5);
		animation: tl-imprime 0.9s var(--tl-ease) both;
	}
	.talon.apagado {
		opacity: 0.6;
	}
	.sello {
		position: absolute;
		right: 18px;
		top: 96px;
		z-index: 2;
		padding: 6px 12px;
		border-radius: 10px;
		border: 3px solid var(--w-on);
		color: var(--w-on);
		font: 700 15px var(--tl-font);
		letter-spacing: 0.12em;
		opacity: 0.9;
		animation: tl-sello 0.6s 0.8s cubic-bezier(0.2, 0.8, 0.2, 1) both;
	}
	.talon-arriba {
		padding: 18px 20px 24px;
		background-color: var(--w-pri);
		background-image: var(--tl-grano);
		color: var(--w-on);
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	.talon-cab {
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.talon-cab > b {
		flex: 1;
		font-size: 15px;
		min-width: 0;
	}
	.logo {
		width: 34px;
		height: 34px;
		border-radius: 50%;
		object-fit: cover;
		flex: none;
	}
	.inicial {
		display: grid;
		place-items: center;
		background: var(--w-on);
		color: var(--w-pri);
		font-weight: 700;
	}
	.estado-pastilla {
		display: inline-flex;
		align-items: center;
		gap: 5px;
		height: 28px;
		padding: 0 11px 0 8px;
		border-radius: var(--w-br);
		background: var(--w-on-a);
		font-size: 12px;
		font-weight: 700;
		white-space: nowrap;
	}
	.cuando {
		display: flex;
		flex-direction: column;
	}
	.cuando span {
		font: 400 13px var(--tl-font);
		letter-spacing: 0.08em;
		opacity: 0.85;
	}
	.hora-turno {
		font: 700 72px var(--tl-font);
		letter-spacing: -0.07em;
		line-height: 0.95;
	}
	.tachado {
		text-decoration: line-through;
	}
	.troquel {
		height: 22px;
		position: relative;
		background: linear-gradient(var(--w-pri) 0 50%, var(--w-papel) 50%);
	}
	.troquel .izq,
	.troquel .der {
		position: absolute;
		top: 0;
		width: 22px;
		height: 22px;
		border-radius: 50%;
		background: var(--w-bg);
	}
	.troquel .izq {
		left: -11px;
	}
	.troquel .der {
		right: -11px;
	}
	.troquel .linea {
		position: absolute;
		left: 18px;
		right: 18px;
		top: 10px;
		border-top: 2px dashed var(--w-line2);
	}
	.talon-abajo {
		padding: 6px 20px 20px;
		display: flex;
		flex-direction: column;
		gap: 12px;
		font-size: 15px;
	}
	.raya {
		padding-top: 12px;
		border-top: 1px solid var(--w-line);
	}
	.con-icono {
		display: flex;
		gap: 6px;
		align-items: center;
	}
	.ok {
		color: var(--w-ok);
	}
	.codigo b.tl-num {
		font: 700 15px var(--tl-font);
		letter-spacing: 0.14em;
	}
	.codigo {
		align-items: center;
	}

	.dos {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 8px;
	}
	.boton-w {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 6px;
		min-height: 50px;
		padding: 0 14px;
		border-radius: var(--w-br);
		background: var(--w-papel);
		box-shadow: inset 0 0 0 1px var(--w-line);
		color: var(--w-ink);
		font-size: 14px;
		font-weight: 700;
		text-decoration: none;
	}
	.boton-w.contorno {
		background: transparent;
		box-shadow: inset 0 0 0 1.5px var(--w-line2);
		font-size: 15px;
	}
	.ancho {
		flex: 1;
	}
	.caja {
		display: flex;
		flex-direction: column;
		gap: 12px;
		padding: 16px;
		border-radius: var(--w-cr);
		background: var(--w-papel);
		box-shadow: inset 0 0 0 1px var(--w-line);
	}
	.caja h2 {
		font-size: 16px;
		letter-spacing: 0;
		font-weight: 700;
	}
	.caja.peligro {
		box-shadow: inset 0 0 0 1.5px #f1c5bf;
	}
	.caja .texto {
		font-size: 14px;
	}
	fieldset {
		border: 0;
		margin: 0;
		padding: 0;
		min-width: 0;
	}

	.repetir {
		display: flex;
		flex-direction: column;
		gap: 14px;
		padding-bottom: 10px;
	}
	.atendido {
		display: inline-flex;
		align-self: flex-start;
		align-items: center;
		gap: 5px;
		height: 28px;
		padding: 0 11px 0 8px;
		border-radius: 999px;
		background: var(--tl-petroleo);
		color: #fff;
		font-size: 12px;
		font-weight: 700;
	}
	.repetir h2.grande {
		font-size: 30px;
		letter-spacing: -0.045em;
	}
	.grande-h {
		grid-template-columns: 100px minmax(0, 1fr);
		min-height: 96px;
		border-radius: 20px;
		background:
			radial-gradient(circle at 100px 0, var(--w-bg) 9px, transparent 9.5px),
			radial-gradient(circle at 100px 100%, var(--w-bg) 9px, transparent 9.5px),
			repeating-linear-gradient(-45deg, var(--w-raya) 0 2px, transparent 2px 10px),
			var(--w-tint);
		box-shadow: inset 0 0 0 1.5px var(--w-troquel);
	}
	.grande-h .th-cuando {
		padding: 0 16px;
	}
	.grande-h .th-cuando b {
		font-size: 26px;
		letter-spacing: -0.05em;
	}
	.th-datos {
		display: flex;
		flex-direction: column;
		justify-content: center;
		gap: 2px;
		padding: 8px 16px;
		font-size: 14px;
		min-width: 0;
	}
	.nota-marca {
		font-size: 12px;
		font-weight: 700;
		color: var(--w-marca);
	}
	.fechas {
		display: flex;
		gap: 6px;
	}
	.fecha-op {
		flex: 1;
		display: grid;
		place-items: center;
		height: 48px;
		border-radius: 12px;
		box-shadow: inset 0 0 0 1.5px var(--w-line2);
		color: var(--w-ink);
		font: 700 13px var(--tl-font);
		text-decoration: none;
	}
	.fecha-op[aria-current] {
		background: var(--w-pri);
		color: var(--w-on);
		box-shadow: none;
	}
	.confirmado {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 6px;
		font-size: 13px;
		font-weight: 700;
		color: var(--w-ok);
	}

	/* Próximo turno: talón rayado */
	.proximo {
		display: flex;
		flex-direction: column;
		gap: 10px;
		margin-top: 10px;
	}
	.proximo h2 {
		font-size: 17px;
		letter-spacing: 0;
		font-weight: 700;
	}
	.talon-hueco {
		display: grid;
		grid-template-columns: 96px minmax(0, 1fr);
		min-height: 64px;
		border-radius: var(--w-sr);
		background:
			radial-gradient(circle at 96px 0, var(--w-bg) 8px, transparent 8.5px),
			radial-gradient(circle at 96px 100%, var(--w-bg) 8px, transparent 8.5px),
			repeating-linear-gradient(-45deg, var(--w-raya) 0 2px, transparent 2px 10px),
			var(--w-tint);
		color: var(--w-ink);
		text-decoration: none;
	}
	.th-cuando {
		display: flex;
		flex-direction: column;
		justify-content: center;
		padding: 0 14px;
		border-right: 2px dashed var(--w-troquel);
	}
	.th-cuando span {
		font: 400 11px var(--tl-font);
	}
	.th-cuando b {
		font: 700 19px var(--tl-font);
	}
	.th-texto {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 0 14px;
	}
	.th-texto span {
		flex: 1;
		font-size: 13px;
		line-height: 1.35;
	}
	.th-texto b {
		font-size: 14px;
		color: var(--w-marca);
	}

	/* Recomendá */
	.recomenda {
		display: flex;
		flex-direction: column;
		gap: 10px;
		padding: 16px;
		border-radius: var(--w-cr);
	}
	.ref-cab {
		display: flex;
		gap: 8px;
		align-items: center;
	}
	.ref-cab :global(.icono) {
		color: var(--tl-menta);
	}
	.ref-cab h2 {
		font-size: 16px;
		letter-spacing: 0;
		font-weight: 700;
		color: #fff;
	}
	.ref-texto {
		font-size: 13px;
		line-height: 1.45;
		color: rgba(255, 255, 255, 0.8);
	}
	.ref-codigo {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 6px 6px 6px 14px;
		border-radius: var(--w-br);
		background: rgba(255, 255, 255, 0.1);
	}
	.ref-codigo span {
		flex: 1;
		font: 700 15px var(--tl-font);
		letter-spacing: 0.1em;
	}
	.ref-boton {
		display: inline-grid;
		place-items: center;
		min-height: 44px;
		padding: 0 16px;
		border-radius: var(--w-br);
		border: 0;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		font: 700 13px var(--tl-font);
		text-decoration: none;
		cursor: pointer;
	}
	.ref-premio {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 13px;
		padding: 10px 12px;
		border-radius: 12px;
		background: rgba(60, 204, 144, 0.15);
	}
	.hecho {
		align-self: center;
		display: grid;
		place-items: center;
		min-height: 44px;
		padding: 0 14px;
		font-size: 13px;
		font-weight: 700;
		color: var(--w-sub);
	}
	@keyframes tl-imprime {
		from {
			clip-path: inset(0 0 100% 0);
			transform: translateY(-14px);
		}
		to {
			clip-path: inset(0 0 0 0);
			transform: none;
		}
	}
	@keyframes tl-sello {
		0% {
			opacity: 0;
			transform: rotate(-14deg) scale(1.6);
		}
		60% {
			opacity: 1;
			transform: rotate(-8deg) scale(0.95);
		}
		100% {
			opacity: 0.9;
			transform: rotate(-8deg) scale(1);
		}
	}
</style>
