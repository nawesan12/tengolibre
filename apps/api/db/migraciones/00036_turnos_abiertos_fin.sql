-- +goose NO TRANSACTION
-- +goose Up
-- Ausentes automáticos (cada 15 minutos): «estado IN ('pendiente','confirmado') AND fin < $1» leía
-- toda la historia de turnos de cada negocio. Concurrente: no frena las reservas mientras se arma.
CREATE INDEX CONCURRENTLY IF NOT EXISTS turnos_abiertos_fin ON turnos (negocio_id, fin) WHERE estado IN ('pendiente', 'confirmado');

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS turnos_abiertos_fin;
