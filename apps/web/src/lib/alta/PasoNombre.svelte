<!--
	Paso 2: el nombre y el link, que se arma solo y se valida al instante. Acá se crea el negocio.
	El link va en grande y editable ahí mismo: es lo que la persona va a compartir (la gente no lee,
	así que se ve antes de que haga falta explicarlo).
-->
<script lang="ts">
	import { enhance } from '$app/forms';
	import { page } from '$app/state';
	import type { SubmitFunction } from '$app/forms';
	import Icono from '#lib/ui/Icono.svelte';
	import type { Borrador } from './estado.svelte.ts';

	let {
		b,
		creado,
		slug,
		alEnviar,
		form,
		listo = $bindable()
	}: { b: Borrador; creado: boolean; slug?: string; alEnviar: SubmitFunction; form: { error?: string } | null | undefined; listo: boolean } = $props();

	const host = $derived(page.url.host);
	$effect(() => {
		if (!creado) b.direccion.poner(b.slug);
	});
	const campo = $derived(b.direccion.campo);
	$effect(() => {
		listo = !!b.nombre.trim() && (creado || campo.ok);
	});
	const sugerencias = $derived(campo.estado === 'tomada' ? (b.direccion.resultado?.sugerencias ?? []) : []);

	// Lo que se tipea en el link: mientras se escribe se deja el guion del final (aSlug lo sacaría y
	// no se podría escribir «lumi-estetica»); fuera del campo, se ve el link ya armado.
	let editando = $state(false);
	let escrito = $state('');
	const limpiar = (v: string) =>
		v
			.normalize('NFD')
			.replace(/[̀-ͯ]/g, '')
			.toLowerCase()
			.replace(/[\s_.]+/g, '-')
			.replace(/[^a-z0-9-]/g, '')
			.slice(0, 48);
	function escribir(e: Event & { currentTarget: HTMLInputElement }) {
		escrito = limpiar(e.currentTarget.value);
		e.currentTarget.value = escrito;
		b.slugManual = escrito || null;
	}
</script>

