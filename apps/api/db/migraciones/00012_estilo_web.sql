-- +goose Up
-- Estilo publicado de la web del negocio (handoff v2): plantilla y ajustes. El borrador vive en el
-- panel hasta que se publica; acá sólo está lo que ven los clientes. Go valida los valores.
ALTER TABLE negocios ADD COLUMN estilo jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(estilo) = 'object' AND length(estilo::text) <= 500);
GRANT UPDATE (estilo) ON negocios TO tengolibre_app;

-- +goose Down
ALTER TABLE negocios DROP COLUMN estilo;
