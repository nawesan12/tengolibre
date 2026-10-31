<!--
	Horas muertas (docs/api/horas-muertas.md): la semana como mapa de calor de las últimas 8 semanas,
	la franja que más queda vacía (talón rayado = libre) y el cupón sugerido para llenarla, que se crea
	de un toque. Bloque papel con la pregunta como título, como el resto de Reportes.
-->
<script lang="ts" module>
	import type { Mapa } from './MapaCalor.svelte';
	export type HorasMuertasDatos = {
		desde: string;
		hasta: string;
		semanas: number;
		mapa: Mapa & { capacidadMinutos: number; ocupadoMinutos: number };
		peor: { dia: number; horaDesde: number; horaHasta: number; vacio: number; libreMinutos: number; texto: string } | null;
		sugerencia: {
			porcentaje: number;
			dias: number[];
			horaDesde: string;
			horaHasta: string;
			antiguedadDias: number;
			destinatarios: number;
			codigo: string;
			texto: string;
		} | null;
	};
</script>

<script lang="ts">
	import { enhance } from '$app/forms';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';
	import type { Form } from '#lib/configuracion/mensajes.ts';
	import MapaCalor from './MapaCalor.svelte';

	let { datos, form }: { datos: HorasMuertasDatos; form: Form } = $props();
	let creando = $state(false);
	const creado = $derived(form?.que === 'cupon' && !form.error ? form.mensaje : null);
	const horas = $derived(datos.peor ? Math.round(datos.peor.libreMinutos / 60) : 0);
	const conHorario = $derived(datos.mapa.dias.length > 0 && datos.mapa.horas.length > 0);
</script>

<section class="g-bloque horas" aria-labelledby="t-horas">
	<div class="g-bloque-cabeza">
		<h2 class="pregunta" id="t-horas">¿En qué horas te sobra&nbsp;lugar?</h2>
		<span class="g-meta">Últimas {datos.semanas}&nbsp;semanas</span>
	</div>

	{#if !conHorario}
		<p class="g-nota">Cuando cargues los horarios del equipo, acá vas a ver qué horas quedan&nbsp;vacías.</p>
	{:else}
		<MapaCalor mapa={datos.mapa} etiqueta={`Ocupación por día y hora en las últimas ${datos.semanas} semanas`} />

		{#if datos.peor}
			<div class="franja">
				<span class="libre"><b class="num">{horas}&nbsp;h</b><span>quedaron libres</span></span>
				<span class="texto"><b>{datos.peor.texto}</b><span>Es la franja que más se te queda vacía. Llenarla es plata que hoy no&nbsp;entra.</span></span>
			</div>
		{/if}

		{#if datos.sugerencia}
			<div class="sugerencia">
				<span class="titulo"><span class="icono lleno" aria-hidden="true">lightbulb</span>Para llenarla</span>
				<p>
					{datos.sugerencia.texto}
					{#if datos.sugerencia.destinatarios > 0}
						Le llega a {datos.sugerencia.destinatarios}&nbsp;{datos.sugerencia.destinatarios === 1 ? 'cliente' : 'clientes'}.
					{:else}
						Todavía no hay clientes con esa antigüedad: el cupón queda listo para&nbsp;compartir.
					{/if}
				</p>
				{#if creado}
					<p class="listo" role="status"><span class="icono lleno" aria-hidden="true">check_circle</span>{creado} <a href="/panel/cupones">Ver cupones</a></p>
				{:else}
					<Mensaje {form} que="cupon" />
					<form
						method="POST"
						action="?/cupon"
						use:enhance={() => {
							creando = true;
							return async ({ update }) => {
								await update({ reset: false });
								creando = false;
							};
						}}
					>
						<button class="g-boton oscuro" disabled={creando}>
							<span class="icono" aria-hidden="true">confirmation_number</span>
							{creando ? 'Creando…' : `Crear el cupón ${datos.sugerencia.codigo}`}
						</button>
					</form>
				{/if}
			</div>
		{:else if !datos.peor}
			<p class="g-nota"><span class="icono ok" aria-hidden="true">check_circle</span> Ninguna franja queda vacía la mitad de las veces.</p>
		{/if}
	{/if}
</section>

<style>
	.horas {
		gap: 14px;
	}
	.pregunta {
		margin: 0;
		font-size: 14px;
		font-weight: 400;
		line-height: 1.35;
		letter-spacing: normal;
		color: var(--tl-sub);
	}
	.franja {
		display: grid;
		grid-template-columns: 120px minmax(0, 1fr);
		border-radius: 20px;
		overflow: hidden;
		background: #f8f3ea;
	}
	.libre {
		display: flex;
		flex-direction: column;
		justify-content: center;
		padding: 16px;
		background: repeating-linear-gradient(-45deg, #3ccc90 0 3px, #cdeedd 3px 10px);
		background-size: 28px 28px;
		color: var(--tl-petroleo-hondo);
		border-right: 3px dashed var(--tl-papel);
		animation: tl-raya 3s linear infinite;
	}
	.libre b {
		font-size: 34px;
		letter-spacing: -0.04em;
		line-height: 1;
	}
	.libre span {
		font-size: 13px;
		font-weight: 700;
	}
	.texto {
		display: flex;
		flex-direction: column;
		justify-content: center;
		gap: 4px;
		padding: 14px 16px;
		font-size: 14px;
		line-height: 1.4;
	}
	.texto b {
		font-size: 15px;
	}
	.sugerencia {
		display: flex;
		flex-direction: column;
		gap: 10px;
		padding: 18px 20px;
		border-radius: 20px;
		background: var(--tl-menta-suave);
		color: var(--tl-petroleo-hondo);
	}
	.titulo {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		font-size: 15px;
		font-weight: 700;
	}
	.titulo .icono {
		font-size: 20px;
		color: var(--tl-menta-texto);
	}
	.sugerencia p {
		margin: 0;
		font-size: 15px;
		line-height: 1.5;
		text-wrap: pretty;
	}
	.sugerencia form {
		margin: 0;
	}
	.listo {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 6px;
		font-weight: 700;
	}
	.listo .icono {
		color: var(--tl-menta-texto);
	}
	.listo a {
		color: var(--tl-menta-texto);
	}
	.ok {
		color: var(--tl-menta-texto);
		font-size: 17px;
		vertical-align: -3px;
	}
	@media (max-width: 640px) {
		.franja {
			grid-template-columns: 100px minmax(0, 1fr);
		}
		.libre b {
			font-size: 28px;
		}
		.sugerencia .g-boton {
			width: 100%;
			min-height: 48px;
		}
	}
</style>
