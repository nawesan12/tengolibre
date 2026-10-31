/** Las formas de la API para el admin de la plataforma y las sucursales. */

export type Estadisticas = {
	negocios: number;
	activos: number;
	enPrueba: number;
	negociosDelMes: number;
	personas: number;
	turnosDelMes: number;
	turnosMesAnterior: number;
	ingresosDelMes: number;
	dominiosVerificados: number;
	recientes: { id: string; slug: string; nombre: string; creadoEn: string }[];
};

export type NegocioAdmin = {
	id: string;
	slug: string;
	nombre: string;
	ciudad: string;
	activo: boolean;
	pruebaHasta: string | null;
	creadoEn: string;
	duenia: string | null;
	miembros: number;
	turnos30Dias: number;
	reservable: boolean;
};

export type ListadoAdmin = { total: number; negocios: NegocioAdmin[]; pagina: number; limite: number };

export type Sucursal = { id: string; slug: string; nombre: string; ciudad: string };
export type Grupo = { id: string; nombre: string; sucursales: Sucursal[] };
export type Grupos = { grupos: Grupo[]; sinGrupo: Sucursal[] };

export type Numeros = {
	turnos: number;
	atendidos: number;
	ausentes: number;
	cancelados: number;
	clientes: number;
	ingresos: number;
};
export type Reporte = {
	grupo: string;
	desde: string;
	hasta: string;
	total: Numeros;
	sucursales: (Sucursal & Numeros)[];
};
