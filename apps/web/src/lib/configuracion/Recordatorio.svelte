<!--
	Un recordatorio automático (cumpleaños o «hace mucho que no venís», docs/api/recordatorios.md):
	prender, el texto corto, el cupón, a quiénes les toca esta semana y el mail tal cual llega. Se
	guarda solo al cambiar, como el resto de Configuración.
-->
<script lang="ts">
	import { enhance } from '$app/forms';
	import { sinReset } from './formularios.ts';
	import Ajuste from './Ajuste.svelte';
	import Mensaje from './Mensaje.svelte';
	import type { Form } from './mensajes.ts';
	import type { Recordatorio } from './avisos.ts';

	let { r, cupones, form }: { r: Recordatorio; cupones: { id: string; codigo: string }[]; form: Form } = $props();

	const guardar = (e: Event) => (e.currentTarget as HTMLFormElement).requestSubmit();
	const DIAS = [14, 21, 30, 45, 60, 90, 120, 180, 365].map((n) => ({ valor: n, texto: `${n} días` }));
	// El cupón elegido aunque ya no esté activo (para no perderlo al guardar otra cosa).
	const lista = $derived(r.cupon && !cupones.some((c) => c.id === r.cupon!.id) ? [...cupones, { id: r.cupon.id, codigo: r.cupon.codigo }] : cupones);
	const pegar = (s: string) => s.replace(/ (\S+)$/, ' $1');
</script>

<form method="POST" action="?/recordatorio" use:enhance={sinReset} onchange={guardar} class="recordatorio" aria-label={r.titulo}>
	<input type="hidden" name="tipo" value={r.tipo} />
	<Ajuste etiqueta={r.titulo} ayuda={r.regla} nombre="activo" clave="activo-{r.tipo}" valor={r.activo} sino />
	{#if r.tipo === 'extraniamos'}
		<Ajuste etiqueta="Le llega después de" ayuda="Sin volver desde su último turno atendido" nombre="dias" valor={r.dias} opciones={DIAS} />
	{/if}
	<div class="fila-campos">
		<label class="g-campo texto">
			Texto del mail
			<textarea name="texto" rows="2" maxlength="300">{r.texto}</textarea>
		</label>
		<label class="g-campo cupon">
			Cupón
			<select name="cuponId">
				<option value="" selected={!r.cuponId}>Sin cupón</option>
				{#each lista as c (c.id)}<option value={c.id} selected={c.id === r.cuponId}>{c.codigo}</option>{/each}
			</select>
			{#if r.cupon && !r.cupon.vigente}<span class="g-ayuda aviso-cupon"><span class="icono" aria-hidden="true">warning</span>Ya no está vigente: el mail sale sin&nbsp;cupón.</span>{/if}
		</label>
	</div>
	<button class="visualmente-oculto">Guardar {r.titulo}</button>
	<div class="mensaje"><Mensaje {form} que="recordatorio-{r.tipo}" /></div>

	<div class="semana">
		<span class="g-rotulo">{r.semanaTexto || 'Esta semana: nadie'}</span>
		{#if r.semana.length}
			<ul>
				{#each r.semana.slice(0, 4) as p (p.clienteId + p.fecha)}
					<li>
						<b>{p.nombre}</b>
						<span class="g-sub">{pegar(p.detalle)}</span>
						{#if p.enviado}
							<span class="g-estado ok"><span class="icono lleno" aria-hidden="true">mark_email_read</span>Enviado</span>
						{:else}
							<span class="g-estado neutro"><span class="icono" aria-hidden="true">schedule</span>{r.activo ? 'Por salir' : 'Saldría'}</span>
						{/if}
					</li>
				{/each}
			</ul>
			{#if r.semana.length > 4}<span class="g-sub">Y {r.semana.length - 4} más.</span>{/if}
		{/if}
	</div>

	<details class="g-plegable vista">
		<summary>Ver el mail · «{r.vistaPrevia.asunto}»</summary>
		<iframe title="El mail de {r.titulo} tal cual llega" srcdoc={r.vistaPrevia.html} sandbox="" loading="lazy"></iframe>
	</details>
</form>

<style>
	.recordatorio {
		display: flex;
		flex-direction: column;
		border-top: 1px solid #f0e8d8;
	}
	.recordatorio + :global(.recordatorio) {
		border-top: 6px solid var(--tl-crema);
	}
	.fila-campos {
		display: grid;
		grid-template-columns: minmax(0, 1fr) 180px;
		gap: 12px;
		padding: 4px 18px 12px;
	}
	.fila-campos textarea {
		min-height: 64px;
	}
	.aviso-cupon {
		display: flex;
		align-items: center;
		gap: 4px;
		color: #7a4a00;
	}
	.aviso-cupon .icono {
		font-size: 16px;
	}
	.mensaje:not(:empty) {
		padding: 0 18px 8px;
	}
	.mensaje :global(.aviso) {
		margin: 0;
		padding: 8px 12px;
		font-size: 13px;
	}
	.semana {
		display: flex;
		flex-direction: column;
		gap: 6px;
		padding: 10px 18px;
		border-top: 1px solid #f0e8d8;
	}
	.semana ul {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
	}
	.semana li {
		display: grid;
		grid-template-columns: minmax(0, 160px) minmax(0, 1fr) auto;
		align-items: center;
		gap: 4px 12px;
		min-height: 40px;
		font-size: 13px;
	}
	.vista {
		padding: 0 18px 8px;
		border-top: 1px solid #f0e8d8;
	}
	.vista summary {
		font-size: 13px;
		color: var(--tl-menta-texto);
		min-height: 48px;
	}
	iframe {
		display: block;
		width: 100%;
		height: 420px;
		margin-bottom: 10px;
		border: 0;
		border-radius: 12px;
		background: #fff;
		box-shadow: inset 0 0 0 1px #ece3d1;
	}
	@media (max-width: 640px) {
		.fila-campos {
			grid-template-columns: minmax(0, 1fr);
			padding: 4px 14px 12px;
		}
		.semana,
		.vista {
			padding-left: 14px;
			padding-right: 14px;
		}
		.semana li {
			grid-template-columns: minmax(0, 1fr) auto;
			padding: 6px 0;
		}
		.semana li .g-sub {
			grid-column: 1 / -1;
			grid-row: 2;
		}
	}
</style>
