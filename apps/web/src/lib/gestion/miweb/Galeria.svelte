<!--
	Galería de Mi web: miniaturas en grilla de 4 que se arrastran para ordenar (o con las flechas del
	teclado sobre la miniatura); al tocar una se abre debajo su epígrafe, moverla y sacarla. El último
	cuadro suma una foto.
-->
<script lang="ts">
	import { enhance } from '$app/forms';
	import { tick } from 'svelte';
	import { confirmar, sinReset } from '#lib/configuracion/formularios.ts';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';
	import type { Form } from '#lib/configuracion/mensajes.ts';

	type Foto = { id: string; url: string; epigrafe: string; orden: number };
	let { fotos, form }: { fotos: Foto[]; form: Form } = $props();

	const MAX_FOTOS = 30;
	const MAX = 5 << 20;
	let elegida = $state<string | null>(null);
	let archivo = $state<string | null>(null);
	let subiendo = $state(false);
	const foto = $derived(fotos.find((f) => f.id === elegida) ?? null);
	const indice = $derived(fotos.findIndex((f) => f.id === elegida));

	function revisar(e: Event & { currentTarget: HTMLInputElement }) {
		const a = e.currentTarget.files?.[0];
		e.currentTarget.setCustomValidity(a && a.size > MAX ? 'La imagen pasa de 5 MB.' : '');
		e.currentTarget.reportValidity();
		archivo = a?.name ?? null;
	}

	let formOrden = $state<HTMLFormElement>();
	let ids = $state('');
	async function ordenar(lista: string[]) {
		ids = lista.join(',');
		await tick();
		formOrden?.requestSubmit();
	}
	function mover(id: string, paso: number) {
		const lista = fotos.map((f) => f.id);
		const i = lista.indexOf(id);
		const j = i + paso;
		if (i < 0 || j < 0 || j >= lista.length) return;
		[lista[i], lista[j]] = [lista[j], lista[i]];
		ordenar(lista);
	}
	let arrastrando = $state<string | null>(null);
	let sobre = $state<string | null>(null);
	function soltar(destino: string) {
		if (!arrastrando || arrastrando === destino) return;
		const lista = fotos.map((f) => f.id);
		const desde = lista.indexOf(arrastrando);
		lista.splice(desde, 1);
		lista.splice(lista.indexOf(destino) + (desde <= fotos.findIndex((f) => f.id === destino) ? 1 : 0), 0, arrastrando);
		arrastrando = sobre = null;
		ordenar(lista);
	}
</script>

