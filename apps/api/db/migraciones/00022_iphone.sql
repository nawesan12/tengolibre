-- +goose Up
-- El iPhone: la suscripción por la App Store (además de Mercado Pago), la Live Activity del turno en
-- curso actualizada por push y los avisos al iPhone de la clienta (por el código de su turno).

-- Suscripción: de dónde se paga. Las de Apple se identifican por la transacción original (la que
-- sigue igual en cada renovación) y se concilian con la API y las notificaciones de la App Store.
ALTER TABLE suscripciones
	ADD COLUMN origen text NOT NULL DEFAULT 'mercadopago' CHECK (origen IN ('mercadopago', 'apple')),
	ADD COLUMN apple_original text UNIQUE,
	ADD COLUMN apple_producto text CHECK (length(apple_producto) <= 120),
	ADD COLUMN apple_entorno text CHECK (apple_entorno IN ('Production', 'Sandbox'));
GRANT UPDATE (plan, monto, apple_producto, apple_entorno) ON suscripciones TO tengolibre_app;

-- Live Activity: el token de push de la actividad de un turno (uno por teléfono). enviado_en: hasta
-- cuándo se le mandó el estado del turno (el worker manda lo que cambió después).
CREATE TABLE turno_actividades (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	turno_id uuid NOT NULL REFERENCES turnos (id) ON DELETE CASCADE,
	token text NOT NULL UNIQUE CHECK (token ~ '^[0-9a-f]{64,400}$'),
	entorno text NOT NULL DEFAULT 'produccion' CHECK (entorno IN ('sandbox', 'produccion')),
	enviado_en timestamptz NOT NULL DEFAULT now(),
	creado_en timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX turno_actividades_turno ON turno_actividades (turno_id);
SELECT app.aplicar_rls('turno_actividades');
GRANT SELECT, INSERT, DELETE ON turno_actividades TO tengolibre_app;
GRANT UPDATE (enviado_en) ON turno_actividades TO tengolibre_app;

-- Los iPhone de la clienta que siguen un turno (la app en modo clienta, por el código del turno).
CREATE TABLE turno_dispositivos (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	turno_id uuid NOT NULL REFERENCES turnos (id) ON DELETE CASCADE,
	token text NOT NULL CHECK (token ~ '^[0-9a-f]{64,200}$'),
	entorno text NOT NULL DEFAULT 'produccion' CHECK (entorno IN ('sandbox', 'produccion')),
	creado_en timestamptz NOT NULL DEFAULT now(),
	UNIQUE (turno_id, token)
);
CREATE INDEX turno_dispositivos_token ON turno_dispositivos (token);
SELECT app.aplicar_rls('turno_dispositivos');
GRANT SELECT, INSERT, DELETE ON turno_dispositivos TO tengolibre_app;

-- Avisos de un turno que todavía no salieron al iPhone de la clienta (los nuevos; los viejos, no).
-- (Se agrega en false para los que ya existen —la RLS forzada no deja recorrerlos acá— y el default
-- pasa a true para los nuevos.)
ALTER TABLE avisos ADD COLUMN push_cliente boolean NOT NULL DEFAULT false;
ALTER TABLE avisos ALTER COLUMN push_cliente SET DEFAULT true;
CREATE INDEX avisos_push_cliente ON avisos (creado_en) WHERE push_cliente;
GRANT UPDATE (push_cliente) ON avisos TO tengolibre_app;

-- +goose Down
DROP INDEX avisos_push_cliente;
ALTER TABLE avisos DROP COLUMN push_cliente;
DROP TABLE turno_dispositivos;
DROP TABLE turno_actividades;
ALTER TABLE suscripciones DROP COLUMN origen, DROP COLUMN apple_original, DROP COLUMN apple_producto, DROP COLUMN apple_entorno;
