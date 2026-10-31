-- +goose Up
-- La plataforma alrededor del negocio: la web (colores, textos, portada, redes, mapa y galería),
-- los archivos subidos, el dominio propio, las sucursales (grupos), el directorio público y la
-- administración de TengoLibre.
--
-- Lo que cruza negocios (directorio, dominio por host, admin) va por funciones SECURITY DEFINER:
-- corren como el migrador, que con la RLS forzada no ve nada salvo por la política `plataforma`,
-- que sólo vale mientras la función tiene puesto app.plataforma (la fija al entrar y la borra al
-- salir, como app.slug en app.negocio_publico). Cada función devuelve sólo columnas públicas o
-- agregados.

ALTER TABLE negocios
	-- Rubro del directorio (barberia, peluqueria…): la lista vive en Go (miweb.Rubros).
	ADD COLUMN rubro text NOT NULL DEFAULT '' CHECK (rubro = '' OR (rubro ~ '^[a-z]+(-[a-z]+)*$' AND length(rubro) <= 40)),
	ADD COLUMN provincia text NOT NULL DEFAULT '' CHECK (length(provincia) <= 80),
	-- El texto largo de la web (about); descripcion es la bajada de una línea.
	ADD COLUMN acerca text NOT NULL DEFAULT '' CHECK (length(acerca) <= 5000),
	ADD COLUMN portada text CHECK (length(portada) <= 2048),
	ADD COLUMN color_primario text NOT NULL DEFAULT '#6366f1' CHECK (color_primario ~ '^#[0-9a-f]{6}$'),
	ADD COLUMN color_acento text NOT NULL DEFAULT '#22d3ee' CHECK (color_acento ~ '^#[0-9a-f]{6}$'),
	ADD COLUMN email text NOT NULL DEFAULT '' CHECK (length(email) <= 120),
	ADD COLUMN sitio_web text NOT NULL DEFAULT '' CHECK (length(sitio_web) <= 200),
	ADD COLUMN facebook text NOT NULL DEFAULT '' CHECK (length(facebook) <= 100),
	ADD COLUMN tiktok text NOT NULL DEFAULT '' CHECK (length(tiktok) <= 60),
	-- Coordenadas del mapa: las dos o ninguna.
	ADD COLUMN latitud double precision CHECK (latitud BETWEEN -90 AND 90),
	ADD COLUMN longitud double precision CHECK (longitud BETWEEN -180 AND 180),
	ADD CONSTRAINT negocios_coordenadas CHECK ((latitud IS NULL) = (longitud IS NULL)),
	ADD COLUMN grupo_id uuid;
-- El fin de la prueba gratis (prueba_hasta) lo agrega 00004_cobros.
GRANT UPDATE (rubro, provincia, acerca, portada, color_primario, color_acento, email, sitio_web, facebook, tiktok,
	latitud, longitud, grupo_id) ON negocios TO tengolibre_app;

