<!--
	La caja (Propuestas 07 y 08): al marcar «Atendido» se abre con lo que resta cobrar (el servicio
	menos la seña pagada), lo que se llevó y cómo paga. «Cobrar después» sólo lo marca atendido.
-->
<script lang="ts">
	import { enhance } from '$app/forms';
	import Icono from '#lib/ui/Icono.svelte';
	import { pesos } from '#lib/formato.ts';
	import { MEDIOS, type Medio, type Producto } from './contrato.ts';
	import type { Turno } from './tipos.ts';

	let { t, productos = [], alCerrar }: { t: Turno; productos?: Producto[]; alCerrar: () => void } = $props();

	let medio = $state<Medio>('efectivo');
	let elegidos = $state<string[]>([]);
	let enviando = $state(false);
	const sena = $derived(t.sena?.estado === 'pagada' ? t.sena.monto : 0);
	const extras = $derived(productos.filter((p) => elegidos.includes(p.id)));
	const total = $derived(Math.max(0, t.precio - sena) + extras.reduce((s, p) => s + p.precio, 0));
	const cerrarAl = () => {
		enviando = true;
		return async ({ result, update }: { result: { type: string }; update: (o?: { reset?: boolean }) => Promise<void> }) => {
			await update({ reset: false });
			enviando = false;
			if (result.type === 'success') alCerrar();
		};
	};
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && alCerrar()} />

