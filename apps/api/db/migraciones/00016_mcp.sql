-- +goose Up
-- Servidor MCP del negocio: el asistente de IA del negocio (Claude, ChatGPT u otro cliente MCP) se
-- conecta con una clave secreta y consulta u opera la agenda. La clave se muestra una sola vez: acá
-- queda sólo su hash. Cada clave tiene sus permisos (los interruptores del panel) y cada herramienta
-- que se llama queda en el registro.
CREATE TABLE mcp_tokens (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	nombre text NOT NULL CHECK (length(nombre) BETWEEN 1 AND 60),
	-- sha256 de la clave.
	hash bytea NOT NULL UNIQUE CHECK (length(hash) = 32),
	-- Los últimos caracteres de la clave, para reconocerla en el panel.
	pista text NOT NULL CHECK (length(pista) <= 8),
	permisos text[] NOT NULL DEFAULT '{}'
		CHECK (permisos <@ ARRAY['ver_agenda', 'mover_turnos', 'cancelar_turnos', 'ver_clientes', 'ver_numeros']),
	creado_por text NOT NULL DEFAULT '' CHECK (length(creado_por) <= 80),
	-- El clientInfo del último initialize (p. ej. «claude-ai 0.1.0»).
	cliente text NOT NULL DEFAULT '' CHECK (length(cliente) <= 80),
	creado_en timestamptz NOT NULL DEFAULT now(),
	usado_en timestamptz,
	-- Revocada: deja de andar pero queda para que el registro siga diciendo quién fue.
	revocado_en timestamptz,
	UNIQUE (negocio_id, id)
);
CREATE INDEX mcp_tokens_negocio ON mcp_tokens (negocio_id, creado_en);
SELECT app.aplicar_rls('mcp_tokens');
GRANT SELECT, INSERT ON mcp_tokens TO tengolibre_app;
GRANT UPDATE (nombre, permisos, cliente, usado_en, revocado_en) ON mcp_tokens TO tengolibre_app;

CREATE POLICY por_hash ON mcp_tokens FOR SELECT TO tengolibre_migrador
	USING (hash = decode(coalesce(nullif(current_setting('app.mcp_hash', true), ''), '00'), 'hex'));

-- +goose StatementBegin
-- De la clave (su hash, en hex) al negocio, antes de fijarlo (como app.negocio_del_calendario).
-- Una clave revocada no resuelve nada.
CREATE FUNCTION app.negocio_del_mcp(p_hash text) RETURNS uuid
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
REVOKE ALL ON FUNCTION app.negocio_del_mcp(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.negocio_del_mcp(text) TO tengolibre_app;

-- Registro de acciones: cada tools/call, con los argumentos resumidos (sin teléfonos ni emails).
CREATE TABLE mcp_registro (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	token_id uuid NOT NULL,
	herramienta text NOT NULL CHECK (length(herramienta) BETWEEN 1 AND 60),
	argumentos jsonb,
	ok boolean NOT NULL,
	-- Lo que pasó, en una frase para el panel («creó un turno de Corte para María…»).
	resumen text NOT NULL DEFAULT '' CHECK (length(resumen) <= 300),
	error text NOT NULL DEFAULT '' CHECK (length(error) <= 300),
	-- Cambió algo (crear, confirmar, mover, cancelar).
	escribe boolean NOT NULL DEFAULT false,
	creado_en timestamptz NOT NULL DEFAULT now(),
	FOREIGN KEY (negocio_id, token_id) REFERENCES mcp_tokens (negocio_id, id) ON DELETE CASCADE
);
CREATE INDEX mcp_registro_negocio ON mcp_registro (negocio_id, creado_en DESC, id DESC);
CREATE INDEX mcp_registro_creado ON mcp_registro (creado_en);
SELECT app.aplicar_rls('mcp_registro');
GRANT SELECT, INSERT ON mcp_registro TO tengolibre_app;
-- «Lo último que hizo la IA» se actualiza solo en el panel.
CREATE TRIGGER tiempo_real AFTER INSERT ON mcp_registro FOR EACH ROW EXECUTE FUNCTION app.avisar_cambio();

-- +goose Down
DROP TABLE mcp_registro;
DROP FUNCTION app.negocio_del_mcp(text);
DROP TABLE mcp_tokens;
