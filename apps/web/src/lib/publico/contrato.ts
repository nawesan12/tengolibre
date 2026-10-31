/**
 * Lo que la parte pública supone de la API y todavía no existe (o no se sabe si existe). Todo es
 * opcional del lado del front: si la API no lo manda, la pantalla se arma sin eso. Al integrar,
 * ajustar acá y en los load que lo usan.
 *
 * 1. GET /v1/p/{slug} → `horarioAtencion`: FranjaAtencion[] (los días que abre el negocio). Con eso
 *    la página muestra «Abierto hasta las 20» y el bloque de horarios (LUN–VIE 10–20).
 *
 * 2. GET /v1/t/{codigo} → `sena`: PagoTurno | null (lo cobrado por adelantado), `referencia`
 *    (código corto para mostrar en el local; el `codigo` del link es secreto) y `conMembresia`.
 *    Con eso Mi turno muestra el sello «SEÑA OK» / «SOCIA», «Seña pagada» y «En el local».
 *
 * 3. Hueco ofrecido a la lista de espera (rutas /espera/[token]):
 *    GET  /v1/espera/ofertas/{token}         → OfertaHueco
 *    POST /v1/espera/ofertas/{token}/tomar   → { codigo: string; pago?: { url?: string } }
 *         (el turno queda a nombre de quien esperaba; si hay seña, `pago.url` es Mercado Pago)
 *    POST /v1/espera/ofertas/{token}/pasar   → 204 (no le sirve: pasa al siguiente de la lista)
 *    Errores con código: `oferta_vencida`, `oferta_tomada`, `horario_ocupado`.
 */
import type { Estilo } from './estilo.ts';

/** Una franja del horario de atención: dia 0 = domingo; horas "HH:MM". */
export type FranjaAtencion = { dia: number; desde: string; hasta: string };

export type PagoTurno = { monto: number; estado: 'aprobado' | 'pendiente' | 'devuelto' | 'rechazado' };

export type OfertaHueco = {
	estado: 'ofrecida' | 'tomada' | 'vencida' | 'pasada';
	negocio: { nombre: string; slug: string; logo: string | null; colorPrimario: string; colorAcento: string; estilo?: Estilo; zonaHoraria: string };
	servicio: string;
	profesional: string;
	inicio: string;
	fin: string;
	precio: number;
	/** Hasta cuándo lo puede tomar (después pasa al siguiente). */
	venceEn: string;
	/** Si ya lo tomó: el código de su turno. */
	turnoCodigo?: string | null;
};
