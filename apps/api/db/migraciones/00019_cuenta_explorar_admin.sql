-- +goose Up
-- Lo que el front rediseñado de cuenta, explorar y admin espera (apps/web/src/lib/cuenta/contratos.ts):
--  · Preferencias del cliente por tipo de mail: además de `email` (ahora sólo los cambios en sus
--    turnos) y `recordatorios`, el pedido de reseña, la lista de espera y las novedades (cumpleaños,
--    «hace mucho que no venís»). La confirmación de un turno nuevo sale siempre: es el comprobante.
--  · Explorar: el puntaje de cada ficha, su zona y el filtro «hoy con lugar» (el próximo horario
--    libre lo calcula la API con el motor de disponibilidad).
--  · Admin: el pulso de hoy y la plata de las suscripciones, y el estado de la cuenta de cada negocio.

-- Las llaves nuevas arrancan como estaba `email`: quien había apagado los mails no empieza a
-- recibir reseñas ni novedades. La migración es dueña de la tabla: sin FORCE ve todas las filas.
ALTER TABLE preferencias
	ADD COLUMN resena boolean NOT NULL DEFAULT true,
	ADD COLUMN lista_espera boolean NOT NULL DEFAULT true,
	ADD COLUMN novedades boolean NOT NULL DEFAULT true;
ALTER TABLE preferencias NO FORCE ROW LEVEL SECURITY;
UPDATE preferencias SET resena = email, lista_espera = email, novedades = email WHERE NOT email;
ALTER TABLE preferencias FORCE ROW LEVEL SECURITY;
GRANT UPDATE (resena, lista_espera, novedades) ON preferencias TO tengolibre_app;

-- La vista de plataforma (ver 00006) suma las reseñas (el puntaje del directorio) y las
-- suscripciones (el estado de cada cuenta y la plata, para el admin).
CREATE POLICY plataforma ON resenas FOR SELECT TO tengolibre_migrador USING (current_setting('app.plataforma', true) = 'si');
CREATE POLICY plataforma ON suscripciones FOR SELECT TO tengolibre_migrador USING (current_setting('app.plataforma', true) = 'si');

-- +goose StatementBegin
-- El estado de la cuenta de un negocio, como lo muestra el admin: con una suscripción autorizada, su
-- plan; pausada por un cobro que no pasó y en gracia, cobro_fallido (y hasta cuándo); dada de baja
-- con el mes pago, su plan hasta el próximo cobro; si no, en prueba o en sólo lectura. Es la misma
-- regla que suscripcion.LeerAcceso. Corre con los permisos de quien la llama.
CREATE FUNCTION app.estado_cuenta(p_id uuid, p_prueba_hasta timestamptz, p_ahora timestamptz,
	OUT estado text, OUT gracia_hasta timestamptz)
LANGUAGE sql STABLE SET search_path = public, app AS $$
	WITH s AS (
		SELECT s.plan, s.estado, s.gracia_hasta, s.cancelada_en, s.creado_en FROM suscripciones s
		WHERE s.negocio_id = p_id AND (s.estado = 'autorizada'
			OR (s.estado = 'pausada' AND (s.gracia_hasta IS NULL OR s.gracia_hasta > p_ahora)))
	), autorizada AS (
		SELECT plan FROM s WHERE estado = 'autorizada' ORDER BY creado_en DESC LIMIT 1
	), fallida AS (
		SELECT gracia_hasta FROM s WHERE estado = 'pausada' AND cancelada_en IS NULL ORDER BY creado_en DESC LIMIT 1
	), baja AS (
		SELECT plan FROM s WHERE estado = 'pausada' AND cancelada_en IS NOT NULL ORDER BY creado_en DESC LIMIT 1
	)
	SELECT CASE
			WHEN EXISTS (SELECT 1 FROM autorizada) THEN (SELECT plan FROM autorizada)
			WHEN EXISTS (SELECT 1 FROM fallida) THEN 'cobro_fallido'
			WHEN EXISTS (SELECT 1 FROM baja) THEN (SELECT plan FROM baja)
			WHEN p_prueba_hasta > p_ahora THEN 'prueba'
			ELSE 'solo_lectura'
		END,
		CASE WHEN NOT EXISTS (SELECT 1 FROM autorizada) THEN (SELECT gracia_hasta FROM fallida) END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.estado_cuenta(uuid, timestamptz, timestamptz) FROM PUBLIC;

