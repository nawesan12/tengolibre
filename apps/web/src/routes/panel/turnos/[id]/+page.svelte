<!--
	El detalle de un turno (handoff «Panel v2», el panel del turno): cabeza petróleo con la hora grande,
	troquel, el cliente con WhatsApp y llamar, la plata, la nota interna, la historia y los avisos, y
	reprogramar. Abajo, las acciones.
-->
<script lang="ts">
	import { enhance } from '$app/forms';
	import { page } from '$app/state';
	import Icono from '#lib/ui/Icono.svelte';
	import { fechaHora, hora, pesos } from '#lib/formato.ts';
	import { nombreDia } from '#lib/operacion/fechas.ts';
	import { ESTILO, ICONO_ACCION, accionPrincipal, accionesRapidas, estadoVisible } from '#lib/operacion/estados.ts';
	import { cuandoCorto } from '#lib/operacion/ui.ts';
	import Caja from '#lib/operacion/Caja.svelte';
	import { iniciales, primerNombre } from '#lib/panel/dia.ts';

	let { data, form } = $props();
	const t = $derived(data.turno);
	let forzar = $state(false);
	let notaCambiada = $state(false);
	let caja = $state(false);
	let enviando = $state(false);

	const puedeActuar = $derived(!data.acceso?.bloqueado);
	const est = $derived(estadoVisible(t.estado, t.inicio, t.fin));
	const e = $derived(ESTILO[est] ?? ESTILO.pendiente);
	const principal = $derived(puedeActuar ? accionPrincipal(t.estado, t.inicio) : null);
	const otras = $derived(puedeActuar ? accionesRapidas(t.estado).filter((a) => a.estado !== principal?.estado) : []);
	const vivo = $derived(t.estado !== 'cancelado' && t.estado !== 'atendido' && t.estado !== 'ausente');
	const wa = $derived(t.cliente.telefono ? `https://wa.me/${t.cliente.telefono.replace(/\D/g, '').replace(/^(?!54)/, '549')}` : '');
	const MOTIVO_L: Record<string, string> = { cambio: 'Pidió cambiarlo', sin_respuesta: 'No contestó el WhatsApp', no_llego: 'No le llegó el WhatsApp' };

	const ORIGENES: Record<string, string> = { web: 'Reservó por la web', panel: 'Lo cargó el equipo', app: 'Reservó por la app' };
	const ESTADO_AVISO: Record<string, string> = { pendiente: 'En cola', enviado: 'Enviado', fallido: 'Falló', descartado: 'Descartado' };
	const COLOR_AVISO: Record<string, string> = { pendiente: '#c9bda4', enviado: '#3ccc90', fallido: '#a3241a', descartado: '#4a5d61' };
	const trabajar = () => {
		enviando = true;
		return async ({ update }: { update: (o?: { reset?: boolean }) => Promise<void> }) => {
			await update({ reset: false });
			enviando = false;
		};
	};
</script>

<svelte:head><title>Turno de {t.cliente.nombre} · Panel</title></svelte:head>

