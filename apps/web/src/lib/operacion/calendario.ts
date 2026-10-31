/**
 * Cuentas de la grilla del calendario: desde qué hora hasta cuál se dibuja, y en qué carril va cada
 * turno cuando se pisan (un sobreturno), así se ven todos lado a lado.
 */
export type Tramo = { id: string; inicio: number; fin: number };
export type Ubicado<T extends Tramo> = T & { carril: number; carriles: number };

/** Las horas a dibujar: de 8 a 21 por defecto, más lo que haga falta para que entren todos. */
export function horasVisibles(tramos: Tramo[], base: [number, number] = [8, 21]): [number, number] {
	let [desde, hasta] = base;
	for (const t of tramos) {
		desde = Math.min(desde, Math.floor(t.inicio / 60));
		hasta = Math.max(hasta, Math.ceil(t.fin / 60));
	}
	return [Math.max(0, desde), Math.min(24, Math.max(hasta, desde + 1))];
}

/**
 * Reparte los tramos de una columna en carriles: los que se pisan entre sí (un grupo) se dividen el
 * ancho; un tramo suelto ocupa la columna entera.
 */
export function carriles<T extends Tramo>(tramos: T[]): Ubicado<T>[] {
	const orden = [...tramos].sort((a, b) => a.inicio - b.inicio || b.fin - a.fin);
	const out: Ubicado<T>[] = [];
	let grupo: Ubicado<T>[] = [];
	let finGrupo = -1;
	let libres: number[] = []; // fin de cada carril del grupo
	const cerrar = () => {
		for (const t of grupo) t.carriles = libres.length;
		out.push(...grupo);
		grupo = [];
		libres = [];
	};
	for (const t of orden) {
		if (t.inicio >= finGrupo) cerrar();
		let c = libres.findIndex((fin) => fin <= t.inicio);
		if (c === -1) {
			c = libres.length;
			libres.push(t.fin);
		} else libres[c] = t.fin;
		grupo.push({ ...t, carril: c, carriles: 1 });
		finGrupo = Math.max(finGrupo, t.fin);
	}
	cerrar();
	return out;
}