DROP FUNCTION app.directorio(text, text, text, text, text, int, int);
-- +goose StatementBegin
-- El directorio (explorar), como en 00006, más el puntaje (las reseñas visibles), la zona y el id
-- (la API lo usa para calcular el próximo horario libre; no sale en la respuesta). p_hoy: sólo los
-- que atienden hoy (en su zona): algún profesional activo, que hace un servicio activo, tiene
-- horario ese día de la semana. Si además queda lugar lo decide la API con el motor.
CREATE FUNCTION app.directorio(p_ciudad text, p_provincia text, p_rubro text, p_categoria text, p_texto text,
	p_hoy boolean, p_ahora timestamptz, p_limite int, p_desde int) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	out jsonb;
BEGIN
	PERFORM set_config('app.plataforma', 'si', true);
	WITH elegidos AS (
		SELECT n.* FROM negocios n
		WHERE n.activo AND app.es_reservable(n.id)
			AND (p_ciudad = '' OR strpos(lower(n.ciudad), lower(p_ciudad)) > 0)
			AND (p_provincia = '' OR strpos(lower(n.provincia), lower(p_provincia)) > 0)
			AND (p_rubro = '' OR n.rubro = p_rubro)
			AND (p_categoria = '' OR EXISTS (SELECT 1 FROM servicios s JOIN categorias c ON c.id = s.categoria_id
				WHERE s.negocio_id = n.id AND s.activo AND strpos(lower(c.nombre), lower(p_categoria)) > 0))
			AND (p_texto = '' OR strpos(lower(n.nombre), lower(p_texto)) > 0 OR strpos(lower(n.descripcion), lower(p_texto)) > 0
				OR EXISTS (SELECT 1 FROM servicios s WHERE s.negocio_id = n.id AND s.activo AND strpos(lower(s.nombre), lower(p_texto)) > 0))
			AND (NOT p_hoy OR EXISTS (
				SELECT 1 FROM servicios s
				JOIN profesional_servicios ps ON ps.servicio_id = s.id
				JOIN profesionales p ON p.id = ps.profesional_id
				JOIN horarios h ON h.profesional_id = p.id
				WHERE s.negocio_id = n.id AND s.activo AND p.activo
					AND h.dia = extract(dow FROM p_ahora AT TIME ZONE n.zona_horaria)::int))
	), pagina AS (
		SELECT * FROM elegidos ORDER BY nombre, id LIMIT p_limite OFFSET p_desde
	)
	SELECT jsonb_build_object(
		'total', (SELECT count(*) FROM elegidos),
		'negocios', coalesce((SELECT jsonb_agg(jsonb_build_object(
			'id', p.id, 'slug', p.slug, 'nombre', p.nombre, 'descripcion', p.descripcion, 'logo', p.logo, 'portada', p.portada,
			'ciudad', p.ciudad, 'provincia', p.provincia, 'rubro', p.rubro, 'zonaHoraria', p.zona_horaria,
			'servicios', to_jsonb(ARRAY(SELECT s.nombre FROM servicios s WHERE s.negocio_id = p.id AND s.activo ORDER BY s.orden, s.nombre LIMIT 3)),
			'desde', (SELECT min(s.precio) FROM servicios s WHERE s.negocio_id = p.id AND s.activo),
			'promedio', (SELECT round(avg(r.puntaje), 1)::float8 FROM resenas r WHERE r.negocio_id = p.id AND r.visible),
			'resenas', (SELECT count(*) FROM resenas r WHERE r.negocio_id = p.id AND r.visible)
		) ORDER BY p.nombre, p.id) FROM pagina p), '[]'::jsonb)
	) INTO out;
	PERFORM set_config('app.plataforma', '', true);
	RETURN out;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.directorio(text, text, text, text, text, boolean, timestamptz, int, int) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.directorio(text, text, text, text, text, boolean, timestamptz, int, int) TO tengolibre_app;

DROP FUNCTION app.admin_estadisticas(timestamptz, timestamptz);
-- +goose StatementBegin
-- Admin: los números de 00006 más el pulso de hoy y la plata. p_hoy y p_manana son el comienzo de
-- hoy y de mañana en la hora de Argentina; p_ahora, el reloj de la API.
CREATE FUNCTION app.admin_estadisticas(p_mes timestamptz, p_anterior timestamptz, p_ahora timestamptz,
	p_hoy timestamptz, p_manana timestamptz) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	out jsonb;
	hoy bigint;
	web bigint;
	terminadas bigint;
	convertidas bigint;
