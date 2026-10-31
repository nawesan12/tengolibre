-- +goose Up
-- La plata: cobros con Mercado Pago (cada negocio con su cuenta, por OAuth), la seña o el pago de la
-- reserva, cupones, referidos, membresías y la suscripción del negocio a TengoLibre (con la cuenta de
-- la plataforma). Montos en pesos enteros, como en 00002.

-- Configuración de cobro y del programa de referidos (BusinessSettings.paymentMode, deposit*,
-- referral*). sena_porcentaje y sena_fija sólo cuentan en su modo.
ALTER TABLE ajustes
	ADD COLUMN cobro_modo text NOT NULL DEFAULT 'sin_cobro' CHECK (cobro_modo IN ('sin_cobro', 'porcentaje', 'fija', 'total')),
	ADD COLUMN sena_porcentaje int NOT NULL DEFAULT 50 CHECK (sena_porcentaje BETWEEN 1 AND 100),
	ADD COLUMN sena_fija bigint NOT NULL DEFAULT 0 CHECK (sena_fija >= 0),
	ADD COLUMN referidos_activo boolean NOT NULL DEFAULT false,
	ADD COLUMN referidos_tipo text CHECK (referidos_tipo IN ('porcentaje', 'fijo')),
	ADD COLUMN referidos_valor bigint CHECK (referidos_valor > 0),
	-- Cuántas veces puede usarse el código de una misma persona. NULL: sin tope.
	ADD COLUMN referidos_tope int CHECK (referidos_tope > 0);
GRANT UPDATE (cobro_modo, sena_porcentaje, sena_fija, referidos_activo, referidos_tipo, referidos_valor, referidos_tope)
	ON ajustes TO tengolibre_app;

-- La suscripción del negocio a TengoLibre (Business.plan y trialEndsAt). Hasta prueba_hasta tiene
-- todo; después, sin una suscripción vigente, el panel queda de sólo lectura. NULL: sin prueba (una
-- cuenta de cortesía depende sólo de la suscripción). La prueba se extiende a mano, nunca desde la app.
ALTER TABLE negocios
	ADD COLUMN plan text NOT NULL DEFAULT 'sin_plan' CHECK (plan IN ('sin_plan', 'inicial', 'completo')),
	ADD COLUMN prueba_hasta timestamptz DEFAULT now() + interval '7 days';
GRANT UPDATE (plan) ON negocios TO tengolibre_app;

-- La cuenta de Mercado Pago del negocio (MercadoPagoAccount). Los tokens van cifrados (AES-GCM con una
-- clave por negocio): la base sola no alcanza para cobrar. Nunca se cae al token de la plataforma.
CREATE TABLE mp_conexiones (
	negocio_id uuid PRIMARY KEY REFERENCES negocios (id) ON DELETE CASCADE,
	mp_usuario bigint NOT NULL UNIQUE,
	acceso bytea NOT NULL,
	renovacion bytea NOT NULL,
	vence_en timestamptz NOT NULL,
	alcances text NOT NULL DEFAULT '',
	conectada_en timestamptz NOT NULL DEFAULT now(),
	renovada_en timestamptz,
	-- La renovación falló: hay que volver a conectar. ultimo_error dice por qué (para el panel).
	reautorizar boolean NOT NULL DEFAULT false,
	ultimo_error text NOT NULL DEFAULT '' CHECK (length(ultimo_error) <= 300)
);
CREATE INDEX mp_conexiones_vence ON mp_conexiones (vence_en);
SELECT app.aplicar_rls('mp_conexiones');
GRANT SELECT, INSERT, UPDATE, DELETE ON mp_conexiones TO tengolibre_app;

