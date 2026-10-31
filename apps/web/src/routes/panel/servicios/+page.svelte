<script lang="ts">
	import { enhance } from '$app/forms';
	import { pesos } from '#lib/formato.ts';
	let { data, form } = $props();
	let editando = $state<string | null>(null);
	const nombreCategoria = (id: string | null) => data.categorias.find((c) => c.id === id)?.nombre ?? 'Sin categoría';
</script>

<svelte:head><title>Servicios · Panel</title></svelte:head>

<div class="pila">
	<h1>Servicios</h1>
	{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}

	<section class="tarjeta pila">
		<h2>Nuevo servicio</h2>
		<form method="POST" action="?/crear" use:enhance class="grilla">
			<label class="campo">Nombre <input name="nombre" required minlength="2" maxlength="80" /></label>
			<label class="campo">Duración (minutos) <input name="duracion" type="number" min="5" max="480" step="5" value="30" required /></label>
			<label class="campo">Precio <input name="precio" type="number" min="0" step="1" required /></label>
			<label class="campo">
				Categoría
				<select name="categoriaId">
					<option value="">Sin categoría</option>
					{#each data.categorias as c (c.id)}<option value={c.id}>{c.nombre}</option>{/each}
				</select>
			</label>
			<label class="campo">Descripción <input name="descripcion" maxlength="1000" /></label>
			<label class="fila chico"><input type="checkbox" name="activo" checked /> Se puede reservar</label>
			<button class="boton primario">Agregar</button>
		</form>
	</section>

	<section class="tarjeta pila">
		<h2>Tus servicios</h2>
		{#if data.servicios.length === 0}
			<p class="suave">Todavía no cargaste servicios.</p>
		{/if}
		{#each data.servicios as s (s.id)}
			<article class="pila" style="border-bottom: 1px solid var(--borde); padding-bottom: 12px">
				{#if editando === s.id}
					<form method="POST" action="?/editar" use:enhance={() => async ({ update }) => { await update(); editando = null; }} class="grilla">
						<input type="hidden" name="id" value={s.id} />
						<label class="campo">Nombre <input name="nombre" value={s.nombre} required /></label>
						<label class="campo">Duración <input name="duracion" type="number" min="5" max="480" step="5" value={s.duracion} /></label>
						<label class="campo">Precio <input name="precio" type="number" min="0" value={s.precio} /></label>
						<label class="campo">
							Categoría
							<select name="categoriaId">
								<option value="">Sin categoría</option>
								{#each data.categorias as c (c.id)}<option value={c.id} selected={c.id === s.categoriaId}>{c.nombre}</option>{/each}
							</select>
						</label>
						<label class="campo">Descripción <input name="descripcion" value={s.descripcion} /></label>
						<label class="fila chico"><input type="checkbox" name="activo" checked={s.activo} /> Se puede reservar</label>
						<div class="fila">
							<button class="boton primario">Guardar</button>
							<button type="button" class="boton" onclick={() => (editando = null)}>Cancelar</button>
						</div>
					</form>
				{:else}
					<div class="fila" style="justify-content: space-between">
						<div>
							<strong>{s.nombre}</strong>
							{#if !s.activo}<span class="etiqueta">Pausado</span>{/if}
							<div class="suave chico">{nombreCategoria(s.categoriaId)} · {s.duracion} min · {pesos(s.precio)}</div>
						</div>
						<div class="fila">
							<button class="boton" onclick={() => (editando = s.id)}>Editar</button>
							<form method="POST" action="?/borrar" use:enhance>
								<input type="hidden" name="id" value={s.id} />
								<button class="boton peligro">Borrar</button>
							</form>
						</div>
					</div>
				{/if}
			</article>
		{/each}
	</section>

	<section class="tarjeta pila">
		<h2>Categorías</h2>
		<div class="fila">
			{#each data.categorias as c (c.id)}
				<form method="POST" action="?/borrarCategoria" use:enhance class="fila">
					<input type="hidden" name="id" value={c.id} />
					<span class="etiqueta">{c.nombre}</span>
					<button class="boton chico" aria-label={`Borrar ${c.nombre}`}>×</button>
				</form>
			{/each}
		</div>
		<form method="POST" action="?/categoria" use:enhance class="fila">
			<label class="campo">Nueva categoría <input name="nombre" required maxlength="60" /></label>
			<button class="boton">Agregar</button>
		</form>
	</section>
</div>
