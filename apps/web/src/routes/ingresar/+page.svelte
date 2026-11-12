<script lang="ts">
	import { tick } from 'svelte';
	import { enhance } from '$app/forms';
	import {
		confirmarCodigo,
		empezarConMail,
		entrarConClave,
		ErrorClerk,
		guardarClave,
		iniciar,
		mensajeClerk,
		requisitosClave,
		type FlujoMail,
		type PasoMail,
		type ProveedorClerk
	} from '#lib/clerk.ts';
	import Icono from '#lib/ui/Icono.svelte';
	import Requisitos from '#lib/cuenta/Requisitos.svelte';
	import { iniciales, textoSobre } from '#lib/cuenta/presentacion.ts';

	let { data, form } = $props();
	let error = $state('');
	let yendo = $state<ProveedorClerk | null>(null);

	const mensajes: Record<string, string> = {
		cancelado: 'Cancelaste el ingreso.',
		token_invalido: 'No pudimos verificar tu ingreso. Probá de&nbsp;nuevo.',
		sin_email: 'Tu cuenta no tiene un email verificado.',
		clerk_no_responde: 'El ingreso no responde. Probá en un&nbsp;rato.',
		cuenta_borrada:
			'Esta cuenta se borró. Si fue un error, escribinos a hola@tengolibre.com.ar dentro de los 30&nbsp;días y la&nbsp;recuperamos.'
	};
	const aviso = $derived(error || (data.error ? mensajes[data.error] || 'No pudimos completar el&nbsp;ingreso.' : ''));
	const n = $derived(data.negocio);
	const color = $derived(n?.color ?? '#183c44');

	async function con(p: ProveedorClerk) {
		if (!data.fapi) return;
		yendo = p;
		error = '';
		try {
			location.href = await iniciar(data.fapi, p, `${location.origin}/ingresar/clerk`);
		} catch (e) {
			error = mensajes[e instanceof ErrorClerk ? e.codigo : ''] ?? mensajes.clerk_no_responde;
			yendo = null;
		}
	}

	/*
	 * Con mail y contraseña, todo acá (lib/clerk.ts): contraseña o código, y si es la primera vez se
	 * crea la cuenta. Desde un navegador nuevo Clerk puede pedir confirmarlo con un código por mail.
	 * Al final el token de Clerk va a /ingresar/clerk, que lo canjea por la sesión de Tengo Libre.
	 */
	let paso = $state<'mail' | PasoMail['paso']>('mail');
	let email = $state('');
	let clave = $state('');
	let codigo = $state('');
	let alta = $state(false);
	let confiar = $state(false);
	let enviando = $state(false);
	let tokenMail = $state('');
	let flujo: FlujoMail | null = null;
	let formToken: HTMLFormElement;
	const req = $derived(requisitosClave(clave));

	async function hacer(fn: () => Promise<PasoMail>) {
		enviando = true;
		error = '';
		try {
			const sig = await fn();
			if (sig.paso === 'listo') {
				tokenMail = sig.token;
				await tick();
				formToken.requestSubmit();
				return;
			}
			alta = 'alta' in sig ? sig.alta : false;
			confiar = sig.paso === 'codigo' && !!sig.confiar;
			if (sig.paso === 'codigo') codigo = '';
			paso = sig.paso;
		} catch (x) {
			error = x instanceof ErrorClerk ? mensajeClerk(x.codigo) : 'No pudimos conectar. Probá de&nbsp;nuevo.';
		} finally {
			if (!tokenMail) enviando = false;
		}
	}

	function seguirConMail(e: SubmitEvent) {
		e.preventDefault();
		const fapi = data.fapi;
		if (!fapi || enviando) return;
		if (paso === 'mail')
			void hacer(async () => {
				const r = await empezarConMail(fapi, email.trim());
				flujo = r.flujo;
				return r.sigue;
			});
		else if (paso === 'clave' && flujo) void hacer(() => entrarConClave(flujo!, clave));
		else if (paso === 'codigo' && flujo) void hacer(() => confirmarCodigo(flujo!, codigo));
		else if (paso === 'clave_nueva' && flujo) {
			if (!req.largo || !req.comun) return void (error = 'La contraseña todavía no cumple los&nbsp;requisitos.');
			void hacer(() => guardarClave(flujo!, clave));
		}
	}

	function otroMail() {
		paso = 'mail';
		clave = codigo = error = '';
		flujo = null;
	}
</script>

<svelte:head><title>Ingresar · Tengo Libre</title></svelte:head>

