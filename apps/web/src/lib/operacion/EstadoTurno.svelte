<!-- La pastilla de estado de un turno: ícono + palabra (nunca sólo color). -->
<script lang="ts">
	import Icono from '#lib/ui/Icono.svelte';
	import { ESTILO } from './estados.ts';

	let { estado, chico = false }: { estado: string; chico?: boolean } = $props();
	const e = $derived(ESTILO[estado] ?? { texto: estado, icono: 'info', fondo: '#ece5d6', color: '#0f2328', borde: 'none' });
</script>

<span class="estado" class:chico style:background={e.fondo} style:color={e.color} style:box-shadow={e.borde}>
	<Icono nombre={e.icono} lleno tam={chico ? 13 : 14} /><span>{e.texto}</span>
</span>

<style>
	.estado {
		display: inline-flex;
		align-items: center;
		gap: 5px;
		height: 24px;
		padding: 0 9px 0 7px;
		border-radius: 999px;
		font-size: 12px;
		font-weight: 700;
		white-space: nowrap;
		line-height: 1;
	}
	.chico {
		height: 22px;
		gap: 4px;
		padding: 0 8px 0 6px;
		font-size: 11px;
	}
</style>