-- Un OAuth de conexión en curso: un solo uso, 10 minutos, atado al negocio y a quien lo empezó.
CREATE TABLE mp_estados (
	estado_hash text PRIMARY KEY,
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	clerk_id text NOT NULL,
	-- PKCE: la aplicación lo exige.
	verificador text NOT NULL,
	vence_en timestamptz NOT NULL,
	usado_en timestamptz
);
SELECT app.aplicar_rls('mp_estados');
GRANT SELECT, INSERT, UPDATE (usado_en), DELETE ON mp_estados TO tengolibre_app;

-- Las FK compuestas de abajo apuntan a turnos y clientes por (negocio_id, id).
ALTER TABLE turnos ADD CONSTRAINT turnos_negocio_id UNIQUE (negocio_id, id);

-- El cobro de una reserva (Payment): la seña o el total. referencia es el external_reference de la
-- preferencia; se concilia contra esta tabla, nunca contra lo que diga el aviso.
CREATE TABLE pagos (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	turno_id uuid NOT NULL UNIQUE,
	referencia text NOT NULL UNIQUE CHECK (referencia ~ '^[A-Za-z0-9]{16,64}$'),
	monto bigint NOT NULL CHECK (monto > 0),
	modo text NOT NULL CHECK (modo IN ('porcentaje', 'fija', 'total')),
	estado text NOT NULL DEFAULT 'pendiente'
		CHECK (estado IN ('pendiente', 'en_proceso', 'aprobado', 'rechazado', 'cancelado', 'devuelto')),
	-- El init_point de Checkout Pro (no es secreto): se reusa si el cliente vuelve a pedir el link.
	link text,
	mp_pago bigint,
	mp_estado text NOT NULL DEFAULT '',
	-- Lo devuelto (total o parcial).
	devuelto bigint NOT NULL DEFAULT 0 CHECK (devuelto >= 0 AND devuelto <= monto),
	creado_en timestamptz NOT NULL DEFAULT now(),
	actualizado_en timestamptz NOT NULL DEFAULT now(),
	FOREIGN KEY (negocio_id, turno_id) REFERENCES turnos (negocio_id, id) ON DELETE CASCADE
);
CREATE INDEX pagos_negocio ON pagos (negocio_id, creado_en DESC);
SELECT app.aplicar_rls('pagos');
GRANT SELECT, INSERT ON pagos TO tengolibre_app;
GRANT UPDATE (estado, link, mp_pago, mp_estado, devuelto, actualizado_en) ON pagos TO tengolibre_app;

-- El cupón baja el precio del turno; la membresía lo cubre (y se guarda para devolver la visita).
ALTER TABLE turnos ADD COLUMN membresia_id uuid;
GRANT UPDATE (precio, membresia_id) ON turnos TO tengolibre_app;

-- Cupones (Coupon). El código se guarda en mayúsculas: se busca sin distinguir.
CREATE TABLE cupones (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	codigo text NOT NULL CHECK (codigo ~ '^[A-Z0-9_-]{3,40}$'),
	tipo text NOT NULL CHECK (tipo IN ('porcentaje', 'fijo')),
	valor bigint NOT NULL CHECK (valor > 0 AND (tipo = 'fijo' OR valor <= 100)),
	-- Precio mínimo del servicio para que aplique.
	minimo bigint CHECK (minimo >= 0),
	max_usos int CHECK (max_usos > 0),
	usos int NOT NULL DEFAULT 0 CHECK (usos >= 0),
	desde timestamptz NOT NULL DEFAULT now(),
	hasta timestamptz,
	activo boolean NOT NULL DEFAULT true,
	-- La recompensa de un referido: de quién es.
	referido_id uuid,
	creado_en timestamptz NOT NULL DEFAULT now(),
	UNIQUE (negocio_id, codigo),
	UNIQUE (negocio_id, id),
	CHECK (hasta IS NULL OR hasta > desde)
);
SELECT app.aplicar_rls('cupones');
GRANT SELECT, INSERT, DELETE ON cupones TO tengolibre_app;
GRANT UPDATE (tipo, valor, minimo, max_usos, usos, desde, hasta, activo) ON cupones TO tengolibre_app;

