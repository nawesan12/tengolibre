<script lang="ts">
	import '#lib/gestion/gestion.css';
	import { enhance } from '$app/forms';
	import { tick } from 'svelte';
	import { confirmar } from '#lib/configuracion/formularios.ts';
	import type { SubmitFunction } from '$app/forms';
	import type { Servicio } from './+page.server.ts';
	import CartaProductos from '#lib/gestion/productos/CartaProductos.svelte';

	let { data, form } = $props();

	type Grupo = { id: string | null; nombre: string; servicios: Servicio[] };

	// Una carta por categoría, en su orden; lo que no tiene categoría va al final (o solo, si no hay).
	const grupos = $derived.by<Grupo[]>(() => {
		const g: Grupo[] = data.categorias.map((c) => ({ id: c.id, nombre: c.nombre, servicios: [] }));
		const sueltos: Grupo = { id: null, nombre: g.length ? 'Sin categoría' : 'Tus servicios', servicios: [] };
		for (const s of data.servicios) (g.find((x) => x.id === s.categoriaId) ?? sueltos).servicios.push(s);
		if (sueltos.servicios.length || g.length === 0) g.push(sueltos);
		return g;
	});

	let abierto = $state<string | null>(null);
	let nuevaCategoria = $state(false);
	let guardando = $state(0);

	// Guardar sin perder lo escrito: se recarga lo de la API pero no se vacían los campos.
	const guardar: SubmitFunction = () => {
		guardando++;
		return async ({ update }) => {
			await update({ reset: false });
			guardando--;
		};
	};
	const enviar = (e: Event) => (e.currentTarget as HTMLElement).closest('form')?.requestSubmit();

	const miles = (n: number) => n.toLocaleString('es-AR', { maximumFractionDigits: 0 });
	const quienes = (id: string) => data.profesionales.filter((p) => p.servicios.includes(id)).map((p) => p.nombre);

	// Reordenar: el mismo formulario oculto sirve para arrastrar, para las flechas del teclado y para
	// los botones «Subir» y «Bajar».
	let formOrden = $state<HTMLFormElement>();
	let ordenQue = $state<'servicios' | 'categorias'>('servicios');
	let ordenIds = $state('');
	async function ordenar(que: 'servicios' | 'categorias', ids: string[]) {
		ordenQue = que;
		ordenIds = ids.join(',');
		await tick();
		formOrden?.requestSubmit();
	}
	function mover<T extends { id: string | null }>(lista: T[], id: string, paso: number) {
		const ids = lista.map((x) => x.id).filter((x): x is string => !!x);
		const i = ids.indexOf(id);
		const j = i + paso;
		if (i < 0 || j < 0 || j >= ids.length) return null;
		[ids[i], ids[j]] = [ids[j], ids[i]];
		return ids;
	}
	function moverServicio(g: Grupo, id: string, paso: number) {
		const ids = mover(g.servicios, id, paso);
		if (ids) ordenar('servicios', ids);
	}
	function moverCategoria(id: string, paso: number) {
		const ids = mover(
			grupos.filter((g) => g.id),
			id,
			paso
		);
		if (ids) ordenar('categorias', ids);
	}
	function teclas(e: KeyboardEvent, fn: (paso: number) => void) {
		if (e.key === 'ArrowUp' || e.key === 'ArrowDown') {
			e.preventDefault();
			fn(e.key === 'ArrowUp' ? -1 : 1);
		}
	}

	// Arrastrar (con el mouse o el dedo sobre la manija): sólo dentro de la misma lista.
	let arrastrando = $state<{ que: 'servicios' | 'categorias'; id: string; grupo: string | null } | null>(null);
	let sobre = $state<string | null>(null);
	function soltar(destino: string, lista: { id: string | null }[]) {
		if (!arrastrando || arrastrando.id === destino) return;
		const ids = lista.map((x) => x.id).filter((x): x is string => !!x && x !== arrastrando!.id);
		ids.splice(ids.indexOf(destino) + (indice(lista, arrastrando.id) < indice(lista, destino) ? 1 : 0), 0, arrastrando.id);
		ordenar(arrastrando.que, ids);
		arrastrando = null;
		sobre = null;
	}
	const indice = (lista: { id: string | null }[], id: string) => lista.findIndex((x) => x.id === id);
