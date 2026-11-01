<script lang="ts">
	import '#lib/gestion/gestion.css';
	import { enhance } from '$app/forms';
	import { tick } from 'svelte';
	import { pesos } from '#lib/formato.ts';
	import { confirmar, sinReset } from '#lib/configuracion/formularios.ts';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';
	import CuponFila from '#lib/gestion/cobros/Cupon.svelte';
	import Visitas from '#lib/gestion/cobros/Visitas.svelte';
	import { fechaCorta } from '#lib/gestion/cobros/formato.ts';
	import type { Plan } from './+page.server.ts';

	let { data, form } = $props();
	const esGestion = $derived(['duenio', 'encargado'].includes(data.negocio.rol));
	const planesActivos = $derived(data.planes.filter((p) => p.activo));
	let editando = $state<string | null>(null);
	// svelte-ignore state_referenced_locally
	let nuevoPlan = $state(data.planes.length === 0);
	let formPlan = $state<HTMLElement>();
	const nombreServicio = $derived(new Map(data.servicios.map((s) => [s.id, s.nombre])));

	const FILTROS = [
		{ clave: 'activa', nombre: 'Activas' },
		{ clave: 'vencida', nombre: 'Vencidas' },
		{ clave: 'cancelada', nombre: 'Canceladas' }
	];
	const VACIO: Record<string, string> = {
		activa: 'No hay membresías activas.',
		vencida: 'No hay membresías vencidas.',
		cancelada: 'No hay membresías canceladas.',
		todas: 'Todavía no hay socios.'
	};

	const periodo = (d: number) => (d === 30 ? 'por mes' : d === 7 ? 'por semana' : `cada ${d} días`);
	function chips(p: Plan) {
		const c = [p.visitas === 0 ? 'Visitas libres' : `${p.visitas} visitas`, `${p.duracionDias} días`];
		if (p.prioridadDias > 0) c.push(`Prioridad ${p.prioridadDias} d`);
		for (const id of p.servicios) {
			const n = nombreServicio.get(id);
			if (n) c.push(n);
		}
		return c;
	}

	async function abrirNuevoPlan() {
		nuevoPlan = true;
		editando = null;
		await tick();
		formPlan?.scrollIntoView({ behavior: 'smooth', block: 'start' });
		formPlan?.querySelector('input')?.focus();
	}
</script>

<svelte:head><title>Membresías · Panel</title></svelte:head>

