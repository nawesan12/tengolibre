<!--
	Activar los avisos del panel en este navegador (docs/api/push-web.md): el permiso se pide recién al
	tocar «Activar»; el navegador se suscribe con la clave VAPID y la suscripción va a la API por la
	acción del servidor (el navegador nunca habla con la API).
-->
<script lang="ts">
	import { enhance } from '$app/forms';
	import { onMount } from 'svelte';
	import type { SuscripcionWeb } from './avisos.ts';

	let { navegador }: { navegador: SuscripcionWeb | null } = $props();

	let soporta = $state(true);
	let permiso = $state<NotificationPermission>('default');
	let trabajando = $state(false);
	let problema = $state('');
	let campo = $state<HTMLInputElement>();
	let aparato = $state<HTMLInputElement>();
	let formSuscribir = $state<HTMLFormElement>();

	onMount(() => {
		soporta = 'serviceWorker' in navigator && 'PushManager' in window && 'Notification' in window;
		if (soporta) permiso = Notification.permission;
	});

	/** La clave VAPID en base64url, como la pide pushManager.subscribe. */
	function bytes(b64: string) {
		const s = atob((b64 + '='.repeat((4 - (b64.length % 4)) % 4)).replace(/-/g, '+').replace(/_/g, '/'));
		return Uint8Array.from(s, (c) => c.charCodeAt(0));
	}
	function nombreDelAparato() {
		const ua = navigator.userAgent;
		const b = /Edg\//.test(ua) ? 'Edge' : /Firefox\//.test(ua) ? 'Firefox' : /Chrome\//.test(ua) ? 'Chrome' : /Safari\//.test(ua) ? 'Safari' : 'Navegador';
		const s = /iPhone|iPad/.test(ua) ? 'iPhone' : /Android/.test(ua) ? 'Android' : /Mac/.test(ua) ? 'Mac' : /Windows/.test(ua) ? 'Windows' : 'Linux';
		return `${b} · ${s}`;
	}

	async function activar() {
		if (!navegador?.clave) return;
		problema = '';
		trabajando = true;
		try {
			permiso = await Notification.requestPermission();
			if (permiso !== 'granted') {
				problema = 'El navegador no dio permiso. Lo podés cambiar en los ajustes del sitio.';
				return;
			}
			const reg = await navigator.serviceWorker.register('/sw-panel.js', { scope: '/panel' });
			await navigator.serviceWorker.ready;
			const sub =
				(await reg.pushManager.getSubscription()) ??
				(await reg.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: bytes(navegador.clave) }));
			if (campo && aparato && formSuscribir) {
				campo.value = JSON.stringify(sub.toJSON());
				aparato.value = nombreDelAparato();
				formSuscribir.requestSubmit();
			}
		} catch {
			problema = 'No pudimos activarlos en este navegador.';
		} finally {
			trabajando = false;
		}
	}

	async function apagarLocal() {
		try {
			const reg = await navigator.serviceWorker.getRegistration('/panel');
			await (await reg?.pushManager.getSubscription())?.unsubscribe();
		} catch {
			/* lo de la API se borra igual */
		}
	}
</script>

<div class="navegador">
	<span class="icono lleno" aria-hidden="true">notifications_active</span>
	<span class="texto">
		<b>En este navegador</b>
		{#if !navegador?.clave}
			<span>Los avisos del navegador todavía no están&nbsp;habilitados.</span>
		{:else if !soporta}
			<span>Este navegador no recibe avisos. En el iPhone, instalá el panel en la pantalla de&nbsp;inicio.</span>
		{:else if navegador.suscripta}
			<span class="g-estado ok"><span class="icono lleno" aria-hidden="true">check_circle</span>Activados</span>
		{:else if permiso === 'denied'}
			<span class="g-estado error"><span class="icono" aria-hidden="true">block</span>Bloqueados en el navegador</span>
		{:else}
			<span>Enterate en el momento de cada reserva, aunque tengas el panel&nbsp;cerrado.</span>
		{/if}
		{#if problema}<span class="g-estado error" role="alert"><span class="icono" aria-hidden="true">error</span>{problema}</span>{/if}
	</span>
	{#if navegador?.clave && soporta}
		{#if navegador.suscripta}
			<form method="POST" action="?/desuscribir" use:enhance={() => { apagarLocal(); return async ({ update }) => update({ reset: false }); }}>
				<button class="g-boton chico">Apagar</button>
			</form>
		{:else if permiso !== 'denied'}
			<button type="button" class="g-boton oscuro chico" onclick={activar} disabled={trabajando}>{trabajando ? 'Activando…' : 'Activar'}</button>
		{/if}
	{/if}
	<form method="POST" action="?/suscribir" use:enhance={() => async ({ update }) => update({ reset: false })} bind:this={formSuscribir} hidden>
		<input type="hidden" name="suscripcion" bind:this={campo} />
		<input type="hidden" name="aparato" bind:this={aparato} />
	</form>
</div>

<style>
	.navegador {
		display: flex;
		align-items: center;
		gap: 12px;
		min-height: 60px;
		padding: 10px 18px;
		border-top: 1px solid #f0e8d8;
	}
	.navegador > .icono {
		font-size: 22px;
		color: var(--tl-menta-texto);
	}
	.texto {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
		font-size: 12px;
		color: var(--tl-sub);
	}
	.texto b {
		font-size: 14px;
		color: var(--tl-tinta);
	}
	@media (max-width: 640px) {
		.navegador {
			padding: 10px 14px;
			flex-wrap: wrap;
		}
	}
</style>
