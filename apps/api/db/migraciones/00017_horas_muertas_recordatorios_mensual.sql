-- +goose Up
-- Diseño v2, propuestas 11 y 12 y el reporte mensual:
--  · Cupones restringidos a una franja (días de la semana y horas) y a clientes que ya vienen hace
--    un tiempo: el cupón listo de «Horas muertas» en Reportes. Sin restricción, como hasta ahora.
--  · La fecha de nacimiento del cliente (opcional) y los recordatorios automáticos por negocio:
--    cumpleaños y «hace mucho que no venís».
--  · El reporte mensual por mail: se prende por negocio, y cada encargado elige si lo recibe (la
--    dueña lo recibe siempre que esté prendido).

ALTER TABLE cupones
	-- Días de la semana en que vale (0 = domingo). Vacío: todos.
	ADD COLUMN dias int[] NOT NULL DEFAULT '{}' CHECK (dias <@ ARRAY[0, 1, 2, 3, 4, 5, 6] AND cardinality(dias) <= 7),
	-- Franja horaria (hora local del turno): el turno tiene que empezar en [hora_desde, hora_hasta).
	ADD COLUMN hora_desde time,
	ADD COLUMN hora_hasta time,
	-- Sólo para clientes cuyo primer turno atendido fue hace al menos estos días.
	ADD COLUMN antiguedad_dias int CHECK (antiguedad_dias BETWEEN 1 AND 3650),
	ADD CONSTRAINT cupones_franja CHECK ((hora_desde IS NULL AND hora_hasta IS NULL)
		OR (hora_desde IS NOT NULL AND hora_hasta IS NOT NULL AND hora_hasta > hora_desde));
GRANT UPDATE (dias, hora_desde, hora_hasta, antiguedad_dias) ON cupones TO tengolibre_app;

-- Fecha de nacimiento: la carga el negocio o el cliente al reservar (si no tenía).
ALTER TABLE clientes ADD COLUMN nacimiento date CHECK (nacimiento >= '1900-01-01');
GRANT UPDATE (nacimiento) ON clientes TO tengolibre_app;

-- Recordatorios automáticos: uno por tipo y negocio. Sin fila, apagado con los textos de siempre.
CREATE TABLE recordatorios (
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	tipo text NOT NULL CHECK (tipo IN ('cumpleanios', 'extraniamos')),
	activo boolean NOT NULL DEFAULT false,
	texto text NOT NULL DEFAULT '' CHECK (length(texto) <= 300),
	-- Cupón que va en el mail (opcional). Si se borra, el mail sale sin cupón.
	cupon_id uuid,
	-- Sólo «hace mucho que no venís»: días desde el último turno atendido.
	dias int NOT NULL DEFAULT 60 CHECK (dias BETWEEN 14 AND 365),
	actualizado_en timestamptz NOT NULL DEFAULT now(),
	PRIMARY KEY (negocio_id, tipo),
	FOREIGN KEY (negocio_id, cupon_id) REFERENCES cupones (negocio_id, id) ON DELETE SET NULL (cupon_id)
);
SELECT app.aplicar_rls('recordatorios');
GRANT SELECT, INSERT ON recordatorios TO tengolibre_app;
GRANT UPDATE (activo, texto, cupon_id, dias, actualizado_en) ON recordatorios TO tengolibre_app;

-- Reporte mensual: prendido por negocio; los encargados lo piden uno por uno.
ALTER TABLE ajustes ADD COLUMN reporte_mensual boolean NOT NULL DEFAULT true;
GRANT UPDATE (reporte_mensual) ON ajustes TO tengolibre_app;
ALTER TABLE miembros ADD COLUMN reporte_mensual boolean NOT NULL DEFAULT false;
GRANT UPDATE (reporte_mensual) ON miembros TO tengolibre_app;

-- Los tipos nuevos de aviso. Se suman a la lista que haya (otras migraciones también la amplían).
-- +goose StatementBegin
DO $$
DECLARE
	def text;
BEGIN
	SELECT pg_get_constraintdef(oid) INTO def FROM pg_constraint WHERE conrelid = 'avisos'::regclass AND conname = 'avisos_tipo_check';
	EXECUTE 'ALTER TABLE avisos DROP CONSTRAINT avisos_tipo_check';
	EXECUTE 'ALTER TABLE avisos ADD CONSTRAINT avisos_tipo_check '
		|| replace(def, 'ARRAY[', 'ARRAY[''cumpleanios''::text, ''extraniamos''::text, ''mensual''::text, ');
END
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
DECLARE
	def text;
BEGIN
	DELETE FROM avisos WHERE tipo IN ('cumpleanios', 'extraniamos', 'mensual');
	SELECT pg_get_constraintdef(oid) INTO def FROM pg_constraint WHERE conrelid = 'avisos'::regclass AND conname = 'avisos_tipo_check';
	EXECUTE 'ALTER TABLE avisos DROP CONSTRAINT avisos_tipo_check';
	EXECUTE 'ALTER TABLE avisos ADD CONSTRAINT avisos_tipo_check '
		|| replace(def, '''cumpleanios''::text, ''extraniamos''::text, ''mensual''::text, ', '');
END
$$;
-- +goose StatementEnd
ALTER TABLE miembros DROP COLUMN reporte_mensual;
ALTER TABLE ajustes DROP COLUMN reporte_mensual;
DROP TABLE recordatorios;
ALTER TABLE clientes DROP COLUMN nacimiento;
ALTER TABLE cupones DROP CONSTRAINT cupones_franja, DROP COLUMN antiguedad_dias, DROP COLUMN hora_hasta, DROP COLUMN hora_desde, DROP COLUMN dias;
