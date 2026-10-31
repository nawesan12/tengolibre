/**
 * Integraciones e IA: contrato SUPUESTO con la API (todavía no existe; lo arma Nahuel). Ajustar acá
 * al integrar; la página (`/panel/integraciones`) sólo usa estos tipos y rutas.
 *
 *   GET  /v1/panel/ia                 → AsistenteIA (404 = todavía sin activar)
 *   POST /v1/panel/ia/activar         → AsistenteIA            (crea la dirección MCP del negocio)
 *   PUT  /v1/panel/ia/asistente       { asistente }            → 204
 *   PUT  /v1/panel/ia/permisos        { permisos: Permisos }   → 204
 *   PUT  /v1/panel/ia/publico         { activo: boolean }      → 204   (la IA de los clientes)
 *   POST /v1/panel/ia/clave           → { clave: string }      (se muestra una sola vez; invalida la anterior)
 *
 * Lo demás que muestra la pantalla ya existe: Mercado Pago (`GET /v1/panel/mercadopago`), los links
 * iCal (`GET /v1/panel/calendarios`) y WhatsApp/Instagram del negocio (`GET /v1/panel/negocio`).
 */
export const RUTAS_IA = {
	leer: '/v1/panel/ia',
	activar: '/v1/panel/ia/activar',
	asistente: '/v1/panel/ia/asistente',
	permisos: '/v1/panel/ia/permisos',
	publico: '/v1/panel/ia/publico',
	clave: '/v1/panel/ia/clave'
} as const;

export const ASISTENTES = ['claude', 'chatgpt', 'gemini', 'otro'] as const;
export type Asistente = (typeof ASISTENTES)[number];
export const NOMBRE_ASISTENTE: Record<Asistente, string> = { claude: 'Claude', chatgpt: 'ChatGPT', gemini: 'Gemini', otro: 'Otro (MCP)' };

export const PERMISOS = ['agenda', 'turnos', 'clientes', 'cancelar', 'plata'] as const;
export type Permiso = (typeof PERMISOS)[number];
export type Permisos = Record<Permiso, boolean>;

/** Qué puede hacer el asistente: el texto de cada interruptor. */
export const TEXTO_PERMISO: Record<Permiso, { icono: string; t: string; s: string; apagado: string }> = {
	agenda: { icono: 'view_agenda', t: 'Ver agenda y huecos', s: 'Turnos, horarios libres y lista de espera', apagado: 'Apagado' },
	turnos: { icono: 'edit_calendar', t: 'Cargar y mover turnos', s: 'Siempre te pide confirmación', apagado: 'Apagado' },
	clientes: { icono: 'group', t: 'Ver clientes', s: 'Nombre, contacto e historial', apagado: 'Apagado' },
	cancelar: { icono: 'cancel', t: 'Cancelar turnos', s: 'Siempre te pide confirmación', apagado: 'Apagado: lo hacés vos' },
	plata: { icono: 'payments', t: 'Ver plata y reportes', s: 'Cobros, señas y reportes', apagado: 'Apagado' }
};

export type AccionIA = {
	/** Instante ISO. */
	en: string;
	/** Quién la hizo: «Claude», «Asistente de una clienta»… */
	quien: string;
	/** Qué hizo, en una línea. */
	texto: string;
};

export type AsistenteIA = {
	/** Si ya se conectó un asistente al menos una vez. */
	conectado: boolean;
	asistente: Asistente;
	/** La dirección del servidor MCP del negocio, p. ej. «mcp.tengolibre.com.ar/lumi». */
	direccion: string;
	permisos: Permisos;
	/** La IA de los clientes: que sus asistentes encuentren el negocio y reserven. */
	publico: { activo: boolean; reservasMes: number };
	/** Lo último que hizo una IA (lo más nuevo primero). */
	registro: AccionIA[];
};
