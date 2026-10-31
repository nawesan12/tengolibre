<script lang="ts">
	import '#lib/gestion/gestion.css';
	import { enhance } from '$app/forms';
	import { INTERVALOS } from '#lib/configuracion/textos.ts';
	import { sinReset } from '#lib/configuracion/formularios.ts';
	import { cuandoCorto, describir } from '#lib/configuracion/actividad.ts';
	import Ajuste from '#lib/configuracion/Ajuste.svelte';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';

	let { data, form } = $props();
	const a = $derived(data.ajustes);
	const ahora = new Date();

	// Se guarda solo: cada grupo es un formulario que se manda al cambiar un valor.
	const guardar = (e: Event) => (e.currentTarget as HTMLFormElement).requestSubmit();

	const min = (n: number) => (n === 0 ? 'Nada' : n < 60 ? `${n} min` : n % 60 ? `${Math.floor(n / 60)} h ${n % 60}` : `${n / 60} h`);
	const horas = (n: number) => (n === 0 ? 'Sin mínimo' : n < 1 ? `${n * 60} min` : `${n} h`);
	const op = (valores: number[], texto: (n: number) => string) => valores.map((v) => ({ valor: v, texto: texto(v) }));

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
			</section>

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
	@media (min-width: 1101px) {
		.lado {
			padding-top: 0;
		}
	}
</style>
