import { json } from '@sveltejs/kit';
import { apiDe, ErrorApi } from '#lib/server/api.ts';
import type { RespuestaAsistente, TurnoChat } from '#lib/panel/asistente.ts';
import type { RequestHandler } from './$types';

type Pedido =
	| { accion: 'preguntar'; pregunta: string; historial?: TurnoChat[] }
	| { accion: 'confirmar' | 'descartar'; propuestaId: string };

/**
 * El asistente de Inicio habla con la API por acá (el navegador nunca habla directo con la API):
 * preguntar, y confirmar o descartar una propuesta. Los errores vuelven como { error } con el estado
 * de la API, para mostrarlos en la conversación.
 */
export const POST: RequestHandler = async (event) => {
	const p = (await event.request.json().catch(() => null)) as Pedido | null;
	const api = apiDe(event);
	try {
		if (p?.accion === 'preguntar') {
			const pregunta = String(p.pregunta ?? '').trim().slice(0, 500);
			if (!pregunta) return json({ error: 'Escribí la pregunta.' }, { status: 400 });
			const historial = (Array.isArray(p.historial) ? p.historial : [])
				.slice(-8)
				.map((t) => ({ rol: t.rol === 'asistente' ? 'asistente' : 'usuario', texto: String(t.texto ?? '').slice(0, 600) }));
			return json(await api.post<RespuestaAsistente>('/v1/panel/asistente', { pregunta, historial }));
		}
		if ((p?.accion === 'confirmar' || p?.accion === 'descartar') && p.propuestaId)
			return json(await api.post<{ respuesta: string }>(`/v1/panel/asistente/${p.accion}`, { propuestaId: String(p.propuestaId) }));
		return json({ error: 'Pedido inválido.' }, { status: 400 });
	} catch (e) {
		if (e instanceof ErrorApi) return json({ error: e.message, codigo: e.codigo }, { status: e.estado });
		return json({ error: 'No pudimos hablar con el asistente. Probá de nuevo en un rato.' }, { status: 502 });
	}
};
