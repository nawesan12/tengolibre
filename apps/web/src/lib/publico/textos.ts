/** Textos y cuentas chicas de las páginas públicas. */

export const RUBROS: Record<string, string> = {
	barberia: 'Barbería',
	peluqueria: 'Peluquería',
	estetica: 'Estética',
	unias: 'Uñas',
	'cejas-pestanias': 'Cejas y pestañas',
	maquillaje: 'Maquillaje',
	depilacion: 'Depilación',
	masajes: 'Masajes',
	spa: 'Spa',
	tatuajes: 'Tatuajes',
	salud: 'Salud',
	entrenamiento: 'Entrenamiento',
	otro: 'Otro'
};

export const rubro = (r: string) => RUBROS[r] ?? r;

/** Duración en minutos, como se lee: "45 min", "1 h", "1 h 30 min". */
export function duracion(min: number) {
	if (min < 60) return `${min} min`;
	const h = Math.floor(min / 60);
	const m = min % 60;
	return m ? `${h} h ${m} min` : `${h} h`;
}

/** Pega la última palabra con un espacio duro, así no queda sola en el último renglón. */
export const sinViuda = (t: string) => t.trim().replace(/\s+(\S+)$/, ' $1');

/** Lo que se cobra por adelantado, en palabras (o vacío si no se cobra). */
export function textoCobro(c: { modo: string; senaPorcentaje: number; senaFija: number } | null, pesos: (n: number) => string) {
	if (!c) return '';
	if (c.modo === 'porcentaje') return `Para confirmar se paga una seña del ${c.senaPorcentaje}% con Mercado Pago.`;
	if (c.modo === 'fija') return `Para confirmar se paga una seña de ${pesos(c.senaFija)} con Mercado Pago.`;
	if (c.modo === 'total') return 'El turno se paga por adelantado con Mercado Pago.';
	return '';
}

/** Link a Google Maps con las coordenadas, o con la dirección si no hay. */
export function linkMapa(lat: number | null, lng: number | null, direccion = '') {
	if (lat != null && lng != null) return `https://www.google.com/maps/search/?api=1&query=${lat},${lng}`;
	if (direccion.trim()) return `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(direccion.trim())}`;
	return '';
}

/** WhatsApp: sólo dígitos; un número argentino sin código de país se completa con 54. */
export function linkWhatsapp(n: string) {
	const d = n.replace(/\D/g, '');
	if (!d) return '';
	return `https://wa.me/${d.startsWith('54') ? d : '54' + d}`;
}

export const linkInstagram = (u: string) =>
	u ? `https://instagram.com/${u.replace(/^https?:\/\/(www\.)?instagram\.com\//, '').replace(/^@/, '')}` : '';

/** Una URL que pone el negocio: con https si no lo trae. */
export const linkExterno = (u: string) => (!u ? '' : /^https?:\/\//.test(u) ? u : `https://${u}`);

export const estrellas = (n: number) => '★'.repeat(Math.round(n)) + '☆'.repeat(5 - Math.round(n));
