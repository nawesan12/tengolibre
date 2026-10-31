/**
 * Cupones, cobros y planes: del formulario a lo que pide la API. Las fechas de un cupón se cargan
 * como días y se mandan como instantes en hora de Argentina (UTC-3, sin horario de verano).
 */
const vacio = (v: FormDataEntryValue | null) => v === null || String(v).trim() === '';
const entero = (v: FormDataEntryValue | null) => (vacio(v) ? null : Math.trunc(Number(v)));

/** Un día AAAA-MM-DD como instante: el principio del día, o el final si es `hasta`. */
export const diaComoInstante = (d: string, finDelDia = false) =>
	/^\d{4}-\d{2}-\d{2}$/.test(d) ? `${d}T${finDelDia ? '23:59:59' : '00:00:00'}-03:00` : null;

/** El día AAAA-MM-DD de un instante, en Argentina (para volver a mostrarlo en el formulario). */
export const instanteComoDia = (i: string | null | undefined) =>
	i ? new Intl.DateTimeFormat('en-CA', { timeZone: 'America/Argentina/Buenos_Aires' }).format(new Date(i)) : '';

export type EntradaCupon = {
	codigo?: string;
	tipo: string;
	valor: number;
	minimo: number | null;
	maxUsos: number | null;
	desde?: string;
	hasta?: string;
	activo: boolean;
};

/** Del formulario de un cupón. En la edición no va el código (la API no deja cambiarlo). */
export function cuponDeForm(f: FormData, conCodigo: boolean): EntradaCupon {
	const c: EntradaCupon = {
		tipo: String(f.get('tipo') ?? 'porcentaje'),
		valor: entero(f.get('valor')) ?? 0,
		minimo: entero(f.get('minimo')),
		maxUsos: entero(f.get('maxUsos')),
		activo: f.get('activo') === 'on'
	};
	const desde = diaComoInstante(String(f.get('desde') ?? ''));
	const hasta = diaComoInstante(String(f.get('hasta') ?? ''), true);
	if (desde) c.desde = desde;
	if (hasta) c.hasta = hasta;
	if (conCodigo) c.codigo = String(f.get('codigo') ?? '').trim().toUpperCase();
	return c;
}

export const MODOS_COBRO: Record<string, string> = {
	sin_cobro: 'Sin cobro por adelantado',
	porcentaje: 'Seña en porcentaje',
	fija: 'Seña de monto fijo',
	total: 'El total al reservar'
};

/** Cómo se lee un descuento: "20 %" o "$ 1.500". */
export const descuento = (tipo: string, valor: number, pesos: (n: number) => string) =>
	tipo === 'porcentaje' ? `${valor} %` : pesos(valor);

/** Los beneficios de un plan: uno por renglón, hasta 12. */
export const beneficiosDe = (v: FormDataEntryValue | null) =>
	String(v ?? '')
		.split('\n')
		.map((b) => b.trim())
		.filter(Boolean)
		.slice(0, 12);

/** Cuánto se puede devolver todavía de un pago. */
export const resta = (p: { monto: number; devuelto: number }) => Math.max(p.monto - p.devuelto, 0);
