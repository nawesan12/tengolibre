<script lang="ts">
	import '#lib/gestion/gestion.css';
	import { goto } from '$app/navigation';
	import { pesos } from '#lib/formato.ts';
	import MapaCalor from '#lib/gestion/reportes/MapaCalor.svelte';
	import { diasDeSemana, otroMes } from '#lib/gestion/reportes/mensual.ts';

	let { data } = $props();
	const r = $derived(data.reporte);

	/** Sin palabra sola en el último renglón: la última va pegada a la anterior. */
	const sinViuda = (s: string) => s.replace(/\s+(\S+)\s*$/, ' $1');
	const num = (n: number | null | undefined, dec = 0) =>
		n === null || n === undefined ? '–' : new Intl.NumberFormat('es-AR', { maximumFractionDigits: dec, minimumFractionDigits: dec }).format(n);
	const mil = (n: number) => (n >= 1000 ? `$ ${num(Math.round(n / 1000))} mil` : pesos(n));
	const iniciales = (s: string) =>
		s
			.split(/\s+/)
			.slice(0, 2)
			.map((p) => p.charAt(0).toUpperCase())
			.join('');

	const ICONO: Record<string, string> = { sube: 'trending_up', baja: 'trending_down' };
	const maxSemana = $derived(r ? Math.max(1, ...r.plata.semanas.map((s) => s.ingresos)) : 1);
	const mejorSemana = $derived(r ? r.plata.semanas.reduce((m, s, i, a) => (s.ingresos > a[m].ingresos ? i : m), 0) : -1);
	const totalClientes = $derived(r ? r.clientes.nuevos + r.clientes.recurrentes : 0);
	const volvieron = $derived(r && totalClientes ? Math.round((r.clientes.recurrentes / totalClientes) * 100) : 0);

	const mesAnterior = $derived(r ? otroMes(r.mes, -1) : '');
	const mesSiguiente = $derived(r ? otroMes(r.mes, 1) : '');
	const hayDespues = $derived(r ? mesSiguiente <= data.mesActual : false);

	function elegir(e: Event & { currentTarget: HTMLInputElement }) {
		const v = e.currentTarget.value;
		if (/^\d{4}-\d{2}$/.test(v)) goto(`?mes=${v}`);
	}
</script>

<svelte:head><title>Reporte mensual{r ? ` · ${r.nombre}` : ''} · Panel</title></svelte:head>

