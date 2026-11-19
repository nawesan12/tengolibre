<!--
	El aviso del art. 6 de la Ley 25.326, junto a los datos que se piden (reserva y lista de espera),
	debajo de los campos y antes del ticket: quién los recibe (el negocio, responsable) y que Tengo Libre
	los procesa por cuenta del negocio (encargado), para qué se usan, que son obligatorios, cómo ejercer
	los derechos y el link a la política. «Por su cuenta» se leía como «por cuenta propia». Lo completo
	está en /privacidad. El texto en capas lo tiene que revisar legal (el registro en la AAIP está en
	trámite).
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
	Tus datos los recibe <b>{negocio}</b>, que da el turno; Tengo Libre los procesa por cuenta de {negocio}. Se usan {uso}.
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
