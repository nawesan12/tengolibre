import { estadoCampo, formatoValido, RESERVADOS, type Disponibilidad } from './slug.ts';

/**
 * Chequea al instante si una dirección está libre (debounce de 300 ms) contra /empezar/direccion, que
 * le pregunta a la API. Lo usan el cierre de la landing, el alta y el paso del nombre.
 */
export class Direccion {
	slug = $state('');
	resultado = $state<Disponibilidad | null>(null);
	#t: ReturnType<typeof setTimeout> | undefined;
	#pedido = 0;

	vacio = $state<string | undefined>(undefined);
	/** El rubro elegido, para que las alternativas suenen a ese rubro. */
	rubro = '';
	campo = $derived(estadoCampo(this.slug, this.resultado, this.vacio));

	constructor(vacio?: string) {
		this.vacio = vacio;
	}

	/** Cambia la dirección a chequear. */
	poner(slug: string) {
		if (slug === this.slug) return;
		this.slug = slug;
		this.resultado = null;
		clearTimeout(this.#t);
		if (slug.length < 3) return;
		if (!formatoValido(slug) || RESERVADOS.has(slug)) {
			this.resultado = { estado: 'invalida', sugerencias: [] };
			return;
		}
		const n = ++this.#pedido;
		this.#t = setTimeout(async () => {
			try {
				const r = await fetch(`/empezar/direccion?slug=${encodeURIComponent(slug)}&rubro=${encodeURIComponent(this.rubro)}`);
				const d = (await r.json()) as Disponibilidad;
				if (n === this.#pedido) this.resultado = d;
			} catch {
				// Sin conexión: se deja «buscando»; al crear, la API igual lo valida.
			}
		}, 300);
	}
}
