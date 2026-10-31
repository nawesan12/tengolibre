/**
 * Rubros del alta y del onboarding (Onboarding v2 del handoff): nombre, ícono, foto, color de la
 * tarjeta del alta y los servicios típicos que se cargan solos. `api` es el rubro de Mi web
 * (miweb.Rubros en la API): el consultorio va como «salud».
 */
export type Sugerido = { nombre: string; duracion: number; precio: number };
export type Rubro = {
	clave: string;
	api: string;
	nombre: string;
	icono: string;
	color: string;
	foto?: string;
	pos?: string;
	servicios: Sugerido[];
};

const s = (nombre: string, duracion: number, precio: number): Sugerido => ({ nombre, duracion, precio });

export const RUBROS: Rubro[] = [
	{
		clave: 'estetica',
		api: 'estetica',
		nombre: 'Estética',
		icono: 'spa',
		color: '#7a3b5e',
		foto: 'recepcion.jpg',
		pos: '50% 40%',
		servicios: [s('Limpieza facial', 60, 28000), s('Perfilado de cejas', 30, 9000), s('Lifting de pestañas', 60, 22000), s('Depilación de axilas', 15, 8000)]
	},
	{
		clave: 'barberia',
		api: 'barberia',
		nombre: 'Barbería',
		icono: 'content_cut',
		color: '#1f2a24',
		foto: 'barbero-cortando.jpg',
		pos: '50% 30%',
		servicios: [s('Corte', 30, 9000), s('Corte y barba', 45, 13000), s('Barba', 20, 6000), s('Color', 60, 18000)]
	},
	{
		clave: 'peluqueria',
		api: 'peluqueria',
		nombre: 'Peluquería',
		icono: 'face_retouching_natural',
		color: '#7a3b5e',
		foto: 'peluquera-con-clienta.jpg',
		pos: '60% 40%',
		servicios: [s('Corte', 45, 14000), s('Brushing', 40, 10000), s('Color', 120, 35000), s('Mechas', 150, 48000)]
	},
	{
		clave: 'unias',
		api: 'unias',
		nombre: 'Uñas',
		icono: 'back_hand',
		color: '#b24a3b',
		foto: 'manicura.jpg',
		pos: '50% 50%',
		servicios: [s('Semipermanente', 60, 12000), s('Esculpidas', 120, 25000), s('Kapping', 75, 15000), s('Pedicuría', 60, 14000)]
	},
	{
		clave: 'masajes',
		api: 'masajes',
		nombre: 'Masajes',
		icono: 'self_improvement',
		color: '#2f5d50',
		servicios: [s('Descontracturante', 60, 24000), s('Relajante', 60, 22000), s('Piedras calientes', 75, 28000)]
	},
	{
		clave: 'consultorio',
		api: 'salud',
		nombre: 'Consultorio',
		icono: 'medical_services',
		color: '#24527a',
		servicios: [s('Primera consulta', 45, 25000), s('Consulta', 30, 20000), s('Control', 20, 15000)]
	},
	{ clave: 'otro', api: 'otro', nombre: 'Otro', icono: 'storefront', color: '#183c44', servicios: [s('Servicio', 60, 10000)] }
];

export const rubroDe = (clave: string | null | undefined) => RUBROS.find((r) => r.clave === clave);

/** El rubro del onboarding que corresponde al de Mi web (otro rubro de la API cae en «Otro»). */
export const rubroDeApi = (api: string | null | undefined) =>
	api ? (RUBROS.find((r) => r.api === api) ?? RUBROS[RUBROS.length - 1]) : undefined;

/** «45 min», «1 h», «1 h 30 min». */
export const duracion = (m: number) => (m < 60 ? `${m} min` : m % 60 ? `${Math.floor(m / 60)} h ${m % 60} min` : `${m / 60} h`);

/** «$ 8.000», con espacio duro (el número nunca queda solo en otro renglón). */
export const pesos = (n: number) => '$\u00a0' + Math.round(n).toLocaleString('es-AR');