<button class="velo" aria-label="Cerrar la caja" onclick={alCerrar}></button>
<aside class="caja" aria-label="Cobrar">
	<div class="cabeza tl-petroleo">
		<div class="fila-c">
			<span class="tl-rotulo rot">Atendida · {t.cliente.nombre}</span>
			<button class="cerrar" aria-label="Cerrar" onclick={alCerrar}><Icono nombre="close" tam={22} /></button>
		</div>
		<span class="resta">Resta cobrar</span>
		<b class="total tl-num">{pesos(total)}</b>
	</div>
	<div class="troquel" aria-hidden="true"><span></span></div>
	<form method="POST" action="?/cobrar" class="cuerpo" id="form-caja" use:enhance={cerrarAl}>
		<input type="hidden" name="id" value={t.id} />
		<div class="linea"><span>{t.servicio}</span><span class="tl-num">{pesos(t.precio)}</span></div>
		{#if sena}<div class="linea verde"><span>Seña pagada (Mercado Pago)</span><span class="tl-num">− {pesos(sena)}</span></div>{/if}
		{#each extras as p (p.id)}<div class="linea"><span>{p.nombre}</span><span class="tl-num">{pesos(p.precio)}</span></div>{/each}

		{#if productos.length}
			<span class="tl-rotulo sub">¿Se llevó algo?</span>
			<div class="productos">
				{#each productos as p (p.id)}
					<label class="prod" class:on={elegidos.includes(p.id)}>
						<input type="checkbox" name="productos" value={p.id} bind:group={elegidos} />
						{p.nombre}<span class="tl-num">{pesos(p.precio)}</span>
					</label>
				{/each}
			</div>
		{/if}

		<span class="tl-rotulo sub">Cómo paga</span>
		<div class="medios" role="radiogroup" aria-label="Cómo paga">
			{#each MEDIOS as m (m.clave)}
				<label class="medio" class:on={medio === m.clave}>
					<input type="radio" name="medio" value={m.clave} bind:group={medio} />
					<Icono nombre={m.icono} tam={22} />{m.texto}
				</label>
			{/each}
		</div>
	</form>
	<div class="pie">
		<button class="cobrar" form="form-caja" disabled={enviando}>Cobrar {pesos(total)}</button>
		<form method="POST" action="?/estado" use:enhance={cerrarAl}>
			<input type="hidden" name="id" value={t.id} />
			<button class="despues" name="estado" value="atendido" disabled={enviando}>Cobrar después</button>
		</form>
	</div>
</aside>

<style>
	.velo {
		position: fixed;
		inset: 0;
		z-index: 40;
		border: 0;
		background: rgba(15, 35, 40, 0.35);
		backdrop-filter: blur(2px);
		animation: fundido 250ms both;
	}
	@keyframes fundido {
		from {
			opacity: 0;
		}
	}
	@keyframes entra {
		from {
			opacity: 0;
			transform: translateX(48px);
		}
	}
	.caja {
		position: fixed;
		top: 0;
		right: 0;
		bottom: 0;
		z-index: 41;
		width: 480px;
		max-width: 100vw;
		border-radius: 24px 0 0 24px;
		background: var(--tl-papel);
		box-shadow: -30px 0 60px -30px rgba(15, 35, 40, 0.5);
		display: flex;
		flex-direction: column;
		overflow: hidden;
		animation: entra 320ms cubic-bezier(0.2, 0.8, 0.2, 1) both;
	}
	.cabeza {
		padding: 18px 20px 22px;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	.fila-c {
		display: flex;
		align-items: center;
	}
	.rot {
		flex: 1;
		font-size: 11px;
		color: var(--tl-menta);
	}
	.cerrar {
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
	.resta {
		font-size: 14px;
		color: rgba(255, 255, 255, 0.8);
	}
	.total {
		font-size: 52px;
		font-weight: 700;
		letter-spacing: -0.06em;
		line-height: 1;
	}
	.troquel {
		height: 18px;
		flex: none;
		position: relative;
		background: linear-gradient(var(--tl-petroleo) 0 50%, var(--tl-papel) 50%);
	}
	.troquel::before,
	.troquel::after {
		content: '';
		position: absolute;
		top: 0;
		width: 18px;
		height: 18px;
		border-radius: 50%;
		background: rgba(15, 35, 40, 0.35);
	}
	.troquel::before {
		left: -9px;
	}
	.troquel::after {
		right: -9px;
	}
	.troquel span {
		position: absolute;
		left: 16px;
		right: 16px;
		top: 8px;
		border-top: 2px dashed var(--tl-linea-2);
	}
	.cuerpo {
		flex: 1;
		overflow-y: auto;
		padding: 6px 20px 20px;
		display: flex;
		flex-direction: column;
		gap: 12px;
		font-size: 14px;
	}
	.linea {
		display: flex;
		justify-content: space-between;
		gap: 10px;
	}
	.linea.verde {
		color: var(--tl-menta-texto);
	}
	.sub {
		font-size: 11px;
		color: var(--tl-sub);
		margin-top: 8px;
	}
	.productos {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
	}
	.prod,
	.medio {
		position: relative;
		cursor: pointer;
	}
	.prod input,
	.medio input {
		position: absolute;
		opacity: 0;
		pointer-events: none;
	}
	.prod {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		min-height: 40px;
		padding: 0 14px;
		border-radius: 999px;
		background: #fff;
		box-shadow: inset 0 0 0 1.5px #d8ccb4;
		font-size: 13px;
		font-weight: 700;
	}
	.prod span {
		opacity: 0.8;
	}
	.prod.on,
	.medio.on {
		background: var(--tl-petroleo);
		color: #fff;
		box-shadow: none;
	}
	.prod:has(input:focus-visible),
	.medio:has(input:focus-visible) {
		outline: 3px solid var(--tl-menta);
		outline-offset: 2px;
	}
	.medios {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 6px;
	}
	.medio {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 4px;
		min-height: 64px;
		border-radius: 14px;
		background: var(--tl-crema);
		font-size: 12px;
		font-weight: 700;
	}
	.pie {
		flex: none;
		display: flex;
		gap: 8px;
		padding: 14px 20px 18px;
		border-top: 1px solid #f0e8d8;
	}
	.cobrar {
		flex: 1;
		height: 52px;
		border-radius: 999px;
		border: 0;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		font: 700 16px var(--tl-font);
		cursor: pointer;
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.45), 0 12px 24px -10px rgba(60, 204, 144, 0.8);
	}
	.despues {
		height: 52px;
		padding: 0 16px;
		border: 0;
		border-radius: 999px;
		background: transparent;
		font: 700 14px var(--tl-font);
		color: var(--tl-sub);
		cursor: pointer;
	}
	@media (max-width: 859px) {
		.caja {
			width: 100vw;
			border-radius: 0;
		}
		.troquel::before,
		.troquel::after {
			background: transparent;
		}
	}
</style>
