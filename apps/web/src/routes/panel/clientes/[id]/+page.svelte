<script lang="ts">
	import { enhance } from '$app/forms';
	import { fecha, fechaHora, pesos } from '#lib/formato.ts';
	import EstadoTurno from '#lib/operacion/EstadoTurno.svelte';
	import { ATENCION, GESTION } from '#lib/operacion/secciones.ts';

	let { data, form } = $props();
	const c = $derived(data.ficha);
	const puedeEditar = $derived(ATENCION.includes(data.negocio.rol));
	const gestiona = $derived(GESTION.includes(data.negocio.rol));
	const sinPoner = $derived(data.etiquetas.filter((e) => !c.etiquetas.some((x) => x.id === e.id)));
	let editando = $state(false);

	const MOTIVOS: Record<string, string> = { ausencias: 'Por ausencias', manual: 'Puesta a mano' };
</script>

<svelte:head><title>{c.nombre} · Clientes · Panel</title></svelte:head>

<div class="pila">
	<p><a href="/panel/clientes" class="chico">← Clientes</a></p>
	{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}

	<section class="tarjeta pila">
		{#if editando}
			<form
				method="POST"
				action="?/editar"
				class="grilla"
				use:enhance={() =>
					async ({ result, update }) => {
						await update();
						if (result.type === 'success') editando = false;
					}}
			>
				<label class="campo">Nombre <input name="nombre" value={c.nombre} required maxlength="80" /></label>
				<label class="campo">Email <input name="email" type="email" value={c.email} maxlength="120" /></label>
				<div class="fila" style="align-self: end">
					<button class="boton primario">Guardar</button>
					<button type="button" class="boton" onclick={() => (editando = false)}>Cancelar</button>
				</div>
			</form>
		{:else}
			<div class="fila" style="justify-content: space-between">
				<h1>{c.nombre}</h1>
				<div class="fila">
					{#if puedeEditar}<button class="boton" onclick={() => (editando = true)}>Editar</button>{/if}
					<a class="boton primario" href={`/panel/turnos/nuevo?cliente=${c.id}`}>Nuevo turno</a>
				</div>
			</div>
			<p class="suave">
				{#if c.telefono}<a href={`tel:${c.telefono}`}>{c.telefono}</a>{/if}
				{#if c.telefono && c.email}·{/if}
				{#if c.email}<a href={`mailto:${c.email}`}>{c.email}</a>{/if}
				· Cliente desde {fecha(c.creadoEn)}
			</p>
		{/if}
		<div class="grilla numeros">
			<div><div class="suave chico">Visitas</div><strong>{c.visitas}</strong></div>
			<div><div class="suave chico">Gastado</div><strong>{pesos(c.gastado)}</strong></div>
			<div><div class="suave chico">No vino</div><strong>{c.ausencias}</strong></div>
			<div><div class="suave chico">Próximo turno</div><strong>{c.proximoTurno ? fechaHora(c.proximoTurno) : '—'}</strong></div>
		</div>
	</section>

	<section class="tarjeta pila" aria-labelledby="t-etiquetas">
		<h2 id="t-etiquetas">Etiquetas</h2>
		<div class="fila">
			{#each c.etiquetas as e (e.id)}
				<form method="POST" action="?/etiqueta" use:enhance class="fila">
					<input type="hidden" name="etiqueta" value={e.id} />
					<input type="hidden" name="quitar" value="1" />
					<span class="etiqueta" style={`background: ${e.color}22; color: ${e.color}`}>{e.nombre}</span>
					{#if puedeEditar}<button class="boton chico" aria-label={`Quitar ${e.nombre}`}>×</button>{/if}
				</form>
			{:else}
				<span class="suave chico">Sin etiquetas.</span>
			{/each}
		</div>
		{#if puedeEditar && sinPoner.length > 0}
			<form method="POST" action="?/etiqueta" use:enhance class="fila">
				<label class="campo">
					<span class="visualmente-oculto">Etiqueta para agregar</span>
					<select name="etiqueta" aria-label="Etiqueta para agregar">
						{#each sinPoner as e (e.id)}<option value={e.id}>{e.nombre}</option>{/each}
					</select>
				</label>
				<button class="boton">Poner etiqueta</button>
			</form>
		{:else if puedeEditar && data.etiquetas.length === 0}
			<p class="suave chico">Todavía no hay etiquetas. Creálas en <a href="/panel/etiquetas">Etiquetas</a>.</p>
		{/if}
	</section>

	<section class="tarjeta pila" aria-labelledby="t-notas">
		<h2 id="t-notas">Notas internas</h2>
		<form method="POST" action="?/nota" use:enhance class="pila">
			<label class="campo"><span class="visualmente-oculto">Nota nueva</span><textarea name="texto" required maxlength="2000" rows="2" placeholder="Algo para recordar de este cliente"></textarea></label>
			<div><button class="boton">Agregar nota</button></div>
		</form>
		<ul class="notas">
			{#each data.notas as n (n.id)}
				<li>
					<p>{n.texto}</p>
					<div class="fila chico suave">
						{n.autor || 'Alguien del equipo'} · {fechaHora(n.creadoEn)}
						<form method="POST" action="?/borrarNota" use:enhance>
							<input type="hidden" name="nota" value={n.id} />
							<button class="boton chico peligro" aria-label="Borrar nota">Borrar</button>
						</form>
					</div>
				</li>
			{/each}
		</ul>
	</section>

	<section class="tarjeta pila" aria-labelledby="t-penalidad">
		<h2 id="t-penalidad">Penalidad</h2>
		{#if c.penalidad}
			<p class="aviso error">
				No puede reservar online hasta el {fecha(c.penalidad.hasta)} ({MOTIVOS[c.penalidad.motivo] ?? c.penalidad.motivo}{c.penalidad.nota ? `: ${c.penalidad.nota}` : ''}).
			</p>
			{#if gestiona}
				<form method="POST" action="?/levantar" use:enhance>
					<input type="hidden" name="id" value={c.penalidad.id} />
					<button class="boton">Levantar penalidad</button>
				</form>
			{/if}
		{:else}
			<p class="suave">Sin penalidad vigente.</p>
			{#if gestiona}
				<form method="POST" action="?/penalizar" use:enhance class="grilla">
					<label class="campo">Días sin reservar <input type="number" name="dias" min="1" max="365" value="30" required /></label>
					<label class="campo">Motivo <input name="nota" maxlength="500" placeholder="Opcional" /></label>
					<div class="fila" style="align-self: end"><button class="boton peligro">Penalizar</button></div>
				</form>
			{/if}
		{/if}
		{#if data.penalidades.some((p) => !p.vigente)}
			<details>
				<summary class="chico">Penalidades anteriores</summary>
				<ul class="chico">
					{#each data.penalidades.filter((p) => !p.vigente) as p (p.id)}
						<li>{fecha(p.creadoEn)} hasta {fecha(p.levantadaEn ?? p.hasta)} · {MOTIVOS[p.motivo] ?? p.motivo}{p.levantadaEn ? ' (levantada)' : ''}</li>
					{/each}
				</ul>
			</details>
		{/if}
	</section>

	<section class="tarjeta pila" aria-labelledby="t-historial">
		<h2 id="t-historial">Historial de turnos</h2>
		{#if c.turnos.length === 0}
			<p class="suave">Todavía no tiene turnos.</p>
		{:else}
			<ul class="historial">
				{#each c.turnos as t (t.id)}
					<li class="fila" style="justify-content: space-between">
						<span>
							<a href={`/panel/turnos/${t.id}`}>{fechaHora(t.inicio)}</a>
							<span class="suave chico">· {t.servicio} · {t.profesional || 'Sin profesional'} · {pesos(t.precio)}</span>
						</span>
						<EstadoTurno estado={t.estado} />
					</li>
				{/each}
			</ul>
		{/if}
	</section>
</div>

<style>
	.numeros {
		grid-template-columns: repeat(auto-fill, minmax(min(140px, 100%), 1fr));
	}
	.notas,
	.historial {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.notas li {
		border-top: 1px solid var(--borde);
		padding-top: 8px;
	}
	.notas p {
		margin: 0 0 4px;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}
	.historial li {
		border-bottom: 1px solid var(--borde);
		padding-bottom: 8px;
	}
</style>
