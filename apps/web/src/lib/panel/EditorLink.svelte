<!--
	Cambiar el link (tengolibre.com.ar/<slug>): vista previa en vivo, si está libre mientras se escribe
	y el aviso de que el viejo sigue andando (la API lo guarda y la web redirige con 301). Lo usan Mi web
	y el panel «Tu link» de la barra. Habla con /panel/link (el servidor de SvelteKit llama a la API).
-->
<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import Icono from '#lib/ui/Icono.svelte';
	import { campoLink, type EstadoLink, type Link } from './link.ts';

	let { slug, host, alGuardar }: { slug: string; host: string; alGuardar?: (l: Link) => void } = $props();

	let link = $state<Link | null>(null);
	let nuevo = $state('');
	let estado = $state<EstadoLink | null>(null);
	let guardando = $state(false);
	let error = $state('');
	let listo = $state(false);
	let t: ReturnType<typeof setTimeout> | undefined;
	let pedido = 0;

	// El link de ahora y los anteriores (se cargan al abrir: no hacen falta para mostrar el panel).
	$effect(() => {
		void slug;
		nuevo = slug;
		estado = 'propia';
		fetch('/panel/link')
			.then((r) => (r.ok ? r.json() : null))
			.then((l: Link | null) => (link = l))
			.catch(() => {});
	});

	// Mientras se escribe: minúsculas, sin acentos, espacios a guiones (el resto lo valida la API).
	const limpiar = (s: string) =>
		s
			.normalize('NFD')
			.replace(/[̀-ͯ]/g, '')
			.toLowerCase()
			.replace(/[\s_.]+/g, '-')
			.replace(/[^a-z0-9-]/g, '')
			.slice(0, 48);

	function escribir(e: Event & { currentTarget: HTMLInputElement }) {
		nuevo = limpiar(e.currentTarget.value);
		e.currentTarget.value = nuevo;
		listo = false;
		error = '';
		estado = null;
		clearTimeout(t);
		if (nuevo === slug) {
			estado = 'propia';
			return;
		}
		if (nuevo.length < 3) return;
		const n = ++pedido;
		t = setTimeout(async () => {
			try {
				const r = await fetch(`/panel/link?slug=${encodeURIComponent(nuevo)}`);
				const d = (await r.json()) as { estado: EstadoLink };
				if (n === pedido) estado = d.estado;
			} catch {
				// Sin conexión: queda «buscando»; al guardar, la API igual lo valida.
			}
		}, 300);
	}

	const campo = $derived(campoLink(nuevo, estado));
	const sinCambios = $derived(link?.restantes === 0);
	const puede = $derived(campo.ok && !guardando && !sinCambios);

	async function guardar(e: SubmitEvent) {
		e.preventDefault();
		if (!puede) return;
		guardando = true;
		error = '';
		try {
			const r = await fetch('/panel/link', { method: 'POST', headers: { 'content-type': 'application/json' }, body: JSON.stringify({ slug: nuevo }) });
			const d = await r.json();
			if (!r.ok) {
				error = d.error ?? 'No se pudo cambiar. Probá de nuevo.';
				return;
			}
			link = d as Link;
			listo = true;
			await invalidateAll();
			alGuardar?.(link);
		} catch {
			error = 'Sin conexión. Probá de nuevo.';
		} finally {
			guardando = false;
		}
	}
</script>

