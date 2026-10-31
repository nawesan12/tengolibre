<!--
	El alta sin sesión («Landing y alta v3», vista alta): nombre, rubro y dirección validada al instante.
	La cuenta se crea con Google o Apple (Clerk): no hay alta por mail, así que el campo del mail del
	diseño se reemplazó por los dos botones. Lo cargado vuelve después del ingreso al paso del nombre.
-->
<script lang="ts">
	import { untrack } from 'svelte';
	import { enhance } from '$app/forms';
	import { goto } from '$app/navigation';
	import type { SubmitFunction } from '$app/forms';
	import Icono from '#lib/ui/Icono.svelte';
	import { iniciar, ErrorClerk, type ProveedorClerk } from '#lib/clerk.ts';
	import { Direccion } from './direccion.svelte.ts';
	import { aSlug } from './slug.ts';
	import { RUBROS, rubroDe } from './rubros.ts';

	let { borrador, fapi, origen }: { borrador: { nombre: string; slug: string; rubro: string }; fapi: string | null; origen: string } = $props();

	// Lo que vino en la URL es sólo el valor inicial: después manda lo que se escribe.
	const inicial = untrack(() => ({ ...borrador }));
	let nombre = $state(inicial.nombre);
	let rubro = $state(inicial.rubro || 'estetica');
	let escrito = $state(inicial.slug);
	let tocado = $state(!!inicial.slug);
	const slug = $derived(aSlug(tocado ? escrito : nombre));
	const dir = new Direccion();
	$effect(() => {
		dir.rubro = rubro;
		dir.poner(slug);
	});
	const campo = $derived(dir.campo);
	const listo = $derived(!!nombre.trim() && campo.ok);
	const color = $derived(rubroDe(rubro)?.color ?? '#183c44');
	const rubroNombre = $derived(rubroDe(rubro)?.nombre ?? 'Tu rubro');
	const host = $derived(new URL(origen).host);
	const sugerencias = $derived(campo.estado === 'tomada' ? (dir.resultado?.sugerencias ?? []) : []);

	let yendo = $state<ProveedorClerk | null>(null);
	let error = $state('');
	const seguir: SubmitFunction = ({ submitter }) => {
		const p = (submitter?.getAttribute('value') ?? 'google') as ProveedorClerk;
		yendo = p;
		error = '';
		return async ({ result }) => {
			if (result.type !== 'success') {
				error = 'Revisá los datos.';
				yendo = null;
				return;
			}
			const volver = String(result.data?.volver ?? '/empezar');
			if (!fapi) {
				await goto(`/ingresar?volver=${encodeURIComponent(volver)}`);
				return;
			}
			try {
				location.href = await iniciar(fapi, p, `${location.origin}/ingresar/clerk`);
			} catch (e) {
				error = e instanceof ErrorClerk && e.codigo === 'cancelado' ? 'Cancelaste el ingreso.' : 'El ingreso no responde. Probá en un rato.';
				yendo = null;
			}
		};
	};
</script>

