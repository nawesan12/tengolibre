<!--
	El aviso del art. 6 de la Ley 25.326, antes de pedir los datos (reserva y lista de espera): quién los
	recibe (el negocio, responsable) y que Tengo Libre los procesa por su cuenta (encargado), para qué
	se usan, que son obligatorios, cómo ejercer los derechos y el link a la política. Corto: lo
	completo está en /privacidad.
-->
<script lang="ts">
	let { negocio, para = 'reservar' }: { negocio: string; para?: 'reservar' | 'espera' } = $props();
	const uso = $derived(
		para === 'espera'
			? 'para avisarte si se libera un lugar'
			: 'para darte el turno, confirmártelo, recordártelo y avisarte si cambia'
	);
</script>

<p class="aviso-datos">
	Tus datos los recibe <b>{negocio}</b>, que da el turno; Tengo Libre los guarda y procesa por su cuenta. Se usan {uso}.
	{para === 'espera' ? 'Nombre y email son obligatorios: sin eso no podemos anotarte.' : 'Nombre y teléfono o email son obligatorios: sin eso no podemos reservarte el\u00a0turno.'} Podés pedir verlos,
	corregirlos o borrarlos al negocio o&nbsp;a&nbsp;<a href="mailto:privacidad@tengolibre.com.ar">privacidad@tengolibre.com.ar</a>. Más en la
	<a href="/privacidad" target="_blank" rel="noopener">política de&nbsp;privacidad</a>.
</p>

<style>
	.aviso-datos {
		margin: 0;
		font-size: 12px;
		line-height: 1.5;
		text-wrap: pretty;
		color: var(--w-sub);
	}
	.aviso-datos a {
		color: inherit;
		text-decoration: underline;
	}
</style>
