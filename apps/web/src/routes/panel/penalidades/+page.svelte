<!--
	Clientes › Penalidades (handoff «Pantallas que faltaban», 1d): las vigentes como talones ámbar con
	cuánto falta y «Levantar»; las levantadas, apagadas; y las reglas de ausencias.
-->
<script lang="ts">
	import { page } from '$app/state';
	import { enhance } from '$app/forms';
	import Icono from '#lib/ui/Icono.svelte';
	import { dia } from '#lib/formato.ts';
	import Cabecera from '#lib/operacion/Cabecera.svelte';
	import { GESTION } from '#lib/operacion/secciones.ts';
	import { accion, fechaCorta } from '#lib/operacion/ui.ts';

	let { data, form } = $props();
	const gestiona = $derived(GESTION.includes(data.negocio.rol));
	const paginas = $derived(Math.max(1, Math.ceil(data.total / data.porPagina)));
	const enlace = (pagina: number) => `?${new URLSearchParams({ pagina: String(pagina), ...(data.todas ? { todas: '1' } : {}) })}`;
	const MOTIVOS: Record<string, string> = { ausencias: 'ausencias', manual: 'Puesta a mano' };
	const marcados = $derived(form && 'datos' in form && form.datos && typeof form.datos === 'object' && 'marcados' in form.datos ? (form.datos.marcados as number) : null);
	const faltan = (hasta: string) => Math.max(0, Math.ceil((Date.parse(`${dia(hasta)}T12:00:00Z`) - Date.parse(`${dia()}T12:00:00Z`)) / 86_400_000));
	const vigentes = $derived(data.penalidades.filter((p) => p.vigente));
	const anteriores = $derived(data.penalidades.filter((p) => !p.vigente));
</script>

<svelte:head><title>Penalidades · Clientes · Panel</title></svelte:head>

<Cabecera titulo="Penalidades" rotulo="Clientes" volver={{ href: '/panel/clientes', texto: 'Clientes' }} />