{#snippet camposPlan(p: Partial<Plan>)}
	<label class="g-campo">Nombre <input name="nombre" value={p.nombre ?? ''} required minlength="2" maxlength="60" /></label>
	<label class="g-campo">Precio por período ($) <input name="precio" type="number" min="1" value={p.precio ?? ''} required /></label>
	<label class="g-campo">
		Visitas por período
		<input name="visitas" type="number" min="0" max="1000" value={p.visitas ?? 4} required />
		<span class="g-ayuda">0: sin límite.</span>
	</label>
	<label class="g-campo">Duración (días) <input name="duracionDias" type="number" min="1" max="365" value={p.duracionDias ?? 30} required /></label>
	<label class="g-campo">
		Reserva antes (días)
		<input name="prioridadDias" type="number" min="0" max="90" value={p.prioridadDias ?? 0} />
		<span class="g-ayuda">Días extra para reservar antes que el&nbsp;resto.</span>
	</label>
	<label class="g-campo">Cupo de socios <input name="topeSocios" type="number" min="1" value={p.topeSocios ?? ''} placeholder="Sin cupo" /></label>
	<label class="g-campo g-ancho">Descripción <textarea name="descripcion" rows="2" maxlength="400">{p.descripcion ?? ''}</textarea></label>
	<label class="g-campo g-ancho">
		Beneficios (uno por renglón)
		<textarea name="beneficios" rows="3">{(p.beneficios ?? []).join('\n')}</textarea>
	</label>
	{#if data.servicios.length > 0}
		<fieldset class="g-ancho servicios">
			<legend class="g-campo">Servicios que cubre (ninguno: todos)</legend>
			<div class="g-chips">
				{#each data.servicios as s (s.id)}
					<label class="g-chip"><input type="checkbox" name="servicios" value={s.id} checked={p.servicios?.includes(s.id)} />{s.nombre}</label>
				{/each}
			</div>
		</fieldset>
	{/if}
	<label class="g-check g-ancho"><input type="checkbox" name="activo" checked={p.activo ?? true} /> Se ofrece en la&nbsp;página</label>
{/snippet}

<div class="g-pagina">
	<div class="g-cabeza">
		<h1 class="g-titulo">Membresías y&nbsp;cupones</h1>
		{#if esGestion}<button type="button" class="g-boton oscuro" onclick={abrirNuevoPlan}>Nuevo plan</button>{/if}
	</div>

	<section aria-labelledby="t-planes" class="g-col">
		<h2 id="t-planes" class="visualmente-oculto">Planes</h2>
		<Mensaje {form} que="planes" />
		{#if data.planes.length > 0}
			<div class="planes">
				{#each data.planes as p (p.id)}
					{#if editando === p.id}
						<form
							method="POST"
							action="?/editarPlan"
							use:enhance={() => async ({ update, result }) => {
								await update({ reset: false });
								if (result.type === 'success') editando = null;
							}}
							class="g-bloque g-form plan-form"
							aria-label="Editar {p.nombre}"
						>
							<input type="hidden" name="id" value={p.id} />
							{@render camposPlan(p)}
							<div class="fila g-ancho">
								<button class="g-boton oscuro">Guardar</button>
								<button type="button" class="g-accion neutro" onclick={() => (editando = null)}>Cancelar</button>
							</div>
						</form>
					{:else}
						{@const socias = data.porPlan[p.id] ?? 0}
						<article class="plan" class:pausado={!p.activo}>
							<div class="plan-izq">
								<span class="plan-estado">
									<span class="icono lleno" aria-hidden="true">{p.activo ? 'check_circle' : 'pause_circle'}</span>{p.activo ? 'Activo' : 'Pausado'}
								</span>
								<h3 class="plan-nombre">{p.nombre}</h3>
								{#if p.descripcion}<p class="plan-desc">{p.descripcion}</p>{/if}
								<span class="plan-chips">
									{#each chips(p) as c (c)}<span class="g-etiquetita">{c}</span>{/each}
								</span>
								{#if esGestion}
									<span class="plan-acciones">
										<button class="g-accion" onclick={() => ((editando = p.id), (nuevoPlan = false))}>Editar</button>
										<form method="POST" action="?/borrarPlan" use:enhance={confirmar(`¿Sacar el plan ${p.nombre}?`)}>
											<input type="hidden" name="id" value={p.id} />
											<button class="g-accion peligro">Sacar</button>
										</form>
									</span>
								{/if}
							</div>
							<div class="plan-der">
								<b class="plan-precio num">{pesos(p.precio)}</b>
								<span class="plan-periodo">{periodo(p.duracionDias)}</span>
								{#if p.topeSocios}
									<span class="plan-cupo" aria-hidden="true"><span style:width="{Math.min(100, (socias / p.topeSocios) * 100)}%"></span></span>
									<span class="plan-cuenta">{socias} de {p.topeSocios} socias</span>
								{:else}
									<span class="plan-cuenta">{socias} {socias === 1 ? 'socia' : 'socias'} · sin cupo</span>
								{/if}
							</div>
						</article>
					{/if}
				{/each}
			</div>
		{/if}
		{#if esGestion && nuevoPlan}
			<div class="g-bloque" bind:this={formPlan}>
				<div class="g-bloque-cabeza">
					<h3 class="g-bloque-titulo">Nuevo plan</h3>
					{#if data.planes.length > 0}<button type="button" class="g-accion neutro" onclick={() => (nuevoPlan = false)}>Cerrar</button>{/if}
				</div>
				<p class="g-nota">Un abono con visitas incluidas. Los planes activos se ven en tu&nbsp;página.</p>
				<form method="POST" action="?/crearPlan" use:enhance class="g-form">
					{@render camposPlan({})}
					<div class="g-ancho"><button class="g-cta">Crear plan</button></div>
				</form>
			</div>
		{/if}
	</section>

	<div class="abajo">
		<section class="g-bloque sin-pad" aria-labelledby="t-socios">
			<div class="g-bloque-cabeza">
				<h2 id="t-socios" class="g-bloque-titulo">Socios</h2>
				<nav class="g-chips" aria-label="Estado de los socios">
					{#each FILTROS as f (f.clave)}
						<a class="g-chip chico" href="?estado={f.clave}" aria-current={data.estado === f.clave ? 'page' : undefined}>
							{f.nombre}&nbsp;<span class="num">{data.cuenta[f.clave] ?? 0}</span>
						</a>
					{/each}
				</nav>
			</div>
			<div class="pad"><Mensaje {form} que="socios" /></div>
			{#if planesActivos.length > 0}
				<details class="g-plegable alta">
					<summary>+ Sumar un socio</summary>
					<form method="POST" action="?/alta" use:enhance class="g-form">
						<label class="g-campo">
							Plan
							<select name="planId" required>
								{#each planesActivos as p (p.id)}<option value={p.id}>{p.nombre} · {pesos(p.precio)}</option>{/each}
							</select>
						</label>
						<label class="g-campo">Nombre <input name="nombre" required maxlength="80" autocomplete="off" /></label>
						<label class="g-campo">Teléfono <input name="telefono" type="tel" required maxlength="30" autocomplete="off" /></label>
						<label class="g-campo">Email (opcional) <input name="email" type="email" maxlength="120" /></label>
						<label class="g-campo g-ancho">Nota (opcional) <input name="nota" maxlength="300" /></label>
						<div class="g-ancho"><button class="g-boton oscuro">Sumar socio</button></div>
					</form>
				</details>
			{:else if data.planes.length === 0}
				<p class="g-nota pad">Para sumar socios, primero creá un&nbsp;plan.</p>
			{/if}
			{#if data.socios.length === 0}
				<p class="g-vacio"><span class="icono" aria-hidden="true">card_membership</span>{VACIO[data.estado]}</p>
			{:else}
				<ul class="g-filas">
					{#each data.socios as s (s.id)}
						<li class="socio">
							<a class="g-fila-texto socio-nombre" href="/panel/membresias/{s.id}">
								<b>{s.cliente}</b>
								<span>{s.plan} · {s.estado === 'cancelada' ? 'cancelada' : s.estado === 'vencida' ? 'venció' : 'vence'} {fechaCorta(s.hasta)}</span>
							</a>
							<Visitas quedan={s.quedan} total={s.visitasDelPlan} />
							<span class="socio-accion">
								{#if s.estado === 'cancelada'}
									<form method="POST" action="?/estado" use:enhance>
										<input type="hidden" name="id" value={s.id} />
										<input type="hidden" name="estado" value="activa" />
										<button class="g-accion">Reactivar</button>
									</form>
								{:else if s.estado === 'vencida' || (s.visitasDelPlan > 0 && s.quedan <= 1)}
									<form method="POST" action="?/renovar" use:enhance={confirmar(`¿Renovar la membresía de ${s.cliente} desde hoy?`)}>
										<input type="hidden" name="id" value={s.id} />
										<button class="g-accion">Renovar</button>
									</form>
								{:else}
									<a class="g-accion" href="/panel/membresias/{s.id}">Ajustar</a>
								{/if}
							</span>
						</li>
					{/each}
				</ul>
			{/if}
		</section>

		{#if data.cupones}
			<section class="g-bloque sin-pad" aria-labelledby="t-cupones-resumen">
				<div class="g-bloque-cabeza">
					<h2 id="t-cupones-resumen" class="g-bloque-titulo">Cupones</h2>
					<a class="g-accion" href="/panel/cupones#nuevo-cupon">+ Nuevo</a>
				</div>
				{#if data.cupones.length === 0}
					<p class="g-vacio"><span class="icono" aria-hidden="true">sell</span>Todavía no creaste&nbsp;cupones.</p>
				{:else}
					<ul class="g-filas">
						{#each data.cupones.slice(0, 5) as c (c.id)}<li><CuponFila {c} /></li>{/each}
					</ul>
					<a class="g-accion todos" href="/panel/cupones">Ver y editar los cupones</a>
				{/if}
			</section>
		{/if}
	</div>
</div>

<style>
	.planes {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 14px;
	}
	.plan-form {
		grid-column: 1 / -1;
	}
	.plan {
		display: grid;
		grid-template-columns: minmax(0, 1fr) 150px;
		border-radius: 20px;
		background: #f3e8d0;
		color: #4a3a17;
	}
	.plan.pausado :is(.plan-nombre, .plan-desc, .plan-chips, .plan-der) {
		opacity: 0.7;
	}
	.plan-izq {
		padding: 18px;
		display: flex;
		flex-direction: column;
		gap: 6px;
		border-right: 2px dashed #dccaa2;
		min-width: 0;
	}
	.plan-estado {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		font: 700 11px var(--tl-font);
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: #6b4e17;
	}
	.plan-estado .icono {
		font-size: 14px;
	}
	.plan-nombre {
		margin: 0;
		font-size: 22px;
		font-weight: 700;
		letter-spacing: -0.03em;
		color: var(--tl-tinta);
	}
	.plan-desc {
		margin: 0;
		font-size: 13px;
		line-height: 1.45;
	}
	.plan-chips {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
		margin-top: 4px;
	}
	.plan-chips .g-etiquetita {
		background: rgba(255, 253, 249, 0.6);
		color: #4a3a17;
	}
	.plan-acciones {
		display: flex;
		gap: 16px;
		margin-top: 4px;
	}
	.plan-der {
		padding: 18px 14px;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 8px;
		text-align: center;
	}
	.plan-precio {
		font-size: 22px;
		letter-spacing: -0.03em;
		color: var(--tl-tinta);
	}
	.plan-periodo {
		font-size: 12px;
	}
	.plan-cupo {
		width: 100%;
		height: 6px;
		border-radius: 3px;
		background: rgba(122, 90, 30, 0.18);
		overflow: hidden;
	}
	.plan-cupo span {
		display: block;
		height: 100%;
		background: #7a5a1e;
	}
	.plan-cuenta {
		font: 400 11px var(--tl-font);
		font-variant-numeric: tabular-nums;
		text-transform: uppercase;
	}
	.servicios {
		border: 0;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.servicios legend {
		padding: 0;
		margin-bottom: 8px;
	}
	.abajo {
		display: grid;
		grid-template-columns: minmax(0, 1.4fr) minmax(0, 1fr);
		gap: 16px;
		align-items: start;
	}
	.g-chip.chico {
		min-height: 32px;
		padding: 0 12px;
		font-size: 12px;
	}
	.pad:empty {
		display: none;
	}
	.pad {
		padding: 0 18px 12px;
	}
	.alta {
		padding: 0 18px 12px;
	}
	.alta > summary {
		color: var(--tl-menta-texto);
	}
	.alta > summary::after {
		content: none;
	}
	.alta form {
		margin-top: 8px;
	}
	.socio {
		display: grid;
		grid-template-columns: minmax(0, 1fr) 120px 70px auto;
		align-items: center;
		gap: 12px;
		min-height: 52px;
		padding: 6px 18px;
		position: relative;
	}
	.socio-nombre {
		color: inherit;
		text-decoration: none;
	}
	.socio-nombre:hover b {
		text-decoration: underline;
		text-underline-offset: 3px;
	}
	.socio-accion {
		justify-self: end;
	}
	.todos {
		margin: 4px 18px 14px;
	}
	@media (max-width: 1000px) {
		.planes,
		.abajo {
			grid-template-columns: minmax(0, 1fr);
		}
	}
	@media (max-width: 640px) {
		.plan {
			grid-template-columns: minmax(0, 1fr) 118px;
		}
		.plan-izq {
			padding: 16px 14px;
		}
		.socio {
			grid-template-columns: minmax(0, 1fr) auto;
			gap: 4px 12px;
			padding: 10px 14px;
		}
		.socio-accion {
			grid-row: 1 / 3;
			grid-column: 2;
		}
		.g-chip.chico {
			min-height: 44px;
		}
	}
</style>
