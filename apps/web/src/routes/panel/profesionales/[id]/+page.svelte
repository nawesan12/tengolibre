<script lang="ts">
	import '#lib/gestion/gestion.css';
	import { enhance } from '$app/forms';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { DIAS, ROLES, dia as hoy, hora } from '#lib/formato.ts';
	import { FRANJAS_POR_DIA, SEMANA, esHora, franjasDe, type Franja } from '#lib/configuracion/horario.ts';
	import { confirmar, sinReset } from '#lib/configuracion/formularios.ts';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';
	import ListaEquipo from '#lib/gestion/equipo/ListaEquipo.svelte';
	import GrillaSemana from '#lib/gestion/equipo/GrillaSemana.svelte';
	import { MOTIVOS_LIBRES, NOMBRE_MOTIVO } from '#lib/gestion/api-equipo.ts';
	import {
		DIA_CORTO,
		diasAgrupados,
		diasEntre,
		fechaTalon,
		horaCorta,
		horasSemana,
		horasTexto,
		minutos,
		rango,
		rangoTalon
	} from '#lib/gestion/equipo/semana.ts';

	let { data, form } = $props();
	const p = $derived(data.profesional);
	const rol = $derived(data.negocio.rol);
	const esGestion = $derived(['duenio', 'encargado'].includes(rol));
	const esProfesional = $derived(rol === 'profesional');
	const nombreCorto = $derived(p.nombre.split(/\s+/)[0]);
	const quien = $derived(data.miembros?.find((m) => m.id === p.miembroId));

	// El horario que se edita: arranca del guardado y la grilla de arriba lo muestra al instante.
	type DiaEd = { abre: boolean; f: { desde: string; hasta: string }[] };
	const edicion = $derived.by(() => {
		const e = $state(
			Object.fromEntries(
				SEMANA.map((d) => {
					const fr = franjasDe(p.horario, d);
					return [
						d,
						{
							abre: fr.length > 0,
							f: Array.from({ length: FRANJAS_POR_DIA }, (_, i) => ({
								desde: fr[i]?.desde.slice(0, 5) ?? (i === 0 ? '09:00' : ''),
								hasta: fr[i]?.hasta.slice(0, 5) ?? (i === 0 ? '18:00' : '')
							}))
						}
					];
				})
			) as Record<number, DiaEd>
		);
		return e;
	});
	const vista = $derived<Franja[]>(
		SEMANA.flatMap((d) =>
			edicion[d].abre
				? edicion[d].f.filter((x) => esHora(x.desde) && esHora(x.hasta) && x.desde < x.hasta).map((x) => ({ dia: d, ...x }))
				: []
		)
	);
	const cambiado = $derived(JSON.stringify(vista.map(rango)) !== JSON.stringify(p.horario.map(rango)) || vista.length !== p.horario.length);

	// Bloqueos, semanales, excepciones y pedidos como talones, ordenados por fecha.
	type Talon = {
		clave: string;
		f: string;
		t: string;
		s: string;
		tono: 'ambar' | 'crema' | 'verde';
		orden: string;
		sacar?: { accion: string; campos: [string, string][]; etiqueta: string };
		estado?: string;
		pedido?: string;
	};
	const talones = $derived.by(() => {
		const out: Talon[] = [];
		// Los bloqueos de día entero con el mismo motivo, en días seguidos, van en un solo talón.
		const bl = [...data.bloqueos].sort((a, b) => a.fecha.localeCompare(b.fecha));
		for (let i = 0; i < bl.length; ) {
			const b = bl[i];
			let j = i;
			while (
				!b.desde &&
				j + 1 < bl.length &&
				!bl[j + 1].desde &&
				(bl[j + 1].motivo ?? '') === (b.motivo ?? '') &&
				diasEntre(bl[j].fecha, bl[j + 1].fecha).length === 2
			)
				j++;
			const grupo = bl.slice(i, j + 1);
			const ultimo = grupo[grupo.length - 1].fecha;
			out.push({
				clave: `b-${b.id}`,
				f: rangoTalon(b.fecha, ultimo),
				t: b.motivo || 'Bloqueado',
				s: `Puntual · ${b.desde ? `de ${horaCorta(b.desde)} a ${horaCorta(b.hasta ?? '')}` : grupo.length > 1 ? `${grupo.length} días` : 'todo el día'}`,
				tono: 'ambar',
				orden: b.fecha,
				sacar: {
					accion: '?/borrarBloqueo',
					campos: grupo.map((x) => ['bloqueo', x.id] as [string, string]),
					etiqueta: `Sacar el bloqueo ${rangoTalon(b.fecha, ultimo).toLowerCase()}`
				}
			});
			i = j + 1;
		}
		for (const s of data.semanales)
			out.push({
				clave: `s-${s.dia}-${s.hora}`,
				f: `${DIA_CORTO[s.dia]} ${s.hora}`,
				t: 'Bloqueado',
				s: `${DIAS[s.dia]} ${s.hora} · todas las semanas`,
				tono: 'crema',
				orden: `0-${SEMANA.indexOf(s.dia)}-${s.hora}`,
				sacar: { accion: '?/borrarSemanal', campos: [['dia', String(s.dia)], ['hora', s.hora]], etiqueta: `Sacar ${DIAS[s.dia]} ${s.hora}` }
			});
		for (const e of data.excepciones)
			out.push({
				clave: `e-${e.fecha}-${e.hora}`,
				f: `${fechaTalon(e.fecha)} ${e.hora}`,
				t: e.tipo === 'libre' ? 'Se puede reservar' : 'No se puede reservar',
				s: 'Excepción · sólo ese día',
				tono: e.tipo === 'libre' ? 'verde' : 'crema',
				orden: e.fecha,
				sacar: {
					accion: '?/borrarExcepcion',
					campos: [['fecha', e.fecha], ['hora', e.hora]],
					etiqueta: `Sacar la excepción del ${fechaTalon(e.fecha).toLowerCase()} ${e.hora}`
				}
			});
		for (const x of data.pedidos)
			if (x.estado === 'pendiente')
				out.push({
					clave: `p-${x.id}`,
					f: rangoTalon(x.desde, x.hasta),
					t: NOMBRE_MOTIVO[x.motivo] ?? 'Días libres',
					s:
						(esProfesional ? 'Pedido · esperando respuesta' : `Lo pidió ${nombreCorto}`) +
						(x.afectados > 0 ? ` · ${x.afectados} ${x.afectados === 1 ? 'turno' : 'turnos'} para mover` : ''),
					tono: 'ambar',
					orden: x.desde,
					estado: 'Pendiente',
					pedido: x.id
				});
		return out.sort((a, b) => a.orden.localeCompare(b.orden));
	});

	// Nuevo bloqueo: abierto al llegar recién creado o si se pidió.
	let bloqueando = $state(page.url.searchParams.has('nuevo') || page.url.searchParams.has('bloquear'));
	let tipo = $state<'puntual' | 'semanal' | 'especial'>('puntual');
	let fecha = $state('');
	let hastaDia = $state('');
	let entero = $state(true);
	let desdeH = $state('');
	let hastaH = $state('');
	// Qué turnos toca el bloqueo, antes de guardarlo (de los que ya cargó la página).
	const afectados = $derived.by(() => {
		if (!fecha) return [];
		const dias = new Set(diasEntre(fecha, hastaDia && hastaDia > fecha ? hastaDia : fecha));
		return data.proximos.filter((t) => {
			if (!dias.has(hoy(t.inicio))) return false;
			if (entero || !esHora(desdeH) || !esHora(hastaH)) return true;
			const m = minutos(hora(t.inicio));
			return m >= minutos(desdeH) && m < minutos(hastaH);
		});
	});
	const listaAfectados = (desde: string, hasta: string) =>
		`/panel/turnos?${new URLSearchParams({ desde, hasta: hasta || desde, profesional: p.id })}`;

	let pidiendo = $state(false);
	const pedidosPendientes = $derived(data.pedidos.filter((x) => x.estado === 'pendiente'));
	const agrupados = $derived(diasAgrupados(p.horario));