</script>

<svelte:head><title>Servicios · Panel</title></svelte:head>

<div class="g-pagina">
	<div class="g-cabeza">
		<h1 class="g-titulo">Servicios</h1>
		<span class="g-meta" aria-live="polite">{guardando ? 'Guardando…' : 'Se guarda solo'}</span>
		<button type="button" class="g-boton" aria-expanded={nuevaCategoria} onclick={() => (nuevaCategoria = !nuevaCategoria)}>
			Nueva categoría
		</button>
	</div>

	{#if form?.error}
		<p class="g-aviso rojo" role="alert"><span class="icono" aria-hidden="true">error</span><span>{form.error}</span></p>
	{/if}

	{#if nuevaCategoria}
		<form
			method="POST"
			action="?/categoria"
			class="g-bloque nueva-categoria"
			use:enhance={() =>
				async ({ result, update }) => {
					await update();
					if (result.type === 'success') nuevaCategoria = false;
				}}
		>
			<input type="hidden" name="orden" value={data.categorias.length + 1} />
			<label class="g-campo crece">
				Nombre de la categoría
				<!-- svelte-ignore a11y_autofocus -->
				<input name="nombre" required maxlength="60" placeholder="Cejas y pestañas" autofocus />
			</label>
			<button class="g-boton oscuro">Crear categoría</button>
		</form>
	{/if}

	<form method="POST" action="?/ordenar" bind:this={formOrden} use:enhance={guardar} hidden>
		<input type="hidden" name="que" value={ordenQue} />
		<input type="hidden" name="ids" value={ordenIds} />
	</form>

	{#each grupos as g (g.id ?? 'sueltos')}
		{@const gid = g.id}
		<section
			class="g-bloque sin-pad carta"
			class:sobre={gid && sobre === gid && arrastrando?.que === 'categorias'}
			aria-label={g.nombre}
			ondragover={(e) => {
				if (gid && arrastrando?.que === 'categorias') {
					e.preventDefault();
					sobre = gid;
				}
			}}
			ondrop={(e) => {
				if (gid && arrastrando?.que === 'categorias') {
					e.preventDefault();
					soltar(gid, grupos.filter((x) => x.id));
				}
			}}
		>
			<div class="cabeza-categoria">
				{#if gid}
					<button
						type="button"
						class="manija"
						draggable="true"
						aria-label={`Mover la categoría ${g.nombre} (flechas arriba y abajo)`}
						onkeydown={(e) => teclas(e, (p) => moverCategoria(gid, p))}
						ondragstart={(e) => {
							arrastrando = { que: 'categorias', id: gid, grupo: null };
							e.dataTransfer?.setData('text/plain', gid);
						}}
						ondragend={() => ((arrastrando = null), (sobre = null))}
					>
						<span class="icono" aria-hidden="true">drag_indicator</span>
					</button>
					<form method="POST" action="?/renombrarCategoria" use:enhance={guardar} class="crece">
						<input type="hidden" name="id" value={gid} />
						<input class="nombre-categoria" name="nombre" value={g.nombre} required maxlength="60" aria-label="Nombre de la categoría {g.nombre}" onchange={enviar} />
					</form>
				{:else}
					<span class="manija" aria-hidden="true"></span>
					<h2 class="nombre-categoria crece">{g.nombre}</h2>
				{/if}
				<span class="g-meta">{g.servicios.length} {g.servicios.length === 1 ? 'servicio' : 'servicios'}</span>
				{#if gid}
					<form
						method="POST"
						action="?/borrarCategoria"
						use:enhance={confirmar(`¿Borrar la categoría ${g.nombre}? Sus servicios quedan sin categoría.`)}
					>
						<input type="hidden" name="id" value={gid} />
						<button class="g-accion neutro icono-boton" aria-label={`Borrar la categoría ${g.nombre}`}>
							<span class="icono" aria-hidden="true">delete</span>
						</button>
					</form>
				{/if}
			</div>

			<div class="fila-servicio encabezado" aria-hidden="true">
				<span></span><span>Nombre</span><span>Duración</span><span>Precio</span><span>Activo</span><span></span>
			</div>

			<ul class="lista">
				{#each g.servicios as s, i (s.id)}
					<li
						class:inactivo={!s.activo}
						class:sobre={sobre === s.id && arrastrando?.que === 'servicios'}
						ondragover={(e) => {
							if (arrastrando?.que === 'servicios' && arrastrando.grupo === gid) {
								e.preventDefault();
								e.stopPropagation();
								sobre = s.id;
							}
						}}
						ondrop={(e) => {
							if (arrastrando?.que === 'servicios' && arrastrando.grupo === gid) {
								e.preventDefault();
								e.stopPropagation();
								soltar(s.id, g.servicios);
							}
						}}
					>
						<form method="POST" action="?/editar" use:enhance={guardar} class="fila-servicio">
							<input type="hidden" name="id" value={s.id} />
							<input type="hidden" name="conActivo" value="1" />
							<button
								type="button"
								class="manija"
								draggable="true"
								aria-label={`Mover ${s.nombre} (flechas arriba y abajo)`}
								onkeydown={(e) => teclas(e, (p) => moverServicio(g, s.id, p))}
								ondragstart={(e) => {
									arrastrando = { que: 'servicios', id: s.id, grupo: gid };
									e.dataTransfer?.setData('text/plain', s.id);
								}}
								ondragend={() => ((arrastrando = null), (sobre = null))}
							>
								<span class="icono" aria-hidden="true">drag_indicator</span>
							</button>
							<input class="celda nombre" name="nombre" value={s.nombre} required minlength="2" maxlength="80" aria-label="Nombre de {s.nombre}" onchange={enviar} />
							<span class="celda caja">
								<input
									name="duracion"
									type="number"
									min="5"
									max="480"
									step="5"
									value={s.duracion}
									required
									aria-label="Duración de {s.nombre} en minutos"
									onchange={enviar}
								/>
								<span aria-hidden="true">min</span>
							</span>
							<span class="celda caja precio">
								<span aria-hidden="true">$</span>
								<input name="precio" inputmode="numeric" pattern="[0-9.]*" value={miles(s.precio)} required aria-label="Precio de {s.nombre} en pesos" onchange={enviar} />
							</span>
							<span class="activo">
								<input class="g-interruptor" type="checkbox" role="switch" name="activo" checked={s.activo} aria-label="{s.nombre} activo" onchange={enviar} />
								<span class="estado-activo">{s.activo ? 'Sí' : 'No'}</span>
							</span>
							<button
								type="button"
								class="mas"
								aria-expanded={abierto === s.id}
								aria-controls="detalle-{s.id}"
								aria-label="Descripción, foto y más de {s.nombre}"
								onclick={() => (abierto = abierto === s.id ? null : s.id)}
							>
								<span class="icono" aria-hidden="true">{abierto === s.id ? 'expand_less' : 'more_horiz'}</span>
							</button>
						</form>

						{#if abierto === s.id}
							{@const lo = quienes(s.id)}
							<div class="detalle" id="detalle-{s.id}">
								<form method="POST" action="?/editar" use:enhance={guardar} class="detalle-datos">
									<input type="hidden" name="id" value={s.id} />
									<label class="g-campo">
										Descripción
										<textarea name="descripcion" rows="3" maxlength="1000" placeholder="Qué incluye, cuánto dura el efecto…" onchange={enviar}>{s.descripcion}</textarea>
									</label>
									<label class="g-campo">
										Categoría
										<select name="categoriaId" onchange={enviar}>
											<option value="">Sin categoría</option>
											{#each data.categorias as c (c.id)}<option value={c.id} selected={c.id === s.categoriaId}>{c.nombre}</option>{/each}
										</select>
									</label>
								</form>
								<div class="detalle-lado">
									<div class="foto">
										{#if s.foto}<img src={s.foto} alt="Foto de {s.nombre}" />{:else}<span class="sin-foto"><span class="icono" aria-hidden="true">image</span>Sin foto</span>{/if}
										<form method="POST" action="?/foto" enctype="multipart/form-data" use:enhance={guardar} class="g-chips">
											<input type="hidden" name="id" value={s.id} />
											<label class="g-chip">
												<span class="icono" aria-hidden="true">add_photo_alternate</span>{s.foto ? 'Cambiar foto' : 'Subir foto'}
												<input type="file" name="archivo" accept="image/jpeg,image/png,image/webp" onchange={enviar} />
											</label>
											{#if s.foto}
												<button class="g-accion peligro" formaction="?/sacarFoto" formenctype="application/x-www-form-urlencoded">Sacar foto</button>
											{/if}
										</form>
									</div>
									<p class="g-nota">
										{#if lo.length}Lo hacen: <b>{lo.join(', ')}</b>.{:else}Todavía no lo hace nadie del&nbsp;equipo.{/if}
										<a class="g-accion" href="/panel/equipo">Cambiar en Equipo</a>
									</p>
								</div>
								<div class="detalle-acciones">
									<button type="button" class="g-boton chico" disabled={i === 0} onclick={() => moverServicio(g, s.id, -1)}>
										<span class="icono" aria-hidden="true">arrow_upward</span>Subir
									</button>
									<button type="button" class="g-boton chico" disabled={i === g.servicios.length - 1} onclick={() => moverServicio(g, s.id, 1)}>
										<span class="icono" aria-hidden="true">arrow_downward</span>Bajar
									</button>
									<form method="POST" action="?/borrar" use:enhance={confirmar(`¿Borrar ${s.nombre}? Los turnos que ya tiene no se tocan.`)} class="empuja">
										<input type="hidden" name="id" value={s.id} />
										<button class="g-accion peligro">Borrar servicio</button>
									</form>
								</div>
							</div>
						{/if}
					</li>
				{/each}
			</ul>

			<form
				method="POST"
				action="?/crear"
				class="fila-servicio agregar"
				aria-label="Agregar a {g.nombre}"
				use:enhance={({ formElement }) =>
					async ({ result, update }) => {
						await update();
						if (result.type === 'success') formElement.querySelector<HTMLInputElement>('input[name="nombre"]')?.focus();
					}}
			>
				<input type="hidden" name="categoriaId" value={gid ?? ''} />
				<input type="hidden" name="activo" value="on" />
				<button class="manija sumar" aria-label="Agregar"><span class="icono" aria-hidden="true">add</span></button>
				<input class="celda nombre" name="nombre" required minlength="2" maxlength="80" placeholder="Agregar a {g.nombre}…" aria-label="Nombre" />
				<span class="celda caja">
					<input name="duracion" type="number" min="5" max="480" step="5" placeholder="30" aria-label="Duración (minutos)" />
					<span aria-hidden="true">min</span>
				</span>
				<span class="celda caja precio">
					<span aria-hidden="true">$</span>
					<input name="precio" inputmode="numeric" pattern="[0-9.]*" placeholder="0" required aria-label="Precio" />
				</span>
				<span></span><span></span>
			</form>
		</section>
	{/each}

	{#if data.productos}<CartaProductos productos={data.productos} {guardar} />{/if}

	<p class="g-nota">
		La descripción, la foto y quién lo hace se abren desde la fila. Los inactivos no se ven en tu página, pero se pueden cargar desde el&nbsp;panel.
	</p>
</div>

<style>
	.crece {
		flex: 1;
		min-width: 0;
	}
	.nueva-categoria {
		flex-direction: row;
		align-items: flex-end;
		flex-wrap: wrap;
	}
	.carta {
		transition: box-shadow var(--tl-tocar) var(--tl-ease);
	}
	.carta.sobre {
		box-shadow: 0 0 0 2px var(--tl-menta);
	}
	.cabeza-categoria {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 8px 18px;
		background: #f8f3ea;
		min-height: 56px;
	}
	.nombre-categoria {
		margin: 0;
		width: 100%;
		min-height: 40px;
		padding: 0 8px;
		margin-left: -8px;
		border: 0;
		border-radius: 10px;
		background: transparent;
		color: var(--tl-tinta);
		font: 700 15px/40px var(--tl-font);
		letter-spacing: normal;
	}
	input.nombre-categoria:hover,
	input.nombre-categoria:focus {
		background: #fff;
		outline: none;
		box-shadow: inset 0 0 0 2px var(--tl-menta);
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
		flex: none;
	}
	.manija .icono {
		font-size: 20px;
	}
	.manija.sumar {
		color: var(--tl-menta-texto);
		cursor: pointer;
	}
	.icono-boton {
		width: 40px;
		justify-content: center;
		color: var(--tl-sub);
	}
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.lista > li {
		border-top: 1px solid #f0e8d8;
	}
	.lista > li.sobre {
		box-shadow: inset 0 3px 0 var(--tl-menta);
	}
	.fila-servicio {
		display: grid;
		grid-template-columns: 28px minmax(0, 1fr) 110px 130px 70px 40px;
		gap: 12px;
		align-items: center;
		padding: 6px 18px;
	}
	.fila-servicio.encabezado {
		padding: 8px 18px;
		font: 900 11px var(--tl-font);
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--tl-sub);
	}
	.fila-servicio.agregar {
		padding: 6px 18px 10px;
		border-top: 1px dashed var(--tl-linea-2);
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
		-moz-appearance: textfield;
		appearance: textfield;
	}
	.caja input::-webkit-inner-spin-button,
	.caja input::-webkit-outer-spin-button {
		appearance: none;
		margin: 0;
	}
	.caja.precio {
		font-weight: 700;
	}
	.caja:not(.precio) input {
		flex: 0 1 4ch;
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
		gap: 6px;
	}
	.estado-activo {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip: rect(0 0 0 0);
	}
	.inactivo .nombre,
	.inactivo .caja {
		color: var(--tl-sub);
	}
	.mas {
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
	.mas:hover,
	.mas[aria-expanded='true'] {
		background: #f4eee2;
		color: var(--tl-tinta);
	}
	.detalle {
		display: grid;
		grid-template-columns: minmax(0, 1fr) minmax(0, 300px);
		gap: 16px;
		padding: 8px 18px 16px 58px;
		animation: tl-gin var(--tl-entrar) var(--tl-ease) both;
	}
	.detalle-datos {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.detalle-lado {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.foto {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.foto img,
	.sin-foto {
		width: 100%;
		aspect-ratio: 16 / 9;
		object-fit: cover;
		border-radius: 12px;
	}
	.sin-foto {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 6px;
		background: repeating-linear-gradient(135deg, #f4eee2 0 8px, #ece4d3 8px 16px);
		font-size: 13px;
		color: var(--tl-sub);
	}
	.foto .g-chip {
		position: relative;
		cursor: pointer;
		width: fit-content;
	}
	.foto .g-chip input[type='file'] {
		position: absolute;
		inset: 0;
		opacity: 0;
		cursor: pointer;
	}
	.foto .g-chip:focus-within {
		outline: 3px solid var(--tl-menta);
		outline-offset: 2px;
	}
	.detalle-acciones {
		grid-column: 1 / -1;
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px;
	}
	.empuja {
		margin-left: auto;
	}
	.detalle-acciones .icono,
	.g-chip .icono {
		font-size: 18px;
	}

	@media (max-width: 760px) {
		.fila-servicio.encabezado {
			display: none;
		}
		.fila-servicio {
			grid-template-columns: 28px minmax(0, 1fr) minmax(0, 1fr) auto 40px;
			grid-template-areas:
				'm n n n x'
				'. d p a .';
			gap: 8px;
			padding: 8px 14px;
		}
		.fila-servicio.agregar {
			grid-template-areas:
				'm n n n n'
				'. d p p p';
			padding: 8px 14px 12px;
		}
		.fila-servicio > .manija {
			grid-area: m;
		}
		.fila-servicio > .nombre {
			grid-area: n;
		}
		.fila-servicio > .caja:not(.precio) {
			grid-area: d;
		}
		.fila-servicio > .precio {
			grid-area: p;
		}
		.fila-servicio > .activo {
			grid-area: a;
		}
		.fila-servicio > .mas {
			grid-area: x;
		}
		.fila-servicio.agregar > span:empty {
			display: none;
		}
		.celda,
		.mas,
		.manija {
			height: 48px;
		}
		.mas {
			width: 44px;
		}
		.celda {
			font-size: 16px;
		}
		.cabeza-categoria {
			padding: 8px 14px;
		}
		.detalle {
			grid-template-columns: minmax(0, 1fr);
			padding: 8px 14px 16px;
		}
	}
</style>
