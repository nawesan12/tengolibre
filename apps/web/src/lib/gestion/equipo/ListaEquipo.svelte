<!--
	La columna del equipo (handoff v2, «Equipo y horarios»): cada profesional con sus iniciales,
	cuántos servicios hace y cuántas horas atiende. Quien no entra al panel lleva una pastilla «Sin
	acceso» (el ícono solo, con el sentido en el title, no se entendía en el teléfono).
-->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { Franja } from '#lib/configuracion/horario.ts';
	import { horasSemana, horasTexto, iniciales } from './semana.ts';

	type P = {
		id: string;
		nombre: string;
		especialidad: string;
		activo: boolean;
		/** Activo y dentro del tope del plan (false: bajaron de plan; no toma turnos, no se borró). */
		enPlan?: boolean;
		miembroId: string | null;
		servicios: string[];
		horario: Franja[];
	};
	let { profesionales, actual, children }: { profesionales: P[]; actual?: string; children?: Snippet } = $props();

	const sub = (p: P) => {
		const s = p.servicios.length;
		const h = horasSemana(p.horario);
		// Espacios duros dentro de cada parte («48 h/sem» no se corta) y antes de la última: el corte
		// cae en un «·».
		const partes = [p.especialidad, s === 1 ? '1 servicio' : `${s} servicios`, h ? `${horasTexto(h)} h/sem` : 'sin horario']
			.filter(Boolean)
			.map((x) => x.replaceAll(' ', '\u00a0'));
		return partes.join(' · ').replace(/ · (?=[^·]*$)/, '\u00a0·\u00a0');
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
					<span
						>{sub(p)}{#if !p.activo} · <span class="pausa">pausado</span>{:else if p.enPlan === false} · <span class="pausa">fuera de tu plan</span>{/if}</span
					>
				</span>
				{#if !p.miembroId}<span class="sin-acceso">Sin acceso</span>{/if}
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
	.sin-acceso {
		flex: none;
		padding: 3px 10px;
		border-radius: 999px;
		background: rgba(15, 35, 40, 0.06);
		color: var(--tl-sub);
		font-size: 12px;
		font-weight: 700;
		white-space: nowrap;
	}
</style>
