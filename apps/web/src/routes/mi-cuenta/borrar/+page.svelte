<script lang="ts">
	import { enhance } from '$app/forms';
	import Icono from '#lib/ui/Icono.svelte';
	import Logo from '#lib/ui/Logo.svelte';

	let { data, form } = $props();
	let email = $state('');
	let enviando = $state(false);
	const p = $derived(data.persona);
	const coincide = $derived(!!p && email.trim().toLowerCase() === p.email.trim().toLowerCase());
</script>

<svelte:head>
	<title>Borrar mi cuenta · Tengo Libre</title>
	<meta name="description" content="Cómo borrar tu cuenta de Tengo Libre y qué pasa con tus datos." />
</svelte:head>

<div class="pagina">
	<main class="ancho">
		<a href="/" class="inicio" aria-label="Tengo Libre, inicio"><Logo alto={24} /></a>

		{#if data.listo}
			<section class="hoja" role="status">
				<span class="sello listo"><Icono nombre="check_circle" lleno tam={30} /></span>
				<h1>Tu cuenta se&nbsp;borró</h1>
				<p>Ya no podés entrar con ella. Tus datos personales se borran del todo a los 30&nbsp;días.</p>
				<p class="suave">
					Si fue un error, escribinos a <a href="mailto:hola@tengolibre.com.ar">hola@tengolibre.com.ar</a> dentro de esos 30 días y la&nbsp;recuperamos.
				</p>
				<a class="boton" href="/">Ir al inicio</a>
			</section>
		{:else}
			<section class="hoja">
				<span class="sello"><Icono nombre="person_remove" tam={30} /></span>
				<h1>Borrar mi&nbsp;cuenta</h1>
				<p class="clave">Tu cuenta se desactiva ahora y tus datos personales se borran a los 30&nbsp;días.</p>
				<p>En esos 30 días la recuperás escribiendo a <a href="mailto:hola@tengolibre.com.ar">hola@tengolibre.com.ar</a>. Después no se puede&nbsp;deshacer.</p>

				<h2 class="tl-rotulo rotulo">Qué pasa</h2>
				<ul class="lista">
					<li><Icono nombre="logout" tam={20} /><span>Se cierran tus sesiones en la web y en la app, y dejan de llegarte&nbsp;avisos.</span></li>
					<li><Icono nombre="group_remove" tam={20} /><span>Salís de los equipos de los negocios donde trabajás. Los negocios siguen&nbsp;andando.</span></li>
					<li><Icono nombre="storefront" tam={20} /><span>Si sos la única dueña de un negocio, se cierra: su página deja de tomar reservas y su suscripción de Mercado Pago se&nbsp;cancela.</span></li>
					<li><Icono nombre="smartphone" tam={20} /><span>Si pagás con la App Store o Google Play, cancelá la suscripción ahí: no la podemos cancelar&nbsp;nosotros.</span></li>
				</ul>

				{#if p}
					{#if data.duenia.length}
						<p class="aviso-negocios">
							<Icono nombre="warning" tam={20} />
							<span>Sos dueña de <b>{data.duenia.join(', ')}</b>. Si nadie más es dueña, se&nbsp;cierra.</span>
						</p>
					{/if}
					<form
						method="POST"
						class="confirmar"
						use:enhance={() => {
							enviando = true;
							return async ({ update }) => {
								await update({ reset: false });
								enviando = false;
							};
						}}
					>
						<label class="campo">
							<span>Para confirmar, escribí tu mail: <b class="tl-num mail">{p.email}</b></span>
							<input
								name="email"
								type="email"
								autocomplete="off"
								autocapitalize="off"
								spellcheck="false"
								inputmode="email"
								required
								bind:value={email}
								aria-invalid={form?.error ? 'true' : undefined}
							/>
							{#if form?.error}<span class="error" role="alert">{form.error}</span>{/if}
						</label>
						<button class="boton borrar" disabled={!coincide || enviando}>
							<Icono nombre="delete_forever" tam={20} />{enviando ? 'Borrando…' : 'Borrar mi cuenta'}
						</button>
						<a class="volver" href="/mi-cuenta">No, volver a mi cuenta</a>
					</form>
				{:else}
					<p class="suave">Para borrarla, ingresá con la misma cuenta que usás en Tengo&nbsp;Libre.</p>
					<a class="boton oscuro" href="/ingresar?volver=/mi-cuenta/borrar">Ingresar para borrar mi&nbsp;cuenta</a>
					<p class="suave chico">
						¿No podés entrar? Escribinos desde el mail de tu cuenta a <a href="mailto:hola@tengolibre.com.ar">hola@tengolibre.com.ar</a> y la borramos&nbsp;nosotros.
					</p>
				{/if}
			</section>
		{/if}
	</main>
</div>

<style>
	.pagina {
		min-height: 100dvh;
		background-color: var(--tl-crema);
		background-image: radial-gradient(rgba(24, 60, 68, 0.08) 1px, transparent 1.3px);
		background-size: 18px 18px;
	}
	.ancho {
		width: min(560px, 100%);
		margin: 0 auto;
		padding: 16px 16px 48px;
		display: flex;
		flex-direction: column;
		gap: 18px;
	}
	.inicio {
		display: inline-flex;
		align-self: flex-start;
		align-items: center;
		min-height: 48px;
	}
	.hoja {
		display: flex;
		flex-direction: column;
		gap: 14px;
		padding: 24px 20px;
		border-radius: var(--tl-r-l);
		background: var(--tl-papel);
		box-shadow: var(--tl-sh-tarjeta);
		animation: tl-gin var(--tl-entrar) var(--tl-ease) both;
	}
	.sello {
		width: 56px;
		height: 56px;
		border-radius: 16px;
		display: grid;
		place-items: center;
		background: #fbe0dc;
		color: #a3241a;
	}
	.sello.listo {
		background: var(--tl-menta-suave);
		color: var(--tl-menta-texto);
	}
	h1 {
		margin: 4px 0 0;
		font-size: clamp(30px, 6vw, 38px);
		line-height: 1;
		font-weight: 700;
		letter-spacing: -0.05em;
	}
	p {
		margin: 0;
		font-size: 15px;
		line-height: 1.5;
	}
	.clave {
		font-size: 17px;
		font-weight: 700;
	}
	.suave {
		color: var(--tl-sub);
	}
	.chico {
		font-size: 13px;
	}
	a:not(.boton):not(.volver):not(.inicio) {
		color: var(--tl-menta-texto);
		font-weight: 700;
		overflow-wrap: anywhere;
	}
	.rotulo {
		margin: 8px 0 0;
		font-size: 12px;
		color: var(--tl-sub);
	}
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 12px;
		font-size: 14px;
		line-height: 1.45;
	}
	.lista li {
		display: flex;
		gap: 10px;
	}
	.lista :global(.icono) {
		flex: none;
		color: var(--tl-petroleo);
	}
	.aviso-negocios {
		display: flex;
		gap: 10px;
		padding: 14px;
		border-radius: var(--tl-r-m);
		background: #f3e8d0;
		color: #4a3a17;
		font-size: 14px;
		line-height: 1.45;
	}
	.aviso-negocios :global(.icono) {
		flex: none;
		color: #7a5a1e;
	}
	.confirmar {
		display: flex;
		flex-direction: column;
		gap: 14px;
		margin-top: 4px;
	}
	.campo span {
		font-weight: 400;
		line-height: 1.45;
	}
	.mail {
		overflow-wrap: anywhere;
	}
	.boton {
		min-height: 52px;
	}
	.borrar {
		background: #a3241a;
		border-color: transparent;
		color: #fff;
	}
	.borrar:hover {
		background: #8c1e15;
	}
	.volver {
		align-self: center;
		min-height: 44px;
		display: grid;
		place-items: center;
		font-size: 14px;
		font-weight: 700;
		color: var(--tl-sub);
		text-decoration: none;
	}
	.volver:hover {
		text-decoration: underline;
	}
	@media (min-width: 720px) {
		.ancho {
			padding-top: 32px;
		}
		.hoja {
			padding: 32px;
		}
	}
</style>
