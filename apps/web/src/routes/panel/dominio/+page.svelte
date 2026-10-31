<script lang="ts">
	import '#lib/gestion/gestion.css';
	import { enhance } from '$app/forms';
	import { fecha, hora } from '#lib/formato.ts';
	import { confirmar } from '#lib/configuracion/formularios.ts';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';
	import Copiar from '#lib/gestion/Copiar.svelte';

	let { data, form } = $props();
	const d = $derived(data.dominio);
	const esDuenio = $derived(data.negocio.rol === 'duenio');
	const listo = $derived(d?.estado === 'verificado');
	let verificando = $state(false);

	// «10:04» si fue hoy; si no, «vie 30 oct 10:04».
	const cuando = (iso: string) => {
		const hoy = new Date().toDateString() === new Date(iso).toDateString();
		return hoy ? hora(iso) : `${fecha(iso)} ${hora(iso)}`;
	};
</script>

<svelte:head><title>Dominio propio · Panel</title></svelte:head>

<div class="g-pagina dominio">
	<div class="g-col cabeza">
		<a class="g-kicker volver" href="/panel/mi-web"><span class="icono" aria-hidden="true">arrow_back</span>Mi web</a>
		<h1 class="g-titulo">Tu propio&nbsp;dominio</h1>
		<p class="g-bajada">
			Tu página puede verse en una dirección tuya, como <b>turnos.tunegocio.com</b>, además de la de Tengo&nbsp;Libre.
		</p>
	</div>

	<Mensaje {form} que="dominio" />

	<ol class="pasos">
		<!-- 1. Elegir el dominio -->
		<li class="g-paso" class:ok={!!d} class:ahora={!d}>
			<span class="g-paso-num">
				{#if d}<span class="icono" aria-hidden="true">check</span><span class="visualmente-oculto">Hecho:</span>{:else}1{/if}
			</span>
			<div class="g-paso-cuerpo">
				{#if d}
					<b>Elegiste {d.nombre}</b>
					<span>Puede ser un subdominio de tu&nbsp;web.</span>
				{:else}
					<b>Elegí tu dominio</b>
					<span>Puede ser un subdominio de tu web, como turnos.tunegocio.com.</span>
					{#if esDuenio}
						<form method="POST" action="?/poner" use:enhance class="poner">
							<label class="g-campo crece">
								Dominio
								<input name="dominio" required maxlength="253" placeholder="turnos.tunegocio.com" autocapitalize="none" spellcheck="false" />
							</label>
							<button class="g-boton oscuro">Conectar</button>
						</form>
					{:else}
						<span>Sólo la dueña o el dueño del negocio puede conectar un&nbsp;dominio.</span>
					{/if}
				{/if}
			</div>
		</li>

		<!-- 2. Apuntar el DNS -->
		<li class="g-paso" class:ok={listo} class:ahora={d && !listo}>
			<span class="g-paso-num">
				{#if listo}<span class="icono" aria-hidden="true">check</span><span class="visualmente-oculto">Hecho:</span>
				{:else if d}<span class="icono" aria-hidden="true">sync</span><span class="visualmente-oculto">En curso:</span>
				{:else}2{/if}
			</span>
			<div class="g-paso-cuerpo">
				<b>Apuntá el DNS a Tengo&nbsp;Libre</b>
				{#if listo}
					<span>Ya responde: tu dominio apunta a tu&nbsp;página.</span>
				{:else}
					<span>En tu proveedor de dominio (NIC, GoDaddy, Donweb), creá {d && d.registros.length > 1 ? 'estos registros' : 'este registro'}. Los cambios pueden tardar desde minutos hasta un&nbsp;día.</span>
					{#if d}
						<div class="registros g-desplazable">
							<table class="g-tabla">
								<thead><tr><th>Tipo</th><th>Nombre</th><th>Valor</th><th><span class="visualmente-oculto">Copiar</span></th></tr></thead>
								<tbody>
									{#each d.registros as r (r.tipo + r.nombre + r.valor)}
										<tr>
											<td><b>{r.tipo}</b>{#if r.alternativa}<span class="alt"> o en vez del CNAME</span>{/if}</td>
											<td><b>{r.nombre}</b></td>
											<td class="celda-valor"><b class="valor">{r.valor}</b></td>
											<td class="copiar"><Copiar texto={r.valor} /></td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					{/if}
				{/if}
			</div>
		</li>

		<!-- 3. Certificado -->
		<li class="g-paso" class:ok={listo}>
			<span class="g-paso-num">
				{#if listo}<span class="icono" aria-hidden="true">check</span><span class="visualmente-oculto">Hecho:</span>{:else}<span class="icono" aria-hidden="true">lock</span>{/if}
			</span>
			<div class="g-paso-cuerpo">
				<b>Certificado de seguridad</b>
				{#if listo}
					<span>Tu página se ve con candado en <a class="g-accion" href={`https://${d?.nombre}`} target="_blank" rel="noopener">{d?.nombre}</a>.</span>
				{:else}
					<span>Se genera solo cuando el DNS responda. Tarda unos&nbsp;minutos.</span>
				{/if}
			</div>
		</li>
	</ol>

	{#if d}
		{#if d.estado === 'error' && d.detalle}
			<p class="g-aviso rojo"><span class="icono" aria-hidden="true">error</span><span>{d.detalle}</span></p>
		{/if}
		<div class="g-aviso" class:ambar={!listo}>
			<span class="icono" aria-hidden="true">{listo ? 'check_circle' : 'schedule'}</span>
			<span>
				{#if listo}
					Conectado{d.verificadoEn ? ` desde ${fecha(d.verificadoEn)}` : ''}.
				{:else if d.chequeadoEn}
					Último intento {cuando(d.chequeadoEn)}: el DNS todavía no apunta. Puede tardar hasta&nbsp;24&nbsp;h.
				{:else}
					Cuando cargues {d.registros.length > 1 ? 'los registros' : 'el registro'}, tocá «Probar ahora». Puede tardar hasta&nbsp;24&nbsp;h.
				{/if}
			</span>
			<form
				method="POST"
				action="?/verificar"
				use:enhance={() => {
					verificando = true;
					return async ({ update }) => {
						await update();
						verificando = false;
					};
				}}
			>
				<button class="g-boton" disabled={verificando}>{verificando ? 'Probando…' : listo ? 'Volver a probar' : 'Probar ahora'}</button>
			</form>
		</div>
		{#if esDuenio}
			<form method="POST" action="?/quitar" use:enhance={confirmar(`¿Desconectar ${d.nombre}? Tu página deja de verse ahí.`)}>
				<button class="g-accion peligro">Desconectar {d.nombre}</button>
			</form>
		{/if}
	{/if}
</div>

<style>
	.dominio {
		max-width: 720px;
	}
	.cabeza {
		gap: 8px;
	}
	.volver {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		min-height: 32px;
		text-decoration: none;
		width: fit-content;
	}
	.volver .icono {
		font-size: 16px;
	}
	.pasos {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.g-paso-num .icono {
		font-size: 18px;
	}
	.poner {
		display: flex;
		flex-wrap: wrap;
		align-items: flex-end;
		gap: 8px;
		margin-top: 8px;
	}
	.poner .g-boton {
		min-height: 46px;
	}
	.crece {
		flex: 1;
		min-width: min(240px, 100%);
	}
	.registros {
		margin-top: 4px;
		border-radius: 12px;
		background: var(--tl-crema);
	}
	.registros :global(.g-tabla td) {
		border-top-color: #e6dcc8;
		font-size: 13px;
	}
	.registros :global(.g-tabla th),
	.registros :global(.g-tabla td) {
		padding: 8px 12px;
	}
	.valor {
		overflow-wrap: anywhere;
	}
	.alt {
		display: block;
		font-size: 11px;
		color: var(--tl-sub);
	}
	.copiar {
		text-align: right;
	}
	.g-aviso {
		align-items: center;
		flex-wrap: wrap;
	}
	.g-aviso > span:not(.icono) {
		flex: 1 1 200px;
	}
	.g-aviso > form {
		flex: none;
		margin-left: auto;
	}
	.registros :global(.g-tabla) {
		table-layout: auto;
	}
	.registros b {
		overflow-wrap: anywhere;
	}
	/* En el teléfono cada registro es una tarjetita: tipo y nombre arriba, el valor entero abajo. */
	@media (max-width: 640px) {
		.registros :global(thead) {
			position: absolute;
			width: 1px;
			height: 1px;
			overflow: hidden;
			clip: rect(0 0 0 0);
		}
		.registros :global(.g-tabla tr) {
			display: grid;
			grid-template-columns: auto minmax(0, 1fr);
			gap: 4px 10px;
			padding: 10px 12px;
			border-top: 1px solid #e6dcc8;
		}
		.registros :global(.g-tabla tbody tr:first-child) {
			border-top: 0;
		}
		.registros :global(.g-tabla td) {
			display: block;
			padding: 0;
			border: 0;
		}
		.registros .celda-valor {
			grid-column: 1 / -1;
		}
		.copiar {
			grid-column: 1 / -1;
			text-align: left;
		}
	}
</style>
