<!-- Paso 7: color, plantilla y ajustes del estilo. «Publicar mi página» manda el estilo entero. -->
<script lang="ts">
	import { enhance } from '$app/forms';
	import type { SubmitFunction } from '$app/forms';
	import Icono from '#lib/ui/Icono.svelte';
	import type { Borrador } from './estado.svelte.ts';
	import { COLORES, PLANTILLAS, type Estilo } from './tipos.ts';
	import Previa from './Previa.svelte';

	let { b, alEnviar, form, listo = $bindable() }: { b: Borrador; alEnviar: SubmitFunction; form: { error?: string } | null | undefined; listo: boolean } = $props();
	$effect(() => {
		listo = true;
	});

	type Ajuste = { clave: keyof Estilo; titulo: string; ayuda: string; ops: { v: string; t: string; muestra: string; radio?: string; aa?: string; peso?: number }[] };
	const ajustes = $derived<Ajuste[]>([
		{
			clave: 'portada',
			titulo: 'Portada',
			ayuda: 'Con tu mejor foto o sólo con tu color.',
			ops: [
				{ v: 'foto', t: 'Con foto', muestra: 'url(/fotos/manicura.jpg) center/cover' },
				{ v: 'color', t: 'Sólo color', muestra: b.color }
			]
		},
		{
			clave: 'fondo',
			titulo: 'Fondo',
			ayuda: 'El color del papel de toda la página.',
			ops: [
				{ v: 'crema', t: 'Crema', muestra: '#faf6ee', radio: '50%' },
				{ v: 'blanco', t: 'Blanco', muestra: '#ffffff', radio: '50%' },
				{ v: 'noche', t: 'Noche', muestra: '#14221f', radio: '50%' }
			]
		},
		{
			clave: 'titulos',
			titulo: 'Títulos',
			ayuda: 'Cuánto peso tiene tu nombre.',
			ops: [
				{ v: 'grandes', t: 'Grandes', muestra: '#fff', aa: 'Aa', peso: 900 },
				{ v: 'medianos', t: 'Medianos', muestra: '#fff', aa: 'Aa', peso: 700 },
				{ v: 'livianos', t: 'Livianos', muestra: '#fff', aa: 'Aa', peso: 400 }
			]
		},
		{
			clave: 'forma',
			titulo: 'Botones y tarjetas',
			ayuda: 'Redondos, suaves o rectos.',
			ops: [
				{ v: 'redondos', t: 'Redondos', muestra: b.color, radio: '999px' },
				{ v: 'suaves', t: 'Suaves', muestra: b.color, radio: '8px' },
				{ v: 'rectos', t: 'Rectos', muestra: b.color, radio: '2px' }
			]
		}
	]);
</script>

