-- +goose Up
-- Segunda tanda de Jiku: el CRM del panel. Etiquetas (ClientTag) y su asignación, notas internas
-- (ClientNote), penalidades por ausencias (ClientPenalty, BusinessSettings.noShow*) y lo que
-- necesitan los reportes. Los clientes y los turnos ya están en 00002.

-- Las notas y las penalidades guardan qué miembro las hizo: FK compuesta, como el resto.
ALTER TABLE miembros ADD CONSTRAINT miembros_negocio_id UNIQUE (negocio_id, id);

-- Ausencias (noShowThreshold, noShowPenaltyDays, noShowAutoMark).
ALTER TABLE ajustes
	-- Cuántas ausencias en 90 días bloquean al cliente. 0: no se penaliza solo.
	ADD COLUMN umbral_ausencias int NOT NULL DEFAULT 3 CHECK (umbral_ausencias BETWEEN 0 AND 20),
	-- Cuántos días dura la penalidad automática.
	ADD COLUMN dias_penalidad int NOT NULL DEFAULT 30 CHECK (dias_penalidad BETWEEN 1 AND 365),
	-- Marcar ausente, pasadas dos horas del fin, el turno pendiente o confirmado que nadie cerró.
	ADD COLUMN marcar_ausentes boolean NOT NULL DEFAULT false;
GRANT UPDATE (umbral_ausencias, dias_penalidad, marcar_ausentes) ON ajustes TO tengolibre_app;

-- Etiquetas de clientes, con color para el panel.
CREATE TABLE etiquetas (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	nombre text NOT NULL CHECK (length(nombre) BETWEEN 1 AND 40),
	color text NOT NULL DEFAULT '#6366F1' CHECK (color ~ '^#[0-9A-Fa-f]{6}$'),
	creado_en timestamptz NOT NULL DEFAULT now(),
	UNIQUE (negocio_id, id)
);
-- "VIP" y "vip" son la misma etiqueta.
CREATE UNIQUE INDEX etiquetas_nombre ON etiquetas (negocio_id, lower(nombre));
SELECT app.aplicar_rls('etiquetas');
GRANT SELECT, INSERT, DELETE ON etiquetas TO tengolibre_app;
GRANT UPDATE (nombre, color) ON etiquetas TO tengolibre_app;

CREATE TABLE cliente_etiquetas (
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	cliente_id uuid NOT NULL,
	etiqueta_id uuid NOT NULL,
	creado_en timestamptz NOT NULL DEFAULT now(),
	PRIMARY KEY (cliente_id, etiqueta_id),
	FOREIGN KEY (negocio_id, cliente_id) REFERENCES clientes (negocio_id, id) ON DELETE CASCADE,
	FOREIGN KEY (negocio_id, etiqueta_id) REFERENCES etiquetas (negocio_id, id) ON DELETE CASCADE
);
CREATE INDEX cliente_etiquetas_etiqueta ON cliente_etiquetas (etiqueta_id);
SELECT app.aplicar_rls('cliente_etiquetas');
GRANT SELECT, INSERT, DELETE ON cliente_etiquetas TO tengolibre_app;

-- Notas internas sobre un cliente. Copian el nombre de quien la escribió: si sale del equipo, la
-- nota sigue diciendo de quién era.
CREATE TABLE notas_cliente (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	cliente_id uuid NOT NULL,
	miembro_id uuid,
	autor text NOT NULL CHECK (length(autor) <= 120),
	texto text NOT NULL CHECK (length(texto) BETWEEN 1 AND 2000),
	creado_en timestamptz NOT NULL DEFAULT now(),
	FOREIGN KEY (negocio_id, cliente_id) REFERENCES clientes (negocio_id, id) ON DELETE CASCADE,
	FOREIGN KEY (negocio_id, miembro_id) REFERENCES miembros (negocio_id, id) ON DELETE SET NULL (miembro_id)
);
CREATE INDEX notas_cliente_cliente ON notas_cliente (cliente_id, creado_en DESC);
SELECT app.aplicar_rls('notas_cliente');
GRANT SELECT, INSERT, DELETE ON notas_cliente TO tengolibre_app;

