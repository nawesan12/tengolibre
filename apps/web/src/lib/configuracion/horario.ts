/**
 * Horario semanal de un profesional: franjas por día (horario cortado = dos franjas el mismo día),
 * como las pide la API (`PUT /v1/panel/profesionales/{id}/horario`). Funciones puras, para el
 * formulario y sus pruebas.
 */
export type Franja = { dia: number; desde: string; hasta: string };
export type Semanal = { dia: number; hora: string };

/** La semana como la piensa un negocio: de lunes a domingo (0 es domingo, como en la API). */
export const SEMANA = [1, 2, 3, 4, 5, 6, 0];

/** Cuántas franjas se pueden cargar por día en el formulario (mañana y tarde). */
export const FRANJAS_POR_DIA = 2;

const reHora = /^([01]\d|2[0-3]):[0-5]\d$/;

export const esHora = (v: string) => reHora.test(v);

/** Las franjas de un día, ordenadas. */
export const franjasDe = (horario: Franja[], dia: number) =>
	horario.filter((f) => f.dia === dia).sort((a, b) => a.desde.localeCompare(b.desde));

/**
 * Del formulario a franjas: `abre-{dia}` dice si trabaja ese día y `desde-{dia}-{i}` /
 * `hasta-{dia}-{i}` son las franjas. Una franja a medio llenar se ignora; que no se pisen ni
 * terminen antes de empezar lo valida la API.
 */
export function horarioDeForm(f: FormData): Franja[] {
	const out: Franja[] = [];
	for (const dia of SEMANA) {
		if (f.get(`abre-${dia}`) !== 'on') continue;
		for (let i = 0; i < FRANJAS_POR_DIA; i++) {
			const desde = String(f.get(`desde-${dia}-${i}`) ?? '').slice(0, 5);
			const hasta = String(f.get(`hasta-${dia}-${i}`) ?? '').slice(0, 5);
			if (esHora(desde) && esHora(hasta)) out.push({ dia, desde, hasta });
		}
	}
	return out;
}

const posicion = (s: Semanal) => SEMANA.indexOf(s.dia);

/** Suma un bloqueo semanal sin repetir, ordenado de lunes a domingo y por hora. */
export function sumarSemanal(lista: Semanal[], nuevo: Semanal): Semanal[] {
	const sin = lista.filter((s) => !(s.dia === nuevo.dia && s.hora === nuevo.hora));
	return [...sin, nuevo].sort((a, b) => posicion(a) - posicion(b) || a.hora.localeCompare(b.hora));
}

export const sacarSemanal = (lista: Semanal[], dia: number, hora: string) =>
	lista.filter((s) => !(s.dia === dia && s.hora === hora));
