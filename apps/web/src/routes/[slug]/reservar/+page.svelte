<script lang="ts">
	import { enhance } from '$app/forms';
	import { page } from '$app/state';
	import { dia as diaDe, pesos } from '#lib/formato.ts';
	import Calendario from '#lib/publico/Calendario.svelte';
	import Horarios from '#lib/publico/Horarios.svelte';
	import { fechaLarga, sumarDias } from '#lib/publico/calendario.ts';
	import { CUALQUIERA, urlDePaso, type Paso } from '#lib/publico/reserva.ts';
	import { duracion, textoCobro } from '#lib/publico/textos.ts';

	let { data, form } = $props();
	const n = $derived(data.negocio);
	const base = $derived(`/${n.slug}/reservar`);
	// Las acciones van a la misma URL con lo elegido: sin JavaScript, si fallan, la página vuelve al mismo paso.
	const accion = (nombre: string) => {
		const q = new URLSearchParams([...page.url.searchParams].filter(([k]) => !k.startsWith('/')));
		const s = q.toString();
		return `?${s ? s + '&' : ''}/${nombre}`;
	};
	const ir = (paso: Paso, cambios: Record<string, string> = {}) => urlDePaso(base, page.url.searchParams, paso, cambios);

	const PASOS: { paso: Paso; nombre: string }[] = [
		{ paso: 'servicio', nombre: 'Servicio' },
		{ paso: 'profesional', nombre: 'Profesional' },
		{ paso: 'dia', nombre: 'Día' },
		{ paso: 'horario', nombre: 'Horario' },
		{ paso: 'datos', nombre: 'Tus datos' }
	];
	const indice = $derived(PASOS.findIndex((p) => p.paso === data.paso));

	const grupos = $derived.by(() => {
		const out = data.categorias
			.map((c) => ({ id: c.id, nombre: c.nombre, servicios: data.servicios.filter((s) => s.categoriaId === c.id) }))
			.filter((g) => g.servicios.length);
		const sueltos = data.servicios.filter((s) => !s.categoriaId || !data.categorias.some((c) => c.id === s.categoriaId));
		if (sueltos.length) out.push({ id: '', nombre: out.length ? 'Otros servicios' : '', servicios: sueltos });
		return out;
	});

	// Lo cargado vuelve si la acción falla; el cupón validado ajusta el total.
	const v = $derived((form?.valores ?? {}) as Record<string, string>);
	const cupon = $derived(form && 'cupon' in form ? form.cupon : null);
	const precio = $derived(data.servicio?.precio ?? 0);
	const descuento = $derived(cupon?.valido ? Math.min(cupon.descuento ?? 0, precio) : 0);
	const cobroTexto = $derived(textoCobro(data.cobro, pesos));
	const nombreProfesional = $derived(
		data.profesional?.nombre ?? (data.profesionalId === CUALQUIERA ? 'Cualquiera' : '')
	);
	const enEspera = $derived(!!form && ('anotado' in form || ('enEspera' in form && form.enEspera)));
	const sinDias = $derived(data.paso === 'dia' && data.dias.every((d) => d.libres === 0));
	const hoy = $derived(diaDe(new Date(), n.zonaHoraria));
	let enviando = $state(false);
	const abierto = (si: boolean | undefined) => (si ? { open: true } : {});
</script>

<svelte:head>
	<title>Reservar turno · {n.nombre}</title>
	<meta name="description" content={`Elegí servicio, día y horario y reservá tu turno en ${n.nombre}.`} />
</svelte:head>

