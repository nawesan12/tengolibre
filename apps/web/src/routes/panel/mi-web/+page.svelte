<script lang="ts">
	import '#lib/gestion/gestion.css';
	import { enhance } from '$app/forms';
	import { page } from '$app/state';
	import type { SubmitFunction } from '$app/forms';
	import { hora } from '#lib/formato.ts';
	import { RUBROS } from '#lib/configuracion/textos.ts';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';
	import Copiar from '#lib/gestion/Copiar.svelte';
	import Galeria from '#lib/gestion/miweb/Galeria.svelte';
	import VistaEnVivo from '#lib/gestion/miweb/VistaEnVivo.svelte';
	import { contraste, ratioTexto, textoSobre } from '#lib/gestion/miweb/color.ts';

	let { data, form } = $props();
	const n = $derived(data.datos);
	const w = $derived(data.web);
	const d = $derived(data.dominio);

	// Lo que se escribe se ve al instante en la vista en vivo (y se guarda al salir del campo).
	let nombre = $derived(n.nombre);
	let ciudad = $derived(n.ciudad);
	let rubro = $derived(w.rubro);
	let primario = $derived(w.colorPrimario || '#183c44');
	let acento = $derived(w.colorAcento || '#3ccc90');
	const bajada = $derived([RUBROS[rubro], ciudad].filter(Boolean).join(' · '));
	const legible = $derived(textoSobre(primario));
	const acentoSobrePapel = $derived(contraste(acento, '#faf6ee'));

	// Se guarda solo: cada formulario se manda al cambiar un campo. Arriba, cuándo fue lo último.
	let guardando = $state(0);
	let guardadoEn = $state<string | null>(null);
	const autoguardar: SubmitFunction = () => {
		guardando++;
		return async ({ result, update }) => {
			await update({ reset: false });
			guardando--;
			if (result.type === 'success') guardadoEn = hora(new Date());
		};
	};
	const enviar = (e: Event) => (e.currentTarget as HTMLInputElement).form?.requestSubmit();

	let subiendo = $state<string | null>(null);
	const subir: (que: string) => SubmitFunction = (que) => () => {
		subiendo = que;
		return async ({ result, update }) => {
			await update();
			subiendo = null;
			if (result.type === 'success') guardadoEn = hora(new Date());
		};
	};
	const MAX = 5 << 20;
	// Una imagen de más de 5 MB no se manda: se avisa en el campo. Si está bien, se sube sola.
	function subirSiSirve(e: Event & { currentTarget: HTMLInputElement }) {
		const a = e.currentTarget.files?.[0];
		e.currentTarget.setCustomValidity(a && a.size > MAX ? 'La imagen pasa de 5 MB.' : '');
		if (!e.currentTarget.reportValidity()) return;
		if (a) e.currentTarget.form?.requestSubmit();
	}

	// «7 de 8 cargados»: lo que hay en «Acerca de, contactos y redes».
	const extras = $derived([w.acerca, n.telefono, n.whatsapp, n.instagram, w.email, w.sitioWeb, w.facebook, w.tiktok]);
	const cargados = $derived(extras.filter((x) => x && String(x).trim()).length);
	let abrirExtras = $state(false);

	const direccion = $derived(`${page.url.host}/`);
	const pasoDominio = $derived(d ? (d.estado === 'verificado' ? 3 : 2) : 1);
</script>

<svelte:head><title>Mi web · Panel</title></svelte:head>

<!-- Los dos formularios que se guardan solos: los campos de cada tarjeta se asocian con form="…". -->
<form id="f-negocio" method="POST" action="?/negocio" use:enhance={autoguardar} hidden></form>
<form id="f-web" method="POST" action="?/web" use:enhance={autoguardar} hidden></form>

