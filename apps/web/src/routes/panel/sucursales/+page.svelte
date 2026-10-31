<script lang="ts">
	import '#lib/gestion/gestion.css';
	import { enhance } from '$app/forms';
	import { pesos } from '#lib/formato.ts';
	import { dma } from '#lib/cuenta/util.ts';
	import type { Numeros } from '#lib/cuenta/tipos.ts';

	let { data, form } = $props();
	let renombrando = $state(false);
	const esDuenia = $derived(data.grupos.some((g) => g.sucursales.length > 0) || data.sinGrupo.length > 0);
	const grupo = $derived(data.grupos.find((g) => g.id === data.grupoId));
	const r = $derived(data.reporte);
	const numerosDe = (id: string) => r?.sucursales.find((s) => s.id === id);

	/** «$ 4,8 M» para millones; el resto, en pesos. */
	const corto = (n: number) =>
		n >= 1_000_000 ? `$ ${(n / 1_000_000).toLocaleString('es-AR', { maximumFractionDigits: 1 })} M` : pesos(n);
	const porc = (a: number, b: number) => (b > 0 ? `${Math.round((a / b) * 100)}%` : '—');
	const entre = (n: number) => (n === 2 ? 'Entre los dos' : n > 2 ? `Entre los ${n}` : 'En total');

	const filas: { nombre: string; valor: (n: Numeros) => string }[] = [
		{ nombre: 'Facturación', valor: (n) => corto(n.ingresos) },
		{ nombre: 'Turnos', valor: (n) => String(n.turnos) },
		{ nombre: 'Turnos atendidos', valor: (n) => String(n.atendidos) },
		{ nombre: 'Ausencias', valor: (n) => porc(n.ausentes, n.turnos) },
		{ nombre: 'Cancelados', valor: (n) => String(n.cancelados) },
		{ nombre: 'Clientes', valor: (n) => String(n.clientes) }
	];
	const TONOS = ['#7a3b5e', '#183c44', '#17805a', '#5c3800', '#33474b'];
</script>

<svelte:head><title>Sucursales · Panel</title></svelte:head>

