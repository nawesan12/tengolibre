// Hace aparecer un bloque cuando entra en pantalla (una sola vez). Sin IntersectionObserver o con
// «menos movimiento» pedido por la persona, el bloque está visible desde el principio.
export function aparecer(nodo: HTMLElement, demora = 0) {
	const quieto =
		typeof IntersectionObserver === 'undefined' ||
		matchMedia('(prefers-reduced-motion: reduce)').matches;
	if (quieto) return;
	nodo.classList.add('aparece', 'oculto');
	nodo.style.transitionDelay = `${demora}ms`;
	const obs = new IntersectionObserver(
		(entradas) => {
			for (const e of entradas) {
				if (e.isIntersecting) {
					nodo.classList.remove('oculto');
					obs.disconnect();
				}
			}
		},
		{ rootMargin: '0px 0px -10% 0px' }
	);
	obs.observe(nodo);
	return { destroy: () => obs.disconnect() };
}
