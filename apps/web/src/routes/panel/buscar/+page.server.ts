import { apiDe } from '#lib/server/api.ts';
import { param } from '#lib/operacion/servidor.ts';
import type { Encontrado } from '#lib/operacion/tipos.ts';
import type { PageServerLoad } from './$types';

/** La búsqueda global de la barra: clientes, turnos y servicios. */
export const load: PageServerLoad = async (event) => {
	event.depends('panel:clientes', 'panel:turnos');
	const q = param(event.url, 'q').slice(0, 100);
	const vacio: Encontrado = { clientes: [], turnos: [], servicios: [] };
	if (q.length < 2) return { q, ...vacio };
	const r = await apiDe(event).get<Encontrado>(`/v1/panel/buscar?${new URLSearchParams({ q })}`);
	return { q, clientes: r.clientes ?? [], turnos: r.turnos ?? [], servicios: r.servicios ?? [] };
};
