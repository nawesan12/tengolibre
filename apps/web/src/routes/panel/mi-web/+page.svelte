<script lang="ts">
	import { enhance } from '$app/forms';
	import { RUBROS } from '#lib/configuracion/textos.ts';
	import { confirmar, sinReset } from '#lib/configuracion/formularios.ts';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';

	let { data, form } = $props();
	const n = $derived(data.datos);
	const w = $derived(data.web);
	let subiendo = $state<string | null>(null);

	// Subir: avisa mientras manda y vacía el campo al terminar bien.
	const subir = (que: string) => () => {
		subiendo = que;
		return async ({ update }: { update: (o?: { reset?: boolean }) => Promise<void> }) => {
			await update();
			subiendo = null;
		};
	};
	const MAX = 5 << 20;
	// Una imagen de más de 5 MB no se manda: se avisa en el campo.
	function revisarTamanio(e: Event & { currentTarget: HTMLInputElement }) {
		const a = e.currentTarget.files?.[0];
		e.currentTarget.setCustomValidity(a && a.size > MAX ? 'La imagen pasa de 5 MB.' : '');
		e.currentTarget.reportValidity();
	}
</script>

<svelte:head><title>Mi web · Panel</title></svelte:head>

<div class="pila">
	<div class="fila entre">
		<h1>Mi web</h1>
		<div class="fila">
			<a class="boton" href="/panel/dominio">Dominio propio</a>
			<a class="boton" href={`/${n.slug}`} target="_blank" rel="noopener">Ver mi página</a>
		</div>
	</div>

	<section class="tarjeta pila" aria-labelledby="t-negocio">
		<h2 id="t-negocio">Datos del negocio</h2>
		<Mensaje {form} que="negocio" />
		<form method="POST" action="?/negocio" use:enhance={sinReset} class="grilla">
			<label class="campo">Nombre <input name="nombre" value={n.nombre} required maxlength="80" /></label>
			<label class="campo">Dirección <input name="direccion" value={n.direccion} maxlength="200" autocomplete="street-address" /></label>
			<label class="campo">Ciudad <input name="ciudad" value={n.ciudad} maxlength="80" /></label>
			<label class="campo">Teléfono <input name="telefono" type="tel" value={n.telefono} maxlength="30" /></label>
			<label class="campo">WhatsApp <input name="whatsapp" type="tel" value={n.whatsapp} maxlength="30" placeholder="+54 9 11…" /></label>
			<label class="campo">Instagram <input name="instagram" value={n.instagram} maxlength="60" placeholder="tunegocio" /></label>
			<label class="campo ancho">Descripción corta <textarea name="descripcion" rows="2" maxlength="500">{n.descripcion}</textarea></label>
			<div><button class="boton primario">Guardar datos</button></div>
		</form>
	</section>

	<section class="tarjeta pila" aria-labelledby="t-imagenes">
		<h2 id="t-imagenes">Logo y portada</h2>
		<p class="suave chico">JPG, PNG o WebP de hasta 5&nbsp;MB.</p>
		<div class="grilla">
			<div class="pila">
				<h3>Logo</h3>
				{#if n.logo}<img class="logo" src={n.logo} alt={`Logo de ${n.nombre}`} />{/if}
				<Mensaje {form} que="logo" />
				<form method="POST" action="?/logo" enctype="multipart/form-data" use:enhance={subir('logo')} class="pila">
					<label class="campo">
						{n.logo ? 'Cambiar el logo' : 'Subir un logo'}
						<input type="file" name="archivo" accept="image/jpeg,image/png,image/webp" required onchange={revisarTamanio} />
					</label>
					<div><button class="boton" disabled={subiendo === 'logo'}>{subiendo === 'logo' ? 'Subiendo…' : 'Subir logo'}</button></div>
				</form>
			</div>
			<div class="pila">
				<h3>Portada</h3>
				{#if w.portada}<img class="portada" src={w.portada} alt="Portada de tu página" />{/if}
				<Mensaje {form} que="portada" />
				<form method="POST" action="?/portada" enctype="multipart/form-data" use:enhance={subir('portada')} class="pila">
					<label class="campo">
						{w.portada ? 'Cambiar la portada' : 'Subir una portada'}
						<input type="file" name="archivo" accept="image/jpeg,image/png,image/webp" required onchange={revisarTamanio} />
					</label>
					<div class="fila">
						<button class="boton" disabled={subiendo === 'portada'}>{subiendo === 'portada' ? 'Subiendo…' : 'Subir portada'}</button>
						{#if w.portada}
							<button class="boton peligro" formaction="?/sacarPortada" formnovalidate formenctype="application/x-www-form-urlencoded">Sacar portada</button>
						{/if}
					</div>
				</form>
			</div>
		</div>
	</section>

	<section class="tarjeta pila" aria-labelledby="t-galeria">
		<h2 id="t-galeria">Galería</h2>
		<p class="suave chico">Fotos de tus trabajos y del lugar, en el orden en que se ven en tu página (hasta&nbsp;30).</p>
		<Mensaje {form} que="galeria" />
		<form method="POST" action="?/foto" enctype="multipart/form-data" use:enhance={subir('galeria')} class="grilla">
			<label class="campo">
				Foto
				<input type="file" name="archivo" accept="image/jpeg,image/png,image/webp" required onchange={revisarTamanio} />
			</label>
			<label class="campo">Epígrafe (opcional) <input name="epigrafe" maxlength="200" /></label>
			<div class="fila" style="align-self: end">
				<button class="boton primario" disabled={subiendo === 'galeria' || data.fotos.length >= 30}>
					{subiendo === 'galeria' ? 'Subiendo…' : 'Sumar a la galería'}
				</button>
			</div>
		</form>
		{#if data.fotos.length === 0}
			<p class="suave">Todavía no hay fotos.</p>
		{:else}
			<ol class="galeria">
				{#each data.fotos as foto, i (foto.id)}
					<li class="tarjeta pila">
						<img src={foto.url} alt={foto.epigrafe || `Foto ${i + 1} de la galería`} loading="lazy" />
						<form method="POST" action="?/epigrafe" use:enhance={sinReset} class="fila">
							<input type="hidden" name="id" value={foto.id} />
							<label class="campo crece">
								<span class="visualmente-oculto">Epígrafe de la foto {i + 1}</span>
								<input name="epigrafe" value={foto.epigrafe} maxlength="200" placeholder="Epígrafe" />
							</label>
							<button class="boton chico">Guardar</button>
						</form>
						<div class="fila">
							<form method="POST" action="?/mover" use:enhance>
								<input type="hidden" name="id" value={foto.id} />
								<input type="hidden" name="hacia" value="antes" />
								<button class="boton" disabled={i === 0} aria-label={`Mover la foto ${i + 1} antes`}>↑</button>
							</form>
							<form method="POST" action="?/mover" use:enhance>
								<input type="hidden" name="id" value={foto.id} />
								<input type="hidden" name="hacia" value="despues" />
								<button class="boton" disabled={i === data.fotos.length - 1} aria-label={`Mover la foto ${i + 1} después`}>↓</button>
							</form>
							<form method="POST" action="?/borrarFoto" use:enhance={confirmar('¿Sacar esta foto de la galería?')}>
								<input type="hidden" name="id" value={foto.id} />
								<button class="boton peligro" aria-label={`Sacar la foto ${i + 1}`}>Sacar</button>
							</form>
						</div>
					</li>
				{/each}
			</ol>
		{/if}
	</section>

	<section class="tarjeta pila" aria-labelledby="t-web">
		<h2 id="t-web">Tu página</h2>
		<Mensaje {form} que="web" />
		<form method="POST" action="?/web" use:enhance={sinReset} class="grilla">
			<label class="campo">
				Rubro
				<select name="rubro">
					<option value="">Sin elegir</option>
					{#each Object.entries(RUBROS) as [clave, nombre] (clave)}<option value={clave} selected={clave === w.rubro}>{nombre}</option>{/each}
				</select>
			</label>
			<label class="campo">Provincia <input name="provincia" value={w.provincia} maxlength="80" /></label>
			<label class="campo">Color principal <input type="color" name="colorPrimario" value={w.colorPrimario || '#0f9d58'} /></label>
			<label class="campo">Color de acento <input type="color" name="colorAcento" value={w.colorAcento || '#16181a'} /></label>
			<label class="campo ancho">Sobre el negocio <textarea name="acerca" rows="5" maxlength="5000">{w.acerca}</textarea></label>
			<label class="campo">Email de contacto <input name="email" type="email" value={w.email} maxlength="120" /></label>
			<label class="campo">Sitio web <input name="sitioWeb" type="url" value={w.sitioWeb} maxlength="200" placeholder="https://" /></label>
			<label class="campo">Facebook <input name="facebook" value={w.facebook} maxlength="100" /></label>
			<label class="campo">TikTok <input name="tiktok" value={w.tiktok} maxlength="60" /></label>
			<fieldset class="campo ancho ubicacion">
				<legend>Ubicación en el mapa</legend>
				<p class="suave chico">Las coordenadas de Google Maps (clic derecho sobre el lugar). Vacías, no se muestra&nbsp;mapa.</p>
				<div class="fila">
					<label class="campo">Latitud <input name="latitud" type="number" step="any" min="-90" max="90" value={w.latitud ?? ''} /></label>
					<label class="campo">Longitud <input name="longitud" type="number" step="any" min="-180" max="180" value={w.longitud ?? ''} /></label>
				</div>
			</fieldset>
			<div><button class="boton primario">Guardar tu página</button></div>
		</form>
	</section>
</div>

<style>
	.entre {
		justify-content: space-between;
	}
	.ancho {
		grid-column: 1 / -1;
	}
	h3 {
		font-size: 16px;
		margin: 0;
	}
	.logo {
		width: 96px;
		height: 96px;
		object-fit: contain;
		border: 1px solid var(--borde);
		border-radius: 8px;
	}
	.portada {
		width: 100%;
		aspect-ratio: 3 / 1;
		object-fit: cover;
		border-radius: 8px;
	}
	.galeria {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		gap: 12px;
		grid-template-columns: repeat(auto-fill, minmax(min(220px, 100%), 1fr));
	}
	.galeria img {
		width: 100%;
		aspect-ratio: 4 / 3;
		object-fit: cover;
		border-radius: 8px;
		background: var(--fondo);
	}
	.crece {
		flex: 1;
		min-width: 0;
	}
	.ubicacion {
		border: 1px solid var(--borde);
		border-radius: 8px;
		padding: 8px 12px 12px;
		margin: 0;
	}
	.ubicacion p {
		margin: 0;
	}
</style>