-- Penalidades: mientras haya una vigente (sin levantar y con hasta en el futuro), el cliente no
-- reserva desde la página pública. Desde el panel sí.
CREATE TABLE penalidades (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	cliente_id uuid NOT NULL,
	motivo text NOT NULL CHECK (motivo IN ('ausencias', 'manual')),
	nota text NOT NULL DEFAULT '' CHECK (length(nota) <= 500),
	-- Las automáticas: cuántas ausencias la dispararon y el inicio de la última. Las ausencias de
	-- hasta ahí ya "se pagaron": la próxima penalidad cuenta sólo las posteriores.
	ausencias int NOT NULL DEFAULT 0,
	ultima_ausencia timestamptz,
	hasta timestamptz NOT NULL,
	creado_por uuid,
	creado_en timestamptz NOT NULL DEFAULT now(),
	levantada_en timestamptz,
	levantada_por uuid,
	CHECK (hasta > creado_en),
	CHECK ((motivo = 'ausencias') = (ultima_ausencia IS NOT NULL)),
	FOREIGN KEY (negocio_id, cliente_id) REFERENCES clientes (negocio_id, id) ON DELETE CASCADE,
	FOREIGN KEY (negocio_id, creado_por) REFERENCES miembros (negocio_id, id) ON DELETE SET NULL (creado_por),
	FOREIGN KEY (negocio_id, levantada_por) REFERENCES miembros (negocio_id, id) ON DELETE SET NULL (levantada_por)
);
CREATE INDEX penalidades_cliente ON penalidades (cliente_id, hasta) WHERE levantada_en IS NULL;
CREATE INDEX penalidades_negocio ON penalidades (negocio_id, creado_en DESC);
SELECT app.aplicar_rls('penalidades');
GRANT SELECT, INSERT ON penalidades TO tengolibre_app;
GRANT UPDATE (levantada_en, levantada_por) ON penalidades TO tengolibre_app;

-- +goose StatementBegin
-- Penalidad automática: cuando un turno pasa a ausente (desde el panel o por el marcado
-- automático), si el cliente llegó al umbral de ausencias en los 90 días anteriores a ese turno y
-- no tiene una penalidad vigente, se crea una. Corre como quien actualiza (sin SECURITY DEFINER):
-- la RLS del negocio fijado vale también acá.
CREATE FUNCTION app.penalizar_ausencias() RETURNS trigger
LANGUAGE plpgsql SET search_path = public, app AS $$
DECLARE
	umbral int;
	dias int;
	desde timestamptz;
	cuantas int;
	ultima timestamptz;
BEGIN
	SELECT a.umbral_ausencias, a.dias_penalidad INTO umbral, dias FROM ajustes a WHERE a.negocio_id = NEW.negocio_id;
	IF NOT FOUND OR umbral = 0 THEN
		RETURN NULL;
	END IF;
	IF EXISTS (SELECT 1 FROM penalidades p WHERE p.cliente_id = NEW.cliente_id AND p.levantada_en IS NULL AND p.hasta > now()) THEN
		RETURN NULL;
	END IF;
	SELECT max(p.ultima_ausencia) INTO desde FROM penalidades p WHERE p.cliente_id = NEW.cliente_id;
	SELECT count(*), max(t.inicio) INTO cuantas, ultima FROM turnos t
		WHERE t.cliente_id = NEW.cliente_id AND t.estado = 'ausente'
		AND t.inicio > NEW.inicio - interval '90 days' AND t.inicio <= NEW.inicio
		AND t.inicio > coalesce(desde, '-infinity');
	IF cuantas >= umbral THEN
		INSERT INTO penalidades (negocio_id, cliente_id, motivo, ausencias, ultima_ausencia, hasta)
		VALUES (NEW.negocio_id, NEW.cliente_id, 'ausencias', cuantas, ultima, now() + make_interval(days => dias));
	END IF;
	RETURN NULL;
END
$$;
-- +goose StatementEnd
CREATE TRIGGER penalizar_ausencias AFTER UPDATE OF estado ON turnos
	FOR EACH ROW WHEN (NEW.estado = 'ausente' AND OLD.estado <> 'ausente')
	EXECUTE FUNCTION app.penalizar_ausencias();

-- Búsquedas y reportes: los turnos de un cliente en orden, y los clientes por fecha de alta.
CREATE INDEX turnos_cliente_inicio ON turnos (cliente_id, inicio);
CREATE INDEX clientes_negocio_nombre ON clientes (negocio_id, lower(nombre));

-- +goose Down
DROP INDEX clientes_negocio_nombre;
DROP INDEX turnos_cliente_inicio;
DROP TRIGGER penalizar_ausencias ON turnos;
DROP FUNCTION app.penalizar_ausencias();
DROP TABLE penalidades;
DROP TABLE notas_cliente;
DROP TABLE cliente_etiquetas;
DROP TABLE etiquetas;
ALTER TABLE ajustes DROP COLUMN umbral_ausencias, DROP COLUMN dias_penalidad, DROP COLUMN marcar_ausentes;
ALTER TABLE miembros DROP CONSTRAINT miembros_negocio_id;
