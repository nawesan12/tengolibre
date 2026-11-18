-- +goose Up
-- Asistente del panel con IA (Claude Haiku). Dos cosas viven en la base y no en memoria: la medición
-- (para los topes por negocio y el de toda la plataforma, que tienen que sobrevivir a un reinicio) y
-- las propuestas que la persona confirma con un botón (puede tocar «Confirmar» un rato después, y
-- puede haber más de un proceso de API).

-- Uso del modelo por negocio, día y ruta («asistente», …). Se suma con cada respuesta.
CREATE TABLE ia_uso (
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	dia date NOT NULL,
	ruta text NOT NULL CHECK (length(ruta) BETWEEN 1 AND 40),
	-- Respuestas con IA (lo que cuenta el cupo) y llamadas al modelo (una respuesta puede tener hasta 4).
	respuestas integer NOT NULL DEFAULT 0 CHECK (respuestas >= 0),
	pedidos integer NOT NULL DEFAULT 0 CHECK (pedidos >= 0),
	tokens_entrada bigint NOT NULL DEFAULT 0 CHECK (tokens_entrada >= 0),
	cache_leido bigint NOT NULL DEFAULT 0 CHECK (cache_leido >= 0),
	cache_escrito bigint NOT NULL DEFAULT 0 CHECK (cache_escrito >= 0),
	tokens_salida bigint NOT NULL DEFAULT 0 CHECK (tokens_salida >= 0),
	-- Costo en millonésimas de dólar.
	costo_micro bigint NOT NULL DEFAULT 0 CHECK (costo_micro >= 0),
	PRIMARY KEY (negocio_id, dia, ruta)
);
CREATE INDEX ia_uso_dia ON ia_uso (dia);
SELECT app.aplicar_rls('ia_uso');
GRANT SELECT, INSERT ON ia_uso TO tengolibre_app;
GRANT UPDATE (respuestas, pedidos, tokens_entrada, cache_leido, cache_escrito, tokens_salida, costo_micro) ON ia_uso TO tengolibre_app;

-- +goose StatementBegin
-- Lo gastado por toda la plataforma desde un día (el tope global). Cruza negocios, por eso es
-- SECURITY DEFINER y devuelve sólo la suma.
CREATE FUNCTION app.ia_gasto_desde(p_desde date) RETURNS bigint
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, app AS $$
	SELECT coalesce(sum(costo_micro), 0)::bigint FROM ia_uso WHERE dia >= p_desde
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.ia_gasto_desde(date) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.ia_gasto_desde(date) TO tengolibre_app;

-- Propuestas del asistente: una acción (crear, mover, confirmar o cancelar un turno, ofrecer un hueco a
-- la lista) que no se hace hasta que la persona que preguntó toca «Confirmar». Vencen solas.
CREATE TABLE asistente_propuestas (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	-- Quién preguntó: sólo esa persona la puede confirmar.
	clerk_id text NOT NULL CHECK (length(clerk_id) BETWEEN 1 AND 80),
	accion text NOT NULL CHECK (accion IN ('crear_turno', 'confirmar_turno', 'reprogramar_turno', 'cancelar_turno', 'ofrecer_a_la_lista')),
	argumentos jsonb NOT NULL,
	-- La frase que ve la persona («Mover a Juana del jueves 20 a las 15:00 al viernes 21 a las 17:00»).
	resumen text NOT NULL CHECK (length(resumen) BETWEEN 1 AND 300),
	creada_en timestamptz NOT NULL DEFAULT now(),
	vence_en timestamptz NOT NULL,
	-- confirmada | descartada | fallida (null: esperando).
	estado text CHECK (estado IN ('confirmada', 'descartada', 'fallida')),
	resuelta_en timestamptz
);
CREATE INDEX asistente_propuestas_negocio ON asistente_propuestas (negocio_id, creada_en DESC);
SELECT app.aplicar_rls('asistente_propuestas');
GRANT SELECT, INSERT ON asistente_propuestas TO tengolibre_app;
GRANT UPDATE (estado, resuelta_en) ON asistente_propuestas TO tengolibre_app;

-- +goose Down
DROP TABLE asistente_propuestas;
DROP FUNCTION app.ia_gasto_desde(date);
DROP TABLE ia_uso;
