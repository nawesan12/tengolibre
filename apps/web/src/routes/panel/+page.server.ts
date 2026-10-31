import { apiDe, ErrorApi } from '#lib/server/api.ts';
import type { PageServerLoad } from './$types';

type Paso = { clave: string; titulo: string; listo: boolean; obligatorio: boolean };
type Proximo = { id: string; inicio: string; estado: string; servicio: string; profesional: string; cliente: string };
export type Tablero = {
	turnosHoy: number;
	turnosAyer: number;
	ingresosMes: number;
	ingresosMesAnterior: number;
	clientesActivos: number;
	ocupacionHoy: number | null;
	proximos: Proximo[];
};

export const load: PageServerLoad = async (event) => {
	event.depends('panel:turnos', 'panel:pagos');
	const api = apiDe(event);
	const [arranque, tablero] = await Promise.all([
		api.get<{ completo: boolean; reservable: boolean; pasos: Paso[] }>('/v1/panel/arranque'),
		// El tablero es de quien gestiona; los demás ven sólo el arranque y la agenda.
		api.get<Tablero>('/v1/panel/estadisticas').catch((e) => {
			if (e instanceof ErrorApi && e.estado === 403) return null;
			throw e;
		})
	]);
	return { arranque, tablero };
};
