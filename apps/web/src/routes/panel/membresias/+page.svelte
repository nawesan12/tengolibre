<script lang="ts">
	import { enhance } from '$app/forms';
	import { fecha, pesos } from '#lib/formato.ts';
	import { ESTADOS_SOCIO } from '#lib/configuracion/textos.ts';
	import { confirmar, sinReset } from '#lib/configuracion/formularios.ts';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';
	import Pestanias from '#lib/configuracion/Pestanias.svelte';
	import type { Plan } from './+page.server.ts';

	let { data, form } = $props();
	const esGestion = $derived(['duenio', 'encargado'].includes(data.negocio.rol));
	const planesActivos = $derived(data.planes.filter((p) => p.activo));
	let editando = $state<string | null>(null);
	let ajustando = $state<string | null>(null);
	const visitas = (n: number) => (n === 0 ? 'visitas libres' : `${n} visitas`);
</script>

<svelte:head><title>Membresías · Panel</title></svelte:head>

{#snippet camposPlan(p: Partial<Plan>)}
	<label class="campo">Nombre <input name="nombre" value={p.nombre ?? ''} required minlength="2" maxlength="60" /></label>
	<label class="campo">Precio por período ($) <input name="precio" type="number" min="1" value={p.precio ?? ''} required /></label>
	<label class="campo">
		Visitas por período
		<input name="visitas" type="number" min="0" max="1000" value={p.visitas ?? 4} required />
		<span class="suave chico">0: sin límite.</span>
	</label>
	<label class="campo">Duración (días) <input name="duracionDias" type="number" min="1" max="365" value={p.duracionDias ?? 30} required /></label>
	<label class="campo">
		Reserva antes (días)
		<input name="prioridadDias" type="number" min="0" max="90" value={p.prioridadDias ?? 0} />
		<span class="suave chico">Días extra para reservar antes que el resto.</span>
	</label>
	<label class="campo">Cupo de socios <input name="topeSocios" type="number" min="1" value={p.topeSocios ?? ''} placeholder="Sin cupo" /></label>
	<label class="campo ancho">Descripción <textarea name="descripcion" rows="2" maxlength="400">{p.descripcion ?? ''}</textarea></label>
	<label class="campo ancho">
		Beneficios (uno por renglón)
		<textarea name="beneficios" rows="3">{(p.beneficios ?? []).join('\n')}</textarea>
	</label>
	{#if data.servicios.length > 0}
		<fieldset class="campo ancho caja">
			<legend>Servicios que cubre (ninguno: todos)</legend>
			<div class="fila">
				{#each data.servicios as s (s.id)}
					<label class="fila chico"><input type="checkbox" name="servicios" value={s.id} checked={p.servicios?.includes(s.id)} /> {s.nombre}</label>
				{/each}
			</div>
		</fieldset>
	{/if}
	<label class="fila chico"><input type="checkbox" name="activo" checked={p.activo ?? true} /> Se ofrece en la página</label>
{/snippet}

<div class="pila">
	<h1>Membresías y cupones</h1>
	<Pestanias
		etiqueta="Membresías y cupones"
		pestanias={[
			{ nombre: 'Membresías', href: '/panel/membresias' },
			{ nombre: 'Cupones', href: '/panel/cupones' }
		]}
	/>

	<section class="tarjeta pila" aria-labelledby="t-socios">
		<div class="fila entre">
			<h2 id="t-socios" style="margin: 0">Socios</h2>
			<nav class="fila chico" aria-label="Estado de los socios">
				{#each ['activa', 'vencida', 'cancelada', 'todas'] as e (e)}
					<a href={`?estado=${e}`} aria-current={data.estado === e ? 'page' : undefined}>{e === 'todas' ? 'Todas' : `${ESTADOS_SOCIO[e]}s`}</a>
				{/each}
			</nav>
		</div>
		<Mensaje {form} que="socios" />
		{#if planesActivos.length > 0}
			<details>
				<summary class="boton">Sumar un socio</summary>
				<form method="POST" action="?/alta" use:enhance class="grilla" style="margin-top: 12px">
					<label class="campo">
						Plan
						<select name="planId" required>
							{#each planesActivos as p (p.id)}<option value={p.id}>{p.nombre} · {pesos(p.precio)}</option>{/each}
						</select>
					</label>
					<label class="campo">Nombre <input name="nombre" required maxlength="80" autocomplete="off" /></label>
					<label class="campo">Teléfono <input name="telefono" type="tel" required maxlength="30" autocomplete="off" /></label>
					<label class="campo">Email (opcional) <input name="email" type="email" maxlength="120" /></label>
					<label class="campo">Nota (opcional) <input name="nota" maxlength="300" /></label>
					<div class="fila" style="align-self: end"><button class="boton primario">Sumar socio</button></div>
				</form>
			</details>
		{/if}
		{#if data.socios.length === 0}
			<p class="suave">No hay membresías {data.estado === 'todas' ? '' : `${ESTADOS_SOCIO[data.estado]?.toLowerCase()}s`}.</p>
		{:else}
			<ul class="lista">
				{#each data.socios as s (s.id)}
					<li class="pila">
						<div class="fila entre">
							<div>
								<strong>{s.cliente}</strong>
								<span class="etiqueta">{ESTADOS_SOCIO[s.estado] ?? s.estado}</span>
								<div class="suave chico">
									{s.plan} · {s.visitasDelPlan === 0 ? 'visitas libres' : `le quedan ${s.quedan}`} · hasta el {fecha(s.hasta)}
									{#if s.telefono}· {s.telefono}{/if}
								</div>
							</div>
							<div class="fila">
								{#if s.estado !== 'cancelada' && s.visitasDelPlan > 0}
									<button class="boton" onclick={() => (ajustando = ajustando === s.id ? null : s.id)} aria-expanded={ajustando === s.id}>
										Ajustar visitas
									</button>
								{/if}
								<form method="POST" action="?/renovar" use:enhance={confirmar(`¿Renovar la membresía de ${s.cliente} desde hoy?`)}>
									<input type="hidden" name="id" value={s.id} />
									<button class="boton">Renovar</button>
								</form>
								<form
									method="POST"
									action="?/estado"
									use:enhance={s.estado === 'cancelada' ? undefined : confirmar(`¿Cancelar la membresía de ${s.cliente}?`)}
								>
									<input type="hidden" name="id" value={s.id} />
									<input type="hidden" name="estado" value={s.estado === 'cancelada' ? 'activa' : 'cancelada'} />
									<button class="boton" class:peligro={s.estado !== 'cancelada'}>{s.estado === 'cancelada' ? 'Reactivar' : 'Cancelar'}</button>
								</form>
							</div>
						</div>
						{#if ajustando === s.id}
							<form
								method="POST"
								action="?/ajustar"
								use:enhance={() => async ({ update, result }) => {
									await update();
									if (result.type === 'success') ajustando = null;
								}}
								class="fila"
							>
								<input type="hidden" name="id" value={s.id} />
								<label class="campo">Visitas (+ suma, − resta) <input name="ajustar" type="number" min="-100" max="100" required /></label>
								<label class="campo">Motivo <input name="motivo" maxlength="120" placeholder="Ajuste manual" /></label>
								<button class="boton primario" style="align-self: end">Aplicar</button>
							</form>
						{/if}
					</li>
				{/each}
			</ul>
		{/if}
	</section>

	{#if esGestion}
		<section class="tarjeta pila" aria-labelledby="t-planes">
			<h2 id="t-planes">Planes</h2>
			<p class="suave chico">Un abono mensual con visitas incluidas. Los planes activos se ven en tu&nbsp;página.</p>
			<Mensaje {form} que="planes" />
			{#each data.planes as p (p.id)}
				<article class="plan pila">
					{#if editando === p.id}
						<form
							method="POST"
							action="?/editarPlan"
							use:enhance={() => async ({ update, result }) => {
								await update({ reset: false });
								if (result.type === 'success') editando = null;
							}}
							class="grilla"
						>
							<input type="hidden" name="id" value={p.id} />
							{@render camposPlan(p)}
							<div class="fila ancho">
								<button class="boton primario">Guardar</button>
								<button type="button" class="boton" onclick={() => (editando = null)}>Cancelar</button>
							</div>
						</form>
					{:else}
						<div class="fila entre">
							<div>
								<strong>{p.nombre}</strong>
								{#if !p.activo}<span class="etiqueta">Pausado</span>{/if}
								<div class="suave chico">{pesos(p.precio)} · {visitas(p.visitas)} · {p.duracionDias} días</div>
							</div>
							<div class="fila">
								<button class="boton" onclick={() => (editando = p.id)}>Editar</button>
								<form method="POST" action="?/borrarPlan" use:enhance={confirmar(`¿Sacar el plan ${p.nombre}?`)}>
									<input type="hidden" name="id" value={p.id} />
									<button class="boton peligro">Sacar</button>
								</form>
							</div>
						</div>
					{/if}
				</article>
			{/each}
			<details open={data.planes.length === 0}>
				<summary class="boton">Nuevo plan</summary>
				<form method="POST" action="?/crearPlan" use:enhance class="grilla" style="margin-top: 12px">
					{@render camposPlan({})}
					<div class="ancho"><button class="boton primario">Crear plan</button></div>
				</form>
			</details>
		</section>
	{/if}
</div>

<style>
	.entre {
		justify-content: space-between;
	}
	.ancho {
		grid-column: 1 / -1;
	}
	.caja {
		border: 1px solid var(--borde);
		border-radius: 8px;
		padding: 8px 10px;
		margin: 0;
	}
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.lista li,
	.plan {
		padding: 10px 0;
		border-bottom: 1px solid var(--borde);
	}
	nav a[aria-current='page'] {
		font-weight: 650;
	}
	summary {
		list-style: none;
		width: fit-content;
	}
	summary::-webkit-details-marker {
		display: none;
	}
</style>
