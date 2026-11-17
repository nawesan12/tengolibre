// Service worker de los avisos push del panel (docs/api/push-web.md). Sólo muestra el aviso y, al
// tocarlo, abre (o enfoca) la pantalla a la que lleva. Es el único: lo registran la tarjeta de Inicio
// y Configuración › Lo que te llega, con scope /panel.
self.addEventListener('install', () => self.skipWaiting());
self.addEventListener('activate', (e) => e.waitUntil(self.clients.claim()));
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
	const { url, negocio } = ev.notification.data || {};
	let destino = new URL(url || '/panel', self.location.origin);
	// El aviso puede ser de otro negocio de la persona: se abre con ese negocio activo. El worker no
	// sabe cuál está elegido (va en una cookie httpOnly), así que pasa siempre por el cambio, que valida
	// que sea suyo y vuelve a la pantalla del aviso (si ya era el activo, no cambia nada).
	if (negocio && destino.origin === self.location.origin && destino.pathname.startsWith('/panel'))
		destino = new URL(
			`/panel/negocio?${new URLSearchParams({ ir: negocio, volver: destino.pathname + destino.search })}`,
			self.location.origin
		);
	destino = destino.href;
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
