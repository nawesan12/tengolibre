<!--
	La página del negocio en chiquito (Onboarding v2): los bloques de cada plantilla con el color, el
	estilo y los servicios que se van cargando. La usan el teléfono «Tu página, en vivo» y las
	miniaturas del paso de plantillas. Las fotos son las de muestra de cada plantilla.
-->
<script lang="ts">
	import type { Borrador } from './estado.svelte.ts';
	import { duracion, pesos } from './rubros.ts';

	let { b, plantilla, boton = 'Reservar turno' }: { b: Borrador; plantilla?: string; boton?: string } = $props();

	const F = {
		crema: ['#faf6ee', '#0f2328', '#4a5d61', '#e2d6bf'],
		blanco: ['#ffffff', '#0f2328', '#4a5d61', '#e6e9e8'],
		noche: ['#14221f', '#f4eee2', '#b9c4c1', '#2c3d39']
	} as Record<string, string[]>;
	const T = { grandes: ['700', '-0.06em', 1], medianos: ['700', '-0.03em', 0.82], livianos: ['400', '-0.01em', 0.78] } as Record<string, [string, string, number]>;
	const FM = { redondos: ['999px', '14px', '20px'], suaves: ['12px', '12px', '14px'], rectos: ['4px', '4px', '6px'] } as Record<string, string[]>;

	const mezclar = (a: string, c: string, t: number) =>
		'#' +
		[0, 2, 4]
			.map((i) =>
				Math.round(parseInt(a.slice(1 + i, 3 + i), 16) * (1 - t) + parseInt(c.slice(1 + i, 3 + i), 16) * t)
					.toString(16)
					.padStart(2, '0')
			)
			.join('');

	const plan = $derived(plantilla ?? b.estilo.plantilla);
	const f = $derived(F[b.estilo.fondo] ?? F.crema);
	const t = $derived(T[b.estilo.titulos] ?? T.grandes);
	const fm = $derived(FM[b.estilo.forma] ?? FM.redondos);
	const conFoto = $derived(b.estilo.portada === 'foto');
	const color = $derived(b.color);
	const tint = $derived(mezclar(color, '#ffffff', 0.82));
	const srvs = $derived(b.activos.slice(0, 4));
	const rubro = $derived(b.rubroInfo?.nombre ?? 'Tu rubro');
	const t1 = $derived(Math.round(24 * t[2]) + 'px');
	const t2 = $derived(Math.round(32 * t[2]) + 'px');
	const t3 = $derived(Math.round(48 * t[2]) + 'px');

	type Bloque =
		| { k: 'foto'; img: string; alto: string; velo: 'color' | 'oscuro'; centro?: boolean; boton?: boolean }
		| { k: 'color' | 'carta' | 'lista' | 'editorial' | 'mosaico' | 'bento' | 'chips' };
	const bloques = $derived.by((): Bloque[] => {
		const foto = (img: string, alto: string, velo: 'color' | 'oscuro', centro = false, boton = false): Bloque =>
			conFoto ? { k: 'foto', img, alto, velo, centro, boton } : { k: 'color' };
		switch (plan) {
			case 'vidriera':
				return [foto('peluquera-con-clienta.jpg', '300px', 'oscuro', true, true), { k: 'lista' }];
			case 'sobria':
				return [{ k: 'color' }, { k: 'lista' }];
			case 'galeria':
				return [conFoto ? { k: 'mosaico' } : { k: 'color' }, { k: 'chips' }, { k: 'lista' }];
			case 'editorial':
				return [{ k: 'editorial' }, { k: 'carta' }];
			case 'bloques':
				return [{ k: 'bento' }, { k: 'lista' }];
			default:
				return [foto('manicura.jpg', '180px', 'color'), { k: 'carta' }];
		}
	});
</script>

<div
	class="previa"
	style:--bg={f[0]}
	style:--ink={f[1]}
	style:--sub={f[2]}
	style:--line={f[3]}
	style:--c={color}
	style:--tint={tint}
	style:--tw={t[0]}
	style:--tl={t[1]}
	style:--br={fm[0]}
	style:--lr={fm[1]}
	style:--cr={fm[2]}
