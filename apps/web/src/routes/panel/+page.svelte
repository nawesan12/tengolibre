<!--
	Inicio (handoff «Panel v2»): saludo con las horas libres de hoy, lo que necesita atención, lo que
	sigue (talones con la línea de ahora y un hueco), los huecos de hoy y los primeros pasos.
-->
<script lang="ts">
	import { enhance } from '$app/forms';
	import { onMount } from 'svelte';
	import Icono from '#lib/ui/Icono.svelte';
	import { hora } from '#lib/formato.ts';
	import EstadoTurno from '#lib/operacion/EstadoTurno.svelte';
	import LineaAhora from '#lib/operacion/LineaAhora.svelte';
	import { estadoVisible } from '#lib/operacion/estados.ts';
	import { minutosDelDia } from '#lib/operacion/fechas.ts';
	import { armarDia, cargarEn, duracion, hhmm, libresTexto, minutosAhora, primerNombre, profesionalesDelDia, rangoDelDia } from '#lib/panel/dia.ts';

	let { data, form } = $props();

	let ahoraMs = $state(Date.now());
	onMount(() => {
		const r = setInterval(() => (ahoraMs = Date.now()), 30_000);
		return () => clearInterval(r);
	});
	const ahora = $derived(minutosAhora(new Date(ahoraMs)));
	const rol = $derived(data.negocio.rol);
	const esProfesional = $derived(rol === 'profesional');
	const turnos = $derived(data.hoy.turnos);
	const propio = $derived((data.negocio as { profesionalId?: string | null }).profesionalId ?? null);
	const profes = $derived(profesionalesDelDia(data.hoy.profesionales, turnos, rol, propio));
	const dias = $derived(armarDia(profes, turnos, data.hoy.fecha, ahora));
	const conHorario = $derived(dias.filter((d) => d.franjas.length));
	const libres = $derived(dias.reduce((s, d) => s + d.libres, 0));
	const huecos = $derived(dias.flatMap((d) => d.huecos).sort((a, b) => a.desde - b.desde));
	const vivos = $derived(turnos.filter((t) => t.estado !== 'cancelado'));

	const saludo = $derived.by(() => {
		const h = Math.floor(ahora / 60);
		const s = h < 13 ? 'Buen día' : h < 20 ? 'Buenas tardes' : 'Buenas noches';
		const n = primerNombre(data.persona.nombre ?? '');
		return n ? `${s}, ${n}.` : `${s}.`;
	});

	// Lo que sigue: los próximos cinco y, en el medio, el primer hueco largo.
	type Fila = { tipo: 'turno'; t: (typeof turnos)[number] } | { tipo: 'hueco'; h: (typeof huecos)[number] };
	const sigue = $derived.by<Fila[]>(() => {
		const prox = vivos.filter((t) => Date.parse(t.fin) > ahoraMs).slice(0, 5);
		const filas: Fila[] = prox.map((t) => ({ tipo: 'turno', t }));
		const largo = huecos.find((h) => h.hasta - h.desde >= 60);
		if (largo) {
			let i = filas.findIndex((f) => f.tipo === 'turno' && minutosDelDia(f.t.inicio) > largo.desde);
			if (i < 0) i = filas.length;
			filas.splice(Math.min(i, 4), 0, { tipo: 'hueco', h: largo });
		}
		return filas;
	});

	// Lo que necesita atención (sólo con datos reales; si no hay nada, no se muestra).
	type Atencion = { num: string; color: string; titulo: string; sub: string; boton: string; href?: string; recordar?: string; hueco?: boolean };
	const atencion = $derived.by<Atencion[]>(() => {
		if (esProfesional) return [];
		const out: Atencion[] = [];
		const pago = turnos
			.filter((t) => t.estado === 'pendiente_pago' && t.venceEn && Date.parse(t.venceEn) > ahoraMs)
			.sort((a, b) => Date.parse(a.venceEn!) - Date.parse(b.venceEn!))[0];
		if (pago) {
			const min = Math.max(1, Math.ceil((Date.parse(pago.venceEn!) - ahoraMs) / 60000));
			out.push({
				num: min >= 60 ? duracion(min) : `${min}′`,
				color: '#7a4a00',
				titulo: `para que se libere el turno de ${primerNombre(pago.cliente.nombre)}`,
				sub: `No pagó la seña de las ${hora(pago.inicio)}`,
				boton: 'Recordarle',
				recordar: pago.id
			});
		}
		const liberado = turnos.find((t) => {
			if (t.estado !== 'cancelado' || Date.parse(t.inicio) <= ahoraMs || !t.profesionalId) return false;
			const d = dias.find((x) => x.id === t.profesionalId);
			const m = minutosDelDia(t.inicio);
			return !!d?.huecos.some((h) => h.desde <= m && h.hasta > m);
		});
		if (liberado) {
			const esperan = data.espera.filter(
				(a) => !a.avisadoEn && a.desde <= data.hoy.fecha && a.hasta >= data.hoy.fecha && (!a.profesionalId || a.profesionalId === liberado.profesionalId)
			).length;
			out.push({
				num: hora(liberado.inicio),
				color: '#17805a',
				titulo: `se liberó con ${primerNombre(liberado.profesional)}`,
				sub: esperan === 0 ? 'Nadie en la lista para hoy' : esperan === 1 ? '1 persona espera ese día' : `${esperan} personas esperan ese día`,
				boton: 'Ver el hueco',
				href: `/panel/turnos?profesional=${liberado.profesionalId}`,
				hueco: true
			});
		}
		const sinConf = vivos.filter((t) => t.estado === 'pendiente' || t.estado === 'pendiente_pago').length;
		if (sinConf)
			out.push({
				num: String(sinConf),
				color: '#0f2328',
				titulo: sinConf === 1 ? 'turno de hoy sin confirmar' : 'turnos de hoy sin confirmar',
				sub: 'Al confirmar sale el recordatorio',
				boton: 'Ver',
				href: '/panel/turnos?estado=sin'
			});
		if (data.resena)
			out.push({
				num: `${data.resena.puntaje}★`,
				color: '#0f2328',
				titulo: `reseña de ${primerNombre(data.resena.nombre)} sin responder`,
				sub: data.resena.comentario ? `«${data.resena.comentario.length > 60 ? data.resena.comentario.slice(0, 58) + '…' : data.resena.comentario}»` : 'Sin comentario',
				boton: 'Responder',
				href: '/panel/resenas'
			});
		return out;
	});

	// El anillo del teléfono: una vuelta por profesional, del primer al último horario del día.
	const rango = $derived(rangoDelDia(dias));
	function anillo(d: (typeof dias)[number]) {
		const [a, b] = rango;
		const deg = (m: number) => (((Math.min(Math.max(m, a), b) - a) / (b - a)) * 360).toFixed(2);
		const puntos = new Set([a, b, Math.min(Math.max(ahora, a), b)]);
		for (const x of [...d.franjas, ...d.ocupado]) {
			puntos.add(Math.min(Math.max(x.desde, a), b));
			puntos.add(Math.min(Math.max(x.hasta, a), b));
		}
		const p = [...puntos].sort((x, y) => x - y);
		const tramos: string[] = [];
		for (let i = 0; i < p.length - 1; i++) {
			const medio = (p[i] + p[i + 1]) / 2;
			const trabaja = d.franjas.some((f) => f.desde <= medio && f.hasta > medio);
			const ocupado = d.ocupado.some((o) => o.desde <= medio && o.hasta > medio);
			const paso = medio < ahora;
			const c = ocupado
				? paso
					? 'rgba(255,255,255,.1)'
					: 'rgba(255,255,255,.34)'
				: trabaja
					? paso
						? 'rgba(60,204,144,.22)'
						: '#3ccc90'
					: 'rgba(255,255,255,.05)';
			tramos.push(`${c} ${deg(p[i])}deg ${deg(p[i + 1])}deg`);
		}
		return `conic-gradient(${tramos.join(',')})`;
	}
	const aguja = $derived(`${((Math.min(Math.max(ahora, rango[0]), rango[1]) - rango[0]) / (rango[1] - rango[0])) * 360}deg`);
	const nombresHoy = $derived.by(() => {
		const n = conHorario.map((d) => primerNombre(d.nombre));
		return n.length <= 1 ? n.join('') : `${n.slice(0, -1).join(', ')} y ${n.at(-1)}`;
	});

	// Primeros pasos.
	const pasos = $derived(data.arranque.pasos);
	const listos = $derived(pasos.filter((p) => p.listo).length);
	const DESTINO: Record<string, { href: string; icono: string; accion: string }> = {
		servicios: { href: '/panel/servicios', icono: 'spa', accion: 'Cargar' },
		profesionales: { href: '/panel/equipo', icono: 'badge', accion: 'Sumar' },
		horarios: { href: '/panel/equipo', icono: 'schedule', accion: 'Poner' },
		asignaciones: { href: '/panel/equipo', icono: 'assignment_ind', accion: 'Elegir' },
		perfil: { href: '/panel/mi-web', icono: 'edit_note', accion: 'Escribir' },
		contacto: { href: '/panel/configuracion', icono: 'call', accion: 'Poner' },
		ubicacion: { href: '/panel/configuracion', icono: 'location_on', accion: 'Poner' },
		logo: { href: '/panel/mi-web', icono: 'image', accion: 'Subir' },
		fotos: { href: '/panel/mi-web', icono: 'photo_library', accion: 'Sumar' }
	};
	const faltan = $derived(pasos.filter((p) => !p.listo).slice(0, 3));
	const verChecklist = $derived(data.gestiona && (!data.arranque.completo || data.mpConectado === false) && !data.acceso?.bloqueado);
	let copiado = $state(false);
	async function copiarLink() {
		try {
			await navigator.clipboard.writeText(`${location.origin}/${data.negocio.slug}`);
			copiado = true;
			setTimeout(() => (copiado = false), 2500);
		} catch {
			copiado = false;
		}
	}
