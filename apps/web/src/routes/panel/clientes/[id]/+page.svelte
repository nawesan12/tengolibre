<!--
	La ficha del cliente (handoff «Panel v2», ficha lateral): cabeza petróleo con sus números y
	contacto, la penalidad, las notas del equipo, las etiquetas y sus turnos como talones. Al pie,
	los derechos del cliente: corregir sus datos (el lápiz) y borrarlos si lo pide.
-->
<script lang="ts">
	import { page } from '$app/state';
	import { enhance } from '$app/forms';
	import Icono from '#lib/ui/Icono.svelte';
	import { fecha, fechaHora, hora, pesos } from '#lib/formato.ts';
	import EstadoTurno from '#lib/operacion/EstadoTurno.svelte';
	import { ATENCION, GESTION } from '#lib/operacion/secciones.ts';
	import { accion, colorEtiqueta, fechaCorta } from '#lib/operacion/ui.ts';
	import { iniciales, primerNombre } from '#lib/panel/dia.ts';

	let { data, form } = $props();
	const c = $derived(data.ficha);
	const puedeEditar = $derived(ATENCION.includes(data.negocio.rol));
	const gestiona = $derived(GESTION.includes(data.negocio.rol));
	const sinPoner = $derived(data.etiquetas.filter((e) => !c.etiquetas.some((x) => x.id === e.id)));
	let editando = $state(false);
	let poniendo = $state(false);
	let penalizando = $state(false);
	const MOTIVOS: Record<string, string> = { ausencias: 'Por ausencias', manual: 'Puesta a mano' };
	const wa = $derived(c.telefono ? `https://wa.me/${c.telefono.replace(/\D/g, '').replace(/^(?!54)/, '549')}` : '');
	const fechaLarga = (iso: string) => new Intl.DateTimeFormat('es-AR', { day: 'numeric', month: 'long', timeZone: 'America/Argentina/Buenos_Aires' }).format(new Date(iso));
	const recargar = () => async ({ update }: { update: (o?: { reset?: boolean }) => Promise<void> }) => {
		await update();
		poniendo = false;
		penalizando = false;
	};
</script>

<svelte:head><title>{c.nombre} · Clientes · Panel</title></svelte:head>

<p class="volver"><a href="/panel/clientes"><Icono nombre="arrow_back" tam={18} />Clientes</a></p>
{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}

