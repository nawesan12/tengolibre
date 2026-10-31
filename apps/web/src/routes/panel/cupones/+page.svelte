<script lang="ts">
	import '#lib/gestion/gestion.css';
	import { enhance } from '$app/forms';
	import { instanteComoDia } from '#lib/configuracion/dinero.ts';
	import { confirmar } from '#lib/configuracion/formularios.ts';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';
	import CuponFila from '#lib/gestion/cobros/Cupon.svelte';
	import type { Cupon } from './+page.server.ts';

	let { data, form } = $props();
	let editando = $state<string | null>(null);
	let codigo = $state<HTMLInputElement>();
	const propios = $derived(data.cupones.filter((c) => !c.referidoId));
	const deReferidos = $derived(data.cupones.length - propios.length);
</script>

<svelte:head><title>Cupones · Panel</title></svelte:head>

{#snippet campos(c: Partial<Cupon>)}
	<label class="g-campo">
		Tipo
		<select name="tipo">
			<option value="porcentaje" selected={c.tipo !== 'fijo'}>Porcentaje</option>
			<option value="fijo" selected={c.tipo === 'fijo'}>Monto fijo</option>
		</select>
	</label>
	<label class="g-campo">Descuento <input name="valor" type="number" min="1" value={c.valor ?? ''} required /></label>
	<label class="g-campo">Compra mínima ($) <input name="minimo" type="number" min="0" value={c.minimo ?? ''} placeholder="Sin mínimo" /></label>
	<label class="g-campo">Usos máximos <input name="maxUsos" type="number" min="1" value={c.maxUsos ?? ''} placeholder="Sin límite" /></label>
	<label class="g-campo">Desde <input name="desde" type="date" value={instanteComoDia(c.desde)} /></label>
	<label class="g-campo">Hasta <input name="hasta" type="date" value={instanteComoDia(c.hasta)} /></label>
	<label class="g-check g-ancho"><input type="checkbox" name="activo" checked={c.activo ?? true} /> Activo: se puede usar al&nbsp;reservar</label>
{/snippet}

<div class="g-pagina">
	<div class="g-cabeza">
		<div class="titulos">
			<a class="g-kicker volver" href="/panel/membresias"><span class="icono" aria-hidden="true">arrow_back</span>Membresías y cupones</a>
			<h1 class="g-titulo">Cupones</h1>
		</div>
	</div>

	<div class="grilla">
		<section class="g-bloque sin-pad" aria-labelledby="t-cupones">
			<div class="g-bloque-cabeza">
				<h2 id="t-cupones" class="g-bloque-titulo">Tus cupones</h2>
				<button type="button" class="g-accion" onclick={() => codigo?.focus()}>+ Nuevo</button>
			</div>
			<div class="pad"><Mensaje {form} que="cupones" /></div>
			{#if propios.length === 0}
				<p class="g-vacio"><span class="icono" aria-hidden="true">sell</span>Todavía no creaste&nbsp;cupones.</p>
			{:else}
				<ul class="g-filas">
					{#each propios as c (c.id)}
						<li>
							{#if editando === c.id}
								<form
									method="POST"
									action="?/editar"
									use:enhance={() => async ({ update, result }) => {
										await update({ reset: false });
										if (result.type === 'success') editando = null;
									}}
									class="g-form editar"
								>
									<input type="hidden" name="id" value={c.id} />
									<b class="codigo g-ancho">{c.codigo}</b>
									{@render campos(c)}
									<div class="fila g-ancho">
										<button class="g-boton oscuro">Guardar</button>
										<button type="button" class="g-accion neutro" onclick={() => (editando = null)}>Cancelar</button>
									</div>
								</form>
							{:else}
								<CuponFila {c}>
									{#snippet acciones()}
										<button class="g-accion" onclick={() => (editando = c.id)}>Editar</button>
										<form method="POST" action="?/activo" use:enhance>
											<input type="hidden" name="id" value={c.id} />
											<input type="hidden" name="activo" value={String(!c.activo)} />
											<button class="g-accion neutro">{c.activo ? 'Pausar' : 'Activar'}</button>
										</form>
										<form method="POST" action="?/borrar" use:enhance={confirmar(`¿Sacar el cupón ${c.codigo}?`)}>
											<input type="hidden" name="id" value={c.id} />
											<button class="g-accion peligro">Sacar</button>
										</form>
									{/snippet}
								</CuponFila>
							{/if}
						</li>
					{/each}
				</ul>
			{/if}
			{#if deReferidos > 0}
				<p class="g-nota pad">Además hay {deReferidos} cupones de premio del <a href="/panel/referidos">programa de&nbsp;referidos</a>.</p>
			{/if}
		</section>

		<section class="g-bloque" aria-labelledby="t-nuevo" id="nuevo-cupon">
			<h2 id="t-nuevo" class="g-bloque-titulo">Nuevo cupón</h2>
			<p class="g-nota">La clienta escribe el código al reservar y el descuento se aplica sobre el precio del&nbsp;servicio.</p>
			<Mensaje {form} que="crear" />
			<form method="POST" action="?/crear" use:enhance class="g-form">
				<label class="g-campo g-ancho">
					Código
					<input
						bind:this={codigo}
						name="codigo"
						class="entrada-codigo"
						required
						minlength="3"
						maxlength="40"
						pattern="[A-Za-z0-9_\-]+"
						autocapitalize="characters"
						autocomplete="off"
						placeholder="VERANO25"
					/>
				</label>
				{@render campos({})}
				<div class="g-ancho"><button class="g-cta">Crear cupón</button></div>
			</form>
		</section>
	</div>
</div>

<style>
	.titulos {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.volver {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		min-height: 32px;
		text-decoration: none;
	}
	.volver .icono {
		font-size: 16px;
	}
	.grilla {
		display: grid;
		grid-template-columns: minmax(0, 1.2fr) minmax(0, 1fr);
		gap: 16px;
		align-items: start;
	}
	.pad:empty {
		display: none;
	}
	.pad {
		padding: 0 18px 12px;
	}
	.editar {
		padding: 14px 18px;
	}
	.codigo {
		font: 700 15px var(--tl-font);
		letter-spacing: 0.08em;
	}
	.entrada-codigo {
		text-transform: uppercase;
		letter-spacing: 0.08em;
		font-weight: 700 !important;
	}
	@media (max-width: 860px) {
		.grilla {
			grid-template-columns: minmax(0, 1fr);
		}
	}
</style>
