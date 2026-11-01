/**
 * Lo que la parte pública usa además de lo que ya existía. La API ya lo manda con estas formas; sigue
 * opcional del lado del front: si una API vieja no lo manda, la pantalla se arma sin eso.
 * `horarioAtencion` une los horarios de los profesionales activos (franjas que se pisan quedan en
 * una, por día); `referencia` son los 6 primeros caracteres del id del turno, en mayúsculas.
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
