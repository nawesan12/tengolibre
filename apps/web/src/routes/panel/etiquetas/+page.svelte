<!--
	Clientes › Etiquetas (handoff «Pantallas que faltaban», 1c): las etiquetas con cuántos clientes
	tiene cada una, se renombran o borran en la misma fila, y se crean abajo.
-->
<script lang="ts">
	import { page } from '$app/state';
	import { enhance } from '$app/forms';
	import Icono from '#lib/ui/Icono.svelte';
	import Cabecera from '#lib/operacion/Cabecera.svelte';
	import { GESTION } from '#lib/operacion/secciones.ts';
	import { accion, colorEtiqueta } from '#lib/operacion/ui.ts';

	let { data, form } = $props();
	const gestiona = $derived(GESTION.includes(data.negocio.rol));
	let editando = $state<string | null>(null);
	/** La paleta del handoff para las etiquetas: nunca van solas, siempre con su nombre. */
	const PALETA = ['#183c44', '#17805a', '#7a5a1e', '#a3241a', '#7a4a00', '#4a5d61'];
	let color = $state(PALETA[1]);
	const cerrar = () => async ({ result, update }: { result: { type: string }; update: () => Promise<void> }) => {
		await update();
		if (result.type === 'success') editando = null;
	};
</script>

<svelte:head><title>Etiquetas · Clientes · Panel</title></svelte:head>

<Cabecera titulo="Etiquetas" rotulo="Clientes" volver={{ href: '/panel/clientes', texto: 'Clientes' }} />

