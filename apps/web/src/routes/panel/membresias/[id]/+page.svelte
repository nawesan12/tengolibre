<script lang="ts">
	import '#lib/gestion/gestion.css';
	import { enhance } from '$app/forms';
	import { confirmar } from '#lib/configuracion/formularios.ts';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';
	import { cuandoCorto, fechaCorta } from '#lib/gestion/cobros/formato.ts';
	import type { Credito } from './+page.server.ts';

	let { data, form } = $props();
	const s = $derived(data.socio);

	function movimiento(c: Credito) {
		if (c.cantidad < 0 && c.turnoId) return { que: `Usó ${-c.cantidad}`, color: '#8a5a00' };
		if (c.cantidad < 0) return { que: `${c.cantidad}`.replace('-', '−'), color: 'var(--tl-petroleo)' };
		if (!c.clerkId) return { que: `+${c.cantidad}`, color: 'var(--tl-menta-texto)' };
		return { que: `+${c.cantidad}`, color: 'var(--tl-petroleo)' };
	}
	// «HOY» o «19 OCT»: en una línea de tiempo alcanza con el día.
	const cuando = (i: string) => {
		const c = cuandoCorto(i);
		return c.startsWith('HOY') ? 'HOY' : fechaCorta(i).toUpperCase();
	};
</script>

<svelte:head><title>{s.cliente} · Membresías · Panel</title></svelte:head>

<div class="g-pagina ficha">
	<a class="g-accion neutro volver" href="/panel/membresias"><span class="icono" aria-hidden="true">arrow_back</span>Membresías</a>
	<div class="g-col cabeza">
		<span class="g-kicker">{s.plan} · {s.estado === 'activa' ? 'vence' : s.estado === 'vencida' ? 'venció' : 'cancelada, vencía'} {fechaCorta(s.hasta)}</span>
		<h1 class="g-titulo-2">{s.cliente}</h1>
	</div>
	<Mensaje {form} que="socio" />

	<p class="g-aviso socios">
		<span class="icono lleno" aria-hidden="true">card_membership</span>
		<span>
			{#if s.estado === 'cancelada'}
				La membresía está cancelada. Podés reactivarla cuando&nbsp;quieras.
			{:else if s.visitasDelPlan === 0}
				Tiene visitas libres hasta el {fechaCorta(s.hasta)}. La usa sola al reservar online con su&nbsp;email.
			{:else}
				Le quedan {s.quedan} de {s.visitasDelPlan} visitas. La usa sola al reservar online con su&nbsp;email.
			{/if}
		</span>
	</p>

	<section class="g-col" aria-labelledby="t-mov">
		<h2 id="t-mov" class="g-rotulo">Movimientos</h2>
		{#if s.creditos.length === 0}
			<p class="g-nota">Todavía no hay movimientos.</p>
		{:else}
			<ol class="g-lineas">
				{#each s.creditos as c, i (i)}
					{@const m = movimiento(c)}
					<li class="g-linea">
						<span>{cuando(c.creadoEn)}</span>
						<span class="g-punto" style:background={m.color} aria-hidden="true"></span>
						<span><b>{m.que}</b> {c.motivo}</span>
					</li>
				{/each}
			</ol>
		{/if}
	</section>

	{#if s.estado !== 'cancelada' && s.visitasDelPlan > 0}
		<section class="g-col" aria-labelledby="t-ajustar">
			<h2 id="t-ajustar" class="g-bloque-titulo">Ajustar visitas</h2>
			<Mensaje {form} que="ajustar" />
			<form method="POST" action="?/ajustar" use:enhance class="ajuste">
				<label class="g-campo">
					Visitas (+ suma, − resta)
					<input name="ajustar" type="number" min="-100" max="100" required placeholder="+1" />
				</label>
				<label class="g-campo">
					Motivo
					<input name="motivo" required maxlength="120" placeholder="La atendimos tarde" />
				</label>
				<button class="g-boton oscuro">Aplicar</button>
			</form>
		</section>
	{/if}

	<div class="acciones">
		{#if s.estado === 'cancelada'}
			<form method="POST" action="?/estado" use:enhance>
				<input type="hidden" name="estado" value="activa" />
				<button class="g-cta oscuro">Reactivar</button>
			</form>
		{:else}
			<form method="POST" action="?/renovar" use:enhance={confirmar(`¿Renovar la membresía de ${s.cliente} desde hoy?`)}>
				<button class="g-cta oscuro">Renovar</button>
			</form>
			<form method="POST" action="?/estado" use:enhance={confirmar(`¿Cancelar la membresía de ${s.cliente}?`)}>
				<input type="hidden" name="estado" value="cancelada" />
				<button class="g-accion peligro cancelar">Cancelar la membresía</button>
			</form>
		{/if}
	</div>
</div>

<style>
	.ficha {
		max-width: 560px;
	}
	.volver {
		align-self: flex-start;
	}
	.volver .icono {
		font-size: 18px;
	}
	.cabeza {
		gap: 8px;
	}
	.ajuste {
		display: grid;
		grid-template-columns: 140px minmax(0, 1fr) auto;
		gap: 10px;
		align-items: end;
	}
	.acciones {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 4px;
		margin-top: 16px;
	}
	.acciones form:first-child {
		width: 100%;
	}
	.cancelar {
		min-height: 48px;
		font-size: 15px;
	}
	@media (max-width: 640px) {
		.ajuste {
			grid-template-columns: minmax(0, 1fr);
		}
	}
</style>