<form id="paso" method="POST" action="?/web" use:enhance={alEnviar} class="hoja" data-paso="web">
	<input type="hidden" name="color" value={b.color} />
	<input type="hidden" name="estilo" value={JSON.stringify(b.estilo)} />
	<h1>Elegí cómo se ve tu&nbsp;página</h1>
	<span class="rot">Tu color</span>
	<div role="radiogroup" aria-label="Color de tu marca" class="colores">
		{#each COLORES as [v, nombre] (v)}
			<button type="button" role="radio" aria-checked={b.color === v} aria-label={nombre} class="color" style:--c={v} onclick={() => (b.color = v)}>
				{#if b.color === v}<Icono nombre="check" tam={24} />{/if}
			</button>
		{/each}
	</div>

	<span class="rot">Plantilla</span>
	<div class="plantillas">
		{#each PLANTILLAS as [v, nombre] (v)}
			<button type="button" class="plantilla" aria-pressed={b.estilo.plantilla === v} onclick={() => (b.estilo.plantilla = v)}>
				<span class="mini" aria-hidden="true"><span class="escala"><Previa {b} plantilla={v} boton="" /></span></span>
				<span class="pl-n"><b>{nombre}</b>{#if b.estilo.plantilla === v}<Icono nombre="check_circle" lleno tam={20} />{/if}</span>
			</button>
		{/each}
	</div>

	<div class="estilo">
		<b class="estilo-t">Ajustá el estilo</b>
		{#each ajustes as a (a.clave)}
			<div class="ajuste">
				<span class="titulo-campo"><b>{a.titulo}</b><span>{a.ayuda}</span></span>
				<div role="group" aria-label={a.titulo} class="opciones">
					{#each a.ops as o (o.v)}
						<button type="button" class="opcion con-muestra" aria-pressed={b.estilo[a.clave] === o.v} onclick={() => (b.estilo[a.clave] = o.v)}>
							<span class="muestra" style:background={o.muestra} style:border-radius={o.radio ?? '7px'} style:font-weight={o.peso}>{o.aa ?? ''}</span>{o.t}
						</button>
					{/each}
				</div>
			</div>
		{/each}
	</div>
	{#if form?.error}<p class="error-paso" role="alert"><Icono nombre="error" lleno tam={16} />{form.error}</p>{/if}
</form>

<style>
	.colores {
		display: flex;
		flex-wrap: wrap;
		gap: 10px;
	}
	.color {
		width: 52px;
		height: 52px;
		border-radius: 16px;
		border: 0;
		background: var(--c);
		display: grid;
		place-items: center;
		color: #fff;
		cursor: pointer;
		transition: transform var(--tl-tocar);
	}
	.color:hover {
		transform: scale(1.08);
	}
	.color[aria-checked='true'] {
		box-shadow: 0 0 0 3px #fffdf9, 0 0 0 5px var(--c);
	}
	.color :global(.icono) {
		animation: tl-pop 0.3s both;
	}
	.rot {
		margin-top: 6px;
	}
	.plantillas {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(158px, 1fr));
		gap: 10px;
	}
	.plantilla {
		display: flex;
		flex-direction: column;
		gap: 10px;
		padding: 10px;
		border-radius: 20px;
		border: 0;
		background: #fffdf9;
		box-shadow: inset 0 0 0 1px #eee5d3;
		text-align: left;
		font-family: var(--tl-font);
		color: var(--tl-tinta);
		cursor: pointer;
		transition: transform var(--tl-tocar);
	}
	.plantilla:hover {
		transform: translateY(-2px);
	}
	.plantilla[aria-pressed='true'] {
		box-shadow: 0 0 0 2px #3ccc90, 0 16px 30px -20px rgba(15, 35, 40, 0.5);
	}
	.mini {
		position: relative;
		display: block;
		height: 168px;
		border-radius: 14px;
		overflow: hidden;
		box-shadow: inset 0 0 0 1px #eee5d3;
	}
	.escala {
		position: absolute;
		left: 0;
		top: 0;
		width: calc(100% / 0.62);
		height: calc(168px / 0.62);
		transform: scale(0.62);
		transform-origin: 0 0;
		pointer-events: none;
	}
	.pl-n {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 0 4px;
	}
	.pl-n b {
		flex: 1;
		font-size: 15px;
	}
	.pl-n :global(.icono) {
		color: #17805a;
	}
	@media (max-width: 899px) {
		.plantillas {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
		.colores {
			gap: 8px;
		}
		.color {
			width: 44px;
			height: 48px;
			border-radius: 14px;
		}
	}
	.estilo {
		display: flex;
		flex-direction: column;
		gap: 16px;
		padding: 18px;
		border-radius: 22px;
		background: #f8f3ea;
	}
	.estilo-t {
		font-size: 17px;
		letter-spacing: -0.01em;
	}
	.ajuste {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.ajuste .titulo-campo {
		margin-top: 0;
	}
	.con-muestra {
		padding: 0 16px 0 10px;
		font-size: 14px;
	}
	.muestra {
		width: 26px;
		height: 26px;
		flex: none;
		box-shadow: inset 0 0 0 1px rgba(15, 35, 40, 0.12);
		display: grid;
		place-items: center;
		font-size: 12px;
		color: #0f2328;
	}
</style>