CREATE TABLE cupon_usos (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	cupon_id uuid NOT NULL,
	turno_id uuid NOT NULL,
	cliente_id uuid NOT NULL,
	descuento bigint NOT NULL CHECK (descuento >= 0),
	creado_en timestamptz NOT NULL DEFAULT now(),
	FOREIGN KEY (negocio_id, cupon_id) REFERENCES cupones (negocio_id, id) ON DELETE CASCADE,
	FOREIGN KEY (negocio_id, turno_id) REFERENCES turnos (negocio_id, id) ON DELETE CASCADE
);
CREATE INDEX cupon_usos_cupon ON cupon_usos (cupon_id);
SELECT app.aplicar_rls('cupon_usos');
GRANT SELECT, INSERT ON cupon_usos TO tengolibre_app;

-- Referidos (Referral): un código por cliente. Lo pide desde el link de su turno.
CREATE TABLE referidos (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	cliente_id uuid NOT NULL,
	codigo text NOT NULL CHECK (codigo ~ '^[A-Z0-9]{6,12}$'),
	creado_en timestamptz NOT NULL DEFAULT now(),
	UNIQUE (negocio_id, codigo),
	UNIQUE (negocio_id, cliente_id),
	UNIQUE (negocio_id, id),
	FOREIGN KEY (negocio_id, cliente_id) REFERENCES clientes (negocio_id, id) ON DELETE CASCADE
);
SELECT app.aplicar_rls('referidos');
GRANT SELECT, INSERT ON referidos TO tengolibre_app;
ALTER TABLE cupones ADD FOREIGN KEY (negocio_id, referido_id) REFERENCES referidos (negocio_id, id) ON DELETE SET NULL (referido_id);

-- Cada reserva que llegó con un código (ReferralRedemption).
CREATE TABLE referido_usos (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	referido_id uuid NOT NULL,
	turno_id uuid NOT NULL,
	cliente_id uuid NOT NULL,
	creado_en timestamptz NOT NULL DEFAULT now(),
	FOREIGN KEY (negocio_id, referido_id) REFERENCES referidos (negocio_id, id) ON DELETE CASCADE,
	FOREIGN KEY (negocio_id, turno_id) REFERENCES turnos (negocio_id, id) ON DELETE CASCADE
);
CREATE INDEX referido_usos_referido ON referido_usos (referido_id);
SELECT app.aplicar_rls('referido_usos');
GRANT SELECT, INSERT ON referido_usos TO tengolibre_app;

-- Planes de membresía (MembershipPlan): un paquete de visitas por período. servicios: los que cubre
-- (vacío: todos).
CREATE TABLE membresia_planes (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	nombre text NOT NULL CHECK (length(nombre) BETWEEN 2 AND 60),
	descripcion text NOT NULL DEFAULT '' CHECK (length(descripcion) <= 400),
	precio bigint NOT NULL CHECK (precio > 0 AND precio <= 100000000),
	visitas int NOT NULL DEFAULT 0 CHECK (visitas BETWEEN 0 AND 1000),
	duracion_dias int NOT NULL DEFAULT 30 CHECK (duracion_dias BETWEEN 1 AND 365),
	prioridad_dias int NOT NULL DEFAULT 0 CHECK (prioridad_dias BETWEEN 0 AND 90),
	beneficios text[] NOT NULL DEFAULT '{}' CHECK (cardinality(beneficios) <= 12),
	tope_socios int CHECK (tope_socios BETWEEN 1 AND 100000),
	servicios uuid[] NOT NULL DEFAULT '{}',
	activo boolean NOT NULL DEFAULT true,
	orden int NOT NULL DEFAULT 0 CHECK (orden BETWEEN 0 AND 999),
	creado_en timestamptz NOT NULL DEFAULT now(),
	UNIQUE (negocio_id, nombre),
	UNIQUE (negocio_id, id)
);
SELECT app.aplicar_rls('membresia_planes');
GRANT SELECT, INSERT, DELETE ON membresia_planes TO tengolibre_app;
GRANT UPDATE (nombre, descripcion, precio, visitas, duracion_dias, prioridad_dias, beneficios, tope_socios, servicios, activo, orden)
	ON membresia_planes TO tengolibre_app;

