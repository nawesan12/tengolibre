<script lang="ts">
	import Icono from '#lib/ui/Icono.svelte';
	import Logo from '#lib/ui/Logo.svelte';

	let { data, form } = $props();

	/** Sin palabra sola en el último renglón: la última separación pasa a ser un espacio duro. */
	const sinViuda = (t: string) => t.replace(/\s+(\S+)$/, ' $1');

	const errores: Record<string, { titulo: string; texto: string }> = {
		vencida: {
			titulo: 'Este pedido ya no sirve',
			texto: 'Venció o ya se usó. Volvé a tu asistente y empezá la conexión otra vez.'
		},
		cliente_desconocido: {
			titulo: 'No conocemos esa app',
			texto: 'La app que te mandó acá no está registrada. Volvé a ella y probá conectar de nuevo.'
		},
		redirect_invalido: {
			titulo: 'La dirección de vuelta no cierra',
			texto: 'Por seguridad no seguimos: la app pidió volver a un lugar que no tiene registrado.'
		}
	};
	const error = $derived(
		data.estado === 'error'
			? (errores[data.codigo] ?? { titulo: 'No pudimos seguir', texto: 'Algo falló con el pedido. Volvé a tu asistente y probá de nuevo.' })
			: null
	);
	const s = $derived(data.estado === 'pedido' ? data.s : null);
	const ROLES: Record<string, string> = { duenio: 'Dueño', encargado: 'Encargado' };
	let enviando = $state(false);
</script>

<svelte:head>
	<title>Conectar un asistente · Tengo Libre</title>
	<meta name="robots" content="noindex, nofollow" />
	<meta name="referrer" content="no-referrer" />
</svelte:head>

