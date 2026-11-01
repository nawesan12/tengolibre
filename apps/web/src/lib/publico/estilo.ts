/**
 * El estilo de la web pública (handoff v2, «Reserva v2»): plantilla × ajustes × colores del negocio.
 * Todo sale en variables CSS (--w-*) que usan PaginaNegocio, la reserva, Mi turno y la reseña.
 * Las cuentas (contraste, mezcla) son las del prototipo, tal cual.
 */

export const PLANTILLAS = ['carta', 'vidriera', 'sobria', 'galeria', 'editorial', 'bloques'] as const;
export const PORTADAS = ['foto', 'color'] as const;
export const FONDOS = ['crema', 'blanco', 'noche'] as const;
export const TITULOS = ['grandes', 'medianos', 'livianos'] as const;
export const FORMAS = ['redondos', 'suaves', 'rectos'] as const;

export type Plantilla = (typeof PLANTILLAS)[number];
export type Portada = (typeof PORTADAS)[number];
export type Fondo = (typeof FONDOS)[number];
export type Titulos = (typeof TITULOS)[number];
export type Forma = (typeof FORMAS)[number];

/** El estilo publicado, como lo devuelve la API en `web.estilo`. */
export type Estilo = { plantilla: Plantilla; portada: Portada; fondo: Fondo; titulos: Titulos; forma: Forma };

export const ESTILO_BASE: Estilo = { plantilla: 'carta', portada: 'foto', fondo: 'crema', titulos: 'grandes', forma: 'redondos' };

const de = <T extends string>(v: unknown, lista: readonly T[]): T => (lista.includes(v as T) ? (v as T) : lista[0]);

/** Completa un estilo que puede venir a medias (o de una API vieja) con los valores por defecto. */
export function completar(e: Partial<Record<keyof Estilo, unknown>> | null | undefined): Estilo {
	return {
		plantilla: de(e?.plantilla, PLANTILLAS),
		portada: de(e?.portada, PORTADAS),
		fondo: de(e?.fondo, FONDOS),
		titulos: de(e?.titulos, TITULOS),
		forma: de(e?.forma, FORMAS)
	};
}

const esHex = (h: string | null | undefined): h is string => /^#[0-9a-f]{6}$/i.test(h ?? '');
const canales = (hex: string) => [0, 2, 4].map((i) => parseInt(hex.slice(1 + i, 3 + i), 16));

export function luminancia(hex: string) {
	const c = canales(hex)
		.map((v) => v / 255)
		.map((v) => (v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4)));
	return 0.2126 * c[0] + 0.7152 * c[1] + 0.0722 * c[2];
}

export function contraste(a: string, b: string) {
	const x = luminancia(a);
	const y = luminancia(b);
	return (Math.max(x, y) + 0.05) / (Math.min(x, y) + 0.05);
}

/** Mezcla `hex` hacia `hacia` en proporción t (0 = igual, 1 = el otro). */
export function mezclar(hex: string, hacia: string, t: number) {
	const a = canales(hex);
	const b = canales(hacia);
	return '#' + a.map((v, i) => Math.round(v * (1 - t) + b[i] * t).toString(16).padStart(2, '0')).join('');
}

export const rgba = (hex: string, a: number) => `rgba(${canales(hex).join(',')},${a})`;

/** El texto que va sobre el color del negocio: blanco o tinta, el que más contraste. */
export const textoSobre = (hex: string) => (contraste(hex, '#ffffff') >= contraste(hex, '#14181a') ? '#ffffff' : '#14181a');

