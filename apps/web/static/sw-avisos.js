/*
 * Service worker de los avisos del panel (docs/api/push-web.md). Sólo muestra los push y abre el
 * panel al tocarlos: no intercepta pedidos ni guarda nada en caché.
 */
self.addEventListener('install', () => self.skipWaiting());
self.addEventListener('activate', (e) => e.waitUntil(self.clients.claim()));

self.addEventListener('push', (e) => {
	let d = {};
	try {
		d = e.data ? e.data.json() : {};
	} catch {
		d = { title: 'Tengo Libre', body: e.data ? e.data.text() : '' };
	}
	const url = d.url || '/panel/notificaciones';
	e.waitUntil(
		self.registration.showNotification(d.title || 'Tengo Libre', {
			body: d.body || '',
			tag: d.tag,
			icon: '/marca-icono.png',
			badge: '/marca-icono.png',
			data: { url, negocio: d.negocio || '' }
		})
	);
});

self.addEventListener('notificationclick', (e) => {
	e.notification.close();
	// `negocio` viaja en los datos: cambiar de negocio antes de abrirlo queda para el panel.
	const { url } = e.notification.data || {};
	e.waitUntil(
		self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((ventanas) => {
			for (const v of ventanas) {
				if (new URL(v.url).origin === self.location.origin && 'focus' in v) {
					v.navigate(url);
					return v.focus();
				}
			}
			return self.clients.openWindow(url);
		})
	);
});
