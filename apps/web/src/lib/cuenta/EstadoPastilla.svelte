<!-- Pastilla de estado: siempre ícono + palabra (nunca sólo color). `plano` la deja sin fondo (filas del admin). -->
<script lang="ts">
	import Icono from '#lib/ui/Icono.svelte';
	import type { Estado } from './presentacion.ts';
	let { estado, plano = false }: { estado: Estado; plano?: boolean } = $props();
</script>

<span class="estado {estado.tono}" class:plano><Icono nombre={estado.icono} lleno tam={plano ? 16 : 15} />{estado.texto}</span>

<style>
	.estado {
		display: inline-flex;
		align-self: flex-start;
		align-items: center;
		gap: 5px;
		min-height: 26px;
		padding: 0 10px 0 8px;
		border-radius: 999px;
		font-size: 12px;
		font-weight: 700;
		white-space: nowrap;
	}
	.ok {
		background: var(--tl-menta-suave);
		color: var(--tl-menta-texto);
	}
	.aviso {
		background: #fbe9c2;
		color: #7a4a00;
	}
	.apagado {
		background: #ece5d6;
		color: var(--tl-sub);
	}
	.peligro {
		background: #fbe0dc;
		color: #a3241a;
	}
	.plano {
		background: none;
		padding: 0;
		font-size: 13px;
		white-space: normal;
	}
</style>