{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}
{#if marcados !== null}<p class="aviso" role="status">Se marcaron {marcados}&nbsp;{marcados === 1 ? 'turno' : 'turnos'} como no&nbsp;vino.</p>{/if}

<div class="numeros">
	<div><b class="tl-num">{data.ausencias.penalidadesVigentes}</b><span>penalidades vigentes</span></div>
	<div><b class="tl-num">{data.ausencias.ultimos30}</b><span>ausencias en 30 días</span></div>
	<div><b class="tl-num">{data.ausencias.total}</b><span>ausencias en total</span></div>
	<div><b class="tl-num">{data.ausencias.reincidentes}</b><span>reincidentes</span></div>
</div>

<div class="hoja">
	<p class="bajada">No pueden reservar online; vos sí podés cargarles turnos desde el&nbsp;panel.</p>
	{#if vigentes.length === 0}<p class="vacio">No hay penalidades vigentes.</p>{/if}
	<ul class="lista">
		{#each vigentes as p (p.id)}
			<li class="talon ambar">
				<span class="stub tl-num"><span class="r">Hasta</span><b>{fechaCorta(p.hasta)}</b><span class="r">Faltan {faltan(p.hasta)}&nbsp;d</span></span>
				<span class="medio">
					<a href={`/panel/clientes/${p.clienteId}`}><b>{p.cliente}</b></a>
					<span>{p.motivo === 'ausencias' ? `${p.ausencias} ${MOTIVOS.ausencias}` : MOTIVOS[p.motivo] ?? p.motivo}{p.nota ? ` · ${p.nota}` : ''}</span>
				</span>
				{#if gestiona}
					<form method="POST" action={accion(page.url, 'levantar')} use:enhance>
						<input type="hidden" name="id" value={p.id} />
						<button aria-label={`Levantar la penalidad de ${p.cliente}`}>Levantar</button>
					</form>
				{/if}
			</li>
		{/each}
	</ul>
	{#if data.todas && anteriores.length}
		<span class="tl-rotulo rot">Levantadas</span>
		<ul class="lista">
			{#each anteriores as p (p.id)}
				<li class="talon apagado">
					<span class="stub tl-num"><b>{fechaCorta(p.levantadaEn ?? p.hasta)}</b></span>
					<span class="medio"><a href={`/panel/clientes/${p.clienteId}`}><b>{p.cliente}</b></a><span>{p.nota || (MOTIVOS[p.motivo] ?? p.motivo)}</span></span>
					<span class="tl-rotulo est">{p.levantadaEn ? 'Levantada' : 'Venció'}</span>
				</li>
			{/each}
		</ul>
	{/if}
	<a class="ver" href={data.todas ? '?' : '?todas=1'}>{data.todas ? 'Ver sólo las vigentes' : 'Ver también las levantadas'}</a>
	{#if paginas > 1}
		<nav class="paginas" aria-label="Páginas">
			{#if data.pagina > 1}<a href={enlace(data.pagina - 1)}>← Anterior</a>{/if}
			<span class="tl-num">Página {data.pagina} de {paginas}</span>
			{#if data.pagina < paginas}<a href={enlace(data.pagina + 1)}>Siguiente →</a>{/if}
		</nav>
	{/if}
</div>

{#if data.ajustes && gestiona}
	<section class="reglas" aria-labelledby="t-reglas">
		<h2 id="t-reglas">Reglas de ausencias</h2>
		<form method="POST" action={accion(page.url, 'ajustes')} use:enhance={() => async ({ update }) => update({ reset: false })} class="form-r">
			<label class="campo">Penalizar con tantas ausencias en 90 días (0&nbsp;apaga) <input type="number" name="umbralAusencias" min="0" max="20" value={data.ajustes.umbralAusencias} /></label>
			<label class="campo">Días sin reservar <input type="number" name="diasPenalidad" min="1" max="365" value={data.ajustes.diasPenalidad} /></label>
			<label class="check"><input type="checkbox" name="marcarAusentes" checked={data.ajustes.marcarAusentes} />Marcar solos como «no vino» los turnos que&nbsp;pasaron</label>
			<button class="boton oscuro">Guardar</button>
		</form>
		<form method="POST" action={accion(page.url, 'marcar')} use:enhance>
			<button class="boton"><Icono nombre="person_off" tam={18} />Marcar ausentes ahora</button>
		</form>
	</section>
{/if}

<style>
	.numeros {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
		gap: 10px;
	}
	.numeros div {
		display: flex;
		flex-direction: column;
		padding: 14px 16px;
		border-radius: 18px;
		background: var(--tl-papel);
		box-shadow: 0 10px 20px -16px rgba(15, 35, 40, 0.45);
	}
	.numeros b {
		font-size: 26px;
		font-weight: 700;
		letter-spacing: -0.03em;
	}
	.numeros span {
		font-size: 12px;
		color: var(--tl-sub);
	}
	.hoja {
		max-width: 620px;
		padding: 24px;
		border-radius: 24px;
		background-color: var(--tl-crema);
		background-image: var(--tl-grano-osc), radial-gradient(rgba(24, 60, 68, 0.1) 1.2px, transparent 1.4px);
		background-size: auto, 18px 18px;
		box-shadow: var(--tl-sh-alta);
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.bajada,
	.vacio {
		margin: 0;
		font-size: 14px;
		color: var(--tl-sub-2);
	}
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.talon {
		--x: 88px;
		display: grid;
		grid-template-columns: var(--x) minmax(0, 1fr) auto;
		align-items: center;
		min-height: 58px;
		border-radius: 14px;
		background:
			radial-gradient(circle at var(--x) 0, var(--tl-crema) 7px, transparent 7.5px),
			radial-gradient(circle at var(--x) 100%, var(--tl-crema) 7px, transparent 7.5px),
			var(--fondo, var(--tl-papel));
		box-shadow: 0 10px 20px -16px rgba(15, 35, 40, 0.45);
	}
	.ambar {
		--fondo: #fbe9c2;
	}
	.apagado {
		--fondo: #faf6ee;
		opacity: 0.7;
	}
	.stub {
		align-self: stretch;
		display: flex;
		flex-direction: column;
		justify-content: center;
		padding: 6px 12px;
		border-right: 2px dashed var(--tl-linea-2);
		line-height: 1.15;
	}
	.stub b {
		font-size: 15px;
	}
	.r {
		font-size: 11px;
		text-transform: uppercase;
		color: #5c3800;
	}
	.medio {
		display: flex;
		flex-direction: column;
		padding: 8px 14px;
		min-width: 0;
	}
	.medio a {
		color: var(--tl-tinta);
		text-decoration: none;
	}
	.medio span {
		font-size: 12px;
		color: var(--tl-sub-2);
	}
	.talon button {
		margin-right: 12px;
		min-height: 40px;
		padding: 0 14px;
		border: 0;
		border-radius: 999px;
		background: var(--tl-petroleo);
		color: #fff;
		font: 700 13px var(--tl-font);
		cursor: pointer;
	}
	.est {
		margin-right: 14px;
		font-size: 11px;
		color: var(--tl-sub);
	}
	.rot {
		font-size: 11px;
		color: var(--tl-sub);
		margin-top: 6px;
	}
	.ver,
	.paginas a {
		font-size: 13px;
		font-weight: 700;
		color: var(--tl-menta-texto);
		min-height: 40px;
		display: inline-flex;
		align-items: center;
	}
	.paginas {
		display: flex;
		gap: 12px;
		align-items: center;
		font-size: 13px;
	}
	.reglas {
		max-width: 620px;
		padding: 20px;
		border-radius: 20px;
		background: var(--tl-papel);
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.reglas h2 {
		margin: 0;
		font-size: 18px;
	}
	.form-r {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(220px, 100%), 1fr));
		gap: 12px;
		align-items: end;
	}
	.check {
		display: flex;
		align-items: center;
		gap: 10px;
		min-height: 48px;
		font-size: 14px;
	}
	.check input {
		width: 20px;
		height: 20px;
		accent-color: var(--tl-petroleo);
	}
	@media (max-width: 859px) {
		.hoja {
			padding: 16px;
		}
	}
</style>
