<script lang="ts">
	import '#lib/gestion/gestion.css';
	import { enhance } from '$app/forms';
	import { onMount } from 'svelte';
	import Mensaje from '#lib/configuracion/Mensaje.svelte';
	import Miniatura from '#lib/gestion/plantillas/Miniatura.svelte';
	import {
		DESCRIPCIONES,
		FONDOS,
		FORMAS,
		NOMBRES,
		PLANTILLAS,
		PORTADAS,
		SECCIONES,
		TITULOS,
		diferencias,
		urlVistaPrevia,
		type Ajuste,
		type Estilo,
		type Seccion
	} from '#lib/gestion/plantillas.ts';

	let { data, form } = $props();

	const TODAS: Seccion[] = [...SECCIONES];
	/** Lo publicado, con las secciones completas (si la API todavía no las guarda, se ven todas). */
	const publicado = $derived<Estilo>({ ...data.publicado, secciones: data.publicado.secciones ?? TODAS });
	// El borrador vive en la página (y en el navegador, para no perderlo al recargar).
	// svelte-ignore state_referenced_locally
	let borrador = $state<Estilo>({ ...data.publicado, secciones: data.publicado.secciones ?? TODAS });
	let publicada = $state(false);
	let enviando = $state(false);
	let arrastrada = $state<Seccion | null>(null);
	let timer: ReturnType<typeof setTimeout> | undefined;

	const clave = $derived(`tl-borrador-${data.negocio.slug}`);
	onMount(() => {
		try {
			const guardado = localStorage.getItem(clave);
			if (guardado) borrador = { ...borrador, ...JSON.parse(guardado) };
		} catch {
			/* sin almacenamiento: el borrador arranca de lo publicado */
		}
		return () => clearTimeout(timer);
	});
	$effect(() => {
		const b = JSON.stringify(borrador);
		try {
			if (cambios.length) localStorage.setItem(clave, b);
			else localStorage.removeItem(clave);
		} catch {
			/* nada */
		}
	});

	const cambios = $derived(
		diferencias(publicado, data.conSecciones ? borrador : { ...borrador, secciones: publicado.secciones })
	);
	const vista = $derived(urlVistaPrevia(data.negocio.slug, borrador));

	function poner<K extends keyof Estilo>(k: K, v: Estilo[K]) {
		borrador = { ...borrador, [k]: v };
		publicada = false;
	}
	const descartar = () => {
		borrador = { ...publicado, secciones: [...(publicado.secciones ?? TODAS)] };
	};

	// Secciones: tocar prende o apaga; la manija (o arrastrar) la mueve un lugar antes.
	const visibles = $derived(borrador.secciones ?? TODAS);
	const orden = $derived([...visibles, ...TODAS.filter((s) => !visibles.includes(s))]);
	function alternar(s: Seccion) {
		poner('secciones', visibles.includes(s) ? visibles.filter((x) => x !== s) : orden.filter((x) => x === s || visibles.includes(x)));
	}
	function mover(s: Seccion, antesDe?: Seccion) {
		const lista = orden.filter((x) => x !== s);
		const i = antesDe ? lista.indexOf(antesDe) : Math.max(0, orden.indexOf(s) - 1);
		lista.splice(i < 0 ? lista.length : i, 0, s);
		poner('secciones', lista.filter((x) => visibles.includes(x)));
		// Las apagadas quedan al final, en el orden en que estaban.
	}

	const AJUSTES: { k: Ajuste; t: string; s: string; ops: readonly string[] }[] = [
		{ k: 'portada', t: 'Portada', s: 'Con tu mejor foto o sólo tu color', ops: PORTADAS },
		{ k: 'fondo', t: 'Fondo', s: 'El papel de toda la página', ops: FONDOS },
		{ k: 'titulos', t: 'Títulos', s: 'Cuánto peso tiene tu nombre', ops: TITULOS },
		{ k: 'forma', t: 'Botones y tarjetas', s: 'Redondos, suaves o rectos', ops: FORMAS }
	];
	const FONDO_MUESTRA: Record<string, string> = { crema: '#faf6ee', blanco: '#ffffff', noche: '#14221f' };
	const PESO: Record<string, number> = { grandes: 900, medianos: 700, livianos: 400 };
	const RADIO: Record<string, string> = { redondos: '999px', suaves: '8px', rectos: '2px' };
	const fotoPortada = $derived(data.fotos[0] ?? null);
