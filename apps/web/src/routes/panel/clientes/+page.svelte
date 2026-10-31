<script lang="ts">
	import { enhance } from '$app/forms';
	import { fecha, pesos } from '#lib/formato.ts';
	import Pestanias from '#lib/operacion/Pestanias.svelte';
	import { ATENCION, CLIENTES } from '#lib/operacion/secciones.ts';

	let { data, form } = $props();
	let alta = $state(false);
	const paginas = $derived(Math.max(1, Math.ceil(data.total / data.porPagina)));
	const puedeEditar = $derived(ATENCION.includes(data.negocio.rol));

	const enlace = (pagina: number) => {
		const q = new URLSearchParams({ orden: data.filtros.orden, pagina: String(pagina) });
		if (data.filtros.q) q.set('q', data.filtros.q);
		if (data.filtros.etiqueta) q.set('etiqueta', data.filtros.etiqueta);
		if (data.filtros.penalizados) q.set('penalizados', '1');
		return `?${q}`;
	};
</script>

<svelte:head><title>Clientes · Panel</title></svelte:head>

<div class="pila">
	<div class="fila" style="justify-content: space-between">
		<h1>Clientes</h1>
		{#if puedeEditar}<button class="boton primario" aria-expanded={alta} onclick={() => (alta = !alta)}>Nuevo cliente</button>{/if}
	</div>
	<Pestanias items={CLIENTES} etiqueta="Clientes" />

	{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}

	{#if alta}
		<form method="POST" action="?/crear" use:enhance class="tarjeta grilla" aria-label="Nuevo cliente">
			<label class="campo">Nombre <input name="nombre" required maxlength="80" /></label>
			<label class="campo">Teléfono <input name="telefono" type="tel" maxlength="30" /></label>
			<label class="campo">Email <input name="email" type="email" maxlength="120" /></label>
			<div class="fila" style="align-self: end"><button class="boton primario">Agregar</button></div>
		</form>
	{/if}

	<form method="GET" class="tarjeta grilla filtros" role="search" aria-label="Buscar clientes" data-sveltekit-reset="false">
		<label class="campo">Buscar <input type="search" name="q" value={data.filtros.q} placeholder="Nombre, teléfono o email" /></label>
		<label class="campo">
			Etiqueta
			<select name="etiqueta">
				<option value="">Todas</option>
				{#each data.etiquetas as e (e.id)}<option value={e.id} selected={e.id === data.filtros.etiqueta}>{e.nombre}</option>{/each}
			</select>
		</label>
		<label class="campo">
			Orden
			<select name="orden">
				<option value="nombre" selected={data.filtros.orden === 'nombre'}>Por nombre</option>
				<option value="reciente" selected={data.filtros.orden === 'reciente'}>Más nuevos</option>
				<option value="ultima" selected={data.filtros.orden === 'ultima'}>Última visita</option>
				<option value="visitas" selected={data.filtros.orden === 'visitas'}>Más visitas</option>
				<option value="gastado" selected={data.filtros.orden === 'gastado'}>Más gastado</option>
			</select>
		</label>
		<label class="fila chico" style="align-self: end; min-height: 40px"><input type="checkbox" name="penalizados" value="1" checked={data.filtros.penalizados} /> Sólo penalizados</label>
		<div class="fila" style="align-self: end"><button class="boton">Buscar</button></div>
	</form>

	<p class="suave chico" aria-live="polite">{data.total}&nbsp;{data.total === 1 ? 'cliente' : 'clientes'}</p>

	{#if data.clientes.length === 0}
		<p class="tarjeta suave">No encontramos clientes con esos filtros.</p>
	{:else}
		<ul class="lista">
			{#each data.clientes as c (c.id)}
				<li class="tarjeta cliente">
					<div class="quien">
						<a href={`/panel/clientes/${c.id}`}><strong>{c.nombre}</strong></a>
						{#if c.penalizado}<span class="etiqueta penalizado">Penalizado</span>{/if}
						<div class="suave chico">{[c.telefono, c.email].filter(Boolean).join(' · ') || 'Sin contacto'}</div>
						{#if c.etiquetas.length}
							<div class="fila">
								{#each c.etiquetas as e (e.id)}<span class="etiqueta" style={`background: ${e.color}22; color: ${e.color}`}>{e.nombre}</span>{/each}
							</div>
						{/if}
					</div>
					<dl class="numeros chico">
						<div><dt class="suave">Visitas</dt><dd>{c.visitas}</dd></div>
						<div><dt class="suave">Gastado</dt><dd>{pesos(c.gastado)}</dd></div>
						<div><dt class="suave">Última</dt><dd>{c.ultimaVisita ? fecha(c.ultimaVisita) : '—'}</dd></div>
						<div><dt class="suave">Próximo</dt><dd>{c.proximoTurno ? fecha(c.proximoTurno) : '—'}</dd></div>
					</dl>
				</li>
			{/each}
		</ul>
	{/if}

	{#if paginas > 1}
		<nav class="fila" aria-label="Páginas">
			{#if data.pagina > 1}<a class="boton chico" href={enlace(data.pagina - 1)}>← Anterior</a>{/if}
			<span class="chico suave">Página {data.pagina} de {paginas}</span>
			{#if data.pagina < paginas}<a class="boton chico" href={enlace(data.pagina + 1)}>Siguiente →</a>{/if}
		</nav>
	{/if}
</div>

<style>
	.filtros {
		grid-template-columns: repeat(auto-fill, minmax(min(180px, 100%), 1fr));
	}
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.cliente {
		display: flex;
		flex-wrap: wrap;
		gap: 8px 16px;
		justify-content: space-between;
	}
	.quien {
		min-width: 0;
		overflow-wrap: anywhere;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}
	.penalizado {
		background: #fdecea;
		color: var(--peligro);
		align-self: start;
	}
	.numeros {
		display: grid;
		grid-template-columns: repeat(4, auto);
		gap: 4px 16px;
		margin: 0;
	}
	.numeros dd {
		margin: 0;
	}
	@media (max-width: 480px) {
		.numeros {
			grid-template-columns: repeat(2, 1fr);
			width: 100%;
		}
	}
</style>
