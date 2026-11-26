-- +goose Up
-- Sugerencias de la casa (planes/promos-cruzadas-produccion-corregido.md): en el Inicio del panel, a
-- la dueña o la encargada, una recomendación de otra app de Grupo Vimo que resuelve algo que Tengo
-- Libre no hace. Todo es aditivo: dos columnas en negocios y dos tablas nuevas.

-- El interruptor «Sugerencias» de Ajustes (retiro o bloqueo del art. 27 de la Ley 25.326): prendido
-- por defecto, guardado en el servidor para que valga en la web y en las apps.
ALTER TABLE negocios ADD COLUMN sugerencias_apagadas boolean NOT NULL DEFAULT false;
GRANT UPDATE (sugerencias_apagadas) ON negocios TO tengolibre_app;

-- De dónde llegó quien creó el negocio (utm del link que abrió /empezar): fuente, medio, campaña y
-- contenido, validados en la API. Para atribuir altas a una sugerencia de otra app. NULL: sin origen.
ALTER TABLE negocios ADD COLUMN origen jsonb CHECK (origen IS NULL OR jsonb_typeof(origen) = 'object');
GRANT UPDATE (origen) ON negocios TO tengolibre_app;

-- Las sugerencias que el negocio cerró: hasta una fecha («Ocultar por 30 días») o para siempre («Ya la
-- uso / No me interesa», hasta = 9999-12-31). Se van con el negocio (cascada de la purga).
CREATE TABLE sugerencias_cerradas (
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	marca text NOT NULL CHECK (marca ~ '^[a-z]{2,20}$'),
	hasta timestamptz NOT NULL,
	creada_en timestamptz NOT NULL DEFAULT now(),
	PRIMARY KEY (negocio_id, marca)
);
SELECT app.aplicar_rls('sugerencias_cerradas');
GRANT SELECT, INSERT, DELETE ON sugerencias_cerradas TO tengolibre_app;
GRANT UPDATE (hasta, creada_en) ON sugerencias_cerradas TO tengolibre_app;

-- Cuántas veces se mostró cada sugerencia y cuántos clics tuvo, por día. Agregado: sin negocio ni
-- persona (no hay nada que borrar con una cuenta).
CREATE TABLE sugerencias_metricas (
	dia date NOT NULL,
	sugerencia_id text NOT NULL CHECK (sugerencia_id ~ '^[a-z0-9-]{2,40}$'),
	mostradas integer NOT NULL DEFAULT 0 CHECK (mostradas >= 0),
	clics integer NOT NULL DEFAULT 0 CHECK (clics >= 0),
	PRIMARY KEY (dia, sugerencia_id)
);
-- RLS como toda tabla, con una política abierta: no es de ningún negocio.
ALTER TABLE sugerencias_metricas ENABLE ROW LEVEL SECURITY;
ALTER TABLE sugerencias_metricas FORCE ROW LEVEL SECURITY;
CREATE POLICY agregado ON sugerencias_metricas TO tengolibre_app, tengolibre_migrador USING (true) WITH CHECK (true);
GRANT SELECT, INSERT ON sugerencias_metricas TO tengolibre_app;
GRANT UPDATE (mostradas, clics) ON sugerencias_metricas TO tengolibre_app;

-- +goose Down
DROP TABLE sugerencias_metricas;
DROP TABLE sugerencias_cerradas;
ALTER TABLE negocios DROP COLUMN origen;
ALTER TABLE negocios DROP COLUMN sugerencias_apagadas;