<main class="conectar">
	<div class="marca"><Logo alto={24} /></div>

	{#if error}
		<section class="estado">
			<Icono nombre="link_off" tam={32} />
			<h1>{error.titulo}</h1>
			<p>{error.texto}</p>
			<a class="boton oscuro" href="/panel">Ir a mi panel</a>
		</section>
	{:else if s}
		<span class="tl-rotulo pide">Pide acceso</span>
		<form method="POST" action="?/permitir" class="pila-form" onsubmit={() => (enviando = true)}>
			<input type="hidden" name="solicitud" value={data.solicitud} />
			<article class="pase">
				<header class="tl-petroleo cabeza">
					<span class="app">
						<span class="avatar"><Icono nombre="smart_toy" lleno tam={22} /></span>
						<b>{s.cliente}</b>
					</span>
					<span class="quiere">quiere conectarse a Tengo Libre para ayudarte&nbsp;con</span>
					<b class="que">tu agenda</b>
				</header>
				<div class="troquel" aria-hidden="true"><span></span></div>

				<div class="cuerpo">
					{#if s.negocios.length === 0}
						<p class="aviso error" role="alert">
							<Icono nombre="block" />
							<span>Para conectar un asistente tenés que ser dueño o encargado de un&nbsp;negocio.</span>
						</p>
					{:else}
						<fieldset>
							<legend class="tl-rotulo rotulo">Negocio</legend>
							{#each s.negocios as n, i (n.id)}
								<label class="negocio">
									<input type="radio" name="negocio" value={n.id} checked={i === 0} required />
									<span class="inicial" aria-hidden="true">{n.nombre.slice(0, 1).toUpperCase()}</span>
									<span class="datos"><b>{n.nombre}</b><span>{ROLES[n.rol] ?? n.rol}</span></span>
									<span class="marcado"><Icono nombre="check_circle" lleno tam={22} /></span>
								</label>
							{/each}
						</fieldset>

						<fieldset>
							<legend class="tl-rotulo rotulo">Qué va a poder hacer</legend>
							<div class="permisos">
								{#each s.permisos as p (p.clave)}
									<label class="permiso">
										<span class="texto"><b>{p.nombre}</b><span>{sinViuda(p.descripcion)}</span></span>
										<input
											type="checkbox"
											role="switch"
											name="permiso"
											value={p.clave}
											checked={s.porDefecto.includes(p.clave)}
										/>
										<span class="llave" aria-hidden="true">
											<span class="perilla"><span class="si icono">check</span><span class="no icono">close</span></span>
										</span>
									</label>
								{/each}
							</div>
						</fieldset>
						<p class="letra">
							<Icono nombre="undo" tam={18} />
							<span>Vuelve a <b>{s.destino}</b>. Lo desconectás cuando quieras en Integraciones e&nbsp;IA.</span>
						</p>
					{/if}
				</div>
			</article>

			{#if form?.error}
				<p class="aviso error" role="alert"><Icono nombre="error" /><span>{form.error}</span></p>
			{/if}

			{#if s.negocios.length > 0}
				<button class="tl-cta permitir" disabled={enviando}>
					<Icono nombre="check" />
					Permitir
				</button>
			{/if}
		</form>
		<form method="POST" action="?/cancelar">
			<input type="hidden" name="solicitud" value={data.solicitud} />
			<button class="cancelar">Cancelar</button>
		</form>
	{/if}
</main>

<style>
	.conectar {
		min-height: 100dvh;
		width: min(440px, 100%);
		margin: 0 auto;
		padding: 24px 16px 32px;
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	:global(body:has(.conectar)) {
		background: var(--tl-crema);
	}
	.marca {
		padding: 4px 0 8px;
	}
	.pide {
		color: var(--tl-menta-texto);
		font-size: 12px;
	}
	.pila-form {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	.pase {
		border-radius: 22px;
		overflow: hidden;
		background: var(--tl-papel);
		box-shadow: 0 24px 40px -24px rgba(15, 35, 40, 0.5);
		animation: tl-gin var(--tl-entrar) var(--tl-ease) both;
	}
	.cabeza {
		padding: 18px 20px;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.app {
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.app b {
		font-size: 18px;
		overflow-wrap: anywhere;
	}
	.avatar {
		width: 40px;
		height: 40px;
		border-radius: 11px;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		display: grid;
		place-items: center;
		flex: none;
	}
	.quiere {
		font-size: 14px;
		color: rgba(255, 255, 255, 0.8);
	}
	.que {
		font-size: 28px;
		line-height: 1.05;
		letter-spacing: -0.04em;
		color: var(--tl-menta);
	}
	.troquel {
		height: 18px;
		position: relative;
		background: linear-gradient(var(--tl-petroleo) 0 50%, var(--tl-papel) 50%);
	}
	.troquel::before,
	.troquel::after {
		content: '';
		position: absolute;
		top: 0;
		width: 18px;
		height: 18px;
		border-radius: 50%;
		background: var(--tl-crema);
	}
	.troquel::before {
		left: -9px;
	}
	.troquel::after {
		right: -9px;
	}
	.troquel span {
		position: absolute;
		left: 16px;
		right: 16px;
		top: 8px;
		border-top: 2px dashed var(--tl-linea-2);
	}
	.cuerpo {
		padding: 6px 20px 18px;
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	fieldset {
		border: 0;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
		min-width: 0;
	}
	.rotulo {
		font-size: 11px;
		color: var(--tl-sub);
		padding: 0 0 8px;
	}
	.negocio {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr) auto;
		align-items: center;
		gap: 12px;
		min-height: 60px;
		padding: 8px 14px;
		border-radius: var(--tl-r-m);
		box-shadow: inset 0 0 0 1.5px var(--tl-linea-2);
		cursor: pointer;
		position: relative;
		transition: box-shadow var(--tl-tocar) var(--tl-ease);
	}
	.negocio input,
	.permiso input {
		position: absolute;
		opacity: 0;
		width: 1px;
		height: 1px;
	}
	.negocio:has(input:checked) {
		box-shadow: inset 0 0 0 2px var(--tl-petroleo);
	}
	.negocio:has(input:focus-visible),
	.permiso:has(input:focus-visible) {
		outline: 3px solid var(--tl-menta);
		outline-offset: 2px;
	}
	.inicial {
		width: 36px;
		height: 36px;
		border-radius: 10px;
		background: var(--tl-petroleo);
		color: #fff;
		display: grid;
		place-items: center;
		font-weight: 900;
	}
	.datos {
		display: flex;
		flex-direction: column;
		min-width: 0;
	}
	.datos b {
		font-size: 15px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.datos span {
		font-size: 13px;
		color: var(--tl-sub);
	}
	.marcado {
		color: var(--tl-menta-texto);
		visibility: hidden;
	}
	.negocio:has(input:checked) .marcado {
		visibility: visible;
	}
	.permisos {
		border-radius: var(--tl-r-m);
		box-shadow: inset 0 0 0 1px var(--tl-linea);
		overflow: hidden;
	}
	.permiso {
		position: relative;
		display: grid;
		grid-template-columns: minmax(0, 1fr) auto;
		gap: 12px;
		align-items: center;
		min-height: 68px;
		padding: 10px 16px;
		border-top: 1px solid var(--tl-linea);
		cursor: pointer;
	}
	.permiso:first-child {
		border-top: 0;
	}
	.texto {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}
	.texto b {
		font-size: 15px;
		font-weight: 700;
	}
	.texto span {
		font-size: 13px;
		line-height: 1.4;
		color: var(--tl-sub);
		text-wrap: pretty;
	}
	.llave {
		width: 48px;
		height: 28px;
		border-radius: 999px;
		background: #d8ccb4;
		position: relative;
		transition: background var(--tl-tocar) var(--tl-ease);
	}
	.perilla {
		position: absolute;
		top: 3px;
		left: 3px;
		width: 22px;
		height: 22px;
		border-radius: 50%;
		background: #fff;
		display: grid;
		place-items: center;
		color: var(--tl-sub);
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.perilla .icono {
		font-size: 14px;
		grid-area: 1 / 1;
	}
	.perilla .si {
		visibility: hidden;
		color: var(--tl-menta-texto);
	}
	.permiso:has(input:checked) .llave {
		background: var(--tl-menta);
	}
	.permiso:has(input:checked) .perilla {
		transform: translateX(20px);
	}
	.permiso:has(input:checked) .si {
		visibility: visible;
	}
	.permiso:has(input:checked) .no {
		visibility: hidden;
	}
	.letra {
		margin: 0;
		display: flex;
		gap: 8px;
		align-items: flex-start;
		font-size: 13px;
		line-height: 1.45;
		color: var(--tl-sub);
	}
	.letra b {
		color: var(--tl-tinta);
		overflow-wrap: anywhere;
	}
	.permitir {
		width: 100%;
		justify-content: center;
		margin-top: 8px;
	}
	.permitir:disabled {
		opacity: 0.6;
		cursor: progress;
	}
	.cancelar {
		width: 100%;
		height: 48px;
		border: 0;
		background: none;
		font: 700 15px var(--tl-font);
		color: var(--tl-tinta);
		cursor: pointer;
		border-radius: 999px;
	}
	.cancelar:hover {
		background: rgba(15, 35, 40, 0.06);
	}
	.estado {
		margin-top: 24px;
		padding: 22px;
		border-radius: var(--tl-r-l);
		background: var(--tl-papel);
		display: flex;
		flex-direction: column;
		gap: 10px;
		align-items: flex-start;
		color: var(--tl-petroleo);
	}
	.estado h1 {
		font-size: 22px;
		letter-spacing: -0.02em;
		margin: 0;
		color: var(--tl-tinta);
	}
	.estado p {
		margin: 0 0 6px;
		font-size: 14px;
		line-height: 1.5;
		color: var(--tl-sub-2);
	}
</style>
