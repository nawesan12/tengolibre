<script lang="ts">
	import { enhance } from '$app/forms';
	import { fecha, pesos } from '#lib/formato.ts';
	import { ESTADOS_SUSCRIPCION } from '#lib/configuracion/textos.ts';
	import { confirmar } from '#lib/configuracion/formularios.ts';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';

	let { data, form } = $props();
	const e = $derived(data.estado);
	const s = $derived(e.suscripcion);
	const esDuenio = $derived(data.negocio.rol === 'duenio');
	const enPrueba = $derived(e.pruebaHasta !== null);
	const vigente = $derived(s && (s.estado === 'autorizada' || s.estado === 'pausada'));
</script>

<svelte:head><title>Suscripción · Panel</title></svelte:head>

<div class="pila">
	<h1>Suscripción</h1>
	{#if data.vuelta}
		<p class="aviso" role="status">Volviste de Mercado Pago. Apenas confirme el alta, tu plan se actualiza solo (puede tardar unos&nbsp;minutos).</p>
	{/if}

	<section class="tarjeta pila" aria-labelledby="t-estado">
		<h2 id="t-estado">Tu plan</h2>
		{#if enPrueba && !e.conSuscripcion}
			<p class="estado">
				<span class="etiqueta">Prueba gratis</span>
				<strong>{e.diasDePrueba === 1 ? 'Te queda 1 día' : `Te quedan ${e.diasDePrueba} días`}</strong>
			</p>
			<p class="suave">Hasta el {fecha(e.pruebaHasta!)} tenés el plan Completo entero. Si te suscribís antes, los días que quedan no se&nbsp;cobran.</p>
		{:else if e.bloqueado}
			<p class="aviso error">Tu prueba gratis terminó. Podés seguir viendo todo, pero para hacer cambios y recibir turnos nuevos hay que&nbsp;suscribirse.</p>
		{:else}
			<p class="estado"><span class="etiqueta">{e.nombrePlan}</span></p>
		{/if}

		{#if s}
			<dl class="datos">
				<dt>Suscripción</dt>
				<dd>{ESTADOS_SUSCRIPCION[s.estado] ?? s.estado} · {pesos(s.monto)} por mes</dd>
				{#if s.proximoCobro}<dt>Próximo cobro</dt><dd>{fecha(s.proximoCobro)}</dd>{/if}
				{#if s.ultimoCobro}<dt>Último cobro</dt><dd>{fecha(s.ultimoCobro)}</dd>{/if}
				{#if s.estado === 'pausada' && s.graciaHasta}<dt>Sigue andando hasta</dt><dd>{fecha(s.graciaHasta)}</dd>{/if}
			</dl>
			{#if s.estado === 'pendiente' && s.link}
				<p><a class="boton" href={s.link} rel="noopener">Terminar el alta en Mercado Pago</a></p>
			{/if}
		{/if}
		<Mensaje {form} que="cancelar" />
		{#if esDuenio && s && s.estado !== 'cancelada'}
			<form method="POST" action="?/cancelar" use:enhance={confirmar('¿Cancelar la suscripción? Si el mes ya está pago, sigue andando hasta el próximo cobro.')}>
				<button class="boton peligro">Cancelar suscripción</button>
			</form>
		{/if}
	</section>

	{#if !vigente}
		<section class="tarjeta pila" aria-labelledby="t-planes">
			<h2 id="t-planes">Suscribirte</h2>
			<p class="suave chico">Se paga por mes con Mercado Pago y se puede cancelar cuando&nbsp;quieras.</p>
			<Mensaje {form} que="suscribir" />
			<div class="grilla">
				{#each e.catalogo as p (p.clave)}
					<article class="tarjeta pila">
						<h3>{p.nombre}</h3>
						<p><strong class="precio">{pesos(p.precio)}</strong> <span class="suave">por mes</span></p>
						{#if esDuenio}
							<!-- Sin enhance: la respuesta es una redirección a Mercado Pago. -->
							<form method="POST" action="?/suscribir">
								<input type="hidden" name="plan" value={p.clave} />
								<button class="boton primario">Suscribirme al {p.nombre}</button>
							</form>
						{/if}
					</article>
				{/each}
			</div>
			{#if !esDuenio}<p class="suave chico">La suscripción la hace la dueña o el dueño del negocio.</p>{/if}
		</section>
	{/if}
</div>

<style>
	.estado {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		align-items: center;
		margin: 0;
	}
	.datos {
		display: grid;
		grid-template-columns: auto 1fr;
		gap: 4px 16px;
		margin: 0;
	}
	.datos dt {
		color: var(--suave);
	}
	.datos dd {
		margin: 0;
	}
	h3 {
		margin: 0;
		font-size: 17px;
	}
	.precio {
		font-size: 22px;
	}
</style>
