<script lang="ts">
	import { enhance } from '$app/forms';
	import { fecha } from '#lib/formato.ts';
	import { paginas, prueba } from '#lib/cuenta/util.ts';
	let { data, form } = $props();
	const l = $derived(data.listado);
	const total = $derived(paginas(l.total, l.limite));
	const enlace = (p: number) => `?${new URLSearchParams({ ...(data.q ? { q: data.q } : {}), pagina: String(p) })}`;
</script>

<svelte:head><title>Negocios · Administración</title></svelte:head>

<div class="pila">
	<h1>Negocios</h1>
	<form method="GET" class="fila buscar" role="search">
		<label class="campo">
			<span class="visualmente-oculto">Buscar</span>
			<input type="search" name="q" value={data.q} placeholder="Nombre, dirección o email" maxlength="120" />
		</label>
		<button class="boton">Buscar</button>
		{#if data.q}<a class="boton" href="/admin/negocios">Limpiar</a>{/if}
	</form>
	<p class="suave chico" aria-live="polite">
		{l.total === 1 ? 'Un negocio' : `${l.total} negocios`}{data.q ? ` para “${data.q}”` : ''}
	</p>
	{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}

	{#if l.negocios.length === 0}
		<p class="tarjeta suave">No hay negocios que coincidan.</p>
	{/if}
	<ul class="lista pila">
		{#each l.negocios as n (n.id)}
			<li class="tarjeta pila" aria-label={n.nombre}>
				<div class="fila" style="justify-content: space-between">
					<div>
						<strong>{n.nombre}</strong>
						<span class="etiqueta" class:apagado={!n.activo}>{n.activo ? 'Activo' : 'Desactivado'}</span>
						{#if n.activo && !n.reservable}<span class="etiqueta apagado">No recibe turnos</span>{/if}
						<div class="suave chico">
							<a href={`/${n.slug}`} target="_blank" rel="noopener">/{n.slug}</a>{n.ciudad ? ` · ${n.ciudad}` : ''} · alta {fecha(n.creadoEn)}
						</div>
					</div>
				</div>
				<dl class="datos chico">
					<div><dt class="suave">Dueña</dt><dd>{n.duenia ?? '—'}</dd></div>
					<div><dt class="suave">Equipo</dt><dd>{n.miembros}</dd></div>
					<div><dt class="suave">Turnos (30 días)</dt><dd>{n.turnos30Dias}</dd></div>
					<div><dt class="suave">Prueba</dt><dd>{prueba(n.pruebaHasta)}</dd></div>
				</dl>
				<div class="fila">
					<form method="POST" action="?/activo" use:enhance>
						<input type="hidden" name="id" value={n.id} />
						<input type="hidden" name="activo" value={String(!n.activo)} />
						<button class="boton" class:peligro={n.activo}>{n.activo ? 'Desactivar' : 'Activar'}</button>
					</form>
					<form method="POST" action="?/extender" use:enhance class="fila">
						<input type="hidden" name="id" value={n.id} />
						<label class="campo dias">
							<span class="visualmente-oculto">Días para sumar a la prueba de {n.nombre}</span>
							<input name="dias" type="number" min="1" max="365" value="7" required />
						</label>
						<button class="boton">Extender prueba</button>
					</form>
				</div>
			</li>
		{/each}
	</ul>

	{#if total > 1}
		<nav class="fila" aria-label="Páginas" style="justify-content: space-between">
			{#if l.pagina > 1}<a class="boton" href={enlace(l.pagina - 1)}>Anterior</a>{:else}<span></span>{/if}
			<span class="suave chico">Página {l.pagina} de {total}</span>
			{#if l.pagina < total}<a class="boton" href={enlace(l.pagina + 1)}>Siguiente</a>{:else}<span></span>{/if}
		</nav>
	{/if}
</div>

<style>
	.buscar .campo {
		flex: 1 1 220px;
	}
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.etiqueta.apagado {
		background: #fdecea;
		color: var(--peligro);
	}
	.datos {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(140px, 100%), 1fr));
		gap: 8px;
		margin: 0;
	}
	.datos dd {
		margin: 0;
		overflow-wrap: anywhere;
	}
	.dias input {
		width: 80px;
	}
</style>