<form class="editor" onsubmit={guardar}>
	<span class="previa" aria-live="polite">
		<span class="host">{host}/</span><b>{nuevo || '…'}</b>
	</span>
	<label class="caja" style:--borde={campo.borde}>
		<span class="sr">Tu link nuevo</span>
		<span class="pre" aria-hidden="true">{host}/</span>
		<input
			value={nuevo}
			oninput={escribir}
			autocapitalize="none"
			autocomplete="off"
			spellcheck="false"
			inputmode="url"
			maxlength="48"
			aria-describedby="link-msg"
		/>
		<span style:color={campo.borde}><Icono nombre={campo.icono} lleno tam={22} /></span>
	</label>
	<span id="link-msg" class="msg" style:color={campo.color} role="status">{campo.mensaje}</span>

	<div class="aviso">
		<Icono nombre="verified" lleno tam={22} />
		<span><b>Tu link de antes sigue andando.</b> Los QR y carteles impresos llevan al&nbsp;nuevo.</span>
	</div>

	{#if error}<p class="error" role="alert"><Icono nombre="error" lleno tam={18} />{error}</p>{/if}
	{#if listo}<p class="ok" role="status"><Icono nombre="check_circle" lleno tam={18} />Listo. Tu link nuevo ya&nbsp;anda.</p>{/if}

	<div class="pie">
		<button class="guardar" disabled={!puede}>{guardando ? 'Guardando…' : 'Guardar link'}</button>
		{#if link}
			<span class="quedan tl-num">
				{#if sinCambios}Ya lo cambiaste 5 veces este&nbsp;mes.{:else}Te {link.restantes === 1 ? 'queda 1 cambio' : `quedan ${link.restantes} cambios`} este&nbsp;mes.{/if}
			</span>
		{/if}
	</div>

	{#if link?.anteriores.length}
		<div class="anteriores">
			<span class="tl-rotulo">Siguen llevando a tu página</span>
			<ul>
				{#each link.anteriores as a (a.slug)}<li><Icono nombre="subdirectory_arrow_right" tam={16} />{host}/{a.slug}</li>{/each}
			</ul>
		</div>
	{/if}
</form>

<style>
	.editor {
		display: flex;
		flex-direction: column;
		gap: 10px;
		min-width: 0;
	}
	.previa {
		font-size: 26px;
		line-height: 1.1;
		font-weight: 900;
		letter-spacing: -0.03em;
		overflow-wrap: anywhere;
	}
	.previa .host {
		color: var(--tl-sub);
		font-weight: 700;
	}
	.previa b {
		color: #17805a;
		font-weight: 900;
	}
	/* En el teléfono el host va arriba y chico, como en TuLink: el slug entero y sin cortes raros. */
	@media (max-width: 899px) {
		.previa .host {
			display: block;
			font-size: 15px;
			letter-spacing: 0;
		}
		.pre {
			font-size: 14px;
		}
	}
	.caja {
		--borde: #ddd1b9;
		display: flex;
		align-items: center;
		height: 56px;
		padding: 0 14px;
		border-radius: 16px;
		background: #fff;
		box-shadow: inset 0 0 0 2px var(--borde);
		font-size: 16px;
	}
	.pre {
		color: var(--tl-sub);
		white-space: nowrap;
	}
	.caja input {
		flex: 1;
		min-width: 0;
		height: 100%;
		border: 0;
		background: none;
		font: 700 16px var(--tl-font);
		color: var(--tl-tinta);
		outline: none;
	}
	.sr {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip: rect(0 0 0 0);
	}
	.msg {
		font-size: 13px;
		font-weight: 700;
	}
	.aviso {
		display: flex;
		gap: 10px;
		align-items: flex-start;
		padding: 12px 14px;
		border-radius: 18px;
		background: #e3f6ec;
		font-size: 14px;
		line-height: 1.4;
		color: #102b31;
		text-wrap: pretty;
	}
	.aviso :global(.icono) {
		color: #17805a;
		flex: none;
	}
	.error,
	.ok {
		display: flex;
		align-items: center;
		gap: 6px;
		margin: 0;
		font-size: 14px;
		font-weight: 700;
	}
	.error {
		color: #a3241a;
	}
	.ok {
		color: #17805a;
	}
	.pie {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px 14px;
	}
	.guardar {
		min-height: 48px;
		padding: 0 24px;
		border: 0;
		border-radius: 999px;
		background: #3ccc90;
		color: #102b31;
		font: 700 15px var(--tl-font);
		cursor: pointer;
		transition: transform var(--tl-tocar);
	}
	.guardar:active {
		transform: scale(0.96);
	}
	.guardar:disabled {
		background: #e6dcc8;
		color: var(--tl-sub);
		cursor: default;
	}
	.quedan {
		font-size: 13px;
		color: var(--tl-sub);
	}
	.anteriores {
		display: flex;
		flex-direction: column;
		gap: 6px;
		color: var(--tl-sub);
	}
	.anteriores ul {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
		font-size: 14px;
	}
	.anteriores li {
		display: flex;
		align-items: center;
		gap: 6px;
		overflow-wrap: anywhere;
	}
</style>