>
	<div class="bloques">
		{#each bloques as bl, i (i)}
			{#if bl.k === 'foto'}
				<div class="foto" style:height={bl.alto}>
					<span class="foto-img" style:background-image="url(/fotos/{bl.img})"></span>
					<span class="velo velo-{bl.velo}"></span>
					<span class="foto-pie" class:centro={bl.centro}>
						<span class="ini">{b.inicial}</span>
						<b style:font-size={t1}>{b.visible}</b>
						{#if bl.boton}<span class="mini-boton">Reservar</span>{/if}
					</span>
				</div>
			{:else if bl.k === 'color'}
				<div class="color">
					<span class="aro" aria-hidden="true"></span>
					<span class="ini grande">{b.inicial}</span>
					<b style:font-size={t2}>{b.visible}</b>
					<span class="color-sub">{rubro} · Reservá online</span>
				</div>
			{:else if bl.k === 'editorial'}
				<div class="editorial">
					<span class="ed-rubro">{rubro}</span>
					<b style:font-size={t3}>{b.visible}.</b>
					<span class="ed-fila"
						><img src="/fotos/peluquera-con-clienta.jpg" alt="" /><span>Turnos sin espera, con reserva&nbsp;online.</span></span
					>
				</div>
			{:else if bl.k === 'mosaico'}
				<div class="mosaico">
					<img src="/fotos/manicura.jpg" alt="" class="alto" />
					<img src="/fotos/recepcion.jpg" alt="" />
					<img src="/fotos/peluquera-con-clienta.jpg" alt="" />
				</div>
				<div class="mosaico-n"><span class="ini chica">{b.inicial}</span><b>{b.visible}</b></div>
			{:else if bl.k === 'bento'}
				<div class="bento">
					<span class="bento-a"><span class="ini">{b.inicial}</span><b style:font-size={t1}>{b.visible}</b></span>
					<img src="/fotos/barbero-cortando.jpg" alt="" />
					<span class="bento-c"><b>{b.activos.length}</b>{b.activos.length === 1 ? 'servicio' : 'servicios'}</span>
				</div>
			{:else if bl.k === 'chips'}
				<div class="chips">
					<span class="on">Todo</span>
					{#each srvs.slice(0, 2) as s (s.nombre)}<span>{s.nombre}</span>{/each}
				</div>
			{:else if bl.k === 'carta'}
				<div class="carta">
					<b class="sec">Servicios</b>
					{#each srvs as s (s.nombre)}
						<span class="carta-f"><span class="n">{s.nombre}</span><span class="puntos"></span><b>{pesos(s.precio)}</b></span>
					{/each}
				</div>
			{:else if bl.k === 'lista'}
				<div class="lista">
					{#each srvs as s (s.nombre)}
						<span class="lista-f"
							><span class="lista-n"><b>{s.nombre}</b><span>{duracion(s.duracion)}</span></span><b>{pesos(s.precio)}</b></span
						>
					{/each}
				</div>
			{/if}
		{/each}
	</div>
	<span class="reservar">{boton}</span>
</div>

<style>
	.previa {
		height: 100%;
		display: flex;
		flex-direction: column;
		background: var(--bg);
		color: var(--ink);
		transition: background 0.3s;
		font-size: 13px;
	}
	.bloques {
		flex: 1;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}
	.bloques > * {
		flex: none;
	}
	b {
		font-weight: 700;
	}
	.ini {
		width: 42px;
		height: 42px;
		flex: none;
		border-radius: var(--lr);
		background: #fff;
		color: var(--c);
		display: grid;
		place-items: center;
		font-weight: 700;
		font-size: 18px;
	}
	.ini.grande {
		position: relative;
		width: 46px;
		height: 46px;
		font-size: 20px;
	}
	.ini.chica {
		width: 34px;
		height: 34px;
		font-size: 15px;
		background: var(--c);
		color: #fff;
	}
	.foto {
		position: relative;
		background: var(--c);
	}
	.foto-img {
		display: block;
		width: 100%;
		height: 100%;
		background-size: cover;
		background-position: 50% 40%;
	}
	.velo {
		position: absolute;
		inset: 0;
	}
	.velo-color {
		background: linear-gradient(to top, var(--c) 6%, color-mix(in srgb, var(--c) 60%, transparent) 40%, transparent 78%);
	}
	.velo-oscuro {
		background: linear-gradient(to top, rgba(15, 35, 40, 0.82), rgba(15, 35, 40, 0.1) 60%);
	}
	.foto-pie {
		position: absolute;
		left: 14px;
		right: 14px;
		bottom: 14px;
		display: flex;
		align-items: flex-end;
		gap: 10px;
		color: #fff;
	}
	.foto-pie.centro {
		flex-direction: column;
		align-items: center;
		text-align: center;
	}
	.foto-pie b,
	.color b,
	.editorial b,
	.bento-a b,
	.mosaico-n b,
	.sec {
		font-weight: var(--tw);
		letter-spacing: var(--tl);
		line-height: 1;
	}
	.mini-boton {
		height: 36px;
		padding: 0 18px;
		border-radius: var(--br);
		background: var(--c);
		display: grid;
		place-items: center;
		font-size: 13px;
		font-weight: 700;
	}
	.color {
		position: relative;
		overflow: hidden;
		padding: 44px 16px 22px;
		background: var(--c);
		color: #fff;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.color b {
		position: relative;
		line-height: 0.95;
	}
	.aro {
		position: absolute;
		right: -50px;
		top: -50px;
		width: 170px;
		height: 170px;
		border-radius: 50%;
		box-shadow: inset 0 0 0 28px rgba(255, 255, 255, 0.08);
	}
	.color-sub {
		position: relative;
		font-size: 12px;
		opacity: 0.85;
	}
	.editorial {
		padding: 30px 16px 8px;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.ed-rubro {
		font-size: 12px;
		font-weight: 700;
		color: var(--c);
	}
	.editorial b {
		line-height: 0.86;
	}
	.ed-fila {
		display: flex;
		align-items: center;
		gap: 12px;
		font-size: 12px;
		line-height: 1.4;
		color: var(--sub);
	}
	.ed-fila img {
		width: 68px;
		height: 68px;
		flex: none;
		border-radius: 50%;
		object-fit: cover;
		box-shadow: 0 0 0 4px var(--tint);
	}
	.mosaico {
		display: grid;
		grid-template-columns: 2fr 1fr;
		grid-template-rows: 92px 92px;
		gap: 3px;
	}
	.mosaico img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.mosaico .alto {
		grid-row: span 2;
	}
	.mosaico-n {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 14px 14px 0;
	}
	.mosaico-n b {
		font-size: 19px;
	}
	.bento {
		display: grid;
		grid-template-columns: 1fr 1fr;
		grid-auto-rows: 78px;
		gap: 6px;
		padding: 8px;
	}
	.bento-a {
		grid-column: span 2;
		grid-row: span 2;
		border-radius: var(--cr);
		background: var(--c);
		color: #fff;
		padding: 14px;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
	}
	.bento-a .ini {
		width: 38px;
		height: 38px;
	}
	.bento img {
		width: 100%;
		height: 100%;
		object-fit: cover;
		border-radius: var(--cr);
	}
	.bento-c {
		border-radius: var(--cr);
		background: var(--tint);
		padding: 10px;
		display: flex;
		flex-direction: column;
		justify-content: flex-end;
		font-size: 12px;
		color: #0f2328;
	}
	.bento-c b {
		font-size: 19px;
		font-variant-numeric: tabular-nums;
	}
	.chips {
		display: flex;
		gap: 6px;
		padding: 14px 14px 0;
		overflow: hidden;
	}
	.chips span {
		flex: none;
		height: 30px;
		padding: 0 12px;
		border-radius: var(--br);
		box-shadow: inset 0 0 0 1px var(--line);
		display: grid;
		place-items: center;
		font-size: 12px;
		font-weight: 700;
	}
	.chips .on {
		background: var(--c);
		color: #fff;
		box-shadow: none;
	}
	.carta {
		display: flex;
		flex-direction: column;
		gap: 10px;
		padding: 16px 14px 8px;
	}
	.sec {
		font-size: 17px;
	}
	.carta-f {
		display: flex;
		align-items: baseline;
		gap: 6px;
	}
	.carta-f .n,
	.lista-n b {
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.puntos {
		flex: 1;
		min-width: 12px;
		border-bottom: 2px dotted var(--line);
	}
	.carta-f b,
	.lista-f > b {
		font-variant-numeric: tabular-nums;
		white-space: nowrap;
	}
	.lista {
		display: flex;
		flex-direction: column;
		padding: 8px 14px;
	}
	.lista-f {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 8px;
		padding: 10px 0;
		border-bottom: 1px solid var(--line);
	}
	.lista-n {
		display: flex;
		flex-direction: column;
		min-width: 0;
	}
	.lista-n span {
		font-size: 11px;
		color: var(--sub);
	}
	.reservar {
		margin: 8px 14px 18px;
		height: 46px;
		flex: none;
		border-radius: var(--br);
		background: var(--c);
		color: #fff;
		display: grid;
		place-items: center;
		font-size: 15px;
		font-weight: 700;
		transition:
			background 0.3s,
			border-radius 0.3s;
	}
</style>