<div class="g-pagina pagina">
	<div class="acciones">
		<a class="g-accion neutro volver" href="/panel/reportes"><span class="icono" aria-hidden="true">arrow_back</span>Reportes</a>
		{#if r}
			<nav class="meses" aria-label="Mes del reporte">
				<a class="g-boton chico flecha" href={`?mes=${mesAnterior}`} aria-label="Mes anterior"><span class="icono" aria-hidden="true">chevron_left</span></a>
				<label class="mes">
					<span class="visualmente-oculto">Mes</span>
					<input type="month" value={r.mes} max={data.mesActual} onchange={elegir} />
				</label>
				{#if hayDespues}
					<a class="g-boton chico flecha" href={`?mes=${mesSiguiente}`} aria-label="Mes siguiente"><span class="icono" aria-hidden="true">chevron_right</span></a>
				{/if}
			</nav>
			<button type="button" class="g-boton oscuro" onclick={() => window.print()}><span class="icono" aria-hidden="true">download</span>Bajar PDF</button>
		{/if}
	</div>

	{#if data.error}
		<p class="aviso error" role="alert">{data.error}</p>
	{/if}

	{#if r}
		{#if !r.completo}
			<p class="g-aviso ambar"><span class="icono" aria-hidden="true">schedule</span><span>{r.nombre} todavía no terminó: los números son hasta&nbsp;hoy.</span></p>
		{/if}

		<article class="reporte" aria-label={`Reporte mensual de ${r.negocio}, ${r.nombre}`}>
			<!-- 01 · Portada -->
			<section class="hoja portada">
				<div class="tapa tl-petroleo">
					<img class="simbolo" src="/marca/simbolo_una_tinta_blanco.svg" alt="" aria-hidden="true" />
					<div class="arriba"><img src="/marca/logotipo_horizontal_negativo.svg" alt="Tengo Libre" class="logo-neg" /><span class="rotulo-claro">REPORTE MENSUAL</span></div>
					<span class="negocio">{r.negocio} · {r.nombre}</span>
					<h1>{sinViuda(r.titulo)}</h1>
					<p>{sinViuda(r.bajada)}</p>
				</div>
				<div class="cuerpo-portada">
					<ul class="kpis">
						{#each r.indicadores as k (k.clave)}
							<li>
								<span class="kpi-titulo">{k.titulo}</span>
								<b class="num">{k.valorTexto}</b>
								<span class="kpi-cambio" class:malo={k.bueno === false} class:neutro={k.bueno === null}>
									<span class="icono" aria-hidden="true">{ICONO[k.tendencia] ?? 'trending_flat'}</span>{k.texto}
								</span>
							</li>
						{/each}
					</ul>
					<div class="pie"><span>{r.periodo}</span><span>1 / 4</span></div>
				</div>
			</section>

			<!-- 02 · Agenda -->
			<section class="hoja">
				<div class="cabecera"><span class="seccion">02 · AGENDA</span><img src="/marca/logotipo_horizontal.svg" alt="Tengo Libre" class="logo" /></div>
				<h2>{sinViuda(r.agenda.titulo)}</h2>
				{#if r.agenda.mapa.dias.length && r.agenda.mapa.horas.length}
					<div class="g-col mapa"><MapaCalor mapa={r.agenda.mapa} etiqueta={`Ocupación por día y hora en ${r.nombre}`} /></div>
				{:else}
					<p class="g-nota">Sin horarios cargados ese mes: no hay mapa de la&nbsp;agenda.</p>
				{/if}
				<div class="dos">
					<div class="talon">
						<span class="lado rayado"><b class="num">{num(r.agenda.horasLibres)}&nbsp;h</b><span>quedaron libres</span></span>
						<span class="dato">
							{#if r.agenda.franjaLibre}<b>{r.agenda.franjaLibre.texto}</b>{sinViuda(r.agenda.franjaLibre.detalle)}{:else}<b>Sin una franja que se repita</b>Lo libre quedó repartido en la&nbsp;semana.{/if}
						</span>
					</div>
					<div class="talon">
						<span class="lado oscuro tl-petroleo"><b class="num menta">{num(r.agenda.huecosLlenados)}</b><span>huecos llenados</span></span>
						<span class="dato"><b>Con la lista de espera</b>Cancelaciones que se volvieron turnos en menos de 2&nbsp;horas.</span>
					</div>
				</div>
				{#if r.agenda.sugerencia}
					<div class="idea">
						<span class="idea-titulo"><span class="icono lleno" aria-hidden="true">lightbulb</span>{r.agenda.sugerencia.titulo}</span>
						<p>{sinViuda(r.agenda.sugerencia.texto)}</p>
					</div>
				{/if}
				<div class="pie"><span>{r.negocio} · {r.nombre}</span><span>2 / 4</span></div>
			</section>

			<!-- 03 · Plata y servicios -->
			<section class="hoja">
				<div class="cabecera"><span class="seccion">03 · PLATA Y SERVICIOS</span><img src="/marca/logotipo_horizontal.svg" alt="Tengo Libre" class="logo" /></div>
				<h2>{sinViuda(r.plata.titulo)}</h2>
				{#if r.plata.semanas.length}
					<div class="semanas" style:--n={r.plata.semanas.length}>
						<ul class="barras" aria-label="Ingresos por semana">
							{#each r.plata.semanas as s, i (s.etiqueta)}
								<li>
									<b class="num">{mil(s.ingresos)}</b>
									<span
										class="barra"
										class:mejor={i === mejorSemana && s.ingresos > 0}
										class:parcial={diasDeSemana(s.etiqueta) < 7}
										style:height="{Math.max(2, (s.ingresos / maxSemana) * 100)}%"
									></span>
									<span class="visualmente-oculto">{s.etiqueta}{i === mejorSemana ? ', la mejor semana' : ''}{diasDeSemana(s.etiqueta) < 7 ? ', semana corta' : ''}</span>
								</li>
							{/each}
						</ul>
						<div class="etiquetas" aria-hidden="true">{#each r.plata.semanas as s (s.etiqueta)}<span class="num">{s.etiqueta}</span>{/each}</div>
					</div>
				{/if}
				{#if r.plata.servicios.length}
					<div class="g-desplazable">
						<table class="servicios">
							<thead><tr><th>SERVICIO</th><th class="der">TURNOS</th><th class="der">FACTURADO</th><th class="parte">PARTE DEL TOTAL</th></tr></thead>
							<tbody>
								{#each r.plata.servicios as s (s.nombre)}
									<tr>
										<td class="nombre">{s.nombre}</td>
										<td class="der num">{num(s.turnos)}</td>
										<td class="der num fuerte">{pesos(s.facturado)}</td>
										<td class="parte"><span class="parte-fila"><span class="pista"><span style:width="{s.parte}%"></span></span><span class="num">{num(s.parte)}%</span></span></td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
				<ul class="tres">
					<li><span>Señas cobradas</span><b class="num">{pesos(r.plata.senas)}</b><span class="sub">{num(r.plata.turnosConSena)} {r.plata.turnosConSena === 1 ? 'turno' : 'turnos'} con seña</span></li>
					<li><span>Membresías</span><b class="num">{pesos(r.plata.membresias)}</b><span class="sub">{num(r.plata.sociosActivos)} {r.plata.sociosActivos === 1 ? 'socia activa' : 'socias activas'}</span></li>
					<li><span>Ticket promedio</span><b class="num">{pesos(r.plata.ticketPromedio)}</b><span class="sub">{r.plata.ticketTexto}</span></li>
				</ul>
				<div class="pie"><span>{sinViuda(r.plata.nota)}</span><span>3 / 4</span></div>
			</section>

			<!-- 04 · Equipo y clientes -->
			<section class="hoja">
				<div class="cabecera"><span class="seccion">04 · EQUIPO Y CLIENTES</span><img src="/marca/logotipo_horizontal.svg" alt="Tengo Libre" class="logo" /></div>
				{#if r.equipo.length}
					<ul class="equipo">
						{#each r.equipo as e (e.id)}
							<li>
								{#if data.fotos[e.id]}<img class="avatar" src={data.fotos[e.id]} alt="" />{:else}<span class="avatar ini" aria-hidden="true">{iniciales(e.nombre)}</span>{/if}
								<span class="quien"><b>{e.nombre}</b><span class="ocupa" aria-hidden="true"><span style:width="{e.ocupacion ?? 0}%"></span></span></span>
								<span class="cifra"><b class="num">{e.ocupacion === null ? '–' : `${num(e.ocupacion)}%`}</b><span>ocupación</span></span>
								<span class="cifra extra"><b class="num">{num(e.turnos)}</b><span>turnos</span></span>
								<span class="cifra extra"><b class="num">{e.puntaje === null ? '–' : num(e.puntaje, 1)}</b><span>reseñas</span></span>
							</li>
						{/each}
					</ul>
				{/if}
				<div class="dos">
					<div class="caja">
						<b class="caja-titulo">Quiénes vinieron</b>
						{#if totalClientes}
							<div class="vinieron">
								<span class="volvieron num" style:flex={Math.max(volvieron, 1)}>{volvieron}% volvieron</span>
								<span class="nuevas num" style:flex={Math.max(100 - volvieron, 1)}>{100 - volvieron}% nuevas</span>
							</div>
						{/if}
						<span class="caja-texto">{sinViuda(r.clientes.texto)}</span>
					</div>
					<div class="caja">
						{#if r.resenas.cantidad > 0}
							<span class="estrellas"><b class="num">{num(r.resenas.promedio, 1)} ★</b><span>{r.resenas.cantidad} {r.resenas.cantidad === 1 ? 'reseña nueva' : 'reseñas nuevas'}</span></span>
							{#if r.resenas.destacada}
								<span class="cita">«{sinViuda(r.resenas.destacada.comentario)}»</span>
								<span class="firma">{r.resenas.destacada.nombre}{r.resenas.destacada.profesional ? `, con ${r.resenas.destacada.profesional}` : ''}</span>
							{/if}
						{:else}
							<b class="caja-titulo">Reseñas</b>
							<span class="caja-texto">Sin reseñas nuevas este mes. Pedirlas después de cada turno ayuda a que te&nbsp;elijan.</span>
						{/if}
					</div>
				</div>
				{#if r.siguiente.length}
					<div class="para tl-petroleo">
						<h2 class="para-titulo">Para {r.nombreSiguiente}</h2>
						<ol>
							{#each r.siguiente as s (s.numero)}
								<li><b class="num">{s.numero}</b><span><b>{s.titulo}</b><span>{sinViuda(s.texto)}</span></span></li>
							{/each}
						</ol>
					</div>
				{/if}
				<div class="pie">
					<span class="hecho"><img src="/marca/simbolo_una_tinta_petroleo.svg" alt="" width="18" height="18" />{r.pie}</span><span>4 / 4</span>
				</div>
			</section>
		</article>
	{/if}
</div>

<style>
	.pagina {
		gap: 20px;
		align-items: center;
	}
	.acciones {
		width: 100%;
		max-width: 794px;
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 12px;
	}
	.volver {
		margin-right: auto;
	}
	.volver .icono {
		font-size: 18px;
	}
	.meses {
		display: flex;
		align-items: center;
		gap: 6px;
	}
	.flecha {
		width: 40px;
		padding: 0;
	}
	.mes input {
		min-height: 40px;
		padding: 0 12px;
		border: 0;
		border-radius: var(--tl-r-s);
		box-shadow: inset 0 0 0 1.5px #e6dcc8;
		background: #fff;
		font: 700 15px var(--tl-font);
		font-variant-numeric: tabular-nums;
		color: var(--tl-tinta);
	}
	.g-aviso {
		width: 100%;
		max-width: 794px;
	}
	.reporte {
		display: flex;
		flex-direction: column;
		gap: 24px;
		width: 100%;
		max-width: 794px;
	}
	.hoja {
		position: relative;
		display: flex;
		flex-direction: column;
		gap: 22px;
		min-height: 1123px;
		padding: 48px 52px 40px;
		border-radius: 8px;
		background: var(--tl-papel);
		box-shadow: 0 20px 40px -28px rgba(15, 35, 40, 0.45);
		overflow: hidden;
	}
	h1,
	h2 {
		margin: 0;
		text-wrap: balance;
	}
	h2 {
		font-size: 40px;
		line-height: 1;
		font-weight: 700;
		letter-spacing: -0.05em;
	}
	.pie {
		margin-top: auto;
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 16px;
		font-size: 13px;
		color: var(--tl-sub);
		font-variant-numeric: tabular-nums;
	}
	.cabecera {
		display: flex;
		justify-content: space-between;
		align-items: center;
	}
	.seccion {
		font-size: 13px;
		font-weight: 700;
		letter-spacing: 0.08em;
		color: var(--tl-menta-texto);
	}
	.logo {
		height: 16px;
		width: auto;
	}

	/* Portada */
	.portada {
		padding: 0;
		gap: 0;
		background: var(--tl-crema);
	}
	.tapa {
		position: relative;
		overflow: hidden;
		flex: none;
		min-height: 628px;
		padding: 48px 52px 40px;
		display: flex;
		flex-direction: column;
	}
	.simbolo {
		position: absolute;
		right: -90px;
		bottom: -110px;
		width: 460px;
		height: 460px;
		opacity: 0.05;
	}
	.arriba {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 12px;
	}
	.logo-neg {
		height: 24px;
		width: auto;
	}
	.rotulo-claro {
		font-size: 13px;
		font-weight: 700;
		letter-spacing: 0.08em;
		color: rgba(255, 255, 255, 0.7);
	}
	.negocio {
		margin-top: auto;
		font-size: 20px;
		font-weight: 700;
		color: var(--tl-menta);
	}
	.tapa h1 {
		margin-top: 10px;
		font-size: 64px;
		line-height: 0.92;
		font-weight: 700;
		letter-spacing: -0.06em;
		color: #fff;
	}
	.tapa p {
		margin: 16px 0 0;
		max-width: 30em;
		font-size: 16px;
		line-height: 1.5;
		color: rgba(255, 255, 255, 0.82);
		text-wrap: pretty;
	}
	.cuerpo-portada {
		flex: 1;
		padding: 32px 52px 40px;
		display: flex;
		flex-direction: column;
		gap: 20px;
	}
	.kpis {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 12px;
	}
	.kpis li {
		display: flex;
		flex-direction: column;
		gap: 6px;
		padding: 18px 20px;
		border-radius: 20px;
		background: var(--tl-papel);
	}
	.kpi-titulo {
		font-size: 13px;
		font-weight: 700;
		letter-spacing: 0.08em;
		color: var(--tl-sub);
	}
	.kpis b {
		font-size: 38px;
		line-height: 1;
		font-weight: 700;
		letter-spacing: -0.045em;
	}
	.kpi-cambio {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		font-size: 14px;
		font-weight: 700;
		color: var(--tl-menta-texto);
	}
	.kpi-cambio .icono {
		font-size: 16px;
	}
	.kpi-cambio.malo {
		color: #a3241a;
	}
	.kpi-cambio.neutro {
		color: var(--tl-sub);
	}

	/* Agenda */
	.dos {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 12px;
	}
	.talon {
		display: grid;
		grid-template-columns: 120px minmax(0, 1fr);
		border-radius: 20px;
		overflow: hidden;
		background: #f8f3ea;
	}
	.lado {
		display: flex;
		flex-direction: column;
		justify-content: center;
		padding: 16px;
		border-right: 3px dashed var(--tl-papel);
	}
	.lado b {
		font-size: 34px;
		line-height: 1.05;
		letter-spacing: -0.04em;
	}
	.lado span {
		font-size: 13px;
		font-weight: 700;
	}
	.rayado {
		background: repeating-linear-gradient(-45deg, #3ccc90 0 3px, #cdeedd 3px 10px);
		color: var(--tl-petroleo-hondo);
	}
	.oscuro {
		color: #fff;
	}
	.menta {
		color: var(--tl-menta);
	}
	.dato {
		display: flex;
		flex-direction: column;
		justify-content: center;
		gap: 4px;
		padding: 14px 16px;
		font-size: 14px;
		line-height: 1.4;
		text-wrap: pretty;
	}
	.dato b {
		font-size: 15px;
	}
	.idea {
		display: flex;
		flex-direction: column;
		gap: 10px;
		padding: 18px 20px;
		border-radius: 20px;
		background: var(--tl-menta-suave);
		color: var(--tl-petroleo-hondo);
	}
	.idea-titulo {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		font-size: 15px;
		font-weight: 700;
	}
	.idea-titulo .icono {
		font-size: 20px;
		color: var(--tl-menta-texto);
	}
	.idea p {
		margin: 0;
		font-size: 15px;
		line-height: 1.5;
		text-wrap: pretty;
	}

	/* Plata */
	.semanas {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.barras {
		list-style: none;
		margin: 0;
		display: grid;
		grid-template-columns: repeat(var(--n), minmax(0, 1fr));
		gap: 14px;
		align-items: end;
		height: 190px;
		padding: 0 4px;
		border-bottom: 2px solid var(--tl-linea-2);
	}
	.barras li {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: flex-end;
		gap: 6px;
		height: 100%;
	}
	.barras b {
		font-size: 14px;
		white-space: nowrap;
	}
	.barra {
		width: 100%;
		max-height: calc(100% - 26px);
		border-radius: 10px 10px 0 0;
		background: var(--tl-petroleo);
	}
	.barra.mejor {
		background: var(--tl-menta);
	}
	.barra.parcial {
		background: repeating-linear-gradient(-45deg, rgba(255, 255, 255, 0.18) 0 2px, transparent 2px 8px), var(--tl-petroleo);
	}
	.etiquetas {
		display: grid;
		grid-template-columns: repeat(var(--n), minmax(0, 1fr));
		gap: 14px;
		padding: 0 4px;
		font-size: 13px;
		font-weight: 700;
		color: var(--tl-sub);
		text-align: center;
	}
	.servicios {
		width: 100%;
		border-collapse: collapse;
		font-size: 15px;
		font-variant-numeric: tabular-nums;
	}
	.servicios th {
		padding: 8px 0;
		font-size: 12px;
		font-weight: 700;
		letter-spacing: 0.08em;
		color: var(--tl-sub);
		text-align: left;
	}
	.servicios td {
		padding: 11px 0;
		border-top: 1px solid #eee5d3;
	}
	.servicios .der {
		text-align: right;
	}
	.servicios .nombre,
	.servicios .fuerte {
		font-weight: 700;
	}
	.servicios .parte {
		width: 150px;
		padding-left: 16px;
	}
	.parte-fila {
		display: flex;
		align-items: center;
		gap: 8px;
	}
	.pista {
		flex: 1;
		height: 8px;
		border-radius: 4px;
		background: #eee5d3;
		overflow: hidden;
	}
	.pista span {
		display: block;
		height: 100%;
		border-radius: 4px;
		background: var(--tl-petroleo);
	}
	.parte-fila .num {
		width: 36px;
		text-align: right;
		font-size: 13px;
		color: var(--tl-sub);
	}
	.tres {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 12px;
	}
	.tres li {
		display: flex;
		flex-direction: column;
		gap: 4px;
		padding: 16px 18px;
		border-radius: 18px;
		background: #f8f3ea;
		font-size: 13px;
		font-weight: 700;
		color: var(--tl-sub);
	}
	.tres b {
		font-size: 24px;
		letter-spacing: -0.03em;
		color: var(--tl-tinta);
	}
	.tres .sub {
		font-weight: 400;
		color: var(--tl-sub-2);
	}

	/* Equipo y clientes */
	.equipo {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.equipo li {
		display: grid;
		grid-template-columns: 52px minmax(0, 1fr) 92px 92px 92px;
		align-items: center;
		gap: 12px;
		padding: 12px 16px 12px 12px;
		border-radius: 18px;
		background: #f8f3ea;
		font-variant-numeric: tabular-nums;
	}
	.avatar {
		width: 52px;
		height: 52px;
		border-radius: 50%;
		object-fit: cover;
	}
	.avatar.ini {
		display: grid;
		place-items: center;
		background: var(--tl-papel);
		font-weight: 700;
	}
	.quien {
		display: flex;
		flex-direction: column;
		gap: 6px;
		min-width: 0;
	}
	.quien b {
		font-size: 16px;
	}
	.ocupa {
		height: 8px;
		border-radius: 4px;
		background: repeating-linear-gradient(-45deg, #3ccc90 0 2px, #dff4e9 2px 6px);
		overflow: hidden;
	}
	.ocupa span {
		display: block;
		height: 100%;
		border-radius: 4px;
		background: var(--tl-petroleo);
	}
	.cifra {
		display: flex;
		flex-direction: column;
		text-align: right;
	}
	.cifra b {
		font-size: 18px;
	}
	.cifra span {
		font-size: 12px;
		color: var(--tl-sub);
	}
	.caja {
		display: flex;
		flex-direction: column;
		gap: 12px;
		padding: 18px 20px;
		border-radius: 20px;
		background: #f8f3ea;
	}
	.caja-titulo {
		font-size: 16px;
	}
	.vinieron {
		display: flex;
		height: 40px;
		border-radius: 10px;
		overflow: hidden;
		font-size: 13px;
		font-weight: 700;
	}
	.vinieron > span {
		display: flex;
		align-items: center;
		white-space: nowrap;
		min-width: max-content;
	}
	.volvieron {
		padding: 0 12px;
		background: var(--tl-petroleo);
		color: #fff;
	}
	.nuevas {
		padding: 0 10px;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
	}
	.caja-texto {
		font-size: 14px;
		line-height: 1.45;
		color: var(--tl-sub-2);
		text-wrap: pretty;
	}
	.estrellas {
		display: flex;
		flex-wrap: wrap;
		align-items: baseline;
		gap: 8px;
	}
	.estrellas b {
		font-size: 34px;
		letter-spacing: -0.04em;
	}
	.estrellas span {
		font-size: 14px;
		color: var(--tl-sub);
	}
	.cita {
		font-size: 15px;
		line-height: 1.45;
		font-weight: 700;
		text-wrap: pretty;
	}
	.firma {
		font-size: 13px;
		color: var(--tl-sub);
	}
	.para {
		display: flex;
		flex-direction: column;
		border-radius: 24px;
		overflow: hidden;
		padding-bottom: 8px;
	}
	.para-titulo {
		padding: 18px 22px 12px;
		font-size: 18px;
		letter-spacing: -0.01em;
		line-height: 1.2;
	}
	.para ol {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.para li {
		display: grid;
		grid-template-columns: 44px minmax(0, 1fr);
		gap: 12px;
		padding: 12px 22px;
		border-top: 2px dashed rgba(255, 255, 255, 0.14);
	}
	.para li > b {
		font-size: 22px;
		color: var(--tl-menta);
	}
	.para li > span {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}
	.para li > span > b {
		font-size: 15px;
	}
	.para li > span > span {
		font-size: 14px;
		line-height: 1.4;
		color: rgba(255, 255, 255, 0.78);
		text-wrap: pretty;
	}
	.hecho {
		display: inline-flex;
		align-items: center;
		gap: 8px;
	}

	/* Teléfono: la hoja se vuelve fluida y lo de dos columnas, una. */
	@media (max-width: 720px) {
		.hoja {
			min-height: 0;
			padding: 28px 20px 24px;
			gap: 18px;
		}
		.portada {
			padding: 0;
		}
		.tapa {
			min-height: 460px;
			padding: 28px 20px 28px;
		}
		.tapa h1 {
			font-size: 44px;
		}
		.cuerpo-portada {
			padding: 20px;
		}
		h2 {
			font-size: 30px;
		}
		.kpis,
		.dos,
		.tres {
			grid-template-columns: minmax(0, 1fr);
		}
		.kpis b {
			font-size: 32px;
		}
		.talon {
			grid-template-columns: 104px minmax(0, 1fr);
		}
		.lado b {
			font-size: 28px;
		}
		.barras,
		.etiquetas {
			gap: 6px;
		}
		.barras b {
			font-size: 11px;
		}
		.etiquetas {
			font-size: 11px;
		}
		.servicios {
			font-size: 14px;
		}
		.servicios .parte {
			width: 96px;
			padding-left: 10px;
		}
		.equipo li {
			grid-template-columns: 44px minmax(0, 1fr) repeat(3, auto);
			gap: 8px;
			padding: 10px 12px 10px 10px;
		}
		.avatar {
			width: 44px;
			height: 44px;
		}
		.cifra b {
			font-size: 15px;
		}
		.cifra span {
			font-size: 11px;
		}
		.pie {
			flex-wrap: wrap;
		}
		.flecha {
			width: 48px;
			min-height: 48px;
		}
		.mes input {
			min-height: 48px;
		}
	}
	@media (max-width: 420px) {
		.equipo li {
			grid-template-columns: 44px minmax(0, 1fr) auto;
		}
		.equipo .extra {
			display: none;
		}
	}

	/* PDF: sólo el reporte, una hoja A4 por sección. */
	@page {
		size: A4;
		margin: 0;
	}
	@media print {
		:global(body *) {
			visibility: hidden;
		}
		.reporte,
		.reporte :global(*) {
			visibility: visible;
		}
		.reporte {
			position: absolute;
			left: 0;
			top: 0;
			width: 210mm;
			max-width: none;
			gap: 0;
		}
		.hoja {
			width: 210mm;
			height: 297mm;
			min-height: 0;
			border-radius: 0;
			box-shadow: none;
			break-after: page;
			print-color-adjust: exact;
			-webkit-print-color-adjust: exact;
		}
		.hoja:last-child {
			break-after: auto;
		}
	}
</style>
