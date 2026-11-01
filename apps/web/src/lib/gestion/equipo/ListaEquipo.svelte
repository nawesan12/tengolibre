<!--
	La columna del equipo (handoff v2, «Equipo y horarios»): cada profesional con sus iniciales,
	cuántos servicios hace y cuántas horas atiende; el ícono dice si tiene cuenta en el panel.
-->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { Franja } from '#lib/configuracion/horario.ts';
	import { horasSemana, horasTexto, iniciales } from './semana.ts';

	type P = { id: string; nombre: string; especialidad: string; activo: boolean; miembroId: string | null; servicios: string[]; horario: Franja[] };
	let { profesionales, actual, children }: { profesionales: P[]; actual?: string; children?: Snippet } = $props();

	const sub = (p: P) => {
		const s = p.servicios.length;
		const h = horasSemana(p.horario);
		return [p.especialidad, s === 1 ? '1 servicio' : `${s} servicios`, h ? `${horasTexto(h)} h/sem` : 'sin horario'].filter(Boolean).join(' · ');
	};
</script>

<ul class="equipo">
	{#each profesionales as p (p.id)}
		<li>
			<a
				class="persona"
				class:actual={p.id === actual}
				class:pausado={!p.activo}
				href={`/panel/profesionales/${p.id}`}
				aria-current={p.id === actual ? 'page' : undefined}
			>
				<span class="g-avatar" class:menta={p.id === actual} aria-hidden="true">{iniciales(p.nombre)}</span>
				<span class="texto">
					<b>{p.nombre}</b>
					<span>{sub(p)}{#if !p.activo} · <span class="pausa">pausado</span>{/if}</span>
				</span>
				<span
					class="icono vinculo"
					class:sin={!p.miembroId}
					role="img"
					aria-label={p.miembroId ? 'Entra al panel' : 'Sin cuenta del panel'}
					title={p.miembroId ? 'Entra al panel' : 'Sin cuenta del panel'}>{p.miembroId ? 'link' : 'link_off'}</span
				>
			</a>
		</li>
	{/each}
</ul>
{#if children}{@render children()}{/if}

<style>
	.equipo {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.persona {
		display: flex;
		align-items: center;
		gap: 12px;
		min-height: 68px;
		padding: 12px;
		border-radius: 16px;
		background: var(--tl-papel);
		color: inherit;
		text-decoration: none;
		transition: transform var(--tl-tocar) var(--tl-ease);
	}
	.persona:hover {
		transform: translateY(-1px);
	}
	.persona.actual {
		box-shadow: 0 0 0 2px var(--tl-petroleo);
	}
	.persona.pausado .texto b {
		color: var(--tl-sub);
	}
	.g-avatar.menta {
		background: var(--tl-menta-suave);
	}
	.texto {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}
	.texto b {
		font-size: 15px;
	}
	.texto > span {
		font-size: 12px;
		color: var(--tl-sub);
	}
	.pausa {
		font-weight: 700;
		color: #7a4a00;
	}
	.vinculo {
		font-size: 18px;
		color: var(--tl-menta-texto);
	}
	.vinculo.sin {
		color: var(--tl-sub);
	}
</style>
