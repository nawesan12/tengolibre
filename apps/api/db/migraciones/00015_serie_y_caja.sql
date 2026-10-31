-- +goose Up
-- Diseño v2, propuestas 03, 07 y 08: el turno fijo (una serie de turnos que se repite cada 1 a 4
-- semanas) y la caja del local (lo que se cobra al atender, con productos del negocio). Montos en
-- pesos enteros, como en 00002.

-- Turno fijo: la serie guarda cómo se armó; cada turno apunta a su serie para poder cancelar «este y
-- los siguientes». Si la serie se borra, los turnos quedan sueltos (no se cancelan).
CREATE TABLE series (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	cliente_id uuid NOT NULL,
	-- Cada cuántas semanas se repite.
	cada_semanas int NOT NULL CHECK (cada_semanas BETWEEN 1 AND 4),
	-- Quién la cargó en el panel.
	clerk_id text NOT NULL DEFAULT '',
	creado_en timestamptz NOT NULL DEFAULT now(),
	UNIQUE (negocio_id, id),
	FOREIGN KEY (negocio_id, cliente_id) REFERENCES clientes (negocio_id, id) ON DELETE CASCADE
);
SELECT app.aplicar_rls('series');
GRANT SELECT, INSERT ON series TO tengolibre_app;

ALTER TABLE turnos ADD COLUMN serie_id uuid;
ALTER TABLE turnos ADD FOREIGN KEY (negocio_id, serie_id) REFERENCES series (negocio_id, id) ON DELETE SET NULL (serie_id);
CREATE INDEX turnos_serie ON turnos (serie_id, inicio) WHERE serie_id IS NOT NULL;

-- Catálogo de productos que se venden en el local (sérum, cera…). stock NULL: no se lleva la cuenta.
CREATE TABLE productos (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	nombre text NOT NULL CHECK (length(nombre) BETWEEN 2 AND 80),
	precio bigint NOT NULL CHECK (precio >= 0 AND precio <= 100000000),
	stock int CHECK (stock BETWEEN 0 AND 1000000),
	activo boolean NOT NULL DEFAULT true,
	orden int NOT NULL DEFAULT 0 CHECK (orden BETWEEN 0 AND 999),
	creado_en timestamptz NOT NULL DEFAULT now(),
	UNIQUE (negocio_id, nombre),
	UNIQUE (negocio_id, id)
);
SELECT app.aplicar_rls('productos');
GRANT SELECT, INSERT, DELETE ON productos TO tengolibre_app;
-- UPDATE (stock) también lo pide el SELECT ... FOR UPDATE con que la caja descuenta stock.
GRANT UPDATE (nombre, precio, stock, activo, orden) ON productos TO tengolibre_app;

-- La caja de un turno atendido: un cobro por turno. Guarda la cuenta como quedó al cobrar (precio,
-- seña, descuentos, membresía y productos) para que el reporte no dependa de precios que cambian.
-- medio «despues»: queda debiendo (estado debe) hasta que se cobra con otro medio.
CREATE TABLE caja_cobros (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	turno_id uuid NOT NULL UNIQUE,
	servicio bigint NOT NULL CHECK (servicio >= 0),
	sena bigint NOT NULL DEFAULT 0 CHECK (sena >= 0),
	-- descuento: el cupón de la reserva; descuento_caja: el que se pone al cobrar.
	descuento bigint NOT NULL DEFAULT 0 CHECK (descuento >= 0),
	descuento_caja bigint NOT NULL DEFAULT 0 CHECK (descuento_caja >= 0),
	membresia bigint NOT NULL DEFAULT 0 CHECK (membresia >= 0),
	productos bigint NOT NULL DEFAULT 0 CHECK (productos >= 0),
	-- Lo que se cobró (o se debe) en el local.
	total bigint NOT NULL CHECK (total >= 0),
	medio text NOT NULL CHECK (medio IN ('efectivo', 'transferencia', 'debito', 'credito', 'mercadopago', 'despues')),
	estado text NOT NULL CHECK (estado IN ('cobrado', 'debe')),
	CHECK ((medio = 'despues') = (estado = 'debe')),
	-- Cuándo entró la plata (NULL mientras se debe): es el día de la caja.
	cobrado_en timestamptz,
	clerk_id text NOT NULL DEFAULT '',
	creado_en timestamptz NOT NULL DEFAULT now(),
	actualizado_en timestamptz NOT NULL DEFAULT now(),
	CHECK ((estado = 'cobrado') = (cobrado_en IS NOT NULL)),
	UNIQUE (negocio_id, id),
	FOREIGN KEY (negocio_id, turno_id) REFERENCES turnos (negocio_id, id) ON DELETE CASCADE
);
CREATE INDEX caja_cobros_dia ON caja_cobros (negocio_id, cobrado_en) WHERE estado = 'cobrado';
SELECT app.aplicar_rls('caja_cobros');
GRANT SELECT, INSERT, DELETE ON caja_cobros TO tengolibre_app;
GRANT UPDATE (medio, estado, cobrado_en, clerk_id, actualizado_en) ON caja_cobros TO tengolibre_app;

-- Lo que se llevó: copia nombre y precio del producto (si después se borra, la caja no cambia).
CREATE TABLE caja_items (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	cobro_id uuid NOT NULL,
	producto_id uuid,
	nombre text NOT NULL CHECK (length(nombre) <= 80),
	precio bigint NOT NULL CHECK (precio >= 0),
	cantidad int NOT NULL CHECK (cantidad BETWEEN 1 AND 99),
	FOREIGN KEY (negocio_id, cobro_id) REFERENCES caja_cobros (negocio_id, id) ON DELETE CASCADE,
	FOREIGN KEY (negocio_id, producto_id) REFERENCES productos (negocio_id, id) ON DELETE SET NULL (producto_id)
);
CREATE INDEX caja_items_cobro ON caja_items (cobro_id);
SELECT app.aplicar_rls('caja_items');
GRANT SELECT, INSERT ON caja_items TO tengolibre_app;

CREATE TRIGGER tiempo_real AFTER INSERT OR UPDATE OR DELETE ON caja_cobros FOR EACH ROW EXECUTE FUNCTION app.avisar_cambio();

-- +goose Down
DROP TRIGGER tiempo_real ON caja_cobros;
DROP TABLE caja_items;
DROP TABLE caja_cobros;
DROP TABLE productos;
DROP INDEX turnos_serie;
ALTER TABLE turnos DROP COLUMN serie_id;
DROP TABLE series;
