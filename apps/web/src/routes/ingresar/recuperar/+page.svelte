<script lang="ts">
	import { tick, untrack } from 'svelte';
	import {
		confirmarCodigo,
		ErrorClerk,
		guardarClave,
		mensajeClerk,
		pedirRecuperar,
		requisitosClave,
		type FlujoMail,
		type PasoMail
	} from '#lib/clerk.ts';
	import Icono from '#lib/ui/Icono.svelte';
	import Requisitos from '#lib/cuenta/Requisitos.svelte';

	let { data } = $props();

	/*
	 * Recuperar la contraseña con Clerk (calcado de Corredor): un código por mail
	 * (reset_password_email_code), con el código se elige la contraseña nueva, se cierran las otras
	 * sesiones y se entra. Si el mail no tiene cuenta, la pantalla dice lo mismo (no se revela quién
	 * está registrado). Desde un navegador nuevo, Clerk puede pedir otro código para confirmarlo.
	 */
	let email = $state(untrack(() => data.email));
	let codigo = $state('');
	let clave = $state('');
	let ver = $state(false);
	let paso = $state<'mail' | 'codigo' | 'nueva' | 'confiar'>('mail');
	let enviado = $state(false);
	let enviando = $state(false);
	let error = $state('');
	let flujo: FlujoMail | null = null;
	let token = $state('');
	let form: HTMLFormElement;
	const req = $derived(requisitosClave(clave));

	async function hacer(fn: () => Promise<void>) {
		enviando = true;
		error = '';
		try {
			await fn();
		} catch (x) {
			error = x instanceof ErrorClerk ? mensajeClerk(x.codigo) : 'No pudimos conectar. Probá de nuevo.';
		} finally {
			if (!token) enviando = false;
		}
	}

	async function seguir(sig: PasoMail) {
		if (sig.paso === 'listo') {
			token = sig.token;
			await tick();
			form.requestSubmit();
		} else if (sig.paso === 'clave_nueva') paso = 'nueva';
		else if (sig.paso === 'codigo') {
			codigo = '';
			paso = 'confiar';
		}
	}

	function enviar(e: SubmitEvent) {
		e.preventDefault();
		const fapi = data.fapi;
		if (!fapi || enviando) return;
		if (paso === 'mail')
			void hacer(async () => {
				flujo = await pedirRecuperar(fapi, email.trim());
				enviado = true;
				if (flujo) paso = 'codigo';
			});
		else if ((paso === 'codigo' || paso === 'confiar') && flujo)
			void hacer(async () => seguir(await confirmarCodigo(flujo!, codigo, paso === 'codigo')));
		else if (paso === 'nueva' && flujo) {
			if (!req.largo || !req.comun) return void (error = 'La contraseña todavía no cumple los requisitos.');
			void hacer(async () => seguir(await guardarClave(flujo!, clave)));
		}
	}
</script>

<svelte:head>
	<title>{paso === 'nueva' ? 'Contraseña nueva' : 'Recuperar la contraseña'} · Tengo Libre</title>
	<meta name="robots" content="noindex" />
</svelte:head>