<div class="hoja">
	{#if data.etiquetas.length}
		<div class="resumen">
			{#each data.etiquetas as e (e.id)}
				{@const c = colorEtiqueta(e.color)}
				<a class="pastilla" href={`/panel/clientes?etiqueta=${e.id}`} style:background={c.fondo} style:color={c.texto}>{e.nombre}<span class="tl-num">{e.clientes}</span></a>
			{/each}
		</div>
	{/if}
	{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}

	{#if data.etiquetas.length === 0}
		<p class="vacio">Todavía no hay etiquetas. Sirven para agrupar clientes (frecuentes, VIP, a reactivar) y filtrarlos en el&nbsp;listado.</p>
	{:else}
		<ul class="lista">
			{#each data.etiquetas as e (e.id)}
				{@const c = colorEtiqueta(e.color)}
				<li>
					{#if editando === e.id}
						<form method="POST" action={accion(page.url, 'editar')} class="editar" use:enhance={cerrar}>
							<input type="hidden" name="id" value={e.id} />
							<label class="campo">Nombre <input name="nombre" value={e.nombre} required maxlength="40" /></label>
							<label class="campo">Color <input name="color" type="color" value={e.color} /></label>
							<button class="boton oscuro">Guardar</button>
							<button type="button" class="boton" onclick={() => (editando = null)}>Cancelar</button>
						</form>
					{:else}
						<span class="marca" style:background={c.fondo} style:color={c.texto}><Icono nombre="sell" lleno tam={18} /></span>
						<span class="txt"><b>{e.nombre}</b><a href={`/panel/clientes?etiqueta=${e.id}`}>{e.clientes}&nbsp;{e.clientes === 1 ? 'cliente' : 'clientes'}</a></span>
						{#if gestiona}
							<button class="mini" onclick={() => (editando = e.id)}>Editar</button>
							<form
								method="POST"
								action={accion(page.url, 'borrar')}
								use:enhance={({ cancel }) => {
									if (!confirm(`¿Borrar la etiqueta ${e.nombre}? Se les saca a todos los clientes.`)) cancel();
								}}
							>
								<input type="hidden" name="id" value={e.id} />
								<button class="mini peligro" aria-label={`Borrar ${e.nombre}`}><Icono nombre="delete" tam={18} /></button>
							</form>
						{/if}
					{/if}
				</li>
			{/each}
		</ul>
	{/if}

	{#if gestiona}
		<form method="POST" action={accion(page.url, 'crear')} use:enhance class="nueva" aria-label="Nueva etiqueta">
			<label class="campo">Nombre <input name="nombre" required maxlength="40" placeholder="Nueva etiqueta" /></label>
			<fieldset class="paleta">
				<legend>Color</legend>
				{#each PALETA as p (p)}
					<label style:--c={p}><input type="radio" name="color" value={p} bind:group={color} /><span class="visualmente-oculto">{p}</span></label>
				{/each}
			</fieldset>
			<p class="nota">Elegí un color de la paleta: nunca va solo, siempre con su&nbsp;nombre.</p>
			<button class="crear">Crear etiqueta</button>
		</form>
	{/if}
</div>

<style>
	.hoja {
		max-width: 620px;
		padding: 24px;
		border-radius: 24px;
		background-color: var(--tl-crema);
		background-image: var(--tl-grano-osc), radial-gradient(rgba(24, 60, 68, 0.1) 1.2px, transparent 1.4px);
		background-size: auto, 18px 18px;
		box-shadow: var(--tl-sh-alta);
		display: flex;
		flex-direction: column;
		gap: 14px;
	}
	.resumen {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
	}
	.pastilla {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		min-height: 40px;
		padding: 0 14px;
		border-radius: 999px;
		font-size: 14px;
		font-weight: 700;
		text-decoration: none;
	}
	.pastilla span {
		opacity: 0.75;
	}
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.lista li {
		display: flex;
		align-items: center;
		gap: 12px;
		min-height: 56px;
		padding: 8px 12px 8px 14px;
		border-radius: 16px;
		background: var(--tl-papel);
	}
	.marca {
		width: 36px;
		height: 36px;
		border-radius: 10px;
		display: grid;
		place-items: center;
		flex: none;
	}
	.txt {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
	}
	.txt b {
		font-size: 15px;
	}
	.txt a {
		font-size: 12px;
		color: var(--tl-sub);
	}
	.mini {
		min-height: 40px;
		padding: 0 14px;
		border: 0;
		border-radius: 999px;
		background: transparent;
		box-shadow: inset 0 0 0 1.5px var(--tl-linea-2);
		font: 700 13px var(--tl-font);
		color: var(--tl-tinta);
		cursor: pointer;
		display: inline-flex;
		align-items: center;
	}
	.mini.peligro {
		color: #a3241a;
		padding: 0 10px;
	}
	.editar {
		flex: 1;
		display: grid;
		grid-template-columns: 1fr auto auto auto;
		gap: 8px;
		align-items: end;
	}
	.nueva {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.paleta {
		border: 0;
		margin: 0;
		padding: 0;
		display: flex;
		gap: 8px;
		align-items: center;
	}
	.paleta legend {
		font-size: 14px;
		font-weight: 700;
		margin-bottom: 6px;
	}
	.paleta label {
		position: relative;
		width: 36px;
		height: 36px;
		border-radius: 50%;
		background: var(--c);
		cursor: pointer;
		box-shadow: 0 0 0 2px var(--tl-papel);
	}
	.paleta input {
		position: absolute;
		opacity: 0;
	}
	.paleta label:has(input:checked) {
		box-shadow: 0 0 0 2px var(--tl-papel), 0 0 0 4px var(--tl-petroleo);
	}
	.paleta label:has(input:focus-visible) {
		outline: 3px solid var(--tl-menta);
		outline-offset: 4px;
	}
	.nota,
	.vacio {
		margin: 0;
		font-size: 13px;
		color: var(--tl-sub);
	}
	.vacio {
		font-size: 15px;
	}
	.crear {
		height: 52px;
		border: 0;
		border-radius: 999px;
		background: var(--tl-petroleo);
		color: #fff;
		font: 700 16px var(--tl-font);
		cursor: pointer;
	}
	@media (max-width: 859px) {
		.hoja {
			padding: 16px;
		}
		.editar {
			grid-template-columns: 1fr 1fr;
		}
	}
</style>
