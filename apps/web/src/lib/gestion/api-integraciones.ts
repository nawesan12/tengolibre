/**
 * Integraciones e IA. El asistente de IA usa el servidor MCP real (docs/api/mcp.md):
 *
 *   GET    /v1/panel/mcp                 → Mcp (url, claves vigentes y el catálogo de permisos)
 *   POST   /v1/panel/mcp/tokens          { nombre, permisos? } → { token, clave, url }  (la clave sólo viene acá)
 *   PATCH  /v1/panel/mcp/tokens/{id}     { nombre?, permisos? } → Token  (permisos = lista entera)
 *   DELETE /v1/panel/mcp/tokens/{id}     → 204 (revoca)
 *   GET    /v1/panel/mcp/registro        ?token&soloCambios=1&limite → { acciones: AccionMcp[] }
 *
 * «La IA de tus clientes»: el servidor MCP público (sin clave) donde el asistente de cualquier
 * persona encuentra el negocio y reserva (docs/api/mcp.md, «La IA de tus clientes»):
 *
 *   GET /v1/panel/ia/publico → IaPublica
 *   PUT /v1/panel/ia/publico   { activo: boolean } → 204
 */
export const RUTAS_MCP = {
	leer: '/v1/panel/mcp',
	tokens: '/v1/panel/mcp/tokens',
	registro: '/v1/panel/mcp/registro'
} as const;
export const RUTA_IA_PUBLICO = '/v1/panel/ia/publico';

export type Token = {
	id: string;
	nombre: string;
	/** Los últimos 4 caracteres de la clave. */
	pista: string;
	permisos: string[];
	/** El clientInfo del último initialize: vacío si nunca se conectó. */
	cliente: string;
	creadoEn: string;
	usadoEn: string | null;
};
export type PermisoMcp = { clave: string; nombre: string; descripcion: string; herramientas: string[] };
export type Mcp = { url: string; tokens: Token[]; permisos: PermisoMcp[] };
export type AccionMcp = {
	id: string;
	tokenId: string;
	token: string;
	cliente: string;
	herramienta: string;
	titulo: string;
	ok: boolean;
	escribe: boolean;
	/** Frase lista para ir después del nombre de la clave. */
	resumen: string;
	error: string;
	creadoEn: string;
};
/** Una reserva que hizo el asistente de una persona: `resumen` va después de «Asistente de una clienta». */
export type UltimaIa = { turnoId: string; resumen: string; creadoEn: string };
export type IaPublica = { activo: boolean; reservasMes: number; ultimas: UltimaIa[]; url: string };

/** Los asistentes del diseño: cada uno es una clave con ese nombre. */
export const ASISTENTES = ['Claude', 'ChatGPT', 'Gemini', 'Otro (MCP)'] as const;

/** El ícono de cada permiso (el texto viene de la API). */
export const ICONO_PERMISO: Record<string, string> = {
	ver_agenda: 'view_agenda',
	mover_turnos: 'edit_calendar',
	cancelar_turnos: 'cancel',
	ver_clientes: 'group',
	ver_numeros: 'payments'
};
