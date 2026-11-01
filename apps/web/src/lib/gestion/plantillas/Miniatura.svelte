<!--
	Miniatura de una plantilla de la web pública (handoff «Gestion secciones» 1m), armada con lo real
	del negocio: nombre, colores, hasta tres servicios y sus fotos. Sin fotos, el lugar de la foto va
	rayado en el color del negocio (nunca una foto que no es suya).
-->
<script lang="ts">
	import { pesos } from '#lib/formato.ts';
	import { textoSobre, type Plantilla } from '#lib/gestion/plantillas.ts';
	import { RUBROS } from '#lib/configuracion/textos.ts';

	let {
		plantilla,
		nombre,
		primario,
		acento,
		servicios,
		fotos,
		ciudad,
		rubro,
		logo,
		resenas
	}: {
		plantilla: Plantilla;
		nombre: string;
		primario: string;
		acento: string;
		servicios: { nombre: string; precio: number }[];
		fotos: string[];
		ciudad: string;
		rubro: string;
		logo: string | null;
		resenas: { promedio: number; cantidad: number } | null;
	} = $props();

	const inicial = $derived(nombre.trim().charAt(0).toUpperCase() || 'T');
	const foto = (i: number) => fotos[i] ?? fotos[0] ?? null;
	const lugar = $derived([RUBROS[rubro], ciudad].filter(Boolean).join(' · '));
	const estrellas = $derived(resenas ? `★ ${resenas.promedio.toLocaleString('es-AR', { maximumFractionDigits: 1 })}` : '');
	const partes = $derived(nombre.trim().split(/\s+/));
	const lineas = $derived(servicios.length ? servicios : [{ nombre: 'Tus servicios', precio: 0 }]);
	const precio = (p: number) => (p ? pesos(p) : '');
</script>

