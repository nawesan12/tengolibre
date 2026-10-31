/** El reporte mensual: GET /v1/panel/reportes/mensual?mes=AAAA-MM (docs/api/reporte-mensual.md). */
import type { Mapa } from './MapaCalor.svelte';

export type Indicador = {
	clave: string;
	titulo: string;
	valor: number | null;
	anterior: number | null;
	valorTexto: string;
	texto: string;
	tendencia: 'sube' | 'baja' | 'igual' | string;
	bueno: boolean | null;
};

export type ReporteMensual = {
	negocio: string;
	slug: string;
	mes: string;
	anterior: string;
	nombre: string;
	nombreAnterior: string;
	nombreSiguiente: string;
	desde: string;
	hasta: string;
	periodo: string;
	completo: boolean;
	titulo: string;
	bajada: string;
	indicadores: Indicador[];
	agenda: {
		ocupacion: number | null;
		titulo: string;
		mapa: Mapa;
		horasLibres: number;
		franjaLibre: { dias: number[]; horaDesde: number; horaHasta: number; texto: string; detalle: string; momento: string } | null;
		huecosLlenados: number;
		sugerencia: { titulo: string; texto: string } | null;
	};
	plata: {
		titulo: string;
		ingresos: number;
		senas: number;
		turnosConSena: number;
		membresias: number;
		sociosActivos: number;
		ticketPromedio: number;
		ticketAnterior: number;
		ticketTexto: string;
		semanas: { etiqueta: string; ingresos: number }[];
		servicios: { nombre: string; turnos: number; facturado: number; parte: number }[];
		nota: string;
	};
	equipo: { id: string; nombre: string; ocupacion: number | null; turnos: number; resenas: number; puntaje: number | null }[];
	clientes: { nuevos: number; recurrentes: number; volvieron: number; nuevas: number; porReferido: number; texto: string };
	resenas: {
		cantidad: number;
		promedio: number | null;
		destacada: { comentario: string; nombre: string; profesional: string; puntaje: number } | null;
	};
	destacados: { titulo: string; texto: string }[];
	siguiente: { numero: string; titulo: string; texto: string }[];
	pie: string;
};

/** «2026-10» ± n meses. */
export function otroMes(mes: string, n: number) {
	const [a, m] = mes.split('-').map(Number);
	const d = new Date(Date.UTC(a, m - 1 + n, 1));
	return `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, '0')}`;
}

/** Los días de una semana del reporte a partir de su etiqueta («1–7 oct», «29–31»). */
export function diasDeSemana(etiqueta: string) {
	const m = /(\d+)\s*[–-]\s*(\d+)/.exec(etiqueta);
	return m ? Number(m[2]) - Number(m[1]) + 1 : 7;
}
