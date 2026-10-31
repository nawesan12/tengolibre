/**
 * El estado del onboarding mientras se completa un paso: arranca de lo que ya está guardado en la API
 * (o de lo típico del rubro) y lo leen el paso, la vista previa del teléfono y la barra de abajo.
 */
import { Direccion } from './direccion.svelte.ts';
import { aSlug } from './slug.ts';
import { leerHorario } from './agenda.ts';
import { RUBROS, rubroDe, rubroDeApi, type Rubro } from './rubros.ts';
import { COLORES, ESTILO_BASE, type Cargado, type Estilo } from './tipos.ts';

/** `nuevo`: lo sumó la persona en este paso (se le escribe el nombre y la duración). */
export type ServicioEditable = { id?: string; nombre: string; duracion: number; precio: number; activo: boolean; nuevo?: boolean };
export type Persona = { id?: string; nombre: string };
export type ModoCobro = 'sin_cobro' | 'porcentaje' | 'fija' | 'total';

export const ENTRES = [0, 5, 10, 15];
export const ANTICIPACIONES = [
	[30, '30 min'],
	[120, '2 h'],
	[1440, '1 día']
] as const;

type Datos = {
	borrador: { nombre: string; slug: string; rubro: string };
	cargado: Cargado | null;
	persona?: string;
};

export class Borrador {
	rubro = $state('');
	nombre = $state('');
	/** La dirección escrita a mano (si no, se arma sola con el nombre). */
	slugManual = $state<string | null>(null);
	direccion = new Direccion('Se arma sola con el nombre.');

	servicios = $state<ServicioEditable[]>([]);
	personas = $state<Persona[]>([]);
	quitar = $state<string[]>([]);
	dias = $state<number[]>([1, 2, 3, 4, 5, 6]);
	abre = $state('10:00');
	cierra = $state('19:00');

	descOn = $state(true);
	descDesde = $state('13:00');
	descHasta = $state('14:00');
	entre = $state(10);
	antic = $state(120);
	feriado = $state<'cierro' | 'abro' | 'especial'>('cierro');
	/** Los feriados que se apartan de la regla elegida (se tocan uno por uno). */
	ferExcepciones = $state<string[]>([]);
	ferAbre = $state('10:00');
	ferCierra = $state('14:00');

	cobro = $state<ModoCobro>('porcentaje');
	senaFija = $state(5000);

	color = $state<string>(COLORES[0][0]);
	estilo = $state<Estilo>({ ...ESTILO_BASE });

	slug = $derived(aSlug(this.slugManual ?? this.nombre));
	rubroInfo: Rubro | undefined = $derived(rubroDe(this.rubro));
	activos = $derived(this.servicios.filter((s) => s.activo));

	constructor(d: Datos) {
		const c = d.cargado;
		this.rubro = c ? (rubroDeApi(c.web.rubro)?.clave ?? d.borrador.rubro) : d.borrador.rubro;
		this.nombre = c?.negocio.nombre ?? d.borrador.nombre;
		if (!c && d.borrador.slug && d.borrador.slug !== aSlug(d.borrador.nombre)) this.slugManual = d.borrador.slug;
		this.direccion.rubro = this.rubro;
		const r = rubroDe(this.rubro) ?? RUBROS[0];

		if (c?.servicios.length)
			this.servicios = c.servicios.map((s) => ({ id: s.id, nombre: s.nombre, duracion: s.duracion, precio: s.precio, activo: s.activo }));
		else this.servicios = r.servicios.map((s) => ({ ...s, activo: true }));

		if (c?.profesionales.length) {
			this.personas = c.profesionales.map((p) => ({ id: p.id, nombre: p.nombre }));
			const h = leerHorario(c.profesionales[0].horario);
			if (h) {
				this.dias = h.dias;
				this.abre = h.abre;
				this.cierra = h.cierra;
				if (h.descanso) {
					this.descDesde = h.descanso.desde;
					this.descHasta = h.descanso.hasta;
				}
			}
		} else this.personas = [{ nombre: (d.persona ?? '').trim() }];

		if (c) {
			if (ENTRES.includes(c.ajustes.margenMin)) this.entre = c.ajustes.margenMin;
			if (ANTICIPACIONES.some(([m]) => m === c.ajustes.anticipacionMin)) this.antic = c.ajustes.anticipacionMin;
			// Un negocio nuevo trae «sin cobro» con el 50 % de la base: se propone la seña recomendada.
			// Lo elegido en este paso se guarda con 30 %, así al volver se ve lo que se eligió.
			const elegido = c.cobro.modo !== 'sin_cobro' || c.cobro.senaPorcentaje !== 50;
			this.cobro = elegido ? c.cobro.modo : 'porcentaje';
			if (c.cobro.senaFija > 0) this.senaFija = c.cobro.senaFija;
			const propio = COLORES.some(([v]) => v === c.web.colorPrimario.toLowerCase());
			this.color = propio ? c.web.colorPrimario.toLowerCase() : r.color;
			this.estilo = { ...ESTILO_BASE, ...c.web.estilo };
		} else this.color = r.color;
	}

	/** El nombre que va en la vista previa. */
	get visible() {
		return this.nombre.trim() || 'Tu negocio';
	}
	get inicial() {
		return (this.nombre.trim() || 'T').charAt(0).toUpperCase();
	}
}
