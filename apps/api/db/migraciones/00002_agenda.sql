-- +goose Up
-- Primera tanda de Jiku: el equipo del negocio (con Clerk), los ajustes de reserva, el catálogo
-- (categorías, servicios, profesionales), los horarios y bloqueos de cada profesional, los clientes
-- y los turnos. Precios en pesos enteros; horas en la zona del negocio.

ALTER TABLE negocios
	ADD COLUMN descripcion text NOT NULL DEFAULT '' CHECK (length(descripcion) <= 2000),
	ADD COLUMN direccion text NOT NULL DEFAULT '' CHECK (length(direccion) <= 200),
	ADD COLUMN ciudad text NOT NULL DEFAULT '' CHECK (length(ciudad) <= 80),
	ADD COLUMN telefono text NOT NULL DEFAULT '' CHECK (length(telefono) <= 30),
	ADD COLUMN whatsapp text NOT NULL DEFAULT '' CHECK (length(whatsapp) <= 30),
	ADD COLUMN instagram text NOT NULL DEFAULT '' CHECK (length(instagram) <= 60),
	ADD COLUMN logo text CHECK (length(logo) <= 2048),
	ADD COLUMN activo boolean NOT NULL DEFAULT true;
GRANT UPDATE (descripcion, direccion, ciudad, telefono, whatsapp, instagram, logo) ON negocios TO tengolibre_app;

-- Ajustes de reserva (BusinessSettings). Una fila por negocio, creada con el alta.
CREATE TABLE ajustes (
	negocio_id uuid PRIMARY KEY REFERENCES negocios (id) ON DELETE CASCADE,
	-- Cada cuántos minutos arranca un horario (slotInterval).
	intervalo int NOT NULL DEFAULT 30 CHECK (intervalo IN (5, 10, 15, 20, 30, 45, 60)),
	-- Con cuánta anticipación mínima se puede reservar, en minutos (minAdvanceHours).
	anticipacion_min int NOT NULL DEFAULT 120 CHECK (anticipacion_min BETWEEN 0 AND 10080),
	-- Cuántos días hacia adelante se ve la agenda (maxAdvanceDays).
	dias_vista int NOT NULL DEFAULT 30 CHECK (dias_vista BETWEEN 1 AND 180),
	-- Minutos libres después de cada turno (bufferMinutes).
	margen_min int NOT NULL DEFAULT 0 CHECK (margen_min BETWEEN 0 AND 120),
	-- Hasta cuántas horas antes el cliente puede cancelar solo.
	cancelar_hasta_horas int NOT NULL DEFAULT 2 CHECK (cancelar_hasta_horas BETWEEN 0 AND 168),
	politica_cancelacion text NOT NULL DEFAULT '' CHECK (length(politica_cancelacion) <= 2000),
	actualizado_en timestamptz NOT NULL DEFAULT now()
);
SELECT app.aplicar_rls('ajustes');
GRANT SELECT ON ajustes TO tengolibre_app;
GRANT UPDATE (intervalo, anticipacion_min, dias_vista, margen_min, cancelar_hasta_horas, politica_cancelacion, actualizado_en)
	ON ajustes TO tengolibre_app;

-- El equipo que entra al panel (UserBusiness). La persona es la de Clerk.
CREATE TABLE miembros (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	clerk_id text NOT NULL CHECK (clerk_id ~ '^user_[A-Za-z0-9]{10,64}$'),
	email text NOT NULL DEFAULT '' CHECK (length(email) <= 120),
	nombre text NOT NULL DEFAULT '' CHECK (length(nombre) <= 80),
	rol text NOT NULL CHECK (rol IN ('duenio', 'encargado', 'recepcion', 'profesional')),
	activo boolean NOT NULL DEFAULT true,
	creado_en timestamptz NOT NULL DEFAULT now(),
	UNIQUE (negocio_id, clerk_id)
);
CREATE INDEX miembros_clerk ON miembros (clerk_id);
SELECT app.aplicar_rls('miembros');
GRANT SELECT, INSERT ON miembros TO tengolibre_app;
GRANT UPDATE (email, nombre, rol, activo) ON miembros TO tengolibre_app;

-- La RLS forzada alcanza también al dueño de las funciones SECURITY DEFINER: estas políticas le
-- dejan ver sólo lo que se está buscando (la persona, el slug o el código fijados por la función).
CREATE POLICY por_persona ON miembros FOR SELECT TO tengolibre_migrador
	USING (clerk_id = current_setting('app.clerk_id', true));
CREATE POLICY por_persona ON negocios FOR SELECT TO tengolibre_migrador
	USING (id IN (SELECT negocio_id FROM miembros WHERE clerk_id = current_setting('app.clerk_id', true)));
CREATE POLICY por_slug ON negocios FOR SELECT TO tengolibre_migrador
	USING (slug = current_setting('app.slug', true));

