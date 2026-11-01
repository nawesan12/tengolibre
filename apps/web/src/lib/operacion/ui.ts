import { dia, hora } from '#lib/formato.ts';

/** «HOY 10:30», «MAR 03 10:30» (rótulos de cuándo, en mayúsculas). */
export function cuandoCorto(iso: string, conHora = true) {
	const d = dia(iso);
	const p = Object.fromEntries(
		new Intl.DateTimeFormat('es-AR', { weekday: 'short', day: '2-digit', timeZone: 'America/Argentina/Buenos_Aires' })
			.formatToParts(new Date(iso))
			.map((x) => [x.type, x.value])
	);
	const base = d === dia() ? 'HOY' : `${String(p.weekday).replace('.', '').toUpperCase()} ${p.day}`;
	return conHora ? `${base} ${hora(iso)}` : base;
}

/**
 * El `action` de un formulario que conserva la consulta de la página (filtros, día, pestaña): con
 * SvelteKit 3, después de la acción se vuelve a la URL del `action`, y un «?/nombre» suelto la pierde.
 */
export function accion(url: { search: string }, nombre: string) {
	const q = new URLSearchParams(url.search);
	for (const k of [...q.keys()]) if (k.startsWith('/')) q.delete(k);
	const s = q.toString();
	return `?${s ? `${s}&` : ''}/${nombre}`;
}

/** «14 SEP» */
export const fechaCorta = (iso: string) =>
	new Intl.DateTimeFormat('es-AR', { day: '2-digit', month: 'short', timeZone: 'America/Argentina/Buenos_Aires' })
		.format(new Date(iso))
		.replace('.', '')
		.replace(' de ', ' ')
		.toUpperCase();

/** Fondo y texto legibles para una etiqueta con el color que eligió el negocio. */
export const colorEtiqueta = (c: string) => ({
	fondo: `color-mix(in srgb, ${c} 16%, #fffdf9)`,
	texto: `color-mix(in srgb, ${c} 62%, #0f2328)`
});