BEGIN
	PERFORM set_config('app.plataforma', 'si', true);
	SELECT count(*), count(*) FILTER (WHERE origen = 'web') INTO hoy, web
		FROM turnos WHERE inicio >= p_hoy AND inicio < p_manana AND estado <> 'cancelado';
	-- Conversión: de las pruebas que terminaron en los últimos 90 días, cuántas llegaron a pagar.
	SELECT count(*), count(*) FILTER (WHERE EXISTS (SELECT 1 FROM suscripciones s WHERE s.negocio_id = n.id
			AND (s.estado IN ('autorizada', 'pausada') OR s.ultimo_cobro IS NOT NULL)))
		INTO terminadas, convertidas
		FROM negocios n WHERE n.prueba_hasta > p_ahora - interval '90 days' AND n.prueba_hasta <= p_ahora;
	SELECT jsonb_build_object(
		'negocios', (SELECT count(*) FROM negocios),
		'activos', (SELECT count(*) FROM negocios WHERE activo),
		'enPrueba', (SELECT count(*) FROM negocios WHERE activo AND prueba_hasta > p_ahora),
		'negociosDelMes', (SELECT count(*) FROM negocios WHERE creado_en >= p_mes),
		'personas', (SELECT count(DISTINCT clerk_id) FROM miembros WHERE activo),
		'turnosDelMes', (SELECT count(*) FROM turnos WHERE inicio >= p_mes AND estado <> 'cancelado'),
		'turnosMesAnterior', (SELECT count(*) FROM turnos WHERE inicio >= p_anterior AND inicio < p_mes AND estado <> 'cancelado'),
		'ingresosDelMes', (SELECT coalesce(sum(precio), 0) FROM turnos WHERE inicio >= p_mes AND estado = 'atendido'),
		'dominiosVerificados', (SELECT count(*) FROM dominios WHERE estado = 'verificado'),
		'recientes', coalesce((SELECT jsonb_agg(jsonb_build_object('id', r.id, 'slug', r.slug, 'nombre', r.nombre, 'creadoEn', r.creado_en)
			ORDER BY r.creado_en DESC) FROM (SELECT * FROM negocios ORDER BY creado_en DESC LIMIT 5) r), '[]'::jsonb),
		'turnosHoy', hoy,
		'porcentajeWeb', CASE WHEN hoy > 0 THEN round(100.0 * web / hoy)::int END,
		'suscripcionesPagas', (SELECT count(DISTINCT negocio_id) FROM suscripciones WHERE estado = 'autorizada'),
		'mrr', (SELECT coalesce(sum(monto), 0) FROM suscripciones WHERE estado = 'autorizada'),
		'pruebasPorVencer', (SELECT count(*) FROM negocios n WHERE n.activo AND n.prueba_hasta > p_ahora
			AND n.prueba_hasta <= p_ahora + interval '3 days'
			AND NOT EXISTS (SELECT 1 FROM suscripciones s WHERE s.negocio_id = n.id AND s.estado = 'autorizada')),
		'conversion', CASE WHEN terminadas > 0 THEN round(100.0 * convertidas / terminadas)::int END
	) INTO out;
	PERFORM set_config('app.plataforma', '', true);
	RETURN out;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.admin_estadisticas(timestamptz, timestamptz, timestamptz, timestamptz, timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.admin_estadisticas(timestamptz, timestamptz, timestamptz, timestamptz, timestamptz) TO tengolibre_app;

DROP FUNCTION app.admin_negocios(text, int, int);
-- +goose StatementBegin
-- Admin: los negocios como en 00006, con el rubro y el estado de la cuenta (app.estado_cuenta), los
-- días de gracia que le quedan a un cobro fallido, el filtro por estado y los conteos por estado
-- (de la búsqueda, sin el filtro de estado: son las pastillas).
CREATE FUNCTION app.admin_negocios(p_texto text, p_estado text, p_ahora timestamptz, p_limite int, p_desde int) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	out jsonb;
BEGIN
	PERFORM set_config('app.plataforma', 'si', true);
	WITH buscados AS (
		SELECT n.*, e.estado, e.gracia_hasta FROM negocios n
		CROSS JOIN LATERAL app.estado_cuenta(n.id, n.prueba_hasta, p_ahora) e
		WHERE p_texto = '' OR strpos(lower(n.nombre), lower(p_texto)) > 0 OR strpos(n.slug, lower(p_texto)) > 0
			OR EXISTS (SELECT 1 FROM miembros m WHERE m.negocio_id = n.id AND m.rol = 'duenio' AND strpos(lower(m.email), lower(p_texto)) > 0)
	), elegidos AS (
		SELECT * FROM buscados WHERE p_estado = '' OR estado = p_estado
	), pagina AS (
		SELECT * FROM elegidos ORDER BY creado_en DESC, id LIMIT p_limite OFFSET p_desde
	)
	SELECT jsonb_build_object(
		'total', (SELECT count(*) FROM elegidos),
		'conteos', (SELECT coalesce(jsonb_object_agg(c.estado, c.cantidad), '{}'::jsonb)
			FROM (SELECT estado, count(*) AS cantidad FROM buscados GROUP BY estado) c),
		'negocios', coalesce((SELECT jsonb_agg(jsonb_build_object(
			'id', p.id, 'slug', p.slug, 'nombre', p.nombre, 'ciudad', p.ciudad, 'rubro', p.rubro, 'activo', p.activo,
			'pruebaHasta', p.prueba_hasta, 'creadoEn', p.creado_en, 'estado', p.estado,
			'graciaDias', CASE WHEN p.estado = 'cobro_fallido' AND p.gracia_hasta IS NOT NULL
				THEN ceil(extract(epoch FROM p.gracia_hasta - p_ahora) / 86400)::int END,
			'duenia', (SELECT m.email FROM miembros m WHERE m.negocio_id = p.id AND m.rol = 'duenio' ORDER BY m.creado_en LIMIT 1),
			'miembros', (SELECT count(*) FROM miembros m WHERE m.negocio_id = p.id AND m.activo),
			'turnos30Dias', (SELECT count(*) FROM turnos t WHERE t.negocio_id = p.id AND t.inicio >= p_ahora - interval '30 days' AND t.estado <> 'cancelado'),
			'reservable', app.es_reservable(p.id)
		) ORDER BY p.creado_en DESC, p.id) FROM pagina p), '[]'::jsonb)
	) INTO out;
	PERFORM set_config('app.plataforma', '', true);
	RETURN out;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.admin_negocios(text, text, timestamptz, int, int) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.admin_negocios(text, text, timestamptz, int, int) TO tengolibre_app;

-- +goose Down
DROP FUNCTION app.admin_negocios(text, text, timestamptz, int, int);
-- +goose StatementBegin
CREATE FUNCTION app.admin_negocios(p_texto text, p_limite int, p_desde int) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	out jsonb;
BEGIN
	PERFORM set_config('app.plataforma', 'si', true);
	WITH elegidos AS (
		SELECT n.* FROM negocios n
		WHERE p_texto = '' OR strpos(lower(n.nombre), lower(p_texto)) > 0 OR strpos(n.slug, lower(p_texto)) > 0
			OR EXISTS (SELECT 1 FROM miembros m WHERE m.negocio_id = n.id AND m.rol = 'duenio' AND strpos(lower(m.email), lower(p_texto)) > 0)
	), pagina AS (
		SELECT * FROM elegidos ORDER BY creado_en DESC, id LIMIT p_limite OFFSET p_desde
	)
	SELECT jsonb_build_object(
		'total', (SELECT count(*) FROM elegidos),
		'negocios', coalesce((SELECT jsonb_agg(jsonb_build_object(
			'id', p.id, 'slug', p.slug, 'nombre', p.nombre, 'ciudad', p.ciudad, 'activo', p.activo,
			'pruebaHasta', p.prueba_hasta, 'creadoEn', p.creado_en,
			'duenia', (SELECT m.email FROM miembros m WHERE m.negocio_id = p.id AND m.rol = 'duenio' ORDER BY m.creado_en LIMIT 1),
			'miembros', (SELECT count(*) FROM miembros m WHERE m.negocio_id = p.id AND m.activo),
			'turnos30Dias', (SELECT count(*) FROM turnos t WHERE t.negocio_id = p.id AND t.inicio >= now() - interval '30 days' AND t.estado <> 'cancelado'),
			'reservable', app.es_reservable(p.id)
		) ORDER BY p.creado_en DESC, p.id) FROM pagina p), '[]'::jsonb)
	) INTO out;
	PERFORM set_config('app.plataforma', '', true);
	RETURN out;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.admin_negocios(text, int, int) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.admin_negocios(text, int, int) TO tengolibre_app;

DROP FUNCTION app.admin_estadisticas(timestamptz, timestamptz, timestamptz, timestamptz, timestamptz);
-- +goose StatementBegin
CREATE FUNCTION app.admin_estadisticas(p_mes timestamptz, p_anterior timestamptz) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	out jsonb;
BEGIN
	PERFORM set_config('app.plataforma', 'si', true);
	SELECT jsonb_build_object(
		'negocios', (SELECT count(*) FROM negocios),
		'activos', (SELECT count(*) FROM negocios WHERE activo),
		'enPrueba', (SELECT count(*) FROM negocios WHERE activo AND prueba_hasta > now()),
		'negociosDelMes', (SELECT count(*) FROM negocios WHERE creado_en >= p_mes),
		'personas', (SELECT count(DISTINCT clerk_id) FROM miembros WHERE activo),
		'turnosDelMes', (SELECT count(*) FROM turnos WHERE inicio >= p_mes AND estado <> 'cancelado'),
		'turnosMesAnterior', (SELECT count(*) FROM turnos WHERE inicio >= p_anterior AND inicio < p_mes AND estado <> 'cancelado'),
		'ingresosDelMes', (SELECT coalesce(sum(precio), 0) FROM turnos WHERE inicio >= p_mes AND estado = 'atendido'),
		'dominiosVerificados', (SELECT count(*) FROM dominios WHERE estado = 'verificado'),
		'recientes', coalesce((SELECT jsonb_agg(jsonb_build_object('id', r.id, 'slug', r.slug, 'nombre', r.nombre, 'creadoEn', r.creado_en)
			ORDER BY r.creado_en DESC) FROM (SELECT * FROM negocios ORDER BY creado_en DESC LIMIT 5) r), '[]'::jsonb)
	) INTO out;
	PERFORM set_config('app.plataforma', '', true);
	RETURN out;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.admin_estadisticas(timestamptz, timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.admin_estadisticas(timestamptz, timestamptz) TO tengolibre_app;

DROP FUNCTION app.directorio(text, text, text, text, text, boolean, timestamptz, int, int);
-- +goose StatementBegin
CREATE FUNCTION app.directorio(p_ciudad text, p_provincia text, p_rubro text, p_categoria text, p_texto text,
	p_limite int, p_desde int) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	out jsonb;
BEGIN
	PERFORM set_config('app.plataforma', 'si', true);
	WITH elegidos AS (
		SELECT n.* FROM negocios n
		WHERE n.activo AND app.es_reservable(n.id)
			AND (p_ciudad = '' OR strpos(lower(n.ciudad), lower(p_ciudad)) > 0)
			AND (p_provincia = '' OR strpos(lower(n.provincia), lower(p_provincia)) > 0)
			AND (p_rubro = '' OR n.rubro = p_rubro)
			AND (p_categoria = '' OR EXISTS (SELECT 1 FROM servicios s JOIN categorias c ON c.id = s.categoria_id
				WHERE s.negocio_id = n.id AND s.activo AND strpos(lower(c.nombre), lower(p_categoria)) > 0))
			AND (p_texto = '' OR strpos(lower(n.nombre), lower(p_texto)) > 0 OR strpos(lower(n.descripcion), lower(p_texto)) > 0
				OR EXISTS (SELECT 1 FROM servicios s WHERE s.negocio_id = n.id AND s.activo AND strpos(lower(s.nombre), lower(p_texto)) > 0))
	), pagina AS (
		SELECT * FROM elegidos ORDER BY nombre, id LIMIT p_limite OFFSET p_desde
	)
	SELECT jsonb_build_object(
		'total', (SELECT count(*) FROM elegidos),
		'negocios', coalesce((SELECT jsonb_agg(jsonb_build_object(
			'slug', p.slug, 'nombre', p.nombre, 'descripcion', p.descripcion, 'logo', p.logo, 'portada', p.portada,
			'ciudad', p.ciudad, 'provincia', p.provincia, 'rubro', p.rubro,
			'servicios', to_jsonb(ARRAY(SELECT s.nombre FROM servicios s WHERE s.negocio_id = p.id AND s.activo ORDER BY s.orden, s.nombre LIMIT 3)),
			'desde', (SELECT min(s.precio) FROM servicios s WHERE s.negocio_id = p.id AND s.activo)
		) ORDER BY p.nombre, p.id) FROM pagina p), '[]'::jsonb)
	) INTO out;
	PERFORM set_config('app.plataforma', '', true);
	RETURN out;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.directorio(text, text, text, text, text, int, int) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.directorio(text, text, text, text, text, int, int) TO tengolibre_app;

DROP FUNCTION app.estado_cuenta(uuid, timestamptz, timestamptz);
DROP POLICY plataforma ON suscripciones;
DROP POLICY plataforma ON resenas;
REVOKE UPDATE (resena, lista_espera, novedades) ON preferencias FROM tengolibre_app;
ALTER TABLE preferencias DROP COLUMN novedades, DROP COLUMN lista_espera, DROP COLUMN resena;