<div class="alta">
	<aside class="lado tl-petroleo">
		<img class="simbolo" src="/marca/simbolo_una_tinta_blanco.svg" alt="" aria-hidden="true" />
		<span class="arriba">
			<a href="/" class="volver" aria-label="Volver"><Icono nombre="arrow_back" tam={24} /></a>
			<img src="/marca/logotipo_horizontal_negativo.svg" alt="Tengo Libre" height="24" />
		</span>
		<div class="vitrina">
			<h1>Tu página, lista antes de que se enfríe el&nbsp;mate.</h1>
			<div class="tarjeta-neg" aria-hidden="true">
				<span class="proximo" style:background={color}><span>PRÓXIMO LIBRE</span><b>Hoy 15:30</b></span>
				<div class="hoja-neg">
					<b>{nombre.trim() || 'Tu negocio'}</b>
					<span class="sub">{rubroNombre} · Reservá en un minuto</span>
					<span class="url"><Icono nombre="link" tam={18} />{host}/<b>{slug || 'tu-negocio'}</b></span>
				</div>
			</div>
		</div>
		<span class="ventajas">
			<span><Icono nombre="check_circle" lleno tam={18} />7 días con todo</span>
			<span><Icono nombre="check_circle" lleno tam={18} />Sin tarjeta</span>
			<span><Icono nombre="check_circle" lleno tam={18} />Cancelás cuando quieras</span>
		</span>
	</aside>

	<main class="form-lado">
		<form method="POST" action="?/borrador" use:enhance={seguir} class="form">
			<span class="rotulo">CREAR TU NEGOCIO · 1 MINUTO</span>
			<h2>Empecemos por lo&nbsp;básico.</h2>

			<label class="campo-a">
				Nombre del negocio
				<input name="nombre" bind:value={nombre} placeholder="Lumi Estética" autocomplete="organization" maxlength="80" required />
			</label>

			<div class="grupo">
				<span class="etq" id="que-haces">¿Qué hacés?</span>
				<input type="hidden" name="rubro" value={rubro} />
				<div class="rubros" role="group" aria-labelledby="que-haces">
					{#each RUBROS as r (r.clave)}
						<button type="button" class="rubro" aria-pressed={rubro === r.clave} onclick={() => (rubro = r.clave)}>
							<Icono nombre={r.icono} lleno tam={18} />{r.nombre}
						</button>
					{/each}
				</div>
			</div>

			<div class="grupo">
				<span class="etq">Tu dirección web</span>
				<input type="hidden" name="slug" value={slug} />
				<label class="dir" style:--borde={campo.borde}>
					<span class="pre">{host}/</span>
					<input
						value={slug}
						oninput={(e) => {
							tocado = true;
							escrito = e.currentTarget.value;
						}}
						aria-label="Tu dirección web"
						autocapitalize="none"
						spellcheck="false"
						autocomplete="off"
					/>
					<span style:color={campo.borde}><Icono nombre={campo.icono} lleno tam={22} /></span>
				</label>
				<span class="msg" style:color={campo.color} role="status">{campo.mensaje}</span>
				{#if sugerencias.length}
					<div class="alts">
						{#each sugerencias as s (s)}
							<button
								type="button"
								class="alt"
								onclick={() => {
									tocado = true;
									escrito = s;
								}}>{s}</button
							>
						{/each}
					</div>
				{/if}
			</div>

			<div class="cuenta">
				<button class="seguir" name="proveedor" value="google" disabled={!listo || !!yendo}>
					<span class="logo-b"><img src="/logos/google.svg" alt="" width="20" height="20" /></span>Crear con Google<Icono nombre="arrow_forward" tam={22} />
				</button>
				<button class="otra" name="proveedor" value="apple" disabled={!listo || !!yendo}>
					<img src="/logos/apple.svg" alt="" width="20" height="20" />Crear con Apple
				</button>
				{#if error}<p class="error" role="alert"><Icono nombre="error" lleno tam={16} />{error}</p>{/if}
				<span class="legal">Entrás con tu cuenta de Google o de Apple: no hay contraseña que recordar. Después te guiamos con servicios, horarios y&nbsp;cobros.</span>
				<span class="ya">¿Ya tenés cuenta? <a href="/ingresar?volver=/panel">Ingresá</a></span>
			</div>
		</form>
	</main>
</div>

<style>
	.alta {
		min-height: 100vh;
		display: grid;
		grid-template-columns: minmax(0, 1fr) minmax(0, 1.1fr);
	}
	.lado {
		position: relative;
		overflow: hidden;
		padding: clamp(20px, 4vw, 48px);
		display: flex;
		flex-direction: column;
		gap: 28px;
	}
	.simbolo {
		position: absolute;
		right: -120px;
		bottom: -140px;
		width: 520px;
		opacity: 0.04;
		pointer-events: none;
	}
	.arriba {
		display: flex;
		align-items: center;
		gap: 12px;
	}
	.arriba img {
		height: 24px;
		width: auto;
	}
	.volver {
		width: 48px;
		height: 48px;
		border-radius: 14px;
		background: rgba(255, 255, 255, 0.08);
		color: #fff;
		display: grid;
		place-items: center;
	}
	.vitrina {
		margin-top: auto;
		display: flex;
		flex-direction: column;
		gap: 24px;
		max-width: 460px;
	}
	h1 {
		margin: 0;
		font-size: clamp(40px, 4.6vw, 64px);
		line-height: 0.94;
		font-weight: 900;
		letter-spacing: -0.045em;
		text-wrap: balance;
	}
	.tarjeta-neg {
		position: relative;
		padding-top: 44px;
	}
	.proximo {
		position: absolute;
		left: 50%;
		top: 0;
		z-index: 2;
		min-width: 180px;
		height: 88px;
		transform: translateX(-50%);
		padding: 0 18px;
		border-radius: 22px;
		color: #fff;
		box-shadow: 0 0 0 10px #183c44;
		display: flex;
		flex-direction: column;
		justify-content: center;
		transition: background var(--tl-entrar);
	}
	.proximo span {
		font-size: 11px;
		font-weight: 900;
		letter-spacing: 0.08em;
		opacity: 0.85;
	}
	.proximo b {
		font-size: 30px;
		font-weight: 900;
		font-variant-numeric: tabular-nums;
	}
	.hoja-neg {
		border-radius: 32px;
		background: #fffdf9;
		color: var(--tl-tinta);
		padding: 64px 22px 22px;
		display: flex;
		flex-direction: column;
		gap: 12px;
		box-shadow: 0 40px 70px -30px rgba(0, 0, 0, 0.6);
		text-align: center;
	}
	.hoja-neg > b {
		font-size: 28px;
		line-height: 1;
		font-weight: 900;
		letter-spacing: -0.03em;
		overflow-wrap: anywhere;
	}
	.sub {
		font-size: 14px;
		color: var(--tl-sub);
	}
	.url {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 6px;
		min-height: 44px;
		padding: 0 12px;
		border-radius: 999px;
		background: #f4eee2;
		font-size: 15px;
		font-variant-numeric: tabular-nums;
		overflow-wrap: anywhere;
	}
	.url :global(.icono) {
		color: #17805a;
	}
	.ventajas {
		display: flex;
		flex-wrap: wrap;
		gap: 8px 20px;
		font-size: 14px;
		color: rgba(255, 255, 255, 0.8);
	}
	.ventajas span {
		display: inline-flex;
		align-items: center;
		gap: 6px;
	}
	.ventajas :global(.icono) {
		color: #3ccc90;
	}
	.form-lado {
		background-color: #f4eee2;
		background-image: radial-gradient(rgba(24, 60, 68, 0.07) 1px, transparent 1.3px);
		background-size: 18px 18px;
		padding: clamp(24px, 5vw, 64px) clamp(20px, 4vw, 48px);
		display: flex;
		justify-content: center;
	}
	.form {
		width: 100%;
		max-width: 520px;
		display: flex;
		flex-direction: column;
		gap: 22px;
	}
	.rotulo {
		font-size: 13px;
		font-weight: 900;
		letter-spacing: 0.08em;
		color: #17805a;
	}
	h2 {
		margin: 0;
		font-size: clamp(32px, 4vw, 44px);
		line-height: 1;
		font-weight: 900;
		letter-spacing: -0.04em;
		text-wrap: balance;
	}
	.campo-a {
		display: flex;
		flex-direction: column;
		gap: 8px;
		font-size: 15px;
		font-weight: 700;
	}
	.campo-a input {
		height: 56px;
		padding: 0 16px;
		border: 0;
		border-radius: 16px;
		background: #fffdf9;
		box-shadow: inset 0 0 0 1.5px #ddd1b9;
		font: 400 16px var(--tl-font);
		color: var(--tl-tinta);
	}
	.grupo {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.etq {
		font-size: 15px;
		font-weight: 700;
	}
	.rubros {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}
	.rubro {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		height: 48px;
		padding: 0 16px;
		border: 0;
		border-radius: 999px;
		background: #fffdf9;
		color: var(--tl-tinta);
		box-shadow: inset 0 0 0 1.5px #ddd1b9;
		font: 700 15px var(--tl-font);
		cursor: pointer;
		transition: background var(--tl-tocar);
	}
	.rubro[aria-pressed='true'] {
		background: #183c44;
		color: #fff;
		box-shadow: none;
	}
	.dir {
		display: flex;
		align-items: center;
		height: 56px;
		padding: 0 14px 0 16px;
		border-radius: 16px;
		background: #fffdf9;
		box-shadow: inset 0 0 0 2px var(--borde);
		font-size: 16px;
	}
	.pre {
		color: var(--tl-sub);
		white-space: nowrap;
	}
	.dir input {
		flex: 1;
		min-width: 0;
		height: 50px;
		border: 0;
		background: transparent;
		font: 900 16px var(--tl-font);
		color: var(--tl-tinta);
		outline: none;
	}
	.msg {
		font-size: 14px;
		font-weight: 700;
	}
	.alts {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}
	.alt {
		height: 44px;
		padding: 0 14px;
		border: 0;
		border-radius: 999px;
		background: repeating-linear-gradient(-45deg, rgba(60, 204, 144, 0.22) 0 2px, transparent 2px 9px), #f3fbf7;
		box-shadow: inset 0 0 0 1.5px rgba(23, 128, 90, 0.4);
		font: 700 14px var(--tl-font);
		color: #17805a;
		cursor: pointer;
	}
	.cuenta {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.seguir,
	.otra {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 10px;
		border: 0;
		border-radius: 999px;
		font-family: var(--tl-font);
		cursor: pointer;
		transition:
			background var(--tl-tocar),
			transform var(--tl-tocar);
	}
	.seguir {
		height: 60px;
		background: #3ccc90;
		color: #102b31;
		font-size: 18px;
		font-weight: 900;
	}
	.logo-b {
		width: 32px;
		height: 32px;
		border-radius: 50%;
		background: #fff;
		display: grid;
		place-items: center;
	}
	.otra {
		height: 56px;
		background: #fffdf9;
		box-shadow: inset 0 0 0 1.5px #ddd1b9;
		color: var(--tl-tinta);
		font-size: 16px;
		font-weight: 700;
	}
	.seguir:not(:disabled):hover,
	.otra:not(:disabled):hover {
		transform: translateY(-2px);
	}
	.seguir:disabled {
		background: #e6dcc8;
		color: #5b6a6d;
	}
	.otra:disabled {
		color: #5b6a6d;
		cursor: not-allowed;
	}
	.seguir:disabled {
		cursor: not-allowed;
	}
	.error {
		display: flex;
		align-items: center;
		gap: 6px;
		margin: 0;
		font-size: 14px;
		font-weight: 700;
		color: #a3241a;
	}
	.legal {
		font-size: 13px;
		line-height: 1.5;
		color: var(--tl-sub);
		text-wrap: pretty;
	}
	.ya {
		font-size: 14px;
		color: var(--tl-sub-2);
	}
	.ya a {
		color: #17805a;
		font-weight: 700;
	}
	@media (max-width: 1039px) {
		.alta {
			grid-template-columns: minmax(0, 1fr);
		}
		.vitrina,
		.ventajas {
			display: none;
		}
		.lado {
			padding: 16px;
		}
		.simbolo {
			display: none;
		}
	}
</style>
