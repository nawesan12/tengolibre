<!-- «Así se ve · en vivo»: la cabecera de la página pública con lo que se está editando, al instante. -->
<script lang="ts">
	import { pesos } from '#lib/formato.ts';
	import { textoSobre } from './color.ts';

	let {
		nombre,
		bajada,
		logo,
		portada,
		primario,
		servicios,
		href
	}: {
		nombre: string;
		bajada: string;
		logo: string | null;
		portada: string | null;
		primario: string;
		servicios: { id: string; nombre: string; precio: number }[];
		href: string;
	} = $props();

	const sobre = $derived(textoSobre(primario).color);
	const inicial = $derived((nombre.trim()[0] ?? '·').toUpperCase());
</script>

<div class="vivo">
	<span class="g-kicker gris">Así se ve · <span class="verde">en vivo</span></span>
	<div class="telefono" style:--primario={primario} style:--sobre={sobre}>
		<div class="tapa" class:sin-foto={!portada}>
			{#if portada}<img src={portada} alt="" />{/if}
			<div class="degrade"></div>
			<div class="marca">
				<span class="logo">{#if logo}<img src={logo} alt="" />{:else}{inicial}{/if}</span>
				<span class="nombres">
					<b>{nombre || 'Tu negocio'}</b>
					{#if bajada}<span>{bajada}</span>{/if}
				</span>
			</div>
		</div>
		<div class="cuerpo">
			<b class="servicios">Servicios</b>
			{#each servicios.slice(0, 3) as s (s.id)}
				<div class="renglon"><span>{s.nombre}</span><span class="puntos"></span><b>{pesos(s.precio)}</b></div>
			{:else}
				<a class="vacio" href="/panel/servicios">Todavía no cargaste servicios. Cargalos&nbsp;acá.</a>
			{/each}
			<span class="reservar">Reservar turno</span>
		</div>
	</div>
	<a class="g-accion" {href} target="_blank" rel="noopener">Abrir mi página<span class="icono" aria-hidden="true">open_in_new</span></a>
</div>

<style>
	.vivo {
		position: sticky;
		top: 24px;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.gris {
		color: var(--tl-sub);
	}
	.verde {
		color: var(--tl-menta-texto);
	}
	.telefono {
		margin: 8px;
		border-radius: 28px;
		overflow: hidden;
		background: var(--tl-crema-pagina);
		box-shadow: 0 0 0 8px var(--tl-tinta), 0 30px 50px -30px rgba(15, 35, 40, 0.6);
	}
	.tapa {
		position: relative;
		height: 200px;
		background: var(--primario);
	}
	.tapa.sin-foto {
		height: 150px;
	}
	.tapa img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.degrade {
		position: absolute;
		inset: 0;
		background: linear-gradient(
			to top,
			var(--primario) 4%,
			color-mix(in srgb, var(--primario) 55%, transparent) 38%,
			transparent 72%
		);
	}
	.marca {
		position: absolute;
		left: 14px;
		right: 14px;
		bottom: 14px;
		display: flex;
		align-items: flex-end;
		gap: 10px;
		color: var(--sobre);
	}
	.logo {
		width: 46px;
		height: 46px;
		flex: none;
		border-radius: 13px;
		background: #fff;
		color: var(--primario);
		display: grid;
		place-items: center;
		overflow: hidden;
		font-weight: 700;
		font-size: 20px;
	}
	.logo img {
		width: 100%;
		height: 100%;
		object-fit: contain;
	}
	.nombres {
		display: flex;
		flex-direction: column;
		min-width: 0;
	}
	.nombres b {
		font-size: 26px;
		line-height: 1;
		letter-spacing: -0.05em;
		overflow-wrap: anywhere;
	}
	.nombres span {
		font-size: 11px;
	}
	.cuerpo {
		padding: 14px;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.servicios {
		font-size: 18px;
		letter-spacing: -0.03em;
	}
	.renglon {
		display: flex;
		align-items: baseline;
		gap: 6px;
		font-size: 13px;
	}
	.renglon b {
		font-variant-numeric: tabular-nums;
	}
	.puntos {
		flex: 1;
		min-width: 12px;
		border-bottom: 2px dotted #d6cab2;
	}
	.vacio {
		font-size: 13px;
		color: var(--tl-sub);
	}
	.reservar {
		margin-top: 8px;
		height: 42px;
		border-radius: 999px;
		background: var(--primario);
		color: var(--sobre);
		display: grid;
		place-items: center;
		font-size: 14px;
		font-weight: 700;
	}
	.g-accion .icono {
		font-size: 16px;
	}
</style>