{#snippet espera(desde: string, hasta: string)}
	{#if form && 'anotado' in form && form.anotado}
		<p class="aviso" role="status">
			¡Listo! Te anotamos en la lista de espera para {form.anotado.servicio}. Si se libera un lugar, te avisamos por&nbsp;mail.
		</p>
	{:else}
		<!-- Sólo se abre solo (sin cerrar lo que la persona abrió antes de que cargue la página). -->
		<details class="tarjeta" {...abierto(enEspera)}>
			<summary><strong>Anotarme en la lista de espera</strong></summary>
			<form method="POST" action={accion('espera')} use:enhance class="pila" style="margin-top: 12px">
				<p class="suave chico" style="margin: 0">Si alguien cancela, te avisamos por mail para que reserves&nbsp;primero.</p>
				<input type="hidden" name="servicioId" value={data.servicio?.id} />
				<input type="hidden" name="profesionalId" value={data.profesionalId} />
				{#if form && 'enEspera' in form && form.error}<p class="aviso error" role="alert">{form.error}</p>{/if}
				<label class="campo">Nombre <input name="nombre" required maxlength="80" autocomplete="name" value={v.nombre ?? ''} /></label>
				<label class="campo">Email <input name="email" type="email" required maxlength="120" autocomplete="email" value={v.email ?? ''} /></label>
				<label class="campo">Teléfono (opcional) <input name="telefono" type="tel" maxlength="30" autocomplete="tel" value={v.telefono ?? ''} /></label>
				<div class="fila">
					<label class="campo" style="flex: 1">Desde <input name="desde" type="date" required min={hoy} value={v.desde ?? desde} /></label>
					<label class="campo" style="flex: 1">Hasta <input name="hasta" type="date" required min={hoy} value={v.hasta ?? hasta} /></label>
				</div>
				<button class="boton">Anotarme</button>
			</form>
		</details>
	{/if}
{/snippet}

<main class="contenedor pila reserva">
	<header class="pila" style="gap: 4px">
		<a class="suave chico" href={`/${n.slug}`}>← {n.nombre}</a>
		<h1>Reservá tu&nbsp;turno</h1>
	</header>

	<ol class="pasos" aria-label="Pasos de la reserva">
		{#each PASOS as p, i (p.paso)}
			<li class:hecho={i < indice} class:actual={i === indice} aria-current={i === indice ? 'step' : undefined}>{p.nombre}</li>
		{/each}
	</ol>

	{#if indice > 0}
		<dl class="tarjeta resumen">
			{#if data.servicio}
				<div>
					<dt>Servicio</dt>
					<dd>{data.servicio.nombre} · {duracion(data.servicio.duracion)} · {pesos(data.servicio.precio)}</dd>
					<a href={ir('servicio')} aria-label="Cambiar servicio">Cambiar</a>
				</div>
			{/if}
			{#if nombreProfesional}
				<div>
					<dt>Profesional</dt>
					<dd>{nombreProfesional}</dd>
					{#if data.profesionales.length > 1}<a href={ir('profesional', { profesional: '' })} aria-label="Cambiar profesional">Cambiar</a>{/if}
				</div>
			{/if}
			{#if data.fecha && indice >= 3}
				<div>
					<dt>Día</dt>
					<dd>{fechaLarga(data.fecha)}</dd>
					<a href={ir('dia')} aria-label="Cambiar día">Cambiar</a>
				</div>
			{/if}
			{#if data.horario}
				<div>
					<dt>Horario</dt>
					<dd>{data.horario.hora}&nbsp;h</dd>
					<a href={ir('horario')} aria-label="Cambiar horario">Cambiar</a>
				</div>
			{/if}
		</dl>
	{/if}

	{#if data.aviso}<p class="aviso error" role="alert">{data.aviso}</p>{/if}

	{#if data.paso === 'servicio'}
		<section class="pila" aria-labelledby="t-paso">
			<h2 id="t-paso">¿Qué te querés&nbsp;hacer?</h2>
			{#if data.servicios.length === 0}<p class="suave">Este negocio todavía no cargó servicios.</p>{/if}
			{#each grupos as g (g.id)}
				{#if g.nombre}<h3>{g.nombre}</h3>{/if}
				<ul class="opciones">
					{#each g.servicios as s (s.id)}
						<li>
							<a class="tarjeta opcion" href={ir('profesional', { servicio: s.id })}>
								<span class="pila" style="gap: 2px">
									<strong>{s.nombre}</strong>
									{#if s.descripcion}<span class="suave chico">{s.descripcion}</span>{/if}
									<span class="suave chico">{duracion(s.duracion)}</span>
								</span>
								<span class="precio">{s.precio > 0 ? pesos(s.precio) : 'Sin cargo'}</span>
							</a>
						</li>
					{/each}
				</ul>
			{/each}
		</section>
	{:else if data.paso === 'profesional'}
		<section class="pila" aria-labelledby="t-paso">
			<h2 id="t-paso">¿Con&nbsp;quién?</h2>
			{#if data.profesionales.length}
				<ul class="opciones">
					<li>
						<a class="tarjeta opcion" href={ir('dia', { profesional: CUALQUIERA })}>
							<span class="pila" style="gap: 2px">
								<strong>Cualquiera</strong>
								<span class="suave chico">El primero que tenga lugar: más horarios para&nbsp;elegir.</span>
							</span>
						</a>
					</li>
					{#each data.profesionales as p (p.id)}
						<li>
							<a class="tarjeta opcion" href={ir('dia', { profesional: p.id })}>
								<span class="fila" style="flex-wrap: nowrap">
									{#if p.foto}<img src={p.foto} alt="" width="40" height="40" class="foto" />{/if}
									<span class="pila" style="gap: 2px">
										<strong>{p.nombre}</strong>
										{#if p.especialidad}<span class="suave chico">{p.especialidad}</span>{/if}
									</span>
								</span>
							</a>
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	{:else if data.paso === 'dia'}
		<section class="pila" aria-labelledby="t-paso">
			<h2 id="t-paso">Elegí el&nbsp;día</h2>
			{#if sinDias}
				<p>No quedan horarios libres en los próximos días.</p>
				{@render espera(hoy, sumarDias(hoy, 14))}
			{:else}
				<Calendario dias={data.dias} href={(f) => ir('horario', { fecha: f })} />
				<p class="suave chico">Los días tachados no tienen horarios&nbsp;libres.</p>
			{/if}
		</section>
	{:else if data.paso === 'horario'}
		<section class="pila" aria-labelledby="t-paso">
			<h2 id="t-paso">Elegí el horario</h2>
			<p class="suave" style="margin: 0">{fechaLarga(data.fecha)}</p>
			{#if data.horarios.length}
				<Horarios horarios={data.horarios} href={(h) => ir('datos', { inicio: h.inicio })} />
			{:else}
				<p>Ese día ya no quedan horarios. <a href={ir('dia')}>Elegí otro día</a> o anotate en la lista de&nbsp;espera.</p>
				{@render espera(data.fecha, data.fecha)}
			{/if}
		</section>
	{:else if data.paso === 'datos' && data.servicio && data.horario}
		<section class="pila" aria-labelledby="t-paso">
			<h2 id="t-paso">Tus datos</h2>
			<form
				method="POST"
				action={accion('reservar')}
				class="pila"
				use:enhance={({ action }) => {
					enviando = action.searchParams.has('/reservar');
					return async ({ update }) => {
						await update({ reset: false });
						enviando = false;
					};
				}}
			>
				<input type="hidden" name="servicioId" value={data.servicio.id} />
				<input type="hidden" name="profesionalId" value={data.profesionalId} />
				<input type="hidden" name="inicio" value={data.horario.inicio} />
				{#if form && 'error' in form && form.error && !('enEspera' in form)}
					<p class="aviso error" role="alert">
						{form.error}
						{#if form.codigo === 'horario_ocupado'}<a href={ir('horario')}>Elegí otro horario</a>{/if}
					</p>
				{/if}
				<label class="campo">
					Nombre y apellido
					<input name="nombre" required maxlength="80" autocomplete="name" value={v.nombre ?? ''} aria-invalid={!!form?.campos?.['cliente.nombre']} />
				</label>
				<label class="campo">
					Teléfono
					<input name="telefono" type="tel" maxlength="30" autocomplete="tel" inputmode="tel" value={v.telefono ?? ''} aria-invalid={!!form?.campos?.['cliente.telefono']} />
				</label>
				<label class="campo">
					Email
					<input name="email" type="email" maxlength="120" autocomplete="email" value={v.email ?? ''} aria-invalid={!!form?.campos?.['cliente.email']} aria-describedby="ayuda-contacto" />
				</label>
				<p id="ayuda-contacto" class="suave chico" style="margin: -6px 0 0">Dejá teléfono o email: ahí te llega la confirmación y el&nbsp;recordatorio.</p>
				<label class="campo">
					Algo que el negocio tenga que saber (opcional)
					<textarea name="nota" maxlength="500" rows="2">{v.nota ?? ''}</textarea>
				</label>
				<details {...abierto(!!(v.cupon || v.referido || data.referido))}>
					<summary>Tengo un cupón o un código de referido</summary>
					<div class="pila" style="margin-top: 10px">
						<div class="fila" style="align-items: flex-end">
							<label class="campo" style="flex: 1">
								Cupón
								<input name="cupon" maxlength="40" autocomplete="off" autocapitalize="characters" value={v.cupon ?? ''} />
							</label>
							<button class="boton" formaction={accion('cupon')} formnovalidate>Aplicar</button>
						</div>
						{#if cupon}
							<p class={cupon.valido ? 'aviso' : 'aviso error'} role="status">
								{cupon.valido ? `Cupón aplicado: ${pesos(descuento)} de descuento.` : cupon.motivo || 'Ese cupón no sirve.'}
							</p>
						{/if}
						<label class="campo">
							Código de referido
							<input name="referido" maxlength="40" autocomplete="off" autocapitalize="characters" value={v.referido ?? data.referido} />
							<span class="suave chico">Si te recomendó alguien, poné su&nbsp;código.</span>
						</label>
					</div>
				</details>

				<div class="tarjeta pila total">
					<div class="fila" style="justify-content: space-between">
						<span>{data.servicio.nombre}</span><span>{pesos(precio)}</span>
					</div>
					{#if descuento > 0}
						<div class="fila" style="justify-content: space-between">
							<span>Descuento</span><span>−{pesos(descuento)}</span>
						</div>
					{/if}
					<div class="fila" style="justify-content: space-between">
						<strong>Total</strong><strong>{pesos(precio - descuento)}</strong>
					</div>
					{#if cobroTexto}<p class="chico" style="margin: 0">{cobroTexto}</p>{/if}
					{#if data.politicaCancelacion}<p class="suave chico" style="margin: 0; white-space: pre-line">{data.politicaCancelacion}</p>{/if}
				</div>

				<button class="boton primario grande" disabled={enviando}>
					{enviando ? 'Reservando…' : cobroTexto ? 'Confirmar e ir a pagar' : 'Confirmar turno'}
				</button>
			</form>
		</section>
	{/if}
</main>

<style>
	.reserva {
		max-width: 640px;
	}
	.pasos {
		display: flex;
		gap: 4px;
		list-style: none;
		padding: 0;
		margin: 0;
		font-size: 12px;
		color: var(--suave);
		counter-reset: paso;
	}
	.pasos li {
		flex: 1;
		padding-top: 6px;
		border-top: 3px solid var(--borde);
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.pasos li.hecho {
		border-color: var(--marca);
	}
	.pasos li.actual {
		border-color: var(--marca);
		color: var(--tinta);
		font-weight: 600;
	}
	.resumen {
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.resumen div {
		display: grid;
		grid-template-columns: 90px 1fr auto;
		gap: 8px;
		align-items: baseline;
	}
	.resumen dt {
		color: var(--suave);
		font-size: 13px;
	}
	.resumen dd {
		margin: 0;
	}
	.resumen a {
		font-size: 13px;
	}
	.opciones {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.opcion {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 12px;
		text-decoration: none;
		min-height: 56px;
	}
	.opcion:hover {
		border-color: var(--marca);
	}
	.precio {
		font-weight: 600;
		white-space: nowrap;
	}
	.foto {
		border-radius: 50%;
		object-fit: cover;
	}
	.grande {
		min-height: 48px;
	}
	summary {
		cursor: pointer;
		min-height: 32px;
	}
	@media (max-width: 420px) {
		.resumen div {
			grid-template-columns: 1fr auto;
		}
		.resumen dt {
			grid-column: 1 / -1;
		}
	}
</style>