{#snippet imagen(i: number, clase = '')}
	{#if foto(i)}
		<img class="foto {clase}" src={foto(i)} alt="" loading="lazy" />
	{:else}
		<span class="foto sin {clase}" style:--c={primario}></span>
	{/if}
{/snippet}

<div class="mini" aria-hidden="true" style:--p={primario} style:--a={acento} style:--sp={textoSobre(primario)} style:--sa={textoSobre(acento)}>
	{#if plantilla === 'carta'}
		<div class="carta-tapa">
			{@render imagen(0)}
			<span class="velo"></span>
			<b>{nombre}</b>
		</div>
		<div class="lista">
			{#each lineas as l (l.nombre)}
				<span class="linea punteada"><span>{l.nombre}</span><i></i><b class="num">{precio(l.precio)}</b></span>
			{/each}
		</div>
	{:else if plantilla === 'vidriera'}
		<div class="vid-tapa">
			{@render imagen(0)}
			<span class="velo oscuro"></span>
			<span class="vid-nombre"><b>{nombre}</b><span class="mini-boton">Reservar</span></span>
		</div>
		<div class="vid-grilla">
			{@render imagen(1, 'chica')}
			{@render imagen(2, 'chica')}
			{#each lineas.slice(0, 2) as l (l.nombre)}<span class="vid-precio">{l.nombre}{l.precio ? ` · ${precio(l.precio)}` : ''}</span>{/each}
		</div>
	{:else if plantilla === 'sobria'}
		<div class="sob-tapa">
			{#if logo}<img class="logo" src={logo} alt="" />{:else}<span class="logo">{inicial}</span>{/if}
			<b>{#each partes as p, i (i)}{p}{#if i < partes.length - 1}<br />{/if}{/each}</b>
			{#if ciudad || estrellas}<span class="sob-sub">{[ciudad, estrellas].filter(Boolean).join(' · ')}</span>{/if}
		</div>
		<div class="lista">
			{#each lineas as l (l.nombre)}<span class="linea raya"><span>{l.nombre}</span><b class="num">{precio(l.precio)}</b></span>{/each}
		</div>
	{:else if plantilla === 'galeria'}
		<div class="gal-mosaico">
			{@render imagen(0, 'grande')}
			{@render imagen(1)}
			{@render imagen(2)}
		</div>
		<div class="lista">
			<span class="gal-nombre"><span class="logo chico">{inicial}</span><b>{nombre}</b></span>
			{#each lineas as l (l.nombre)}<span class="linea"><span>{l.nombre}</span><b class="num">{precio(l.precio)}</b></span>{/each}
		</div>
	{:else if plantilla === 'editorial'}
		<div class="edi">
			{#if lugar}<span class="edi-lugar">{lugar}</span>{/if}
			<b class="edi-nombre">{#each partes as p, i (i)}{p}{#if i < partes.length - 1}<br />{/if}{/each}.</b>
			<span class="edi-foto">{@render imagen(0, 'redonda')}</span>
			{#each lineas as l (l.nombre)}<span class="linea arriba"><span>{l.nombre}</span><b class="num">{precio(l.precio)}</b></span>{/each}
		</div>
	{:else}
		<div class="blo">
			<span class="blo-nombre">{nombre}</span>
			{@render imagen(0, 'bloque')}
			<span class="blo-acento">{#if resenas}{estrellas}<span>{resenas.cantidad} reseñas</span>{:else}Reservá<span>online</span>{/if}</span>
			<span class="blo-precio">{lineas[0].nombre}<b class="num">{precio(lineas[0].precio)}</b></span>
			{@render imagen(1, 'bloque')}
		</div>
	{/if}
	<span class="reservar">Reservar turno</span>
</div>

<style>
	.mini {
		height: 330px;
		border-radius: 16px;
		overflow: hidden;
		background: var(--tl-crema-pagina);
		display: flex;
		flex-direction: column;
		box-shadow: inset 0 0 0 1px #eee5d3;
		color: var(--tl-tinta);
		text-align: left;
	}
	.foto {
		display: block;
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.foto.sin {
		background: repeating-linear-gradient(-45deg, color-mix(in srgb, var(--c) 40%, transparent) 0 2px, rgba(255, 253, 249, 0.9) 2px 10px);
	}
	.velo {
		position: absolute;
		inset: 0;
		background: linear-gradient(to top, var(--p) 6%, color-mix(in srgb, var(--p) 50%, transparent) 45%, transparent 80%);
	}
	.velo.oscuro {
		background: linear-gradient(to top, rgba(15, 35, 40, 0.75), transparent 60%);
	}
	.lista {
		padding: 10px 12px;
		display: flex;
		flex-direction: column;
		gap: 7px;
	}
	.linea {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: 4px;
		font-size: 11px;
		min-width: 0;
	}
	.linea > span {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.linea.punteada > span {
		font-weight: 700;
	}
	.linea i {
		flex: 1;
		border-bottom: 1.5px dotted #d6cab2;
	}
	.linea.raya {
		padding-bottom: 6px;
		border-bottom: 1px solid #eee5d3;
	}
	.linea.raya > span {
		font-weight: 700;
	}
	.linea.arriba {
		padding-top: 6px;
		border-top: 1px solid var(--tl-linea-2);
	}
	.reservar {
		margin: auto 10px 10px;
		height: 26px;
		flex: none;
		border-radius: 999px;
		background: var(--p);
		color: var(--sp);
		display: grid;
		place-items: center;
		font-size: 11px;
		font-weight: 700;
	}
	.carta-tapa {
		position: relative;
		height: 120px;
		flex: none;
	}
	.carta-tapa b {
		position: absolute;
		left: 12px;
		right: 12px;
		bottom: 10px;
		color: #fff;
		font-size: 18px;
		letter-spacing: -0.04em;
	}
	.vid-tapa {
		position: relative;
		height: 200px;
		flex: none;
	}
	.vid-nombre {
		position: absolute;
		left: 0;
		right: 0;
		bottom: 14px;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 6px;
		color: #fff;
		text-align: center;
	}
	.vid-nombre b {
		font-size: 20px;
		letter-spacing: -0.04em;
	}
	.mini-boton {
		height: 22px;
		padding: 0 10px;
		border-radius: 999px;
		background: var(--p);
		color: var(--sp);
		display: grid;
		place-items: center;
		font-size: 11px;
		font-weight: 700;
	}
	.vid-grilla {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 6px;
		padding: 8px;
	}
	.foto.chica {
		height: 56px;
		border-radius: 8px;
	}
	.vid-precio {
		font-size: 11px;
		font-weight: 700;
	}
	.sob-tapa {
		flex: none;
		padding: 22px 14px 18px;
		background: var(--p);
		color: var(--sp);
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.logo {
		width: 38px;
		height: 38px;
		border-radius: 11px;
		background: #fff;
		color: var(--p);
		display: grid;
		place-items: center;
		font-weight: 700;
		object-fit: cover;
	}
	.logo.chico {
		width: 24px;
		height: 24px;
		border-radius: 7px;
		background: var(--p);
		color: var(--sp);
		font-size: 11px;
	}
	.sob-tapa b {
		font-size: 24px;
		line-height: 1;
		letter-spacing: -0.05em;
	}
	.sob-sub {
		font-size: 11px;
		opacity: 0.85;
	}
	.gal-mosaico {
		display: grid;
		grid-template-columns: 2fr 1fr;
		grid-template-rows: 70px 70px;
		gap: 3px;
		flex: none;
	}
	.gal-mosaico :global(.grande) {
		grid-row: span 2;
	}
	.gal-nombre {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 13px;
	}
	.edi {
		padding: 18px 14px 10px;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.edi-lugar {
		font-size: 11px;
		font-weight: 700;
		color: var(--p);
	}
	.edi-nombre {
		font-size: 34px;
		line-height: 0.86;
		letter-spacing: -0.07em;
	}
	.edi-foto {
		width: 54px;
		height: 54px;
		border-radius: 50%;
		overflow: hidden;
	}
	.blo {
		display: grid;
		grid-template-columns: 1fr 1fr;
		grid-auto-rows: 64px;
		gap: 5px;
		padding: 6px;
	}
	.blo-nombre {
		grid-column: span 2;
		border-radius: 10px;
		background: var(--p);
		color: var(--sp);
		padding: 10px;
		display: flex;
		align-items: flex-end;
		font-size: 16px;
		font-weight: 700;
		letter-spacing: -0.04em;
	}
	.foto.bloque {
		border-radius: 10px;
	}
	.blo-acento {
		border-radius: 10px;
		background: var(--a);
		color: var(--sa);
		padding: 8px;
		display: flex;
		flex-direction: column;
		justify-content: flex-end;
		font-size: 11px;
		font-weight: 700;
	}
	.blo-acento span {
		font-weight: 400;
	}
	.blo-precio {
		border-radius: 10px;
		background: var(--tl-papel);
		box-shadow: inset 0 0 0 1px #eee5d3;
		padding: 8px;
		font-size: 11px;
		font-weight: 700;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		overflow: hidden;
	}
</style>