</script>

<svelte:head><title>Plantillas · Panel</title></svelte:head>

<div class="g-pagina plantillas">
	<div class="cabeza">
		<div class="g-col titulo">
			<h1 class="g-titulo">Plantillas</h1>
			<p class="g-bajada suave">Elegí cómo se arma tu página. Los servicios, las fotos y los colores son los mismos en&nbsp;todas.</p>
		</div>
		<a class="colores" href="/panel/mi-web" title="Cambiar los colores en Mi web">
			<span class="muestra" style:background={data.colores.primario}></span>
			<span class="muestra" style:background={data.colores.acento}></span>
			Tus colores
		</a>
	</div>

	<div class="cuerpo">
		<div class="g-col izquierda">
			<ul class="grilla" aria-label="Plantillas">
				{#each PLANTILLAS as p (p)}
					{@const enVista = borrador.plantilla === p}
					{@const uso = publicado.plantilla === p}
					<li>
						<button
							type="button"
							class="tarjeta-plantilla"
							class:en-vista={enVista}
							class:en-uso={uso && !enVista}
							aria-pressed={enVista}
							onclick={() => poner('plantilla', p)}
						>
							<Miniatura
								plantilla={p}
								nombre={data.negocio.nombre}
								primario={data.colores.primario}
								acento={data.colores.acento}
								servicios={data.servicios}
								fotos={data.fotos}
								ciudad={data.negocio.ciudad}
								rubro={data.rubro}
								logo={data.negocio.logo}
								resenas={data.resenas}
							/>
							<span class="info">
								<span class="nombre">
									<b>{NOMBRES[p]}</b>
									{#if uso}<span class="g-pildora"><span class="icono lleno" aria-hidden="true">check_circle</span>En uso</span>{/if}
								</span>
								<span class="descripcion">{DESCRIPCIONES[p]}</span>
							</span>
							<span class="probar" class:activo={enVista}>
								<span class="icono lleno" aria-hidden="true">{enVista ? 'visibility' : 'play_arrow'}</span>
								{enVista ? 'En la vista previa' : 'Probar'}
							</span>
						</button>
					</li>
				{/each}
			</ul>

			<section class="g-bloque estilo" aria-labelledby="t-estilo">
				<div class="estilo-cabeza">
					<h2 id="t-estilo" class="g-bloque-titulo">Ajustá el estilo</h2>
					<span class="g-nota">Se combina con cualquier plantilla</span>
				</div>
				{#each AJUSTES as a (a.k)}
					<div class="ajuste">
						<span class="ajuste-nombre"><b id="aj-{a.k}">{a.t}</b><span>{a.s}</span></span>
						<span class="opciones" role="group" aria-labelledby="aj-{a.k}">
							{#each a.ops as o (o)}
								<button type="button" class="opcion" aria-pressed={borrador[a.k] === o} onclick={() => poner(a.k, o as never)}>
									{#if a.k === 'portada'}
										<span
											class="muestra-op"
											style:border-radius="7px"
											style:background={o === 'foto' && fotoPortada ? `center / cover url(${fotoPortada})` : data.colores.primario}
										></span>
									{:else if a.k === 'fondo'}
										<span class="muestra-op" style:border-radius="50%" style:background={FONDO_MUESTRA[o]}></span>
									{:else if a.k === 'titulos'}
										<span class="muestra-op letra" style:font-weight={PESO[o]}>Aa</span>
									{:else}
										<span class="muestra-op" style:border-radius={RADIO[o]} style:background={data.colores.primario}></span>
									{/if}
									{NOMBRES[o]}
								</button>
							{/each}
						</span>
					</div>
				{/each}
				<div class="ajuste">
					<span class="ajuste-nombre"><b id="aj-secciones">Secciones</b><span>Qué se muestra y en qué&nbsp;orden</span></span>
					<ul class="opciones secciones" aria-labelledby="aj-secciones">
						{#each orden as s (s)}
							{@const on = visibles.includes(s)}
							<li
								class="seccion"
								class:on
								draggable="true"
								ondragstart={() => (arrastrada = s)}
								ondragover={(e) => e.preventDefault()}
								ondrop={(e) => {
									e.preventDefault();
									if (arrastrada && arrastrada !== s) mover(arrastrada, s);
									arrastrada = null;
								}}
							>
								<button
									type="button"
									class="manija"
									aria-label="Mover {NOMBRES[s]} un lugar antes"
									disabled={orden.indexOf(s) === 0 || !on}
									onclick={() => mover(s)}><span class="icono" aria-hidden="true">drag_indicator</span></button
								>
								<button type="button" class="alternar" aria-pressed={on} onclick={() => alternar(s)}>
									{NOMBRES[s]}<span class="visualmente-oculto">{on ? ', se muestra' : ', oculta'}</span>
								</button>
							</li>
						{/each}
					</ul>
				</div>
			</section>
		</div>

		<aside class="lado" aria-label="Vista previa de tu página">
			<div class="lado-cabeza">
				<span class="vivo" aria-hidden="true"></span>
				<b>Vista previa</b>
				<span class="g-nota">{NOMBRES[borrador.plantilla]} · {borrador.plantilla === publicado.plantilla ? 'en uso' : 'probando'}</span>
			</div>
			<div class="telefono">
				<div class="pantalla">
					<iframe src={vista} title="Tu página con los cambios sin publicar" loading="lazy"></iframe>
				</div>
			</div>

			{#if form && 'error' in form && form.error}<Mensaje {form} que="publicar" />{/if}

			{#if cambios.length}
				<form
					method="POST"
					action="?/publicar"
					class="cambios tl-petroleo"
					use:enhance={() => {
						enviando = true;
						return async ({ result, update }) => {
							await update({ reset: false });
							enviando = false;
							if (result.type === 'success') {
								publicada = true;
								clearTimeout(timer);
								timer = setTimeout(() => (publicada = false), 4000);
							}
						};
					}}
				>
					<span class="cambios-titulo"><span class="icono lleno" aria-hidden="true">edit</span>Cambios sin publicar</span>
					<span class="cambios-resumen">{cambios.join(' · ')}. Tus clientes siguen viendo la página de&nbsp;antes.</span>
					<input type="hidden" name="plantilla" value={borrador.plantilla} />
					<input type="hidden" name="portada" value={borrador.portada} />
					<input type="hidden" name="fondo" value={borrador.fondo} />
					<input type="hidden" name="titulos" value={borrador.titulos} />
					<input type="hidden" name="forma" value={borrador.forma} />
					<input type="hidden" name="secciones" value={visibles.join(',')} />
					<span class="cambios-botones">
						<button type="button" class="descartar" onclick={descartar}>Descartar</button>
						<button class="publicar" disabled={enviando}>{enviando ? 'Publicando…' : 'Publicar'}</button>
					</span>
				</form>
			{:else if publicada}
				<p class="listo" role="status"><span class="icono lleno" aria-hidden="true">check_circle</span>Publicada. Tus clientes ya la&nbsp;ven.</p>
			{:else}
				<p class="quieto">
					<span class="icono lleno" aria-hidden="true">visibility</span>
					<span>Así ven tu página tus clientes. Probá una plantilla y nada cambia hasta que&nbsp;publiques.</span>
				</p>
			{/if}
		</aside>
	</div>
</div>

<style>
	.plantillas {
		gap: 20px;
	}
	.cabeza {
		display: flex;
		flex-wrap: wrap;
		align-items: flex-end;
		gap: 12px 16px;
	}
	.titulo {
		flex: 1 1 320px;
		gap: 8px;
	}
	.suave {
		color: var(--tl-sub);
	}
	.colores {
		display: flex;
		align-items: center;
		gap: 8px;
		min-height: 44px;
		font-size: 13px;
		color: var(--tl-sub);
		text-decoration: none;
	}
	.muestra {
		width: 22px;
		height: 22px;
		border-radius: 6px;
		box-shadow: inset 0 0 0 1px rgba(15, 35, 40, 0.12);
	}
	.cuerpo {
		display: grid;
		grid-template-columns: minmax(0, 1fr) 376px;
		gap: 24px;
		align-items: start;
	}
	.izquierda {
		gap: 20px;
	}
	.grilla {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 16px;
	}
	.tarjeta-plantilla {
		width: 100%;
		height: 100%;
		display: flex;
		flex-direction: column;
		gap: 12px;
		padding: 12px;
		border: 0;
		border-radius: 24px;
		background: var(--tl-papel);
		box-shadow: var(--tl-sh-tarjeta);
		font-family: var(--tl-font);
		color: var(--tl-tinta);
		text-align: left;
		cursor: pointer;
		transition:
			transform var(--tl-tocar) var(--tl-ease),
			box-shadow var(--tl-tocar) var(--tl-ease);
	}
	.tarjeta-plantilla:hover {
		transform: translateY(-4px);
	}
	.tarjeta-plantilla.en-uso {
		box-shadow:
			0 0 0 2px var(--tl-menta),
			var(--tl-sh-tarjeta);
	}
	.tarjeta-plantilla.en-vista {
		box-shadow:
			0 0 0 2.5px var(--tl-petroleo),
			0 24px 44px -24px rgba(15, 35, 40, 0.55);
	}
	.info {
		display: flex;
		flex-direction: column;
		gap: 4px;
		padding: 0 4px;
	}
	.nombre {
		display: flex;
		align-items: center;
		gap: 8px;
	}
	.nombre b {
		flex: 1;
		font-size: 17px;
	}
	.descripcion {
		font-size: 13px;
		line-height: 1.4;
		color: var(--tl-sub);
		text-wrap: pretty;
	}
	.probar {
		margin-top: auto;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 6px;
		min-height: 44px;
		border-radius: 999px;
		background: var(--tl-petroleo);
		color: #fff;
		font-size: 14px;
		font-weight: 700;
	}
	.probar .icono {
		font-size: 18px;
	}
	.probar.activo {
		background: transparent;
		color: var(--tl-menta-texto);
		box-shadow: inset 0 0 0 1.5px var(--tl-menta);
	}
	.estilo {
		padding: 20px 22px 10px;
		border-radius: 22px;
		gap: 0;
	}
	.estilo-cabeza {
		display: flex;
		flex-wrap: wrap;
		align-items: baseline;
		gap: 4px 10px;
		padding-bottom: 12px;
	}
	.estilo-cabeza h2 {
		flex: 1;
		font-size: 18px;
		letter-spacing: -0.01em;
	}
	.ajuste {
		display: grid;
		grid-template-columns: 180px minmax(0, 1fr);
		gap: 12px 16px;
		align-items: center;
		padding: 12px 0;
		border-top: 1px solid #f0e8d8;
	}
	.ajuste-nombre {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}
	.ajuste-nombre b {
		font-size: 15px;
	}
	.ajuste-nombre span {
		font-size: 12px;
		color: var(--tl-sub);
	}
	.opciones {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		margin: 0;
		padding: 0;
		list-style: none;
	}
	.opcion {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		min-height: 44px;
		padding: 0 16px 0 10px;
		border: 0;
		border-radius: 999px;
		background: var(--tl-papel);
		color: var(--tl-tinta);
		box-shadow: inset 0 0 0 1.5px #e0d5bf;
		font: 700 14px var(--tl-font);
		cursor: pointer;
		transition: background var(--tl-tocar) var(--tl-ease);
	}
	.opcion[aria-pressed='true'] {
		background: var(--tl-petroleo);
		color: #fff;
		box-shadow: none;
	}
	.muestra-op {
		width: 26px;
		height: 26px;
		flex: none;
		box-shadow: inset 0 0 0 1px rgba(15, 35, 40, 0.12);
	}
	.muestra-op.letra {
		display: grid;
		place-items: center;
		border-radius: 7px;
		background: #fff;
		color: var(--tl-tinta);
		font-size: 12px;
	}
	.seccion {
		display: inline-flex;
		align-items: center;
		min-height: 44px;
		border-radius: 999px;
		background: var(--tl-papel);
		box-shadow: inset 0 0 0 1.5px #e0d5bf;
		color: var(--tl-tinta);
	}
	.seccion.on {
		background: var(--tl-petroleo);
		color: #fff;
		box-shadow: none;
	}
	.seccion button {
		border: 0;
		background: none;
		color: inherit;
		font: 700 14px var(--tl-font);
		cursor: pointer;
		min-height: 44px;
	}
	.manija {
		display: grid;
		place-items: center;
		padding: 0 2px 0 10px;
		cursor: grab;
	}
	.manija .icono {
		font-size: 18px;
		color: #6f7d7a;
	}
	.on .manija .icono {
		color: var(--tl-menta);
	}
	.manija:disabled {
		cursor: default;
	}
	.alternar {
		padding: 0 16px 0 6px;
	}
	.lado {
		position: sticky;
		top: 16px;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.lado-cabeza {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 0 4px;
	}
	.lado-cabeza b {
		flex: 1;
		font-size: 15px;
	}
	.vivo {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: var(--tl-menta);
		animation: tl-pulse 1.6s ease-in-out infinite;
	}
	.telefono {
		width: 376px;
		max-width: 100%;
		height: 720px;
		border-radius: 52px;
		padding: 10px;
		background: linear-gradient(145deg, #2a3a3e, #0b1214);
		box-shadow:
			0 40px 70px -36px rgba(15, 35, 40, 0.8),
			inset 0 0 0 1.5px rgba(255, 255, 255, 0.12);
	}
	.pantalla {
		height: 100%;
		border-radius: 42px;
		overflow: hidden;
		background: var(--tl-crema-pagina);
		transform: translateZ(0);
	}
	iframe {
		display: block;
		width: 100%;
		height: 100%;
		border: 0;
	}
	.cambios {
		display: flex;
		flex-direction: column;
		gap: 12px;
		padding: 18px;
		border-radius: 22px;
		animation: tl-gin var(--tl-entrar) var(--tl-ease) both;
	}
	.cambios-titulo {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 15px;
		font-weight: 900;
	}
	.cambios-titulo .icono {
		font-size: 20px;
		color: var(--tl-menta);
	}
	.cambios-resumen {
		font-size: 13px;
		line-height: 1.45;
		color: rgba(255, 255, 255, 0.8);
		text-wrap: pretty;
	}
	.cambios-botones {
		display: flex;
		gap: 8px;
	}
	.descartar,
	.publicar {
		height: 44px;
		border: 0;
		border-radius: 999px;
		font: 700 14px var(--tl-font);
		cursor: pointer;
	}
	.descartar {
		flex: 1;
		background: transparent;
		box-shadow: inset 0 0 0 1.5px rgba(255, 255, 255, 0.3);
		color: #fff;
	}
	.publicar {
		flex: 1.4;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		font-weight: 900;
	}
	.publicar:disabled {
		opacity: 0.6;
	}
	.listo,
	.quieto {
		margin: 0;
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 14px 16px;
		border-radius: 18px;
	}
	.listo {
		background: var(--tl-menta-suave);
		color: #0f3d2b;
		font-size: 14px;
		font-weight: 700;
		animation: tl-gin var(--tl-entrar) var(--tl-ease) both;
	}
	.listo .icono {
		font-size: 22px;
		color: var(--tl-menta-texto);
	}
	.quieto {
		background: var(--tl-papel);
		font-size: 13px;
		line-height: 1.45;
		color: var(--tl-sub-2);
	}
	.quieto .icono {
		font-size: 20px;
		color: var(--tl-menta-texto);
	}

	@media (max-width: 1100px) {
		.cuerpo {
			grid-template-columns: minmax(0, 1fr);
		}
		.lado {
			position: static;
			align-items: center;
		}
		.lado > :not(.telefono) {
			align-self: stretch;
		}
	}
	@media (max-width: 640px) {
		.grilla {
			grid-template-columns: minmax(0, 1fr);
		}
		.ajuste {
			grid-template-columns: minmax(0, 1fr);
		}
		.estilo-cabeza {
			flex-direction: column;
		}
		.opcion,
		.seccion,
		.seccion button,
		.probar {
			min-height: 48px;
		}
		.telefono {
			height: 640px;
			border-radius: 44px;
			padding: 8px;
		}
		.pantalla {
			border-radius: 36px;
		}
		/* La acción principal queda al alcance del pulgar mientras haya cambios. */
		.cambios {
			position: sticky;
			bottom: 12px;
			z-index: 2;
			box-shadow: var(--tl-sh-alta);
		}
		.descartar,
		.publicar {
			height: 48px;
		}
	}
</style>
