/** Lo que devuelven las rutas públicas de la API (sin sesión). */

export type Negocio = {
	id: string;
	slug: string;
	nombre: string;
	descripcion: string;
	direccion: string;
	ciudad: string;
	telefono: string;
	whatsapp: string;
	instagram: string;
	logo: string | null;
	zonaHoraria: string;
};

export type Categoria = { id: string; nombre: string; orden: number };

export type Servicio = {
	id: string;
	categoriaId: string | null;
	nombre: string;
	descripcion: string;
	duracion: number;
	precio: number;
	foto: string | null;
	orden: number;
	activo: boolean;
};

export type Profesional = {
	id: string;
	nombre: string;
	especialidad: string;
	bio: string;
	foto: string | null;
	servicios: string[];
};

export type Web = {
	rubro: string;
	provincia: string;
	acerca: string;
	portada: string | null;
	colorPrimario: string;
	colorAcento: string;
	email: string;
	sitioWeb: string;
	facebook: string;
	tiktok: string;
	latitud: number | null;
	longitud: number | null;
};

export type Foto = { id: string; url: string; epigrafe: string; orden: number };

export type Resumen = { promedio: number; cantidad: number };

export type Pagina = {
	negocio: Negocio;
	categorias: Categoria[];
	servicios: Servicio[];
	profesionales: Profesional[];
	diasVista: number;
	politicaCancelacion: string;
	web: Web;
	galeria: Foto[];
	resenas: Resumen;
};

export type ResenaPublica = {
	nombre: string;
	servicio: string;
	puntaje: number;
	comentario: string;
	respuesta: string;
	creadoEn: string;
};

export type Plan = {
	id: string;
	nombre: string;
	descripcion: string;
	precio: number;
	visitas: number;
	duracionDias: number;
	beneficios: string[];
};

export type Cobro = { modo: 'sin_cobro' | 'porcentaje' | 'fija' | 'total'; senaPorcentaje: number; senaFija: number };

export type Dia = { fecha: string; libres: number };
export type Horario = { hora: string; inicio: string; profesionales: string[] };

export type Pago = { monto: number; modo: string; url?: string };

export type Turno = {
	id: string;
	codigo: string;
	servicioId: string | null;
	servicio: string;
	profesionalId: string | null;
	profesional: string;
	cliente: { nombre: string; telefono: string; email: string };
	inicio: string;
	fin: string;
	precio: number;
	estado: string;
	nota: string;
	venceEn: string | null;
	descuento?: number;
	pago?: Pago;
};

export type TurnoCliente = Turno & { negocio: Negocio; puedeCancelar: boolean; cancelarHastaHoras: number };

export type Cupon = { id: string; codigo: string; tipo: 'porcentaje' | 'fijo'; valor: number; hasta: string | null; activo: boolean; usos: number; maxUsos: number | null };
export type MiReferido = { codigo: string | null; referidos: number; recompensas: Cupon[]; programaActivo: boolean };

export type Sugerencia = {
	dia: number;
	hora: string;
	profesionalId: string;
	profesional: string;
	confianza: number;
	motivo: string;
	proxima: string;
};

export type Validacion = { valido: boolean; motivo?: string; codigo?: string; tipo?: string; valor?: number; descuento?: number };

export type Ficha = {
	slug: string;
	nombre: string;
	descripcion: string;
	logo: string | null;
	portada: string | null;
	ciudad: string;
	provincia: string;
	rubro: string;
	servicios: string[];
	desde: number | null;
};

export type Directorio = { negocios: Ficha[]; total: number; pagina: number; limite: number; paginas: number };
export type Ciudad = { ciudad: string; provincia: string; cantidad: number };
