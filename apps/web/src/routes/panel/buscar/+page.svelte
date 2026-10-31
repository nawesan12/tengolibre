<script lang="ts">
	import { fechaHora, pesos } from '#lib/formato.ts';
	import EstadoTurno from '#lib/operacion/EstadoTurno.svelte';
	import { GESTION } from '#lib/operacion/secciones.ts';

	let { data } = $props();
	const nada = $derived(data.q.length >= 2 && !data.clientes.length && !data.turnos.length && !data.servicios.length);
</script>

<svelte:head><title>Buscar · Panel</title></svelte:head>

<div class="pila">
	<h1>Buscar</h1>
	<form method="GET" role="search" class="fila" data-sveltekit-reset="false">
		<label class="campo crece">
			<span class="visualmente-oculto">Buscar en el panel</span>
			<input type="search" name="q" value={data.q} minlength="2" placeholder="Cliente, teléfono, email o servicio" />
		</label>
		<button class="boton primario">Buscar</button>
	</form>

	{#if data.q.length < 2}
		<p class="suave">Escribí al menos dos letras o números.</p>
	{:else if nada}
		<p class="tarjeta suave">No encontramos nada con «{data.q}».</p>
	{/if}

	{#if data.clientes.length}
		<section class="tarjeta pila">
			<h2>Clientes</h2>
			<ul class="lista">
				{#each data.clientes as c (c.id)}
					<li><a href={`/panel/clientes/${c.id}`}><strong>{c.nombre}</strong></a> <span class="suave chico">{[c.telefono, c.email].filter(Boolean).join(' · ')}</span></li>
				{/each}
			</ul>
		</section>
	{/if}

	{#if data.turnos.length}
		<section class="tarjeta pila">
			<h2>Turnos</h2>
			<ul class="lista">
				{#each data.turnos as t (t.id)}
					<li class="fila" style="justify-content: space-between">
						<span><a href={`/panel/turnos/${t.id}`}>{fechaHora(t.inicio)}</a> <span class="suave chico">· {t.cliente} · {t.servicio}{t.profesional ? ` · ${t.profesional}` : ''}</span></span>
						<EstadoTurno estado={t.estado} />
					</li>
				{/each}
			</ul>
		</section>
	{/if}

	{#if data.servicios.length}
		<section class="tarjeta pila">
			<h2>Servicios</h2>
			<ul class="lista">
				{#each data.servicios as s (s.id)}
					<li>
						{#if GESTION.includes(data.negocio.rol)}<a href="/panel/servicios"><strong>{s.nombre}</strong></a>{:else}<strong>{s.nombre}</strong>{/if}
						<span class="suave chico">· {s.duracion} min · {pesos(s.precio)}{s.activo ? '' : ' · pausado'}</span>
					</li>
				{/each}
			</ul>
		</section>
	{/if}
</div>

<style>
	.crece {
		flex: 1;
		min-width: 0;
	}
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
</style>
