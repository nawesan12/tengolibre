-- +goose Up
-- Fundación: el negocio activo vive en una variable de la transacción, y RLS la usa en cada tabla.
-- Los roles (tengolibre_app, tengolibre_migrador) los crea `tengolibrectl bootstrap`: son del
-- clúster y una migración no debería tener permiso para eso.

CREATE SCHEMA IF NOT EXISTS app;
GRANT USAGE ON SCHEMA app TO tengolibre_app;

-- +goose StatementBegin
-- El negocio de la transacción en curso. NULL si no se fijó: entonces RLS no deja ver nada.
CREATE FUNCTION app.negocio_actual() RETURNS uuid
LANGUAGE sql STABLE AS $$
	SELECT nullif(current_setting('app.negocio_id', true), '')::uuid
$$;
-- +goose StatementEnd

-- +goose StatementBegin
-- Aplica el aislamiento por negocio a una tabla: RLS habilitada y FORZADA (alcanza también al
-- dueño) y la política que compara negocio_id. Toda tabla con negocio_id pasa por acá.
CREATE FUNCTION app.aplicar_rls(tabla regclass) RETURNS void
LANGUAGE plpgsql AS $$
BEGIN
	EXECUTE format('ALTER TABLE %s ENABLE ROW LEVEL SECURITY', tabla);
	EXECUTE format('ALTER TABLE %s FORCE ROW LEVEL SECURITY', tabla);
	EXECUTE format(
		'CREATE POLICY negocio ON %s TO tengolibre_app, tengolibre_migrador USING (negocio_id = app.negocio_actual()) WITH CHECK (negocio_id = app.negocio_actual())',
		tabla);
END
$$;
-- +goose StatementEnd

-- Negocios (Business en Jiku). La política compara el id: un negocio sólo se ve a sí mismo.
CREATE TABLE negocios (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	slug text NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9]([a-z0-9-]{0,46}[a-z0-9])?$'),
	nombre text NOT NULL CHECK (length(nombre) BETWEEN 1 AND 80),
	zona_horaria text NOT NULL DEFAULT 'America/Argentina/Buenos_Aires',
	creado_en timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE negocios ENABLE ROW LEVEL SECURITY;
ALTER TABLE negocios FORCE ROW LEVEL SECURITY;
CREATE POLICY negocio ON negocios TO tengolibre_app, tengolibre_migrador
	USING (id = app.negocio_actual()) WITH CHECK (id = app.negocio_actual());
GRANT SELECT, UPDATE (nombre, zona_horaria) ON negocios TO tengolibre_app;

-- +goose Down
DROP TABLE negocios;
DROP FUNCTION app.aplicar_rls(regclass);
DROP FUNCTION app.negocio_actual();
DROP SCHEMA app;
