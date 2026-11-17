-- +goose Up
-- Fotos de cada servicio (hasta 5) para la página pública y la reserva. La primera es la portada y
-- queda también en servicios.foto, que siguen leyendo los clientes de antes; la API mantiene las dos
-- iguales (ver catalogo.validarServicio). Las fotos que ya había pasan a ser la primera de la lista.
ALTER TABLE servicios ADD COLUMN fotos text[] NOT NULL DEFAULT '{}'
	CHECK (cardinality(fotos) <= 5 AND array_position(fotos, NULL) IS NULL AND length(array_to_string(fotos, '')) <= 5 * 2048);
ALTER TABLE servicios NO FORCE ROW LEVEL SECURITY;
UPDATE servicios SET fotos = ARRAY[foto] WHERE foto IS NOT NULL AND foto <> '';
ALTER TABLE servicios FORCE ROW LEVEL SECURITY;
GRANT UPDATE (fotos) ON servicios TO tengolibre_app;

-- +goose Down
REVOKE UPDATE (fotos) ON servicios FROM tengolibre_app;
ALTER TABLE servicios DROP COLUMN fotos;
