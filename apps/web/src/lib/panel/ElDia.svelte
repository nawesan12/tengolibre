<!--
	«El día» (firma 3 del handoff): una fila por profesional con su franja de hoy. Lo ocupado en
	petróleo, lo libre rayado, lo que ya pasó apagado; la hora actual en una pestaña menta. Lleva a los
	huecos de hoy.
-->
<script lang="ts">
	import { libresTexto, primerNombre, type DiaProfesional } from './dia.ts';

	let {
		dias,
		rango,
		ahora,
		ahoraTexto,
		esProfesional = false
	}: { dias: DiaProfesional[]; rango: [number, number]; ahora: number; ahoraTexto: string; esProfesional?: boolean } = $props();

	const [a, b] = $derived(rango);
	const pct = (m: number) => `${(((Math.min(Math.max(m, a), b) - a) / (b - a)) * 100).toFixed(2)}%`;
	const ancho = (x: number, y: number) => `${(((Math.min(y, b) - Math.max(x, a)) / (b - a)) * 100).toFixed(2)}%`;
	const fr = $derived(Math.min(1, Math.max(0, (ahora - a) / (b - a))));
	const libres = $derived(dias.reduce((s, d) => s + d.libres, 0));
	const conHorario = $derived(dias.filter((d) => d.franjas.length > 0 || d.ocupado.length > 0));
	/** Lo que no es horario de trabajo (antes, entre y después de las franjas). */
	function cerrado(d: DiaProfesional) {
		const out: [number, number][] = [];
		let ini = a;
		for (const f of d.franjas) {
			if (f.desde > ini) out.push([ini, f.desde]);
			ini = Math.max(ini, f.hasta);
		}
		if (ini < b) out.push([ini, b]);
		return out;
	}
	const medio = $derived(Math.round((a + b) / 2 / 60) * 60);
	const resumen = $derived(
		conHorario.length === 0
			? 'Hoy no atiende nadie.'
			: `Hoy quedan ${libresTexto(libres)} libres. ` + conHorario.map((d) => `${d.nombre}: ${libresTexto(d.libres)}`).join(', ') + '.'
	);
</script>

<a class="dia" href="/panel/turnos?huecos=1" aria-label={`${resumen} Ver los huecos de hoy`}>
	<span class="cabeza"><b>Hoy</b><span class="tl-num">{a / 60} · {b / 60} h</span></span>
	<span class="cuerpo" aria-hidden="true">
		<span class="pestania"><b class="tl-num">{ahoraTexto}</b><span>ahora</span></span>
		<span class="caja">
			{#each conHorario as d (d.id)}
				<span class="carril">
					<span class="quien">{primerNombre(d.nombre)}</span>
					<span class="barra">
						{#each cerrado(d) as [x, y], i (i)}<span class="seg cerrado" style:left={pct(x)} style:width={ancho(x, y)}></span>{/each}
						{#each d.ocupado as o (o.turno.id)}<span class="seg ocupado" style:left={pct(o.desde)} style:width={ancho(o.desde, o.hasta)}></span>{/each}
						<span class="seg pasado" style:left="0" style:width={pct(ahora)}></span>
					</span>
				</span>
			{:else}
				<span class="nadie">Hoy no atiende nadie</span>
			{/each}
			{#if conHorario.length}<span class="linea" style:left={`calc(58px + (100% - 70px) * ${fr.toFixed(4)})`}></span>{/if}
			<span class="horas tl-num"><span>{a / 60}</span><span>{medio / 60}</span><span>{b / 60}</span></span>
		</span>
		{#if conHorario.length}<span class="libres tl-num">{libres ? `${libresTexto(libres)} libres` : 'Sin huecos'}</span>{/if}
	</span>
	<span class="leyenda">
		{esProfesional ? 'Tu día. Lo rayado está libre.' : 'Una fila por profesional. Lo rayado está libre: tocá para ver los huecos.'}
	</span>
</a>

<style>
	.dia {
		position: relative;
		margin: auto 14px 0 0;
		flex: none;
		padding: 16px 12px 12px;
		border-radius: 18px;
		background: rgba(16, 43, 49, 0.6);
		box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.06);
		color: #fff;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 10px;
		text-decoration: none;
		transition: background var(--tl-tocar) var(--tl-ease);
	}
	.dia:hover {
		background: rgba(16, 43, 49, 0.8);
	}
	.cabeza {
		display: flex;
		width: 100%;
		justify-content: space-between;
		align-items: baseline;
		padding: 0 2px;
	}
	.cabeza b {
		font-size: 13px;
		font-weight: 700;
	}
	.cabeza span {
		font-size: 11px;
		color: rgba(255, 255, 255, 0.6);
	}
	.cuerpo {
		position: relative;
		width: 100%;
		display: block;
		padding-top: 30px;
	}
	.pestania {
		position: absolute;
		left: 50%;
		top: 0;
		z-index: 2;
		width: 96px;
		height: 62px;
		margin-left: -48px;
		border-radius: 16px;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		box-shadow: 0 0 0 6px #15323a;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		line-height: 1.1;
	}
	.pestania b {
		font: 900 21px var(--tl-font);
		letter-spacing: -0.02em;
	}
	.pestania span {
		font-size: 11px;
		font-weight: 700;
	}
	.caja {
		position: relative;
		display: flex;
		flex-direction: column;
		gap: 8px;
		padding: 44px 12px 14px;
		border-radius: 24px;
		background: var(--tl-tinta);
	}
	.carril {
		display: grid;
		grid-template-columns: 38px minmax(0, 1fr);
		align-items: center;
		gap: 8px;
	}
	.quien {
		font-size: 11px;
		font-weight: 700;
		color: rgba(255, 255, 255, 0.75);
		text-align: left;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.barra {
		position: relative;
		height: 16px;
		border-radius: 6px;
		overflow: hidden;
		background: repeating-linear-gradient(-45deg, #3ccc90 0 3px, rgba(60, 204, 144, 0.35) 3px 7px);
	}
	.seg {
		position: absolute;
		top: 0;
		bottom: 0;
	}
	.ocupado {
		background: #2c5059;
	}
	.cerrado {
		background: var(--tl-petroleo-hondo);
	}
	.pasado {
		background: rgba(15, 35, 40, 0.62);
	}
	.linea {
		position: absolute;
		top: 36px;
		bottom: 10px;
		width: 2px;
		margin-left: -1px;
		border-radius: 1px;
		background: #fff;
		box-shadow: 0 0 0 2px rgba(60, 204, 144, 0.35);
	}
	.horas {
		display: flex;
		justify-content: space-between;
		padding-left: 46px;
		font-size: 11px;
		color: rgba(255, 255, 255, 0.55);
	}
	.nadie {
		font-size: 12px;
		color: rgba(255, 255, 255, 0.7);
		text-align: center;
	}
	.libres {
		display: block;
		margin-top: 10px;
		font-size: 15px;
		font-weight: 900;
		color: var(--tl-menta);
		text-align: center;
	}
	.leyenda {
		font-size: 11px;
		color: rgba(255, 255, 255, 0.62);
		text-align: center;
		line-height: 1.35;
		text-wrap: balance;
	}
</style>
