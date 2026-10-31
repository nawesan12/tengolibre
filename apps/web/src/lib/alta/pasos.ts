/** Los pasos del onboarding (Onboarding v2 del handoff), en orden. */
export const PASOS = ['rubro', 'nombre', 'servicios', 'equipo', 'agenda', 'cobro', 'web', 'listo'] as const;
export type Paso = (typeof PASOS)[number];

export const esPaso = (v: string | null | undefined): v is Paso => !!v && (PASOS as readonly string[]).includes(v);

export const INFO: Record<Paso, { nombre: string; icono: string; consejo: string; saltable: boolean }> = {
	rubro: { nombre: 'Tu rubro', icono: 'storefront', consejo: 'Te dejamos servicios y horarios típicos de tu rubro. Todo se cambia después.', saltable: false },
	nombre: { nombre: 'El nombre', icono: 'badge', consejo: 'Es lo que van a ver tus clientes y tu dirección para compartir.', saltable: false },
	servicios: { nombre: 'Servicios', icono: 'spa', consejo: 'Precios finales, como los cobrás. Las clientas los ven antes de reservar.', saltable: true },
	equipo: { nombre: 'Equipo y horario', icono: 'groups', consejo: 'Si trabajás sola, dejalo así. Cada persona después tiene su propio horario.', saltable: true },
	agenda: { nombre: 'Tu día', icono: 'schedule', consejo: 'Descansos, tiempo entre turnos y feriados: lo que hace que la agenda sea la tuya.', saltable: true },
	cobro: { nombre: 'Seña', icono: 'payments', consejo: 'La seña baja las ausencias. Se cobra al reservar y va directo a tu cuenta.', saltable: true },
	web: { nombre: 'Tu página', icono: 'palette', consejo: 'Probá colores y plantillas: a la derecha ves cómo queda, al instante.', saltable: true },
	listo: { nombre: '¡Listo!', icono: 'celebration', consejo: 'Tu página ya está publicada. Compartila y esperá el primer turno.', saltable: false }
};

export const siguiente = (p: Paso): Paso => PASOS[Math.min(PASOS.indexOf(p) + 1, PASOS.length - 1)];
export const anterior = (p: Paso): Paso => PASOS[Math.max(PASOS.indexOf(p) - 1, 0)];

/** Los pasos que necesitan el negocio ya creado. */
export const conNegocio = (p: Paso) => PASOS.indexOf(p) >= PASOS.indexOf('servicios');

/** La URL de un paso. `n=1` en rubro y nombre: se está editando el negocio ya creado. */
export function urlPaso(p: Paso, extra: Record<string, string | undefined> = {}) {
	const q = new URLSearchParams({ paso: p });
	for (const [k, v] of Object.entries(extra)) if (v) q.set(k, v);
	return `/empezar?${q}`;
}
