<script lang="ts">
	import '#lib/gestion/gestion.css';

	let { data } = $props();
	const esDuenio = $derived(data.negocio.rol === 'duenio');
	const permiso = $derived(data.motivo === 'permiso');
</script>

<svelte:head><title>Mercado Pago · Panel</title></svelte:head>

<div class="g-pagina vuelta">
	<p class="banda"><span class="icono lleno" aria-hidden="true">error</span>No se pudo conectar</p>
	<div class="cuerpo">
		<span class="g-kicker">Mercado Pago</span>
		<h1 class="g-titulo-2">
			{#if permiso}Cancelaste el permiso o&nbsp;venció{:else}No pudimos conectar Mercado&nbsp;Pago{/if}
		</h1>
		<p class="g-bajada" role="alert">{data.error}</p>

		<ol class="pasos">
			<li class="g-paso ok">
				<span class="g-paso-num"><span class="icono" aria-hidden="true">check</span></span>
				<span class="g-paso-cuerpo">
					<b>Entrá con la cuenta donde querés recibir la&nbsp;plata</b>
					<span>No tiene que ser la de tu mail de Tengo&nbsp;Libre.</span>
				</span>
			</li>
			<li class="g-paso ahora">
				<span class="g-paso-num"><span class="icono" aria-hidden="true">touch_app</span></span>
				<span class="g-paso-cuerpo">
					<b>Tocá «Permitir»</b>
					<span>{permiso ? 'Es el paso que faltó.' : 'Así Tengo Libre puede cobrar las señas a tu nombre.'}</span>
				</span>
			</li>
		</ol>
		<p class="g-aviso ambar">
			<span class="icono lleno" aria-hidden="true">help</span>
			<span>¿Tu cuenta es de empresa? La tiene que conectar quien la&nbsp;administra.</span>
		</p>

		<div class="acciones">
			{#if esDuenio}
				<!-- Sin enhance: la respuesta es una redirección a Mercado Pago. -->
				<form method="POST" action="/panel/pagos?/conectar">
					<button class="g-cta">Intentar de nuevo</button>
				</form>
			{/if}
			<a class="g-accion neutro ahora-no" href="/panel/pagos">{esDuenio ? 'Ahora no' : 'Volver a Pagos'}</a>
		</div>
	</div>
</div>

<style>
	.vuelta {
		max-width: 560px;
		gap: 0;
		border-radius: var(--tl-r-l);
		overflow: hidden;
		background: var(--tl-crema-pagina);
	}
	.banda {
		margin: 0;
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 16px 20px;
		background: #5b1a13;
		color: #fff;
		font-size: 15px;
	}
	.banda .icono {
		color: #f4b7ae;
	}
	.cuerpo {
		display: flex;
		flex-direction: column;
		gap: 14px;
		padding: 20px;
	}
	.pasos {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.acciones {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 4px;
		margin-top: 12px;
	}
	.acciones form {
		width: 100%;
	}
	.ahora-no {
		min-height: 48px;
		font-size: 15px;
	}
</style>