</script>

<svelte:head><title>Inicio · Panel</title></svelte:head>

{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}
{#if form && 'ok' in form && form.ok}<p class="aviso" role="status">Listo, le mandamos el recordatorio de la&nbsp;seña.</p>{/if}

{#if conHorario.length}
	<div class="anillo-tel tl-petroleo">
		<span class="anillos" aria-hidden="true">
			{#each conHorario.slice(0, 4) as d, i (d.id)}
				<span class="vuelta" style:inset={`${i * 13}px`} style:background={anillo(d)}></span>
			{/each}
			<span class="aguja" style:transform={`rotate(${aguja})`}></span>
		</span>
		<span class="anillo-txt">
			<span class="tl-rotulo chico-r">Hoy quedan</span>
			<b class="tl-num">{libresTexto(libres)}</b>
			<span class="entre">libres entre {nombresHoy}</span>
			<a class="ver-huecos" href="/panel/turnos?huecos=1">Ver huecos</a>
		</span>
	</div>
{/if}

<h1 class="saludo">
	{saludo}
	{#if conHorario.length === 0}
		Hoy no atiende&nbsp;nadie.
	{:else if libres === 0}
		Hoy no quedan&nbsp;huecos.
	{:else}
		{esProfesional ? 'Hoy tenés' : 'Hoy quedan'} <span class="resalte tl-num">{libresTexto(libres)} libres.</span>
	{/if}
</h1>

{#if atencion.length}
	<div class="atencion">
		{#each atencion as a (a.titulo)}
			{#if a.recordar}
				<form method="POST" action="?/recordar" use:enhance class="tarjeta-a">
					<input type="hidden" name="id" value={a.recordar} />
					<button class="card">
						<span class="num tl-num" style:color={a.color}>{a.num}</span>
						<b>{a.titulo}</b>
						<span class="sub">{a.sub}</span>
						<span class="ir">{a.boton} →</span>
					</button>
				</form>
			{:else}
				<a class="card" class:rayada={a.hueco} href={a.href}>
					<span class="num tl-num" style:color={a.color}>{a.num}</span>
					<b>{a.titulo}</b>
					<span class="sub">{a.sub}</span>
					<span class="ir">{a.boton} →</span>
				</a>
			{/if}
		{/each}
	</div>
{/if}

<div class="dos">
	<section class="sigue" aria-labelledby="t-sigue">
		<div class="cabeza">
			<h2 id="t-sigue">Lo que sigue</h2>
			<a href="/panel/turnos">Ver {vivos.length === 1 ? 'el turno' : `los ${vivos.length}`} →</a>
		</div>
		<LineaAhora hora={hhmm(ahora)} />
		{#each sigue as f (f.tipo === 'turno' ? f.t.id : `h${f.h.profesionalId}${f.h.desde}`)}
			{#if f.tipo === 'turno'}
				{@const t = f.t}
				{@const est = estadoVisible(t.estado, t.inicio, t.fin, ahoraMs)}
				<a class="talon-chico" class:en-curso={est === 'en_curso'} href={`/panel/turnos/${t.id}`}>
					<span class="stub tl-num"><b>{hora(t.inicio)}</b><span>{hora(t.fin)}</span></span>
					<span class="medio"><b>{t.cliente.nombre}</b><span>{t.servicio}{t.profesional ? ` · ${primerNombre(t.profesional)}` : ''}</span></span>
					<span class="est"><EstadoTurno estado={est} /></span>
				</a>
			{:else}
				<div class="hueco-fila">
					<span class="rango tl-num">{hhmm(f.h.desde)} – {hhmm(f.h.hasta)}</span>
					<b>{primerNombre(f.h.profesional)} libre {duracion(f.h.hasta - f.h.desde)}</b>
					<a href={cargarEn(f.h.profesionalId, data.hoy.fecha, f.h.desde)}>Cargar acá</a>
				</div>
			{/if}
		{:else}
			<div class="vacio">
				<Icono nombre="event_available" tam={28} />
				<b>{vivos.length ? 'Ya no quedan turnos por hoy' : 'Todavía no hay turnos hoy'}</b>
				<span>{vivos.length ? 'Mirá mañana en Turnos o cargá uno nuevo.' : 'Tu página ya está tomando reservas. Compartí el link o cargá uno vos.'}</span>
			</div>
		{/each}
	</section>

	<div class="lado">
		{#if conHorario.length}
			<section class="caja" aria-labelledby="t-huecos">
				<div class="cabeza-c"><h2 id="t-huecos">Huecos de hoy</h2><span class="tl-num total">{libresTexto(libres)}</span></div>
				<p class="bajada">Para ofrecer cuando llaman o pasan por la&nbsp;puerta.</p>
				<div class="huecos">
					{#each huecos.slice(0, 6) as h (`${h.profesionalId}${h.desde}`)}
						<a href={cargarEn(h.profesionalId, data.hoy.fecha, h.desde)} aria-label={`Cargar un turno con ${h.profesional} a las ${hhmm(h.desde)}`}>
							<span class="rango tl-num">{hhmm(h.desde)}–{hhmm(h.hasta)}</span>
							<span class="quien">{primerNombre(h.profesional)}</span>
							<span class="dur tl-num">{duracion(h.hasta - h.desde)}</span>
						</a>
					{:else}
						<p class="bajada">No quedan huecos por hoy.</p>
					{/each}
				</div>
			</section>
		{/if}

		{#if verChecklist}
			<section class="caja pasos" aria-labelledby="t-pasos">
				<div class="pasos-cab">
					<div class="cabeza-c">
						<h2 id="t-pasos">Para recibir turnos</h2>
						{#if faltan[0]}<a class="guia" href={DESTINO[faltan[0].clave]?.href ?? '/panel/configuracion'}>Seguir la guía<Icono nombre="arrow_forward" tam={16} /></a>{/if}
						<span class="tl-num cuenta">{listos}/{pasos.length}</span>
					</div>
					<span class="barras" style:--n={pasos.length} aria-hidden="true">
						{#each pasos as p (p.clave)}<span class:lleno={p.listo}></span>{/each}
					</span>
				</div>
				<div class="troquel" aria-hidden="true"><span></span></div>
				<ul>
					{#each faltan as p (p.clave)}
						{@const d = DESTINO[p.clave] ?? { href: '/panel/configuracion', icono: 'check_circle', accion: 'Hacer' }}
						<li>
							<Icono nombre={d.icono} tam={22} />
							<span class="paso">{p.titulo}{p.obligatorio ? '' : ' (opcional)'}</span>
							<a href={d.href}>{d.accion}</a>
						</li>
					{/each}
					{#if data.mpConectado === false}
						<li>
							<img src="/logos/mercadopago.svg" alt="" width="22" height="22" />
							<span class="paso">Conectá Mercado Pago para cobrar señas</span>
							<a href="/panel/pagos">Conectar</a>
						</li>
					{/if}
					<li>
						<Icono nombre="link" tam={22} />
						<span class="paso">Poné tu link en la bio de Instagram</span>
						<button onclick={copiarLink}>{copiado ? 'Copiado' : 'Copiar'}</button>
					</li>
				</ul>
			</section>
		{/if}
	</div>
</div>

<style>
	.saludo {
		margin: 4px 0 0;
		font-size: 50px;
		line-height: 1.02;
		font-weight: 700;
		letter-spacing: -0.05em;
		max-width: 15em;
	}
	.resalte {
		white-space: nowrap;
		color: var(--tl-menta-texto);
		background: url("data:image/svg+xml;utf8,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 300 20' preserveAspectRatio='none'%3E%3Cpath d='M4 14 C80 4 180 4 296 10' fill='none' stroke='%233ccc90' stroke-width='5' stroke-linecap='round' opacity='.6'/%3E%3C/svg%3E")
			left 100% / 100% 0.28em no-repeat;
		padding-bottom: 0.06em;
	}
	/* Lo que necesita atención */
	.atencion {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(210px, 1fr));
		gap: 12px;
	}
	.tarjeta-a {
		display: contents;
	}
	.card {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 6px;
		padding: 16px;
		border: 0;
		border-radius: 18px;
		background: var(--tl-papel);
		box-shadow: 0 10px 20px -16px rgba(15, 35, 40, 0.45);
		color: var(--tl-tinta);
		text-align: left;
		text-decoration: none;
		font-family: var(--tl-font);
		cursor: pointer;
		transition: transform var(--tl-tocar) var(--tl-ease);
		animation: tl-gin var(--tl-entrar) var(--tl-ease) both;
	}
	.card:hover {
		transform: translateY(-2px);
	}
	.card.rayada {
		background: repeating-linear-gradient(-45deg, rgba(60, 204, 144, 0.2) 0 2px, transparent 2px 10px), #f3fbf7;
		box-shadow: inset 0 0 0 1.5px rgba(23, 128, 90, 0.3);
	}
	.num {
		font-size: 26px;
		font-weight: 700;
		letter-spacing: -0.03em;
		line-height: 1.1;
	}
	.card b {
		font-size: 14px;
		line-height: 1.3;
	}
	.sub {
		font-size: 12px;
		color: var(--tl-sub);
	}
	.ir {
		margin-top: 4px;
		font-size: 13px;
		font-weight: 700;
		color: var(--tl-menta-texto);
	}
	/* Dos columnas */
	.dos {
		display: grid;
		grid-template-columns: minmax(0, 1.5fr) minmax(0, 1fr);
		gap: 20px;
		align-items: start;
	}
	.sigue {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.cabeza {
		display: flex;
		align-items: baseline;
		gap: 10px;
	}
	.cabeza h2 {
		flex: 1;
		margin: 0;
		font-size: 18px;
		font-weight: 700;
		letter-spacing: -0.01em;
	}
	.cabeza a,
	.hueco-fila a {
		min-height: 36px;
		display: inline-flex;
		align-items: center;
		color: var(--tl-menta-texto);
		font-size: 14px;
		font-weight: 700;
		text-decoration: none;
	}
	.talon-chico {
		display: grid;
		grid-template-columns: 96px minmax(0, 1fr) auto;
		align-items: stretch;
		min-height: 62px;
		border-radius: 14px;
		background:
			radial-gradient(circle at 96px 0, var(--tl-crema) 9px, transparent 9.5px),
			radial-gradient(circle at 96px 100%, var(--tl-crema) 9px, transparent 9.5px),
			var(--tl-papel);
		box-shadow: 0 1px 0 rgba(15, 35, 40, 0.05), 0 10px 20px -16px rgba(15, 35, 40, 0.45);
		color: var(--tl-tinta);
		text-decoration: none;
		transition: transform var(--tl-tocar) var(--tl-ease);
		animation: tl-gin var(--tl-entrar) var(--tl-ease) both;
	}
	.talon-chico:hover {
		transform: translateY(-2px);
	}
	.talon-chico.en-curso {
		box-shadow: 0 0 0 2px var(--tl-petroleo), 0 10px 20px -16px rgba(15, 35, 40, 0.45);
	}
	.stub {
		display: flex;
		flex-direction: column;
		justify-content: center;
		padding: 0 14px;
		border-right: 2px dashed var(--tl-linea-2);
	}
	.stub b {
		font-size: 18px;
		font-weight: 700;
		letter-spacing: -0.02em;
	}
	.stub span,
	.medio span {
		font-size: 11px;
		color: var(--tl-sub);
	}
	.medio {
		display: flex;
		flex-direction: column;
		justify-content: center;
		padding: 8px 14px;
		min-width: 0;
	}
	.medio b {
		font-size: 15px;
		font-weight: 700;
	}
	.medio span {
		font-size: 12px;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.est {
		display: flex;
		align-items: center;
		padding-right: 14px;
	}
	.hueco-fila {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 8px 14px;
		min-height: 48px;
		padding: 8px 14px 8px 16px;
		border-radius: 14px;
		background: repeating-linear-gradient(-45deg, rgba(60, 204, 144, 0.22) 0 2px, transparent 2px 10px), rgba(60, 204, 144, 0.08);
		background-size: 28.28px 28.28px;
		box-shadow: inset 0 0 0 1.5px rgba(23, 128, 90, 0.35);
		animation: tl-raya-c 3.5s linear infinite;
	}
	@keyframes tl-raya-c {
		to {
			background-position: 28.28px 0;
		}
	}
	.rango {
		font-size: 13px;
		font-weight: 700;
		color: var(--tl-menta-texto);
	}
	.hueco-fila b {
		flex: 1;
		font-size: 13px;
		color: #0f3d2b;
	}
	.hueco-fila a {
		min-height: 34px;
		padding: 0 12px;
		border-radius: 999px;
		background: var(--tl-menta-texto);
		color: #fff;
		font-size: 13px;
	}
	.vacio {
		padding: 32px 20px;
		border-radius: 18px;
		background: var(--tl-papel);
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 8px;
	}
	.vacio :global(.icono) {
		color: var(--tl-menta-texto);
	}
	.vacio span {
		font-size: 14px;
		color: var(--tl-sub);
		max-width: 32em;
		text-wrap: pretty;
	}
	/* Columna de al lado */
	.lado {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	.caja {
		display: flex;
		flex-direction: column;
		gap: 10px;
		padding: 16px;
		border-radius: 18px;
		background: var(--tl-papel);
		box-shadow: 0 10px 20px -16px rgba(15, 35, 40, 0.45);
	}
	.cabeza-c {
		display: flex;
		align-items: baseline;
		gap: 8px;
	}
	.cabeza-c h2 {
		flex: 1;
		margin: 0;
		font-size: 16px;
		font-weight: 700;
	}
	.total {
		font-size: 12px;
		color: var(--tl-menta-texto);
	}
	.bajada {
		margin: -4px 0 0;
		font-size: 13px;
		color: var(--tl-sub);
	}
	.huecos {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	.huecos a {
		display: flex;
		align-items: center;
		gap: 10px;
		min-height: 44px;
		padding: 0 12px;
		border-radius: 12px;
		background: repeating-linear-gradient(-45deg, rgba(60, 204, 144, 0.18) 0 2px, transparent 2px 9px), #f3fbf7;
		color: var(--tl-tinta);
		text-decoration: none;
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.huecos a:hover {
		transform: translateX(2px);
	}
	.huecos .rango {
		min-width: 96px;
	}
	.quien {
		flex: 1;
		font-size: 13px;
		font-weight: 700;
	}
	.dur {
		font-size: 11px;
		color: var(--tl-sub);
	}
	.pasos {
		padding: 0;
		gap: 0;
		overflow: hidden;
	}
	.pasos-cab {
		padding: 16px 16px 12px;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.guia {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		margin-right: 10px;
		font-size: 13px;
		font-weight: 700;
		color: var(--tl-menta-texto);
		text-decoration: none;
	}
	.cuenta {
		font-size: 12px;
		font-weight: 700;
	}
	.barras {
		display: grid;
		grid-template-columns: repeat(var(--n), 1fr);
		gap: 3px;
	}
	.barras span {
		height: 6px;
		border-radius: 3px;
		background: #efe6d4;
	}
	.barras span.lleno {
		background: var(--tl-menta);
	}
	.troquel {
		height: 14px;
		position: relative;
		background:
			radial-gradient(circle at 0 50%, var(--tl-crema) 7px, transparent 7.5px),
			radial-gradient(circle at 100% 50%, var(--tl-crema) 7px, transparent 7.5px);
	}
	.troquel span {
		position: absolute;
		left: 12px;
		right: 12px;
		top: 6px;
		border-top: 2px dashed var(--tl-linea-2);
	}
	.pasos ul {
		list-style: none;
		margin: 0;
		padding: 0 0 6px;
	}
	.pasos li {
		display: flex;
		align-items: center;
		gap: 10px;
		min-height: 52px;
		padding: 0 16px;
	}
	.pasos li :global(.icono) {
		color: var(--tl-menta-texto);
	}
	.paso {
		flex: 1;
		font-size: 14px;
		font-weight: 700;
		text-wrap: pretty;
	}
	.pasos li a,
	.pasos li button {
		min-height: 40px;
		display: inline-flex;
		align-items: center;
		border: 0;
		background: transparent;
		padding: 0;
		font: 700 13px var(--tl-font);
		color: var(--tl-menta-texto);
		text-decoration: none;
		cursor: pointer;
	}
	/* El anillo: sólo en el teléfono */
	.anillo-tel {
		display: none;
	}
	@media (max-width: 1099px) {
		.dos {
			grid-template-columns: minmax(0, 1fr);
		}
	}
	@media (max-width: 859px) {
		.saludo {
			font-size: 34px;
		}
		.talon-chico {
			grid-template-columns: 76px minmax(0, 1fr) auto;
			background:
				radial-gradient(circle at 76px 0, var(--tl-crema) 9px, transparent 9.5px),
				radial-gradient(circle at 76px 100%, var(--tl-crema) 9px, transparent 9.5px),
				var(--tl-papel);
		}
		.stub {
			padding: 0 12px;
		}
		.anillo-tel {
			display: flex;
			align-items: center;
			gap: 18px;
			padding: 16px;
			border-radius: 22px;
		}
		.anillos {
			position: relative;
			width: 132px;
			height: 132px;
			flex: none;
			display: block;
		}
		.vuelta {
			position: absolute;
			border-radius: 50%;
			-webkit-mask: radial-gradient(farthest-side, transparent calc(100% - 10px), #000 calc(100% - 9px));
			mask: radial-gradient(farthest-side, transparent calc(100% - 10px), #000 calc(100% - 9px));
		}
		.aguja {
			position: absolute;
			left: 50%;
			top: -5px;
			width: 6px;
			height: 38px;
			margin-left: -3px;
			border-radius: 3px;
			background: #fff;
			transform-origin: 50% 71px;
		}
		.anillo-txt {
			display: flex;
			flex-direction: column;
			gap: 4px;
			min-width: 0;
		}
		.chico-r {
			font-size: 11px;
			color: rgba(255, 255, 255, 0.7);
		}
		.anillo-txt b {
			font-size: 30px;
			font-weight: 700;
			letter-spacing: -0.04em;
			color: var(--tl-menta);
			line-height: 1;
		}
		.entre {
			font-size: 13px;
			color: rgba(255, 255, 255, 0.75);
			text-wrap: pretty;
		}
		.ver-huecos {
			align-self: flex-start;
			margin-top: 4px;
			min-height: 44px;
			display: inline-flex;
			align-items: center;
			padding: 0 14px;
			border-radius: 999px;
			background: rgba(60, 204, 144, 0.18);
			color: #bff0d9;
			font-size: 13px;
			font-weight: 700;
			text-decoration: none;
		}
	}
</style>
