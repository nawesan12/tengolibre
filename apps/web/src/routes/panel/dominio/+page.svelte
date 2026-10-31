<script lang="ts">
	import { enhance } from '$app/forms';
	import { fechaHora } from '#lib/formato.ts';
	import { ESTADOS_DOMINIO } from '#lib/configuracion/textos.ts';
	import { confirmar } from '#lib/configuracion/formularios.ts';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';

	let { data, form } = $props();
	const d = $derived(data.dominio);
	const esDuenio = $derived(data.negocio.rol === 'duenio');
	let verificando = $state(false);
</script>

<svelte:head><title>Dominio propio · Panel</title></svelte:head>

<div class="pila">
	<p class="chico"><a href="/panel/mi-web">← Mi web</a></p>
	<h1>Dominio propio</h1>
	<p class="suave">Tu página puede verse en una dirección tuya, como <strong>turnos.tunegocio.com</strong>, además de la de&nbsp;TengoLibre.</p>
	<Mensaje {form} que="dominio" />

	{#if !d}
		<section class="tarjeta pila">
			<h2>Conectar un dominio</h2>
			{#if esDuenio}
				<form method="POST" action="?/poner" use:enhance class="fila">
					<label class="campo crece">
						Dominio
						<input name="dominio" required maxlength="253" placeholder="turnos.tunegocio.com" autocapitalize="none" spellcheck="false" />
					</label>
					<button class="boton primario" style="align-self: end">Conectar</button>
				</form>
			{:else}
				<p class="suave">Sólo la dueña o el dueño del negocio puede conectar un&nbsp;dominio.</p>
			{/if}
		</section>
	{:else}
		<section class="tarjeta pila">
			<div class="fila entre">
				<h2 style="margin: 0">{d.nombre}</h2>
				<span class="etiqueta" class:mal={d.estado === 'error'}>{ESTADOS_DOMINIO[d.estado] ?? d.estado}</span>
			</div>
			{#if d.estado === 'error' && d.detalle}<p class="aviso error">{d.detalle}</p>{/if}
			{#if d.chequeadoEn}<p class="suave chico">Último chequeo: {fechaHora(d.chequeadoEn)}</p>{/if}

			{#if d.estado !== 'verificado'}
				<h3>Cargá estos registros en tu proveedor de dominio</h3>
				<div class="desplazable">
					<table class="tabla">
						<thead><tr><th>Tipo</th><th>Nombre</th><th>Valor</th></tr></thead>
						<tbody>
							{#each d.registros as r (r.tipo + r.nombre + r.valor)}
								<tr>
									<td>{r.tipo}{r.alternativa ? ' (o en vez del CNAME)' : ''}</td>
									<td><code>{r.nombre}</code></td>
									<td><code class="valor">{r.valor}</code></td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
				<p class="suave chico">Los cambios de DNS pueden tardar desde minutos hasta un día. Cuando estén, tocá&nbsp;verificar.</p>
			{/if}

			<div class="fila">
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
					<button class="boton primario" disabled={verificando}>{verificando ? 'Verificando…' : 'Verificar ahora'}</button>
				</form>
				{#if esDuenio}
					<form method="POST" action="?/quitar" use:enhance={confirmar(`¿Desconectar ${d.nombre}? Tu página deja de verse ahí.`)}>
						<button class="boton peligro">Desconectar</button>
					</form>
				{/if}
			</div>
		</section>
	{/if}
</div>

<style>
	.entre {
		justify-content: space-between;
	}
	.crece {
		flex: 1;
		min-width: min(260px, 100%);
	}
	h3 {
		font-size: 16px;
		margin: 0;
	}
	.valor {
		word-break: break-all;
	}
	.mal {
		background: #fdecea;
		color: var(--peligro);
	}
</style>