<article class="ficha" aria-label="Ficha de cliente">
	<header class="cabeza tl-petroleo">
		{#if editando}
			<form
				method="POST"
				action={accion(page.url, 'editar')}
				class="editar"
				use:enhance={() =>
					async ({ result, update }) => {
						await update();
						if (result.type === 'success') editando = false;
					}}
			>
				<label class="campo">Nombre <input name="nombre" value={c.nombre} required maxlength="80" /></label>
				<label class="campo">Teléfono <input name="telefono" type="tel" value={c.telefono} maxlength="30" autocomplete="off" /></label>
				<label class="campo">Email <input name="email" type="email" value={c.email} maxlength="120" /></label>
				<div class="botones">
					<button class="boton primario">Guardar</button>
					<button type="button" class="boton" onclick={() => (editando = false)}>Cancelar</button>
				</div>
			</form>
		{:else}
			<div class="quien">
				<span class="avatar">{iniciales(c.nombre)}</span>
				<span class="nombre">
					<h1>{c.nombre}</h1>
					<span class="tags">
						{#each c.etiquetas as e (e.id)}
							{@const col = colorEtiqueta(e.color)}
							<form method="POST" action={accion(page.url, 'etiqueta')} use:enhance class="tag" style:background={col.fondo} style:color={col.texto}>
								<input type="hidden" name="etiqueta" value={e.id} />
								<input type="hidden" name="quitar" value="1" />
								{e.nombre}
								{#if puedeEditar}<button aria-label={`Quitar ${e.nombre}`}><Icono nombre="close" tam={14} /></button>{/if}
							</form>
						{/each}
						{#if c.penalidad}<span class="tag penal">Penalizada</span>{/if}
						{#if puedeEditar && data.etiquetas.length}
							<button class="mas-tag" aria-expanded={poniendo} onclick={() => (poniendo = !poniendo)}>+ Etiqueta</button>
						{/if}
					</span>
				</span>
				{#if puedeEditar}<button class="icono-b" aria-label="Corregir nombre, teléfono y email" onclick={() => (editando = true)}><Icono nombre="edit" tam={20} /></button>{/if}
			</div>
			{#if poniendo}
				{#if sinPoner.length}
					<form method="POST" action={accion(page.url, 'etiqueta')} use:enhance={recargar} class="poner">
						<label class="campo">
							<span class="visualmente-oculto">Etiqueta para agregar</span>
							<select name="etiqueta" aria-label="Etiqueta para agregar">
								{#each sinPoner as e (e.id)}<option value={e.id}>{e.nombre}</option>{/each}
							</select>
						</label>
						<button class="boton primario">Poner etiqueta</button>
					</form>
				{:else}
					<p class="claro">Ya tiene todas. Creá otras en <a href="/panel/etiquetas">Etiquetas</a>.</p>
				{/if}
			{/if}
		{/if}
		<div class="numeros">
			<div><b class="tl-num">{c.visitas}</b><span>visitas</span></div>
			<div><b class="tl-num">{c.ausencias}</b><span>{c.ausencias === 1 ? 'ausencia' : 'ausencias'}</span></div>
			<div><b class="tl-num">{pesos(c.gastado)}</b><span>gastado</span></div>
		</div>
		<div class="contacto">
			{#if c.telefono}<a class="chip-c tl-num" href={wa} target="_blank" rel="noopener"><img src="/logos/whatsapp.svg" alt="" width="16" height="16" />{c.telefono}</a>{/if}
			{#if c.telefono}<a class="chip-c" href={`tel:${c.telefono}`}><Icono nombre="call" tam={17} />Llamar</a>{/if}
			{#if c.email}<a class="chip-c" href={`mailto:${c.email}`}>{c.email}</a>{/if}
			{#if !c.telefono && !c.email}<span class="claro">Sin datos de contacto</span>{/if}
		</div>
		<span class="desde">Cliente desde {fecha(c.creadoEn)}</span>
	</header>

	<div class="cuerpo">
		<section aria-labelledby="t-penalidad" class="seccion">
			<h2 id="t-penalidad" class="tl-rotulo rot">Penalidad</h2>
			{#if c.penalidad}
				<div class="penal-caja" role="status">
					<span class="penal-txt">
						<Icono nombre="block" tam={20} />
						<span><b>No puede reservar online hasta el {fechaLarga(c.penalidad.hasta)}.</b> {MOTIVOS[c.penalidad.motivo] ?? c.penalidad.motivo}{c.penalidad.nota ? `: ${c.penalidad.nota}` : ''}. Vos sí podés cargarle turnos desde el&nbsp;panel.</span>
					</span>
					{#if gestiona}
						<form method="POST" action={accion(page.url, 'levantar')} use:enhance>
							<input type="hidden" name="id" value={c.penalidad.id} />
							<button class="levantar">Levantar la penalidad</button>
						</form>
					{/if}
				</div>
			{:else}
				<p class="claro-c">Sin penalidad vigente.</p>
				{#if gestiona}
					{#if penalizando}
						<form method="POST" action={accion(page.url, 'penalizar')} use:enhance={recargar} class="penalizar">
							<label class="campo">Días sin reservar <input type="number" name="dias" min="1" max="365" value="30" required /></label>
							<label class="campo">Motivo <input name="nota" maxlength="500" placeholder="Opcional" /></label>
							<button class="boton peligro">Penalizar</button>
						</form>
					{:else}
						<button class="link" onclick={() => (penalizando = true)}>Penalizar a mano</button>
					{/if}
				{/if}
			{/if}
			{#if data.penalidades.some((p) => !p.vigente)}
				<details>
					<summary>Penalidades anteriores</summary>
					<ul class="anteriores">
						{#each data.penalidades.filter((p) => !p.vigente) as p (p.id)}
							<li>{fecha(p.creadoEn)} hasta {fecha(p.levantadaEn ?? p.hasta)} · {MOTIVOS[p.motivo] ?? p.motivo}{p.levantadaEn ? ' (levantada)' : ''}</li>
						{/each}
					</ul>
				</details>
			{/if}
		</section>

		<section aria-labelledby="t-notas" class="seccion">
			<h2 id="t-notas" class="tl-rotulo rot">Notas del equipo</h2>
			<ul class="notas">
				{#each data.notas as n (n.id)}
					<li>
						<p>{n.texto}</p>
						<span class="quien-nota tl-num">
							{(n.autor || 'Alguien del equipo').toUpperCase()} · {fechaCorta(n.creadoEn)}
							<form method="POST" action={accion(page.url, 'borrarNota')} use:enhance>
								<input type="hidden" name="nota" value={n.id} />
								<button aria-label="Borrar nota">Borrar</button>
							</form>
						</span>
					</li>
				{/each}
			</ul>
			<form method="POST" action={accion(page.url, 'nota')} use:enhance class="nueva-nota">
				<label class="visualmente-oculto" for="nota-nueva">Nota nueva</label>
				<input id="nota-nueva" name="texto" required maxlength="2000" placeholder="Agregar una nota…" autocomplete="off" />
				<button class="agregar">Agregar nota</button>
			</form>
		</section>

		<section aria-labelledby="t-historial" class="seccion">
			<h2 id="t-historial" class="tl-rotulo rot">Historial de turnos</h2>
			{#if c.turnos.length === 0}
				<p class="claro-c">Todavía no tiene turnos.</p>
			{:else}
				<ul class="historial">
					{#each c.turnos as t (t.id)}
						<li class:paso={t.estado === 'atendido' || t.estado === 'ausente'}>
							<a href={`/panel/turnos/${t.id}`} aria-label={`${fechaHora(t.inicio)}, ${t.servicio}`}>
								<span class="stub tl-num"><b>{fechaCorta(t.inicio)}</b><span>{hora(t.inicio)}</span></span>
								<span class="medio"><b>{t.servicio}</b><span>{t.profesional ? `con ${primerNombre(t.profesional)}` : 'Sin profesional'} · {pesos(t.precio)}</span></span>
								<span class="est"><EstadoTurno estado={t.estado} chico /></span>
							</a>
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	</div>

	<footer class="pie">
		<a class="turno-para" href={`/panel/turnos/nuevo?cliente=${c.id}`}><Icono nombre="add" tam={20} />Turno para {primerNombre(c.nombre)}</a>
	</footer>
</article>

{#if puedeEditar}
	<section aria-labelledby="t-borrar" class="borrar-datos">
		<h2 id="t-borrar" class="tl-rotulo rot">Borrar sus datos</h2>
		<p class="claro-c">
			Si te lo pide, borramos su nombre, teléfono, email, notas y etiquetas. Sus turnos pasados quedan en la agenda, sin&nbsp;datos.
		</p>
		<form
			method="POST"
			action={accion(page.url, 'suprimir')}
			use:enhance={({ cancel }) => {
				if (!confirm(`¿Borrar los datos de ${c.nombre}? No se puede deshacer.`)) cancel();
			}}
		>
			<button class="link">Borrar los datos de {primerNombre(c.nombre)}</button>
		</form>
	</section>
{/if}

<style>
	.volver {
		margin: 0;
	}
	.volver a {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		min-height: 40px;
		font-size: 14px;
		font-weight: 700;
		color: var(--tl-menta-texto);
		text-decoration: none;
	}
	.ficha {
		width: 100%;
		max-width: 560px;
		border-radius: 24px;
		background: var(--tl-papel);
		box-shadow: 0 30px 60px -36px rgba(15, 35, 40, 0.55);
		overflow: hidden;
		display: flex;
		flex-direction: column;
		animation: tl-gin var(--tl-entrar) var(--tl-ease) both;
	}
	.cabeza {
		padding: 20px 20px 18px;
		display: flex;
		flex-direction: column;
		gap: 14px;
	}
	.quien {
		display: flex;
		align-items: center;
		gap: 12px;
	}
	.avatar {
		flex: none;
		width: 56px;
		height: 56px;
		border-radius: 50%;
		background: var(--tl-crema);
		color: var(--tl-tinta);
		display: grid;
		place-items: center;
		font-size: 18px;
		font-weight: 700;
	}
	.nombre {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	h1 {
		margin: 0;
		font-size: 22px;
		line-height: 1.1;
		letter-spacing: -0.03em;
		overflow-wrap: anywhere;
	}
	.tags {
		display: flex;
		gap: 6px;
		flex-wrap: wrap;
		align-items: center;
	}
	.tag {
		display: inline-flex;
		align-items: center;
		gap: 2px;
		font-size: 11px;
		font-weight: 700;
		letter-spacing: 0.05em;
		text-transform: uppercase;
		padding: 2px 4px 2px 7px;
		border-radius: 5px;
		min-height: 22px;
	}
	.tag button {
		border: 0;
		background: transparent;
		color: inherit;
		display: grid;
		place-items: center;
		width: 22px;
		height: 22px;
		cursor: pointer;
	}
	.tag.penal {
		background: #fbe9c2;
		color: #7a4a00;
		padding-right: 7px;
	}
	.mas-tag {
		min-height: 28px;
		padding: 0 10px;
		border: 1px dashed rgba(255, 255, 255, 0.4);
		border-radius: 6px;
		background: transparent;
		color: #fff;
		font: 700 11px var(--tl-font);
		letter-spacing: 0.05em;
		text-transform: uppercase;
		cursor: pointer;
	}
	.icono-b {
		flex: none;
		width: 44px;
		height: 44px;
		border-radius: 14px;
		border: 0;
		background: rgba(255, 255, 255, 0.1);
		color: #fff;
		display: grid;
		place-items: center;
		cursor: pointer;
	}
	.editar,
	.poner {
		display: grid;
		gap: 10px;
	}
	.poner {
		grid-template-columns: 1fr auto;
		align-items: end;
	}
	.editar .campo,
	.poner .campo {
		color: #fff;
	}
	.botones {
		display: flex;
		gap: 8px;
	}
	.numeros {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 8px;
	}
	.numeros div {
		display: flex;
		flex-direction: column;
		gap: 2px;
		padding: 10px 12px;
		border-radius: 12px;
		background: rgba(255, 255, 255, 0.08);
		min-width: 0;
	}
	.numeros b {
		font-size: 20px;
		font-weight: 700;
		letter-spacing: -0.03em;
		overflow-wrap: anywhere;
	}
	.numeros span {
		font-size: 11px;
		color: rgba(255, 255, 255, 0.7);
	}
	.contacto {
		display: flex;
		gap: 8px;
		flex-wrap: wrap;
	}
	.chip-c {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		min-height: 40px;
		padding: 0 12px;
		border-radius: 999px;
		background: rgba(255, 255, 255, 0.1);
		color: #fff;
		font-size: 13px;
		font-weight: 700;
		text-decoration: none;
		overflow-wrap: anywhere;
	}
	.chip-c img {
		background: #fff;
		border-radius: 50%;
	}
	.claro,
	.desde {
		margin: 0;
		font-size: 12px;
		color: rgba(255, 255, 255, 0.7);
	}
	.claro a {
		color: #bff0d9;
	}
	.cuerpo {
		padding: 18px 20px 24px;
		display: flex;
		flex-direction: column;
		gap: 22px;
	}
	.seccion {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.rot {
		margin: 0;
		font-size: 11px;
		color: var(--tl-sub);
	}
	.claro-c {
		margin: 0;
		font-size: 14px;
		color: var(--tl-sub);
	}
	.penal-caja {
		display: flex;
		flex-direction: column;
		gap: 10px;
		padding: 14px 16px;
		border-radius: 16px;
		background: #fbe9c2;
		color: #5c3800;
	}
	.penal-txt {
		display: flex;
		gap: 8px;
		align-items: flex-start;
		font-size: 14px;
		line-height: 1.45;
	}
	.penal-txt :global(.icono) {
		color: #7a4a00;
	}
	.levantar {
		align-self: flex-start;
		min-height: 40px;
		padding: 0 16px;
		border: 0;
		border-radius: 999px;
		background: var(--tl-petroleo);
		color: #fff;
		font: 700 13px var(--tl-font);
		cursor: pointer;
	}
	.penalizar {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(150px, 100%), 1fr));
		gap: 10px;
		align-items: end;
	}
	.link {
		align-self: flex-start;
		min-height: 40px;
		border: 0;
		background: transparent;
		padding: 0;
		font: 700 13px var(--tl-font);
		color: #a3241a;
		cursor: pointer;
	}
	details summary {
		font-size: 13px;
		font-weight: 700;
		color: var(--tl-menta-texto);
		cursor: pointer;
		min-height: 36px;
		display: flex;
		align-items: center;
	}
	.anteriores {
		margin: 0;
		padding-left: 18px;
		font-size: 13px;
		color: var(--tl-sub);
	}
	.notas,
	.historial {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.notas li {
		padding: 10px 12px;
		border-radius: 12px;
		background: #f8f3ea;
		font-size: 14px;
		line-height: 1.45;
	}
	.notas p {
		margin: 0;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}
	.quien-nota {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-top: 4px;
		font-size: 11px;
		color: var(--tl-sub);
	}
	.quien-nota button {
		border: 0;
		background: transparent;
		padding: 0 4px;
		min-height: 28px;
		font: 700 11px var(--tl-font);
		color: #a3241a;
		cursor: pointer;
	}
	.nueva-nota {
		display: flex;
		gap: 8px;
	}
	.nueva-nota input {
		flex: 1;
		min-width: 0;
		height: 46px;
		padding: 0 14px;
		border: 0;
		border-radius: 12px;
		background: var(--tl-papel);
		box-shadow: inset 0 0 0 1.5px var(--tl-linea-2);
		font: 400 16px var(--tl-font);
	}
	.nueva-nota input:focus {
		outline: none;
		box-shadow: inset 0 0 0 2px var(--tl-menta);
	}
	.agregar {
		height: 46px;
		padding: 0 14px;
		border: 0;
		border-radius: 12px;
		background: var(--tl-crema);
		font: 700 13px var(--tl-font);
		color: var(--tl-tinta);
		cursor: pointer;
		white-space: nowrap;
	}
	.historial a {
		display: grid;
		grid-template-columns: 84px minmax(0, 1fr) auto;
		min-height: 56px;
		border-radius: 12px;
		background:
			radial-gradient(circle at 84px 0, var(--tl-papel) 7px, transparent 7.5px),
			radial-gradient(circle at 84px 100%, var(--tl-papel) 7px, transparent 7.5px),
			#fffdf9;
		box-shadow: inset 0 0 0 1px #efe6d4;
		color: var(--tl-tinta);
		text-decoration: none;
	}
	.historial .paso a {
		background:
			radial-gradient(circle at 84px 0, var(--tl-papel) 7px, transparent 7.5px),
			radial-gradient(circle at 84px 100%, var(--tl-papel) 7px, transparent 7.5px),
			#f8f2e6;
	}
	.historial .stub {
		display: flex;
		flex-direction: column;
		justify-content: center;
		padding: 0 12px;
		border-right: 2px dashed var(--tl-linea-2);
	}
	.historial .stub b {
		font-size: 13px;
		font-weight: 700;
	}
	.historial .stub span,
	.historial .medio span {
		font-size: 11px;
		color: var(--tl-sub);
	}
	.historial .medio {
		display: flex;
		flex-direction: column;
		justify-content: center;
		padding: 6px 12px;
		min-width: 0;
	}
	.historial .medio b {
		font-size: 14px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.historial .est {
		display: flex;
		align-items: center;
		padding-right: 10px;
	}
	.borrar-datos {
		width: 100%;
		max-width: 560px;
		display: flex;
		flex-direction: column;
		gap: 6px;
		padding: 0 4px;
	}
	.borrar-datos h2 {
		margin: 0;
		font-size: 11px;
		color: var(--tl-sub);
	}
	.pie {
		display: flex;
		gap: 8px;
		padding: 14px 20px 18px;
		border-top: 1px solid #f0e8d8;
	}
	.turno-para {
		flex: 1;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		height: 52px;
		border-radius: 999px;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		font-size: 16px;
		font-weight: 700;
		text-decoration: none;
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.45), 0 12px 24px -10px rgba(60, 204, 144, 0.8);
	}
	@media (max-width: 859px) {
		.ficha {
			border-radius: 20px;
		}
		.historial a {
			grid-template-columns: 84px minmax(0, 1fr);
		}
		.historial .stub {
			grid-row: span 2;
		}
		.historial .est {
			grid-column: 2;
			padding: 0 12px 8px;
		}
	}
</style>
