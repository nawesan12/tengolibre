<!--
	La carta «Productos» de /panel/servicios (docs/api/caja.md): el catálogo que se vende en la caja.
	Mismo lenguaje que las cartas de servicios: filas que se guardan solas, interruptor de activo, una
	fila al final para agregar y una manija que reordena (arrastrando o con las flechas).
-->
<script lang="ts">
	import { enhance, type SubmitFunction } from '$app/forms';
	import { tick } from 'svelte';
	import { confirmar } from '#lib/configuracion/formularios.ts';

	type Producto = { id: string; nombre: string; precio: number; stock: number | null; activo: boolean; orden: number };
	let { productos, guardar }: { productos: Producto[]; guardar: SubmitFunction } = $props();

	const enviar = (e: Event) => (e.currentTarget as HTMLElement).closest('form')?.requestSubmit();
	const miles = (n: number) => n.toLocaleString('es-AR', { maximumFractionDigits: 0 });

	let formOrden = $state<HTMLFormElement>();
	let ordenIds = $state('');
	async function ordenar(ids: string[]) {
		ordenIds = ids.join(',');
		await tick();
		formOrden?.requestSubmit();
	}
	function mover(id: string, paso: number) {
		const ids = productos.map((p) => p.id);
		const i = ids.indexOf(id);
		const j = i + paso;
		if (i < 0 || j < 0 || j >= ids.length) return;
		[ids[i], ids[j]] = [ids[j], ids[i]];
		ordenar(ids);
	}
	let arrastrando = $state<string | null>(null);
	let sobre = $state<string | null>(null);
	function soltar(destino: string) {
		if (!arrastrando || arrastrando === destino) return;
		const desde = productos.findIndex((p) => p.id === arrastrando);
		const hasta = productos.findIndex((p) => p.id === destino);
		const ids = productos.map((p) => p.id).filter((x) => x !== arrastrando);
		ids.splice(ids.indexOf(destino) + (desde < hasta ? 1 : 0), 0, arrastrando);
		ordenar(ids);
		arrastrando = null;
		sobre = null;
	}
</script>

