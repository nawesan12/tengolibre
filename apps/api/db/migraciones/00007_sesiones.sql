-- +goose Up
-- Sesiones propias de la API. El token de sesión de Clerk dura un minuto: la web (y la app) lo
-- canjean una vez en POST /v1/auth/clerk por una sesión de TengoLibre, y después mandan esa.
-- Viven en el esquema auth (no son datos de un negocio). Se guarda sólo el hash del token.
CREATE SCHEMA IF NOT EXISTS auth;
GRANT USAGE ON SCHEMA auth TO tengolibre_app;

CREATE TABLE auth.sesiones (
	hash bytea PRIMARY KEY CHECK (length(hash) = 32),
	clerk_id text NOT NULL CHECK (clerk_id ~ '^user_[A-Za-z0-9]{10,64}$'),
	email text NOT NULL CHECK (length(email) <= 120),
	nombre text NOT NULL DEFAULT '' CHECK (length(nombre) <= 120),
	foto text NOT NULL DEFAULT '' CHECK (length(foto) <= 2048),
	creada_en timestamptz NOT NULL DEFAULT now(),
	usada_en timestamptz NOT NULL DEFAULT now(),
	vence_en timestamptz NOT NULL
);
CREATE INDEX sesiones_clerk ON auth.sesiones (clerk_id);
CREATE INDEX sesiones_vence ON auth.sesiones (vence_en);
GRANT SELECT, INSERT, DELETE ON auth.sesiones TO tengolibre_app;
GRANT UPDATE (usada_en) ON auth.sesiones TO tengolibre_app;

-- +goose Down
DROP TABLE auth.sesiones;
DROP SCHEMA auth;
