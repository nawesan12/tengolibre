<script lang="ts">
	import '#lib/gestion/gestion.css';
	import { enhance } from '$app/forms';
	import { pesos, ZONA } from '#lib/formato.ts';
	import { confirmar } from '#lib/configuracion/formularios.ts';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';
	import { diasHasta, fechaLarga } from '#lib/gestion/cobros/formato.ts';

	let { data, form } = $props();
	const e = $derived(data.estado);
	const s = $derived(e.suscripcion);
	const esDuenio = $derived(data.negocio.rol === 'duenio');
	const enPrueba = $derived(e.pruebaHasta !== null && !e.conSuscripcion);
	const vigente = $derived(!!s && (s.estado === 'autorizada' || s.estado === 'pausada'));
	const DIA = 86_400_000;
	const PRUEBA = 7;

	// Qué incluye cada plan, fila por fila (ícono + palabra en cada celda).
	const FILAS: [string, string, string][] = [
		['Turnos online y agenda', 'Sí', 'Sí'],
		['Recordatorios por mail', 'Sí', 'Sí'],
		['Profesionales', 'Hasta 2', 'Sin límite'],
		['Señas y pagos con Mercado Pago', 'No', 'Sí'],
		['Membresías, cupones y referidos', 'No', 'Sí'],
		['Reportes y descargas', 'Básicos', 'Completos'],
		['Dominio propio y plantillas', 'No', 'Sí'],
		['Integraciones con IA', 'No', 'Sí'],
		['Sucursales', 'No', 'Sí']
	];
	const COLUMNA: Record<string, 1 | 2> = { inicial: 1, completo: 2 };
	const valor = (fila: [string, string, string], clave: string) => (COLUMNA[clave] ? fila[COLUMNA[clave]] : '—');

	// La semana de prueba: lo que pasó, hoy y lo que queda.
	const semana = $derived.by(() => {
		if (!e.pruebaHasta) return [];
		const fin = new Date(e.pruebaHasta).getTime();
		const inicio = fin - PRUEBA * DIA;
		const hoy = Math.min(PRUEBA - 1, Math.max(0, Math.floor((Date.now() - inicio) / DIA)));
		const f = new Intl.DateTimeFormat('es-AR', { weekday: 'short', timeZone: ZONA });
		return Array.from({ length: PRUEBA }, (_, i) => ({
			nombre: f.format(new Date(inicio + i * DIA + DIA / 2)).replace('.', '').toUpperCase(),
			cuando: i < hoy ? 'paso' : i === hoy ? 'hoy' : 'falta'
		}));
	});

	const pill = $derived.by(() => {
		if (enPrueba) return { texto: 'Prueba gratis', icono: 'timer', tono: 'ambar' };
		if (s?.estado === 'autorizada') return { texto: 'Activa', icono: 'check_circle', tono: '' };
		if (s?.estado === 'pausada') return { texto: 'Cobro fallido', icono: 'credit_card_off', tono: 'rojo' };
		if (s?.estado === 'pendiente') return { texto: 'Esperando el alta', icono: 'hourglass_top', tono: 'ambar' };
		if (s?.estado === 'cancelada') return { texto: 'Cancelada', icono: 'event_busy', tono: 'neutro' };
		if (e.bloqueado) return { texto: 'Sólo lectura', icono: 'lock', tono: 'rojo' };
		return null;
	});
	const gracia = $derived(s?.graciaHasta ? diasHasta(s.graciaHasta) : 0);
	const cancelHasta = $derived(s?.estado === 'cancelada' && s.proximoCobro && new Date(s.proximoCobro).getTime() > Date.now() ? s.proximoCobro : null);
	const finGracia = (i: string) => fechaLarga(new Date(new Date(i).getTime() + DIA).toISOString());
	const graciaCorta = (i: string) =>
		new Intl.DateTimeFormat('es-AR', { weekday: 'short', day: '2-digit', month: 'short', timeZone: ZONA })
			.format(new Date(i))
			.replace(/\./g, '')
			.replace(',', '')
			.toUpperCase();
</script>

<svelte:head><title>Suscripción · Panel</title></svelte:head>

