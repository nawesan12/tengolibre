<script lang="ts">
	import '#lib/gestion/gestion.css';
	import { enhance } from '$app/forms';
	import { INTERVALOS } from '#lib/configuracion/textos.ts';
	import { sinReset } from '#lib/configuracion/formularios.ts';
	import { cuandoCorto, describir } from '#lib/configuracion/actividad.ts';
	import Ajuste from '#lib/configuracion/Ajuste.svelte';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';
	import Recordatorio from '#lib/configuracion/Recordatorio.svelte';
	import AvisosNavegador from '#lib/configuracion/AvisosNavegador.svelte';
	import { estadoWhatsapp, TIPO_WHATSAPP } from '#lib/configuracion/avisos.ts';
	import { ZONA } from '#lib/formato.ts';

	let { data, form } = $props();
	const a = $derived(data.ajustes);
	const ahora = new Date();

	// Se guarda solo: cada grupo es un formulario que se manda al cambiar un valor.
	const guardar = (e: Event) => (e.currentTarget as HTMLFormElement).requestSubmit();

	const min = (n: number) => (n === 0 ? 'Nada' : n < 60 ? `${n} min` : n % 60 ? `${Math.floor(n / 60)} h ${n % 60}` : `${n / 60} h`);
	const horas = (n: number) => (n === 0 ? 'Sin mínimo' : n < 1 ? `${n * 60} min` : `${n} h`);
	const op = (valores: number[], texto: (n: number) => string) => valores.map((v) => ({ valor: v, texto: texto(v) }));

	const w = $derived(data.whatsapp);
	const whatsappApagado = $derived(!w?.disponible);
	const horaWhatsapp = $derived(Number((w?.hora || '10:00').slice(0, 2)));
	const ESPERA = op([5, 10, 15, 20, 30, 45, 60, 90, 120, 180, 240, 360, 720], min);
	const cuandoMensaje = (iso: string) =>
		new Intl.DateTimeFormat('es-AR', { timeZone: ZONA, weekday: 'short', hour: '2-digit', minute: '2-digit' }).format(new Date(iso)).toUpperCase();

	const ejemplo = $derived.by(() => {
		const hs = [0, 1, 2].map((i) => {
			const m = 10 * 60 + i * a.intervalo;
			return `${Math.floor(m / 60)}:${String(m % 60).padStart(2, '0')}`;
		});
		return `Con ${a.intervalo} min se ofrecen ${hs.join(', ')}…`;
	});
</script>

<svelte:head><title>Configuración · Panel</title></svelte:head>

