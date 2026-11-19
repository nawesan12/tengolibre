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

/**
 * El acento de cada color principal (estrellas, bloque de acento). Antes el alta mandaba sólo el
 * principal y todo negocio quedaba con el cian de la base, ajeno a las seis paletas. Los mismos que la
 * migración 00035 les puso a los negocios que ya estaban.
 */
export const ACENTOS: Record<string, string> = {
	'#7a3b5e': '#e9a6b8',
	'#183c44': '#f0a98c',
	'#1f2a24': '#d9b26a',
	'#b4532a': '#f2c38b',
	'#3d5a40': '#e3b48e',
	'#1d3557': '#f4b183'
};

/** Los colores por defecto (los del handoff), para quien todavía no eligió. */
export const PRIMARIO_POR_DEFECTO = '#183c44';
export const ACENTO_POR_DEFECTO = '#f0a98c';

/** El acento que va con un color principal (el del handoff si no es de la paleta). */
export const acentoDe = (primario: string) => ACENTOS[primario.toLowerCase()] ?? ACENTO_POR_DEFECTO;

export const ESTILO_BASE: Estilo = { plantilla: 'carta', portada: 'foto', fondo: 'crema', titulos: 'grandes', forma: 'redondos' };
