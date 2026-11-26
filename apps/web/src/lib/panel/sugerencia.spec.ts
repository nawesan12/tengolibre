import { afterEach, describe, expect, it, vi } from 'vitest';
import { cerrarConReintento } from './sugerencia.ts';

const cuerpo = { id: 'tl-tlw', negocio: '00000000-0000-0000-0000-000000000000', hasta: 'siempre' as const };
const resp = (status: number) => new Response(null, { status });

afterEach(() => vi.unstubAllGlobals());

describe('cerrarConReintento', () => {
	it('guarda al primer intento', async () => {
		const f = vi.fn().mockResolvedValue(resp(204));
		vi.stubGlobal('fetch', f);
		expect(await cerrarConReintento(cuerpo, 0)).toBe(true);
		expect(f).toHaveBeenCalledTimes(1);
		expect(JSON.parse(f.mock.calls[0][1].body)).toEqual({ ...cuerpo, que: 'cerrar' });
	});

	it('reintenta una vez si se cae la red o la API', async () => {
		const f = vi.fn().mockRejectedValueOnce(new TypeError('sin red')).mockResolvedValueOnce(resp(204));
		vi.stubGlobal('fetch', f);
		expect(await cerrarConReintento(cuerpo, 0)).toBe(true);
		expect(f).toHaveBeenCalledTimes(2);
	});

	it('si vuelve a fallar, avisa que no se guardó', async () => {
		const f = vi.fn().mockResolvedValue(resp(502));
		vi.stubGlobal('fetch', f);
		expect(await cerrarConReintento(cuerpo, 0)).toBe(false);
		expect(f).toHaveBeenCalledTimes(2);
	});

	it('un 4xx no se reintenta', async () => {
		const f = vi.fn().mockResolvedValue(resp(403));
		vi.stubGlobal('fetch', f);
		expect(await cerrarConReintento(cuerpo, 0)).toBe(false);
		expect(f).toHaveBeenCalledTimes(1);
	});
});