<section class="g-bloque galeria" aria-labelledby="t-galeria">
	<h2 class="g-bloque-titulo" id="t-galeria">Galería · {fotos.length} {fotos.length === 1 ? 'foto' : 'fotos'}</h2>
	<Mensaje {form} que="galeria" />

	<form method="POST" action="?/ordenar" bind:this={formOrden} use:enhance={sinReset} hidden>
		<input type="hidden" name="ids" value={ids} />
	</form>

	<ol class="grilla">
		{#each fotos as f, i (f.id)}
			<li class:sobre={sobre === f.id}>
				<button
					type="button"
					class="mini"
					draggable="true"
					aria-pressed={elegida === f.id}
					aria-label={`Foto ${i + 1}${f.epigrafe ? `: ${f.epigrafe}` : ''}. Tocá para editarla; flechas para moverla.`}
					onclick={() => (elegida = elegida === f.id ? null : f.id)}
					onkeydown={(e) => {
						if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') (e.preventDefault(), mover(f.id, -1));
						if (e.key === 'ArrowRight' || e.key === 'ArrowDown') (e.preventDefault(), mover(f.id, 1));
					}}
					ondragstart={(e) => {
						arrastrando = f.id;
						e.dataTransfer?.setData('text/plain', f.id);
					}}
					ondragover={(e) => {
						if (arrastrando) (e.preventDefault(), (sobre = f.id));
					}}
					ondrop={(e) => (e.preventDefault(), soltar(f.id))}
					ondragend={() => (arrastrando = sobre = null)}
				>
					<img src={f.url} alt={f.epigrafe || `Foto ${i + 1} de la galería`} loading="lazy" />
				</button>
			</li>
		{/each}
		{#if fotos.length < MAX_FOTOS}
			<li>
				<label class="mini sumar" class:con-archivo={archivo}>
					<span class="icono" aria-hidden="true">{archivo ? 'check_circle' : 'add_photo_alternate'}</span>
					<span class="visualmente-oculto">Foto</span>
					<input
						form="f-foto"
						type="file"
						name="archivo"
						accept="image/jpeg,image/png,image/webp"
						required
						aria-label="Foto"
						onchange={revisar}
					/>
				</label>
			</li>
		{/if}
	</ol>

	{#if archivo}
		<form
			id="f-foto"
			method="POST"
			action="?/foto"
			enctype="multipart/form-data"
			class="nueva"
			use:enhance={() => {
				subiendo = true;
				return async ({ result, update }) => {
					await update();
					subiendo = false;
					if (result.type === 'success') archivo = null;
				};
			}}
		>
			<span class="g-sub archivo">{archivo}</span>
			<label class="g-campo">Epígrafe (opcional) <input name="epigrafe" maxlength="200" placeholder="El local, un trabajo…" /></label>
			<button class="g-boton oscuro" disabled={subiendo}>{subiendo ? 'Subiendo…' : 'Sumar a la galería'}</button>
		</form>
	{:else}
		<form id="f-foto" method="POST" action="?/foto" enctype="multipart/form-data" hidden></form>
	{/if}

	{#if foto}
		<div class="editar">
			<form method="POST" action="?/epigrafe" use:enhance={sinReset} class="epigrafe">
				<input type="hidden" name="id" value={foto.id} />
				<label class="g-campo crece">
					Epígrafe de la foto {indice + 1}
					<input name="epigrafe" value={foto.epigrafe} maxlength="200" placeholder="Epígrafe" />
				</label>
				<button class="g-boton chico">Guardar</button>
			</form>
			<div class="acciones">
				<button type="button" class="g-boton chico" disabled={indice === 0} onclick={() => mover(foto.id, -1)}>
					<span class="icono" aria-hidden="true">arrow_back</span>Antes
				</button>
				<button type="button" class="g-boton chico" disabled={indice === fotos.length - 1} onclick={() => mover(foto.id, 1)}>
					Después<span class="icono" aria-hidden="true">arrow_forward</span>
				</button>
				<form
					method="POST"
					action="?/borrarFoto"
					class="empuja"
					use:enhance={confirmar('¿Sacar esta foto de la galería?')}
				>
					<input type="hidden" name="id" value={foto.id} />
					<button class="g-accion peligro">Sacar la foto</button>
				</form>
			</div>
		</div>
	{/if}

	<p class="g-nota">
		{fotos.length ? 'Arrastrá para ordenar: así se ven en tu página.' : 'Fotos de tus trabajos y del lugar, hasta 30.'} JPG, PNG o WebP de hasta 5&nbsp;MB.
	</p>
</section>

<style>
	.grilla {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		gap: 6px;
	}
	.grilla li.sobre .mini {
		box-shadow: 0 0 0 2px var(--tl-menta);
	}
	.mini {
		display: block;
		width: 100%;
		aspect-ratio: 1;
		padding: 0;
		border: 0;
		border-radius: 10px;
		overflow: hidden;
		background: var(--tl-crema);
		cursor: grab;
	}
	.mini[aria-pressed='true'] {
		box-shadow: 0 0 0 3px var(--tl-petroleo);
	}
	.mini img {
		width: 100%;
		height: 100%;
		object-fit: cover;
		display: block;
	}
	.sumar {
		position: relative;
		display: grid;
		place-items: center;
		background: #fff;
		box-shadow: inset 0 0 0 1.5px #e0d5bf;
		color: var(--tl-menta-texto);
		cursor: pointer;
	}
	.sumar.con-archivo {
		box-shadow: inset 0 0 0 2px var(--tl-menta);
	}
	.sumar:focus-within {
		outline: 3px solid var(--tl-menta);
		outline-offset: 2px;
	}
	.sumar input {
		position: absolute;
		inset: 0;
		opacity: 0;
		cursor: pointer;
	}
	.nueva {
		display: flex;
		flex-direction: column;
		gap: 10px;
		padding: 12px;
		border-radius: 12px;
		background: #f8f3ea;
	}
	.archivo {
		overflow-wrap: anywhere;
	}
	.editar {
		display: flex;
		flex-direction: column;
		gap: 10px;
		padding: 12px;
		border-radius: 12px;
		background: #f8f3ea;
		animation: tl-gin var(--tl-entrar) var(--tl-ease) both;
	}
	.epigrafe {
		display: flex;
		align-items: flex-end;
		gap: 8px;
		flex-wrap: wrap;
	}
	.crece {
		flex: 1;
		min-width: min(200px, 100%);
	}
	.epigrafe .g-boton {
		min-height: 46px;
	}
	.acciones {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 8px;
	}
	.acciones .icono {
		font-size: 18px;
	}
	.empuja {
		margin-left: auto;
	}
</style>