-- +goose StatementBegin
-- Los negocios de una persona (para elegir en cuál entra): todavía no hay negocio fijado.
CREATE FUNCTION app.negocios_de(p_clerk text)
RETURNS TABLE (negocio_id uuid, slug text, nombre text, rol text)
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
BEGIN
	PERFORM set_config('app.clerk_id', p_clerk, true);
	RETURN QUERY SELECT n.id, n.slug, n.nombre, m.rol
		FROM miembros m JOIN negocios n ON n.id = m.negocio_id
		WHERE m.clerk_id = p_clerk AND m.activo AND n.activo
		ORDER BY n.nombre;
	PERFORM set_config('app.clerk_id', '', true);
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.negocios_de(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.negocios_de(text) TO tengolibre_app;

-- +goose StatementBegin
-- Alta de negocio: crea el negocio, sus ajustes y a la persona como dueña, y deja el negocio fijado
-- en la transacción.
CREATE FUNCTION app.crear_negocio(p_slug text, p_nombre text, p_clerk text, p_email text, p_persona text)
RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	nuevo uuid := gen_random_uuid();
BEGIN
	PERFORM set_config('app.negocio_id', nuevo::text, true);
	INSERT INTO negocios (id, slug, nombre) VALUES (nuevo, p_slug, p_nombre);
	INSERT INTO ajustes (negocio_id) VALUES (nuevo);
	INSERT INTO miembros (negocio_id, clerk_id, email, nombre, rol) VALUES (nuevo, p_clerk, p_email, p_persona, 'duenio');
	RETURN nuevo;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.crear_negocio(text, text, text, text, text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.crear_negocio(text, text, text, text, text) TO tengolibre_app;

-- +goose StatementBegin
-- La página pública: del slug al negocio activo (después se lee con el negocio fijado). También
-- dice si un slug está tomado (con activo = false lo ignora, pero el índice único lo sigue frenando).
CREATE FUNCTION app.negocio_publico(p_slug text) RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	n uuid;
BEGIN
	PERFORM set_config('app.slug', p_slug, true);
	SELECT id INTO n FROM negocios WHERE slug = p_slug AND activo;
	PERFORM set_config('app.slug', '', true);
	RETURN n;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.negocio_publico(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.negocio_publico(text) TO tengolibre_app;

CREATE TABLE categorias (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	nombre text NOT NULL CHECK (length(nombre) BETWEEN 1 AND 60),
	orden int NOT NULL DEFAULT 0,
	UNIQUE (negocio_id, nombre),
	-- Para las FK compuestas: una categoría sólo se usa en su negocio.
	UNIQUE (negocio_id, id)
);
SELECT app.aplicar_rls('categorias');
GRANT SELECT, INSERT, DELETE ON categorias TO tengolibre_app;
GRANT UPDATE (nombre, orden) ON categorias TO tengolibre_app;

CREATE TABLE servicios (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	categoria_id uuid,
	nombre text NOT NULL CHECK (length(nombre) BETWEEN 2 AND 80),
	descripcion text NOT NULL DEFAULT '' CHECK (length(descripcion) <= 1000),
	duracion int NOT NULL CHECK (duracion BETWEEN 5 AND 480 AND duracion % 5 = 0),
	precio bigint NOT NULL CHECK (precio >= 0),
	foto text CHECK (length(foto) <= 2048),
	orden int NOT NULL DEFAULT 0,
	activo boolean NOT NULL DEFAULT true,
	creado_en timestamptz NOT NULL DEFAULT now(),
	UNIQUE (negocio_id, id),
	FOREIGN KEY (negocio_id, categoria_id) REFERENCES categorias (negocio_id, id) ON DELETE SET NULL (categoria_id)
);
CREATE INDEX servicios_negocio ON servicios (negocio_id, orden);
SELECT app.aplicar_rls('servicios');
GRANT SELECT, INSERT, DELETE ON servicios TO tengolibre_app;
GRANT UPDATE (categoria_id, nombre, descripcion, duracion, precio, foto, orden, activo) ON servicios TO tengolibre_app;

-- Profesionales (StaffMember). miembro_id: si además entra al panel.
CREATE TABLE profesionales (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	miembro_id uuid REFERENCES miembros (id) ON DELETE SET NULL,
	nombre text NOT NULL CHECK (length(nombre) BETWEEN 2 AND 60),
	especialidad text NOT NULL DEFAULT '' CHECK (length(especialidad) <= 60),
	bio text NOT NULL DEFAULT '' CHECK (length(bio) <= 1000),
	foto text CHECK (length(foto) <= 2048),
	email text NOT NULL DEFAULT '' CHECK (length(email) <= 120),
	telefono text NOT NULL DEFAULT '' CHECK (length(telefono) <= 30),
	orden int NOT NULL DEFAULT 0,
	activo boolean NOT NULL DEFAULT true,
	creado_en timestamptz NOT NULL DEFAULT now(),
	UNIQUE (negocio_id, id)
);
CREATE INDEX profesionales_negocio ON profesionales (negocio_id, orden);
SELECT app.aplicar_rls('profesionales');
GRANT SELECT, INSERT, DELETE ON profesionales TO tengolibre_app;
-- UPDATE también lo pide el SELECT ... FOR UPDATE con que se reserva (agenda.Reservar).
GRANT UPDATE (miembro_id, nombre, especialidad, bio, foto, email, telefono, orden, activo) ON profesionales TO tengolibre_app;

-- Qué servicios hace cada profesional (StaffService).
CREATE TABLE profesional_servicios (
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	profesional_id uuid NOT NULL,
	servicio_id uuid NOT NULL,
	PRIMARY KEY (profesional_id, servicio_id),
	FOREIGN KEY (negocio_id, profesional_id) REFERENCES profesionales (negocio_id, id) ON DELETE CASCADE,
	FOREIGN KEY (negocio_id, servicio_id) REFERENCES servicios (negocio_id, id) ON DELETE CASCADE
);
CREATE INDEX profesional_servicios_servicio ON profesional_servicios (servicio_id);
SELECT app.aplicar_rls('profesional_servicios');
GRANT SELECT, INSERT, DELETE ON profesional_servicios TO tengolibre_app;

-- Horario semanal de cada profesional (WorkingHours). Puede haber varias franjas por día (cortado).
CREATE TABLE horarios (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	profesional_id uuid NOT NULL,
	-- 0 = domingo.
	dia int NOT NULL CHECK (dia BETWEEN 0 AND 6),
	desde time NOT NULL,
	hasta time NOT NULL,
	CHECK (hasta > desde),
	UNIQUE (profesional_id, dia, desde),
	FOREIGN KEY (negocio_id, profesional_id) REFERENCES profesionales (negocio_id, id) ON DELETE CASCADE
);
SELECT app.aplicar_rls('horarios');
GRANT SELECT, INSERT, DELETE ON horarios TO tengolibre_app;

-- Días u horas en que un profesional no atiende (BlockedDate). Sin desde/hasta: el día entero.
CREATE TABLE bloqueos (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	profesional_id uuid NOT NULL,
	fecha date NOT NULL,
	desde time,
	hasta time,
	motivo text NOT NULL DEFAULT '' CHECK (length(motivo) <= 200),
	creado_en timestamptz NOT NULL DEFAULT now(),
	CHECK ((desde IS NULL AND hasta IS NULL) OR (desde IS NOT NULL AND hasta IS NOT NULL AND hasta > desde)),
	FOREIGN KEY (negocio_id, profesional_id) REFERENCES profesionales (negocio_id, id) ON DELETE CASCADE
);
CREATE INDEX bloqueos_profesional ON bloqueos (profesional_id, fecha);
SELECT app.aplicar_rls('bloqueos');
GRANT SELECT, INSERT, DELETE ON bloqueos TO tengolibre_app;

-- Un horario puntual que se bloquea todas las semanas (RecurringBlockedSlot), p. ej. el almuerzo.
CREATE TABLE bloqueos_semanales (
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	profesional_id uuid NOT NULL,
	dia int NOT NULL CHECK (dia BETWEEN 0 AND 6),
	hora time NOT NULL,
	PRIMARY KEY (profesional_id, dia, hora),
	FOREIGN KEY (negocio_id, profesional_id) REFERENCES profesionales (negocio_id, id) ON DELETE CASCADE
);
SELECT app.aplicar_rls('bloqueos_semanales');
GRANT SELECT, INSERT, DELETE ON bloqueos_semanales TO tengolibre_app;

-- Excepción de un horario en una fecha (DateSlotOverride): lo bloquea, o lo libera aunque haya un
-- bloqueo semanal.
CREATE TABLE excepciones (
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	profesional_id uuid NOT NULL,
	fecha date NOT NULL,
	hora time NOT NULL,
	tipo text NOT NULL CHECK (tipo IN ('bloqueado', 'libre')),
	PRIMARY KEY (profesional_id, fecha, hora),
	FOREIGN KEY (negocio_id, profesional_id) REFERENCES profesionales (negocio_id, id) ON DELETE CASCADE
);
SELECT app.aplicar_rls('excepciones');
GRANT SELECT, INSERT, DELETE ON excepciones TO tengolibre_app;
GRANT UPDATE (tipo) ON excepciones TO tengolibre_app;

-- Clientes del negocio (GuestClient): el teléfono los identifica cuando lo hay.
CREATE TABLE clientes (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	nombre text NOT NULL CHECK (length(nombre) BETWEEN 1 AND 80),
	telefono text NOT NULL DEFAULT '' CHECK (length(telefono) <= 30),
	email text NOT NULL DEFAULT '' CHECK (length(email) <= 120),
	creado_en timestamptz NOT NULL DEFAULT now(),
	UNIQUE (negocio_id, id)
);
CREATE UNIQUE INDEX clientes_telefono ON clientes (negocio_id, telefono) WHERE telefono <> '';
SELECT app.aplicar_rls('clientes');
GRANT SELECT, INSERT ON clientes TO tengolibre_app;
GRANT UPDATE (nombre, email) ON clientes TO tengolibre_app;

-- Turnos (Appointment). Copian el nombre del servicio y del profesional: si después se borran, la
-- agenda no queda en blanco (por eso las FK quedan en NULL y no en cascada).
CREATE TABLE turnos (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	-- Para el link del cliente: nadie adivina el de otro.
	codigo text NOT NULL UNIQUE CHECK (codigo ~ '^[A-Za-z0-9_-]{16,64}$'),
	servicio_id uuid,
	servicio text NOT NULL CHECK (length(servicio) <= 80),
	profesional_id uuid,
	profesional text NOT NULL CHECK (length(profesional) <= 60),
	cliente_id uuid NOT NULL,
	inicio timestamptz NOT NULL,
	fin timestamptz NOT NULL,
	precio bigint NOT NULL CHECK (precio >= 0),
	estado text NOT NULL CHECK (estado IN ('pendiente_pago', 'pendiente', 'confirmado', 'atendido', 'cancelado', 'ausente')),
	origen text NOT NULL CHECK (origen IN ('web', 'panel')),
	nota text NOT NULL DEFAULT '' CHECK (length(nota) <= 500),
	-- Sólo mientras espera el pago: pasado, el horario se libera solo (sin cron).
	vence_en timestamptz,
	creado_en timestamptz NOT NULL DEFAULT now(),
	actualizado_en timestamptz NOT NULL DEFAULT now(),
	CHECK (fin > inicio),
	FOREIGN KEY (negocio_id, servicio_id) REFERENCES servicios (negocio_id, id) ON DELETE SET NULL (servicio_id),
	FOREIGN KEY (negocio_id, profesional_id) REFERENCES profesionales (negocio_id, id) ON DELETE SET NULL (profesional_id),
	FOREIGN KEY (negocio_id, cliente_id) REFERENCES clientes (negocio_id, id) ON DELETE RESTRICT
);
-- Superposiciones por profesional (inicio < $fin AND fin > $inicio), sin los cancelados.
CREATE INDEX turnos_profesional ON turnos (negocio_id, profesional_id, inicio) WHERE estado <> 'cancelado';
CREATE INDEX turnos_agenda ON turnos (negocio_id, inicio);
CREATE INDEX turnos_cliente ON turnos (cliente_id);
SELECT app.aplicar_rls('turnos');
GRANT SELECT, INSERT ON turnos TO tengolibre_app;
GRANT UPDATE (estado, nota, vence_en, actualizado_en, inicio, fin, profesional_id, profesional) ON turnos TO tengolibre_app;

CREATE POLICY por_codigo ON turnos FOR SELECT TO tengolibre_migrador
	USING (codigo = current_setting('app.turno_codigo', true));
-- +goose StatementBegin
-- El link del cliente: del código al negocio del turno.
CREATE FUNCTION app.negocio_del_turno(p_codigo text) RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	n uuid;
BEGIN
	PERFORM set_config('app.turno_codigo', p_codigo, true);
	SELECT negocio_id INTO n FROM turnos WHERE codigo = p_codigo;
	PERFORM set_config('app.turno_codigo', '', true);
	RETURN n;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.negocio_del_turno(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.negocio_del_turno(text) TO tengolibre_app;

-- +goose Down
DROP FUNCTION app.negocio_del_turno(text);
DROP POLICY por_slug ON negocios;
DROP POLICY por_persona ON negocios;
DROP TABLE turnos;
DROP TABLE clientes;
DROP TABLE excepciones;
DROP TABLE bloqueos_semanales;
DROP TABLE bloqueos;
DROP TABLE horarios;
DROP TABLE profesional_servicios;
DROP TABLE profesionales;
DROP TABLE servicios;
DROP TABLE categorias;
DROP FUNCTION app.negocio_publico(text);
DROP FUNCTION app.crear_negocio(text, text, text, text, text);
DROP FUNCTION app.negocios_de(text);
DROP TABLE miembros;
DROP TABLE ajustes;
ALTER TABLE negocios DROP COLUMN descripcion, DROP COLUMN direccion, DROP COLUMN ciudad, DROP COLUMN telefono,
	DROP COLUMN whatsapp, DROP COLUMN instagram, DROP COLUMN logo, DROP COLUMN activo;
