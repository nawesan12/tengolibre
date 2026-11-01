/**
 * El estilo de la web pública (plantilla y ajustes), como lo guarda la API en
 * `GET/PATCH /v1/panel/mi-web` → `estilo`. El PATCH reemplaza el publicado entero.
 *
 * `secciones`: qué secciones se muestran y en qué orden (miweb.Secciones en la API). Se manda sólo
 * si el GET lo devuelve (así no rompe con una API vieja, que no conoce el campo).
 */
export const PLANTILLAS = ['carta', 'vidriera', 'sobria', 'galeria', 'editorial', 'bloques'] as const;
export const PORTADAS = ['foto', 'color'] as const;
export const FONDOS = ['crema', 'blanco', 'noche'] as const;
export const TITULOS = ['grandes', 'medianos', 'livianos'] as const;
export const FORMAS = ['redondos', 'suaves', 'rectos'] as const;
export const SECCIONES = ['servicios', 'resenas', 'equipo', 'fotos', 'membresias'] as const;

export type Plantilla = (typeof PLANTILLAS)[number];
export type Seccion = (typeof SECCIONES)[number];
export type Estilo = {
	plantilla: Plantilla;
	portada: (typeof PORTADAS)[number];
	fondo: (typeof FONDOS)[number];
	titulos: (typeof TITULOS)[number];
	forma: (typeof FORMAS)[number];
	/** Las secciones visibles, en orden. */
	secciones?: Seccion[];
};
export type Ajuste = 'portada' | 'fondo' | 'titulos' | 'forma';

export const NOMBRES: Record<string, string> = {
	carta: 'Carta',
	vidriera: 'Vidriera',
	sobria: 'Sobria',
	galeria: 'Galería',
	editorial: 'Editorial',
	bloques: 'Bloques',
	foto: 'Con foto',
	color: 'Sólo color',
	crema: 'Crema',
	blanco: 'Blanco',
	noche: 'Noche',
	grandes: 'Grandes',
	medianos: 'Medianos',
	livianos: 'Livianos',
	redondos: 'Redondos',
	suaves: 'Suaves',
	rectos: 'Rectos',
	servicios: 'Servicios',
	resenas: 'Reseñas',
	equipo: 'Equipo',
	fotos: 'Fotos',
	membresias: 'Membresías'
};

export const DESCRIPCIONES: Record<Plantilla, string> = {
	carta: 'La portada con tu color y los servicios como una carta. Ideal si tenés muchos servicios.',
	vidriera: 'Una foto grande de punta a punta y servicios con foto. Para lugares que se lucen.',
	sobria: 'Sin fotos: tu color, tu logo y tipografía grande. Para consultorios y quien no tiene fotos.',
	galeria: 'Mosaico de fotos arriba y filtros por categoría. Para uñas, peinados y tatuajes.',
	editorial: 'El nombre en grande como tapa de revista y una foto chica. Para estudios con personalidad.',
	bloques: 'Todo en una grilla de bloques: nombre, fotos, reseñas y precios a la vista de un golpe.'
};

const RESUMEN: Record<keyof Estilo, string> = {
	plantilla: 'Plantilla',
	portada: 'Portada',
	fondo: 'Fondo',
	titulos: 'Títulos',
	forma: 'Botones',
	secciones: 'Secciones'
};

const uno = <T extends readonly string[]>(v: FormDataEntryValue | null, lista: T, def: T[number]): T[number] =>
	lista.includes(String(v)) ? (String(v) as T[number]) : def;

/** Del formulario de «Publicar» al `estilo` que pide la API. */
export function estiloDeForm(f: FormData): Estilo {
	const e: Estilo = {
		plantilla: uno(f.get('plantilla'), PLANTILLAS, 'carta'),
		portada: uno(f.get('portada'), PORTADAS, 'foto'),
		fondo: uno(f.get('fondo'), FONDOS, 'crema'),
		titulos: uno(f.get('titulos'), TITULOS, 'grandes'),
		forma: uno(f.get('forma'), FORMAS, 'redondos')
	};
	const s = f.get('secciones');
	if (s !== null) e.secciones = String(s).split(',').filter((x): x is Seccion => (SECCIONES as readonly string[]).includes(x));
	return e;
}

/** Qué cambió entre lo publicado y el borrador («Plantilla: vidriera · Fondo: noche»). */
export function diferencias(pub: Estilo, bor: Estilo): string[] {
	const claves: (keyof Estilo)[] = ['plantilla', 'portada', 'fondo', 'titulos', 'forma'];
	const out = claves.filter((k) => pub[k] !== bor[k]).map((k) => `${RESUMEN[k]}: ${NOMBRES[bor[k] as string].toLowerCase()}`);
	if (bor.secciones && (pub.secciones ?? []).join() !== bor.secciones.join()) out.push(`${RESUMEN.secciones}: ${bor.secciones.map((s) => NOMBRES[s].toLowerCase()).join(', ') || 'ninguna'}`);
	return out;
}

/**
 * La vista previa: la página pública real del negocio con el borrador en la query. La página pública
 * lee estos parámetros sólo con `vista=1` (validados contra estas listas) y no guarda nada.
 */
export function urlVistaPrevia(slug: string, e: Estilo) {
	const q = new URLSearchParams({ vista: '1', plantilla: e.plantilla, portada: e.portada, fondo: e.fondo, titulos: e.titulos, forma: e.forma });
	if (e.secciones) q.set('secciones', e.secciones.join(','));
	return `/${slug}?${q}`;
}

/** Luminancia relativa de un color #rrggbb (WCAG). */
function luminancia(hex: string) {
	const m = /^#?([0-9a-f]{6})$/i.exec(hex.trim());
	if (!m) return 0;
	const n = parseInt(m[1], 16);
	const c = [n >> 16, (n >> 8) & 255, n & 255].map((v) => {
		const s = v / 255;
		return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
	});
	return 0.2126 * c[0] + 0.7152 * c[1] + 0.0722 * c[2];
}

/** Contraste entre dos colores (1 a 21). */
export function contraste(a: string, b: string) {
	const [x, y] = [luminancia(a), luminancia(b)].sort((p, q) => q - p);
	return (x + 0.05) / (y + 0.05);
}

/** El texto que va sobre un color: blanco o tinta, el que más contraste dé. */
export const textoSobre = (fondo: string) => (contraste(fondo, '#ffffff') >= contraste(fondo, '#0f2328') ? '#ffffff' : '#0f2328');