<form id="paso" method="POST" action="?/nombre" use:enhance={alEnviar} class="hoja" data-paso="nombre">
	<h1>¿Cómo se&nbsp;llama?</h1>
	{#if creado}<input type="hidden" name="editando" value="1" />{/if}
	<input type="hidden" name="rubro" value={b.rubro} />
	<input type="hidden" name="slug" value={b.slug} />
	<label class="campo-nombre">
		Nombre del negocio
		<input
			name="nombre"
			bind:value={b.nombre}
			oninput={() => (b.slugManual = null)}
			autocomplete="organization"
			maxlength="80"
			required
			placeholder="Lumi Estética"
		/>
	</label>
	<div class="link" class:ok={creado || campo.ok}>
		<span class="tl-rotulo rot" id="dir-t"><Icono nombre="link" tam={16} />Tu link</span>
		{#if creado}
			<span class="url" aria-labelledby="dir-t"><span class="pre">{host}/</span><b>{slug}</b></span>
			<span class="msg ok" role="status"><Icono nombre="check_circle" lleno tam={18} />Es tuyo. Lo cambiás cuando quieras en Mi&nbsp;web.</span>
		{:else}
			<label class="url editable" style:--borde={campo.borde}>
				<span class="pre">{host}/</span>
				<input
					value={editando ? escrito : b.slug}
					oninput={escribir}
					onfocus={() => {
						escrito = b.slug;
						editando = true;
					}}
					onblur={() => (editando = false)}
					aria-labelledby="dir-t"
					aria-describedby="dir-msg"
					autocapitalize="none"
					autocomplete="off"
					spellcheck="false"
					inputmode="url"
					maxlength="48"
					placeholder="tu-negocio"
				/>
				<span class="ic" style:color={campo.borde}><Icono nombre={campo.icono} lleno tam={24} /></span>
			</label>
			<span id="dir-msg" class="msg" style:color={campo.color} role="status">{campo.mensaje}</span>
		{/if}
		<span class="clave"><Icono nombre="campaign" lleno tam={20} />Este es tu link: lo compartís y te reservan&nbsp;solos.</span>
	</div>
	{#if sugerencias.length}
		<div class="alts">
			{#each sugerencias as s (s)}
				<button type="button" class="alt" onclick={() => (b.slugManual = s)}>{s}</button>
			{/each}
		</div>
	{/if}
	{#if form?.error}<p class="error-paso" role="alert"><Icono nombre="error" lleno tam={16} />{form.error}</p>{/if}
</form>

<style>
	.campo-nombre {
		display: flex;
		flex-direction: column;
		gap: 8px;
		font-size: 14px;
		font-weight: 700;
	}
	.campo-nombre input {
		height: 56px;
		padding: 0 16px;
		border-radius: 16px;
		border: 0;
		background: #fff;
		box-shadow: inset 0 0 0 2px #3ccc90;
		font: 400 18px var(--tl-font);
		color: var(--tl-tinta);
		outline: none;
	}
	/* El link, protagonista: tarjeta con borde menta y el link grande, editable en el lugar. */
	.link {
		display: flex;
		flex-direction: column;
		gap: 10px;
		padding: 16px;
		border-radius: 24px;
		background: #fffdf9;
		box-shadow:
			0 14px 28px -22px rgba(15, 35, 40, 0.45),
			inset 0 0 0 2px #e6dcc8;
		transition: box-shadow 0.2s;
	}
	.link.ok {
		box-shadow:
			0 14px 28px -22px rgba(15, 35, 40, 0.45),
			inset 0 0 0 2px #3ccc90;
	}
	.rot {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		color: #17805a;
	}
	.url {
		--borde: #17805a;
		display: flex;
		align-items: center;
		min-height: 60px;
		font-size: 24px;
		font-weight: 900;
		letter-spacing: -0.03em;
		overflow-wrap: anywhere;
	}
	.url.editable {
		padding: 0 12px;
		border-radius: 16px;
		background: #fff;
		box-shadow: inset 0 0 0 2px var(--borde);
		cursor: text;
	}
	.pre {
		color: var(--tl-sub);
		font-weight: 700;
		white-space: nowrap;
	}
	.url b,
	.url input {
		color: #17805a;
		font: 900 24px var(--tl-font);
		letter-spacing: -0.03em;
	}
	.url input {
		flex: 1;
		min-width: 0;
		height: 56px;
		padding: 0;
		border: 0;
		background: none;
		outline: none;
	}
	.ic {
		display: grid;
		flex: none;
	}
	.msg {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 13px;
		font-weight: 700;
	}
	.msg.ok {
		color: #17805a;
	}
	.clave {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 12px;
		border-radius: 14px;
		background: #e3f6ec;
		color: #102b31;
		font-size: 14px;
		font-weight: 700;
		text-wrap: pretty;
	}
	.clave :global(.icono) {
		color: #17805a;
		flex: none;
	}
	@media (max-width: 899px) {
		.url,
		.url b,
		.url input {
			font-size: 19px;
		}
		/* En el teléfono el host va arriba, chico, y el link entero abajo: no se corta. */
		.url.editable {
			flex-wrap: wrap;
			padding: 8px 12px 4px;
		}
		.url.editable .pre {
			flex-basis: 100%;
			font-size: 13px;
			letter-spacing: 0;
		}
		.url input {
			height: 44px;
			/* Con ancho 0 de base entraría en el renglón del host: que arranque abajo. */
			min-width: 70%;
		}
	}
	.alts {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}
	.alt {
		height: 44px;
		padding: 0 14px;
		border-radius: 999px;
		border: 0;
		background: repeating-linear-gradient(-45deg, rgba(60, 204, 144, 0.22) 0 2px, transparent 2px 9px), #f3fbf7;
		box-shadow: inset 0 0 0 1.5px rgba(23, 128, 90, 0.35);
		font: 700 14px var(--tl-font);
		color: #17805a;
		cursor: pointer;
	}
</style>
