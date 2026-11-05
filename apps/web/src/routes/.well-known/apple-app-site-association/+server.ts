import type { RequestHandler } from './$types';

/**
 * Universal Links de la app de iPhone (ar.com.tengolibre.app, equipo M6935MSNQB): el link del turno
 * que recibe la clienta (/turno/<código>), los turnos del panel y las invitaciones al equipo se abren
 * en la app si está instalada; si no, siguen en la web.
 */
export const GET: RequestHandler = () =>
	new Response(
		JSON.stringify({
			applinks: {
				details: [
					{
						appIDs: ['M6935MSNQB.ar.com.tengolibre.app'],
						components: [
							{ '/': '/turno/*', comment: 'El turno de la clienta' },
							{ '/': '/invitacion/*', comment: 'Invitación al equipo' },
							{ '/': '/panel/turnos/*', comment: 'Un turno del panel' }
						]
					}
				]
			},
			webcredentials: { apps: ['M6935MSNQB.ar.com.tengolibre.app'] }
		}),
		{ headers: { 'content-type': 'application/json', 'cache-control': 'public, max-age=3600' } }
	);