<div class="g-pagina">
	<div class="g-cabeza">
		<h1 class="g-titulo titulo">Suscripción</h1>
		{#if pill}<span class="g-pildora {pill.tono}"><span class="icono lleno" aria-hidden="true">{pill.icono}</span>{pill.texto}</span>{/if}
	</div>

	{#if data.vuelta}
		<p class="g-aviso" role="status">
			<span class="icono lleno" aria-hidden="true">info</span>
			<span>Volviste de Mercado Pago. Apenas confirme el alta, tu plan se actualiza solo (puede tardar unos&nbsp;minutos).</span>
		</p>
	{/if}
	<Mensaje {form} que="cancelar" />
	<Mensaje {form} que="pagar" />

	{#if enPrueba}
		<section class="prueba" aria-labelledby="t-prueba">
			<div class="prueba-izq">
				<h2 id="t-prueba" class="prueba-titulo">Tenés todo el plan Completo hasta el {fechaLarga(e.pruebaHasta!)}</h2>
				<p class="prueba-texto">
					Elegí un plan antes de esa fecha y no se corta nada. Si no elegís, el panel queda en sólo lectura y tu página sigue tomando&nbsp;reservas.
				</p>
				<ol class="dias" aria-hidden="true">
					{#each semana as d, i (i)}
						<li class={d.cuando}><span class="raya"></span>{d.nombre}</li>
					{/each}
				</ol>
			</div>
			<div class="prueba-der">
				<b class="prueba-num num" aria-hidden="true">{e.diasDePrueba}</b>
				<span aria-hidden="true">{e.diasDePrueba === 1 ? 'día de prueba' : 'días de prueba'}</span>
				<span class="visualmente-oculto">{e.diasDePrueba === 1 ? 'Te queda 1 día' : `Te quedan ${e.diasDePrueba} días`} de prueba</span>
			</div>
		</section>
	{:else if s?.estado === 'pausada'}
		<section class="fallo" aria-labelledby="t-fallo">
			<p class="banda"><span class="icono" aria-hidden="true">credit_card_off</span>No pudimos cobrar el plan {e.nombrePlan}</p>
			<div class="fallo-cuerpo">
				<h2 id="t-fallo" class="g-titulo-2">
					{gracia > 0 ? `Tenés ${gracia} ${gracia === 1 ? 'día' : 'días'} para resolverlo` : 'Resolvelo hoy para no cortar nada'}
				</h2>
				{#if s.graciaHasta}
					<div class="gracia">
						<span class="gracia-texto">
							<span class="gracia-rotulo">Período de gracia hasta</span>
							<b class="gracia-fecha num">{graciaCorta(s.graciaHasta)}</b>
							<span class="gracia-sub">Todo sigue andando hasta ese día</span>
						</span>
						{#if esDuenio}
							{#if s.link}
								<a class="g-boton menta" href={s.link} rel="noopener">Pagar ahora</a>
							{:else}
								<form method="POST" action="?/pagar"><button class="g-boton menta">Pagar ahora</button></form>
							{/if}
						{/if}
					</div>
				{/if}
				<div class="g-paso ahora">
					<span class="g-paso-num"><span class="icono lleno" aria-hidden="true">credit_card</span></span>
					<span class="g-paso-cuerpo">
						<b>Mercado Pago no pudo cobrar la&nbsp;cuota</b>
						<span>Suele ser una tarjeta vencida o sin fondos. Podés cambiarla o pagar con dinero en&nbsp;cuenta.</span>
						{#if esDuenio}
							<form method="POST" action="?/tarjeta"><button class="g-accion">Cambiar tarjeta</button></form>
						{/if}
					</span>
				</div>
				{#if s.graciaHasta}
					<div class="g-paso">
						<span class="g-paso-num"><span class="icono lleno" aria-hidden="true">visibility</span></span>
						<span class="g-paso-cuerpo">
							<b>Si no se resuelve</b>
							<span>El {finGracia(s.graciaHasta)} el panel pasa a sólo lectura. Tu página sigue tomando&nbsp;reservas.</span>
						</span>
					</div>
				{/if}
			</div>
		</section>
	{:else if s?.estado === 'autorizada'}
		<section class="g-bloque estado" aria-labelledby="t-estado">
			<h2 id="t-estado" class="g-bloque-titulo grande">Plan {e.nombrePlan}</h2>
			<p class="g-nota">
				{#if s.proximoCobro}Próximo cobro el {fechaLarga(s.proximoCobro)}: {pesos(s.monto)}.{:else}{pesos(s.monto)} por mes, con débito automático.{/if}
			</p>
			{#if esDuenio}
				<div class="fila acciones">
					<a class="g-accion" href="#planes">Cambiar de plan</a>
					<form method="POST" action="?/cancelar" use:enhance={confirmar('¿Cancelar la suscripción? Si el mes ya está pago, sigue andando hasta el próximo cobro.')}>
						<button class="g-accion peligro">Cancelar</button>
					</form>
				</div>
			{/if}
		</section>
	{:else if s?.estado === 'cancelada'}
		<section class="g-bloque estado" aria-labelledby="t-estado">
			<h2 id="t-estado" class="g-bloque-titulo grande">
				{cancelHasta ? `Seguís con todo hasta el ${fechaLarga(cancelHasta)}` : 'Tu suscripción está cancelada'}
			</h2>
			<p class="g-nota">No se te vuelve a cobrar. Tus datos quedan&nbsp;guardados.</p>
			{#if esDuenio}
				<form method="POST" action="?/reactivar"><button class="g-accion">Reactivar</button></form>
			{/if}
		</section>
	{:else if s?.estado === 'pendiente'}
		<div class="g-aviso ambar">
			<span class="icono" aria-hidden="true">hourglass_top</span>
			<span>Falta terminar el alta del plan {e.nombrePlan} en Mercado Pago. Hasta que se confirme, no se cobra&nbsp;nada.</span>
			{#if s.link}<a class="g-boton" href={s.link} rel="noopener">Terminar el alta</a>{/if}
		</div>
	{/if}

	{#if e.bloqueado && !vigente}
		<p class="g-aviso rojo">
			<span class="icono" aria-hidden="true">lock</span>
			<span>Tu prueba gratis terminó. Podés seguir viendo todo, pero para hacer cambios y recibir turnos nuevos hay que&nbsp;suscribirse.</span>
		</p>
	{/if}

	<div class="cuerpo" id="planes">
		<section class="g-bloque sin-pad planes" aria-labelledby="t-planes">
			<h2 id="t-planes" class="visualmente-oculto">Planes</h2>
			<div class="pad"><Mensaje {form} que="suscribir" /></div>
			<table class="comparar">
				<thead>
					<tr>
						<th scope="col" class="que">Qué incluye</th>
						{#each e.catalogo as p (p.clave)}
							<th scope="col" class:rec={p.clave === 'completo'}>
								{#if p.clave === 'completo'}<span class="recomendado">Recomendado</span>{/if}
								<span class="plan-nombre">{p.nombre}</span>
								<span class="plan-precio"><b class="num">{pesos(p.precio)}</b> por&nbsp;mes</span>
							</th>
						{/each}
					</tr>
				</thead>
				<tbody>
					{#each FILAS as f (f[0])}
						<tr>
							<th scope="row">{f[0]}</th>
							{#each e.catalogo as p (p.clave)}
								{@const v = valor(f, p.clave)}
								<td class:rec={p.clave === 'completo'} class:no={v === 'No'}>
									<span class="icono lleno" aria-hidden="true">{v === 'No' ? 'remove' : 'check_circle'}</span>
									<span class="solo-tel">{p.nombre}:&nbsp;</span>{v}
								</td>
							{/each}
						</tr>
					{/each}
				</tbody>
				{#if esDuenio && s?.estado !== 'pausada'}
					<tfoot>
						<tr>
							<td class="que"></td>
							{#each e.catalogo as p (p.clave)}
								<td class:rec={p.clave === 'completo'}>
									{#if !vigente}
										<!-- Sin enhance: la respuesta es una redirección a Mercado Pago. -->
										<form method="POST" action="?/suscribir">
											<input type="hidden" name="plan" value={p.clave} />
											<button class={p.clave === 'completo' ? 'g-cta' : 'g-cta claro'}>Elegir {p.nombre}</button>
										</form>
									{:else if s?.plan === p.clave}
										<span class="g-pildora actual"><span class="icono lleno" aria-hidden="true">check_circle</span>Tu plan</span>
									{:else}
										<form method="POST" action="?/cambiar" use:enhance={confirmar(`¿Pasarte al plan ${p.nombre}? El precio nuevo rige desde el próximo cobro.`)}>
											<input type="hidden" name="plan" value={p.clave} />
											<button class="g-cta claro">Cambiar a {p.nombre}</button>
										</form>
									{/if}
								</td>
							{/each}
						</tr>
					</tfoot>
				{/if}
			</table>
			{#if !esDuenio}<p class="g-nota pad">La suscripción la elige la dueña o el dueño del&nbsp;negocio.</p>{/if}
		</section>

		<div class="g-col lado">
			<section class="g-bloque grande" aria-labelledby="t-como">
				<h2 id="t-como" class="g-bloque-titulo">Cómo se cobra</h2>
				<ol class="pasos">
					<li><span class="n num">1</span><span><b>Elegís el plan</b> y te lleva a Mercado&nbsp;Pago.</span></li>
					<li><span class="n num">2</span><span><b>Autorizás el débito</b> con tarjeta o dinero en&nbsp;cuenta.</span></li>
					<li><span class="n num">3</span><span><b>Listo:</b> cada mes se cobra solo y te llega la factura por&nbsp;mail.</span></li>
				</ol>
				<p class="mp"><img src="/logos/mercadopago.svg" alt="" width="20" height="20" />Débito automático con Mercado Pago. Precios finales, con&nbsp;IVA.</p>
			</section>
			<section class="g-bloque grande" aria-labelledby="t-cobros">
				<h2 id="t-cobros" class="g-bloque-titulo">Cobros</h2>
				<p class="g-nota">
					{#if s?.ultimoCobro}
						El último cobro fue el {fechaLarga(s.ultimoCobro)}: {pesos(s.monto)}.
						{#if s.proximoCobro && s.estado === 'autorizada'}El próximo, el {fechaLarga(s.proximoCobro)}.{/if}
					{:else}
						Todavía no hay cobros. El primero va a ser el día que elijas el plan, y después cada mes ese mismo&nbsp;día.
					{/if}
				</p>
			</section>
		</div>
	</div>
</div>

<style>
	.titulo {
		flex: 0 1 auto;
	}
	.g-cabeza {
		align-items: center;
	}
	.prueba {
		display: grid;
		grid-template-columns: minmax(0, 1fr) 200px;
		border-radius: 22px;
		overflow: hidden;
		background-color: var(--tl-petroleo);
		background-image: var(--tl-grano);
		color: #fff;
	}
	.prueba-izq {
		padding: 22px 24px;
		display: flex;
		flex-direction: column;
		gap: 12px;
		border-right: 2px dashed rgba(255, 255, 255, 0.2);
	}
	.prueba-titulo {
		margin: 0;
		font-size: 22px;
		line-height: 1.2;
		font-weight: 700;
		letter-spacing: -0.02em;
	}
	.prueba-texto {
		margin: 0;
		font-size: 14px;
		line-height: 1.5;
		color: rgba(255, 255, 255, 0.78);
	}
	.dias {
		list-style: none;
		margin: 4px 0 0;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(7, minmax(0, 1fr));
		gap: 4px;
	}
	.dias li {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 4px;
		font-size: 11px;
		font-weight: 700;
		letter-spacing: 0.06em;
		color: rgba(255, 255, 255, 0.7);
	}
	.raya {
		width: 100%;
		height: 8px;
		border-radius: 4px;
		background: rgba(255, 255, 255, 0.22);
	}
	.dias .hoy {
		color: var(--tl-menta);
	}
	.dias .hoy .raya {
		background: var(--tl-menta);
	}
	.dias .falta .raya {
		background: rgba(60, 204, 144, 0.35);
	}
	.prueba-der {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 2px;
		padding: 18px;
		font-size: 14px;
	}
	.prueba-num {
		font-size: 64px;
		font-weight: 700;
		letter-spacing: -0.06em;
		line-height: 0.9;
		color: var(--tl-menta);
	}
	.fallo {
		border-radius: var(--tl-r-l);
		overflow: hidden;
		background: var(--tl-crema-pagina);
	}
	.banda {
		margin: 0;
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 16px 20px;
		background: #5b1a13;
		color: #fff;
		font-size: 15px;
	}
	.fallo-cuerpo {
		display: flex;
		flex-direction: column;
		gap: 14px;
		padding: 20px;
	}
	.gracia {
		display: flex;
		align-items: center;
		gap: 14px;
		flex-wrap: wrap;
		padding: 16px 18px;
		border-radius: var(--tl-r-m);
		background-color: var(--tl-petroleo);
		background-image: var(--tl-grano);
		color: #fff;
	}
	.gracia-texto {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}
	.gracia-rotulo {
		font-size: 13px;
		color: rgba(255, 255, 255, 0.78);
	}
	.gracia-fecha {
		font-size: 34px;
		font-weight: 700;
		letter-spacing: -0.05em;
		line-height: 1;
		color: #f4b7ae;
	}
	.gracia-sub {
		font: 400 12px var(--tl-font);
		text-transform: uppercase;
		color: var(--tl-menta);
	}
	.estado .grande {
		font-size: 18px;
	}
	.acciones {
		gap: 16px;
	}
	.cuerpo {
		display: grid;
		grid-template-columns: minmax(0, 1.6fr) minmax(0, 1fr);
		gap: 20px;
		align-items: start;
	}
	.planes {
		border-radius: 22px;
		box-shadow: 0 20px 40px -30px rgba(15, 35, 40, 0.5);
	}
	.pad:empty {
		display: none;
	}
	.pad {
		padding: 14px 18px 0;
	}
	.comparar {
		width: 100%;
		border-collapse: collapse;
	}
	.comparar th,
	.comparar td {
		text-align: left;
		vertical-align: middle;
	}
	thead th {
		padding: 20px 16px 16px;
		vertical-align: bottom;
	}
	thead th.que {
		padding-left: 20px;
		font-size: 13px;
		font-weight: 400;
		color: var(--tl-sub);
	}
	thead th:not(.que) {
		width: 30%;
	}
	thead th.rec {
		background: #f3fbf7;
		border-radius: 18px 18px 0 0;
	}
	.recomendado {
		display: inline-block;
		margin-bottom: 4px;
		padding: 3px 8px;
		border-radius: 6px;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		font: 700 11px var(--tl-font);
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}
	.plan-nombre {
		display: block;
		font-size: 20px;
		font-weight: 700;
		letter-spacing: -0.02em;
	}
	.plan-precio {
		display: block;
		font-size: 13px;
		font-weight: 400;
		color: var(--tl-sub-2);
	}
	.plan-precio b {
		font-size: 28px;
		letter-spacing: -0.04em;
		color: var(--tl-tinta);
	}
	tbody th {
		padding: 8px 20px;
		min-height: 48px;
		font-size: 14px;
		font-weight: 700;
		border-top: 1px solid #f0e8d8;
	}
	tbody td {
		padding: 8px 16px;
		height: 48px;
		font-size: 13px;
		color: var(--tl-menta-texto);
		border-top: 1px solid #f0e8d8;
	}
	tbody td .icono {
		font-size: 18px;
		vertical-align: -4px;
		margin-right: 4px;
	}
	tbody td.no {
		color: #5e6b68;
	}
	tbody td.rec {
		background: #f3fbf7;
		font-weight: 700;
	}
	tfoot td {
		padding: 16px;
		border-top: 1px solid #f0e8d8;
	}
	tfoot td.rec {
		background: #f3fbf7;
	}
	/* El plan de cada celda, para quien no ve las columnas (en el teléfono la tabla es una grilla). */
	.solo-tel {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip: rect(0 0 0 0);
		white-space: nowrap;
	}
	tfoot .g-cta {
		min-height: 46px;
		padding: 0 14px;
		font-size: 14px;
		white-space: nowrap;
	}
	.actual {
		height: 46px;
		padding: 0 16px;
	}
	.pasos {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.pasos li {
		display: grid;
		grid-template-columns: 28px minmax(0, 1fr);
		gap: 10px;
		font-size: 13px;
		line-height: 1.45;
	}
	.pasos .n {
		width: 26px;
		height: 26px;
		border-radius: 50%;
		display: grid;
		place-items: center;
		background: var(--tl-crema);
		font-size: 12px;
		font-weight: 700;
	}
	.mp {
		margin: 0;
		display: flex;
		align-items: center;
		gap: 8px;
		padding-top: 10px;
		border-top: 1px solid #f0e8d8;
		font-size: 12px;
		color: var(--tl-sub);
	}
	@media (max-width: 1000px) {
		.cuerpo {
			grid-template-columns: minmax(0, 1fr);
		}
	}
	@media (max-width: 640px) {
		.prueba {
			grid-template-columns: minmax(0, 1fr);
		}
		.prueba-izq {
			border-right: 0;
			border-bottom: 2px dashed rgba(255, 255, 255, 0.2);
			padding: 18px;
		}
		.prueba-der {
			flex-direction: row;
			gap: 10px;
		}
		.prueba-num {
			font-size: 48px;
		}
		/* La tabla en el teléfono: cada fila, el nombre arriba y los dos planes lado a lado. */
		.comparar,
		.comparar thead,
		.comparar tbody,
		.comparar tfoot {
			display: block;
		}
		.comparar tr {
			display: grid;
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
		.comparar th.que,
		.comparar td.que {
			display: none;
		}
		thead th {
			width: auto !important;
			padding: 16px 14px 12px;
		}
		.plan-precio b {
			font-size: 22px;
		}
		tbody th {
			grid-column: 1 / -1;
			padding: 12px 14px 4px;
		}
		tbody td {
			height: auto;
			padding: 4px 14px 12px;
			border-top: 0;
		}
		tbody td.rec {
			padding-top: 4px;
		}
		tfoot td {
			padding: 12px 10px;
		}
		tfoot .g-cta {
			min-height: 48px;
			font-size: 15px;
			padding: 0 10px;
		}
	}
</style>
