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

const DIAS_RANGO = ['DOM', 'LUN', 'MAR', 'MIÉ', 'JUE', 'VIE', 'SÁB'];
const horaCorta = (h: string) => h.replace(/:00$/, '').replace(/^0(\d)/, '$1');

/**
 * El horario de atención en renglones cortos: días seguidos con el mismo horario se juntan
 * ("LUN–VIE 10–20", "SÁB 10–14"). La semana empieza el lunes.
 */
export function renglonesAtencion(franjas: { dia: number; desde: string; hasta: string }[]) {
	const porDia = new Map<number, string>();
	for (const f of [...franjas].sort((a, b) => a.desde.localeCompare(b.desde))) {
		const r = `${horaCorta(f.desde)}–${horaCorta(f.hasta)}`;
		porDia.set(f.dia, porDia.has(f.dia) ? `${porDia.get(f.dia)}, ${r}` : r);
	}
	const orden = [1, 2, 3, 4, 5, 6, 0];
	const out: string[] = [];
	let i = 0;
	while (i < orden.length) {
		const h = porDia.get(orden[i]);
		if (!h) {
			i++;
			continue;
		}
		let j = i;
		while (j + 1 < orden.length && porDia.get(orden[j + 1]) === h) j++;
		const dias = i === j ? DIAS_RANGO[orden[i]] : `${DIAS_RANGO[orden[i]]}–${DIAS_RANGO[orden[j]]}`;
		out.push(`${dias} ${h}`);
		i = j + 1;
	}
	return out;
}

/** "Abierto hasta las 20" o "Cerrado ahora" (con el día y la hora del negocio). */
export function estadoAtencion(franjas: { dia: number; desde: string; hasta: string }[], ahora: Date, zona: string) {
	const p = new Intl.DateTimeFormat('en-US', { timeZone: zona, weekday: 'short', hour: '2-digit', minute: '2-digit', hourCycle: 'h23' }).formatToParts(ahora);
	const dow = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'].indexOf(p.find((x) => x.type === 'weekday')?.value ?? '');
	const hm = `${p.find((x) => x.type === 'hour')?.value}:${p.find((x) => x.type === 'minute')?.value}`;
	const f = franjas.find((x) => x.dia === dow && x.desde <= hm && hm < x.hasta);
	return f ? { abierto: true, texto: `Abierto hasta las ${horaCorta(f.hasta)}` } : { abierto: false, texto: 'Cerrado ahora' };
}

/** La seña de un precio con la configuración de cobro (la misma cuenta que la API, cobros.Monto). */
export function montoSena(precio: number, c: { modo: string; senaPorcentaje: number; senaFija: number } | null) {
	if (!c || precio <= 0) return 0;
	if (c.modo === 'total') return precio;
	if (c.modo === 'porcentaje') return Math.floor((precio * c.senaPorcentaje + 50) / 100);
	if (c.modo === 'fija') return Math.min(c.senaFija, precio);
	return 0;
}

/** Iniciales para el círculo de una persona: "Carla Ruiz" → "CR". */
export const iniciales = (n: string) =>
	n
		.split(/\s+/)
		.filter(Boolean)
		.slice(0, 2)
		.map((p) => p[0]!.toUpperCase())
		.join('');

/** El link para compartir por WhatsApp un texto (sin número: elige la persona). */
export const compartirWhatsapp = (texto: string) => `https://wa.me/?text=${encodeURIComponent(texto)}`;
