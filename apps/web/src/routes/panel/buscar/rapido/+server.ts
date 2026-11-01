import { json } from '@sveltejs/kit';
import { apiDe } from '#lib/server/api.ts';
import type { Encontrado } from '#lib/operacion/tipos.ts';
import type { RequestHandler } from './$types';

/** La búsqueda mientras se escribe (el buscador de la barra): lo mismo que /panel/buscar, en JSON. */
export const GET: RequestHandler = async (event) => {
	const q = (event.url.searchParams.get('q') ?? '').trim().slice(0, 100);
	const vacio: Encontrado = { clientes: [], turnos: [], servicios: [] };
	if (q.length < 2) return json(vacio);
	const r = await apiDe(event)
		.get<Encontrado>(`/v1/panel/buscar?${new URLSearchParams({ q })}`)
		.catch(() => vacio);
	return json({ clientes: r.clientes ?? [], turnos: r.turnos ?? [], servicios: r.servicios ?? [] });
};
