<script lang="ts">
	import { ESTADOS } from '#lib/formato.ts';
	let { estado }: { estado: string } = $props();
</script>

<span class="etiqueta estado {estado}">{ESTADOS[estado] ?? estado}</span>

<style>
	.estado.cancelado,
	.estado.ausente {
		background: #fdecea;
		color: var(--peligro);
	}
	.estado.pendiente,
	.estado.pendiente_pago {
		background: #fff4e0;
		color: var(--aviso);
	}
	.estado.atendido {
		background: var(--borde);
	}
</style>
