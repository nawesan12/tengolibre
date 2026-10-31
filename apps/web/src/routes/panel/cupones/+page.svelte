<script lang="ts">
	import { enhance } from '$app/forms';
	import { fecha, pesos } from '#lib/formato.ts';
	import { descuento, instanteComoDia } from '#lib/configuracion/dinero.ts';
	import { confirmar } from '#lib/configuracion/formularios.ts';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';
	import Pestanias from '#lib/configuracion/Pestanias.svelte';
	import type { Cupon } from './+page.server.ts';

	let { data, form } = $props();
	let editando = $state<string | null>(null);
	const propios = $derived(data.cupones.filter((c) => !c.referidoId));
	const deReferidos = $derived(data.cupones.length - propios.length);
</script>

<svelte:head><title>Cupones · Panel</title></svelte:head>

{#snippet campos(c: Partial<Cupon>)}
	<label class="campo">
		Tipo
		<select name="tipo">
			<option value="porcentaje" selected={c.tipo !== 'fijo'}>Porcentaje</option>
			<option value="fijo" selected={c.tipo === 'fijo'}>Monto fijo</option>
		</select>
	</label>
	<label class="campo">Descuento <input name="valor" type="number" min="1" value={c.valor ?? ''} required /></label>
	<label class="campo">Compra mínima ($) <input name="minimo" type="number" min="0" value={c.minimo ?? ''} placeholder="Sin mínimo" /></label>
	<label class="campo">Usos máximos <input name="maxUsos" type="number" min="1" value={c.maxUsos ?? ''} placeholder="Sin límite" /></label>
	<label class="campo">Desde <input name="desde" type="date" value={instanteComoDia(c.desde)} /></label>
	<label class="campo">Hasta <input name="hasta" type="date" value={instanteComoDia(c.hasta)} /></label>
	<label class="fila chico"><input type="checkbox" name="activo" checked={c.activo ?? true} /> Activo</label>
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

	<section class="tarjeta pila" aria-labelledby="t-nuevo">
		<h2 id="t-nuevo">Nuevo cupón</h2>
		<p class="suave chico">El cliente escribe el código al reservar y el descuento se aplica sobre el precio del&nbsp;servicio.</p>
		<Mensaje {form} que="crear" />
		<form method="POST" action="?/crear" use:enhance class="grilla">
			<label class="campo">
				Código
				<input
					name="codigo"
					required
					minlength="3"
					maxlength="40"
					pattern="[A-Za-z0-9_\-]+"
					autocapitalize="characters"
					autocomplete="off"
					placeholder="VERANO25"
					style="text-transform: uppercase"
				/>
			</label>
			{@render campos({})}
			<div class="fila" style="align-self: end"><button class="boton primario">Crear cupón</button></div>
		</form>
	</section>

	<section class="tarjeta pila" aria-labelledby="t-cupones">
		<h2 id="t-cupones">Tus cupones</h2>
		<Mensaje {form} que="cupones" />
		{#if propios.length === 0}
			<p class="suave">Todavía no creaste cupones.</p>
		{:else}
			<ul class="lista">
				{#each propios as c (c.id)}
					<li class="pila">
						{#if editando === c.id}
							<form
								method="POST"
								action="?/editar"
								use:enhance={() => async ({ update, result }) => {
									await update({ reset: false });
									if (result.type === 'success') editando = null;
								}}
								class="grilla"
							>
								<input type="hidden" name="id" value={c.id} />
								<strong style="align-self: center">{c.codigo}</strong>
								{@render campos(c)}
								<div class="fila">
									<button class="boton primario">Guardar</button>
									<button type="button" class="boton" onclick={() => (editando = null)}>Cancelar</button>
								</div>
							</form>
						{:else}
							<div class="fila entre">
								<div>
									<code class="codigo">{c.codigo}</code>
									<strong>{descuento(c.tipo, c.valor, pesos)}</strong>
									{#if !c.activo}<span class="etiqueta">Pausado</span>{/if}
									<div class="suave chico">
										{c.usos}{c.maxUsos ? ` de ${c.maxUsos}` : ''} usos
										{#if c.minimo}· desde {pesos(c.minimo)}{/if}
										{#if c.hasta}· hasta el {fecha(c.hasta)}{/if}
									</div>
								</div>
								<div class="fila">
									<button class="boton" onclick={() => (editando = c.id)}>Editar</button>
									<form method="POST" action="?/activo" use:enhance>
										<input type="hidden" name="id" value={c.id} />
										<input type="hidden" name="activo" value={String(!c.activo)} />
										<button class="boton">{c.activo ? 'Pausar' : 'Activar'}</button>
									</form>
									<form method="POST" action="?/borrar" use:enhance={confirmar(`¿Sacar el cupón ${c.codigo}?`)}>
										<input type="hidden" name="id" value={c.id} />
										<button class="boton peligro">Sacar</button>
									</form>
								</div>
							</div>
						{/if}
					</li>
				{/each}
			</ul>
		{/if}
		{#if deReferidos > 0}
			<p class="suave chico">Además hay {deReferidos} cupones de premio del <a href="/panel/referidos">programa de&nbsp;referidos</a>.</p>
		{/if}
	</section>
</div>

<style>
	.entre {
		justify-content: space-between;
	}
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.lista li {
		padding: 10px 0;
		border-bottom: 1px solid var(--borde);
	}
	.codigo {
		font-weight: 650;
		margin-right: 6px;
	}
</style>