-- Grupos de sucursales (BusinessGroup): de una persona, que es dueña de cada negocio del grupo.
-- No tiene negocio_id: se aísla por la persona (app.clerk_id, que la API fija con la sesión).
CREATE TABLE grupos (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	duenio text NOT NULL CHECK (duenio ~ '^user_[A-Za-z0-9]{10,64}$'),
	nombre text NOT NULL CHECK (length(nombre) BETWEEN 1 AND 80),
	creado_en timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX grupos_duenio ON grupos (duenio);
ALTER TABLE grupos ENABLE ROW LEVEL SECURITY;
ALTER TABLE grupos FORCE ROW LEVEL SECURITY;
CREATE POLICY persona ON grupos TO tengolibre_app, tengolibre_migrador
	USING (duenio = current_setting('app.clerk_id', true)) WITH CHECK (duenio = current_setting('app.clerk_id', true));
GRANT SELECT, INSERT, DELETE ON grupos TO tengolibre_app;
GRANT UPDATE (nombre) ON grupos TO tengolibre_app;
ALTER TABLE negocios ADD CONSTRAINT negocios_grupo FOREIGN KEY (grupo_id) REFERENCES grupos (id) ON DELETE SET NULL;
CREATE INDEX negocios_grupo ON negocios (grupo_id) WHERE grupo_id IS NOT NULL;

-- Galería de la web (BusinessPhoto), ordenable.
CREATE TABLE fotos (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	url text NOT NULL CHECK (length(url) BETWEEN 1 AND 2048),
	epigrafe text NOT NULL DEFAULT '' CHECK (length(epigrafe) <= 200),
	orden int NOT NULL DEFAULT 0,
	creado_en timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX fotos_negocio ON fotos (negocio_id, orden);
SELECT app.aplicar_rls('fotos');
GRANT SELECT, INSERT, DELETE ON fotos TO tengolibre_app;
GRANT UPDATE (epigrafe, orden) ON fotos TO tengolibre_app;

-- Imágenes subidas al disco del VPS (<carpeta>/<negocio>/<nombre>). El nombre es aleatorio.
CREATE TABLE archivos (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	nombre text NOT NULL UNIQUE CHECK (nombre ~ '^[0-9a-f]{32}\.(jpg|png|webp)$'),
	tipo text NOT NULL CHECK (tipo IN ('image/jpeg', 'image/png', 'image/webp')),
	bytes int NOT NULL CHECK (bytes > 0),
	ancho int NOT NULL CHECK (ancho > 0),
	alto int NOT NULL CHECK (alto > 0),
	creado_en timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX archivos_negocio ON archivos (negocio_id, creado_en);
SELECT app.aplicar_rls('archivos');
GRANT SELECT, INSERT, DELETE ON archivos TO tengolibre_app;

-- Dominio propio (customDomain*): uno por negocio. La verificación es por DNS: un TXT con el token
-- prueba que es suyo, y el CNAME (o el A) que apunta al VPS. Pendientes puede haber varios con el
-- mismo nombre (nadie traba el dominio de otro anotándolo); verificado, uno solo.
CREATE TABLE dominios (
	negocio_id uuid PRIMARY KEY REFERENCES negocios (id) ON DELETE CASCADE,
	nombre text NOT NULL CHECK (length(nombre) <= 253 AND nombre ~ '^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$'),
	token text NOT NULL CHECK (token ~ '^[a-z0-9]{24,64}$'),
	estado text NOT NULL DEFAULT 'pendiente' CHECK (estado IN ('pendiente', 'verificado', 'error')),
	-- Qué faltó en el último chequeo, para mostrárselo a la dueña.
	detalle text NOT NULL DEFAULT '' CHECK (length(detalle) <= 500),
	creado_en timestamptz NOT NULL DEFAULT now(),
	chequeado_en timestamptz,
	verificado_en timestamptz
);
CREATE UNIQUE INDEX dominios_verificado ON dominios (nombre) WHERE estado = 'verificado';
SELECT app.aplicar_rls('dominios');
GRANT SELECT, INSERT, DELETE ON dominios TO tengolibre_app;
GRANT UPDATE (estado, detalle, chequeado_en, verificado_en) ON dominios TO tengolibre_app;

-- La vista de plataforma: el migrador (dueño de las funciones de abajo) lee todo sólo mientras una
-- de ellas tiene puesto app.plataforma.
-- +goose StatementBegin
DO $$
DECLARE
	t text;
BEGIN
	FOREACH t IN ARRAY ARRAY['negocios', 'dominios', 'servicios', 'categorias', 'profesionales', 'profesional_servicios',
		'horarios', 'turnos', 'miembros'] LOOP
		EXECUTE format('CREATE POLICY plataforma ON %I FOR SELECT TO tengolibre_migrador USING (current_setting(''app.plataforma'', true) = ''si'')', t);
	END LOOP;
END
$$;
-- +goose StatementEnd

-- +goose StatementBegin
-- Si un negocio se puede reservar: un servicio activo que hace un profesional activo con horario.
-- Sin SECURITY DEFINER: corre con los permisos de quien la llama (las funciones de abajo).
CREATE FUNCTION app.es_reservable(n uuid) RETURNS boolean
LANGUAGE sql STABLE SET search_path = public, app AS $$
	SELECT EXISTS (
		SELECT 1 FROM servicios s
		JOIN profesional_servicios ps ON ps.servicio_id = s.id
		JOIN profesionales p ON p.id = ps.profesional_id
		WHERE s.negocio_id = n AND s.activo AND p.activo
			AND EXISTS (SELECT 1 FROM horarios h WHERE h.profesional_id = p.id))
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.es_reservable(uuid) FROM PUBLIC;

-- +goose StatementBegin
-- La página por dominio propio: del host al slug, sólo si está verificado y el negocio activo.
CREATE FUNCTION app.negocio_por_host(p_host text) RETURNS text
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	s text;
BEGIN
	PERFORM set_config('app.plataforma', 'si', true);
	SELECT n.slug INTO s FROM dominios d JOIN negocios n ON n.id = d.negocio_id
		WHERE d.nombre = p_host AND d.estado = 'verificado' AND n.activo;
	PERFORM set_config('app.plataforma', '', true);
	RETURN s;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.negocio_por_host(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.negocio_por_host(text) TO tengolibre_app;

-- +goose StatementBegin
-- El directorio (explorar): negocios activos y reservables, por ciudad, provincia, rubro, categoría
-- de servicio y texto (nombre, descripción o servicio). Los filtros vacíos no filtran. strpos y no
-- ILIKE: lo que escribe el visitante no se interpreta como patrón.
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

-- +goose StatementBegin
-- Las ciudades con algo para mostrar en el directorio, las más pobladas primero.
CREATE FUNCTION app.directorio_ciudades() RETURNS TABLE (ciudad text, provincia text, cantidad bigint)
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
REVOKE ALL ON FUNCTION app.directorio_ciudades() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.directorio_ciudades() TO tengolibre_app;

-- +goose StatementBegin
-- Admin: los números de la plataforma. p_mes y p_anterior son los comienzos del mes en curso y del
-- anterior (los calcula la API en la hora de Argentina).
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

-- +goose StatementBegin
-- Admin: los negocios, buscando por nombre, slug o email de la dueña; los más nuevos primero.
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

-- +goose StatementBegin
-- Admin: activar o desactivar un negocio y mover el fin de su prueba (NULL: no se toca). Escribe con
-- el negocio fijado (la política `negocio` del migrador) y deja como estaba app.negocio_id.
-- Devuelve false si el negocio no existe.
CREATE FUNCTION app.admin_cambiar_negocio(p_id uuid, p_activo boolean, p_prueba_hasta timestamptz) RETURNS boolean
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	antes text := current_setting('app.negocio_id', true);
	hecho boolean;
BEGIN
	PERFORM set_config('app.negocio_id', p_id::text, true);
	UPDATE negocios SET activo = coalesce(p_activo, activo), prueba_hasta = coalesce(p_prueba_hasta, prueba_hasta)
		WHERE id = p_id;
	hecho := FOUND;
	PERFORM set_config('app.negocio_id', coalesce(antes, ''), true);
	RETURN hecho;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.admin_cambiar_negocio(uuid, boolean, timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.admin_cambiar_negocio(uuid, boolean, timestamptz) TO tengolibre_app;

-- +goose Down
DROP FUNCTION app.admin_cambiar_negocio(uuid, boolean, timestamptz);
DROP FUNCTION app.admin_negocios(text, int, int);
DROP FUNCTION app.admin_estadisticas(timestamptz, timestamptz);
DROP FUNCTION app.directorio_ciudades();
DROP FUNCTION app.directorio(text, text, text, text, text, int, int);
DROP FUNCTION app.negocio_por_host(text);
DROP FUNCTION app.es_reservable(uuid);
-- +goose StatementBegin
DO $$
DECLARE
	t text;
BEGIN
	FOREACH t IN ARRAY ARRAY['negocios', 'servicios', 'categorias', 'profesionales', 'profesional_servicios',
		'horarios', 'turnos', 'miembros'] LOOP
		EXECUTE format('DROP POLICY plataforma ON %I', t);
	END LOOP;
END
$$;
-- +goose StatementEnd
DROP TABLE dominios;
DROP TABLE archivos;
DROP TABLE fotos;
ALTER TABLE negocios DROP CONSTRAINT negocios_grupo;
DROP TABLE grupos;
ALTER TABLE negocios DROP CONSTRAINT negocios_coordenadas, DROP COLUMN rubro, DROP COLUMN provincia, DROP COLUMN acerca,
	DROP COLUMN portada, DROP COLUMN color_primario, DROP COLUMN color_acento, DROP COLUMN email, DROP COLUMN sitio_web,
	DROP COLUMN facebook, DROP COLUMN tiktok, DROP COLUMN latitud, DROP COLUMN longitud, DROP COLUMN grupo_id;