<section class="g-bloque sin-pad carta" aria-labelledby="t-productos">
	<div class="cabeza">
		<span class="icono lleno" aria-hidden="true">shopping_bag</span>
		<h2 id="t-productos" class="titulo">Productos</h2>
		<span class="g-meta">{productos.length} {productos.length === 1 ? 'producto' : 'productos'}</span>
	</div>

	<form method="POST" action="?/ordenarProductos" bind:this={formOrden} use:enhance={guardar} hidden>
		<input type="hidden" name="ids" value={ordenIds} />
	</form>

	<div class="fila encabezado" aria-hidden="true">
		<span></span><span>Nombre</span><span>Stock</span><span>Precio</span><span>Activo</span><span></span>
	</div>

	<ul class="lista">
		{#each productos as p (p.id)}
			<li
				class:inactivo={!p.activo}
				class:sobre={sobre === p.id}
				ondragover={(e) => {
					if (arrastrando) {
						e.preventDefault();
						sobre = p.id;
					}
				}}
				ondrop={(e) => {
					e.preventDefault();
					soltar(p.id);
				}}
			>
				<form method="POST" action="?/editarProducto" use:enhance={guardar} class="fila">
					<input type="hidden" name="id" value={p.id} />
					<input type="hidden" name="conActivo" value="1" />
					<button
						type="button"
						class="manija"
						draggable="true"
						aria-label={`Mover ${p.nombre} (flechas arriba y abajo)`}
						onkeydown={(e) => {
							if (e.key === 'ArrowUp' || e.key === 'ArrowDown') {
								e.preventDefault();
								mover(p.id, e.key === 'ArrowUp' ? -1 : 1);
							}
						}}
						ondragstart={(e) => {
							arrastrando = p.id;
							e.dataTransfer?.setData('text/plain', p.id);
						}}
						ondragend={() => ((arrastrando = null), (sobre = null))}
					>
						<span class="icono" aria-hidden="true">drag_indicator</span>
					</button>
					<input class="celda nombre" name="nombre" value={p.nombre} required minlength="2" maxlength="80" aria-label="Nombre del producto {p.nombre}" onchange={enviar} />
					<span class="celda caja stock">
						<input
							name="stock"
							type="number"
							min="0"
							max="1000000"
							value={p.stock ?? ''}
							placeholder="—"
							aria-label="Stock de {p.nombre} (vacío: sin contar)"
							onchange={enviar}
						/>
						<span aria-hidden="true">u.</span>
					</span>
					<span class="celda caja precio">
						<span aria-hidden="true">$</span>
						<input name="precio" inputmode="numeric" pattern="[0-9.]*" value={miles(p.precio)} required aria-label="Precio de {p.nombre} en pesos" onchange={enviar} />
					</span>
					<span class="activo">
						<input class="g-interruptor" type="checkbox" role="switch" name="activo" checked={p.activo} aria-label="{p.nombre} a la venta" onchange={enviar} />
					</span>
				</form>
				<form method="POST" action="?/borrarProducto" class="borrar" use:enhance={confirmar(`¿Borrar ${p.nombre}? Lo ya vendido conserva nombre y precio.`)}>
					<input type="hidden" name="id" value={p.id} />
					<button class="quitar" aria-label="Borrar {p.nombre}"><span class="icono" aria-hidden="true">delete</span></button>
				</form>
				{#if p.stock === 0}
					<p class="sin-stock"><span class="icono" aria-hidden="true">inventory_2</span>Sin stock: no se puede vender hasta que&nbsp;cargues.</p>
				{/if}
			</li>
		{/each}
	</ul>

	<form
		method="POST"
		action="?/crearProducto"
		class="fila agregar"
		aria-label="Agregar producto"
		use:enhance={({ formElement }) =>
			async ({ result, update }) => {
				await update();
				if (result.type === 'success') formElement.querySelector<HTMLInputElement>('input[name="nombre"]')?.focus();
			}}
	>
		<input type="hidden" name="orden" value={productos.length + 1} />
		<button class="manija sumar" aria-label="Agregar producto"><span class="icono" aria-hidden="true">add</span></button>
		<input class="celda nombre" name="nombre" required minlength="2" maxlength="80" placeholder="Agregar un producto…" aria-label="Nombre del producto" />
		<span class="celda caja stock">
			<input name="stock" type="number" min="0" max="1000000" placeholder="—" aria-label="Stock (opcional)" />
			<span aria-hidden="true">u.</span>
		</span>
		<span class="celda caja precio">
			<span aria-hidden="true">$</span>
			<input name="precio" inputmode="numeric" pattern="[0-9.]*" placeholder="0" required aria-label="Precio del producto" />
		</span>
		<span></span>
	</form>
	<p class="g-nota pie">Lo que vendés en el local además del turno. Se suma al cobrar en la caja; con stock, se descuenta&nbsp;solo.</p>
</section>

<style>
	.cabeza {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 8px 18px;
		min-height: 56px;
		background: #f8f3ea;
	}
	.cabeza > .icono {
		font-size: 20px;
		color: var(--tl-menta-texto);
		width: 28px;
		text-align: center;
	}
	.titulo {
		flex: 1;
		margin: 0;
		font: 700 15px var(--tl-font);
		letter-spacing: normal;
	}
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.lista > li {
		position: relative;
		border-top: 1px solid #f0e8d8;
	}
	.lista > li.sobre {
		box-shadow: inset 0 3px 0 var(--tl-menta);
	}
	.fila {
		display: grid;
		grid-template-columns: 28px minmax(0, 1fr) 110px 130px 70px 40px;
		gap: 12px;
		align-items: center;
		padding: 6px 18px;
	}
	.fila.encabezado {
		padding: 8px 18px;
		font: 900 11px var(--tl-font);
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--tl-sub);
	}
	.fila.agregar {
		padding: 6px 18px 10px;
		border-top: 1px dashed var(--tl-linea-2);
	}
	/* El botón de borrar va en su propio formulario, encima de la última columna. */
	.borrar {
		position: absolute;
		top: 6px;
		right: 18px;
	}
	.quitar {
		width: 40px;
		height: 40px;
		display: grid;
		place-items: center;
		padding: 0;
		border: 0;
		border-radius: 50%;
		background: none;
		color: var(--tl-sub);
		cursor: pointer;
	}
	.quitar:hover {
		background: #fbe0dc;
		color: #a3241a;
	}
	.manija {
		width: 28px;
		height: 40px;
		display: grid;
		place-items: center;
		padding: 0;
		border: 0;
		border-radius: 8px;
		background: none;
		color: #6f7d7a;
		cursor: grab;
	}
	.manija .icono {
		font-size: 20px;
	}
	.manija.sumar {
		color: var(--tl-menta-texto);
		cursor: pointer;
	}
	.celda {
		height: 40px;
		min-width: 0;
		border-radius: 10px;
		font: 400 15px var(--tl-font);
		color: var(--tl-tinta);
	}
	.celda.nombre {
		width: 100%;
		padding: 0 10px;
		border: 0;
		background: transparent;
	}
	.celda.nombre:hover {
		background: #faf6ee;
	}
	.celda.nombre::placeholder {
		color: #6f7d7a;
	}
	.caja {
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 0 10px;
		background: #f8f3ea;
		font-size: 14px;
		font-variant-numeric: tabular-nums;
	}
	.caja input {
		width: 100%;
		flex: 1;
		min-width: 0;
		height: 100%;
		padding: 0;
		border: 0;
		background: transparent;
		font: inherit;
		color: inherit;
		appearance: textfield;
		-moz-appearance: textfield;
	}
	.caja input::-webkit-inner-spin-button,
	.caja input::-webkit-outer-spin-button {
		appearance: none;
		margin: 0;
	}
	.caja.precio {
		font-weight: 700;
	}
	.caja > span {
		color: var(--tl-sub);
		font-weight: 400;
	}
	.celda.nombre:focus,
	.caja:focus-within {
		outline: none;
		background: #fff;
		box-shadow: inset 0 0 0 2px var(--tl-menta);
	}
	.activo {
		display: flex;
		align-items: center;
	}
	.inactivo .nombre,
	.inactivo .caja {
		color: var(--tl-sub);
	}
	.sin-stock {
		display: flex;
		align-items: center;
		gap: 6px;
		margin: 0;
		padding: 0 18px 8px 58px;
		font-size: 12px;
		font-weight: 700;
		color: #7a4a00;
	}
	.sin-stock .icono {
		font-size: 16px;
	}
	.pie {
		padding: 4px 18px 14px 58px;
	}

	@media (max-width: 760px) {
		.fila.encabezado {
			display: none;
		}
		.fila {
			grid-template-columns: 28px minmax(0, 1fr) minmax(0, 1fr) auto 44px;
			grid-template-areas:
				'm n n n x'
				'. s p a .';
			gap: 8px;
			padding: 8px 14px;
		}
		.fila.agregar {
			grid-template-areas:
				'm n n n n'
				'. s p p p';
			padding: 8px 14px 12px;
		}
		.fila > .manija {
			grid-area: m;
		}
		.fila > .nombre {
			grid-area: n;
		}
		.fila > .stock {
			grid-area: s;
		}
		.fila > .precio {
			grid-area: p;
		}
		.fila > .activo {
			grid-area: a;
		}
		.fila.agregar > span:empty {
			display: none;
		}
		.borrar {
			top: 8px;
			right: 14px;
		}
		.celda,
		.manija,
		.quitar {
			height: 48px;
		}
		.quitar {
			width: 44px;
		}
		.celda {
			font-size: 16px;
		}
		.cabeza {
			padding: 8px 14px;
		}
		.sin-stock,
		.pie {
			padding-left: 14px;
		}
	}
</style>
