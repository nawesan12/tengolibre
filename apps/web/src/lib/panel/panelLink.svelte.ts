/** El panel «Tu link» de la barra: cualquier pantalla lo abre (p. ej. «Cambiar link» de Inicio). */
export const panelLink = $state({ abierto: false, cambiar: false });

export function abrirLink(cambiar = false) {
	panelLink.cambiar = cambiar;
	panelLink.abierto = true;
}
