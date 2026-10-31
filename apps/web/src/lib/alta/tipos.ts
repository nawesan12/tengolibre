/** Lo que el onboarding lee de la API (los mismos endpoints que el panel). */
import type { Franja } from '#lib/configuracion/horario.ts';

export type Estilo = { plantilla: string; portada: string; fondo: string; titulos: string; forma: string };
export type NegocioAlta = { id: string; slug: string; nombre: string };
export type WebAlta = { rubro: string; colorPrimario: string; colorAcento: string; estilo: Estilo };
export type ServicioAlta = { id: string; nombre: string; duracion: number; precio: number; activo: boolean };
export type ProfesionalAlta = { id: string; nombre: string; activo: boolean; servicios: string[]; horario: Franja[] };
export type CobroAlta = { modo: 'sin_cobro' | 'porcentaje' | 'fija' | 'total'; senaPorcentaje: number; senaFija: number };
export type Ajustes = {
	intervalo: number;
	anticipacionMin: number;
	diasVista: number;
	margenMin: number;
	cancelarHastaHoras: number;
	politicaCancelacion: string;
};
export type BloqueoAlta = { id: string; fecha: string; desde: string | null; hasta: string | null; motivo?: string };

/** Todo lo cargado del negocio en el onboarding. */
export type Cargado = {
	negocio: NegocioAlta;
	web: WebAlta;
	servicios: ServicioAlta[];
	profesionales: ProfesionalAlta[];
	cobro: CobroAlta;
	ajustes: Ajustes;
	bloqueos: BloqueoAlta[];
	mercadoPago: boolean;
};

/** Los ajustes del estilo (miweb.Estilo en la API). */
export const PLANTILLAS = [
	['carta', 'Carta'],
	['vidriera', 'Vidriera'],
	['sobria', 'Sobria'],
	['galeria', 'Galería'],
	['editorial', 'Editorial'],
	['bloques', 'Bloques']
] as const;

export const COLORES = [
	['#7a3b5e', 'Ciruela'],
	['#183c44', 'Petróleo'],
	['#1f2a24', 'Carbón'],
	['#b4532a', 'Terracota'],
	['#3d5a40', 'Salvia'],
	['#1d3557', 'Marino']
] as const;

export const ESTILO_BASE: Estilo = { plantilla: 'carta', portada: 'foto', fondo: 'crema', titulos: 'grandes', forma: 'redondos' };
