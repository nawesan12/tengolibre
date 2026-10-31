<!-- Un QR de verdad (uqr) como SVG: se escanea con la cámara y lleva a `texto`. -->
<script lang="ts">
	import { encode } from 'uqr';
	let { texto, color = '#0f2328', fondo = '#fff', etiqueta }: { texto: string; color?: string; fondo?: string; etiqueta: string } = $props();
	const qr = $derived(encode(texto, { ecc: 'M', border: 1 }));
	const camino = $derived(
		qr.data.flatMap((fila, y) => fila.map((on, x) => (on ? `M${x} ${y}h1v1h-1z` : ''))).join('')
	);
</script>

<svg viewBox="0 0 {qr.size} {qr.size}" role="img" aria-label={etiqueta} shape-rendering="crispEdges">
	<rect width={qr.size} height={qr.size} fill={fondo} />
	<path d={camino} fill={color} />
</svg>

<style>
	svg {
		display: block;
		width: 100%;
		height: 100%;
	}
</style>
