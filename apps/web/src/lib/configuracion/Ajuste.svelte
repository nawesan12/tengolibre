<!--
	Una fila de la configuración (handoff v2, «Configuración»): qué cambia para la clienta a la
	izquierda y el valor en una pastilla a la derecha. Opciones: un select; sí/no: un interruptor con
	la palabra. El formulario del grupo se manda solo al cambiar (lo hace la página).
-->
<script lang="ts">
	type Opcion = { valor: number; texto: string };
	let {
		etiqueta,
		ayuda,
		nombre,
		valor,
		opciones,
		sino = false,
		deshabilitado = false,
		clave
	}: {
		etiqueta: string;
		ayuda: string;
		nombre: string;
		valor: number | boolean;
		opciones?: Opcion[];
		sino?: boolean;
		/** Se ve pero no se puede cambiar (p. ej. WhatsApp sin configurar en la plataforma). */
		deshabilitado?: boolean;
		/** Para el id del campo cuando el mismo nombre se repite en la página. */
		clave?: string;
	} = $props();
	const id = $derived(`aj-${clave ?? nombre}`);
	// Sin palabra sola en el último renglón: la última va pegada a la anterior.
	const pegar = (s: string) => s.replace(/ (\S+)$/, ' $1');
	// svelte-ignore state_referenced_locally
	let marcado = $state(valor === true);
	// El valor de ahora siempre está entre las opciones (aunque se haya guardado uno fuera de la lista).
	const lista = $derived.by(() => {
		const o = opciones ?? [];
		if (typeof valor !== 'number' || o.some((x) => x.valor === valor)) return o;
		return [...o, { valor, texto: String(valor) }].sort((a, b) => a.valor - b.valor);
	});
</script>

<div class="ajuste">
	<label class="texto" for={id}>
		<span class="t">{pegar(etiqueta)}</span>
		<span class="s">{pegar(ayuda)}</span>
	</label>
	{#if sino}
		<span class="g-valor sino" class:si={marcado}>
			<input class="visualmente-oculto" type="checkbox" role="switch" {id} name={nombre} bind:checked={marcado} disabled={deshabilitado} />
			<span aria-hidden="true">{marcado ? 'Sí' : 'No'}</span>
			<span class="icono lleno" aria-hidden="true">{marcado ? 'toggle_on' : 'toggle_off'}</span>
		</span>
	{:else}
		<select class="g-valor" {id} name={nombre} disabled={deshabilitado}>
			{#each lista as o (o.valor)}<option value={o.valor} selected={o.valor === valor}>{o.texto}</option>{/each}
		</select>
	{/if}
</div>

<style>
	.ajuste {
		display: grid;
		grid-template-columns: minmax(0, 1fr) auto;
		align-items: center;
		gap: 16px;
		min-height: 60px;
		padding: 8px 18px;
		border-top: 1px solid #f0e8d8;
	}
	.texto {
		display: flex;
		flex-direction: column;
		gap: 2px;
		cursor: pointer;
	}
	.t {
		font-size: 14px;
		font-weight: 700;
	}
	.s {
		font-size: 12px;
		color: var(--tl-sub);
		text-wrap: pretty;
	}
	select.g-valor {
		max-width: 46vw;
	}
	.sino {
		position: relative;
		cursor: pointer;
		padding: 0 10px 0 14px;
		color: var(--tl-sub-2);
	}
	.sino.si {
		background: var(--tl-menta-suave);
		color: var(--tl-menta-texto);
	}
	.sino .icono {
		font-size: 24px;
	}
	/* El checkbox real cubre la pastilla: se toca en cualquier parte. */
	.sino input {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		clip: auto;
		opacity: 0;
		margin: 0;
		cursor: pointer;
	}
	.sino:has(input:disabled),
	select:disabled {
		opacity: 0.55;
		cursor: not-allowed;
	}
	.sino:has(input:focus-visible) {
		outline: 3px solid var(--tl-menta);
		outline-offset: 2px;
	}
	@media (max-width: 640px) {
		.ajuste {
			padding: 8px 14px;
			gap: 12px;
		}
	}
</style>
