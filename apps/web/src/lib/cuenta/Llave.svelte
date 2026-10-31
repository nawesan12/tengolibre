<!--
	Fila con llave (Cuenta y accesos v2, 1c): título, cuándo llega y un interruptor que es un checkbox
	de verdad (role="switch"). Prendida, la llave es menta texto con la perilla a la derecha y un tilde.
-->
<script lang="ts">
	let {
		nombre,
		titulo,
		detalle,
		checked = false,
		onchange
	}: { nombre: string; titulo: string; detalle: string; checked?: boolean; onchange?: (e: Event) => void } = $props();
</script>

<label class="fila-llave">
	<span class="textos">
		<b>{titulo}</b>
		<span>{detalle}</span>
	</span>
	<input type="checkbox" role="switch" name={nombre} {checked} {onchange} />
	<span class="llave" aria-hidden="true"><span class="perilla"></span></span>
</label>

<style>
	.fila-llave {
		position: relative;
		display: grid;
		grid-template-columns: 1fr auto;
		gap: 12px;
		align-items: center;
		min-height: 68px;
		padding: 10px 16px;
		border-top: 1px solid #f0e8d8;
		cursor: pointer;
	}
	.fila-llave:first-child {
		border-top: 0;
	}
	.textos {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}
	.textos b {
		font-size: 15px;
		font-weight: 700;
	}
	.textos span {
		font-size: 13px;
		color: var(--tl-sub);
	}
	/* El checkbox real queda encima de la llave dibujada (toda la fila también lo toca). */
	input {
		position: absolute;
		right: 16px;
		top: 50%;
		transform: translateY(-50%);
		width: 48px;
		height: 28px;
		margin: 0;
		opacity: 0;
		z-index: 1;
		cursor: pointer;
	}
	.llave {
		position: relative;
		width: 48px;
		height: 28px;
		border-radius: 999px;
		background: #d8ccb4;
		transition: background var(--tl-tocar) var(--tl-ease);
	}
	.perilla {
		position: absolute;
		top: 3px;
		left: 3px;
		width: 22px;
		height: 22px;
		border-radius: 50%;
		background: #fff;
		display: grid;
		place-items: center;
		transition: left var(--tl-tocar) var(--tl-ease);
	}
	/* Ningún estado sólo con color: prendida lleva un tilde en la perilla. */
	.perilla::after {
		content: '';
		width: 6px;
		height: 10px;
		margin-top: -2px;
		border: solid var(--tl-menta-texto);
		border-width: 0 2.5px 2.5px 0;
		transform: rotate(45deg);
		opacity: 0;
	}
	input:checked + .llave {
		background: var(--tl-menta-texto);
	}
	input:checked + .llave .perilla {
		left: 23px;
	}
	input:checked + .llave .perilla::after {
		opacity: 1;
	}
	input:focus-visible + .llave {
		outline: 3px solid var(--tl-menta);
		outline-offset: 2px;
	}
</style>
