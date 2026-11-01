/**
 * Avisos push del panel en este navegador (docs/api/push-web.md): el permiso se pide sólo al tocar
 * «Activar» en la tarjeta de Inicio; se registra el service worker (/sw-panel.js), se suscribe con
 * la clave VAPID de la API y la suscripción se guarda por /panel/push (el navegador nunca habla con
 * la API).
 */
function claveABytes(b64: string) {
	const relleno = '='.repeat((4 - (b64.length % 4)) % 4);
	const s = atob((b64 + relleno).replace(/-/g, '+').replace(/_/g, '/'));
	return Uint8Array.from(s, (c) => c.charCodeAt(0));
}

function aparato() {
	const ua = navigator.userAgent;
	const nav = /Edg\//.test(ua) ? 'Edge' : /Firefox\//.test(ua) ? 'Firefox' : /Chrome\//.test(ua) ? 'Chrome' : /Safari\//.test(ua) ? 'Safari' : 'Navegador';
	const so = /iPhone|iPad/.test(ua) ? 'iPhone' : /Android/.test(ua) ? 'Android' : /Mac/.test(ua) ? 'Mac' : /Windows/.test(ua) ? 'Windows' : 'Linux';
	return `${nav} · ${so}`;
}

export async function activarAvisos(clave: string): Promise<'listo' | 'negado' | 'error'> {
	if (!('serviceWorker' in navigator) || !('PushManager' in window) || !('Notification' in window)) return 'error';
	try {
		const permiso = await Notification.requestPermission();
		if (permiso !== 'granted') return 'negado';
		const reg = await navigator.serviceWorker.register('/sw-panel.js', { scope: '/panel' });
		await navigator.serviceWorker.ready;
		const sub = (await reg.pushManager.getSubscription()) ?? (await reg.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: claveABytes(clave) }));
		const r = await fetch('/panel/push', {
			method: 'POST',
			headers: { 'content-type': 'application/json' },
			body: JSON.stringify({ ...sub.toJSON(), aparato: aparato() })
		});
		return r.ok ? 'listo' : 'error';
	} catch {
		return 'error';
	}
}
