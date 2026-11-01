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
 *    Con eso Mi turno muestra el sello «SEÑA OK» / «MEMBRESÍA», «Seña pagada» y «En el local».
 */

/** Una franja del horario de atención: dia 0 = domingo; horas "HH:MM". */
export type FranjaAtencion = { dia: number; desde: string; hasta: string };

export type PagoTurno = { monto: number; estado: 'aprobado' | 'pendiente' | 'devuelto' | 'rechazado' };
