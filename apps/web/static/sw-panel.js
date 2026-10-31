// Service worker de los avisos push del panel (docs/api/push-web.md). Sólo muestra el aviso y, al
// tocarlo, abre (o enfoca) la pantalla a la que lleva.
self.addEventListener('push', (ev) => {
	let d = {};
	try {
		d = ev.data ? ev.data.json() : {};
	} catch {
		d = { title: 'Tengo Libre', body: ev.data ? ev.data.text() : '' };
	}
	ev.waitUntil(
		self.registration.showNotification(d.title || 'Tengo Libre', {
			body: d.body || '',
			tag: d.tag,
			icon: '/marca/favicon_64.png',
			badge: '/marca/favicon_64.png',
			data: { url: d.url || '/panel/notificaciones', negocio: d.negocio || '' }
		})
	);
});

self.addEventListener('notificationclick', (ev) => {
	ev.notification.close();
	const { url } = ev.notification.data || {};
	const destino = new URL(url || '/panel', self.location.origin).href;
	ev.waitUntil(
		self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((ventanas) => {
			for (const v of ventanas) {
				if (v.url.startsWith(self.location.origin + '/panel') && 'focus' in v) {
					v.navigate(destino);
					return v.focus();
				}
			}
			return self.clients.openWindow(destino);
		})
	);
});
