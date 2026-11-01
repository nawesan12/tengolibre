<!--
	Mails enviados (¿le llegó el recordatorio?): el registro de avisos con su estado, el error si
	falló y «Reenviar». Mismo lenguaje que los avisos (filas papel sobre crema, rótulos, ícono+palabra).
-->
<script lang="ts">
	import { page } from '$app/state';
	import { accion } from '#lib/operacion/ui.ts';
	import { enhance } from '$app/forms';
	import Icono from '#lib/ui/Icono.svelte';
	import { fechaHora } from '#lib/formato.ts';
	import Cabecera from '#lib/operacion/Cabecera.svelte';
	import Chip from '#lib/operacion/Chip.svelte';
	import SinPermiso from '#lib/operacion/SinPermiso.svelte';

	let { data, form } = $props();
	const ESTADO: Record<string, { texto: string; icono: string; color: string }> = {
		pendiente: { texto: 'En cola', icono: 'schedule', color: '#4a5d61' },
		enviado: { texto: 'Enviado', icono: 'check_circle', color: '#17805a' },
		fallido: { texto: 'Falló', icono: 'error', color: '#a3241a' },
		descartado: { texto: 'Descartado', icono: 'block', color: '#7a4a00' }
	};
	const ultimo = $derived(data.avisos.at(-1));
	const enlace = (estado: string) => (estado ? `?estado=${estado}` : '?');
</script>

<svelte:head><title>Mails enviados · Panel</title></svelte:head>

<Cabecera titulo="Mails enviados" rotulo="Avisos" volver={{ href: '/panel/notificaciones', texto: 'Avisos' }} />

{#if data.prohibido}
	<SinPermiso que="el registro de mails" />
{:else}
	{#if form?.error}<p class="aviso error" role="alert">{form.error}</p>{/if}
	{#if form && 'ok' in form && form.ok}<p class="aviso" role="status">Listo, vuelve a salir en un&nbsp;momento.</p>{/if}
	<div class="chips" aria-label="Filtrar por estado">
		<Chip href={enlace('')} on={!data.estado}>Todos</Chip>
		{#each Object.entries(ESTADO) as [k, v] (k)}<Chip href={enlace(k)} on={data.estado === k}>{v.texto}</Chip>{/each}
	</div>

	{#if data.avisos.length === 0}
		<p class="vacio">No hay mails con ese filtro.</p>
	{:else}
		<ul class="lista">
			{#each data.avisos as a (a.id)}
				{@const e = ESTADO[a.estado] ?? { texto: a.estado, icono: 'mail', color: '#4a5d61' }}
				<li>
					<span class="ic" style:color={e.color}><Icono nombre={e.icono} lleno tam={20} /></span>
					<span class="txt">
						<b>{a.tipo.replaceAll('_', ' ')}{a.cliente ? ` · ${a.cliente}` : ''}</b>
						<span class="tl-num">{a.para} · {fechaHora(a.enviadoEn ?? a.creadoEn)}</span>
						<span class="tl-rotulo est" style:color={e.color}>{e.texto}</span>
						{#if a.ultimoError}<span class="error-a">{a.ultimoError}</span>{/if}
						{#if a.turnoId}<a href={`/panel/turnos/${a.turnoId}`}>Ver el turno</a>{/if}
					</span>
					{#if a.estado !== 'pendiente'}
						<form method="POST" action={accion(page.url, 'reenviar')} use:enhance>
							<input type="hidden" name="id" value={a.id} />
							<button>Reenviar</button>
						</form>
					{/if}
				</li>
			{/each}
		</ul>
		{#if data.avisos.length === 50 && ultimo}
			<a class="mas" href={`?${new URLSearchParams({ ...(data.estado ? { estado: data.estado } : {}), antes: ultimo.creadoEn })}`}>Más viejos →</a>
		{/if}
	{/if}
{/if}

<style>
	.chips {
		display: flex;
		gap: 6px;
		overflow-x: auto;
		scrollbar-width: none;
		padding: 2px;
	}
	.vacio {
		margin: 0;
		font-size: 15px;
		color: var(--tl-sub);
	}
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		max-width: 760px;
		border-radius: 20px;
		background: var(--tl-papel);
		overflow: hidden;
		box-shadow: 0 20px 40px -30px rgba(15, 35, 40, 0.5);
	}
	.lista li {
		display: flex;
		align-items: flex-start;
		gap: 12px;
		padding: 14px 16px;
	}
	.lista li + li {
		border-top: 1px solid #f0e8d8;
	}
	.txt {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}
	.txt b {
		font-size: 15px;
	}
	.txt > span {
		font-size: 12px;
		color: var(--tl-sub);
		overflow-wrap: anywhere;
	}
	.est {
		font-size: 11px !important;
	}
	.error-a {
		color: #a3241a !important;
	}
	.txt a,
	.mas {
		font-size: 13px;
		font-weight: 700;
		color: var(--tl-menta-texto);
	}
	.lista button {
		min-height: 40px;
		padding: 0 14px;
		border: 0;
		border-radius: 999px;
		background: var(--tl-crema);
		font: 700 13px var(--tl-font);
		cursor: pointer;
	}
</style>