<div class="pagina tl-petroleo">
	<img class="marca-agua" src="/marca/simbolo_una_tinta_blanco.svg" alt="" aria-hidden="true" />
	<main class="cuerpo">
		<span class="llave"><Icono nombre={paso === 'nueva' ? 'password' : 'key'} tam={30} /></span>
		{#if paso === 'nueva'}
			<h1>Elegí una contraseña&nbsp;nueva</h1>
		{:else}
			<h1>¿Te olvidaste la&nbsp;contraseña?</h1>
			<p class="bajada">
				Si entrás con Google o Apple, no necesitás contraseña: usá ese botón. Si usás tu mail, te mandamos un código para crear una&nbsp;nueva.
			</p>
		{/if}
		<form onsubmit={enviar}>
			{#if paso === 'mail'}
				<label class="solo-lector" for="email">Tu mail</label>
				<input id="email" class="entrada" type="email" autocomplete="email" inputmode="email" required placeholder="tu@mail.com" bind:value={email} />
				<button class="seguir" disabled={enviando || !data.fapi}>{enviando ? 'Mandando…' : 'Mandarme el código'}</button>
			{:else if paso === 'codigo' || paso === 'confiar'}
				<label class="etiqueta" for="codigo">
					{paso === 'confiar' ? `Para confirmar este dispositivo, escribí el código nuevo que te llegó a ${email}` : `Código que te llegó a ${email}`}
				</label>
				<input id="codigo" class="entrada codigo tl-num" inputmode="numeric" autocomplete="one-time-code" maxlength="6" required bind:value={codigo} />
				<button class="seguir" disabled={enviando}>{enviando ? 'Revisando…' : 'Seguir'}</button>
			{:else}
				<label class="solo-lector" for="clave">Contraseña nueva</label>
				<div class="con-ojo">
					<input id="clave" class="entrada" type={ver ? 'text' : 'password'} autocomplete="new-password" required bind:value={clave} />
					<button type="button" class="ojo" aria-label={ver ? 'Ocultar la contraseña' : 'Ver la contraseña'} onclick={() => (ver = !ver)}>
						<Icono nombre={ver ? 'visibility_off' : 'visibility'} tam={22} />
					</button>
				</div>
				<Requisitos {req} />
				<button class="seguir" disabled={enviando}>{enviando ? 'Guardando…' : 'Guardar y entrar'}</button>
				<p class="chico">Cerramos las otras sesiones abiertas por&nbsp;seguridad.</p>
			{/if}
			{#if error}<p class="error-ingreso" role="alert"><Icono nombre="error" lleno tam={20} /><span>{error}</span></p>{/if}
		</form>
		{#if enviado && paso === 'codigo'}
			<p class="aviso-mail" role="status">
				<Icono nombre="mark_email_read" tam={20} />
				<span>Listo: si ese mail tiene cuenta, te llega un código en un minuto. Revisá también&nbsp;spam.</span>
			</p>
		{:else if enviado && paso === 'mail' && !error}
			<p class="aviso-mail" role="status">
				<Icono nombre="mark_email_read" tam={20} />
				<span>Si ese mail tiene una cuenta con contraseña, te llega un código en un minuto. Si entrás con Google o Apple, usá ese&nbsp;botón.</span>
			</p>
		{/if}
		<a class="volver" href="/ingresar">Volver a ingresar</a>
	</main>
</div>

<form method="POST" action="/ingresar/clerk" bind:this={form} hidden>
	<input type="hidden" name="token" value={token} />
</form>

<style>
	.pagina {
		position: relative;
		overflow: hidden;
		min-height: 100dvh;
		display: flex;
		justify-content: center;
		padding: 64px 24px calc(28px + env(safe-area-inset-bottom));
	}
	.marca-agua {
		position: absolute;
		left: -120px;
		bottom: -120px;
		width: 420px;
		height: 420px;
		opacity: 0.04;
		pointer-events: none;
	}
	.cuerpo {
		position: relative;
		width: min(100%, 420px);
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	.llave {
		width: 56px;
		height: 56px;
		border-radius: 16px;
		background: var(--tl-petroleo-hondo);
		color: var(--tl-menta);
		display: grid;
		place-items: center;
		box-shadow: var(--tl-luz-arriba-osc);
	}
	h1 {
		margin: 8px 0 0;
		font-size: 34px;
		line-height: 1;
		font-weight: 700;
		letter-spacing: -0.05em;
	}
	.bajada {
		margin: 0;
		font-size: 15px;
		line-height: 1.45;
		color: rgba(255, 255, 255, 0.75);
	}
	form {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.etiqueta {
		font-size: 14px;
		font-weight: 700;
		line-height: 1.4;
		overflow-wrap: anywhere;
	}
	.entrada {
		width: 100%;
		box-sizing: border-box;
		height: 56px;
		padding: 0 20px;
		border: 0;
		border-radius: 999px;
		background: var(--tl-papel);
		color: var(--tl-tinta);
		font: 400 16px var(--tl-font);
	}
	.entrada:focus {
		outline: none;
		box-shadow: 0 0 0 3px rgba(60, 204, 144, 0.55);
	}
	.codigo {
		font-size: 22px;
		letter-spacing: 0.3em;
		text-align: center;
	}
	.con-ojo {
		position: relative;
	}
	.con-ojo .entrada {
		padding-right: 56px;
	}
	.ojo {
		position: absolute;
		right: 6px;
		top: 50%;
		transform: translateY(-50%);
		width: 44px;
		height: 44px;
		border: 0;
		background: none;
		color: var(--tl-sub);
		display: grid;
		place-items: center;
		cursor: pointer;
	}
	.seguir {
		height: 56px;
		border: 0;
		border-radius: 999px;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		font: 900 16px var(--tl-font);
		cursor: pointer;
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.4), 0 12px 24px -14px rgba(60, 204, 144, 0.9);
	}
	.seguir:disabled {
		cursor: progress;
		opacity: 0.7;
	}
	.chico {
		margin: 0;
		font-size: 13px;
		color: rgba(255, 255, 255, 0.7);
	}
	.error-ingreso,
	.aviso-mail {
		display: flex;
		gap: 10px;
		margin: 0;
		padding: 14px;
		border-radius: 16px;
		font-size: 14px;
		line-height: 1.45;
	}
	.error-ingreso {
		background: #fbe0dc;
		color: #5b1a13;
	}
	.aviso-mail {
		background: rgba(255, 255, 255, 0.08);
		color: rgba(255, 255, 255, 0.9);
	}
	.aviso-mail :global(.icono) {
		flex: none;
		color: var(--tl-menta);
	}
	.volver {
		align-self: center;
		min-height: 44px;
		display: grid;
		place-items: center;
		font-size: 14px;
		font-weight: 700;
		color: var(--tl-menta);
		text-decoration: none;
	}
	.solo-lector {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip-path: inset(50%);
		white-space: nowrap;
	}
	@media (min-width: 720px) {
		.pagina {
			padding-top: 96px;
		}
	}
</style>