{#snippet proveedores(claro: boolean)}
	{#if data.fapi}
		<button class="proveedor google" class:claro disabled={!!yendo} aria-busy={yendo === 'google'} onclick={() => con('google')}>
			<img src="/logos/google.svg" alt="" width="20" height="20" />
			{yendo === 'google' ? 'Abriendo Google…' : 'Seguir con Google'}
		</button>
		<button class="proveedor apple" disabled={!!yendo} aria-busy={yendo === 'apple'} onclick={() => con('apple')}>
			<img src="/logos/apple.svg" alt="" width="18" height="18" />
			{yendo === 'apple' ? 'Abriendo Apple…' : 'Seguir con Apple'}
		</button>
	{:else}
		<p class="sin-clerk">El ingreso con Google y Apple no está configurado (falta PUBLIC_CLERK_PUBLICABLE).</p>
	{/if}
{/snippet}

{#snippet conMail(claro: boolean)}
	{#if data.fapi}
		<form class="con-mail" class:claro onsubmit={seguirConMail}>
			{#if paso === 'mail'}
				<span class="separador">o con tu mail</span>
				<label class="solo-lector" for="email">Tu mail</label>
				<input id="email" class="entrada" type="email" autocomplete="email" inputmode="email" required placeholder="tu@mail.com" bind:value={email} />
				<button class="seguir" disabled={enviando || !!yendo}>{enviando ? 'Un segundo…' : 'Seguir con mail'}</button>
			{:else}
				<button type="button" class="volver" onclick={otroMail}><Icono nombre="arrow_back" tam={18} /><span class="tl-num">{email}</span></button>
				{#if paso === 'clave'}
					<label class="etiqueta" for="clave">Tu contraseña</label>
					<input id="clave" class="entrada" type="password" autocomplete="current-password" required bind:value={clave} />
					<button class="seguir" disabled={enviando}>{enviando ? 'Entrando…' : 'Entrar'}</button>
					<a class="olvide" href="/ingresar/recuperar?email={encodeURIComponent(email)}">¿Te olvidaste la contraseña?</a>
				{:else if paso === 'codigo'}
					<p class="explica">
						{#if confiar}
							Es la primera vez que entrás desde este dispositivo. Te mandamos un código a <b>{email}</b> para&nbsp;confirmarlo.
						{:else if alta}
							Te mandamos un código a <b>{email}</b>. Con él creamos tu&nbsp;cuenta.
						{:else}
							Te mandamos un código a <b>{email}</b>. Escribilo para&nbsp;entrar.
						{/if}
					</p>
					<label class="etiqueta" for="codigo">Código</label>
					<input id="codigo" class="entrada codigo tl-num" inputmode="numeric" autocomplete="one-time-code" maxlength="6" required bind:value={codigo} />
					<button class="seguir" disabled={enviando}>{enviando ? 'Revisando…' : alta ? 'Crear mi cuenta' : 'Entrar'}</button>
				{:else if paso === 'clave_nueva'}
					<p class="explica">Elegí una contraseña para entrar con tu&nbsp;mail.</p>
					<label class="etiqueta" for="nueva">Contraseña</label>
					<input id="nueva" class="entrada" type="password" autocomplete="new-password" required bind:value={clave} />
					<Requisitos {req} />
					<button class="seguir" disabled={enviando}>{enviando ? 'Guardando…' : 'Guardar y entrar'}</button>
				{/if}
			{/if}
		</form>
	{/if}
{/snippet}

{#snippet error_()}
	{#if aviso}
		<p class="error-ingreso" role="alert"><Icono nombre="error" lleno tam={20} /><span>{@html aviso}</span></p>
	{/if}
{/snippet}

{#snippet prueba()}
	{#if data.prueba}
		<form method="POST" action="?/prueba" use:enhance class="prueba">
			<span class="tl-rotulo">Ingreso de prueba</span>
			<p>Sólo en desarrollo: entra como esa persona, sin&nbsp;Clerk.</p>
			<label class="campo">
				Id de la persona
				<input name="id" value="user_prueba00001" autocomplete="off" />
			</label>
			{#if form?.error}<p class="campo"><span class="error">{form.error}</span></p>{/if}
			<button class="boton oscuro">Entrar</button>
		</form>
	{/if}
{/snippet}

{#if data.socio}
	<div class="socio">
		<header class="marca" style:--marca={color} style:color={textoSobre(color)}>
			<div class="ancho">
				{#if n}
					<span class="negocio">
						{#if n.logo}
							<img class="cuadro" src={n.logo} alt="" width="36" height="36" />
						{:else}
							<span class="cuadro" style:color={color} aria-hidden="true">{iniciales(n.nombre, 1)}</span>
						{/if}
						<b>{n.nombre}</b>
					</span>
				{/if}
				<h1>Ingresá para usar tu&nbsp;membresía</h1>
			</div>
		</header>
		<main class="ancho cuerpo">
			<p class="mail">
				<Icono nombre="mail" lleno tam={20} />
				<span>Usá <b>el mismo mail que le diste {n ? `a ${n.nombre}` : 'al negocio'}</b> cuando te hiciste socio. Así sabemos que la membresía es&nbsp;tuya.</span>
			</p>
			{#if n?.turno}
				<p class="guardado">
					<span>Tu turno queda guardado mientras ingresás:</span>
					<b class="tl-num">{n.turno}</b>
				</p>
			{/if}
			{@render error_()}
			<div class="acciones">
				{#if paso === 'mail'}{@render proveedores(true)}{/if}
				{@render conMail(true)}
				{#if data.volver}
					<p class="pie">¿No sos socio? <a href={data.volver}>Seguí sin ingresar</a></p>
				{/if}
			</div>
			{@render prueba()}
		</main>
	</div>
{:else}
	<div class="panel tl-petroleo">
		<img class="marca-agua" src="/marca/simbolo_una_tinta_blanco.svg" alt="" aria-hidden="true" />
		<main class="ancho cuerpo">
			<img class="icono-app" src="/marca-icono.png" alt="" width="56" height="56" />
			<h1>Entrá a tu panel</h1>
			<p class="bajada">Con Google, Apple o tu mail, la misma cuenta de&nbsp;siempre.</p>
			{@render error_()}
			<div class="acciones">
				{#if paso === 'mail'}{@render proveedores(false)}{/if}
				{@render conMail(false)}
				<p class="pie">¿Todavía no tenés cuenta? <a href="/empezar">Probá 7 días gratis</a></p>
			</div>
			{@render prueba()}
		</main>
	</div>
{/if}

<form method="POST" action="/ingresar/clerk" bind:this={formToken} hidden>
	<input type="hidden" name="token" value={tokenMail} />
</form>

<style>
	.ancho {
		width: min(100%, 420px);
		margin: 0 auto;
	}
	.cuerpo {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	h1 {
		margin: 0;
		font-weight: 700;
	}

	/* Equipo: petróleo con grano, el símbolo de fondo y los botones al alcance del pulgar. */
	.panel {
		position: relative;
		overflow: hidden;
		min-height: 100dvh;
		display: flex;
		flex-direction: column;
		padding: 84px 24px 28px;
	}
	.panel .cuerpo {
		position: relative;
		flex: 1;
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
	.icono-app {
		border-radius: 15px;
	}
	.panel h1 {
		margin-top: 8px;
		font-size: 36px;
		line-height: 1;
		letter-spacing: -0.05em;
	}
	.bajada {
		margin: 0;
		font-size: 15px;
		color: rgba(255, 255, 255, 0.75);
	}
	.acciones {
		margin-top: auto;
		display: flex;
		flex-direction: column;
		gap: 16px;
		padding-top: 24px;
	}
	.pie {
		margin: 0;
		font-size: 13px;
		text-align: center;
	}
	.panel .pie {
		color: rgba(255, 255, 255, 0.7);
	}
	.panel .pie a {
		color: var(--tl-menta);
		font-weight: 700;
		text-decoration: none;
	}
	.panel .pie a:hover {
		text-decoration: underline;
	}

	.proveedor {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 10px;
		height: 56px;
		width: 100%;
		border: 0;
		border-radius: 999px;
		font: 700 16px var(--tl-font);
		cursor: pointer;
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.proveedor:active {
		transform: scale(0.97);
	}
	.proveedor:disabled {
		cursor: progress;
		opacity: 0.7;
	}
	.google {
		background: #fff;
		color: var(--tl-tinta);
	}
	.google.claro {
		box-shadow: inset 0 0 0 1.5px #e0d5bf;
	}
	.apple {
		background: #000;
		color: #fff;
		box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.2);
	}
	.apple img {
		filter: invert(1);
	}
	.sin-clerk {
		margin: 0;
		font-size: 13px;
		opacity: 0.8;
	}
	.error-ingreso {
		display: flex;
		gap: 10px;
		margin: 0;
		padding: 14px;
		border-radius: 16px;
		background: #fbe0dc;
		color: #5b1a13;
		font-size: 14px;
		line-height: 1.45;
	}

	/* Con mail: sobre petróleo, campos papel; en la vista del socio (crema), con borde. */
	.con-mail {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.separador {
		display: flex;
		align-items: center;
		gap: 12px;
		font-size: 13px;
		color: rgba(255, 255, 255, 0.7);
	}
	.separador::before,
	.separador::after {
		content: '';
		flex: 1;
		height: 1px;
		background: rgba(255, 255, 255, 0.2);
	}
	.claro .separador {
		color: var(--tl-sub);
	}
	.claro .separador::before,
	.claro .separador::after {
		background: var(--tl-linea-2);
	}
	.entrada {
		height: 56px;
		padding: 0 20px;
		border: 0;
		border-radius: 999px;
		background: var(--tl-papel);
		color: var(--tl-tinta);
		font: 400 16px var(--tl-font);
	}
	.claro .entrada {
		box-shadow: inset 0 0 0 1.5px #e0d5bf;
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
	.seguir {
		height: 56px;
		border: 0;
		border-radius: 999px;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		font: 900 16px var(--tl-font);
		cursor: pointer;
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.4), 0 12px 24px -14px rgba(60, 204, 144, 0.9);
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.seguir:active {
		transform: scale(0.97);
	}
	.seguir:disabled {
		cursor: progress;
		opacity: 0.7;
	}
	.volver {
		align-self: flex-start;
		display: inline-flex;
		align-items: center;
		gap: 6px;
		min-height: 44px;
		max-width: 100%;
		padding: 0;
		border: 0;
		background: none;
		color: inherit;
		font: 700 14px var(--tl-font);
		cursor: pointer;
		overflow-wrap: anywhere;
		text-align: left;
	}
	.etiqueta {
		font-size: 14px;
		font-weight: 700;
	}
	.explica {
		margin: 0;
		font-size: 15px;
		line-height: 1.45;
		overflow-wrap: anywhere;
	}
	.panel .explica {
		color: rgba(255, 255, 255, 0.85);
	}
	.olvide {
		align-self: center;
		min-height: 44px;
		display: grid;
		place-items: center;
		font-size: 14px;
		font-weight: 700;
		color: var(--tl-menta);
		text-decoration: none;
	}
	.claro .olvide {
		color: var(--tl-menta-texto);
	}
	.solo-lector {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip-path: inset(50%);
		white-space: nowrap;
	}

	/* Socio: con los colores del negocio, avisado de qué mail usar. */
	.socio {
		min-height: 100dvh;
		display: flex;
		flex-direction: column;
		background: var(--tl-crema-pagina);
	}
	.marca {
		padding: 68px 24px 24px;
		background-color: var(--marca);
		background-image: var(--tl-grano);
	}
	.marca .ancho {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.negocio {
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.cuadro {
		width: 36px;
		height: 36px;
		border-radius: 10px;
		background: #fff;
		display: grid;
		place-items: center;
		font-weight: 700;
		object-fit: cover;
		flex: none;
	}
	.marca h1 {
		margin-top: 6px;
		font-size: 30px;
		line-height: 1.02;
		letter-spacing: -0.045em;
	}
	.socio .cuerpo {
		flex: 1;
		width: min(100%, 468px);
		padding: 22px 24px 28px;
		gap: 14px;
	}
	.mail {
		display: flex;
		gap: 10px;
		margin: 0;
		padding: 14px;
		border-radius: 16px;
		background: #f3e8d0;
		font-size: 14px;
		line-height: 1.45;
		color: #4a3a17;
	}
	.mail :global(.icono) {
		color: #7a5a1e;
	}
	.guardado {
		display: flex;
		flex-direction: column;
		gap: 4px;
		margin: 0;
		font-size: 13px;
		color: var(--tl-sub);
	}
	.guardado b {
		font-size: 14px;
		color: var(--tl-tinta);
	}
	.socio .acciones {
		gap: 14px;
	}
	.socio .pie {
		color: var(--tl-sub);
	}
	.socio .pie a {
		color: var(--tl-tinta);
		font-weight: 700;
	}

	/* Ingreso de prueba (sólo desarrollo): una hoja papel aparte. */
	.prueba {
		display: flex;
		flex-direction: column;
		gap: 12px;
		margin-top: 8px;
		padding: 20px;
		border-radius: var(--tl-r-l);
		background: var(--tl-papel);
		color: var(--tl-tinta);
		box-shadow: var(--tl-sh-tarjeta);
	}
	.prueba p {
		margin: 0;
		font-size: 13px;
		color: var(--tl-sub);
	}
	.prueba .tl-rotulo {
		font-size: 11px;
		color: var(--tl-menta-texto);
	}

	@media (min-width: 720px) {
		.panel {
			justify-content: center;
			padding-top: 48px;
		}
		.panel .cuerpo {
			flex: none;
		}
		.socio .cuerpo {
			flex: none;
		}
		.marca {
			padding-top: 48px;
		}
	}
</style>