</script>

<svelte:head><title>{esProfesional ? 'Mi horario' : p.nombre} · Panel</title></svelte:head>

{#snippet talon(x: Talon)}
	<li class="g-talon {x.tono === 'crema' ? '' : x.tono}" style:--tx="92px">
		<span>{x.f}</span>
		<span><b>{x.t}</b><span>{x.s}</span></span>
		<span>
			{#if x.estado}<span class="pendiente"><span class="icono" aria-hidden="true">hourglass_top</span>{x.estado}</span>{/if}
			{#if x.pedido && esGestion}
				<form method="POST" action="?/resolverPedido" use:enhance>
					<input type="hidden" name="pedido" value={x.pedido} />
					<button class="g-accion" name="estado" value="aprobado">Aprobar</button>
					<button class="g-accion peligro" name="estado" value="rechazado">Rechazar</button>
				</form>
			{:else if x.pedido && esProfesional}
				<form method="POST" action="?/resolverPedido" use:enhance>
					<input type="hidden" name="pedido" value={x.pedido} />
					<button class="g-accion peligro" name="estado" value="cancelado">Cancelar</button>
				</form>
			{/if}
			{#if x.sacar && !esProfesional}
				<form method="POST" action={x.sacar.accion} use:enhance>
					{#each x.sacar.campos as [k, v] (`${k}${v}`)}<input type="hidden" name={k} value={v} />{/each}
					<button class="sacar" aria-label={x.sacar.etiqueta} title="Sacar"><span class="icono" aria-hidden="true">close</span></button>
				</form>
			{/if}
		</span>
	</li>
{/snippet}

{#if esProfesional}
	<!-- «Mi horario» (handoff v2, Pantallas que faltaban 1m): su semana y sus bloqueos, sin tocar el resto. -->
	<div class="g-pagina mio">
		<div class="g-col cabeza">
			<span class="g-kicker">{p.nombre}</span>
			<h1 class="g-titulo">Mi horario</h1>
		</div>
		{#if agrupados.length === 0}
			<div class="g-bloque"><p class="g-nota">Todavía no tenés horario cargado. Pedíselo a quien maneja el&nbsp;negocio.</p></div>
		{/if}
		<ul class="dias">
			{#each agrupados as g (g.nombre)}
				<li class="g-bloque dia">
					<span class="g-fila-texto"><b>{g.nombre}</b><span class="num">{g.texto}</span></span>
					<b class="num">{horasTexto(g.horas)} H</b>
				</li>
			{/each}
		</ul>
		{#if talones.length > 0}
			<section aria-labelledby="t-mios" class="g-col">
				<h2 id="t-mios" class="g-rotulo">Bloqueos y días libres</h2>
				<ul class="talones">
					{#each talones as x (x.clave)}{@render talon(x)}{/each}
				</ul>
			</section>
		{/if}
		<Mensaje {form} que="pedidos" />
		{#if pidiendo}
			<form method="POST" action="?/pedirLibres" use:enhance={() => async ({ update, result }) => {
				await update();
				if (result.type === 'success') pidiendo = false;
			}} class="g-bloque">
				<h2 class="g-bloque-titulo">Pedir días libres</h2>
				<div class="g-form">
					<label class="g-campo">Desde <input type="date" name="desde" required min={hoy()} /></label>
					<label class="g-campo">Hasta <input type="date" name="hasta" min={hoy()} /></label>
					<label class="g-campo">
						Motivo
						<select name="motivo">
							{#each MOTIVOS_LIBRES as m (m)}<option value={m}>{NOMBRE_MOTIVO[m]}</option>{/each}
						</select>
					</label>
					<label class="g-campo">Nota (lo ve sólo el equipo) <input name="nota" maxlength="300" /></label>
				</div>
				<button class="g-cta">Mandar el pedido</button>
			</form>
		{:else}
			<button type="button" class="g-cta" onclick={() => (pidiendo = true)}>Pedir días libres</button>
		{/if}
	</div>
{:else}
	<div class="g-pagina prof">
		<div class="cabeza">
			<a class="g-kicker volver" href="/panel/equipo"><span class="icono" aria-hidden="true">arrow_back</span>Equipo y horarios</a>
			<div class="g-cabeza">
				<h1 class="g-titulo">{p.nombre}</h1>
				{#if !p.activo}<span class="g-pildora ambar"><span class="icono" aria-hidden="true">pause_circle</span>Pausado</span>{/if}
			</div>
			{#if page.url.searchParams.has('nuevo')}
				<p class="g-aviso" role="status">
					<span class="icono" aria-hidden="true">check_circle</span>
					<span>Listo, ya está cargado. Ahora poné los días y horarios en que&nbsp;atiende.</span>
				</p>
			{/if}
		</div>

		<aside class="izquierda" aria-label="Equipo">
			<ListaEquipo profesionales={data.profesionales} actual={p.id}>
				{#if esGestion}
					<a class="sumar" href="/panel/equipo?sumar=1#sumar"><span class="icono" aria-hidden="true">person_add</span>Sumar profesional</a>
				{/if}
			</ListaEquipo>
		</aside>

		<div class="derecha g-col">
			<section class="g-bloque" aria-labelledby="t-horario">
				<div class="g-bloque-cabeza">
					<h2 id="t-horario" class="g-bloque-titulo grande">Horario de {nombreCorto}</h2>
					<span class="g-meta">{horasTexto(horasSemana(vista))} h por semana</span>
				</div>
				<GrillaSemana horario={vista} />
				<Mensaje {form} que="horario" />
				<form method="POST" action="?/horario" use:enhance={sinReset} class="editar">
					<p class="g-nota">Tocá el día para abrirlo o cerrarlo y cambiá las horas: la grilla se acomoda sola. Si corta al mediodía, cargá la tarde como segunda&nbsp;franja.</p>
					{#each SEMANA as d (d)}
						{@const e = edicion[d]}
						<div class="dia-ed">
							<label class="g-chip dia-chip">
								<input type="checkbox" name={`abre-${d}`} bind:checked={e.abre} />
								<span class="icono" aria-hidden="true">{e.abre ? 'check' : 'add'}</span>
								<span aria-hidden="true">{DIA_CORTO[d]}</span>
								<span class="visualmente-oculto">Atiende el {DIAS[d].toLowerCase()}</span>
							</label>
							{#if e.abre}
								<div class="franjas">
									{#each e.f as fr, i (i)}
										<span class="franja-ed">
											<label>
												<span class="visualmente-oculto">{i === 0 ? 'Desde' : 'Vuelve a las'} ({DIAS[d]})</span>
												<input type="time" name={`desde-${d}-${i}`} bind:value={fr.desde} step="300" />
											</label>
											<span aria-hidden="true">a</span>
											<label>
												<span class="visualmente-oculto">Hasta ({DIAS[d]}{i === 1 ? ', tarde' : ''})</span>
												<input type="time" name={`hasta-${d}-${i}`} bind:value={fr.hasta} step="300" />
											</label>
										</span>
									{/each}
								</div>
							{:else}
								<span class="cerrado">No atiende</span>
							{/if}
						</div>
					{/each}
					<div class="guardar">
						{#if cambiado}<span class="g-estado espera"><span class="icono" aria-hidden="true">edit</span>Cambios sin guardar</span>{/if}
						<button class="g-boton menta">Guardar horario</button>
					</div>
				</form>
			</section>

			<div class="g-dos">
				<section class="g-bloque" aria-labelledby="t-bloqueos">
					<h2 id="t-bloqueos" class="g-bloque-titulo">Bloqueos y excepciones</h2>
					{#if !bloqueando}
						<Mensaje {form} que="bloqueos" />
						<Mensaje {form} que="semanales" />
						<Mensaje {form} que="excepciones" />
					{/if}
					<Mensaje {form} que="pedidos" />
					{#if talones.length === 0}
						<p class="g-nota">No hay días ni horarios bloqueados de acá en&nbsp;adelante.</p>
					{:else}
						<ul class="talones">
							{#each talones as x (x.clave)}{@render talon(x)}{/each}
						</ul>
					{/if}
					<button type="button" class="g-accion" aria-expanded={bloqueando} onclick={() => (bloqueando = !bloqueando)}>
						{bloqueando ? 'Cerrar el bloqueo' : '+ Bloquear días u horas'}
					</button>
				</section>

				<section class="g-bloque" aria-labelledby="t-hace">
					<h2 id="t-hace" class="g-bloque-titulo">Qué hace</h2>
					<Mensaje {form} que="servicios" />
					{#if data.servicios.length === 0}
						<p class="g-nota">Todavía no hay <a href="/panel/servicios">servicios</a> cargados.</p>
					{:else if esGestion}
						<form method="POST" action="?/servicios" use:enhance={sinReset} class="g-chips">
							{#each data.servicios as s (s.id)}
								<label class="g-chip hace">
									<input
										type="checkbox"
										name="servicios"
										value={s.id}
										checked={p.servicios.includes(s.id)}
										onchange={(e) => e.currentTarget.form?.requestSubmit()}
									/>
									{#if !p.servicios.includes(s.id)}<span aria-hidden="true">+</span>{/if}{s.nombre}{s.activo ? '' : ' (pausado)'}
								</label>
							{/each}
							<noscript><button class="g-boton chico">Guardar servicios</button></noscript>
						</form>
					{:else}
						<ul class="g-chips sin-lista">
							{#each data.servicios.filter((s) => p.servicios.includes(s.id)) as s (s.id)}<li class="g-chip on">{s.nombre}</li>{/each}
						</ul>
					{/if}
					<p class="vinculo">
						<span class="icono" aria-hidden="true">{p.miembroId ? 'link' : 'link_off'}</span>
						{#if quien}
							<span>Vinculada a <b class="num">{quien.email}</b> · {quien.rol === 'profesional' ? 've sólo sus turnos' : `entra como ${(ROLES[quien.rol] ?? quien.rol).toLowerCase()}`}</span>
						{:else if p.miembroId}
							<span>Entra al panel con su cuenta.</span>
						{:else}
							<span>Sin cuenta del panel. Se vincula en&nbsp;Datos.</span>
						{/if}
					</p>
				</section>
			</div>

			{#if bloqueando}
				<!-- Nuevo bloqueo (handoff v2, Pantallas que faltaban 1f). -->
				<section class="g-bloque nuevo" aria-labelledby="t-nuevo">
					<span class="g-kicker">{p.nombre}</span>
					<h2 id="t-nuevo" class="g-titulo-2">Nuevo bloqueo</h2>
					<div class="g-chips" role="radiogroup" aria-label="Tipo de bloqueo">
						<label class="g-chip"><input type="radio" name="tipo-bloqueo" value="puntual" bind:group={tipo} />Puntual</label>
						<label class="g-chip"><input type="radio" name="tipo-bloqueo" value="semanal" bind:group={tipo} />Todas las semanas</label>
						<label class="g-chip"><input type="radio" name="tipo-bloqueo" value="especial" bind:group={tipo} />Día especial</label>
					</div>
					<Mensaje {form} que="bloqueos" />
					<Mensaje {form} que="semanales" />
					<Mensaje {form} que="excepciones" />

					{#if tipo === 'puntual'}
						<form
							method="POST"
							action="?/bloqueo"
							class="g-col"
							use:enhance={() => {
								const ir = afectados.length > 0 ? listaAfectados(fecha, hastaDia) : null;
								return async ({ update, result }) => {
									await update();
									if (result.type === 'success') {
										fecha = '';
										hastaDia = '';
										if (ir) await goto(ir);
									}
								};
							}}
						>
							<div class="g-form">
								<label class="g-campo">Fecha <input type="date" name="fecha" required min={hoy()} bind:value={fecha} /></label>
								<label class="g-campo">Hasta el día (opcional) <input type="date" name="hastaDia" min={fecha || hoy()} bind:value={hastaDia} /></label>
								<label class="g-check g-ancho"><input type="checkbox" name="entero" bind:checked={entero} /> Todo el día</label>
								{#if !entero}
									<label class="g-campo">Desde <input type="time" name="desde" required step="300" bind:value={desdeH} /></label>
									<label class="g-campo">Hasta <input type="time" name="hasta" required step="300" bind:value={hastaH} /></label>
								{/if}
								<label class="g-campo g-ancho">Motivo (opcional) <input name="motivo" maxlength="120" placeholder="Vacaciones" /></label>
							</div>
							{#if afectados.length > 0}
								<p class="g-aviso ambar">
									<span class="icono" aria-hidden="true">event_busy</span>
									<span>
										{afectados.length === 1 ? '1 turno cae' : `${afectados.length} turnos caen`} en {entero ? 'esos días' : 'ese horario'}
										({afectados.slice(0, 3).map((t) => `${t.cliente} ${hora(t.inicio)}`).join(', ')}{afectados.length > 3 ? '…' : ''}). Al guardar, te los dejamos en una lista
										para&nbsp;reprogramar.
									</span>
								</p>
							{:else if fecha}
								<p class="g-aviso">
									<span class="icono" aria-hidden="true">check_circle</span>
									<span>No hay turnos en {entero ? 'esos días' : 'ese horario'}: no hay que mover a&nbsp;nadie.</span>
								</p>
							{/if}
							<button class="g-cta">{afectados.length > 0 ? `Bloquear y ver ${afectados.length === 1 ? 'el turno' : `los ${afectados.length} turnos`}` : 'Bloquear'}</button>
						</form>
					{:else if tipo === 'semanal'}
						<form method="POST" action="?/semanal" use:enhance class="g-col">
							<p class="g-nota">Un horario que nunca se da, como el almuerzo de los martes o la&nbsp;facultad.</p>
							<div class="g-form">
								<label class="g-campo">
									Día
									<select name="dia">{#each SEMANA as d (d)}<option value={d}>{DIAS[d]}</option>{/each}</select>
								</label>
								<label class="g-campo">Hora <input type="time" name="hora" required step="300" /></label>
							</div>
							<button class="g-cta">Bloquear ese horario</button>
						</form>
					{:else}
						<form method="POST" action="?/excepcion" use:enhance class="g-col">
							<p class="g-nota">Bloquear un horario de una fecha, o abrir ese día uno de los bloqueados todas las&nbsp;semanas.</p>
							<div class="g-form">
								<label class="g-campo">Fecha <input type="date" name="fecha" required min={hoy()} /></label>
								<label class="g-campo">Hora <input type="time" name="hora" required step="300" /></label>
								<label class="g-campo g-ancho">
									Qué pasa
									<select name="tipo">
										<option value="bloqueado">No se puede reservar</option>
										<option value="libre">Se puede reservar</option>
									</select>
								</label>
							</div>
							<button class="g-cta">Guardar excepción</button>
						</form>
					{/if}
				</section>
			{/if}

			{#if esGestion}
				<section class="g-bloque" aria-labelledby="t-datos">
					<h2 id="t-datos" class="g-bloque-titulo">Datos</h2>
					<Mensaje {form} que="datos" />
					<form method="POST" action="?/datos" use:enhance={sinReset} class="g-form">
						<label class="g-campo">Nombre <input name="nombre" value={p.nombre} required minlength="2" maxlength="60" /></label>
						<label class="g-campo">Especialidad <input name="especialidad" value={p.especialidad} maxlength="80" /></label>
						<label class="g-campo">Email <input name="email" type="email" value={p.email} maxlength="120" /></label>
						<label class="g-campo">Teléfono <input name="telefono" type="tel" value={p.telefono} maxlength="30" /></label>
						<label class="g-campo">Orden en la página <input name="orden" type="number" min="0" max="999" value={p.orden} /></label>
						{#if data.miembros}
							<label class="g-campo">
								Su cuenta del panel
								<select name="miembroId">
									<option value="">Sin cuenta</option>
									{#each data.miembros as m (m.id)}
										<option value={m.id} selected={m.id === p.miembroId}>{m.nombre || m.email} ({ROLES[m.rol] ?? m.rol})</option>
									{/each}
								</select>
							</label>
						{/if}
						<label class="g-campo g-ancho">Sobre su trabajo <textarea name="bio" rows="3" maxlength="1000">{p.bio}</textarea></label>
						<label class="g-check g-ancho"><input type="checkbox" name="activo" checked={p.activo} /> Se le pueden reservar turnos</label>
						<div class="g-ancho acciones">
							<button class="g-boton menta">Guardar datos</button>
						</div>
					</form>
					<form method="POST" action="?/borrar" use:enhance={confirmar(`¿Borrar a ${p.nombre}? Si tiene turnos, mejor pausalo.`)}>
						<button class="g-accion peligro"><span class="icono" aria-hidden="true">delete</span>Borrar profesional</button>
					</form>
				</section>
			{/if}
		</div>
	</div>
{/if}

<style>
	.prof {
		display: grid;
		grid-template-columns: 300px minmax(0, 1fr);
		gap: 16px 24px;
		align-items: start;
	}
	.cabeza {
		grid-column: 1 / -1;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.volver {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		min-height: 36px;
		text-decoration: none;
		width: fit-content;
	}
	.volver .icono {
		font-size: 16px;
	}
	.izquierda {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.sumar {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 6px;
		min-height: 48px;
		border-radius: 16px;
		box-shadow: inset 0 0 0 1.5px #d8ccb4;
		font-size: 14px;
		font-weight: 700;
		color: inherit;
		text-decoration: none;
	}
	.sumar .icono {
		font-size: 20px;
	}
	.grande {
		font-size: 17px;
	}

	/* Edición del horario */
	.editar {
		display: flex;
		flex-direction: column;
		gap: 8px;
		padding-top: 12px;
		border-top: 1px solid #f0e8d8;
	}
	.dia-ed {
		display: grid;
		grid-template-columns: 92px minmax(0, 1fr);
		gap: 12px;
		align-items: center;
		min-height: 48px;
	}
	.dia-chip {
		justify-content: flex-start;
		font-variant-numeric: tabular-nums;
	}
	.dia-chip .icono {
		font-size: 16px;
	}
	.franjas {
		display: flex;
		flex-wrap: wrap;
		gap: 8px 16px;
	}
	.franja-ed {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		font-size: 12px;
		color: var(--tl-sub);
	}
	.franja-ed input {
		height: 40px;
		padding: 0 8px;
		border: 0;
		border-radius: 10px;
		background: var(--tl-crema);
		color: var(--tl-tinta);
		font: 700 14px var(--tl-font);
		font-variant-numeric: tabular-nums;
	}
	.franja-ed input:focus {
		outline: none;
		box-shadow: inset 0 0 0 2px var(--tl-menta);
	}
	.cerrado {
		font-size: 13px;
		color: var(--tl-sub);
	}
	.guardar {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: flex-end;
		gap: 12px;
		padding-top: 4px;
	}

	/* Talones de bloqueos */
	.talones {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.talones .g-talon {
		min-height: 48px;
		border-radius: 12px;
	}
	.talones .g-talon > :first-child {
		font-size: 12px;
	}
	.talones .g-talon > :nth-child(2) > b {
		font-size: 13px;
	}
	.talones .g-talon > :nth-child(3) {
		padding-right: 6px;
		gap: 4px;
	}
	.talones form {
		display: flex;
		gap: 8px;
	}
	.sacar {
		width: 36px;
		height: 36px;
		border: 0;
		border-radius: 50%;
		background: transparent;
		color: var(--tl-sub);
		display: grid;
		place-items: center;
		cursor: pointer;
	}
	.sacar:hover {
		background: rgba(15, 35, 40, 0.06);
		color: var(--tl-tinta);
	}
	.sacar .icono {
		font-size: 18px;
	}
	.pendiente {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		padding-right: 6px;
		font: 700 11px var(--tl-font);
		letter-spacing: 0.04em;
		text-transform: uppercase;
		color: #5c3800;
	}
	.pendiente .icono {
		font-size: 14px;
	}

	/* Qué hace */
	.hace input:not(:checked) ~ * {
		color: var(--tl-sub);
	}
	.hace:not(:has(input:checked)) {
		color: var(--tl-sub);
	}
	.sin-lista {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.vinculo {
		margin: 0;
		display: flex;
		align-items: center;
		gap: 8px;
		padding-top: 10px;
		border-top: 1px solid #f0e8d8;
		font-size: 13px;
	}
	.vinculo .icono {
		font-size: 18px;
		color: var(--tl-menta-texto);
	}
	.vinculo b {
		font-weight: 400;
		overflow-wrap: anywhere;
	}

	.nuevo {
		background-color: var(--tl-crema);
		background-image: radial-gradient(rgba(24, 60, 68, 0.08) 1px, transparent 1.3px);
		background-size: 18px 18px;
		box-shadow: inset 0 0 0 1.5px #e2d6bf;
	}
	.nuevo .g-chip:not(:has(input:checked)) {
		background: var(--tl-papel);
	}
	.acciones {
		display: flex;
		justify-content: flex-end;
	}

	/* Mi horario */
	.mio {
		max-width: 560px;
	}
	.dias {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.dias .dia {
		flex-direction: row;
		align-items: center;
		padding: 12px 14px;
		border-radius: 14px;
	}
	.dias .num {
		font-variant-numeric: tabular-nums;
	}

	@media (max-width: 1100px) {
		.prof {
			grid-template-columns: minmax(0, 1fr);
		}
		/* En el teléfono primero el horario; el resto del equipo, al final. */
		.izquierda {
			order: 3;
		}
	}
	@media (max-width: 640px) {
		.dia-ed {
			grid-template-columns: minmax(0, 1fr);
			gap: 6px;
		}
		.dia-chip {
			width: fit-content;
		}
		.franja-ed input {
			height: 48px;
			font-size: 16px;
		}
		.sacar {
			width: 44px;
			height: 44px;
		}
		.guardar .g-boton {
			width: 100%;
			min-height: 52px;
		}
	}
</style>
