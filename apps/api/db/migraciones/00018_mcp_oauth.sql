-- +goose Up
-- OAuth 2.1 para el servidor MCP (spec de autorización de MCP): los conectores de claude.ai y
-- ChatGPT no dejan cargar un encabezado con la clave, así que se registran solos (RFC 7591), mandan
-- a la persona a dar permiso (la pantalla /conectar del front) y canjean un código con PKCE por
-- tokens. Una conexión OAuth es una fila más de mcp_tokens (mismos permisos, mismo registro), con
-- su cliente y tokens que vencen y rotan.

-- Clientes registrados (públicos: sin secreto). Globales, como las sesiones.
CREATE TABLE auth.oauth_clientes (
	id text PRIMARY KEY CHECK (id ~ '^tlcli_[A-Za-z0-9_-]{16,64}$'),
	nombre text NOT NULL CHECK (length(nombre) BETWEEN 1 AND 60),
	redirect_uris text[] NOT NULL CHECK (cardinality(redirect_uris) BETWEEN 1 AND 10),
	creado_en timestamptz NOT NULL DEFAULT now(),
	-- El primer canje de un código: un cliente que nunca se usó se borra a la semana.
	usado_en timestamptz
);
CREATE INDEX oauth_clientes_creado ON auth.oauth_clientes (creado_en);
GRANT SELECT, INSERT, DELETE ON auth.oauth_clientes TO tengolibre_app;
GRANT UPDATE (usado_en) ON auth.oauth_clientes TO tengolibre_app;

-- Pedidos de autorización: nacen en /v1/oauth/authorize, la persona los decide en /conectar y el
-- código que sale se canjea una sola vez. Se guardan sólo hashes (del pedido y del código).
CREATE TABLE auth.oauth_autorizaciones (
	hash bytea PRIMARY KEY CHECK (length(hash) = 32),
	cliente_id text NOT NULL REFERENCES auth.oauth_clientes (id) ON DELETE CASCADE,
	redirect_uri text NOT NULL CHECK (length(redirect_uri) <= 500),
	state text NOT NULL DEFAULT '' CHECK (length(state) <= 500),
	-- code_challenge (S256).
	desafio text NOT NULL CHECK (desafio ~ '^[A-Za-z0-9._~-]{43,128}$'),
	recurso text NOT NULL CHECK (length(recurso) <= 500),
	creada_en timestamptz NOT NULL DEFAULT now(),
	vence_en timestamptz NOT NULL,
	-- La decisión: quién, para qué negocio y con qué permisos.
	decidida_en timestamptz,
	clerk_id text CHECK (length(clerk_id) <= 80),
	negocio_id uuid REFERENCES negocios (id) ON DELETE CASCADE,
	permisos text[]
		CHECK (permisos <@ ARRAY['ver_agenda', 'mover_turnos', 'cancelar_turnos', 'ver_clientes', 'ver_numeros']),
	codigo_hash bytea UNIQUE CHECK (length(codigo_hash) = 32),
	codigo_vence timestamptz,
	canjeada_en timestamptz,
	-- La conexión (mcp_tokens) que salió del canje: si el código se reusa, se revoca.
	token_id uuid
);
CREATE INDEX oauth_autorizaciones_vence ON auth.oauth_autorizaciones (vence_en);
GRANT SELECT, INSERT, DELETE ON auth.oauth_autorizaciones TO tengolibre_app;
GRANT UPDATE (decidida_en, clerk_id, negocio_id, permisos, codigo_hash, codigo_vence, canjeada_en, token_id)
	ON auth.oauth_autorizaciones TO tengolibre_app;

-- La conexión OAuth en mcp_tokens: hash es el del token de acceso vigente (vence), y el de
-- refresco rota en cada uso. Las claves del panel quedan como estaban (oauth_cliente nulo).
ALTER TABLE mcp_tokens
	ADD COLUMN oauth_cliente text REFERENCES auth.oauth_clientes (id),
	ADD COLUMN acceso_vence timestamptz,
	ADD COLUMN refresco_hash bytea UNIQUE CHECK (length(refresco_hash) = 32),
	ADD COLUMN refresco_vence timestamptz;
GRANT UPDATE (hash, acceso_vence, refresco_hash, refresco_vence) ON mcp_tokens TO tengolibre_app;

CREATE POLICY por_refresco ON mcp_tokens FOR SELECT TO tengolibre_migrador
	USING (refresco_hash = decode(coalesce(nullif(current_setting('app.mcp_refresco', true), ''), '00'), 'hex'));

-- +goose StatementBegin
-- Un token de acceso vencido ya no resuelve (las claves del panel no vencen).
CREATE OR REPLACE FUNCTION app.negocio_del_mcp(p_hash text) RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	n uuid;
BEGIN
	IF p_hash !~ '^[0-9a-f]{64}$' THEN
		RETURN NULL;
	END IF;
	PERFORM set_config('app.mcp_hash', p_hash, true);
	SELECT negocio_id INTO n FROM mcp_tokens WHERE hash = decode(p_hash, 'hex') AND revocado_en IS NULL
		AND (acceso_vence IS NULL OR acceso_vence > now());
	PERFORM set_config('app.mcp_hash', '', true);
	RETURN n;
END
$$;
-- +goose StatementEnd

-- +goose StatementBegin
-- Del token de refresco (su hash, en hex) al negocio, antes de fijarlo.
CREATE FUNCTION app.negocio_del_refresco(p_hash text) RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	n uuid;
BEGIN
	IF p_hash !~ '^[0-9a-f]{64}$' THEN
		RETURN NULL;
	END IF;
	PERFORM set_config('app.mcp_refresco', p_hash, true);
	SELECT negocio_id INTO n FROM mcp_tokens WHERE refresco_hash = decode(p_hash, 'hex') AND revocado_en IS NULL
		AND refresco_vence > now();
	PERFORM set_config('app.mcp_refresco', '', true);
	RETURN n;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.negocio_del_refresco(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.negocio_del_refresco(text) TO tengolibre_app;

-- +goose Down
DROP FUNCTION app.negocio_del_refresco(text);
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.negocio_del_mcp(p_hash text) RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	n uuid;
BEGIN
	IF p_hash !~ '^[0-9a-f]{64}$' THEN
		RETURN NULL;
	END IF;
	PERFORM set_config('app.mcp_hash', p_hash, true);
	SELECT negocio_id INTO n FROM mcp_tokens WHERE hash = decode(p_hash, 'hex') AND revocado_en IS NULL;
	PERFORM set_config('app.mcp_hash', '', true);
	RETURN n;
END
$$;
-- +goose StatementEnd
DROP POLICY por_refresco ON mcp_tokens;
ALTER TABLE mcp_tokens DROP COLUMN refresco_vence, DROP COLUMN refresco_hash, DROP COLUMN acceso_vence, DROP COLUMN oauth_cliente;
DROP TABLE auth.oauth_autorizaciones;
DROP TABLE auth.oauth_clientes;
