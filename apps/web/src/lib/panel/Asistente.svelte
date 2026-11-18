<!--
	El asistente de Inicio: se le pregunta en castellano («¿qué tengo mañana?», «pasá a Juana al jueves
	a las 5»). Contesta primero el intérprete de la API y, si no entiende, la IA. Lo que cambia la
	agenda vuelve como propuesta con «Confirmar» / «Cancelar»: nada se hace sin tocar el botón.
-->
<script lang="ts">
	import { invalidate } from '$app/navigation';
	import Icono from '#lib/ui/Icono.svelte';
	import { historial, lineaCupo, type EstadoAsistente, type Propuesta, type RespuestaAsistente } from './asistente.ts';

	let { estado }: { estado: EstadoAsistente | null } = $props();

	type EstadoPropuesta = 'esperando' | 'confirmando' | 'hecha' | 'cancelada';
	type Mensaje = {
		id: number;
		rol: 'usuario' | 'asistente';
		texto: string;
		error?: boolean;
		propuestas?: (Propuesta & { estado: EstadoPropuesta })[];
	};

	let mensajes = $state<Mensaje[]>([]);
	let texto = $state('');
	let pensando = $state(false);
	let tope = $state('');
	let usadas = $state<number | null>(null);
	let campo: HTMLInputElement | undefined = $state();
	let lista: HTMLElement | undefined = $state();
	let n = 0;

	const sugerencias = $derived(estado?.sugerencias ?? []);
	const cupo = $derived.by(() => {
		if (!estado) return '';
		if (usadas === null || !estado.cupo) return lineaCupo(estado);
		return lineaCupo({ ...estado, cupo: { ...estado.cupo, usadasMes: usadas } });
	});

	function sumar(m: Omit<Mensaje, 'id'>) {
		mensajes.push({ ...m, id: ++n });
		// Que se vea lo último sin mover toda la página.
		queueMicrotask(() => lista?.scrollTo({ top: lista.scrollHeight, behavior: 'smooth' }));
	}

	async function pedir(cuerpo: Record<string, unknown>) {
		const r = await fetch('/panel/asistente', {
			method: 'POST',
			headers: { 'content-type': 'application/json' },
			body: JSON.stringify(cuerpo)
		});
		const j = await r.json().catch(() => ({ error: 'No pudimos hablar con el asistente. Probá de nuevo en un rato.' }));
		if (!r.ok || j.error) throw new Error(j.error ?? 'No pudimos completar el pedido.');
		return j;
	}

	async function preguntar(p?: string) {
		const pregunta = (p ?? texto).trim();
		if (!pregunta || pensando) return;
		const previo = historial(mensajes);
		sumar({ rol: 'usuario', texto: pregunta });
		texto = '';
		pensando = true;
		try {
			const r = (await pedir({ accion: 'preguntar', pregunta, historial: previo })) as RespuestaAsistente;
			tope = r.tope?.mensaje ?? '';
			if (r.origen === 'ia' && usadas !== null) usadas += 1;
			else if (r.origen === 'ia' && estado?.cupo) usadas = estado.cupo.usadasMes + 1;
			sumar({ rol: 'asistente', texto: r.respuesta, propuestas: r.propuestas.map((x) => ({ ...x, estado: 'esperando' as const })) });
		} catch (e) {
			sumar({ rol: 'asistente', texto: (e as Error).message, error: true });
		} finally {
			pensando = false;
			campo?.focus();
		}
	}

	async function resolver(m: Mensaje, i: number, confirmar: boolean) {
		const prop = m.propuestas?.[i];
		if (!prop || prop.estado !== 'esperando') return;
		prop.estado = 'confirmando';
		try {
			const r = (await pedir({ accion: confirmar ? 'confirmar' : 'descartar', propuestaId: prop.id })) as { respuesta: string };
			prop.estado = confirmar ? 'hecha' : 'cancelada';
			sumar({ rol: 'asistente', texto: r.respuesta });
			if (confirmar) await invalidate('panel:turnos');
		} catch (e) {
			prop.estado = 'esperando';
			sumar({ rol: 'asistente', texto: (e as Error).message, error: true });
		}
	}

	function enviar(e: SubmitEvent) {
		e.preventDefault();
		preguntar();
	}
</script>