<div class="g-pagina">
	<h1 class="g-titulo">Configuración</h1>

	<div class="g-lado">
		<div class="g-col grupos">
			<section class="g-bloque sin-pad" aria-labelledby="t-reserva">
				<h2 class="g-bloque-titulo titulo" id="t-reserva">Reservas</h2>
				<Mensaje {form} que="reserva" />
				<form id="f-reserva" method="POST" action="?/reserva" use:enhance={sinReset} onchange={guardar}>
					<Ajuste
						etiqueta="Los horarios arrancan cada"
						ayuda={ejemplo}
						nombre="intervalo"
						valor={a.intervalo}
						opciones={op(INTERVALOS, (n) => `${n} min`)}
					/>
					<Ajuste
						etiqueta="Se puede reservar con al menos"
						ayuda={'Evita que te reserven para dentro de 10 minutos'}
						nombre="anticipacionHoras"
						valor={a.anticipacionMin / 60}
						opciones={op([0, 0.5, 1, 2, 3, 4, 6, 12, 24, 48, 72], horas)}
					/>
					<Ajuste
						etiqueta="La agenda se ve hasta"
						ayuda="Más adelante no se puede reservar online"
						nombre="diasVista"
						valor={a.diasVista}
						opciones={op([7, 14, 21, 30, 45, 60, 90, 120, 180], (n) => `${n} días`)}
					/>
					<Ajuste
						etiqueta="Tiempo libre entre turnos"
						ayuda="Para limpiar y preparar"
						nombre="margenMin"
						valor={a.margenMin}
						opciones={op([0, 5, 10, 15, 20, 30, 45, 60, 90, 120], min)}
					/>
					<Ajuste
						etiqueta="Se puede cancelar hasta"
						ayuda="Después, sólo escribiéndote"
						nombre="cancelarHastaHoras"
						valor={a.cancelarHastaHoras}
						opciones={op([0, 1, 2, 3, 6, 12, 24, 48, 72, 168], (n) => (n === 0 ? 'Hasta el turno' : `${n} h antes`))}
					/>
					<button class="guardar visualmente-oculto">Guardar reservas</button>
				</form>
			</section>

			<section class="g-bloque sin-pad" aria-labelledby="t-avisos">
				<h2 class="g-bloque-titulo titulo" id="t-avisos">Avisos</h2>
				<Mensaje {form} que="avisos" />
				<form method="POST" action="?/avisos" use:enhance={sinReset} onchange={guardar}>
					<Ajuste
						etiqueta="Resumen del día por mail"
						ayuda="A la tarde, con los turnos de mañana"
						nombre="resumenDiario"
						valor={data.avisos.resumenDiario}
						sino
					/>
					<Ajuste
						etiqueta="Pedir reseña"
						ayuda="Después del turno atendido"
						nombre="resenaDemoraHoras"
						valor={data.avisos.resenaDemoraHoras}
						opciones={op([0, 1, 2, 3, 6, 12, 24, 48, 72, 168], (n) => (n === 0 ? 'Al terminar' : `${n} h después`))}
					/>
					<button class="guardar visualmente-oculto">Guardar avisos</button>
				</form>

				{#if w}
					<Mensaje {form} que="whatsapp" />
					<form method="POST" action="?/whatsapp" use:enhance={sinReset} onchange={guardar} aria-label="WhatsApp">
						{#if whatsappApagado}
							<p class="nota-grupo">
								<span class="icono" aria-hidden="true">info</span>
								<span>Los avisos por WhatsApp todavía no están habilitados en Tengo&nbsp;Libre.</span>
							</p>
						{/if}
						<Ajuste
							etiqueta="Pedir confirmación por WhatsApp"
							ayuda="El día antes, con «Confirmo» y «Necesito cambiarlo»"
							nombre="confirmacion"
							valor={w.confirmacion}
							deshabilitado={whatsappApagado}
							sino
						/>
						<Ajuste
							etiqueta="Se manda desde las"
							ayuda="Hora del día anterior al turno"
							nombre="hora"
							valor={horaWhatsapp}
							opciones={op([7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20], (n) => `${n}:00`)}
							deshabilitado={whatsappApagado}
						/>
						<Ajuste
							etiqueta="Si no contesta, para llamar después de"
							ayuda="Queda en la lista de recepción para llamarlo"
							nombre="sinRespuestaHoras"
							valor={w.sinRespuestaHoras}
							opciones={op([1, 2, 3, 4, 5, 6, 8, 10, 12], (n) => (n === 1 ? '1 h' : `${n} h`))}
							deshabilitado={whatsappApagado}
						/>
						<Ajuste
							etiqueta="Recordatorio por WhatsApp"
							ayuda={w.confirmacion ? 'Con la confirmación prendida no sale: ya es el recordatorio' : '24 h antes, además del mail'}
							nombre="recordatorio"
							valor={w.recordatorio}
							deshabilitado={whatsappApagado}
							sino
						/>
						<button class="guardar visualmente-oculto" disabled={whatsappApagado}>Guardar WhatsApp</button>
					</form>
				{/if}

				{#if data.mensual}
					<Mensaje {form} que="mensual" />
					<form method="POST" action="?/mensual" use:enhance={sinReset} onchange={guardar} aria-label="Reporte mensual">
						<input type="hidden" name="hayActivo" value="1" />
						<Ajuste
							etiqueta="Reporte del mes por mail"
							ayuda="El día 1, con el mes comparado con el anterior"
							nombre="activo"
							clave="mensual-activo"
							valor={data.mensual.activo}
							sino
						/>
						{#each data.mensual.encargados as e (e.id)}
							<input type="hidden" name="encargado" value={e.id} />
							<Ajuste etiqueta="También a {e.nombre}" ayuda={e.email} nombre="recibe-{e.id}" valor={e.recibe} sino />
						{/each}
						<button class="guardar visualmente-oculto">Guardar reporte mensual</button>
					</form>
				{/if}
			</section>

			{#if data.espera}
				<section class="g-bloque sin-pad" aria-labelledby="t-espera">
					<h2 class="g-bloque-titulo titulo" id="t-espera">Lista de espera</h2>
					<Mensaje {form} que="espera" />
					<form method="POST" action="?/espera" use:enhance={sinReset} onchange={guardar}>
						<Ajuste
							etiqueta="Ofrecer el hueco solo"
							ayuda="Cuando se libera un horario, se le ofrece a uno por vez, en orden"
							nombre="automatica"
							valor={data.espera.automatica}
							sino
						/>
						<Ajuste
							etiqueta="Tiempo para tomarlo"
							ayuda="Si no lo toma, pasa a la persona siguiente"
							nombre="minutos"
							valor={data.espera.minutos}
							opciones={ESPERA}
						/>
						<button class="guardar visualmente-oculto">Guardar lista de espera</button>
					</form>
				</section>
			{/if}

			{#if data.recordatorios.length}
				<section class="g-bloque sin-pad" aria-labelledby="t-recordatorios">
					<h2 class="g-bloque-titulo titulo" id="t-recordatorios">Recordatorios</h2>
					{#each data.recordatorios as r (r.tipo)}
						<Recordatorio {r} cupones={data.cupones} {form} />
					{/each}
				</section>
			{/if}

			{#if data.preferencias}
				<section class="g-bloque sin-pad" aria-labelledby="t-tus-avisos">
					<h2 class="g-bloque-titulo titulo" id="t-tus-avisos">Lo que te llega</h2>
					<p class="g-nota bajada-grupo">Qué te llega a vos en la campanita, el teléfono y el navegador. No cambia lo de los&nbsp;demás.</p>
					<Mensaje {form} que="navegador" />
					<AvisosNavegador navegador={data.navegador} />
					<Mensaje {form} que="push" />
					<form method="POST" action="?/push" use:enhance={sinReset} onchange={guardar}>
						{#each data.preferencias.tipos as t (t.tipo)}
							<input type="hidden" name="tipo" value={t.tipo} />
							<Ajuste etiqueta={t.nombre} ayuda="" nombre="activo-{t.tipo}" valor={t.activo} sino />
						{/each}
						<button class="guardar visualmente-oculto">Guardar tus avisos</button>
					</form>
					{#if !data.preferencias.porDefecto}
						<form method="POST" action="?/push" use:enhance={sinReset} class="por-defecto">
							<input type="hidden" name="porDefecto" value="1" />
							<button class="g-accion">Volver a los de mi rol</button>
						</form>
					{/if}
				</section>
			{/if}

			<section class="g-bloque sin-pad" aria-labelledby="t-ausencias">
				<h2 class="g-bloque-titulo titulo" id="t-ausencias">Ausencias</h2>
				<Mensaje {form} que="ausencias" />
				<form method="POST" action="?/ausencias" use:enhance={sinReset} onchange={guardar}>
					<Ajuste
						etiqueta="Marcar «No vino» solo"
						ayuda="Si nadie cierra el turno, queda como ausencia"
						nombre="marcarAusentes"
						valor={data.ausencias.marcarAusentes}
						sino
					/>
					<Ajuste
						etiqueta="Penalizar después de"
						ayuda="Ausencias sin aviso en 90 días"
						nombre="umbralAusencias"
						valor={data.ausencias.umbralAusencias}
						opciones={op([0, 1, 2, 3, 4, 5, 6, 8, 10], (n) => (n === 0 ? 'Nunca' : n === 1 ? '1 ausencia' : `${n} ausencias`))}
					/>
					<Ajuste
						etiqueta="La penalidad dura"
						ayuda="No puede reservar online, sí por vos"
						nombre="diasPenalidad"
						valor={data.ausencias.diasPenalidad}
						opciones={op([1, 3, 7, 15, 30, 60, 90, 180, 365], (n) => (n === 1 ? '1 día' : `${n} días`))}
					/>
					<button class="guardar visualmente-oculto">Guardar ausencias</button>
				</form>
			</section>
		</div>

		<aside class="g-col lado">
			<div class="g-bloque politica">
				<label class="g-bloque-titulo" for="politica">Política de cancelación</label>
				<!-- Va con el formulario de Reservas (la API guarda todo junto) y se guarda al salir del campo. -->
				<textarea
					id="politica"
					form="f-reserva"
					name="politicaCancelacion"
					rows="3"
					maxlength="2000"
					onchange={() => (document.getElementById('f-reserva') as HTMLFormElement | null)?.requestSubmit()}
					placeholder="Por ejemplo: sin cargo hasta 24 h antes. Después, la seña no se devuelve."
					>{a.politicaCancelacion}</textarea
				>
				<span class="g-sub">Se muestra al reservar y en el link del&nbsp;turno.</span>
			</div>

			<a class="g-bloque enlace" href="/panel/sucursales">
				<span class="icono lleno" aria-hidden="true">store</span>
				<b>Sucursales</b>
				<span class="icono" aria-hidden="true">chevron_right</span>
			</a>

			{#if w && data.mensajes.length}
				<div class="g-bloque mensajes">
					<span class="cab">
						<img src="/logos/whatsapp.svg" alt="" width="20" height="20" />
						<b>WhatsApp enviados</b>
					</span>
					<ul>
						{#each data.mensajes as m (m.id)}
							{@const e = estadoWhatsapp(m)}
							<li>
								<span class="cuando">{cuandoMensaje(m.creadoEn)}</span>
								<span class="quien"><b>{TIPO_WHATSAPP[m.tipo] ?? m.tipo}</b> a {m.cliente || m.para}</span>
								<span class="g-estado {e.clase}"><span class="icono lleno" aria-hidden="true">{e.icono}</span>{e.texto}</span>
							</li>
						{/each}
					</ul>
					<a class="g-accion" href="/panel/notificaciones/envios">Ver los mails enviados</a>
				</div>
			{/if}

			{#if data.actividad.length > 0}
				<a class="g-bloque actividad" href="/panel/auditoria">
					<span class="cab">
						<span class="icono" aria-hidden="true">history</span>
						<b>Registro de actividad</b>
						<span class="icono" aria-hidden="true">chevron_right</span>
					</span>
					{#each data.actividad as e (e.id)}
						{@const x = describir(e)}
						<span class="renglon"><span class="cuando">{cuandoCorto(e.creadoEn, ahora)}</span><span><b>{x.quien}</b> {x.texto}</span></span>
					{/each}
				</a>
			{:else}
				<a class="g-bloque enlace" href="/panel/auditoria">
					<span class="icono" aria-hidden="true">history</span>
					<b>Registro de actividad</b>
					<span class="icono" aria-hidden="true">chevron_right</span>
				</a>
			{/if}
		</aside>
	</div>
</div>

<style>
	.g-titulo {
		margin-bottom: 0;
	}
	.grupos {
		gap: 14px;
	}
	.titulo {
		padding: 14px 18px 6px;
	}
	.g-bloque :global(.aviso) {
		margin: 0 18px 8px;
		padding: 8px 12px;
		font-size: 13px;
	}
	.guardar:focus {
		position: static;
		width: auto;
		height: auto;
		clip: auto;
		margin: 0 18px 12px;
	}
	.lado {
		gap: 12px;
		padding-top: 0;
	}
	.politica {
		padding: 16px;
		gap: 8px;
	}
	.politica .g-bloque-titulo {
		font-size: 15px;
	}
	.politica textarea {
		border: 0;
		border-radius: 12px;
		background: #f8f3ea;
		padding: 10px 12px;
		font: 400 13px/1.5 var(--tl-font);
		color: var(--tl-sub-2);
		resize: vertical;
		min-height: 64px;
	}
	.politica textarea:focus {
		outline: none;
		box-shadow: inset 0 0 0 2px var(--tl-menta);
	}
	.enlace {
		flex-direction: row;
		align-items: center;
		gap: 10px;
		padding: 14px 16px;
		color: var(--tl-tinta);
		text-decoration: none;
		min-height: 48px;
	}
	.enlace b,
	.cab b {
		flex: 1;
		font-size: 14px;
	}
	.enlace .icono:first-child,
	.cab .icono:first-child {
		color: var(--tl-menta-texto);
	}
	.actividad {
		padding: 14px 16px;
		gap: 8px;
		color: var(--tl-tinta);
		text-decoration: none;
	}
	.cab {
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.renglon {
		display: grid;
		grid-template-columns: 44px 1fr;
		gap: 8px;
		font-size: 12px;
		line-height: 1.4;
	}
	.cuando {
		font-variant-numeric: tabular-nums;
		text-transform: uppercase;
		color: var(--tl-sub);
	}
	.renglon b {
		font-weight: 700;
	}
	.nota-grupo {
		display: flex;
		align-items: flex-start;
		gap: 8px;
		margin: 0;
		padding: 10px 18px;
		border-top: 1px solid #f0e8d8;
		font-size: 13px;
		color: var(--tl-sub-2);
	}
	.nota-grupo .icono {
		font-size: 18px;
		color: var(--tl-menta-texto);
	}
	.bajada-grupo {
		padding: 0 18px 10px;
	}
	.por-defecto {
		padding: 4px 18px 10px;
		border-top: 1px solid #f0e8d8;
	}
	.mensajes {
		padding: 14px 16px;
		gap: 8px;
	}
	.mensajes ul {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.mensajes li {
		display: grid;
		grid-template-columns: 76px minmax(0, 1fr);
		gap: 2px 8px;
		font-size: 12px;
		line-height: 1.4;
	}
	.mensajes .g-estado {
		grid-column: 2;
		font-size: 12px;
		white-space: normal;
	}
	.quien {
		overflow-wrap: anywhere;
	}
	@media (min-width: 1101px) {
		.lado {
			padding-top: 0;
		}
	}
</style>