-- Un socio en un plan, por un período (Membership). Vencida se calcula con el reloj: estado sólo
-- dice si se canceló.
CREATE TABLE membresias (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	plan_id uuid NOT NULL,
	cliente_id uuid NOT NULL,
	desde timestamptz NOT NULL,
	hasta timestamptz NOT NULL,
	estado text NOT NULL DEFAULT 'activa' CHECK (estado IN ('activa', 'cancelada')),
	nota text NOT NULL DEFAULT '' CHECK (length(nota) <= 300),
	creado_en timestamptz NOT NULL DEFAULT now(),
	CHECK (hasta > desde),
	UNIQUE (negocio_id, id),
	FOREIGN KEY (negocio_id, plan_id) REFERENCES membresia_planes (negocio_id, id) ON DELETE RESTRICT,
	FOREIGN KEY (negocio_id, cliente_id) REFERENCES clientes (negocio_id, id) ON DELETE CASCADE
);
CREATE INDEX membresias_cliente ON membresias (negocio_id, cliente_id, hasta);
SELECT app.aplicar_rls('membresias');
GRANT SELECT, INSERT ON membresias TO tengolibre_app;
-- UPDATE también lo pide el SELECT ... FOR UPDATE con que se gasta una visita.
GRANT UPDATE (desde, hasta, estado, nota) ON membresias TO tengolibre_app;
ALTER TABLE turnos ADD FOREIGN KEY (negocio_id, membresia_id) REFERENCES membresias (negocio_id, id) ON DELETE SET NULL (membresia_id);

-- Cada movimiento de visitas (MembershipCredit): el saldo es la suma, nunca una columna.
CREATE TABLE membresia_creditos (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	membresia_id uuid NOT NULL,
	-- Positivo da visitas, negativo las gasta.
	cantidad int NOT NULL CHECK (cantidad <> 0 AND cantidad BETWEEN -1000 AND 1000),
	motivo text NOT NULL DEFAULT '' CHECK (length(motivo) <= 120),
	turno_id uuid,
	-- Quién lo cargó a mano. NULL: el sistema.
	clerk_id text,
	creado_en timestamptz NOT NULL DEFAULT now(),
	FOREIGN KEY (negocio_id, membresia_id) REFERENCES membresias (negocio_id, id) ON DELETE CASCADE,
	FOREIGN KEY (negocio_id, turno_id) REFERENCES turnos (negocio_id, id) ON DELETE SET NULL (turno_id)
);
CREATE INDEX membresia_creditos_membresia ON membresia_creditos (membresia_id);
CREATE INDEX membresia_creditos_turno ON membresia_creditos (turno_id) WHERE turno_id IS NOT NULL;
SELECT app.aplicar_rls('membresia_creditos');
GRANT SELECT, INSERT ON membresia_creditos TO tengolibre_app;