<div class="g-pagina sucursales">
	<div class="izquierda g-col">
		<div class="cabeza">
			<h1 class="g-titulo">Sucursales</h1>
			{#if grupo}
				<p class="g-nota">Grupo «{grupo.nombre}» · {grupo.sucursales.length === 1 ? '1 negocio' : `${grupo.sucursales.length} negocios`}</p>
			{:else}
				<p class="g-nota">Juntá los negocios de los que sos dueña o dueño para ver un reporte en común y compararlos entre&nbsp;sí.</p>
			{/if}
		</div>
		{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}

		{#if data.grupos.length > 1}
			<nav class="g-chips" aria-label="Grupos">
				{#each data.grupos as g (g.id)}
					<a class="g-chip" href={`?grupo=${g.id}`} aria-current={g.id === data.grupoId ? 'page' : undefined}>{g.nombre}</a>
				{/each}
			</nav>
		{/if}

		{#if grupo}
			{#if grupo.sucursales.length === 0}
				<div class="g-bloque"><p class="g-nota">Este grupo todavía no tiene negocios. Sumá el primero&nbsp;abajo.</p></div>
			{:else}
				<ul class="locales">
					{#each grupo.sucursales as s, i (s.id)}
						{@const n = numerosDe(s.id)}
						<li class="local">
							<span class="g-avatar cuadrado" style:background={TONOS[i % TONOS.length]} aria-hidden="true">{s.nombre.trim()[0]?.toUpperCase()}</span>
							<span class="g-fila-texto">
								<b>{s.nombre}</b>
								<span class="mayus">{[s.ciudad || 'Sin ciudad', n ? `${n.turnos} turnos` : ''].filter(Boolean).join(' · ')}</span>
							</span>
							{#if n}<b class="plata">{corto(n.ingresos)}</b>{/if}
						</li>
					{/each}
				</ul>
			{/if}

			{#if data.sinGrupo.length > 0}
				<details class="g-plegable sumar">
					<summary>Sumar un negocio al grupo</summary>
					<form method="POST" action="?/sumar" use:enhance class="g-bloque">
						<input type="hidden" name="id" value={grupo.id} />
						<label class="g-campo">
							Sumar un negocio a {grupo.nombre}
							<select name="negocio" required>
								{#each data.sinGrupo as s (s.id)}<option value={s.id}>{s.nombre}</option>{/each}
							</select>
						</label>
						<button class="g-boton oscuro">Sumar</button>
					</form>
				</details>
			{/if}

			<details class="g-plegable g-bloque administrar">
				<summary>Administrar el grupo</summary>
				{#if renombrando}
					<form
						method="POST"
						action="?/renombrar"
						use:enhance={() => async ({ update, result }) => {
							await update();
							if (result.type === 'success') renombrando = false;
						}}
						class="g-col"
					>
						<input type="hidden" name="id" value={grupo.id} />
						<label class="g-campo">Nombre del grupo <input name="nombre" value={grupo.nombre} required maxlength="80" /></label>
						<div class="fila-botones">
							<button class="g-boton menta">Guardar</button>
							<button type="button" class="g-boton" onclick={() => (renombrando = false)}>Cancelar</button>
						</div>
					</form>
				{:else}
					<button type="button" class="g-accion neutro" onclick={() => (renombrando = true)}><span class="icono" aria-hidden="true">edit</span>Renombrar</button>
				{/if}
				{#if grupo.sucursales.length > 0}
					<ul class="g-filas">
						{#each grupo.sucursales as s (s.id)}
							<li class="g-fila sacar">
								<span class="g-fila-texto"><b>{s.nombre}</b></span>
								<form method="POST" action="?/sacar" use:enhance>
									<input type="hidden" name="id" value={grupo.id} />
									<input type="hidden" name="negocio" value={s.id} />
									<button class="g-accion peligro" aria-label={`Sacar ${s.nombre} de ${grupo.nombre}`}>Sacar del grupo</button>
								</form>
							</li>
						{/each}
					</ul>
				{/if}
				<form
					method="POST"
					action="?/borrar"
					use:enhance={({ cancel }) => {
						if (!confirm(`¿Borrar el grupo ${grupo.nombre}? Los negocios quedan, sólo salen del grupo.`)) cancel();
					}}
				>
					<input type="hidden" name="id" value={grupo.id} />
					<button class="g-accion peligro"><span class="icono" aria-hidden="true">delete</span>Borrar el grupo</button>
				</form>
			</details>
		{/if}

		{#if esDuenia}
			<details class="g-plegable nuevo" open={data.grupos.length === 0}>
				<summary>Nuevo grupo</summary>
				<form method="POST" action="?/crear" use:enhance class="g-bloque">
					<label class="g-campo">Nombre del grupo <input name="nombre" required maxlength="80" placeholder="Lumi" /></label>
					<button class="g-boton menta">Crear grupo</button>
				</form>
			</details>
			{#if data.sinGrupo.length > 0 && !grupo}
				<p class="g-nota">Sin grupo: {data.sinGrupo.map((s) => s.nombre).join(', ')}.</p>
			{/if}
		{:else}
			<p class="g-aviso ambar">
				<span class="icono" aria-hidden="true">info</span>
				<span>Sólo se pueden agrupar los negocios de los que sos dueña o&nbsp;dueño.</span>
			</p>
		{/if}
	</div>

	{#if grupo}
		<div class="derecha g-col" id="reporte">
			{#if data.errorReporte}
				<p class="aviso error" role="alert">{data.errorReporte}</p>
			{:else if r}
				<section class="g-oscuro total" aria-labelledby="t-total">
					<h2 id="t-total" class="total-titulo">{entre(r.sucursales.length)}, del {dma(r.desde)} al {dma(r.hasta)}</h2>
					<b class="g-grande">{pesos(r.total.ingresos)}</b>
					<span class="total-sub">{r.total.turnos} turnos · {porc(r.total.ausentes, r.total.turnos)} de ausencias</span>
				</section>

				{#if r.sucursales.length > 0}
					<section class="g-bloque sin-pad" aria-labelledby="t-comparar">
						<h2 id="t-comparar" class="visualmente-oculto">Comparación entre locales</h2>
						<div class="g-desplazable">
							<table class="g-tabla">
								<thead>
									<tr>
										<th scope="col"><span class="visualmente-oculto">Qué</span></th>
										{#each r.sucursales as s (s.id)}<th scope="col">{s.nombre}</th>{/each}
									</tr>
								</thead>
								<tbody>
									{#each filas as f (f.nombre)}
										<tr>
											<th scope="row">{f.nombre}</th>
											{#each r.sucursales as s (s.id)}<td><b>{f.valor(s)}</b></td>{/each}
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					</section>
				{/if}
				<p class="g-nota">Turnos sin cancelados ni los que esperan el pago; facturación de los&nbsp;atendidos.</p>
			{/if}

			<form method="GET" class="g-bloque rango">
				<input type="hidden" name="grupo" value={grupo.id} />
				<label class="g-campo">Desde <input type="date" name="desde" value={data.desde} required /></label>
				<label class="g-campo">Hasta <input type="date" name="hasta" value={data.hasta} required /></label>
				<button class="g-boton oscuro">Ver reporte</button>
			</form>
		</div>
	{/if}
</div>

<style>
	.sucursales {
		display: grid;
		grid-template-columns: minmax(0, 1fr) minmax(0, 1.1fr);
		gap: 24px;
		align-items: start;
	}
	.cabeza {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.locales {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.local {
		display: flex;
		align-items: center;
		gap: 12px;
		min-height: 72px;
		padding: 14px;
		border-radius: var(--tl-r-m);
		background: var(--tl-papel);
	}
	.local .g-avatar {
		color: #fff;
	}
	.mayus {
		text-transform: uppercase;
		font-variant-numeric: tabular-nums;
	}
	.plata {
		font-size: 14px;
		font-variant-numeric: tabular-nums;
		white-space: nowrap;
	}
	.sumar > summary,
	.nuevo > summary {
		justify-content: center;
		min-height: 48px;
		border-radius: 999px;
		box-shadow: inset 0 0 0 1.5px #d8ccb4;
	}
	.sumar[open] > summary,
	.nuevo[open] > summary {
		margin-bottom: 12px;
	}
	.administrar {
		display: block;
		padding: 6px 18px;
	}
	.administrar[open] > :not(summary) {
		margin-top: 12px;
	}
	.administrar[open] {
		padding-bottom: 18px;
	}
	.administrar .g-fila {
		padding: 6px 0;
	}
	.fila-botones {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}
	.total {
		gap: 6px;
		padding: 20px;
	}
	.total-titulo {
		margin: 0;
		font-size: 13px;
		font-weight: 400;
		letter-spacing: normal;
		color: rgba(255, 255, 255, 0.78);
	}
	.total .g-grande {
		color: #fff;
	}
	.total-sub {
		font: 700 11px var(--tl-font);
		letter-spacing: 0.08em;
		text-transform: uppercase;
		font-variant-numeric: tabular-nums;
		color: var(--tl-menta);
	}
	.g-tabla th[scope='row'] {
		font: 400 14px var(--tl-font);
		letter-spacing: normal;
		text-transform: none;
		color: var(--tl-tinta);
		border-top: 1px solid #f0e8d8;
	}
	.g-tabla thead th {
		padding-top: 16px;
	}
	.g-tabla td b {
		font-weight: 700;
	}
	.rango {
		display: grid;
		grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) auto;
		align-items: end;
		gap: 12px;
	}
	@media (max-width: 1100px) {
		.sucursales {
			grid-template-columns: minmax(0, 1fr);
		}
	}
	@media (max-width: 640px) {
		.rango {
			grid-template-columns: minmax(0, 1fr);
		}
		.total .g-grande {
			font-size: 36px;
		}
	}
</style>