const FONDO = {
	crema: { bg: '#faf6ee', papel: '#fffdf9', campo: '#ffffff', chip: '#f4eee2', socios: '#f3e8d0', sociosTxt: '#4a3a17', ink: '#0f2328', sub: '#4a5d61', sub2: '#33474b', line: '#eee5d3', line2: '#e0d5bf' },
	blanco: { bg: '#ffffff', papel: '#f7f8f7', campo: '#ffffff', chip: '#f1f3f2', socios: '#f4efe4', sociosTxt: '#4a3a17', ink: '#0f2328', sub: '#4a5d61', sub2: '#33474b', line: '#e6e9e8', line2: '#d5dad8' },
	noche: { bg: '#14221f', papel: '#1d2f2b', campo: '#22362f', chip: '#26393a', socios: '#2e2a1d', sociosTxt: '#f1dfb4', ink: '#f4eee2', sub: '#b9c4c1', sub2: '#d6dcda', line: '#2f4440', line2: '#3c524d' }
} as const;
const TIT = { grandes: ['700', '-0.055em'], medianos: ['650', '-0.03em'], livianos: ['500', '-0.01em'] } as const;
const FORMA = { redondos: ['999px', '24px', '14px'], suaves: ['12px', '16px', '10px'], rectos: ['4px', '6px', '4px'] } as const;

export type Tema = ReturnType<typeof tema>;

/** Todos los colores y medidas de la web de un negocio con un estilo. */
export function tema(e: Estilo, primario?: string | null, acento?: string | null) {
	const pri = esHex(primario) ? primario.toLowerCase() : '#7a3b5e';
	const acc = esHex(acento) ? acento.toLowerCase() : '#f0a98c';
	const on = textoSobre(pri);
	const blanco = on === '#ffffff';
	// El color de marca como texto sobre claro: se oscurece hasta llegar a 4,5:1.
	let texto = pri;
	for (let t = 0; contraste(texto, '#ffffff') < 4.5 && t < 1; ) {
		t += 0.05;
		texto = mezclar(pri, '#000000', t);
	}
	const noche = e.fondo === 'noche';
	const f = FONDO[e.fondo];
	return {
		...f,
		noche,
		pri,
		acc,
		on,
		onA: blanco ? 'rgba(255,255,255,.18)' : 'rgba(20,24,26,.1)',
		priA: rgba(pri, 0.55),
		tw: TIT[e.titulos][0],
		tl: TIT[e.titulos][1],
		br: FORMA[e.forma][0],
		cr: FORMA[e.forma][1],
		sr: FORMA[e.forma][2],
		conFoto: e.portada === 'foto',
		// En Noche el color de marca se aclara 60 % hacia blanco.
		marca: noche ? mezclar(pri, '#ffffff', 0.6) : texto,
		ok: noche ? '#3ccc90' : '#17805a',
		tint: noche ? mezclar(pri, '#14221f', 0.7) : mezclar(pri, '#fffdf9', 0.88),
		raya: noche ? rgba(pri, 0.3) : rgba(pri, 0.14),
		troquel: noche ? mezclar(pri, '#14221f', 0.4) : mezclar(pri, '#fffdf9', 0.7),
		// Estrellas y el acento sobre claro: el acento tal cual decora; como texto, oscurecido.
		accSuave: mezclar(acc, noche ? '#14221f' : '#fffdf9', 0.6)
	};
}

/** Las variables CSS del tema, para un `style=`. */
export function variables(t: Tema) {
	const v: Record<string, string> = {
		bg: t.bg, papel: t.papel, campo: t.campo, chip: t.chip, socios: t.socios, 'socios-txt': t.sociosTxt, ink: t.ink, sub: t.sub,
		sub2: t.sub2, line: t.line, line2: t.line2, pri: t.pri, acc: t.acc, on: t.on, 'on-a': t.onA, 'pri-a': t.priA, tw: t.tw, tl: t.tl,
		br: t.br, cr: t.cr, sr: t.sr, marca: t.marca, ok: t.ok, tint: t.tint, raya: t.raya, troquel: t.troquel, 'acc-suave': t.accSuave,
		'foto-op': t.conFoto ? '1' : '0'
	};
	return Object.entries(v)
		.map(([k, x]) => `--w-${k}:${x}`)
		.join(';');
}