<section class="asistente" aria-labelledby="t-asistente">
	<div class="cab">
		<span class="chispa"><Icono nombre="auto_awesome" lleno tam={22} /></span>
		<h2 id="t-asistente">Preguntale al asistente</h2>
		{#if cupo}<span class="cupo tl-num">{cupo}</span>{/if}
	</div>

	{#if mensajes.length}
		<ol class="chat" bind:this={lista} aria-live="polite">
			{#each mensajes as m (m.id)}
				<li class="msj" class:yo={m.rol === 'usuario'} class:error={m.error}>
					<p>{m.texto}</p>
					{#each m.propuestas ?? [] as p, i (p.id)}
						<div class="propuesta">
							<p class="resumen">{p.resumen}</p>
							{#if p.estado === 'hecha'}
								<span class="listo"><Icono nombre="check_circle" lleno tam={18} />Hecho</span>
							{:else if p.estado === 'cancelada'}
								<span class="listo apagado">Cancelada</span>
							{:else}
								<div class="botones">
									<button class="confirmar" onclick={() => resolver(m, i, true)} disabled={p.estado === 'confirmando'}>
										{p.estado === 'confirmando' ? 'Confirmando…' : 'Confirmar'}
									</button>
									<button class="cancelar" onclick={() => resolver(m, i, false)} disabled={p.estado === 'confirmando'}>Cancelar</button>
								</div>
							{/if}
						</div>
					{/each}
				</li>
			{/each}
			{#if pensando}<li class="msj pensando" aria-label="Pensando"><span></span><span></span><span></span></li>{/if}
		</ol>
	{:else if sugerencias.length}
		<div class="sugerencias" aria-label="Ejemplos">
			{#each sugerencias.slice(0, 4) as s (s)}
				<button type="button" onclick={() => preguntar(s)}>{s}</button>
			{/each}
		</div>
	{/if}

	{#if tope}<p class="aviso tope" role="status"><Icono nombre="info" tam={18} />{tope}</p>{/if}

	<form onsubmit={enviar}>
		<input
			bind:this={campo}
			bind:value={texto}
			name="pregunta"
			maxlength="500"
			autocomplete="off"
			placeholder="Escribí o dictá tu pregunta"
			aria-label="Tu pregunta"
		/>
		<button class="mandar" disabled={pensando || !texto.trim()} aria-label="Preguntar"><Icono nombre="arrow_upward" tam={22} /></button>
	</form>
</section>

<style>
	.asistente {
		display: flex;
		flex-direction: column;
		gap: 12px;
		padding: 16px;
		border-radius: 18px;
		background: var(--tl-papel);
		box-shadow: 0 10px 20px -16px rgba(15, 35, 40, 0.45);
	}
	.cab {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 6px 10px;
	}
	.chispa {
		display: grid;
		place-items: center;
		width: 36px;
		height: 36px;
		border-radius: 50%;
		background: var(--tl-menta-suave);
		color: var(--tl-menta-texto);
		flex: none;
	}
	.cab h2 {
		flex: 1;
		margin: 0;
		font-size: 18px;
		letter-spacing: -0.02em;
	}
	.cupo {
		font-size: 12px;
		color: var(--tl-sub);
	}
	.sugerencias {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}
	.sugerencias button {
		min-height: 36px;
		padding: 0 14px;
		border: 1px solid var(--tl-linea-2);
		border-radius: 999px;
		background: transparent;
		color: var(--tl-tinta);
		font: 700 13px var(--tl-font);
		cursor: pointer;
	}
	.sugerencias button:hover {
		border-color: var(--tl-menta-texto);
		color: var(--tl-menta-texto);
	}
	.chat {
		display: flex;
		flex-direction: column;
		gap: 10px;
		margin: 0;
		padding: 0;
		list-style: none;
		max-height: 420px;
		overflow-y: auto;
		overscroll-behavior: contain;
	}
	.msj {
		align-self: flex-start;
		max-width: min(100%, 560px);
		padding: 10px 14px;
		border-radius: 18px 18px 18px 6px;
		background: var(--tl-crema);
		font-size: 15px;
		line-height: 1.45;
	}
	.msj p {
		margin: 0;
		white-space: pre-line;
		overflow-wrap: anywhere;
	}
	.msj.yo {
		align-self: flex-end;
		border-radius: 18px 18px 6px 18px;
		background: var(--tl-petroleo);
		color: #fff;
	}
	.msj.error {
		background: var(--peligro-suave);
		color: var(--peligro);
	}
	.propuesta {
		margin-top: 10px;
		padding: 12px;
		border-radius: 14px;
		background: var(--tl-papel);
		border: 2px dashed var(--tl-linea-2);
	}
	.propuesta .resumen {
		font-weight: 700;
	}
	.botones {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		margin-top: 10px;
	}
	.confirmar,
	.cancelar {
		min-height: 44px;
		padding: 0 18px;
		border: 0;
		border-radius: 999px;
		font: 700 14px var(--tl-font);
		cursor: pointer;
	}
	.confirmar {
		background: var(--tl-petroleo);
		color: #fff;
	}
	.cancelar {
		background: transparent;
		color: #a3241a;
	}
	.confirmar:disabled,
	.cancelar:disabled {
		opacity: 0.6;
		cursor: default;
	}
	.listo {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		margin-top: 8px;
		font-size: 14px;
		font-weight: 700;
		color: var(--tl-menta-texto);
	}
	.listo.apagado {
		color: var(--tl-sub);
	}
	.pensando {
		display: flex;
		gap: 5px;
		padding: 14px 16px;
	}
	.pensando span {
		width: 7px;
		height: 7px;
		border-radius: 50%;
		background: var(--tl-sub);
		animation: latido 1.1s infinite ease-in-out;
	}
	.pensando span:nth-child(2) {
		animation-delay: 0.15s;
	}
	.pensando span:nth-child(3) {
		animation-delay: 0.3s;
	}
	@keyframes latido {
		0%,
		80%,
		100% {
			opacity: 0.25;
		}
		40% {
			opacity: 1;
		}
	}
	.tope {
		margin: 0;
		background: #fff4e0;
		color: #7a4a00;
	}
	form {
		display: flex;
		gap: 8px;
	}
	input {
		flex: 1;
		min-width: 0;
		min-height: 48px;
		padding: 0 16px;
		border: 1px solid var(--tl-linea-2);
		border-radius: 999px;
		background: #fff;
		color: var(--tl-tinta);
		font: 400 16px var(--tl-font);
	}
	input:focus-visible {
		outline: 2px solid var(--tl-menta-texto);
		outline-offset: 1px;
	}
	.mandar {
		display: grid;
		place-items: center;
		width: 48px;
		height: 48px;
		flex: none;
		border: 0;
		border-radius: 50%;
		background: var(--tl-menta);
		color: var(--tl-petroleo-hondo);
		cursor: pointer;
	}
	.mandar:disabled {
		opacity: 0.5;
		cursor: default;
	}
	@media (prefers-reduced-motion: reduce) {
		.pensando span {
			animation: none;
		}
	}
</style>
