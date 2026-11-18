/**
 * El link de la página (tengolibre.com.ar/<slug>): cuándo va primero en Inicio y cómo se ve el
 * campo para cambiarlo. Las apps usan la misma regla (TuLink.swift / TuLink.kt).
 */
import { estadoCampo, type Disponibilidad, type EstadoCampo } from '#lib/alta/slug.ts';
import { sumarDias } from '#lib/operacion/fechas.ts';

/** Con menos turnos que esto en la ventana, el link va arriba de todo y en grande. */
export const POCOS_TURNOS = 20;

/** La ventana para contar movimiento: los últimos 30 días (lo que ya pasó y lo de hoy). */
export const ventanaMovimiento = (hoy: string) => ({ desde: sumarDias(hoy, -29), hasta: hoy });

/** Lo que devuelve /panel/link para un slug: como el alta, más «propia» (es el link de ahora). */
export type EstadoLink = 'libre' | 'propia' | 'tomada' | 'invalida';

/** El link como lo manda la API: el de ahora, los viejos que siguen andando y los cambios que quedan. */
export type Link = { slug: string; anteriores: { slug: string; dejadoEn: string }[]; restantes: number };

export function campoLink(slug: string, estado: EstadoLink | null): EstadoCampo {
	if (estado === 'propia')
		return { estado: 'libre', icono: 'check_circle', borde: '#17805a', color: '#4a5d61', mensaje: 'Es tu link de ahora.', ok: false };
	const d: Disponibilidad | null = estado ? { estado, sugerencias: [] } : null;
	const c = estadoCampo(slug, d, 'Escribí tu link nuevo');
	return c.estado === 'libre' ? { ...c, mensaje: 'Está libre.' } : c;
}
