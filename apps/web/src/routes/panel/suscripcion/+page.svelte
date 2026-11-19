<script lang="ts">
	import '#lib/gestion/gestion.css';
	import { enhance } from '$app/forms';
	import { pesos, ZONA } from '#lib/formato.ts';
	import { confirmar } from '#lib/configuracion/formularios.ts';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';
	import { diasHasta, fechaLarga } from '#lib/gestion/cobros/formato.ts';
	import type { PlanVenta } from './+page.server.ts';

	let { data, form } = $props();
	const e = $derived(data.estado);
	const s = $derived(e.suscripcion);
	const esDuenio = $derived(data.negocio.rol === 'duenio');
	const enPrueba = $derived(e.pruebaHasta !== null && !e.conSuscripcion);
	const vigente = $derived(!!s && (s.estado === 'autorizada' || s.estado === 'pausada'));
	// Contratada desde el teléfono: la cobra la App Store o Google Play, y se cambia o cancela desde ahí.
	const deApple = $derived(s?.origen === 'apple');
	const deGoogle = $derived(s?.origen === 'google');
	const deTienda = $derived(deApple || deGoogle);
	const COMO_APPLE = 'En el iPhone: Ajustes › tu nombre › Suscripciones › Tengo Libre.';
	const COMO_GOOGLE = 'En Android: Play Store › Pagos y suscripciones › Suscripciones › Tengo Libre.';
	const COMO_TIENDA = $derived(deGoogle ? COMO_GOOGLE : COMO_APPLE);
	const DIA = 86_400_000;

	// El carrito: el plan elegido (por defecto, el destacado), el período, cuánto se paga hoy, cuándo
	// es el primer cobro y cuánto después. Los precios salen de la API.
	const destacado = $derived(e.catalogo.find((p) => p.destacado)?.clave ?? e.catalogo[0]?.clave);
	let elegidoA = $state<string | null>(null);
	const elegido = $derived(elegidoA ?? destacado);
	let periodo = $state<'mensual' | 'anual'>('mensual');
	const plan = $derived(e.catalogo.find((p) => p.clave === elegido) ?? e.catalogo[0]);
	const precioDe = (p: PlanVenta) => (periodo === 'anual' ? p.precioAnual : p.precio);
	const ahorro = $derived(plan.precio * 12 - plan.precioAnual);
	const conPrueba = $derived(!!e.pruebaHasta && new Date(e.pruebaHasta).getTime() > Date.now());
	const primerCobro = $derived(conPrueba ? new Date(e.pruebaHasta!) : new Date());
	const fechaCorta = (d: Date) => new Intl.DateTimeFormat('es-AR', { day: 'numeric', month: 'long', timeZone: ZONA }).format(d);
	const PRUEBA = $derived(e.extras?.diasDePrueba ?? 14);
	const planRango = (clave: string) => e.catalogo.findIndex((p) => p.clave === clave);
	const nombreDe = (clave: string) => e.catalogo.find((p) => p.clave === clave)?.nombre ?? clave;

	// Qué incluye cada plan, fila por fila (ícono + palabra en cada celda), armado con lo que dice la
	// API de cada plan: nada que no exista.
	const tiene = (p: PlanVenta, f: string) => p.funciones.includes(f);
	const siNo = (b: boolean) => (b ? 'Sí' : 'No');
	const FILAS = $derived<{ que: string; valor: (p: PlanVenta) => string }[]>([
		{ que: 'Profesionales', valor: (p) => `Hasta ${p.profesionales}` + (p.clave === 'centro' ? ` (+${pesos(e.extras.precioProfesional)} c/u)` : '') },
		{ que: 'Link propio y reservas por la web', valor: () => 'Sí' },
		{ que: 'Recordatorios por mail y en el celular', valor: () => 'Sin límite' },
		{ que: 'Ficha de cada cliente y caja', valor: () => 'Sí' },
		{ que: 'Seña con Mercado Pago al reservar', valor: (p) => siNo(tiene(p, 'sena')) },
		{ que: 'Lista de espera que ofrece el hueco sola', valor: (p) => siNo(tiene(p, 'espera')) },
		{ que: 'Membresías, cupones y referidos', valor: (p) => siNo(tiene(p, 'membresias')) },
		{ que: 'Penalidades por ausencias', valor: (p) => siNo(tiene(p, 'penalidades')) },
		{ que: 'Reportes', valor: (p) => (tiene(p, 'csv') ? 'Completos y CSV' : tiene(p, 'reportes_completos') ? 'Completos' : 'Resumen') },
		{ que: 'Sucursales', valor: (p) => `${p.sucursales}` },
		{ que: 'Dominio propio y plantillas de página', valor: (p) => siNo(tiene(p, 'dominio')) },
		{
			que: 'Recordatorios por WhatsApp',
			valor: (p) => (p.whatsappIncluidos ? `${p.whatsappIncluidos} por\u00a0mes` : `Packs de ${e.extras.packWhatsApp.cantidad}`)
		},
		{ que: 'Asistente con IA', valor: (p) => `${p.cupoIA.toLocaleString('es-AR')} respuestas por\u00a0mes` }
	]);

	// Lo que lleva el negocio contra los topes del plan que rige.
	const u = $derived(e.uso);
	const pctProfes = $derived(u ? Math.min(100, Math.round((100 * u.profesionalesActivos) / Math.max(1, u.topeProfesionales))) : 0);
	const sobran = $derived(u ? Math.max(0, u.profesionalesActivos - u.topeProfesionales) : 0);
	const waIncluidosQuedan = $derived(u ? Math.max(0, u.whatsappIncluidos - u.whatsappUsadosMes) : 0);
	let extrasA = $state<number | null>(null);
	const extras = $derived(extrasA ?? s?.profesionalesExtra ?? 0);
	const montoConExtras = $derived(
		s ? (e.catalogo.find((p) => p.clave === 'centro')!.precio + extras * e.extras.precioProfesional) * (s.periodo === 'anual' ? e.extras.mesesAnual : 1) : 0
	);

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
	{#if data.vueltaWhatsApp}
		<p class="g-aviso" role="status">
			<span class="icono lleno" aria-hidden="true">info</span>
			<span>Volviste de Mercado Pago. Apenas se acredite el pago, los recordatorios se suman a tu saldo (puede tardar unos&nbsp;minutos).</span>
		</p>
	{/if}
	<Mensaje {form} que="cancelar" />
	<Mensaje {form} que="pagar" />
	<Mensaje {form} que="whatsapp" />
	<Mensaje {form} que="extras" />

	{#if enPrueba}
		<section class="prueba" aria-labelledby="t-prueba">
			<div class="prueba-izq">
				<h2 id="t-prueba" class="prueba-titulo">Tenés el plan {e.nombrePlan} hasta el {fechaLarga(e.pruebaHasta!)}</h2>
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
	{:else if s?.estado === 'pausada' && deTienda}
		<section class="g-bloque estado" aria-labelledby="t-estado">
			<h2 id="t-estado" class="g-bloque-titulo grande">{deGoogle ? 'Google Play' : 'La App Store'} no pudo cobrar el plan {e.nombrePlan}</h2>
			<p class="g-nota">
				{#if s.graciaHasta}Todo sigue andando hasta el {fechaLarga(s.graciaHasta)}.{/if} Revisá el medio de pago de tu cuenta de {deGoogle ? 'Google' : 'Apple'}. {COMO_TIENDA}
			</p>
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
							<form method="POST" action="?/pagar"><button class="g-boton menta">Pagar ahora</button></form>
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
			{#if deTienda}
				<p class="g-nota">
					<span class="icono" aria-hidden="true">{deGoogle ? 'phone_android' : 'phone_iphone'}</span>
					Contratada desde el teléfono: la cobra {deGoogle ? 'Google Play' : 'la App Store'}{#if s.proximoCobro}{s.canceladaEn ? ' y termina' : ' y se renueva'} el {fechaLarga(s.proximoCobro)}{/if}.
					Para cambiar de plan o cancelarla: {COMO_TIENDA}
				</p>
			{:else}
				<p class="g-nota">
					{#if s.proximoCobro}Próximo cobro el {fechaLarga(s.proximoCobro)}: {pesos(s.monto)}.{:else}{pesos(s.monto)} por {s.periodo === 'anual' ? 'año' : 'mes'}, con débito automático.{/if}
					{#if s.periodo === 'anual'}Pago anual: 12 meses al precio de {e.extras.mesesAnual}.{/if}
				</p>
			{/if}
			{#if esDuenio && !deTienda}
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
			{#if deTienda}<p class="g-nota">Se contrató desde el teléfono: para volver, suscribite otra vez desde la app o elegí un plan acá abajo.</p>{/if}
			{#if esDuenio && !deTienda}
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
							<th scope="col" class:rec={p.destacado}>
								{#if p.destacado}<span class="recomendado">El más elegido</span>{/if}
								<span class="plan-nombre">{p.nombre}</span>
								<span class="plan-precio"><b class="num">{pesos(p.precio)}</b> por&nbsp;mes</span>
							</th>
						{/each}
					</tr>
				</thead>
				<tbody>
					{#each FILAS as f (f.que)}
						<tr>
							<th scope="row">{f.que}</th>
							{#each e.catalogo as p (p.clave)}
								{@const v = f.valor(p)}
								<td class:rec={p.destacado} class:no={v === 'No'}>
									<span class="icono lleno" aria-hidden="true">{v === 'No' ? 'remove' : 'check_circle'}</span>
									<span class="solo-tel">{p.nombre}:&nbsp;</span>{v}
								</td>
							{/each}
						</tr>
					{/each}
				</tbody>
				{#if esDuenio && vigente && s?.estado !== 'pausada' && !deTienda}
					<tfoot>
						<tr>
							<td class="que"></td>
							{#each e.catalogo as p (p.clave)}
								<td class:rec={p.destacado}>
									{#if s?.plan === p.clave}
										<span class="g-pildora actual"><span class="icono lleno" aria-hidden="true">check_circle</span>Tu plan</span>
									{:else}
										<form
											method="POST"
											action="?/cambiar"
											use:enhance={confirmar(
												`¿Pasarte al plan ${p.nombre}? El precio nuevo rige desde el próximo cobro.` +
													(planRango(p.clave) < planRango(s?.plan ?? '') ? ' No se borra nada: lo que tu plan nuevo no incluye queda guardado, en sólo lectura.' : '')
											)}
										>
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
			{#if esDuenio && !vigente && s?.estado !== 'pausada'}
				<div class="carrito" id="elegir">
					<fieldset class="opciones">
						<legend class="rotulo">1 · Elegí tu plan</legend>
						<div class="periodos" role="radiogroup" aria-label="Cómo pagás">
							<label class:on={periodo === 'mensual'}><input type="radio" name="periodo-vista" value="mensual" bind:group={periodo} />Por mes</label>
							<label class:on={periodo === 'anual'}
								><input type="radio" name="periodo-vista" value="anual" bind:group={periodo} />Por año
								<small>12 meses al precio de {e.extras.mesesAnual}</small></label
							>
						</div>
						{#each e.catalogo as p (p.clave)}
							<label class="opcion" class:on={elegido === p.clave}>
								<input type="radio" name="plan-vista" value={p.clave} checked={elegido === p.clave} onchange={() => (elegidoA = p.clave)} />
								<span class="opcion-marca" aria-hidden="true"><span class="icono lleno">{elegido === p.clave ? 'radio_button_checked' : 'radio_button_unchecked'}</span></span>
								<span class="opcion-cuerpo">
									<span class="opcion-titulo">
										{p.nombre}
										{#if p.destacado}<span class="recomendado">El más elegido</span>{/if}
									</span>
									<span class="opcion-texto">{p.bajada} {p.incluye[0]}.</span>
								</span>
								<span class="opcion-precio"><b class="num">{pesos(precioDe(p))}</b><small>{periodo === 'anual' ? 'por año' : 'por mes'}</small></span>
							</label>
						{/each}
					</fieldset>

					<section class="resumen" aria-labelledby="t-resumen">
						<h3 id="t-resumen" class="rotulo">2 · Tu suscripción</h3>
						<div class="linea-plan">
							<span class="icono lleno" aria-hidden="true">workspace_premium</span>
							<span>
								<b>Plan {plan.nombre}</b>
								<small>{periodo === 'anual' ? `Anual · ahorrás ${pesos(ahorro)}` : 'Mensual'} · se renueva sola · cancelás cuando quieras</small>
							</span>
							<b class="num">{pesos(precioDe(plan))}</b>
						</div>
						<dl class="cuentas">
							{#if conPrueba}
								<div><dt>Prueba gratis</dt><dd class="menta">hasta el {fechaCorta(primerCobro)}</dd></div>
								<div><dt>Primer cobro</dt><dd>{fechaLarga(primerCobro.toISOString())}</dd></div>
							{:else}
								<div><dt>Primer cobro</dt><dd>hoy, al confirmar</dd></div>
							{/if}
							<div>
								<dt>Después</dt>
								<dd>
									<b class="num">{pesos(precioDe(plan))}</b>
									{periodo === 'anual' ? `cada año, el ${fechaCorta(primerCobro)}` : `cada mes, el día ${primerCobro.getDate()}`}
								</dd>
							</div>
							<div><dt>Medio de pago</dt><dd>Mercado Pago · tarjeta o dinero en cuenta</dd></div>
						</dl>
						<div class="total">
							<span>Hoy pagás</span>
							<b class="num">{conPrueba ? pesos(0) : pesos(precioDe(plan))}</b>
						</div>
						<!-- Sin enhance: la respuesta es una redirección a Mercado Pago. -->
						<form method="POST" action="?/suscribir">
							<input type="hidden" name="plan" value={plan.clave} />
							<input type="hidden" name="periodo" value={periodo} />
							<button class="g-cta pagar">
								<span class="icono" aria-hidden="true">lock</span>
								{conPrueba ? `Suscribirme · ${pesos(0)} hoy` : `Pagar ${pesos(precioDe(plan))}`}
							</button>
						</form>
						<p class="letra-chica">
							{conPrueba ? `No se cobra nada hasta que termine la prueba. ` : ''}Te lleva a Mercado Pago para cargar el medio de pago. Precios finales en pesos, con
							impuestos incluidos y sin comisión por turno. <a href="/terminos">Términos</a> · <a href="/privacidad">Privacidad</a>
						</p>
					</section>
				</div>
			{/if}
			{#if !esDuenio}<p class="g-nota pad">La suscripción la elige la dueña o el dueño del&nbsp;negocio.</p>{/if}
			{#if esDuenio && deTienda && vigente}<p class="g-nota pad">Tu plan se paga con {deGoogle ? 'Google Play' : 'la App Store'}. Para cambiarlo: {COMO_TIENDA}</p>{/if}
		</section>

		<div class="g-col lado">
			{#if u}
				<section class="g-bloque grande" aria-labelledby="t-uso">
					<h2 id="t-uso" class="g-bloque-titulo">Tu plan {e.nombrePlan}</h2>
					<div class="uso">
						<div class="uso-fila">
							<span>Profesionales activos</span>
							<b class="num">{u.profesionalesActivos} de {u.topeProfesionales}</b>
						</div>
						<span class="barra" class:llena={pctProfes >= 100} aria-hidden="true"><span style:width="{pctProfes}%"></span></span>
						{#if sobran > 0}
							<p class="g-nota alerta">
								{sobran === 1 ? 'La más nueva queda' : `Las ${sobran} más nuevas quedan`} fuera de la agenda (no se borra nada). Desactivá a alguien en Equipo o pasá a
								un plan más&nbsp;grande.
							</p>
						{:else if u.profesionalesActivos >= u.topeProfesionales && e.plan !== 'centro'}
							<p class="g-nota">Llegaste al tope. Para sumar más, pasá a {nombreDe(e.plan === 'inicial' ? 'equipo' : 'centro')}.</p>
						{/if}
					</div>
					<div class="uso">
						<div class="uso-fila">
							<span>Recordatorios por WhatsApp</span>
							<b class="num">{u.whatsappIncluidos || u.whatsappSaldo ? waIncluidosQuedan + u.whatsappSaldo : '—'}</b>
						</div>
						<p class="g-nota">
							{#if u.whatsappIncluidos}Quedan {waIncluidosQuedan} de los {u.whatsappIncluidos} de este mes{#if u.whatsappSaldo}, más {u.whatsappSaldo} de packs{/if}.
							{:else if u.whatsappSaldo}Te quedan {u.whatsappSaldo} de packs (no vencen).
							{:else}Tu plan no trae WhatsApp: los recordatorios salen por mail y en el celular. Con un pack de {e.extras.packWhatsApp.cantidad} también salen por
								WhatsApp.{/if}
						</p>
						{#if esDuenio}
							<form method="POST" action="?/whatsapp">
								<button class="g-accion">Comprar {e.extras.packWhatsApp.cantidad} por {pesos(e.extras.packWhatsApp.precio)}</button>
							</form>
						{/if}
					</div>
					{#if esDuenio && s?.plan === 'centro' && s.estado === 'autorizada' && !deTienda}
						<form class="uso" method="POST" action="?/extras" use:enhance>
							<label class="uso-fila" for="extras">
								<span>Profesionales extra <small>({pesos(e.extras.precioProfesional)} por mes c/u)</small></span>
								<input id="extras" class="g-entrada extras" name="cantidad" type="number" min="0" max="100" inputmode="numeric" value={extras}
									oninput={(ev) => (extrasA = Math.max(0, Math.min(100, Number(ev.currentTarget.value) || 0)))} />
							</label>
							<p class="g-nota">Con {extras} extra: hasta {25 + extras} profesionales por {pesos(montoConExtras)} {s.periodo === 'anual' ? 'por año' : 'por mes'}.</p>
							<button class="g-accion" disabled={extras === (s.profesionalesExtra ?? 0)}>Guardar</button>
						</form>
					{/if}
				</section>
			{/if}
			<section class="g-bloque grande" aria-labelledby="t-como">
				<h2 id="t-como" class="g-bloque-titulo">Cómo se cobra</h2>
				<ol class="pasos">
					<li><span class="n num">1</span><span><b>Elegís el plan</b> y te lleva a Mercado&nbsp;Pago.</span></li>
					<li><span class="n num">2</span><span><b>Autorizás el débito</b> con tarjeta o dinero en&nbsp;cuenta.</span></li>
					<li><span class="n num">3</span><span><b>Listo:</b> se cobra solo cada mes (o cada año, si elegís el pago&nbsp;anual).</span></li>
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
	.uso {
		display: flex;
		flex-direction: column;
		gap: 8px;
		padding-top: 12px;
		margin-top: 12px;
		border-top: 1px solid var(--tl-linea);
	}
	.uso:first-of-type {
		border-top: 0;
		margin-top: 4px;
		padding-top: 0;
	}
	.uso-fila {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		font-size: 14px;
	}
	.uso-fila small {
		color: var(--suave);
		font-weight: 400;
	}
	.barra {
		display: block;
		height: 8px;
		border-radius: 4px;
		background: var(--tl-linea);
		overflow: hidden;
	}
	.barra > span {
		display: block;
		height: 100%;
		background: var(--tl-menta);
	}
	.barra.llena > span {
		background: var(--aviso);
	}
	.alerta {
		color: var(--peligro);
	}
	.extras {
		width: 84px;
		text-align: right;
	}
	.periodos {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 8px;
		margin-bottom: 10px;
	}
	.periodos label {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 2px;
		padding: 10px 8px;
		border-radius: 14px;
		border: 1.5px solid var(--tl-linea);
		font-weight: 700;
		font-size: 14px;
		cursor: pointer;
		text-align: center;
	}
	.periodos label.on {
		border-color: var(--tl-petroleo);
		background: color-mix(in srgb, var(--tl-menta) 12%, transparent);
	}
	.periodos input {
		position: absolute;
		opacity: 0;
		pointer-events: none;
	}
	.periodos small {
		font-weight: 400;
		font-size: 12px;
	}
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
	.carrito {
		display: grid;
		grid-template-columns: minmax(0, 1.15fr) minmax(0, 1fr);
		gap: 18px;
		padding: 20px;
		border-top: 1px solid #f0e8d8;
		background: #fbf7ef;
		border-radius: 0 0 22px 22px;
	}
	.rotulo {
		display: block;
		margin: 0 0 10px;
		font: var(--tl-rotulo);
		letter-spacing: var(--tl-rotulo-ls);
		text-transform: uppercase;
		color: var(--tl-sub);
	}
	.opciones {
		border: 0;
		margin: 0;
		padding: 0;
		display: grid;
		gap: 10px;
		align-content: start;
	}
	.opciones legend {
		padding: 0;
	}
	.opcion {
		position: relative;
		display: grid;
		grid-template-columns: auto minmax(0, 1fr) auto;
		gap: 12px;
		align-items: center;
		padding: 16px;
		border-radius: 18px;
		background: var(--tl-papel);
		border: 2px solid var(--tl-linea);
		cursor: pointer;
		transition: border-color 0.15s, box-shadow 0.15s, transform 0.15s;
	}
	.opcion:hover {
		border-color: var(--tl-linea-2);
	}
	.opcion.on {
		border-color: var(--tl-menta);
		box-shadow: 0 14px 30px -22px rgba(23, 128, 90, 0.8);
	}
	.opcion input {
		position: absolute;
		opacity: 0;
		pointer-events: none;
	}
	.opcion:has(input:focus-visible) {
		outline: 3px solid var(--tl-menta);
		outline-offset: 2px;
	}
	.opcion-marca .icono {
		font-size: 24px;
		color: var(--tl-sub);
	}
	.opcion.on .opcion-marca .icono {
		color: var(--tl-menta-texto);
	}
	.opcion-titulo {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 18px;
		font-weight: 900;
		letter-spacing: -0.02em;
	}
	.opcion-titulo .recomendado {
		margin: 0;
	}
	.opcion-texto {
		display: block;
		margin-top: 2px;
		font-size: 13px;
		line-height: 1.4;
		color: var(--tl-sub);
		text-wrap: pretty;
	}
	.opcion-precio {
		text-align: right;
		display: flex;
		flex-direction: column;
	}
	.opcion-precio b {
		font-size: 22px;
		letter-spacing: -0.03em;
	}
	.opcion-precio small {
		font-size: 12px;
		color: var(--tl-sub);
	}
	.resumen {
		align-self: start;
		position: sticky;
		top: 16px;
		padding: 18px;
		border-radius: 20px;
		background: var(--tl-papel);
		box-shadow: 0 24px 48px -34px rgba(15, 35, 40, 0.55);
		border: 1px solid var(--tl-linea);
	}
	.linea-plan {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr) auto;
		gap: 12px;
		align-items: center;
		padding-bottom: 14px;
		border-bottom: 2px dashed var(--tl-linea-2);
	}
	.linea-plan > .icono {
		width: 40px;
		height: 40px;
		display: grid;
		place-items: center;
		border-radius: 12px;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
	}
	.linea-plan b {
		display: block;
		font-size: 16px;
	}
	.linea-plan small {
		display: block;
		font-size: 12px;
		color: var(--tl-sub);
	}
	.linea-plan > b.num {
		font-size: 18px;
	}
	.cuentas {
		margin: 12px 0;
		display: grid;
		gap: 8px;
	}
	.cuentas div {
		display: flex;
		justify-content: space-between;
		gap: 12px;
		font-size: 14px;
	}
	.cuentas dt {
		color: var(--tl-sub);
		white-space: nowrap;
	}
	.cuentas dd {
		margin: 0;
		text-align: right;
		color: var(--tl-tinta);
	}
	.cuentas dd.menta {
		color: var(--tl-menta-texto);
		font-weight: 700;
	}
	.total {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		padding: 14px 0;
		border-top: 2px dashed var(--tl-linea-2);
		font-weight: 700;
	}
	.total b {
		font-size: 32px;
		font-weight: 900;
		letter-spacing: -0.04em;
	}
	.pagar {
		width: 100%;
		justify-content: center;
		gap: 8px;
	}
	.letra-chica {
		margin: 10px 0 0;
		font-size: 12px;
		line-height: 1.5;
		color: var(--tl-sub);
		text-wrap: pretty;
	}
	.letra-chica a {
		color: var(--tl-menta-texto);
	}
	/* El carrito se acomoda a su columna (no a la pantalla): angosta, el resumen va abajo. */
	@container planes (max-width: 720px) {
		.carrito {
			grid-template-columns: 1fr;
			padding: 16px;
		}
		.resumen {
			position: static;
		}
	}
	.planes {
		container: planes / inline-size;
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
		/* Tres planes, tres columnas: cada valor queda debajo de su plan. */
		.comparar tr {
			display: grid;
			grid-template-columns: repeat(3, minmax(0, 1fr));
		}
		.comparar th.que,
		.comparar td.que {
			display: none;
		}
		thead th {
			width: auto !important;
			padding: 14px 8px 10px;
		}
		.plan-precio b {
			font-size: 17px;
		}
		.plan-precio {
			font-size: 12px;
		}
		thead .recomendado {
			font-size: 9px;
		}
		tbody th {
			grid-column: 1 / -1;
			padding: 12px 14px 4px;
		}
		tbody td {
			height: auto;
			padding: 4px 8px 12px;
			border-top: 0;
			font-size: 13px;
		}
		tbody td.rec {
			padding-top: 4px;
		}
		tfoot td {
			padding: 12px 4px;
		}
		tfoot .g-cta {
			min-height: 48px;
			font-size: 13px;
			padding: 0 6px;
		}
	}
</style>