<p class="volver"><a href="/panel/turnos"><Icono nombre="arrow_back" tam={18} />Turnos</a></p>
{#if page.url.searchParams.get('nuevo')}<p class="aviso" role="status"><Icono nombre="check_circle" lleno tam={20} />Listo, el turno quedó&nbsp;reservado.</p>{/if}
{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}

<article class="detalle" aria-label="Detalle del turno">
	<header class="cabeza tl-petroleo">
		<div class="arriba">
			<span class="chip-est"><Icono nombre={e.icono} lleno tam={16} /><span>{e.texto}</span></span>
			<span class="crece"></span>
			<a class="cerrar" href="/panel/turnos" aria-label="Cerrar"><Icono nombre="close" tam={22} /></a>
		</div>
		<div class="hora">
			<b class="tl-num">{hora(t.inicio)}</b>
			<span class="tl-num">→ {hora(t.fin)} · {cuandoCorto(t.inicio, false)}</span>
		</div>
		<span class="srv">{t.servicio} · con {t.profesional || 'cualquiera'}</span>
	</header>
	<div class="troquel" aria-hidden="true"><span></span></div>

	<div class="cuerpo">
		<section class="bloque">
			<div class="cliente">
				<span class="avatar">{iniciales(t.cliente.nombre)}</span>
				<span class="nom">
					<h1>{t.cliente.nombre}</h1>
					<span>{ORIGENES[t.origen] ?? t.origen} · {fechaHora(t.creadoEn)}</span>
				</span>
				{#if t.cliente.id}<a href={`/panel/clientes/${t.cliente.id}`}>Ficha</a>{/if}
			</div>
			<div class="contacto">
				{#if t.cliente.telefono}
					<a class="chip-c tl-num" href={wa} target="_blank" rel="noopener"><img src="/logos/whatsapp.svg" alt="" width="16" height="16" />{t.cliente.telefono}</a>
					<a class="chip-c" href={`tel:${t.cliente.telefono}`}><Icono nombre="call" tam={17} />Llamar</a>
				{/if}
				{#if t.cliente.email}<a class="chip-c" href={`mailto:${t.cliente.email}`}><Icono nombre="mail" tam={17} />{t.cliente.email}</a>{/if}
				{#if !t.cliente.telefono && !t.cliente.email}<span class="suave chico">Sin datos de contacto</span>{/if}
			</div>
			{#if t.llamarDesde && t.estado !== 'cancelado'}
				<div class="llamar" role="status">
					<Icono nombre="call" tam={20} />
					<span><b>Para llamar:</b> {MOTIVO_L[t.llamarMotivo ?? ''] ?? 'pendiente'}.</span>
					{#if puedeActuar}
						<form method="POST" action="?/llamado" use:enhance><button>Ya lo llamé</button></form>
					{/if}
				</div>
			{/if}
		</section>

		<section class="bloque plata">
			{#if data.caja}
				{#each data.caja.lineas as l, i (i)}
					<div class="linea" class:verde={l.monto < 0}><span>{l.concepto}</span><b class="tl-num">{l.monto < 0 ? `− ${pesos(-l.monto)}` : pesos(l.monto)}</b></div>
				{/each}
				{#if data.caja.estado === 'cobrado'}
					<div class="linea verde"><span class="ic"><Icono nombre="paid" tam={18} />Cobrado{data.caja.medio ? ` · ${data.caja.medio}` : ''}</span><b class="tl-num">{pesos(data.caja.total)}</b></div>
				{:else if data.caja.estado === 'debe'}
					<div class="linea ambar"><span class="ic"><Icono nombre="pending" tam={18} />Quedó debiendo</span><b class="tl-num">{pesos(data.caja.total)}</b></div>
				{:else}
					<div class="linea suave-l"><span>Resta cobrar en el local</span><b class="tl-num">{pesos(data.caja.total)}</b></div>
				{/if}
			{:else}
				<div class="linea"><span>Precio</span><b class="tl-num">{pesos(t.precio)}</b></div>
			{/if}
			{#if t.estado === 'pendiente_pago'}
				<div class="linea ambar"><span class="ic"><Icono nombre="hourglass_top" tam={18} />Esperando la seña{t.venceEn ? ` hasta las ${hora(t.venceEn)}` : ''}</span></div>
			{/if}
		</section>

		<section class="bloque">
			<form
				method="POST"
				action="?/nota"
				class="nota"
				use:enhance={() =>
					async ({ update }) => {
						await update({ reset: false });
						notaCambiada = false;
					}}
			>
				<span class="tl-rotulo rot" aria-hidden="true">Nota interna</span>
				<label class="visualmente-oculto" for="nota-turno">Nota</label>
				<textarea id="nota-turno" name="nota" maxlength="500" rows="2" placeholder="Sólo la ve el equipo" oninput={() => (notaCambiada = true)}>{t.nota}</textarea>
				<div class="fila-n">
					<button class="guardar">Guardar nota</button>
					{#if form && 'guardado' in form && form.guardado === 'nota' && !notaCambiada}<span class="listo" role="status"><Icono nombre="check" tam={16} />Guardada</span>{/if}
				</div>
			</form>
		</section>

		<section class="bloque" aria-labelledby="t-historia">
			<h2 id="t-historia" class="tl-rotulo rot">Historia</h2>
			<ul class="historia">
				<li><span class="punto" style:background="#c9bda4"></span><span>{ORIGENES[t.origen] ?? t.origen}</span><span class="cuando tl-num">{fechaHora(t.creadoEn)}</span></li>
				{#if t.confirmadoPor}
					<li><span class="punto" style:background="#3ccc90"></span><span>{t.confirmadoPor === 'whatsapp' ? 'Confirmó por WhatsApp' : t.confirmadoPor === 'web' ? 'Confirmó desde su link' : 'Confirmado por el equipo'}</span><span class="cuando tl-num">{t.confirmadoEn ? fechaHora(t.confirmadoEn) : ''}</span></li>
				{/if}
				{#if t.cambioPedidoEn}
					<li><span class="punto" style:background="#f0c46a"></span><span>Pidió cambiarlo por WhatsApp</span><span class="cuando tl-num">{fechaHora(t.cambioPedidoEn)}</span></li>
				{/if}
				{#each data.avisos ?? [] as a (a.id)}
					<li>
						<span class="punto" style:background={COLOR_AVISO[a.estado] ?? '#183c44'}></span>
						<span>
							Mail «{a.tipo.replaceAll('_', ' ')}» a {a.para} · {ESTADO_AVISO[a.estado] ?? a.estado}
							{#if a.ultimoError}<span class="error-a">· {a.ultimoError}</span>{/if}
							{#if a.estado !== 'pendiente'}
								<form method="POST" action="?/reenviar" use:enhance class="reenviar">
									<input type="hidden" name="id" value={a.id} />
									<button>Reenviar</button>
								</form>
							{/if}
						</span>
						<span class="cuando tl-num">{fechaHora(a.enviadoEn ?? a.creadoEn)}</span>
					</li>
				{/each}
			</ul>
		</section>

		{#if vivo && puedeActuar}
			<section class="bloque" id="reprogramar" aria-labelledby="t-reprogramar">
				<h2 id="t-reprogramar" class="tl-rotulo rot">Reprogramar</h2>
				<form method="GET" class="elegir" data-sveltekit-reset="false" data-sveltekit-noscroll aria-label="Día y profesional para reprogramar">
					<label class="campo">Día <input type="date" name="fecha" value={data.fecha} onchange={(e) => e.currentTarget.form?.requestSubmit()} /></label>
					<label class="campo">
						Profesional
						<select name="profesional" onchange={(e) => e.currentTarget.form?.requestSubmit()}>
							{#each data.profesionales as p (p.id)}<option value={p.id} selected={p.id === data.profesional}>{p.nombre}</option>{/each}
						</select>
					</label>
					<noscript><button class="boton">Ver horarios</button></noscript>
				</form>
				<form method="POST" action="?/reprogramar" use:enhance class="pila-r">
					<input type="hidden" name="profesionalId" value={data.profesional} />
					<input type="hidden" name="fecha" value={data.fecha} />
					{#if !forzar}
						{#if data.errorLibres}
							<p class="aviso error">{data.errorLibres}</p>
						{:else if data.horarios.length === 0}
							<p class="suave chico">No hay horarios libres el {nombreDia(data.fecha)}.</p>
						{:else}
							<fieldset class="horas">
								<legend class="visualmente-oculto">Horario del {nombreDia(data.fecha)}</legend>
								{#each data.horarios as h, i (h.inicio)}
									<label class="h"><input type="radio" name="inicio" value={h.inicio} required checked={i === 0} /><span class="tl-num">{h.hora}</span></label>
								{/each}
							</fieldset>
						{/if}
					{/if}
					<label class="sobre"><input type="checkbox" name="forzar" bind:checked={forzar} />Sobreturno: a una hora que no figura&nbsp;libre</label>
					{#if forzar}<label class="campo">Hora <input type="time" name="hora" required step="300" /></label>{/if}
					<button class="boton oscuro">Reprogramar</button>
				</form>
			</section>
		{/if}
	</div>

	{#if puedeActuar && t.estado !== 'cancelado'}
		<footer class="pie">
			{#if principal}
				{#if principal.estado === 'atendido'}
					<button class="primaria" onclick={() => (caja = true)}><Icono nombre={principal.icono} tam={20} />{principal.texto}</button>
				{:else}
					<form method="POST" action="?/estado" use:enhance={trabajar} class="crece-f">
						<input type="hidden" name="id" value={t.id} />
						<button class="primaria" name="estado" value={principal.estado} disabled={enviando}><Icono nombre={principal.icono} tam={20} />{principal.texto}</button>
					</form>
				{/if}
			{/if}
			<form method="POST" action="?/estado" use:enhance={trabajar} class="otras">
				<input type="hidden" name="id" value={t.id} />
				{#each otras as a (a.estado)}
					{#if a.estado === 'atendido'}
						<button type="button" class="sec" onclick={() => (caja = true)}><Icono nombre={ICONO_ACCION.atendido} tam={18} />{a.texto}</button>
					{:else}
						<button
							class="sec"
							class:peligro={a.estado === 'cancelado'}
							name="estado"
							value={a.estado}
							disabled={enviando}
							onclick={(ev) => {
								if (a.estado === 'cancelado' && !confirm('¿Cancelar este turno? Se le avisa al cliente y queda el hueco.')) ev.preventDefault();
							}}>{a.texto}</button
						>
					{/if}
				{/each}
			</form>
			{#if t.serieId && vivo}
				<form
					method="POST"
					action="?/cancelarSiguientes"
					use:enhance={({ cancel }) => {
						if (!confirm('¿Cancelar este turno y los siguientes de su turno fijo? Se le avisa al cliente.')) cancel();
						return trabajar();
					}}
				>
					<button class="sec peligro"><Icono nombre="event_repeat" tam={18} />Cancelar este y los siguientes</button>
				</form>
			{/if}
		</footer>
	{:else if data.acceso?.bloqueado}
		<footer class="pie lectura"><Icono nombre="lock" tam={18} />Sólo lectura: para cambiar el turno hace falta un&nbsp;plan.</footer>
	{/if}
</article>

{#if caja}<Caja {t} productos={data.productos} alCerrar={() => (caja = false)} />{/if}

<style>
	.volver {
		margin: 0;
	}
	.volver a {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		min-height: 40px;
		font-size: 14px;
		font-weight: 700;
		color: var(--tl-menta-texto);
		text-decoration: none;
	}
	.detalle {
		width: 100%;
		max-width: 520px;
		border-radius: 24px;
		background: var(--tl-papel);
		box-shadow: 0 30px 60px -36px rgba(15, 35, 40, 0.55);
		overflow: hidden;
		display: flex;
		flex-direction: column;
		animation: tl-gin var(--tl-entrar) var(--tl-ease) both;
	}
	.cabeza {
		padding: 18px 20px 22px;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.arriba {
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.chip-est {
		display: inline-flex;
		align-items: center;
		gap: 5px;
		height: 28px;
		padding: 0 10px 0 8px;
		border-radius: 999px;
		background: rgba(255, 255, 255, 0.14);
		font-size: 13px;
		font-weight: 700;
	}
	.chip-est :global(.icono) {
		color: var(--tl-menta);
	}
	.crece {
		flex: 1;
	}
	.cerrar {
		width: 44px;
		height: 44px;
		border-radius: 14px;
		background: rgba(255, 255, 255, 0.1);
		color: #fff;
		display: grid;
		place-items: center;
	}
	.hora {
		display: flex;
		align-items: flex-end;
		gap: 12px;
		flex-wrap: wrap;
	}
	.hora b {
		font-size: 52px;
		font-weight: 700;
		letter-spacing: -0.06em;
		line-height: 0.9;
	}
	.hora span {
		font-size: 13px;
		color: rgba(255, 255, 255, 0.7);
		padding-bottom: 4px;
	}
	.srv {
		font-size: 15px;
		color: rgba(255, 255, 255, 0.85);
	}
	.troquel {
		height: 18px;
		flex: none;
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
	.troquel span {
		position: absolute;
		left: 16px;
		right: 16px;
		top: 8px;
		border-top: 2px dashed var(--tl-linea-2);
	}
	.cuerpo {
		padding: 4px 20px 20px;
		display: flex;
		flex-direction: column;
	}
	.bloque {
		padding: 14px 0 16px;
		border-bottom: 1px solid #f0e8d8;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.bloque:last-child {
		border-bottom: 0;
	}
	.cliente {
		display: flex;
		align-items: center;
		gap: 12px;
	}
	.avatar {
		flex: none;
		width: 44px;
		height: 44px;
		border-radius: 50%;
		background: var(--tl-crema);
		display: grid;
		place-items: center;
		font-size: 14px;
		font-weight: 700;
	}
	.nom {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
	}
	h1 {
		margin: 0;
		font-size: 17px;
		font-weight: 700;
		letter-spacing: -0.01em;
		line-height: 1.3;
	}
	.nom span {
		font-size: 13px;
		color: var(--tl-sub);
	}
	.cliente > a {
		font-size: 14px;
		font-weight: 700;
		color: var(--tl-menta-texto);
		min-height: 40px;
		display: inline-flex;
		align-items: center;
	}
	.contacto {
		display: flex;
		gap: 8px;
		flex-wrap: wrap;
	}
	.chip-c {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		min-height: 40px;
		padding: 0 14px;
		border-radius: 999px;
		background: var(--tl-crema);
		color: var(--tl-tinta);
		font-size: 13px;
		font-weight: 700;
		text-decoration: none;
		overflow-wrap: anywhere;
	}
	.linea {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 10px;
		font-size: 14px;
	}
	.linea b {
		font-size: 16px;
		font-weight: 700;
	}
	.ic {
		display: inline-flex;
		align-items: center;
		gap: 6px;
	}
	.verde {
		color: var(--tl-menta-texto);
	}
	.ambar {
		color: #7a4a00;
	}
	.suave-l {
		color: var(--tl-sub);
	}
	.suave-l b {
		color: var(--tl-tinta);
		font-size: 15px;
	}
	.rot {
		margin: 0;
		font-size: 11px;
		color: var(--tl-sub);
	}
	.nota {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.nota textarea {
		font: 400 16px var(--tl-font);
		padding: 10px 12px;
		border: 0;
		border-radius: 12px;
		background: #f8f3ea;
		resize: vertical;
		min-height: 64px;
	}
	.nota textarea:focus {
		outline: none;
		box-shadow: inset 0 0 0 2px var(--tl-menta);
	}
	.fila-n {
		display: flex;
		align-items: center;
		gap: 12px;
	}
	.guardar {
		min-height: 40px;
		padding: 0 14px;
		border: 0;
		border-radius: 999px;
		background: var(--tl-crema);
		font: 700 13px var(--tl-font);
		cursor: pointer;
	}
	.listo {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		font-size: 13px;
		font-weight: 700;
		color: var(--tl-menta-texto);
	}
	.historia {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.historia li {
		display: grid;
		grid-template-columns: 14px 1fr auto;
		gap: 10px;
		align-items: baseline;
		font-size: 13px;
	}
	.punto {
		width: 8px;
		height: 8px;
		border-radius: 50%;
	}
	.cuando {
		font-size: 11px;
		color: var(--tl-sub);
		white-space: nowrap;
	}
	.error-a {
		color: #a3241a;
	}
	.reenviar {
		display: inline;
	}
	.reenviar button {
		border: 0;
		background: transparent;
		padding: 0 4px;
		min-height: 32px;
		font: 700 13px var(--tl-font);
		color: var(--tl-menta-texto);
		cursor: pointer;
	}
	.elegir {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 10px;
	}
	.pila-r {
		display: flex;
		flex-direction: column;
		gap: 10px;
		align-items: flex-start;
	}
	.horas {
		border: 0;
		padding: 0;
		margin: 0;
		width: 100%;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(76px, 1fr));
		gap: 6px;
	}
	.h {
		position: relative;
		cursor: pointer;
	}
	.h input {
		position: absolute;
		opacity: 0;
		pointer-events: none;
	}
	.h span {
		display: grid;
		place-items: center;
		height: 44px;
		border-radius: 12px;
		background: repeating-linear-gradient(-45deg, rgba(60, 204, 144, 0.18) 0 2px, transparent 2px 9px), #f3fbf7;
		font-size: 14px;
		font-weight: 700;
		color: var(--tl-menta-texto);
	}
	.h:has(input:checked) span {
		background: var(--tl-petroleo);
		color: #fff;
	}
	.h:has(input:focus-visible) span {
		outline: 3px solid var(--tl-menta);
		outline-offset: 2px;
	}
	.sobre {
		display: flex;
		align-items: center;
		gap: 10px;
		min-height: 44px;
		font-size: 14px;
		cursor: pointer;
	}
	.sobre input {
		width: 20px;
		height: 20px;
		accent-color: var(--tl-petroleo);
	}
	.pie {
		flex: none;
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		padding: 14px 20px 18px;
		border-top: 1px solid #f0e8d8;
	}
	.crece-f {
		flex: 1 1 160px;
		display: flex;
	}
	.primaria {
		flex: 1 1 160px;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		height: 52px;
		padding: 0 18px;
		border: 0;
		border-radius: 999px;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		font: 700 16px var(--tl-font);
		cursor: pointer;
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.45), 0 12px 24px -10px rgba(60, 204, 144, 0.8);
	}
	.otras {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}
	.sec {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		height: 52px;
		padding: 0 16px;
		border: 0;
		border-radius: 999px;
		background: var(--tl-crema);
		color: var(--tl-tinta);
		font: 700 14px var(--tl-font);
		cursor: pointer;
	}
	.sec.peligro {
		background: transparent;
		color: #a3241a;
	}
	.llamar {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 8px 10px;
		padding: 10px 14px;
		border-radius: 14px;
		background: #fbe9c2;
		color: #5c3800;
		font-size: 14px;
	}
	.llamar span {
		flex: 1 1 180px;
	}
	.llamar button {
		min-height: 40px;
		padding: 0 14px;
		border: 0;
		border-radius: 999px;
		background: var(--tl-petroleo);
		color: #fff;
		font: 700 13px var(--tl-font);
		cursor: pointer;
	}
	.lectura {
		align-items: center;
		font-size: 13px;
		color: var(--tl-sub);
	}
</style>