-- La suscripción del negocio (Subscription): un preapproval de la cuenta de la plataforma.
CREATE TABLE suscripciones (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	negocio_id uuid NOT NULL REFERENCES negocios (id) ON DELETE CASCADE,
	-- external_reference: empieza con el negocio, así el aviso sabe dónde buscar.
	referencia text NOT NULL UNIQUE,
	mp_id text UNIQUE,
	plan text NOT NULL CHECK (plan IN ('inicial', 'completo')),
	estado text NOT NULL DEFAULT 'pendiente' CHECK (estado IN ('pendiente', 'autorizada', 'pausada', 'cancelada', 'vencida')),
	email text NOT NULL DEFAULT '',
	monto bigint NOT NULL CHECK (monto > 0),
	link text,
	proximo_cobro timestamptz,
	ultimo_cobro timestamptz,
	-- Pausada (tarjeta rechazada, o cancelada con el mes pago): sigue valiendo hasta acá.
	gracia_hasta timestamptz,
	cancelada_en timestamptz,
	motivo text NOT NULL DEFAULT '' CHECK (length(motivo) <= 200),
	creado_en timestamptz NOT NULL DEFAULT now(),
	actualizado_en timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX suscripciones_negocio ON suscripciones (negocio_id, creado_en DESC);
SELECT app.aplicar_rls('suscripciones');
GRANT SELECT, INSERT ON suscripciones TO tengolibre_app;
GRANT UPDATE (mp_id, estado, email, link, proximo_cobro, ultimo_cobro, gracia_hasta, cancelada_en, motivo, actualizado_en)
	ON suscripciones TO tengolibre_app;

-- Los avisos de pagos traen el usuario de Mercado Pago del vendedor, no el negocio: esto dice de cuál
-- es (la RLS forzada también alcanza al dueño de la función).
CREATE POLICY por_usuario_mp ON mp_conexiones FOR SELECT TO tengolibre_migrador
	USING (mp_usuario::text = current_setting('app.mp_usuario', true));
-- +goose StatementBegin
CREATE FUNCTION app.negocio_de_mp(p_usuario bigint) RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
DECLARE
	n uuid;
BEGIN
	PERFORM set_config('app.mp_usuario', p_usuario::text, true);
	SELECT negocio_id INTO n FROM mp_conexiones WHERE mp_usuario = p_usuario;
	PERFORM set_config('app.mp_usuario', '', true);
	RETURN n;
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.negocio_de_mp(bigint) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.negocio_de_mp(bigint) TO tengolibre_app;

-- La renovación anticipada (mercadopago-renewal.ts): qué negocios tienen el token por vencer. Sólo
-- devuelve ids; los tokens se leen después con el negocio fijado.
CREATE POLICY por_vencer ON mp_conexiones FOR SELECT TO tengolibre_migrador
	USING (current_setting('app.mp_por_vencer', true) = 'si');
-- +goose StatementBegin
CREATE FUNCTION app.mp_por_vencer(p_hasta timestamptz, p_tope int) RETURNS SETOF uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, app AS $$
BEGIN
	PERFORM set_config('app.mp_por_vencer', 'si', true);
	RETURN QUERY SELECT negocio_id FROM mp_conexiones WHERE vence_en <= p_hasta AND NOT reautorizar
		ORDER BY vence_en LIMIT least(p_tope, 100);
	PERFORM set_config('app.mp_por_vencer', '', true);
END
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.mp_por_vencer(timestamptz, int) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.mp_por_vencer(timestamptz, int) TO tengolibre_app;

-- +goose Down
DROP FUNCTION app.mp_por_vencer(timestamptz, int);
DROP FUNCTION app.negocio_de_mp(bigint);
DROP TABLE suscripciones;
DROP TABLE membresia_creditos;
ALTER TABLE turnos DROP COLUMN membresia_id;
DROP TABLE membresias;
DROP TABLE membresia_planes;
DROP TABLE referido_usos;
ALTER TABLE cupones DROP COLUMN referido_id;
DROP TABLE referidos;
DROP TABLE cupon_usos;
DROP TABLE cupones;
DROP TABLE pagos;
ALTER TABLE turnos DROP CONSTRAINT turnos_negocio_id;
DROP TABLE mp_estados;
DROP TABLE mp_conexiones;
ALTER TABLE negocios DROP COLUMN plan, DROP COLUMN prueba_hasta;
ALTER TABLE ajustes DROP COLUMN cobro_modo, DROP COLUMN sena_porcentaje, DROP COLUMN sena_fija, DROP COLUMN referidos_activo,
	DROP COLUMN referidos_tipo, DROP COLUMN referidos_valor, DROP COLUMN referidos_tope;
