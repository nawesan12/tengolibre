-- +goose Up
-- Negocios fuera del directorio (explorar): el de la cuenta de revisión de las tiendas, para que no lo
-- vean clientas reales. La página propia del negocio sigue andando.
ALTER TABLE negocios ADD COLUMN en_directorio boolean NOT NULL DEFAULT true;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.directorio(p_ciudad text, p_provincia text, p_rubro text, p_categoria text, p_texto text,
	p_hoy boolean, p_ahora timestamptz, p_limite int, p_desde int) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	out jsonb;
BEGIN
	PERFORM set_config('app.plataforma', 'si', true);
	WITH elegidos AS (
		SELECT n.* FROM negocios n
		WHERE n.activo AND app.es_reservable(n.id) AND n.en_directorio
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

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.directorio_ciudades() RETURNS TABLE (ciudad text, provincia text, cantidad bigint)
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
BEGIN
	PERFORM set_config('app.plataforma', 'si', true);
	RETURN QUERY SELECT min(btrim(n.ciudad)), min(n.provincia), count(*)
		FROM negocios n
		WHERE n.activo AND btrim(n.ciudad) <> '' AND app.es_reservable(n.id) AND n.en_directorio
		GROUP BY lower(btrim(n.ciudad))
		ORDER BY count(*) DESC, min(btrim(n.ciudad));
	PERFORM set_config('app.plataforma', '', true);
END
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.directorio(p_ciudad text, p_provincia text, p_rubro text, p_categoria text, p_texto text,
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

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.directorio_ciudades() RETURNS TABLE (ciudad text, provincia text, cantidad bigint)
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
BEGIN
	PERFORM set_config('app.plataforma', 'si', true);
	RETURN QUERY SELECT min(btrim(n.ciudad)), min(n.provincia), count(*)
		FROM negocios n
		WHERE n.activo AND btrim(n.ciudad) <> '' AND app.es_reservable(n.id)
		GROUP BY lower(btrim(n.ciudad))
		ORDER BY count(*) DESC, min(btrim(n.ciudad));
	PERFORM set_config('app.plataforma', '', true);
END
$$;
-- +goose StatementEnd
ALTER TABLE negocios DROP COLUMN en_directorio;