<div class="g-pagina mi-web">
	<div class="editor">
		<div class="g-cabeza">
			<h1 class="g-titulo">Mi web</h1>
			<span class="g-meta ok" aria-live="polite">
				{#if guardando}Guardando…{:else if guardadoEn}<span class="icono" aria-hidden="true">check</span>Guardado {guardadoEn}{:else}Se guarda solo{/if}
			</span>
		</div>

		<section class="g-bloque" aria-label="Tu página">
			<Mensaje {form} que="negocio" />
			<Mensaje {form} que="web" />
			<div class="g-form">
				<label class="g-campo">
					Nombre
					<input form="f-negocio" name="nombre" bind:value={nombre} required maxlength="80" onchange={enviar} />
				</label>
				<label class="g-campo">
					Rubro
					<select form="f-web" name="rubro" bind:value={rubro} onchange={enviar}>
						<option value="">Sin elegir</option>
						{#each Object.entries(RUBROS) as [clave, r] (clave)}<option value={clave}>{r}</option>{/each}
					</select>
				</label>
				<label class="g-campo g-ancho">
					Descripción corta
					<input form="f-negocio" name="descripcion" value={n.descripcion} maxlength="500" placeholder="Qué hacés, en una línea." onchange={enviar} />
				</label>
				<label class="g-campo">
					Dirección
					<input form="f-negocio" name="direccion" value={n.direccion} maxlength="200" autocomplete="street-address" onchange={enviar} />
				</label>
				<div class="g-campo">
					Ubicación
					{#if w.latitud != null && w.longitud != null}
						<a class="pin" href={`https://www.google.com/maps?q=${w.latitud},${w.longitud}`} target="_blank" rel="noopener">
							<span class="icono lleno" aria-hidden="true">location_on</span>Ver el pin en el mapa
						</a>
					{:else}
						<button type="button" class="pin" onclick={() => (abrirExtras = true)}>
							<span class="icono lleno" aria-hidden="true">location_on</span>Sin pin: cargalo abajo
						</button>
					{/if}
				</div>

				<details class="g-plegable g-ancho extras" bind:open={abrirExtras}>
					<summary>Acerca de, contactos y redes<span class="g-meta empuja">{cargados} de 8 cargados</span></summary>
					<div class="g-form adentro">
						<label class="g-campo">Ciudad <input form="f-negocio" name="ciudad" bind:value={ciudad} maxlength="80" onchange={enviar} /></label>
						<label class="g-campo">Provincia <input form="f-web" name="provincia" value={w.provincia} maxlength="80" onchange={enviar} /></label>
						<label class="g-campo g-ancho">
							Sobre el negocio
							<textarea form="f-web" name="acerca" rows="4" maxlength="5000" onchange={enviar}>{w.acerca}</textarea>
						</label>
						<label class="g-campo">Teléfono <input form="f-negocio" name="telefono" type="tel" value={n.telefono} maxlength="30" onchange={enviar} /></label>
						<label class="g-campo">
							WhatsApp
							<input form="f-negocio" name="whatsapp" type="tel" value={n.whatsapp} maxlength="30" placeholder="+54 9 11…" onchange={enviar} />
						</label>
						<label class="g-campo">Email de contacto <input form="f-web" name="email" type="email" value={w.email} maxlength="120" onchange={enviar} /></label>
						<label class="g-campo">
							Sitio web
							<input form="f-web" name="sitioWeb" type="url" value={w.sitioWeb} maxlength="200" placeholder="https://" onchange={enviar} />
						</label>
						<label class="g-campo">Instagram <input form="f-negocio" name="instagram" value={n.instagram} maxlength="60" placeholder="tunegocio" onchange={enviar} /></label>
						<label class="g-campo">Facebook <input form="f-web" name="facebook" value={w.facebook} maxlength="100" onchange={enviar} /></label>
						<label class="g-campo">TikTok <input form="f-web" name="tiktok" value={w.tiktok} maxlength="60" onchange={enviar} /></label>
						<fieldset class="g-ancho ubicacion">
							<legend class="g-campo">Ubicación en el mapa</legend>
							<p class="g-nota">Las coordenadas de Google Maps (clic derecho sobre el lugar). Vacías, no se muestra&nbsp;mapa.</p>
							<div class="g-form">
								<label class="g-campo">
									Latitud
									<input form="f-web" name="latitud" type="number" step="any" min="-90" max="90" value={w.latitud ?? ''} onchange={enviar} />
								</label>
								<label class="g-campo">
									Longitud
									<input form="f-web" name="longitud" type="number" step="any" min="-180" max="180" value={w.longitud ?? ''} onchange={enviar} />
								</label>
							</div>
						</fieldset>
					</div>
				</details>
			</div>
			<!-- Los colores van en el mismo formulario de la web: viajan siempre juntos. -->
			<input form="f-web" type="hidden" name="colorPrimario" value={primario} />
			<input form="f-web" type="hidden" name="colorAcento" value={acento} />
		</section>

		<div class="g-dos">
			<section class="g-bloque" aria-labelledby="t-colores">
				<h2 class="g-bloque-titulo" id="t-colores">Colores de tu marca</h2>
				<div class="colores">
					<label class="color">
						<input type="color" bind:value={primario} onchange={() => document.querySelector<HTMLFormElement>('#f-web')?.requestSubmit()} aria-label="Color principal" />
						<span class="hex">{primario.toUpperCase()} · principal</span>
					</label>
					<label class="color">
						<input type="color" bind:value={acento} onchange={() => document.querySelector<HTMLFormElement>('#f-web')?.requestSubmit()} aria-label="Color de acento" />
						<span class="hex">{acento.toUpperCase()} · acento</span>
					</label>
				</div>
				{#if legible.ratio >= 4.5}
					<span class="g-estado ok contraste">
						<span class="icono lleno" aria-hidden="true">check_circle</span>
						Se lee bien: el texto sobre tu color va en {legible.nombre}&nbsp;({ratioTexto(legible.ratio)})
					</span>
				{:else}
					<span class="g-estado espera contraste">
						<span class="icono lleno" aria-hidden="true">warning</span>
						Se lee poco: el texto sobre tu color queda en {ratioTexto(legible.ratio)}. Probá uno más oscuro o más&nbsp;claro.
					</span>
				{/if}
				{#if acentoSobrePapel < 3}
					<span class="g-sub">El acento se usa en detalles: sobre el fondo claro casi no se&nbsp;nota.</span>
				{/if}
			</section>

			<Galeria fotos={data.fotos} {form} />
		</div>

		<section class="g-bloque" aria-labelledby="t-imagenes">
			<div class="g-bloque-cabeza">
				<h2 class="g-bloque-titulo" id="t-imagenes">Logo y portada</h2>
				<span class="g-meta">JPG, PNG o WebP · hasta 5 MB</span>
			</div>
			<Mensaje {form} que="logo" />
			<Mensaje {form} que="portada" />
			<div class="imagenes">
				<form method="POST" action="?/logo" enctype="multipart/form-data" use:enhance={subir('logo')} class="imagen">
					<label class="cuadro logo">
						{#if n.logo}<img src={n.logo} alt={`Logo de ${n.nombre}`} />{:else}<span class="inicial" style:background={primario} style:color={legible.color}>{(nombre.trim()[0] ?? '·').toUpperCase()}</span>{/if}
						<input type="file" name="archivo" accept="image/jpeg,image/png,image/webp" required onchange={subirSiSirve} aria-label={n.logo ? 'Cambiar el logo' : 'Subir un logo'} />
					</label>
					<span class="g-estado neutro">
						<span class="icono" aria-hidden="true">{subiendo === 'logo' ? 'hourglass_top' : 'upload'}</span>
						{subiendo === 'logo' ? 'Subiendo…' : n.logo ? 'Cambiar el logo' : 'Subir un logo'}
					</span>
				</form>
				<form method="POST" action="?/portada" enctype="multipart/form-data" use:enhance={subir('portada')} class="imagen crece">
					<label class="cuadro portada" style:background={primario}>
						{#if w.portada}<img src={w.portada} alt="Portada de tu página" />{:else}<span class="sin" style:color={legible.color}><span class="icono" aria-hidden="true">add_photo_alternate</span>Sin portada: se ve tu color</span>{/if}
						<input type="file" name="archivo" accept="image/jpeg,image/png,image/webp" required onchange={subirSiSirve} aria-label={w.portada ? 'Cambiar la portada' : 'Subir una portada'} />
					</label>
					<span class="fila-imagen">
						<span class="g-estado neutro">
							<span class="icono" aria-hidden="true">{subiendo === 'portada' ? 'hourglass_top' : 'upload'}</span>
							{subiendo === 'portada' ? 'Subiendo…' : w.portada ? 'Cambiar la portada' : 'Subir una portada'}
						</span>
						{#if w.portada}
							<button class="g-accion peligro" formaction="?/sacarPortada" formnovalidate formenctype="application/x-www-form-urlencoded">Sacar portada</button>
						{/if}
					</span>
				</form>
			</div>
		</section>

		<section class="g-bloque" aria-labelledby="t-direccion">
			<div class="g-bloque-cabeza">
				<h2 class="g-bloque-titulo" id="t-direccion">Tu dirección web</h2>
				<a class="direccion" href={`/${n.slug}`} target="_blank" rel="noopener">{direccion}<span class="verde">{n.slug}</span></a>
			</div>
			{#if d}
				<div class="estado-dominio" class:ok={d.estado === 'verificado'} class:mal={d.estado === 'error'}>
					<span class="icono" aria-hidden="true">{d.estado === 'verificado' ? 'check_circle' : d.estado === 'error' ? 'error' : 'sync'}</span>
					<span class="texto">
						<b>{d.nombre} · {d.estado === 'verificado' ? 'conectado' : d.estado === 'error' ? 'todavía no anda' : 'verificando'}</b>
						<span>
							{#if d.estado === 'verificado'}Tu página ya se ve en esa dirección.
							{:else if d.estado === 'error' && d.detalle}{d.detalle}
							{:else}Falta que el DNS apunte a nosotros.{#if d.chequeadoEn} Último intento {hora(d.chequeadoEn)}.{/if}{/if}
						</span>
					</span>
					<span class="g-meta paso">Paso {pasoDominio} de 3</span>
				</div>
				{#if d.estado !== 'verificado'}
					<div class="registros">
						<div class="registro rot" aria-hidden="true"><span>Tipo</span><span>Nombre</span><span>Valor</span><span></span></div>
						{#each d.registros as r (r.tipo + r.nombre + r.valor)}
							<div class="registro"><b>{r.tipo}</b><b>{r.nombre}</b><b class="valor">{r.valor}</b><Copiar texto={r.valor} /></div>
						{/each}
					</div>
				{/if}
				<a class="g-accion" href="/panel/dominio">Ver los pasos del dominio propio<span class="icono" aria-hidden="true">chevron_right</span></a>
			{:else}
				<a class="sin-dominio" href="/panel/dominio">
					<span class="icono" aria-hidden="true">language</span>
					<span class="texto"><b>¿Tenés un dominio propio?</b><span>Tu página también puede verse en turnos.tunegocio.com.</span></span>
					<span class="g-accion">Conectarlo<span class="icono" aria-hidden="true">chevron_right</span></span>
				</a>
			{/if}
		</section>
	</div>

	<aside class="lado" aria-label="Vista en vivo">
		<VistaEnVivo {nombre} {bajada} logo={n.logo} portada={w.portada} {primario} servicios={data.servicios} href={`/${n.slug}`} />
	</aside>
</div>

<style>
	.mi-web {
		display: grid;
		grid-template-columns: minmax(0, 1fr) 340px;
		gap: 32px;
		align-items: start;
	}
	.editor {
		display: flex;
		flex-direction: column;
		gap: 16px;
		min-width: 0;
	}
	.g-meta .icono {
		font-size: 14px;
		vertical-align: -2px;
		margin-right: 2px;
	}
	.pin {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 6px;
		min-height: 46px;
		padding: 0 12px;
		border: 0;
		border-radius: 12px;
		background: repeating-linear-gradient(135deg, #f4eee2 0 8px, #ece4d3 8px 16px);
		color: var(--tl-sub);
		font: 400 12px var(--tl-font);
		text-transform: uppercase;
		text-decoration: none;
		cursor: pointer;
	}
	.pin .icono {
		font-size: 18px;
		color: #a3241a;
	}
	.extras {
		border-top: 1px solid #eee5d3;
		padding-top: 4px;
	}
	.empuja {
		margin-left: auto;
		white-space: nowrap;
	}
	.adentro {
		padding: 8px 0 4px;
	}
	.ubicacion {
		margin: 0;
		padding: 12px;
		border: 0;
		border-radius: 12px;
		background: #f8f3ea;
		display: flex;
		flex-direction: column;
		gap: 8px;
		min-width: 0;
	}
	.ubicacion legend {
		float: left;
		padding: 0;
	}
	.colores {
		display: flex;
		gap: 10px;
	}
	.color {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 6px;
		min-width: 0;
		cursor: pointer;
	}
	.color input {
		width: 100%;
		height: 56px;
		padding: 0;
		border: 0;
		border-radius: 12px;
		background: none;
		cursor: pointer;
	}
	.color input::-webkit-color-swatch-wrapper {
		padding: 0;
	}
	.color input::-webkit-color-swatch {
		border: 0;
		border-radius: 12px;
	}
	.color input::-moz-color-swatch {
		border: 0;
		border-radius: 12px;
	}
	.hex {
		font: 400 12px var(--tl-font);
		font-variant-numeric: tabular-nums;
	}
	.contraste {
		white-space: normal;
		align-items: flex-start;
		font-size: 12px;
		font-weight: 400;
		line-height: 1.4;
	}
	.contraste .icono {
		font-size: 16px;
	}
	.imagenes {
		display: flex;
		gap: 16px;
		align-items: flex-start;
	}
	.imagen {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.crece {
		flex: 1;
		min-width: 0;
	}
	.cuadro {
		position: relative;
		display: grid;
		place-items: center;
		overflow: hidden;
		border-radius: 14px;
		cursor: pointer;
		box-shadow: inset 0 0 0 1.5px #e0d5bf;
	}
	.cuadro:focus-within {
		outline: 3px solid var(--tl-menta);
		outline-offset: 2px;
	}
	.cuadro input {
		position: absolute;
		inset: 0;
		opacity: 0;
		cursor: pointer;
	}
	.cuadro.logo {
		width: 96px;
		height: 96px;
		background: #fff;
	}
	.cuadro.logo img {
		width: 100%;
		height: 100%;
		object-fit: contain;
	}
	.inicial {
		width: 100%;
		height: 100%;
		display: grid;
		place-items: center;
		font-size: 36px;
		font-weight: 700;
	}
	.cuadro.portada {
		width: 100%;
		height: 96px;
	}
	.cuadro.portada img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.sin {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 13px;
		font-weight: 700;
	}
	.fila-imagen {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: 8px;
	}
	.direccion {
		font: 700 14px var(--tl-font);
		font-variant-numeric: tabular-nums;
		color: var(--tl-tinta);
		text-decoration: none;
		overflow-wrap: anywhere;
	}
	.verde {
		color: var(--tl-menta-texto);
	}
	.estado-dominio,
	.sin-dominio {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr) auto;
		gap: 14px;
		align-items: center;
		padding: 14px;
		border-radius: 14px;
		background: #fbe9c2;
		color: #5c3800;
	}
	.estado-dominio > .icono {
		color: #7a4a00;
	}
	.estado-dominio.ok {
		background: var(--tl-menta-suave);
		color: #0f3d2b;
	}
	.estado-dominio.ok > .icono {
		color: var(--tl-menta-texto);
	}
	.estado-dominio.mal {
		background: #fbe0dc;
		color: #5b1a13;
	}
	.estado-dominio.mal > .icono {
		color: #a3241a;
	}
	.paso {
		color: inherit;
	}
	.texto {
		display: flex;
		flex-direction: column;
		gap: 2px;
		font-size: 14px;
		min-width: 0;
	}
	.texto b {
		overflow-wrap: anywhere;
	}
	.texto span {
		font-size: 12px;
	}
	.sin-dominio {
		background: #f8f3ea;
		color: var(--tl-tinta);
		text-decoration: none;
	}
	.sin-dominio .texto span {
		color: var(--tl-sub);
	}
	.sin-dominio > .icono {
		color: var(--tl-menta-texto);
	}
	.registros {
		display: flex;
		flex-direction: column;
		gap: 10px;
		padding: 10px 14px;
		border-radius: 12px;
		background: var(--tl-crema);
		font: 400 13px var(--tl-font);
		font-variant-numeric: tabular-nums;
	}
	.registro {
		display: grid;
		grid-template-columns: 64px minmax(0, 1fr) minmax(0, 1.3fr) auto;
		gap: 4px 12px;
		align-items: center;
	}
	.rot {
		color: var(--tl-sub);
		text-transform: uppercase;
	}
	.registros b {
		overflow-wrap: anywhere;
	}
	.g-accion .icono {
		font-size: 18px;
	}

	@media (max-width: 1100px) {
		.mi-web {
			grid-template-columns: minmax(0, 1fr);
		}
		.lado {
			max-width: 360px;
			width: 100%;
			margin: 0 auto;
		}
	}
	@media (max-width: 640px) {
		.imagenes {
			flex-direction: column;
			align-items: stretch;
		}
		.estado-dominio {
			grid-template-columns: auto minmax(0, 1fr);
		}
		.estado-dominio .paso {
			grid-column: 2;
		}
		.registro {
			grid-template-columns: auto minmax(0, 1fr);
		}
		.registro + .registro {
			padding-top: 10px;
			border-top: 1px solid #e6dcc8;
		}
		.registro.rot {
			display: none;
		}
		.registro .valor {
			grid-column: 1 / -1;
		}
		.sin-dominio {
			grid-template-columns: auto minmax(0, 1fr);
		}
		.sin-dominio > .g-accion {
			grid-column: 2;
		}
		.pin {
			min-height: 48px;
		}
	}
</style>
